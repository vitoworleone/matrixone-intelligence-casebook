import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, App, Button, Input, Modal, Space, Table } from 'antd';
import { PlusOutlined } from '@ant-design/icons';

import type { SemanticEntry, SemanticKind } from '@moi/shared-moi-api/knowledge';
import { useHttpClient, useTimezone } from '@moi/shared-moi-app-protocol/app-context';
import { extractApiErrorMessage } from '../../../../service/http';
import {
  createSemanticEntry,
  deleteSemanticEntry,
  getSemanticModel,
  updateSemanticEntry,
} from '../../../../service/semanticModel';
import { useSemanticColumns } from './hooks/useSemanticColumns';
import { useSemanticEntries } from './hooks/useSemanticEntries';
import { SemanticEntryFormModal } from './SemanticEntryFormModal';
import styles from './SemanticEntrySetting.module.css';
import { SemanticModelToolbar } from './SemanticModelToolbar';

interface SemanticEntrySettingProps {
  modelId: number;
  activeKind?: SemanticKind;
  visible: boolean;
  canUpdate: boolean;
}

export function SemanticEntrySetting({ modelId, activeKind, visible, canUpdate }: SemanticEntrySettingProps) {
  const { t } = useTranslation('moi-knowledge');
  const { message: appMessage } = App.useApp();
  const http = useHttpClient();
  const timezone = useTimezone();

  const [modalOpen, setModalOpen] = useState(false);
  const [editEntry, setEditEntry] = useState<SemanticEntry | null>(null);
  const [saving, setSaving] = useState(false);
  const [availableTables, setAvailableTables] = useState<string[]>([]);
  const [hasDataSource, setHasDataSource] = useState(true);

  const { entries, total, loading, hasMore, refresh, loadMore } = useSemanticEntries({
    http,
    modelId,
    kind: activeKind,
    pageSize: 20,
  });

  // Load available tables and files from semantic model
  useEffect(() => {
    if (!visible || !modelId) return;
    let cancelled = false;

    const load = async () => {
      try {
        const model = await getSemanticModel(http, modelId);
        if (cancelled) return;
        const tableNames = (model.tables || []).flatMap((t) => t.table_names || []);
        const fileIds = model.files?.file_ids || [];
        setAvailableTables(tableNames);
        // 表和文件都没有时才认为没有数据源
        setHasDataSource(tableNames.length > 0 || fileIds.length > 0);
      } catch {
        if (!cancelled) {
          setAvailableTables([]);
          setHasDataSource(false);
        }
      }
    };

    load();
    return () => {
      cancelled = true;
    };
  }, [visible, modelId, http]);

  const handleEdit = useCallback((entry: SemanticEntry) => {
    setEditEntry(entry);
    setModalOpen(true);
  }, []);

  const handleDelete = useCallback(
    (entry: SemanticEntry) => {
      Modal.confirm({
        title: t('knowledge.semantic.delete-confirm-title'),
        content: t('knowledge.semantic.delete-confirm-content'),
        okText: t('knowledge.semantic.delete-confirm-ok'),
        okButtonProps: { danger: true },
        cancelText: t('knowledge.semantic.delete-confirm-cancel'),
        async onOk() {
          try {
            await deleteSemanticEntry(http, modelId, entry.id);
            appMessage.success(t('knowledge.semantic.delete-success'));
            refresh();
          } catch (error) {
            appMessage.error(extractApiErrorMessage(error, t('knowledge.semantic.form-save-errorGeneric')));
          }
        },
      });
    },
    [http, modelId, appMessage, t, refresh],
  );

  const columns = useSemanticColumns({ t, timezone, onEdit: handleEdit, onDelete: handleDelete, canUpdate });

  const handleFormOk = useCallback(
    async (values: { kind: SemanticKind; key: string; tables: string[]; spec: Record<string, unknown> }) => {
      setSaving(true);
      try {
        const body = { kind: values.kind, key: values.key, tables: values.tables, spec: values.spec as never };
        if (editEntry) {
          await updateSemanticEntry(http, modelId, editEntry.id, body);
          appMessage.success(t('knowledge.semantic.form-save-successUpdate'));
        } else {
          await createSemanticEntry(http, modelId, body);
          appMessage.success(t('knowledge.semantic.form-save-successCreate'));
        }
        setModalOpen(false);
        setEditEntry(null);
        refresh();
      } catch (error) {
        const msg = extractApiErrorMessage(error, t('knowledge.semantic.form-save-errorGeneric'));
        // Handle 409 conflict specifically
        const errObj = error as { response?: { status?: number } };
        if (errObj?.response?.status === 409) {
          appMessage.error(t('knowledge.entry.error-conflict'));
        } else {
          appMessage.error(msg);
        }
      } finally {
        setSaving(false);
      }
    },
    [editEntry, http, modelId, appMessage, t, refresh],
  );

  if (!visible) return null;

  return (
    <div className={styles.container} data-testid="semantic-entry-setting">
      {!hasDataSource && (
        <Alert type="warning" showIcon title={t('knowledge.entry.no-tables-warning')} className={styles.alert} />
      )}

      <div className={styles.toolbar}>
        <Space>
          <Input.Search
            placeholder={t('knowledge.entry.search-placeholder')}
            onSearch={() => refresh()}
            allowClear
            style={{ width: 240 }}
          />
          {canUpdate && (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              disabled={!hasDataSource}
              onClick={() => {
                setEditEntry(null);
                setModalOpen(true);
              }}
              data-testid="semantic-entry-add-btn"
            >
              {t('knowledge.entry.add-entry')}
            </Button>
          )}
        </Space>
        {canUpdate && <SemanticModelToolbar http={http} modelId={modelId} t={t} onImportSuccess={refresh} />}
      </div>

      <Table<SemanticEntry>
        rowKey="id"
        dataSource={entries}
        columns={columns}
        loading={loading}
        pagination={false}
        size="middle"
        scroll={{ x: 'max-content' }}
        data-testid="semantic-entry-table"
      />

      {hasMore && (
        <div className={styles.loadMore}>
          <Button onClick={loadMore} loading={loading}>
            {t('knowledge.entry.load-more')}
          </Button>
          <span className={styles.totalHint}>{t('knowledge.entry.total-count', { total })}</span>
        </div>
      )}

      <SemanticEntryFormModal
        open={modalOpen}
        loading={saving}
        editEntry={editEntry}
        defaultKind={activeKind}
        availableTables={availableTables}
        t={t}
        onOk={handleFormOk}
        onCancel={() => {
          setModalOpen(false);
          setEditEntry(null);
        }}
      />
    </div>
  );
}
