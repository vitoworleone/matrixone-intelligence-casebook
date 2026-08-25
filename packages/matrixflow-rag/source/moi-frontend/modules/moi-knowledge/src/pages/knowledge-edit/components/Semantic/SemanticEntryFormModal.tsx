import { useEffect } from 'react';
import { Form, Input, Modal, Select } from 'antd';
import type { TFunction } from 'i18next';

import type { SemanticEntry, SemanticKind } from '@moi/shared-moi-api/knowledge';
import { SpecFormRenderer } from './spec-forms/SpecFormRenderer';

const ALL_KINDS: SemanticKind[] = [
  'dimension',
  'fact',
  'metric',
  'relationship',
  'column_preference',
  'named_filter',
  'verified_query',
  'glossary',
  'logic_text',
  'sql_resultset',
];

const KIND_LABEL_MAP = {
  dimension: 'knowledge.entry.kind-dimension',
  fact: 'knowledge.entry.kind-fact',
  metric: 'knowledge.entry.kind-metric',
  relationship: 'knowledge.entry.kind-relationship',
  column_preference: 'knowledge.entry.kind-column-preference',
  named_filter: 'knowledge.entry.kind-named-filter',
  verified_query: 'knowledge.entry.kind-verified-query',
  glossary: 'knowledge.entry.kind-glossary',
  logic_text: 'knowledge.entry.kind-logic-text',
  sql_resultset: 'knowledge.entry.kind-sql-resultset',
} as const satisfies Record<SemanticKind, string>;

interface SemanticEntryFormModalProps {
  open: boolean;
  loading: boolean;
  editEntry: SemanticEntry | null;
  defaultKind?: SemanticKind;
  availableTables: string[];
  t: TFunction<'moi-knowledge'>;
  onOk: (values: { kind: SemanticKind; key: string; tables: string[]; spec: Record<string, unknown> }) => void;
  onCancel: () => void;
}

export function SemanticEntryFormModal({
  open,
  loading,
  editEntry,
  defaultKind,
  availableTables,
  t,
  onOk,
  onCancel,
}: SemanticEntryFormModalProps) {
  const [form] = Form.useForm();
  const selectedKind = Form.useWatch('kind', form) as SemanticKind | undefined;

  useEffect(() => {
    if (open) {
      if (editEntry) {
        form.setFieldsValue({
          kind: editEntry.kind,
          key: editEntry.key,
          tables: editEntry.tables ?? [],
          spec: editEntry.spec,
        });
      } else {
        form.resetFields();
        if (defaultKind) {
          form.setFieldsValue({ kind: defaultKind });
        }
      }
    }
  }, [open, editEntry, defaultKind, form]);

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      onOk({
        kind: values.kind,
        key: values.key,
        tables: values.tables || [],
        spec: normalizeSpec(values.kind, values.spec || {}),
      });
    } catch {
      // validation errors shown by form
    }
  };

  return (
    <Modal
      title={editEntry ? t('knowledge.entry.modal-title-edit') : t('knowledge.entry.modal-title-create')}
      open={open}
      onOk={handleOk}
      onCancel={onCancel}
      confirmLoading={loading}
      width={640}
      destroyOnClose
      data-testid="semantic-entry-form-modal"
    >
      <Form form={form} layout="vertical" autoComplete="off">
        <Form.Item name="kind" label={t('knowledge.entry.form-kind-label')} rules={[{ required: true }]}>
          <Select
            placeholder={t('knowledge.entry.form-kind-placeholder')}
            disabled
            options={ALL_KINDS.map((k) => ({ value: k, label: t(KIND_LABEL_MAP[k]) }))}
          />
        </Form.Item>

        <Form.Item
          name="key"
          label={t('knowledge.entry.form-key-label')}
          rules={[{ required: true, message: t('knowledge.entry.form-key-required') }]}
        >
          <Input placeholder={t('knowledge.entry.form-key-placeholder')} />
        </Form.Item>

        <Form.Item name="tables" label={t('knowledge.entry.form-tables-label')}>
          <Select
            mode="multiple"
            placeholder={t('knowledge.entry.form-tables-placeholder')}
            options={availableTables.map((tbl) => ({ value: tbl, label: tbl }))}
          />
        </Form.Item>

        {selectedKind && <SpecFormRenderer kind={selectedKind} t={t} availableTables={availableTables} />}
      </Form>
    </Modal>
  );
}

function normalizeSpec(kind: SemanticKind, spec: Record<string, unknown>) {
  if (kind !== 'sql_resultset') {
    return spec;
  }

  const next = { ...spec };
  delete next.display_name;

  if (!next.resolve_mode) {
    delete next.resolve_mode;
  }
  if (next.resolve_mode === 'passthrough') {
    delete next.expand_sql;
  }

  for (const key of ['max_rows', 'max_bytes', 'timeout_seconds']) {
    if (next[key] === undefined || next[key] === null || next[key] === '') {
      delete next[key];
    }
  }

  const expandSQL = next.expand_sql as { sql?: string; params?: string[] } | undefined;
  if (expandSQL) {
    const sql = String(expandSQL.sql ?? '').trim();
    const params = Array.isArray(expandSQL.params) ? expandSQL.params.filter((item) => String(item).trim()) : [];
    if (!sql && params.length === 0) {
      delete next.expand_sql;
    } else {
      next.expand_sql = { sql, ...(params.length > 0 ? { params } : {}) };
    }
  }

  const retrieval = next.retrieval as { embedding_model?: string } | undefined;
  const embeddingModel = String(retrieval?.embedding_model ?? '').trim();
  if (embeddingModel) {
    next.retrieval = {
      enabled: true,
      embedding_model: embeddingModel,
    };
  } else {
    delete next.retrieval;
  }

  return next;
}
