import { useCallback, useEffect, useRef, useState } from 'react';

import type { SemanticKind } from '@moi/shared-moi-api/knowledge';
import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { listSemanticEntries, type SemanticEntry, type SemanticEntryListResponse } from '../../../../../service/semanticModel';

interface UseSemanticEntriesOptions {
  http: AppHttpClient;
  modelId: number;
  kind?: SemanticKind;
  pageSize?: number;
}

interface UseSemanticEntriesReturn {
  entries: SemanticEntry[];
  total: number;
  loading: boolean;
  hasMore: boolean;
  /** Reload from the first page */
  refresh: () => Promise<void>;
  /** Load next page (cursor-based) */
  loadMore: () => Promise<void>;
}

/**
 * Hook for managing semantic entries list with cursor-based pagination.
 * Maintains a pageTokenStack to support sequential page loading.
 */
export function useSemanticEntries({ http, modelId, kind, pageSize = 20 }: UseSemanticEntriesOptions): UseSemanticEntriesReturn {
  const [entries, setEntries] = useState<SemanticEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const nextPageTokenRef = useRef('');

  const fetchPage = useCallback(
    async (pageToken: string, append: boolean) => {
      if (!Number.isFinite(modelId) || modelId <= 0) {
        console.warn('[useSemanticEntries] invalid modelId', { modelId });
        return;
      }

      setLoading(true);
      try {
        const result: SemanticEntryListResponse = await listSemanticEntries(http, modelId, {
          kind,
          page_size: pageSize,
          page_token: pageToken || undefined,
        });

        const items = Array.isArray(result.items) ? result.items : [];
        if (append) {
          setEntries((prev) => [...prev, ...items]);
        } else {
          setEntries(items);
        }
        setTotal(result.total);
        nextPageTokenRef.current = result.next_page_token ?? '';
        setHasMore(Boolean(result.next_page_token));
      } catch (error) {
        console.warn('[useSemanticEntries] fetch failed', { modelId, kind, msg: (error as Error).message });
        if (!append) {
          setEntries([]);
          setTotal(0);
        }
        setHasMore(false);
      } finally {
        setLoading(false);
      }
    },
    [http, modelId, kind, pageSize],
  );

  const refresh = useCallback(async () => {
    nextPageTokenRef.current = '';
    await fetchPage('', false);
  }, [fetchPage]);

  const loadMore = useCallback(async () => {
    if (!nextPageTokenRef.current || loading) return;
    await fetchPage(nextPageTokenRef.current, true);
  }, [fetchPage, loading]);

  useEffect(() => {
    nextPageTokenRef.current = '';
    setEntries([]);
    setTotal(0);
    setHasMore(false);
    fetchPage('', false);
  }, [fetchPage]);

  return { entries, total, loading, hasMore, refresh, loadMore };
}
