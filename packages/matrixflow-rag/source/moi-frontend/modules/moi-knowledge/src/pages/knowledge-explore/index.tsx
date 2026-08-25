import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { App, Button, Empty, Image, Popover, Space, Spin, Tag, Tooltip, Typography } from 'antd';
import {
  CopyOutlined,
  DislikeFilled,
  DislikeOutlined,
  FileTextOutlined,
  LikeFilled,
  LikeOutlined,
  LoadingOutlined,
  PictureOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import type { TFunction } from 'i18next';

import { previewConcreteAgentQueryVisualApi } from '@moi/shared-moi-api/agent';
import { uploadCatalogFileApi } from '@moi/shared-moi-api/catalog';
import { dislikeMessageApi, likeMessageApi, type MessageRecord } from '@moi/shared-moi-api/explore';
import { useHttpClient, useSSEClient, useTimezone, type AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { useWorkspaceId } from '@moi/shared-moi-app-protocol/business-context';
import type {
  AgentA2AInputRequestView,
  AgentA2AInputSubmitValues,
  AgentA2ATraceViewMode,
} from '@moi/shared-moi-components/ai-chat-message';
import { AgentChatPage, type AgentChatWorkspaceMessage } from '@moi/shared-moi-components/ai-chat-message/agent-chat-page';
import { AgentChatTraceViewToggle } from '@moi/shared-moi-components/ai-chat-message/agent-chat-trace-view-toggle';
import {
  AiChatSessionManager,
  type SessionRecord as ChatSessionRecord,
} from '@moi/shared-moi-components/ai-chat-session-manager';
import { FilePreviewModal } from '@moi/shared-moi-components/file-preview';
import { formatDateTime } from '@moi/shared-moi-utils/datetime';
import { getSessionMessages, isSessionPinned, parseSessionConfig, type SessionRecord } from '../../service/dialogSession';
import { unwrapApiResponse } from '../../service/http';
import {
  answerInlineVisualPreviews,
  answerSourceDisplayLabel,
  answerSourceIdentity,
  answerSourcePages,
  answerSourcePreviews,
  requestAnswerSourcePreviewBlob,
  type AnswerSourcePreview,
} from './answerSourcePreview';
import { mapExploreAssistantChatMessage } from './chatMessageMapping';
import { ChatComposer } from './components/ChatComposer';
import KnowledgeSummary from './components/KnowledgeSummary';
import { useExploreA2ARuntimeManager } from './hooks/useExploreA2ARuntimeManager';
import { useExploreKnowledgeOptions } from './hooks/useExploreKnowledgeOptions';
import {
  decodeModelValue,
  encodeModelValue,
  formatExploreModelLabel,
  isSystemDefaultModel,
  useExploreModelOptions,
} from './hooks/useExploreModelOptions';
import { useExploreSessionManager } from './hooks/useExploreSessionManager';
import { useSessionConfigPersistence } from './hooks/useSessionConfigPersistence';
import styles from './KnowledgeExplore.module.css';
import { getCachedPreviewBlob } from './previewBlobCache';
import {
  createDefaultA2ARuntimeState,
  type ExploreA2AMessageItem,
  type ExploreA2ARuntimeState,
} from './services/exploreA2ARuntimeStore';
import type { ExploreA2AAnswerSourceRef } from './services/exploreA2AStreamParser';
import {
  normalizeSemanticModelRefs,
  normalizeSessionModel,
  normalizeSessionModelBackendId,
  normalizeSessionReasoningEffort,
} from './utils/sessionConfig';

type KnowledgeT = TFunction<'moi-knowledge'>;
type AnswerFeedbackValue = 'like' | 'dislike' | null;
type AnswerFeedbackResult = { messageId: number; feedback: AnswerFeedbackValue };
const PENDING_LAZY_SESSION_ID = -1;
const EXPLORE_QUERY_VISUAL_AGENT_ID = 'explore';

export interface KnowledgeExplorePageProps {
  defaultKnowledgeId?: number | null;
  defaultKnowledgeIds?: number[];
  createRequest?: { knowledgeId: number; requestId: number } | null;
  lockedKnowledgeIds?: number[];
  lockedModelValue?: string;
  appContext?: Record<string, unknown>;
  transformQuestion?: (question: string) => string;
  features?: {
    selectedKnowledgeHeader?: boolean;
    knowledgeSelector?: boolean;
    imageUpload?: boolean;
  };
  quickActions?: Array<{
    key: string;
    label: string;
    question: string;
    disabled?: boolean;
  }>;
  /** Only the knowledge-board conversation enables the compact Trace view switch. */
  enableDeveloperView?: boolean;
  embedded?: boolean;
}

export function KnowledgeExplorePage({
  defaultKnowledgeId = null,
  defaultKnowledgeIds,
  createRequest = null,
  lockedKnowledgeIds,
  lockedModelValue,
  appContext,
  transformQuestion,
  features,
  quickActions,
  enableDeveloperView = false,
  embedded = false,
}: KnowledgeExplorePageProps) {
  const { t, i18n } = useTranslation('moi-knowledge');
  const { message: appMessage } = App.useApp();
  const http = useHttpClient();
  const sseClient = useSSEClient();
  const timezone = useTimezone();
  const workspaceId = useWorkspaceId();
  const sessionLocale = i18n.language === 'en-US' ? 'en-US' : ('zh-CN' as const);

  const {
    pinnedSessionList,
    groupedSessions,
    selectedSessionId,
    selectedSession,
    searchKeyword,
    isLoading,
    isPinnedLoading,
    isSearching,
    isActionLoading,
    isSessionCreating,
    isLoadingMore,
    hasMore,
    setSelectedSessionId,
    releaseEmptySelection,
    handleSearch,
    loadMore,
    createNewSession,
    createFixedSession,
    renameSession,
    syncSessionTitle,
    deleteSession,
    togglePin,
  } = useExploreSessionManager({ http, locale: i18n.language, t, workspaceId });
  const handledCreateRequestRef = useRef<number | null>(null);
  const lockedKnowledgeSelection = useMemo(() => normalizePositiveIds(lockedKnowledgeIds ?? []), [lockedKnowledgeIds]);
  const configuredDefaultKnowledgeIds = useMemo(() => {
    if (lockedKnowledgeSelection.length > 0) return lockedKnowledgeSelection;
    if (defaultKnowledgeIds?.length) return normalizePositiveIds(defaultKnowledgeIds);
    return normalizePositiveIds(defaultKnowledgeId ? [defaultKnowledgeId] : []);
  }, [defaultKnowledgeId, defaultKnowledgeIds, lockedKnowledgeSelection]);
  const activeDefaultKnowledgeIds = useMemo(
    () => (createRequest?.knowledgeId ? [createRequest.knowledgeId] : configuredDefaultKnowledgeIds),
    [configuredDefaultKnowledgeIds, createRequest?.knowledgeId],
  );
  const lockedModel = useMemo(() => (lockedModelValue ? decodeModelValue(lockedModelValue) : null), [lockedModelValue]);
  const [pendingSessionRuntime, setPendingSessionRuntime] = useState<ExploreA2ARuntimeState>(() =>
    createDefaultA2ARuntimeState(),
  );
  const [pendingSessionCreatedAt, setPendingSessionCreatedAt] = useState<number>(() => Math.floor(Date.now() / 1000));
  const [pendingLazyKnowledgeId, setPendingLazyKnowledgeId] = useState<number | null>(null);

  const { knowledgeList, isLoading: isKnowledgeLoading } = useExploreKnowledgeOptions({ http, t });
  const { modelList, isLoading: isModelLoading } = useExploreModelOptions({ http, t });
  const persistSessionConfig = useSessionConfigPersistence(http);
  const {
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
    sendSessionMessage,
    sendCurrentSessionMessage,
    stopCurrentSessionStream,
    submitCurrentSessionInput,
  } = useExploreA2ARuntimeManager({
    http,
    sseClient,
    selectedSessionId,
    workspaceId,
    t,
    knowledgeList,
    appContext,
    transformQuestion,
    onBeforeSend: persistSessionConfig,
  });
  const [inputSubmitting, setInputSubmitting] = useState(false);
  const [developerView, setDeveloperView] = useState(false);
  const traceViewMode: AgentA2ATraceViewMode | undefined = enableDeveloperView
    ? developerView
      ? 'developer'
      : 'summary'
    : undefined;
  const pendingLazyKnowledgeIds = useMemo(
    () => (pendingLazyKnowledgeId !== null ? [pendingLazyKnowledgeId] : []),
    [pendingLazyKnowledgeId],
  );
  const hasPendingLazySession = !selectedSessionId && pendingLazyKnowledgeIds.length > 0;
  const visibleRuntime = useMemo<ExploreA2ARuntimeState>(() => {
    if (selectedSessionId) {
      return currentRuntime;
    }
    if (!hasPendingLazySession) {
      return pendingSessionRuntime;
    }
    return {
      ...pendingSessionRuntime,
      selectedKnowledgeIds: pendingLazyKnowledgeIds,
    };
  }, [currentRuntime, hasPendingLazySession, pendingLazyKnowledgeIds, pendingSessionRuntime, selectedSessionId]);
  const pendingSessionRecord = useMemo<SessionRecord | null>(() => {
    if (!hasPendingLazySession) {
      return null;
    }
    return {
      id: PENDING_LAZY_SESSION_ID,
      title: t('knowledge.explore.default-session-title'),
      source: 'draft',
      config: JSON.stringify({
        type: 'fixed',
        semantic_models: pendingLazyKnowledgeIds.map((id) => ({ semantic_model_id: id })),
      }),
      created_at: pendingSessionCreatedAt,
      updated_at: pendingSessionCreatedAt,
    };
  }, [hasPendingLazySession, pendingLazyKnowledgeIds, pendingSessionCreatedAt, t]);
  const displayGroupedSessions = useMemo(() => {
    if (!pendingSessionRecord) {
      return groupedSessions;
    }
    const todayLabel = t('knowledge.explore.group-today');
    const [firstGroup, ...restGroups] = groupedSessions;
    if (firstGroup?.label === todayLabel) {
      return [{ ...firstGroup, sessions: [pendingSessionRecord, ...firstGroup.sessions] }, ...restGroups];
    }
    return [
      {
        key: 'pending-lazy-session',
        label: todayLabel,
        timestamp: pendingSessionCreatedAt * 1000,
        sessions: [pendingSessionRecord],
      },
      ...groupedSessions,
    ];
  }, [groupedSessions, pendingSessionCreatedAt, pendingSessionRecord, t]);
  const displayCurrentSessionId = selectedSessionId ?? (hasPendingLazySession ? PENDING_LAZY_SESSION_ID : null);

  useEffect(() => {
    if (!selectedSessionId) return;
    const cfg = parseSessionConfig(selectedSession?.config);
    const refs = lockedKnowledgeSelection.length
      ? lockedKnowledgeSelection
      : normalizeSemanticModelRefs(cfg.semantic_models).map((ref) => ref.semantic_model_id);
    syncSessionDefaultKnowledge(selectedSessionId, refs);
    if (lockedModel) {
      syncSessionDefaultModel(selectedSessionId, lockedModel.model, lockedModel.backendId);
    } else {
      syncSessionDefaultModel(selectedSessionId, normalizeSessionModel(cfg), normalizeSessionModelBackendId(cfg));
    }
    syncSessionDefaultReasoningEffort(selectedSessionId, normalizeSessionReasoningEffort(cfg));
  }, [
    lockedKnowledgeSelection,
    lockedModel,
    selectedSession?.config,
    selectedSessionId,
    syncSessionDefaultKnowledge,
    syncSessionDefaultModel,
    syncSessionDefaultReasoningEffort,
  ]);

  useEffect(() => {
    if (!selectedSessionId || lockedKnowledgeSelection.length === 0) return;
    setCurrentKnowledgeSelection(lockedKnowledgeSelection);
  }, [lockedKnowledgeSelection, selectedSessionId, setCurrentKnowledgeSelection]);

  useEffect(() => {
    if (!selectedSessionId || !lockedModel) return;
    setCurrentModel(lockedModel.model, lockedModel.backendId);
  }, [lockedModel, selectedSessionId, setCurrentModel]);

  useEffect(() => {
    if (!selectedSessionId || !currentRuntime.projection.sessionTitle) return;
    syncSessionTitle(selectedSessionId, currentRuntime.projection.sessionTitle);
  }, [currentRuntime.projection.sessionTitle, selectedSessionId, syncSessionTitle]);

  useEffect(() => {
    if (!createRequest) return;
    if (handledCreateRequestRef.current === createRequest.requestId) return;
    handledCreateRequestRef.current = createRequest.requestId;
    if (!Number.isInteger(createRequest.knowledgeId) || createRequest.knowledgeId <= 0) {
      console.warn('[KnowledgeExplorePage] ignore invalid lazy session request', { knowledgeId: createRequest.knowledgeId });
      return;
    }
    setSelectedSessionId(null);
    setPendingSessionCreatedAt(Math.floor(Date.now() / 1000));
    setPendingLazyKnowledgeId(createRequest.knowledgeId);
    setPendingSessionRuntime((previous) => {
      const next = createDefaultA2ARuntimeState();
      next.selectedKnowledgeIds = [createRequest.knowledgeId];
      next.selectedModel = previous.selectedModel;
      next.selectedModelBackendId = previous.selectedModelBackendId;
      next.updatedAt = Date.now();
      return next;
    });
    return () => {
      releaseEmptySelection();
      handledCreateRequestRef.current = null;
    };
  }, [createRequest, releaseEmptySelection, setSelectedSessionId]);

  const modelOptions = useMemo(
    () =>
      modelList.map((model) => ({
        value: encodeModelValue(model.model, model.backend_id),
        label: formatExploreModelLabel(model, t('knowledge.explore.model-system-default')),
        backendId: model.backend_id,
      })),
    [modelList, t],
  );

  const modelSelectOptions = useMemo(() => {
    const workspace = modelList.filter((m) => !isSystemDefaultModel(m));
    const system = modelList.filter((m) => isSystemDefaultModel(m));
    const toOption = (m: (typeof modelList)[number]) => ({
      value: encodeModelValue(m.model, m.backend_id),
      label: formatExploreModelLabel(m, t('knowledge.explore.model-system-default')),
      backendId: m.backend_id,
    });
    return [
      ...(workspace.length
        ? [
            {
              label: t('knowledge.explore.model-group-workspace', { defaultValue: 'Workspace' }),
              options: workspace.map(toOption),
            },
          ]
        : []),
      ...(system.length
        ? [{ label: t('knowledge.explore.model-group-system', { defaultValue: 'System' }), options: system.map(toOption) }]
        : []),
    ];
  }, [modelList, t]);

  const selectedModelCompositeValue = useMemo(() => {
    if (lockedModelValue) return lockedModelValue;
    const model = visibleRuntime.selectedModel.trim();
    if (!model) return undefined;
    const backendId = visibleRuntime.selectedModelBackendId;
    if (backendId !== null && backendId !== undefined && backendId !== 0) {
      return encodeModelValue(model, backendId);
    }
    const match = modelOptions.find((opt) => decodeModelValue(opt.value).model === model);
    return match?.value;
  }, [lockedModelValue, modelOptions, visibleRuntime.selectedModel, visibleRuntime.selectedModelBackendId]);

  const composerModelOptions = useMemo(() => {
    if (!lockedModelValue) return modelOptions;
    const matched = modelOptions.find((option) => option.value === lockedModelValue);
    if (matched) return [matched];
    return [
      {
        value: lockedModelValue,
        label: lockedModel ? formatFixedModelLabel(lockedModel.model) : lockedModelValue,
        backendId: lockedModel?.backendId ?? 0,
      },
    ];
  }, [lockedModel, lockedModelValue, modelOptions]);

  useEffect(() => {
    if (modelOptions.length === 0) return;
    if (lockedModel) {
      if (selectedSessionId) {
        setCurrentModel(lockedModel.model, lockedModel.backendId);
      } else if (hasPendingLazySession) {
        setPendingSessionRuntime((previous) => ({
          ...previous,
          selectedModel: lockedModel.model,
          selectedModelBackendId: lockedModel.backendId ?? null,
          updatedAt: Date.now(),
        }));
      }
      return;
    }
    const configuredModel = normalizeSessionModel(parseSessionConfig(selectedSession?.config));
    const selectedModel = visibleRuntime.selectedModel.trim();
    if (!selectedModel && configuredModel) return;
    const selectedOption = modelOptions.find((option) => option.value === selectedModelCompositeValue);
    if (selectedOption) {
      if (visibleRuntime.selectedModelBackendId !== selectedOption.backendId) {
        const decoded = decodeModelValue(selectedOption.value);
        if (selectedSessionId) {
          setCurrentModel(decoded.model, selectedOption.backendId);
        } else if (hasPendingLazySession) {
          setPendingSessionRuntime((previous) => ({
            ...previous,
            selectedModel: decoded.model,
            selectedModelBackendId: selectedOption.backendId ?? null,
            updatedAt: Date.now(),
          }));
        }
      }
      return;
    }
    if (selectedModel) {
      console.warn('[KnowledgeExplorePage] selected model unavailable', { sessionId: selectedSessionId ?? 0, selectedModel });
    }
    const firstOption = modelOptions[0];
    if (firstOption) {
      const decoded = decodeModelValue(firstOption.value);
      if (selectedSessionId) {
        setCurrentModel(decoded.model, firstOption.backendId ?? null);
      } else if (hasPendingLazySession) {
        setPendingSessionRuntime((previous) => ({
          ...previous,
          selectedModel: decoded.model,
          selectedModelBackendId: firstOption.backendId ?? null,
          updatedAt: Date.now(),
        }));
      }
    } else if (selectedSessionId) {
      setCurrentModel('', null);
    }
  }, [
    hasPendingLazySession,
    lockedModel,
    selectedModelCompositeValue,
    modelOptions,
    selectedSession?.config,
    selectedSessionId,
    setCurrentModel,
    visibleRuntime.selectedModel,
    visibleRuntime.selectedModelBackendId,
  ]);

  const handleSessionPin = useCallback(
    async (session: ChatSessionRecord) => {
      await togglePin(session as SessionRecord);
    },
    [togglePin],
  );

  const handleSessionRename = useCallback(
    async (session: ChatSessionRecord, nextTitle: string) => {
      await renameSession(session.id, nextTitle);
    },
    [renameSession],
  );

  const handleSessionDelete = useCallback(
    async (session: ChatSessionRecord) => {
      await deleteSession(session.id);
    },
    [deleteSession],
  );

  const handleIsSessionPinned = useCallback((session: ChatSessionRecord) => isSessionPinned(session as SessionRecord), []);

  const handleSend = useCallback(
    async (question?: string) => {
      if (selectedSessionId) {
        await sendCurrentSessionMessage(question);
        return;
      }
      if (!hasPendingLazySession) {
        appMessage.warning(t('knowledge.explore.chat-empty-session-warning'));
        return;
      }
      const draftQuestion = question !== undefined ? question.trim() : visibleRuntime.draftInput.trim();
      const pendingQuestion = transformQuestion ? transformQuestion(draftQuestion).trim() : draftQuestion;
      if (!pendingQuestion && visibleRuntime.queryVisuals.length === 0) {
        appMessage.warning(t('knowledge.explore.composer-empty-warning'));
        return;
      }
      if (!visibleRuntime.selectedModel.trim()) {
        appMessage.warning(t('knowledge.explore.model-required'));
        return;
      }
      if (pendingLazyKnowledgeIds.length === 0) {
        appMessage.warning(t('knowledge.explore.knowledge-required'));
        return;
      }
      if (visibleRuntime.isStreaming || visibleRuntime.isSendPending) {
        appMessage.warning(t('knowledge.explore.stream-running-warning'));
        return;
      }
      const pendingLazyRequestId = handledCreateRequestRef.current;
      const targetKnowledgeId = pendingLazyKnowledgeIds[0];
      const createdSession = await createFixedSession(targetKnowledgeId);
      if (!createdSession) {
        return;
      }
      const sent = await sendSessionMessage(createdSession.id, visibleRuntime, question);
      if (sent && pendingLazyRequestId !== null && handledCreateRequestRef.current === pendingLazyRequestId) {
        setPendingSessionRuntime(createDefaultA2ARuntimeState());
        setPendingSessionCreatedAt(Math.floor(Date.now() / 1000));
        setPendingLazyKnowledgeId(null);
      }
    },
    [
      appMessage,
      createFixedSession,
      hasPendingLazySession,
      pendingLazyKnowledgeIds,
      selectedSessionId,
      sendCurrentSessionMessage,
      sendSessionMessage,
      t,
      transformQuestion,
      visibleRuntime,
    ],
  );

  const composerQuickActions = useMemo(
    () =>
      quickActions?.map((action) => ({
        key: action.key,
        label: action.label,
        disabled: action.disabled,
        onClick: () => {
          handleSend(action.question).catch(() => {});
        },
      })) ?? [],
    [handleSend, quickActions],
  );

  const handleUploadQueryVisual = useCallback(
    async (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      const response = await uploadCatalogFileApi(formData, http);
      const data = unwrapApiResponse(response, 'uploadExploreQueryVisual');
      addCurrentQueryVisual({
        fileId: data.file_id,
        fileName: file.name || data.file_id,
        mimeType: file.type || 'application/octet-stream',
        size: file.size,
      });
    },
    [addCurrentQueryVisual, http],
  );

  const handleStop = useCallback(() => {
    stopCurrentSessionStream()
      .then((stopped) => {
        if (!stopped) {
          appMessage.warning(t('knowledge.explore.stream-stop-noop'));
        }
      })
      .catch((error) => {
        console.warn('[KnowledgeExplorePage] stop stream failed', { msg: (error as Error).message });
        appMessage.error(t('knowledge.explore.stream-error-generic'));
      });
  }, [appMessage, stopCurrentSessionStream, t]);

  const handleSubmitInput = useCallback(
    async (request: AgentA2AInputRequestView, values: AgentA2AInputSubmitValues) => {
      setInputSubmitting(true);
      try {
        await submitCurrentSessionInput(request, values);
      } catch (error) {
        console.warn('[KnowledgeExplorePage] submit input failed', { msg: (error as Error).message });
        appMessage.error(error instanceof Error ? error.message : t('knowledge.explore.stream-error-generic'));
      } finally {
        setInputSubmitting(false);
      }
    },
    [appMessage, submitCurrentSessionInput, t],
  );

  const handleAssistantFeedback = useCallback(
    async (
      messageId: number | undefined,
      taskId: string | null | undefined,
      feedback: 'like' | 'dislike',
    ): Promise<AnswerFeedbackResult> => {
      const targetMessageId =
        messageId ?? (selectedSessionId && taskId ? await resolveAssistantMessageId(http, selectedSessionId, taskId) : undefined);
      if (!targetMessageId) {
        throw new Error('Explore assistant message id is unavailable');
      }
      const response =
        feedback === 'like' ? await likeMessageApi(targetMessageId, http) : await dislikeMessageApi(targetMessageId, http);
      const updated = unwrapApiResponse(response, feedback === 'like' ? 'likeExploreMessage' : 'dislikeExploreMessage');
      return { messageId: updated.id || targetMessageId, feedback: feedbackFromMessageTags(updated.tags) };
    },
    [http, selectedSessionId],
  );

  const chatMessages = useMemo<AgentChatWorkspaceMessage<ExploreA2AMessageItem>[]>(
    () =>
      visibleRuntime.messages.map((message) => {
        const isUser = message.role === 'user';
        const knowledgeItems = message.knowledgeIds.map((id) => {
          const item = knowledgeList.find((knowledge) => knowledge.id === id);
          return { id, name: item?.name || t('knowledge.explore.knowledge-fallback-name', { id }) };
        });

        if (isUser) {
          const userExtras = [
            message.queryVisuals?.length ? <UserQueryVisuals key="images" images={message.queryVisuals} /> : null,
            knowledgeItems.length > 0 ? (
              <div key="knowledge" className={styles.userKnowledgeSummary}>
                <span className={styles.userKnowledgeSummaryLabel}>{t('knowledge.explore.message-knowledge-label')}</span>
                <KnowledgeSummary items={knowledgeItems} />
              </div>
            ) : null,
          ].filter(Boolean);
          return {
            id: message.id,
            role: 'user',
            content: message.content,
            time: formatDateTime(message.createdAt, { timezone }),
            testId: `knowledge-explore-message-${message.id}`,
            data: message,
            extra: userExtras.length ? <>{userExtras}</> : null,
          };
        }

        return mapExploreAssistantChatMessage(message, timezone);
      }),
    [knowledgeList, t, timezone, visibleRuntime.messages],
  );

  return (
    <div className={`${styles.wrapper} ${embedded ? styles.embeddedWrapper : ''}`} data-testid="knowledge-explore-tab-content">
      <div className={styles.sidebar} data-testid="knowledge-explore-sidebar">
        <div className={styles.sidebarHeader}>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              createNewSession(activeDefaultKnowledgeIds.length ? activeDefaultKnowledgeIds : undefined).catch(() => {});
            }}
            loading={isActionLoading}
            className={styles.createButton}
            data-testid="knowledge-explore-create-session-btn"
          >
            {t('knowledge.explore.create-session-btn')}
          </Button>
        </div>
        <AiChatSessionManager
          pinnedSessions={pinnedSessionList}
          groupedSessions={displayGroupedSessions}
          currentSessionId={displayCurrentSessionId}
          searchKeyword={searchKeyword}
          searchDisabled={isSessionCreating}
          isLoading={isLoading}
          isPinnedLoading={isPinnedLoading}
          isSearching={isSearching}
          isLoadingMore={isLoadingMore}
          hasMore={hasMore}
          onSessionClick={(sessionId) => {
            if (isSessionCreating || sessionId === PENDING_LAZY_SESSION_ID) {
              return;
            }
            setSelectedSessionId(sessionId);
          }}
          onSearch={handleSearch}
          onLoadMore={() => {
            loadMore().catch(() => {});
          }}
          locale={sessionLocale}
          isSessionPinned={handleIsSessionPinned}
          onSessionPin={handleSessionPin}
          onSessionUnpin={handleSessionPin}
          onSessionRename={handleSessionRename}
          onSessionDelete={handleSessionDelete}
          sessionActionsLoading={isActionLoading}
          testIdPrefix="knowledge-explore-session-manager"
        />
      </div>

      <AgentChatPage
        className={styles.chatPanel}
        headerClassName={enableDeveloperView ? styles.developerViewHeader : undefined}
        title={
          <Typography.Text ellipsis className={styles.chatHeaderValue}>
            {selectedSession?.title || pendingSessionRecord?.title || t('knowledge.explore.chat-empty-session')}
          </Typography.Text>
        }
        headerRight={
          <Space size={8} className={styles.chatHeaderRight}>
            {enableDeveloperView ? (
              <AgentChatTraceViewToggle
                checked={developerView}
                onChange={setDeveloperView}
                label={t('knowledge.explore.developer-view-label')}
                ariaLabel={t('knowledge.explore.developer-view-aria')}
                testId="knowledge-explore-developer-view-switch"
              />
            ) : null}
            {runningSessionCount > 0 ? (
              <Tag color="processing" icon={<LoadingOutlined />} className={styles.statusTag}>
                {t('knowledge.explore.running-session-count', { count: runningSessionCount })}
              </Tag>
            ) : null}
          </Space>
        }
        messages={chatMessages}
        locale={sessionLocale}
        traceViewMode={traceViewMode}
        labels={{
          planTitle: t('knowledge.explore.a2a-plan-title'),
          planStatus: (status) => renderPlanStatus(status, t),
          toolCallTitle: t('knowledge.explore.trace-tool-call-title'),
          toolArguments: t('knowledge.explore.trace-tool-arguments'),
          toolResult: t('knowledge.explore.trace-tool-result'),
          toolStatus: (status) => renderToolStatus(status, t),
          progressText: (key, params) => t(key, { ...params, defaultValue: key }),
          activityText: (key, params) => renderExploreActivityText(key, params, t),
        }}
        renderAnswerFooter={(message, answer) => {
          const item = message.data;
          if (!item) return null;
          if (!item.projection?.final) return null;
          // Grounding evidence must remain page-visible whenever a final projection
          // carries answerSources, even if the answer body is empty/repaired later.
          return (
            <>
              <AnswerInlineVisualRefs sources={item.projection?.answerSources ?? []} t={t} />
              <AnswerSourceRefs sources={item.projection?.answerSources ?? []} t={t} />
              <AnswerFeedback
                content={answer}
                messageId={item.messageId}
                taskId={item.projection?.taskId ?? null}
                initialFeedback={item.feedback ?? null}
                onFeedback={handleAssistantFeedback}
              />
            </>
          );
        }}
        onSubmitInput={(request, values) => handleSubmitInput(request, values)}
        inputSubmitting={inputSubmitting}
        showPlaceholder={!selectedSessionId && !hasPendingLazySession}
        placeholder={
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={
              <div className={styles.placeholderTextWrap}>
                <div className={styles.placeholderTitle}>{t('knowledge.explore.chat-placeholder-title')}</div>
              </div>
            }
          />
        }
        empty={<Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('knowledge.explore.chat-welcome-text')} />}
        composer={
          <ChatComposer
            currentRuntime={visibleRuntime}
            selectedSessionId={selectedSessionId}
            canCreateSessionOnSend={hasPendingLazySession}
            isSessionCreating={isSessionCreating}
            selectedModelValue={selectedModelCompositeValue}
            t={t}
            knowledgeList={knowledgeList}
            isKnowledgeLoading={isKnowledgeLoading}
            modelOptions={composerModelOptions}
            modelSelectOptions={lockedModelValue ? undefined : modelSelectOptions}
            isModelLoading={isModelLoading}
            isModelLocked={Boolean(lockedModelValue)}
            isKnowledgeLocked={lockedKnowledgeSelection.length > 0 || hasPendingLazySession}
            features={features}
            quickActions={composerQuickActions}
            onDraftChange={(value) => {
              if (selectedSessionId) {
                setCurrentDraftInput(value);
                return;
              }
              setPendingSessionRuntime((previous) => ({ ...previous, draftInput: value, updatedAt: Date.now() }));
            }}
            onModelChange={(compositeValue, backendId) => {
              const decoded = decodeModelValue(compositeValue);
              const nextBackendId = backendId ?? decoded.backendId;
              if (selectedSessionId) {
                setCurrentModel(decoded.model, nextBackendId);
                return;
              }
              setPendingSessionRuntime((previous) => ({
                ...previous,
                selectedModel: decoded.model,
                selectedModelBackendId: nextBackendId ?? null,
                updatedAt: Date.now(),
              }));
            }}
            onToggleKnowledge={(knowledgeId) => {
              if (selectedSessionId) {
                toggleCurrentKnowledgeSelection(knowledgeId);
              }
            }}
            onRemoveKnowledge={(knowledgeId) => {
              if (selectedSessionId) {
                removeCurrentKnowledgeSelection(knowledgeId);
              }
            }}
            onUploadImage={handleUploadQueryVisual}
            onRemoveQueryVisual={removeCurrentQueryVisual}
            onSend={() => {
              handleSend().catch(() => {});
            }}
            onStop={handleStop}
          />
        }
        autoScrollKey={visibleRuntime.updatedAt}
        testIdPrefix="knowledge-explore"
        chatPanelTestId="knowledge-explore-chat-panel"
        messageListTestId="knowledge-explore-message-list"
        assistantTestIdPrefix="knowledge-explore"
      />
    </div>
  );
}

