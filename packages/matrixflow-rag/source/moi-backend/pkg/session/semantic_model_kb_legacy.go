package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/coreclient"
	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"

	moi "github.com/matrixflow/moi-core/go-sdk"
	"gorm.io/gorm"
)

const (
	legacyCandidateResourceFile  = "file"
	legacyCandidateResourceTable = "table"
)

type legacySourceCandidateRecord struct {
	record KnowledgeBaseSourceRecord
	origin string
}

func (s *semanticModelService) enrichSemanticModelSourceCounts(ctx context.Context, items []*SemanticModelInfo) error {
	if len(items) == 0 {
		return nil
	}

	models := make([]*SemanticModelInfo, 0, len(items))
	seenModelIDs := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item == nil || item.ID <= 0 {
			continue
		}
		if _, ok := seenModelIDs[item.ID]; ok {
			continue
		}
		seenModelIDs[item.ID] = struct{}{}
		models = append(models, item)
	}
	if len(models) == 0 {
		return nil
	}
	if len(models) == 1 {
		counts, err := s.semanticModelSourceCounts(ctx, models[0])
		if err != nil {
			return err
		}
		for _, item := range items {
			if item != nil && item.ID == models[0].ID {
				item.SourceCounts = counts
			}
		}
		return nil
	}

	countsByModel, err := s.semanticModelSourceCountsBatch(ctx, models)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item == nil || item.ID <= 0 {
			continue
		}
		if counts, ok := countsByModel[item.ID]; ok {
			item.SourceCounts = counts
		}
	}
	return nil
}

func (s *semanticModelService) semanticModelSourceCounts(ctx context.Context, model *SemanticModelInfo) (SemanticModelSourceCounts, error) {
	if model == nil || model.ID <= 0 {
		return SemanticModelSourceCounts{}, nil
	}
	if ctxutil.TenantDBFrom(ctx) == nil {
		return SemanticModelSourceCounts{}, fmt.Errorf("tenant db is required to count semantic model sources")
	}
	allRecords, err := s.listKnowledgeBaseSourceRows(ctx, model.ID, true)
	if err != nil {
		return SemanticModelSourceCounts{}, fmt.Errorf("list semantic model sources for counts: %w", err)
	}
	records := activeKnowledgeBaseSourceRecords(allRecords)
	counts := countKnowledgeBaseSourceRecords(records)
	legacyJobs, err := s.listKnowledgeBaseSourceJobs(ctx, model.ID)
	if err != nil {
		return SemanticModelSourceCounts{}, fmt.Errorf("list semantic model legacy source jobs for counts: %w", err)
	}
	legacyCounts, err := s.legacySourceCandidateCountsWithJobs(ctx, model, allRecords, legacyJobs)
	if err != nil {
		return SemanticModelSourceCounts{}, fmt.Errorf("count semantic model legacy source candidates: %w", err)
	}
	counts.Files += legacyCounts.Files
	counts.Tables += legacyCounts.Tables
	counts.Total = counts.Files + counts.Tables
	return counts, nil
}

type semanticModelLegacyCountIndex struct {
	seen       *legacyCandidateSeenSet
	tableNames map[string]struct{}
}

func newSemanticModelLegacyCountIndex(existing []KnowledgeBaseSourceRecord) *semanticModelLegacyCountIndex {
	index := &semanticModelLegacyCountIndex{
		seen:       newLegacyCandidateSeenSet(existing),
		tableNames: make(map[string]struct{}, len(existing)),
	}
	for _, record := range existing {
		if record.DBName == nil || record.TableName == nil || *record.DBName == "" || *record.TableName == "" {
			continue
		}
		index.tableNames[semanticModelTableKey(*record.DBName, *record.TableName)] = struct{}{}
	}
	return index
}

func (s *semanticModelService) semanticModelSourceCountsBatch(ctx context.Context, models []*SemanticModelInfo) (map[int64]SemanticModelSourceCounts, error) {
	if len(models) == 0 {
		return map[int64]SemanticModelSourceCounts{}, nil
	}
	if ctxutil.TenantDBFrom(ctx) == nil {
		return nil, fmt.Errorf("tenant db is required to count semantic model sources")
	}

	modelIDs := make([]int64, 0, len(models))
	for _, model := range models {
		if model == nil || model.ID <= 0 {
			continue
		}
		modelIDs = append(modelIDs, model.ID)
	}
	counts := make(map[int64]*SemanticModelSourceCounts, len(modelIDs))
	for _, modelID := range modelIDs {
		counts[modelID] = &SemanticModelSourceCounts{}
	}

	recordsByModel, err := s.listKnowledgeBaseSourceRowsForCountBatch(ctx, modelIDs)
	if err != nil {
		return nil, fmt.Errorf("list semantic model sources for counts: %w", err)
	}
	indexes := make(map[int64]*semanticModelLegacyCountIndex, len(modelIDs))
	for _, modelID := range modelIDs {
		records := recordsByModel[modelID]
		indexes[modelID] = newSemanticModelLegacyCountIndex(records)
		for _, record := range activeKnowledgeBaseSourceRecords(records) {
			addSemanticModelSourceCount(counts[modelID], sourceRecordToSemanticModelSource(record).SourceType)
		}
	}

	if err := s.addExplicitLegacySourceCountsBatch(ctx, models, indexes, counts); err != nil {
		return nil, fmt.Errorf("count explicit semantic model legacy source candidates: %w", err)
	}
	legacyJobsByModel, err := s.listKnowledgeBaseSourceJobsBatch(ctx, modelIDs)
	if err != nil {
		return nil, fmt.Errorf("list semantic model legacy source jobs for counts: %w", err)
	}
	if err := s.addLegacyJobSourceCountsBatch(ctx, legacyJobsByModel, indexes, counts); err != nil {
		return nil, fmt.Errorf("count semantic model legacy source job candidates: %w", err)
	}
	if err := s.addRawVolumeLegacySourceCandidateCountsBatch(ctx, modelIDs, indexes, counts); err != nil {
		return nil, fmt.Errorf("count semantic model raw volume legacy source candidates: %w", err)
	}
	if err := s.addLineageLegacySourceCandidateCountsBatch(ctx, models, indexes, counts); err != nil {
		return nil, fmt.Errorf("count semantic model lineage legacy source candidates: %w", err)
	}

	out := make(map[int64]SemanticModelSourceCounts, len(counts))
	for modelID, count := range counts {
		count.Total = count.Files + count.Tables
		out[modelID] = *count
	}
	return out, nil
}

