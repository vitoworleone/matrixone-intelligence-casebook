import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import type { ApiResponse } from '../types';
import type {
  GetTableColumnsRequest,
  GetTableColumnsResponse,
  ListDatabaseTablesRequest,
  ListDatabaseTablesResponse,
  TableOverviewResponse,
} from './table.types';

/** POST /table/columns — fetch one table columns by database/table */
export async function getTableColumnsApi(
  req: GetTableColumnsRequest,
  http: AppHttpClient,
): Promise<ApiResponse<GetTableColumnsResponse>> {
  const res = await http.post<ApiResponse<GetTableColumnsResponse>>('/table/columns', req);
  return res.data;
}

/** POST /table/database/tables — fetch one database table overview list */
export async function listDatabaseTablesApi(
  req: ListDatabaseTablesRequest,
  http: AppHttpClient,
): Promise<ApiResponse<ListDatabaseTablesResponse>> {
  const res = await http.post<ApiResponse<ListDatabaseTablesResponse>>('/table/database/tables', req);
  return res.data;
}

/** POST /table/overview — fetch table overview */
export async function getTableOverviewApi(http: AppHttpClient): Promise<ApiResponse<TableOverviewResponse>> {
  const res = await http.post<ApiResponse<TableOverviewResponse>>('/table/overview', {});
  return res.data;
}
