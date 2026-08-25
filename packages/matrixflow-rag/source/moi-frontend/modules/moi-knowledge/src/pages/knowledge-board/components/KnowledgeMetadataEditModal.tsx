import { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Form, Input, Modal, Typography, type FormInstance } from 'antd';

import styles from './KnowledgeMetadataEditModal.module.css';

const { Text } = Typography;

export const KNOWLEDGE_NAME_MAX_LENGTH = 255;

export interface KnowledgeMetadataValues {
  name: string;
  description: string;
}

export type KnowledgeMetadataForm = FormInstance<KnowledgeMetadataValues>;

interface KnowledgeMetadataEditModalProps {
  open: boolean;
  loading?: boolean;
  title?: string;
  okText?: string;
  cancelText?: string;
  initialValues: KnowledgeMetadataValues;
  onCancel: () => void;
  onSubmit: (values: KnowledgeMetadataValues, form: KnowledgeMetadataForm) => Promise<void> | void;
}

export default function KnowledgeMetadataEditModal({
  open,
  loading = false,
  title,
  okText,
  cancelText,
  initialValues,
  onCancel,
  onSubmit,
}: KnowledgeMetadataEditModalProps) {
  const { t } = useTranslation('moi-knowledge');
  const [form] = Form.useForm<KnowledgeMetadataValues>();

  useEffect(() => {
    if (!open) return;
    form.setFieldsValue(initialValues);
  }, [form, initialValues, open]);

  const handleOk = async () => {
    const values = await form.validateFields();
    await onSubmit(values, form);
  };

  return (
    <Modal
      open={open}
      width={620}
      title={title || t('knowledge.base.edit-metadata-modal-title')}
      okText={okText || t('knowledge.base.form-save-button')}
      cancelText={cancelText || t('knowledge.base.form-cancel-button')}
      onCancel={onCancel}
      onOk={handleOk}
      confirmLoading={loading}
      styles={{
        body: { paddingBottom: 20 },
        footer: {
          borderTop: '1px solid var(--moi-colorBorderSecondary, #f0f0f0)',
          marginTop: 20,
          paddingTop: 14,
        },
      }}
      destroyOnHidden
    >
      <Form form={form} layout="vertical" initialValues={initialValues} preserve={false}>
        <Form.Item name="name" label={t('knowledge.base.form-name-label')} extra={t('knowledge.base.form-name-immutable')}>
          <Input
            maxLength={KNOWLEDGE_NAME_MAX_LENGTH}
            placeholder={t('knowledge.base.form-name-placeholder')}
            showCount
            disabled
          />
        </Form.Item>

        <Form.Item
          name="description"
          label={
            <div className={styles.remarkLabel}>
              <Text className={styles.remarkTitle}>{t('knowledge.base.form-remark-label')}</Text>
              <Text type="secondary" className={styles.remarkDesc}>
                {t('knowledge.base.form-remark-desc')}
              </Text>
            </div>
          }
          rules={[{ required: true, message: t('knowledge.base.form-remark-required') }]}
        >
          <Input.TextArea rows={4} maxLength={10000} showCount placeholder={t('knowledge.base.form-remark-placeholder')} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
