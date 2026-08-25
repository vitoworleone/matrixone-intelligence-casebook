package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	agentruntimev2 "github.com/matrixflow/moi-core/agent-runtime-v2"
	xhtml "golang.org/x/net/html"
)

type Options struct {
	Registry     *Registry
	Scope        WorkspaceScope
	QueryVisuals []QueryVisualRef
}

func Tools(opts Options) ([]agentruntimev2.Tool, error) {
	if opts.Registry == nil {
		return nil, fmt.Errorf("knowledge tools require registry")
	}
	out := make([]agentruntimev2.Tool, 0, 8)
	if opts.Registry.FindRAGFiles != nil {
		out = append(out, NewFindRAGFilesTool(opts))
	}
	if opts.Registry.SearchRAGChunks != nil {
		out = append(out, NewSearchRAGChunksTool(opts))
	}
	if opts.Registry.SearchVisualImage != nil {
		out = append(out, NewSearchVisualImageTool(opts))
	}
	if opts.Registry.ReadParsedMarkdown != nil {
		out = append(out, NewReadParsedMarkdownTool(opts))
	}
	if opts.Registry.SearchParsedMarkdown != nil {
		out = append(out, NewSearchParsedMarkdownTool(opts))
	}
	if opts.Registry.DescribeSchema != nil {
		out = append(out, NewDescribeSchemaTool(opts))
	}
	if opts.Registry.QuerySQL != nil {
		out = append(out, NewQuerySQLTool(opts))
	}
	if opts.Registry.UpsertKnowledgeTable != nil {
		out = append(out, NewUpsertKnowledgeTableTool(opts))
	}
	out = append(out, NewSelectFinalSourcesTool())
	out = append(out, NewSubmitFinalAnswerTool())
	return out, nil
}

type searchVisualImageParams struct {
	QueryText      string `json:"query_text,omitempty"`
	QueryVisual    int    `json:"query_visual,omitempty"`
	RankingProfile string `json:"ranking_profile,omitempty"`
	TopK           int    `json:"top_k,omitempty"`
}

func NewSearchVisualImageTool(opts Options) agentruntimev2.Tool {
	return agentruntimev2.NewTool(ToolNameSearchVisualImage, SearchVisualImageDescription, schema(SearchVisualImageSchema), nil,
		agentruntimev2.WithToolInvocationFunc(func(ctx context.Context, invocation agentruntimev2.ToolInvocation, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			params, err := decodeParams[searchVisualImageParams](ToolNameSearchVisualImage, raw)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			queryText := strings.TrimSpace(params.QueryText)
			queryVisualFileID := ""
			index := params.QueryVisual
			if len(opts.QueryVisuals) > 0 {
				if index <= 0 {
					index = 1
				}
				if index > 0 && index > len(opts.QueryVisuals) {
					return nil, agentruntimev2.RespondToModelError(fmt.Sprintf("%s: query_visual %d exceeds available visual input count %d", ToolNameSearchVisualImage, index, len(opts.QueryVisuals)))
				}
				if index > 0 {
					queryVisual := opts.QueryVisuals[index-1]
					queryVisualFileID = strings.TrimSpace(queryVisual.FileID)
					if queryVisualFileID == "" {
						return nil, agentruntimev2.RespondToModelError(ToolNameSearchVisualImage + ": selected query visual input has empty file_id")
					}
				}
			} else if index > 0 && queryText == "" {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchVisualImage + ": query_visual was provided but no query visual is available in the current message")
			}
			if queryText == "" && queryVisualFileID == "" {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchVisualImage + ": query_text or query_visual is required")
			}
			rankingProfile := strings.TrimSpace(params.RankingProfile)
			if rankingProfile != "" &&
				rankingProfile != VisualSearchRankingProfileVisualObjectFirst &&
				rankingProfile != VisualSearchRankingProfileTextRegionFirst {
				return nil, agentruntimev2.RespondToModelError(fmt.Sprintf("%s: unsupported ranking_profile %q", ToolNameSearchVisualImage, rankingProfile))
			}
			if rankingProfile == "" && queryVisualFileID != "" {
				rankingProfile = VisualSearchRankingProfileVisualObjectFirst
			}
			if opts.Registry == nil || opts.Registry.SearchVisualImage == nil {
				return nil, fmt.Errorf("%s: tool is not configured", ToolNameSearchVisualImage)
			}
			ctx = ContextWithScope(ctx, opts.Scope)
			result, err := opts.Registry.SearchVisualImage.Execute(ctx, SearchVisualImageRequest{
				Scope:             opts.Scope,
				QueryText:         queryText,
				QueryVisualFileID: queryVisualFileID,
				RankingProfile:    rankingProfile,
				TopK:              normalizeTopK(params.TopK),
			})
			if err != nil {
				return nil, fmt.Errorf("%s: search visual image: %w", ToolNameSearchVisualImage, err)
			}
			if result == nil {
				return nil, fmt.Errorf("%s: visual searcher returned nil result", ToolNameSearchVisualImage)
			}
			artifactID := invocationArtifactID(invocation, "visual_search")
			result.ArtifactID = artifactID
			if rc := RunContextFrom(ctx); rc != nil {
				rc.RecordVisualSearchArtifact(artifactID, *result)
			}
			metadata := map[string]any{
				"artifact_id": artifactID,
			}
			if queryText != "" {
				metadata["query_text"] = queryText
			}
			if queryVisualFileID != "" {
				metadata["query_visual_file_id"] = queryVisualFileID
			}
			if rankingProfile != "" {
				metadata["ranking_profile"] = rankingProfile
			}
			out, err := marshalToolResultWithMetadata(result, metadata)
			if err != nil {
				return nil, err
			}
			out.Artifacts = append(out.Artifacts, visualSearchArtifact(artifactID, result, metadata))
			return out, nil
		}))
}

type findRAGFilesParams struct {
	Query string `json:"query"`
	TopK  int    `json:"top_k,omitempty"`
}

func NewFindRAGFilesTool(opts Options) agentruntimev2.Tool {
	return agentruntimev2.NewTool(ToolNameFindRAGFiles, FindRAGFilesDescription, schema(FindRAGFilesSchema), nil,
		agentruntimev2.WithToolInvocationFunc(func(ctx context.Context, _ agentruntimev2.ToolInvocation, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			params, err := decodeParams[findRAGFilesParams](ToolNameFindRAGFiles, raw)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			if strings.TrimSpace(params.Query) == "" {
				return nil, agentruntimev2.RespondToModelError(ToolNameFindRAGFiles + ": query is required")
			}
			if opts.Registry == nil || opts.Registry.FindRAGFiles == nil {
				return nil, fmt.Errorf("%s: tool is not configured", ToolNameFindRAGFiles)
			}
			ctx = ContextWithScope(ctx, opts.Scope)
			result, err := opts.Registry.FindRAGFiles.Execute(ctx, FindRAGFilesRequest{
				Scope:    opts.Scope,
				Query:    params.Query,
				MaxFiles: normalizeTopK(params.TopK),
			})
			if err != nil {
				return nil, fmt.Errorf("%s: find rag files: %w", ToolNameFindRAGFiles, err)
			}
			if result == nil {
				return nil, fmt.Errorf("%s: rag searcher returned nil file result", ToolNameFindRAGFiles)
			}
			return marshalToolResult(result)
		}))
}

