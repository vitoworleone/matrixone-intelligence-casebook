import { useEffect, useState } from 'react';
import { message } from 'antd';
import type { TFunction } from 'i18next';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { queryKnowledgeList, type KnowledgeListItem } from '../../../service/knowledge';

interface UseExploreKnowledgeOptionsOptions {
  http: AppHttpClient;
  t: TFunction<'moi-knowledge'>;
}

const PAGE_SIZE = 100;

export function useExploreKnowledgeOptions({ http, t }: UseExploreKnowledgeOptionsOptions) {
  const [knowledgeList, setKnowledgeList] = useState<KnowledgeListItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;

    const fetchAllPages = async () => {
      setIsLoading(true);
      try {
        const allItems: KnowledgeListItem[] = [];

        const fetchPage = async (pageToken?: string): Promise<void> => {
          const response = await queryKnowledgeList(http, { page_size: PAGE_SIZE, page_token: pageToken });
          if (cancelled) return;

          allItems.push(...(response.items || []));
          const nextToken = response.next_page_token || undefined;
          if (nextToken) {
            return fetchPage(nextToken);
          }
        };

        await fetchPage();

        if (!cancelled) {
          setKnowledgeList(allItems);
        }
      } catch (error) {
        if (cancelled) return;
        console.warn('[useExploreKnowledgeOptions] load knowledge list failed', { msg: (error as Error).message });
        message.error(t('knowledge.explore.knowledge-load-failed'));
        setKnowledgeList([]);
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    fetchAllPages();
    return () => {
      cancelled = true;
    };
  }, [http, t]);

  return {
    knowledgeList,
    isLoading,
  };
}
