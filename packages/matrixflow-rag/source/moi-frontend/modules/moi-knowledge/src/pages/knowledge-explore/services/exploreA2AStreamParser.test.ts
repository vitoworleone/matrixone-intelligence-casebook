import { describe, expect, it } from 'vitest';

import type { AgentA2AResponse, AgentA2AResult } from '@moi/shared-moi-api/agent';
import {
  completeExploreA2AProjection,
  createExploreA2AProjection,
  failExploreA2AProjection,
  omitRepeatedFinalAnswerTraceOutput,
  reduceExploreA2AResponse,
} from './exploreA2AStreamParser';

describe('explore A2A stream parser', () => {
  it('projects status updates with display metadata into timeline and thinking', () => {
    const projection = reduceExploreA2AResponse(
      createExploreA2AProjection(),
      response({
        kind: 'status-update',
        taskId: 'task_1',
        status: {
          state: 'working',
          message: {
            role: 'agent',
            parts: [
              {
                kind: 'text',
                text: '正在核对完整年度资料',
                metadata: {
                  i18n_key: 'explore.status.thinking',
                  display: {
                    i18n_key: 'explore.status.thinking',
                    default_text: '正在核对完整年度资料',
                  },
                },
              },
            ],
          },
        },
      }),
      1,
    );

    expect(projection.taskId).toBe('task_1');
    expect(projection.status).toBe('working');
    expect(projection.thinking).toEqual(['正在核对完整年度资料']);
    expect(projection.timeline[0]).toMatchObject({
      event: 'status-update',
      summary: '正在核对完整年度资料',
      display: {
        i18nKey: 'explore.status.thinking',
        defaultText: '正在核对完整年度资料',
      },
    });
  });

  it('projects task metadata trace id', () => {
    const projection = reduceExploreA2AResponse(
      createExploreA2AProjection(),
      response({
        kind: 'task',
        id: 'task_1',
        contextId: 'ctx_1',
        metadata: {
          trace_id: 'trace_abc',
        },
        status: {
          state: 'working',
        },
      }),
      1,
    );

    expect(projection.traceId).toBe('trace_abc');
  });

  it('appends assistant deltas and closes with final answer artifact', () => {
    let projection = createExploreA2AProjection();
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'assistant_stream_task_1',
          name: 'assistant_delta',
          parts: [{ kind: 'text', text: '营收增速' }],
          metadata: { matrixflow_type: 'explore.assistant_delta', data_type: 'matrixflow.explore.assistant_delta' },
        },
        append: true,
      }),
      1,
    );
    expect(projection.answerDraft).toBe('营收增速');
    expect(projection.assistantDraft).toBe('');
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'assistant_stream_task_1',
          name: 'assistant_delta',
          parts: [{ kind: 'text', text: '提升' }],
          metadata: { matrixflow_type: 'explore.assistant_delta', data_type: 'matrixflow.explore.assistant_delta' },
        },
        append: true,
      }),
      2,
    );
    expect(projection.answerDraft).toBe('营收增速提升');
    expect(projection.assistantDraft).toBe('');
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'answer_task_1',
          name: 'answer',
          parts: [{ kind: 'text', text: '最终答案' }],
          metadata: { matrixflow_type: 'explore.answer', data_type: 'matrixflow.explore.answer' },
        },
        lastChunk: true,
      }),
      3,
    );

    expect(projection.assistantDraft).toBe('');
    expect(projection.answerDraft).toBe('');
    expect(projection.answer).toBe('最终答案');
    expect(projection.artifacts.map((item) => item.type)).toEqual(['explore.assistant_delta', 'explore.answer']);
    expect(projection.trace).toEqual([]);
  });

  it('completes streaming model traces when stream closes normally', () => {
    let projection = createExploreA2AProjection();
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'assistant_stream_task_1',
          name: 'assistant_delta',
          parts: [{ kind: 'text', text: '最终答案正文' }],
          metadata: { matrixflow_type: 'explore.assistant_delta', data_type: 'matrixflow.explore.assistant_delta' },
        },
        append: true,
      }),
      1,
    );

    const completed = completeExploreA2AProjection(projection);

    expect(completed.final).toBe(true);
    expect(completed.status).toBe('completed');
    expect(completed.trace).toEqual([]);
  });

  it('redacts raw stream failures from the projection', () => {
    let projection = createExploreA2AProjection();
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'assistant_stream_task_1',
          name: 'assistant_delta',
          parts: [{ kind: 'text', text: '部分输出' }],
          metadata: { matrixflow_type: 'explore.assistant_delta', data_type: 'matrixflow.explore.assistant_delta' },
        },
        append: true,
      }),
      1,
    );

    const failed = failExploreA2AProjection(projection, 'stream closed');

    expect(failed.final).toBe(true);
    expect(failed.status).toBe('failed');
    expect(failed.error).toBe('');
    expect(failed.failureDisplay?.i18nKey).toBe('agent.a2a.failure');
    expect(failed.trace).toEqual([]);
  });

  it('marks streaming model trace completed when task completes', () => {
    let projection = createExploreA2AProjection();
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'assistant_stream_task_1',
          name: 'assistant_delta',
          parts: [{ kind: 'text', text: '最终答案正文' }],
          metadata: { matrixflow_type: 'explore.assistant_delta', data_type: 'matrixflow.explore.assistant_delta' },
        },
        append: true,
      }),
      1,
    );

    expect(projection.trace).toEqual([]);

    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'status-update',
        taskId: 'task_1',
        status: {
          state: 'completed',
          message: {
            role: 'agent',
            parts: [{ kind: 'text', text: '分析完成', metadata: { i18n_key: 'explore.status.completed' } }],
          },
        },
        final: true,
      }),
      2,
    );

    expect(projection.final).toBe(true);
    expect(projection.trace).toEqual([]);
  });

  it('appends answer artifact chunks when server streams answer parts', () => {
    let projection = createExploreA2AProjection();
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'answer_task_1',
          name: 'answer',
          parts: [{ kind: 'text', text: '最终' }],
          metadata: { matrixflow_type: 'explore.answer', data_type: 'matrixflow.explore.answer' },
        },
        append: true,
      }),
      1,
    );
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'answer_task_1',
          name: 'answer',
          parts: [{ kind: 'text', text: '答案' }],
          metadata: { matrixflow_type: 'explore.answer', data_type: 'matrixflow.explore.answer' },
        },
        append: true,
      }),
      2,
    );

    expect(projection.answer).toBe('最终答案');
    expect(projection.artifacts).toHaveLength(1);
  });

  it('extracts update_plan artifact into plan view model', () => {
    const projection = reduceExploreA2AResponse(
      createExploreA2AProjection(),
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'plan_task_1',
          name: 'plan',
          parts: [
            {
              kind: 'data',
              data: {
                type: 'matrixflow.explore.plan',
                explanation: '先定位资料',
                plan: [{ step: '查年度报告', status: 'in_progress' }],
              },
            },
          ],
          metadata: {
            matrixflow_type: 'explore.plan',
            data_type: 'matrixflow.explore.plan',
            display: { i18n_key: 'explore.artifact.plan', default_text: '分析计划' },
          },
        },
      }),
      1,
    );

    expect(projection.plan).toEqual({
      explanation: '先定位资料',
      items: [{ step: '查年度报告', status: 'in_progress' }],
    });
    expect(projection.artifacts[0].display).toMatchObject({
      i18nKey: 'explore.artifact.plan',
      defaultText: '分析计划',
    });
  });

  it('does not project update_plan status events as tool trace items', () => {
    let projection = createExploreA2AProjection();
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'status-update',
        taskId: 'task_1',
        status: {
          state: 'working',
          message: {
            role: 'agent',
            parts: [
              {
                kind: 'text',
                text: '调用工具：update_plan',
                metadata: {
                  i18n_key: 'explore.status.tool_started',
                  params: {
                    tool: 'update_plan',
                    call_id: 'call_plan',
                    arguments: { plan: [{ step: '分析问题', status: 'in_progress' }] },
                  },
                },
              },
            ],
          },
        },
      }),
      1,
    );
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'status-update',
        taskId: 'task_1',
        status: {
          state: 'working',
          message: {
            role: 'agent',
            parts: [
              {
                kind: 'text',
                text: '工具完成：update_plan',
                metadata: {
                  i18n_key: 'explore.status.tool_finished',
                  params: { tool: 'update_plan', call_id: 'call_plan', duration_ms: 0 },
                },
              },
            ],
          },
        },
      }),
      2,
    );

    expect(projection.trace).toEqual([]);
  });

  it('projects tool call arguments and artifact result into trace items', () => {
    let projection = createExploreA2AProjection();
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'status-update',
        taskId: 'task_1',
        status: {
          state: 'working',
          message: {
            role: 'agent',
            parts: [
              {
                kind: 'text',
                text: '调用工具：run_sql',
                metadata: {
                  i18n_key: 'explore.status.tool_started',
                  params: {
                    tool: 'run_sql',
                    call_id: 'call_sql',
                    arguments: { sql: 'select 1 as value' },
                  },
                },
              },
            ],
          },
        },
      }),
      1,
    );
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'sql_result_call_sql',
          name: 'sql_result',
          parts: [
            {
              kind: 'data',
              data: {
                columns: [{ name: 'value' }],
                rows: [{ value: 1 }],
                row_count: 1,
                executed_sql: 'select 1 as value',
              },
            },
          ],
          metadata: {
            matrixflow_type: 'explore.sql_result',
            data_type: 'matrixflow.explore.sql_result',
            tool: 'run_sql',
            call_id: 'call_sql',
            display: { i18n_key: 'explore.artifact.sql_result', default_text: '查询结果' },
          },
        },
      }),
      2,
    );
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'status-update',
        taskId: 'task_1',
        status: {
          state: 'working',
          message: {
            role: 'agent',
            parts: [
              {
                kind: 'text',
                text: '工具完成：run_sql',
                metadata: {
                  i18n_key: 'explore.status.tool_finished',
                  params: { tool: 'run_sql', call_id: 'call_sql', duration_ms: 12 },
                },
              },
            ],
          },
        },
      }),
      3,
    );

    expect(projection.trace).toHaveLength(1);
    expect(projection.trace[0]).toMatchObject({
      kind: 'tool',
      callId: 'call_sql',
      name: 'run_sql',
      status: 'completed',
      argumentsText: '{\n  "sql": "select 1 as value"\n}',
      resultText: expect.stringContaining('"value": 1'),
      artifactType: 'explore.sql_result',
      durationMs: 12,
    });
  });

  it('projects final answer source refs from answer artifact metadata', () => {
    const projection = reduceExploreA2AResponse(
      createExploreA2AProjection(),
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'answer_task_1',
          name: 'answer',
          parts: [{ kind: 'text', text: '最终答案' }],
          metadata: {
            matrixflow_type: 'explore.answer',
            source_refs: [
              {
                type: 'rag_chunk',
                artifact_id: 'rag_chunks_call_1',
                chunk_ids: ['chunk_1', 'chunk_2'],
                file_id: 'file_1',
                file_name: 'resume.pdf',
                volume_id: 'volume_1',
                pages: [8, 9],
                object_id: 'obj_1',
                object_kind: 'drawing_view',
                image_file_id: 'image_1',
                page_image_file_id: 'page_image_1',
                bbox: [10, 20, 110, 220],
                visual_refs: [
                  {
                    chunk_id: 'chunk_1',
                    object_id: 'obj_1',
                    object_kind: 'drawing_view',
                    image_file_id: 'image_1',
                    page_image_file_id: 'page_image_1',
                    page: 8,
                    bbox: [10, 20, 110, 220],
                  },
                ],
              },
              {
                type: 'sql_table',
                artifact_id: 'sql_result_call_2',
                database: 'sales',
                table: 'orders',
                sql: 'select * from orders',
              },
            ],
          },
        },
      }),
      1,
    );

    expect(projection.answer).toBe('最终答案');
    expect(projection.answerSources).toEqual([
      expect.objectContaining({
        type: 'rag_chunk',
        artifactId: 'rag_chunks_call_1',
        chunkIds: ['chunk_1', 'chunk_2'],
        fileId: 'file_1',
        fileName: 'resume.pdf',
        volumeId: 'volume_1',
        pages: [8, 9],
        objectId: 'obj_1',
        objectKind: 'drawing_view',
        imageFileId: 'image_1',
        pageImageFileId: 'page_image_1',
        bbox: [10, 20, 110, 220],
        visualRefs: [
          expect.objectContaining({
            chunkId: 'chunk_1',
            imageFileId: 'image_1',
            pageImageFileId: 'page_image_1',
            page: 8,
            bbox: [10, 20, 110, 220],
          }),
        ],
      }),
      expect.objectContaining({
        type: 'sql_table',
        artifactId: 'sql_result_call_2',
        database: 'sales',
        table: 'orders',
        sql: 'select * from orders',
      }),
    ]);
  });

  it('projects session title from session title artifact metadata', () => {
    const projection = reduceExploreA2AResponse(
      createExploreA2AProjection(),
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'session_title_task_1',
          name: 'session_title',
          parts: [{ kind: 'data', data: { type: 'matrixflow.explore.session_title', title: '候选人评价' } }],
          metadata: {
            matrixflow_type: 'explore.session_title',
            title: '候选人评价',
          },
        },
      }),
      1,
    );

    expect(projection.sessionTitle).toBe('候选人评价');
    expect(projection.artifacts).toEqual([]);
    expect(projection.trace).toEqual([]);
    expect(projection.timeline).toEqual([]);
  });

  it('projects llm call metrics from A2A data parts into trace items', () => {
    let projection = createExploreA2AProjection();
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'status-update',
        taskId: 'task_1',
        status: {
          state: 'working',
          message: {
            role: 'agent',
            parts: [
              {
                kind: 'text',
                text: '调用模型：qwen3.6-plus',
                metadata: {
                  i18n_key: 'explore.status.llm_started',
                  params: { call_id: 'turn-1_llm_1', model: 'qwen3.6-plus', turn: 1 },
                },
              },
              {
                kind: 'data',
                data: {
                  type: 'matrixflow.explore.llm_call',
                  status: 'started',
                  call_id: 'turn-1_llm_1',
                  turn: 1,
                  model: 'qwen3.6-plus',
                  message_count: 2,
                  tool_count: 4,
                },
              },
            ],
          },
        },
      }),
      1,
    );
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'status-update',
        taskId: 'task_1',
        status: {
          state: 'working',
          message: {
            role: 'agent',
            parts: [
              {
                kind: 'text',
                text: '模型调用完成：qwen3.6-plus',
                metadata: {
                  i18n_key: 'explore.status.llm_finished',
                  params: { call_id: 'turn-1_llm_1', model: 'qwen3.6-plus', turn: 1 },
                },
              },
              {
                kind: 'data',
                data: {
                  type: 'matrixflow.explore.llm_call',
                  status: 'completed',
                  call_id: 'turn-1_llm_1',
                  turn: 1,
                  model: 'qwen3.6-plus',
                  input_tokens: 100,
                  output_tokens: 20,
                  cached_tokens: 10,
                  reasoning_tokens: 3,
                  event_count: 8,
                  ttft_ms: 1200,
                  duration_ms: 3400,
                },
              },
            ],
          },
        },
      }),
      2,
    );

    expect(projection.trace).toHaveLength(1);
    expect(projection.trace[0]).toMatchObject({
      kind: 'llm',
      callId: 'turn-1_llm_1',
      status: 'completed',
      model: 'qwen3.6-plus',
      turn: 1,
      messageCount: 2,
      toolCount: 4,
      inputTokens: 100,
      outputTokens: 20,
      cachedTokens: 10,
      reasoningTokens: 3,
      eventCount: 8,
      ttftMs: 1200,
      durationMs: 3400,
    });
  });

  it('projects llm call metrics from A2A artifact data parts', () => {
    const projection = reduceExploreA2AResponse(
      createExploreA2AProjection(),
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'llm_call_turn-1',
          name: 'llm_call',
          parts: [
            {
              kind: 'data',
              data: {
                status: 'completed',
                call_id: 'turn-1_llm_1',
                turn: 1,
                model: 'qwen3.6-plus',
                input_tokens: 100,
                output_tokens: 20,
                event_count: 8,
                duration_ms: 3400,
              },
              metadata: {
                data_type: 'matrixflow.explore.llm_call',
                matrixflow_type: 'explore.llm_call',
              },
            },
          ],
          metadata: {
            matrixflow_type: 'explore.llm_call',
            data_type: 'matrixflow.explore.llm_call',
          },
        },
      }),
      1,
    );

    expect(projection.trace).toHaveLength(1);
    expect(projection.trace[0]).toMatchObject({
      kind: 'llm',
      callId: 'turn-1_llm_1',
      status: 'completed',
      model: 'qwen3.6-plus',
      inputTokens: 100,
      outputTokens: 20,
      eventCount: 8,
      durationMs: 3400,
    });
  });

  it('keeps llm text deltas in the llm trace item', () => {
    let projection = createExploreA2AProjection();
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'status-update',
        taskId: 'task_1',
        status: {
          state: 'working',
          message: {
            role: 'agent',
            parts: [
              {
                kind: 'data',
                data: {
                  type: 'matrixflow.explore.llm_call',
                  status: 'started',
                  call_id: 'turn-1_llm_1',
                  turn: 1,
                  model: 'qwen3.6-plus',
                },
              },
            ],
          },
        },
      }),
      1,
    );
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'llm_call_turn-1_llm_1',
          name: 'llm_call',
          parts: [
            {
              kind: 'data',
              data: {
                type: 'matrixflow.explore.llm_call',
                status: 'running',
                call_id: 'turn-1_llm_1',
                turn: 1,
                model: 'qwen3.6-plus',
                thinking_delta: '先分析',
              },
            },
          ],
          metadata: {
            matrixflow_type: 'explore.llm_call',
            data_type: 'matrixflow.explore.llm_call',
          },
        },
        append: true,
      }),
      2,
    );
    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'llm_call_turn-1_llm_1',
          name: 'llm_call',
          parts: [
            {
              kind: 'data',
              data: {
                type: 'matrixflow.explore.llm_call',
                status: 'running',
                call_id: 'turn-1_llm_1',
                turn: 1,
                model: 'qwen3.6-plus',
                content_delta: '结论',
              },
            },
            {
              kind: 'data',
              data: {
                type: 'matrixflow.explore.llm_call',
                status: 'running',
                call_id: 'turn-1_llm_1',
                turn: 1,
                model: 'qwen3.6-plus',
                content_delta: '继续',
              },
            },
            {
              kind: 'data',
              data: {
                type: 'matrixflow.explore.llm_call',
                status: 'running',
                call_id: 'turn-1_llm_1',
                turn: 1,
                model: 'qwen3.6-plus',
                tool_call_name: 'submit_final_answer',
                tool_call_args_delta: '{"answer":"结论',
              },
            },
            {
              kind: 'data',
              data: {
                type: 'matrixflow.explore.llm_call',
                status: 'running',
                call_id: 'turn-1_llm_1',
                turn: 1,
                model: 'qwen3.6-plus',
                tool_call_args_delta: '继续"}',
              },
            },
          ],
          metadata: {
            matrixflow_type: 'explore.llm_call',
            data_type: 'matrixflow.explore.llm_call',
          },
        },
        append: true,
      }),
      3,
    );

    expect(projection.trace).toHaveLength(1);
    expect(projection.trace[0]).toMatchObject({
      kind: 'llm',
      callId: 'turn-1_llm_1',
      model: 'qwen3.6-plus',
      thinking: '先分析',
      content: '结论继续',
      toolCallName: 'submit_final_answer',
      toolCallArgs: '{"answer":"结论继续"}',
    });
  });

  it('deduplicates status thinking when the same content arrives as llm thinking delta', () => {
    let projection = reduceExploreA2AResponse(
      createExploreA2AProjection(),
      response({
        kind: 'status-update',
        taskId: 'task_1',
        status: {
          state: 'working',
          message: {
            role: 'agent',
            parts: [
              {
                kind: 'text',
                text: '用户的问题需要先搜索知识库。',
                metadata: {
                  i18n_key: 'explore.status.thinking',
                  display: {
                    i18n_key: 'explore.status.thinking',
                    default_text: '用户的问题需要先搜索知识库。',
                  },
                },
              },
            ],
          },
        },
      }),
      1,
    );

    expect(projection.trace).toHaveLength(1);
    expect(projection.trace[0]).toMatchObject({
      kind: 'model',
      role: 'thinking',
      content: '用户的问题需要先搜索知识库。',
    });

    projection = reduceExploreA2AResponse(
      projection,
      response({
        kind: 'artifact-update',
        taskId: 'task_1',
        artifact: {
          artifactId: 'llm_call_turn-1_llm_1',
          name: 'llm_call',
          parts: [
            {
              kind: 'data',
              data: {
                type: 'matrixflow.explore.llm_call',
                status: 'running',
                call_id: 'turn-1_llm_1',
                turn: 1,
                model: 'qwen3.5-plus',
                thinking_delta: '用户的问题需要先搜索知识库。',
              },
            },
            {
              kind: 'data',
              data: {
                type: 'matrixflow.explore.llm_call',
                status: 'running',
                call_id: 'turn-1_llm_1',
                turn: 1,
                model: 'qwen3.5-plus',
                content_delta: '我需要先搜索知识库。',
              },
            },
            {
              kind: 'data',
              data: {
                type: 'matrixflow.explore.llm_call',
                status: 'running',
                call_id: 'turn-1_llm_1',
                turn: 1,
                model: 'qwen3.5-plus',
                tool_call_name: 'search_rag_chunks',
                tool_call_args_delta: '{"keywords":["厚度","研发"],"max_hits":10}',
              },
            },
          ],
          metadata: {
            matrixflow_type: 'explore.llm_call',
            data_type: 'matrixflow.explore.llm_call',
          },
        },
        append: true,
      }),
      2,
    );

    expect(projection.trace).toHaveLength(1);
    expect(projection.trace[0]).toMatchObject({
      kind: 'llm',
      callId: 'turn-1_llm_1',
      thinking: '用户的问题需要先搜索知识库。',
      content: '我需要先搜索知识库。',
      toolCallName: 'search_rag_chunks',
      toolCallArgs: '{"keywords":["厚度","研发"],"max_hits":10}',
    });
  });

  it('marks JSON-RPC errors as failed terminal state', () => {
    const projection = reduceExploreA2AResponse(
      createExploreA2AProjection(),
      { jsonrpc: '2.0', id: 'req_1', error: { code: -32000, message: 'runtime failed' } },
      1,
    );

    expect(projection.status).toBe('failed');
    expect(projection.final).toBe(true);
    expect(projection.error).toBe('');
    expect(projection.failureDisplay?.i18nKey).toBe('agent.a2a.failure');
    expect(projection.timeline).toEqual([]);
  });

  it('only omits LLM trace content when it exactly repeats the final answer', () => {
    const trace = [
      {
        id: 'llm-1',
        seq: 1,
        kind: 'llm' as const,
        callId: 'turn-1_llm_1',
        status: 'completed' as const,
        model: 'qwen-plus',
        backendId: null,
        turn: 1,
        messageCount: null,
        toolCount: null,
        inputTokens: null,
        outputTokens: null,
        cachedTokens: null,
        reasoningTokens: null,
        eventCount: null,
        durationMs: null,
        ttftMs: null,
        stopReason: 'tool_calls',
        error: '',
        content: '核对证据后准备提交。',
        thinking: '先分析',
        toolCallArgs: '{"answer":"结论：value 为 1。"}',
        toolCallName: 'submit_final_answer',
      },
    ];

    expect(omitRepeatedFinalAnswerTraceOutput(trace, '结论：value 为 1。')[0]).toMatchObject({
      content: '核对证据后准备提交。',
      thinking: '先分析',
    });

    expect(
      omitRepeatedFinalAnswerTraceOutput([{ ...trace[0], content: '结论：value 为 1。' }], '结论：value 为 1。')[0],
    ).toMatchObject({
      content: '',
      thinking: '先分析',
    });
  });
});

function response(result: AgentA2AResult): AgentA2AResponse<AgentA2AResult> {
  return { jsonrpc: '2.0', id: 'req_1', result };
}