type searchRAGChunksParams struct {
	Keywords []string `json:"keywords,omitempty"`
	VolumeID string   `json:"volume_id,omitempty"`
	FileIDs  []string `json:"file_ids,omitempty"`
	MaxHits  int      `json:"max_hits,omitempty"`
	MaxRows  int      `json:"max_rows,omitempty"`
	Before   int      `json:"before,omitempty"`
	After    int      `json:"after,omitempty"`
}

func NewSearchRAGChunksTool(opts Options) agentruntimev2.Tool {
	return agentruntimev2.NewTool(ToolNameSearchRAGChunks, SearchRAGChunksDescription, schema(SearchRAGChunksSchema), nil,
		agentruntimev2.WithToolInvocationFunc(func(ctx context.Context, invocation agentruntimev2.ToolInvocation, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			params, err := decodeParams[searchRAGChunksParams](ToolNameSearchRAGChunks, raw)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			keywards := compactStrings(params.Keywords)
			if len(keywards) == 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchRAGChunks + ": keywords is required")
			}
			if params.MaxHits <= 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchRAGChunks + ": max_hits is required")
			}
			if params.MaxRows < 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchRAGChunks + ": max_rows must be >= 0")
			}
			if params.Before < 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchRAGChunks + ": before must be >= 0")
			}
			if params.After < 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchRAGChunks + ": after must be >= 0")
			}
			if opts.Registry == nil || opts.Registry.SearchRAGChunks == nil {
				return nil, fmt.Errorf("%s: tool is not configured", ToolNameSearchRAGChunks)
			}
			ctx = ContextWithScope(ctx, opts.Scope)
			result, err := opts.Registry.SearchRAGChunks.Execute(ctx, SearchRAGChunksRequest{
				Scope:    opts.Scope,
				Keywards: keywards,
				VolumeID: strings.TrimSpace(params.VolumeID),
				FileIDs:  params.FileIDs,
				MaxHits:  params.MaxHits,
				MaxRows:  params.MaxRows,
				Before:   params.Before,
				After:    params.After,
			})
			if err != nil {
				return nil, fmt.Errorf("%s: search rag chunks: %w", ToolNameSearchRAGChunks, err)
			}
			if result == nil {
				return nil, fmt.Errorf("%s: rag searcher returned nil chunk result", ToolNameSearchRAGChunks)
			}
			artifactID := invocationArtifactID(invocation, "rag_chunks")
			result.ArtifactID = artifactID
			if rc := RunContextFrom(ctx); rc != nil {
				rc.RecordRAGChunksArtifact(artifactID, *result)
			}
			out, err := marshalToolResult(result)
			if err != nil {
				return nil, err
			}
			out.ModelView = searchRAGChunksToolModelView(result)
			out.Artifacts = append(out.Artifacts, ragChunksArtifact(artifactID, result))
			return out, nil
		}))
}

func searchRAGChunksToolModelView(result *SearchRAGChunksResponse) *agentruntimev2.ToolResultModelView {
	if result == nil {
		return nil
	}
	view := agentruntimev2.NewToolResultModelView("rag_chunks").
		WithArtifactIDs(result.ArtifactID, result.ArtifactID).
		WithBudget(agentruntimev2.ToolResultModelViewBudget{
			MaxItems:             ragChunksModelViewMaxItems,
			MaxPreviewRunes:      ragChunksModelViewPreviewRunes,
			MaxTotalPreviewRunes: ragChunksModelViewTotalPreviewRunes,
		}).
		AddSummaryInt("row_count", result.RowCount).
		AddSummaryInt("chunk_count", len(result.Chunks))
	if result.ExpandedGroups > 0 {
		view.AddSummaryInt("expanded_groups", result.ExpandedGroups)
	}
	if result.EmbeddingModel != "" {
		view.AddSummaryString("embedding_model", result.EmbeddingModel)
	}
	if len(result.Keywards) > 0 {
		view.AddSummaryStrings("keywords", result.Keywards)
	}
	if len(result.Routes) > 0 {
		view.AddSummaryStrings("routes", result.Routes)
	}
	for _, chunk := range ragChunksModelViewOrder(result.Chunks) {
		item := agentruntimev2.NewToolResultModelViewItem(chunk.Content).
			WithID(chunk.ChunkID).
			WithRef("chunk_id", chunk.ChunkID).
			WithRef("file_id", chunk.FileID).
			WithRef("markdown_file_id", chunk.MarkdownFileID).
			WithSource(chunk.FileID, chunk.FileName).
			WithPageNumber(chunk.PageNumber).
			WithGroupID(chunk.EvidenceGroupID).
			WithScore(chunk.Score).
			WithRoutes(chunk.Routes).
			WithTags(chunk.SourceTags)
		if ragChunkModelViewKeepsPrefix(chunk) {
			if preview := queryAwareRAGTablePreview(chunk.Content, result.Keywards, ragChunksModelViewPreviewRunes); preview != "" {
				item = item.WithPreview(preview)
			}
		} else {
			if preview := queryAwareRAGExcerpt(chunk.Content, result.Keywards, ragChunksModelViewPreviewRunes); preview != chunk.Content {
				item = item.WithPreview(preview)
			}
		}
		view.AddItem(item)
	}
	return view
}

func ragChunkModelViewKeepsPrefix(chunk RAGChunkHit) bool {
	if strings.EqualFold(strings.TrimSpace(chunk.ChunkType), "table") {
		return true
	}
	content := strings.ToLower(chunk.Content)
	return strings.Contains(content, "<table") || strings.Contains(content, "<tr") || strings.Contains(content, "<td")
}

type ragModelViewTableCell struct {
	text      string
	inherited bool
}

type ragModelViewTableRow struct {
	cells []ragModelViewTableCell
}

type ragModelViewTablePreviewAtom struct {
	text    string
	matched bool
}

type ragModelViewActiveTableCell struct {
	cell      ragModelViewTableCell
	column    int
	remaining int
}

