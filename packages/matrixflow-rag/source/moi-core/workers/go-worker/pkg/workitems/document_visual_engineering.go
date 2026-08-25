package workitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path"
	"strings"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"github.com/matrixflow/moi-core/model/data"
	"github.com/matrixflow/moi-core/workers/go-worker/pkg/runtime"
	"github.com/matrixflow/moi-core/workers/go-worker/pkg/workitems/parser"
	"golang.org/x/net/html"
)

const (
	engineeringPagePlanBBoxCoordinate = "page_normalized_1000"

	engineeringRegionExtractFailedWarning = "region_extract_failed"

	engineeringTableStatusExtracted   = "extracted"
	engineeringTableStatusUnavailable = "unavailable"

	engineeringTableSourceVLMExtract  = "engineering_vlm_table_extract"
	engineeringTableSourceUnavailable = "unavailable"

	engineeringTableCandidateSourcePagePlan = "page_plan"

	engineeringTableValidationPassed = "passed"
	engineeringTableValidationFailed = "failed"

	engineeringTableFailureStageExtract        = "extract"
	engineeringTableFailureStageHTMLValidation = "extract_html_validation"
)

type documentVisualTableSummary struct {
	Total       int `json:"total"`
	Extracted   int `json:"extracted"`
	Unavailable int `json:"unavailable"`
}

type documentVisualTableMetadata struct {
	TableStatus            string    `json:"table_status,omitempty"`
	TableSource            string    `json:"table_source,omitempty"`
	HTMLValidationStatus   string    `json:"html_validation_status,omitempty"`
	CandidateSource        string    `json:"candidate_source,omitempty"`
	PagePlanMatch          bool      `json:"page_plan_match"`
	PagePlanRegionID       string    `json:"page_plan_region_id,omitempty"`
	PagePlanRegionKind     string    `json:"page_plan_region_kind,omitempty"`
	PagePlanBBox           []float64 `json:"page_plan_bbox,omitempty"`
	PagePlanBBoxCoordinate string    `json:"page_plan_bbox_coordinate,omitempty"`
	FailureStage           string    `json:"failure_stage,omitempty"`
	TableUnavailableReason string    `json:"table_unavailable_reason,omitempty"`
}

type engineeringDrawingPagePlan struct {
	PageNumber int                        `json:"page_number"`
	Regions    []engineeringDrawingRegion `json:"regions"`
}

type engineeringDrawingRegion struct {
	RegionID           string    `json:"region_id"`
	Kind               string    `json:"kind"`
	Label              string    `json:"label"`
	BBox               []float64 `json:"bbox"`
	ContainsDimensions bool      `json:"contains_dimensions"`
	ContainsTables     bool      `json:"contains_tables"`
	Reason             string    `json:"reason"`
}

type engineeringDrawingRegionExtract struct {
	RegionID string                          `json:"region_id"`
	ViewName string                          `json:"view_name"`
	Items    []engineeringDrawingExtractItem `json:"items"`
}

type engineeringDrawingExtractItem struct {
	Text         string                          `json:"text"`
	Kind         string                          `json:"kind"`
	BBox         []float64                       `json:"bbox"`
	Orientation  string                          `json:"orientation"`
	Confidence   float64                         `json:"confidence"`
	Uncertain    bool                            `json:"uncertain"`
	Truncated    bool                            `json:"truncated"`
	Source       string                          `json:"source"`
	VisualMarker *engineeringDrawingVisualMarker `json:"visual_marker,omitempty"`
}

type engineeringDrawingVisualMarker struct {
	Shape string    `json:"shape"`
	BBox  []float64 `json:"bbox"`
}

type engineeringRegionCandidate struct {
	RegionID           string
	Kind               string
	Label              string
	PageNumber         int
	BBox               []float64
	PlanBBox           []float64
	ObjectID           string
	PageImageFileID    string
	ContainsDimensions bool
	ContainsTables     bool
	Reason             string
}

type engineeringTableExtract struct {
	RegionID string                         `json:"region_id"`
	Tables   []engineeringTableExtractTable `json:"tables"`
}

type engineeringTableExtractTable struct {
	Caption     string    `json:"caption"`
	HTML        string    `json:"html"`
	BBox        []float64 `json:"bbox"`
	RowCount    int       `json:"row_count"`
	ColumnCount int       `json:"column_count"`
	Confidence  float64   `json:"confidence"`
	Uncertain   bool      `json:"uncertain"`
	Truncated   bool      `json:"truncated"`
}

type documentVisualJSONDecodeError struct {
	Op  string
	Raw string
	Err error
}

func (e documentVisualJSONDecodeError) Error() string {
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e documentVisualJSONDecodeError) Unwrap() error {
	return e.Err
}

