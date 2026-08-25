import { useCallback, useEffect, useMemo, useSyncExternalStore } from 'react';
import { message } from 'antd';
import type { TFunction } from 'i18next';

import type { AppHttpClient, AppSSEClient } from '@moi/shared-moi-app-protocol/app-context';
import type { AgentA2AInputRequestView, AgentA2AInputSubmitValues } from '@moi/shared-moi-components/ai-chat-message';
import { getSessionMessages } from '../../../service/dialogSession';
import type { KnowledgeListItem } from '../../../service/knowledge';
import {
  createDefaultA2ARuntimeState,
  getExploreA2ARuntimeStore,
  type ExploreA2AControlOptionKey,
  type ExploreA2AQueryVisual,
  type ExploreA2ARuntimeState,
} from '../services/exploreA2ARuntimeStore';

interface UseExploreA2ARuntimeManagerOptions {
  http: AppHttpClient;
  sseClient: AppSSEClient;
  selectedSessionId: number | null;
  workspaceId: string;
  t: TFunction<'moi-knowledge'>;
  knowledgeList: KnowledgeListItem[];
  appContext?: Record<string, unknown>;
  transformQuestion?: (question: string) => string;
  onBeforeSend?: (params: {
    sessionId: number;
    selectedKnowledgeIds: number[];
    selectedModel: string;
    selectedModelBackendId?: number | null;
    appContext?: Record<string, unknown>;
    selectedReasoningEffort?: string;
  }) => Promise<void>;
}

const runtimeStore = getExploreA2ARuntimeStore();

