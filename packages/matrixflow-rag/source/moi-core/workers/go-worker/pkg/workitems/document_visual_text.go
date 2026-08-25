package workitems

import (
	"fmt"
	"strings"
)

func buildDocumentVisualTextIndexDocuments(manifest documentVisualManifest) ([]map[string]interface{}, error) {
	if manifest.Source.FileID == "" {
		return nil, fmt.Errorf("document_visual.index.text: manifest.source.file_id is required")
	}
	docContext := documentVisualDocumentContext(manifest)
	docs := []map[string]interface{}{
		documentVisualIndexDocument("document", documentVisualIndexID(manifest.Source.FileID, "document", "document"), docContext, map[string]interface{}{
			"scope":            "document",
			"level":            "doc",
			"file_id":          manifest.Source.FileID,
			"file_name":        manifest.Source.FileName,
			"source_file_id":   manifest.Source.FileID,
			"source_file_name": manifest.Source.FileName,
			"profile":          manifest.Profile,
		}),
	}
	for _, page := range manifest.Pages {
		content := documentVisualPageIndexText(manifest, page)
		meta := map[string]interface{}{
			"scope":              "page",
			"level":              "chunk",
			"file_id":            manifest.Source.FileID,
			"file_name":          manifest.Source.FileName,
			"object_id":          documentVisualPageObjectID(manifest.Source.FileID, page.PageNumber),
			"object_kind":        "page",
			"source_file_id":     manifest.Source.FileID,
			"source_file_name":   manifest.Source.FileName,
			"page_number":        page.PageNumber,
			"image_file_id":      page.PageImageFileID,
			"page_image_file_id": page.PageImageFileID,
			"bbox":               page.BBox,
			"profile":            manifest.Profile,
		}
		docs = append(docs, documentVisualIndexDocument("page", documentVisualIndexID(manifest.Source.FileID, "page", fmt.Sprint(page.PageNumber)), content, meta))
	}
	for _, obj := range manifest.Objects {
		if !documentVisualObjectTextIndexable(obj) {
			continue
		}
		content := documentVisualObjectIndexText(obj)
		meta := map[string]interface{}{
			"scope":              "visual_object",
			"level":              "chunk",
			"file_id":            manifest.Source.FileID,
			"file_name":          manifest.Source.FileName,
			"object_id":          obj.ObjectID,
			"object_kind":        obj.ObjectKind,
			"source_file_id":     manifest.Source.FileID,
			"source_file_name":   manifest.Source.FileName,
			"page_number":        obj.PageNumber,
			"image_file_id":      obj.ImageFileID,
			"page_image_file_id": obj.PageImageFileID,
			"bbox":               obj.BBox,
			"profile":            manifest.Profile,
		}
		for key, value := range documentVisualTableMetadataMap(obj) {
			meta[key] = value
		}
		addDocumentVisualObjectSemanticMetadata(meta, obj)
		docs = append(docs, documentVisualIndexDocument("visual_object", documentVisualIndexID(manifest.Source.FileID, "object", obj.ObjectID), content, meta))
	}
	for i, entity := range manifest.Entities {
		content := joinNonEmpty([]string{entity.Type, entity.Value, entity.Evidence}, "\n")
		meta := map[string]interface{}{
			"scope":            "entity",
			"level":            "chunk",
			"file_id":          manifest.Source.FileID,
			"file_name":        manifest.Source.FileName,
			"source_file_id":   manifest.Source.FileID,
			"source_file_name": manifest.Source.FileName,
			"page_number":      entity.PageNumber,
			"object_id":        entity.ObjectID,
			"profile":          manifest.Profile,
		}
		docs = append(docs, documentVisualIndexDocument("entity", documentVisualIndexID(manifest.Source.FileID, "entity", firstNonEmptyString(entity.EntityID, fmt.Sprint(i))), content, meta))
	}
	return docs, nil
}

func documentVisualPageIndexText(manifest documentVisualManifest, page documentVisualPage) string {
	return joinNonEmpty([]string{
		manifest.Source.FileName,
		documentVisualEntityTextForPage(manifest.Entities, page.PageNumber),
		page.Text,
	}, "\n")
}

func documentVisualEntityTextForPage(entities []documentVisualEntity, pageNumber int) string {
	pageEntities := make([]documentVisualEntity, 0)
	for _, entity := range entities {
		if entity.PageNumber == 0 || entity.PageNumber == pageNumber {
			pageEntities = append(pageEntities, entity)
		}
	}
	return documentVisualEntityText(pageEntities)
}

func addDocumentVisualObjectSemanticMetadata(meta map[string]interface{}, obj documentVisualObject) {
	addDocumentVisualSemanticMetadata(meta, imageSemantics{
		OCR:              obj.OCR,
		FigureCaption:    obj.FigureCaption,
		GeneratedCaption: obj.Caption,
	}, obj.FigureNo, obj.CaptionBlockUUID, obj.SourceBlockID)
}

func addDocumentVisualSemanticMetadata(
	meta map[string]interface{},
	semantics imageSemantics,
	figureNo, captionBlockUUID, sourceBlockID string,
) {
	if meta == nil {
		return
	}
	for key, value := range map[string]string{
		"ocr_text":           semantics.OCR,
		"caption":            semantics.GeneratedCaption,
		"figure_caption":     semantics.FigureCaption,
		"figure_no":          figureNo,
		"caption_block_uuid": captionBlockUUID,
		"source_block_id":    sourceBlockID,
	} {
		if strings.TrimSpace(value) != "" {
			meta[key] = value
		}
	}
}

func documentVisualObjectIndexText(obj documentVisualObject) string {
	if context := strings.TrimSpace(obj.Context); context != "" {
		return context
	}
	return documentVisualObjectLocalContext(obj)
}

func documentVisualObjectTextIndexable(obj documentVisualObject) bool {
	if documentVisualTableIsUnavailable(obj) {
		return false
	}
	if stringSliceContains(obj.ExtractionWarnings, engineeringRegionExtractFailedWarning) {
		return false
	}
	return true
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