func queryAwareRAGTablePreview(content string, keywards []string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	rows := ragModelViewTableRows(content)
	bestScore := 0
	bestRows := make([]ragModelViewTableRow, 0, 1)
	for _, row := range rows {
		score := ragModelViewTableRowMatchScore(row, keywards)
		if score > bestScore {
			bestScore = score
			bestRows = append(bestRows[:0], row)
			continue
		}
		if score > 0 && score == bestScore {
			bestRows = append(bestRows, row)
		}
	}
	if len(bestRows) == 0 {
		return ""
	}
	if len(bestRows) == 1 {
		return ragModelViewTableRowPreview(bestRows[0], keywards, maxRunes)
	}
	return ragModelViewTableRowsPreview(bestRows, keywards, maxRunes)
}

func ragModelViewTableRowsPreview(rows []ragModelViewTableRow, keywards []string, maxRunes int) string {
	if len(rows) == 0 || maxRunes <= 0 {
		return ""
	}
	const separator = "\n"
	separatorRunes := len([]rune(separator))
	textBudget := maxRunes - separatorRunes*(len(rows)-1)
	if textBudget <= 0 {
		return ragModelViewTableRowPreview(rows[0], keywards, maxRunes)
	}

	previews := make([]string, 0, len(rows))
	remainingBudget := textBudget
	for index, row := range rows {
		remainingRows := len(rows) - index
		rowBudget := remainingBudget / remainingRows
		if rowBudget <= 0 {
			break
		}
		preview := ragModelViewTiedTableRowPreview(row, keywards, rowBudget)
		if preview == "" {
			continue
		}
		previews = append(previews, preview)
		remainingBudget -= len([]rune(preview))
	}
	return strings.Join(previews, separator)
}

func ragModelViewTableRows(content string) []ragModelViewTableRow {
	root, err := xhtml.Parse(strings.NewReader("<table>" + content + "</table>"))
	if err != nil {
		return nil
	}
	trNodes := make([]*xhtml.Node, 0)
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node == nil {
			return
		}
		if node.Type == xhtml.ElementNode && strings.EqualFold(node.Data, "tr") {
			trNodes = append(trNodes, node)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)

	rows := make([]ragModelViewTableRow, 0, len(trNodes))
	active := make([]ragModelViewActiveTableCell, 0)
	for _, tr := range trNodes {
		cellsByColumn := make(map[int]ragModelViewTableCell)
		nextActive := make([]ragModelViewActiveTableCell, 0, len(active))
		for _, item := range active {
			inherited := item.cell
			inherited.inherited = true
			cellsByColumn[item.column] = inherited
			item.remaining--
			if item.remaining > 0 {
				nextActive = append(nextActive, item)
			}
		}
		active = nextActive
		column := 0
		for child := tr.FirstChild; child != nil; child = child.NextSibling {
			if child.Type != xhtml.ElementNode || (!strings.EqualFold(child.Data, "td") && !strings.EqualFold(child.Data, "th")) {
				continue
			}
			for {
				if _, occupied := cellsByColumn[column]; !occupied {
					break
				}
				column++
			}
			cell := ragModelViewTableCell{text: ragModelViewHTMLText(child)}
			cellsByColumn[column] = cell
			if rowspan := ragModelViewHTMLAttrPositiveInt(child, "rowspan"); rowspan > 1 {
				active = append(active, ragModelViewActiveTableCell{cell: cell, column: column, remaining: rowspan - 1})
			}
			column++
		}
		if len(cellsByColumn) > 0 {
			columns := make([]int, 0, len(cellsByColumn))
			for column := range cellsByColumn {
				columns = append(columns, column)
			}
			sort.Ints(columns)
			cells := make([]ragModelViewTableCell, 0, len(columns))
			for _, column := range columns {
				cells = append(cells, cellsByColumn[column])
			}
			rows = append(rows, ragModelViewTableRow{cells: cells})
		}
	}
	return rows
}

func ragModelViewTableRowMatchScore(row ragModelViewTableRow, keywards []string) int {
	score, _ := ragModelViewTableRowMatches(row, keywards)
	return score
}

func ragModelViewTableRowMatches(row ragModelViewTableRow, keywards []string) (int, []bool) {
	matchedCells := make([]bool, len(row.cells))
	parts := make([]string, len(row.cells))
	cellStarts := make([]int, len(row.cells))
	cellEnds := make([]int, len(row.cells))
	textLength := 0
	for index, cell := range row.cells {
		if index > 0 {
			textLength++
		}
		cellStarts[index] = textLength
		parts[index] = strings.ToLower(cell.text)
		textLength += len(parts[index])
		cellEnds[index] = textLength
	}
	text := strings.Join(parts, " ")
	score := 0
	for _, keyward := range keywards {
		keyward = strings.ToLower(strings.TrimSpace(keyward))
		if keyward == "" {
			continue
		}
		if !strings.Contains(text, keyward) {
			continue
		}
		score += 2
		for searchStart := 0; searchStart < len(text); {
			offset := strings.Index(text[searchStart:], keyward)
			if offset < 0 {
				break
			}
			matchStart := searchStart + offset
			matchEnd := matchStart + len(keyward)
			for index := range row.cells {
				if matchStart < cellEnds[index] && cellStarts[index] < matchEnd {
					matchedCells[index] = true
				}
			}
			searchStart = matchStart + 1
		}
	}
	return score, matchedCells
}

