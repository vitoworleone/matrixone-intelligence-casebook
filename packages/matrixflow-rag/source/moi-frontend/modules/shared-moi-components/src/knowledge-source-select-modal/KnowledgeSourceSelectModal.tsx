import { useCallback, useEffect, useMemo, useRef, useState, type DragEvent, type ReactNode } from 'react';
import {
  App,
  Button,
  Checkbox,
  Input,
  InputNumber,
  Modal,
  Radio,
  Select,
  Space,
  Spin,
  Switch,
  Tooltip,
  Typography,
  Upload,
  type UploadFile,
} from 'antd';
import {
  DatabaseOutlined,
  DeleteOutlined,
  FileExcelOutlined,
  FileOutlined,
  FolderOutlined,
  FormatPainterOutlined,
  InboxOutlined,
  PlusOutlined,
  ReloadOutlined,
  TableOutlined,
  UploadOutlined,
} from '@ant-design/icons';

import { FileType, previewFileApi, type PreviewRow, type PreviewSheet } from '@moi/shared-moi-api/data-connection';
import {
  checkSemanticModelSourceExistenceApi,
  uploadSemanticModelLocalFileApi,
  type CheckSemanticModelSourceExistenceRequest,
  type SemanticModelCreateSource,
  type SemanticModelSourceSelection,
  type SemanticModelSourceSubmitPayload,
} from '@moi/shared-moi-api/knowledge';
import { CatalogDataSelector, type CatalogDataSelectionPreviewStatus } from '@moi/shared-moi-components/catalog-data-selector';
import { DataTypeSelector, isPrimaryKeyType } from '@moi/shared-moi-components/data-type-selector';
import { createValidCatalogIdentifierName, isValidCatalogIdentifier } from '@moi/shared-moi-utils/catalog/identifier';
import { ListActionButton } from '../list-action';
import styles from './KnowledgeSourceSelectModal.module.css';

type KnowledgeSourceMode = 'local' | 'catalog';
type KnowledgeLoadKind = 'unstructured' | 'structured';
type KnowledgeSourceStep = 'basic' | 'source' | 'config';
type KnowledgeStructuredConflict = 0 | 1 | 2;

const DEFAULT_SOURCE_MODES: KnowledgeSourceMode[] = ['local', 'catalog'];
export const KNOWLEDGE_CREATE_MODAL_WIDTH = {
  xs: 'calc(100vw - 24px)',
  sm: 680,
  md: 720,
  lg: 920,
  xl: 1040,
  xxl: 1040,
};
export const KNOWLEDGE_BASE_CATALOG_FILE_EXTENSION_FILTERS: string[] = [
  'pdf',
  'doc',
  'docx',
  'ppt',
  'pptx',
  'xls',
  'xlsx',
  'txt',
  'md',
  'htm',
  'html',
  'eml',
  'msg',
];
const KNOWLEDGE_BASE_CATALOG_FILE_EXTENSIONS = new Set<string>(KNOWLEDGE_BASE_CATALOG_FILE_EXTENSION_FILTERS);

export type KnowledgeSourceSelectModalTranslate = (key: string, params?: Record<string, unknown>) => string;

export interface KnowledgeSourceSelectModalHttpClient {
  post<T = unknown>(url: string, data?: unknown): Promise<{ data: T }>;
}

interface KnowledgeStructuredColumn {
  id: string;
  column: string;
  dataType: string;
  precision?: [number, number?];
  isKey: boolean;
  description: string;
  defaultValue: string;
  colNumInFile: number;
}

interface KnowledgeStructuredConfig {
  activeSheetName: string;
  selectedSheetNames: string[];
  tableName: string;
  tableDescription: string;
  isColumnName: boolean;
  columnNameRow: number;
  rowStart: number;
  conflict: KnowledgeStructuredConflict;
  csv: {
    separator: string;
    delimiter: string;
    isEscape: boolean;
  };
  columns: KnowledgeStructuredColumn[];
}

export interface KnowledgeSourceSelectModalProps {
  open: boolean;
  title: string;
  okText: string;
  cancelText: string;
  http: KnowledgeSourceSelectModalHttpClient;
  translate: KnowledgeSourceSelectModalTranslate;
  language?: string;
  submitting?: boolean;
  showCreateSteps?: boolean;
  basicStepOnly?: boolean;
  basicStepContent?: ReactNode;
  basicNextText?: string;
  basicNextButtonTestId?: string;
  cancelButtonTestId?: string;
  allowedSourceModes?: KnowledgeSourceMode[];
  testIdPrefix: string;
  sourceBackText?: string;
  knowledgeBaseId?: number | string;
  onCancel: () => void;
  onBasicNext?: () => boolean | Promise<boolean>;
  onBack?: () => void;
  onSubmit: (payload: SemanticModelSourceSubmitPayload) => Promise<void> | void;
}

interface KnowledgeBaseCatalogFileCandidate {
  file_ext?: string;
  name?: string;
}

interface KnowledgeStructuredPreviewUploadData {
  conn_file_ids?: string[];
  file_ids?: string[];
}

export function hasDroppedDirectory(dataTransfer: Pick<DataTransfer, 'items'>): boolean {
  return Array.from(dataTransfer.items).some((item) => {
    const entry = item.webkitGetAsEntry?.();
    return entry?.isDirectory === true;
  });
}

function getKnowledgeBaseCatalogFileExt(item: KnowledgeBaseCatalogFileCandidate): string {
  let ext = item.file_ext || '';
  if (!ext && item.name) {
    const extensionStart = item.name.lastIndexOf('.');
    if (extensionStart >= 0 && extensionStart < item.name.length - 1) {
      ext = item.name.slice(extensionStart + 1);
    }
  }
  return ext.startsWith('.') ? ext.slice(1).toLowerCase() : ext.toLowerCase();
}

export function isKnowledgeBaseCatalogFileSelectable(item: KnowledgeBaseCatalogFileCandidate): boolean {
  return KNOWLEDGE_BASE_CATALOG_FILE_EXTENSIONS.has(getKnowledgeBaseCatalogFileExt(item));
}

function createDefaultStructuredConfig(): KnowledgeStructuredConfig {
  return {
    activeSheetName: '',
    selectedSheetNames: [],
    tableName: '',
    tableDescription: '',
    isColumnName: true,
    columnNameRow: 1,
    rowStart: 2,
    conflict: 0,
    csv: {
      separator: ',',
      delimiter: '"',
      isEscape: false,
    },
    columns: [
      {
        id: 'col-1',
        column: 'column_1',
        dataType: 'VARCHAR',
        precision: [255],
        isKey: false,
        description: '',
        defaultValue: '',
        colNumInFile: 1,
      },
      {
        id: 'col-2',
        column: 'column_2',
        dataType: 'VARCHAR',
        precision: [255],
        isKey: false,
        description: '',
        defaultValue: '',
        colNumInFile: 2,
      },
    ],
  };
}

function getStructuredFileType(fileName: string): FileType {
  const lower = fileName.toLowerCase();
  if (lower.endsWith('.csv')) return FileType.CSV;
  if (lower.endsWith('.xls')) return FileType.XLS;
  if (lower.endsWith('.xlsx')) return FileType.XLSX;
  return FileType.Nil;
}

export function createDefaultStructuredTableName(candidateName: string): string {
  return createValidCatalogIdentifierName(candidateName.replace(/\.(csv|xls|xlsx)$/i, ''));
}

