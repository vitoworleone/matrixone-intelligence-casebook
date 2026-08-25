import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';

import { getCatalogTreeApi, listCatalogFilesApi, type CatalogFileFilter } from '@moi/shared-moi-api/data-connection';
import { useHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import {
  TreeOfFiles as SharedTreeOfFiles,
  type SelectedFileItem,
  type TreeOfFilesCatalogNode,
  type TreeOfFilesFileItem,
  type TreeOfFilesFileSelectMode,
  type TreeOfFilesFilter,
} from '@moi/shared-moi-components/tree-of-files';

interface TreeOfFilesProps {
  onChange?: (selectedFiles: SelectedFileItem[]) => void;
  value?: SelectedFileItem[];
  disableCheckSource?: boolean;
  fetchFilesDataFiltersParam?: CatalogFileFilter[];
  allowSelectTable?: boolean;
  allowSelectVolume?: boolean;
  fileSelectMode?: TreeOfFilesFileSelectMode;
  isFileSelectable?: (item: TreeOfFilesFileItem) => boolean;
  defaultExpandedKeys?: string[];
}

/**
 * Tree selector wrapper for moi-knowledge.
 * Behavior baseline follows moi-connection, with knowledge-specific defaults
 * controlled via props (e.g. allowSelectTable / disableCheckSource).
 */
export function TreeOfFiles({
  onChange,
  value,
  disableCheckSource = false,
  fetchFilesDataFiltersParam = [],
  allowSelectTable = false,
  allowSelectVolume = false,
  fileSelectMode = 'legacy-workflow-output',
  isFileSelectable,
  defaultExpandedKeys = [],
}: TreeOfFilesProps) {
  const { t, i18n } = useTranslation('moi-knowledge');
  const http = useHttpClient();

  const fetchTreeData = useCallback(async (): Promise<TreeOfFilesCatalogNode[]> => {
    const res = await getCatalogTreeApi(http);
    if (res.code !== 'OK' || !res.data) {
      console.warn('[TreeOfFiles] fetch tree failed', { code: res.code });
      throw new Error(res.msg || 'fetch tree failed');
    }
    return (res.data.tree || []) as TreeOfFilesCatalogNode[];
  }, [http]);

  const fetchFilesData = useCallback(
    async (filters: TreeOfFilesFilter[]): Promise<TreeOfFilesFileItem[]> => {
      const res = await listCatalogFilesApi({ page: 1, page_size: 1000, filters: filters as CatalogFileFilter[] }, http);
      if (res.code !== 'OK' || !res.data) {
        console.warn('[TreeOfFiles] load children failed', { code: res.code });
        return [];
      }
      return (res.data.list || []) as TreeOfFilesFileItem[];
    },
    [http],
  );

  return (
    <SharedTreeOfFiles
      onChange={onChange}
      value={value}
      disableCheckSource={disableCheckSource}
      fetchFilesDataFiltersParam={fetchFilesDataFiltersParam}
      allowSelectTable={allowSelectTable}
      allowSelectVolume={allowSelectVolume}
      fileSelectMode={fileSelectMode}
      isFileSelectable={isFileSelectable}
      defaultExpandedKeys={defaultExpandedKeys}
      loadingText={t('knowledge.base.tree-loading')}
      loadFailedText={t('knowledge.base.tree-load-failed')}
      dataTestId="tree-of-files"
      language={i18n.language}
      fetchTreeData={fetchTreeData}
      fetchFilesData={fetchFilesData}
    />
  );
}

export type { SelectedFileItem };
