import { extractApiErrorCode, extractApiErrorMessage as extractSharedApiErrorMessage } from '@moi/shared-moi-api/response';
import type { ApiResponse } from '@moi/shared-moi-api/types';
import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';

function toError(msg: string, code?: string): Error & { msg: string; code?: string } {
  const error = new Error(msg) as Error & { msg: string; code?: string };
  error.msg = msg;
  if (code) {
    error.code = code;
  }
  return error;
}

function isSuccessCode(code: string | number): boolean {
  return code === 'OK' || code === 0 || code === '0';
}

export interface ApiErrorDetail {
  code?: string;
  msg?: string;
  status?: number;
  requestId?: string;
}

function getResponseHeader(headers: unknown, name: string): string | undefined {
  if (!headers || typeof headers !== 'object') {
    return undefined;
  }

  if (headers instanceof Map) {
    for (const [key, value] of headers) {
      if (typeof key === 'string' && key.toLowerCase() === name.toLowerCase() && typeof value === 'string') {
        return value;
      }
    }
    return undefined;
  }

  const get = (headers as { get?: unknown }).get;
  if (typeof get === 'function') {
    const value = get.call(headers, name);
    return typeof value === 'string' ? value : undefined;
  }

  for (const [key, value] of Object.entries(headers)) {
    if (key.toLowerCase() === name.toLowerCase() && typeof value === 'string') {
      return value;
    }
  }

  return undefined;
}

export function extractApiErrorDetail(error: unknown): ApiErrorDetail {
  if (!error || typeof error !== 'object') {
    return {};
  }

  const response = (
    error as {
      response?: { data?: { code?: unknown; msg?: unknown }; status?: unknown; headers?: unknown };
    }
  ).response;
  const responseData = response?.data;
  const responseCode = responseData?.code;
  const responseMsg = responseData?.msg;

  const code =
    typeof responseCode === 'string' ? responseCode : typeof responseCode === 'number' ? String(responseCode) : undefined;
  const detail: ApiErrorDetail = { code };

  if (typeof response?.status === 'number') {
    detail.status = response.status;
  }

  const requestId = getResponseHeader(response?.headers, 'x-request-id');
  if (requestId) {
    detail.requestId = requestId;
  }

  if (typeof responseMsg === 'string' && responseMsg.trim()) {
    return { ...detail, msg: responseMsg };
  }

  const codeFromError = typeof (error as { code?: unknown }).code === 'string' ? (error as { code?: string }).code : undefined;

  const msg = (error as { msg?: unknown }).msg;
  if (typeof msg === 'string' && msg.trim()) {
    return { ...detail, code: code || codeFromError, msg };
  }

  const sharedMsg = extractSharedApiErrorMessage(error);
  if (sharedMsg) {
    return { ...detail, code: code || codeFromError || extractApiErrorCode(error), msg: sharedMsg };
  }

  return { ...detail, code: code || codeFromError || extractApiErrorCode(error) };
}

export function extractApiErrorMessage(error: unknown, fallback: string): string {
  return extractApiErrorDetail(error).msg || fallback;
}

export function unwrapApiResponse<T>(
  payload: ApiResponse<T>,
  functionName: string,
  options?: {
    allowEmptyData?: boolean;
    defaultData?: T;
  },
): T {
  if (!isSuccessCode(payload.code)) {
    console.warn(`[${functionName}] non-OK response`, { code: payload.code, msg: payload.msg });
    throw toError(payload.msg || 'Request failed', String(payload.code));
  }

  if (payload.data === undefined) {
    if (options?.allowEmptyData) {
      if (options.defaultData !== undefined) {
        return options.defaultData;
      }
      console.warn(`[${functionName}] response data is undefined and defaultData is missing`);
      throw toError(payload.msg || 'Empty response data', String(payload.code));
    }
    console.warn(`[${functionName}] response data is undefined`);
    throw toError(payload.msg || 'Empty response data', String(payload.code));
  }

  return payload.data;
}

export async function postData<TReq, TData>(http: AppHttpClient, url: string, body: TReq, functionName: string): Promise<TData> {
  const response = await http.post<ApiResponse<TData>>(url, body);
  return unwrapApiResponse(response.data, functionName);
}

export async function postNoData<TReq>(http: AppHttpClient, url: string, body: TReq, functionName: string): Promise<void> {
  const response = await http.post<ApiResponse>(url, body);
  if (!isSuccessCode(response.data.code)) {
    console.warn(`[${functionName}] non-OK response`, { code: response.data.code, msg: response.data.msg });
    throw toError(response.data.msg || 'Request failed', String(response.data.code));
  }
}
