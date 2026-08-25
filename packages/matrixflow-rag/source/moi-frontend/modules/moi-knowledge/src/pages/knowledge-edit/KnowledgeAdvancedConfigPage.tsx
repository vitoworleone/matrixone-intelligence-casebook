import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  App,
  Button,
  Dropdown,
  Empty,
  Grid,
  Input,
  Modal,
  Pagination,
  Popconfirm,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tabs,
  Tag,
  Tooltip,
  Typography,
  type MenuProps,
  type TableColumnsType,
} from 'antd';
import {
  ArrowLeftOutlined,
  CaretRightOutlined,
  CheckOutlined,
  CloseOutlined,
  FileTextOutlined,
  FilterOutlined,
  HddOutlined,
  LeftOutlined,
  PauseOutlined,
  PlusOutlined,
  RightOutlined,
  RobotOutlined,
  TableOutlined,
} from '@ant-design/icons';

import type { AgentRecord } from '@moi/shared-moi-api/agent';
import {
  downloadTableApi,
  getTableInfoApi,
  getTableSampleDataApi,
  originFileDownloadApi,
  type CatalogTableColumn,
  type CatalogTableInfo,
  type CatalogTableSampleResp,
  type CatalogTableStat,
} from '@moi/shared-moi-api/catalog';
import {
  previewSemanticModelArtifactApi,
  previewSemanticModelSourceFileApi,
  type KnowledgeBaseSourceJobRun,
  type SemanticModelDocumentSegment,
  type SemanticModelSegmentVersion,
  type SemanticModelSource,
  type SemanticModelSourceDocument,
  type SemanticModelSourceSubmitPayload,
  type SemanticModelSourceType,
} from '@moi/shared-moi-api/knowledge';
import { isSuccessApiCode } from '@moi/shared-moi-api/response';
import { useHttpClient, useNavigator, useTimezone } from '@moi/shared-moi-app-protocol/app-context';
import { useUser, useWorkspaceId } from '@moi/shared-moi-app-protocol/business-context';
import { useModuleNavigator } from '@moi/shared-moi-app-protocol/module-context';
import { EmbeddedFilePreview, isMediaPreviewSupported } from '@moi/shared-moi-components/file-preview';
import { ListActionButton } from '@moi/shared-moi-components/list-action';
import { downloadBlob } from '@moi/shared-moi-utils/browser/blob';
import { formatDateTime } from '@moi/shared-moi-utils/datetime';
import KnowledgeSourceSelectModal from '../../components/KnowledgeSourceSelectModal/KnowledgeSourceSelectModal';
import { getAgentKnowledgeBaseIds, listKnowledgeAssociatedAgents } from '../../service/agent';
import {
  appendKnowledgeSources,
  createKnowledgeSegment,
  deleteKnowledgeSegment,
  deleteKnowledgeSource,
  getKnowledgeDetail,
  getKnowledgeSourceDocument,
  importInitialKnowledgeSegments,
  listKnowledgeSourceJobs,
  listKnowledgeSources,
  reconcileKnowledgeSourceJobs,
  setCurrentKnowledgeSegmentVersion,
  updateKnowledgeSegment,
  updateKnowledgeSegmentEnabled,
  updateKnowledgeSourceGovernance,
  type GetKnowledgeDetailResponse,
  type QueryKnowledgeSourcesResponse,
} from '../../service/knowledge';
import { resolveKnowledgeErrorMessage } from '../../shared/knowledge/error-message';
import KnowledgeAdvancedConfig from './components/KnowledgeAdvancedConfig';
import styles from './KnowledgeAdvancedConfigPage.module.css';

const { Title, Paragraph, Text } = Typography;
type AdvancedTabKey = 'source' | 'semantic' | 'agents';
type DocumentTabKey = 'preview' | 'info';
type SegmentSortKey = 'source_order' | 'recall_priority';
type SegmentDraft = { content?: string; ocr_text?: string; image_description?: string };

const SEGMENT_PAGE_SIZE = 10;
const SOURCE_PAGE_SIZE = 10;
const SOURCE_JOB_POLL_INTERVAL_MS = 5000;
const TABLE_SAMPLE_PAGE_SIZE = 20;
const SOURCE_ACTION_COLUMN_WIDTH = 280;
const SOURCE_MOBILE_ACTION_COLUMN_WIDTH = 96;
const SOURCE_TABLE_SCROLL_X = 1620;
const ACTIVE_SOURCE_JOB_STATUSES = new Set(['queued', 'running', 'pending']);

interface TableDetailState {
  source: SemanticModelSource;
  info: CatalogTableInfo;
  sample: CatalogTableSampleResp;
}

interface KnowledgeSourcesLoadResult {
  loaded: boolean;
  page: number;
  pageSize: number;
  total: number;
}

function getSourceDisplayName(entry: SemanticModelSource) {
  return entry.display_name || entry.resource_id;
}

function getFileExtFromName(name: string): string {
  const dotIndex = name.lastIndexOf('.');
  if (dotIndex < 0 || dotIndex === name.length - 1) return '';
  return name.slice(dotIndex + 1);
}

export function resolveDocumentPreviewFileId(source: SemanticModelSource): string {
  return source.source_file_id ?? '';
}

