import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Menu } from 'antd';

import type { SemanticKind } from '@moi/shared-moi-api/knowledge';
import styles from './KnowledgeAdvancedConfig.module.css';
import { SemanticEntrySetting } from './Semantic/SemanticEntrySetting';

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

type KnowledgeAdvancedConfigProps = {
  knowledgeId?: number;
  visible?: boolean;
};

/**
 * Semantic configuration panel for a knowledge base.
 * Renders a left-side menu of 9 semantic kinds and the entry list on the right.
 */
export default function KnowledgeAdvancedConfig({ knowledgeId, visible = true }: KnowledgeAdvancedConfigProps) {
  const { t } = useTranslation('moi-knowledge');
  const [activeKind, setActiveKind] = useState<SemanticKind>('dimension');

  const canUpdate = true;

  const menuItems = ALL_KINDS.map((kind) => ({
    key: kind,
    label: t(KIND_LABEL_MAP[kind]),
  }));

  return (
    <div className={styles.panel}>
      <Menu
        onClick={({ key }) => setActiveKind(key as SemanticKind)}
        className={styles.menu}
        defaultSelectedKeys={['dimension']}
        mode="inline"
        items={menuItems}
        selectedKeys={[activeKind]}
      />
      <div className={styles.content}>
        {knowledgeId ? (
          <SemanticEntrySetting modelId={knowledgeId} activeKind={activeKind} visible={visible} canUpdate={canUpdate} />
        ) : null}
      </div>
    </div>
  );
}
