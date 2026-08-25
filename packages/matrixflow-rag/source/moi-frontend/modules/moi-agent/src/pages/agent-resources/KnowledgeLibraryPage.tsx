import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from 'antd';
import { PlusOutlined } from '@ant-design/icons';

import { useOptionalModuleNavigator } from '@moi/shared-moi-app-protocol/module-context';
import { KBDetailPanel } from '../agent-chat/components/KBDetailPanel';
import { AgentWorkbenchPageFrame } from '../agent-workbench/AgentWorkbenchPageFrame';

import '../../navigation-map';

function readSearchString(search: unknown, key: string): string | undefined {
  if (!search || typeof search !== 'object') return undefined;
  const value = (search as Record<string, unknown>)[key];
  return typeof value === 'string' && value.trim() ? value.trim() : undefined;
}

export function KnowledgeLibraryPage() {
  const { t } = useTranslation('agent');
  const moduleNav = useOptionalModuleNavigator();
  const search = moduleNav?.callbacks.getSearchParams?.();
  const initialKbId = readSearchString(search, 'kb_id');
  const [createNonce, setCreateNonce] = useState(0);

  return (
    <AgentWorkbenchPageFrame
      testId="agent-knowledge-library-page"
      title={t('agent.page.knowledgeLibraryTitle')}
      extra={
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setCreateNonce((current) => current + 1)}
          data-testid="knowledge-library-create-btn"
        >
          {t('agent.action.newKB')}
        </Button>
      }
    >
      <KBDetailPanel currentAgent={null} initialSelectedId={initialKbId} createNonce={createNonce} />
    </AgentWorkbenchPageFrame>
  );
}