func (s *semanticModelService) listKnowledgeBaseSourceRowsForCountBatch(ctx context.Context, modelIDs []int64) (map[int64][]KnowledgeBaseSourceRecord, error) {
	out := make(map[int64][]KnowledgeBaseSourceRecord, len(modelIDs))
	if len(modelIDs) == 0 {
		return out, nil
	}
	if len(modelIDs) == 1 {
		records, err := s.listKnowledgeBaseSourceRows(ctx, modelIDs[0], true)
		if err != nil {
			return nil, err
		}
		out[modelIDs[0]] = records
		return out, nil
	}

	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return out, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT kbs.source_id AS source_id, kbs.model_id AS model_id, kbs.source_type AS source_type, kbs.source_file_id AS source_file_id, kbs.source_table_id AS source_table_id, kbs.kb_file_id AS kb_file_id, kbs.kb_table_id AS kb_table_id, kbs.db_name AS db_name, kbs.table_name AS table_name, kbs.status AS status
		FROM knowledge_base_sources kbs
		WHERE kbs.model_id IN (`+queryPlaceholders(len(modelIDs))+`)
		ORDER BY kbs.model_id ASC, kbs.created_at ASC, kbs.source_id ASC`, int64Args(modelIDs)...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		record, err := scanKnowledgeBaseSourceRecord(rows)
		if err != nil {
			return nil, err
		}
		out[record.ModelID] = append(out[record.ModelID], record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *semanticModelService) listKnowledgeBaseSourceJobsBatch(ctx context.Context, modelIDs []int64) (map[int64][]KnowledgeBaseSourceJob, error) {
	out := make(map[int64][]KnowledgeBaseSourceJob, len(modelIDs))
	if len(modelIDs) == 0 {
		return out, nil
	}
	if len(modelIDs) == 1 {
		jobs, err := s.listKnowledgeBaseSourceJobs(ctx, modelIDs[0])
		if err != nil {
			return nil, err
		}
		out[modelIDs[0]] = jobs
		return out, nil
	}

	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return out, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT id, model_id, source_type, source_file_id, kb_file_id, raw_volume_id, job_status, error, segment_version_id, index_version, workflow_execution_id
		FROM knowledge_base_source_jobs
		WHERE model_id IN (`+queryPlaceholders(len(modelIDs))+`)
		ORDER BY model_id ASC, id ASC`, int64Args(modelIDs)...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var job KnowledgeBaseSourceJob
		if err := rows.Scan(
			&job.ID,
			&job.ModelID,
			&job.SourceType,
			&job.SourceFileID,
			&job.KBFileID,
			&job.RawVolumeID,
			&job.JobStatus,
			&job.Error,
			&job.SegmentVersionID,
			&job.IndexVersion,
			&job.WorkflowExecutionID,
		); err != nil {
			return nil, err
		}
		out[job.ModelID] = append(out[job.ModelID], job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *semanticModelService) addExplicitLegacySourceCountsBatch(ctx context.Context, models []*SemanticModelInfo, indexes map[int64]*semanticModelLegacyCountIndex, counts map[int64]*SemanticModelSourceCounts) error {
	fileIDsByModel := make(map[int64][]string, len(models))
	tableNamesByModel := make(map[int64][]semanticModelTableName, len(models))
	allFileIDs := make([]string, 0)
	allTableNames := make([]semanticModelTableName, 0)

	for _, model := range models {
		if model == nil || model.ID <= 0 {
			continue
		}
		index := indexes[model.ID]
		if index == nil {
			continue
		}
		fileIDs, err := semanticModelFileIDs(model.Files)
		if err != nil {
			return err
		}
		for _, fileID := range fileIDs {
			if index.seen.has(legacyCandidateResourceFile + ":" + fileID) {
				continue
			}
			fileIDsByModel[model.ID] = append(fileIDsByModel[model.ID], fileID)
			allFileIDs = append(allFileIDs, fileID)
		}

		tables, err := semanticModelTableSources(model.ID, model.Tables)
		if err != nil {
			return err
		}
		for _, source := range tables {
			if source.DBName == nil || source.TableName == nil {
				continue
			}
			tableName := semanticModelTableName{dbName: *source.DBName, tableName: *source.TableName}
			key := tableName.key()
			if _, ok := index.tableNames[key]; ok {
				continue
			}
			tableNamesByModel[model.ID] = append(tableNamesByModel[model.ID], tableName)
			allTableNames = append(allTableNames, tableName)
		}
	}

	validFiles, err := batchCurrentCatalogFileValidity(ctx, allFileIDs)
	if err != nil {
		return err
	}
	for _, model := range models {
		if model == nil || model.ID <= 0 {
			continue
		}
		index := indexes[model.ID]
		count := counts[model.ID]
		if index == nil || count == nil {
			continue
		}
		for _, fileID := range fileIDsByModel[model.ID] {
			key := legacyCandidateResourceFile + ":" + fileID
			if index.seen.has(key) {
				continue
			}
			if _, ok := validFiles[fileID]; !ok {
				continue
			}
			index.seen.add(key)
			addSemanticModelSourceCount(count, SemanticModelSourceTypeFile)
		}
	}

	validTables, err := batchResolveCatalogTableIDsByName(ctx, allTableNames)
	if err != nil {
		return err
	}
	for _, model := range models {
		if model == nil || model.ID <= 0 {
			continue
		}
		index := indexes[model.ID]
		count := counts[model.ID]
		if index == nil || count == nil {
			continue
		}
		for _, tableName := range tableNamesByModel[model.ID] {
			tableID, ok := validTables[tableName.key()]
			if !ok {
				continue
			}
			resourceKey := legacyCandidateResourceTable + ":" + strconv.FormatInt(tableID, 10)
			if index.seen.has(resourceKey) {
				continue
			}
			index.seen.add(resourceKey)
			index.tableNames[tableName.key()] = struct{}{}
			addSemanticModelSourceCount(count, SemanticModelSourceTypeTable)
		}
	}
	return nil
}

func (s *semanticModelService) addLegacyJobSourceCountsBatch(ctx context.Context, jobsByModel map[int64][]KnowledgeBaseSourceJob, indexes map[int64]*semanticModelLegacyCountIndex, counts map[int64]*SemanticModelSourceCounts) error {
	fileIDsByModel := make(map[int64][]string, len(jobsByModel))
	allFileIDs := make([]string, 0)
	for modelID, jobs := range jobsByModel {
		index := indexes[modelID]
		if index == nil {
			continue
		}
		for _, job := range jobs {
			fileID := legacyJobFileID(job.SourceFileID, job.KBFileID)
			if fileID == "" || index.seen.has(legacyCandidateResourceFile+":"+fileID) {
				continue
			}
			fileIDsByModel[modelID] = append(fileIDsByModel[modelID], fileID)
			allFileIDs = append(allFileIDs, fileID)
		}
	}

	validFiles, err := batchCurrentCatalogFileValidity(ctx, allFileIDs)
	if err != nil {
		return err
	}
	for modelID, fileIDs := range fileIDsByModel {
		index := indexes[modelID]
		count := counts[modelID]
		if index == nil || count == nil {
			continue
		}
		for _, fileID := range fileIDs {
			key := legacyCandidateResourceFile + ":" + fileID
			if index.seen.has(key) {
				continue
			}
			if _, ok := validFiles[fileID]; !ok {
				continue
			}
			index.seen.add(key)
			addSemanticModelSourceCount(count, SemanticModelSourceTypeFile)
		}
	}
	return nil
}

func (s *semanticModelService) addRawVolumeLegacySourceCandidateCountsBatch(ctx context.Context, modelIDs []int64, indexes map[int64]*semanticModelLegacyCountIndex, counts map[int64]*SemanticModelSourceCounts) error {
	volumeIDsByModel, err := knowledgeBaseRawVolumeIDsByModelIDs(ctx, modelIDs)
	if err != nil {
		return err
	}
	allVolumeIDs := make([]int64, 0)
	for _, volumeIDs := range volumeIDsByModel {
		allVolumeIDs = append(allVolumeIDs, volumeIDs...)
	}
	fileIDsByVolume, err := knowledgeBaseRawVolumeFileIDsByVolumeIDs(ctx, allVolumeIDs)
	if err != nil {
		return err
	}
	for _, modelID := range modelIDs {
		index := indexes[modelID]
		count := counts[modelID]
		if index == nil || count == nil {
			continue
		}
		for _, volumeID := range volumeIDsByModel[modelID] {
			for _, fileID := range fileIDsByVolume[volumeID] {
				key := legacyCandidateResourceFile + ":" + fileID
				if index.seen.has(key) {
					continue
				}
				index.seen.add(key)
				count.Files++
			}
		}
	}
	return nil
}

func (s *semanticModelService) addLineageLegacySourceCandidateCountsBatch(ctx context.Context, models []*SemanticModelInfo, indexes map[int64]*semanticModelLegacyCountIndex, counts map[int64]*SemanticModelSourceCounts) error {
	modelIDsByVectorTable := make(map[string][]int64)
	for _, model := range models {
		if model == nil || model.ID <= 0 {
			continue
		}
		vectorTables, err := semanticModelVectorTables(model.Files)
		if err != nil {
			return err
		}
		for _, vectorTable := range vectorTables {
			modelIDsByVectorTable[vectorTable] = append(modelIDsByVectorTable[vectorTable], model.ID)
		}
	}
	if len(modelIDsByVectorTable) == 0 {
		return nil
	}

	vectorTables := make([]string, 0, len(modelIDsByVectorTable))
	for vectorTable := range modelIDsByVectorTable {
		vectorTables = append(vectorTables, vectorTable)
	}
	sort.Strings(vectorTables)
	fileIDsByVectorTable, err := lineageCandidateFileIDsByVectorTables(ctx, vectorTables)
	if err != nil {
		return err
	}

	fileIDsByModel := make(map[int64][]string, len(models))
	allFileIDs := make([]string, 0)
	for _, vectorTable := range vectorTables {
		for _, fileID := range fileIDsByVectorTable[vectorTable] {
			for _, modelID := range modelIDsByVectorTable[vectorTable] {
				index := indexes[modelID]
				if index == nil || index.seen.has(legacyCandidateResourceFile+":"+fileID) {
					continue
				}
				fileIDsByModel[modelID] = append(fileIDsByModel[modelID], fileID)
				allFileIDs = append(allFileIDs, fileID)
			}
		}
	}

	validFiles, err := batchCurrentCatalogFileValidity(ctx, allFileIDs)
	if err != nil {
		return err
	}
	for _, model := range models {
		if model == nil || model.ID <= 0 {
			continue
		}
		index := indexes[model.ID]
		count := counts[model.ID]
		if index == nil || count == nil {
			continue
		}
		for _, fileID := range fileIDsByModel[model.ID] {
			key := legacyCandidateResourceFile + ":" + fileID
			if index.seen.has(key) {
				continue
			}
			if _, ok := validFiles[fileID]; !ok {
				continue
			}
			index.seen.add(key)
			count.Files++
		}
	}
	return nil
}

type semanticModelTableName struct {
	dbName    string
	tableName string
}

func (t semanticModelTableName) key() string {
	return semanticModelTableKey(t.dbName, t.tableName)
}

func batchCurrentCatalogFileValidity(ctx context.Context, fileIDs []string) (map[string]struct{}, error) {
	valid := make(map[string]struct{})
	fileIDs = uniqueStringsSorted(fileIDs)
	if len(fileIDs) == 0 {
		return valid, nil
	}
	if len(fileIDs) == 1 {
		if _, err := currentCatalogFileMetadata(ctx, fileIDs[0]); err != nil {
			var missing knowledgeBaseSourceMissingError
			if errors.As(err, &missing) {
				return valid, nil
			}
			return nil, err
		}
		valid[fileIDs[0]] = struct{}{}
		return valid, nil
	}

	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT f.file_id, COALESCE(CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END, 0) AS catalog_id, COALESCE(v.database_id, 0) AS database_id, COALESCE(vf.volume_id, 0) AS volume_id, f.size, UNIX_TIMESTAMP(COALESCE(vf.updated_at, f.updated_at)) AS updated_at, COALESCE(c.catalog_name, '') AS catalog_name, COALESCE(cd.database_name, '') AS database_name, COALESCE(v.volume_name, '') AS volume_name, COALESCE(vf.file_path, '') AS file_path, COALESCE(vf.file_name, f.original_name) AS file_name
		FROM `+"`file`"+` f
		LEFT JOIN volume_files vf ON vf.file_id = f.file_id
		LEFT JOIN volume v ON v.volume_id = vf.volume_id AND v.deleted = FALSE
		LEFT JOIN catalog_database cd ON cd.database_id = v.database_id
		LEFT JOIN catalog c ON c.catalog_id = CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END
		WHERE f.file_id IN (`+queryPlaceholders(len(fileIDs))+`)
		ORDER BY f.file_id ASC, vf.updated_at DESC, vf.id DESC`, stringArgs(fileIDs)...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	scanned := make(map[string]struct{}, len(fileIDs))
	for rows.Next() {
		var fileID string
		var catalogID, databaseID, volumeID, sizeBytes, updatedAt int64
		var catalogName, databaseName, volumeName, filePath, fileName string
		if err := rows.Scan(&fileID, &catalogID, &databaseID, &volumeID, &sizeBytes, &updatedAt, &catalogName, &databaseName, &volumeName, &filePath, &fileName); err != nil {
			return nil, err
		}
		if _, ok := scanned[fileID]; ok {
			continue
		}
		scanned[fileID] = struct{}{}
		if fileName == "" {
			return nil, fmt.Errorf("catalog file %s has empty display name", fileID)
		}
		if volumeName == "" || catalogID <= 0 || databaseID <= 0 || volumeID <= 0 {
			continue
		}
		valid[fileID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return valid, nil
}

func batchResolveCatalogTableIDsByName(ctx context.Context, tableNames []semanticModelTableName) (map[string]int64, error) {
	tableNames = uniqueSemanticModelTableNamesSorted(tableNames)
	out := make(map[string]int64, len(tableNames))
	if len(tableNames) == 0 {
		return out, nil
	}
	if len(tableNames) == 1 {
		ref, err := (&semanticModelService{}).resolveCatalogTableByName(ctx, tableNames[0].dbName, tableNames[0].tableName)
		if err != nil {
			var missing knowledgeBaseSourceMissingError
			if errors.As(err, &missing) {
				return out, nil
			}
			return nil, err
		}
		out[tableNames[0].key()] = ref.tableID
		return out, nil
	}

	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	clauses := make([]string, 0, len(tableNames))
	args := make([]any, 0, len(tableNames)*2)
	for _, tableName := range tableNames {
		clauses = append(clauses, `(cd.database_name = ? AND t.table_name = ?)`)
		args = append(args, tableName.dbName, tableName.tableName)
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT t.table_id, t.database_id, t.catalog_id, COALESCE(cd.database_name, ''), t.table_name, COALESCE(c.catalog_name, '')
		FROM catalog_table t
		INNER JOIN catalog_database cd ON cd.database_id = t.database_id
		LEFT JOIN catalog c ON c.catalog_id = t.catalog_id
		WHERE `+strings.Join(clauses, " OR ")+`
		ORDER BY cd.database_name ASC, t.table_name ASC, t.table_id ASC`, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := make(map[string][]int64, len(tableNames))
	for rows.Next() {
		var ref catalogTableSourceRef
		var catalogName string
		if err := rows.Scan(&ref.tableID, &ref.databaseID, &ref.catalogID, &ref.dbName, &ref.tableName, &catalogName); err != nil {
			return nil, err
		}
		key := semanticModelTableKey(ref.dbName, ref.tableName)
		matches[key] = append(matches[key], ref.tableID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for key, tableIDs := range matches {
		if len(tableIDs) == 1 {
			out[key] = tableIDs[0]
		}
	}
	return out, nil
}

func knowledgeBaseRawVolumeIDsByModelIDs(ctx context.Context, modelIDs []int64) (map[int64][]int64, error) {
	out := make(map[int64][]int64, len(modelIDs))
	if len(modelIDs) == 0 {
		return out, nil
	}
	if len(modelIDs) == 1 {
		volumeIDs, err := knowledgeBaseRawVolumeIDs(ctx, modelIDs[0])
		if err != nil {
			return nil, err
		}
		out[modelIDs[0]] = volumeIDs
		return out, nil
	}

	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	collect := func(query string) error {
		rows, err := db.WithContext(ctx).Raw(query, int64Args(modelIDs)...).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var modelID, volumeID int64
			if err := rows.Scan(&modelID, &volumeID); err != nil {
				return err
			}
			if volumeID > 0 {
				out[modelID] = append(out[modelID], volumeID)
			}
		}
		return rows.Err()
	}
	if err := collect(`SELECT model_id, raw_volume_id
		FROM knowledge_base_raw_volumes
		WHERE model_id IN (` + queryPlaceholders(len(modelIDs)) + `) AND raw_volume_id > 0
		  AND COALESCE(raw_kind, '') <> '` + kbRawKindStructured + `'`); err != nil {
		return nil, err
	}
	if err := collect(`SELECT model_id, raw_volume_id
		FROM knowledge_base_data_domains
		WHERE model_id IN (` + queryPlaceholders(len(modelIDs)) + `) AND raw_volume_id > 0`); err != nil {
		return nil, err
	}
	for modelID, volumeIDs := range out {
		out[modelID] = uniqueInt64sSorted(volumeIDs)
	}
	return out, nil
}

func knowledgeBaseRawVolumeFileIDsByVolumeIDs(ctx context.Context, volumeIDs []int64) (map[int64][]string, error) {
	out := make(map[int64][]string)
	volumeIDs = uniqueInt64sSorted(volumeIDs)
	if len(volumeIDs) == 0 {
		return out, nil
	}
	if len(volumeIDs) == 1 {
		files, err := knowledgeBaseRawVolumeFileMetadataBatch(ctx, volumeIDs[0], 0, 0)
		if err != nil {
			return nil, err
		}
		fileIDs := make([]string, 0, len(files))
		for _, file := range files {
			fileIDs = append(fileIDs, file.fileID)
		}
		out[volumeIDs[0]] = fileIDs
		return out, nil
	}

	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT vf.file_id, COALESCE(CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END, 0) AS catalog_id, COALESCE(v.database_id, 0) AS database_id, vf.volume_id, f.size, UNIX_TIMESTAMP(COALESCE(vf.updated_at, f.updated_at)) AS updated_at, COALESCE(c.catalog_name, '') AS catalog_name, COALESCE(cd.database_name, '') AS database_name, COALESCE(v.volume_name, '') AS volume_name, COALESCE(vf.file_name, f.original_name) AS file_name
		FROM volume_files vf
		INNER JOIN volume v ON v.volume_id = vf.volume_id AND v.deleted = FALSE
		INNER JOIN `+"`file`"+` f ON f.file_id = vf.file_id
		LEFT JOIN catalog_database cd ON cd.database_id = v.database_id
		LEFT JOIN catalog c ON c.catalog_id = CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END
		WHERE vf.volume_id IN (`+queryPlaceholders(len(volumeIDs))+`)
		ORDER BY vf.volume_id ASC, vf.updated_at DESC, vf.id DESC`, int64Args(volumeIDs)...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var file catalogFileSourceRef
		var catalogName, databaseName, volumeName string
		if err := rows.Scan(&file.fileID, &file.catalogID, &file.databaseID, &file.volumeID, &file.sizeBytes, &file.updatedAt, &catalogName, &databaseName, &volumeName, &file.fileName); err != nil {
			return nil, err
		}
		if file.fileID == "" {
			return nil, fmt.Errorf("knowledge base raw volume %d returned empty file id", file.volumeID)
		}
		if file.fileName == "" {
			return nil, fmt.Errorf("knowledge base raw volume file %s has empty display name", file.fileID)
		}
		if file.catalogID <= 0 || file.databaseID <= 0 || file.volumeID <= 0 || catalogName == "" || databaseName == "" || volumeName == "" {
			continue
		}
		out[file.volumeID] = append(out[file.volumeID], file.fileID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func lineageCandidateFileIDsByVectorTables(ctx context.Context, vectorTables []string) (map[string][]string, error) {
	out := make(map[string][]string)
	vectorTables = uniqueStringsSorted(vectorTables)
	if len(vectorTables) == 0 {
		return out, nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return out, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT DISTINCT vector.asset_ref, COALESCE(pm.source_file_id, root.asset_ref) AS file_id
			FROM data_asset vector
			INNER JOIN data_derivation d ON d.target_asset_id = vector.asset_id AND d.kind = 'indexed_from'
			INNER JOIN data_asset root ON root.asset_id = d.root_asset_id AND root.asset_type = 'file'
			LEFT JOIN parsed_manifest pm ON pm.root_asset_id = d.root_asset_id
		WHERE vector.asset_type = 'vector_index'
		  AND vector.asset_ref IN (`+queryPlaceholders(len(vectorTables))+`)
		ORDER BY vector.asset_ref ASC, file_id ASC`, stringArgs(vectorTables)...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var vectorTable, fileID string
		if err := rows.Scan(&vectorTable, &fileID); err != nil {
			return nil, err
		}
		if fileID == "" {
			continue
		}
		out[vectorTable] = append(out[vectorTable], fileID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func queryPlaceholders(count int) string {
	return strings.TrimRight(strings.Repeat("?,", count), ",")
}

func int64Args(values []int64) []any {
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	return args
}

func stringArgs(values []string) []any {
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}
	return args
}

func uniqueStringsSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func uniqueInt64sSorted(values []int64) []int64 {
	seen := make(map[int64]struct{}, len(values))
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func uniqueSemanticModelTableNamesSorted(values []semanticModelTableName) []semanticModelTableName {
	seen := make(map[string]struct{}, len(values))
	out := make([]semanticModelTableName, 0, len(values))
	for _, value := range values {
		if value.dbName == "" || value.tableName == "" {
			continue
		}
		key := value.key()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].key() < out[j].key() })
	return out
}

func (s *semanticModelService) legacySourceCandidateCountsWithJobs(ctx context.Context, model *SemanticModelInfo, existing []KnowledgeBaseSourceRecord, legacyJobs []KnowledgeBaseSourceJob) (SemanticModelSourceCounts, error) {
	var counts SemanticModelSourceCounts
	if model == nil || model.ID <= 0 {
		return counts, nil
	}
	seen := newLegacyCandidateSeenSet(existing)
	add := func(resourceKind, resourceID string, sourceType SemanticModelSourceType) {
		if resourceID == "" {
			return
		}
		key := resourceKind + ":" + resourceID
		if seen.has(key) {
			return
		}
		seen.add(key)
		addSemanticModelSourceCount(&counts, sourceType)
	}

	explicit, err := s.explicitSemanticModelSourceRecords(ctx, model, existing)
	if err != nil {
		return SemanticModelSourceCounts{}, err
	}
	for _, record := range explicit {
		add(recordCandidateResourceKind(record), recordCandidateResourceID(record), sourceRecordToSemanticModelSource(record).SourceType)
	}

	for _, job := range legacyJobs {
		fileID := legacyJobFileID(job.SourceFileID, job.KBFileID)
		if fileID == "" || seen.has(legacyCandidateResourceFile+":"+fileID) {
			continue
		}
		record, err := legacySourceRecordFromJob(ctx, model.ID, job, nil)
		if err != nil {
			var missing knowledgeBaseSourceMissingError
			if errors.As(err, &missing) {
				continue
			}
			return SemanticModelSourceCounts{}, err
		}
		add(legacyCandidateResourceFile, fileID, sourceRecordToSemanticModelSource(record).SourceType)
	}

	if err := s.addRawVolumeLegacySourceCandidateCounts(ctx, model.ID, seen, &counts); err != nil {
		return SemanticModelSourceCounts{}, err
	}
	if err := s.addLineageLegacySourceCandidateCounts(ctx, model, seen, &counts); err != nil {
		return SemanticModelSourceCounts{}, err
	}
	counts.Total = counts.Files + counts.Tables
	return counts, nil
}

func (s *semanticModelService) addRawVolumeLegacySourceCandidateCounts(ctx context.Context, modelID int64, seen *legacyCandidateSeenSet, counts *SemanticModelSourceCounts) error {
	volumeIDs, err := knowledgeBaseRawVolumeIDs(ctx, modelID)
	if err != nil {
		return err
	}
	for _, volumeID := range volumeIDs {
		offset := 0
		for {
			rawVolumeFiles, err := knowledgeBaseRawVolumeFileMetadataBatch(ctx, volumeID, kbLegacyBackfillBatchSize, offset)
			if err != nil {
				return err
			}
			for _, file := range rawVolumeFiles {
				key := legacyCandidateResourceFile + ":" + file.fileID
				if seen != nil && seen.has(key) {
					continue
				}
				if seen != nil {
					seen.add(key)
				}
				counts.Files++
			}
			if len(rawVolumeFiles) < kbLegacyBackfillBatchSize {
				break
			}
			offset += kbLegacyBackfillBatchSize
		}
	}
	return nil
}

func (s *semanticModelService) addLineageLegacySourceCandidateCounts(ctx context.Context, model *SemanticModelInfo, seen *legacyCandidateSeenSet, counts *SemanticModelSourceCounts) error {
	if model == nil || model.ID <= 0 {
		return nil
	}
	vectorTables, err := semanticModelVectorTables(model.Files)
	if err != nil {
		return err
	}
	if len(vectorTables) == 0 {
		return nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(vectorTables)), ",")
	baseArgs := make([]any, 0, len(vectorTables)+1)
	for _, table := range vectorTables {
		baseArgs = append(baseArgs, table)
	}
	baseArgs = append(baseArgs, model.ID)
	query := fmt.Sprintf(`SELECT DISTINCT COALESCE(pm.source_file_id, root.asset_ref) AS file_id
			FROM data_asset vector
			INNER JOIN data_derivation d ON d.target_asset_id = vector.asset_id AND d.kind = 'indexed_from'
			INNER JOIN data_asset root ON root.asset_id = d.root_asset_id AND root.asset_type = 'file'
			LEFT JOIN parsed_manifest pm ON pm.root_asset_id = d.root_asset_id
		WHERE vector.asset_type = 'vector_index'
		  AND vector.asset_ref IN (%s)
		  AND NOT EXISTS (
		    SELECT 1 FROM knowledge_base_sources kbs
		    WHERE kbs.model_id = ?
		      AND (
		        kbs.kb_file_id = COALESCE(pm.source_file_id, root.asset_ref)
		        OR kbs.source_file_id = COALESCE(pm.source_file_id, root.asset_ref)
		      )
		  )
		ORDER BY file_id
		LIMIT ? OFFSET ?`, placeholders)

	offset := 0
	for {
		args := append([]any{}, baseArgs...)
		args = append(args, kbLegacyBackfillBatchSize, offset)
		rows, err := db.WithContext(ctx).Raw(query, args...).Rows()
		if err != nil {
			return err
		}
		fileIDs, scanned, err := scanLineageCandidateFileIDs(rows, seen)
		if err != nil {
			return err
		}
		for _, fileID := range fileIDs {
			_, err := semanticModelFileSourceRecord(ctx, model.ID, fileID)
			if err != nil {
				var missing knowledgeBaseSourceMissingError
				if errors.As(err, &missing) {
					continue
				}
				return err
			}
			key := legacyCandidateResourceFile + ":" + fileID
			if seen != nil && seen.has(key) {
				continue
			}
			if seen != nil {
				seen.add(key)
			}
			counts.Files++
		}
		if scanned < kbLegacyBackfillBatchSize {
			break
		}
		offset += kbLegacyBackfillBatchSize
	}
	return nil
}

func countKnowledgeBaseSourceRecords(records []KnowledgeBaseSourceRecord) SemanticModelSourceCounts {
	var counts SemanticModelSourceCounts
	for _, record := range records {
		addSemanticModelSourceCount(&counts, sourceRecordToSemanticModelSource(record).SourceType)
	}
	counts.Total = counts.Files + counts.Tables
	return counts
}

func addSemanticModelSourceCount(counts *SemanticModelSourceCounts, sourceType SemanticModelSourceType) {
	if sourceType == SemanticModelSourceTypeTable {
		counts.Tables++
		return
	}
	counts.Files++
}

func (s *semanticModelService) legacyBackfillRequired(ctx context.Context, model *SemanticModelInfo, records []KnowledgeBaseSourceRecord, runs []KnowledgeBaseSourceJobRun) (bool, error) {
	if model == nil {
		return false, nil
	}
	modelID := model.ID
	if semanticModelExplicitSourcesBackfillRequired(model, records) {
		return true, nil
	}
	legacyJobs, err := s.listKnowledgeBaseSourceJobs(ctx, modelID)
	if err != nil {
		return false, err
	}
	rawRequired, err := s.rawVolumeBackfillRequired(ctx, modelID)
	if err != nil {
		return false, err
	}
	if rawRequired {
		return true, nil
	}
	return legacySourceJobRunsBackfillRequired(records, runs, legacyJobs), nil
}

func legacySourceJobRunsBackfillRequired(records []KnowledgeBaseSourceRecord, runs []KnowledgeBaseSourceJobRun, legacyJobs []KnowledgeBaseSourceJob) bool {
	if len(legacyJobs) == 0 {
		return false
	}
	recordsByKBFile := make(map[string]struct{}, len(records))
	recordsBySourceFile := make(map[string]struct{}, len(records))
	removedByKBFile := make(map[string]struct{}, len(records))
	removedBySourceFile := make(map[string]struct{}, len(records))
	for _, record := range records {
		if record.KBFileID != nil && *record.KBFileID != "" {
			recordsByKBFile[*record.KBFileID] = struct{}{}
			if isKnowledgeBaseSourceRemoved(record) {
				removedByKBFile[*record.KBFileID] = struct{}{}
			}
		}
		if record.SourceFileID != nil && *record.SourceFileID != "" {
			recordsBySourceFile[*record.SourceFileID] = struct{}{}
			if isKnowledgeBaseSourceRemoved(record) {
				removedBySourceFile[*record.SourceFileID] = struct{}{}
			}
		}
	}
	runsByLegacyKey := make(map[string]struct{}, len(runs))
	for _, run := range runs {
		fileID := legacyJobFileID(run.SourceFileID, run.KBFileID)
		if fileID != "" {
			runsByLegacyKey[legacySourceJobRunKey(run.ModelID, fileID, run.WorkflowExecutionID)] = struct{}{}
		}
	}
	for _, job := range legacyJobs {
		fileID := legacyJobFileID(job.SourceFileID, job.KBFileID)
		if fileID == "" {
			continue
		}
		if _, ok := removedByKBFile[fileID]; ok {
			continue
		}
		if job.SourceFileID != nil && *job.SourceFileID != "" {
			if _, ok := removedBySourceFile[*job.SourceFileID]; ok {
				continue
			}
		}
		if _, ok := recordsByKBFile[fileID]; !ok {
			if job.KBFileID != nil && *job.KBFileID != "" {
				return true
			}
			if job.SourceFileID == nil || *job.SourceFileID == "" {
				return true
			}
			if _, ok := recordsBySourceFile[*job.SourceFileID]; !ok {
				return true
			}
		}
		if _, ok := runsByLegacyKey[legacySourceJobRunKey(job.ModelID, fileID, job.WorkflowExecutionID)]; !ok {
			return true
		}
	}
	return false
}

func (s *semanticModelService) rawVolumeBackfillRequired(ctx context.Context, modelID int64) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, nil
	}
	volumeIDs, err := knowledgeBaseRawVolumeIDs(ctx, modelID)
	if err != nil {
		return false, err
	}
	if len(volumeIDs) == 0 {
		return false, nil
	}
	for _, volumeID := range volumeIDs {
		rows, err := db.WithContext(ctx).Raw(`SELECT vf.file_id
			FROM volume_files vf
			INNER JOIN volume v ON v.volume_id = vf.volume_id AND v.deleted = FALSE
			INNER JOIN `+"`file`"+` f ON f.file_id = vf.file_id
			LEFT JOIN catalog_database cd ON cd.database_id = v.database_id
			LEFT JOIN catalog c ON c.catalog_id = CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END
			WHERE vf.volume_id = ?
			  AND COALESCE(CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END, 0) > 0
			  AND COALESCE(v.database_id, 0) > 0
			  AND COALESCE(v.volume_name, '') <> ''
			  AND COALESCE(c.catalog_name, '') <> ''
			  AND COALESCE(cd.database_name, '') <> ''
			  AND NOT EXISTS (
			    SELECT 1 FROM knowledge_base_sources kbs
			    WHERE kbs.model_id = ?
			      AND (
			        kbs.kb_file_id = vf.file_id
			        OR ((kbs.kb_file_id IS NULL OR kbs.kb_file_id = '') AND kbs.source_file_id = vf.file_id)
			      )
			  )
			LIMIT 1`, volumeID, modelID).Rows()
		if err != nil {
			return false, err
		}
		found := rows.Next()
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return false, err
		}
		if err := rows.Close(); err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

