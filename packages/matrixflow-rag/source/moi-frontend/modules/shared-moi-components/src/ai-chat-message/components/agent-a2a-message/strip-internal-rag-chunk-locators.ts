/**
 * Hide internal RAG chunk locators from user-facing answer text (#12968).
 *
 * Models sometimes paste raw evidence ids like:
 *   70b9db39-31e5-4d52-a58e-01159ddf7e8b:1:chunk:0:0
 * into the answer body. Those ids belong in source metadata / debug views,
 * not the plain-language answer.
 */

// file_uuid:version:chunk:start:end  (optionally with surrounding brackets/backticks)
const RAW_CHUNK_LOCATOR_RE =
  /(?:\[|\()?[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}:\d+:chunk:\d+:\d+(?:\]|\))?/g;

// Bare "file_uuid:version:chunk:start:end" already covered above.
// Also strip shorter forms like "uuid:chunk:12" if they appear.
const RAW_CHUNK_SHORT_RE =
  /(?:\[|\()?[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}:chunk:\d+(?::\d+)?(?:\]|\))?/g;

export function stripInternalRAGChunkLocators(answer: string): string {
  if (!answer || typeof answer !== 'string') return answer ?? '';
  let next = answer.replace(RAW_CHUNK_LOCATOR_RE, '');
  next = next.replace(RAW_CHUNK_SHORT_RE, '');
  return next;
}
