import { previewFileStreamApi } from '@moi/shared-moi-api/catalog';
import { previewSemanticModelArtifactApi, previewSemanticModelSourceFileApi } from '@moi/shared-moi-api/knowledge';
import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import type { ExploreA2AAnswerSourceRef } from './services/exploreA2AStreamParser';

export type AnswerSourceDisplayFallbacks = {
  rag: string;
  sql: string;
  image: string;
};

export type AnswerSourcePreview = {
  kind: 'source' | 'visual' | 'page_visual';
  fileId: string;
  fileName: string;
  volumeId: string;
  semanticModelId: number | null;
  sourceRowId: string;
  page: number | null;
  bbox: number[];
};

export function answerSourcePages(source: ExploreA2AAnswerSourceRef): number[] {
  return source.pages.length ? source.pages : source.page !== null ? [source.page] : [];
}

export function firstAnswerSourcePage(source: ExploreA2AAnswerSourceRef): number | null {
  return answerSourcePages(source).find((page) => Number.isFinite(page) && page > 0) ?? null;
}

export function answerSourceIdentity(semanticModelId: number | null, sourceRowId: string, fileId: string): string {
  if (isPositiveInteger(semanticModelId)) {
    return sourceRowId
      ? `source\x00${semanticModelId}\x00${sourceRowId}`
      : fileId
        ? `source_file\x00${semanticModelId}\x00${fileId}`
        : '';
  }
  return fileId ? `file\x00${fileId}` : '';
}

export function answerSourceDisplayLabel(source: ExploreA2AAnswerSourceRef, fallback: AnswerSourceDisplayFallbacks): string {
  if (source.type === 'sql_table') {
    if (source.database && source.table) {
      return `${source.database}.${source.table}`;
    }
    return source.table || source.database || source.label || fallback.sql;
  }
  if (source.type === 'visual_hit') {
    return source.fileName || source.label || source.objectId || source.imageFileId || source.pageImageFileId || fallback.image;
  }
  return source.fileName || source.label || source.fileId || fallback.rag;
}

export function answerSourcePreviews(source: ExploreA2AAnswerSourceRef): AnswerSourcePreview[] {
  const out: AnswerSourcePreview[] = [];
  const sourcePage = firstAnswerSourcePage(source);
  const pushPreview = (preview: AnswerSourcePreview) => {
    if (!preview.fileId) {
      return;
    }
    const key = answerSourcePreviewIdentity(preview);
    if (out.some((item) => answerSourcePreviewIdentity(item) === key)) {
      return;
    }
    out.push(preview);
  };
  const visualRefs = source.visualRefs.length
    ? source.visualRefs
    : source.imageFileId || source.pageImageFileId
      ? [
          {
            semanticModelId: source.semanticModelId,
            chunkId: source.chunkId,
            objectId: source.objectId,
            objectKind: source.objectKind,
            imageFileId: source.imageFileId,
            pageImageFileId: source.pageImageFileId,
            page: sourcePage,
            bbox: source.bbox,
          },
        ]
      : [];
  for (const ref of visualRefs) {
    if (ref.imageFileId) {
      pushPreview({
        kind: 'visual',
        fileId: ref.imageFileId,
        fileName: `${ref.objectId || ref.imageFileId}.png`,
        volumeId: '',
        semanticModelId: ref.semanticModelId,
        sourceRowId: '',
        page: ref.page,
        bbox: ref.bbox,
      });
      continue;
    }
    if (ref.pageImageFileId) {
      pushPreview({
        kind: 'page_visual',
        fileId: ref.pageImageFileId,
        fileName: `${ref.objectId || ref.pageImageFileId}.png`,
        volumeId: '',
        semanticModelId: ref.semanticModelId,
        sourceRowId: '',
        page: ref.page,
        bbox: ref.bbox,
      });
    }
  }
  if (source.fileId) {
    pushPreview({
      kind: 'source',
      fileId: source.fileId,
      fileName: source.fileName || source.fileId,
      volumeId: source.volumeId,
      semanticModelId: source.semanticModelId,
      sourceRowId: source.sourceRowId,
      page: sourcePage,
      bbox: source.bbox,
    });
  }
  return out;
}

export function answerInlineVisualPreviews(sources: ExploreA2AAnswerSourceRef[]): AnswerSourcePreview[] {
  const out: AnswerSourcePreview[] = [];
  const seen = new Set<string>();
  for (const source of sources) {
    for (const preview of answerSourcePreviews(source)) {
      if (preview.kind !== 'visual') {
        continue;
      }
      const key = [preview.semanticModelId ?? '', preview.fileId, preview.page ?? '', preview.bbox.join(',')].join('\x00');
      if (seen.has(key)) {
        continue;
      }
      seen.add(key);
      out.push(preview);
    }
  }
  return out;
}

export async function requestAnswerSourcePreviewBlob(http: AppHttpClient, preview: AnswerSourcePreview): Promise<Blob> {
  const options = {
    responseType: 'blob' as const,
    responseContentType: 'blob' as const,
  };
  if (preview.kind === 'source') {
    if (isPositiveInteger(preview.semanticModelId) && preview.fileId) {
      const result = await previewSemanticModelSourceFileApi(preview.semanticModelId, preview.fileId, http, options);
      return previewResponseBlob(result.data);
    }
    const result = await previewFileStreamApi(
      { file_id: preview.fileId, volume_id: preview.volumeId || undefined },
      http,
      options,
    );
    const resp = result as { data: Blob };
    return previewResponseBlob(resp.data);
  }
  if (!isPositiveInteger(preview.semanticModelId)) {
    throw new Error('Semantic model owner is required for artifact preview');
  }
  const result = await previewSemanticModelArtifactApi(preview.semanticModelId, preview.fileId, http, options);
  return previewResponseBlob(result.data);
}

function answerSourcePreviewIdentity(preview: AnswerSourcePreview): string {
  return preview.kind === 'source'
    ? answerSourceIdentity(preview.semanticModelId, preview.sourceRowId, preview.fileId)
    : `${preview.semanticModelId ?? ''}\x00${preview.fileId}`;
}

function isPositiveInteger(value: number | null): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value > 0;
}

function previewResponseBlob(value: Blob): Blob {
  return value instanceof Blob ? value : new Blob([value as unknown as BlobPart]);
}
