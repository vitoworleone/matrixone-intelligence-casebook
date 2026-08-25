package session

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	moi "github.com/matrixflow/moi-core/go-sdk"
)

func semanticModelTablesToRawJSON(tables json.RawMessage) (json.RawMessage, error) {
	if len(tables) == 0 {
		return json.RawMessage("[]"), nil
	}
	if !json.Valid(tables) {
		return nil, fmt.Errorf("invalid semantic model tables json")
	}
	return append(json.RawMessage(nil), tables...), nil
}

type semanticModelFilesPayload struct {
	FileIDs                 []string                    `json:"file_ids"`
	Parents                 []string                    `json:"parents"`
	VolumeIDs               []string                    `json:"volume_ids"`
	Volumes                 []semanticModelVolumeSource `json:"volumes"`
	VectorTable             string                      `json:"vector_table"`
	EmbeddingModel          string                      `json:"embedding_model"`
	ImageVectorTable        string                      `json:"image_vector_table"`
	ImageEmbeddingModel     string                      `json:"image_embedding_model"`
	ImageEmbeddingBackendID string                      `json:"image_embedding_backend_id"`
	ImageEmbeddingDimension int                         `json:"image_embedding_dimension"`
	ImagePreprocessVersion  string                      `json:"image_preprocess_version"`
	ImageDistanceMetric     string                      `json:"image_distance_metric"`
	Tags                    []string                    `json:"tags"`
}

type semanticModelVolumeSource struct {
	VolumeID string   `json:"volume_id"`
	Parents  []string `json:"parents"`
	Path     []string `json:"path"`
}

type semanticModelTableSource struct {
	DBName     string   `json:"db_name"`
	TableNames []string `json:"table_names"`
}

func semanticModelSourcesFromModel(model *moi.SemanticModel) ([]SemanticModelSource, error) {
	if model == nil {
		return nil, semanticModelNotFoundError()
	}

	items := make([]SemanticModelSource, 0)
	if len(model.Files) > 0 {
		fileItems, err := semanticModelFileSources(model.ID, model.Files)
		if err != nil {
			return nil, err
		}
		items = append(items, fileItems...)
	}
	if len(model.Tables) > 0 {
		tableItems, err := semanticModelTableSources(model.ID, model.Tables)
		if err != nil {
			return nil, err
		}
		items = append(items, tableItems...)
	}
	return items, nil
}

func mapToSemanticModelTables(items map[string][]string) []semanticModelTableSource {
	out := make([]semanticModelTableSource, 0, len(items))
	dbNames := make([]string, 0, len(items))
	for dbName := range items {
		dbNames = append(dbNames, dbName)
	}
	sort.Strings(dbNames)
	for _, dbName := range dbNames {
		tableNames := append([]string{}, items[dbName]...)
		sort.Strings(tableNames)
		out = append(out, semanticModelTableSource{DBName: dbName, TableNames: tableNames})
	}
	return out
}

