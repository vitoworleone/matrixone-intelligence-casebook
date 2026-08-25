import { describe, expect, it } from 'vitest';

import type { SemanticModelSource } from '@moi/shared-moi-api/knowledge';
import { resolveDocumentPreviewFileId } from '../KnowledgeAdvancedConfigPage';

function source(overrides: Partial<SemanticModelSource>): SemanticModelSource {
  return overrides as SemanticModelSource;
}

describe('KnowledgeAdvancedConfigPage document preview file ID', () => {
  it('does not substitute a KB file ID when the source file ID is empty', () => {
    expect(
      resolveDocumentPreviewFileId(
        source({
          source_file_id: '',
          source_resource_id: 'source-resource-id',
          kb_file_id: 'kb-file-id',
          kb_resource_id: 'kb-resource-id',
        }),
      ),
    ).toBe('');
  });

  it('uses the source file ID when it is present', () => {
    expect(
      resolveDocumentPreviewFileId(
        source({
          source_file_id: 'source-file-id',
          kb_file_id: 'kb-file-id',
        }),
      ),
    ).toBe('source-file-id');
  });
});