func (s *semanticModelService) BackfillLegacySources(ctx context.Context, params BackfillLegacyKnowledgeBaseSourcesParams) error {
	wsID, err := callerWorkspaceID(ctx)
	if err != nil {
		return err
	}
	return coreclient.Execute(ctx, coreclient.FromContext(ctx), func(callCtx context.Context, client *moi.Client) error {
		_, callErr := s.backfillLegacySourcesBatch(callCtx, client, wsID, params.ModelID, semanticModelActor(callCtx), kbLegacyBackfillBatchSize)
		return callErr
	})
}

func (s *semanticModelService) backfillLegacySourcesBatch(ctx context.Context, c *moi.Client, wsID string, modelID int64, actor string, limit int) (int, error) {
	if modelID == 0 {
		return 0, semanticModelNotFoundError()
	}
	if limit <= 0 {
		return 0, nil
	}
	model, err := c.SemanticModels(wsID).Get(ctx, modelID)
	if err != nil {
		return 0, err
	}
	modelInfo := toSemanticModelInfo(model)
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return 0, fmt.Errorf("tenant db is required")
	}
	updated := 0
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := ctxutil.WithTenantDB(ctx, tx)
		records, err := s.listKnowledgeBaseSourceRows(txCtx, modelID, true)
		if err != nil {
			return err
		}
		legacyJobs, err := s.listKnowledgeBaseSourceJobs(txCtx, modelID)
		if err != nil {
			return err
		}
		var legacyJobRuns []KnowledgeBaseSourceJobRun
		if len(legacyJobs) > 0 {
			legacyJobRuns, err = s.listKnowledgeBaseSourceJobRuns(txCtx, modelID)
			if err != nil {
				return err
			}
		}
		domain, _, err := s.getKnowledgeBaseDataDomain(txCtx, modelID)
		if err != nil {
			return err
		}
		byKBFile := make(map[string]KnowledgeBaseSourceRecord, len(records))
		bySourceFile := make(map[string]KnowledgeBaseSourceRecord, len(records))
		byTable := make(map[int64]KnowledgeBaseSourceRecord, len(records))
		rawVolumeFallbackFiles := make(map[string]struct{}, len(records))
		for _, record := range records {
			if record.KBFileID != nil && *record.KBFileID != "" {
				byKBFile[*record.KBFileID] = record
			}
			if record.SourceFileID != nil && *record.SourceFileID != "" {
				bySourceFile[*record.SourceFileID] = record
				if record.KBFileID == nil || *record.KBFileID == "" {
					rawVolumeFallbackFiles[*record.SourceFileID] = struct{}{}
				}
			}
			if record.KBTableID != nil && *record.KBTableID > 0 {
				byTable[*record.KBTableID] = record
			}
			if record.SourceTableID != nil && *record.SourceTableID > 0 {
				byTable[*record.SourceTableID] = record
			}
		}
		sourceLimit := limit - updated
		sourceUpdated, err := s.backfillLegacyCandidateSourcesWithTx(ctx, tx, modelInfo, records, legacyJobs, byKBFile, bySourceFile, byTable, rawVolumeFallbackFiles, domain, actor, sourceLimit)
		if err != nil {
			return err
		}
		updated += sourceUpdated
		jobLimit := limit - updated
		if jobLimit <= 0 {
			return nil
		}
		jobUpdated, err := s.backfillLegacyJobRunsWithTx(tx, legacyJobs, legacyJobRuns, byKBFile, bySourceFile, actor, jobLimit)
		if err != nil {
			return err
		}
		updated += jobUpdated
		return nil
	})
	return updated, err
}