func appendSemanticModelFiles(raw json.RawMessage, modelID int64, fileIDs []string) (json.RawMessage, error) {
	files := map[string]json.RawMessage{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &files); err != nil {
			return nil, semanticModelFilesInvalidError()
		}
	}
	var existingFileIDs []string
	if rawFileIDs, ok := files["file_ids"]; ok && len(rawFileIDs) > 0 && string(rawFileIDs) != "null" {
		if err := json.Unmarshal(rawFileIDs, &existingFileIDs); err != nil {
			return nil, semanticModelFilesInvalidError()
		}
	}
	existingFileIDs = appendUniqueStrings(existingFileIDs, fileIDs)
	fileIDsJSON, err := json.Marshal(existingFileIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal semantic model file_ids: %w", err)
	}
	files["file_ids"] = fileIDsJSON
	if stringJSONFieldEmpty(files, "vector_table") {
		vectorTableJSON, err := json.Marshal(defaultKnowledgeBaseVectorTable(modelID))
		if err != nil {
			return nil, fmt.Errorf("marshal semantic model vector_table: %w", err)
		}
		files["vector_table"] = vectorTableJSON
	}
	if stringJSONFieldEmpty(files, "embedding_model") {
		embeddingModelJSON, err := json.Marshal(kbDefaultEmbeddingModel)
		if err != nil {
			return nil, fmt.Errorf("marshal semantic model embedding_model: %w", err)
		}
		files["embedding_model"] = embeddingModelJSON
	}
	if stringJSONFieldEmpty(files, "image_vector_table") && createFilesHasCompleteImageEmbeddingConfig(files) {
		imageVectorTableJSON, err := json.Marshal(defaultKnowledgeBaseImageVectorTable(modelID))
		if err != nil {
			return nil, fmt.Errorf("marshal semantic model image_vector_table: %w", err)
		}
		files["image_vector_table"] = imageVectorTableJSON
	}
	out, err := json.Marshal(files)
	if err != nil {
		return nil, fmt.Errorf("marshal semantic model files: %w", err)
	}
	return out, nil
}

func semanticModelCreateFilesBase(raw json.RawMessage, modelID int64) (json.RawMessage, error) {
	files := map[string]json.RawMessage{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &files); err != nil {
			return nil, semanticModelFilesInvalidError()
		}
	}
	delete(files, "file_ids")
	return semanticModelFilesMapWithBackendOwnedVectorBindings(files, modelID)
}

func semanticModelCreateFilesBaseWithFixedIndex(raw json.RawMessage, modelID int64, imageIndexEnabled bool, imageEmbeddingBackendID string) (json.RawMessage, error) {
	files := map[string]json.RawMessage{}
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &files); err != nil {
			return nil, semanticModelFilesInvalidError()
		}
	}
	delete(files, "file_ids")
	if err := applyFixedKnowledgeBaseCreateIndexConfig(files, imageIndexEnabled, imageEmbeddingBackendID); err != nil {
		return nil, err
	}
	return semanticModelFilesMapWithBackendOwnedVectorBindings(files, modelID)
}

func applyFixedKnowledgeBaseCreateIndexConfig(files map[string]json.RawMessage, imageIndexEnabled bool, imageEmbeddingBackendID string) error {
	if err := setStringJSONField(files, "embedding_model", kbDefaultEmbeddingModel); err != nil {
		return err
	}
	for _, key := range []string{
		"image_vector_table",
		"image_embedding_model",
		"image_embedding_backend_id",
		"image_embedding_dimension",
		"image_preprocess_version",
		"image_distance_metric",
		"image_index_configs",
		"active_image_index_config_id",
		"image_index_status",
		"image_index_file_statuses",
	} {
		delete(files, key)
	}
	if !imageIndexEnabled {
		return nil
	}

	stringFields := map[string]string{
		"image_embedding_model":      kbDefaultImageEmbeddingModel,
		"image_embedding_backend_id": imageEmbeddingBackendID,
		"image_preprocess_version":   kbDefaultImagePreprocessVersion,
		"image_distance_metric":      kbDefaultImageDistanceMetric,
	}
	for key, value := range stringFields {
		if err := setStringJSONField(files, key, value); err != nil {
			return err
		}
	}
	dimension, err := json.Marshal(kbDefaultImageEmbeddingDimension)
	if err != nil {
		return fmt.Errorf("marshal semantic model files.image_embedding_dimension: %w", err)
	}
	files["image_embedding_dimension"] = dimension
	return nil
}

