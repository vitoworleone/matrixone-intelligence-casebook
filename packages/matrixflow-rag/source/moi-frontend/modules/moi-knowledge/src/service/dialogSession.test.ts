import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import {
  createSession,
  getPinnedSessions,
  getSessionList,
  getSessionMessages,
  getUnpinnedSessions,
  updateSession,
} from './dialogSession';

const apiMocks = vi.hoisted(() => ({
  createSessionApi: vi.fn(),
  getSessionMessagesApi: vi.fn(),
  listPinnedSessionsApi: vi.fn(),
  listSessionsApi: vi.fn(),
  listUnpinnedSessionsApi: vi.fn(),
  updateSessionApi: vi.fn(),
}));

vi.mock('@moi/shared-moi-api/explore', () => ({
  createSessionApi: apiMocks.createSessionApi,
  deleteSessionApi: vi.fn(),
  getSessionApi: vi.fn(),
  getSessionMessagesApi: apiMocks.getSessionMessagesApi,
  listPinnedSessionsApi: apiMocks.listPinnedSessionsApi,
  listSessionsApi: apiMocks.listSessionsApi,
  listUnpinnedSessionsApi: apiMocks.listUnpinnedSessionsApi,
  pinSessionApi: vi.fn(),
  unpinSessionApi: vi.fn(),
  updateSessionApi: apiMocks.updateSessionApi,
}));

describe('dialogSession', () => {
  const http = {} as AppHttpClient;

  beforeEach(() => {
    vi.clearAllMocks();
    apiMocks.createSessionApi.mockResolvedValue({ code: 'OK', data: { session_id: 1 } });
    apiMocks.updateSessionApi.mockResolvedValue({ code: 'OK', data: { session_id: 1 } });
    apiMocks.listSessionsApi.mockResolvedValue({ code: 'OK', data: { sessions: [] } });
    apiMocks.listUnpinnedSessionsApi.mockResolvedValue({ code: 'OK', data: { sessions: [] } });
    apiMocks.listPinnedSessionsApi.mockResolvedValue({ code: 'OK', data: { sessions: [] } });
    apiMocks.getSessionMessagesApi.mockResolvedValue({ code: 'OK', data: [] });
  });

  it('uses moi as the knowledge conversation source for sessions and messages', async () => {
    await createSession(http, { name: 'new session' });
    expect(apiMocks.createSessionApi).toHaveBeenCalledWith(expect.objectContaining({ source: 'moi' }), http);

    await updateSession(http, { session_id: 1, name: 'renamed session' });
    expect(apiMocks.updateSessionApi).toHaveBeenCalledWith(1, expect.objectContaining({ source: 'moi' }), http);

    await getSessionList(http, { page: 1, page_size: 20 });
    expect(apiMocks.listSessionsApi).toHaveBeenCalledWith(expect.objectContaining({ source: 'moi' }), http);

    await getUnpinnedSessions(http, { page: 1, page_size: 20 });
    expect(apiMocks.listUnpinnedSessionsApi).toHaveBeenCalledWith(expect.objectContaining({ source: 'moi' }), http);

    await getPinnedSessions(http, { page: 1, page_size: 20 });
    expect(apiMocks.listPinnedSessionsApi).toHaveBeenCalledWith(expect.objectContaining({ source: 'moi' }), http);

    await getSessionMessages(http, 1);
    expect(apiMocks.getSessionMessagesApi).toHaveBeenCalledWith(
      expect.objectContaining({ limit: 100, session_id: 1, source: 'moi' }),
      http,
    );
  });
});