func (s *semanticModelService) backfillLegacyCandidateSourcesWithTx(ctx context.Context, tx *gorm.DB, model *SemanticModelInfo, existing []KnowledgeBaseSourceRecord, legacyJobs []KnowledgeBaseSourceJob, byKBFile map[string]KnowledgeBaseSourceRecord, bySourceFile map[string]KnowledgeBaseSourceRecord, byTable map[int64]KnowledgeBaseSourceRecord, fallbackSourceFiles map[string]struct{}, domain *KnowledgeBaseDataDomain, actor string, limit int) (int, error) {
	if limit <= 0 {
		return 0, nil
	}
	candidates, err := s.collectLegacySourceCandidateRecordsWithJobs(ctx, model, existing, legacyJobs, limit)
	if err != nil {
		return 0, err
	}
	updated := 0
	for _, candidate := range candidates {
		record := candidate.record
		if candidate.origin == SemanticModelSourceLegacyOriginLineage {
			record.Status = kbSourceStatusPending
			record.Error = nil
		}
		if record.KBFileID != nil && *record.KBFileID != "" {
			if _, ok := byKBFile[*record.KBFileID]; ok {
				continue
			}
		}
		if record.SourceFileID != nil && *record.SourceFileID != "" {
			if _, ok := bySourceFile[*record.SourceFileID]; ok {
				continue
			}
			if _, ok := fallbackSourceFiles[*record.SourceFileID]; ok {
				continue
			}
		}
		if record.KBTableID != nil && *record.KBTableID > 0 {
			if _, ok := byTable[*record.KBTableID]; ok {
				continue
			}
		}
		if record.SourceTableID != nil && *record.SourceTableID > 0 {
			if _, ok := byTable[*record.SourceTableID]; ok {
				continue
			}
		}
		if record.ProcessedVolumeID == 0 && domain != nil {
			record.ProcessedVolumeID = domain.ProcessedVolumeID
		}
		inserted, err := insertKnowledgeBaseSourceIdempotentWithTx(tx, &record, actor)
		if err != nil {
			return updated, err
		}
		if !inserted {
			existingRecord, found, err := findKnowledgeBaseSourceByRecordIdentity(tx, &record)
			if err != nil {
				return updated, err
			}
			if found {
				record = existingRecord
			}
		}
		if !inserted && record.SourceID == "" {
			continue
		}
		if isKnowledgeBaseSourceRemoved(record) {
			continue
		}
		if candidate.origin == SemanticModelSourceLegacyOriginLineage {
			fileID := legacyJobFileID(record.SourceFileID, record.KBFileID)
			if fileID == "" {
				continue
			}
			operationID := "lineage_register:" + fileID
			run := newKnowledgeBaseTriggerRAGJob(&record, &operationID, actor)
			run.JobStatus = kbSourceJobSucceeded
			if err := upsertKnowledgeBaseSourceJobRunWithTx(tx, &run, actor); err != nil {
				return updated, err
			}
		}
		if record.KBFileID != nil && *record.KBFileID != "" {
			byKBFile[*record.KBFileID] = record
		}
		if record.SourceFileID != nil && *record.SourceFileID != "" {
			bySourceFile[*record.SourceFileID] = record
		}
		if record.KBTableID != nil && *record.KBTableID > 0 {
			byTable[*record.KBTableID] = record
		}
		if record.SourceTableID != nil && *record.SourceTableID > 0 {
			byTable[*record.SourceTableID] = record
		}
		if inserted {
			updated++
			if updated >= limit {
				return updated, nil
			}
		}
	}
	return updated, nil
}

