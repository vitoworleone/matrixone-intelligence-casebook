/**
 * KnowledgeBaseService — manages Knowledge Base resources within a Workspace.
 *
 * URL pattern: `/api/v1/workspaces/{workspaceId}/knowledge-bases`
 *
 * Workspace-scoped service: constructor takes (http, workspaceId).
 *
 * @module services/knowledge-base
 */
import type { HttpClient } from "../internal/http";
import { buildPaginationQuery } from "../internal/query";
import type { KnowledgeBase, ListOptions } from "../types";

// ---------------------------------------------------------------------------
// Options
// ---------------------------------------------------------------------------

/** Options for creating a knowledge base. */
export interface CreateKnowledgeBaseOptions {
  usageNotes?: string;
  tables?: unknown[];
  files?: Record<string, unknown>;
}

/** Options for updating a knowledge base. */
export interface UpdateKnowledgeBaseOptions {
  name?: string;
  usageNotes?: string;
  tables?: unknown[];
  files?: Record<string, unknown>;
}

// ============================================================================
// KnowledgeBaseService
// ============================================================================

export class KnowledgeBaseService {
  constructor(
    readonly http: HttpClient,
    readonly workspaceId: string,
  ) {}

  private get basePath(): string {
    return `/api/v1/workspaces/${this.workspaceId}/knowledge-bases`;
  }

  /** Create a new knowledge base. */
  async create(
    name: string,
    options?: CreateKnowledgeBaseOptions,
  ): Promise<KnowledgeBase> {
    const body: Record<string, unknown> = { name };
    if (options?.usageNotes !== undefined)
      body.usage_notes = options.usageNotes;
    if (options?.tables !== undefined) body.tables = options.tables;
    if (options?.files !== undefined) body.files = options.files;
    return this.http.post<KnowledgeBase>(this.basePath, body);
  }

  /** Get a knowledge base by ID. */
  async get(knowledgeBaseId: number): Promise<KnowledgeBase> {
    return this.http.get<KnowledgeBase>(
      `${this.basePath}/${knowledgeBaseId}`,
    );
  }

  /** List knowledge bases with optional pagination. */
  async list(
    options?: ListOptions,
  ): Promise<{ knowledge_bases: KnowledgeBase[]; next_page_token: string }> {
    let path = this.basePath;
    const qs = buildPaginationQuery(options);
    if (qs) path += `?${qs}`;
    return this.http.get(path);
  }

  /** Update an existing knowledge base. */
  async update(
    knowledgeBaseId: number,
    options?: UpdateKnowledgeBaseOptions,
  ): Promise<KnowledgeBase> {
    const body: Record<string, unknown> = {};
    if (options?.name !== undefined) body.name = options.name;
    if (options?.usageNotes !== undefined)
      body.usage_notes = options.usageNotes;
    if (options?.tables !== undefined) body.tables = options.tables;
    if (options?.files !== undefined) body.files = options.files;
    return this.http.put<KnowledgeBase>(
      `${this.basePath}/${knowledgeBaseId}`,
      body,
    );
  }

  /** Delete a knowledge base by ID. */
  async delete(knowledgeBaseId: number): Promise<void> {
    await this.http.delete(`${this.basePath}/${knowledgeBaseId}`);
  }
}