function UserQueryVisuals({ images }: { images: NonNullable<ExploreA2AMessageItem['queryVisuals']> }) {
  const http = useHttpClient();
  const workspaceId = useWorkspaceId();
  const { t } = useTranslation('moi-knowledge');
  const [preview, setPreview] = useState<(typeof images)[number] | null>(null);

  return (
    <>
      <div className={styles.userQueryVisualList} data-testid="knowledge-explore-message-query-visuals">
        {images.map((image) => (
          <UserQueryVisualPreview
            key={image.fileId}
            image={image}
            http={http}
            workspaceId={workspaceId}
            unavailableLabel={t('knowledge.explore.source-preview-image-unavailable')}
            onOpen={() => setPreview(image)}
          />
        ))}
      </div>
      {preview ? (
        <FilePreviewModal
          open
          onClose={() => setPreview(null)}
          fileName={preview.fileName || preview.fileId}
          fileExt={fileExtFromName(preview.fileName || preview.fileId)}
          unsupportedText=""
          markdownPreviewLabel=""
          fetchBlob={async () => {
            return fetchQueryVisualPreviewBlob(
              http,
              workspaceId,
              preview.fileId,
              t('knowledge.explore.source-preview-image-unavailable'),
            );
          }}
        />
      ) : null}
    </>
  );
}

