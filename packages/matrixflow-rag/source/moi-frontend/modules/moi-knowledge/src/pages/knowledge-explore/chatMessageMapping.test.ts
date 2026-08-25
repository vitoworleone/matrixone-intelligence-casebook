import { describe, expect, it } from 'vitest';

import { mapExploreAssistantChatMessage } from './chatMessageMapping';
import type { ExploreA2AMessageItem } from './services/exploreA2ARuntimeStore';
import { createExploreA2AProjection } from './services/exploreA2AStreamParser';

describe('Explore chat message mapping', () => {
  it('passes draft visibility to shared A2A assistant messages for streaming Explore answers', () => {
    const projection = {
      ...createExploreA2AProjection(),
      taskId: 'task_1',
      contextId: 'ctx_1',
      status: 'working' as const,
      assistantDraft: '流式草稿',
    };
    const message: ExploreA2AMessageItem = {
      id: 'assistant_task_1',
      role: 'assistant',
      content: '流式草稿',
      createdAt: 1000,
      knowledgeIds: [7],
      projection,
      feedback: null,
    };

    expect(mapExploreAssistantChatMessage(message, 'Asia/Shanghai')).toMatchObject({
      id: 'assistant_task_1',
      role: 'assistant',
      content: '流式草稿',
      projection,
      showDraftAnswer: true,
      testId: 'knowledge-explore-message-assistant_task_1',
      data: message,
    });
  });
});
