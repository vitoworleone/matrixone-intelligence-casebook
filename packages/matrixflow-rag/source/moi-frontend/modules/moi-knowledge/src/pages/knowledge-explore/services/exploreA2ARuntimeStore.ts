import { type AgentA2APart, type AgentA2AResponse, type AgentA2AResult } from '@moi/shared-moi-api/agent';
import type { AppHttpClient, AppSSEClient } from '@moi/shared-moi-app-protocol/app-context';
import type { AgentA2AInputRequestView, AgentA2AInputSubmitValues } from '@moi/shared-moi-components/ai-chat-message';
import {
  A2AChatRuntime,
  formatAgentA2AInputSubmitText,
  type MessageSendOptions,
} from '@moi/shared-moi-components/ai-chat-message/a2a-runtime';
import { hasAgentA2AStructuredFailure } from '@moi/shared-moi-components/ai-chat-message/agent-a2a-projection';
import type { MessageRecord } from '../../../service/dialogSession';
import type { KnowledgeListItem } from '../../../service/knowledge';
import {
  completeExploreA2AProjection,
  createExploreA2AProjection,
  failExploreA2AProjection,
  reduceExploreA2AResponse,
  type ExploreA2AProjection,
} from './exploreA2AStreamParser';

export interface ExploreA2AControlState {
  enableDeepAttribution: boolean;
  enableQualityEvaluation: boolean;
  enableHallucinationEvaluation: boolean;
}

export type ExploreA2AControlOptionKey = keyof ExploreA2AControlState;

export interface ExploreA2AMessageItem {
  id: string;
  messageId?: number;
  role: 'user' | 'assistant';
  content: string;
  createdAt: number;
  knowledgeIds: number[];
  queryVisuals?: ExploreA2AQueryVisual[];
  projection?: ExploreA2AProjection;
  feedback?: 'like' | 'dislike' | null;
}

export interface ExploreA2ARuntimeState extends ExploreA2AControlState {
  draftInput: string;
  selectedModel: string;
  selectedModelBackendId: number | null;
  selectedReasoningEffort: string;
  selectedKnowledgeIds: number[];
  isSendPending: boolean;
  isStreaming: boolean;
  streamError: string;
  taskId: string | null;
  localTaskId: string | null;
  contextId: string | null;
  connectionId: string | null;
  eventCursor: number;
  recoveryAttempts: number;
  pendingQuestion: string;
  projection: ExploreA2AProjection;
  messages: ExploreA2AMessageItem[];
  streamStartedAt: number | null;
  streamEndedAt: number | null;
  updatedAt: number;
  queryVisuals: ExploreA2AQueryVisual[];
}

export type ExploreA2ARuntimeStateMap = Record<number, ExploreA2ARuntimeState>;

export interface ExploreA2AQueryVisual {
  fileId: string;
  fileName: string;
  mimeType: string;
  size: number;
}

export interface SendExploreA2AMessageParams {
  sessionId: number;
  question: string;
  sessionTitleQuestion?: string;
  queryVisuals?: ExploreA2AQueryVisual[];
  model: string;
  modelBackendId?: number | null;
  reasoningEffort?: string;
  selectedKnowledgeIds: number[];
  workspaceId: string;
  sseClient: AppSSEClient;
  knowledgeList: KnowledgeListItem[];
  appContext?: Record<string, unknown>;
  streamErrorFallbackText: string;
  beforeCreateStream?: () => Promise<void>;
}

type RuntimeListener = () => void;

const MAX_CACHED_RUNTIME_SESSION_COUNT = 10;
const MAX_STREAM_RECOVERY_ATTEMPTS = 3;
const AGENT_CODE = 'explore';
const SESSION_TITLE_QUESTION_METADATA_KEY = 'session_title_question';

export function createDefaultA2ARuntimeState(): ExploreA2ARuntimeState {
  return {
    draftInput: '',
    selectedModel: '',
    selectedModelBackendId: null,
    selectedReasoningEffort: '',
    selectedKnowledgeIds: [],
    enableDeepAttribution: false,
    enableQualityEvaluation: false,
    enableHallucinationEvaluation: false,
    isSendPending: false,
    isStreaming: false,
    streamError: '',
    taskId: null,
    localTaskId: null,
    contextId: null,
    connectionId: null,
    eventCursor: 0,
    recoveryAttempts: 0,
    pendingQuestion: '',
    projection: createExploreA2AProjection(),
    messages: [],
    streamStartedAt: null,
    streamEndedAt: null,
    updatedAt: Date.now(),
    queryVisuals: [],
  };
}

class ExploreA2ARuntimeStore {
  private runtimeStateMap: ExploreA2ARuntimeStateMap = {};
  private listeners = new Set<RuntimeListener>();
  private a2aRuntime = new A2AChatRuntime<number>();
  private runtimeAccessTimeMap = new Map<number, number>();
  private pendingSendSessions = new Set<number>();

