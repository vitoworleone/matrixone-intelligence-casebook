import {
  createSessionApi,
  deleteSessionApi,
  getSessionApi,
  getSessionMessagesApi,
  listPinnedSessionsApi,
  listSessionsApi,
  listUnpinnedSessionsApi,
  pinSessionApi,
  unpinSessionApi,
  updateSessionApi,
  type SessionTag as SessionTagRecord,
  type CreateSessionRequest as SharedCreateSessionRequest,
  type MessageRecord as SharedMessageRecord,
  type PinnedSessionListResponse as SharedPinnedSessionListResponse,
  type SessionListParams as SharedSessionListRequest,
  type SessionListResponse as SharedSessionListResponse,
  type SessionMutationResponse as SharedSessionMutationResponse,
  type SessionRecord as SharedSessionRecord,
  type UpdateSessionRequest as SharedUpdateSessionRequest,
} from '@moi/shared-moi-api/explore';
import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { unwrapApiResponse } from './http';

export type SessionTag = SessionTagRecord | string;
export type SessionRecord = SharedSessionRecord;
export type CreateSessionRequest = SharedCreateSessionRequest;
export type UpdateSessionRequest = SharedUpdateSessionRequest;
export type SessionListRequest = SharedSessionListRequest;
export type SessionListResponse = SharedSessionListResponse;
export type PinnedSessionListResponse = SharedPinnedSessionListResponse;
export type SessionMutationResponse = SharedSessionMutationResponse;
export type MessageRecord = SharedMessageRecord;

export interface CommonConfig {
  type?: 'free' | 'fixed';
  semantic_models?: Array<{ semantic_model_id: number }>;
}

export function parseSessionConfig(config: string | undefined): CommonConfig & Record<string, unknown> {
  if (!config) {
    return {};
  }

  try {
    const parsed = JSON.parse(config);
    if (!parsed || typeof parsed !== 'object') {
      console.warn('[parseSessionConfig] parsed config is not object', { configLength: config.length });
      return {};
    }
    return parsed as CommonConfig & Record<string, unknown>;
  } catch (error) {
    console.warn('[parseSessionConfig] parse config failed', { msg: (error as Error).message });
    return {};
  }
}

function getTagName(tag: SessionTag): string {
  if (typeof tag === 'string') {
    return tag;
  }
  return tag.name;
}

export function isSessionPinned(session: SessionRecord): boolean {
  if (!Array.isArray(session.tags) || session.tags.length === 0) {
    return false;
  }
  return session.tags.some((tag) => getTagName(tag) === 'pinned');
}

export async function createSession(http: AppHttpClient, body: CreateSessionRequest): Promise<SessionRecord> {
  return unwrapApiResponse(
    await createSessionApi(
      {
        ...body,
        source: body.source || 'moi',
      },
      http,
    ),
    'createSession',
  );
}

export async function updateSession(http: AppHttpClient, body: UpdateSessionRequest): Promise<SessionRecord> {
  const { session_id, ...payload } = body;
  if (!session_id || session_id <= 0) {
    console.warn('[updateSession] invalid session_id', { session_id });
    throw new Error('Invalid session_id');
  }
  return unwrapApiResponse(
    await updateSessionApi(
      session_id,
      {
        ...payload,
        source: payload.source || 'moi',
      },
      http,
    ),
    'updateSession',
  );
}

export async function getSessionList(http: AppHttpClient, body: SessionListRequest): Promise<SessionListResponse> {
  return unwrapApiResponse(
    await listSessionsApi(
      {
        ...body,
        source: body.source || 'moi',
      },
      http,
    ),
    'getSessionList',
  );
}

export async function getUnpinnedSessions(http: AppHttpClient, body: SessionListRequest): Promise<SessionListResponse> {
  return unwrapApiResponse(
    await listUnpinnedSessionsApi(
      {
        ...body,
        source: body.source || 'moi',
      },
      http,
    ),
    'getUnpinnedSessions',
  );
}

export async function getPinnedSessions(http: AppHttpClient, body: SessionListRequest): Promise<PinnedSessionListResponse> {
  return unwrapApiResponse(
    await listPinnedSessionsApi(
      {
        ...body,
        source: body.source || 'moi',
      },
      http,
    ),
    'getPinnedSessions',
  );
}

export async function pinSession(http: AppHttpClient, sessionId: number): Promise<SessionMutationResponse> {
  return unwrapApiResponse(await pinSessionApi(sessionId, http), 'pinSession');
}

export async function unpinSession(http: AppHttpClient, sessionId: number): Promise<SessionMutationResponse> {
  return unwrapApiResponse(await unpinSessionApi(sessionId, http), 'unpinSession');
}

export async function deleteSession(http: AppHttpClient, sessionId: number): Promise<SessionMutationResponse> {
  return unwrapApiResponse(await deleteSessionApi(sessionId, http), 'deleteSession');
}

export async function getSession(http: AppHttpClient, sessionId: number): Promise<SessionRecord> {
  return unwrapApiResponse(await getSessionApi(sessionId, http), 'getSession');
}

export async function getSessionMessages(http: AppHttpClient, sessionId: number): Promise<MessageRecord[]> {
  if (!sessionId || sessionId <= 0) {
    console.warn('[getSessionMessages] invalid session_id', { sessionId });
    throw new Error('Invalid session_id');
  }
  return unwrapApiResponse(
    await getSessionMessagesApi(
      {
        session_id: sessionId,
        source: 'moi',
        limit: 100,
      },
      http,
    ),
    'getSessionMessages',
    { allowEmptyData: true, defaultData: [] },
  );
}
