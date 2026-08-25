export type PreviewBlobCacheKey = {
  scope: string;
  fileId: string;
  volumeId?: string;
  semanticModelId?: number | null;
  sourceRowId?: string;
};

type PendingPreviewBlobEntry = {
  status: 'pending';
  promise: Promise<Blob>;
  touchedAt: number;
};

type FulfilledPreviewBlobEntry = {
  status: 'fulfilled';
  blob: Blob;
  touchedAt: number;
};

type RejectedPreviewBlobEntry = {
  status: 'rejected';
  error: unknown;
  touchedAt: number;
  expiresAt: number;
};

type PreviewBlobCacheEntry = PendingPreviewBlobEntry | FulfilledPreviewBlobEntry | RejectedPreviewBlobEntry;

export const PREVIEW_BLOB_ERROR_TTL_MS = 30_000;
const PREVIEW_BLOB_CACHE_LIMIT = 64;
const previewBlobCache = new Map<string, PreviewBlobCacheEntry>();

function previewBlobCacheKey(key: PreviewBlobCacheKey): string {
  return `${key.scope}\x00${key.semanticModelId ?? ''}\x00${key.sourceRowId ?? ''}\x00${key.fileId}\x00${key.volumeId ?? ''}`;
}

function enforcePreviewBlobCacheLimit() {
  if (previewBlobCache.size <= PREVIEW_BLOB_CACHE_LIMIT) {
    return;
  }
  const removable = Array.from(previewBlobCache.entries())
    .filter(([, entry]) => entry.status !== 'pending')
    .sort((left, right) => left[1].touchedAt - right[1].touchedAt);

  for (const [key] of removable) {
    if (previewBlobCache.size <= PREVIEW_BLOB_CACHE_LIMIT) {
      return;
    }
    previewBlobCache.delete(key);
  }
}

export function getCachedPreviewBlob(key: PreviewBlobCacheKey, load: () => Promise<Blob>): Promise<Blob> {
  const cacheKey = previewBlobCacheKey(key);
  const now = Date.now();
  const cached = previewBlobCache.get(cacheKey);
  if (cached?.status === 'pending') {
    cached.touchedAt = now;
    return cached.promise;
  }
  if (cached?.status === 'fulfilled') {
    cached.touchedAt = now;
    return Promise.resolve(cached.blob);
  }
  if (cached?.status === 'rejected') {
    if (cached.expiresAt > now) {
      cached.touchedAt = now;
      return Promise.reject(cached.error);
    }
    previewBlobCache.delete(cacheKey);
  }

  const promise = load().then(
    (blob) => {
      previewBlobCache.set(cacheKey, { status: 'fulfilled', blob, touchedAt: Date.now() });
      enforcePreviewBlobCacheLimit();
      return blob;
    },
    (error: unknown) => {
      const failedAt = Date.now();
      previewBlobCache.set(cacheKey, {
        status: 'rejected',
        error,
        touchedAt: failedAt,
        expiresAt: failedAt + PREVIEW_BLOB_ERROR_TTL_MS,
      });
      enforcePreviewBlobCacheLimit();
      throw error;
    },
  );
  previewBlobCache.set(cacheKey, { status: 'pending', promise, touchedAt: now });
  enforcePreviewBlobCacheLimit();
  return promise;
}

export function clearPreviewBlobCacheForTest() {
  previewBlobCache.clear();
}
