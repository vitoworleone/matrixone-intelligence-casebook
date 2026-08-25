import { beforeEach, describe, expect, it, vi } from 'vitest';

import type { AgentA2AResponse, AgentA2AResult } from '@moi/shared-moi-api/agent';
import type { AppHttpClient, AppSSEClient } from '@moi/shared-moi-app-protocol/app-context';
import type { MessageRecord } from '../../../service/dialogSession';
import type { KnowledgeListItem } from '../../../service/knowledge';
import { getExploreA2ARuntimeStore } from './exploreA2ARuntimeStore';

const streamAgentA2AApiMock = vi.hoisted(() => vi.fn());

let streamHandlers: {
  onMessage?: (message: { data?: AgentA2AResponse<AgentA2AResult>; id?: string }) => void;
  onComplete?: () => void;
  onError?: (error: unknown) => void;
} | null = null;

vi.mock('@moi/shared-moi-api/agent', async () => {
  const actual = await vi.importActual<typeof import('@moi/shared-moi-api/agent')>('@moi/shared-moi-api/agent');
  return {
    ...actual,
    streamAgentA2AApi: streamAgentA2AApiMock.mockImplementation((_request, _sseClient, options) => {
      streamHandlers = options;
      return { abort: vi.fn() };
    }),
  };
});

