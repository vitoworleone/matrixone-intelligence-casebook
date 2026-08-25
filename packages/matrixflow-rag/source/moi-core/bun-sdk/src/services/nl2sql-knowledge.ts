/**
 * Nl2SqlKnowledgeService — manages NL2SQL knowledge items within a Workspace.
 *
 * URL pattern: `/api/v1/workspaces/{workspaceId}/nl2sql-knowledge`
 *
 * Workspace-scoped service: constructor takes (http, workspaceId).
 *
 * @module services/nl2sql-knowledge
 */
import type { HttpClient } from "../internal/http";
import type { Nl2SqlKnowledge, ListOptions } from "../types";

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

/** Options for creating a NL2SQL knowledge item. */
export interface CreateNl2SqlKnowledgeOptions {
  name?: string;
  knowledgeValue?: string[];
  associateTables?: string[];
  explanationType?: string;
}

// ============================================================================
// Nl2SqlKnowledgeService
// ============================================================================

export class Nl2SqlKnowledgeService {
  constructor(
    readonly http: HttpClient,
    readonly workspaceId: string,
  ) {}

  private get basePath(): string {
    return `/api/v1/workspaces/${this.workspaceId}/nl2sql-knowledge`;
  }

  /** Create a NL2SQL knowledge item. */
  async create(
    knowledgeBaseId: number,
    knowledgeType: string,
    knowledgeKey: string,
    options?: CreateNl2SqlKnowledgeOptions,
  ): Promise<Nl2SqlKnowledge> {
    const body: Record<string, unknown> = {
      knowledge_base_id: knowledgeBaseId,
      knowledge_type: knowledgeType,
      knowledge_key: knowledgeKey,
    };
    if (options?.name !== undefined) body.name = options.name;
    if (options?.knowledgeValue !== undefined)
      body.knowledge_value = options.knowledgeValue;
    if (options?.associateTables !== undefined)
      body.associate_tables = options.associateTables;
    if (options?.explanationType !== undefined)
      body.explanation_type = options.explanationType;
    return this.http.post<Nl2SqlKnowledge>(this.basePath, body);
  }

  /** List NL2SQL knowledge items (uses POST /list endpoint). */
  async list(
    knowledgeType: string,
    knowledgeBaseIds: number[],
    options?: ListOptions,
  ): Promise<{ items: Nl2SqlKnowledge[]; next_page_token: string }> {
    const body: Record<string, unknown> = {
      knowledge_type: knowledgeType,
      knowledge_base_ids: knowledgeBaseIds,
    };
    if (options?.pageSize !== undefined && options.pageSize > 0)
      body.page_size = options.pageSize;
    if (options?.pageToken) body.page_token = options.pageToken;
    return this.http.post(`${this.basePath}/list`, body);
  }

  /** Delete a NL2SQL knowledge item by ID. */
  async delete(knowledgeId: number): Promise<void> {
    await this.http.delete(`${this.basePath}/${knowledgeId}`);
  }
}