func semanticModelFilesMapWithBackendOwnedVectorBindings(files map[string]json.RawMessage, modelID int64) (json.RawMessage, error) {
	// KB vector table names are backend-owned bindings derived from modelID.
	if err := setStringJSONField(files, "vector_table", defaultKnowledgeBaseVectorTable(modelID)); err != nil {
		return nil, err
	}
	// image_vector_table is also a backend-owned binding. When image indexing is
	// enabled, replace any client value with the backend table name that workflow
	// will write; an isolated client table name is not an enablement signal.
	if createFilesHasCompleteImageEmbeddingConfig(files) {
		if err := setStringJSONField(files, "image_vector_table", defaultKnowledgeBaseImageVectorTable(modelID)); err != nil {
			return nil, err
		}
	} else {
		delete(files, "image_vector_table")
	}
	out, err := json.Marshal(files)
	if err != nil {
		return nil, fmt.Errorf("marshal semantic model create files: %w", err)
	}
	return out, nil
}

func createFilesHasCompleteImageEmbeddingConfig(files map[string]json.RawMessage) bool {
	for _, key := range []string{"image_embedding_model", "image_embedding_backend_id", "image_preprocess_version", "image_distance_metric"} {
		if stringJSONValue(files, key) == "" {
			return false
		}
	}
	value, ok := intJSONValue(files, "image_embedding_dimension")
	return ok && value > 0
}

func setStringJSONField(fields map[string]json.RawMessage, key, value string) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal semantic model files.%s: %w", key, err)
	}
	fields[key] = raw
	return nil
}

func stringJSONFieldEmpty(fields map[string]json.RawMessage, key string) bool {
	raw, ok := fields[key]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return true
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	return value == ""
}

func stringJSONValue(fields map[string]json.RawMessage, key string) string {
	raw, ok := fields[key]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return ""
	}
	return value
}

func intJSONValue(fields map[string]json.RawMessage, key string) (int, bool) {
	raw, ok := fields[key]
	if !ok || len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return 0, false
	}
	return value, true
}

func appendSemanticModelTables(raw json.RawMessage, tables []semanticModelTableSource) (json.RawMessage, error) {
	var existing []semanticModelTableSource
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &existing); err != nil {
			return nil, semanticModelTablesInvalidError()
		}
	}
	merged := make([]semanticModelTableSource, 0, len(existing)+len(tables))
	indexByDB := make(map[string]int, len(existing)+len(tables))
	for _, table := range existing {
		if idx, ok := indexByDB[table.DBName]; ok {
			merged[idx].TableNames = appendUniqueStrings(merged[idx].TableNames, table.TableNames)
			continue
		}
		table.TableNames = appendUniqueStrings(nil, table.TableNames)
		if len(table.TableNames) == 0 {
			continue
		}
		indexByDB[table.DBName] = len(merged)
		merged = append(merged, table)
	}
	for _, table := range tables {
		if idx, ok := indexByDB[table.DBName]; ok {
			merged[idx].TableNames = appendUniqueStrings(merged[idx].TableNames, table.TableNames)
			continue
		}
		table.TableNames = appendUniqueStrings(nil, table.TableNames)
		if len(table.TableNames) == 0 {
			continue
		}
		indexByDB[table.DBName] = len(merged)
		merged = append(merged, table)
	}
	out, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("marshal semantic model tables: %w", err)
	}
	return out, nil
}

