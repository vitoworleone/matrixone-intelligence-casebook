import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';

import { useHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import {
  KnowledgeSourceSelectModal as SharedKnowledgeSourceSelectModal,
  type KnowledgeSourceSelectModalTranslate,
  type KnowledgeSourceSelectModalProps as SharedKnowledgeSourceSelectModalProps,
} from '@moi/shared-moi-components/knowledge-source-select-modal';

export type KnowledgeSourceSelectModalProps = Omit<SharedKnowledgeSourceSelectModalProps, 'http' | 'language' | 'translate'>;

export default function KnowledgeSourceSelectModal(props: KnowledgeSourceSelectModalProps) {
  const { t, i18n } = useTranslation('moi-knowledge');
  const http = useHttpClient();
  const knowledgeT = t as (key: string, params?: Record<string, unknown>) => string;
  const translate = useCallback<KnowledgeSourceSelectModalTranslate>((key, params) => knowledgeT(key, params), [knowledgeT]);

  return <SharedKnowledgeSourceSelectModal {...props} http={http} language={i18n?.language} translate={translate} />;
}
