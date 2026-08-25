import type { Locator, Page } from '@playwright/test';

import type { SemanticModel, SemanticModelSource } from '@moi/shared-moi-api/knowledge';
import { createKnowledgeSourceFixture } from '@moi/shared-moi-mock/knowledge/factories';
import { expect, test } from '../_fixtures';

const TABLE_SOURCE_ROW_ID = '9002:table:200';
const FILE_SOURCE_ROW_ID = '9002:file:knowledge-source-file-1';

const model: SemanticModel = {
  id: 9002,
  name: 'Issue 14801 Knowledge Base',
  description: 'Knowledge source action layout regression fixture',
  tables: [],
  files: { file_ids: ['knowledge-source-file-1'] },
  source_counts: { files: 1, tables: 4, total: 5 },
  table_set_hash: '',
  created_at: 1782705000,
  updated_at: 1782705000,
};

function createTableSource(index: number): SemanticModelSource {
  return {
    row_id: `9002:table:${200 + index}`,
    source_id: `9002:table:${200 + index}`,
    source_type: 'table',
    model_id: 9002,
    resource_id: String(200 + index),
    source_resource_id: String(100 + index),
    kb_resource_id: String(200 + index),
    source_table_id: 100 + index,
    kb_table_id: 200 + index,
    display_name: `sqlserver_import.dbo.customer_orders_${index + 1}`,
    path: [],
    source_path: 'sqlserver_import/dbo',
    db_name: 'sqlserver_import',
    table_name: `customer_orders_${index + 1}`,
    size_bytes: null,
    row_count: 12800 + index,
    ingest_status: 'completed',
    enabled: true,
    expires_at: null,
    expired: false,
    effective_enabled: true,
    force_enabled_after_expiry: false,
    tags: [],
    segment_version_id: null,
    index_version: null,
    created_by: null,
    updated_by: null,
    updated_at: 1782705000,
    error: null,
    governance_status: 'managed',
    legacy_origin: null,
  };
}

const tableSources: SemanticModelSource[] = Array.from({ length: 4 }, (_, index) => createTableSource(index));

const fileSource = createKnowledgeSourceFixture({
  row_id: FILE_SOURCE_ROW_ID,
  source_id: FILE_SOURCE_ROW_ID,
  source_type: 'file',
  model_id: 9002,
  resource_id: 'knowledge-source-file-1',
  kb_resource_id: 'knowledge-source-file-1',
  display_name: 'knowledge-source-guide.pdf',
  size_bytes: 2048,
  ingest_status: 'completed',
  enabled: true,
});

const sources = [...tableSources, fileSource];

interface KnowledgeSourceRouteFixture {
  model: SemanticModel;
  sources: SemanticModelSource[];
}

interface SourceActionLayout {
  actionCellPosition?: string;
  actionCount: number;
  actionsFit: boolean;
  actionsVisible: boolean;
  clientWidth?: number;
  scrollWidth?: number;
}

async function setupKnowledgeSourceRoutes(page: Page, fixture: KnowledgeSourceRouteFixture) {
  await page.route('**/newmoi/semantic-models/9002', (route) => {
    if (route.request().method() !== 'GET') return route.abort('connectionrefused');
    return route.fulfill({ status: 200, json: { code: 'OK', msg: '', data: fixture.model } });
  });
  await page.route('**/newmoi/semantic-models/9002/sources*', (route) => {
    if (route.request().method() !== 'GET') return route.abort('connectionrefused');
    const requestUrl = new URL(route.request().url());
    const page = Number(requestUrl.searchParams.get('page'));
    const pageSize = Number(requestUrl.searchParams.get('page_size'));
    if (!Number.isInteger(page) || page < 1 || !Number.isInteger(pageSize) || pageSize < 1) {
      return route.abort('connectionrefused');
    }
    const offset = (page - 1) * pageSize;
    return route.fulfill({
      status: 200,
      json: {
        code: 'OK',
        msg: '',
        data: {
          items: fixture.sources.slice(offset, offset + pageSize),
          page,
          page_size: pageSize,
          total: fixture.sources.length,
        },
      },
    });
  });
  await page.route('**/newmoi/semantic-models/9002/source-jobs', (route) => {
    if (route.request().method() !== 'GET') return route.abort('connectionrefused');
    return route.fulfill({
      status: 200,
      json: { code: 'OK', msg: '', data: { items: [], total: 0, reconcile_required: false } },
    });
  });
}