func (p *DocumentVisualParse) enhanceIndustrialDrawingManifest(ctx context.Context, execCtx data.ExecutionContext, client *moi.Client, workspaceID string, manifest documentVisualManifest, pageAssets map[int]documentVisualPageAsset, source documentVisualSource, opts documentVisualEngineeringOptions, buildOpts documentVisualBuildOptions) (documentVisualManifest, error) {
	finalManifest := documentVisualManifest{
		SchemaVersion: manifest.SchemaVersion,
		Profile:       manifest.Profile,
		Source:        manifest.Source,
		Pages:         engineeringDrawingFinalPages(manifest.Pages, pageAssets),
	}
	pageIndex := map[int]int{}
	for i, page := range finalManifest.Pages {
		pageIndex[page.PageNumber] = i
	}
	skippedRegionWarnings := []string{}
	for _, basePage := range manifest.Pages {
		asset, ok := pageAssets[basePage.PageNumber]
		if !ok || len(asset.Bytes) == 0 || asset.FileID == "" {
			return manifest, fmt.Errorf("document_visual.parse: engineering page plan requires page image for page %d", basePage.PageNumber)
		}
		page := engineeringImageCoordinatePage(basePage, asset)
		rawPlan, err := parser.VLMEngineeringDrawingPagePlan(ctx, p.Factory, execCtx, asset.Bytes, documentVisualPageAssetMimeType(asset), opts.Model, opts.ReasoningEffort, asset.Width, asset.Height)
		if err != nil {
			return manifest, fmt.Errorf("document_visual.parse: engineering page plan page=%d model=%s: %w", page.PageNumber, opts.Model, err)
		}
		plan, err := parseEngineeringDrawingPagePlan(rawPlan)
		if err != nil {
			return manifest, fmt.Errorf("document_visual.parse: engineering page plan page=%d model=%s: %w", page.PageNumber, opts.Model, err)
		}
		candidates, skippedWarnings := engineeringRegionCandidatesWithSkippedReasons(manifest, page, asset, plan)
		if len(plan.Regions) > 0 && len(candidates) == 0 {
			skippedRegionWarnings = append(skippedRegionWarnings, fmt.Sprintf("engineering page plan page=%d has no valid regions: %s", page.PageNumber, strings.Join(skippedWarnings, "; ")))
			continue
		}
		skippedRegionWarnings = append(skippedRegionWarnings, skippedWarnings...)
		for _, candidate := range candidates {
			obj, imageBytes, err := p.engineeringCandidateObjectAndImage(ctx, client, workspaceID, source.FileName, candidate, asset, page)
			if err != nil {
				return manifest, err
			}
			if engineeringRegionIsTableLike(candidate) {
				obj = p.extractEngineeringTableRegionObject(ctx, execCtx, candidate, obj, imageBytes, opts.Model, opts.ReasoningEffort)
			} else {
				obj, finalManifest.Entities = p.extractEngineeringOCRRegion(ctx, execCtx, manifest.Source.FileID, candidate, obj, imageBytes, opts.Model, opts.ReasoningEffort, finalManifest.Entities)
			}
			finalManifest.Objects = append(finalManifest.Objects, obj)
			if idx, ok := pageIndex[obj.PageNumber]; ok {
				finalManifest.Pages[idx].ObjectIDs = append(finalManifest.Pages[idx].ObjectIDs, obj.ObjectID)
			}
		}
	}
	if len(finalManifest.Objects) == 0 {
		return manifest, fmt.Errorf("document_visual.parse: engineering page plan produced no valid objects")
	}
	finalManifest.TableSummary = documentVisualComputeTableSummary(finalManifest.Objects)
	finalManifest.Validation = validateDocumentVisualManifest(finalManifest, buildOpts)
	finalManifest.Validation.Warnings = append(finalManifest.Validation.Warnings, skippedRegionWarnings...)
	if !finalManifest.Validation.Valid {
		return manifest, fmt.Errorf("document_visual.parse: manifest validation failed after engineering enhancement: %s", strings.Join(finalManifest.Validation.Errors, "; "))
	}
	if err := validateEngineeringDrawingObjectEntities(finalManifest); err != nil {
		return manifest, err
	}
	return finalManifest, nil
}

func engineeringDrawingFinalPages(pages []documentVisualPage, pageAssets map[int]documentVisualPageAsset) []documentVisualPage {
	finalPages := make([]documentVisualPage, len(pages))
	for i, page := range pages {
		finalPages[i] = engineeringImageCoordinatePage(page, pageAssets[page.PageNumber])
		finalPages[i].ObjectIDs = nil
		finalPages[i].Summary = ""
		finalPages[i].Text = ""
	}
	return finalPages
}

func engineeringImageCoordinatePage(page documentVisualPage, asset documentVisualPageAsset) documentVisualPage {
	if asset.FileID != "" {
		page.PageImageFileID = asset.FileID
	}
	if asset.Width > 0 && asset.Height > 0 {
		page.Width = float64(asset.Width)
		page.Height = float64(asset.Height)
		page.BBox = []float64{0, 0, float64(asset.Width), float64(asset.Height)}
	}
	return page
}

func parseEngineeringDrawingPagePlan(raw string) (engineeringDrawingPagePlan, error) {
	var plan engineeringDrawingPagePlan
	if err := json.Unmarshal([]byte(stripDocumentVisualJSON(raw)), &plan); err != nil {
		return plan, fmt.Errorf("decode page plan JSON: %w", err)
	}
	return plan, nil
}

