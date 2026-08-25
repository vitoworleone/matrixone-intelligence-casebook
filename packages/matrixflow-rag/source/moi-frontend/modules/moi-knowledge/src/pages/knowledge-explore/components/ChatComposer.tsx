import { useMemo } from 'react';
import { Button, Checkbox, Dropdown, Flex, message, Popover, Select, Spin, Typography, Upload, type UploadProps } from 'antd';
import type { DefaultOptionType } from 'antd/es/select';
import { CloseOutlined, FolderOpenFilled, PictureOutlined, PlusOutlined } from '@ant-design/icons';
import { Sender } from '@ant-design/x';
import type { TFunction } from 'i18next';

import styles from '../KnowledgeExplore.module.css';

interface KnowledgeOption {
  id: number;
  name: string;
  tables?: Array<{ table_names?: string[] }>;
  files?: { file_ids?: string[] };
  /** First file name for display (from old KnowledgeListItem) */
  first_file_name?: string;
}

interface ModelOption {
  value: string;
  label: string;
  backendId: number;
}

interface QueryVisualOption {
  fileId: string;
  fileName: string;
  mimeType: string;
  size: number;
}

interface ComposerRuntimeState {
  draftInput: string;
  selectedModel: string;
  selectedReasoningEffort: string;
  selectedKnowledgeIds: number[];
  isSendPending: boolean;
  isStreaming: boolean;
  queryVisuals: QueryVisualOption[];
}

interface ModelOptGroup {
  label: string;
  options: ModelOption[];
}

interface ChatComposerProps {
  currentRuntime: ComposerRuntimeState;
  selectedSessionId: number | null;
  canCreateSessionOnSend?: boolean;
  isSessionCreating?: boolean;
  selectedModelValue?: string;
  t: TFunction<'moi-knowledge'>;
  knowledgeList: KnowledgeOption[];
  isKnowledgeLoading: boolean;
  modelOptions: ModelOption[];
  modelSelectOptions?: ModelOptGroup[];
  isModelLoading: boolean;
  isModelLocked?: boolean;
  isKnowledgeLocked?: boolean;
  features?: {
    selectedKnowledgeHeader?: boolean;
    knowledgeSelector?: boolean;
    imageUpload?: boolean;
  };
  quickActions?: Array<{
    key: string;
    label: string;
    disabled?: boolean;
    loading?: boolean;
    onClick: () => void;
  }>;
  onDraftChange: (value: string) => void;
  onModelChange: (value: string, backendId?: number | null) => void;
  onToggleKnowledge: (id: number) => void;
  onRemoveKnowledge: (id: number) => void;
  onUploadImage: (file: File) => Promise<void>;
  onRemoveQueryVisual: (fileId: string) => void;
  onSend: () => void;
  onStop: () => void;
}