func ragModelViewHTMLText(node *xhtml.Node) string {
	parts := make([]string, 0)
	var walk func(*xhtml.Node)
	walk = func(current *xhtml.Node) {
		if current == nil {
			return
		}
		if current.Type == xhtml.TextNode {
			if text := strings.TrimSpace(current.Data); text != "" {
				parts = append(parts, text)
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.Join(parts, " ")
}

func ragModelViewHTMLAttrPositiveInt(node *xhtml.Node, name string) int {
	for _, attr := range node.Attr {
		if !strings.EqualFold(attr.Key, name) {
			continue
		}
		value, err := strconv.Atoi(strings.TrimSpace(attr.Val))
		if err != nil || value <= 0 {
			return 0
		}
		return value
	}
	return 0
}

func ragModelViewTableRowPreview(row ragModelViewTableRow, keywards []string, maxRunes int) string {
	cells := row.cells
	if len(cells) == 0 || maxRunes <= 0 {
		return ""
	}
	_, matchedCells := ragModelViewTableRowMatches(row, keywards)
	include := make([]bool, len(cells))
	include[0] = true
	if len(cells) > 1 {
		include[1] = true
	}
	include[len(cells)-1] = true
	for index, cell := range cells {
		if cell.inherited || matchedCells[index] {
			include[index] = true
		}
	}
	selected := make([]ragModelViewTableCell, 0, len(cells))
	selectedMatches := make([]bool, 0, len(cells))
	for index, cell := range cells {
		if include[index] {
			selected = append(selected, cell)
			selectedMatches = append(selectedMatches, matchedCells[index])
		}
	}
	return ragModelViewTableCellsPreview(selected, selectedMatches, keywards, maxRunes)
}

func ragModelViewTiedTableRowPreview(row ragModelViewTableRow, keywards []string, maxRunes int) string {
	cells := row.cells
	if len(cells) == 0 || maxRunes <= 0 {
		return ""
	}
	_, matchedCells := ragModelViewTableRowMatches(row, keywards)
	used := make([]bool, len(cells))

	conditionIndexes := []int{len(cells) - 1}
	used[len(cells)-1] = true

	identityIndexes := make([]int, 0, len(cells))
	for index, cell := range cells {
		if (index < 2 || cell.inherited) && !used[index] {
			identityIndexes = append(identityIndexes, index)
			used[index] = true
		}
	}

	matchedIndexes := make([]int, 0, len(cells))
	for index, matched := range matchedCells {
		if matched && !used[index] {
			matchedIndexes = append(matchedIndexes, index)
			used[index] = true
		}
	}

	atoms := make([]ragModelViewTablePreviewAtom, 0, 3)
	appendAtom := func(indexes []int, separator string) {
		if len(indexes) == 0 {
			return
		}
		parts := make([]string, 0, len(indexes))
		matched := false
		for _, index := range indexes {
			parts = append(parts, cells[index].text)
			matched = matched || matchedCells[index]
		}
		atoms = append(atoms, ragModelViewTablePreviewAtom{
			text:    strings.Join(parts, separator),
			matched: matched,
		})
	}
	appendAtom(identityIndexes, " / ")
	appendAtom(matchedIndexes, " ")
	appendAtom(conditionIndexes, "")
	return ragModelViewTableAtomsPreview(atoms, keywards, maxRunes)
}

func ragModelViewTableAtomsPreview(atoms []ragModelViewTablePreviewAtom, keywards []string, maxRunes int) string {
	if len(atoms) == 0 || maxRunes <= 0 {
		return ""
	}
	const fullSeparator = " | "
	fullParts := make([]string, 0, len(atoms))
	for _, atom := range atoms {
		fullParts = append(fullParts, atom.text)
	}
	full := strings.Join(fullParts, fullSeparator)
	if len([]rune(full)) <= maxRunes {
		return full
	}

	const compactSeparator = "|"
	textBudget := maxRunes - len([]rune(compactSeparator))*(len(atoms)-1)
	if textBudget < len(atoms) {
		return string([]rune(strings.Join(fullParts, compactSeparator))[:maxRunes])
	}

	lengths := make([]int, len(atoms))
	budgets := make([]int, len(atoms))
	for index, atom := range atoms {
		lengths[index] = len([]rune(atom.text))
		budgets[index] = 1
	}
	for remaining := textBudget - len(atoms); remaining > 0; {
		allocated := false
		for index := range atoms {
			if budgets[index] >= lengths[index] {
				continue
			}
			budgets[index]++
			remaining--
			allocated = true
			if remaining == 0 {
				break
			}
		}
		if !allocated {
			break
		}
	}

	previewParts := make([]string, 0, len(atoms))
	for index, atom := range atoms {
		preview := atom.text
		if atom.matched {
			preview = queryAwareRAGExcerpt(preview, keywards, budgets[index])
		}
		previewParts = append(previewParts, ragModelViewPrefixRunes(preview, budgets[index]))
	}
	return strings.Join(previewParts, compactSeparator)
}

func ragModelViewTableCellsPreview(cells []ragModelViewTableCell, matchedCells []bool, keywards []string, maxRunes int) string {
	if len(cells) == 0 || maxRunes <= 0 {
		return ""
	}
	const separator = " | "
	parts := make([]string, 0, len(cells))
	matching := make([]bool, 0, len(cells))
	matchedParts := make([]string, 0, len(cells))
	for index, cell := range cells {
		parts = append(parts, cell.text)
		matched := matchedCells[index]
		matching = append(matching, matched)
		if matched {
			matchedParts = append(matchedParts, cell.text)
		}
	}
	full := strings.Join(parts, separator)
	if len([]rune(full)) <= maxRunes {
		return full
	}

	matchedPreview := strings.Join(matchedParts, separator)
	if len([]rune(matchedPreview)) > maxRunes {
		return queryAwareRAGExcerpt(matchedPreview, keywards, maxRunes)
	}
	separatorRunes := len([]rune(separator))
	textBudget := maxRunes - separatorRunes*(len(parts)-1)
	matchedRunes := len([]rune(strings.Join(matchedParts, "")))
	if textBudget <= matchedRunes {
		return matchedPreview
	}
	contextBudget := textBudget - matchedRunes
	contextCount := len(parts) - len(matchedParts)
	previewParts := make([]string, 0, len(parts))
	for index, part := range parts {
		if matching[index] {
			previewParts = append(previewParts, part)
			continue
		}
		if contextCount == 0 || contextBudget <= 0 {
			continue
		}
		budget := contextBudget / contextCount
		contextCount--
		if budget <= 0 {
			continue
		}
		preview := ragModelViewTruncateRunes(part, budget)
		contextBudget -= len([]rune(preview))
		previewParts = append(previewParts, preview)
	}
	return strings.Join(previewParts, separator)
}

func ragModelViewPrefixRunes(value string, maxRunes int) string {
	runes := []rune(value)
	if maxRunes <= 0 {
		return ""
	}
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func ragModelViewTruncateRunes(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

func queryAwareRAGExcerpt(content string, keywards []string, maxRunes int) string {
	contentRunes := []rune(content)
	if maxRunes <= 0 || len(contentRunes) <= maxRunes {
		return content
	}
	lowerContent := strings.ToLower(content)
	matchStart := -1
	matchRunes := 0
	for _, keyward := range keywards {
		keyward = strings.TrimSpace(keyward)
		if keyward == "" {
			continue
		}
		byteIndex := strings.Index(lowerContent, strings.ToLower(keyward))
		if byteIndex < 0 {
			continue
		}
		matchStart = len([]rune(lowerContent[:byteIndex]))
		matchRunes = len([]rune(keyward))
		break
	}
	if matchStart < 0 {
		return content
	}

	const omissionMarker = "..."
	markerRunes := len([]rune(omissionMarker))
	windowRunes := maxRunes - 2*markerRunes
	if windowRunes <= 0 {
		start := matchStart
		if start+maxRunes > len(contentRunes) {
			start = len(contentRunes) - maxRunes
		}
		return string(contentRunes[start : start+maxRunes])
	}
	if matchRunes > windowRunes {
		matchRunes = windowRunes
	}
	start := matchStart - (windowRunes-matchRunes)/2
	if start < 0 {
		start = 0
	}
	if start+windowRunes > len(contentRunes) {
		start = len(contentRunes) - windowRunes
	}
	end := start + windowRunes
	prefix := ""
	suffix := ""
	if start > 0 {
		prefix = omissionMarker
	}
	if end < len(contentRunes) {
		suffix = omissionMarker
	}
	return prefix + string(contentRunes[start:end]) + suffix
}

const (
	ragChunksModelViewMaxItems          = 20
	ragChunksModelViewPreviewRunes      = 120
	ragChunksModelViewTotalPreviewRunes = 2400
)

func ragChunksModelViewOrder(chunks []RAGChunkHit) []RAGChunkHit {
	if len(chunks) <= 1 {
		return append([]RAGChunkHit(nil), chunks...)
	}
	anchors := make([]RAGChunkHit, 0, len(chunks))
	neighbors := make([]RAGChunkHit, 0, len(chunks))
	for _, chunk := range chunks {
		if chunk.RetrievalAnchorRank > 0 {
			anchors = append(anchors, chunk)
			continue
		}
		neighbors = append(neighbors, chunk)
	}
	if len(anchors) == 0 {
		return ragChunksModelViewBreadthOrder(chunks)
	}
	sort.SliceStable(anchors, func(i, j int) bool {
		if anchors[i].RetrievalAnchorRank != anchors[j].RetrievalAnchorRank {
			return anchors[i].RetrievalAnchorRank < anchors[j].RetrievalAnchorRank
		}
		return ragChunkModelViewStableKey(anchors[i]) < ragChunkModelViewStableKey(anchors[j])
	})
	return append(anchors, ragChunksModelViewBreadthOrder(neighbors)...)
}

func ragChunksModelViewBreadthOrder(chunks []RAGChunkHit) []RAGChunkHit {
	firstByGroup := make([]RAGChunkHit, 0, len(chunks))
	remaining := make([]RAGChunkHit, 0, len(chunks))
	seenGroups := map[string]struct{}{}
	for _, chunk := range chunks {
		groupID := strings.TrimSpace(chunk.EvidenceGroupID)
		if groupID != "" {
			groupID = semanticModelScopedEvidenceKey(chunk.SemanticModelID, groupID)
		}
		if groupID == "" {
			firstByGroup = append(firstByGroup, chunk)
			continue
		}
		if _, ok := seenGroups[groupID]; ok {
			remaining = append(remaining, chunk)
			continue
		}
		seenGroups[groupID] = struct{}{}
		firstByGroup = append(firstByGroup, chunk)
	}
	out := make([]RAGChunkHit, 0, len(chunks))
	out = append(out, firstByGroup...)
	out = append(out, remaining...)
	return out
}

func ragChunkModelViewStableKey(chunk RAGChunkHit) string {
	return strings.Join([]string{
		semanticModelScopedEvidenceKey(chunk.SemanticModelID, ""),
		chunk.FileID,
		chunk.IndexVersion,
		chunk.Level,
		ragChunkModelViewIndexKey(chunk.ParentIndex),
		ragChunkModelViewIndexKey(chunk.ChunkIndex),
		chunk.ChunkID,
	}, "|")
}

func semanticModelScopedEvidenceKey(semanticModelID int64, evidenceKey string) string {
	evidenceKey = strings.TrimSpace(evidenceKey)
	if semanticModelID <= 0 {
		return evidenceKey
	}
	return "semantic_model:" + strconv.FormatInt(semanticModelID, 10) + "\x00" + evidenceKey
}

func ragChunkModelViewIndexKey(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

type readParsedMarkdownParams struct {
	MarkdownFileID string `json:"markdown_file_id"`
	Cursor         int    `json:"cursor,omitempty"`
	LimitChars     int    `json:"limit_chars,omitempty"`
}

func NewReadParsedMarkdownTool(opts Options) agentruntimev2.Tool {
	return agentruntimev2.NewTool(ToolNameReadParsedMarkdown, ReadParsedMarkdownDescription, schema(ReadParsedMarkdownSchema), nil,
		agentruntimev2.WithToolInvocationFunc(func(ctx context.Context, _ agentruntimev2.ToolInvocation, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			params, err := decodeParams[readParsedMarkdownParams](ToolNameReadParsedMarkdown, raw)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			if strings.TrimSpace(params.MarkdownFileID) == "" {
				return nil, agentruntimev2.RespondToModelError(ToolNameReadParsedMarkdown + ": markdown_file_id is required")
			}
			if params.Cursor < 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameReadParsedMarkdown + ": cursor must be >= 0")
			}
			if params.LimitChars < 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameReadParsedMarkdown + ": limit_chars must be >= 0")
			}
			if opts.Registry == nil || opts.Registry.ReadParsedMarkdown == nil {
				return nil, fmt.Errorf("%s: tool is not configured", ToolNameReadParsedMarkdown)
			}
			ctx = ContextWithScope(ctx, opts.Scope)
			result, err := opts.Registry.ReadParsedMarkdown.Execute(ctx, ReadParsedMarkdownRequest{
				Scope:          opts.Scope,
				MarkdownFileID: params.MarkdownFileID,
				Cursor:         params.Cursor,
				LimitChars:     params.LimitChars,
			})
			if err != nil {
				return nil, fmt.Errorf("%s: read parsed markdown: %w", ToolNameReadParsedMarkdown, err)
			}
			if result == nil {
				return nil, fmt.Errorf("%s: rag searcher returned nil markdown result", ToolNameReadParsedMarkdown)
			}
			return marshalToolResult(result)
		}))
}

type searchParsedMarkdownParams struct {
	MarkdownFileID string `json:"markdown_file_id"`
	Query          string `json:"query"`
	MaxMatches     int    `json:"max_matches,omitempty"`
	ContextChars   int    `json:"context_chars,omitempty"`
}

func NewSearchParsedMarkdownTool(opts Options) agentruntimev2.Tool {
	return agentruntimev2.NewTool(ToolNameSearchParsedMarkdown, SearchParsedMarkdownDescription, schema(SearchParsedMarkdownSchema), nil,
		agentruntimev2.WithToolInvocationFunc(func(ctx context.Context, _ agentruntimev2.ToolInvocation, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			params, err := decodeParams[searchParsedMarkdownParams](ToolNameSearchParsedMarkdown, raw)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			if strings.TrimSpace(params.MarkdownFileID) == "" {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchParsedMarkdown + ": markdown_file_id is required")
			}
			if strings.TrimSpace(params.Query) == "" {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchParsedMarkdown + ": query is required")
			}
			if params.MaxMatches < 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchParsedMarkdown + ": max_matches must be >= 0")
			}
			if params.ContextChars < 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameSearchParsedMarkdown + ": context_chars must be >= 0")
			}
			if opts.Registry == nil || opts.Registry.SearchParsedMarkdown == nil {
				return nil, fmt.Errorf("%s: tool is not configured", ToolNameSearchParsedMarkdown)
			}
			ctx = ContextWithScope(ctx, opts.Scope)
			result, err := opts.Registry.SearchParsedMarkdown.Execute(ctx, SearchParsedMarkdownRequest{
				Scope:          opts.Scope,
				MarkdownFileID: params.MarkdownFileID,
				Query:          params.Query,
				MaxMatches:     params.MaxMatches,
				ContextChars:   params.ContextChars,
			})
			if err != nil {
				return nil, fmt.Errorf("%s: search parsed markdown: %w", ToolNameSearchParsedMarkdown, err)
			}
			if result == nil {
				return nil, fmt.Errorf("%s: rag searcher returned nil markdown search result", ToolNameSearchParsedMarkdown)
			}
			return marshalToolResult(result)
		}))
}