func parseEngineeringDrawingRegionExtract(raw string) (engineeringDrawingRegionExtract, error) {
	var extract engineeringDrawingRegionExtract
	if err := json.Unmarshal([]byte(stripDocumentVisualJSON(raw)), &extract); err != nil {
		return extract, fmt.Errorf("decode region extract JSON: %w", err)
	}
	return extract, nil
}

func parseEngineeringTableExtract(raw, op string) (engineeringTableExtract, error) {
	var out engineeringTableExtract
	if err := json.Unmarshal([]byte(stripDocumentVisualJSON(raw)), &out); err != nil {
		return out, documentVisualJSONDecodeError{Op: op, Raw: raw, Err: err}
	}
	return out, nil
}

func engineeringRegionCandidates(manifest documentVisualManifest, page documentVisualPage, asset documentVisualPageAsset, plan engineeringDrawingPagePlan) []engineeringRegionCandidate {
	candidates, _ := engineeringRegionCandidatesWithSkippedReasons(manifest, page, asset, plan)
	return candidates
}

func engineeringRegionCandidatesWithSkippedReasons(manifest documentVisualManifest, page documentVisualPage, asset documentVisualPageAsset, plan engineeringDrawingPagePlan) ([]engineeringRegionCandidate, []string) {
	candidates := make([]engineeringRegionCandidate, 0, len(plan.Regions))
	skippedWarnings := make([]string, 0)
	for i, region := range plan.Regions {
		regionRef := engineeringRegionWarningRef(page.PageNumber, i, region.RegionID)
		if len(region.BBox) != 4 {
			skippedWarnings = append(skippedWarnings, fmt.Sprintf("engineering page plan skipped %s: bbox must have 4 coordinates", regionRef))
			continue
		}
		if !engineeringRegionKindIsValid(region.Kind) {
			skippedWarnings = append(skippedWarnings, fmt.Sprintf("engineering page plan skipped %s: invalid kind %q", regionRef, region.Kind))
			continue
		}
		imageBBox, err := engineeringPagePlanBBoxToImageBBox(region.BBox, asset.Width, asset.Height)
		if err != nil {
			skippedWarnings = append(skippedWarnings, fmt.Sprintf("engineering page plan skipped %s: %v", regionRef, err))
			continue
		}
		if err := validateDocumentVisualPageBBox(imageBBox, page.Width, page.Height, fmt.Sprintf("engineering region %s bbox", firstNonEmptyString(region.RegionID, fmt.Sprint(i)))); err != nil {
			skippedWarnings = append(skippedWarnings, fmt.Sprintf("engineering page plan skipped %s: %v", regionRef, err))
			continue
		}
		regionID := firstNonEmptyString(region.RegionID, hashShort("engineering-region", manifest.Source.FileID, fmt.Sprint(page.PageNumber), fmt.Sprint(i), fmt.Sprint(imageBBox)))
		candidates = append(candidates, engineeringRegionCandidate{
			RegionID:           regionID,
			Kind:               region.Kind,
			Label:              region.Label,
			PageNumber:         page.PageNumber,
			BBox:               imageBBox,
			PlanBBox:           append([]float64(nil), region.BBox...),
			ObjectID:           hashShort("engineering-object", manifest.Source.FileID, fmt.Sprint(page.PageNumber), regionID, fmt.Sprint(imageBBox)),
			PageImageFileID:    page.PageImageFileID,
			ContainsDimensions: region.ContainsDimensions,
			ContainsTables:     region.ContainsTables,
			Reason:             region.Reason,
		})
	}
	return candidates, skippedWarnings
}

func engineeringRegionWarningRef(pageNumber, index int, regionID string) string {
	if regionID != "" {
		return fmt.Sprintf("page=%d region_id=%s index=%d", pageNumber, regionID, index)
	}
	return fmt.Sprintf("page=%d index=%d", pageNumber, index)
}

func engineeringPagePlanBBoxToImageBBox(bbox []float64, imageWidth, imageHeight int) ([]float64, error) {
	if len(bbox) != 4 {
		return nil, fmt.Errorf("document_visual.parse: page plan bbox must have 4 coordinates")
	}
	if imageWidth <= 0 || imageHeight <= 0 {
		return nil, fmt.Errorf("document_visual.parse: rendered image dimensions are required")
	}
	for _, v := range bbox {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("document_visual.parse: page plan bbox coordinate is not finite: bbox=%v", bbox)
		}
		if v < 0 || v > 1000 {
			return nil, fmt.Errorf("document_visual.parse: page plan bbox out of normalized bounds: bbox=%v", bbox)
		}
	}
	if bbox[2] <= bbox[0] || bbox[3] <= bbox[1] {
		return nil, fmt.Errorf("document_visual.parse: page plan bbox is invalid: bbox=%v", bbox)
	}
	imageBBox := []float64{
		bbox[0] / 1000 * float64(imageWidth),
		bbox[1] / 1000 * float64(imageHeight),
		bbox[2] / 1000 * float64(imageWidth),
		bbox[3] / 1000 * float64(imageHeight),
	}
	if err := validateDocumentVisualImageBBox(imageBBox, imageWidth, imageHeight, "page plan image bbox"); err != nil {
		return nil, err
	}
	return imageBBox, nil
}