function UserQueryVisualPreview({
  image,
  http,
  workspaceId,
  unavailableLabel,
  onOpen,
}: {
  image: NonNullable<ExploreA2AMessageItem['queryVisuals']>[number];
  http: AppHttpClient;
  workspaceId: string;
  unavailableLabel: string;
  onOpen: () => void;
}) {
  const previewState = useQueryVisualPreviewObjectURL(http, workspaceId, image.fileId, unavailableLabel);
  const label = image.fileName || image.fileId;
  const title = previewState.error ? `${label} - ${previewState.error}` : label;

  return (
    <button type="button" className={styles.userQueryVisualPreview} title={title} onClick={onOpen}>
      {previewState.src ? (
        <img className={styles.userQueryVisualPreviewImage} src={previewState.src} alt={label} loading="lazy" />
      ) : (
        <span className={styles.userQueryVisualPreviewFallback} data-unavailable={previewState.error ? 'true' : undefined}>
          <PictureOutlined />
          {previewState.error ? <span>{unavailableLabel}</span> : null}
        </span>
      )}
      {label ? <span className={styles.userQueryVisualPreviewName}>{label}</span> : null}
    </button>
  );
}

export function AnswerInlineVisualRefs({ sources, t }: { sources: ExploreA2AAnswerSourceRef[]; t: KnowledgeT }) {
  const http = useHttpClient();
  const workspaceId = useWorkspaceId();
  const previews = answerInlineVisualPreviews(sources);

  if (!previews.length) {
    return null;
  }
  return (
    <div className={styles.answerInlineVisualGrid} data-testid="knowledge-explore-answer-inline-visuals">
      {previews.map((preview, index) => (
        <div key={`${preview.fileId}-${index}`} className={styles.answerInlineVisualItem} title={answerVisualTitle(preview)}>
          <AnswerInlineVisualImage
            preview={preview}
            http={http}
            cacheScope={workspaceId}
            unsupportedText={t('knowledge.explore.source-preview-unsupported')}
          />
        </div>
      ))}
    </div>
  );
}