func (s *semanticModelService) backfillLegacyJobRunsWithTx(tx *gorm.DB, legacyJobs []KnowledgeBaseSourceJob, existingRuns []KnowledgeBaseSourceJobRun, byKBFile map[string]KnowledgeBaseSourceRecord, bySourceFile map[string]KnowledgeBaseSourceRecord, actor string, limit int) (int, error) {
	existingRunKeys := make(map[string]struct{}, len(existingRuns))
	for _, run := range existingRuns {
		fileID := legacyJobFileID(run.SourceFileID, run.KBFileID)
		if fileID != "" {
			existingRunKeys[legacySourceJobRunKey(run.ModelID, fileID, run.WorkflowExecutionID)] = struct{}{}
		}
	}
	updated := 0
	for _, job := range legacyJobs {
		fileID := legacyJobFileID(job.SourceFileID, job.KBFileID)
		if fileID == "" {
			continue
		}
		runKey := legacySourceJobRunKey(job.ModelID, fileID, job.WorkflowExecutionID)
		if _, ok := existingRunKeys[runKey]; ok {
			continue
		}
		source, found := byKBFile[fileID]
		if !found && (job.KBFileID == nil || *job.KBFileID == "") && job.SourceFileID != nil && *job.SourceFileID != "" {
			source, found = bySourceFile[*job.SourceFileID]
		}
		if !found {
			continue
		}
		if isKnowledgeBaseSourceRemoved(source) {
			continue
		}
		if err := updateLegacyBackfillSourceWithTx(tx, source.SourceID, job, actor); err != nil {
			return updated, err
		}
		run := legacyJobRunFromSourceJob(job, source.SourceID)
		if err := upsertKnowledgeBaseSourceJobRunWithTx(tx, &run, actor); err != nil {
			return updated, err
		}
		existingRunKeys[runKey] = struct{}{}
		updated++
		if limit > 0 && updated >= limit {
			return updated, nil
		}
	}
	return updated, nil
}

func (s *semanticModelService) legacySourceCandidateRowsWithJobs(ctx context.Context, model *SemanticModelInfo, existing []KnowledgeBaseSourceRecord, legacyJobs []KnowledgeBaseSourceJob, limit int) ([]SemanticModelSource, error) {
	candidates, err := s.collectLegacySourceCandidateRecordsWithJobs(ctx, model, existing, legacyJobs, limit)
	if err != nil {
		return nil, err
	}
	return legacyCandidateRecordsToSemanticModelSources(candidates), nil
}

func legacyCandidateRecordsToSemanticModelSources(candidates []legacySourceCandidateRecord) []SemanticModelSource {
	out := make([]SemanticModelSource, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, legacySourceCandidateToSemanticModelSource(candidate))
	}
	return out
}

func (s *semanticModelService) collectLegacySourceCandidateRecordsWithJobs(ctx context.Context, model *SemanticModelInfo, existing []KnowledgeBaseSourceRecord, legacyJobs []KnowledgeBaseSourceJob, limit int) ([]legacySourceCandidateRecord, error) {
	if model == nil || model.ID <= 0 {
		return nil, nil
	}
	seen := newLegacyCandidateSeenSet(existing)
	out := make([]legacySourceCandidateRecord, 0)
	add := func(resourceKind, resourceID string, candidate legacySourceCandidateRecord) bool {
		if resourceID == "" {
			return false
		}
		key := resourceKind + ":" + resourceID
		if seen.has(key) {
			return false
		}
		seen.add(key)
		out = append(out, candidate)
		return limit > 0 && len(out) >= limit
	}

	explicit, err := s.explicitSemanticModelSourceRecords(ctx, model, existing)
	if err != nil {
		return nil, err
	}
	for _, record := range explicit {
		if add(recordCandidateResourceKind(record), recordCandidateResourceID(record), legacySourceCandidateRecord{
			record: record,
			origin: SemanticModelSourceLegacyOriginExplicit,
		}) {
			return out, nil
		}
	}

	for _, job := range legacyJobs {
		fileID := legacyJobFileID(job.SourceFileID, job.KBFileID)
		if fileID == "" {
			continue
		}
		if seen.has(legacyCandidateResourceFile + ":" + fileID) {
			continue
		}
		record, err := legacySourceRecordFromJob(ctx, model.ID, job, nil)
		if err != nil {
			var missing knowledgeBaseSourceMissingError
			if errors.As(err, &missing) {
				continue
			}
			return nil, err
		}
		if add(legacyCandidateResourceFile, fileID, legacySourceCandidateRecord{
			record: record,
			origin: SemanticModelSourceLegacyOriginExplicit,
		}) {
			return out, nil
		}
	}

	rawRecords, err := s.rawVolumeLegacySourceCandidateRecords(ctx, model.ID, seen, remainingLegacyCandidateLimit(limit, len(out)))
	if err != nil {
		return nil, err
	}
	for _, candidate := range rawRecords {
		if add(recordCandidateResourceKind(candidate.record), recordCandidateResourceID(candidate.record), candidate) {
			return out, nil
		}
	}

	lineageRecords, err := s.lineageLegacySourceCandidateRecords(ctx, model, seen, remainingLegacyCandidateLimit(limit, len(out)))
	if err != nil {
		return nil, err
	}
	for _, candidate := range lineageRecords {
		if add(recordCandidateResourceKind(candidate.record), recordCandidateResourceID(candidate.record), candidate) {
			return out, nil
		}
	}
	return out, nil
}

type legacyCandidateSeenSet struct {
	keys map[string]struct{}
}

func newLegacyCandidateSeenSet(existing []KnowledgeBaseSourceRecord) *legacyCandidateSeenSet {
	seen := &legacyCandidateSeenSet{keys: make(map[string]struct{}, len(existing)*4)}
	for _, record := range existing {
		if record.KBFileID != nil && *record.KBFileID != "" {
			seen.add(legacyCandidateResourceFile + ":" + *record.KBFileID)
		}
		if record.SourceFileID != nil && *record.SourceFileID != "" {
			seen.add(legacyCandidateResourceFile + ":" + *record.SourceFileID)
		}
		if record.KBTableID != nil && *record.KBTableID > 0 {
			seen.add(legacyCandidateResourceTable + ":" + strconv.FormatInt(*record.KBTableID, 10))
		}
		if record.SourceTableID != nil && *record.SourceTableID > 0 {
			seen.add(legacyCandidateResourceTable + ":" + strconv.FormatInt(*record.SourceTableID, 10))
		}
	}
	return seen
}

func (s *legacyCandidateSeenSet) has(key string) bool {
	_, ok := s.keys[key]
	return ok
}

func (s *legacyCandidateSeenSet) add(key string) {
	if key != "" {
		s.keys[key] = struct{}{}
	}
}

func remainingLegacyCandidateLimit(limit int, used int) int {
	if limit <= 0 {
		return 0
	}
	remaining := limit - used
	if remaining < 0 {
		return 0
	}
	return remaining
}

func recordCandidateResourceKind(record KnowledgeBaseSourceRecord) string {
	if record.SourceType == kbSourceTypeCatalogTable {
		return legacyCandidateResourceTable
	}
	return legacyCandidateResourceFile
}