  subscribe(listener: RuntimeListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  getRuntimeStateMap = (): ExploreA2ARuntimeStateMap => {
    return this.runtimeStateMap;
  };

  ensureSessionRuntime(sessionId: number): ExploreA2ARuntimeState {
    if (sessionId <= 0) {
      console.warn('[ExploreA2ARuntimeStore.ensureSessionRuntime] invalid session id', { sessionId });
      return createDefaultA2ARuntimeState();
    }
    const existing = this.runtimeStateMap[sessionId];
    if (existing) {
      this.touchRuntimeAccess(sessionId);
      return existing;
    }
    const next = {
      ...this.runtimeStateMap,
      [sessionId]: createDefaultA2ARuntimeState(),
    };
    this.runtimeStateMap = next;
    this.touchRuntimeAccess(sessionId);
    this.evictRuntimeLRUIfNeeded(sessionId);
    this.emitRuntimeChange();
    return next[sessionId];
  }

  clearSessionRuntime(sessionId: number): void {
    this.a2aRuntime.abort(sessionId);
    if (!(sessionId in this.runtimeStateMap)) {
      return;
    }
    const next = { ...this.runtimeStateMap };
    delete next[sessionId];
    this.runtimeStateMap = next;
    this.runtimeAccessTimeMap.delete(sessionId);
    this.pendingSendSessions.delete(sessionId);
    this.emitRuntimeChange();
  }

  hasActiveConnection(sessionId: number): boolean {
    return this.a2aRuntime.hasActiveStream(sessionId);
  }

  syncSessionDefaultKnowledge(sessionId: number, defaultKnowledgeIds: number[]): void {
    if (sessionId <= 0) return;
    const normalized = normalizeKnowledgeIds(defaultKnowledgeIds);
    this.patchRuntime(sessionId, (previous) => {
      if (previous.selectedKnowledgeIds.length > 0 || normalized.length === 0) {
        return previous;
      }
      return { ...previous, selectedKnowledgeIds: normalized, updatedAt: Date.now() };
    });
  }

  syncSessionDefaultModel(sessionId: number, defaultModel: string, defaultBackendId?: number | null): void {
    if (sessionId <= 0) return;
    const model = defaultModel.trim();
    if (!model) return;
    this.patchRuntime(sessionId, (previous) => {
      if (previous.selectedModel.trim()) {
        return previous;
      }
      return {
        ...previous,
        selectedModel: model,
        selectedModelBackendId: normalizeBackendId(defaultBackendId),
        updatedAt: Date.now(),
      };
    });
  }

  syncSessionDefaultReasoningEffort(sessionId: number, defaultReasoningEffort: string): void {
    if (sessionId <= 0) return;
    if (defaultReasoningEffort === '') return;
    this.patchRuntime(sessionId, (previous) => {
      if (previous.selectedReasoningEffort !== '') {
        return previous;
      }
      return {
        ...previous,
        selectedReasoningEffort: defaultReasoningEffort,
        updatedAt: Date.now(),
      };
    });
  }

  setDraftInput(sessionId: number, value: string): void {
    this.patchRuntime(sessionId, (previous) => ({ ...previous, draftInput: value, updatedAt: Date.now() }));
  }

  addQueryVisual(sessionId: number, image: ExploreA2AQueryVisual): void {
    const normalized = normalizeQueryVisual(image);
    if (!normalized) {
      console.warn('[ExploreA2ARuntimeStore.addQueryVisual] invalid query visual', { sessionId, image });
      return;
    }
    this.patchRuntime(sessionId, (previous) => ({
      ...previous,
      queryVisuals: normalizeQueryVisuals([...previous.queryVisuals, normalized]),
      updatedAt: Date.now(),
    }));
  }

  removeQueryVisual(sessionId: number, fileId: string): void {
    const target = fileId.trim();
    if (!target) return;
    this.patchRuntime(sessionId, (previous) => ({
      ...previous,
      queryVisuals: previous.queryVisuals.filter((image) => image.fileId !== target),
      updatedAt: Date.now(),
    }));
  }

  setSelectedModel(sessionId: number, model: string, backendId?: number | null): void {
    this.patchRuntime(sessionId, (previous) => ({
      ...previous,
      selectedModel: model,
      selectedModelBackendId: normalizeBackendId(backendId),
      updatedAt: Date.now(),
    }));
  }

  toggleKnowledgeSelection(sessionId: number, knowledgeId: number): void {
    if (!Number.isInteger(knowledgeId) || knowledgeId <= 0) {
      console.warn('[ExploreA2ARuntimeStore.toggleKnowledgeSelection] invalid knowledge id', { sessionId, knowledgeId });
      return;
    }
    this.patchRuntime(sessionId, (previous) => {
      const selected = previous.selectedKnowledgeIds.includes(knowledgeId)
        ? previous.selectedKnowledgeIds.filter((id) => id !== knowledgeId)
        : [...previous.selectedKnowledgeIds, knowledgeId];
      return { ...previous, selectedKnowledgeIds: normalizeKnowledgeIds(selected), updatedAt: Date.now() };
    });
  }

  removeKnowledgeSelection(sessionId: number, knowledgeId: number): void {
    this.patchRuntime(sessionId, (previous) => ({
      ...previous,
      selectedKnowledgeIds: previous.selectedKnowledgeIds.filter((id) => id !== knowledgeId),
      updatedAt: Date.now(),
    }));
  }

  setKnowledgeSelection(sessionId: number, knowledgeIds: number[]): void {
    const normalized = normalizeKnowledgeIds(knowledgeIds);
    this.patchRuntime(sessionId, (previous) => ({
      ...previous,
      selectedKnowledgeIds: sameNumberList(previous.selectedKnowledgeIds, normalized)
        ? previous.selectedKnowledgeIds
        : normalized,
      updatedAt: Date.now(),
    }));
  }

  setControlOption(sessionId: number, key: ExploreA2AControlOptionKey, value: boolean): void {
    this.patchRuntime(sessionId, (previous) => ({ ...previous, [key]: value, updatedAt: Date.now() }));
  }

  patchSendPending(sessionId: number, isSendPending: boolean): void {
    this.patchRuntime(sessionId, (previous) => ({ ...previous, isSendPending, updatedAt: Date.now() }));
  }

  hydrateSessionMessages(sessionId: number, records: MessageRecord[], requestedAt = Date.now()): void {
    if (sessionId <= 0) {
      console.warn('[ExploreA2ARuntimeStore.hydrateSessionMessages] invalid session id', { sessionId });
      return;
    }
    const current = this.runtimeStateMap[sessionId];
    if (this.a2aRuntime.hasActiveStream(sessionId) || current?.isSendPending || current?.isStreaming) {
      return;
    }
    if (current && current.messages.length > 0 && current.updatedAt > requestedAt) {
      return;
    }
    const messages = buildMessagesFromRecords(records);
    const latestProjection = latestAssistantProjection(messages);
    this.patchRuntime(sessionId, (previous) => ({
      ...previous,
      messages,
      isSendPending: false,
      isStreaming: false,
      streamError: '',
      pendingQuestion: '',
      projection: createExploreA2AProjection(),
      taskId: latestProjection?.taskId ?? previous.taskId,
      contextId: latestProjection?.contextId ?? previous.contextId,
      connectionId: null,
      eventCursor: 0,
      recoveryAttempts: 0,
      streamEndedAt: null,
      updatedAt: Date.now(),
    }));
  }

  async sendSessionMessage(params: SendExploreA2AMessageParams): Promise<boolean> {
    if (params.sessionId <= 0) {
      console.warn('[ExploreA2ARuntimeStore.sendSessionMessage] invalid session id', { sessionId: params.sessionId });
      return false;
    }
    const question = params.question.trim();
    const queryVisuals = normalizeQueryVisuals(params.queryVisuals ?? []);
    if (!question && queryVisuals.length === 0) {
      console.warn('[ExploreA2ARuntimeStore.sendSessionMessage] empty question and image list', { sessionId: params.sessionId });
      return false;
    }
    const selectedKnowledgeIds = normalizeKnowledgeIds(params.selectedKnowledgeIds);
    if (selectedKnowledgeIds.length === 0) {
      console.warn('[ExploreA2ARuntimeStore.sendSessionMessage] knowledge list is empty', { sessionId: params.sessionId });
      return false;
    }
    if (this.pendingSendSessions.has(params.sessionId) || this.a2aRuntime.hasActiveStream(params.sessionId)) {
      console.warn('[ExploreA2ARuntimeStore.sendSessionMessage] stream is already active', { sessionId: params.sessionId });
      return false;
    }

    this.pendingSendSessions.add(params.sessionId);
    this.patchSendPending(params.sessionId, true);
    try {
      await params.beforeCreateStream?.();
      const taskId = createTaskId(params.sessionId);
      const contextId = `explore_session_${params.sessionId}`;
      const requestId = `req_${taskId}`;
      const connectionId = `conn_${taskId}`;
      const projection = createExploreA2AProjection();
      const scopeMetadata = buildScopeMetadata({
        sessionId: params.sessionId,
        workspaceId: params.workspaceId,
        selectedKnowledgeIds,
        knowledgeList: params.knowledgeList,
      });
      const requestMetadata =
        params.appContext === undefined ? scopeMetadata : { ...scopeMetadata, app_context: params.appContext };
      const now = Date.now();
      const pendingQuestion = question || queryVisuals.map((image) => image.fileName || image.fileId).join(', ');

      this.patchRuntime(params.sessionId, (previous) => ({
        ...previous,
        isSendPending: false,
        isStreaming: true,
        streamError: '',
        taskId,
        localTaskId: taskId,
        contextId,
        connectionId,
        eventCursor: 0,
        recoveryAttempts: 0,
        pendingQuestion,
        projection,
        streamStartedAt: now,
        streamEndedAt: null,
        messages: [
          ...previous.messages,
          {
            id: `user_${taskId}`,
            role: 'user',
            content: question,
            createdAt: now,
            knowledgeIds: selectedKnowledgeIds,
            queryVisuals,
          },
          {
            id: `assistant_${taskId}`,
            role: 'assistant',
            content: '',
            createdAt: now,
            knowledgeIds: selectedKnowledgeIds,
            projection,
            feedback: null,
          },
        ],
        draftInput: '',
        queryVisuals: [],
        updatedAt: now,
      }));

      let localSeq = 0;
      this.a2aRuntime.streamUserMessage({
        key: params.sessionId,
        selector: { agent_code: AGENT_CODE },
        requestId,
        sseClient: params.sseClient,
        streamOptions: params.workspaceId ? { headers: { 'X-Workspace-ID': params.workspaceId } } : undefined,
        options: buildMessageParams({
          taskId,
          contextId,
          question,
          sessionTitleQuestion: params.sessionTitleQuestion,
          queryVisuals,
          model: params.model,
          modelBackendId: params.modelBackendId,
          reasoningEffort: params.reasoningEffort,
          metadata: requestMetadata,
        }),
        onResponse: (response, event) => {
          const parsedSeq = Number(event.id);
          const seq = Number.isFinite(parsedSeq) && parsedSeq > 0 ? parsedSeq : ++localSeq;
          this.applyA2AResponse(params.sessionId, taskId, response, seq, params.streamErrorFallbackText);
        },
        onComplete: () => {
          this.handleStreamClosure({
            sessionId: params.sessionId,
            connectionId,
            workspaceId: params.workspaceId,
            sseClient: params.sseClient,
            streamErrorFallbackText: params.streamErrorFallbackText,
          });
        },
        onError: (error) => {
          const message = (error as Error)?.message || params.streamErrorFallbackText;
          this.handleStreamClosure({
            sessionId: params.sessionId,
            connectionId,
            workspaceId: params.workspaceId,
            sseClient: params.sseClient,
            streamErrorFallbackText: params.streamErrorFallbackText,
            errorMessage: message,
          });
        },
      });
      this.patchSendPending(params.sessionId, false);
      return true;
    } catch (error) {
      const message = (error as Error).message || params.streamErrorFallbackText;
      this.markStreamInterrupted(params.sessionId, message);
      console.warn('[ExploreA2ARuntimeStore.sendSessionMessage] failed', { sessionId: params.sessionId, msg: message });
      return false;
    } finally {
      this.pendingSendSessions.delete(params.sessionId);
      this.patchSendPending(params.sessionId, false);
    }
  }

  async stopSessionStream(sessionId: number, http: AppHttpClient): Promise<boolean> {
    const state = this.runtimeStateMap[sessionId];
    const result = await this.a2aRuntime.cancelTask({
      key: sessionId,
      selector: { agent_code: AGENT_CODE },
      http,
      taskId: state?.taskId,
      onResponse: (response, taskId) => {
        this.applyA2AResponse(sessionId, taskId, response, Date.now(), 'Canceled');
      },
    });
    if (!result) {
      return false;
    }
    this.finishTerminalStream(sessionId, state?.connectionId ?? '');
    return true;
  }

  async submitSessionInput(
    sessionId: number,
    request: AgentA2AInputRequestView,
    values: AgentA2AInputSubmitValues,
    params: {
      workspaceId: string;
      sseClient: AppSSEClient;
      knowledgeList: KnowledgeListItem[];
      appContext?: Record<string, unknown>;
      streamErrorFallbackText: string;
    },
  ): Promise<void> {
    if (sessionId <= 0) {
      throw new Error('session id is required for input submit');
    }
    if (!request.contextId) {
      throw new Error('context_id is required for input submit');
    }
    const contextId = request.contextId;
    if (this.pendingSendSessions.has(sessionId) || this.a2aRuntime.hasActiveStream(sessionId)) {
      throw new Error('session stream is already active');
    }
    const state = this.runtimeStateMap[sessionId];
    if (!state) {
      throw new Error('runtime session state is required for input submit');
    }
    const selectedKnowledgeIds = normalizeKnowledgeIds(state.selectedKnowledgeIds);
    if (selectedKnowledgeIds.length === 0) {
      throw new Error('knowledge list is required for input submit');
    }
    const messageText = formatAgentA2AInputSubmitText(request, values);
    if (!messageText) {
      throw new Error('input submit message is required');
    }
    this.pendingSendSessions.add(sessionId);
    const taskId = createTaskId(sessionId);
    const requestId = `req_${taskId}`;
    const connectionId = `conn_${taskId}`;
    const projection = createExploreA2AProjection();
    const scopeMetadata = buildScopeMetadata({
      sessionId,
      workspaceId: params.workspaceId,
      selectedKnowledgeIds,
      knowledgeList: params.knowledgeList,
    });
    const requestMetadata =
      params.appContext === undefined ? scopeMetadata : { ...scopeMetadata, app_context: params.appContext };
    const now = Date.now();
    this.patchRuntime(sessionId, (previous) => ({
      ...previous,
      isSendPending: false,
      isStreaming: true,
      streamError: '',
      taskId,
      localTaskId: taskId,
      contextId,
      connectionId,
      eventCursor: 0,
      recoveryAttempts: 0,
      pendingQuestion: messageText,
      projection,
      streamStartedAt: now,
      streamEndedAt: null,
      messages: [
        ...previous.messages.map((item) =>
          item.id === `assistant_${previous.localTaskId || request.taskId}` && item.projection
            ? { ...item, projection: { ...item.projection, pendingInput: null } }
            : item,
        ),
        {
          id: `user_${taskId}`,
          role: 'user',
          content: messageText,
          createdAt: now,
          knowledgeIds: selectedKnowledgeIds,
        },
        {
          id: `assistant_${taskId}`,
          role: 'assistant',
          content: '',
          createdAt: now,
          knowledgeIds: selectedKnowledgeIds,
          projection,
          feedback: null,
        },
      ],
      updatedAt: now,
    }));
    try {
      let localSeq = 0;
      this.a2aRuntime.streamUserMessage({
        key: sessionId,
        selector: { agent_code: AGENT_CODE },
        requestId,
        sseClient: params.sseClient,
        streamOptions: params.workspaceId ? { headers: { 'X-Workspace-ID': params.workspaceId } } : undefined,
        options: buildMessageParams({
          taskId,
          contextId,
          question: messageText,
          queryVisuals: [],
          model: state.selectedModel,
          modelBackendId: state.selectedModelBackendId,
          reasoningEffort: state.selectedReasoningEffort,
          metadata: requestMetadata,
        }),
        onResponse: (response, event) => {
          const parsedSeq = Number(event.id);
          const seq = Number.isFinite(parsedSeq) && parsedSeq > 0 ? parsedSeq : ++localSeq;
          this.applyA2AResponse(sessionId, taskId, response, seq, params.streamErrorFallbackText);
        },
        onComplete: () => {
          this.handleStreamClosure({
            sessionId,
            connectionId,
            workspaceId: params.workspaceId,
            sseClient: params.sseClient,
            streamErrorFallbackText: params.streamErrorFallbackText,
          });
        },
        onError: (error) => {
          const message = (error as Error)?.message || params.streamErrorFallbackText;
          this.handleStreamClosure({
            sessionId,
            connectionId,
            workspaceId: params.workspaceId,
            sseClient: params.sseClient,
            streamErrorFallbackText: params.streamErrorFallbackText,
            errorMessage: message,
          });
        },
      });
    } finally {
      this.pendingSendSessions.delete(sessionId);
      this.patchSendPending(sessionId, false);
    }
  }

  private resubscribeSessionStream({
    sessionId,
    workspaceId,
    sseClient,
    streamErrorFallbackText,
    errorMessage,
  }: {
    sessionId: number;
    workspaceId: string;
    sseClient: AppSSEClient;
    streamErrorFallbackText: string;
    errorMessage?: string;
  }): void {
    const state = this.runtimeStateMap[sessionId];
    if (!state?.taskId || !state.contextId) {
      console.warn('[ExploreA2ARuntimeStore.resubscribeSessionStream] missing task identity', {
        sessionId,
        taskId: state?.taskId,
        contextId: state?.contextId,
      });
      this.markStreamInterrupted(sessionId, errorMessage || streamErrorFallbackText);
      return;
    }
    if (state.recoveryAttempts >= MAX_STREAM_RECOVERY_ATTEMPTS) {
      console.warn('[ExploreA2ARuntimeStore.resubscribeSessionStream] recovery attempts exhausted', {
        sessionId,
        taskId: state.taskId,
        eventCursor: state.eventCursor,
        recoveryAttempts: state.recoveryAttempts,
      });
      this.markStreamInterrupted(sessionId, errorMessage || streamErrorFallbackText);
      return;
    }
    const taskId = state.taskId;
    const contextId = state.contextId;
    const afterSeq = state.eventCursor;
    const recoveryAttempt = state.recoveryAttempts + 1;
    const connectionId = `conn_resub_${taskId}_${recoveryAttempt}_${Date.now()}`;
    let localSeq = afterSeq;
    this.patchRuntime(sessionId, (previous) => ({
      ...previous,
      connectionId,
      recoveryAttempts: previous.recoveryAttempts + 1,
      isStreaming: true,
      streamError: '',
      updatedAt: Date.now(),
    }));
    try {
      this.a2aRuntime.resubscribeTask({
        key: sessionId,
        selector: { agent_code: AGENT_CODE },
        taskId,
        sseClient,
        requestId: `resub_${taskId}_${recoveryAttempt}_${Date.now()}`,
        afterSeq,
        contextId,
        connectionId,
        streamOptions: workspaceId ? { headers: { 'X-Workspace-ID': workspaceId } } : undefined,
        onResponse: (response, event) => {
          const parsedSeq = Number(event.id);
          const seq = Number.isFinite(parsedSeq) && parsedSeq > 0 ? parsedSeq : ++localSeq;
          this.applyA2AResponse(sessionId, taskId, response, seq, streamErrorFallbackText);
        },
        onComplete: () => {
          this.handleStreamClosure({
            sessionId,
            connectionId,
            workspaceId,
            sseClient,
            streamErrorFallbackText,
          });
        },
        onError: (error) => {
          const message = (error as Error)?.message || streamErrorFallbackText;
          this.handleStreamClosure({
            sessionId,
            connectionId,
            workspaceId,
            sseClient,
            streamErrorFallbackText,
            errorMessage: message,
          });
        },
      });
    } catch (error) {
      const message = (error as Error)?.message || streamErrorFallbackText;
      this.handleStreamClosure({
        sessionId,
        connectionId,
        workspaceId,
        sseClient,
        streamErrorFallbackText,
        errorMessage: message,
      });
    }
  }

  private applyA2AResponse(
    sessionId: number,
    taskId: string,
    response: AgentA2AResponse<AgentA2AResult>,
    seq: number,
    fallbackErrorText: string,
  ): void {
    this.patchRuntime(sessionId, (previous) => {
      if (seq > 0 && seq <= previous.eventCursor) {
        console.warn('[ExploreA2ARuntimeStore.applyA2AResponse] duplicate or stale event ignored', {
          sessionId,
          taskId,
          seq,
          eventCursor: previous.eventCursor,
        });
        return previous;
      }
      const projection = reduceExploreA2AResponse(previous.projection, response, seq);
      const structuredFailure = hasStructuredFailure(projection);
      const error = structuredFailure
        ? ''
        : projection.error || (response.error ? response.error.message || fallbackErrorText : previous.streamError);
      const canceledMessage = isCanceledProjection(projection) ? canceledMessageText(response) : '';
      const content =
        projection.visibleAnswer ||
        (structuredFailure ? '' : error || canceledMessage || previous.messages.at(-1)?.content || '');
      const nextTaskId = projection.taskId || previous.taskId;
      const messageTaskId = previous.localTaskId || taskId;
      return {
        ...previous,
        projection,
        taskId: nextTaskId,
        contextId: projection.contextId || previous.contextId,
        eventCursor: seq > 0 ? seq : previous.eventCursor,
        recoveryAttempts: 0,
        streamError: error,
        isStreaming: projection.final ? false : previous.isStreaming,
        streamEndedAt: projection.final ? Date.now() : previous.streamEndedAt,
        messages: previous.messages.map((item) =>
          item.id === `assistant_${messageTaskId}` ? { ...item, content, projection } : item,
        ),
        updatedAt: Date.now(),
      };
    });
    const state = this.runtimeStateMap[sessionId];
    if (state?.projection.final) {
      this.a2aRuntime.abort(sessionId);
    }
  }

  private handleStreamClosure({
    sessionId,
    connectionId,
    workspaceId,
    sseClient,
    streamErrorFallbackText,
    errorMessage,
  }: {
    sessionId: number;
    connectionId: string;
    workspaceId: string;
    sseClient: AppSSEClient;
    streamErrorFallbackText: string;
    errorMessage?: string;
  }): void {
    const state = this.runtimeStateMap[sessionId];
    if (!state) {
      console.warn('[ExploreA2ARuntimeStore.handleStreamClosure] runtime state missing', { sessionId });
      return;
    }
    if (state.connectionId !== connectionId) {
      console.warn('[ExploreA2ARuntimeStore.handleStreamClosure] stale connection closure ignored', {
        sessionId,
        connectionId,
        activeConnectionId: state.connectionId,
      });
      return;
    }
    if (state.projection.final) {
      this.finishTerminalStream(sessionId, connectionId);
      return;
    }
    // Direct a2a.Message responses (authoring clarify/reject) never create a
    // runtime task. The local taskId is only a UI key, so resubscribe would 404.
    if (state.eventCursor <= 0) {
      const completed = completeExploreA2AProjection(state.projection);
      this.patchRuntime(sessionId, (previous) => {
        if (previous.connectionId !== connectionId) {
          return previous;
        }
        const messageTaskId = previous.localTaskId || previous.taskId || '';
        const content = completed.visibleAnswer || previous.messages.at(-1)?.content || '';
        return {
          ...previous,
          projection: completed,
          isStreaming: false,
          isSendPending: false,
          streamError: hasStructuredFailure(completed) ? '' : completed.error || previous.streamError,
          streamEndedAt: Date.now(),
          messages: previous.messages.map((item) =>
            item.id === `assistant_${messageTaskId}` ? { ...item, content, projection: completed } : item,
          ),
          updatedAt: Date.now(),
        };
      });
      this.a2aRuntime.abort(sessionId);
      return;
    }
    this.resubscribeSessionStream({
      sessionId,
      workspaceId,
      sseClient,
      streamErrorFallbackText,
      errorMessage,
    });
  }

  private finishTerminalStream(sessionId: number, connectionId: string): void {
    this.patchRuntime(sessionId, (previous) => {
      if (connectionId && previous.connectionId !== connectionId) {
        console.warn('[ExploreA2ARuntimeStore.finishTerminalStream] stale connection ignored', {
          sessionId,
          connectionId,
          activeConnectionId: previous.connectionId,
        });
        return previous;
      }
      if (!previous.projection.final) {
        console.warn('[ExploreA2ARuntimeStore.finishTerminalStream] terminal event missing', {
          sessionId,
          taskId: previous.taskId,
          eventCursor: previous.eventCursor,
        });
        return previous;
      }
      return {
        ...previous,
        isStreaming: false,
        isSendPending: false,
        streamError: hasStructuredFailure(previous.projection) ? '' : previous.projection.error,
        streamEndedAt: previous.streamEndedAt || Date.now(),
        updatedAt: Date.now(),
      };
    });
  }

  private markStreamInterrupted(sessionId: number, errorMessage: string): void {
    this.a2aRuntime.abort(sessionId);
    this.patchRuntime(sessionId, (previous) => {
      // A terminal projection already owns the user-visible outcome. Keep it and
      // only stop local streaming flags so concurrent multi-session recovery does
      // not wipe a completed/failed answer with a transport-level interrupt.
      if (previous.projection.final) {
        return {
          ...previous,
          isStreaming: false,
          isSendPending: false,
          streamError: hasStructuredFailure(previous.projection) ? '' : previous.streamError || errorMessage,
          streamEndedAt: previous.streamEndedAt || Date.now(),
          updatedAt: Date.now(),
        };
      }

      // streamError alone is not rendered by the chat transcript. Fail the live
      // assistant projection so multi-session transport errors surface a structured
      // failure alert instead of an unfinished bubble / opaque unknown state.
      const failedProjection = {
        ...failExploreA2AProjection(previous.projection, errorMessage),
        failureDisplay: {
          i18nKey: 'agent.a2a.stream_interrupted',
          defaultText: errorMessage || '',
          params: {},
        },
      };
      const messageTaskId = previous.localTaskId || previous.taskId || '';
      return {
        ...previous,
        projection: failedProjection,
        isStreaming: false,
        isSendPending: false,
        streamError: '',
        streamEndedAt: Date.now(),
        messages: previous.messages.map((item) =>
          item.id === `assistant_${messageTaskId}` ? { ...item, content: '', projection: failedProjection } : item,
        ),
        updatedAt: Date.now(),
      };
    });
  }

  private patchRuntime(sessionId: number, updater: (previous: ExploreA2ARuntimeState) => ExploreA2ARuntimeState): void {
    if (sessionId <= 0) {
      console.warn('[ExploreA2ARuntimeStore.patchRuntime] invalid session id', { sessionId });
      return;
    }
    const previous = this.runtimeStateMap[sessionId] ?? createDefaultA2ARuntimeState();
    const next = updater(previous);
    if (next === previous) {
      return;
    }
    this.runtimeStateMap = {
      ...this.runtimeStateMap,
      [sessionId]: next,
    };
    this.touchRuntimeAccess(sessionId);
    this.evictRuntimeLRUIfNeeded(sessionId);
    this.emitRuntimeChange();
  }

  private emitRuntimeChange(): void {
    this.listeners.forEach((listener) => listener());
  }

  private touchRuntimeAccess(sessionId: number): void {
    this.runtimeAccessTimeMap.set(sessionId, Date.now());
  }

  private evictRuntimeLRUIfNeeded(protectedSessionId: number): void {
    const sessionIds = Object.keys(this.runtimeStateMap).map(Number);
    if (sessionIds.length <= MAX_CACHED_RUNTIME_SESSION_COUNT) {
      return;
    }
    const candidates = sessionIds
      .filter((id) => id !== protectedSessionId && !this.a2aRuntime.hasActiveStream(id))
      .sort((left, right) => (this.runtimeAccessTimeMap.get(left) ?? 0) - (this.runtimeAccessTimeMap.get(right) ?? 0));
    const evictId = candidates[0];
    if (!evictId) {
      return;
    }
    const next = { ...this.runtimeStateMap };
    delete next[evictId];
    this.runtimeStateMap = next;
    this.runtimeAccessTimeMap.delete(evictId);
  }
}

function buildMessageParams({
  taskId,
  contextId,
  question,
  sessionTitleQuestion,
  queryVisuals,
  model,
  modelBackendId,
  reasoningEffort,
  metadata,
}: {
  taskId: string;
  contextId: string;
  question: string;
  sessionTitleQuestion?: string;
  queryVisuals: ExploreA2AQueryVisual[];
  model: string;
  modelBackendId?: number | null;
  reasoningEffort?: string;
  metadata: Record<string, unknown>;
}): MessageSendOptions {
  const normalizedBackendId = normalizeBackendId(modelBackendId);
  const queryVisualMetadata = normalizeQueryVisuals(queryVisuals).map((image) => ({
    file_id: image.fileId,
    file_name: image.fileName,
    mime_type: image.mimeType,
    size: image.size,
  }));
  const requestMetadata: Record<string, unknown> = {
    ...metadata,
    ...(queryVisualMetadata.length > 0 ? { query_visuals: queryVisualMetadata } : {}),
    ...(normalizedBackendId !== null ? { llm_backend_id: normalizedBackendId } : {}),
    ...(reasoningEffort !== undefined && reasoningEffort !== '' ? { llm_reasoning_effort: reasoningEffort } : {}),
  };
  const titleQuestion = sessionTitleQuestion?.trim();
  const messageMetadata = titleQuestion
    ? { ...requestMetadata, [SESSION_TITLE_QUESTION_METADATA_KEY]: titleQuestion }
    : requestMetadata;
  return {
    taskId,
    contextId,
    model,
    ...(normalizedBackendId !== null ? { llm_backend_id: normalizedBackendId } : {}),
    metadata: requestMetadata,
    parts: buildMessageParts(question, queryVisuals),
    messageMetadata,
  };
}

function buildMessageParts(question: string, queryVisuals: ExploreA2AQueryVisual[]): AgentA2APart[] {
  const parts: AgentA2APart[] = [];
  if (question.trim()) {
    parts.push({ kind: 'text', text: question.trim() });
  }
  for (const image of normalizeQueryVisuals(queryVisuals)) {
    const data = {
      file_id: image.fileId,
      file_name: image.fileName,
      mime_type: image.mimeType,
      size: image.size,
    };
    parts.push({
      kind: 'image',
      data,
      metadata: data,
    });
  }
  return parts;
}

function buildScopeMetadata({
  sessionId,
  workspaceId,
  selectedKnowledgeIds,
  knowledgeList,
}: {
  sessionId: number;
  workspaceId: string;
  selectedKnowledgeIds: number[];
  knowledgeList: KnowledgeListItem[];
}): Record<string, unknown> {
  const selected = selectedKnowledgeIds
    .map((id) => knowledgeList.find((item) => item.id === id))
    .filter((item): item is KnowledgeListItem => Boolean(item));
  const fileIds = uniqueStrings(selected.flatMap((item) => modelScopedFileIDs(item)));
  const tables = uniqueStrings(selected.flatMap((item) => item.tables?.flatMap((table) => table.table_names ?? []) ?? []));
  const databases = uniqueStrings(selected.flatMap((item) => item.tables?.map((table) => table.db_name) ?? []));
  const semanticModelIds = selectedKnowledgeIds.map(String).join(',');
  const semanticModelNames = selected
    .map((item) => item.name)
    .filter(Boolean)
    .join(',');
  const scopeMetadata: Record<string, string> = {
    semantic_model_ids: semanticModelIds,
  };
  if (semanticModelNames) {
    scopeMetadata.semantic_model_names = semanticModelNames;
  }
  const scope: Record<string, unknown> = {
    ...(workspaceId ? { workspace_id: workspaceId } : {}),
    session_id: String(sessionId),
    scope_metadata: scopeMetadata,
  };
  if (fileIds.length > 0) {
    scope.file_ids = fileIds;
  }
  if (tables.length > 0) {
    scope.tables = tables;
  }
  if (databases.length === 1) {
    scope.database = databases[0];
  }
  return {
    matrixflow_client: 'moi-frontend',
    workspace_id: workspaceId,
    session_id: String(sessionId),
    scope,
    scope_metadata: scopeMetadata,
    semantic_model_ids: selectedKnowledgeIds,
    ...(fileIds.length > 0 ? { file_ids: fileIds } : {}),
    ...(tables.length > 0 ? { tables } : {}),
    ...(databases.length === 1 ? { database: databases[0] } : {}),
  };
}

function normalizeKnowledgeIds(ids: number[]): number[] {
  return Array.from(new Set(ids.filter((id) => Number.isInteger(id) && id > 0))).sort((left, right) => left - right);
}

function uniqueStrings(values: string[]): string[] {
  return Array.from(new Set(values.map((value) => value.trim()).filter(Boolean))).sort();
}

function normalizeQueryVisuals(images: ExploreA2AQueryVisual[]): ExploreA2AQueryVisual[] {
  const out: ExploreA2AQueryVisual[] = [];
  const seen = new Set<string>();
  for (const image of images) {
    const normalized = normalizeQueryVisual(image);
    if (!normalized || seen.has(normalized.fileId)) {
      continue;
    }
    seen.add(normalized.fileId);
    out.push(normalized);
  }
  return out;
}

function normalizeQueryVisual(image: ExploreA2AQueryVisual): ExploreA2AQueryVisual | null {
  const fileId = image.fileId.trim();
  if (!fileId) {
    return null;
  }
  return {
    fileId,
    fileName: image.fileName.trim() || fileId,
    mimeType: image.mimeType.trim() || 'application/octet-stream',
    size: Number.isFinite(image.size) && image.size > 0 ? image.size : 0,
  };
}

function modelScopedFileIDs(item: KnowledgeListItem): string[] {
  const volumeKeys = new Set<string>();
  item.files?.volumes?.forEach((volume) => {
    const volumeID = String(volume.volume_id ?? '').trim();
    if (volumeID) {
      volumeKeys.add(`volume-${volumeID}`);
    }
  });
  item.files?.volume_ids?.forEach((volumeID) => {
    const id = String(volumeID ?? '').trim();
    if (id) {
      volumeKeys.add(`volume-${id}`);
    }
  });
  const parents = (item.files?.parents ?? []).map((parent) => String(parent));
  if (parents.some((parent) => volumeKeys.has(parent))) {
    return [];
  }
  return item.files?.file_ids ?? [];
}

function normalizeBackendId(value: unknown): number | null {
  return Number.isInteger(value) && value !== 0 ? (value as number) : null;
}

function hasStructuredFailure(projection: ExploreA2AProjection): boolean {
  return hasAgentA2AStructuredFailure(projection);
}

function isCanceledProjection(projection: ExploreA2AProjection): boolean {
  return projection.status === 'canceled';
}

function canceledMessageText(response: AgentA2AResponse<AgentA2AResult>): string {
  const result = response.result;
  if (!result || !('status' in result) || result.status.state !== 'canceled') {
    console.warn('[canceledMessageText] canceled terminal payload missing', { kind: result?.kind });
    return '';
  }
  return (result.status.message?.parts ?? [])
    .map((part) => (part.kind === 'text' && typeof part.text === 'string' ? part.text : ''))
    .join('');
}

function buildMessagesFromRecords(records: MessageRecord[]): ExploreA2AMessageItem[] {
  const filtered = records.sort(
    (left, right) => numericTime(left.created_at) - numericTime(right.created_at) || left.id - right.id,
  );
  const messages: ExploreA2AMessageItem[] = [];
  let pendingTrace = '';
  let pendingKnowledgeIds: number[] = [];

  for (const record of filtered) {
    const role = normalizeMessageRole(record.role);
    if (role === 'user') {
      pendingTrace = record.modified_response || '';
      pendingKnowledgeIds = knowledgeIdsFromMessageConfig(record.config);
      messages.push({
        id: `stored_user_${record.id}`,
        messageId: record.id,
        role: 'user',
        content: record.content || record.original_content || '',
        createdAt: timestampMs(record.created_at, record.updated_at),
        knowledgeIds: pendingKnowledgeIds,
        queryVisuals: queryVisualsFromMessageConfig(record.config),
      });
      continue;
    }
    if (role !== 'assistant') {
      continue;
    }
    const rawTrace = record.modified_response || pendingTrace;
    const projection = projectionFromStoredA2AEvents(rawTrace, record);
    // Structured failures keep diagnostics out of chat text. Prefer empty content over
    // any stale visibleAnswer/content that older payloads may still carry.
    const content = hasStructuredFailure(projection)
      ? ''
      : record.content || record.response || projection.visibleAnswer || projection.error || '';
    messages.push({
      id: `stored_assistant_${record.id}`,
      messageId: record.id,
      role: 'assistant',
      content,
      createdAt: timestampMs(record.created_at, record.updated_at),
      knowledgeIds: knowledgeIdsFromMessageConfig(record.config).length
        ? knowledgeIdsFromMessageConfig(record.config)
        : pendingKnowledgeIds,
      projection,
      feedback: feedbackFromMessageTags(record.tags),
    });
    pendingTrace = '';
    pendingKnowledgeIds = [];
  }
  return messages;
}

function feedbackFromMessageTags(tags: MessageRecord['tags']): 'like' | 'dislike' | null {
  if (!Array.isArray(tags) || tags.length === 0) {
    return null;
  }
  const names = new Set(tags.map((tag) => tag.name));
  if (names.has('disliked')) {
    return 'dislike';
  }
  if (names.has('liked')) {
    return 'like';
  }
  return null;
}

function projectionFromStoredA2AEvents(rawTrace: string, record: MessageRecord): ExploreA2AProjection {
  let projection = createExploreA2AProjection();
  const events = parseStoredA2AEvents(rawTrace);
  for (const event of events) {
    projection = reduceExploreA2AResponse(projection, event.response, event.seq);
  }
  const status = record.status || '';
  if (status === 'failed') {
    return failExploreA2AProjection(
      projection,
      hasStructuredFailure(projection) ? '' : record.content || record.response || projection.error || 'Failed',
    );
  }
  if (status === 'canceled' || status === 'aborted') {
    const projectionAnswer = completeExploreA2AProjection(projection);
    const error = record.content || record.response || projection.error;
    return {
      ...projectionAnswer,
      status: 'canceled',
      final: true,
      error,
      visibleAnswer: projectionAnswer.visibleAnswer || error,
      visibleAnswerState: projectionAnswer.visibleAnswer ? projectionAnswer.visibleAnswerState : 'error',
    };
  }
  if (status === 'running' || status === 'pending') {
    const answer = projection.answer || projection.visibleAnswer || record.content || record.response || '';
    return {
      ...projection,
      status: projection.status || 'working',
      final: false,
      answer,
      visibleAnswer: projection.visibleAnswer || answer,
      visibleAnswerState: projection.visibleAnswer ? projection.visibleAnswerState : answer ? 'final' : 'empty',
    };
  }
  return completeExploreA2AProjection({
    ...projection,
    answer: projection.answer || projection.visibleAnswer || record.content || record.response || '',
  });
}

function parseStoredA2AEvents(rawTrace: string): Array<{ seq: number; response: AgentA2AResponse<AgentA2AResult> }> {
  const text = rawTrace.trim();
  if (!text) {
    return [];
  }
  try {
    const parsed: unknown = JSON.parse(text);
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .map((item, index) => {
        const record = item && typeof item === 'object' ? (item as Record<string, unknown>) : null;
        const response = record?.response;
        if (!isAgentA2AResponse(response)) {
          return null;
        }
        const rawSeq = Number(record?.seq);
        return { seq: Number.isFinite(rawSeq) && rawSeq > 0 ? rawSeq : index + 1, response };
      })
      .filter((item): item is { seq: number; response: AgentA2AResponse<AgentA2AResult> } => Boolean(item));
  } catch (error) {
    console.warn('[ExploreA2ARuntimeStore.parseStoredA2AEvents] parse failed', { msg: (error as Error).message });
    return [];
  }
}

function isAgentA2AResponse(value: unknown): value is AgentA2AResponse<AgentA2AResult> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return false;
  }
  const record = value as Record<string, unknown>;
  return record.jsonrpc === '2.0' && (Boolean(record.result) || Boolean(record.error));
}