describe('ExploreA2ARuntimeStore', () => {
  beforeEach(() => {
    streamHandlers = null;
    streamAgentA2AApiMock.mockClear();
    getExploreA2ARuntimeStore().clearSessionRuntime(1001);
  });

  it('sends user role in A2A message params', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: '2026年第一季度各地区已支付订单的净销售额是多少，按净销售额降序？',
      model: 'kimi-k2.6',
      modelBackendId: -900001,
      selectedKnowledgeIds: [1],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [
        {
          id: 1,
          name: 'NL2SQL Demo Retail 20260613',
          tables: [{ db_name: 'nl2sql_demo_retail', table_names: ['orders'] }],
        } as KnowledgeListItem,
      ],
      streamErrorFallbackText: '请求失败',
    });

    expect(sent).toBe(true);
    expect(streamAgentA2AApiMock).toHaveBeenCalledTimes(1);
    expect(streamAgentA2AApiMock.mock.calls[0]?.[0]).toMatchObject({
      agent_code: 'explore',
      method: 'message/stream',
      params: {
        message: {
          role: 'user',
          parts: [{ kind: 'text', text: '2026年第一季度各地区已支付订单的净销售额是多少，按净销售额降序？' }],
        },
      },
    });
    const request = streamAgentA2AApiMock.mock.calls[0]?.[0] as { params?: Record<string, unknown> } | undefined;
    const params = request?.params as
      | { metadata?: Record<string, unknown>; message?: { metadata?: Record<string, unknown> } }
      | undefined;
    expect(params?.metadata?.session_id).toBe('1001');
    expect(params?.message?.metadata?.session_id).toBe('1001');
    expect((params?.metadata?.scope as Record<string, unknown> | undefined)?.session_id).toBe('1001');
  });

  it('persists query visuals in A2A metadata for stored conversation rendering', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: '',
      queryVisuals: [{ fileId: 'query_img_1', fileName: '1.png', mimeType: 'image/png', size: 123 }],
      model: 'qwen-plus',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: '中鼎图纸知识库' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });

    expect(sent).toBe(true);
    const request = streamAgentA2AApiMock.mock.calls[0]?.[0] as { params?: Record<string, unknown> } | undefined;
    const params = request?.params as
      | { metadata?: Record<string, unknown>; message?: { metadata?: Record<string, unknown>; parts?: unknown[] } }
      | undefined;
    const expectedImages = [{ file_id: 'query_img_1', file_name: '1.png', mime_type: 'image/png', size: 123 }];
    expect(params?.metadata?.query_visuals).toEqual(expectedImages);
    expect(params?.message?.metadata?.query_visuals).toEqual(expectedImages);
    expect(params?.message?.parts).toEqual([
      {
        kind: 'image',
        data: expectedImages[0],
        metadata: expectedImages[0],
      },
    ]);
  });

  it('sends reasoning effort through A2A metadata when configured', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: '哪张图纸最匹配',
      model: 'gpt-5.5',
      reasoningEffort: 'xhigh',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: '中鼎图纸知识库' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });

    expect(sent).toBe(true);
    const request = streamAgentA2AApiMock.mock.calls[0]?.[0] as { params?: Record<string, unknown> } | undefined;
    const params = request?.params as
      | { metadata?: Record<string, unknown>; message?: { metadata?: Record<string, unknown> } }
      | undefined;
    expect(params?.metadata?.llm_reasoning_effort).toBe('xhigh');
    expect(params?.message?.metadata?.llm_reasoning_effort).toBe('xhigh');
  });

  it('keeps completed answer when SSE transport errors after final event', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: '论文结论是什么',
      model: 'qwen-plus',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: '论文知识库' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });

    expect(sent).toBe(true);
    expect(streamHandlers).toBeTruthy();

    streamHandlers?.onMessage?.({
      id: '1',
      data: response({
        kind: 'artifact-update',
        taskId: currentTaskId(store, 1001),
        artifact: {
          artifactId: 'answer_1',
          name: 'answer',
          parts: [{ kind: 'text', text: '最终答案' }],
          metadata: { matrixflow_type: 'explore.answer', data_type: 'matrixflow.explore.answer' },
        },
        lastChunk: true,
      }),
    });
    streamHandlers?.onMessage?.({
      id: '2',
      data: response({
        kind: 'status-update',
        taskId: currentTaskId(store, 1001),
        status: { state: 'completed' },
        final: true,
      }),
    });
    streamHandlers?.onError?.(new Error('network error'));

    const state = store.getRuntimeStateMap()[1001];
    expect(state.streamError).toBe('');
    expect(state.projection.final).toBe(true);
    expect(state.projection.status).toBe('completed');
    expect(state.messages.at(-1)?.content).toBe('最终答案');
  });

  it('ends an input-required stream without losing the pending input or resubscribing', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: '从 Catalog 读取、解析并写回 Catalog',
      model: 'kimi-k2.6',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: '工作流设计' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });

    expect(sent).toBe(true);
    const initialHandlers = streamHandlers;
    const taskId = currentTaskId(store, 1001);
    initialHandlers?.onMessage?.({
      id: '1',
      data: response({
        kind: 'status-update',
        taskId,
        contextId: 'explore_session_1001',
        status: {
          state: 'working',
          message: {
            role: 'agent',
            parts: [
              {
                kind: 'data',
                data: {
                  type: 'moi.tool.call',
                  toolId: 'request_user_input',
                  toolKind: 'request_user_input',
                  callId: 'functions.request_user_input:8',
                  turnId: taskId,
                  arguments: {
                    questions: [
                      {
                        id: 'save_target',
                        header: '保存内容',
                        question: '要保存什么？',
                        options: [{ label: '解析文档', description: '保存解析后的文档' }],
                      },
                    ],
                  },
                },
              },
            ],
          },
        },
      }),
    });
    initialHandlers?.onMessage?.({
      id: '2',
      data: response({
        kind: 'status-update',
        taskId,
        contextId: 'explore_session_1001',
        status: { state: 'input-required' },
        final: true,
      }),
    });
    initialHandlers?.onComplete?.();

    const state = store.getRuntimeStateMap()[1001];
    expect(streamAgentA2AApiMock).toHaveBeenCalledTimes(1);
    expect(state.isStreaming).toBe(false);
    expect(state.streamError).toBe('');
    expect(state.recoveryAttempts).toBe(0);
    expect(state.projection.final).toBe(true);
    expect(state.projection.status).toBe('input-required');
    expect(state.projection.failureDisplay).toBeUndefined();
    expect(state.projection.pendingInput).toMatchObject({
      callId: 'functions.request_user_input:8',
      taskId,
      contextId: 'explore_session_1001',
    });
    expect(state.messages.at(-1)?.projection?.pendingInput?.callId).toBe('functions.request_user_input:8');
  });

  it('resubscribes after the last persisted seq when transport fails before terminal status', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: '年度销售与利润分析',
      model: 'qwen-plus',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: '年度销售与利润分析' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });

    expect(sent).toBe(true);
    expect(streamHandlers).toBeTruthy();

    streamHandlers?.onMessage?.({
      id: '1',
      data: response({
        kind: 'artifact-update',
        taskId: currentTaskId(store, 1001),
        artifact: {
          artifactId: 'answer_1',
          name: 'answer',
          parts: [{ kind: 'text', text: '年度销售与利润分析最终答案' }],
          metadata: { matrixflow_type: 'explore.answer', data_type: 'matrixflow.explore.answer' },
        },
        lastChunk: true,
      }),
    });
    const initialHandlers = streamHandlers;
    initialHandlers?.onError?.(new Error('Error in input stream'));

    expect(streamAgentA2AApiMock).toHaveBeenCalledTimes(2);
    expect(streamAgentA2AApiMock.mock.calls[1]?.[0]).toMatchObject({
      method: 'tasks/resubscribe',
      params: { taskId: currentTaskId(store, 1001), afterSeq: 1 },
    });
    let state = store.getRuntimeStateMap()[1001];
    expect(state.isStreaming).toBe(true);
    expect(state.projection.final).toBe(false);
    expect(state.messages.at(-1)?.content).toBe('年度销售与利润分析最终答案');

    streamHandlers?.onMessage?.({
      id: '2',
      data: response({
        kind: 'status-update',
        taskId: currentTaskId(store, 1001),
        status: { state: 'completed' },
        final: true,
      }),
    });

    state = store.getRuntimeStateMap()[1001];
    expect(state.streamError).toBe('');
    expect(state.projection.final).toBe(true);
    expect(state.projection.status).toBe('completed');
    expect(state.messages.at(-1)?.content).toBe('年度销售与利润分析最终答案');
  });

  it('updates the current assistant message when cancel returns the real runtime task id', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: 'screen resumes',
      model: 'qwen-plus',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: 'HR Knowledge' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });
    expect(sent).toBe(true);

    streamHandlers?.onMessage?.({
      id: '1',
      data: response({
        kind: 'task',
        id: 'task-real',
        contextId: 'explore_session_1001',
        status: {
          state: 'working',
          message: { role: 'agent', parts: [{ kind: 'text', text: 'working' }] },
        },
      }),
    });

    await store.stopSessionStream(1001, {
      post: vi.fn().mockResolvedValue({
        data: response({
          kind: 'task',
          id: 'task-real',
          contextId: 'explore_session_1001',
          status: {
            state: 'canceled',
            message: { role: 'agent', parts: [{ kind: 'text', text: 'stopped' }] },
          },
        }),
      }),
    } as unknown as AppHttpClient);

    const state = store.getRuntimeStateMap()[1001];
    expect(state.taskId).toBe('task-real');
    expect(state.projection.status).toBe('canceled');
    expect(state.projection.final).toBe(true);
    expect(state.messages.at(-1)?.content).toBe('stopped');
    expect(state.messages.at(-1)?.projection?.status).toBe('canceled');
  });

  it('does not synthesize completion from EOF and waits for recovered terminal status', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: '哪张图纸最匹配',
      model: 'qwen-plus',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: '图纸知识库' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });

    expect(sent).toBe(true);
    streamHandlers?.onMessage?.({
      id: '1',
      data: response({
        kind: 'artifact-update',
        taskId: currentTaskId(store, 1001),
        artifact: {
          artifactId: 'answer_1',
          name: 'answer',
          parts: [
            {
              kind: 'data',
              data: {
                answer: '最匹配的图纸是 **20C114820.pdf**。',
                sources: [{ type: 'visual_hit', image_file_id: 'image_1' }],
              },
            },
          ],
          metadata: {
            matrixflow_type: 'knowledge.answer',
            data_type: 'matrixflow.knowledge.answer',
            source_refs: [{ type: 'visual_hit', image_file_id: 'image_1', file_name: '20C114820.pdf' }],
          },
        },
        lastChunk: true,
      }),
    });
    const initialHandlers = streamHandlers;
    initialHandlers?.onComplete?.();

    expect(streamAgentA2AApiMock.mock.calls[1]?.[0]).toMatchObject({
      method: 'tasks/resubscribe',
      params: { taskId: currentTaskId(store, 1001), afterSeq: 1 },
    });
    let state = store.getRuntimeStateMap()[1001];
    expect(state.isStreaming).toBe(true);
    expect(state.projection.final).toBe(false);

    streamHandlers?.onMessage?.({
      id: '2',
      data: response({
        kind: 'status-update',
        taskId: currentTaskId(store, 1001),
        status: { state: 'completed' },
        final: true,
      }),
    });

    state = store.getRuntimeStateMap()[1001];
    expect(state.isStreaming).toBe(false);
    expect(state.projection.final).toBe(true);
    expect(state.projection.status).toBe('completed');
    expect(state.messages.at(-1)?.content).toBe('最匹配的图纸是 **20C114820.pdf**。');
    expect(state.messages.at(-1)?.content).not.toContain('sources');
  });

  it('updates assistant message content from assistant delta before final answer', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: '哪张图纸最匹配',
      model: 'qwen-plus',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: '图纸知识库' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });

    expect(sent).toBe(true);
    streamHandlers?.onMessage?.({
      id: '1',
      data: response({
        kind: 'artifact-update',
        taskId: currentTaskId(store, 1001),
        artifact: {
          artifactId: 'assistant_stream_1',
          name: 'assistant_delta',
          parts: [{ kind: 'text', text: '最匹配的图纸' }],
          metadata: { matrixflow_type: 'explore.assistant_delta', data_type: 'matrixflow.explore.assistant_delta' },
        },
        append: true,
      }),
    });
    streamHandlers?.onMessage?.({
      id: '2',
      data: response({
        kind: 'artifact-update',
        taskId: currentTaskId(store, 1001),
        artifact: {
          artifactId: 'assistant_stream_1',
          name: 'assistant_delta',
          parts: [{ kind: 'text', text: '是 **20C114820.pdf**。' }],
          metadata: { matrixflow_type: 'explore.assistant_delta', data_type: 'matrixflow.explore.assistant_delta' },
        },
        append: true,
      }),
    });

    const state = store.getRuntimeStateMap()[1001];
    expect(state.projection.answerDraft).toBe('最匹配的图纸是 **20C114820.pdf**。');
    expect(state.projection.assistantDraft).toBe('');
    expect(state.messages.at(-1)?.content).toBe('最匹配的图纸是 **20C114820.pdf**。');
  });

  it('keeps transport interruption nonterminal until a recovered failed terminal arrives', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: '哪张图纸最匹配',
      model: 'qwen-plus',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: '图纸知识库' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });

    expect(sent).toBe(true);
    streamHandlers?.onMessage?.({
      id: '1',
      data: response({
        kind: 'artifact-update',
        taskId: currentTaskId(store, 1001),
        artifact: {
          artifactId: 'assistant_stream_1',
          name: 'assistant_delta',
          parts: [{ kind: 'text', text: '最匹配的图纸是 **20C114820.pdf**。' }],
          metadata: { matrixflow_type: 'explore.assistant_delta', data_type: 'matrixflow.explore.assistant_delta' },
        },
        append: true,
      }),
    });
    const initialHandlers = streamHandlers;
    initialHandlers?.onError?.(new Error('network error'));

    let state = store.getRuntimeStateMap()[1001];
    expect(state.isStreaming).toBe(true);
    expect(state.streamError).toBe('');
    expect(state.projection.final).toBe(false);
    expect(state.projection.status).not.toBe('failed');
    expect(state.messages.at(-1)?.content).toBe('最匹配的图纸是 **20C114820.pdf**。');
    expect(streamAgentA2AApiMock.mock.calls[1]?.[0]).toMatchObject({
      method: 'tasks/resubscribe',
      params: { taskId: currentTaskId(store, 1001), afterSeq: 1 },
    });

    streamHandlers?.onMessage?.({
      id: '2',
      data: response({
        kind: 'status-update',
        taskId: currentTaskId(store, 1001),
        final: true,
        status: { state: 'failed' },
      }),
    });

    state = store.getRuntimeStateMap()[1001];
    expect(state.isStreaming).toBe(false);
    expect(state.streamError).toBe('');
    expect(state.projection.final).toBe(true);
    expect(state.projection.status).toBe('failed');
    expect(state.projection.failureDisplay).toMatchObject({ i18nKey: 'agent.a2a.failure' });
    expect(state.projection.answerSources).toEqual([]);
    expect(state.messages.at(-1)?.content).toBe('');
  });

  it('keeps structured result-evidence failures out of the live assistant message content', async () => {
    const store = getExploreA2ARuntimeStore();
    const rawError = 'agent-runtime-v2 completed after failed result evidence tool result: search_rag_chunks';
    const sent = await store.sendSessionMessage({
      sessionId: 1001,
      question: '联想天津如何参观',
      model: 'qwen-plus',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: '联想知识库' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });

    expect(sent).toBe(true);
    streamHandlers?.onMessage?.({
      id: '1',
      data: response({
        kind: 'status-update',
        taskId: currentTaskId(store, 1001),
        final: true,
        status: {
          state: 'failed',
          message: { role: 'agent', parts: [{ kind: 'text', text: rawError }] },
        },
        metadata: {
          reason_display: {
            i18n_key: 'reason.agent_runtime.finish.error',
            default_text: 'The agent run failed.',
          },
        },
      }),
    });

    const state = store.getRuntimeStateMap()[1001];
    expect(state.streamError).toBe('');
    expect(state.messages.at(-1)?.content).toBe('');
    expect(state.projection.failureDisplay).toMatchObject({ i18nKey: 'reason.agent_runtime.finish.error' });
    expect(JSON.stringify(state.projection)).not.toContain(rawError);
  });

  it('keeps structured failed history records out of assistant message content', () => {
    const store = getExploreA2ARuntimeStore();
    const rawError = 'agent-runtime-v2 completed after failed result evidence tool result: search_rag_chunks';
    const failedEvent = response({
      kind: 'status-update',
      taskId: 'task_1',
      final: true,
      status: {
        state: 'failed',
        message: { role: 'agent', parts: [{ kind: 'text', text: rawError }] },
      },
      metadata: {
        reason_display: {
          i18n_key: 'reason.agent_runtime.finish.error',
          default_text: 'The agent run failed.',
        },
      },
    });

    store.hydrateSessionMessages(1001, [
      messageRecord({
        id: 1,
        source: 'moi',
        role: 'user',
        content: '联想天津如何参观',
        modified_response: JSON.stringify([{ seq: 1, response: failedEvent }]),
        created_at: 10,
      }),
      messageRecord({
        id: 2,
        source: 'moi',
        role: 'assistant',
        status: 'failed',
        content: rawError,
        created_at: 20,
      }),
    ]);

    const state = store.getRuntimeStateMap()[1001];
    expect(state.messages.at(-1)?.content).toBe('');
    expect(state.messages.at(-1)?.projection?.failureDisplay).toMatchObject({
      i18nKey: 'reason.agent_runtime.finish.error',
    });
  });

  it('restores pending input from a clean input-required history record', () => {
    const store = getExploreA2ARuntimeStore();
    const inputCall = response({
      kind: 'status-update',
      taskId: 'task_1',
      contextId: 'explore_session_1001',
      status: {
        state: 'working',
        message: {
          role: 'agent',
          parts: [
            {
              kind: 'data',
              data: {
                type: 'moi.tool.call',
                toolId: 'request_user_input',
                callId: 'functions.request_user_input:8',
                turnId: 'task_1',
                arguments: {
                  questions: [
                    {
                      id: 'save_target',
                      header: '保存内容',
                      question: '要保存什么？',
                      options: [{ label: '解析文档', description: '保存解析后的文档' }],
                    },
                  ],
                },
              },
            },
          ],
        },
      },
    });
    const inputRequired = response({
      kind: 'status-update',
      taskId: 'task_1',
      contextId: 'explore_session_1001',
      status: { state: 'input-required' },
      final: true,
    });

    store.hydrateSessionMessages(1001, [
      messageRecord({
        id: 1,
        source: 'moi',
        role: 'user',
        content: '从 Catalog 读取、解析并写回 Catalog',
        modified_response: JSON.stringify([
          { seq: 1, response: inputCall },
          { seq: 2, response: inputRequired },
        ]),
        created_at: 10,
      }),
      messageRecord({
        id: 2,
        source: 'moi',
        role: 'assistant',
        status: 'success',
        content: '',
        created_at: 20,
      }),
    ]);

    const projection = store.getRuntimeStateMap()[1001].messages.at(-1)?.projection;
    expect(projection?.status).toBe('input-required');
    expect(projection?.final).toBe(true);
    expect(projection?.failureDisplay).toBeUndefined();
    expect(projection?.pendingInput).toMatchObject({
      callId: 'functions.request_user_input:8',
      taskId: 'task_1',
      contextId: 'explore_session_1001',
    });
  });

  it('hydrates stored knowledge conversation messages with moi source', () => {
    const store = getExploreA2ARuntimeStore();
    store.hydrateSessionMessages(1001, [
      messageRecord({
        id: 1,
        source: 'moi',
        role: 'user',
        content: '找一个熟悉 docker 的人',
        config: JSON.stringify({
          metadata: {
            semantic_model_ids: [7],
            query_visuals: [{ file_id: 'query_img_1', file_name: '9.png', mime_type: 'image/png' }],
          },
        }),
        created_at: 10,
      }),
      messageRecord({
        id: 2,
        source: 'moi',
        role: 'assistant',
        content: '推荐 Wei Hassan',
        created_at: 20,
      }),
    ]);

    const state = store.getRuntimeStateMap()[1001];
    expect(state.messages).toHaveLength(2);
    expect(state.messages[0]).toMatchObject({ messageId: 1, role: 'user', content: '找一个熟悉 docker 的人' });
    expect(state.messages[0].queryVisuals).toEqual([
      { fileId: 'query_img_1', fileName: '9.png', mimeType: 'image/png', size: 0 },
    ]);
    expect(state.messages[1]).toMatchObject({ messageId: 2, role: 'assistant', content: '推荐 Wei Hassan' });
    expect(state.messages[1].knowledgeIds).toEqual([7]);
  });

  it('hydrates legacy query images metadata as query visuals', () => {
    const store = getExploreA2ARuntimeStore();
    store.hydrateSessionMessages(1001, [
      messageRecord({
        id: 1,
        source: 'moi',
        role: 'user',
        content: '找这张图对应的图纸',
        config: JSON.stringify({
          metadata: {
            query_images: [{ file_id: 'legacy_query_img_1', file_name: 'legacy.png', mime_type: 'image/png', size: 321 }],
          },
        }),
        created_at: 10,
      }),
    ]);

    const state = store.getRuntimeStateMap()[1001];
    expect(state.messages[0].queryVisuals).toEqual([
      { fileId: 'legacy_query_img_1', fileName: 'legacy.png', mimeType: 'image/png', size: 321 },
    ]);
  });

  it('marks non-final interrupted multi-session streams as structured failures', async () => {
    const store = getExploreA2ARuntimeStore();
    const sent = await store.sendSessionMessage({
      sessionId: 2001,
      question: '并发会话 A',
      model: 'qwen-plus',
      selectedKnowledgeIds: [7],
      workspaceId: 'ws_1',
      sseClient: {} as AppSSEClient,
      knowledgeList: [{ id: 7, name: '知识库' } as KnowledgeListItem],
      streamErrorFallbackText: '请求失败',
    });
    expect(sent).toBe(true);
    expect(streamHandlers).toBeTruthy();

    // Advance the cursor so transport errors enter resubscribe recovery instead of
    // the direct-message completion path used when no runtime events arrived yet.
    streamHandlers?.onMessage?.({
      id: '1',
      data: response({
        kind: 'status-update',
        taskId: currentTaskId(store, 2001),
        contextId: 'explore_session_2001',
        status: { state: 'working' },
      }),
    });

    // Exhaust recovery so markStreamInterrupted owns the terminal UI state.
    for (let attempt = 0; attempt < 4; attempt += 1) {
      streamHandlers?.onError?.(new Error('network error'));
    }

    const state = store.getRuntimeStateMap()[2001];
    expect(state.isStreaming).toBe(false);
    expect(state.streamError).toBe('');
    expect(state.projection.final).toBe(true);
    expect(state.projection.status).toBe('failed');
    expect(state.projection.failureDisplay).toMatchObject({
      i18nKey: 'agent.a2a.stream_interrupted',
    });
    expect(state.messages.at(-1)?.content).toBe('');
    expect(state.messages.at(-1)?.projection?.status).toBe('failed');
  });
});

function currentTaskId(store: ReturnType<typeof getExploreA2ARuntimeStore>, sessionId: number): string {
  const taskId = store.getRuntimeStateMap()[sessionId]?.taskId;
  if (!taskId) {
    throw new Error('missing task id');
  }
  return taskId;
}

function response(result: AgentA2AResult): AgentA2AResponse<AgentA2AResult> {
  return {
    jsonrpc: '2.0',
    id: '1',
    result,
  };
}

function messageRecord(record: Partial<MessageRecord> & Pick<MessageRecord, 'id'>): MessageRecord {
  return {
    session_id: 1001,
    ...record,
  };
}
