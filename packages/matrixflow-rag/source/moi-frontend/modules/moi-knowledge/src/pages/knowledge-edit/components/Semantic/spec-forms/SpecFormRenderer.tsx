import { useEffect, useMemo, useState } from 'react';
import { Button, Checkbox, Form, Input, InputNumber, Select, Space, Tooltip } from 'antd';
import { MinusCircleOutlined, PlusOutlined, QuestionCircleOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';

import type { InjectionStage, SemanticKind } from '@moi/shared-moi-api/knowledge';
import { decodeModelValue, encodeModelValue, listModelsApi, type ModelInfo } from '@moi/shared-moi-api/model';
import { useHttpClient } from '@moi/shared-moi-app-protocol/app-context';

interface SpecFormRendererProps {
  kind: SemanticKind;
  t: TFunction<'moi-knowledge'>;
  availableTables: string[];
}

const INJECTION_STAGE_LABEL_MAP = {
  planner_policy: 'knowledge.entry.injection-stage-planner-policy',
  sql_generation: 'knowledge.entry.injection-stage-sql-generation',
  sql_followup: 'knowledge.entry.injection-stage-sql-followup',
  sql_regenerate: 'knowledge.entry.injection-stage-sql-regenerate',
  sql_decomposition: 'knowledge.entry.injection-stage-sql-decomposition',
  executor_rule: 'knowledge.entry.injection-stage-executor-rule',
  renderer_rule: 'knowledge.entry.injection-stage-renderer-rule',
} as const satisfies Record<InjectionStage, string>;

const SEMANTIC_PATTERN_OPTIONS = [
  { value: 'ratio', label: 'Ratio' },
  { value: 'semi_anti_join', label: 'Semi/Anti Join' },
  { value: 'window', label: 'Window' },
];

/** Renders spec-specific form fields based on the selected kind */
export function SpecFormRenderer({ kind, t, availableTables }: SpecFormRendererProps) {
  switch (kind) {
    case 'dimension':
      return <DimensionFields t={t} />;
    case 'fact':
      return <FactFields t={t} />;
    case 'metric':
      return <MetricFields t={t} availableTables={availableTables} />;
    case 'relationship':
      return <RelationshipFields t={t} availableTables={availableTables} />;
    case 'column_preference':
      return <ColumnPreferenceFields t={t} />;
    case 'named_filter':
      return <NamedFilterFields t={t} availableTables={availableTables} />;
    case 'verified_query':
      return <VerifiedQueryFields t={t} />;
    case 'glossary':
      return <GlossaryFields t={t} />;
    case 'logic_text':
      return <LogicTextFields t={t} />;
    case 'sql_resultset':
      return <SQLResultsetFields t={t} />;
    default:
      return null;
  }
}

type FieldProps = { t: TFunction<'moi-knowledge'> };
type FieldPropsWithTables = FieldProps & { availableTables: string[] };

type EmbeddingModelOption = { value: string; label: string };

function formatEmbeddingModelLabel(model: ModelInfo): string {
  return model.backend_name ? `${model.backend_name} / ${model.model}` : model.model;
}

function normalizeEmbeddingModelOptions(models: ModelInfo[]): EmbeddingModelOption[] {
  const seen = new Set<string>();
  const result: EmbeddingModelOption[] = [];
  for (const item of models) {
    const model = item.model.trim();
    if (!model) continue;
    const key = encodeModelValue(model, item.backend_id);
    if (seen.has(key)) continue;
    seen.add(key);
    result.push({ value: key, label: formatEmbeddingModelLabel({ ...item, model }) });
  }
  return result.sort((left, right) => left.label.localeCompare(right.label));
}

function DimensionFields({ t }: FieldProps) {
  return (
    <>
      <Form.Item
        name={['spec', 'column']}
        label={t('knowledge.entry.spec-column-label')}
        rules={[{ required: true, message: t('knowledge.entry.spec-column-placeholder') }]}
      >
        <Input placeholder={t('knowledge.entry.spec-column-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'data_type']} label={t('knowledge.entry.spec-data-type-label')}>
        <Input placeholder={t('knowledge.entry.spec-data-type-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'synonyms']} label={t('knowledge.entry.spec-synonyms-label')}>
        <Select mode="tags" placeholder={t('knowledge.entry.spec-synonyms-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'description']} label={t('knowledge.entry.spec-description-label')}>
        <Input.TextArea rows={2} placeholder={t('knowledge.entry.spec-description-placeholder')} />
      </Form.Item>
      <Space size="middle">
        <Form.Item name={['spec', 'is_enum']} valuePropName="checked">
          <Checkbox>{t('knowledge.entry.spec-is-enum')}</Checkbox>
        </Form.Item>
        <Form.Item name={['spec', 'is_time']} valuePropName="checked">
          <Checkbox>{t('knowledge.entry.spec-is-time')}</Checkbox>
        </Form.Item>
        <Form.Item name={['spec', 'deprecated']} valuePropName="checked">
          <Checkbox>{t('knowledge.entry.spec-deprecated')}</Checkbox>
        </Form.Item>
      </Space>
      <Form.Item noStyle shouldUpdate={(prev, cur) => prev?.spec?.is_enum !== cur?.spec?.is_enum}>
        {({ getFieldValue }) =>
          getFieldValue(['spec', 'is_enum']) ? (
            <Form.Item name={['spec', 'sample_values']} label={t('knowledge.entry.spec-sample-values-label')}>
              <Select mode="tags" placeholder={t('knowledge.entry.spec-sample-values-placeholder')} />
            </Form.Item>
          ) : null
        }
      </Form.Item>
    </>
  );
}

function FactFields({ t }: FieldProps) {
  return (
    <>
      <Form.Item
        name={['spec', 'column']}
        label={t('knowledge.entry.spec-column-label')}
        rules={[{ required: true, message: t('knowledge.entry.spec-column-placeholder') }]}
      >
        <Input placeholder={t('knowledge.entry.spec-column-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'data_type']} label={t('knowledge.entry.spec-data-type-label')}>
        <Input placeholder={t('knowledge.entry.spec-data-type-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'description']} label={t('knowledge.entry.spec-description-label')}>
        <Input.TextArea rows={2} placeholder={t('knowledge.entry.spec-description-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'private']} valuePropName="checked">
        <Checkbox>{t('knowledge.entry.spec-private')}</Checkbox>
      </Form.Item>
    </>
  );
}

function MetricFields({ t, availableTables }: FieldPropsWithTables) {
  return (
    <>
      <Form.Item
        name={['spec', 'expr']}
        label={t('knowledge.entry.spec-expr-label')}
        rules={[{ required: true, message: t('knowledge.entry.spec-expr-required') }]}
      >
        <Input placeholder={t('knowledge.entry.spec-expr-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'synonyms']} label={t('knowledge.entry.spec-synonyms-label')}>
        <Select mode="tags" placeholder={t('knowledge.entry.spec-synonyms-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'unit']} label={t('knowledge.entry.spec-unit-label')}>
        <Input placeholder={t('knowledge.entry.spec-unit-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'description']} label={t('knowledge.entry.spec-description-label')}>
        <Input.TextArea rows={2} placeholder={t('knowledge.entry.spec-description-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'requires_tables']} label={t('knowledge.entry.spec-requires-tables-label')}>
        <Select
          mode="multiple"
          placeholder={t('knowledge.entry.spec-requires-tables-label')}
          options={availableTables.map((tbl) => ({ value: tbl, label: tbl }))}
        />
      </Form.Item>
      <Form.Item name={['spec', 'requires_join']} label={t('knowledge.entry.spec-requires-join-label')}>
        <Input placeholder={t('knowledge.entry.spec-requires-join-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'semantic_pattern']} label={t('knowledge.entry.spec-semantic-pattern-label')}>
        <Select allowClear options={SEMANTIC_PATTERN_OPTIONS} />
      </Form.Item>
    </>
  );
}

function RelationshipFields({ t, availableTables }: FieldPropsWithTables) {
  const tableOptions = availableTables.map((tbl) => ({ value: tbl, label: tbl }));
  return (
    <>
      <Form.Item name={['spec', 'left_table']} label={t('knowledge.entry.spec-left-table-label')} rules={[{ required: true }]}>
        <Select placeholder={t('knowledge.entry.spec-left-table-placeholder')} options={tableOptions} />
      </Form.Item>
      <Form.Item name={['spec', 'right_table']} label={t('knowledge.entry.spec-right-table-label')} rules={[{ required: true }]}>
        <Select placeholder={t('knowledge.entry.spec-right-table-placeholder')} options={tableOptions} />
      </Form.Item>
      <Form.Item label={t('knowledge.entry.spec-join-columns-label')} required>
        <Form.List name={['spec', 'join_columns']} initialValue={[{ left: '', right: '' }]}>
          {(fields, { add, remove }) => (
            <>
              {fields.map(({ key, name, ...restField }) => (
                <Space key={key} align="baseline" style={{ display: 'flex', marginBottom: 8 }}>
                  <Form.Item {...restField} name={[name, 'left']} rules={[{ required: true }]} noStyle>
                    <Input placeholder={t('knowledge.entry.spec-join-left-placeholder')} />
                  </Form.Item>
                  <Form.Item {...restField} name={[name, 'right']} rules={[{ required: true }]} noStyle>
                    <Input placeholder={t('knowledge.entry.spec-join-right-placeholder')} />
                  </Form.Item>
                  {fields.length > 1 && <MinusCircleOutlined onClick={() => remove(name)} />}
                </Space>
              ))}
              <Button type="dashed" onClick={() => add({ left: '', right: '' })} icon={<PlusOutlined />}>
                {t('knowledge.entry.spec-join-add')}
              </Button>
            </>
          )}
        </Form.List>
      </Form.Item>
      <Form.Item name={['spec', 'description']} label={t('knowledge.entry.spec-description-label')}>
        <Input.TextArea rows={2} placeholder={t('knowledge.entry.spec-description-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'semantic_match']} valuePropName="checked">
        <Checkbox>{t('knowledge.entry.spec-semantic-match')}</Checkbox>
      </Form.Item>
    </>
  );
}

function ColumnPreferenceFields({ t }: FieldProps) {
  return (
    <>
      <Form.Item name={['spec', 'preferred']} label={t('knowledge.entry.spec-preferred-label')} rules={[{ required: true }]}>
        <Input placeholder={t('knowledge.entry.spec-preferred-placeholder')} />
      </Form.Item>
      <Form.Item
        name={['spec', 'deprecated']}
        label={t('knowledge.entry.spec-deprecated-col-label')}
        rules={[{ required: true }]}
      >
        <Input placeholder={t('knowledge.entry.spec-deprecated-col-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'reason']} label={t('knowledge.entry.spec-reason-label')}>
        <Input placeholder={t('knowledge.entry.spec-reason-placeholder')} />
      </Form.Item>
    </>
  );
}

function NamedFilterFields({ t, availableTables }: FieldPropsWithTables) {
  return (
    <>
      <Form.Item
        name={['spec', 'expr']}
        label={t('knowledge.entry.spec-filter-expr-label')}
        rules={[{ required: true, message: t('knowledge.entry.spec-filter-expr-required') }]}
      >
        <Input.TextArea rows={2} placeholder={t('knowledge.entry.spec-filter-expr-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'synonyms']} label={t('knowledge.entry.spec-synonyms-label')}>
        <Select mode="tags" placeholder={t('knowledge.entry.spec-synonyms-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'description']} label={t('knowledge.entry.spec-description-label')}>
        <Input.TextArea rows={2} placeholder={t('knowledge.entry.spec-description-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'applies_to']} label={t('knowledge.entry.spec-applies-to-label')}>
        <Select mode="multiple" options={availableTables.map((tbl) => ({ value: tbl, label: tbl }))} />
      </Form.Item>
    </>
  );
}

function VerifiedQueryFields({ t }: FieldProps) {
  return (
    <>
      <Form.Item
        name={['spec', 'question']}
        label={t('knowledge.entry.spec-question-label')}
        rules={[{ required: true, message: t('knowledge.entry.spec-question-required') }]}
      >
        <Input.TextArea rows={2} placeholder={t('knowledge.entry.spec-question-placeholder')} />
      </Form.Item>
      <Form.Item
        name={['spec', 'sql']}
        label={t('knowledge.entry.spec-sql-label')}
        rules={[{ required: true, message: t('knowledge.entry.spec-sql-required') }]}
      >
        <Input.TextArea rows={4} placeholder={t('knowledge.entry.spec-sql-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'verified_by']} label={t('knowledge.entry.spec-verified-by-label')}>
        <Input />
      </Form.Item>
      <Form.Item name={['spec', 'tags']} label={t('knowledge.entry.spec-tags-label')}>
        <Select mode="tags" placeholder={t('knowledge.entry.spec-tags-placeholder')} />
      </Form.Item>
    </>
  );
}

function GlossaryFields({ t }: FieldProps) {
  return (
    <>
      <Form.Item
        name={['spec', 'term']}
        label={t('knowledge.entry.spec-term-label')}
        rules={[{ required: true, message: t('knowledge.entry.spec-term-required') }]}
      >
        <Input placeholder={t('knowledge.entry.spec-term-placeholder')} />
      </Form.Item>
      <Form.Item
        name={['spec', 'definition']}
        label={t('knowledge.entry.spec-definition-label')}
        rules={[{ required: true, message: t('knowledge.entry.spec-definition-required') }]}
      >
        <Input.TextArea rows={3} placeholder={t('knowledge.entry.spec-definition-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'synonyms']} label={t('knowledge.entry.spec-synonyms-label')}>
        <Select mode="tags" placeholder={t('knowledge.entry.spec-synonyms-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'related_metrics']} label={t('knowledge.entry.spec-related-metrics-label')}>
        <Select mode="tags" placeholder={t('knowledge.entry.spec-related-metrics-placeholder')} />
      </Form.Item>
      <Form.Item name={['spec', 'formula_hint']} label={t('knowledge.entry.spec-formula-hint-label')}>
        <Input placeholder={t('knowledge.entry.spec-formula-hint-placeholder')} />
      </Form.Item>
    </>
  );
}

function LogicTextFields({ t }: FieldProps) {
  return (
    <>
      <Form.Item
        name={['spec', 'content']}
        label={t('knowledge.entry.spec-content-label')}
        rules={[{ required: true, message: t('knowledge.entry.spec-content-required') }]}
      >
        <Input.TextArea rows={4} placeholder={t('knowledge.entry.spec-content-placeholder')} />
      </Form.Item>
      <Form.Item
        name={['spec', 'injection_stages']}
        label={t('knowledge.entry.spec-injection-stages-label')}
        rules={[{ required: true, type: 'array', min: 1, message: t('knowledge.entry.spec-injection-stages-required') }]}
      >
        <Select
          mode="multiple"
          options={Object.entries(INJECTION_STAGE_LABEL_MAP).map(([value, labelKey]) => ({
            value,
            label: t(labelKey),
          }))}
        />
      </Form.Item>
      <Form.Item name={['spec', 'priority']} label={t('knowledge.entry.spec-priority-label')}>
        <InputNumber placeholder={t('knowledge.entry.spec-priority-placeholder')} style={{ width: '100%' }} />
      </Form.Item>
    </>
  );
}

function SQLResultsetFields({ t }: FieldProps) {
  const http = useHttpClient();
  const [embeddingModelOptions, setEmbeddingModelOptions] = useState<EmbeddingModelOption[]>([]);
  const [embeddingModelLoading, setEmbeddingModelLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setEmbeddingModelLoading(true);
    listModelsApi(http, { type: 'embedding' })
      .then((res) => {
        if (cancelled) {
          return;
        }
        setEmbeddingModelOptions(normalizeEmbeddingModelOptions(res.data?.models ?? []));
      })
      .catch((error: unknown) => {
        if (cancelled) {
          return;
        }
        console.warn('[SQLResultsetFields] load embedding models failed', { msg: (error as Error).message });
        setEmbeddingModelOptions([]);
      })
      .finally(() => {
        if (!cancelled) {
          setEmbeddingModelLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [http]);

  const embeddingModelSelectOptions = useMemo(
    () => embeddingModelOptions.map((item) => ({ value: item.value, label: item.label })),
    [embeddingModelOptions],
  );

  return (
    <>
      <Form.Item
        name={['spec', 'sql']}
        label={t('knowledge.entry.spec-resultset-sql-label')}
        rules={[{ required: true, message: t('knowledge.entry.spec-resultset-sql-required') }]}
      >
        <Input.TextArea rows={5} placeholder={t('knowledge.entry.spec-resultset-sql-placeholder')} />
      </Form.Item>
      <Form.Item
        name={['spec', 'description']}
        label={t('knowledge.entry.spec-resultset-description-label')}
        rules={[
          { required: true, message: t('knowledge.entry.spec-resultset-description-required') },
          { max: 1000, message: t('knowledge.entry.spec-resultset-description-max') },
        ]}
      >
        <Input.TextArea
          rows={3}
          maxLength={1000}
          showCount
          placeholder={t('knowledge.entry.spec-resultset-description-placeholder')}
        />
      </Form.Item>
      <Form.Item
        name={['spec', 'resolve_mode']}
        label={
          <Space size={4}>
            {t('knowledge.entry.spec-resultset-resolve-mode-label')}
            <Tooltip title={t('knowledge.entry.spec-resultset-resolve-mode-tooltip')}>
              <QuestionCircleOutlined style={{ color: 'rgba(0,0,0,0.45)' }} />
            </Tooltip>
          </Space>
        }
      >
        <Select
          allowClear
          placeholder={t('knowledge.entry.spec-resultset-resolve-mode-placeholder')}
          options={[
            { value: 'semantic', label: t('knowledge.entry.spec-resultset-resolve-mode-semantic') },
            { value: 'passthrough', label: t('knowledge.entry.spec-resultset-resolve-mode-passthrough') },
          ]}
        />
      </Form.Item>
      <Space size="small" style={{ display: 'flex' }}>
        <Form.Item name={['spec', 'max_rows']} label={t('knowledge.entry.spec-resultset-max-rows-label')}>
          <InputNumber
            min={1}
            precision={0}
            placeholder={t('knowledge.entry.spec-resultset-max-rows-placeholder')}
            style={{ width: '100%' }}
          />
        </Form.Item>
        <Form.Item name={['spec', 'max_bytes']} label={t('knowledge.entry.spec-resultset-max-bytes-label')}>
          <InputNumber
            min={1}
            precision={0}
            placeholder={t('knowledge.entry.spec-resultset-max-bytes-placeholder')}
            style={{ width: '100%' }}
          />
        </Form.Item>
        <Form.Item name={['spec', 'timeout_seconds']} label={t('knowledge.entry.spec-resultset-timeout-label')}>
          <InputNumber
            min={0}
            max={60}
            precision={0}
            placeholder={t('knowledge.entry.spec-resultset-timeout-placeholder')}
            style={{ width: '100%' }}
          />
        </Form.Item>
      </Space>
      <Form.Item
        name={['spec', 'retrieval', 'embedding_model']}
        label={
          <Space size={4}>
            {t('knowledge.entry.spec-resultset-embedding-model-label')}
            <Tooltip title={t('knowledge.entry.spec-resultset-embedding-model-tooltip')}>
              <QuestionCircleOutlined style={{ color: 'rgba(0,0,0,0.45)' }} />
            </Tooltip>
          </Space>
        }
        normalize={(val: string | undefined) => {
          if (!val) return val;
          return decodeModelValue(val).model;
        }}
        getValueProps={(val: string | undefined) => {
          if (!val) return { value: val };
          const match = embeddingModelSelectOptions.find((opt) => decodeModelValue(opt.value).model === val);
          return { value: match?.value ?? val };
        }}
      >
        <Select
          allowClear
          showSearch
          loading={embeddingModelLoading}
          placeholder={t('knowledge.entry.spec-resultset-embedding-model-placeholder')}
          optionFilterProp="label"
          options={embeddingModelSelectOptions}
        />
      </Form.Item>
      <Form.Item noStyle shouldUpdate={(prev, cur) => prev?.spec?.resolve_mode !== cur?.spec?.resolve_mode}>
        {({ getFieldValue }) => {
          if (getFieldValue(['spec', 'resolve_mode']) === 'passthrough') return null;
          return (
            <>
              <Form.Item
                name={['spec', 'expand_sql', 'sql']}
                label={t('knowledge.entry.spec-resultset-expand-sql-label')}
                dependencies={[['spec', 'expand_sql', 'params']]}
                rules={[
                  ({ getFieldValue: gfv }) => ({
                    validator(_, value) {
                      const params = gfv(['spec', 'expand_sql', 'params']) as string[] | undefined;
                      if (!String(value ?? '').trim() && params?.length) {
                        return Promise.reject(new Error(t('knowledge.entry.spec-resultset-expand-sql-required')));
                      }
                      return Promise.resolve();
                    },
                  }),
                ]}
              >
                <Input.TextArea rows={3} placeholder={t('knowledge.entry.spec-resultset-expand-sql-placeholder')} />
              </Form.Item>
              <Form.Item
                name={['spec', 'expand_sql', 'params']}
                label={t('knowledge.entry.spec-resultset-expand-params-label')}
                dependencies={[['spec', 'expand_sql', 'sql']]}
                rules={[
                  ({ getFieldValue: gfv }) => ({
                    validator(_, value) {
                      const sql = String(gfv(['spec', 'expand_sql', 'sql']) ?? '').trim();
                      if (sql && (!Array.isArray(value) || value.length === 0)) {
                        return Promise.reject(new Error(t('knowledge.entry.spec-resultset-expand-params-required')));
                      }
                      return Promise.resolve();
                    },
                  }),
                ]}
              >
                <Select mode="tags" placeholder={t('knowledge.entry.spec-resultset-expand-params-placeholder')} />
              </Form.Item>
            </>
          );
        }}
      </Form.Item>
    </>
  );
}