type describeSchemaParams struct {
	TableNames     []string `json:"table_names,omitempty"`
	IncludeSamples int      `json:"include_samples,omitempty"`
	MaxDDLChars    int      `json:"max_ddl_chars,omitempty"`
}

func NewDescribeSchemaTool(opts Options) agentruntimev2.Tool {
	return agentruntimev2.NewTool(ToolNameDescribeSchema, DescribeSchemaDescription, schema(DescribeSchemaSchema), nil,
		agentruntimev2.WithToolInvocationFunc(func(ctx context.Context, _ agentruntimev2.ToolInvocation, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			params, err := decodeParams[describeSchemaParams](ToolNameDescribeSchema, raw)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			if params.IncludeSamples < 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameDescribeSchema + ": include_samples must be >= 0")
			}
			if params.MaxDDLChars < 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameDescribeSchema + ": max_ddl_chars must be >= 0")
			}
			if opts.Registry == nil || opts.Registry.DescribeSchema == nil {
				return nil, fmt.Errorf("%s: tool is not configured", ToolNameDescribeSchema)
			}
			ctx = ContextWithScope(ctx, opts.Scope)
			result, err := opts.Registry.DescribeSchema.Execute(ctx, DescribeSchemaRequest{
				Scope:          opts.Scope,
				TableNames:     params.TableNames,
				IncludeSamples: params.IncludeSamples,
				MaxDDLChars:    params.MaxDDLChars,
			})
			if err != nil {
				return nil, fmt.Errorf("%s: describe schema: %w", ToolNameDescribeSchema, err)
			}
			if result == nil {
				return nil, fmt.Errorf("%s: describe schema returned nil result", ToolNameDescribeSchema)
			}
			return marshalToolResultWithMetadata(result, describeSchemaToolMetadata(result))
		}))
}

