import type { Locator, Page } from '@playwright/test';

import { expect, test } from '../_fixtures';

async function openCreateModal(page: Page) {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/ws-001/knowledge');
  await expect(page.getByTestId('knowledge-base-create-btn')).toBeVisible({ timeout: 15_000 });
  await page.getByTestId('knowledge-base-create-btn').click();
  const modal = page.getByTestId('knowledge-base-create-modal').locator('.ant-modal');
  await expect(modal).toBeVisible();
  return modal;
}

async function expectInsideViewport(modal: Locator) {
  const layout = await modal.evaluate((element) => {
    const rect = element.getBoundingClientRect();
    return {
      left: rect.left,
      right: rect.right,
      top: rect.top,
      bottom: rect.bottom,
      width: window.innerWidth,
      height: window.innerHeight,
    };
  });
  expect(layout.left).toBeGreaterThanOrEqual(0);
  expect(layout.right).toBeLessThanOrEqual(layout.width);
  expect(layout.top).toBeGreaterThanOrEqual(0);
  expect(layout.bottom).toBeLessThanOrEqual(layout.height);
}

test.describe('数据侧知识库创建弹窗', () => {
  test('只展示基本信息和高级索引选项，并适配桌面与移动视口', async ({ page }, testInfo) => {
    const modal = await openCreateModal(page);

    await expect(page.getByTestId('knowledge-base-create-name-input')).toBeVisible();
    await expect(page.getByTestId('knowledge-base-create-description-input')).toBeVisible();
    await expect(page.getByTestId('knowledge-base-create-advanced-options')).toBeVisible();
    await page.getByTestId('knowledge-base-create-advanced-options').locator('.ant-collapse-header').click();
    await expect(page.getByTestId('knowledge-base-create-image-index-switch')).toBeVisible();
    await expect(page.getByTestId('knowledge-base-create-source-choice')).toHaveCount(0);
    await expect(page.getByTestId('knowledge-base-create-base-next-btn')).toHaveCount(0);
    await expect(page.getByTestId('knowledge-base-create-active-step-heading')).toHaveCount(0);
    await expect(page.getByTestId('knowledge-base-create-submit-btn')).toHaveText(/完\s*成/);
    await expectInsideViewport(modal);
    await page.screenshot({ path: testInfo.outputPath('knowledge-create-modal-desktop.png') });

    await page.setViewportSize({ width: 390, height: 844 });
    await expectInsideViewport(modal);
    await page.screenshot({ path: testInfo.outputPath('knowledge-create-modal-mobile.png') });
  });

  test('提交空知识库接口且不携带来源字段', async ({ page }) => {
    await openCreateModal(page);
    await page.getByTestId('knowledge-base-create-name-input').fill('data_kb');
    await page.getByTestId('knowledge-base-create-description-input').fill('created from data');

    const requestPromise = page.waitForRequest((request) => request.url().endsWith('/newmoi/semantic-models/create-empty'));
    await page.getByTestId('knowledge-base-create-submit-btn').click();
    const request = await requestPromise;
    const body = request.postDataJSON();

    expect(body).toEqual({ name: 'data_kb', description: 'created from data', image_index_enabled: false });
    expect(body).not.toHaveProperty('sources');
    expect(body).not.toHaveProperty('source_selections');
    await expect(page.getByTestId('knowledge-base-create-modal')).toHaveCount(0);
  });
});