function AnswerInlineVisualImage({
  preview,
  http,
  cacheScope,
  unsupportedText,
}: {
  preview: AnswerSourcePreview;
  http: AppHttpClient;
  cacheScope: string;
  unsupportedText: string;
}) {
  const [blobUrl, setBlobUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    let nextBlobUrl: string | null = null;
    setLoading(true);
    setError(null);
    setBlobUrl(null);
    if (!isAnswerPreviewRequestable(preview)) {
      setLoading(false);
      setError(unsupportedText);
      return () => {
        cancelled = true;
      };
    }

    getCachedPreviewBlob(
      {
        scope: cacheScope,
        fileId: preview.fileId,
        volumeId: preview.volumeId || undefined,
        semanticModelId: preview.semanticModelId,
        sourceRowId: preview.sourceRowId,
      },
      () => requestAnswerSourcePreviewBlob(http, preview),
    )
      .then((blob) => {
        if (cancelled) return;
        nextBlobUrl = URL.createObjectURL(blob);
        setBlobUrl(nextBlobUrl);
      })
      .catch((err) => {
        if (!cancelled) {
          console.warn('[KnowledgeExplorePage] inline visual preview failed', err);
          setError(err instanceof Error ? err.message : unsupportedText);
        }
      })
      .finally(() => {
        if (!cancelled) {
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
      if (nextBlobUrl) {
        URL.revokeObjectURL(nextBlobUrl);
      }
    };
  }, [cacheScope, http, preview, unsupportedText]);

  if (loading) {
    return (
      <div className={styles.answerInlineVisualState}>
        <Spin size="small" />
      </div>
    );
  }
  if (error) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} className={styles.answerInlineVisualState} description={error} />;
  }
  return blobUrl ? <Image src={blobUrl} className={styles.answerInlineVisualImage} /> : null;
}