function normalizeMessageRole(role: unknown): 'user' | 'assistant' | '' {
  const value = typeof role === 'string' ? role.trim().toLowerCase() : '';
  if (value === 'user') return 'user';
  if (value === 'assistant') return 'assistant';
  return '';
}

function knowledgeIdsFromMessageConfig(rawConfig: unknown): number[] {
  if (typeof rawConfig !== 'string' || !rawConfig.trim()) {
    return [];
  }
  try {
    const parsed = JSON.parse(rawConfig);
    const metadata = parsed && typeof parsed === 'object' ? (parsed as Record<string, unknown>).metadata : null;
    const ids = metadata && typeof metadata === 'object' ? (metadata as Record<string, unknown>).semantic_model_ids : null;
    return Array.isArray(ids) ? normalizeKnowledgeIds(ids.map((id) => Number(id))) : [];
  } catch {
    return [];
  }
}

function queryVisualsFromMessageConfig(rawConfig: unknown): ExploreA2AQueryVisual[] {
  if (typeof rawConfig !== 'string' || !rawConfig.trim()) {
    return [];
  }
  try {
    const parsed = JSON.parse(rawConfig);
    const metadata = parsed && typeof parsed === 'object' ? (parsed as Record<string, unknown>).metadata : null;
    const record = metadata && typeof metadata === 'object' ? (metadata as Record<string, unknown>) : null;
    const images = record?.query_visuals ?? record?.query_images ?? null;
    if (!Array.isArray(images)) {
      return [];
    }
    const out: ExploreA2AQueryVisual[] = [];
    for (const item of images) {
      if (!item || typeof item !== 'object' || Array.isArray(item)) {
        continue;
      }
      const record = item as Record<string, unknown>;
      if (typeof record.file_id !== 'string' || typeof record.file_name !== 'string' || typeof record.mime_type !== 'string') {
        continue;
      }
      out.push({
        fileId: record.file_id,
        fileName: record.file_name,
        mimeType: record.mime_type,
        size: typeof record.size === 'number' && Number.isFinite(record.size) ? record.size : 0,
      });
    }
    return normalizeQueryVisuals(out);
  } catch {
    return [];
  }
}

function latestAssistantProjection(messages: ExploreA2AMessageItem[]): ExploreA2AProjection | null {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const projection = messages[index]?.projection;
    if (projection) {
      return projection;
    }
  }
  return null;
}

function sameNumberList(left: number[], right: number[]): boolean {
  if (left.length !== right.length) return false;
  return left.every((value, index) => value === right[index]);
}

function timestampMs(createdAt: unknown, updatedAt: unknown): number {
  const value = numericTime(createdAt) || numericTime(updatedAt);
  if (value <= 0) {
    return Date.now();
  }
  return value < 10_000_000_000 ? value * 1000 : value;
}

function numericTime(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0;
}

function createTaskId(sessionId: number): string {
  const suffix = Math.random().toString(36).slice(2, 8);
  return `explore_${sessionId}_${Date.now()}_${suffix}`;
}

let exploreA2ARuntimeStoreInstance: ExploreA2ARuntimeStore | null = null;

export function getExploreA2ARuntimeStore(): ExploreA2ARuntimeStore {
  if (!exploreA2ARuntimeStoreInstance) {
    exploreA2ARuntimeStoreInstance = new ExploreA2ARuntimeStore();
  }
  return exploreA2ARuntimeStoreInstance;
}