func recordCandidateResourceID(record KnowledgeBaseSourceRecord) string {
	if record.SourceType == kbSourceTypeCatalogTable {
		if record.KBTableID != nil && *record.KBTableID > 0 {
			return strconv.FormatInt(*record.KBTableID, 10)
		}
		if record.SourceTableID != nil && *record.SourceTableID > 0 {
			return strconv.FormatInt(*record.SourceTableID, 10)
		}
		return ""
	}
	if record.KBFileID != nil && *record.KBFileID != "" {
		return *record.KBFileID
	}
	if record.SourceFileID != nil && *record.SourceFileID != "" {
		return *record.SourceFileID
	}
	return ""
}

func legacySourceCandidateToSemanticModelSource(candidate legacySourceCandidateRecord) SemanticModelSource {
	row := sourceRecordToSemanticModelSource(candidate.record)
	row.RowID = stableID("kb-source-candidate", candidate.record.ModelID, candidate.origin, row.SourceType, row.ResourceID)
	row.SourceID = ""
	row.GovernanceStatus = SemanticModelSourceGovernanceLegacyUnbound
	row.LegacyOrigin = stringPtr(candidate.origin)
	row.Enabled = nil
	row.ExpiresAt = nil
	row.Expired = false
	row.EffectiveEnabled = false
	row.ForceEnabled = false
	row.Tags = nil
	row.SegmentVersionID = nil
	row.IndexVersion = nil
	return row
}

func (s *semanticModelService) rawVolumeLegacySourceCandidateRecords(ctx context.Context, modelID int64, seen *legacyCandidateSeenSet, limit int) ([]legacySourceCandidateRecord, error) {
	if limit == 0 {
		return nil, nil
	}
	volumeIDs, err := knowledgeBaseRawVolumeIDs(ctx, modelID)
	if err != nil {
		return nil, err
	}
	out := make([]legacySourceCandidateRecord, 0)
	for _, volumeID := range volumeIDs {
		offset := 0
		for {
			batchLimit := limit
			if batchLimit <= 0 {
				batchLimit = kbLegacyBackfillBatchSize
			}
			if len(out) > 0 && limit > 0 {
				batchLimit = limit - len(out)
			}
			if batchLimit <= 0 {
				return out, nil
			}
			rawVolumeFiles, err := knowledgeBaseRawVolumeFileMetadataBatch(ctx, volumeID, batchLimit, offset)
			if err != nil {
				return nil, err
			}
			if len(rawVolumeFiles) == 0 {
				break
			}
			for _, file := range rawVolumeFiles {
				key := legacyCandidateResourceFile + ":" + file.fileID
				if seen != nil && seen.has(key) {
					continue
				}
				record, err := rawVolumeFileSourceRecord(modelID, file, nil)
				if err != nil {
					return nil, err
				}
				out = append(out, legacySourceCandidateRecord{
					record: record,
					origin: SemanticModelSourceLegacyOriginExplicit,
				})
				if limit > 0 && len(out) >= limit {
					return out, nil
				}
			}
			if len(rawVolumeFiles) < batchLimit {
				break
			}
			offset += batchLimit
		}
	}
	return out, nil
}

func (s *semanticModelService) lineageLegacySourceCandidateRecords(ctx context.Context, model *SemanticModelInfo, seen *legacyCandidateSeenSet, limit int) ([]legacySourceCandidateRecord, error) {
	if model == nil || model.ID <= 0 || limit == 0 {
		return nil, nil
	}
	vectorTables, err := semanticModelVectorTables(model.Files)
	if err != nil {
		return nil, err
	}
	if len(vectorTables) == 0 {
		return nil, nil
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(vectorTables)), ",")
	args := make([]any, 0, len(vectorTables)+2)
	for _, table := range vectorTables {
		args = append(args, table)
	}
	batchLimit := limit
	if batchLimit <= 0 {
		batchLimit = kbLegacyBackfillBatchSize
	}
	args = append(args, model.ID)
	args = append(args, batchLimit)
	rows, err := db.WithContext(ctx).Raw(fmt.Sprintf(`SELECT DISTINCT COALESCE(pm.source_file_id, root.asset_ref) AS file_id
			FROM data_asset vector
			INNER JOIN data_derivation d ON d.target_asset_id = vector.asset_id AND d.kind = 'indexed_from'
			INNER JOIN data_asset root ON root.asset_id = d.root_asset_id AND root.asset_type = 'file'
			LEFT JOIN parsed_manifest pm ON pm.root_asset_id = d.root_asset_id
		WHERE vector.asset_type = 'vector_index'
		  AND vector.asset_ref IN (%s)
		  AND NOT EXISTS (
		    SELECT 1 FROM knowledge_base_sources kbs
		    WHERE kbs.model_id = ?
		      AND (
		        kbs.kb_file_id = COALESCE(pm.source_file_id, root.asset_ref)
		        OR kbs.source_file_id = COALESCE(pm.source_file_id, root.asset_ref)
		      )
		  )
		ORDER BY file_id
		LIMIT ?`, placeholders), args...).Rows()
	if err != nil {
		return nil, err
	}
	fileIDs, _, err := scanLineageCandidateFileIDs(rows, seen)
	if err != nil {
		return nil, err
	}

	out := make([]legacySourceCandidateRecord, 0, len(fileIDs))
	for _, fileID := range fileIDs {
		record, err := semanticModelFileSourceRecord(ctx, model.ID, fileID)
		if err != nil {
			var missing knowledgeBaseSourceMissingError
			if errors.As(err, &missing) {
				continue
			}
			return nil, err
		}
		out = append(out, legacySourceCandidateRecord{
			record: record,
			origin: SemanticModelSourceLegacyOriginLineage,
		})
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func scanLineageCandidateFileIDs(rows *sql.Rows, seen *legacyCandidateSeenSet) ([]string, int, error) {
	defer rows.Close()
	fileIDs := make([]string, 0)
	scanned := 0
	for rows.Next() {
		var fileID string
		if err := rows.Scan(&fileID); err != nil {
			return nil, scanned, err
		}
		scanned++
		if fileID == "" {
			continue
		}
		key := legacyCandidateResourceFile + ":" + fileID
		if seen != nil && seen.has(key) {
			continue
		}
		fileIDs = append(fileIDs, fileID)
	}
	if err := rows.Err(); err != nil {
		return nil, scanned, err
	}
	return fileIDs, scanned, nil
}

func semanticModelVectorTables(raw json.RawMessage) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var files semanticModelFilesPayload
	if err := json.Unmarshal(raw, &files); err != nil {
		return nil, semanticModelFilesInvalidError()
	}
	seen := make(map[string]struct{}, 2)
	out := make([]string, 0, 2)
	for _, table := range []string{files.VectorTable, files.ImageVectorTable} {
		if table == "" {
			continue
		}
		if _, ok := seen[table]; ok {
			continue
		}
		seen[table] = struct{}{}
		out = append(out, table)
	}
	return out, nil
}

type catalogFileSourceRef struct {
	fileID     string
	catalogID  int64
	databaseID int64
	volumeID   int64
	fileName   string
	sizeBytes  int64
	updatedAt  int64
	path       []string
}

func knowledgeBaseRawVolumeFileMetadata(ctx context.Context, modelID int64) ([]catalogFileSourceRef, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	volumeIDs, err := knowledgeBaseRawVolumeIDs(ctx, modelID)
	if err != nil {
		return nil, err
	}
	if len(volumeIDs) == 0 {
		return []catalogFileSourceRef{}, nil
	}
	files := make([]catalogFileSourceRef, 0)
	for _, volumeID := range volumeIDs {
		batch, err := knowledgeBaseRawVolumeFileMetadataBatch(ctx, volumeID, 0, 0)
		if err != nil {
			return nil, err
		}
		files = append(files, batch...)
	}
	return files, nil
}