type querySQLParams struct {
	SQL            string   `json:"sql"`
	MaxRows        int      `json:"max_rows,omitempty"`
	SemanticClaims []string `json:"semantic_claims,omitempty"`
}

func NewQuerySQLTool(opts Options) agentruntimev2.Tool {
	return agentruntimev2.NewTool(ToolNameQuerySQL, QuerySQLDescription, schema(QuerySQLSchema), nil,
		agentruntimev2.WithToolInvocationFunc(func(ctx context.Context, invocation agentruntimev2.ToolInvocation, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			params, err := decodeParams[querySQLParams](ToolNameQuerySQL, raw)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			if strings.TrimSpace(params.SQL) == "" {
				return nil, agentruntimev2.RespondToModelError(ToolNameQuerySQL + ": sql is required")
			}
			if params.MaxRows < 0 {
				return nil, agentruntimev2.RespondToModelError(ToolNameQuerySQL + ": max_rows must be >= 0")
			}
			if opts.Registry == nil || opts.Registry.QuerySQL == nil {
				return nil, fmt.Errorf("%s: tool is not configured", ToolNameQuerySQL)
			}
			ctx = ContextWithScope(ctx, opts.Scope)
			result, err := opts.Registry.QuerySQL.Execute(ctx, QuerySQLRequest{
				Scope:          opts.Scope,
				SQL:            params.SQL,
				MaxRows:        params.MaxRows,
				SemanticClaims: append([]string(nil), params.SemanticClaims...),
			})
			if err != nil {
				return nil, fmt.Errorf("%s: query sql: %w", ToolNameQuerySQL, err)
			}
			if result == nil {
				return nil, fmt.Errorf("%s: query sql returned nil result", ToolNameQuerySQL)
			}
			artifactID := invocationArtifactID(invocation, "sql_result")
			result.ArtifactID = artifactID
			sqlIdx := -1
			if rc := RunContextFrom(ctx); rc != nil {
				sqlIdx = rc.RecordSQLResultArtifact(artifactID, *result)
			}
			if sqlIdx >= 0 {
				result.SQLIdx = intPtr(sqlIdx)
			}
			metadata := querySQLToolMetadata(result)
			metadata = mergeToolMetadata(metadata, sqlResultHandleMetadata(artifactID, sqlIdx))
			out, err := marshalToolResultWithMetadata(result, metadata)
			if err != nil {
				return nil, err
			}
			out.Artifacts = append(out.Artifacts, sqlResultArtifact(artifactID, result))
			return out, nil
		}))
}

type upsertKnowledgeTableParams struct {
	TableName string                       `json:"table_name"`
	Key       map[string]any               `json:"key"`
	Values    map[string]any               `json:"values"`
	Records   []UpsertKnowledgeTableRecord `json:"records"`
}

