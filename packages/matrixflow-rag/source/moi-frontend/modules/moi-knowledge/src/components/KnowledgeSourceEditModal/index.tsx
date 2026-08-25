import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Alert, App, Card, Form, Modal } from 'antd';

import { isKnowledgeBaseCatalogFileSelectable } from '../../shared/knowledge/catalog-source';
import { TreeOfFiles, type SelectedFileItem } from '../TreeOfFiles/TreeOfFiles';
import styles from './KnowledgeSourceEditModal.module.css';

interface SourceFormValues {
  files: SelectedFileItem[];
}

interface KnowledgeSourceEditModalProps {
  open: boolean;
  loading?: boolean;
  initialFiles: SelectedFileItem[];
  defaultExpandedKeys?: string[];
  onCancel: () => void;
  onSubmit: (files: SelectedFileItem[]) => Promise<void> | void;
}

export default function KnowledgeSourceEditModal({
  open,
  loading = false,
  initialFiles,
  defaultExpandedKeys = [],
  onCancel,
  onSubmit,
}: KnowledgeSourceEditModalProps) {
  const { t } = useTranslation('moi-knowledge');
  const { message } = App.useApp();
  const [form] = Form.useForm<SourceFormValues>();

  useEffect(() => {
    if (!open) return;
    form.setFieldsValue({ files: initialFiles });
  }, [form, initialFiles, open]);

  const handleOk = async () => {
    const values = await form.validateFields();
    const files = values.files ?? [];
    if (files.some((item) => item.type !== 'table' && !isKnowledgeBaseCatalogFileSelectable(item))) {
      message.error(t('knowledge.base.create-document-sources-catalog-data-unsupported'));
      return;
    }
    await onSubmit(files);
  };

  return (
    <Modal
      open={open}
      title={t('knowledge.base.edit-source-modal-title')}
      okText={t('knowledge.base.form-save-button')}
      cancelText={t('knowledge.base.form-cancel-button')}
      onCancel={onCancel}
      onOk={handleOk}
      confirmLoading={loading}
      width={860}
      destroyOnHidden
    >
      <Form form={form} layout="vertical" initialValues={{ files: initialFiles }} preserve={false}>
        <Form.Item label={t('knowledge.base.form-source-label')} required>
          <Alert title={t('knowledge.base.form-source-alert-title')} type="warning" showIcon className={styles.sourceAlert} />
          <Card styles={{ body: { padding: 16 } }} type="inner" title={t('knowledge.base.form-source-card-title')}>
            <Form.Item name="files" rules={[{ required: true, message: t('knowledge.base.form-source-required') }]} noStyle>
              <TreeOfFiles
                fileSelectMode="all-returned-files"
                fetchFilesDataFiltersParam={[]}
                allowSelectTable
                allowSelectVolume
                defaultExpandedKeys={defaultExpandedKeys}
                isFileSelectable={isKnowledgeBaseCatalogFileSelectable}
              />
            </Form.Item>
          </Card>
        </Form.Item>
      </Form>
    </Modal>
  );
}