func knowledgeBaseRawVolumeFileMetadataBatch(ctx context.Context, volumeID int64, limit int, offset int) ([]catalogFileSourceRef, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	query := `SELECT vf.file_id, COALESCE(CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END, 0) AS catalog_id, COALESCE(v.database_id, 0) AS database_id, vf.volume_id, f.size, UNIX_TIMESTAMP(COALESCE(vf.updated_at, f.updated_at)) AS updated_at, COALESCE(c.catalog_name, '') AS catalog_name, COALESCE(cd.database_name, '') AS database_name, COALESCE(v.volume_name, '') AS volume_name, COALESCE(vf.file_name, f.original_name) AS file_name
		FROM volume_files vf
		INNER JOIN volume v ON v.volume_id = vf.volume_id AND v.deleted = FALSE
		INNER JOIN ` + "`file`" + ` f ON f.file_id = vf.file_id
		LEFT JOIN catalog_database cd ON cd.database_id = v.database_id
		LEFT JOIN catalog c ON c.catalog_id = CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END
		WHERE vf.volume_id = ?
		  AND COALESCE(CASE WHEN v.catalog_id > 0 THEN v.catalog_id ELSE cd.catalog_id END, 0) > 0
		  AND COALESCE(v.database_id, 0) > 0
		  AND COALESCE(v.volume_name, '') <> ''
		  AND COALESCE(c.catalog_name, '') <> ''
		  AND COALESCE(cd.database_name, '') <> ''
		ORDER BY vf.updated_at DESC, vf.id DESC`
	args := []any{volumeID}
	if limit > 0 {
		query += ` LIMIT ? OFFSET ?`
		args = append(args, limit, offset)
	}
	rows, err := db.WithContext(ctx).Raw(query, args...).Rows()
	if err != nil {
		return nil, err
	}
	files := make([]catalogFileSourceRef, 0)
	for rows.Next() {
		var file catalogFileSourceRef
		var catalogName, databaseName, volumeName string
		if err := rows.Scan(&file.fileID, &file.catalogID, &file.databaseID, &file.volumeID, &file.sizeBytes, &file.updatedAt, &catalogName, &databaseName, &volumeName, &file.fileName); err != nil {
			_ = rows.Close()
			return nil, err
		}
		if file.fileID == "" {
			_ = rows.Close()
			return nil, fmt.Errorf("knowledge base raw volume %d returned empty file id", volumeID)
		}
		if file.fileName == "" {
			_ = rows.Close()
			return nil, fmt.Errorf("knowledge base raw volume file %s has empty display name", file.fileID)
		}
		if file.catalogID <= 0 || file.databaseID <= 0 || file.volumeID <= 0 || catalogName == "" || databaseName == "" || volumeName == "" {
			continue
		}
		file.path = compactNonEmptyStrings(catalogName, databaseName, volumeName)
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return files, nil
}

func semanticModelExplicitSourcesBackfillRequired(model *SemanticModelInfo, records []KnowledgeBaseSourceRecord) bool {
	if model == nil {
		return false
	}
	existingFiles := make(map[string]struct{}, len(records)*2)
	existingTables := make(map[int64]struct{}, len(records)*2)
	existingTableNames := make(map[string]struct{}, len(records))
	for _, record := range records {
		if record.KBFileID != nil && *record.KBFileID != "" {
			existingFiles[*record.KBFileID] = struct{}{}
		}
		if record.SourceFileID != nil && *record.SourceFileID != "" {
			existingFiles[*record.SourceFileID] = struct{}{}
		}
		if record.KBTableID != nil && *record.KBTableID > 0 {
			existingTables[*record.KBTableID] = struct{}{}
		}
		if record.SourceTableID != nil && *record.SourceTableID > 0 {
			existingTables[*record.SourceTableID] = struct{}{}
		}
		if record.DBName != nil && record.TableName != nil && *record.DBName != "" && *record.TableName != "" {
			existingTableNames[semanticModelTableKey(*record.DBName, *record.TableName)] = struct{}{}
		}
	}
	files, err := semanticModelFileIDs(model.Files)
	if err == nil {
		for _, fileID := range files {
			if _, ok := existingFiles[fileID]; !ok {
				return true
			}
		}
	}
	tables, err := semanticModelTableSources(model.ID, model.Tables)
	if err != nil {
		return false
	}
	for _, source := range tables {
		if source.DBName == nil || source.TableName == nil {
			continue
		}
		if _, ok := existingTableNames[semanticModelTableKey(*source.DBName, *source.TableName)]; !ok {
			return true
		}
		if source.ResourceID != "" {
			tableID, err := strconv.ParseInt(source.ResourceID, 10, 64)
			if err == nil && tableID > 0 {
				if _, ok := existingTables[tableID]; !ok {
					return true
				}
			}
		}
	}
	return false
}

func (s *semanticModelService) explicitSemanticModelSourceRecords(ctx context.Context, model *SemanticModelInfo, existing []KnowledgeBaseSourceRecord) ([]KnowledgeBaseSourceRecord, error) {
	if model == nil || model.ID <= 0 {
		return nil, nil
	}
	existingFiles := make(map[string]struct{}, len(existing)*2)
	existingTables := make(map[int64]struct{}, len(existing)*2)
	existingTableNames := make(map[string]struct{}, len(existing))
	for _, record := range existing {
		if record.KBFileID != nil && *record.KBFileID != "" {
			existingFiles[*record.KBFileID] = struct{}{}
		}
		if record.SourceFileID != nil && *record.SourceFileID != "" {
			existingFiles[*record.SourceFileID] = struct{}{}
		}
		if record.KBTableID != nil && *record.KBTableID > 0 {
			existingTables[*record.KBTableID] = struct{}{}
		}
		if record.SourceTableID != nil && *record.SourceTableID > 0 {
			existingTables[*record.SourceTableID] = struct{}{}
		}
		if record.DBName != nil && record.TableName != nil && *record.DBName != "" && *record.TableName != "" {
			existingTableNames[semanticModelTableKey(*record.DBName, *record.TableName)] = struct{}{}
		}
	}
	records := make([]KnowledgeBaseSourceRecord, 0)
	files, err := semanticModelFileIDs(model.Files)
	if err != nil {
		return nil, err
	}
	for _, fileID := range files {
		if _, ok := existingFiles[fileID]; ok {
			continue
		}
		record, err := semanticModelFileSourceRecord(ctx, model.ID, fileID)
		if err != nil {
			var missing knowledgeBaseSourceMissingError
			if errors.As(err, &missing) {
				continue
			}
			return nil, err
		}
		records = append(records, record)
		existingFiles[fileID] = struct{}{}
	}
	tables, err := semanticModelTableSources(model.ID, model.Tables)
	if err != nil {
		return nil, err
	}
	for _, source := range tables {
		if source.DBName == nil || source.TableName == nil {
			continue
		}
		if _, ok := existingTableNames[semanticModelTableKey(*source.DBName, *source.TableName)]; ok {
			continue
		}
		table, err := s.resolveCatalogTableByName(ctx, *source.DBName, *source.TableName)
		if err != nil {
			var missing knowledgeBaseSourceMissingError
			if errors.As(err, &missing) {
				continue
			}
			return nil, fmt.Errorf("resolve semantic model table source %s.%s: %w", *source.DBName, *source.TableName, err)
		}
		if _, ok := existingTables[table.tableID]; ok {
			continue
		}
		record, err := semanticModelTableSourceRecord(model.ID, table)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
		existingTables[table.tableID] = struct{}{}
		existingTableNames[semanticModelTableKey(table.dbName, table.tableName)] = struct{}{}
	}
	return records, nil
}

func semanticModelFileSourceRecord(ctx context.Context, modelID int64, fileID string) (KnowledgeBaseSourceRecord, error) {
	meta, err := currentCatalogFileMetadata(ctx, fileID)
	if err != nil {
		return KnowledgeBaseSourceRecord{}, fmt.Errorf("resolve semantic model file source %s: %w", fileID, err)
	}
	sourcePath, err := json.Marshal(meta.path)
	if err != nil {
		return KnowledgeBaseSourceRecord{}, fmt.Errorf("marshal file source path: %w", err)
	}
	enabled := true
	return KnowledgeBaseSourceRecord{
		SourceID:     stableID("kb-source", modelID, kbSourceTypeCatalogFile, fileID),
		ModelID:      modelID,
		CatalogID:    meta.catalogID,
		DatabaseID:   meta.databaseID,
		RawVolumeID:  meta.volumeID,
		SourceType:   kbSourceTypeCatalogFile,
		SourceFileID: stringPtr(fileID),
		KBFileID:     stringPtr(fileID),
		DisplayName:  stringPtr(meta.fileName),
		SourcePath:   stringPtr(string(sourcePath)),
		Status:       kbSourceStatusSucceeded,
		Enabled:      &enabled,
		Tags:         stringPtr("[]"),
	}, nil
}

func semanticModelTableSourceRecord(modelID int64, table catalogTableSourceRef) (KnowledgeBaseSourceRecord, error) {
	sourcePath, err := json.Marshal(table.path)
	if err != nil {
		return KnowledgeBaseSourceRecord{}, fmt.Errorf("marshal table source path: %w", err)
	}
	enabled := true
	return KnowledgeBaseSourceRecord{
		SourceID:      stableID("kb-source", modelID, kbSourceTypeCatalogTable, table.tableID),
		ModelID:       modelID,
		CatalogID:     table.catalogID,
		DatabaseID:    table.databaseID,
		SourceType:    kbSourceTypeCatalogTable,
		SourceTableID: int64Ptr(table.tableID),
		DisplayName:   stringPtr(table.tableName),
		SourcePath:    stringPtr(string(sourcePath)),
		DBName:        stringPtr(table.dbName),
		TableName:     stringPtr(table.tableName),
		Status:        kbSourceStatusSucceeded,
		Enabled:       &enabled,
		Tags:          stringPtr("[]"),
	}, nil
}

func rawVolumeFileSourceRecord(modelID int64, file catalogFileSourceRef, domain *KnowledgeBaseDataDomain) (KnowledgeBaseSourceRecord, error) {
	sourcePath, err := json.Marshal(file.path)
	if err != nil {
		return KnowledgeBaseSourceRecord{}, fmt.Errorf("marshal raw volume file source path: %w", err)
	}
	processedVolumeID := int64(0)
	if domain != nil {
		processedVolumeID = domain.ProcessedVolumeID
	}
	enabled := true
	tags := "[]"
	return KnowledgeBaseSourceRecord{
		SourceID:          stableID("kb-source", modelID, kbSourceTypeLocalFile, file.fileID),
		ModelID:           modelID,
		CatalogID:         file.catalogID,
		DatabaseID:        file.databaseID,
		RawVolumeID:       file.volumeID,
		ProcessedVolumeID: processedVolumeID,
		SourceType:        kbSourceTypeLocalFile,
		SourceFileID:      stringPtr(file.fileID),
		KBFileID:          stringPtr(file.fileID),
		DisplayName:       stringPtr(file.fileName),
		SourcePath:        stringPtr(string(sourcePath)),
		Status:            kbSourceStatusSucceeded,
		Enabled:           &enabled,
		Tags:              &tags,
	}, nil
}

func knowledgeBaseRawVolumeIDs(ctx context.Context, modelID int64) ([]int64, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, fmt.Errorf("tenant db is required")
	}
	seen := map[int64]struct{}{}
	addRows := func(rows *sql.Rows) error {
		defer rows.Close()
		for rows.Next() {
			var volumeID int64
			if err := rows.Scan(&volumeID); err != nil {
				return err
			}
			if volumeID > 0 {
				seen[volumeID] = struct{}{}
			}
		}
		return rows.Err()
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT raw_volume_id
		FROM knowledge_base_raw_volumes
		WHERE model_id = ? AND raw_volume_id > 0
		  AND COALESCE(raw_kind, '') <> '`+kbRawKindStructured+`'`, modelID).Rows()
	if err != nil {
		return nil, err
	}
	if err := addRows(rows); err != nil {
		return nil, err
	}
	rows, err = db.WithContext(ctx).Raw(`SELECT raw_volume_id
		FROM knowledge_base_data_domains
		WHERE model_id = ? AND raw_volume_id > 0`, modelID).Rows()
	if err != nil {
		return nil, err
	}
	if err := addRows(rows); err != nil {
		return nil, err
	}
	volumeIDs := make([]int64, 0, len(seen))
	for volumeID := range seen {
		volumeIDs = append(volumeIDs, volumeID)
	}
	sort.Slice(volumeIDs, func(i, j int) bool { return volumeIDs[i] < volumeIDs[j] })
	return volumeIDs, nil
}

func knowledgeBaseOwnedVolumeIDs(ctx context.Context, modelID int64, domain *KnowledgeBaseDataDomain) ([]int64, error) {
	seen := map[int64]struct{}{}
	if domain != nil {
		for _, volumeID := range []int64{domain.RawVolumeID, domain.ProcessedVolumeID} {
			if volumeID > 0 {
				seen[volumeID] = struct{}{}
			}
		}
	}
	rawVolumeIDs, err := knowledgeBaseRawVolumeIDs(ctx, modelID)
	if err != nil {
		return nil, err
	}
	for _, volumeID := range rawVolumeIDs {
		if volumeID > 0 {
			seen[volumeID] = struct{}{}
		}
	}
	volumeIDs := make([]int64, 0, len(seen))
	for volumeID := range seen {
		volumeIDs = append(volumeIDs, volumeID)
	}
	sort.Slice(volumeIDs, func(i, j int) bool { return volumeIDs[i] < volumeIDs[j] })
	return volumeIDs, nil
}

type catalogTableSourceRef struct {
	tableID    int64
	databaseID int64
	catalogID  int64
	dbName     string
	tableName  string
	path       []string
}

func (s *semanticModelService) resolveCatalogTableByID(ctx context.Context, tableID int64) (catalogTableSourceRef, error) {
	if tableID <= 0 {
		return catalogTableSourceRef{}, fmt.Errorf("table_id is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return catalogTableSourceRef{}, fmt.Errorf("tenant db is required")
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT t.table_id, t.database_id, t.catalog_id, COALESCE(cd.database_name, ''), t.table_name, COALESCE(c.catalog_name, '')
		FROM catalog_table t
		INNER JOIN catalog_database cd ON cd.database_id = t.database_id
		LEFT JOIN catalog c ON c.catalog_id = t.catalog_id
		WHERE t.table_id = ?
		LIMIT 1`, tableID).Rows()
	if err != nil {
		return catalogTableSourceRef{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return catalogTableSourceRef{}, knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog table %d not found", tableID)}
	}
	var ref catalogTableSourceRef
	var catalogName string
	if err := rows.Scan(&ref.tableID, &ref.databaseID, &ref.catalogID, &ref.dbName, &ref.tableName, &catalogName); err != nil {
		return catalogTableSourceRef{}, err
	}
	if err := rows.Err(); err != nil {
		return catalogTableSourceRef{}, err
	}
	if ref.dbName == "" || ref.tableName == "" {
		return catalogTableSourceRef{}, knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog table %d has incomplete metadata", tableID)}
	}
	ref.path = compactNonEmptyStrings(catalogName, ref.dbName)
	return ref, nil
}

func (s *semanticModelService) resolveCatalogTableByName(ctx context.Context, dbName, tableName string) (catalogTableSourceRef, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return catalogTableSourceRef{}, fmt.Errorf("tenant db is required")
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT t.table_id, t.database_id, t.catalog_id, COALESCE(cd.database_name, ''), COALESCE(c.catalog_name, '')
		FROM catalog_table t
		INNER JOIN catalog_database cd ON cd.database_id = t.database_id
		LEFT JOIN catalog c ON c.catalog_id = t.catalog_id
		WHERE cd.database_name = ? AND t.table_name = ?`,
		dbName, tableName).Rows()
	if err != nil {
		return catalogTableSourceRef{}, err
	}
	defer rows.Close()
	matches := make([]catalogTableSourceRef, 0, 2)
	for rows.Next() {
		var ref catalogTableSourceRef
		var catalogName string
		if err := rows.Scan(&ref.tableID, &ref.databaseID, &ref.catalogID, &ref.dbName, &catalogName); err != nil {
			return catalogTableSourceRef{}, err
		}
		ref.tableName = tableName
		ref.path = compactNonEmptyStrings(catalogName, ref.dbName)
		matches = append(matches, ref)
	}
	if err := rows.Err(); err != nil {
		return catalogTableSourceRef{}, err
	}
	if len(matches) == 0 {
		return catalogTableSourceRef{}, knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog table %s.%s not found", dbName, tableName)}
	}
	if len(matches) > 1 {
		return catalogTableSourceRef{}, knowledgeBaseSourceMissingError{msg: fmt.Sprintf("catalog table %s.%s is ambiguous", dbName, tableName)}
	}
	return matches[0], nil
}

func (s *semanticModelService) resolveCatalogTableByDatabaseAndName(ctx context.Context, databaseID int64, dbName, tableName string) (catalogTableSourceRef, error) {
	if tableName == "" {
		return catalogTableSourceRef{}, fmt.Errorf("table_name is required")
	}
	if databaseID <= 0 && dbName == "" {
		return catalogTableSourceRef{}, fmt.Errorf("database_id or db_name is required")
	}
	if databaseID > 0 {
		ref, found, err := s.resolveCatalogTableByCondition(ctx, "t.database_id = ? AND t.table_name = ?", databaseID, tableName)
		if err != nil || found {
			return ref, err
		}
		return s.resolveSingleCatalogTableByCondition(ctx, "t.database_id = ? AND LOWER(t.table_name) = LOWER(?)", databaseID, tableName)
	}
	ref, found, err := s.resolveCatalogTableByCondition(ctx, "cd.database_name = ? AND t.table_name = ?", dbName, tableName)
	if err != nil || found {
		return ref, err
	}
	return s.resolveSingleCatalogTableByCondition(ctx, "cd.database_name = ? AND LOWER(t.table_name) = LOWER(?)", dbName, tableName)
}

func (s *semanticModelService) resolveSingleCatalogTableByCondition(ctx context.Context, where string, args ...any) (catalogTableSourceRef, error) {
	ref, found, err := s.resolveCatalogTableByCondition(ctx, where, args...)
	if err != nil {
		return catalogTableSourceRef{}, err
	}
	if !found {
		return catalogTableSourceRef{}, knowledgeBaseSourceMissingError{msg: "catalog table not found"}
	}
	return ref, nil
}

func (s *semanticModelService) resolveCatalogTableByCondition(ctx context.Context, where string, args ...any) (catalogTableSourceRef, bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return catalogTableSourceRef{}, false, fmt.Errorf("tenant db is required")
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT t.table_id, t.database_id, t.catalog_id, COALESCE(cd.database_name, ''), t.table_name, COALESCE(c.catalog_name, '')
		FROM catalog_table t
		INNER JOIN catalog_database cd ON cd.database_id = t.database_id
		LEFT JOIN catalog c ON c.catalog_id = t.catalog_id
		WHERE `+where, args...).Rows()
	if err != nil {
		return catalogTableSourceRef{}, false, err
	}
	defer rows.Close()
	matches := make([]catalogTableSourceRef, 0, 2)
	for rows.Next() {
		var ref catalogTableSourceRef
		var catalogName string
		if err := rows.Scan(&ref.tableID, &ref.databaseID, &ref.catalogID, &ref.dbName, &ref.tableName, &catalogName); err != nil {
			return catalogTableSourceRef{}, false, err
		}
		ref.path = compactNonEmptyStrings(catalogName, ref.dbName)
		matches = append(matches, ref)
	}
	if err := rows.Err(); err != nil {
		return catalogTableSourceRef{}, false, err
	}
	if len(matches) == 0 {
		return catalogTableSourceRef{}, false, nil
	}
	if len(matches) > 1 {
		return catalogTableSourceRef{}, false, knowledgeBaseSourceMissingError{msg: "catalog table is ambiguous"}
	}
	return matches[0], true, nil
}

