import type { SemanticModelFiles, SemanticModelTable } from '@moi/shared-moi-api/knowledge';
import type { SelectedFileItem } from '../../components/TreeOfFiles/TreeOfFiles';
import type { SemanticModel } from '../../service/semanticModel';

export function mapKnowledgeDetailToSelectedFiles(detail: SemanticModel): {
  selectedFiles: SelectedFileItem[];
  expandedKeys: string[];
} {
  const selectedFiles: SelectedFileItem[] = [];
  const expandedKeys: string[] = [];
  const selectedVolumeKeys = new Set<string>();

  if (detail.files?.volumes && Array.isArray(detail.files.volumes)) {
    detail.files.volumes.forEach((volume) => {
      if (!volume?.volume_id) return;
      selectedVolumeKeys.add(`volume-${String(volume.volume_id)}`);
    });
  }

  if (detail.files?.volume_ids && Array.isArray(detail.files.volume_ids)) {
    detail.files.volume_ids.forEach((volumeId) => {
      selectedVolumeKeys.add(`volume-${String(volumeId)}`);
    });
  }

  if (detail.files?.file_ids && Array.isArray(detail.files.file_ids)) {
    const fileParents = Array.isArray(detail.files.parents) ? detail.files.parents.map(String) : [];
    selectedFiles.push(
      ...detail.files.file_ids
        .filter(() => !fileParents.some((parentId) => selectedVolumeKeys.has(parentId)))
        .map((fileId) => ({
          file_id: fileId,
          full_path: [],
          type: 'file',
          full_path_id: [...(detail.files?.parents || []), `file-${fileId}`],
        })),
    );

    if (detail.files.parents && Array.isArray(detail.files.parents)) {
      expandedKeys.push(...detail.files.parents);
    }
  }

  if (detail.files?.volumes && Array.isArray(detail.files.volumes)) {
    detail.files.volumes.forEach((volume) => {
      if (!volume?.volume_id) return;
      const id = String(volume.volume_id);
      const parentIds = Array.isArray(volume.parents) ? volume.parents : [];
      selectedFiles.push({
        file_id: id,
        full_path: Array.isArray(volume.path) ? volume.path : [],
        type: 'volume',
        full_path_id: [...parentIds, `volume-${id}`],
      });
      expandedKeys.push(...parentIds);
    });
  } else if (detail.files?.volume_ids && Array.isArray(detail.files.volume_ids)) {
    detail.files.volume_ids.forEach((volumeId) => {
      const id = String(volumeId);
      selectedFiles.push({
        file_id: id,
        full_path: [],
        type: 'volume',
        full_path_id: [`volume-${id}`],
      });
    });
  }

  if (detail.tables && Array.isArray(detail.tables)) {
    detail.tables.forEach((table) => {
      if (table.table_names && Array.isArray(table.table_names)) {
        table.table_names.forEach((tableName) => {
          const stableKey = `${table.db_name}::${tableName}`;
          selectedFiles.push({
            file_id: stableKey,
            full_path: [table.db_name, tableName].filter(Boolean),
            type: 'table',
            full_path_id: [...(table.parents || []), `table-${stableKey}`],
          });
        });
      }

      if (table.parents && Array.isArray(table.parents)) {
        expandedKeys.push(...table.parents);
      }
    });
  }

  return {
    selectedFiles,
    expandedKeys: [...new Set(expandedKeys)],
  };
}

export function mapSelectedFilesToSourcePayload(selectedFiles: SelectedFileItem[] = []): {
  files: SemanticModelFiles;
  tables: SemanticModelTable[];
} {
  const selectedVolumeKeys = new Set(
    selectedFiles
      .filter((item) => item.type === 'volume')
      .map((item) => {
        const volumeID = String(item.file_id);
        return item.full_path_id?.length ? String(item.full_path_id[item.full_path_id.length - 1]) : `volume-${volumeID}`;
      }),
  );
  const selectedFilesIds: string[] = [];
  const selectedFilesParents: string[] = [];
  const selectedVolumeIDs: string[] = [];
  const selectedVolumes: NonNullable<SemanticModelFiles['volumes']> = [];
  const tablesMap = new Map<string, { parents: string[]; table_names: Set<string> }>();

  selectedFiles.forEach((item) => {
    if (item.type === 'volume') {
      const volumeID = String(item.file_id);
      if (!selectedVolumeIDs.includes(volumeID)) {
        selectedVolumeIDs.push(volumeID);
      }
      selectedVolumes.push({
        volume_id: volumeID,
        parents:
          item.full_path_id && Array.isArray(item.full_path_id)
            ? item.full_path_id.slice(0, -1).map((parentId) => String(parentId))
            : [],
        path: item.full_path || [],
      });
      return;
    }

    if (item.type === 'file') {
      const parentIds = item.full_path_id && Array.isArray(item.full_path_id) ? item.full_path_id.slice(0, -1).map(String) : [];
      if (parentIds.some((parentId) => selectedVolumeKeys.has(parentId))) {
        return;
      }

      selectedFilesIds.push(String(item.file_id));

      parentIds.forEach((parentId) => {
        if (!selectedFilesParents.includes(parentId)) {
          selectedFilesParents.push(parentId);
        }
      });

      return;
    }

    if (item.type !== 'table') return;

    const dbName = item.full_path?.[item.full_path.length - 2] || '';
    const tableName = item.full_path?.[item.full_path.length - 1] || '';
    const parentIds =
      item.full_path_id && Array.isArray(item.full_path_id)
        ? item.full_path_id.slice(0, -1).map((parentId) => String(parentId))
        : [];

    if (tablesMap.has(dbName)) {
      const existing = tablesMap.get(dbName);
      if (!existing) {
        console.warn('[mapSelectedFilesToSourcePayload] table map entry missing unexpectedly', { dbName });
        return;
      }

      existing.table_names.add(tableName);

      parentIds.forEach((parentId) => {
        if (!existing.parents.includes(parentId)) {
          existing.parents.push(parentId);
        }
      });
      return;
    }

    tablesMap.set(dbName, {
      parents: [...parentIds],
      table_names: new Set([tableName]),
    });
  });

  const files: SemanticModelFiles = {
    file_ids: selectedFilesIds,
    parents: selectedFilesParents,
    volume_ids: selectedVolumeIDs,
    volumes: selectedVolumes,
  };

  const tables: SemanticModelTable[] = Array.from(tablesMap.entries()).map(([db_name, data]) => ({
    db_name,
    table_names: Array.from(data.table_names),
    parents: data.parents,
  }));

  return { files, tables };
}
