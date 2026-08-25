import type { AgentChatWorkspaceMessage } from '@moi/shared-moi-components/ai-chat-message/agent-chat-page';
import { formatDateTime } from '@moi/shared-moi-utils/datetime';
import type { ExploreA2AMessageItem } from './services/exploreA2ARuntimeStore';

export function mapExploreAssistantChatMessage(
  message: ExploreA2AMessageItem,
  timezone: string,
): AgentChatWorkspaceMessage<ExploreA2AMessageItem> {
  return {
    id: message.id,
    role: 'assistant',
    content: message.content,
    projection: message.projection,
    showDraftAnswer: true,
    time: formatDateTime(message.createdAt, { timezone }),
    testId: `knowledge-explore-message-${message.id}`,
    data: message,
  };
}