func legacyJobRunFromSourceJob(job KnowledgeBaseSourceJob, sourceID string) KnowledgeBaseSourceJobRun {
	fileID := legacyJobFileID(job.SourceFileID, job.KBFileID)
	jobID := stableID("kb-job-legacy", job.ID, job.ModelID, fileID, ptrValue(job.WorkflowExecutionID))
	if job.ID == 0 {
		jobID = stableID("kb-job-legacy", job.ModelID, fileID, ptrValue(job.WorkflowExecutionID))
	}
	kbFileID := job.KBFileID
	if kbFileID == nil || *kbFileID == "" {
		kbFileID = job.SourceFileID
	}
	return KnowledgeBaseSourceJobRun{
		JobID:               jobID,
		SourceID:            sourceID,
		ModelID:             job.ModelID,
		JobType:             kbJobTypeRAGIngest,
		JobStatus:           job.JobStatus,
		IdempotencyKey:      stableID("kb-job-legacy-key", job.ModelID, fileID, ptrValue(job.WorkflowExecutionID)),
		OperationID:         stringPtr("legacy_source_job:" + strconv.FormatInt(job.ID, 10)),
		WorkflowExecutionID: job.WorkflowExecutionID,
		SourceFileID:        job.SourceFileID,
		KBFileID:            kbFileID,
		Error:               job.Error,
	}
}

func legacyJobFileID(sourceFileID, kbFileID *string) string {
	if kbFileID != nil && *kbFileID != "" {
		return *kbFileID
	}
	return ptrValue(sourceFileID)
}

func legacySourceJobRunKey(modelID int64, kbFileID string, workflowExecutionID *string) string {
	return fmt.Sprintf("%d:%s:%s", modelID, kbFileID, ptrValue(workflowExecutionID))
}

func legacySourceRecordFromJob(ctx context.Context, modelID int64, job KnowledgeBaseSourceJob, domain *KnowledgeBaseDataDomain) (KnowledgeBaseSourceRecord, error) {
	fileID := legacyJobFileID(job.SourceFileID, job.KBFileID)
	if fileID == "" {
		return KnowledgeBaseSourceRecord{}, fmt.Errorf("legacy knowledge base source job %d has no file id", job.ID)
	}
	meta, err := currentCatalogFileMetadata(ctx, fileID)
	if err != nil {
		return KnowledgeBaseSourceRecord{}, fmt.Errorf("resolve legacy knowledge base file %s: %w", fileID, err)
	}
	sourcePath, err := json.Marshal(meta.path)
	if err != nil {
		return KnowledgeBaseSourceRecord{}, fmt.Errorf("marshal legacy file source path: %w", err)
	}
	enabled := true
	tags := "[]"
	sourceType := job.SourceType
	if sourceType == "" {
		sourceType = kbSourceTypeCatalogFile
	}
	if sourceType != kbSourceTypeLocalFile && sourceType != kbSourceTypeCatalogFile {
		sourceType = kbSourceTypeCatalogFile
	}
	processedVolumeID := int64(0)
	if domain != nil {
		processedVolumeID = domain.ProcessedVolumeID
	}
	rawVolumeID := job.RawVolumeID
	if rawVolumeID <= 0 {
		rawVolumeID = meta.volumeID
	}
	return KnowledgeBaseSourceRecord{
		SourceID:          stableID("kb-source", modelID, sourceType, fileID),
		ModelID:           modelID,
		CatalogID:         meta.catalogID,
		DatabaseID:        meta.databaseID,
		RawVolumeID:       rawVolumeID,
		ProcessedVolumeID: processedVolumeID,
		SourceType:        sourceType,
		SourceFileID:      firstNonEmptyStringPtr(job.SourceFileID, &fileID),
		KBFileID:          &fileID,
		DisplayName:       stringPtr(meta.fileName),
		SourcePath:        stringPtr(string(sourcePath)),
		Status:            job.JobStatus,
		Error:             job.Error,
		Enabled:           &enabled,
		Tags:              &tags,
		IndexVersion:      job.IndexVersion,
	}, nil
}

func firstNonEmptyStringPtr(values ...*string) *string {
	for _, value := range values {
		if value != nil && *value != "" {
			return value
		}
	}
	return nil
}

func updateLegacyBackfillSourceWithTx(tx *gorm.DB, sourceID string, job KnowledgeBaseSourceJob, actor string) error {
	return tx.Exec(`UPDATE knowledge_base_sources
		SET source_file_id = COALESCE(source_file_id, ?), kb_file_id = COALESCE(kb_file_id, ?), raw_volume_id = CASE WHEN raw_volume_id = 0 THEN ? ELSE raw_volume_id END, status = ?, error = ?, index_version = COALESCE(index_version, ?), updated_by = ?
		WHERE source_id = ?`,
		firstNonEmptyStringPtr(job.SourceFileID, job.KBFileID), job.KBFileID, job.RawVolumeID, job.JobStatus, job.Error, job.IndexVersion, actor, sourceID).Error
}
