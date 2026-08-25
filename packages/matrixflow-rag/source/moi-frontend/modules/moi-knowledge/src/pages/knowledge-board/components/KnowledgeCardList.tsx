/* oxlint-disable react-hooks/exhaustive-deps */
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { App, Button, Card, Col, Collapse, Form, Input, Row, Spin, Switch, Typography } from 'antd';
import { LoadingOutlined, PlusOutlined, SearchOutlined } from '@ant-design/icons';

import { isApiConflictError } from '@moi/shared-moi-api/response';
import { useHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { useModuleNavigator } from '@moi/shared-moi-app-protocol/module-context';
import { isValidCatalogIdentifier } from '@moi/shared-moi-utils/catalog/identifier';
import KnowledgeSourceSelectModal from '../../../components/KnowledgeSourceSelectModal/KnowledgeSourceSelectModal';
import {
  createEmptyKnowledge,
  queryKnowledgeList,
  type CreateEmptyKnowledgeRequest,
  type KnowledgeListItem,
} from '../../../service/knowledge';
import { resolveKnowledgeErrorMessage } from '../../../shared/knowledge/error-message';
import KnowledgeCard from './KnowledgeCard';
import styles from './KnowledgeCardList.module.css';

const PAGE_SIZE = 12;
const CREATE_SUCCESS_REDIRECT_DELAY_MS = 1000;

function mapToCardProps(item: KnowledgeListItem) {
  return {
    id: item.id,
    name: item.name,
    remark: item.description || '',
    volumeCount: 0,
    fileCount: item.source_counts.files,
    tablesCount: item.source_counts.tables,
  };
}

interface KnowledgeCardListProps {
  onDialogStarted?: (knowledgeId: number) => void;
}

interface KnowledgeCreateFormValues {
  name: string;
  description?: string;
  image_index_enabled?: boolean;
}

export default function KnowledgeCardList({ onDialogStarted }: KnowledgeCardListProps) {
  const { t } = useTranslation('moi-knowledge');
  const { message } = App.useApp();
  const http = useHttpClient();
  const nav = useModuleNavigator();
  const [createForm] = Form.useForm<KnowledgeCreateFormValues>();

  const [loading, setLoading] = useState(false);
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [createSubmitting, setCreateSubmitting] = useState(false);
  const [dataSource, setDataSource] = useState<KnowledgeListItem[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [initialLoaded, setInitialLoaded] = useState(false);
  const [searchKeyword, setSearchKeyword] = useState('');
  const nextPageTokenRef = useRef<string>('');
  const loadMoreRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(false);
  const searchTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const createRedirectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const canCreateKnowledgeBase = true;
  const fetchData = useCallback(
    async (pageToken?: string, append = false, search?: string) => {
      if (loadingRef.current) return;
      loadingRef.current = true;
      setLoading(true);

      try {
        const response = await queryKnowledgeList(http, {
          page_size: PAGE_SIZE,
          page_token: pageToken || undefined,
          search: search || undefined,
        });
        const list = response.items || [];

        if (append) {
          setDataSource((prev) => [...prev, ...list]);
        } else {
          setDataSource(list);
        }
        nextPageTokenRef.current = response.next_page_token || '';
        setHasMore(Boolean(response.next_page_token));
        setInitialLoaded(true);
      } catch (error) {
        const errorMessage = resolveKnowledgeErrorMessage(error, 'knowledge.base.load-list-failed');
        message.error(errorMessage);
        console.error('[KnowledgeCardList] fetch list failed', error);
        if (!append) {
          setDataSource([]);
        }
        setHasMore(false);
      } finally {
        setLoading(false);
        loadingRef.current = false;
      }
    },
    [http, message],
  );

  useEffect(() => {
    fetchData(undefined, false, searchKeyword);
  }, [fetchData, searchKeyword]);

  const handleRefresh = useCallback(() => {
    nextPageTokenRef.current = '';
    fetchData(undefined, false, searchKeyword);
  }, [fetchData, searchKeyword]);

  const resetCreateDialog = useCallback(() => {
    createForm.resetFields();
  }, [createForm]);

  const clearCreateRedirectTimer = useCallback(() => {
    if (createRedirectTimerRef.current) {
      clearTimeout(createRedirectTimerRef.current);
      createRedirectTimerRef.current = null;
    }
  }, []);

  useEffect(() => clearCreateRedirectTimer, [clearCreateRedirectTimer]);

  const handleCreateCancel = useCallback(() => {
    if (createSubmitting) return;
    clearCreateRedirectTimer();
    setCreateModalOpen(false);
    resetCreateDialog();
  }, [clearCreateRedirectTimer, createSubmitting, resetCreateDialog]);

  const handleCreateOpen = useCallback(() => {
    clearCreateRedirectTimer();
    setCreateModalOpen(true);
  }, [clearCreateRedirectTimer]);

  const handleCreateSubmit = useCallback(async () => {
    let values: KnowledgeCreateFormValues;
    try {
      values = await createForm.validateFields();
    } catch {
      return;
    }
    if (!values.name) {
      message.error(t('knowledge.base.form-name-required'));
      return;
    }

    try {
      setCreateSubmitting(true);
      const request: CreateEmptyKnowledgeRequest = {
        name: values.name,
        description: values.description,
        image_index_enabled: values.image_index_enabled ?? false,
      };
      const response = await createEmptyKnowledge(http, request);
      message.success({
        content: t('knowledge.base.create-success'),
        duration: CREATE_SUCCESS_REDIRECT_DELAY_MS / 1000,
      });
      setCreateModalOpen(false);
      resetCreateDialog();
      clearCreateRedirectTimer();
      createRedirectTimerRef.current = setTimeout(() => {
        createRedirectTimerRef.current = null;
        nav.goToPage('knowledge-edit', { params: { id: String(response.model.id) } });
      }, CREATE_SUCCESS_REDIRECT_DELAY_MS);
      handleRefresh();
    } catch (error) {
      if (isApiConflictError(error)) {
        const conflictMessage = resolveKnowledgeErrorMessage(error, 'knowledge.base.name-already-exists');
        createForm.setFields([{ name: 'name', errors: [conflictMessage] }]);
        message.error(conflictMessage);
      } else {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.create-failed'));
      }
      console.error('[KnowledgeCardList] create empty knowledge base failed', error);
    } finally {
      setCreateSubmitting(false);
    }
  }, [clearCreateRedirectTimer, createForm, handleRefresh, http, message, nav, resetCreateDialog, t]);

  const hasMoreRef = useRef(false);
  useEffect(() => {
    hasMoreRef.current = hasMore;
  }, [hasMore]);

  const searchKeywordRef = useRef(searchKeyword);
  useEffect(() => {
    searchKeywordRef.current = searchKeyword;
  }, [searchKeyword]);

  const handleLoadMore = useCallback(() => {
    if (!nextPageTokenRef.current || loadingRef.current || !hasMoreRef.current) return;
    fetchData(nextPageTokenRef.current, true, searchKeywordRef.current);
  }, [fetchData]);

  const handleSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    if (searchTimerRef.current) {
      clearTimeout(searchTimerRef.current);
    }
    searchTimerRef.current = setTimeout(() => {
      nextPageTokenRef.current = '';
      setSearchKeyword(value.trim());
    }, 500);
  }, []);

  // Cleanup debounce timer on unmount
  useEffect(() => {
    return () => {
      if (searchTimerRef.current) {
        clearTimeout(searchTimerRef.current);
      }
    };
  }, []);

  // IntersectionObserver for infinite scroll
  useEffect(() => {
    const el = loadMoreRef.current;
    if (!el) return;

    // Find the nearest scrollable ancestor to use as observer root
    let scrollParent: Element | null = el.parentElement;
    while (scrollParent) {
      const style = getComputedStyle(scrollParent);
      if (style.overflowY === 'auto' || style.overflowY === 'scroll') break;
      scrollParent = scrollParent.parentElement;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) handleLoadMore();
      },
      { root: scrollParent, rootMargin: '200px' },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [handleLoadMore, hasMore, loading]);

  return (
    <>
      <div className={styles.searchBar} data-testid="knowledge-card-list-search">
        <Input
          suffix={<SearchOutlined />}
          placeholder={t('knowledge.base.search-placeholder')}
          allowClear
          onChange={handleSearchChange}
          data-testid="knowledge-card-list-search-input"
        />
      </div>

      <Spin spinning={loading && dataSource.length === 0}>
        <Row gutter={[16, 16]}>
          {canCreateKnowledgeBase ? (
            <Col xs={24} sm={12} lg={8}>
              <Card
                hoverable
                className={styles.createCard}
                classNames={{ body: styles.createCardBody }}
                onClick={handleCreateOpen}
                data-testid="knowledge-base-create-card"
              >
                <Button
                  icon={<PlusOutlined />}
                  className={styles.createCardButton}
                  data-testid="knowledge-base-create-btn"
                  onClick={(event) => {
                    event.stopPropagation();
                    handleCreateOpen();
                  }}
                >
                  {t('knowledge.base.create-card-button')}
                </Button>
                <Typography.Text className={styles.createCardDesc}>
                  {t('knowledge.base.create-card-desc-line1')}
                  <br />
                  {t('knowledge.base.create-card-desc-line2')}
                </Typography.Text>
              </Card>
            </Col>
          ) : null}
          {dataSource.map((item) => {
            const cardProps = mapToCardProps(item);
            return (
              <Col xs={24} sm={12} lg={8} key={item.id}>
                <KnowledgeCard {...cardProps} onRefresh={handleRefresh} onDialogStarted={onDialogStarted} />
              </Col>
            );
          })}
        </Row>

        {/* Sentinel for infinite scroll + status text */}
        <div ref={loadMoreRef} className={styles.loadMoreWrap}>
          {hasMore && loading && (
            <>
              <LoadingOutlined style={{ marginRight: 8 }} />
              {t('knowledge.base.loading-more')}
            </>
          )}
          {initialLoaded && !hasMore && dataSource.length > 0 && (
            <Typography.Text type="secondary">{t('knowledge.base.no-more')}</Typography.Text>
          )}
        </div>
      </Spin>

      <KnowledgeSourceSelectModal
        open={createModalOpen}
        title={t('knowledge.base.create-document-sources-modal-title')}
        okText={t('knowledge.base.form-finish-button')}
        cancelText={t('knowledge.base.form-cancel-button')}
        submitting={createSubmitting}
        showCreateSteps
        basicStepOnly
        basicNextText={t('knowledge.base.form-finish-button')}
        basicNextButtonTestId="knowledge-base-create-submit-btn"
        cancelButtonTestId="knowledge-base-create-cancel-btn"
        testIdPrefix="knowledge-base-create"
        onBasicNext={async () => {
          await handleCreateSubmit();
          return false;
        }}
        onCancel={handleCreateCancel}
        onSubmit={async () => undefined}
        basicStepContent={
          <Form form={createForm} layout="vertical" preserve data-testid="knowledge-base-create-form">
            <div className={styles.createBaseStep} data-testid="knowledge-base-create-base-step">
              <Form.Item
                name="name"
                label={t('knowledge.base.form-name-label')}
                validateTrigger={['onChange', 'onBlur']}
                rules={[
                  { required: true, message: t('knowledge.base.form-name-required') },
                  {
                    validator: (_, value: string) => {
                      if (value && !isValidCatalogIdentifier(value)) {
                        return Promise.reject(new Error(t('knowledge.base.form-name-invalid')));
                      }
                      return Promise.resolve();
                    },
                  },
                ]}
              >
                <Input
                  maxLength={255}
                  showCount
                  placeholder={t('knowledge.base.form-name-placeholder')}
                  data-testid="knowledge-base-create-name-input"
                  disabled={createSubmitting}
                />
              </Form.Item>
              <Form.Item name="description" label={t('knowledge.base.form-remark-label')}>
                <Input.TextArea
                  rows={7}
                  placeholder={t('knowledge.base.form-remark-placeholder')}
                  data-testid="knowledge-base-create-description-input"
                  disabled={createSubmitting}
                />
              </Form.Item>
              <div data-testid="knowledge-base-create-advanced-options">
                <Collapse
                  size="small"
                  items={[
                    {
                      key: 'image-index',
                      label: t('knowledge.base.create-document-sources-advanced-options'),
                      children: (
                        <Form.Item
                          name="image_index_enabled"
                          label={t('knowledge.base.create-document-sources-image-index-enable')}
                          valuePropName="checked"
                          initialValue={false}
                        >
                          <Switch data-testid="knowledge-base-create-image-index-switch" disabled={createSubmitting} />
                        </Form.Item>
                      ),
                    },
                  ]}
                />
              </div>
            </div>
          </Form>
        }
      />
    </>
  );
}
