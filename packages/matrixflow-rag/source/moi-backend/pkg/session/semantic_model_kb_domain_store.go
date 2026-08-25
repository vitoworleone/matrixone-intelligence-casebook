package session

import (
	"context"
	"fmt"
	"time"

	"github.com/matrixorigin/matrixflow/moi-backend/pkg/ctxutil"
)

// knowledge base data-domain / raw-volume SQL persistence.
// Orchestration (catalog Core, provision, workflow) stays in semantic_model_kb_domain.go.

func (s *semanticModelService) updateKnowledgeBaseDataDomainCatalog(ctx context.Context, modelID, catalogID int64, actor string) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	result := db.WithContext(ctx).Exec(`UPDATE knowledge_base_data_domains
		SET catalog_id = ?, updated_by = ?
		WHERE model_id = ?`, catalogID, actor, modelID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return knowledgeBaseDataDomainCatalogRepairFailedError(fmt.Errorf("knowledge base data domain %d no longer exists", modelID))
	}
	return nil
}

// upsertKnowledgeBaseDataDomain inserts the initial domain row only.
// It never UPDATE-overwrites an existing row (avoids clobbering a concurrent
// claimer's provisioning/ready + resource IDs). Callers must treat duplicate
// insert as "row already exists" and re-read.
func (s *semanticModelService) upsertKnowledgeBaseDataDomain(ctx context.Context, domain *KnowledgeBaseDataDomain, actor string) error {
	if domain == nil {
		return fmt.Errorf("knowledge base data domain is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	return db.WithContext(ctx).Exec(`INSERT INTO knowledge_base_data_domains
		(model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at, created_by, updated_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		domain.ModelID, domain.CatalogID, domain.DatabaseID, domain.RawVolumeID, domain.ProcessedVolumeID, domain.EnsureStatus, domain.LastEnsureError, domain.LastCheckedAt, actor, actor).Error
}

func (s *semanticModelService) updateKnowledgeBaseDataDomain(ctx context.Context, domain *KnowledgeBaseDataDomain, actor string) error {
	return s.updateKnowledgeBaseDataDomainIfStatus(ctx, domain, actor, "")
}

// updateKnowledgeBaseDataDomainIfStatus writes the domain row. When expectedStatus
// is non-empty, the update is conditional (CAS) on ensure_status matching.
func (s *semanticModelService) updateKnowledgeBaseDataDomainIfStatus(ctx context.Context, domain *KnowledgeBaseDataDomain, actor, expectedStatus string) error {
	if domain == nil {
		return fmt.Errorf("knowledge base data domain is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	var (
		err          error
		rowsAffected int64
	)
	if expectedStatus == "" {
		result := db.WithContext(ctx).Exec(`UPDATE knowledge_base_data_domains
			SET catalog_id = ?, database_id = ?, raw_volume_id = ?, processed_volume_id = ?, ensure_status = ?, last_ensure_error = ?, last_checked_at = ?, updated_by = ?
			WHERE model_id = ?`,
			domain.CatalogID, domain.DatabaseID, domain.RawVolumeID, domain.ProcessedVolumeID, domain.EnsureStatus, domain.LastEnsureError, domain.LastCheckedAt, actor, domain.ModelID)
		err = result.Error
		rowsAffected = result.RowsAffected
	} else {
		result := db.WithContext(ctx).Exec(`UPDATE knowledge_base_data_domains
			SET catalog_id = ?, database_id = ?, raw_volume_id = ?, processed_volume_id = ?, ensure_status = ?, last_ensure_error = ?, last_checked_at = ?, updated_by = ?
			WHERE model_id = ? AND ensure_status = ?`,
			domain.CatalogID, domain.DatabaseID, domain.RawVolumeID, domain.ProcessedVolumeID, domain.EnsureStatus, domain.LastEnsureError, domain.LastCheckedAt, actor, domain.ModelID, expectedStatus)
		err = result.Error
		rowsAffected = result.RowsAffected
	}
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		if expectedStatus != "" {
			return errKnowledgeBaseDataDomainCASFailed
		}
		return knowledgeBaseDataDomainCatalogRepairFailedError(fmt.Errorf("knowledge base data domain %d no longer exists", domain.ModelID))
	}
	return nil
}

// claimKnowledgeBaseDataDomainProvision atomically moves failed → provisioning.
// Returns false when another request already claimed or finished the domain.
func (s *semanticModelService) claimKnowledgeBaseDataDomainProvision(ctx context.Context, modelID int64, actor string) (bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return false, fmt.Errorf("tenant db is required")
	}
	result := db.WithContext(ctx).Exec(`UPDATE knowledge_base_data_domains
		SET ensure_status = ?, last_ensure_error = NULL, last_checked_at = ?, updated_by = ?
		WHERE model_id = ? AND ensure_status = ?`,
		kbEnsureStatusProvisioning, time.Now().Unix(), actor, modelID, kbEnsureStatusFailed)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (s *semanticModelService) getKnowledgeBaseDataDomain(ctx context.Context, modelID int64) (*KnowledgeBaseDataDomain, bool, error) {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return nil, false, nil
	}
	rows, err := db.WithContext(ctx).Raw(`SELECT model_id, catalog_id, database_id, raw_volume_id, processed_volume_id, ensure_status, last_ensure_error, last_checked_at
		FROM knowledge_base_data_domains
		WHERE model_id = ?
		LIMIT 1`, modelID).Rows()
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, false, nil
	}
	var domain KnowledgeBaseDataDomain
	if err := rows.Scan(&domain.ModelID, &domain.CatalogID, &domain.DatabaseID, &domain.RawVolumeID, &domain.ProcessedVolumeID, &domain.EnsureStatus, &domain.LastEnsureError, &domain.LastCheckedAt); err != nil {
		return nil, false, err
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return &domain, true, nil
}

func (s *semanticModelService) deleteKnowledgeBaseRows(ctx context.Context, modelID int64) error {
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	if err := db.WithContext(ctx).Exec(`DELETE FROM knowledge_base_chunk_recall_stats WHERE model_id = ?`, modelID).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(`DELETE FROM knowledge_base_segments WHERE model_id = ?`, modelID).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(`DELETE FROM knowledge_base_segment_versions WHERE model_id = ?`, modelID).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(`DELETE FROM knowledge_base_source_job_runs WHERE model_id = ?`, modelID).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(`DELETE FROM knowledge_base_sources WHERE model_id = ?`, modelID).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(`DELETE FROM knowledge_base_raw_volumes WHERE model_id = ?`, modelID).Error; err != nil {
		return err
	}
	if err := db.WithContext(ctx).Exec(`DELETE FROM knowledge_base_source_jobs WHERE model_id = ?`, modelID).Error; err != nil {
		return err
	}
	return db.WithContext(ctx).Exec(`DELETE FROM knowledge_base_data_domains WHERE model_id = ?`, modelID).Error
}

func (s *semanticModelService) upsertKnowledgeBaseRawVolume(ctx context.Context, domain *KnowledgeBaseDataDomain, rawKind string, rawVolumeID int64, ensureStatus string, ensureError *string, actor string) error {
	if domain == nil {
		return fmt.Errorf("knowledge base data domain is required")
	}
	if rawKind == "" {
		return fmt.Errorf("raw_kind is required")
	}
	db := ctxutil.TenantDBFrom(ctx)
	if db == nil {
		return fmt.Errorf("tenant db is required")
	}
	var count int64
	if err := db.WithContext(ctx).Table("knowledge_base_raw_volumes").
		Where("model_id = ? AND raw_kind = ?", domain.ModelID, rawKind).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return db.WithContext(ctx).Exec(`INSERT INTO knowledge_base_raw_volumes
			(model_id, catalog_id, database_id, raw_kind, raw_volume_id, ensure_status, last_ensure_error, last_checked_at, created_by, updated_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			domain.ModelID, domain.CatalogID, domain.DatabaseID, rawKind, rawVolumeID, ensureStatus, ensureError, time.Now().Unix(), actor, actor).Error
	}
	return db.WithContext(ctx).Exec(`UPDATE knowledge_base_raw_volumes
		SET catalog_id = ?, database_id = ?, raw_volume_id = ?, ensure_status = ?, last_ensure_error = ?, last_checked_at = ?, updated_by = ?
		WHERE model_id = ? AND raw_kind = ?`,
		domain.CatalogID, domain.DatabaseID, rawVolumeID, ensureStatus, ensureError, time.Now().Unix(), actor, domain.ModelID, rawKind).Error
}
