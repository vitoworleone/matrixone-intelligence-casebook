import { describe, expect, it } from 'vitest';

import type { ApiResponse } from '@moi/shared-moi-api/types';
import { extractApiErrorDetail, unwrapApiResponse } from '../http';

describe('unwrapApiResponse', () => {
  it('returns payload data when code is OK', () => {
    const payload: ApiResponse<{ id: number }> = {
      code: 'OK',
      msg: 'ok',
      data: { id: 1 },
    };

    expect(unwrapApiResponse(payload, 'test-unwrap')).toEqual({ id: 1 });
  });

  it('treats code 0 as success', () => {
    const payload: ApiResponse<{ success: boolean }> = {
      code: 0,
      msg: 'ok',
      data: { success: true },
    };

    expect(unwrapApiResponse(payload, 'test-unwrap')).toEqual({ success: true });
  });

  it('returns default data when allowEmptyData is enabled', () => {
    const payload: ApiResponse<{ success: boolean }> = {
      code: 'OK',
      msg: 'ok',
    };

    expect(
      unwrapApiResponse(payload, 'test-unwrap', {
        allowEmptyData: true,
        defaultData: { success: true },
      }),
    ).toEqual({ success: true });
  });

  it('throws when response code is not success', () => {
    const payload: ApiResponse<{ success: boolean }> = {
      code: 'ERR',
      msg: 'bad request',
    };

    expect(() => unwrapApiResponse(payload, 'test-unwrap')).toThrow('bad request');
  });

  it('preserves business error code when response code is not success', () => {
    const payload: ApiResponse<{ success: boolean }> = {
      code: 'ErrNotFound',
      msg: 'knowledge item not found',
    };

    try {
      unwrapApiResponse(payload, 'test-unwrap');
      expect.fail('should throw');
    } catch (error) {
      expect(extractApiErrorDetail(error)).toEqual({
        code: 'ErrNotFound',
        msg: 'knowledge item not found',
      });
    }
  });

  it('throws when data is missing and allowEmptyData is disabled', () => {
    const payload: ApiResponse<{ success: boolean }> = {
      code: 'OK',
      msg: 'ok',
    };

    expect(() => unwrapApiResponse(payload, 'test-unwrap')).toThrow('ok');
  });
});

describe('extractApiErrorDetail', () => {
  it('prefers response.data code and msg', () => {
    const error = {
      response: {
        status: 400,
        headers: { 'x-request-id': 'req-123' },
        data: {
          code: 'ErrTenantDBConnection',
          msg: '获取工作区数据库连接失败',
        },
      },
      message: 'Request failed with status code 500',
    };

    expect(extractApiErrorDetail(error)).toEqual({
      code: 'ErrTenantDBConnection',
      msg: '获取工作区数据库连接失败',
      status: 400,
      requestId: 'req-123',
    });
  });

  it('reads request ID from case-insensitive response headers', () => {
    const error = {
      response: {
        status: 400,
        headers: new Map([['X-Request-ID', 'req-456']]),
        data: { msg: 'invalid semantic config' },
      },
    };

    expect(extractApiErrorDetail(error)).toMatchObject({
      msg: 'invalid semantic config',
      status: 400,
      requestId: 'req-456',
    });
  });

  it('falls back to error msg when response data is missing', () => {
    const error = {
      msg: 'fallback msg',
      message: 'request failed',
    };

    expect(extractApiErrorDetail(error)).toEqual({
      code: undefined,
      msg: 'fallback msg',
    });
  });
});
