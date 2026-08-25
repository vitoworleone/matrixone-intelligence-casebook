import { useMemo } from 'react';
import { Space, Tag, Tooltip, type TableColumnsType } from 'antd';
import type { TFunction } from 'i18next';

import type { SemanticEntry, SemanticKind } from '@moi/shared-moi-api/knowledge';
import { ListActionButton } from '@moi/shared-moi-components/list-action';
import { formatDateTime } from '@moi/shared-moi-utils/datetime';

interface UseSemanticColumnsOptions {
  t: TFunction<'moi-knowledge'>;
  timezone: string;
  onEdit: (entry: SemanticEntry) => void;
  onDelete: (entry: SemanticEntry) => void;
  canUpdate: boolean;
}

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

const KIND_COLOR_MAP: Record<SemanticKind, string> = {
  dimension: 'blue',
  fact: 'cyan',
  metric: 'green',
  relationship: 'orange',
  column_preference: 'purple',
  named_filter: 'magenta',
  verified_query: 'geekblue',
  glossary: 'gold',
  logic_text: 'volcano',
  sql_resultset: 'lime',
};

/** Extract a short summary from the spec for display in the table */
function getSpecSummary(entry: SemanticEntry): string {
  const spec = entry.spec as unknown as Record<string, unknown>;
  switch (entry.kind) {
    case 'dimension':
    case 'fact':
      return String(spec.column || '');
    case 'metric':
      return String(spec.expr || '');
    case 'relationship':
      return `${spec.left_table || ''} ↔ ${spec.right_table || ''}`;
    case 'column_preference':
      return `${spec.deprecated || ''} → ${spec.preferred || ''}`;
    case 'named_filter':
      return String(spec.expr || '');
    case 'verified_query':
      return String(spec.question || '');
    case 'glossary':
      return String(spec.term || '');
    case 'logic_text':
      return String(spec.content || '').slice(0, 60);
    case 'sql_resultset':
      return String(spec.sql || '').slice(0, 60);
    default:
      return '';
  }
}

export function useSemanticColumns({
  t,
  timezone,
  onEdit,
  onDelete,
  canUpdate,
}: UseSemanticColumnsOptions): TableColumnsType<SemanticEntry> {
  return useMemo(
    () => [
      {
        title: t('knowledge.entry.columns-key'),
        dataIndex: 'key',
        key: 'key',
        width: 200,
        ellipsis: true,
      },
      {
        title: t('knowledge.entry.columns-kind'),
        dataIndex: 'kind',
        key: 'kind',
        width: 120,
        render: (kind: SemanticKind) => <Tag color={KIND_COLOR_MAP[kind]}>{t(KIND_LABEL_MAP[kind])}</Tag>,
      },
      {
        title: t('knowledge.entry.columns-tables'),
        dataIndex: 'tables',
        key: 'tables',
        width: 180,
        ellipsis: true,
        render: (tables: string[]) =>
          tables?.length > 0 ? (
            <Tooltip title={tables.join(', ')}>
              <span>{tables.join(', ')}</span>
            </Tooltip>
          ) : (
            '-'
          ),
      },
      {
        title: t('knowledge.entry.columns-summary'),
        key: 'summary',
        ellipsis: true,
        render: (_: unknown, entry: SemanticEntry) => {
          const summary = getSpecSummary(entry);
          return summary ? (
            <Tooltip title={summary}>
              <span>{summary}</span>
            </Tooltip>
          ) : (
            '-'
          );
        },
      },
      {
        title: t('knowledge.entry.columns-updated-at'),
        dataIndex: 'updated_at',
        key: 'updated_at',
        width: 180,
        render: (ts: number) => (ts ? formatDateTime(ts * 1000, { timezone }) : '-'),
      },
      ...(canUpdate
        ? [
            {
              title: t('knowledge.entry.columns-actions'),
              key: 'actions',
              width: 96,
              render: (_: unknown, entry: SemanticEntry) => (
                <Space size="small">
                  <ListActionButton
                    action="edit"
                    label={t('knowledge.semantic.action-edit')}
                    onClick={() => onEdit(entry)}
                    data-testid={`semantic-entry-edit-btn-${entry.id}`}
                  />
                  <ListActionButton
                    action="delete"
                    danger
                    label={t('knowledge.semantic.action-delete')}
                    onClick={() => onDelete(entry)}
                    data-testid={`semantic-entry-delete-btn-${entry.id}`}
                  />
                </Space>
              ),
            },
          ]
        : []),
    ],
    [t, timezone, onEdit, onDelete, canUpdate],
  );
}
