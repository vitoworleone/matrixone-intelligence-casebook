const KNOWLEDGE_BASE_CATALOG_FILE_EXTENSIONS = new Set([
  'pdf',
  'doc',
  'docx',
  'ppt',
  'pptx',
  'txt',
  'md',
  'htm',
  'html',
  'eml',
  'msg',
]);

export interface KnowledgeBaseCatalogFileCandidate {
  file_ext?: string;
  name?: string;
}

function getKnowledgeBaseCatalogFileExt(item: KnowledgeBaseCatalogFileCandidate): string {
  let ext = item.file_ext || '';
  if (!ext && item.name) {
    const extensionStart = item.name.lastIndexOf('.');
    if (extensionStart >= 0 && extensionStart < item.name.length - 1) {
      ext = item.name.slice(extensionStart + 1);
    }
  }
  return ext.startsWith('.') ? ext.slice(1).toLowerCase() : ext.toLowerCase();
}

export function isKnowledgeBaseCatalogFileSelectable(item: KnowledgeBaseCatalogFileCandidate): boolean {
  return KNOWLEDGE_BASE_CATALOG_FILE_EXTENSIONS.has(getKnowledgeBaseCatalogFileExt(item));
}