export function ChatComposer({
  currentRuntime,
  selectedModelValue,
  selectedSessionId,
  canCreateSessionOnSend = false,
  isSessionCreating = false,
  t,
  knowledgeList,
  isKnowledgeLoading,
  modelOptions,
  modelSelectOptions,
  isModelLoading,
  isModelLocked = false,
  isKnowledgeLocked = false,
  features,
  quickActions,
  onDraftChange,
  onModelChange,
  onToggleKnowledge,
  onRemoveKnowledge,
  onUploadImage,
  onRemoveQueryVisual,
  onSend,
  onStop,
}: ChatComposerProps) {
  const isRuntimeBusy = currentRuntime.isStreaming || currentRuntime.isSendPending;
  const isComposerBusy = isRuntimeBusy || isSessionCreating;
  const canUseComposer = Boolean(selectedSessionId) || canCreateSessionOnSend;
  const hasSession = Boolean(selectedSessionId);
  const showSelectedKnowledgeHeader = features?.selectedKnowledgeHeader ?? true;
  const showKnowledgeSelector = features?.knowledgeSelector ?? true;
  const showImageUpload = features?.imageUpload ?? true;
  const hasModel = Boolean(currentRuntime.selectedModel?.trim());
  const hasKnowledge = currentRuntime.selectedKnowledgeIds.length > 0;
  const hasInput = Boolean(currentRuntime.draftInput.trim());
  const hasQueryVisuals = currentRuntime.queryVisuals.length > 0;
  const canSend = hasInput || hasQueryVisuals;
  const visibleQuickActions = quickActions ?? [];
  const isImageUploadDisabled = !hasSession || isComposerBusy;

  // Build selected knowledge items with full info for card display
  const selectedKnowledgeItems = useMemo(
    () =>
      currentRuntime.selectedKnowledgeIds
        .map((id) => knowledgeList.find((k) => k.id === id))
        .filter((k): k is KnowledgeOption => Boolean(k)),
    [currentRuntime.selectedKnowledgeIds, knowledgeList],
  );

  const renderKnowledgeSelector = () => (
    <div className={styles.knowledgePopover} data-testid="knowledge-explore-knowledge-popover">
      <div className={styles.popoverTitle}>{t('knowledge.explore.knowledge-dropdown-title')}</div>
      <div className={styles.knowledgeOptionList}>
        {isKnowledgeLoading ? (
          <div className={styles.popoverLoading}>
            <Spin />
          </div>
        ) : knowledgeList.length > 0 ? (
          knowledgeList.map((item) => (
            <label key={item.id} className={styles.knowledgeOptionItem}>
              <Checkbox
                checked={currentRuntime.selectedKnowledgeIds.includes(item.id)}
                onChange={() => onToggleKnowledge(item.id)}
                data-testid={`knowledge-explore-knowledge-option-${item.id}`}
              >
                {item.name}
              </Checkbox>
            </label>
          ))
        ) : (
          <div className={styles.popoverEmptyText}>{t('knowledge.explore.knowledge-empty')}</div>
        )}
      </div>
    </div>
  );

  const composerHeader = useMemo(() => {
    const visibleKnowledgeItems = showSelectedKnowledgeHeader ? selectedKnowledgeItems : [];
    const hasHeaderContent = visibleKnowledgeItems.length > 0 || currentRuntime.queryVisuals.length > 0;
    if (!hasHeaderContent) return null;
    return (
      <div className={styles.composerHeaderWrap}>
        <div className={styles.knowledgeHeaderWrap}>
          {visibleKnowledgeItems.length > 0 ? (
            <div className={styles.knowledgeScrollList} data-testid="knowledge-explore-selected-knowledge-list">
              {visibleKnowledgeItems.map((item) => {
                return (
                  <div
                    key={item.id}
                    className={styles.knowledgeCard}
                    data-testid={`knowledge-explore-selected-knowledge-${item.id}`}
                  >
                    <div className={styles.knowledgeCardIcon}>
                      <FolderOpenFilled style={{ fontSize: 12, color: '#fff' }} />
                    </div>
                    <div className={styles.knowledgeCardInfo}>
                      <Typography.Paragraph ellipsis={{ rows: 1, tooltip: item.name }} className={styles.knowledgeCardName}>
                        {item.name}
                      </Typography.Paragraph>
                      {/* {totalCount > 0 && (
                    <Typography.Text className={styles.knowledgeCardMeta}>
                      {t('knowledge.explore.knowledge-files-count', { count: totalCount })}
                    </Typography.Text>
                  )} */}
                    </div>
                    {isKnowledgeLocked ? null : (
                      <CloseOutlined
                        className={styles.knowledgeCardClose}
                        onClick={(e) => {
                          e.stopPropagation();
                          onRemoveKnowledge(item.id);
                        }}
                      />
                    )}
                  </div>
                );
              })}
            </div>
          ) : null}
          {currentRuntime.queryVisuals.length > 0 ? (
            <div className={styles.queryVisualList} data-testid="knowledge-explore-query-visual-list">
              {currentRuntime.queryVisuals.map((image) => (
                <div key={image.fileId} className={styles.queryVisualChip}>
                  <PictureOutlined className={styles.queryVisualIcon} />
                  <Typography.Text ellipsis={{ tooltip: image.fileName }} className={styles.queryVisualName}>
                    {image.fileName}
                  </Typography.Text>
                  <CloseOutlined className={styles.queryVisualRemove} onClick={() => onRemoveQueryVisual(image.fileId)} />
                </div>
              ))}
            </div>
          ) : null}
          {visibleKnowledgeItems.length > 0 ? <div className={styles.knowledgeScrollFade} /> : null}
        </div>
      </div>
    );
  }, [
    currentRuntime.queryVisuals,
    isKnowledgeLocked,
    onRemoveKnowledge,
    onRemoveQueryVisual,
    selectedKnowledgeItems,
    showSelectedKnowledgeHeader,
  ]);

  const handleSubmit = () => {
    if (!canUseComposer || isComposerBusy) return;
    if (!hasModel) {
      message.warning(t('knowledge.explore.model-required'));
      return;
    }
    if (!hasKnowledge) {
      message.warning(t('knowledge.explore.knowledge-required'));
      return;
    }
    if (!canSend) return;
    onSend();
  };

  const imageUploadProps: UploadProps = {
    accept: 'image/*',
    showUploadList: false,
    beforeUpload: (file) => {
      onUploadImage(file).catch((error) => {
        console.warn('[ChatComposer] upload image failed', { msg: (error as Error).message });
        message.error(t('knowledge.explore.query-visual-upload-failed'));
      });
      return false;
    },
  };

  const renderUploadMenu = () => (
    <div className={styles.composerUploadMenu} data-testid="knowledge-explore-upload-menu">
      <Upload {...imageUploadProps} disabled={isImageUploadDisabled}>
        <Button
          type="text"
          icon={<PictureOutlined />}
          className={styles.composerUploadAction}
          disabled={isImageUploadDisabled}
          data-testid="knowledge-explore-query-visual-upload-btn"
        >
          {t('knowledge.explore.query-visual-upload-btn')}
        </Button>
      </Upload>
    </div>
  );

  return (
    <div className={styles.composerWrap}>
      <Sender
        value={currentRuntime.draftInput}
        onChange={onDraftChange}
        onSubmit={handleSubmit}
        placeholder={t('knowledge.explore.composer-placeholder')}
        autoSize={{ minRows: 1, maxRows: 6 }}
        loading={isComposerBusy}
        disabled={!canUseComposer || isSessionCreating}
        header={composerHeader}
        suffix={false}
        className={styles.sender}
        data-testid="knowledge-explore-composer-sender"
        footer={(_actions, { components }) => {
          const { SendButton } = components;
          return (
            <Flex
              className={styles.composerFooter}
              justify="space-between"
              align="center"
              data-testid="knowledge-explore-composer-footer"
            >
              <Flex
                className={styles.composerToolbarLeft}
                gap="small"
                align="center"
                wrap
                data-testid="knowledge-explore-composer-toolbar"
              >
                {showImageUpload ? (
                  <Dropdown
                    menu={{ items: [] }}
                    popupRender={renderUploadMenu}
                    trigger={['click']}
                    placement="topLeft"
                    disabled={isImageUploadDisabled}
                  >
                    <Button
                      type="text"
                      icon={<PlusOutlined />}
                      className={styles.composerPlusButton}
                      disabled={isImageUploadDisabled}
                      aria-label={t('knowledge.explore.query-visual-upload-btn')}
                      data-testid="knowledge-explore-upload-menu-btn"
                    />
                  </Dropdown>
                ) : null}
                {visibleQuickActions.map((action) => (
                  <Button
                    key={action.key}
                    type="primary"
                    disabled={!canUseComposer || isComposerBusy || action.disabled}
                    loading={action.loading}
                    onClick={action.onClick}
                    data-testid={`knowledge-explore-quick-action-${action.key}`}
                  >
                    {action.label}
                  </Button>
                ))}
                <Select
                  value={selectedModelValue}
                  options={(modelSelectOptions ?? modelOptions) as DefaultOptionType[]}
                  className={styles.modelSelect}
                  disabled={!canUseComposer || isComposerBusy || modelOptions.length === 0 || isModelLocked}
                  loading={isModelLoading}
                  showSearch
                  optionFilterProp="label"
                  placeholder={t('knowledge.explore.model-select-placeholder')}
                  notFoundContent={t('knowledge.explore.model-empty')}
                  onChange={(value, option) => {
                    const selected = (Array.isArray(option) ? option[0] : option) as { backendId?: number } | undefined;
                    onModelChange(value, selected?.backendId ?? null);
                  }}
                  data-testid="knowledge-explore-model-select"
                />
                {showKnowledgeSelector ? (
                  <Popover trigger="click" placement="topLeft" content={renderKnowledgeSelector}>
                    <Button
                      icon={<PlusOutlined />}
                      disabled={!canUseComposer || isComposerBusy || isKnowledgeLocked}
                      data-testid="knowledge-explore-knowledge-selector-btn"
                    >
                      {t('knowledge.explore.knowledge-select-btn')}
                    </Button>
                  </Popover>
                ) : null}
              </Flex>
              <Flex className={styles.composerToolbarRight} align="center">
                {isRuntimeBusy ? (
                  <Button
                    type="default"
                    danger
                    icon={<CloseOutlined />}
                    onClick={onStop}
                    data-testid="knowledge-explore-composer-stop-btn"
                  >
                    {t('knowledge.explore.stream-stop-btn')}
                  </Button>
                ) : (
                  <SendButton
                    type="primary"
                    disabled={!canUseComposer || !canSend || isSessionCreating}
                    data-testid="knowledge-explore-composer-send-btn"
                  />
                )}
              </Flex>
            </Flex>
          );
        }}
      />
    </div>
  );
}