func NewUpsertKnowledgeTableTool(opts Options) agentruntimev2.Tool {
	return agentruntimev2.NewTool(ToolNameUpsertKnowledgeTable, "Create or update confirmed facts in one table from the currently bound knowledge base. Supply one key/value record or a bounded batch of records with matching column sets. The server validates table and column names and uses one parameterized write.", schema(UpsertKnowledgeTableSchema), nil,
		agentruntimev2.WithToolInvocationFunc(func(ctx context.Context, _ agentruntimev2.ToolInvocation, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			params, err := decodeParams[upsertKnowledgeTableParams](ToolNameUpsertKnowledgeTable, raw)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			if strings.TrimSpace(params.TableName) == "" {
				return nil, agentruntimev2.RespondToModelError(ToolNameUpsertKnowledgeTable + ": table_name is required")
			}
			if len(params.Records) > 0 && (len(params.Key) > 0 || len(params.Values) > 0) {
				return nil, agentruntimev2.RespondToModelError(ToolNameUpsertKnowledgeTable + ": records cannot be combined with key or values")
			}
			if len(params.Records) == 0 && (len(params.Key) == 0 || len(params.Values) == 0) {
				return nil, agentruntimev2.RespondToModelError(ToolNameUpsertKnowledgeTable + ": key and values are required when records is omitted")
			}
			if len(params.Records) > 100 {
				return nil, agentruntimev2.RespondToModelError(ToolNameUpsertKnowledgeTable + ": records exceeds the maximum of 100")
			}
			if opts.Registry == nil || opts.Registry.UpsertKnowledgeTable == nil {
				return nil, fmt.Errorf("%s: tool is not configured", ToolNameUpsertKnowledgeTable)
			}
			ctx = ContextWithScope(ctx, opts.Scope)
			result, err := opts.Registry.UpsertKnowledgeTable.Execute(ctx, UpsertKnowledgeTableRequest{
				Scope:     opts.Scope,
				TableName: params.TableName,
				Key:       cloneKnowledgeMap(params.Key),
				Values:    cloneKnowledgeMap(params.Values),
				Records:   cloneUpsertKnowledgeTableRecords(params.Records),
			})
			if err != nil {
				return nil, fmt.Errorf("%s: %w", ToolNameUpsertKnowledgeTable, err)
			}
			if result == nil {
				return nil, fmt.Errorf("%s: service returned nil result", ToolNameUpsertKnowledgeTable)
			}
			return marshalToolResult(result)
		}))
}

func cloneUpsertKnowledgeTableRecords(records []UpsertKnowledgeTableRecord) []UpsertKnowledgeTableRecord {
	if len(records) == 0 {
		return nil
	}
	out := make([]UpsertKnowledgeTableRecord, len(records))
	for index, record := range records {
		out[index] = UpsertKnowledgeTableRecord{
			Key:    cloneKnowledgeMap(record.Key),
			Values: cloneKnowledgeMap(record.Values),
		}
	}
	return out
}

type selectFinalSourcesParams struct {
	Sources []FinalAnswerSource `json:"sources"`
}

type submitFinalAnswerParams struct {
	Answer  string              `json:"answer"`
	Sources []FinalAnswerSource `json:"sources"`
}

func (p *submitFinalAnswerParams) UnmarshalJSON(raw []byte) error {
	type alias submitFinalAnswerParams
	var aux struct {
		Sources json.RawMessage `json:"sources"`
		*alias
	}
	aux.alias = (*alias)(p)
	if err := json.Unmarshal(raw, &aux); err != nil {
		return err
	}
	if len(aux.Sources) == 0 || string(aux.Sources) == "null" {
		return nil
	}
	var list []FinalAnswerSource
	if err := json.Unmarshal(aux.Sources, &list); err == nil {
		p.Sources = list
		return nil
	}
	var single FinalAnswerSource
	if err := json.Unmarshal(aux.Sources, &single); err != nil {
		return fmt.Errorf("sources must be an array of source objects or a source object: %w", err)
	}
	p.Sources = []FinalAnswerSource{single}
	return nil
}

func NewSelectFinalSourcesTool() agentruntimev2.Tool {
	return agentruntimev2.NewTool(ToolNameSelectFinalSources, SelectFinalSourcesDescription, schema(SelectFinalSourcesSchema), nil,
		agentruntimev2.WithToolInvocationFunc(func(ctx context.Context, _ agentruntimev2.ToolInvocation, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			params, err := decodeParams[selectFinalSourcesParams](ToolNameSelectFinalSources, raw)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			if params.Sources == nil {
				return nil, agentruntimev2.RespondToModelError(ToolNameSelectFinalSources + ": sources is required")
			}
			rc := RunContextFrom(ctx)
			if rc == nil {
				return nil, agentruntimev2.RespondToModelError(ToolNameSelectFinalSources + ": final answer evidence run context is required")
			}
			coverageCandidates, citableEvidenceRetrieved, citableEvidenceRequired := rc.FinalAnswerSourceSelectionCandidates()
			if len(params.Sources) == 0 {
				if citableEvidenceRequired && !citableEvidenceRetrieved {
					return nil, agentruntimev2.RespondToModelError(ToolNameSelectFinalSources + ": sources cannot be empty before a citable evidence retrieval completes")
				}
			}
			sources, err := rc.ResolveFinalAnswerSources(params.Sources)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(ToolNameSelectFinalSources + ": " + err.Error())
			}
			// Keep full resolved sources only in RunContext. Model-visible and
			// message metadata stay ID-level so truncation cannot destroy the
			// accepted selection signal or rehydrate multi-KB candidate blobs.
			rc.SetSelectedFinalAnswerSources(sources)
			sourceRefs := ProjectFinalAnswerSourceRefs(sources)
			output := map[string]any{
				"ok":              true,
				"accepted":        true,
				"selected":        true,
				"source_count":    len(sourceRefs),
				"sources":         sourceRefs,
				"candidate_count": len(coverageCandidates),
			}
			return marshalToolResultWithMetadata(output, map[string]any{
				"ok":                     true,
				"accepted":               true,
				"selected":               true,
				"selected_final_sources": true,
				"source_count":           len(sourceRefs),
				"source_refs":            sourceRefs,
				"candidate_count":        len(coverageCandidates),
			})
		}))
}

