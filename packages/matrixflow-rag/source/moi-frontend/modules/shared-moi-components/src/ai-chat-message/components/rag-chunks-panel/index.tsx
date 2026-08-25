import { memo, useMemo } from 'react';
import { Tag } from 'antd';
import { FileTextOutlined } from '@ant-design/icons';

import { getLocaleMap } from '../../i18n';
import type { RetrievalChunk, RetrievalDocument } from '../../rich-content-types';
import type { LocaleType } from '../../types';
import styles from './index.module.css';

export interface RagChunksPanelProps {
  chunks: RetrievalChunk[];
  documents: RetrievalDocument[];
  locale?: LocaleType;
}

function scoreToColor(score: number): string {
  if (score >= 0.9) return 'green';
  if (score >= 0.7) return 'blue';
  if (score >= 0.5) return 'orange';
  return 'default';
}

function RagChunksPanelInner({ chunks, documents, locale = 'zh-CN' }: RagChunksPanelProps) {
  const i18n = useMemo(() => getLocaleMap(locale), [locale]);

  if (chunks.length === 0 && documents.length === 0) {
    return null;
  }

  return (
    <div data-testid="rag-chunks-panel">
      {/* Chunks */}
      {chunks.length > 0 && (
        <div className={styles.container} data-testid="rag-chunks-list">
          {chunks.map((chunk, idx) => (
            <div key={`${chunk.file_id}-${idx}`} className={styles.chunk} data-testid={`rag-chunk-${idx}`}>
              <div className={styles.chunkMeta}>
                <FileTextOutlined />
                <span className={styles.fileName}>{chunk.file_name}</span>
                {chunk.page !== null && chunk.page !== undefined && (
                  <span>
                    {i18n.ragChunksPanelPage} {chunk.page}
                  </span>
                )}
                {chunk.score > 0 && (
                  <Tag color={scoreToColor(chunk.score)} data-testid={`rag-chunk-score-${idx}`}>
                    {chunk.score.toFixed(2)}
                  </Tag>
                )}
              </div>
              <div className={styles.chunkContent}>{chunk.content || i18n.ragChunksPanelNoContent}</div>
            </div>
          ))}
        </div>
      )}

      {/* Documents */}
      {documents.length > 0 && (
        <div className={styles.container} data-testid="rag-docs-list">
          {documents.map((doc, idx) => (
            <div key={`${doc.file_id}-${idx}`} className={styles.docItem} data-testid={`rag-doc-${idx}`}>
              <div className={styles.docHeader}>
                <FileTextOutlined /> {doc.file_name}
              </div>
              <div className={styles.docContent}>{doc.content}</div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export const RagChunksPanel = memo(RagChunksPanelInner);
