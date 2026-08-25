import { useCallback } from 'react';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { getSession, updateSession } from '../../../service/dialogSession';
import {
  isSameSemanticModelRefs,
  normalizeSemanticModelRefs,
  normalizeSessionModel,
  normalizeSessionModelBackendId,
  normalizeSessionReasoningEffort,
  type SemanticModelRef,
} from '../utils/sessionConfig';

/**
 * Persists semantic-model and model selections to the session config
 * before sending a message.
 *
 * Fetches the latest config from the server as the merge baseline to
 * avoid overwriting changes made by other clients.
 */
export function useSessionConfigPersistence(http: AppHttpClient) {
  return useCallback(
    async ({
      sessionId,
      selectedKnowledgeIds,
      selectedModel,
      selectedModelBackendId,
      appContext,
      selectedReasoningEffort,
    }: {
      sessionId: number;
      selectedKnowledgeIds: number[];
      selectedModel: string;
      selectedModelBackendId?: number | null;
      appContext?: Record<string, unknown>;
      selectedReasoningEffort?: string;
    }) => {
      let freshSession;
      try {
        freshSession = await getSession(http, sessionId);
      } catch (error) {
        throw new Error(`fetch session ${sessionId} before saving explore config failed: ${(error as Error).message}`);
      }

      let baseConfig: Record<string, unknown>;
      try {
        baseConfig = parseSessionConfigStrict(sessionId, freshSession.config);
      } catch (error) {
        throw new Error(`parse session ${sessionId} config before saving explore config failed: ${(error as Error).message}`);
      }

      const nextRefs: SemanticModelRef[] = normalizeSemanticModelRefs(
        selectedKnowledgeIds.map((id) => ({ semantic_model_id: id })),
      );
      if (nextRefs.length !== selectedKnowledgeIds.length) {
        throw new Error(`save session ${sessionId} explore config failed: selected semantic model ids are invalid`);
      }

      const prevRefs = normalizeSemanticModelRefs(baseConfig.semantic_models);
      const normalizedSelectedModel = selectedModel.trim();
      if (!normalizedSelectedModel) {
        throw new Error(`save session ${sessionId} explore config failed: selected model is empty`);
      }
      const previousModel = normalizeSessionModel(baseConfig);

      const previousBackendId = normalizeSessionModelBackendId(baseConfig);
      const previousReasoningEffort = normalizeSessionReasoningEffort(baseConfig);
      const normalizedBackendId =
        Number.isInteger(selectedModelBackendId) && selectedModelBackendId !== 0 ? selectedModelBackendId : null;
      const nextReasoningEffort = selectedReasoningEffort ?? '';
      if (
        isSameSemanticModelRefs(prevRefs, nextRefs) &&
        previousModel === normalizedSelectedModel &&
        previousBackendId === normalizedBackendId &&
        previousReasoningEffort === nextReasoningEffort &&
        appContext === undefined
      ) {
        return;
      }

      const nextLlmConfig =
        baseConfig.llm && typeof baseConfig.llm === 'object'
          ? { ...(baseConfig.llm as Record<string, unknown>) }
          : ({} as Record<string, unknown>);
      nextLlmConfig.model = normalizedSelectedModel;
      if (normalizedBackendId !== null) {
        nextLlmConfig.backend_id = normalizedBackendId;
      } else {
        delete nextLlmConfig.backend_id;
      }
      if (nextReasoningEffort !== '') {
        nextLlmConfig.reasoning_effort = nextReasoningEffort;
      } else {
        delete nextLlmConfig.reasoning_effort;
      }

      const nextConfig: Record<string, unknown> = {
        ...baseConfig,
        semantic_models: nextRefs,
        llm: nextLlmConfig,
        model: normalizedSelectedModel,
      };
      if (appContext !== undefined) {
        nextConfig.app_context = appContext;
      }
      delete nextConfig.llm_backend_id;
      delete nextConfig.backend_id;
      delete nextConfig.llm_reasoning_effort;
      delete nextConfig.reasoning_effort;

      // Clean up legacy fields that may have been carried over from baseConfig merge
      delete nextConfig.knowledge_base_ids;
      delete nextConfig.knowledge_bases;

      try {
        await updateSession(http, {
          session_id: sessionId,
          config: JSON.stringify(nextConfig),
        });
      } catch (error) {
        throw new Error(`update session ${sessionId} explore config failed: ${(error as Error).message}`);
      }
    },
    [http],
  );
}

function parseSessionConfigStrict(sessionId: number, config: string | undefined): Record<string, unknown> {
  if (!config) {
    return {};
  }
  const parsed = JSON.parse(config) as unknown;
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error(`session ${sessionId} config is not an object`);
  }
  return parsed as Record<string, unknown>;
}