function normalizePositiveIds(ids: number[]): number[] {
  return Array.from(new Set(ids.filter((id) => Number.isInteger(id) && id > 0)));
}

function formatFixedModelLabel(model: string): string {
  const value = model.trim();
  return value || 'Model';
}

export function AnswerSourceRefs({ sources, t }: { sources: ExploreA2AAnswerSourceRef[]; t: KnowledgeT }) {
  const http = useHttpClient();
  const workspaceId = useWorkspaceId();
  const [preview, setPreview] = useState<AnswerSourcePreview | null>(null);
  const groups = groupAnswerSources(sources, t);

  if (!sources.length) {
    return null;
  }
  return (
    <>
      <div className={styles.answerSourcePanel} data-testid="knowledge-explore-answer-sources">
        <div className={styles.answerSourceTitle}>{t('knowledge.explore.final-answer-sources')}</div>
        <div className={styles.answerSourceChips}>
          {groups.map((group, index) => {
            const previews = group.previews;
            const primaryPreview = previews.find((item) => item.kind === 'source') ?? previews[0] ?? null;
            const visualPreviews = previews.filter((item) => item.kind !== 'source');
            const canPreview = Boolean(primaryPreview);
            const pages = group.pages;
            const previewPage = primaryPreview?.page || pages[0] || null;
            const chip = (
              <span
                className={styles.answerSourceChip}
                title={group.title}
                data-disabled={!canPreview && !group.sqlSources.length ? 'true' : undefined}
                data-interactive={canPreview || group.sqlSources.length ? 'true' : undefined}
              >
                {canPreview ? (
                  <button
                    type="button"
                    className={styles.answerSourceMainButton}
                    onClick={() => {
                      if (primaryPreview) {
                        setPreview({ ...primaryPreview, page: previewPage });
                      }
                    }}
                  >
                    <FileTextOutlined className={styles.ragSourceIcon} />
                    <span className={styles.answerSourceName}>{group.label || answerSourceLabel(group.primary, t)}</span>
                  </button>
                ) : (
                  <>
                    <FileTextOutlined className={styles.ragSourceIcon} />
                    <span className={styles.answerSourceName}>{group.label || answerSourceLabel(group.primary, t)}</span>
                  </>
                )}
                {pages.length ? (
                  <>
                    <span className={styles.ragSourceDivider} />
                    <span className={styles.ragSourcePageLabel}>{t('knowledge.explore.source-page-label')}</span>
                    <span className={styles.ragSourcePageList}>
                      {pages.map((page) => (
                        <button
                          key={page}
                          type="button"
                          className={styles.ragSourcePage}
                          disabled={!canPreview}
                          onClick={() => {
                            if (primaryPreview) {
                              setPreview({ ...primaryPreview, page });
                            }
                          }}
                        >
                          {page}
                        </button>
                      ))}
                    </span>
                  </>
                ) : null}
              </span>
            );
            return (
              <div key={`${group.key}-${index}`} className={styles.answerSourceGroup}>
                {group.sqlSources.length ? (
                  <Popover
                    trigger="click"
                    placement="bottomLeft"
                    title={t('knowledge.explore.final-answer-source-sql-query')}
                    content={
                      <div className={styles.answerSourceSQLPopover}>
                        {group.sqlSources.map((source) => (
                          <pre key={source.key} className={styles.answerSourceSQLCode}>
                            <code>{source.sql}</code>
                          </pre>
                        ))}
                      </div>
                    }
                  >
                    {chip}
                  </Popover>
                ) : (
                  chip
                )}
                {visualPreviews.length ? (
                  <div className={styles.answerVisualPreviewList}>
                    {visualPreviews.map((item, itemIndex) => (
                      <AnswerVisualPreview
                        key={`${item.semanticModelId ?? 'source'}-${item.fileId}-${item.kind}`}
                        preview={item}
                        label={answerVisualLabel(item, itemIndex, t)}
                        pageLabel={answerVisualPageLabel(item, t)}
                        http={http}
                        cacheScope={workspaceId}
                        unavailableLabel={t('knowledge.explore.source-preview-image-unavailable')}
                        onOpen={() => setPreview(item)}
                      />
                    ))}
                  </div>
                ) : null}
              </div>
            );
          })}
        </div>
      </div>
      {preview ? (
        <FilePreviewModal
          open
          onClose={() => setPreview(null)}
          fileName={preview.fileName || preview.fileId}
          fileExt={fileExtFromName(preview.fileName || preview.fileId)}
          initialPage={preview.page ?? undefined}
          bboxs={toPdfBboxs(preview.bbox)}
          unsupportedText={t('knowledge.explore.source-preview-unsupported')}
          markdownPreviewLabel={t('knowledge.explore.source-preview-default-title')}
          fetchBlob={async () => {
            return fetchAnswerSourcePreviewBlob(
              http,
              workspaceId,
              preview,
              t('knowledge.explore.source-preview-image-unavailable'),
            );
          }}
        />
      ) : null}
    </>
  );
}

