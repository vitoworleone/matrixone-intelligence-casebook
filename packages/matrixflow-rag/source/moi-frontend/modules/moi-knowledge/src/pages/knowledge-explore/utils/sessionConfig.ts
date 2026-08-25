/**
 * Session config normalization utilities.
 * Extracted from index.tsx to keep the page layer thin.
 */

export interface SemanticModelRef {
  semantic_model_id: number;
}

/**
 * Parse unknown input into a normalized array of SemanticModelRef.
 * Deduplicates by semantic_model_id and sorts ascending.
 */
export function normalizeSemanticModelRefs(input: unknown): SemanticModelRef[] {
  if (!Array.isArray(input)) {
    return [];
  }

  const seen = new Set<number>();
  const result: SemanticModelRef[] = [];

  for (const item of input) {
    if (item && typeof item === 'object' && 'semantic_model_id' in item) {
      const id = (item as { semantic_model_id: unknown }).semantic_model_id;
      if (Number.isInteger(id) && (id as number) > 0 && !seen.has(id as number)) {
        seen.add(id as number);
        result.push({ semantic_model_id: id as number });
      }
    }
  }

  return result.sort((a, b) => a.semantic_model_id - b.semantic_model_id);
}

/** Extract sorted ID array from refs for comparison. */
export function extractModelIds(refs: SemanticModelRef[]): number[] {
  return refs.map((r) => r.semantic_model_id);
}

export function isSameSemanticModelRefs(left: SemanticModelRef[], right: SemanticModelRef[]): boolean {
  if (left.length !== right.length) {
    return false;
  }
  return left.every((value, index) => value.semantic_model_id === right[index].semantic_model_id);
}

export function normalizeSessionModel(config: Record<string, unknown>): string {
  const llm = config.llm;
  if (llm && typeof llm === 'object') {
    const modelInLlm = (llm as Record<string, unknown>).model;
    if (typeof modelInLlm === 'string') {
      const normalized = modelInLlm.trim();
      if (normalized) {
        return normalized;
      }
    }
  }

  if (typeof config.model === 'string') {
    return config.model.trim();
  }

  return '';
}

export function normalizeSessionModelBackendId(config: Record<string, unknown>): number | null {
  const llm = config.llm;
  if (llm && typeof llm === 'object') {
    const backendId = normalizeBackendId((llm as Record<string, unknown>).backend_id);
    if (backendId !== null) {
      return backendId;
    }
  }

  return normalizeBackendId(config.llm_backend_id ?? config.backend_id);
}

export function normalizeSessionReasoningEffort(config: Record<string, unknown>): string {
  const llm = config.llm;
  if (llm && typeof llm === 'object') {
    const value = (llm as Record<string, unknown>).reasoning_effort;
    if (typeof value === 'string') {
      return value;
    }
  }

  if (typeof config.llm_reasoning_effort === 'string') {
    return config.llm_reasoning_effort;
  }
  if (typeof config.reasoning_effort === 'string') {
    return config.reasoning_effort;
  }

  return '';
}

function normalizeBackendId(value: unknown): number | null {
  if (Number.isInteger(value) && value !== 0) {
    return value as number;
  }
  if (typeof value === 'string') {
    const parsed = Number(value.trim());
    if (Number.isInteger(parsed) && parsed !== 0) {
      return parsed;
    }
  }
  return null;
}
