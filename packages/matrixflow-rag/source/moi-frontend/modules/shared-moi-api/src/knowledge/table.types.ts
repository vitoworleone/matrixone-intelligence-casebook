export interface TableOverviewItem {
  db_name: string;
  table_name: string;
  col_names: string[];
}

export interface TableOverviewResponse {
  tables: TableOverviewItem[];
}

export interface ListDatabaseTablesRequest {
  database_id: number;
}

export interface ListDatabaseTablesResponse {
  tables: TableOverviewItem[];
}

export interface GetTableColumnsRequest {
  database_id: number;
  table_id?: number;
  table_name?: string;
}

export interface GetTableColumnsResponse {
  db_name: string;
  table_name: string;
  col_names: string[];
}