func engineeringRegionKindIsValid(kind string) bool {
	switch kind {
	case "table", "title_block", "part", "dimension_area", "technical_note", "drawing_view", "notes", "dimension_cluster", "other":
		return true
	default:
		return false
	}
}

func engineeringRegionIsTableLike(candidate engineeringRegionCandidate) bool {
	return candidate.Kind == "table" || candidate.Kind == "title_block" || candidate.ContainsTables
}

func (p *DocumentVisualParse) engineeringCandidateObjectAndImage(ctx context.Context, client *moi.Client, workspaceID, sourceFileName string, candidate engineeringRegionCandidate, asset documentVisualPageAsset, page documentVisualPage) (documentVisualObject, []byte, error) {
	imageBytes, err := cropDocumentVisualPageImage(asset.Bytes, candidate.BBox, page.Width, page.Height, asset.Width, asset.Height, candidate.ObjectID)
	if err != nil {
		return documentVisualObject{}, nil, err
	}
	obj := documentVisualObject{
		ObjectID:           candidate.ObjectID,
		ObjectKind:         candidate.Kind,
		PageNumber:         candidate.PageNumber,
		BBox:               append([]float64(nil), candidate.BBox...),
		OCRBBox:            append([]float64(nil), candidate.BBox...),
		PageImageFileID:    candidate.PageImageFileID,
		Caption:            candidate.Label,
		Context:            engineeringDrawingObjectContext(candidate),
		ExtractionSource:   "engineering_page_plan",
		ExtractionWarnings: nil,
	}
	resp, err := client.Files().UploadBytes(ctx, workspaceID, documentVisualObjectImageName(sourceFileName, obj), imageBytes)
	if err != nil {
		return documentVisualObject{}, nil, fmt.Errorf("document_visual.parse: upload engineering region image %s: %w", obj.ObjectID, err)
	}
	obj.ImageFileID = resp.FileID
	return obj, imageBytes, nil
}

func engineeringDrawingObjectContext(candidate engineeringRegionCandidate) string {
	bbox := ""
	if len(candidate.BBox) == 4 {
		bbox = fmt.Sprintf("bbox=[%.2f,%.2f,%.2f,%.2f]", candidate.BBox[0], candidate.BBox[1], candidate.BBox[2], candidate.BBox[3])
	}
	return joinNonEmpty([]string{
		candidate.Label,
		candidate.Reason,
		fmt.Sprintf("kind=%s", candidate.Kind),
		fmt.Sprintf("page=%d", candidate.PageNumber),
		fmt.Sprintf("region_id=%s", candidate.RegionID),
		bbox,
	}, "\n")
}

func (p *DocumentVisualParse) extractEngineeringOCRRegion(ctx context.Context, execCtx runtime.ExecutionContext, sourceFileID string, candidate engineeringRegionCandidate, obj documentVisualObject, imageBytes []byte, model string, reasoningEffort string, entities []documentVisualEntity) (documentVisualObject, []documentVisualEntity) {
	rawExtract, err := parser.VLMEngineeringDrawingRegionExtract(ctx, p.Factory, execCtx, imageBytes, "image/jpeg", model, reasoningEffort)
	if err != nil {
		obj.ExtractionWarnings = append(obj.ExtractionWarnings, engineeringRegionExtractFailedWarning)
		obj.ExtractionFailureReason = err.Error()
		return obj, entities
	}
	extract, err := parseEngineeringDrawingRegionExtract(rawExtract)
	if err != nil {
		obj.ExtractionWarnings = append(obj.ExtractionWarnings, engineeringRegionExtractFailedWarning)
		obj.ExtractionFailureReason = err.Error()
		return obj, entities
	}
	obj.OCR = engineeringRegionExtractText(extract)
	obj.Text = obj.OCR
	return obj, append(entities, engineeringEntitiesFromExtract(sourceFileID, obj, candidate, extract)...)
}

func documentVisualPageAssetMimeType(asset documentVisualPageAsset) string {
	if strings.TrimSpace(asset.MimeType) != "" {
		return asset.MimeType
	}
	return "image/jpeg"
}

func engineeringRegionExtractText(extract engineeringDrawingRegionExtract) string {
	lines := make([]string, 0, len(extract.Items))
	for _, item := range extract.Items {
		if item.Text != "" {
			lines = append(lines, item.Text)
		}
	}
	return joinNonEmpty(lines, "\n")
}