export function useExploreA2ARuntimeManager({
  http,
  sseClient,
  selectedSessionId,
  workspaceId,
  t,
  knowledgeList,
  appContext,
  transformQuestion,
  onBeforeSend,
}: UseExploreA2ARuntimeManagerOptions) {
  const runtimeStateMap = useSyncExternalStore(
    useCallback((listener: () => void) => runtimeStore.subscribe(listener), []),
    useCallback(() => runtimeStore.getRuntimeStateMap(), []),
    useCallback(() => runtimeStore.getRuntimeStateMap(), []),
  );

  const getRuntime = useCallback(
    (sessionId: number | null) => {
      if (!sessionId) {
        return createDefaultA2ARuntimeState();
      }
      return runtimeStateMap[sessionId] ?? createDefaultA2ARuntimeState();
    },
    [runtimeStateMap],
  );

  const currentRuntime = useMemo(() => getRuntime(selectedSessionId), [getRuntime, selectedSessionId]);

  useEffect(() => {
    if (!selectedSessionId) {
      return;
    }
    let disposed = false;
    const requestedAt = Date.now();
    getSessionMessages(http, selectedSessionId)
      .then((records) => {
        if (!disposed) {
          runtimeStore.hydrateSessionMessages(selectedSessionId, records, requestedAt);
        }
      })
      .catch((error) => {
        if (!disposed) {
          console.warn('[useExploreA2ARuntimeManager] load session messages failed', {
            sessionId: selectedSessionId,
            msg: (error as Error).message,
          });
          message.error(t('knowledge.explore.message-load-failed'));
        }
      });
    return () => {
      disposed = true;
    };
  }, [http, selectedSessionId, t]);

  const runningSessionCount = useMemo(
    () => Object.values(runtimeStateMap).filter((item) => item.isStreaming).length,
    [runtimeStateMap],
  );

  const syncSessionDefaultKnowledge = useCallback((sessionId: number | null, defaultKnowledgeIds: number[]) => {
    if (!sessionId) return;
    runtimeStore.syncSessionDefaultKnowledge(sessionId, defaultKnowledgeIds);
  }, []);

  const syncSessionDefaultModel = useCallback(
    (sessionId: number | null, defaultModel: string, defaultBackendId?: number | null) => {
      if (!sessionId) return;
      runtimeStore.syncSessionDefaultModel(sessionId, defaultModel, defaultBackendId);
    },
    [],
  );

  const syncSessionDefaultReasoningEffort = useCallback((sessionId: number | null, defaultReasoningEffort: string) => {
    if (!sessionId) return;
    runtimeStore.syncSessionDefaultReasoningEffort(sessionId, defaultReasoningEffort);
  }, []);

  const setCurrentDraftInput = useCallback(
    (value: string) => {
      if (!selectedSessionId) return;
      runtimeStore.setDraftInput(selectedSessionId, value);
    },
    [selectedSessionId],
  );

  const addCurrentQueryVisual = useCallback(
    (image: ExploreA2AQueryVisual) => {
      if (!selectedSessionId) return;
      runtimeStore.addQueryVisual(selectedSessionId, image);
    },
    [selectedSessionId],
  );

  const removeCurrentQueryVisual = useCallback(
    (fileId: string) => {
      if (!selectedSessionId) return;
      runtimeStore.removeQueryVisual(selectedSessionId, fileId);
    },
    [selectedSessionId],
  );

  const setCurrentModel = useCallback(
    (model: string, backendId?: number | null) => {
      if (!selectedSessionId) return;
      runtimeStore.setSelectedModel(selectedSessionId, model, backendId);
    },
    [selectedSessionId],
  );

  const toggleCurrentKnowledgeSelection = useCallback(
    (knowledgeId: number) => {
      if (!selectedSessionId) return;
      runtimeStore.toggleKnowledgeSelection(selectedSessionId, knowledgeId);
    },
    [selectedSessionId],
  );

  const removeCurrentKnowledgeSelection = useCallback(
    (knowledgeId: number) => {
      if (!selectedSessionId) return;
      runtimeStore.removeKnowledgeSelection(selectedSessionId, knowledgeId);
    },
    [selectedSessionId],
  );

  const setCurrentKnowledgeSelection = useCallback(
    (knowledgeIds: number[]) => {
      if (!selectedSessionId) return;
      runtimeStore.setKnowledgeSelection(selectedSessionId, knowledgeIds);
    },
    [selectedSessionId],
  );

  const setCurrentControlOption = useCallback(
    (key: ExploreA2AControlOptionKey, value: boolean) => {
      if (!selectedSessionId) return;
      runtimeStore.setControlOption(selectedSessionId, key, value);
    },
    [selectedSessionId],
  );

  const sendSessionMessage = useCallback(
    async (sessionId: number, current: ExploreA2ARuntimeState, overrideQuestion?: string) => {
      const originalQuestion = overrideQuestion !== undefined ? overrideQuestion.trim() : current.draftInput.trim();
      const question = transformQuestion ? transformQuestion(originalQuestion).trim() : originalQuestion;
      if (!question && current.queryVisuals.length === 0) {
        message.warning(t('knowledge.explore.composer-empty-warning'));
        return false;
      }
      if (!current.selectedModel.trim()) {
        message.warning(t('knowledge.explore.model-required'));
        return false;
      }
      if (current.selectedKnowledgeIds.length === 0) {
        message.warning(t('knowledge.explore.knowledge-required'));
        return false;
      }
      if (current.isStreaming || current.isSendPending) {
        message.warning(t('knowledge.explore.stream-running-warning'));
        return false;
      }

      const sent = await runtimeStore.sendSessionMessage({
        sessionId,
        question,
        sessionTitleQuestion: question === originalQuestion ? undefined : originalQuestion,
        queryVisuals: current.queryVisuals,
        model: current.selectedModel,
        modelBackendId: current.selectedModelBackendId,
        reasoningEffort: current.selectedReasoningEffort,
        selectedKnowledgeIds: current.selectedKnowledgeIds,
        workspaceId,
        sseClient,
        knowledgeList,
        appContext,
        streamErrorFallbackText: t('knowledge.explore.stream-error-generic'),
        beforeCreateStream: async () => {
          await onBeforeSend?.({
            sessionId,
            selectedKnowledgeIds: current.selectedKnowledgeIds,
            selectedModel: current.selectedModel,
            selectedModelBackendId: current.selectedModelBackendId,
            appContext,
            selectedReasoningEffort: current.selectedReasoningEffort,
          });
        },
      });
      if (!sent) {
        message.error(t('knowledge.explore.composer-send-failed'));
      }
      return sent;
    },
    [appContext, knowledgeList, onBeforeSend, sseClient, t, transformQuestion, workspaceId],
  );

  const sendCurrentSessionMessage = useCallback(
    async (overrideQuestion?: string) => {
      if (!selectedSessionId) {
        message.warning(t('knowledge.explore.chat-empty-session-warning'));
        return false;
      }
      return sendSessionMessage(selectedSessionId, getRuntime(selectedSessionId), overrideQuestion);
    },
    [getRuntime, selectedSessionId, sendSessionMessage, t],
  );

  const stopCurrentSessionStream = useCallback(async () => {
    if (!selectedSessionId) {
      return false;
    }
    const stopped = await runtimeStore.stopSessionStream(selectedSessionId, http);
    if (stopped) {
      message.success(t('knowledge.explore.stream-stop-success'));
    }
    return stopped;
  }, [http, selectedSessionId, t]);

  const submitCurrentSessionInput = useCallback(
    async (request: AgentA2AInputRequestView, values: AgentA2AInputSubmitValues) => {
      if (!selectedSessionId) {
        message.warning(t('knowledge.explore.chat-empty-session-warning'));
        return false;
      }
      await runtimeStore.submitSessionInput(selectedSessionId, request, values, {
        workspaceId,
        sseClient,
        knowledgeList,
        appContext,
        streamErrorFallbackText: t('knowledge.explore.stream-error-generic'),
      });
      return true;
    },
    [appContext, knowledgeList, selectedSessionId, sseClient, t, workspaceId],
  );

  return {
    runtimeStateMap,
    currentRuntime,
    runningSessionCount,
    syncSessionDefaultModel,
    syncSessionDefaultReasoningEffort,
    syncSessionDefaultKnowledge,
    setCurrentDraftInput,
    addCurrentQueryVisual,
    removeCurrentQueryVisual,
    setCurrentModel,
    toggleCurrentKnowledgeSelection,
    removeCurrentKnowledgeSelection,
    setCurrentKnowledgeSelection,
    setCurrentControlOption,
    sendSessionMessage,
    sendCurrentSessionMessage,
    stopCurrentSessionStream,
    submitCurrentSessionInput,
  };
}
