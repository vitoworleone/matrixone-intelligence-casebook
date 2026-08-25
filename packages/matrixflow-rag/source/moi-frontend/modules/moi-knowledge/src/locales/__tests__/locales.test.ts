import { describe, expect, it } from 'vitest';

import enUS from '../en-US.json';
import zhCN from '../zh-CN.json';

describe('moi-knowledge locale resources', () => {
  it('uses the agreed references title in both supported locales', () => {
    expect(zhCN['knowledge.explore.final-answer-sources']).toBe('参考来源');
    expect(enUS['knowledge.explore.final-answer-sources']).toBe('References');
  });

  it('provides localized answer feedback labels in both supported locales', () => {
    expect(zhCN['knowledge.explore.answer-feedback-like']).toBe('回答有帮助');
    expect(zhCN['knowledge.explore.answer-feedback-dislike']).toBe('回答没有帮助');
    expect(zhCN['knowledge.explore.answer-feedback-submit-failed']).toBe('反馈提交失败，请稍后重试');
    expect(enUS['knowledge.explore.answer-feedback-like']).toBe('Good response');
    expect(enUS['knowledge.explore.answer-feedback-dislike']).toBe('Bad response');
    expect(enUS['knowledge.explore.answer-feedback-submit-failed']).toBe('Failed to submit feedback. Please try again.');
  });
});
