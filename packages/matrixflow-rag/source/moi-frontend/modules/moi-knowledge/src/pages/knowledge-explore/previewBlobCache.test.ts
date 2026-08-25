import { afterEach, describe, expect, it, vi } from 'vitest';

import {
  clearPreviewBlobCacheForTest,
  getCachedPreviewBlob,
  PREVIEW_BLOB_ERROR_TTL_MS,
  type PreviewBlobCacheKey,
} from './previewBlobCache';

const cacheKey: PreviewBlobCacheKey = { scope: 'workspace-1', fileId: 'file-1' };

afterEach(() => {
  vi.useRealTimers();
  clearPreviewBlobCacheForTest();
});

describe('preview blob cache', () => {
  it('dedupes concurrent loads for the same file', async () => {
    const blob = new Blob(['image']);
    const load = vi.fn<() => Promise<Blob>>().mockResolvedValue(blob);

    const first = getCachedPreviewBlob(cacheKey, load);
    const second = getCachedPreviewBlob(cacheKey, load);

    expect(first).toBe(second);
    await expect(first).resolves.toBe(blob);
    expect(load).toHaveBeenCalledTimes(1);
  });

  it('reuses successful blobs for later loads', async () => {
    const blob = new Blob(['image']);
    const load = vi.fn<() => Promise<Blob>>().mockResolvedValue(blob);
    const laterLoad = vi.fn<() => Promise<Blob>>().mockResolvedValue(new Blob(['other']));

    await expect(getCachedPreviewBlob(cacheKey, load)).resolves.toBe(blob);
    await expect(getCachedPreviewBlob(cacheKey, laterLoad)).resolves.toBe(blob);

    expect(load).toHaveBeenCalledTimes(1);
    expect(laterLoad).not.toHaveBeenCalled();
  });

  it('caches failed loads briefly and retries after expiry', async () => {
    vi.useFakeTimers();
    vi.setSystemTime(0);
    const error = new Error('missing');
    const load = vi.fn<() => Promise<Blob>>().mockRejectedValue(error);
    const retryLoad = vi.fn<() => Promise<Blob>>().mockResolvedValue(new Blob(['recovered']));

    await expect(getCachedPreviewBlob(cacheKey, load)).rejects.toThrow('missing');
    await expect(getCachedPreviewBlob(cacheKey, retryLoad)).rejects.toThrow('missing');
    expect(retryLoad).not.toHaveBeenCalled();

    vi.setSystemTime(PREVIEW_BLOB_ERROR_TTL_MS + 1);
    await expect(getCachedPreviewBlob(cacheKey, retryLoad)).resolves.toBeInstanceOf(Blob);
    expect(retryLoad).toHaveBeenCalledTimes(1);
  });

  it('keeps different scopes isolated', async () => {
    const firstBlob = new Blob(['first']);
    const secondBlob = new Blob(['second']);
    const firstLoad = vi.fn<() => Promise<Blob>>().mockResolvedValue(firstBlob);
    const secondLoad = vi.fn<() => Promise<Blob>>().mockResolvedValue(secondBlob);

    await expect(getCachedPreviewBlob(cacheKey, firstLoad)).resolves.toBe(firstBlob);
    await expect(getCachedPreviewBlob({ ...cacheKey, scope: 'workspace-2' }, secondLoad)).resolves.toBe(secondBlob);

    expect(firstLoad).toHaveBeenCalledTimes(1);
    expect(secondLoad).toHaveBeenCalledTimes(1);
  });

  it('keeps the same artifact id isolated between semantic models', async () => {
    const firstBlob = new Blob(['model-101']);
    const secondBlob = new Blob(['model-202']);
    const firstLoad = vi.fn<() => Promise<Blob>>().mockResolvedValue(firstBlob);
    const secondLoad = vi.fn<() => Promise<Blob>>().mockResolvedValue(secondBlob);
    const firstModelKey = { ...cacheKey, semanticModelId: 101 } as PreviewBlobCacheKey & { semanticModelId: number };
    const secondModelKey = { ...cacheKey, semanticModelId: 202 } as PreviewBlobCacheKey & { semanticModelId: number };

    await expect(getCachedPreviewBlob(firstModelKey, firstLoad)).resolves.toBe(firstBlob);
    await expect(getCachedPreviewBlob(secondModelKey, secondLoad)).resolves.toBe(secondBlob);

    expect(firstLoad).toHaveBeenCalledTimes(1);
    expect(secondLoad).toHaveBeenCalledTimes(1);
  });

  it('keeps source previews isolated between source rows in the same semantic model', async () => {
    const firstBlob = new Blob(['source-1']);
    const secondBlob = new Blob(['source-2']);
    const firstLoad = vi.fn<() => Promise<Blob>>().mockResolvedValue(firstBlob);
    const secondLoad = vi.fn<() => Promise<Blob>>().mockResolvedValue(secondBlob);
    const firstSourceKey = { ...cacheKey, semanticModelId: 101, sourceRowId: 'source-1' };
    const secondSourceKey = { ...cacheKey, semanticModelId: 101, sourceRowId: 'source-2' };

    await expect(getCachedPreviewBlob(firstSourceKey, firstLoad)).resolves.toBe(firstBlob);
    await expect(getCachedPreviewBlob(secondSourceKey, secondLoad)).resolves.toBe(secondBlob);

    expect(firstLoad).toHaveBeenCalledTimes(1);
    expect(secondLoad).toHaveBeenCalledTimes(1);
  });

  it('does not share an in-flight promise between semantic models', () => {
    let resolveFirst: ((blob: Blob) => void) | undefined;
    let resolveSecond: ((blob: Blob) => void) | undefined;
    const firstLoad = vi.fn(
      () =>
        new Promise<Blob>((resolve) => {
          resolveFirst = resolve;
        }),
    );
    const secondLoad = vi.fn(
      () =>
        new Promise<Blob>((resolve) => {
          resolveSecond = resolve;
        }),
    );
    const firstModelKey = { ...cacheKey, semanticModelId: 101 } as PreviewBlobCacheKey & { semanticModelId: number };
    const secondModelKey = { ...cacheKey, semanticModelId: 202 } as PreviewBlobCacheKey & { semanticModelId: number };

    const first = getCachedPreviewBlob(firstModelKey, firstLoad);
    const second = getCachedPreviewBlob(secondModelKey, secondLoad);

    expect(first).not.toBe(second);
    expect(firstLoad).toHaveBeenCalledTimes(1);
    expect(secondLoad).toHaveBeenCalledTimes(1);
    resolveFirst?.(new Blob(['model-101']));
    resolveSecond?.(new Blob(['model-202']));
  });

  it('does not leak a rejected cache entry to another semantic model', async () => {
    const firstError = new Error('model 101 unavailable');
    const secondBlob = new Blob(['model-202']);
    const firstLoad = vi.fn<() => Promise<Blob>>().mockRejectedValue(firstError);
    const secondLoad = vi.fn<() => Promise<Blob>>().mockResolvedValue(secondBlob);
    const firstModelKey = { ...cacheKey, semanticModelId: 101 } as PreviewBlobCacheKey & { semanticModelId: number };
    const secondModelKey = { ...cacheKey, semanticModelId: 202 } as PreviewBlobCacheKey & { semanticModelId: number };

    await expect(getCachedPreviewBlob(firstModelKey, firstLoad)).rejects.toBe(firstError);
    await expect(getCachedPreviewBlob(secondModelKey, secondLoad)).resolves.toBe(secondBlob);

    expect(firstLoad).toHaveBeenCalledTimes(1);
    expect(secondLoad).toHaveBeenCalledTimes(1);
  });
});