func (p *DocumentVisualParse) extractEngineeringTableRegionObject(ctx context.Context, execCtx runtime.ExecutionContext, candidate engineeringRegionCandidate, obj documentVisualObject, imageBytes []byte, model string, reasoningEffort string) documentVisualObject {
	raw, err := parser.VLMEngineeringDrawingTableHTMLExtract(ctx, p.Factory, execCtx, imageBytes, "image/jpeg", model, reasoningEffort, engineeringTablePromptMetadata(candidate, obj))
	if err != nil {
		return engineeringUnavailableTableObject(obj, candidate, engineeringTableFailureStageExtract, err.Error())
	}
	extract, err := parseEngineeringTableExtract(raw, "decode table extract JSON")
	if err != nil {
		var decodeErr documentVisualJSONDecodeError
		if errors.As(err, &decodeErr) {
			raw, err = parser.VLMEngineeringDrawingTableHTMLExtractJSONRetry(ctx, p.Factory, execCtx, imageBytes, "image/jpeg", model, reasoningEffort, engineeringTablePromptMetadata(candidate, obj), decodeErr.Raw, err.Error())
			if err == nil {
				extract, err = parseEngineeringTableExtract(raw, "decode table extract retry JSON")
			}
		}
		if err != nil {
			return engineeringUnavailableTableObject(obj, candidate, engineeringTableFailureStageExtract, err.Error())
		}
	}
	if len(extract.Tables) == 0 {
		return engineeringUnavailableTableObject(obj, candidate, engineeringTableFailureStageExtract, "vlm returned no tables")
	}
	table, err := firstValidEngineeringTableHTML(extract.Tables)
	if err != nil {
		return engineeringUnavailableTableObject(obj, candidate, engineeringTableFailureStageHTMLValidation, err.Error())
	}
	obj.Text = table.HTML
	obj.OCR = obj.Text
	obj.Table = engineeringTableMetadata(candidate, engineeringTableStatusExtracted, engineeringTableSourceVLMExtract, engineeringTableValidationPassed, "")
	return obj
}

func firstValidEngineeringTableHTML(tables []engineeringTableExtractTable) (engineeringTableExtractTable, error) {
	var lastErr error
	for _, table := range tables {
		if err := validateDocumentVisualTableHTML(table.HTML); err != nil {
			lastErr = err
			continue
		}
		return table, nil
	}
	if lastErr != nil {
		return engineeringTableExtractTable{}, lastErr
	}
	return engineeringTableExtractTable{}, fmt.Errorf("html table is empty")
}

func engineeringUnavailableTableObject(obj documentVisualObject, candidate engineeringRegionCandidate, stage, reason string) documentVisualObject {
	obj.Text = ""
	obj.OCR = ""
	obj.ExtractionWarnings = append(obj.ExtractionWarnings, "table_extract_unavailable")
	obj.Table = engineeringTableMetadata(candidate, engineeringTableStatusUnavailable, engineeringTableSourceUnavailable, engineeringTableValidationFailed, reason)
	obj.Table.FailureStage = stage
	return obj
}

func engineeringTableMetadata(candidate engineeringRegionCandidate, status, source, htmlStatus, reason string) *documentVisualTableMetadata {
	return &documentVisualTableMetadata{
		TableStatus:            status,
		TableSource:            source,
		HTMLValidationStatus:   htmlStatus,
		CandidateSource:        engineeringTableCandidateSourcePagePlan,
		PagePlanMatch:          true,
		PagePlanRegionID:       candidate.RegionID,
		PagePlanRegionKind:     candidate.Kind,
		PagePlanBBox:           append([]float64(nil), candidate.PlanBBox...),
		PagePlanBBoxCoordinate: engineeringPagePlanBBoxCoordinate,
		TableUnavailableReason: reason,
	}
}

func engineeringTablePromptMetadata(candidate engineeringRegionCandidate, obj documentVisualObject) map[string]interface{} {
	return map[string]interface{}{
		"candidate_id":          obj.ObjectID,
		"candidate_source":      engineeringTableCandidateSourcePagePlan,
		"page_number":           obj.PageNumber,
		"bbox":                  obj.BBox,
		"caption":               obj.Caption,
		"page_plan_match":       true,
		"page_plan_region_id":   candidate.RegionID,
		"page_plan_region_kind": candidate.Kind,
	}
}

func documentVisualComputeTableSummary(objects []documentVisualObject) documentVisualTableSummary {
	var summary documentVisualTableSummary
	for _, obj := range objects {
		if obj.Table == nil {
			continue
		}
		summary.Total++
		switch obj.Table.TableStatus {
		case engineeringTableStatusExtracted:
			summary.Extracted++
		case engineeringTableStatusUnavailable:
			summary.Unavailable++
		}
	}
	return summary
}

func validateDocumentVisualTableHTML(tableHTML string) error {
	if strings.TrimSpace(tableHTML) == "" {
		return fmt.Errorf("html table is empty")
	}
	root, err := html.Parse(strings.NewReader(tableHTML))
	if err != nil {
		return fmt.Errorf("parse HTML table: %w", err)
	}
	tables := documentVisualTableNodes(root)
	if len(tables) != 1 {
		return fmt.Errorf("expected exactly one table, got %d", len(tables))
	}
	if len(documentVisualTableRows(tables[0])) == 0 {
		return fmt.Errorf("html table must contain at least one explicit <tr>")
	}
	hasCell := false
	for _, row := range documentVisualTableRows(tables[0]) {
		if len(documentVisualTableCells(row)) > 0 {
			hasCell = true
			break
		}
	}
	if !hasCell {
		return fmt.Errorf("html table must contain at least one explicit <td> or <th>")
	}
	return nil
}