func NewSubmitFinalAnswerTool() agentruntimev2.Tool {
	return agentruntimev2.NewTool(ToolNameSubmitFinalAnswer, SubmitFinalAnswerDescription, schema(SubmitFinalAnswerSchema), nil,
		agentruntimev2.WithToolInvocationFunc(func(ctx context.Context, invocation agentruntimev2.ToolInvocation, raw json.RawMessage) (*agentruntimev2.ToolResult, error) {
			params, err := decodeParams[submitFinalAnswerParams](ToolNameSubmitFinalAnswer, raw)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(err.Error())
			}
			if strings.TrimSpace(params.Answer) == "" {
				return nil, agentruntimev2.RespondToModelError(ToolNameSubmitFinalAnswer + ": answer is required")
			}
			// User-facing answer must not include internal chunk locators (#12968).
			params.Answer = StripInternalRAGChunkLocators(params.Answer)
			if strings.TrimSpace(params.Answer) == "" {
				return nil, agentruntimev2.RespondToModelError(ToolNameSubmitFinalAnswer + ": answer is required")
			}
			rc := RunContextFrom(ctx)
			if rc == nil {
				return nil, agentruntimev2.RespondToModelError(ToolNameSubmitFinalAnswer + ": final answer evidence run context is required")
			}
			sources, err := rc.ResolveFinalAnswerSources(params.Sources)
			if err != nil {
				return nil, agentruntimev2.RespondToModelError(ToolNameSubmitFinalAnswer + ": " + err.Error())
			}
			if err := rc.ValidateAnswerSourceCoverage(params.Answer, sources); err != nil {
				return nil, agentruntimev2.RespondToModelError(ToolNameSubmitFinalAnswer + ": " + err.Error())
			}
			submission := FinalAnswerSubmission{
				Answer:  params.Answer,
				Sources: sources,
			}
			output := map[string]any{
				"ok":           true,
				"accepted":     true,
				"answer":       submission.Answer,
				"source_count": len(submission.Sources),
				"sources":      submission.Sources,
			}
			result, err := marshalToolResultWithMetadata(output, map[string]any{
				"answer":       submission.Answer,
				"source_refs":  submission.Sources,
				"source_count": len(submission.Sources),
			})
			if err != nil {
				return nil, err
			}
			result.Artifacts = append(result.Artifacts, finalAnswerArtifact(invocationArtifactID(invocation, "answer"), submission))
			return result, nil
		}))
}

func ragChunksArtifact(artifactID string, result *SearchRAGChunksResponse) agentruntimev2.ToolArtifact {
	return agentruntimev2.ToolArtifact{
		ArtifactID:  strings.TrimSpace(artifactID),
		Name:        "RAG Chunks",
		Type:        "rag_chunks",
		Description: "Retrieved document evidence chunks.",
		Data:        result,
		Metadata: map[string]any{
			"tool":            ToolNameSearchRAGChunks,
			"matrixflow_type": "knowledge.rag_chunks",
		},
	}
}

func visualSearchArtifact(artifactID string, result *SearchVisualImageResponse, metadata map[string]any) agentruntimev2.ToolArtifact {
	return agentruntimev2.ToolArtifact{
		ArtifactID:  strings.TrimSpace(artifactID),
		Name:        "Visual Search",
		Type:        "visual_search",
		Description: "Retrieved visual document image evidence.",
		Data:        result,
		Metadata: mergeToolMetadata(metadata, map[string]any{
			"tool":            ToolNameSearchVisualImage,
			"matrixflow_type": "knowledge.visual_search",
		}),
	}
}

func sqlResultArtifact(artifactID string, result *QuerySQLResponse) agentruntimev2.ToolArtifact {
	return agentruntimev2.ToolArtifact{
		ArtifactID:  strings.TrimSpace(artifactID),
		Name:        "SQL Result",
		Type:        "sql_result",
		Description: "Structured query result.",
		Data:        result,
		Metadata: map[string]any{
			"tool":            ToolNameQuerySQL,
			"matrixflow_type": "knowledge.sql_result",
			"table_sources":   tableSourcesFromSQLResult(artifactID, safeQuerySQLResponse(result)),
		},
	}
}

func finalAnswerArtifact(artifactID string, submission FinalAnswerSubmission) agentruntimev2.ToolArtifact {
	return agentruntimev2.ToolArtifact{
		ArtifactID:  strings.TrimSpace(artifactID),
		Name:        "Final Answer",
		Type:        "answer",
		Description: "Final answer submitted by the knowledge agent.",
		DisplayText: submission.Answer,
		Parts: []agentruntimev2.ToolPart{{
			Kind: "text",
			Text: submission.Answer,
		}},
		Data: submission,
		Metadata: map[string]any{
			"tool":            ToolNameSubmitFinalAnswer,
			"matrixflow_type": "knowledge.answer",
			"source_refs":     submission.Sources,
		},
	}
}

func safeQuerySQLResponse(result *QuerySQLResponse) QuerySQLResponse {
	if result == nil {
		return QuerySQLResponse{}
	}
	return *result
}

func intPtr(value int) *int {
	return &value
}

func sqlResultHandleMetadata(artifactID string, sqlIdx int) map[string]any {
	metadata := map[string]any{"artifact_id": artifactID}
	if sqlIdx >= 0 {
		metadata["sql_idx"] = sqlIdx
	}
	return metadata
}

func describeSchemaToolMetadata(result *DescribeSchemaResponse) map[string]any {
	keys := semanticKeysFromDescribeSchema(result)
	if len(keys) == 0 {
		return nil
	}
	return map[string]any{
		"semantic_keys_injected": keys,
		"display": map[string]any{
			"params": map[string]any{
				"semantic_keys_injected": keys,
			},
		},
	}
}

func querySQLToolMetadata(result *QuerySQLResponse) map[string]any {
	if result == nil {
		return nil
	}
	metadata := map[string]any{}
	displayParams := map[string]any{}
	if len(result.SemanticKeysUsed) > 0 {
		keys := compactStrings(result.SemanticKeysUsed)
		if len(keys) > 0 {
			metadata["semantic_keys_used"] = keys
			displayParams["semantic_keys_used"] = keys
		}
	}
	if len(result.AppliedConstraints) > 0 {
		metadata["applied_constraints"] = result.AppliedConstraints
		displayParams["applied_constraints"] = result.AppliedConstraints
	}
	if len(displayParams) > 0 {
		metadata["display"] = map[string]any{
			"params": displayParams,
		}
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
}

func semanticKeysFromDescribeSchema(result *DescribeSchemaResponse) []string {
	if result == nil {
		return nil
	}
	keys := make([]string, 0)
	for _, table := range result.Tables {
		for _, entry := range table.SemanticEntries {
			key := semanticEntryKey(entry)
			if key != "" {
				keys = append(keys, key)
			}
		}
	}
	return compactStrings(keys)
}

func semanticEntryKey(entry SemanticEntry) string {
	kind := strings.TrimSpace(entry.Kind)
	keyName := strings.TrimSpace(entry.KeyName)
	if kind == "" && keyName == "" {
		return ""
	}
	if kind == "" {
		return keyName
	}
	entryKey := semanticScopedEntryKey(keyName, entry.ModelID)
	if entryKey != "" {
		return kind + ":" + entryKey
	}
	if entry.ModelID > 0 {
		return fmt.Sprintf("%s:%d", kind, entry.ModelID)
	}
	return kind
}

func semanticScopedEntryKey(primary string, modelID int64) string {
	primary = strings.TrimSpace(primary)
	if modelID <= 0 {
		return primary
	}
	prefix := fmt.Sprintf("%d:", modelID)
	if primary != "" {
		if strings.HasPrefix(primary, prefix) {
			return primary
		}
		return prefix + primary
	}
	return fmt.Sprintf("%d", modelID)
}