func appendUniqueStrings(existing []string, values []string) []string {
	out := append([]string{}, existing...)
	seen := make(map[string]struct{}, len(out)+len(values))
	for _, value := range out {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func removeKnowledgeBaseSourceFromSemanticModel(model *SemanticModelInfo, record KnowledgeBaseSourceRecord) (json.RawMessage, json.RawMessage, error) {
	if model == nil {
		return nil, nil, fmt.Errorf("semantic model is required")
	}
	files := append(json.RawMessage(nil), model.Files...)
	tables := append(json.RawMessage(nil), model.Tables...)
	switch record.SourceType {
	case kbSourceTypeLocalFile, kbSourceTypeCatalogFile:
		if record.KBFileID == nil || *record.KBFileID == "" {
			return nil, nil, fmt.Errorf("knowledge base file source %s has no kb_file_id", record.SourceID)
		}
		nextFiles, err := removeSemanticModelFileID(files, *record.KBFileID)
		if err != nil {
			return nil, nil, err
		}
		files = nextFiles
	case kbSourceTypeCatalogTable:
		if record.DBName == nil || *record.DBName == "" || record.TableName == nil || *record.TableName == "" {
			return files, tables, nil
		}
		nextTables, err := removeSemanticModelTable(tables, *record.DBName, *record.TableName)
		if err != nil {
			return nil, nil, err
		}
		tables = nextTables
	}
	return files, tables, nil
}

func removeSemanticModelFileID(raw json.RawMessage, fileID string) (json.RawMessage, error) {
	files := map[string]json.RawMessage{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &files); err != nil {
			return nil, semanticModelFilesInvalidError()
		}
	}
	var existingFileIDs []string
	if rawFileIDs, ok := files["file_ids"]; ok && len(rawFileIDs) > 0 && string(rawFileIDs) != "null" {
		if err := json.Unmarshal(rawFileIDs, &existingFileIDs); err != nil {
			return nil, semanticModelFilesInvalidError()
		}
	}
	nextFileIDs := make([]string, 0, len(existingFileIDs))
	for _, existingFileID := range existingFileIDs {
		if existingFileID != fileID {
			nextFileIDs = append(nextFileIDs, existingFileID)
		}
	}
	fileIDsJSON, err := json.Marshal(nextFileIDs)
	if err != nil {
		return nil, fmt.Errorf("marshal semantic model file_ids: %w", err)
	}
	files["file_ids"] = fileIDsJSON
	out, err := json.Marshal(files)
	if err != nil {
		return nil, fmt.Errorf("marshal semantic model files: %w", err)
	}
	return out, nil
}

func semanticModelFileIDs(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var files semanticModelFilesPayload
	if err := json.Unmarshal(raw, &files); err != nil {
		return nil, semanticModelFilesInvalidError()
	}
	out := make([]string, 0, len(files.FileIDs))
	for _, fileID := range files.FileIDs {
		if fileID == "" {
			return nil, semanticModelFilesInvalidError()
		}
		out = append(out, fileID)
	}
	return out, nil
}

func removeSemanticModelTable(raw json.RawMessage, dbName, tableName string) (json.RawMessage, error) {
	var existing []semanticModelTableSource
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &existing); err != nil {
			return nil, semanticModelTablesInvalidError()
		}
	}
	next := make([]semanticModelTableSource, 0, len(existing))
	for _, table := range existing {
		if table.DBName != dbName {
			next = append(next, table)
			continue
		}
		tableNames := make([]string, 0, len(table.TableNames))
		for _, existingTableName := range table.TableNames {
			if existingTableName != tableName {
				tableNames = append(tableNames, existingTableName)
			}
		}
		if len(tableNames) == 0 {
			continue
		}
		table.TableNames = tableNames
		next = append(next, table)
	}
	out, err := json.Marshal(next)
	if err != nil {
		return nil, fmt.Errorf("marshal semantic model tables: %w", err)
	}
	return out, nil
}

func stableID(prefix string, parts ...any) string {
	h := sha1.New()
	for _, part := range parts {
		_, _ = h.Write([]byte(fmt.Sprint(part)))
		_, _ = h.Write([]byte{0})
	}
	return prefix + "-" + hex.EncodeToString(h.Sum(nil))[:24]
}