function AnswerVisualPreview({
  preview,
  label,
  pageLabel,
  http,
  cacheScope,
  unavailableLabel,
  onOpen,
}: {
  preview: AnswerSourcePreview;
  label: string;
  pageLabel: string;
  http: AppHttpClient;
  cacheScope: string;
  unavailableLabel: string;
  onOpen: () => void;
}) {
  const previewState = useAnswerPreviewObjectURL(http, cacheScope, preview, unavailableLabel);
  const title = previewState.error ? `${answerVisualTitle(preview)} - ${previewState.error}` : answerVisualTitle(preview);

  return (
    <button type="button" className={styles.answerVisualPreview} title={title} onClick={onOpen}>
      {previewState.src ? (
        <img className={styles.answerVisualPreviewImage} src={previewState.src} alt={label} loading="lazy" />
      ) : (
        <span className={styles.answerVisualPreviewFallback} data-unavailable={previewState.error ? 'true' : undefined}>
          <PictureOutlined />
          {previewState.error ? <span>{unavailableLabel}</span> : null}
        </span>
      )}
      {pageLabel ? <span className={styles.answerVisualPreviewPageBadge}>{pageLabel}</span> : null}
    </button>
  );
}

type FilePreviewObjectURLState = {
  src: string;
  error: string;
};

function useQueryVisualPreviewObjectURL(
  http: AppHttpClient,
  workspaceId: string,
  fileId: string,
  fallbackError: string,
): FilePreviewObjectURLState {
  const [state, setState] = useState<FilePreviewObjectURLState>({ src: '', error: '' });

  useEffect(() => {
    let cancelled = false;
    let objectURL = '';
    setState({ src: '', error: '' });
    if (!fileId) {
      return () => {
        cancelled = true;
      };
    }
    fetchQueryVisualPreviewBlob(http, workspaceId, fileId, fallbackError)
      .then((blob) => {
        const nextURL = URL.createObjectURL(blob);
        if (cancelled) {
          URL.revokeObjectURL(nextURL);
          return;
        }
        objectURL = nextURL;
        setState({ src: nextURL, error: '' });
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setState({ src: '', error: previewErrorMessage(error, fallbackError) });
        }
      });
    return () => {
      cancelled = true;
      if (objectURL) {
        URL.revokeObjectURL(objectURL);
      }
    };
  }, [http, workspaceId, fileId, fallbackError]);

  return state;
}

