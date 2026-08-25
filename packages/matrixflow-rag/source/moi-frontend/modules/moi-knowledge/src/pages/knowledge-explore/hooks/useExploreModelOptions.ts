import { useEffect, useState } from 'react';
import { message } from 'antd';
import type { TFunction } from 'i18next';

import { encodeModelValue, isSystemDefaultModel, listModelsApi, type ModelInfo } from '@moi/shared-moi-api/model';
import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { unwrapApiResponse } from '../../../service/http';

export { decodeModelValue, encodeModelValue, isSystemDefaultModel } from '@moi/shared-moi-api/model';

interface UseExploreModelOptionsOptions {
  http: AppHttpClient;
  t: TFunction<'moi-knowledge'>;
}

type ModelSelectInfo = Pick<ModelInfo, 'model' | 'backend_id' | 'backend_name' | 'source' | 'system_default'>;

function displayBackendName(backendName: string | undefined): string {
  const name = backendName?.trim() ?? '';
  if (!name) return '';
  if (name === 'TaaS') return 'Genesis';
  if (name === 'TaaS Embedding') return 'Genesis Embedding';
  return name;
}

export function formatExploreModelLabel(model: ModelSelectInfo, systemDefaultLabel: string): string {
  const suffix = isSystemDefaultModel(model) ? ` (${systemDefaultLabel})` : '';
  const backendName = displayBackendName(model.backend_name);
  if (backendName) {
    return `${backendName} / ${model.model}${suffix}`;
  }
  return `${model.model}${suffix}`;
}

function isChatCompatibleModel(model: ModelInfo): boolean {
  const modelType = (model.model_type || 'chat').toLowerCase();
  // Explore agent runtime needs a generative chat model. Embedding/rerank/OCR
  // models can appear in the unified /models list and must not be selectable.
  return modelType === 'chat' || modelType === 'vision' || modelType === 'reasoning';
}

function normalizeModelList(models: ModelInfo[]): ModelInfo[] {
  const seen = new Set<string>();
  const result: ModelInfo[] = [];
  for (const item of models) {
    if (!isChatCompatibleModel(item)) continue;
    const model = item.model.trim();
    if (!model) continue;
    const key = encodeModelValue(model, item.backend_id);
    if (seen.has(key)) continue;
    seen.add(key);
    result.push({ ...item, model });
  }
  return result.sort((left, right) => {
    // Prefer system-default chat models first so explore does not fall back to
    // an embedding model when no session model is configured.
    const lSystem = isSystemDefaultModel(left) ? 0 : 1;
    const rSystem = isSystemDefaultModel(right) ? 0 : 1;
    if (lSystem !== rSystem) return lSystem - rSystem;
    return left.model.localeCompare(right.model);
  });
}

export function useExploreModelOptions({ http, t }: UseExploreModelOptionsOptions) {
  const [modelList, setModelList] = useState<ModelInfo[]>([]);
  const [isLoading, setIsLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;

    const fetchModelList = async () => {
      setIsLoading(true);
      try {
        const payload = unwrapApiResponse(await listModelsApi(http), 'useExploreModelOptions.fetchModelList');
        if (!cancelled) {
          setModelList(normalizeModelList(Array.isArray(payload.models) ? payload.models : []));
        }
      } catch (error) {
        if (cancelled) {
          return;
        }
        console.warn('[useExploreModelOptions] load model list failed', { msg: (error as Error).message });
        message.error(t('knowledge.explore.model-load-failed'));
        setModelList([]);
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    fetchModelList().catch((error) => {
      console.warn('[useExploreModelOptions] unexpected fetch error', { msg: (error as Error).message });
    });
    return () => {
      cancelled = true;
    };
  }, [http, t]);

  return {
    modelList,
    isLoading,
  };
}