function formatSegmentTimestamp(value: number): string {
  const totalSeconds = Math.max(0, Math.floor(value / 1000));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const short = `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
  return hours > 0 ? `${String(hours).padStart(2, '0')}:${short}` : short;
}

function isCompletedProcessStatus(status: string | null | undefined): boolean {
  return status === 'succeeded' || status === 'success' || status === 'ready' || status === 'completed';
}

function isFailedProcessStatus(status: string | null | undefined): boolean {
  return status === 'failed';
}

function isActiveSourceJobStatus(status: string | null | undefined): boolean {
  return typeof status === 'string' && ACTIVE_SOURCE_JOB_STATUSES.has(status);
}

function isPollableSourceJob(job: KnowledgeBaseSourceJobRun): boolean {
  return isActiveSourceJobStatus(job.job_status);
}

function isLegacyUnboundSource(entry: SemanticModelSource): boolean {
  return entry.governance_status === 'legacy_unbound';
}

function getSegmentSearchValues(segment: SemanticModelDocumentSegment): string[] {
  const mediaSearchValues =
    segment.image_file_id || segment.page_image_file_id ? [segment.ocr_text, segment.image_description] : [];
  return [
    segment.content,
    ...mediaSearchValues,
    segment.chunk_id,
    segment.image_file_id,
    segment.page_image_file_id,
    segment.level,
    segment.chunk_index === null || segment.chunk_index === undefined ? null : String(segment.chunk_index),
  ].filter((value): value is string => typeof value === 'string');
}

function getSegmentImageFileId(segment: SemanticModelDocumentSegment): string {
  return segment.image_file_id || segment.page_image_file_id || '';
}

function getVersionOperatorId(version: SemanticModelSegmentVersion): string {
  return version.updated_by || version.created_by || '';
}

function getVersionTimestamp(version: SemanticModelSegmentVersion): number | null {
  return version.updated_at ?? version.created_at ?? null;
}

function getVersionOrdinalMap(versions: SemanticModelSegmentVersion[]): Map<string, number> {
  const ordered = [...versions].sort((left, right) => {
    const leftIndex = left.index_version ?? Number.MAX_SAFE_INTEGER;
    const rightIndex = right.index_version ?? Number.MAX_SAFE_INTEGER;
    if (leftIndex !== rightIndex) return leftIndex - rightIndex;
    return (getVersionTimestamp(left) ?? 0) - (getVersionTimestamp(right) ?? 0);
  });
  return new Map(ordered.map((version, index) => [version.version_id, index + 1]));
}

export default function KnowledgeAdvancedConfigPage() {
  const { t } = useTranslation('moi-knowledge');
  const { message, modal } = App.useApp();
  const http = useHttpClient();
  const timezone = useTimezone();
  const workspaceId = useWorkspaceId();
  const user = useUser();
  const nav = useModuleNavigator();
  const appNavigator = useNavigator();
  const screens = Grid.useBreakpoint();
  const sourceActionColumnWidth = screens.md ? SOURCE_ACTION_COLUMN_WIDTH : SOURCE_MOBILE_ACTION_COLUMN_WIDTH;
  const sourceTableScrollX = SOURCE_TABLE_SCROLL_X - SOURCE_ACTION_COLUMN_WIDTH + sourceActionColumnWidth;

  const getRouteParam = nav.callbacks.getRouteParam as ((name: string) => string | undefined) | undefined;
  const routeKnowledgeId = getRouteParam?.('id');
  const knowledgeId = Number(routeKnowledgeId);
  const hasValidKnowledgeId = Number.isFinite(knowledgeId) && knowledgeId > 0;

  const [detailLoading, setDetailLoading] = useState(false);
  const [sourceLoading, setSourceLoading] = useState(false);
  const [detail, setDetail] = useState<GetKnowledgeDetailResponse | null>(null);
  const [sourceRows, setSourceRows] = useState<SemanticModelSource[]>([]);
  const [sourcePagination, setSourcePagination] = useState({ page: 1, pageSize: SOURCE_PAGE_SIZE, total: 0 });
  const [sourceJobRows, setSourceJobRows] = useState<KnowledgeBaseSourceJobRun[]>([]);
  const [sourceJobsReconcileRequired, setSourceJobsReconcileRequired] = useState(false);
  const [sourceJobsRetryRequired, setSourceJobsRetryRequired] = useState(false);
  const [sourceRefreshRetryRequired, setSourceRefreshRetryRequired] = useState(false);
  const [sourceLoadFailed, setSourceLoadFailed] = useState(false);
  const [associatedAgentRows, setAssociatedAgentRows] = useState<AgentRecord[]>([]);
  const [agentAssociationLoading, setAgentAssociationLoading] = useState(false);
  const [agentAssociationLoadFailed, setAgentAssociationLoadFailed] = useState(false);
  const [sourceSelectOpen, setSourceSelectOpen] = useState(false);
  const [sourceAppending, setSourceAppending] = useState(false);
  const [deletingSourceRowId, setDeletingSourceRowId] = useState<string | null>(null);
  const [governanceUpdatingSourceRowId, setGovernanceUpdatingSourceRowId] = useState<string | null>(null);
  const [downloadingSourceRowId, setDownloadingSourceRowId] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<AdvancedTabKey>('source');
  const [documentOpen, setDocumentOpen] = useState(false);
  const [documentLoading, setDocumentLoading] = useState(false);
  const [documentTagsSaving, setDocumentTagsSaving] = useState(false);
  const [documentExpirySaving, setDocumentExpirySaving] = useState(false);
  const [documentDetail, setDocumentDetail] = useState<SemanticModelSourceDocument | null>(null);
  const [documentTab, setDocumentTab] = useState<DocumentTabKey>('preview');
  const [tableDetailOpen, setTableDetailOpen] = useState(false);
  const [tableDetailLoading, setTableDetailLoading] = useState(false);
  const [tableDetail, setTableDetail] = useState<TableDetailState | null>(null);
  const [segmentSearch, setSegmentSearch] = useState('');
  const [segmentSort, setSegmentSort] = useState<SegmentSortKey>('source_order');
  const [segmentPage, setSegmentPage] = useState(1);
  const [segmentMutatingKey, setSegmentMutatingKey] = useState<string | null>(null);
  const [editingSegmentId, setEditingSegmentId] = useState<string | null>(null);
  const [segmentDrafts, setSegmentDrafts] = useState<Record<string, SegmentDraft>>({});
  const [segmentPreview, setSegmentPreview] = useState<{ open: boolean; url: string; revoke: boolean; title: string }>({
    open: false,
    url: '',
    revoke: false,
    title: '',
  });
  const [segmentPreviewLoadingKey, setSegmentPreviewLoadingKey] = useState<string | null>(null);
  const [playingSegmentId, setPlayingSegmentId] = useState<string | null>(null);
  const [newSegmentOpen, setNewSegmentOpen] = useState(false);
  const [newSegmentContent, setNewSegmentContent] = useState('');
  const [draftTags, setDraftTags] = useState<string[]>([]);
  const [draftExpiresAt, setDraftExpiresAt] = useState<number | null>(null);
  const [expirySource, setExpirySource] = useState<SemanticModelSource | null>(null);
  const [expiryDraftExpiresAt, setExpiryDraftExpiresAt] = useState<number | null>(null);
  const [expirySaving, setExpirySaving] = useState(false);
  const requestedDetailKnowledgeIdRef = useRef<number | null>(null);
  const requestedSourceKnowledgeIdRef = useRef<number | null>(null);
  const currentSourceKnowledgeIdRef = useRef(knowledgeId);
  const sourceRequestKnowledgeIdRef = useRef(knowledgeId);
  const sourceRequestGenerationRef = useRef(0);
  const sourceListRequestSequenceRef = useRef(0);
  const sourceLoadingRequestSequenceRef = useRef(0);
  const sourceForegroundRequestSequenceRef = useRef(0);
  const requestedAgentKnowledgeIdRef = useRef<number | null>(null);
  const requestedSourceJobsKnowledgeIdRef = useRef<number | null>(null);
  const sourceJobPollingInFlightRef = useRef(false);
  const sourceJobDriveRequestedRef = useRef(false);
  const sourceJobWorkObservedRef = useRef(false);
  const sourceRefreshRetryRequiredRef = useRef(false);
  const sourceRefreshPendingRef = useRef(false);
  const sourcePageRef = useRef(sourcePagination.page);
  const invalidKnowledgeIdRef = useRef<string | null>(null);
  const canUpdateKnowledgeSources = hasValidKnowledgeId;
  // Automatic reconciliation is a write side effect. This release exposes
  // collaboration only through Core-verified admin/superadmin roles; Backend
  // still reauthorizes semantic_model.use + semantic_model.update.
  const canAutoReconcileSourceJobs = hasValidKnowledgeId && user.isAdmin;
  const canAppendSources = canUpdateKnowledgeSources;
  const documentSaving = documentTagsSaving || documentExpirySaving;
  if (sourceRequestKnowledgeIdRef.current !== knowledgeId) {
    sourceRequestKnowledgeIdRef.current = knowledgeId;
    sourceRequestGenerationRef.current++;
  }
  const sourceRequestGeneration = sourceRequestGenerationRef.current;
  currentSourceKnowledgeIdRef.current = knowledgeId;
  sourcePageRef.current = sourcePagination.page;

  const isCurrentSourceRequest = useCallback(
    (requestedKnowledgeId: number, requestedGeneration: number) =>
      currentSourceKnowledgeIdRef.current === requestedKnowledgeId && sourceRequestGenerationRef.current === requestedGeneration,
    [],
  );
  const isCurrentSourceListRequest = useCallback(
    (requestedKnowledgeId: number, requestedGeneration: number, requestedSequence: number) =>
      isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration) &&
      sourceListRequestSequenceRef.current === requestedSequence,
    [isCurrentSourceRequest],
  );

  useEffect(() => {
    const drafts: Record<string, SegmentDraft> = {};
    for (const segment of documentDetail?.segments ?? []) {
      drafts[segment.segment_id] = {
        content: segment.content ?? '',
        ocr_text: segment.ocr_text ?? '',
        image_description: segment.image_description ?? '',
      };
    }
    setSegmentDrafts(drafts);
  }, [documentDetail?.selected_segment_version_id, documentDetail?.current_segment_version_id, documentDetail?.segments]);

  useEffect(
    () => () => {
      if (segmentPreview.revoke && segmentPreview.url) {
        URL.revokeObjectURL(segmentPreview.url);
      }
    },
    [segmentPreview],
  );

  const handleBack = useCallback(() => {
    nav.goToPage('knowledge-board');
  }, [nav]);

  const loadKnowledgeDetail = useCallback(async () => {
    if (!hasValidKnowledgeId) return;

    try {
      setDetailLoading(true);
      const data = await getKnowledgeDetail(http, { id: knowledgeId });
      setDetail(data);
    } catch (error) {
      message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.load-detail-failed'));
      console.error('[KnowledgeAdvancedConfigPage] load detail failed', error);
    } finally {
      setDetailLoading(false);
    }
  }, [hasValidKnowledgeId, http, knowledgeId, message]);

  const syncDocumentDraft = useCallback((nextDetail: SemanticModelSourceDocument) => {
    const info = nextDetail.file_info;
    setDraftTags(Array.isArray(info.tags) ? info.tags : []);
    setDraftExpiresAt(info.expires_at ?? null);
  }, []);

  const applyKnowledgeSourceList = useCallback((sourcesData: QueryKnowledgeSourcesResponse, requestedPage: number) => {
    const nextPage = sourcesData.page && sourcesData.page > 0 ? sourcesData.page : requestedPage;
    const nextPageSize = sourcesData.page_size && sourcesData.page_size > 0 ? sourcesData.page_size : SOURCE_PAGE_SIZE;
    const items = Array.isArray(sourcesData.items) ? sourcesData.items : [];
    const nextTotal = Number.isFinite(sourcesData.total) ? sourcesData.total : items.length;
    setSourceRows(items);
    setSourcePagination({ page: nextPage, pageSize: nextPageSize, total: nextTotal });
    return {
      page: nextPage,
      pageSize: nextPageSize,
      total: nextTotal,
    };
  }, []);

  const loadKnowledgeSources = useCallback(
    async (options?: { page?: number; silent?: boolean; retryOnFailure?: boolean }): Promise<KnowledgeSourcesLoadResult> => {
      if (!hasValidKnowledgeId) {
        return { loaded: false, page: 1, pageSize: SOURCE_PAGE_SIZE, total: 0 };
      }
      const requestedPage = options?.page ?? sourcePageRef.current;
      const requestedKnowledgeId = knowledgeId;
      const requestedGeneration = sourceRequestGenerationRef.current;
      const silent = options?.silent === true;
      if (silent && sourceForegroundRequestSequenceRef.current !== 0) {
        sourceRefreshPendingRef.current = true;
        sourceRefreshRetryRequiredRef.current = true;
        setSourceRefreshRetryRequired(true);
        console.warn('[KnowledgeAdvancedConfigPage] defer source refresh while a foreground request is in flight', {
          knowledgeId,
        });
        return { loaded: false, page: requestedPage, pageSize: SOURCE_PAGE_SIZE, total: 0 };
      }
      const requestedSequence = ++sourceListRequestSequenceRef.current;

      try {
        if (!silent) {
          sourceLoadingRequestSequenceRef.current = requestedSequence;
          sourceForegroundRequestSequenceRef.current = requestedSequence;
          setSourceLoading(true);
          setSourceLoadFailed(false);
        }
        const sourcesData = await listKnowledgeSources(http, {
          id: knowledgeId,
          page: requestedPage,
          page_size: SOURCE_PAGE_SIZE,
        });
        if (!isCurrentSourceListRequest(requestedKnowledgeId, requestedGeneration, requestedSequence)) {
          return { loaded: false, page: requestedPage, pageSize: SOURCE_PAGE_SIZE, total: 0 };
        }
        if (silent || !sourceRefreshPendingRef.current) {
          sourceRefreshPendingRef.current = false;
          sourceRefreshRetryRequiredRef.current = false;
          setSourceRefreshRetryRequired(false);
        }
        const nextList = applyKnowledgeSourceList(sourcesData, requestedPage);
        return {
          loaded: true,
          ...nextList,
        };
      } catch (error) {
        if (!isCurrentSourceListRequest(requestedKnowledgeId, requestedGeneration, requestedSequence)) {
          return { loaded: false, page: requestedPage, pageSize: SOURCE_PAGE_SIZE, total: 0 };
        }
        if (options?.retryOnFailure) {
          sourceRefreshPendingRef.current = true;
          sourceRefreshRetryRequiredRef.current = true;
          setSourceRefreshRetryRequired(true);
        }
        if (silent) {
          console.warn('[KnowledgeAdvancedConfigPage] refresh source list failed', {
            error: error instanceof Error ? error.message : String(error),
          });
        } else {
          setSourceLoadFailed(true);
          message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.source-list-load-failed'));
          console.error('[KnowledgeAdvancedConfigPage] load source list failed', error);
        }
        return { loaded: false, page: requestedPage, pageSize: SOURCE_PAGE_SIZE, total: 0 };
      } finally {
        if (!silent && sourceForegroundRequestSequenceRef.current === requestedSequence) {
          sourceForegroundRequestSequenceRef.current = 0;
        }
        if (
          !silent &&
          isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration) &&
          sourceLoadingRequestSequenceRef.current === requestedSequence
        ) {
          setSourceLoading(false);
        }
      }
    },
    [
      applyKnowledgeSourceList,
      hasValidKnowledgeId,
      http,
      isCurrentSourceListRequest,
      isCurrentSourceRequest,
      knowledgeId,
      message,
    ],
  );

  const loadKnowledgeSourceJobs = useCallback(
    async (options?: {
      silent?: boolean;
    }): Promise<{ loaded: boolean; jobs: KnowledgeBaseSourceJobRun[]; reconcileRequired: boolean }> => {
      if (!hasValidKnowledgeId) return { loaded: false, jobs: [], reconcileRequired: false };
      const requestedKnowledgeId = knowledgeId;
      const requestedGeneration = sourceRequestGenerationRef.current;

      try {
        const jobsData = await listKnowledgeSourceJobs(http, { id: knowledgeId });
        if (!isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)) {
          return { loaded: false, jobs: [], reconcileRequired: false };
        }
        setSourceJobRows(jobsData.items);
        setSourceJobsReconcileRequired(jobsData.reconcile_required);
        setSourceJobsRetryRequired(false);
        return { loaded: true, jobs: jobsData.items, reconcileRequired: jobsData.reconcile_required };
      } catch (error) {
        if (!isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)) {
          return { loaded: false, jobs: [], reconcileRequired: false };
        }
        setSourceJobsRetryRequired(true);
        console.warn(
          options?.silent
            ? '[KnowledgeAdvancedConfigPage] poll source jobs failed'
            : '[KnowledgeAdvancedConfigPage] load source jobs failed',
          {
            error: error instanceof Error ? error.message : String(error),
          },
        );
        return { loaded: false, jobs: [], reconcileRequired: false };
      }
    },
    [hasValidKnowledgeId, http, isCurrentSourceRequest, knowledgeId],
  );

  const loadAssociatedAgents = useCallback(async () => {
    if (!hasValidKnowledgeId) return;
    const requestKnowledgeId = knowledgeId;
    requestedAgentKnowledgeIdRef.current = requestKnowledgeId;

    try {
      setAgentAssociationLoading(true);
      setAgentAssociationLoadFailed(false);
      const agents = await listKnowledgeAssociatedAgents(http, { workspaceId, knowledgeId: requestKnowledgeId });
      if (requestedAgentKnowledgeIdRef.current !== requestKnowledgeId) return;
      setAssociatedAgentRows(agents);
    } catch (error) {
      if (requestedAgentKnowledgeIdRef.current !== requestKnowledgeId) return;
      setAgentAssociationLoadFailed(true);
      message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.agent-association-load-failed'));
      console.error('[KnowledgeAdvancedConfigPage] load associated agents failed', error);
    } finally {
      if (requestedAgentKnowledgeIdRef.current === requestKnowledgeId) {
        setAgentAssociationLoading(false);
      }
    }
  }, [hasValidKnowledgeId, http, knowledgeId, message, workspaceId]);

  const refreshOpenDocumentDetail = useCallback(async () => {
    if (!hasValidKnowledgeId || !documentDetail) return;
    const requestedKnowledgeId = knowledgeId;
    const requestedGeneration = sourceRequestGenerationRef.current;
    const nextDetail = await getKnowledgeSourceDocument(http, {
      id: knowledgeId,
      sourceRowId: documentDetail.source.row_id,
    });
    if (!isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)) return;
    setDocumentDetail(nextDetail);
    syncDocumentDraft(nextDetail);
  }, [documentDetail, hasValidKnowledgeId, http, isCurrentSourceRequest, knowledgeId, syncDocumentDraft]);

  const reconcileCompletedSourceJobs = useCallback(
    async (reconcileRequired: boolean, requestedKnowledgeId: number, requestedGeneration: number): Promise<boolean> => {
      if (!hasValidKnowledgeId || !canAutoReconcileSourceJobs) return false;
      if (!reconcileRequired || !isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)) return false;

      try {
        await reconcileKnowledgeSourceJobs(http, { id: requestedKnowledgeId });
        return isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration);
      } catch (error) {
        if (!isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)) return false;
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.source-job-reconcile-failed'));
        console.error('[KnowledgeAdvancedConfigPage] reconcile source jobs failed', error);
        return false;
      }
    },
    [canAutoReconcileSourceJobs, hasValidKnowledgeId, http, isCurrentSourceRequest, message],
  );

  const requestKnowledgeSources = useCallback(
    async (page?: number) => {
      if (!hasValidKnowledgeId) return;
      const requestedPage = page ?? sourcePageRef.current;
      requestedSourceKnowledgeIdRef.current = knowledgeId;
      const next = await loadKnowledgeSources({ page: requestedPage });
      if (!next.loaded && requestedSourceKnowledgeIdRef.current === knowledgeId) {
        requestedSourceKnowledgeIdRef.current = null;
        return;
      }
      if (!next.loaded) return;
      await refreshOpenDocumentDetail();
    },
    [hasValidKnowledgeId, knowledgeId, loadKnowledgeSources, refreshOpenDocumentDetail],
  );

  const driveKnowledgeSourceJobs = useCallback(async () => {
    if (!hasValidKnowledgeId) return;
    const requestedKnowledgeId = knowledgeId;
    const requestedGeneration = sourceRequestGeneration;
    if (!isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)) {
      console.warn('[KnowledgeAdvancedConfigPage] skip stale source jobs driver', { knowledgeId: requestedKnowledgeId });
      return;
    }
    if (sourceJobPollingInFlightRef.current) {
      sourceJobDriveRequestedRef.current = true;
      return;
    }
    sourceJobPollingInFlightRef.current = true;
    try {
      const runRequestedCycles = async (): Promise<void> => {
        sourceJobDriveRequestedRef.current = false;
        const next = await loadKnowledgeSourceJobs({ silent: true });
        if (!next.loaded || !canAutoReconcileSourceJobs) return;
        if (next.reconcileRequired || next.jobs.some(isPollableSourceJob)) {
          sourceJobWorkObservedRef.current = true;
        }
        const reconciled = await reconcileCompletedSourceJobs(next.reconcileRequired, requestedKnowledgeId, requestedGeneration);
        const sourceJobsConverged =
          sourceJobWorkObservedRef.current && !next.reconcileRequired && !next.jobs.some(isPollableSourceJob);
        if (
          (reconciled || sourceJobsConverged || sourceRefreshRetryRequiredRef.current) &&
          isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)
        ) {
          const refreshed = await loadKnowledgeSources({
            page: sourcePageRef.current,
            silent: true,
            retryOnFailure: true,
          });
          if (refreshed.loaded) {
            if (sourceJobsConverged) {
              sourceJobWorkObservedRef.current = false;
            }
            await refreshOpenDocumentDetail();
          }
        }
        if (sourceJobDriveRequestedRef.current && isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)) {
          await runRequestedCycles();
        }
      };
      await runRequestedCycles();
    } catch (error) {
      console.warn('[KnowledgeAdvancedConfigPage] drive source jobs failed', {
        error: error instanceof Error ? error.message : String(error),
      });
    } finally {
      if (isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)) {
        sourceJobPollingInFlightRef.current = false;
      }
    }
  }, [
    canAutoReconcileSourceJobs,
    hasValidKnowledgeId,
    knowledgeId,
    isCurrentSourceRequest,
    loadKnowledgeSourceJobs,
    loadKnowledgeSources,
    reconcileCompletedSourceJobs,
    refreshOpenDocumentDetail,
    sourceRequestGeneration,
  ]);

  useEffect(() => {
    setSourceJobRows([]);
    setSourceJobsReconcileRequired(false);
    setSourceJobsRetryRequired(false);
    setSourceRefreshRetryRequired(false);
    sourceJobPollingInFlightRef.current = false;
    sourceJobDriveRequestedRef.current = false;
    sourceJobWorkObservedRef.current = false;
    sourceRefreshRetryRequiredRef.current = false;
    sourceRefreshPendingRef.current = false;
    sourceForegroundRequestSequenceRef.current = 0;
  }, [knowledgeId]);

  useEffect(() => {
    if (requestedSourceJobsKnowledgeIdRef.current === knowledgeId) return;
    requestedSourceJobsKnowledgeIdRef.current = knowledgeId;
    driveKnowledgeSourceJobs().catch((error: unknown) => {
      console.warn('[KnowledgeAdvancedConfigPage] start source jobs driver failed', {
        error: error instanceof Error ? error.message : String(error),
      });
    });
  }, [driveKnowledgeSourceJobs, knowledgeId]);

  useEffect(() => {
    if (!canAutoReconcileSourceJobs) return;
    if (
      !sourceJobsRetryRequired &&
      !sourceRefreshRetryRequired &&
      !sourceJobsReconcileRequired &&
      !sourceJobRows.some(isPollableSourceJob)
    ) {
      return;
    }
    const timer = window.setInterval(() => {
      driveKnowledgeSourceJobs().catch((error: unknown) => {
        console.warn('[KnowledgeAdvancedConfigPage] poll source jobs driver failed', {
          error: error instanceof Error ? error.message : String(error),
        });
      });
    }, SOURCE_JOB_POLL_INTERVAL_MS);
    return () => window.clearInterval(timer);
  }, [
    canAutoReconcileSourceJobs,
    driveKnowledgeSourceJobs,
    sourceJobRows,
    sourceJobsReconcileRequired,
    sourceJobsRetryRequired,
    sourceRefreshRetryRequired,
  ]);

  useEffect(() => {
    if (activeTab !== 'agents' || !hasValidKnowledgeId) return;
    if (requestedAgentKnowledgeIdRef.current === knowledgeId) return;
    loadAssociatedAgents();
  }, [activeTab, hasValidKnowledgeId, knowledgeId, loadAssociatedAgents]);

  const handleAppendSources = useCallback(
    async (sourcePayload: SemanticModelSourceSubmitPayload) => {
      if (!hasValidKnowledgeId) return;
      const requestedKnowledgeId = knowledgeId;
      const requestedGeneration = sourceRequestGeneration;
      try {
        setSourceAppending(true);
        await appendKnowledgeSources(http, {
          id: requestedKnowledgeId,
          sources: sourcePayload.sources,
          source_selections: sourcePayload.source_selections,
        });
        if (!isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)) {
          console.warn('[KnowledgeAdvancedConfigPage] skip stale append completion', {
            knowledgeId: requestedKnowledgeId,
          });
          return;
        }
        driveKnowledgeSourceJobs().catch((error: unknown) => {
          console.warn('[KnowledgeAdvancedConfigPage] drive source jobs after append failed', {
            error: error instanceof Error ? error.message : String(error),
          });
        });
        const refreshed = await loadKnowledgeSources();
        if (refreshed.loaded) {
          await loadKnowledgeDetail();
          setSourceSelectOpen(false);
        }
      } catch (error) {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.append-source-failed'));
        console.error('[KnowledgeAdvancedConfigPage] append sources failed', error);
      } finally {
        setSourceAppending(false);
      }
    },
    [
      driveKnowledgeSourceJobs,
      hasValidKnowledgeId,
      http,
      isCurrentSourceRequest,
      knowledgeId,
      loadKnowledgeDetail,
      loadKnowledgeSources,
      message,
      sourceRequestGeneration,
    ],
  );

  const getSourceTypeText = useCallback(
    (sourceType: SemanticModelSourceType) =>
      t(
        sourceType === 'file'
          ? 'knowledge.base.source-item-type-file'
          : sourceType === 'volume'
            ? 'knowledge.base.source-item-type-volume'
            : 'knowledge.base.source-item-type-table',
      ),
    [t],
  );

  const mergeDocumentDetailFromSource = useCallback(
    (currentDetail: SemanticModelSourceDocument, source: SemanticModelSource): SemanticModelSourceDocument => ({
      ...currentDetail,
      source,
      file_info: {
        ...currentDetail.file_info,
        tags: Array.isArray(source.tags) ? source.tags : [],
        expires_at: source.expires_at,
        enabled: source.enabled,
        expired: source.expired ?? false,
        effective_enabled: source.effective_enabled ?? true,
        force_enabled_after_expiry: source.force_enabled_after_expiry ?? false,
        index_version: source.index_version ?? null,
        segment_version_id: source.segment_version_id,
      },
    }),
    [],
  );
  const applySegmentMutationDocument = useCallback(
    (nextDocument: SemanticModelSourceDocument) => {
      setDocumentDetail(nextDocument);
      syncDocumentDraft(nextDocument);
      setSourceRows((currentRows) =>
        currentRows.map((row) => (row.row_id === nextDocument.source.row_id ? nextDocument.source : row)),
      );
    },
    [syncDocumentDraft],
  );

  const resetSegmentViewState = useCallback(() => {
    setSegmentSearch('');
    setSegmentSort('source_order');
    setSegmentPage(1);
    setEditingSegmentId(null);
    setPlayingSegmentId(null);
    setNewSegmentOpen(false);
    setNewSegmentContent('');
  }, []);

  const currentSegmentBase = useCallback((detail: SemanticModelSourceDocument) => {
    return {
      base_segment_version_id: detail.current_segment_version_id ?? detail.file_info.segment_version_id ?? '',
      base_index_version: detail.current_index_version ?? detail.file_info.index_version ?? 0,
    };
  }, []);

  const requireCurrentSegmentBase = useCallback(
    (detail: SemanticModelSourceDocument) => {
      const base = currentSegmentBase(detail);
      if (!base || !base.base_segment_version_id || !base.base_index_version) {
        message.error(t('knowledge.base.document-segment-current-required'));
        return null;
      }
      return base;
    },
    [currentSegmentBase, message, t],
  );

  const loadDocumentVersion = useCallback(
    async (versionId: string) => {
      if (!hasValidKnowledgeId || !documentDetail) return;
      try {
        setDocumentLoading(true);
        const nextDetail = await getKnowledgeSourceDocument(http, {
          id: knowledgeId,
          sourceRowId: documentDetail.source.row_id,
          segmentVersionId: versionId,
        });
        setDocumentDetail(nextDetail);
        syncDocumentDraft(nextDetail);
        resetSegmentViewState();
      } catch (error) {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-version-load-failed'));
        console.error('[KnowledgeAdvancedConfigPage] load document segment version failed', error);
      } finally {
        setDocumentLoading(false);
      }
    },
    [documentDetail, hasValidKnowledgeId, http, knowledgeId, message, resetSegmentViewState, syncDocumentDraft],
  );

  const handleSetCurrentSegmentVersion = useCallback(
    async (versionId: string) => {
      if (!hasValidKnowledgeId || !documentDetail || !canAppendSources) return;
      const base = currentSegmentBase(documentDetail);
      try {
        setSegmentMutatingKey(`set-current:${versionId}`);
        const result = await setCurrentKnowledgeSegmentVersion(http, {
          id: knowledgeId,
          sourceRowId: documentDetail.source.row_id,
          versionId,
          ...base,
        });
        applySegmentMutationDocument(result.document);
        message.success(t('knowledge.base.document-version-set-current-success'));
      } catch (error) {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-version-set-current-failed'));
        console.error('[KnowledgeAdvancedConfigPage] set current segment version failed', error);
      } finally {
        setSegmentMutatingKey(null);
      }
    },
    [
      applySegmentMutationDocument,
      canAppendSources,
      currentSegmentBase,
      documentDetail,
      hasValidKnowledgeId,
      http,
      knowledgeId,
      message,
      t,
    ],
  );

  const handleImportInitialSegments = useCallback(async () => {
    if (!hasValidKnowledgeId || !documentDetail || !canAppendSources) return;
    const base = currentSegmentBase(documentDetail);
    try {
      setSegmentMutatingKey('import-initial');
      const result = await importInitialKnowledgeSegments(http, {
        id: knowledgeId,
        sourceRowId: documentDetail.source.row_id,
        ...base,
      });
      applySegmentMutationDocument(result.document);
      message.success(t('knowledge.base.document-segment-import-success'));
    } catch (error) {
      message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-segment-import-failed'));
      console.error('[KnowledgeAdvancedConfigPage] import initial segments failed', error);
    } finally {
      setSegmentMutatingKey(null);
    }
  }, [
    applySegmentMutationDocument,
    canAppendSources,
    currentSegmentBase,
    documentDetail,
    hasValidKnowledgeId,
    http,
    knowledgeId,
    message,
    t,
  ]);

  const closeSegmentPreview = useCallback(() => {
    setSegmentPreview((current) => {
      if (current.revoke && current.url) {
        URL.revokeObjectURL(current.url);
      }
      return { open: false, url: '', revoke: false, title: '' };
    });
  }, []);

  const openSegmentImagePreview = useCallback(
    async (segment: SemanticModelDocumentSegment) => {
      if (!documentDetail) return;
      const fileId = getSegmentImageFileId(segment);
      if (!fileId) return;
      try {
        setSegmentPreviewLoadingKey(segment.segment_id);
        const result = await previewSemanticModelArtifactApi(knowledgeId, fileId, http, {
          responseType: 'blob',
          responseContentType: 'blob',
        });
        const response = result as { data?: Blob | { link?: string } };
        const payload = response.data;
        let nextUrl = '';
        let revoke = false;
        if (payload instanceof Blob) {
          nextUrl = URL.createObjectURL(payload);
          revoke = true;
        } else if (payload && typeof payload === 'object' && typeof payload.link === 'string') {
          nextUrl = payload.link;
        }
        if (!nextUrl) {
          message.error(t('knowledge.base.document-segment-image-preview-failed'));
          return;
        }
        setSegmentPreview((current) => {
          if (current.revoke && current.url) {
            URL.revokeObjectURL(current.url);
          }
          return {
            open: true,
            url: nextUrl,
            revoke,
            title: t('knowledge.base.document-segment-image-preview'),
          };
        });
      } catch (error) {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-segment-image-preview-failed'));
        console.error('[KnowledgeAdvancedConfigPage] preview segment image failed', error);
      } finally {
        setSegmentPreviewLoadingKey(null);
      }
    },
    [documentDetail, http, knowledgeId, message, t],
  );

  const handleToggleSegmentEnabled = useCallback(
    async (segment: SemanticModelDocumentSegment, enabled: boolean) => {
      if (!hasValidKnowledgeId || !documentDetail || !canAppendSources) return;
      const base = requireCurrentSegmentBase(documentDetail);
      if (!base) return;
      try {
        setSegmentMutatingKey(`enabled:${segment.segment_id}`);
        const result = await updateKnowledgeSegmentEnabled(http, {
          id: knowledgeId,
          sourceRowId: documentDetail.source.row_id,
          segmentId: segment.segment_id,
          enabled,
          ...base,
        });
        applySegmentMutationDocument(result.document);
        message.success(t('knowledge.base.document-segment-save-success'));
      } catch (error) {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-segment-save-failed'));
        console.error('[KnowledgeAdvancedConfigPage] update segment enabled failed', error);
      } finally {
        setSegmentMutatingKey(null);
      }
    },
    [
      applySegmentMutationDocument,
      canAppendSources,
      documentDetail,
      hasValidKnowledgeId,
      http,
      knowledgeId,
      message,
      requireCurrentSegmentBase,
      t,
    ],
  );

  const handleDeleteSegment = useCallback(
    async (segment: SemanticModelDocumentSegment) => {
      if (!hasValidKnowledgeId || !documentDetail || !canAppendSources) return;
      const base = requireCurrentSegmentBase(documentDetail);
      if (!base) return;
      try {
        setSegmentMutatingKey(`delete:${segment.segment_id}`);
        const result = await deleteKnowledgeSegment(http, {
          id: knowledgeId,
          sourceRowId: documentDetail.source.row_id,
          segmentId: segment.segment_id,
          ...base,
        });
        applySegmentMutationDocument(result.document);
        setEditingSegmentId((current) => (current === segment.segment_id ? null : current));
        resetSegmentViewState();
        message.success(t('knowledge.base.document-segment-delete-success'));
      } catch (error) {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-segment-delete-failed'));
        console.warn('[KnowledgeAdvancedConfigPage] delete segment failed', {
          message: error instanceof Error ? error.message : String(error),
          knowledgeId,
          rowId: documentDetail.source.row_id,
          segmentId: segment.segment_id,
        });
      } finally {
        setSegmentMutatingKey(null);
      }
    },
    [
      applySegmentMutationDocument,
      canAppendSources,
      documentDetail,
      hasValidKnowledgeId,
      http,
      knowledgeId,
      message,
      requireCurrentSegmentBase,
      resetSegmentViewState,
      t,
    ],
  );

  const handleUpdateSegmentText = useCallback(
    async (segment: SemanticModelDocumentSegment, draft: SegmentDraft, includeMediaFields: boolean) => {
      if (!hasValidKnowledgeId || !documentDetail || !canAppendSources) return;
      const base = requireCurrentSegmentBase(documentDetail);
      if (!base) return;
      try {
        setSegmentMutatingKey(`edit:${segment.segment_id}`);
        const result = await updateKnowledgeSegment(http, {
          id: knowledgeId,
          sourceRowId: documentDetail.source.row_id,
          segmentId: segment.segment_id,
          ...base,
          content: draft.content ?? '',
          ...(includeMediaFields
            ? {
                ocr_text: draft.ocr_text ?? '',
                image_description: draft.image_description ?? '',
              }
            : {}),
        });
        applySegmentMutationDocument(result.document);
        setEditingSegmentId(null);
        message.success(t('knowledge.base.document-segment-save-success'));
      } catch (error) {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-segment-save-failed'));
        console.error('[KnowledgeAdvancedConfigPage] update segment text failed', error);
      } finally {
        setSegmentMutatingKey(null);
      }
    },
    [
      applySegmentMutationDocument,
      canAppendSources,
      documentDetail,
      hasValidKnowledgeId,
      http,
      knowledgeId,
      message,
      requireCurrentSegmentBase,
      t,
    ],
  );

  const cancelSegmentEdit = useCallback((segment: SemanticModelDocumentSegment) => {
    setSegmentDrafts((current) => ({
      ...current,
      [segment.segment_id]: {
        content: segment.content ?? '',
        ocr_text: segment.ocr_text ?? '',
        image_description: segment.image_description ?? '',
      },
    }));
    setEditingSegmentId(null);
  }, []);

  const handleCreateSegment = useCallback(async () => {
    if (!hasValidKnowledgeId || !documentDetail || !canAppendSources) return;
    const base = requireCurrentSegmentBase(documentDetail);
    if (!base) return;
    if (newSegmentContent === '') {
      message.error(t('knowledge.base.document-segment-create-empty'));
      return;
    }
    try {
      setSegmentMutatingKey('create');
      const result = await createKnowledgeSegment(http, {
        id: knowledgeId,
        sourceRowId: documentDetail.source.row_id,
        level: 'chunk',
        content: newSegmentContent,
        ...base,
      });
      setNewSegmentContent('');
      setNewSegmentOpen(false);
      applySegmentMutationDocument(result.document);
      resetSegmentViewState();
      message.success(t('knowledge.base.document-segment-create-success'));
    } catch (error) {
      message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-segment-create-failed'));
      console.error('[KnowledgeAdvancedConfigPage] create segment failed', error);
    } finally {
      setSegmentMutatingKey(null);
    }
  }, [
    applySegmentMutationDocument,
    canAppendSources,
    documentDetail,
    hasValidKnowledgeId,
    http,
    knowledgeId,
    message,
    newSegmentContent,
    requireCurrentSegmentBase,
    resetSegmentViewState,
    t,
  ]);

  const openDocumentDetail = useCallback(
    async (entry: SemanticModelSource) => {
      if (!hasValidKnowledgeId || entry.source_type === 'table' || isLegacyUnboundSource(entry)) return;

      setDocumentOpen(true);
      setDocumentTab('preview');
      setDocumentLoading(true);
      setDocumentDetail(null);
      resetSegmentViewState();
      setNewSegmentContent('');
      try {
        const nextDetail = await getKnowledgeSourceDocument(http, { id: knowledgeId, sourceRowId: entry.row_id });
        setDocumentDetail(nextDetail);
        syncDocumentDraft(nextDetail);
      } catch (error) {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-detail-load-failed'));
        console.error('[KnowledgeAdvancedConfigPage] load document detail failed', error);
      } finally {
        setDocumentLoading(false);
      }
    },
    [hasValidKnowledgeId, http, knowledgeId, message, resetSegmentViewState, syncDocumentDraft],
  );

  const closeDocumentDetail = useCallback(() => {
    if (documentSaving) return;
    setDocumentOpen(false);
    setDocumentDetail(null);
    setDocumentTab('preview');
    resetSegmentViewState();
    setNewSegmentContent('');
  }, [documentSaving, resetSegmentViewState]);

  const getSourceProcessStatus = useCallback((entry: SemanticModelSource) => {
    if (isLegacyUnboundSource(entry)) {
      return {
        status: 'legacy_unbound',
        error: null,
      };
    }
    return {
      status: entry.ingest_status || 'unknown',
      error: entry.error || null,
    };
  }, []);
  const sourceProcessStatusByRowId = useMemo(() => {
    const result: Record<string, { status: string | null; error: string | null; ready: boolean; failed: boolean }> = {};
    sourceRows.forEach((entry) => {
      const process = getSourceProcessStatus(entry);
      result[entry.row_id] = {
        ...process,
        ready: isCompletedProcessStatus(process.status),
        failed: isFailedProcessStatus(process.status),
      };
    });
    return result;
  }, [getSourceProcessStatus, sourceRows]);
  const isSourceProcessReady = useCallback(
    (entry: SemanticModelSource) => sourceProcessStatusByRowId[entry.row_id]?.ready === true,
    [sourceProcessStatusByRowId],
  );
  const isSourceProcessFailed = useCallback(
    (entry: SemanticModelSource) => sourceProcessStatusByRowId[entry.row_id]?.failed === true,
    [sourceProcessStatusByRowId],
  );
  const canOperateSource = useCallback(
    (entry: SemanticModelSource) => !isLegacyUnboundSource(entry) && isSourceProcessReady(entry),
    [isSourceProcessReady],
  );
  const canDeleteSource = useCallback(
    (entry: SemanticModelSource) =>
      !isLegacyUnboundSource(entry) && (isSourceProcessReady(entry) || isSourceProcessFailed(entry)),
    [isSourceProcessFailed, isSourceProcessReady],
  );
  const isSourceSwitchChecked = useCallback(
    (entry: SemanticModelSource) =>
      !isLegacyUnboundSource(entry) &&
      isSourceProcessReady(entry) &&
      entry.effective_enabled !== false &&
      entry.enabled !== false,
    [isSourceProcessReady],
  );

  const handleDeleteSource = useCallback(
    (entry: SemanticModelSource) => {
      if (!hasValidKnowledgeId || !canAppendSources || !canDeleteSource(entry)) return;
      const requestedKnowledgeId = knowledgeId;
      const requestedGeneration = sourceRequestGeneration;

      modal.confirm({
        title: t('knowledge.base.delete-source-confirm-title'),
        content: t('knowledge.base.delete-source-confirm-content', {
          name: getSourceDisplayName(entry),
          type: getSourceTypeText(entry.source_type),
        }),
        okText: t('knowledge.base.delete-source-confirm-ok'),
        cancelText: t('knowledge.base.delete-source-confirm-cancel'),
        okButtonProps: { danger: true },
        onOk: async () => {
          try {
            setDeletingSourceRowId(entry.row_id);
            await deleteKnowledgeSource(http, { id: requestedKnowledgeId, sourceRowId: entry.row_id });
            if (!isCurrentSourceRequest(requestedKnowledgeId, requestedGeneration)) {
              console.warn('[KnowledgeAdvancedConfigPage] skip stale delete completion', {
                knowledgeId: requestedKnowledgeId,
                rowId: entry.row_id,
              });
              return;
            }
            message.success(t('knowledge.base.delete-source-success'));
            if (documentDetail?.source.row_id === entry.row_id) {
              setDocumentOpen(false);
              setDocumentDetail(null);
              setDocumentTab('preview');
            }
            const refreshed = await loadKnowledgeSources();
            const maxPage = Math.max(1, Math.ceil(refreshed.total / refreshed.pageSize));
            if (refreshed.loaded && refreshed.page > maxPage) {
              await loadKnowledgeSources({ page: maxPage });
            }
            await loadKnowledgeDetail();
            driveKnowledgeSourceJobs().catch((error: unknown) => {
              console.warn('[KnowledgeAdvancedConfigPage] drive source jobs after delete failed', {
                error: error instanceof Error ? error.message : String(error),
              });
            });
          } catch (error) {
            message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.delete-source-failed'));
            console.error('[KnowledgeAdvancedConfigPage] delete source failed', error);
            throw error;
          } finally {
            setDeletingSourceRowId(null);
          }
        },
      });
    },
    [
      canAppendSources,
      canDeleteSource,
      documentDetail,
      driveKnowledgeSourceJobs,
      getSourceTypeText,
      hasValidKnowledgeId,
      http,
      isCurrentSourceRequest,
      knowledgeId,
      loadKnowledgeSources,
      loadKnowledgeDetail,
      message,
      modal,
      sourceRequestGeneration,
      t,
    ],
  );

  const handleToggleSourceEnabled = useCallback(
    async (entry: SemanticModelSource, enabled: boolean) => {
      if (!hasValidKnowledgeId || !canAppendSources || governanceUpdatingSourceRowId) return;
      if (isLegacyUnboundSource(entry)) return;
      if (!isSourceProcessReady(entry)) return;

      const previousRows = sourceRows;
      const optimisticSource: SemanticModelSource = {
        ...entry,
        enabled,
        effective_enabled: enabled && (!entry.expired || Boolean(entry.force_enabled_after_expiry)),
      };
      setSourceRows(previousRows.map((row) => (row.row_id === entry.row_id ? optimisticSource : row)));

      try {
        setGovernanceUpdatingSourceRowId(entry.row_id);
        const result = await updateKnowledgeSourceGovernance(http, {
          id: knowledgeId,
          sourceRowId: entry.row_id,
          enabled,
        });
        setSourceRows((currentRows) => currentRows.map((row) => (row.row_id === result.source.row_id ? result.source : row)));
        if (documentDetail?.source.row_id === result.source.row_id) {
          const nextDetail = mergeDocumentDetailFromSource(documentDetail, result.source);
          setDocumentDetail(nextDetail);
          syncDocumentDraft(nextDetail);
        }
        message.success(t('knowledge.base.source-governance-update-success'));
      } catch (error) {
        setSourceRows(previousRows);
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.source-governance-update-failed'));
        console.error('[KnowledgeAdvancedConfigPage] update source governance failed', error);
      } finally {
        setGovernanceUpdatingSourceRowId(null);
      }
    },
    [
      canAppendSources,
      documentDetail,
      governanceUpdatingSourceRowId,
      hasValidKnowledgeId,
      http,
      isSourceProcessReady,
      knowledgeId,
      message,
      mergeDocumentDetailFromSource,
      sourceRows,
      syncDocumentDraft,
      t,
    ],
  );

  const applyDocumentSourceUpdate = useCallback(
    (source: SemanticModelSource) => {
      setSourceRows((currentRows) => currentRows.map((row) => (row.row_id === source.row_id ? source : row)));
      if (documentDetail?.source.row_id !== source.row_id) return;
      const nextDetail = mergeDocumentDetailFromSource(documentDetail, source);
      setDocumentDetail(nextDetail);
      syncDocumentDraft(nextDetail);
    },
    [documentDetail, mergeDocumentDetailFromSource, syncDocumentDraft],
  );

  const saveDocumentTags = useCallback(async () => {
    if (!hasValidKnowledgeId || !documentDetail) return;

    try {
      setDocumentTagsSaving(true);
      const result = await updateKnowledgeSourceGovernance(http, {
        id: knowledgeId,
        sourceRowId: documentDetail.source.row_id,
        tags: draftTags,
      });
      applyDocumentSourceUpdate(result.source);
      message.success(t('knowledge.base.document-tags-save-success'));
    } catch (error) {
      message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-tags-save-failed'));
      console.error('[KnowledgeAdvancedConfigPage] save document tags failed', error);
    } finally {
      setDocumentTagsSaving(false);
    }
  }, [applyDocumentSourceUpdate, documentDetail, draftTags, hasValidKnowledgeId, http, knowledgeId, message, t]);

  const saveDocumentExpiresAt = useCallback(async () => {
    if (!hasValidKnowledgeId || !documentDetail) return;

    try {
      setDocumentExpirySaving(true);
      const result = await updateKnowledgeSourceGovernance(http, {
        id: knowledgeId,
        sourceRowId: documentDetail.source.row_id,
        expires_at: draftExpiresAt,
      });
      applyDocumentSourceUpdate(result.source);
      message.success(t('knowledge.base.document-expires-save-success'));
    } catch (error) {
      message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.document-expires-save-failed'));
      console.error('[KnowledgeAdvancedConfigPage] save document expiry failed', error);
    } finally {
      setDocumentExpirySaving(false);
    }
  }, [applyDocumentSourceUpdate, documentDetail, draftExpiresAt, hasValidKnowledgeId, http, knowledgeId, message, t]);

  useEffect(() => {
    if (!hasValidKnowledgeId) {
      const invalidKnowledgeIdKey = routeKnowledgeId ?? '__missing__';
      if (invalidKnowledgeIdRef.current === invalidKnowledgeIdKey) {
        return;
      }
      invalidKnowledgeIdRef.current = invalidKnowledgeIdKey;
      console.warn('[KnowledgeAdvancedConfigPage] invalid knowledge id in edit mode', { routeKnowledgeId });
      message.error(t('knowledge.base.invalid-id'));
      handleBack();
      return;
    }

    if (requestedDetailKnowledgeIdRef.current !== knowledgeId) {
      requestedDetailKnowledgeIdRef.current = knowledgeId;
      loadKnowledgeDetail();
    }
    if (requestedSourceKnowledgeIdRef.current !== knowledgeId) {
      setSourcePagination({ page: 1, pageSize: SOURCE_PAGE_SIZE, total: 0 });
      requestKnowledgeSources(1);
    }
  }, [handleBack, hasValidKnowledgeId, knowledgeId, loadKnowledgeDetail, message, requestKnowledgeSources, routeKnowledgeId, t]);

  const handleSourcePageChange = useCallback(
    (page: number) => {
      if (!hasValidKnowledgeId || page === sourcePagination.page) return;
      requestKnowledgeSources(page);
    },
    [hasValidKnowledgeId, requestKnowledgeSources, sourcePagination.page],
  );
  const sourcePageCount = Math.max(1, Math.ceil(sourcePagination.total / sourcePagination.pageSize));
  const sourceCurrentPage = Math.min(sourcePagination.page, sourcePageCount);
  const sourceSummaryCounts = detail?.source_counts ?? {
    files: sourceRows.filter((entry) => entry.source_type !== 'table').length,
    tables: sourceRows.filter((entry) => entry.source_type === 'table').length,
    total: sourceRows.length,
  };
  const enableAgentVirtualScroll = associatedAgentRows.length > 80;

  const renderHeaderInfo = () => {
    if (!detail) {
      return <Text type="secondary">{t('knowledge.base.loading')}</Text>;
    }

    return (
      <>
        <Title level={4} className={styles.headerTitle}>
          {detail.name}
        </Title>
        <Paragraph className={styles.headerParagraph} type="secondary">
          {detail.description || '-'}
        </Paragraph>
      </>
    );
  };

  const formatCatalogPath = (parts: string[]) => {
    const nonEmptyParts = parts.filter(Boolean);
    return nonEmptyParts.length > 0 ? nonEmptyParts.join('/') : '-';
  };
  const getCatalogPath = (entry: SemanticModelSource) => {
    if (Array.isArray(entry.path) && entry.path.length > 0) {
      return formatCatalogPath(entry.path);
    }
    if (entry.source_path) {
      return formatCatalogPath(entry.source_path.split('/'));
    }
    if (entry.source_type === 'table') {
      return entry.db_name ? formatCatalogPath([entry.db_name]) : '-';
    }
    return '-';
  };
  const getSourceSqlDatabaseName = (entry: SemanticModelSource) => {
    return entry.db_name ?? '';
  };
  const openSourceSqlEditor = (entry: SemanticModelSource) => {
    if (!canOperateSource(entry)) return;
    const sourceDb = getSourceSqlDatabaseName(entry);
    const sourceTable = entry.table_name ?? '';
    if (!sourceDb || !sourceTable) {
      console.warn('[KnowledgeAdvancedConfigPage] table source missing SQL editor context', {
        rowId: entry.row_id,
        hasSourceDb: !!sourceDb,
        hasSourceTable: !!sourceTable,
      });
      return;
    }
    const openSqlEditor = nav.callbacks.onOpenSqlEditor as
      | ((payload: { sourceDb: string; sourceTable: string }) => void)
      | undefined;
    if (!openSqlEditor) {
      console.warn('[KnowledgeAdvancedConfigPage] onOpenSqlEditor callback is not available');
      return;
    }
    openSqlEditor({ sourceDb, sourceTable });
  };
  const getSourceTypeLabel = (entry: SemanticModelSource) => {
    if (entry.source_type === 'table') return 'TABLE';
    const name = (entry.display_name || entry.resource_id).toLowerCase();
    if (name.endsWith('.pdf')) return 'PDF';
    if (name.endsWith('.doc') || name.endsWith('.docx')) return 'Word';
    if (name.endsWith('.md') || name.endsWith('.markdown')) return 'Markdown';
    if (name.endsWith('.txt')) return 'Text';
    return getSourceTypeText(entry.source_type);
  };
  const formatFileSize = (bytes: number | null | undefined) => {
    if (bytes === null || bytes === undefined) return '-';
    if (bytes < 0) {
      console.warn('[KnowledgeAdvancedConfigPage] invalid negative source size', { bytes });
      return '-';
    }
    if (bytes === 0) return '0 B';
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
    const value = bytes / 1024 ** unitIndex;
    const displayValue = unitIndex === 0 ? String(value) : value.toFixed(value >= 10 ? 1 : 2);
    return `${displayValue} ${units[unitIndex]}`;
  };
  const getSourceSizeText = (entry: SemanticModelSource) => {
    if (entry.source_type === 'table') {
      return entry.row_count === null || entry.row_count === undefined ? '-' : String(entry.row_count);
    }
    return formatFileSize(entry.size_bytes);
  };
  const getSourceUpdatedText = (entry: SemanticModelSource) => formatOptionalTimestamp(entry.updated_at);
  const canDownloadSource = (entry: SemanticModelSource) => entry.source_type === 'file' || entry.source_type === 'table';
  const canEditSourceExpiry = (entry: SemanticModelSource) => entry.source_type === 'file' || entry.source_type === 'table';
  const formatOptionalTimestamp = (timestamp: number | null | undefined) =>
    timestamp ? formatDateTime(timestamp * 1000, { timezone }) : '-';
  const getAgentDescription = (agent: AgentRecord) => agent.description || agent.desc || '-';
  const getAgentUpdatedText = (agent: AgentRecord) => {
    if (typeof agent.updated_at === 'number') {
      return formatOptionalTimestamp(agent.updated_at);
    }
    if (typeof agent.updated_at === 'string') {
      const parsed = Date.parse(agent.updated_at);
      return Number.isFinite(parsed) ? formatDateTime(parsed, { timezone }) : agent.updated_at;
    }
    return '-';
  };
  const resolveAgentStatusKey = (status: AgentRecord['status']) => {
    switch (status) {
      case 'active':
      case 'running':
        return 'knowledge.base.agent-association-status-active';
      case 'disabled':
      case 'stopped':
        return 'knowledge.base.agent-association-status-disabled';
      case 'draft':
        return 'knowledge.base.agent-association-status-draft';
      default:
        return 'knowledge.base.agent-association-status-unknown';
    }
  };
  const resolveAgentStatusClassName = (status: AgentRecord['status']) => {
    switch (status) {
      case 'active':
      case 'running':
        return styles.processStatusSuccess;
      case 'disabled':
      case 'stopped':
        return styles.processStatusNeutral;
      case 'draft':
        return styles.processStatusProcessing;
      default:
        return styles.processStatusNeutral;
    }
  };
  const formatSegmentVersionTitle = (version: SemanticModelSegmentVersion, ordinalByVersionID: Map<string, number>): string => {
    const ordinal = ordinalByVersionID.get(version.version_id) ?? version.index_version ?? '-';
    switch (version.source) {
      case 'initial_import':
        return t('knowledge.base.document-version-title-initial-import', { version: ordinal });
      case 'create_chunk':
        return t('knowledge.base.document-version-title-create-segment', { version: ordinal });
      case 'edit_chunk':
        return t('knowledge.base.document-version-title-edit-segment', { version: ordinal });
      case 'disable_chunk':
        return t('knowledge.base.document-version-title-toggle-segment', { version: ordinal });
      case 'delete_chunk':
        return t('knowledge.base.document-version-title-delete-segment', { version: ordinal });
      case 'reembed':
        return t('knowledge.base.document-version-title-reembed', { version: ordinal });
      case 'external_workflow':
        return t('knowledge.base.document-version-title-external-workflow', { version: ordinal });
      default:
        return t('knowledge.base.document-version-title-fallback', { version: ordinal });
    }
  };
  const formatSegmentVersionSwitcherTitle = (
    version: SemanticModelSegmentVersion | undefined,
    ordinalByVersionID: Map<string, number>,
  ): string => {
    if (!version) return t('knowledge.base.document-version-empty');
    const title = formatSegmentVersionTitle(version, ordinalByVersionID);
    return version.current ? t('knowledge.base.document-version-title-current', { title }) : title;
  };
  const getVersionOperatorName = (version: SemanticModelSegmentVersion) => {
    const operatorID = getVersionOperatorId(version);
    return operatorID || t('knowledge.base.document-version-unknown-user');
  };
  const getTableCreatedByDisplayName = (source: SemanticModelSource, info: CatalogTableInfo) => {
    const createdBy = source.created_by || info.created_by;
    if (!createdBy) return '-';
    return createdBy;
  };
  const resolveFileDownloadId = (entry: SemanticModelSource) =>
    entry.kb_file_id ?? entry.kb_resource_id ?? entry.source_file_id ?? entry.source_resource_id ?? null;
  const resolveTableDownloadId = (entry: SemanticModelSource) => {
    const value = entry.kb_table_id ?? entry.kb_resource_id ?? entry.source_table_id ?? entry.source_resource_id ?? null;
    if (value === null || value === undefined || value === '') return null;
    const tableID = typeof value === 'number' ? value : Number(value);
    return Number.isInteger(tableID) && tableID > 0 ? tableID : null;
  };
  const resolveTableDetailId = (entry: SemanticModelSource) => resolveTableDownloadId(entry);
  const formatDateInputValue = (timestamp: number | null) => {
    if (!timestamp) return '';
    const value = new Date(timestamp * 1000);
    const year = value.getFullYear();
    const month = String(value.getMonth() + 1).padStart(2, '0');
    const day = String(value.getDate()).padStart(2, '0');
    return `${year}-${month}-${day}`;
  };
  const handleExpiresDateChange = (value: string) => {
    if (!value) {
      setDraftExpiresAt(null);
      return;
    }
    const expiresAt = Math.floor(new Date(`${value}T23:59:59`).getTime() / 1000);
    if (!Number.isFinite(expiresAt) || expiresAt <= 0) {
      message.error(t('knowledge.base.document-info-expires-invalid'));
      return;
    }
    setDraftExpiresAt(expiresAt);
  };
  const handleSourceExpiryDateChange = (value: string) => {
    if (!value) {
      setExpiryDraftExpiresAt(null);
      return;
    }
    const expiresAt = Math.floor(new Date(`${value}T23:59:59`).getTime() / 1000);
    if (!Number.isFinite(expiresAt) || expiresAt <= 0) {
      message.error(t('knowledge.base.document-info-expires-invalid'));
      return;
    }
    setExpiryDraftExpiresAt(expiresAt);
  };
  const openSourceExpiryModal = (entry: SemanticModelSource) => {
    if (!canAppendSources || !canOperateSource(entry)) return;
    setExpirySource(entry);
    setExpiryDraftExpiresAt(entry.expires_at ?? null);
  };
  const closeSourceExpiryModal = () => {
    if (expirySaving) return;
    setExpirySource(null);
    setExpiryDraftExpiresAt(null);
  };
  const saveSourceExpiry = useCallback(async () => {
    if (!hasValidKnowledgeId || !expirySource) return;

    try {
      setExpirySaving(true);
      const result = await updateKnowledgeSourceGovernance(http, {
        id: knowledgeId,
        sourceRowId: expirySource.row_id,
        expires_at: expiryDraftExpiresAt,
      });
      setSourceRows((currentRows) => currentRows.map((row) => (row.row_id === result.source.row_id ? result.source : row)));
      if (documentDetail?.source.row_id === result.source.row_id) {
        const nextDetail = mergeDocumentDetailFromSource(documentDetail, result.source);
        setDocumentDetail(nextDetail);
        syncDocumentDraft(nextDetail);
      }
      setExpirySource(result.source);
      setExpiryDraftExpiresAt(result.source.expires_at ?? null);
      message.success(t('knowledge.base.source-expiry-save-success'));
    } catch (error) {
      message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.source-expiry-save-failed'));
      console.error('[KnowledgeAdvancedConfigPage] save source expiry failed', error);
    } finally {
      setExpirySaving(false);
    }
  }, [
    documentDetail,
    expiryDraftExpiresAt,
    expirySource,
    hasValidKnowledgeId,
    http,
    knowledgeId,
    mergeDocumentDetailFromSource,
    message,
    syncDocumentDraft,
    t,
  ]);
  const handleDownloadSource = useCallback(
    async (entry: SemanticModelSource) => {
      if (!canOperateSource(entry)) return;
      try {
        setDownloadingSourceRowId(entry.row_id);
        if (entry.source_type === 'table') {
          const tableId = resolveTableDownloadId(entry);
          if (tableId === null) {
            throw new Error(t('knowledge.base.source-download-missing-id'));
          }
          const response = await downloadTableApi(tableId, http);
          const disposition = response.headers?.['content-disposition'] || response.headers?.['Content-Disposition'] || '';
          const blob = response.data instanceof Blob ? response.data : new Blob([response.data as unknown as BlobPart]);
          downloadBlob(blob, {
            disposition,
            fallbackName: getSourceDisplayName(entry),
            fallbackExtension: 'csv',
          });
        } else {
          const fileId = resolveFileDownloadId(entry);
          if (!fileId) {
            throw new Error(t('knowledge.base.source-download-missing-id'));
          }
          const response = await originFileDownloadApi({ file_id: fileId }, http);
          const disposition = response.headers?.['content-disposition'] || response.headers?.['Content-Disposition'] || '';
          downloadBlob(response.data, {
            disposition,
            fallbackName: getSourceDisplayName(entry),
          });
        }
        message.success(t('knowledge.base.source-download-success'));
      } catch (error) {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.source-download-failed'));
        console.error('[KnowledgeAdvancedConfigPage] download source failed', error);
      } finally {
        setDownloadingSourceRowId(null);
      }
    },
    [canOperateSource, http, message, t],
  );
  const openTableDetail = useCallback(
    async (entry: SemanticModelSource) => {
      if (entry.source_type !== 'table' || !canOperateSource(entry)) return;
      const tableId = resolveTableDetailId(entry);
      if (tableId === null) {
        console.warn('[KnowledgeAdvancedConfigPage] table source missing detail id', { rowId: entry.row_id });
        message.error(t('knowledge.base.source-table-detail-load-failed'));
        return;
      }

      setTableDetailOpen(true);
      setTableDetailLoading(true);
      setTableDetail(null);
      try {
        const [infoResponse, sampleResponse] = await Promise.all([
          getTableInfoApi({ id: tableId }, http),
          getTableSampleDataApi({ id: tableId, page: 1, page_size: TABLE_SAMPLE_PAGE_SIZE }, http),
        ]);
        if (!isSuccessApiCode(infoResponse.code) || !infoResponse.data) {
          throw new Error(infoResponse.msg || t('knowledge.base.source-table-detail-load-failed'));
        }
        if (!isSuccessApiCode(sampleResponse.code) || !sampleResponse.data) {
          throw new Error(sampleResponse.msg || t('knowledge.base.source-table-detail-load-failed'));
        }

        setTableDetail({ source: entry, info: infoResponse.data, sample: sampleResponse.data });
      } catch (error) {
        setTableDetailOpen(false);
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.source-table-detail-load-failed'));
        console.error('[KnowledgeAdvancedConfigPage] load table detail failed', error);
      } finally {
        setTableDetailLoading(false);
      }
    },
    [canOperateSource, http, message, t],
  );
  const isForcedExpiredEffective = (entry: {
    expired?: boolean;
    force_enabled_after_expiry?: boolean;
    enabled?: boolean | null;
    effective_enabled?: boolean;
  }) =>
    Boolean(entry.expired && entry.force_enabled_after_expiry && entry.enabled !== false && entry.effective_enabled !== false);
  const resolveSourceStatusKey = (entry: SemanticModelSource) => {
    if (isLegacyUnboundSource(entry)) {
      return 'knowledge.base.source-status-legacy-unbound';
    }
    if (entry.enabled === false) {
      return 'knowledge.base.source-status-disabled';
    }
    if (isForcedExpiredEffective(entry)) {
      return 'knowledge.base.source-status-expired-forced';
    }
    if (entry.expired) {
      return 'knowledge.base.source-status-expired';
    }
    if (entry.effective_enabled === false) {
      return 'knowledge.base.source-status-disabled';
    }
    return 'knowledge.base.source-status-enabled';
  };
  const resolveProcessStatusKey = (status: string | null) => {
    switch (status) {
      case 'succeeded':
      case 'success':
      case 'ready':
      case 'completed':
        return 'knowledge.base.source-process-status-completed';
      case 'failed':
        return 'knowledge.base.source-process-status-failed';
      case 'queued':
      case 'running':
      case 'pending':
        return 'knowledge.base.source-process-status-processing';
      case 'legacy_unbound':
        return 'knowledge.base.source-process-status-legacy-unbound';
      default:
        return 'knowledge.base.source-process-status-unknown';
    }
  };
  const resolveProcessStatusClassName = (status: string | null) => {
    switch (status) {
      case 'failed':
        return styles.processStatusError;
      case 'queued':
      case 'running':
      case 'pending':
        return styles.processStatusProcessing;
      case 'succeeded':
      case 'success':
      case 'ready':
      case 'completed':
        return styles.processStatusSuccess;
      default:
        return styles.processStatusNeutral;
    }
  };
  const renderTags = (tags: string[] | undefined | null) =>
    Array.isArray(tags) && tags.length > 0 ? (
      <Space size={4} wrap>
        {tags.map((tag) => (
          <Tag key={tag} className={styles.sourceTypeTag}>
            {tag}
          </Tag>
        ))}
      </Space>
    ) : (
      <Text type="secondary">-</Text>
    );
  const documentPreviewFileId = documentDetail ? resolveDocumentPreviewFileId(documentDetail.source) : '';
  const documentFileExtension = documentDetail
    ? getFileExtFromName(getSourceDisplayName(documentDetail.source)).toLowerCase()
    : '';
  const isMediaDocument = [
    'mp3',
    'wav',
    'm4a',
    'aac',
    'flac',
    'ogg',
    'wma',
    'mp4',
    'mov',
    'avi',
    'mkv',
    'webm',
    'mpeg',
    'wmv',
    'flv',
  ].includes(documentFileExtension);
  const canPreviewMedia = isMediaPreviewSupported(documentFileExtension);
  const playingSegment = documentDetail?.segments.find((segment) => segment.segment_id === playingSegmentId);
  const documentPreviewFetchBlob = useCallback(async () => {
    if (!documentPreviewFileId) {
      console.warn('[KnowledgeAdvancedConfigPage] document preview file id is empty');
      throw new Error('Document preview file id is empty');
    }
    const result = await previewSemanticModelSourceFileApi(knowledgeId, documentPreviewFileId, http, {
      responseType: 'blob',
      responseContentType: 'blob',
    });
    const response = result as { data: Blob };
    return response.data instanceof Blob ? response.data : new Blob([response.data as unknown as BlobPart]);
  }, [documentPreviewFileId, http, knowledgeId]);

  const renderDocumentPreview = () => {
    if (!documentDetail) return null;
    if (!documentPreviewFileId) {
      return (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={documentDetail.preview.reason || t('knowledge.base.document-preview-unavailable')}
        />
      );
    }
    return (
      <EmbeddedFilePreview
        fileExt={documentFileExtension}
        fetchBlob={documentPreviewFetchBlob}
        height="calc(100vh - 220px)"
        markdownPreviewLabel={t('knowledge.base.document-tab-preview')}
        markdownRawLabel={t('knowledge.base.document-preview-raw')}
        unsupportedText={t('knowledge.base.document-preview-unavailable')}
        mediaStartTime={typeof playingSegment?.start_ms === 'number' ? playingSegment.start_ms / 1000 : undefined}
        mediaEndTime={typeof playingSegment?.end_ms === 'number' ? playingSegment.end_ms / 1000 : undefined}
        mediaPlaying={playingSegmentId !== null}
        onMediaPlayingChange={(playing) => {
          if (!playing) setPlayingSegmentId(null);
        }}
      />
    );
  };
  const renderDocumentInfo = () => {
    if (!documentDetail) return null;
    const info = documentDetail.file_info;
    const canEditDocumentGovernance = canAppendSources;
    const segmentVersions = Array.isArray(documentDetail.segment_versions) ? documentDetail.segment_versions : [];
    const versionOrdinalByID = getVersionOrdinalMap(segmentVersions);
    return (
      <div className={styles.infoPanel} data-testid="knowledge-document-info-panel">
        <section className={styles.infoCard}>
          <div className={styles.infoCardTitle}>{t('knowledge.base.document-info-tags')}</div>
          <div className={styles.tagEditRow}>
            <div className={styles.tagPreview}>
              {draftTags.length > 0 ? (
                renderTags(draftTags)
              ) : (
                <Text type="secondary">{t('knowledge.base.document-info-tags-empty')}</Text>
              )}
            </div>
            <div className={styles.tagEditActions}>
              <Select
                mode="tags"
                value={draftTags}
                onChange={setDraftTags}
                disabled={!canEditDocumentGovernance}
                className={styles.tagEditControl}
                placeholder={t('knowledge.base.document-info-tags-placeholder')}
                data-testid="knowledge-document-tags-input"
              />
              <Button
                onClick={saveDocumentTags}
                loading={documentTagsSaving}
                disabled={!canEditDocumentGovernance}
                data-testid="knowledge-document-tags-save-btn"
              >
                {t('knowledge.base.document-info-tags-save')}
              </Button>
            </div>
          </div>
        </section>
        <section className={styles.infoCard}>
          <div className={styles.infoCardTitle}>{t('knowledge.base.document-info-expires-at')}</div>
          <div className={styles.expiryInlineEditor}>
            <Input
              type="date"
              value={formatDateInputValue(draftExpiresAt)}
              onChange={(event) => handleExpiresDateChange(event.target.value)}
              disabled={!canEditDocumentGovernance}
              className={styles.expiryDateInput}
              data-testid="knowledge-document-expires-date-input"
              aria-label={t('knowledge.base.document-info-expires-at')}
            />
            <Button
              onClick={saveDocumentExpiresAt}
              loading={documentExpirySaving}
              disabled={!canEditDocumentGovernance}
              data-testid="knowledge-document-expires-save-btn"
            >
              {t('knowledge.base.document-info-expires-save')}
            </Button>
            <Button
              type="text"
              onClick={() => setDraftExpiresAt(null)}
              disabled={!canEditDocumentGovernance}
              data-testid="knowledge-document-expires-clear-btn"
            >
              {t('knowledge.base.document-info-expires-clear')}
            </Button>
          </div>
          <Text type="secondary" className={styles.infoCardHint}>
            {info.expires_at
              ? t('knowledge.base.document-info-expires-current', { time: formatOptionalTimestamp(info.expires_at) })
              : t('knowledge.base.source-expiry-current-empty')}
          </Text>
          {info.expired ? (
            <div className={styles.expiryStatusRow}>
              <Tag className={isForcedExpiredEffective(info) ? styles.forceTag : styles.expiredTag}>
                {t(
                  isForcedExpiredEffective(info)
                    ? 'knowledge.base.document-info-expired-forced'
                    : 'knowledge.base.document-info-expired',
                )}
              </Tag>
            </div>
          ) : null}
        </section>
        <section className={styles.infoCard}>
          <div className={styles.infoCardTitle}>{t('knowledge.base.document-info-version-records')}</div>
          <div className={styles.versionList} data-testid="knowledge-document-version-list">
            {segmentVersions.length > 0 ? (
              segmentVersions.map((version) => (
                <div
                  key={version.version_id}
                  className={`${styles.versionItem} ${version.current ? styles.versionItemCurrent : ''}`}
                >
                  <div className={styles.versionHeader}>
                    <Text strong className={styles.versionTitle}>
                      {formatSegmentVersionTitle(version, versionOrdinalByID)}
                    </Text>
                    {version.current ? (
                      <Tag className={styles.sourceTypeTag}>{t('knowledge.base.document-version-current')}</Tag>
                    ) : null}
                  </div>
                  <div className={styles.versionMeta}>
                    <Text type="secondary">
                      {t('knowledge.base.document-index-version', { version: version.index_version ?? '-' })}
                    </Text>
                    <Text type="secondary">
                      {t('knowledge.base.document-version-operator', { name: getVersionOperatorName(version) })}
                    </Text>
                    <Text type="secondary">
                      {t('knowledge.base.document-version-time', {
                        time: formatOptionalTimestamp(getVersionTimestamp(version)),
                      })}
                    </Text>
                  </div>
                </div>
              ))
            ) : (
              <Text type="secondary">{t('knowledge.base.document-version-empty')}</Text>
            )}
          </div>
        </section>
      </div>
    );
  };
  const renderDocumentSegments = () => {
    if (!documentDetail) return null;
    const segments = Array.isArray(documentDetail.segments) ? documentDetail.segments : [];
    const selectedVersionId = documentDetail.selected_segment_version_id ?? documentDetail.file_info.segment_version_id ?? '';
    const currentVersionId = documentDetail.current_segment_version_id ?? documentDetail.file_info.segment_version_id ?? '';
    const viewingCurrent = selectedVersionId !== '' && selectedVersionId === currentVersionId;
    const canMutateSegments = canAppendSources && viewingCurrent && documentDetail.segment_status.available;
    const searchText = segmentSearch;
    const segmentRows = segments
      .filter((segment) => !isMediaDocument || !segment.segment_type || segment.segment_type === 'transcript')
      .map((segment, sourceIndex) => ({ segment, sourceIndex }));
    const filteredSegmentRows =
      searchText === ''
        ? segmentRows
        : segmentRows.filter(({ segment }) => getSegmentSearchValues(segment).some((value) => value.includes(searchText)));
    const sortedSegmentRows = [...filteredSegmentRows].sort((left, right) => {
      if (segmentSort === 'recall_priority') {
        const recallDiff = (right.segment.recall_count ?? 0) - (left.segment.recall_count ?? 0);
        if (recallDiff !== 0) return recallDiff;
      }
      return left.sourceIndex - right.sourceIndex;
    });
    const pageCount = Math.max(1, Math.ceil(sortedSegmentRows.length / SEGMENT_PAGE_SIZE));
    const currentPage = Math.min(segmentPage, pageCount);
    const pagedSegmentRows = sortedSegmentRows.slice((currentPage - 1) * SEGMENT_PAGE_SIZE, currentPage * SEGMENT_PAGE_SIZE);
    const segmentSortMenuItems: MenuProps['items'] = [
      {
        key: 'source_order',
        label: (
          <span className={styles.segmentSortItem}>
            <span>{t('knowledge.base.document-segment-sort-source-order')}</span>
            {segmentSort === 'source_order' ? <CheckOutlined /> : null}
          </span>
        ),
      },
      {
        key: 'recall_priority',
        label: (
          <span className={styles.segmentSortItem}>
            <span>{t('knowledge.base.document-segment-sort-recall-priority')}</span>
            {segmentSort === 'recall_priority' ? <CheckOutlined /> : null}
          </span>
        ),
      },
    ];
    return (
      <div className={styles.segmentPanel} data-testid="knowledge-document-segment-panel">
        <div className={styles.segmentToolbar}>
          <Input.Search
            value={segmentSearch}
            onChange={(event) => {
              setSegmentSearch(event.target.value);
              setSegmentPage(1);
            }}
            onSearch={(value) => {
              setSegmentSearch(value);
              setSegmentPage(1);
            }}
            disabled={!documentDetail.segment_status.available}
            placeholder={t('knowledge.base.document-segment-search-placeholder')}
            data-testid="knowledge-document-segment-search"
          />
          <Tooltip title={canMutateSegments ? undefined : t('knowledge.base.document-segment-current-required')}>
            <Button
              disabled={!canMutateSegments || newSegmentOpen}
              icon={<PlusOutlined />}
              onClick={() => setNewSegmentOpen(true)}
              data-testid="knowledge-document-segment-create-btn"
              aria-label={t('knowledge.base.document-segment-create')}
            >
              {t('knowledge.base.document-segment-create')}
            </Button>
          </Tooltip>
          <Dropdown
            menu={{
              items: segmentSortMenuItems,
              selectedKeys: [segmentSort],
              onClick: ({ key }) => {
                setSegmentSort(key as SegmentSortKey);
                setSegmentPage(1);
              },
            }}
            trigger={['click']}
            placement="bottomRight"
          >
            <Button
              icon={<FilterOutlined />}
              disabled={!documentDetail.segment_status.available}
              data-testid="knowledge-document-segment-sort-btn"
              aria-label={t('knowledge.base.document-segment-sort')}
            />
          </Dropdown>
        </div>
        <div className={styles.segmentBody}>
          {!documentDetail.segment_status.available ? (
            <div className={styles.segmentUnavailable}>
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={documentDetail.segment_status.reason || t('knowledge.base.document-segment-unavailable')}
              />
              {canAppendSources ? (
                <Button
                  type="primary"
                  onClick={handleImportInitialSegments}
                  loading={segmentMutatingKey === 'import-initial'}
                  data-testid="knowledge-document-segment-import-btn"
                >
                  {t('knowledge.base.document-segment-import')}
                </Button>
              ) : null}
            </div>
          ) : (
            <div className={styles.segmentList}>
              {canMutateSegments && newSegmentOpen ? (
                <div className={styles.segmentCreateRow}>
                  <Input.TextArea
                    value={newSegmentContent}
                    onChange={(event) => setNewSegmentContent(event.target.value)}
                    disabled={!canMutateSegments || segmentMutatingKey !== null}
                    autoSize={{ minRows: 2, maxRows: 4 }}
                    placeholder={t('knowledge.base.document-segment-create-placeholder')}
                    data-testid="knowledge-document-segment-create-input"
                  />
                  <div className={styles.segmentCreateActions}>
                    <Button
                      type="primary"
                      onClick={handleCreateSegment}
                      loading={segmentMutatingKey === 'create'}
                      disabled={!canMutateSegments || segmentMutatingKey !== null}
                      data-testid="knowledge-document-segment-create-submit-btn"
                    >
                      {t('knowledge.base.document-segment-create-submit')}
                    </Button>
                    <Button
                      onClick={() => {
                        setNewSegmentOpen(false);
                        setNewSegmentContent('');
                      }}
                      disabled={!canMutateSegments || segmentMutatingKey !== null}
                      data-testid="knowledge-document-segment-create-cancel-btn"
                    >
                      {t('knowledge.base.form-cancel-button')}
                    </Button>
                  </div>
                </div>
              ) : null}
              {sortedSegmentRows.length === 0 ? (
                <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('knowledge.base.document-segment-empty')} />
              ) : null}
              {pagedSegmentRows.map(({ segment }, pageIndex) => {
                const draft = segmentDrafts[segment.segment_id] ?? {
                  content: segment.content ?? '',
                  ocr_text: segment.ocr_text ?? '',
                  image_description: segment.image_description ?? '',
                };
                const displayIndex = (currentPage - 1) * SEGMENT_PAGE_SIZE + pageIndex + 1;
                const isEditing = editingSegmentId === segment.segment_id;
                const imageFileId = getSegmentImageFileId(segment);
                const segmentType = segment.segment_type || (imageFileId !== '' ? 'image' : 'text');
                const isImageSegment = segmentType === 'image' || imageFileId !== '';
                const isMixedSegment = isImageSegment && Boolean(segment.content);
                const isTableSegment = segmentType === 'table';
                const isTranscriptSegment = segmentType === 'transcript';
                const hasPlayableTimeRange =
                  typeof segment.start_ms === 'number' &&
                  typeof segment.end_ms === 'number' &&
                  segment.start_ms >= 0 &&
                  segment.start_ms < segment.end_ms;
                const isPlayingSegment = playingSegmentId === segment.segment_id;
                const showSegmentMediaFields = isImageSegment;
                const segmentBusy =
                  segmentMutatingKey === `enabled:${segment.segment_id}` ||
                  segmentMutatingKey === `edit:${segment.segment_id}` ||
                  segmentMutatingKey === `delete:${segment.segment_id}` ||
                  segmentMutatingKey?.startsWith(`edit:${segment.segment_id}:`) === true;
                return (
                  <section key={segment.segment_id} className={styles.segmentCard} data-testid="knowledge-document-segment-item">
                    <div className={styles.segmentCardHeader}>
                      <Space size={6} wrap>
                        <Tag className={styles.sourceTypeTag}>{segment.level}</Tag>
                        {isImageSegment ? (
                          <Tag className={styles.segmentKindTag}>
                            {t(
                              isMixedSegment
                                ? 'knowledge.base.document-segment-kind-mixed'
                                : 'knowledge.base.document-segment-kind-image',
                            )}
                          </Tag>
                        ) : isTableSegment ? (
                          <Tag className={styles.segmentKindTag}>{t('knowledge.base.document-segment-kind-table')}</Tag>
                        ) : isTranscriptSegment ? (
                          <Tag className={styles.segmentKindTag}>{t('knowledge.base.document-segment-kind-transcript')}</Tag>
                        ) : (
                          <Tag className={styles.segmentKindTag}>{t('knowledge.base.document-segment-kind-text')}</Tag>
                        )}
                        <Text strong>{t('knowledge.base.document-segment-display-index', { index: displayIndex })}</Text>
                      </Space>
                      <Space size={8}>
                        <Tag>{t('knowledge.base.document-segment-word-count', { count: segment.word_count })}</Tag>
                        <Tag>{t('knowledge.base.document-segment-recall-count', { count: segment.recall_count })}</Tag>
                        <Switch
                          size="small"
                          checked={segment.enabled}
                          disabled={!canMutateSegments || segmentBusy}
                          onChange={(checked) => handleToggleSegmentEnabled(segment, checked)}
                          data-testid="knowledge-document-segment-enabled-switch"
                          aria-label={t('knowledge.base.document-segment-enabled')}
                        />
                        {canPreviewMedia && isTranscriptSegment && hasPlayableTimeRange ? (
                          <Tooltip
                            title={t(
                              isPlayingSegment ? 'knowledge.base.document-segment-pause' : 'knowledge.base.document-segment-play',
                            )}
                          >
                            <Button
                              type="text"
                              size="small"
                              icon={isPlayingSegment ? <PauseOutlined /> : <CaretRightOutlined />}
                              onClick={() => setPlayingSegmentId(isPlayingSegment ? null : segment.segment_id)}
                              data-testid={`knowledge-document-segment-play-${segment.segment_id}`}
                              aria-label={t(
                                isPlayingSegment
                                  ? 'knowledge.base.document-segment-pause'
                                  : 'knowledge.base.document-segment-play',
                              )}
                            />
                          </Tooltip>
                        ) : null}
                        <ListActionButton
                          action="edit"
                          label={t('knowledge.base.document-segment-edit')}
                          disabled={!canMutateSegments || segmentBusy}
                          onClick={() => setEditingSegmentId(isEditing ? null : segment.segment_id)}
                          data-testid="knowledge-document-segment-edit-btn"
                        />
                        <Popconfirm
                          title={t('knowledge.base.document-segment-delete-confirm-title')}
                          description={t('knowledge.base.document-segment-delete-confirm-content')}
                          okText={t('knowledge.base.document-segment-delete-confirm-ok')}
                          cancelText={t('knowledge.base.document-segment-delete-confirm-cancel')}
                          okButtonProps={{ danger: true }}
                          disabled={!canMutateSegments || segmentBusy}
                          onConfirm={() => handleDeleteSegment(segment)}
                        >
                          <ListActionButton
                            action="delete"
                            danger
                            disabled={!canMutateSegments || segmentBusy}
                            disabledTooltip={t('knowledge.base.document-segment-current-required')}
                            label={t('knowledge.base.document-segment-delete')}
                            data-testid="knowledge-document-segment-delete-btn"
                          />
                        </Popconfirm>
                      </Space>
                    </div>
                    {isImageSegment ? (
                      <div className={styles.segmentImageBlock}>
                        <div className={styles.segmentImageThumb}>
                          <FileTextOutlined />
                        </div>
                        <div className={styles.segmentImageInfo}>
                          <Text strong>{t('knowledge.base.document-segment-image-file')}</Text>
                          <Text type="secondary" className={styles.segmentImageId}>
                            {imageFileId}
                          </Text>
                          <Button
                            size="small"
                            onClick={() => openSegmentImagePreview(segment)}
                            loading={segmentPreviewLoadingKey === segment.segment_id}
                            data-testid="knowledge-document-segment-image-preview-btn"
                          >
                            {t('knowledge.base.document-segment-image-preview')}
                          </Button>
                        </div>
                      </div>
                    ) : null}
                    {isTableSegment && segment.content ? (
                      <iframe
                        className={styles.segmentTablePreview}
                        sandbox=""
                        srcDoc={`<!doctype html><html><head><meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'unsafe-inline'"></head><body>${segment.content}</body></html>`}
                        title={t('knowledge.base.document-segment-table-preview')}
                        data-testid="knowledge-document-segment-table-preview"
                      />
                    ) : segment.content || !isImageSegment ? (
                      <Paragraph className={styles.segmentContentPreview}>{segment.content || '-'}</Paragraph>
                    ) : null}
                    {isTranscriptSegment && typeof segment.start_ms === 'number' && typeof segment.end_ms === 'number' ? (
                      <Text type="secondary" className={styles.segmentTimestamp}>
                        {formatSegmentTimestamp(segment.start_ms)}–{formatSegmentTimestamp(segment.end_ms)}
                      </Text>
                    ) : null}
                    {showSegmentMediaFields && segment.ocr_text ? (
                      <Paragraph className={styles.segmentSubPreview}>
                        <Text type="secondary">{t('knowledge.base.document-segment-ocr')}: </Text>
                        {segment.ocr_text}
                      </Paragraph>
                    ) : null}
                    {showSegmentMediaFields && segment.image_description ? (
                      <Paragraph className={styles.segmentSubPreview}>
                        <Text type="secondary">{t('knowledge.base.document-segment-image-description')}: </Text>
                        {segment.image_description}
                      </Paragraph>
                    ) : null}
                    {isEditing ? (
                      <div className={styles.segmentFields}>
                        <div className={styles.segmentField}>
                          <Text type="secondary">{t('knowledge.base.document-segment-content')}</Text>
                          <Input.TextArea
                            value={draft.content}
                            disabled={!canMutateSegments || segmentBusy}
                            autoSize={{ minRows: 3, maxRows: 8 }}
                            onChange={(event) =>
                              setSegmentDrafts((current) => ({
                                ...current,
                                [segment.segment_id]: {
                                  ...(current[segment.segment_id] ?? draft),
                                  content: event.target.value,
                                },
                              }))
                            }
                            data-testid="knowledge-document-segment-content-input"
                          />
                        </div>
                        {showSegmentMediaFields ? (
                          <>
                            <div className={styles.segmentField}>
                              <Text type="secondary">{t('knowledge.base.document-segment-ocr')}</Text>
                              <Input.TextArea
                                value={draft.ocr_text}
                                disabled={!canMutateSegments || segmentBusy}
                                autoSize={{ minRows: 2, maxRows: 6 }}
                                onChange={(event) =>
                                  setSegmentDrafts((current) => ({
                                    ...current,
                                    [segment.segment_id]: {
                                      ...(current[segment.segment_id] ?? draft),
                                      ocr_text: event.target.value,
                                    },
                                  }))
                                }
                                data-testid="knowledge-document-segment-ocr-input"
                              />
                            </div>
                            <div className={styles.segmentField}>
                              <Text type="secondary">{t('knowledge.base.document-segment-image-description')}</Text>
                              <Input.TextArea
                                value={draft.image_description}
                                disabled={!canMutateSegments || segmentBusy}
                                autoSize={{ minRows: 2, maxRows: 6 }}
                                onChange={(event) =>
                                  setSegmentDrafts((current) => ({
                                    ...current,
                                    [segment.segment_id]: {
                                      ...(current[segment.segment_id] ?? draft),
                                      image_description: event.target.value,
                                    },
                                  }))
                                }
                                data-testid="knowledge-document-segment-image-description-input"
                              />
                            </div>
                          </>
                        ) : null}
                        <div className={styles.segmentEditActions}>
                          <Button
                            size="small"
                            disabled={segmentBusy}
                            onClick={() => cancelSegmentEdit(segment)}
                            data-testid="knowledge-document-segment-cancel-btn"
                          >
                            {t('knowledge.base.document-segment-edit-cancel')}
                          </Button>
                          <Button
                            size="small"
                            disabled={!canMutateSegments || segmentBusy}
                            loading={segmentMutatingKey === `edit:${segment.segment_id}`}
                            onClick={() => handleUpdateSegmentText(segment, draft, showSegmentMediaFields)}
                            data-testid="knowledge-document-segment-save-btn"
                          >
                            {t('knowledge.base.document-segment-save')}
                          </Button>
                        </div>
                      </div>
                    ) : null}
                  </section>
                );
              })}
            </div>
          )}
        </div>
        <div className={styles.segmentFooter}>
          <Text type="secondary">
            {t('knowledge.base.document-segment-page-summary', {
              current: sortedSegmentRows.length === 0 ? 0 : currentPage,
              pages: sortedSegmentRows.length === 0 ? 0 : pageCount,
              total: filteredSegmentRows.length,
            })}
          </Text>
          {sortedSegmentRows.length > 0 ? (
            <Pagination
              size="small"
              current={currentPage}
              total={sortedSegmentRows.length}
              pageSize={SEGMENT_PAGE_SIZE}
              showSizeChanger={false}
              onChange={(page) => setSegmentPage(page)}
              data-testid="knowledge-document-segment-pagination"
            />
          ) : null}
        </div>
      </div>
    );
  };
  const renderDocumentModal = () => {
    const documentName = documentDetail ? getSourceDisplayName(documentDetail.source) : '';
    const segmentVersions = documentDetail?.segment_versions ?? [];
    const selectedSegmentVersionID =
      documentDetail?.selected_segment_version_id ?? documentDetail?.file_info.segment_version_id ?? '';
    const currentSegmentVersionID =
      documentDetail?.current_segment_version_id ?? documentDetail?.file_info.segment_version_id ?? '';
    const selectedVersion = segmentVersions.find((version) => version.version_id === selectedSegmentVersionID);
    const versionOrdinalByID = getVersionOrdinalMap(segmentVersions);
    const orderedSegmentVersions = [...segmentVersions].sort((left, right) => {
      const leftOrdinal = versionOrdinalByID.get(left.version_id) ?? Number.MAX_SAFE_INTEGER;
      const rightOrdinal = versionOrdinalByID.get(right.version_id) ?? Number.MAX_SAFE_INTEGER;
      return leftOrdinal - rightOrdinal;
    });
    const selectedVersionIndex = orderedSegmentVersions.findIndex((version) => version.version_id === selectedSegmentVersionID);
    const olderVersion =
      selectedVersionIndex > 0 && selectedVersionIndex < orderedSegmentVersions.length
        ? orderedSegmentVersions[selectedVersionIndex - 1]
        : null;
    const newerVersion =
      selectedVersionIndex >= 0 && selectedVersionIndex < orderedSegmentVersions.length - 1
        ? orderedSegmentVersions[selectedVersionIndex + 1]
        : null;
    return (
      <Modal
        open={documentOpen}
        onCancel={closeDocumentDetail}
        footer={null}
        width="min(1280px, calc(100vw - 64px))"
        centered
        closable={false}
        className={styles.documentModal}
        destroyOnHidden
        data-testid="knowledge-document-detail-modal"
      >
        {documentLoading || !documentDetail ? (
          <div className={styles.documentLoading}>
            <Spin />
          </div>
        ) : (
          <div className={styles.documentLayout}>
            <div className={styles.documentHeader}>
              <div className={styles.documentTitleBar}>
                <Title level={4} className={styles.documentTitle}>
                  {documentName}
                </Title>
                <div className={styles.documentVersionBar}>
                  <Tooltip title={t('knowledge.base.document-version-prev')}>
                    <Button
                      disabled={!olderVersion}
                      icon={<LeftOutlined />}
                      className={styles.documentVersionNavBtn}
                      onClick={() => {
                        if (olderVersion) loadDocumentVersion(olderVersion.version_id);
                      }}
                      data-testid="knowledge-document-version-prev-btn"
                      aria-label={t('knowledge.base.document-version-prev')}
                    />
                  </Tooltip>
                  <div className={styles.versionDisplay} data-testid="knowledge-document-version-display">
                    <Text strong className={styles.versionDisplayTitle}>
                      {formatSegmentVersionSwitcherTitle(selectedVersion, versionOrdinalByID)}
                    </Text>
                  </div>
                  <Button
                    disabled={
                      !canAppendSources ||
                      selectedSegmentVersionID === '' ||
                      selectedSegmentVersionID === currentSegmentVersionID ||
                      selectedVersion?.status !== 'committed'
                    }
                    loading={segmentMutatingKey === `set-current:${selectedSegmentVersionID}`}
                    onClick={() => handleSetCurrentSegmentVersion(selectedSegmentVersionID)}
                    data-testid="knowledge-document-version-set-current-btn"
                  >
                    {t('knowledge.base.document-version-set-current')}
                  </Button>
                  <Tooltip title={t('knowledge.base.document-version-next')}>
                    <Button
                      disabled={!newerVersion}
                      icon={<RightOutlined />}
                      className={styles.documentVersionNavBtn}
                      onClick={() => {
                        if (newerVersion) loadDocumentVersion(newerVersion.version_id);
                      }}
                      data-testid="knowledge-document-version-next-btn"
                      aria-label={t('knowledge.base.document-version-next')}
                    />
                  </Tooltip>
                </div>
              </div>
              <Button
                className={styles.documentCloseBtn}
                icon={<CloseOutlined />}
                onClick={closeDocumentDetail}
                aria-label={t('knowledge.base.document-close')}
                data-testid="knowledge-document-close-btn"
              />
            </div>
            <div className={styles.documentContent}>
              <div className={styles.documentMain}>
                <Tabs
                  activeKey={documentTab}
                  onChange={(key) => setDocumentTab(key as DocumentTabKey)}
                  items={[
                    {
                      key: 'preview',
                      label: t('knowledge.base.document-tab-preview'),
                      children: <div className={styles.previewPanel}>{renderDocumentPreview()}</div>,
                    },
                    {
                      key: 'info',
                      label: t('knowledge.base.document-tab-info'),
                      children: renderDocumentInfo(),
                    },
                  ]}
                />
              </div>
              <aside className={styles.documentAside}>
                <div className={styles.segmentTitle}>{t('knowledge.base.document-segment-title')}</div>
                {renderDocumentSegments()}
              </aside>
            </div>
          </div>
        )}
      </Modal>
    );
  };
  const renderTableDetailModal = () => {
    const tableInfo = tableDetail?.info;
    const sample = tableDetail?.sample;
    const columnRows = Array.isArray(tableInfo?.columns) ? tableInfo.columns : [];
    const sampleColumnDefs = Array.isArray(sample?.columns) ? sample.columns : [];
    const sampleDataRows = Array.isArray(sample?.data) ? sample.data : [];
    const sampleColumns: TableColumnsType<Record<string, string | number>> = sampleColumnDefs.map((column) => ({
      title: column.name,
      dataIndex: column.name,
      key: column.name,
      width: 180,
      ellipsis: true,
      render: (value: string | number | undefined) => (
        <Typography.Text className={styles.tableDetailSampleCell} ellipsis={{ tooltip: String(value || '-') }}>
          {value || '-'}
        </Typography.Text>
      ),
    }));
    const sampleRows = sampleDataRows.map((row, rowIndex) => {
      const rowData: Record<string, string | number> = { key: rowIndex };
      sampleColumnDefs.forEach((column, columnIndex) => {
        rowData[column.name] = row[columnIndex] ?? '';
      });
      return rowData;
    });
    const columnTableColumns: TableColumnsType<CatalogTableColumn> = [
      {
        title: t('knowledge.base.source-table-detail-column-name'),
        dataIndex: 'name',
        key: 'name',
        width: 140,
        ellipsis: true,
      },
      {
        title: t('knowledge.base.source-table-detail-column-type'),
        dataIndex: 'type',
        key: 'type',
        width: 150,
        ellipsis: true,
      },
      {
        title: t('knowledge.base.source-table-detail-column-primary-key'),
        dataIndex: 'is_pk',
        key: 'is_pk',
        width: 100,
        render: (value: boolean | undefined) =>
          value ? t('knowledge.base.source-table-detail-yes') : t('knowledge.base.source-table-detail-no'),
      },
      {
        title: t('knowledge.base.source-table-detail-column-default'),
        dataIndex: 'default',
        key: 'default',
        width: 110,
        ellipsis: true,
        render: (value: string | undefined) => value || '-',
      },
      {
        title: t('knowledge.base.source-table-detail-column-comment'),
        dataIndex: 'comment',
        key: 'comment',
        width: 130,
        ellipsis: true,
        render: (value: string | undefined) => value || '-',
      },
    ];
    const statTableColumns: TableColumnsType<CatalogTableStat> = [
      {
        title: t('knowledge.base.source-table-detail-stat-name'),
        dataIndex: 'name',
        key: 'name',
        width: 180,
        ellipsis: true,
      },
      {
        title: t('knowledge.base.source-table-detail-stat-type'),
        dataIndex: 'type',
        key: 'type',
        width: 150,
        ellipsis: true,
      },
      {
        title: t('knowledge.base.source-table-detail-stat-max'),
        dataIndex: 'max_value',
        key: 'max_value',
        width: 150,
        ellipsis: true,
        render: (value: string | undefined) => value || '-',
      },
      {
        title: t('knowledge.base.source-table-detail-stat-min'),
        dataIndex: 'min_value',
        key: 'min_value',
        width: 150,
        ellipsis: true,
        render: (value: string | undefined) => value || '-',
      },
    ];

    return (
      <Modal
        open={tableDetailOpen}
        onCancel={() => {
          if (tableDetailLoading) return;
          setTableDetailOpen(false);
          setTableDetail(null);
        }}
        footer={null}
        width={720}
        centered
        className={styles.tableDetailModal}
        title={t('knowledge.base.source-table-detail-title', {
          name: tableInfo?.name || (tableDetail ? getSourceDisplayName(tableDetail.source) : ''),
        })}
        destroyOnHidden
        data-testid="knowledge-source-table-detail-modal"
      >
        {tableDetailLoading || !tableInfo ? (
          <div className={styles.tableDetailLoading}>
            <Spin />
          </div>
        ) : (
          <div className={styles.tableDetailContent}>
            <div className={styles.tableDetailInfoBar}>
              <div className={styles.tableDetailInfoPair}>
                <span className={styles.tableDetailInfoLabel}>{t('knowledge.base.source-table-detail-info-name')}</span>
                <span className={styles.tableDetailInfoValue}>{tableInfo.name || '-'}</span>
              </div>
              <div className={styles.tableDetailInfoPair}>
                <span className={styles.tableDetailInfoLabel}>{t('knowledge.base.source-table-detail-info-rows')}</span>
                <span className={styles.tableDetailInfoValue}>{tableInfo.lines ?? 0}</span>
              </div>
              <div className={styles.tableDetailInfoPair}>
                <span className={styles.tableDetailInfoLabel}>{t('knowledge.base.source-table-detail-info-size')}</span>
                <span className={styles.tableDetailInfoValue}>{formatFileSize(tableInfo.size ?? 0)}</span>
              </div>
              <div className={styles.tableDetailInfoPair}>
                <span className={styles.tableDetailInfoLabel}>{t('knowledge.base.source-table-detail-info-created-at')}</span>
                <span className={styles.tableDetailInfoValue}>
                  {tableInfo.created_at ? formatDateTime(tableInfo.created_at, { timezone }) : '-'}
                </span>
              </div>
              <div className={styles.tableDetailInfoPair}>
                <span className={styles.tableDetailInfoLabel}>{t('knowledge.base.source-table-detail-info-created-by')}</span>
                <span className={styles.tableDetailInfoValue}>{getTableCreatedByDisplayName(tableDetail.source, tableInfo)}</span>
              </div>
              <div className={styles.tableDetailInfoPair}>
                <span className={styles.tableDetailInfoLabel}>{t('knowledge.base.source-table-detail-info-comment')}</span>
                <span className={styles.tableDetailInfoValue}>{tableInfo.comment || '-'}</span>
              </div>
            </div>
            <Tabs
              items={[
                {
                  key: 'columns',
                  label: (
                    <span className={styles.tableDetailTabLabel}>
                      {t('knowledge.base.source-table-detail-tab-columns')}
                      <span className={styles.tableDetailTabSuffix}>{columnRows.length}</span>
                    </span>
                  ),
                  children: (
                    <Table<CatalogTableColumn>
                      rowKey={(record) => `${record.name}-${record.type}`}
                      columns={columnTableColumns}
                      dataSource={columnRows}
                      pagination={false}
                      tableLayout="fixed"
                      scroll={{ x: 630 }}
                      className={styles.tableDetailDataTable}
                      data-testid="knowledge-source-table-detail-columns-table"
                    />
                  ),
                },
                {
                  key: 'statistics',
                  label: t('knowledge.base.source-table-detail-tab-statistics'),
                  children: (
                    <Table<CatalogTableStat>
                      rowKey={(record) => `${record.name}-${record.type}`}
                      columns={statTableColumns}
                      dataSource={tableInfo.stats ?? []}
                      pagination={false}
                      tableLayout="fixed"
                      scroll={{ x: 630 }}
                      className={styles.tableDetailDataTable}
                      data-testid="knowledge-source-table-detail-stats-table"
                    />
                  ),
                },
                {
                  key: 'sql',
                  label: t('knowledge.base.source-table-detail-tab-sql'),
                  children: <pre className={styles.tableDetailSql}>{tableInfo.create_sql || '-'}</pre>,
                },
                {
                  key: 'sample',
                  label: t('knowledge.base.source-table-detail-tab-sample'),
                  children: (
                    <Table<Record<string, string | number>>
                      rowKey={(record) => String(record.key)}
                      columns={sampleColumns}
                      dataSource={sampleRows}
                      pagination={false}
                      tableLayout="fixed"
                      className={styles.tableDetailDataTable}
                      scroll={{ x: Math.max(sampleColumns.length * 180, 560) }}
                      data-testid="knowledge-source-table-detail-sample-table"
                    />
                  ),
                },
              ]}
            />
          </div>
        )}
      </Modal>
    );
  };
  const renderSourceExpiryModal = () => (
    <Modal
      open={Boolean(expirySource)}
      onCancel={closeSourceExpiryModal}
      footer={
        <div className={styles.expiryModalFooter}>
          <Text type="secondary">{t('knowledge.base.source-expiry-modal-footer')}</Text>
          <Button onClick={closeSourceExpiryModal} data-testid="knowledge-source-expiry-close-btn">
            {t('knowledge.base.source-expiry-close')}
          </Button>
        </div>
      }
      width={760}
      centered
      closable
      title={t('knowledge.base.source-expiry-title')}
      className={styles.expiryModal}
      data-testid="knowledge-source-expiry-modal"
      destroyOnHidden
    >
      <section className={styles.expiryModalCard}>
        <div className={styles.expiryModalCardTitle}>{t('knowledge.base.source-expiry-card-title')}</div>
        <Paragraph type="secondary" className={styles.expiryModalDesc}>
          {t('knowledge.base.source-expiry-card-desc')}
        </Paragraph>
        <div className={styles.expiryModalControls}>
          <Input
            type="date"
            value={formatDateInputValue(expiryDraftExpiresAt)}
            onChange={(event) => handleSourceExpiryDateChange(event.target.value)}
            disabled={!canAppendSources || expirySaving}
            className={styles.expiryModalDateInput}
            data-testid="knowledge-source-expiry-date-input"
            aria-label={t('knowledge.base.source-expiry-title')}
          />
          <Button
            type="primary"
            onClick={saveSourceExpiry}
            loading={expirySaving}
            disabled={!canAppendSources}
            data-testid="knowledge-source-expiry-save-btn"
          >
            {t('knowledge.base.source-expiry-save')}
          </Button>
          <Button
            type="text"
            onClick={() => setExpiryDraftExpiresAt(null)}
            disabled={!canAppendSources || expirySaving}
            data-testid="knowledge-source-expiry-clear-btn"
          >
            {t('knowledge.base.document-info-expires-clear')}
          </Button>
        </div>
        <Text type="secondary" className={styles.expiryModalCurrent}>
          {expirySource?.expires_at
            ? t('knowledge.base.document-info-expires-current', { time: formatOptionalTimestamp(expirySource.expires_at) })
            : t('knowledge.base.source-expiry-current-empty')}
        </Text>
      </section>
    </Modal>
  );

  return (
    <div className={styles.page} data-testid="knowledge-advanced-config-page">
      <div className={styles.header}>
        <Space align="start" className={styles.headerActions}>
          <Space align="start">
            <Button type="text" icon={<ArrowLeftOutlined />} onClick={handleBack} />
            <div>{renderHeaderInfo()}</div>
          </Space>
          {canAppendSources ? (
            <Button
              type="primary"
              onClick={() => setSourceSelectOpen(true)}
              loading={sourceAppending}
              data-testid="knowledge-source-select-btn"
            >
              {t('knowledge.base.add-source-action')}
            </Button>
          ) : null}
        </Space>
      </div>

      <div className={styles.content}>
        {detailLoading || !detail ? (
          <div className={styles.loading}>
            <Spin />
          </div>
        ) : (
          <Tabs
            className={styles.tabs}
            size="middle"
            activeKey={activeTab}
            onChange={(key) => setActiveTab(key as AdvancedTabKey)}
            items={[
              {
                key: 'source',
                label: t('knowledge.base.advanced-tab-source'),
                children: (
                  <div className={styles.sourcePanel}>
                    <div className={styles.sourceIntro}>
                      <Text type="secondary">{t('knowledge.base.source-list-description')}</Text>
                      <Text type="secondary">
                        {t('knowledge.base.source-list-summary', {
                          fileCount: sourceSummaryCounts.files,
                          tableCount: sourceSummaryCounts.tables,
                        })}
                      </Text>
                    </div>
                    {sourceLoadFailed ? (
                      <div className={styles.sourceRetry}>
                        <Button
                          size="small"
                          onClick={() => requestKnowledgeSources(sourcePagination.page)}
                          loading={sourceLoading}
                          data-testid="knowledge-source-list-retry-btn"
                        >
                          {t('knowledge.base.source-list-retry')}
                        </Button>
                      </div>
                    ) : null}
                    <Table<SemanticModelSource>
                      rowKey="row_id"
                      dataSource={sourceRows}
                      pagination={false}
                      size="middle"
                      loading={sourceLoading}
                      tableLayout="fixed"
                      scroll={{ x: sourceTableScrollX }}
                      className={styles.sourceTable}
                      data-testid="knowledge-source-table"
                      locale={{ emptyText: t('knowledge.base.source-list-empty') }}
                      columns={
                        [
                          {
                            title: t('knowledge.base.source-column-name'),
                            dataIndex: 'display_name',
                            key: 'name',
                            width: 360,
                            ellipsis: true,
                            render: (_value, entry) => {
                              const sourceName = getSourceDisplayName(entry);
                              const operable = canOperateSource(entry);
                              const sourceNameNode =
                                operable && entry.source_type === 'file' ? (
                                  <Button
                                    type="link"
                                    className={styles.sourceNameButton}
                                    data-testid="knowledge-source-name-detail-btn"
                                    onClick={() => openDocumentDetail(entry)}
                                  >
                                    <Text className={styles.sourceName} ellipsis={{ tooltip: sourceName }}>
                                      {sourceName}
                                    </Text>
                                  </Button>
                                ) : operable && entry.source_type === 'table' ? (
                                  <Button
                                    type="link"
                                    className={styles.sourceNameButton}
                                    data-testid="knowledge-source-table-detail-btn"
                                    onClick={() => openTableDetail(entry)}
                                  >
                                    <Text className={styles.sourceName} ellipsis={{ tooltip: sourceName }}>
                                      {sourceName}
                                    </Text>
                                  </Button>
                                ) : (
                                  <Text
                                    className={styles.sourceName}
                                    data-testid="knowledge-source-name-static-text"
                                    ellipsis={{ tooltip: sourceName }}
                                  >
                                    {sourceName}
                                  </Text>
                                );
                              return (
                                <div className={styles.sourceNameCell}>
                                  {entry.source_type === 'file' ? <FileTextOutlined className={styles.sourceIcon} /> : null}
                                  {entry.source_type === 'volume' ? <HddOutlined className={styles.sourceIcon} /> : null}
                                  {entry.source_type === 'table' ? <TableOutlined className={styles.sourceIcon} /> : null}
                                  <div className={styles.sourceNameContent}>{sourceNameNode}</div>
                                </div>
                              );
                            },
                          },
                          {
                            title: t('knowledge.base.source-column-type'),
                            key: 'sourceType',
                            width: 120,
                            render: (_value, entry) => <Text type="secondary">{getSourceTypeLabel(entry)}</Text>,
                          },
                          {
                            title: t('knowledge.base.source-column-size'),
                            key: 'size',
                            width: 140,
                            render: (_value, entry) => <Text type="secondary">{getSourceSizeText(entry)}</Text>,
                          },
                          {
                            title: t('knowledge.base.source-column-catalog-path'),
                            key: 'catalogPath',
                            width: 260,
                            render: (_value, entry) => <Text type="secondary">{getCatalogPath(entry)}</Text>,
                          },
                          {
                            title: t('knowledge.base.source-column-process-status'),
                            key: 'processStatus',
                            width: 160,
                            render: (_value, entry) => {
                              const process = getSourceProcessStatus(entry);
                              const ready = isSourceProcessReady(entry);
                              return (
                                <Tooltip title={process.error || undefined}>
                                  <Tag
                                    className={resolveProcessStatusClassName(process.status)}
                                    data-testid="knowledge-source-process-status"
                                    data-source-row-id={entry.row_id}
                                    data-process-status={process.status || 'unknown'}
                                    data-process-ready={ready ? 'true' : 'false'}
                                  >
                                    {t(resolveProcessStatusKey(process.status))}
                                  </Tag>
                                </Tooltip>
                              );
                            },
                          },
                          {
                            title: t('knowledge.base.source-column-updated-at'),
                            key: 'updatedAt',
                            width: 160,
                            render: (_value, entry) => <Text type="secondary">{getSourceUpdatedText(entry)}</Text>,
                          },
                          {
                            title: t('knowledge.base.source-column-enable-status'),
                            key: 'enableStatus',
                            width: 140,
                            render: (_value, entry) => (
                              <Space size={6} wrap>
                                <Switch
                                  size="small"
                                  checked={isSourceSwitchChecked(entry)}
                                  disabled={
                                    !canAppendSources ||
                                    isLegacyUnboundSource(entry) ||
                                    !isSourceProcessReady(entry) ||
                                    governanceUpdatingSourceRowId === entry.row_id
                                  }
                                  onChange={(checked) => handleToggleSourceEnabled(entry, checked)}
                                  data-testid="knowledge-source-enable-switch"
                                  aria-label={t(resolveSourceStatusKey(entry))}
                                />
                                {governanceUpdatingSourceRowId === entry.row_id ? (
                                  <Spin size="small" data-testid="knowledge-source-enable-loading" />
                                ) : null}
                                {entry.source_type !== 'table' && entry.expired ? (
                                  <Tag className={isForcedExpiredEffective(entry) ? styles.forceTag : styles.expiredTag}>
                                    {t(
                                      isForcedExpiredEffective(entry)
                                        ? 'knowledge.base.source-status-expired-forced'
                                        : 'knowledge.base.source-status-expired-badge',
                                    )}
                                  </Tag>
                                ) : null}
                              </Space>
                            ),
                          },
                          {
                            title: t('knowledge.base.source-column-action'),
                            key: 'action',
                            width: sourceActionColumnWidth,
                            fixed: 'right',
                            render: (_value, entry) => (
                              <Space size="small" className={styles.sourceActions}>
                                {entry.source_type === 'table' ? (
                                  <ListActionButton
                                    action="api"
                                    label={t('knowledge.base.source-sql-placeholder')}
                                    data-testid="knowledge-source-sql-btn"
                                    disabled={!canOperateSource(entry)}
                                    onClick={() => openSourceSqlEditor(entry)}
                                  />
                                ) : null}
                                {canEditSourceExpiry(entry) ? (
                                  <ListActionButton
                                    action="configure"
                                    label={t('knowledge.base.source-expiry-action')}
                                    disabled={!canAppendSources || !canOperateSource(entry)}
                                    onClick={() => openSourceExpiryModal(entry)}
                                    data-testid="knowledge-source-expiry-btn"
                                  />
                                ) : null}
                                {canDownloadSource(entry) ? (
                                  <ListActionButton
                                    action="download"
                                    label={t('knowledge.base.source-download-action')}
                                    data-testid="knowledge-source-download-btn"
                                    loading={downloadingSourceRowId === entry.row_id}
                                    disabled={downloadingSourceRowId === entry.row_id || !canOperateSource(entry)}
                                    onClick={() => handleDownloadSource(entry)}
                                  />
                                ) : null}
                                {canAppendSources ? (
                                  <ListActionButton
                                    action="delete"
                                    danger
                                    label={t('knowledge.base.delete-source-action')}
                                    data-testid="knowledge-source-delete-btn"
                                    loading={deletingSourceRowId === entry.row_id}
                                    disabled={!canDeleteSource(entry)}
                                    onClick={() => handleDeleteSource(entry)}
                                  />
                                ) : null}
                              </Space>
                            ),
                          },
                        ] satisfies TableColumnsType<SemanticModelSource>
                      }
                    />
                    {sourcePagination.total > sourcePagination.pageSize ? (
                      <div className={styles.sourcePagination}>
                        <Pagination
                          size="small"
                          current={sourceCurrentPage}
                          total={sourcePagination.total}
                          pageSize={sourcePagination.pageSize}
                          showSizeChanger={false}
                          disabled={sourceLoading}
                          onChange={handleSourcePageChange}
                          data-testid="knowledge-source-pagination"
                        />
                      </div>
                    ) : null}
                  </div>
                ),
              },
              {
                key: 'semantic',
                label: t('knowledge.base.advanced-tab-semantic'),
                children: <KnowledgeAdvancedConfig knowledgeId={knowledgeId} visible={activeTab === 'semantic'} />,
              },
              {
                key: 'agents',
                label: t('knowledge.base.advanced-tab-agents'),
                children: (
                  <div className={styles.agentPanel}>
                    <div className={styles.agentIntro}>
                      <Text type="secondary">{t('knowledge.base.agent-association-description')}</Text>
                      <Text type="secondary">
                        {t('knowledge.base.agent-association-summary', { count: associatedAgentRows.length })}
                      </Text>
                    </div>
                    {agentAssociationLoadFailed ? (
                      <div className={styles.agentRetry}>
                        <Button
                          size="small"
                          onClick={loadAssociatedAgents}
                          loading={agentAssociationLoading}
                          data-testid="knowledge-agent-association-retry-btn"
                        >
                          {t('knowledge.base.agent-association-retry')}
                        </Button>
                      </div>
                    ) : null}
                    <Table<AgentRecord>
                      rowKey="id"
                      dataSource={associatedAgentRows}
                      pagination={false}
                      size="middle"
                      loading={agentAssociationLoading}
                      virtual={enableAgentVirtualScroll}
                      scroll={enableAgentVirtualScroll ? { y: 520, x: 1040 } : { x: 1040 }}
                      className={styles.sourceTable}
                      data-testid="knowledge-agent-association-table"
                      locale={{ emptyText: t('knowledge.base.agent-association-empty') }}
                      columns={
                        [
                          {
                            title: t('knowledge.base.agent-association-column-name'),
                            dataIndex: 'name',
                            key: 'name',
                            width: 260,
                            render: (_value, agent) => (
                              <Space size={10}>
                                <RobotOutlined className={styles.sourceIcon} />
                                <div>
                                  <Text className={styles.agentName}>{agent.name}</Text>
                                </div>
                              </Space>
                            ),
                          },
                          {
                            title: t('knowledge.base.agent-association-column-description'),
                            key: 'description',
                            width: 320,
                            render: (_value, agent) => (
                              <Text
                                type="secondary"
                                className={styles.agentDescription}
                                ellipsis={{ tooltip: getAgentDescription(agent) }}
                              >
                                {getAgentDescription(agent)}
                              </Text>
                            ),
                          },
                          {
                            title: t('knowledge.base.agent-association-column-status'),
                            key: 'status',
                            width: 140,
                            render: (_value, agent) => (
                              <Tag className={resolveAgentStatusClassName(agent.status)}>
                                {t(resolveAgentStatusKey(agent.status))}
                              </Tag>
                            ),
                          },
                          {
                            title: t('knowledge.base.agent-association-column-knowledge-count'),
                            key: 'knowledgeCount',
                            width: 150,
                            render: (_value, agent) => <Text type="secondary">{getAgentKnowledgeBaseIds(agent).length}</Text>,
                          },
                          {
                            title: t('knowledge.base.agent-association-column-updated-at'),
                            key: 'updatedAt',
                            width: 180,
                            render: (_value, agent) => <Text type="secondary">{getAgentUpdatedText(agent)}</Text>,
                          },
                          {
                            title: t('knowledge.base.agent-association-column-action'),
                            key: 'action',
                            width: 150,
                            render: (_value, agent) => (
                              <ListActionButton
                                action="view"
                                label={t('knowledge.base.agent-association-open')}
                                onClick={() => appNavigator.goTo('agent.chat', {})}
                                data-testid={`knowledge-agent-association-open-${agent.id}`}
                              />
                            ),
                          },
                        ] satisfies TableColumnsType<AgentRecord>
                      }
                    />
                  </div>
                ),
              },
            ]}
          />
        )}
      </div>

      {canAppendSources ? (
        <KnowledgeSourceSelectModal
          open={sourceSelectOpen}
          title={t('knowledge.base.append-source-modal-title')}
          okText={t('knowledge.base.append-source-ok')}
          cancelText={t('knowledge.base.form-cancel-button')}
          submitting={sourceAppending}
          allowedSourceModes={['catalog']}
          testIdPrefix="knowledge-source-select"
          knowledgeBaseId={knowledgeId}
          onCancel={() => {
            if (!sourceAppending) {
              setSourceSelectOpen(false);
            }
          }}
          onSubmit={handleAppendSources}
        />
      ) : null}
      {renderDocumentModal()}
      {renderTableDetailModal()}
      <Modal
        open={segmentPreview.open}
        title={segmentPreview.title}
        footer={null}
        onCancel={closeSegmentPreview}
        className={styles.segmentPreviewModal}
        data-testid="knowledge-document-segment-image-preview-modal"
        destroyOnHidden
      >
        {segmentPreview.url ? (
          <img
            src={segmentPreview.url}
            alt={t('knowledge.base.document-segment-image-preview')}
            className={styles.segmentPreviewImage}
          />
        ) : null}
      </Modal>
      {renderSourceExpiryModal()}
    </div>
  );
}
