import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Tabs } from 'antd';

import { KnowledgeExplorePage } from '../knowledge-explore';
import KnowledgeCardList from './components/KnowledgeCardList';
import styles from './KnowledgeBoard.module.css';

export function KnowledgeBoardPage() {
  const { t } = useTranslation('moi-knowledge');
  const [activeTabKey, setActiveTabKey] = useState('knowledge');
  const [exploreCreateRequest, setExploreCreateRequest] = useState<{ knowledgeId: number; requestId: number } | null>(null);
  const exploreCreateRequestSeqRef = useRef(0);

  const handleDialogStarted = (knowledgeId: number) => {
    exploreCreateRequestSeqRef.current += 1;
    setExploreCreateRequest({ knowledgeId, requestId: exploreCreateRequestSeqRef.current });
    setActiveTabKey('explore');
  };

  return (
    <div className={styles.page} data-active-tab={activeTabKey} data-full-bleed="true" data-testid="knowledge-board-page">
      <Tabs
        size="middle"
        className={styles.boardTabs}
        activeKey={activeTabKey}
        onChange={setActiveTabKey}
        data-testid="knowledge-board-tabs"
        items={[
          {
            key: 'knowledge',
            label: t('knowledge.base.board-tab-knowledge'),
            children: <KnowledgeCardList onDialogStarted={handleDialogStarted} />,
          },
          {
            key: 'explore',
            label: t('knowledge.base.board-tab-explore'),
            children: <KnowledgeExplorePage createRequest={exploreCreateRequest} enableDeveloperView />,
          },
        ]}
      />
    </div>
  );
}
