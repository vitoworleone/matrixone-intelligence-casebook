import i18next from 'i18next';

import type enUS from '../../locales/en-US.json';
import { extractApiErrorDetail } from '../../service/http';

type KnowledgeLocaleKey = keyof typeof enUS;

const t = (key: KnowledgeLocaleKey, values?: Record<string, unknown>) =>
  String(i18next.t(key, { ns: 'moi-knowledge', defaultValue: key, ...values }));

const WORKSPACE_ERROR_CODE_KEY_MAP: Record<string, KnowledgeLocaleKey> = {
  ErrWorkspaceNotFound: 'knowledge.base.error-workspace-not-found',
  ErrWorkspacePermissionDenied: 'knowledge.base.error-workspace-permission-denied',
  ErrApiKeyInvalid: 'knowledge.base.error-api-key-invalid',
  ErrServiceUnavailable: 'knowledge.base.error-service-unavailable',
  ErrWorkspaceUnavailable: 'knowledge.base.error-workspace-unavailable',
  ErrTenantDBConnection: 'knowledge.base.error-tenant-db-connection',
};

export function resolveKnowledgeErrorMessage(
  error: unknown,
  fallbackKey: KnowledgeLocaleKey,
  options?: { codeKeyMap?: Partial<Record<string, KnowledgeLocaleKey>> },
): string {
  const { code, msg } = extractApiErrorDetail(error);

  const mappedKey =
    code && options?.codeKeyMap?.[code] ? options.codeKeyMap[code] : code ? WORKSPACE_ERROR_CODE_KEY_MAP[code] : undefined;

  if (mappedKey) {
    return t(mappedKey);
  }
  if (msg) {
    return msg;
  }
  return t(fallbackKey);
}
