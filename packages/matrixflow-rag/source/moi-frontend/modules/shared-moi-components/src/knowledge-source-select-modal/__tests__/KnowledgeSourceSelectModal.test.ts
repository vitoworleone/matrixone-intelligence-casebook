import { readFileSync } from 'node:fs';
import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

import {
  createDefaultStructuredTableName,
  hasDroppedDirectory,
  isKnowledgeBaseCatalogFileSelectable,
  KNOWLEDGE_BASE_CATALOG_FILE_EXTENSION_FILTERS,
} from '../KnowledgeSourceSelectModal';

const sourcePath = resolve(dirname(fileURLToPath(import.meta.url)), '../KnowledgeSourceSelectModal.tsx');

describe('KnowledgeSourceSelectModal refresh actions', () => {
  it('uses the standard delete action when removing a selected local file', () => {
    const source = readFileSync(sourcePath, 'utf8');

    expect(source).toContain("import { ListActionButton } from '../list-action';");
    expect(source).toMatch(/<ListActionButton[\s\S]*action="delete"[\s\S]*local-file-remove-btn/);
  });

  it('allows Catalog XLS/XLSX files while keeping ZIP unavailable', () => {
    expect(KNOWLEDGE_BASE_CATALOG_FILE_EXTENSION_FILTERS).toContain('xls');
    expect(KNOWLEDGE_BASE_CATALOG_FILE_EXTENSION_FILTERS).toContain('xlsx');
    expect(KNOWLEDGE_BASE_CATALOG_FILE_EXTENSION_FILTERS).not.toContain('zip');

    expect(isKnowledgeBaseCatalogFileSelectable({ file_ext: 'xls' })).toBe(true);
    expect(isKnowledgeBaseCatalogFileSelectable({ file_ext: '.XLSX' })).toBe(true);
    expect(isKnowledgeBaseCatalogFileSelectable({ name: 'quarterly-report.XLSX' })).toBe(true);
    expect(isKnowledgeBaseCatalogFileSelectable({ file_ext: 'zip' })).toBe(false);
  });

  it('creates the same valid default table names used by structured load', () => {
    expect(createDefaultStructuredTableName('DimProductSubcategory.xlsx')).toBe('dimproductsubcategory');
    expect(createDefaultStructuredTableName('Case 03 Inventory.XLS')).toBe('case_03_inventory');
    expect(createDefaultStructuredTableName('-Report.csv')).toBe('_report');
    expect(createDefaultStructuredTableName('')).toBe('');
  });

  it('detects directories in a Chromium drop payload before Upload processes it', () => {
    const dataTransfer = {
      items: [{ webkitGetAsEntry: () => ({ isDirectory: false }) }, { webkitGetAsEntry: () => ({ isDirectory: true }) }],
    } as unknown as Pick<DataTransfer, 'items'>;

    expect(hasDroppedDirectory(dataTransfer)).toBe(true);
    expect(hasDroppedDirectory({ items: [] } as unknown as Pick<DataTransfer, 'items'>)).toBe(false);
  });

  it('shows only the basic information and upload data steps during knowledge base creation', () => {
    const source = readFileSync(sourcePath, 'utf8');

    expect(source).toContain('create-document-sources-step-basic');
    expect(source).toContain('create-document-sources-step-upload');
    expect(source).not.toContain('create-document-sources-step-process');
  });

  it('keeps structured preview refresh as an icon-only tooltip action', () => {
    const source = readFileSync(sourcePath, 'utf8');

    expect(source).toMatch(
      /<Tooltip title=\{t\('knowledge\.base\.create-document-sources-structured-preview-refresh'\)\}>[\s\S]*data-testid=\{`\$\{testIdPrefix\}-structured-preview-refresh-btn`\}[\s\S]*\/>\s*<\/Tooltip>/,
    );
    expect(source).not.toMatch(
      /data-testid=\{`\$\{testIdPrefix\}-structured-preview-refresh-btn`\}[\s\S]*>\s*\{t\('knowledge\.base\.create-document-sources-structured-preview-refresh'\)\}\s*<\/Button>/,
    );
  });

  it('uploads unstructured local files before submitting file_id while preserving structured preview file_id', () => {
    const source = readFileSync(sourcePath, 'utf8');

    expect(source).toContain('uploadSemanticModelLocalFileApi');
    expect(source).toContain("formData.append('file', originFile, file.name)");
    expect(source).toContain('localFileList.map((file) => uploadUnstructuredFile(file, http, knowledgeBaseId))');
    expect(source).toContain('Promise.allSettled');
    expect(source).toContain('return { file_name: file.name, file_id: fileID }');
    expect(source).toContain('uploadSemanticModelLocalFileApi');
    expect(source).not.toContain('uploadCatalogFileApi');
    expect(source).not.toContain('content_base64');
    expect(source).not.toContain('FileReader');
    expect(source).toContain('loading={busy}');
    expect(source).toContain("message.error(t('knowledge.base.create-document-sources-local-file-upload-failed'))");
    expect(source).toContain("formData.append('files', originFile, file.name)");
    expect(source).toContain('file_id: structuredPreviewSourceFileID');
    expect(source).toContain('buildStructuredTableConfig([structuredPreviewConnFileID])');
  });

  it('keeps unstructured upload busy until every started request settles', () => {
    const source = readFileSync(sourcePath, 'utf8');

    expect(source).toContain('Promise.allSettled');
    expect(source).toContain('setUploadingUnstructuredFiles(true)');
    expect(source).toContain('setUploadingUnstructuredFiles(false)');
    // Reject path must wait for allSettled inside buildSubmitPayload before clearing busy.
    expect(source).toMatch(
      /const settled = await Promise\.allSettled\([\s\S]*localFileList\.map\(\(file\) => uploadUnstructuredFile\(file, http, knowledgeBaseId\)\)/,
    );
    expect(source).not.toContain('Promise.all(localFileList.map((file) => uploadUnstructuredFile');
    // Behavioral coverage lives in moi-knowledge KnowledgeCardList multi-file upload test.
    expect(source).toContain('firstError !== null');
  });

  it('resolves existing catalog sources through knowledge base membership API', () => {
    const source = readFileSync(sourcePath, 'utf8');

    expect(source).toContain('checkSemanticModelSourceExistenceApi');
    expect(source).toContain('knowledgeBaseId?: number | string');
    expect(source).toContain('resolveExistingSources={knowledgeBaseId ? resolveExistingCatalogSources : undefined}');
    expect(source).not.toContain('existingCatalogFileIds');
    expect(source).not.toContain('existingCatalogTableIds');
  });

  it('blocks catalog submission until the authoritative count preview succeeds', () => {
    const source = readFileSync(sourcePath, 'utf8');

    expect(source).toContain("catalogPreviewStatus !== 'ready'");
    expect(source).toContain('onPreviewStatusChange={setCatalogPreviewStatus}');
    expect(source).toContain("setCatalogPreviewStatus(selections.length > 0 ? 'loading' : 'ready')");
  });

  it('does not pin knowledge base data domain to the source catalog', () => {
    const source = readFileSync(sourcePath, 'utf8');

    expect(source).not.toContain('onCatalogSelect={setTargetCatalogId}');
    expect(source).not.toContain('target_catalog_id: targetCatalogId');
    expect(source).toContain('source_selections: catalogSelections');
  });
});