function useAnswerPreviewObjectURL(
  http: AppHttpClient,
  cacheScope: string,
  preview: AnswerSourcePreview,
  fallbackError: string,
): FilePreviewObjectURLState {
  const [state, setState] = useState<FilePreviewObjectURLState>({ src: '', error: '' });

  useEffect(() => {
    let cancelled = false;
    let objectURL = '';
    setState({ src: '', error: '' });
    if (!preview.fileId) {
      return () => {
        cancelled = true;
      };
    }
    if (!isAnswerPreviewRequestable(preview)) {
      setState({ src: '', error: fallbackError });
      return () => {
        cancelled = true;
      };
    }
    getCachedPreviewBlob(
      {
        scope: cacheScope,
        fileId: preview.fileId,
        volumeId: preview.volumeId || undefined,
        semanticModelId: preview.semanticModelId,
        sourceRowId: preview.sourceRowId,
      },
      () => requestAnswerSourcePreviewBlob(http, preview),
    )
      .then((blob) => {
        const nextURL = URL.createObjectURL(blob);
        if (cancelled) {
          URL.revokeObjectURL(nextURL);
          return;
        }
        objectURL = nextURL;
        setState({ src: nextURL, error: '' });
      })
      .catch((error: unknown) => {
        if (!cancelled) {
          setState({ src: '', error: previewErrorMessage(error, fallbackError) });
        }
      });
    return () => {
      cancelled = true;
      if (objectURL) {
        URL.revokeObjectURL(objectURL);
      }
    };
  }, [cacheScope, fallbackError, http, preview]);

  return state;
}

async function requestQueryVisualPreviewBlob(http: AppHttpClient, workspaceId: string, fileId: string): Promise<Blob> {
  return previewConcreteAgentQueryVisualApi(workspaceId, EXPLORE_QUERY_VISUAL_AGENT_ID, fileId, http);
}

async function fetchQueryVisualPreviewBlob(
  http: AppHttpClient,
  workspaceId: string,
  fileId: string,
  fallbackError?: string,
): Promise<Blob> {
  try {
    return await getCachedPreviewBlob({ scope: `query-visual:${workspaceId}`, fileId }, () =>
      requestQueryVisualPreviewBlob(http, workspaceId, fileId),
    );
  } catch (error) {
    throw new Error(previewErrorMessage(error, fallbackError || 'Failed to load image'));
  }
}

async function fetchAnswerSourcePreviewBlob(
  http: AppHttpClient,
  cacheScope: string,
  preview: AnswerSourcePreview,
  fallbackError?: string,
): Promise<Blob> {
  if (!isAnswerPreviewRequestable(preview)) {
    throw new Error(fallbackError || 'Failed to load image');
  }
  try {
    return await getCachedPreviewBlob(
      {
        scope: cacheScope,
        fileId: preview.fileId,
        volumeId: preview.volumeId || undefined,
        semanticModelId: preview.semanticModelId,
        sourceRowId: preview.sourceRowId,
      },
      () => requestAnswerSourcePreviewBlob(http, preview),
    );
  } catch (error) {
    throw new Error(previewErrorMessage(error, fallbackError || 'Failed to load image'));
  }
}

function isAnswerPreviewRequestable(preview: AnswerSourcePreview): boolean {
  return (
    preview.kind === 'source' ||
    (typeof preview.semanticModelId === 'number' && Number.isInteger(preview.semanticModelId) && preview.semanticModelId > 0)
  );
}

function previewErrorMessage(error: unknown, fallbackError: string): string {
  if (error && typeof error === 'object') {
    const response = (error as { response?: { data?: unknown; status?: number } }).response;
    if (response?.data && typeof response.data === 'object') {
      const data = response.data as { message?: unknown; msg?: unknown; error?: unknown };
      for (const value of [data.message, data.msg, data.error]) {
        if (typeof value === 'string' && value) {
          return value;
        }
      }
    }
    if (typeof response?.data === 'string' && response.data) {
      return response.data;
    }
    if (response?.status) {
      return `Request failed with status code ${response.status}`;
    }
    const message = (error as { message?: unknown }).message;
    if (typeof message === 'string' && message) {
      return message;
    }
  }
  return fallbackError;
}

type AnswerSourceGroup = {
  key: string;
  primary: ExploreA2AAnswerSourceRef;
  label: string;
  title: string;
  pages: number[];
  previews: AnswerSourcePreview[];
  sqlSources: AnswerSourceSQL[];
};

type AnswerSourceSQL = {
  key: string;
  label: string;
  title: string;
  sql: string;
};

function groupAnswerSources(sources: ExploreA2AAnswerSourceRef[], t: KnowledgeT): AnswerSourceGroup[] {
  const groups: AnswerSourceGroup[] = [];
  const bySourceKey = new Map<string, AnswerSourceGroup>();
  for (const source of sources) {
    const sourceKey = answerSourceGroupKey(source);
    let group = sourceKey ? bySourceKey.get(sourceKey) : undefined;
    if (!group) {
      group = {
        key: sourceKey || answerSourceKey(source, groups.length),
        primary: source,
        label: answerSourceLabel(source, t),
        title: answerSourceTitle(source),
        pages: [],
        previews: [],
        sqlSources: [],
      };
      groups.push(group);
      if (sourceKey) {
        bySourceKey.set(sourceKey, group);
      }
    } else {
      group.title = [group.title, answerSourceTitle(source)].filter(Boolean).join(' · ');
    }
    group.pages = mergeAnswerSourcePages(group.pages, answerSourcePages(source));
    for (const preview of answerSourcePreviews(source)) {
      pushAnswerSourcePreview(group.previews, preview);
    }
    pushAnswerSourceSQL(group.sqlSources, source);
  }
  for (const group of groups) {
    group.pages = sortNumericUnique(group.pages);
  }
  return groups;
}

function answerSourceGroupKey(source: ExploreA2AAnswerSourceRef): string {
  const sourceIdentity = answerSourceIdentity(source.semanticModelId, source.sourceRowId, source.fileId);
  if (sourceIdentity) {
    return sourceIdentity;
  }
  if (source.type === 'sql_table' && (source.database || source.table)) {
    return `sql_table\x00${source.database}\x00${source.table}`;
  }
  return '';
}

function pushAnswerSourcePreview(out: AnswerSourcePreview[], preview: AnswerSourcePreview) {
  if (!preview.fileId) {
    return;
  }
  const key =
    preview.kind === 'source'
      ? answerSourceIdentity(preview.semanticModelId, preview.sourceRowId, preview.fileId)
      : `${preview.semanticModelId ?? ''}\x00${preview.fileId}`;
  if (
    out.some(
      (item) =>
        (item.kind === 'source'
          ? answerSourceIdentity(item.semanticModelId, item.sourceRowId, item.fileId)
          : `${item.semanticModelId ?? ''}\x00${item.fileId}`) === key,
    )
  ) {
    return;
  }
  out.push(preview);
}

function mergeAnswerSourcePages(left: number[], right: number[]): number[] {
  return sortNumericUnique([...left, ...right]);
}

function sortNumericUnique(values: number[]): number[] {
  return Array.from(new Set(values.filter((value) => Number.isFinite(value) && value > 0))).sort((a, b) => a - b);
}

function pushAnswerSourceSQL(out: AnswerSourceSQL[], source: ExploreA2AAnswerSourceRef) {
  if (source.type !== 'sql_table' || !source.sql) {
    return;
  }
  const key = [source.database, source.table, source.sql].join('\x00');
  if (out.some((item) => item.key === key)) {
    return;
  }
  out.push({
    key,
    label: source.label,
    title: answerSourceTitle(source),
    sql: source.sql,
  });
}

function answerSourceLabel(source: ExploreA2AAnswerSourceRef, t: KnowledgeT): string {
  return answerSourceDisplayLabel(source, {
    rag: t('knowledge.explore.final-answer-source-rag'),
    sql: t('knowledge.explore.final-answer-source-sql'),
    image: t('knowledge.explore.final-answer-source-image'),
  });
}

function toPdfBboxs(bbox: number[]): [number, number, number, number][] | undefined {
  if (bbox.length !== 4 || bbox.some((value) => !Number.isFinite(value))) {
    return undefined;
  }
  return [[bbox[0], bbox[1], bbox[2], bbox[3]]];
}

