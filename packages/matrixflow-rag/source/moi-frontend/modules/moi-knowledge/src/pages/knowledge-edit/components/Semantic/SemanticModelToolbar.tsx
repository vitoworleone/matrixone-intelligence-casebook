import { useState } from 'react';
import { Alert, App, Button, Input, Modal, Space, Typography } from 'antd';
import { DownloadOutlined, SafetyCertificateOutlined, UploadOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';

import type { AppHttpClient } from '@moi/shared-moi-app-protocol/app-context';
import { extractApiErrorDetail, extractApiErrorMessage } from '../../../../service/http';
import { exportSemanticModel, importSemanticModel, validateSemanticModel } from '../../../../service/semanticModel';

interface SemanticModelToolbarProps {
  http: AppHttpClient;
  modelId: number;
  t: TFunction<'moi-knowledge'>;
  onImportSuccess: () => void;
}

interface ImportError {
  message: string;
  requestId?: string;
}

function isJSONObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function SemanticModelToolbar({ http, modelId, t, onImportSuccess }: SemanticModelToolbarProps) {
  const { message: appMessage } = App.useApp();
  const [importOpen, setImportOpen] = useState(false);
  const [importText, setImportText] = useState('');
  const [importing, setImporting] = useState(false);
  const [importError, setImportError] = useState<ImportError>();
  const [exporting, setExporting] = useState(false);
  const [validating, setValidating] = useState(false);

  const handleExport = async () => {
    setExporting(true);
    try {
      const data = await exportSemanticModel(http, modelId);
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `semantic-model-${modelId}.json`;
      a.click();
      URL.revokeObjectURL(url);
      appMessage.success(t('knowledge.entry.export-success'));
    } catch (error) {
      appMessage.error(extractApiErrorMessage(error, t('knowledge.entry.export-failed')));
    } finally {
      setExporting(false);
    }
  };

  const handleValidate = async () => {
    setValidating(true);
    try {
      const result = await validateSemanticModel(http, modelId);
      if (result.valid) {
        appMessage.success(t('knowledge.entry.validate-pass'));
      } else {
        Modal.warning({
          title: t('knowledge.entry.validate-fail', { count: result.errors?.length ?? 0 }),
          content: (
            <ul style={{ maxHeight: 300, overflow: 'auto', paddingLeft: 20 }}>
              {result.errors?.map((err) => (
                <li key={err}>{err}</li>
              ))}
            </ul>
          ),
          width: 520,
        });
      }
    } catch (error) {
      appMessage.error(extractApiErrorMessage(error, t('knowledge.entry.validate-failed')));
    } finally {
      setValidating(false);
    }
  };

  const handleImport = async () => {
    const trimmed = importText.trim();
    if (!trimmed) return;

    let parsed: unknown;
    try {
      parsed = JSON.parse(trimmed);
    } catch {
      setImportError({ message: t('knowledge.entry.import-invalid-json', { path: '$' }) });
      return;
    }

    if (!isJSONObject(parsed)) {
      setImportError({ message: t('knowledge.entry.import-invalid-shape', { path: '$' }) });
      return;
    }

    if (!Array.isArray(parsed.entries) || parsed.entries.length === 0) {
      setImportError({ message: t('knowledge.entry.import-invalid-entries', { path: 'entries' }) });
      return;
    }

    for (const [index, entry] of parsed.entries.entries()) {
      if (!isJSONObject(entry)) {
        setImportError({ message: t('knowledge.entry.import-invalid-shape', { path: `entries[${index}]` }) });
        return;
      }
      if (!isJSONObject(entry.spec)) {
        setImportError({ message: t('knowledge.entry.import-invalid-shape', { path: `entries[${index}].spec` }) });
        return;
      }
    }

    setImportError(undefined);
    setImporting(true);
    try {
      const result = await importSemanticModel(http, modelId, { entries: parsed.entries as never[] });
      appMessage.success(t('knowledge.entry.import-success', { count: result.imported }));
      setImportOpen(false);
      setImportText('');
      setImportError(undefined);
      onImportSuccess();
    } catch (error) {
      const detail = extractApiErrorDetail(error);
      const message =
        detail.status === 401 || detail.status === 403
          ? t('knowledge.entry.import-permission-denied')
          : !detail.status || detail.status >= 500
            ? t('knowledge.entry.import-service-unavailable')
            : detail.msg || t('knowledge.entry.import-failed');
      setImportError({ message, requestId: detail.requestId });
    } finally {
      setImporting(false);
    }
  };

  return (
    <>
      <Space>
        <Button
          data-testid="semantic-config-import-btn"
          icon={<UploadOutlined />}
          onClick={() => {
            setImportError(undefined);
            setImportOpen(true);
          }}
        >
          {t('knowledge.entry.toolbar-import')}
        </Button>
        <Button icon={<DownloadOutlined />} loading={exporting} onClick={handleExport}>
          {t('knowledge.entry.toolbar-export')}
        </Button>
        <Button icon={<SafetyCertificateOutlined />} loading={validating} onClick={handleValidate}>
          {t('knowledge.entry.toolbar-validate')}
        </Button>
      </Space>

      <Modal
        title={t('knowledge.entry.import-title')}
        open={importOpen}
        onOk={handleImport}
        onCancel={() => {
          setImportOpen(false);
          setImportText('');
          setImportError(undefined);
        }}
        confirmLoading={importing}
        width={600}
        destroyOnClose
      >
        <Alert type="warning" showIcon title={t('knowledge.entry.import-warning')} style={{ marginBottom: 16 }} />
        {importError ? (
          <Alert
            data-testid="semantic-config-import-error"
            type="error"
            showIcon
            title={importError.message}
            description={
              importError.requestId ? (
                <Typography.Text copyable={{ text: importError.requestId }}>
                  {t('knowledge.entry.import-request-id', { id: importError.requestId })}
                </Typography.Text>
              ) : undefined
            }
            style={{ marginBottom: 16 }}
          />
        ) : null}
        <Input.TextArea
          data-testid="semantic-config-import-input"
          rows={12}
          value={importText}
          onChange={(e) => {
            setImportError(undefined);
            setImportText(e.target.value);
          }}
          placeholder={t('knowledge.entry.import-upload-hint')}
        />
      </Modal>
    </>
  );
}