func documentVisualTableNodes(root *html.Node) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "table" {
			out = append(out, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return out
}

func documentVisualTableRows(table *html.Node) []*html.Node {
	var rows []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			rows = append(rows, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(table)
	return rows
}

func documentVisualTableCells(row *html.Node) []*html.Node {
	var cells []*html.Node
	for c := row.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "td" || c.Data == "th") {
			cells = append(cells, c)
		}
	}
	return cells
}

func validateEngineeringDrawingObjectEntities(manifest documentVisualManifest) error {
	if len(manifest.Objects) == 0 {
		return fmt.Errorf("document_visual.parse: engineering page plan produced no engineering objects")
	}
	return nil
}

func engineeringEntitiesFromExtract(sourceFileID string, obj documentVisualObject, candidate engineeringRegionCandidate, extract engineeringDrawingRegionExtract) []documentVisualEntity {
	out := make([]documentVisualEntity, 0, len(extract.Items))
	viewName := firstNonEmptyString(extract.ViewName, candidate.Label)
	regionID := firstNonEmptyString(extract.RegionID, candidate.RegionID)
	for i, item := range extract.Items {
		value := item.Text
		if value == "" {
			continue
		}
		kind := firstNonEmptyString(item.Kind, "other")
		meta := map[string]interface{}{
			"kind":                 kind,
			"extraction_source":    "engineering_region_extract",
			"region_id":            regionID,
			"view_name":            viewName,
			"bbox":                 floatSliceMetadataValue(obj.BBox),
			"ocr_bbox":             floatSliceMetadataValue(obj.OCRBBox),
			"bbox_coordinate":      "page_image",
			"ocr_bbox_coordinate":  "page_image",
			"source":               firstNonEmptyString(item.Source, "region_crop"),
			"page_image_file_id":   obj.PageImageFileID,
			"image_file_id":        obj.ImageFileID,
			"confidence":           item.Confidence,
			"uncertain":            item.Uncertain,
			"truncated":            item.Truncated,
			"orientation":          item.Orientation,
			"item_bbox":            floatSliceMetadataValue(item.BBox),
			"item_bbox_coordinate": "region_crop",
		}
		if len(candidate.PlanBBox) > 0 {
			meta["vlm_plan_bbox"] = floatSliceMetadataValue(candidate.PlanBBox)
			meta["vlm_plan_bbox_coordinate"] = engineeringPagePlanBBoxCoordinate
		}
		if len(obj.ExtractionWarnings) > 0 {
			meta["extraction_warnings"] = stringSliceMetadataValue(obj.ExtractionWarnings)
		}
		if item.VisualMarker != nil {
			if item.VisualMarker.Shape != "" {
				meta["visual_marker_shape"] = item.VisualMarker.Shape
			}
			if len(item.VisualMarker.BBox) == 4 {
				meta["visual_marker_bbox"] = floatSliceMetadataValue(item.VisualMarker.BBox)
			}
			meta["visual_marker_id"] = hashShort("engineering-visual-marker", sourceFileID, obj.ObjectID, regionID, fmt.Sprint(i), item.VisualMarker.Shape, fmt.Sprint(item.VisualMarker.BBox))
		}
		out = append(out, documentVisualEntity{
			EntityID:   hashShort("engineering-entity", sourceFileID, obj.ObjectID, regionID, fmt.Sprint(i), kind, value),
			Type:       kind,
			Value:      value,
			PageNumber: obj.PageNumber,
			ObjectID:   obj.ObjectID,
			Evidence:   viewName,
			Metadata:   meta,
		})
	}
	return out
}

func (p *DocumentVisualParse) appendEngineeringDrawingDocuments(ctx context.Context, client *moi.Client, workspaceID string, manifest documentVisualManifest) ([]*data.Document, error) {
	attachment := engineeringDrawingAttachmentMarkdown(manifest)
	if attachment == "" && !engineeringDrawingHasMarkdownObject(manifest) {
		return nil, nil
	}
	mdFileID := ""
	resp, err := client.Files().UploadBytes(ctx, workspaceID, engineeringDrawingMarkdownFileName(manifest.Source.FileName, manifest.Source.FileID), engineeringDrawingManifestMarkdown(manifest))
	if err != nil {
		return nil, fmt.Errorf("document_visual.parse: upload engineering markdown: %w", err)
	}
	mdFileID = resp.FileID
	doc, err := documentToProto(Document{
		ID:      hashShort("engineering-drawing-attachment", manifest.Source.FileID, manifest.Source.FileName),
		Type:    "text",
		Content: attachment,
		Metadata: map[string]interface{}{
			"file_id":           manifest.Source.FileID,
			"raw_file_id":       manifest.Source.FileID,
			"source_file_id":    manifest.Source.FileID,
			"file_name":         manifest.Source.FileName,
			"source_file_name":  manifest.Source.FileName,
			"source_type":       "pdf",
			"block_type":        "engineering_extraction",
			"parse_method":      "document_visual_industrial_drawing_v1",
			"profile":           manifest.Profile,
			"extraction_source": "engineering_region_extract",
			"md_file_id":        mdFileID,
			"md_file_url":       mdFileID,
		},
	})
	if err != nil {
		return nil, err
	}
	docs, err := appendEngineeringDrawingImageDocuments(nil, manifest, mdFileID)
	if err != nil {
		return nil, err
	}
	return append(docs, doc), nil
}