function answerSourceTitle(source: ExploreA2AAnswerSourceRef): string {
  return [
    source.type,
    source.semanticModelId ? `model: ${source.semanticModelId}` : '',
    source.artifactId ? `artifact: ${source.artifactId}` : '',
    source.chunkIds.length ? `chunks: ${source.chunkIds.join(', ')}` : source.chunkId ? `chunk: ${source.chunkId}` : '',
    source.fileId ? `file: ${source.fileId}` : '',
    source.markdownFileId ? `markdown: ${source.markdownFileId}` : '',
    source.imageFileId ? `image: ${source.imageFileId}` : '',
    source.pageImageFileId ? `page image: ${source.pageImageFileId}` : '',
    source.sql ? `sql: ${source.sql}` : '',
    source.bbox.length ? `bbox: ${source.bbox.join(', ')}` : '',
  ]
    .filter(Boolean)
    .join(' · ');
}

function answerVisualTitle(preview: AnswerSourcePreview): string {
  return [
    preview.kind,
    preview.fileId ? `file: ${preview.fileId}` : '',
    preview.page ? `page: ${preview.page}` : '',
    preview.bbox.length ? `bbox: ${preview.bbox.join(', ')}` : '',
  ]
    .filter(Boolean)
    .join(' · ');
}

function answerVisualLabel(preview: AnswerSourcePreview, index: number, t: KnowledgeT): string {
  const base =
    preview.kind === 'page_visual'
      ? t('knowledge.explore.final-answer-source-page-image')
      : t('knowledge.explore.final-answer-source-image');
  return preview.page ? `${base} ${preview.page}` : `${base} ${index + 1}`;
}

function answerVisualPageLabel(preview: AnswerSourcePreview, t: KnowledgeT): string {
  return preview.page && Number.isFinite(preview.page) && preview.page > 0
    ? `${t('knowledge.explore.source-page-label')} ${preview.page}`
    : '';
}

function answerSourceKey(source: ExploreA2AAnswerSourceRef, index: number): string {
  return [
    index,
    source.type,
    source.artifactId,
    source.chunkId,
    source.chunkIds.join(','),
    source.fileId,
    source.fileName,
    source.volumeId,
    source.pages.join(','),
    source.database,
    source.table,
    source.sql,
    source.label,
    source.imageFileId,
    source.pageImageFileId,
    source.bbox.join(','),
  ].join('-');
}

function fileExtFromName(fileName: string): string {
  const index = fileName.lastIndexOf('.');
  return index >= 0 ? fileName.slice(index + 1) : '';
}

export function AnswerFeedback({
  content,
  messageId,
  taskId,
  initialFeedback,
  onFeedback,
}: {
  content: string;
  messageId?: number;
  taskId?: string | null;
  initialFeedback?: 'like' | 'dislike' | null;
  onFeedback: (
    messageId: number | undefined,
    taskId: string | null | undefined,
    feedback: 'like' | 'dislike',
  ) => Promise<AnswerFeedbackResult>;
}) {
  const { t } = useTranslation('moi-knowledge');
  const { message: appMessage } = App.useApp();
  const [feedback, setFeedback] = useState<'like' | 'dislike' | null>(null);
  const [resolvedMessageId, setResolvedMessageId] = useState<number | undefined>(messageId);
  const [isSubmittingFeedback, setIsSubmittingFeedback] = useState(false);

  useEffect(() => {
    setFeedback(initialFeedback ?? null);
    setResolvedMessageId(messageId);
  }, [initialFeedback, messageId]);

  const handleCopy = useCallback(() => {
    if (!navigator.clipboard) {
      appMessage.error(t('knowledge.explore.answer-copy-failed'));
      return;
    }
    navigator.clipboard
      .writeText(content)
      .then(() => appMessage.success(t('knowledge.explore.answer-copy-success')))
      .catch(() => appMessage.error(t('knowledge.explore.answer-copy-failed')));
  }, [appMessage, content, t]);

  const handleFeedback = useCallback(
    (nextFeedback: 'like' | 'dislike') => {
      setIsSubmittingFeedback(true);
      onFeedback(resolvedMessageId, taskId, nextFeedback)
        .then((result) => {
          setResolvedMessageId(result.messageId);
          setFeedback(result.feedback);
        })
        .catch((error) => {
          console.warn('[KnowledgeExplorePage] feedback failed', { msg: (error as Error).message });
          appMessage.error(t('knowledge.explore.answer-feedback-submit-failed'));
        })
        .finally(() => setIsSubmittingFeedback(false));
    },
    [appMessage, onFeedback, resolvedMessageId, t, taskId],
  );

  return (
    <div className={styles.answerFeedback} data-testid="knowledge-explore-answer-feedback">
      <Tooltip title={t('knowledge.explore.answer-copy')}>
        <Button
          type="text"
          size="small"
          icon={<CopyOutlined />}
          className={styles.answerFeedbackButton}
          aria-label={t('knowledge.explore.answer-copy-aria')}
          onClick={handleCopy}
        />
      </Tooltip>
      <Tooltip title={t('knowledge.explore.answer-feedback-like')}>
        <Button
          type="text"
          size="small"
          icon={feedback === 'like' ? <LikeFilled /> : <LikeOutlined />}
          className={`${styles.answerFeedbackButton} ${feedback === 'like' ? styles.answerFeedbackButtonActive : ''}`}
          aria-label={t('knowledge.explore.answer-feedback-like')}
          loading={isSubmittingFeedback}
          onClick={() => handleFeedback('like')}
        />
      </Tooltip>
      <Tooltip title={t('knowledge.explore.answer-feedback-dislike')}>
        <Button
          type="text"
          size="small"
          icon={feedback === 'dislike' ? <DislikeFilled /> : <DislikeOutlined />}
          className={`${styles.answerFeedbackButton} ${feedback === 'dislike' ? styles.answerFeedbackButtonActive : ''}`}
          aria-label={t('knowledge.explore.answer-feedback-dislike')}
          loading={isSubmittingFeedback}
          onClick={() => handleFeedback('dislike')}
        />
      </Tooltip>
    </div>
  );
}

async function resolveAssistantMessageId(http: AppHttpClient, sessionId: number, taskId: string): Promise<number | undefined> {
  const records = await getSessionMessages(http, sessionId);
  const match = records.find((record) => record.role === 'assistant' && taskIdFromConfig(record.config) === taskId);
  return match?.id;
}

function taskIdFromConfig(rawConfig: unknown): string {
  if (typeof rawConfig !== 'string' || !rawConfig.trim()) {
    return '';
  }
  try {
    const parsed = JSON.parse(rawConfig) as { task_id?: unknown };
    return typeof parsed.task_id === 'string' ? parsed.task_id : '';
  } catch {
    return '';
  }
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

function renderPlanStatus(status: string, t: KnowledgeT): string {
  const normalized = status.trim() || 'pending';
  const translated = t(`knowledge.explore.a2a-status-${normalized}`, { defaultValue: normalized });
  return typeof translated === 'string' && translated ? translated : normalized;
}

function renderToolStatus(status: string, t: KnowledgeT): string {
  const normalized = status.trim() || 'completed';
  const translated = t(`knowledge.explore.trace-tool-status-${normalized}`, { defaultValue: normalized });
  return typeof translated === 'string' && translated ? translated : normalized;
}

function renderExploreActivityText(key: string, params: Record<string, unknown>, t: KnowledgeT): string | undefined {
  if (key === 'agent.a2a.activity.finalizingSources') {
    return t('knowledge.explore.activity-finalizing-sources', params);
  }
  return undefined;
}