func compactNonEmptyStrings(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func semanticModelTableKey(dbName, tableName string) string {
	return dbName + "\x00" + tableName
}

func defaultKnowledgeBaseVectorTable(modelID int64) string {
	return fmt.Sprintf("kb_%d_text_index", modelID)
}

func defaultKnowledgeBaseImageVectorTable(modelID int64) string {
	return fmt.Sprintf("kb_%d_image_index", modelID)
}

func int64Ptr(value int64) *int64 {
	return &value
}

func ptrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func semanticModelFileSources(modelID int64, raw json.RawMessage) ([]SemanticModelSource, error) {
	if len(raw) == 0 {
		return []SemanticModelSource{}, nil
	}
	var files semanticModelFilesPayload
	if err := json.Unmarshal(raw, &files); err != nil {
		return nil, semanticModelFilesInvalidError()
	}

	selectedVolumeKeys := make(map[string]struct{}, len(files.VolumeIDs)+len(files.Volumes))
	for _, volumeID := range files.VolumeIDs {
		if volumeID == "" {
			return nil, semanticModelFilesInvalidError()
		}
		selectedVolumeKeys["volume-"+volumeID] = struct{}{}
	}
	for _, volume := range files.Volumes {
		if volume.VolumeID == "" {
			return nil, semanticModelFilesInvalidError()
		}
		selectedVolumeKeys["volume-"+volume.VolumeID] = struct{}{}
	}

	items := make([]SemanticModelSource, 0, len(files.FileIDs)+len(files.VolumeIDs)+len(files.Volumes))
	if !semanticModelFileCoveredByVolume(files.Parents, selectedVolumeKeys) {
		for _, fileID := range files.FileIDs {
			if fileID == "" {
				return nil, semanticModelFilesInvalidError()
			}
			items = append(items, newSemanticModelSource(SemanticModelSourceTypeFile, modelID, fileID, nil, []string{}))
		}
	}

	emittedVolumes := make(map[string]struct{}, len(files.Volumes)+len(files.VolumeIDs))
	for _, volume := range files.Volumes {
		path := append([]string(nil), volume.Path...)
		var displayName *string
		if len(path) > 0 {
			displayName = stringPtr(path[len(path)-1])
		}
		items = append(items, newSemanticModelSource(SemanticModelSourceTypeVolume, modelID, volume.VolumeID, displayName, path))
		emittedVolumes[volume.VolumeID] = struct{}{}
	}
	for _, volumeID := range files.VolumeIDs {
		if _, ok := emittedVolumes[volumeID]; ok {
			continue
		}
		items = append(items, newSemanticModelSource(SemanticModelSourceTypeVolume, modelID, volumeID, nil, []string{}))
		emittedVolumes[volumeID] = struct{}{}
	}
	return items, nil
}

func semanticModelFileCoveredByVolume(parents []string, selectedVolumeKeys map[string]struct{}) bool {
	for _, parentID := range parents {
		if _, ok := selectedVolumeKeys[parentID]; ok {
			return true
		}
	}
	return false
}

func semanticModelTableSources(modelID int64, raw json.RawMessage) ([]SemanticModelSource, error) {
	if len(raw) == 0 {
		return []SemanticModelSource{}, nil
	}
	var tables []semanticModelTableSource
	if err := json.Unmarshal(raw, &tables); err != nil {
		return nil, semanticModelTablesInvalidError()
	}
	items := make([]SemanticModelSource, 0, len(tables))
	for _, table := range tables {
		for _, tableName := range table.TableNames {
			if tableName == "" {
				return nil, semanticModelTablesInvalidError()
			}
			source := newSemanticModelSource(SemanticModelSourceTypeTable, modelID, table.DBName+"::"+tableName, stringPtr(tableName), []string{table.DBName, tableName})
			source.DBName = stringPtr(table.DBName)
			source.TableName = stringPtr(tableName)
			items = append(items, source)
		}
	}
	return items, nil
}

func newSemanticModelSource(sourceType SemanticModelSourceType, modelID int64, resourceID string, displayName *string, path []string) SemanticModelSource {
	ingestStatus := unsupportedSourceField
	return SemanticModelSource{
		RowID:        fmt.Sprintf("%d:%s:%s", modelID, sourceType, resourceID),
		SourceType:   sourceType,
		ModelID:      modelID,
		ResourceID:   resourceID,
		DisplayName:  displayName,
		Path:         append([]string{}, path...),
		IngestStatus: &ingestStatus,
	}
}

func stringPtr(value string) *string {
	return &value
}