func engineeringDrawingHasMarkdownObject(manifest documentVisualManifest) bool {
	for _, obj := range manifest.Objects {
		if strings.TrimSpace(obj.ImageFileID) == "" || obj.ExtractionSource != "engineering_page_plan" {
			continue
		}
		return true
	}
	return false
}

func appendEngineeringDrawingImageDocuments(docs []*data.Document, manifest documentVisualManifest, mdFileID string) ([]*data.Document, error) {
	for _, obj := range manifest.Objects {
		if strings.TrimSpace(obj.ImageFileID) == "" {
			continue
		}
		imageMeta := map[string]interface{}{
			"file_id":            manifest.Source.FileID,
			"raw_file_id":        manifest.Source.FileID,
			"source_file_id":     manifest.Source.FileID,
			"file_name":          manifest.Source.FileName,
			"source_file_name":   manifest.Source.FileName,
			"source_type":        "pdf",
			"block_type":         "image",
			"parse_method":       "document_visual_industrial_drawing_v1",
			"profile":            manifest.Profile,
			"extraction_source":  "engineering_page_plan",
			"object_id":          obj.ObjectID,
			"object_kind":        obj.ObjectKind,
			"page_num":           obj.PageNumber,
			"bbox":               floatSliceMetadataValue(obj.BBox),
			"ocr_bbox":           floatSliceMetadataValue(obj.OCRBBox),
			"image_url":          obj.ImageFileID,
			"s3_image_url":       obj.ImageFileID,
			"image_file_id":      obj.ImageFileID,
			"page_image_file_id": obj.PageImageFileID,
			"caption":            obj.Caption,
			"md_file_id":         mdFileID,
			"md_file_url":        mdFileID,
		}
		if obj.ExtractionFailureReason != "" {
			imageMeta["extraction_failure_reason"] = obj.ExtractionFailureReason
		}
		if len(obj.ExtractionWarnings) > 0 {
			imageMeta["extraction_warnings"] = stringSliceMetadataValue(obj.ExtractionWarnings)
		}
		for key, value := range documentVisualTableMetadataMap(obj) {
			imageMeta[key] = value
		}
		doc, err := documentToProto(Document{
			ID:       obj.ObjectID,
			Type:     "image",
			Content:  "",
			Metadata: imageMeta,
		})
		if err != nil {
			return nil, fmt.Errorf("document_visual.parse: build engineering image document %s: %w", obj.ObjectID, err)
		}
		docs = append(docs, doc)
		if obj.Table == nil {
			continue
		}
		tableMeta := map[string]interface{}{
			"file_id":            manifest.Source.FileID,
			"raw_file_id":        manifest.Source.FileID,
			"source_file_id":     manifest.Source.FileID,
			"file_name":          manifest.Source.FileName,
			"source_file_name":   manifest.Source.FileName,
			"source_type":        "pdf",
			"block_type":         "table",
			"parse_method":       "document_visual_industrial_drawing_v1",
			"profile":            manifest.Profile,
			"extraction_source":  obj.Table.TableSource,
			"object_id":          obj.ObjectID,
			"object_kind":        obj.ObjectKind,
			"page_num":           obj.PageNumber,
			"bbox":               floatSliceMetadataValue(obj.BBox),
			"image_file_id":      obj.ImageFileID,
			"page_image_file_id": obj.PageImageFileID,
			"caption":            obj.Caption,
			"md_file_id":         mdFileID,
			"md_file_url":        mdFileID,
		}
		for key, value := range documentVisualTableMetadataMap(obj) {
			tableMeta[key] = value
		}
		tableDoc, err := documentToProto(Document{
			ID:       obj.ObjectID + "-table",
			Type:     "table",
			Content:  obj.Text,
			Metadata: tableMeta,
		})
		if err != nil {
			return nil, fmt.Errorf("document_visual.parse: build engineering table document %s: %w", obj.ObjectID, err)
		}
		docs = append(docs, tableDoc)
	}
	return docs, nil
}

func documentVisualTableIsTrusted(obj documentVisualObject) bool {
	return obj.Table != nil && obj.Table.TableStatus == engineeringTableStatusExtracted && obj.Text != ""
}

func documentVisualTableIsUnavailable(obj documentVisualObject) bool {
	return obj.Table != nil && obj.Table.TableStatus == engineeringTableStatusUnavailable
}

func documentVisualTableMetadataMap(obj documentVisualObject) map[string]interface{} {
	if obj.Table == nil {
		return map[string]interface{}{}
	}
	meta := map[string]interface{}{
		"table_status":           obj.Table.TableStatus,
		"table_source":           obj.Table.TableSource,
		"html_validation_status": obj.Table.HTMLValidationStatus,
		"candidate_source":       obj.Table.CandidateSource,
		"page_plan_match":        obj.Table.PagePlanMatch,
	}
	if obj.Table.PagePlanRegionID != "" {
		meta["page_plan_region_id"] = obj.Table.PagePlanRegionID
	}
	if obj.Table.PagePlanRegionKind != "" {
		meta["page_plan_region_kind"] = obj.Table.PagePlanRegionKind
	}
	if len(obj.Table.PagePlanBBox) == 4 {
		meta["page_plan_bbox"] = floatSliceMetadataValue(obj.Table.PagePlanBBox)
	}
	if obj.Table.PagePlanBBoxCoordinate != "" {
		meta["page_plan_bbox_coordinate"] = obj.Table.PagePlanBBoxCoordinate
	}
	if obj.Table.FailureStage != "" {
		meta["failure_stage"] = obj.Table.FailureStage
	}
	if obj.Table.TableUnavailableReason != "" {
		meta["table_unavailable_reason"] = obj.Table.TableUnavailableReason
	}
	return meta
}

