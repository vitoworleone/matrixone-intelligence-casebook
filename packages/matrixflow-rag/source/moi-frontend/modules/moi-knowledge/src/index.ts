import { lazy } from 'react';

import type { ModuleDefinition } from '@moi/shared-moi-app-protocol/module';
import enUS from './locales/en-US.json';
import zhCN from './locales/zh-CN.json';

const LazyKnowledgeBoardPage = lazy(() => import('./pages/knowledge-board').then((m) => ({ default: m.KnowledgeBoardPage })));
const LazyKnowledgeEditPage = lazy(() => import('./pages/knowledge-edit').then((m) => ({ default: m.KnowledgeEditPage })));

export { KnowledgeExplorePage } from './pages/knowledge-explore';
export type { KnowledgeExplorePageProps } from './pages/knowledge-explore';

export const moiKnowledgeModule: ModuleDefinition = {
  name: 'moi-knowledge',
  pages: {
    'knowledge-board': { component: LazyKnowledgeBoardPage, meta: { title: 'Knowledge Base' } },
    'knowledge-edit': { component: LazyKnowledgeEditPage, meta: { title: 'Edit Knowledge Base' } },
  },
  locales: {
    namespace: 'moi-knowledge',
    'zh-CN': zhCN,
    'en-US': enUS,
  },
  requires: {
    httpClient: true,
    sseClient: true,
  },
};

export type MoiKnowledgePages = 'knowledge-board' | 'knowledge-edit';
