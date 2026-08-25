import { listAgentsApi, type AgentRecord } from '@moi/shared-moi-api/agent';
import type { AppHttpClient, AppHttpRequestConfig, AppHttpResponse } from '@moi/shared-moi-app-protocol/app-context';
import { unwrapApiResponse } from './http';

const AGENT_PAGE_SIZE = 100;
const MAX_AGENT_PAGE_REQUESTS = 100;

interface ListKnowledgeAssociatedAgentsParams {
  workspaceId: string;
  knowledgeId: number;
}

function withWorkspace(workspaceId: string, url: string): string {
  if (!url.startsWith('/') || url.startsWith('/workspaces/')) {
    return url;
  }
  return `/workspaces/${encodeURIComponent(workspaceId)}${url}`;
}

function scopedHttp(http: AppHttpClient, workspaceId: string): AppHttpClient {
  return {
    get: <T = unknown>(url: string, config?: AppHttpRequestConfig): Promise<AppHttpResponse<T>> =>
      http.get<T>(withWorkspace(workspaceId, url), config),
    post: <T = unknown>(url: string, data?: unknown, config?: AppHttpRequestConfig): Promise<AppHttpResponse<T>> =>
      http.post<T>(withWorkspace(workspaceId, url), data, config),
    put: <T = unknown>(url: string, data?: unknown, config?: AppHttpRequestConfig): Promise<AppHttpResponse<T>> =>
      http.put<T>(withWorkspace(workspaceId, url), data, config),
    patch: http.patch
      ? <T = unknown>(url: string, data?: unknown, config?: AppHttpRequestConfig): Promise<AppHttpResponse<T>> =>
          http.patch?.<T>(withWorkspace(workspaceId, url), data, config) as Promise<AppHttpResponse<T>>
      : undefined,
    delete: <T = unknown>(url: string, config?: AppHttpRequestConfig): Promise<AppHttpResponse<T>> =>
      http.delete<T>(withWorkspace(workspaceId, url), config),
  };
}

function agentListItems(data: { items?: AgentRecord[]; agents?: AgentRecord[] }): AgentRecord[] {
  return data.items ?? data.agents ?? [];
}

function isVisibleAgent(agent: AgentRecord): boolean {
  return agent.status !== 'archived';
}

export function getAgentKnowledgeBaseIds(agent: AgentRecord): string[] {
  return (agent.binding?.knowledge_base_ids ?? agent.kbs ?? []).filter((id): id is string => typeof id === 'string');
}

function isAgentBoundToKnowledge(agent: AgentRecord, knowledgeId: number): boolean {
  const targetId = String(knowledgeId);
  return getAgentKnowledgeBaseIds(agent).some((id) => id === targetId);
}

export async function listKnowledgeAssociatedAgents(
  http: AppHttpClient,
  params: ListKnowledgeAssociatedAgentsParams,
): Promise<AgentRecord[]> {
  if (!params.workspaceId) {
    throw new Error('Missing workspace id for agent list');
  }
  if (!Number.isFinite(params.knowledgeId) || params.knowledgeId <= 0) {
    throw new Error('Invalid knowledge id');
  }

  const scoped = scopedHttp(http, params.workspaceId);
  const agents: AgentRecord[] = [];
  let offset = 0;
  let total = 0;

  const loadPage = async (requestIndex: number): Promise<void> => {
    if (requestIndex >= MAX_AGENT_PAGE_REQUESTS) {
      throw new Error('Agent list pagination exceeded maximum request count');
    }

    const data = unwrapApiResponse(
      await listAgentsApi({ limit: AGENT_PAGE_SIZE, offset }, scoped),
      'listKnowledgeAssociatedAgents',
    );
    const items = agentListItems(data);
    total = data.total ?? total;
    agents.push(...items.filter(isVisibleAgent).filter((agent) => isAgentBoundToKnowledge(agent, params.knowledgeId)));
    offset += items.length;

    if (items.length === 0 || items.length < AGENT_PAGE_SIZE || (total > 0 && offset >= total)) {
      return;
    }

    await loadPage(requestIndex + 1);
  };

  await loadPage(0);
  return agents;
}