func engineeringDrawingAttachmentMarkdown(manifest documentVisualManifest) string {
	return engineeringDrawingAttachmentMarkdownFiltered(manifest, func(documentVisualEntity) bool { return true })
}

func engineeringDrawingAttachmentMarkdownFiltered(manifest documentVisualManifest, include func(documentVisualEntity) bool) string {
	lines := []string{"### OCR", ""}
	count := 0
	for _, entity := range manifest.Entities {
		if getString(entity.Metadata, "extraction_source") != "engineering_region_extract" {
			continue
		}
		if entity.Value == "" {
			continue
		}
		if include != nil && !include(entity) {
			continue
		}
		lines = append(lines, engineeringDrawingEntityMarkdownLine(entity))
		count++
	}
	if count == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func engineeringDrawingEntityMarkdownLine(entity documentVisualEntity) string {
	kind := firstNonEmptyString(entity.Type, getString(entity.Metadata, "kind"))
	view := getString(entity.Metadata, "view_name")
	extractionSource := getString(entity.Metadata, "extraction_source")
	source := getString(entity.Metadata, "source")
	line := fmt.Sprintf("- %s", entity.Value)
	details := make([]string, 0, 5)
	if kind != "" {
		details = append(details, "kind="+kind)
	}
	if entity.PageNumber > 0 {
		details = append(details, fmt.Sprintf("page=%d", entity.PageNumber))
	}
	if entity.ObjectID != "" {
		details = append(details, "object_id="+entity.ObjectID)
	}
	if view != "" {
		details = append(details, "view="+view)
	}
	if extractionSource != "" {
		details = append(details, "extraction_source="+extractionSource)
	}
	if source != "" {
		details = append(details, "source="+source)
	}
	if len(details) > 0 {
		line += " (" + strings.Join(details, ", ") + ")"
	}
	return line
}

func engineeringDrawingManifestMarkdown(manifest documentVisualManifest) []byte {
	linesByObject := engineeringDrawingEntityLinesByObject(manifest)
	sections := make([]string, 0, len(manifest.Objects)+1)
	for _, obj := range manifest.Objects {
		objectLines := linesByObject[obj.ObjectID]
		if strings.TrimSpace(obj.ImageFileID) == "" || obj.ExtractionSource != "engineering_page_plan" {
			continue
		}
		sections = append(sections, engineeringDrawingObjectMarkdownSection(obj, objectLines.Lines))
	}
	if len(sections) == 0 {
		return []byte(engineeringDrawingAttachmentMarkdown(manifest))
	}
	return []byte(strings.Join(sections, "\n\n") + "\n")
}

func engineeringDrawingObjectMarkdownSection(obj documentVisualObject, lines []string) string {
	section := []string{}
	if obj.Caption != "" {
		section = append(section, "### Caption", "", obj.Caption, "")
	}
	section = append(section, fmt.Sprintf("![](%s)", obj.ImageFileID), "")
	if obj.Table != nil {
		if documentVisualTableIsTrusted(obj) {
			section = append(section, "### Table", "", obj.Text, "")
		} else {
			section = append(section, "### Table", "", fmt.Sprintf("table_status=%s", obj.Table.TableStatus), "")
		}
		return strings.Join(section, "\n")
	}
	section = append(section, "### OCR", "")
	section = append(section, lines...)
	return strings.Join(section, "\n")
}

type engineeringMarkdownObjectLines struct {
	Lines []string
}

func engineeringDrawingEntityLinesByObject(manifest documentVisualManifest) map[string]engineeringMarkdownObjectLines {
	out := map[string][]string{}
	for _, entity := range manifest.Entities {
		if getString(entity.Metadata, "extraction_source") != "engineering_region_extract" {
			continue
		}
		if entity.Value == "" || entity.ObjectID == "" {
			continue
		}
		out[entity.ObjectID] = append(out[entity.ObjectID], engineeringDrawingEntityMarkdownLine(entity))
	}
	mapped := map[string]engineeringMarkdownObjectLines{}
	for objectID, lines := range out {
		mapped[objectID] = engineeringMarkdownObjectLines{Lines: lines}
	}
	return mapped
}

func engineeringDrawingMarkdownFileName(sourceFileName, sourceFileID string) string {
	stem := strings.TrimSuffix(firstNonEmptyString(path.Base(sourceFileName), sourceFileID, "document"), path.Ext(sourceFileName))
	return fmt.Sprintf("%s.engineering.md", stem)
}
