import { useState, type MouseEvent } from 'react';
import { useTranslation } from 'react-i18next';
import { App, Button, Card, Modal, Space, Typography } from 'antd';
import { CommentOutlined, FolderOpenFilled } from '@ant-design/icons';

import { isApiConflictError } from '@moi/shared-moi-api/response';
import { useHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { useModuleNavigator } from '@moi/shared-moi-app-protocol/module-context';
import { ListActionButton } from '@moi/shared-moi-components/list-action';
import { deleteKnowledgeById, getKnowledgeDetail, updateKnowledge } from '../../../service/knowledge';
import { resolveKnowledgeErrorMessage } from '../../../shared/knowledge/error-message';
import styles from './KnowledgeCard.module.css';
import KnowledgeMetadataEditModal, {
  type KnowledgeMetadataForm,
  type KnowledgeMetadataValues,
} from './KnowledgeMetadataEditModal';

const { Text, Paragraph } = Typography;

type KnowledgeCardProps = {
  id: number;
  name: string;
  remark: string;
  volumeCount?: number;
  fileCount?: number;
  tablesCount?: number;
  onRefresh?: () => void;
  onDialogStarted?: (knowledgeId: number) => void;
};

export default function KnowledgeCard({
  id,
  name,
  remark,
  volumeCount = 0,
  fileCount = 0,
  tablesCount = 0,
  onRefresh,
  onDialogStarted,
}: KnowledgeCardProps) {
  const { t } = useTranslation('moi-knowledge');
  const { message } = App.useApp();
  const http = useHttpClient();
  const nav = useModuleNavigator();

  // Backend PEP remains authoritative; the frontend does not hide knowledge actions by permission.
  const canUpdate = true;
  const canDelete = true;
  const canUse = true;
  const sourceStats = [
    volumeCount > 0 ? t('knowledge.base.card-volumes-count', { count: volumeCount }) : null,
    fileCount > 0 ? t('knowledge.base.card-files-count', { count: fileCount }) : null,
    tablesCount > 0 ? t('knowledge.base.card-tables-count', { count: tablesCount }) : null,
  ].filter((item): item is string => Boolean(item));

  const [metadataModalOpen, setMetadataModalOpen] = useState(false);
  const [metadataSubmitting, setMetadataSubmitting] = useState(false);
  const [metadataInitialValues, setMetadataInitialValues] = useState<KnowledgeMetadataValues>({
    name,
    description: remark,
  });

  const [openMetadataLoading, setOpenMetadataLoading] = useState(false);
  const openMetadataModal = async () => {
    setOpenMetadataLoading(true);
    try {
      const detail = await getKnowledgeDetail(http, { id });
      setMetadataInitialValues({
        name: detail.name,
        description: detail.description || '',
      });
      setMetadataModalOpen(true);
    } catch (error) {
      message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.load-detail-failed'));
      console.error('[KnowledgeCard] load detail for metadata edit failed', error);
    } finally {
      setOpenMetadataLoading(false);
    }
  };

  const handleEditAdvanced = (event: MouseEvent) => {
    event.stopPropagation();
    nav.goToPage('knowledge-edit', { params: { id: String(id) } });
  };

  const handleEditMetadata = (event: MouseEvent) => {
    event.stopPropagation();
    openMetadataModal();
  };

  const handleDelete = (event: MouseEvent) => {
    event.stopPropagation();

    Modal.confirm({
      title: t('knowledge.base.delete-modal-title'),
      content: t('knowledge.base.delete-modal-content', { name }),
      okText: t('knowledge.base.delete-modal-ok'),
      cancelText: t('knowledge.base.delete-modal-cancel'),
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteKnowledgeById(http, { id });
          message.success(t('knowledge.base.delete-success'));
          onRefresh?.();
        } catch (error) {
          message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.delete-failed'));
          console.error('[KnowledgeCard] delete failed', error);
        }
      },
    });
  };

  const handleDialog = (event: MouseEvent) => {
    event.stopPropagation();
    onDialogStarted?.(id);
  };

  const handleCardClick = () => {
    if (canUpdate) {
      nav.goToPage('knowledge-edit', { params: { id: String(id) } });
    }
  };

  const handleMetadataSubmit = async (values: KnowledgeMetadataValues, form: KnowledgeMetadataForm) => {
    setMetadataSubmitting(true);
    try {
      await updateKnowledge(http, {
        id,
        name: values.name,
        description: values.description,
      });
      message.success(t('knowledge.base.update-success'));
      setMetadataModalOpen(false);
      onRefresh?.();
    } catch (error) {
      const duplicateNameMessage = t('knowledge.base.name-already-exists');
      if (isApiConflictError(error)) {
        form.setFields([{ name: 'name', errors: [duplicateNameMessage] }]);
        message.error(duplicateNameMessage);
      } else {
        message.error(resolveKnowledgeErrorMessage(error, 'knowledge.base.update-failed'));
      }
      console.error('[KnowledgeCard] update metadata failed', error);
    } finally {
      setMetadataSubmitting(false);
    }
  };

  return (
    <>
      <Card
        hoverable
        className={`${styles.card} ${canUpdate ? styles.cardClickable : styles.cardReadonly}`}
        classNames={{ body: styles.cardBody }}
        onClick={handleCardClick}
      >
        <Space orientation="vertical" size="middle" className={styles.content}>
          <div className={styles.header}>
            <div className={styles.iconWrapper}>
              <div className={styles.folderIconBg}>
                <FolderOpenFilled className={styles.folderIcon} />
              </div>
            </div>

            <Space size={2}>
              {canUpdate && (
                <>
                  <ListActionButton
                    action="edit"
                    label={t('knowledge.base.card-edit-metadata')}
                    loading={openMetadataLoading}
                    onClick={handleEditMetadata}
                    data-testid={`knowledge-base-card-metadata-edit-${id}`}
                  />
                  <ListActionButton
                    action="configure"
                    label={t('knowledge.base.card-edit-advanced')}
                    onClick={handleEditAdvanced}
                    data-testid={`knowledge-base-card-advanced-config-${id}`}
                  />
                </>
              )}

              {canDelete && (
                <ListActionButton
                  action="delete"
                  danger
                  label={t('knowledge.base.card-delete')}
                  onClick={handleDelete}
                  data-testid={`knowledge-base-card-delete-${id}`}
                />
              )}
            </Space>
          </div>

          <Text strong ellipsis={{ tooltip: name }} className={styles.title}>
            {name}
          </Text>

          <Paragraph ellipsis={{ rows: 2, tooltip: remark }} className={styles.remark}>
            {remark}
          </Paragraph>
        </Space>

        <div className={styles.footer}>
          <Space size={12}>
            {sourceStats.map((stat) => (
              <Text key={stat} type="secondary" className={styles.statText}>
                {stat}
              </Text>
            ))}
          </Space>
          {canUse ? (
            <Button
              size="middle"
              icon={<CommentOutlined />}
              onClick={handleDialog}
              className={styles.dialogButton}
              data-testid={`knowledge-base-card-dialog-btn-${id}`}
            >
              {t('knowledge.base.card-dialog-button')}
            </Button>
          ) : null}
        </div>
      </Card>

      <KnowledgeMetadataEditModal
        open={metadataModalOpen}
        loading={metadataSubmitting}
        initialValues={metadataInitialValues}
        onCancel={() => setMetadataModalOpen(false)}
        onSubmit={handleMetadataSubmit}
      />
    </>
  );
}
