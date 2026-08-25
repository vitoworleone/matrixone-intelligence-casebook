import { useRef } from 'react';
import { Popover, Tooltip } from 'antd';
import { FolderOpenFilled } from '@ant-design/icons';

import styles from './KnowledgeSummary.module.css';

export interface KnowledgeSummaryItem {
  id: number;
  name: string;
}

interface KnowledgeSummaryProps {
  items: KnowledgeSummaryItem[];
}

function KnowledgeTag({ item }: { item: KnowledgeSummaryItem }) {
  return (
    <Tooltip title={item.name} placement="top">
      <div className={styles.knowledgeTag} data-testid={`knowledge-summary-tag-${item.id}`}>
        <div className={styles.knowledgeIcon}>
          <FolderOpenFilled style={{ fontSize: 14, color: '#fff' }} />
        </div>
        <span className={styles.knowledgeName}>{item.name}</span>
      </div>
    </Tooltip>
  );
}

export default function KnowledgeSummary({ items }: KnowledgeSummaryProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  if (items.length === 0) {
    return null;
  }

  const firstItem = items[0];
  const remainingItems = items.slice(1);

  return (
    <div className={styles.container} ref={containerRef} data-testid="knowledge-summary">
      <KnowledgeTag item={firstItem} />
      {remainingItems.length > 0 && (
        <Popover
          content={
            <div className={styles.popoverContent}>
              {remainingItems.map((item) => (
                <KnowledgeTag key={item.id} item={item} />
              ))}
            </div>
          }
          trigger="hover"
          placement="top"
        >
          <div className={styles.remainingBadge}>
            <span className={styles.badgeText}>+{remainingItems.length}</span>
          </div>
        </Popover>
      )}
    </div>
  );
}