async function readActionLayout(table: Locator, sourceRowId: string, expectedActionCount: number): Promise<SourceActionLayout> {
  return table.evaluate(
    (element, { sourceRowId: rowId, expectedActions }) => {
      const scrollContainer = element.querySelector<HTMLElement>('.ant-table-tbody-virtual-holder, .ant-table-content');
      const sourceRow = element.querySelector<HTMLElement>(`[data-row-key="${rowId}"]`);
      const actionCell =
        Array.from(sourceRow?.children ?? []).find(
          (cell) =>
            cell.classList.contains('ant-table-cell-fix-right') || cell.classList.contains('ant-table-cell-fix-right-last'),
        ) ?? sourceRow?.lastElementChild;
      const actionControls = Array.from(actionCell?.querySelectorAll<HTMLElement>('[data-testid^="knowledge-source-"]') ?? []);
      const cellRect = actionCell?.getBoundingClientRect();
      const scrollRect = scrollContainer?.getBoundingClientRect();
      const contains = (outer: DOMRect, inner: DOMRect) =>
        inner.left >= outer.left - 0.5 &&
        inner.right <= outer.right + 0.5 &&
        inner.top >= outer.top - 0.5 &&
        inner.bottom <= outer.bottom + 0.5;
      return {
        actionCellPosition: actionCell ? getComputedStyle(actionCell).position : undefined,
        actionCount: actionControls.length,
        actionsFit: Boolean(
          cellRect &&
            actionControls.length === expectedActions &&
            actionControls.every((control) => contains(cellRect, control.getBoundingClientRect())),
        ),
        actionsVisible: Boolean(
          scrollRect && actionControls.every((control) => contains(scrollRect, control.getBoundingClientRect())),
        ),
        clientWidth: scrollContainer?.clientWidth,
        scrollWidth: scrollContainer?.scrollWidth,
      };
    },
    { sourceRowId, expectedActions: expectedActionCount },
  );
}

function expectCompleteFixedActions(layout: SourceActionLayout, expectedActionCount: number) {
  expect(layout.actionCellPosition).toBe('sticky');
  expect(layout.actionCount).toBe(expectedActionCount);
  expect(layout.actionsFit).toBe(true);
  expect(layout.actionsVisible).toBe(true);
}

async function scrollTableToRight(table: Locator) {
  await table.locator('.ant-table-tbody-virtual-holder, .ant-table-content').evaluate((element) => {
    element.scrollLeft = element.scrollWidth;
  });
}

test.describe('知识库数据源操作列布局', () => {
  test('结构化表的操作列固定在右侧且完整容纳全部操作', async ({ page }, testInfo) => {
    await setupKnowledgeSourceRoutes(page, { model, sources });

    await page.setViewportSize({ width: 1502, height: 667 });
    await page.goto('/ws-001/knowledge/9002/edit', { waitUntil: 'domcontentloaded' });

    const table = page.getByTestId('knowledge-source-table');
    await expect(table).toBeVisible({ timeout: 15_000 });
    await expect(page.getByTestId('knowledge-source-sql-btn').first()).toBeVisible();
    await expect(page.getByTestId('knowledge-source-expiry-btn').first()).toBeVisible();
    await expect(page.getByTestId('knowledge-source-download-btn').first()).toBeVisible();
    await expect(page.getByTestId('knowledge-source-delete-btn').first()).toBeVisible();

    const initialTableLayout = await readActionLayout(table, TABLE_SOURCE_ROW_ID, 4);
    const initialFileLayout = await readActionLayout(table, FILE_SOURCE_ROW_ID, 3);
    expectCompleteFixedActions(initialTableLayout, 4);
    expect(initialTableLayout.scrollWidth).toBeGreaterThan(initialTableLayout.clientWidth ?? 0);
    expectCompleteFixedActions(initialFileLayout, 3);

    await scrollTableToRight(table);
    const rightScrollTableLayout = await readActionLayout(table, TABLE_SOURCE_ROW_ID, 4);
    const rightScrollFileLayout = await readActionLayout(table, FILE_SOURCE_ROW_ID, 3);
    expectCompleteFixedActions(rightScrollTableLayout, 4);
    expectCompleteFixedActions(rightScrollFileLayout, 3);
    await page.screenshot({ path: testInfo.outputPath('knowledge-source-actions-right-edge.png') });

    const verifyViewport = async (viewport: { name: 'desktop' | 'mobile'; width: number; height: number }) => {
      await page.setViewportSize(viewport);
      await expect
        .poll(async () => {
          const layout = await readActionLayout(table, TABLE_SOURCE_ROW_ID, 4);
          return layout.actionsFit && layout.actionsVisible;
        })
        .toBe(true);

      expectCompleteFixedActions(await readActionLayout(table, TABLE_SOURCE_ROW_ID, 4), 4);
      expectCompleteFixedActions(await readActionLayout(table, FILE_SOURCE_ROW_ID, 3), 3);
      if (viewport.name === 'mobile') {
        await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true);
      }
      await table.scrollIntoViewIfNeeded();
      await page.screenshot({ path: testInfo.outputPath(`knowledge-source-actions-${viewport.name}.png`) });
    };

    await verifyViewport({ name: 'desktop', width: 1440, height: 900 });
    await verifyViewport({ name: 'mobile', width: 390, height: 844 });
  });
});