async function uploadUnstructuredFile(
  file: UploadFile,
  http: KnowledgeSourceSelectModalHttpClient,
  knowledgeBaseId?: number | string,
): Promise<{ file_name: string; file_id: string }> {
  const originFile = file.originFileObj;
  if (!originFile) {
    throw new Error(`missing file content for ${file.name}`);
  }

  const formData = new FormData();
  formData.append('file', originFile, file.name);
  const modelId = Number(knowledgeBaseId);
  const response = await uploadSemanticModelLocalFileApi(
    formData,
    http as Parameters<typeof uploadSemanticModelLocalFileApi>[1],
    Number.isFinite(modelId) && modelId > 0 ? modelId : undefined,
  );
  const fileID = response.data?.file_id;
  if (response.code !== 'OK' || !fileID) {
    throw new Error(`upload unstructured local file ${file.name}: ${response.msg || 'empty file_id'}`);
  }
  return { file_name: file.name, file_id: fileID };
}

export default function KnowledgeSourceSelectModal({
  open,
  title,
  okText,
  cancelText,
  http,
  translate,
  submitting = false,
  showCreateSteps = false,
  basicStepOnly = false,
  basicStepContent,
  basicNextText,
  basicNextButtonTestId,
  cancelButtonTestId,
  allowedSourceModes = DEFAULT_SOURCE_MODES,
  testIdPrefix,
  sourceBackText,
  knowledgeBaseId,
  onCancel,
  onBasicNext,
  onBack,
  onSubmit,
}: KnowledgeSourceSelectModalProps) {
  const t = translate;
  const { message } = App.useApp();
  const dataTypeCategoryLabels = useMemo(
    () => ({
      INTEGER: t('knowledge.base.create-document-sources-structured-data-type-category-integer'),
      FLOAT: t('knowledge.base.create-document-sources-structured-data-type-category-float'),
      STRING: t('knowledge.base.create-document-sources-structured-data-type-category-string'),
      DATETIME: t('knowledge.base.create-document-sources-structured-data-type-category-datetime'),
      DECIMAL: t('knowledge.base.create-document-sources-structured-data-type-category-decimal'),
      VECTOR: t('knowledge.base.create-document-sources-structured-data-type-category-vector'),
      BOOLEAN: t('knowledge.base.create-document-sources-structured-data-type-category-boolean'),
      OTHER: t('knowledge.base.create-document-sources-structured-data-type-category-other'),
    }),
    [t],
  );

  const allowLocalSource = allowedSourceModes.includes('local');
  const allowCatalogSource = allowedSourceModes.includes('catalog');
  const singleAllowedSourceMode = allowedSourceModes.length === 1 ? allowedSourceModes[0] : null;
  const hasBasicStep = Boolean(showCreateSteps && basicStepContent && onBasicNext);

  const [step, setStep] = useState<KnowledgeSourceStep>(hasBasicStep ? 'basic' : singleAllowedSourceMode ? 'config' : 'source');
  const [sourceMode, setSourceMode] = useState<KnowledgeSourceMode | null>(singleAllowedSourceMode);
  const [loadKind, setLoadKind] = useState<KnowledgeLoadKind>('unstructured');
  const [localFileList, setLocalFileList] = useState<UploadFile[]>([]);
  const [uploadingUnstructuredFiles, setUploadingUnstructuredFiles] = useState(false);
  const [catalogSelections, setCatalogSelections] = useState<SemanticModelSourceSelection[]>([]);
  const [catalogPreviewStatus, setCatalogPreviewStatus] = useState<CatalogDataSelectionPreviewStatus>('ready');
  const [structuredConfig, setStructuredConfig] = useState<KnowledgeStructuredConfig>(() => createDefaultStructuredConfig());
  const [structuredPreviewLoading, setStructuredPreviewLoading] = useState(false);
  const [structuredPreviewConnFileID, setStructuredPreviewConnFileID] = useState('');
  const [structuredPreviewSourceFileID, setStructuredPreviewSourceFileID] = useState('');
  const [structuredPreviewSheets, setStructuredPreviewSheets] = useState<PreviewSheet[]>([]);
  const [structuredPreviewRows, setStructuredPreviewRows] = useState<PreviewRow[]>([]);
  const autoStructuredTableNameRef = useRef('');
  const basicStepContentRef = useRef<HTMLDivElement>(null);
  const activeUploadStepHeadingRef = useRef<HTMLDivElement>(null);

  const focusCurrentStep = useCallback(() => {
    if (step === 'basic') {
      basicStepContentRef.current
        ?.querySelector<HTMLElement>('input:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])')
        ?.focus();
      return;
    }
    activeUploadStepHeadingRef.current?.focus();
  }, [step]);

  const reset = useCallback(() => {
    setStep(hasBasicStep ? 'basic' : singleAllowedSourceMode ? 'config' : 'source');
    setSourceMode(singleAllowedSourceMode);
    setLoadKind('unstructured');
    setLocalFileList([]);
    setUploadingUnstructuredFiles(false);
    setCatalogSelections([]);
    setCatalogPreviewStatus('ready');
    setStructuredConfig(createDefaultStructuredConfig());
    setStructuredPreviewLoading(false);
    setStructuredPreviewConnFileID('');
    setStructuredPreviewSourceFileID('');
    setStructuredPreviewSheets([]);
    setStructuredPreviewRows([]);
    autoStructuredTableNameRef.current = '';
  }, [hasBasicStep, singleAllowedSourceMode]);

  useEffect(() => {
    if (open) {
      reset();
      return;
    }
    if (!open) {
      reset();
    }
  }, [open, reset]);

  useEffect(() => {
    if (!open || !showCreateSteps) return;
    const frame = window.requestAnimationFrame(focusCurrentStep);
    return () => window.cancelAnimationFrame(frame);
  }, [focusCurrentStep, open, showCreateSteps]);

  const handleAfterOpenChange = useCallback(
    (visible: boolean) => {
      if (!visible || !showCreateSteps) return;
      window.requestAnimationFrame(focusCurrentStep);
    },
    [focusCurrentStep, showCreateSteps],
  );

  const updateStructuredConfig = useCallback((patch: Partial<KnowledgeStructuredConfig>) => {
    setStructuredConfig((prev) => ({ ...prev, ...patch }));
  }, []);

  const updateStructuredCsv = useCallback((patch: Partial<KnowledgeStructuredConfig['csv']>) => {
    setStructuredConfig((prev) => ({ ...prev, csv: { ...prev.csv, ...patch } }));
  }, []);

  const updateStructuredColumn = useCallback((id: string, patch: Partial<KnowledgeStructuredColumn>) => {
    setStructuredConfig((prev) => ({
      ...prev,
      columns: prev.columns.map((column) => {
        if (column.id !== id) return column;
        const nextColumn = { ...column, ...patch };
        if (patch.dataType !== undefined && !isPrimaryKeyType(nextColumn.dataType)) nextColumn.isKey = false;
        return nextColumn;
      }),
    }));
  }, []);

  const addStructuredColumn = useCallback(() => {
    setStructuredConfig((prev) => {
      const nextIndex = prev.columns.length + 1;
      return {
        ...prev,
        columns: [
          ...prev.columns,
          {
            id: `col-${Date.now()}-${nextIndex}`,
            column: `column_${nextIndex}`,
            dataType: 'VARCHAR',
            precision: [255],
            isKey: false,
            description: '',
            defaultValue: '',
            colNumInFile: nextIndex,
          },
        ],
      };
    });
  }, []);

  const removeStructuredColumn = useCallback((id: string) => {
    setStructuredConfig((prev) => {
      if (prev.columns.length <= 1) return prev;
      return { ...prev, columns: prev.columns.filter((column) => column.id !== id) };
    });
  }, []);

  const buildStructuredTableConfig = useCallback(
    (connFileIds: string[]) => {
      const tableConfig: Record<string, unknown> = {
        sheet_name: structuredConfig.activeSheetName || localFileList[0]?.name || 'Sheet1',
        new_table: true,
        conn_file_ids: connFileIds,
        isColumnName: structuredConfig.isColumnName,
        columnNameRow: structuredConfig.columnNameRow,
        rowStart: structuredConfig.rowStart,
        conflict: structuredConfig.conflict,
        csv: structuredConfig.csv,
        create_table: {
          name: structuredConfig.tableName,
          description: structuredConfig.tableDescription,
          tableColumn: structuredConfig.columns.map((column) => {
            const parts = (column.precision ?? []).filter((value) => value !== undefined);
            return {
              column: column.column,
              dataType: parts.length > 0 ? `${column.dataType}(${parts.join(',')})` : column.dataType,
              precision: column.precision,
              isKey: column.isKey,
              description: column.description,
              defaultValue: column.defaultValue,
              col_num_in_file: column.colNumInFile,
            };
          }),
        },
      };

      return JSON.stringify(tableConfig);
    },
    [localFileList, structuredConfig],
  );

  const applyStructuredPreviewRows = useCallback((rows: PreviewRow[]) => {
    setStructuredPreviewRows(rows);
    if (rows.length === 0) return;
    setStructuredConfig((prev) => ({
      ...prev,
      columns: rows.map((row) => ({
        id: `preview-${row.number}`,
        column: row.columnName,
        dataType: 'VARCHAR',
        precision: [255],
        isKey: false,
        description: '',
        defaultValue: '',
        colNumInFile: row.number,
      })),
    }));
  }, []);

  const previewStructuredFile = useCallback(
    async (file: UploadFile, sheetName?: string, previewFileID?: string) => {
      const originFile = file.originFileObj;
      if (!originFile) {
        console.warn('[KnowledgeSourceSelectModal] structured preview skipped - missing origin file', { fileName: file.name });
        return;
      }
      setStructuredPreviewLoading(true);
      try {
        let connFileID = previewFileID ?? '';
        if (!connFileID) {
          const formData = new FormData();
          formData.append('files', originFile, file.name);
          const uploadRes = await http.post<{ code: string; msg?: string; data?: KnowledgeStructuredPreviewUploadData }>(
            '/connectors/file/upload',
            formData,
          );
          if (uploadRes.data?.code !== 'OK') {
            console.warn('[KnowledgeSourceSelectModal] structured preview upload non-OK', {
              code: uploadRes.data?.code,
              msg: uploadRes.data?.msg,
            });
            message.error(uploadRes.data?.msg || t('knowledge.base.create-document-sources-structured-preview-failed'));
            return;
          }
          connFileID = uploadRes.data.data?.conn_file_ids?.[0] || '';
          const sourceFileID = uploadRes.data.data?.file_ids?.[0] || '';
          if (!connFileID || !sourceFileID) {
            console.warn('[KnowledgeSourceSelectModal] structured preview upload returned empty conn_file_ids', {
              fileName: file.name,
              hasConnFileID: Boolean(connFileID),
              hasFileID: Boolean(sourceFileID),
            });
            message.error(t('knowledge.base.create-document-sources-structured-preview-failed'));
            return;
          }
          setStructuredPreviewConnFileID(connFileID);
          setStructuredPreviewSourceFileID(sourceFileID);
        }

        const fileType = getStructuredFileType(file.name);
        const previewRes = await previewFileApi(
          {
            connector_id: '',
            conn_file_id: connFileID,
            sheet_name: sheetName,
            isColumnName: structuredConfig.isColumnName,
            rowStart: structuredConfig.rowStart,
            columnNameRow: structuredConfig.isColumnName ? structuredConfig.columnNameRow : undefined,
            file_type: fileType,
            ...(fileType === FileType.CSV ? { csv: structuredConfig.csv } : {}),
          },
          http as Parameters<typeof previewFileApi>[1],
        );
        if (previewRes.code !== 'OK' || !previewRes.data) {
          console.warn('[KnowledgeSourceSelectModal] structured preview non-OK', { code: previewRes.code, msg: previewRes.msg });
          message.error(previewRes.msg || t('knowledge.base.create-document-sources-structured-preview-failed'));
          return;
        }

        const sheets = previewRes.data.sheets ?? [];
        const firstSheetName = sheetName || sheets[0]?.name || file.name;
        const nextAutoTableName = createDefaultStructuredTableName(firstSheetName);
        setStructuredPreviewSheets(sheets);
        setStructuredConfig((prev) => {
          const shouldUpdateTableName = !prev.tableName || prev.tableName === autoStructuredTableNameRef.current;
          if (shouldUpdateTableName) autoStructuredTableNameRef.current = nextAutoTableName;
          return {
            ...prev,
            activeSheetName: firstSheetName,
            selectedSheetNames: prev.selectedSheetNames.length ? prev.selectedSheetNames : [firstSheetName],
            tableName: shouldUpdateTableName ? nextAutoTableName : prev.tableName,
          };
        });
        applyStructuredPreviewRows(previewRes.data.rows ?? []);
      } catch (error) {
        console.warn('[KnowledgeSourceSelectModal] structured preview failed', { message: (error as Error).message });
        message.error(t('knowledge.base.create-document-sources-structured-preview-failed'));
      } finally {
        setStructuredPreviewLoading(false);
      }
    },
    [applyStructuredPreviewRows, http, message, structuredConfig, t],
  );

  const handleSourceSelected = useCallback(
    (mode: KnowledgeSourceMode) => {
      if (!allowedSourceModes.includes(mode)) {
        console.warn('[KnowledgeSourceSelectModal] blocked unsupported source mode', { mode, allowedSourceModes });
        return;
      }
      setSourceMode(mode);
      setStep('config');
    },
    [allowedSourceModes],
  );

  const handleBackToSourceChoice = useCallback(() => {
    setStep('source');
    setSourceMode(null);
    setLocalFileList([]);
    setCatalogSelections([]);
    setCatalogPreviewStatus('ready');
    setLoadKind('unstructured');
    setStructuredConfig(createDefaultStructuredConfig());
    setStructuredPreviewLoading(false);
    setStructuredPreviewConnFileID('');
    setStructuredPreviewSourceFileID('');
    setStructuredPreviewSheets([]);
    setStructuredPreviewRows([]);
  }, []);

  const handleSourceModeChange = useCallback(() => {
    if (singleAllowedSourceMode) {
      setLocalFileList([]);
      setCatalogSelections([]);
      setCatalogPreviewStatus('ready');
      setLoadKind('unstructured');
      setStructuredConfig(createDefaultStructuredConfig());
      setStructuredPreviewLoading(false);
      setStructuredPreviewConnFileID('');
      setStructuredPreviewSourceFileID('');
      setStructuredPreviewSheets([]);
      setStructuredPreviewRows([]);
      return;
    }
    handleBackToSourceChoice();
  }, [handleBackToSourceChoice, singleAllowedSourceMode]);

  const resolveExistingCatalogSources = useCallback(
    async (params: { fileIds: string[]; tableIds: number[] }) => {
      const modelId = Number(knowledgeBaseId);
      if (!Number.isFinite(modelId) || modelId <= 0 || (params.fileIds.length === 0 && params.tableIds.length === 0)) {
        return { fileIds: [], tableIds: [] };
      }
      const req: CheckSemanticModelSourceExistenceRequest = {
        file_ids: params.fileIds,
        table_ids: params.tableIds,
      };
      const res = await checkSemanticModelSourceExistenceApi(modelId, req, http);
      if (res.code !== 'OK' || !res.data) {
        throw new Error(res.msg || 'check semantic model source existence failed');
      }
      return {
        fileIds: res.data.file_ids ?? [],
        tableIds: res.data.table_ids ?? [],
      };
    },
    [http, knowledgeBaseId],
  );

  const handleCatalogSelectionsChange = useCallback((selections: SemanticModelSourceSelection[]) => {
    setCatalogSelections(selections);
    setCatalogPreviewStatus(selections.length > 0 ? 'loading' : 'ready');
  }, []);

  const handleLocalFileChange = useCallback(
    ({ fileList }: { fileList: UploadFile[] }) => {
      const nextFileList = loadKind === 'structured' ? fileList.slice(-1) : fileList;
      setLocalFileList(nextFileList);
      if (loadKind === 'structured' && nextFileList[0]?.name) {
        const defaultTableName = createDefaultStructuredTableName(nextFileList[0].name);
        autoStructuredTableNameRef.current = defaultTableName;
        setStructuredPreviewConnFileID('');
        setStructuredPreviewSourceFileID('');
        setStructuredPreviewSheets([]);
        setStructuredPreviewRows([]);
        setStructuredConfig((prev) => ({
          ...prev,
          tableName: defaultTableName,
          activeSheetName: '',
          selectedSheetNames: [],
        }));
        previewStructuredFile(nextFileList[0], undefined, '').catch((error) => {
          console.warn('[KnowledgeSourceSelectModal] structured preview promise rejected', {
            message: (error as Error).message,
          });
        });
      }
    },
    [loadKind, previewStructuredFile],
  );

  const handleLocalDirectoryDrop = useCallback(
    (event: DragEvent<HTMLDivElement>) => {
      if (!hasDroppedDirectory(event.dataTransfer)) return;

      event.preventDefault();
      event.stopPropagation();
      message.error(t('knowledge.base.local-folder-drop-unsupported'));
    },
    [message, t],
  );

  const handleRemoveLocalFile = useCallback((uid: string) => {
    setLocalFileList((prev) => prev.filter((file) => file.uid !== uid));
    setStructuredPreviewConnFileID('');
    setStructuredPreviewSourceFileID('');
    setStructuredPreviewSheets([]);
    setStructuredPreviewRows([]);
  }, []);

  const buildSubmitPayload = useCallback(async (): Promise<SemanticModelSourceSubmitPayload | null> => {
    if (!sourceMode) {
      message.error(t('knowledge.base.create-document-sources-source-mode-required'));
      return null;
    }

    const hasCatalogSources = sourceMode === 'catalog' && catalogSelections.length > 0;

    if (sourceMode === 'local' && localFileList.length === 0) {
      message.error(t('knowledge.base.create-document-sources-local-file-required'));
      return null;
    }
    if (sourceMode === 'local' && loadKind === 'structured') {
      if (structuredPreviewLoading) {
        message.error(t('knowledge.base.create-document-sources-structured-preview-failed'));
        return null;
      }
      if (!structuredPreviewConnFileID || !structuredPreviewSourceFileID) {
        message.error(t('knowledge.base.create-document-sources-structured-preview-failed'));
        return null;
      }
      if (!structuredConfig.tableName) {
        message.error(t('knowledge.base.create-document-sources-structured-table-name-required'));
        return null;
      }
      if (!isValidCatalogIdentifier(structuredConfig.tableName)) {
        message.error(t('knowledge.base.create-document-sources-structured-table-name-invalid'));
        return null;
      }
      if (structuredConfig.columns.some((column) => !column.column)) {
        message.error(t('knowledge.base.create-document-sources-structured-column-name-required'));
        return null;
      }
    }
    if (sourceMode === 'catalog' && !hasCatalogSources) {
      message.error(t('knowledge.base.create-document-sources-catalog-data-required'));
      return null;
    }

    const localFiles =
      sourceMode === 'local' && loadKind === 'unstructured'
        ? await (async () => {
            // Keep the batch in-flight until every started upload settles.
            const settled = await Promise.allSettled(
              localFileList.map((file) => uploadUnstructuredFile(file, http, knowledgeBaseId)),
            );
            const uploaded: Array<{ file_name: string; file_id: string }> = [];
            let firstError: unknown = null;
            for (const result of settled) {
              if (result.status === 'fulfilled') {
                uploaded.push(result.value);
                continue;
              }
              if (firstError === null) {
                firstError = result.reason;
              }
            }
            if (firstError !== null) {
              throw firstError instanceof Error ? firstError : new Error(String(firstError));
            }
            return uploaded;
          })()
        : [];
    const structuredTableConfig =
      sourceMode === 'local' && loadKind === 'structured' ? buildStructuredTableConfig([structuredPreviewConnFileID]) : '';

    const sources: SemanticModelCreateSource[] =
      sourceMode === 'local' && loadKind === 'structured'
        ? [
            {
              source_type: 'local_file' as const,
              upload_kind: loadKind,
              file_name: localFileList[0].name,
              file_id: structuredPreviewSourceFileID,
              table_config: structuredTableConfig,
            },
          ]
        : localFiles.map((file) => ({
            source_type: 'local_file' as const,
            upload_kind: loadKind,
            ...file,
          }));

    if (sourceMode === 'catalog') {
      // Data domain always uses the workspace's initialized default Catalog;
      // do not pin target_catalog_id to the source file's catalog.
      return { sources: [], source_selections: catalogSelections };
    }

    return { sources };
  }, [
    buildStructuredTableConfig,
    catalogSelections,
    http,
    knowledgeBaseId,
    loadKind,
    localFileList,
    message,
    sourceMode,
    structuredConfig,
    structuredPreviewConnFileID,
    structuredPreviewSourceFileID,
    structuredPreviewLoading,
    t,
  ]);

  const handleSubmit = useCallback(async () => {
    if (submitting || uploadingUnstructuredFiles) {
      console.warn('[KnowledgeSourceSelectModal] duplicate submit blocked', {
        submitting,
        uploadingUnstructuredFiles,
      });
      return;
    }
    if (sourceMode === 'catalog' && catalogPreviewStatus !== 'ready') {
      message.error(t('knowledge.base.catalog-data-selector-preview-count-failed'));
      return;
    }
    const shouldUploadUnstructuredFiles = sourceMode === 'local' && loadKind === 'unstructured';
    if (shouldUploadUnstructuredFiles) setUploadingUnstructuredFiles(true);
    let payload: SemanticModelSourceSubmitPayload | null;
    try {
      payload = await buildSubmitPayload();
    } catch (error) {
      console.warn('[KnowledgeSourceSelectModal] unstructured file upload failed', {
        message: error instanceof Error ? error.message : String(error),
      });
      message.error(t('knowledge.base.create-document-sources-local-file-upload-failed'));
      if (shouldUploadUnstructuredFiles) setUploadingUnstructuredFiles(false);
      return;
    }
    if (!payload) {
      if (shouldUploadUnstructuredFiles) setUploadingUnstructuredFiles(false);
      return;
    }
    try {
      await onSubmit(payload);
    } finally {
      if (shouldUploadUnstructuredFiles) setUploadingUnstructuredFiles(false);
    }
  }, [
    buildSubmitPayload,
    catalogPreviewStatus,
    loadKind,
    message,
    onSubmit,
    sourceMode,
    submitting,
    t,
    uploadingUnstructuredFiles,
  ]);

  const handleBasicNext = useCallback(async () => {
    if (!onBasicNext) return;
    if (await onBasicNext()) {
      setStep(basicStepOnly ? 'basic' : singleAllowedSourceMode ? 'config' : 'source');
    }
  }, [basicStepOnly, onBasicNext, singleAllowedSourceMode]);

  const handleSourceBack = useCallback(() => {
    if (hasBasicStep) {
      setStep('basic');
      return;
    }
    onBack?.();
  }, [hasBasicStep, onBack]);

  const showSourceModeHeader = !singleAllowedSourceMode;
  const busy = submitting || uploadingUnstructuredFiles;
  const structuredTableNameInvalid = Boolean(structuredConfig.tableName) && !isValidCatalogIdentifier(structuredConfig.tableName);
  const formattedStructuredTableName = createValidCatalogIdentifierName(structuredConfig.tableName);

  const localFileRows =
    localFileList.length > 0 ? (
      <div className={styles.localFileList} data-testid={`${testIdPrefix}-local-file-list`}>
        {localFileList.map((file) => (
          <div className={styles.localFileRow} key={file.uid}>
            <span className={styles.localFileName}>
              <FileOutlined />
              {file.name}
            </span>
            <ListActionButton
              action="delete"
              onClick={() => handleRemoveLocalFile(file.uid)}
              disabled={busy}
              label={t('knowledge.base.create-document-sources-local-file-remove')}
              data-testid={`${testIdPrefix}-local-file-remove-btn`}
            />
          </div>
        ))}
      </div>
    ) : null;

  return (
    <Modal
      open={open}
      title={title}
      okText={okText}
      cancelText={cancelText}
      confirmLoading={busy}
      onOk={handleSubmit}
      onCancel={onCancel}
      afterOpenChange={handleAfterOpenChange}
      closable={busy ? false : undefined}
      keyboard={busy ? false : undefined}
      mask={{ closable: busy ? false : undefined }}
      destroyOnHidden
      centered={showCreateSteps}
      width={KNOWLEDGE_CREATE_MODAL_WIDTH}
      className={showCreateSteps ? styles.goldenRatioModal : undefined}
      classNames={
        showCreateSteps
          ? {
              container: styles.goldenRatioModalContainer,
              body: styles.goldenRatioModalBody,
            }
          : undefined
      }
      footer={[
        <Button key="cancel" onClick={onCancel} disabled={busy} data-testid={cancelButtonTestId ?? `${testIdPrefix}-cancel-btn`}>
          {cancelText}
        </Button>,
        step === 'basic' ? (
          <Button
            key="next"
            type="primary"
            onClick={handleBasicNext}
            disabled={busy}
            data-testid={basicNextButtonTestId ?? `${testIdPrefix}-base-next-btn`}
          >
            {basicNextText}
          </Button>
        ) : null,
        step === 'source' && (hasBasicStep || onBack) ? (
          <Button key="previous" onClick={handleSourceBack} disabled={busy} data-testid={`${testIdPrefix}-source-back-btn`}>
            {sourceBackText ?? t('knowledge.base.create-document-sources-previous')}
          </Button>
        ) : null,
        step === 'config' ? (
          <Button
            key="submit"
            type="primary"
            loading={busy}
            disabled={busy || (sourceMode === 'catalog' && catalogPreviewStatus !== 'ready')}
            onClick={handleSubmit}
            data-testid={`${testIdPrefix}-submit-btn`}
          >
            {okText}
          </Button>
        ) : null,
      ]}
      data-testid={`${testIdPrefix}-modal`}
    >
      {showCreateSteps ? (
        <div className={styles.createSteps} data-testid={`${testIdPrefix}-steps`}>
          <div className={`${styles.createStepItem} ${step === 'basic' ? styles.createStepItemActive : ''}`}>
            <span className={styles.createStepNumber}>1</span>
            <Typography.Text>{t('knowledge.base.create-document-sources-step-basic')}</Typography.Text>
          </div>
          {basicStepOnly ? null : (
            <>
              <div className={styles.createStepLine} />
              <div
                ref={step !== 'basic' ? activeUploadStepHeadingRef : undefined}
                className={`${styles.createStepItem} ${step !== 'basic' ? styles.createStepItemActive : ''}`}
                role="heading"
                aria-level={2}
                tabIndex={step !== 'basic' ? -1 : undefined}
                data-testid={step !== 'basic' ? `${testIdPrefix}-active-step-heading` : undefined}
              >
                <span className={styles.createStepNumber}>2</span>
                <Typography.Text>{t('knowledge.base.create-document-sources-step-upload')}</Typography.Text>
              </div>
            </>
          )}
        </div>
      ) : null}

      {step === 'basic' ? (
        <div
          ref={basicStepContentRef}
          key="basic"
          className={styles.stepContent}
          data-testid={`${testIdPrefix}-basic-step-content`}
        >
          {basicStepContent}
        </div>
      ) : null}

      {step === 'source' ? (
        <div
          key="source"
          className={`${styles.sourceChoice} ${styles.stepContent} ${
            allowLocalSource && allowCatalogSource ? styles.sourceChoiceInteractive : styles.sourceChoiceSingle
          }`}
          data-testid={`${testIdPrefix}-source-choice`}
        >
          {allowLocalSource ? (
            <Button
              className={`${styles.sourceChoiceCard} ${styles.sourceChoiceCardLocal}`}
              onClick={() => handleSourceSelected('local')}
              data-testid={`${testIdPrefix}-source-choice-local`}
            >
              <span className={`${styles.sourceChoiceIcon} ${styles.sourceChoiceIconLocal}`}>
                <UploadOutlined />
              </span>
              <span className={styles.sourceChoiceCopy}>
                <Typography.Text strong className={styles.sourceChoiceTitle}>
                  {t('knowledge.base.create-document-sources-source-mode-local')}
                </Typography.Text>
                <span className={styles.sourceChoiceDescriptionReveal}>
                  <Typography.Text className={styles.sourceChoiceDescription}>
                    {t('knowledge.base.create-document-sources-source-mode-local-desc')}
                  </Typography.Text>
                </span>
              </span>
            </Button>
          ) : null}
          {allowCatalogSource ? (
            <Button
              className={`${styles.sourceChoiceCard} ${styles.sourceChoiceCardCatalog}`}
              onClick={() => handleSourceSelected('catalog')}
              data-testid={`${testIdPrefix}-source-choice-catalog`}
            >
              <span className={`${styles.sourceChoiceIcon} ${styles.sourceChoiceIconCatalog}`}>
                <DatabaseOutlined />
              </span>
              <span className={styles.sourceChoiceCopy}>
                <Typography.Text strong className={styles.sourceChoiceTitle}>
                  {t('knowledge.base.create-document-sources-source-mode-catalog')}
                </Typography.Text>
                <span className={styles.sourceChoiceDescriptionReveal}>
                  <Typography.Text className={styles.sourceChoiceDescription}>
                    {t('knowledge.base.create-document-sources-source-mode-catalog-desc')}
                  </Typography.Text>
                </span>
              </span>
            </Button>
          ) : null}
        </div>
      ) : null}

      {step === 'config' ? (
        <div key="config" className={`${styles.createModalBody} ${styles.stepContent}`}>
          <div className={styles.createFlow}>
            {showSourceModeHeader ? (
              <div className={styles.createFlowHeader}>
                <Space size="small">
                  <Typography.Text strong>{t('knowledge.base.create-document-sources-source-mode-label')}</Typography.Text>
                  <Button
                    type="link"
                    size="small"
                    onClick={handleSourceModeChange}
                    disabled={busy}
                    data-testid={`${testIdPrefix}-source-mode-change-btn`}
                  >
                    {t('knowledge.base.create-document-sources-source-mode-change')}
                  </Button>
                </Space>
                <Typography.Text className={styles.sourceModePill} data-testid={`${testIdPrefix}-source-mode-summary`}>
                  {sourceMode === 'local'
                    ? t('knowledge.base.create-document-sources-source-mode-local')
                    : t('knowledge.base.create-document-sources-source-mode-catalog')}
                </Typography.Text>
              </div>
            ) : null}

            {sourceMode === 'local' ? (
              <>
                <div
                  className={styles.loadKindCards}
                  role="tablist"
                  aria-label={t('knowledge.base.create-document-sources-load-kind-label')}
                >
                  <Button
                    className={`${styles.loadKindCard} ${loadKind === 'unstructured' ? styles.loadKindCardActive : ''}`}
                    onClick={() => setLoadKind('unstructured')}
                    disabled={busy}
                    data-testid={`${testIdPrefix}-load-kind-unstructured`}
                  >
                    <FileOutlined className={styles.loadKindIcon} />
                    <span className={styles.loadKindText}>
                      <Typography.Text strong>
                        {t('knowledge.base.create-document-sources-load-kind-unstructured')}
                      </Typography.Text>
                      <Typography.Text type="secondary">
                        {t('knowledge.base.create-document-sources-load-kind-unstructured-desc')}
                      </Typography.Text>
                    </span>
                  </Button>
                  <Button
                    className={`${styles.loadKindCard} ${loadKind === 'structured' ? styles.loadKindCardActive : ''}`}
                    onClick={() => setLoadKind('structured')}
                    disabled={busy}
                    data-testid={`${testIdPrefix}-load-kind-structured`}
                  >
                    <TableOutlined className={styles.loadKindIcon} />
                    <span className={styles.loadKindText}>
                      <Typography.Text strong>{t('knowledge.base.create-document-sources-load-kind-structured')}</Typography.Text>
                      <Typography.Text type="secondary">
                        {t('knowledge.base.create-document-sources-load-kind-structured-desc')}
                      </Typography.Text>
                    </span>
                  </Button>
                </div>

                <div className={styles.sourcePanel} data-testid={`${testIdPrefix}-local-source-panel`}>
                  {loadKind === 'structured' ? (
                    <div className={styles.structuredImportPanel} data-testid={`${testIdPrefix}-local-structured-import-panel`}>
                      <div className={styles.structuredSourceTargetGrid}>
                        <div className={styles.structuredColumn}>
                          <Typography.Text strong className={styles.structuredPanelTitle}>
                            {t('knowledge.base.create-document-sources-structured-data-source')}
                          </Typography.Text>
                          <div onDropCapture={handleLocalDirectoryDrop} data-testid={`${testIdPrefix}-local-files-drop-zone`}>
                            <Upload.Dragger
                              maxCount={1}
                              fileList={localFileList}
                              beforeUpload={() => false}
                              showUploadList={false}
                              accept=".csv,.xls,.xlsx"
                              onChange={handleLocalFileChange}
                              disabled={busy}
                              data-testid={`${testIdPrefix}-local-files-dragger`}
                            >
                              <p className="ant-upload-drag-icon">
                                <InboxOutlined />
                              </p>
                              <p className={styles.uploadTip}>
                                {t('knowledge.base.create-document-sources-local-structured-upload-text')}
                              </p>
                              <Typography.Text type="secondary">
                                {t('knowledge.base.create-document-sources-local-structured-upload-hint')}
                              </Typography.Text>
                            </Upload.Dragger>
                          </div>
                          <div className={styles.uploadButtons}>
                            <Upload
                              maxCount={1}
                              fileList={localFileList}
                              beforeUpload={() => false}
                              showUploadList={false}
                              accept=".csv,.xls,.xlsx"
                              onChange={handleLocalFileChange}
                              disabled={submitting}
                            >
                              <Button
                                type="primary"
                                icon={<FileOutlined />}
                                disabled={submitting}
                                data-testid={`${testIdPrefix}-local-files-upload-btn`}
                              >
                                {t('knowledge.base.create-document-sources-local-files-button')}
                              </Button>
                            </Upload>
                          </div>
                          {localFileRows}
                          {localFileList.length > 0 ? (
                            <div
                              className={styles.structuredPreviewPanel}
                              data-testid={`${testIdPrefix}-structured-preview-panel`}
                            >
                              <div className={styles.structuredSectionHeader}>
                                <Typography.Text strong>
                                  {t('knowledge.base.create-document-sources-structured-file-parsing')}
                                </Typography.Text>
                                <Tooltip title={t('knowledge.base.create-document-sources-structured-preview-refresh')}>
                                  <Button
                                    type="text"
                                    size="small"
                                    icon={<ReloadOutlined />}
                                    loading={structuredPreviewLoading}
                                    aria-label={t('knowledge.base.create-document-sources-structured-preview-refresh')}
                                    onClick={() =>
                                      previewStructuredFile(localFileList[0], undefined, structuredPreviewConnFileID).catch(
                                        (error) => {
                                          console.warn('[KnowledgeSourceSelectModal] structured preview refresh rejected', {
                                            message: (error as Error).message,
                                          });
                                        },
                                      )
                                    }
                                    data-testid={`${testIdPrefix}-structured-preview-refresh-btn`}
                                  />
                                </Tooltip>
                              </div>
                              {structuredPreviewSheets.length > 0 ? (
                                <div className={styles.structuredSheetList} data-testid={`${testIdPrefix}-structured-sheet-list`}>
                                  <Typography.Text type="secondary">
                                    {t('knowledge.base.create-document-sources-structured-sheet-hint')}
                                  </Typography.Text>
                                  <div className={styles.structuredSheetButtons}>
                                    {structuredPreviewSheets.map((sheet) => {
                                      const active = structuredConfig.activeSheetName === sheet.name;
                                      return (
                                        <Button
                                          key={sheet.name}
                                          size="small"
                                          type={active ? 'primary' : 'default'}
                                          ghost={active}
                                          icon={<FileExcelOutlined />}
                                          onClick={() =>
                                            previewStructuredFile(
                                              localFileList[0],
                                              sheet.name,
                                              structuredPreviewConnFileID,
                                            ).catch((error) => {
                                              console.warn('[KnowledgeSourceSelectModal] structured sheet preview rejected', {
                                                message: (error as Error).message,
                                              });
                                            })
                                          }
                                          data-testid={`${testIdPrefix}-structured-sheet-btn`}
                                        >
                                          {t('knowledge.base.create-document-sources-structured-sheet-button', {
                                            name: sheet.name,
                                            rows: sheet.row_count,
                                          })}
                                        </Button>
                                      );
                                    })}
                                  </div>
                                </div>
                              ) : null}
                              <Spin spinning={structuredPreviewLoading}>
                                {structuredPreviewRows.length > 0 ? (
                                  <div className={styles.structuredPreviewRows}>
                                    {structuredPreviewRows.map((row) => (
                                      <div className={styles.structuredPreviewRow} key={row.number}>
                                        <span className={styles.structuredPreviewIndex}>{row.charNumber}</span>
                                        <Typography.Text ellipsis={{ tooltip: row.columnName }}>{row.columnName}</Typography.Text>
                                        <div className={styles.structuredPreviewValues}>
                                          {(row.columnValues ?? []).slice(0, 3).map((value, index) => (
                                            <Typography.Text
                                              key={`${row.number}-${index}-${value}`}
                                              className={styles.structuredPreviewValue}
                                              ellipsis={{ tooltip: value }}
                                            >
                                              {value}
                                            </Typography.Text>
                                          ))}
                                        </div>
                                      </div>
                                    ))}
                                  </div>
                                ) : (
                                  <Typography.Text type="secondary">
                                    {t('knowledge.base.create-document-sources-structured-preview-empty')}
                                  </Typography.Text>
                                )}
                              </Spin>
                            </div>
                          ) : null}
                        </div>
                        <div className={styles.structuredColumn}>
                          <Typography.Text strong className={styles.structuredPanelTitle}>
                            {t('knowledge.base.create-document-sources-structured-target-title')}
                          </Typography.Text>
                          <div className={styles.structuredFieldGrid}>
                            <label className={styles.structuredField}>
                              <span className={styles.structuredFieldLabel}>
                                {t('knowledge.base.create-document-sources-structured-table-name')}
                              </span>
                              <Space.Compact block>
                                <Input
                                  value={structuredConfig.tableName}
                                  onChange={(event) => updateStructuredConfig({ tableName: event.target.value })}
                                  placeholder={t('knowledge.base.create-document-sources-structured-table-name-placeholder')}
                                  maxLength={255}
                                  status={structuredTableNameInvalid ? 'error' : undefined}
                                  disabled={submitting}
                                  data-testid={`${testIdPrefix}-structured-table-name-input`}
                                />
                                <Tooltip title={t('knowledge.base.create-document-sources-structured-table-name-format')}>
                                  <Button
                                    icon={<FormatPainterOutlined />}
                                    aria-label={t('knowledge.base.create-document-sources-structured-table-name-format')}
                                    disabled={submitting || formattedStructuredTableName === structuredConfig.tableName}
                                    onClick={() => updateStructuredConfig({ tableName: formattedStructuredTableName })}
                                    data-testid={`${testIdPrefix}-structured-table-name-format-btn`}
                                  />
                                </Tooltip>
                              </Space.Compact>
                              {structuredTableNameInvalid ? (
                                <span className={styles.structuredFieldError}>
                                  {t('knowledge.base.create-document-sources-structured-table-name-invalid')}
                                </span>
                              ) : null}
                            </label>
                            <label className={styles.structuredField}>
                              <span className={styles.structuredFieldLabel}>
                                {t('knowledge.base.create-document-sources-structured-table-description')}
                              </span>
                              <Input
                                value={structuredConfig.tableDescription}
                                onChange={(event) => updateStructuredConfig({ tableDescription: event.target.value })}
                                placeholder={t('knowledge.base.create-document-sources-structured-table-description-placeholder')}
                                disabled={submitting}
                                data-testid={`${testIdPrefix}-structured-table-description-input`}
                              />
                            </label>
                          </div>
                          <Typography.Text type="secondary">
                            {t('knowledge.base.create-document-sources-structured-target-default-database-hint')}
                          </Typography.Text>
                        </div>
                      </div>
                      <div className={styles.structuredSection} data-testid={`${testIdPrefix}-structured-table-definition`}>
                        <div className={styles.structuredSectionHeader}>
                          <Typography.Text strong>
                            {t('knowledge.base.create-document-sources-structured-table-definition')}
                          </Typography.Text>
                          <Button
                            size="small"
                            icon={<PlusOutlined />}
                            onClick={addStructuredColumn}
                            disabled={submitting}
                            data-testid={`${testIdPrefix}-structured-add-column-btn`}
                          >
                            {t('knowledge.base.create-document-sources-structured-add-column')}
                          </Button>
                        </div>
                        <div className={styles.structuredColumnHeader}>
                          <span>{t('knowledge.base.create-document-sources-structured-column-name')}</span>
                          <span>{t('knowledge.base.create-document-sources-structured-column-type')}</span>
                          <span>{t('knowledge.base.create-document-sources-structured-column-primary-key')}</span>
                          <span>{t('knowledge.base.create-document-sources-structured-column-description')}</span>
                          <span>{t('knowledge.base.create-document-sources-structured-column-default')}</span>
                          <span>{t('knowledge.base.create-document-sources-structured-column-action')}</span>
                        </div>
                        <div className={styles.structuredColumnRows}>
                          {structuredConfig.columns.map((column) => (
                            <div className={styles.structuredColumnRow} key={column.id}>
                              <label className={styles.structuredColumnField}>
                                <span className={styles.structuredColumnMobileLabel}>
                                  {t('knowledge.base.create-document-sources-structured-column-name')}
                                </span>
                                <Input
                                  value={column.column}
                                  onChange={(event) => updateStructuredColumn(column.id, { column: event.target.value })}
                                  placeholder={t('knowledge.base.create-document-sources-structured-column-name-placeholder')}
                                  disabled={submitting}
                                  data-testid={`${testIdPrefix}-structured-column-name-input`}
                                />
                              </label>
                              <label className={styles.structuredColumnField}>
                                <span className={styles.structuredColumnMobileLabel}>
                                  {t('knowledge.base.create-document-sources-structured-column-type')}
                                </span>
                                <div data-testid={`${testIdPrefix}-structured-column-type-select`}>
                                  <DataTypeSelector
                                    value={column.dataType}
                                    precision={column.precision}
                                    onChange={(dataType) => updateStructuredColumn(column.id, { dataType })}
                                    onPrecisionChange={(precision) => updateStructuredColumn(column.id, { precision })}
                                    categoryLabels={dataTypeCategoryLabels}
                                    placeholder={t('knowledge.base.create-document-sources-structured-data-type-placeholder')}
                                    disabled={submitting}
                                  />
                                </div>
                              </label>
                              <div className={styles.structuredColumnField}>
                                <span className={styles.structuredColumnMobileLabel}>
                                  {t('knowledge.base.create-document-sources-structured-column-primary-key')}
                                </span>
                                <Checkbox
                                  checked={isPrimaryKeyType(column.dataType) && column.isKey}
                                  onChange={(event) => updateStructuredColumn(column.id, { isKey: event.target.checked })}
                                  disabled={submitting || !isPrimaryKeyType(column.dataType)}
                                  aria-label={t('knowledge.base.create-document-sources-structured-column-primary-key')}
                                  data-testid={`${testIdPrefix}-structured-column-key-checkbox`}
                                />
                              </div>
                              <label className={styles.structuredColumnField}>
                                <span className={styles.structuredColumnMobileLabel}>
                                  {t('knowledge.base.create-document-sources-structured-column-description')}
                                </span>
                                <Input
                                  value={column.description}
                                  onChange={(event) => updateStructuredColumn(column.id, { description: event.target.value })}
                                  placeholder={t(
                                    'knowledge.base.create-document-sources-structured-column-description-placeholder',
                                  )}
                                  disabled={submitting}
                                  data-testid={`${testIdPrefix}-structured-column-description-input`}
                                />
                              </label>
                              <label className={styles.structuredColumnField}>
                                <span className={styles.structuredColumnMobileLabel}>
                                  {t('knowledge.base.create-document-sources-structured-column-default')}
                                </span>
                                <Input
                                  value={column.defaultValue}
                                  onChange={(event) => updateStructuredColumn(column.id, { defaultValue: event.target.value })}
                                  placeholder={t('knowledge.base.create-document-sources-structured-column-default-placeholder')}
                                  disabled={submitting}
                                  data-testid={`${testIdPrefix}-structured-column-default-input`}
                                />
                              </label>
                              <div className={styles.structuredColumnField}>
                                <span className={styles.structuredColumnMobileLabel}>
                                  {t('knowledge.base.create-document-sources-structured-delete-column')}
                                </span>
                                <Button
                                  type="text"
                                  size="small"
                                  icon={<DeleteOutlined />}
                                  onClick={() => removeStructuredColumn(column.id)}
                                  disabled={submitting || structuredConfig.columns.length <= 1}
                                  aria-label={t('knowledge.base.create-document-sources-structured-delete-column')}
                                  data-testid={`${testIdPrefix}-structured-delete-column-btn`}
                                />
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                      <div className={styles.structuredConfigGrid}>
                        <div className={styles.structuredSection} data-testid={`${testIdPrefix}-structured-import-data-config`}>
                          <Typography.Text strong>
                            {t('knowledge.base.create-document-sources-structured-import-data-config')}
                          </Typography.Text>
                          <div className={styles.structuredConfigRows}>
                            <label className={styles.structuredConfigRow}>
                              <span>{t('knowledge.base.create-document-sources-structured-enable-column-name')}</span>
                              <Switch
                                checked={structuredConfig.isColumnName}
                                onChange={(checked) => updateStructuredConfig({ isColumnName: checked })}
                                disabled={submitting}
                                data-testid={`${testIdPrefix}-structured-column-name-switch`}
                              />
                            </label>
                            {structuredConfig.isColumnName ? (
                              <label className={styles.structuredConfigRow}>
                                <span>{t('knowledge.base.create-document-sources-structured-column-name-row')}</span>
                                <InputNumber
                                  min={1}
                                  max={20}
                                  value={structuredConfig.columnNameRow}
                                  onChange={(value) => updateStructuredConfig({ columnNameRow: Number(value ?? 1) })}
                                  disabled={submitting}
                                  data-testid={`${testIdPrefix}-structured-column-name-row-input`}
                                />
                              </label>
                            ) : null}
                            <label className={styles.structuredConfigRow}>
                              <span>{t('knowledge.base.create-document-sources-structured-row-start')}</span>
                              <InputNumber
                                min={1}
                                value={structuredConfig.rowStart}
                                onChange={(value) => updateStructuredConfig({ rowStart: Number(value ?? 1) })}
                                disabled={submitting}
                                data-testid={`${testIdPrefix}-structured-row-start-input`}
                              />
                            </label>
                            <label className={styles.structuredConfigRow}>
                              <span>{t('knowledge.base.create-document-sources-structured-conflict')}</span>
                              <Radio.Group
                                value={structuredConfig.conflict}
                                onChange={(event) =>
                                  updateStructuredConfig({ conflict: event.target.value as KnowledgeStructuredConflict })
                                }
                                disabled={submitting}
                                data-testid={`${testIdPrefix}-structured-conflict-radio`}
                              >
                                <Radio value={0}>{t('knowledge.base.create-document-sources-structured-conflict-fail')}</Radio>
                                <Radio value={1}>{t('knowledge.base.create-document-sources-structured-conflict-skip')}</Radio>
                                <Radio value={2}>{t('knowledge.base.create-document-sources-structured-conflict-replace')}</Radio>
                              </Radio.Group>
                            </label>
                          </div>
                        </div>
                        <div className={styles.structuredSection} data-testid={`${testIdPrefix}-structured-import-table-config`}>
                          <Typography.Text strong>
                            {t('knowledge.base.create-document-sources-structured-csv-config')}
                          </Typography.Text>
                          <div className={styles.structuredCsvGrid}>
                            <label className={styles.structuredField}>
                              <span className={styles.structuredFieldLabel}>
                                {t('knowledge.base.create-document-sources-structured-csv-separator')}
                              </span>
                              <Select
                                value={structuredConfig.csv.separator}
                                onChange={(value) => updateStructuredCsv({ separator: value })}
                                disabled={submitting}
                                data-testid={`${testIdPrefix}-structured-csv-separator-select`}
                                options={[
                                  {
                                    label: t('knowledge.base.create-document-sources-structured-csv-separator-comma'),
                                    value: ',',
                                  },
                                  {
                                    label: t('knowledge.base.create-document-sources-structured-csv-separator-semicolon'),
                                    value: ';',
                                  },
                                  {
                                    label: t('knowledge.base.create-document-sources-structured-csv-separator-tab'),
                                    value: '\\t',
                                  },
                                  {
                                    label: t('knowledge.base.create-document-sources-structured-csv-separator-pipe'),
                                    value: '|',
                                  },
                                ]}
                              />
                            </label>
                            <label className={styles.structuredField}>
                              <span className={styles.structuredFieldLabel}>
                                {t('knowledge.base.create-document-sources-structured-csv-delimiter')}
                              </span>
                              <Select
                                value={structuredConfig.csv.delimiter}
                                onChange={(value) => updateStructuredCsv({ delimiter: value })}
                                disabled={submitting}
                                data-testid={`${testIdPrefix}-structured-csv-delimiter-select`}
                                options={[
                                  {
                                    label: t('knowledge.base.create-document-sources-structured-csv-delimiter-double-quote'),
                                    value: '"',
                                  },
                                  {
                                    label: t('knowledge.base.create-document-sources-structured-csv-delimiter-single-quote'),
                                    value: "'",
                                  },
                                  { label: t('knowledge.base.create-document-sources-structured-csv-delimiter-none'), value: '' },
                                ]}
                              />
                            </label>
                            <label className={styles.structuredConfigRow}>
                              <span>{t('knowledge.base.create-document-sources-structured-csv-escape')}</span>
                              <Switch
                                checked={structuredConfig.csv.isEscape}
                                onChange={(checked) => updateStructuredCsv({ isEscape: checked })}
                                disabled={submitting}
                                data-testid={`${testIdPrefix}-structured-csv-escape-switch`}
                              />
                            </label>
                          </div>
                        </div>
                      </div>
                    </div>
                  ) : (
                    <>
                      <div onDropCapture={handleLocalDirectoryDrop} data-testid={`${testIdPrefix}-local-files-drop-zone`}>
                        <Upload.Dragger
                          multiple
                          fileList={localFileList}
                          beforeUpload={() => false}
                          showUploadList={false}
                          onChange={handleLocalFileChange}
                          disabled={busy}
                          data-testid={`${testIdPrefix}-local-files-dragger`}
                        >
                          <p className="ant-upload-drag-icon">
                            <InboxOutlined />
                          </p>
                          <p className={styles.uploadTip}>
                            {t('knowledge.base.create-document-sources-local-unstructured-upload-text')}
                          </p>
                          <Typography.Text type="secondary">
                            {t('knowledge.base.create-document-sources-local-unstructured-upload-hint')}
                          </Typography.Text>
                        </Upload.Dragger>
                      </div>
                      <div className={styles.uploadButtons}>
                        <Upload
                          multiple
                          fileList={localFileList}
                          beforeUpload={() => false}
                          showUploadList={false}
                          onChange={handleLocalFileChange}
                          disabled={busy}
                        >
                          <Button
                            type="primary"
                            icon={<FileOutlined />}
                            disabled={busy}
                            data-testid={`${testIdPrefix}-local-files-upload-btn`}
                          >
                            {t('knowledge.base.create-document-sources-local-files-button')}
                          </Button>
                        </Upload>
                        <Upload
                          multiple
                          directory
                          fileList={localFileList}
                          beforeUpload={() => false}
                          showUploadList={false}
                          onChange={handleLocalFileChange}
                          disabled={busy}
                        >
                          <Button
                            icon={<FolderOutlined />}
                            disabled={busy}
                            data-testid={`${testIdPrefix}-local-folder-upload-btn`}
                          >
                            {t('knowledge.base.create-document-sources-local-folder-button')}
                          </Button>
                        </Upload>
                      </div>
                      {localFileRows}
                    </>
                  )}
                </div>
              </>
            ) : (
              <div className={styles.sourcePanel} data-testid={`${testIdPrefix}-catalog-source-panel`}>
                <div className={styles.catalogSelectorHeader}>
                  <div className={styles.catalogSelectorTitleBlock}>
                    <Typography.Text strong>{t('knowledge.base.create-document-sources-catalog-data-label')}</Typography.Text>
                    <Typography.Text type="secondary">
                      {t('knowledge.base.create-document-sources-catalog-data-hint')}
                    </Typography.Text>
                  </div>
                </div>
                <div className={styles.catalogTreePanel} data-testid={`${testIdPrefix}-catalog-tree-panel`}>
                  <CatalogDataSelector
                    http={http}
                    translate={t}
                    value={catalogSelections}
                    onChange={handleCatalogSelectionsChange}
                    disabled={submitting}
                    testIdPrefix={`${testIdPrefix}-catalog-data`}
                    resolveExistingSources={knowledgeBaseId ? resolveExistingCatalogSources : undefined}
                    isFileSelectable={isKnowledgeBaseCatalogFileSelectable}
                    selectionFileExtFilters={KNOWLEDGE_BASE_CATALOG_FILE_EXTENSION_FILTERS}
                    knowledgeBaseId={Number(knowledgeBaseId) > 0 ? Number(knowledgeBaseId) : undefined}
                    onPreviewStatusChange={setCatalogPreviewStatus}
                  />
                </div>
              </div>
            )}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}
