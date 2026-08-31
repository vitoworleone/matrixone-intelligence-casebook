    let currentDataType = 'unstructured';
    const defaultFileTypes = ['TXT', 'PDF', 'PPT', 'DOC', 'DOCX', 'Markdown'];
    let activeFileTypes = new Set(defaultFileTypes);

    // === Edit Mode ===
    var editMode = false;
    var editTaskId = null;
    var editTaskData = null;

    // Mock task data for edit mode (mirrors importTasks from data-import.html)
    // 配置依据：customers/示例制造数据项目/demo_mfg_lakehouse_v2.docx 4.8 节增量策略表
    //   doc 明确规则：
    //   - 示例工业物联 MongoDB:    timestamp range, daily 拉取，1h overlap
    //   - Fiix work_orders:   dtmDateLastModified > last_watermark
    //   - Fiix wo_tasks:      Parent WO modified date filter（mock 简化为 dtmDateLastModified）
    //   - Fiix lookup tables: Full refresh (TRUNCATE + INSERT)
    //   - Fiix assets:        intUpdated > last_watermark
    //   - Fiix meter_readings:dtmDateSubmitted > last_watermark
    //   doc 未列出的（sched_maint / parts / po）按 Fiix 通用变更字段 dtmDateLastModified
    var editableTasksData = window.IMPORT_TASKS_EDIT || {};

    function initEditMode() {
      var params = new URLSearchParams(window.location.search);
      editTaskId = params.get('id');
      if (!editTaskId) return;

      editTaskData = editableTasksData[editTaskId];
      if (!editTaskData) return;

      editMode = true;

      // Change page title
      document.title = 'MOI - 编辑载入任务';

      // Change back header text
      var backLink = document.querySelector('.page-back a');
      if (backLink) {
        backLink.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6"/></svg>编辑载入任务 - ' + editTaskData.name;
      }

      // Change bottom bar button
      var submitBtn = document.querySelector('.bottom-bar .btn-primary');
      if (submitBtn) {
        submitBtn.textContent = '保存配置';
        submitBtn.setAttribute('onclick', 'saveImportEdit()');
      }

      // Hide file count in bottom bar
      var fileCount = document.getElementById('fileCount');
      if (fileCount) fileCount.style.display = 'none';

      // Select data type (and disable switching)
      selectDataType(editTaskData.dataType);
      // Disable type cards
      var cards = document.querySelectorAll('.type-card');
      cards.forEach(function(card) {
        card.style.pointerEvents = 'none';
        card.style.opacity = card.classList.contains('active') ? '1' : '0.4';
      });

      if (editTaskData.dataType === 'unstructured') {
        fillUnstructuredEditForm();
      } else {
        fillStructuredEditForm();
      }
    }

    function fillUnstructuredEditForm() {
      var d = editTaskData;

      // Source type (connector/local/web) — select and disable switching
      var tabContainer = document.querySelector('#formUnstructured .us-row .inline-tabs');
      if (tabContainer) {
        var tabs = tabContainer.querySelectorAll('.inline-tab');
        var sourceMap = { connector: 'connector', local: 'local', web: 'web' };
        var sourceId = sourceMap[d.sourceType] || 'connector';
        tabs.forEach(function(tab) {
          var onclickStr = tab.getAttribute('onclick') || '';
          if (onclickStr.indexOf("'" + sourceId + "'") !== -1) {
            // Simulate click by calling selectSourceTab directly
            selectSourceTab(tab, sourceId);
          }
        });
        // Disable source tabs
        tabs.forEach(function(tab) {
          tab.style.pointerEvents = 'none';
          tab.style.opacity = tab.classList.contains('active') ? '1' : '0.5';
        });
      }

      // Connector — select and disable（优先用 connectorValue 精确匹配，再退回 connector 名称匹配）
      if (d.sourceType === 'connector' && (d.connector || d.connectorValue)) {
        var connSel = document.getElementById('usConnectorSelect');
        if (connSel) {
          var matched = false;
          if (d.connectorValue) {
            for (var i = 0; i < connSel.options.length; i++) {
              if (connSel.options[i].value === d.connectorValue) { connSel.selectedIndex = i; matched = true; break; }
            }
          }
          if (!matched && d.connector) {
            for (var j = 0; j < connSel.options.length; j++) {
              if (connSel.options[j].text === d.connector) { connSel.selectedIndex = j; break; }
            }
          }
          connSel.disabled = true;
          connSel.style.opacity = '0.65';
          connSel.style.cursor = 'not-allowed';
          onUsConnectorChange();
        }
      }

      // Catalog target — show as filled and disable
      if (d.target) {
        var triggerText = document.getElementById('usCatalogTriggerText');
        if (triggerText) {
          var parts = [];
          if (d.target.dir) parts.push('⊙' + d.target.dir);
          if (d.target.db) parts.push('⊙' + d.target.db);
          if (d.target.vol)    parts.push(d.target.vol);
          if (d.target.volume) parts.push(d.target.volume);
          triggerText.textContent = parts.join(' / ');
          triggerText.className = 'trigger-value';
        }
        var trigger = document.getElementById('usCatalogTrigger');
        if (trigger) {
          trigger.style.pointerEvents = 'none';
          trigger.style.opacity = '0.65';
          trigger.style.cursor = 'not-allowed';
        }
        // Also disable web catalog trigger if web mode
        var webTrigger = document.getElementById('usWebCatalogGroup');
        if (webTrigger) {
          var webCatText = document.getElementById('usWebCatalogText');
          if (webCatText && d.target.vol) {
            var wParts = [];
            if (d.target.dir) wParts.push('⊙' + d.target.dir);
            if (d.target.db) wParts.push('⊙' + d.target.db);
            if (d.target.vol) wParts.push(d.target.vol);
            webCatText.textContent = wParts.join(' / ');
            webCatText.style.color = 'rgba(0,0,0,0.88)';
          }
          webTrigger.style.pointerEvents = 'none';
          webTrigger.style.opacity = '0.65';
        }
      }

      // Load mode — editable
      if (d.loadMode === 'periodic') {
        var periodicRadio = document.querySelector('input[name="loadMode"][value="periodic"]');
        if (periodicRadio) {
          periodicRadio.checked = true;
          onLoadModeChange();
        }
        if (d.periodicInterval) {
          var intSel = document.getElementById('periodicInterval');
          if (intSel) {
            intSel.value = d.periodicInterval;
            onPeriodicIntervalChange();
          }
        }
        if (d.periodicTime) {
          var timeSel = document.getElementById('periodicTime');
          if (timeSel) {
            for (var j = 0; j < timeSel.options.length; j++) {
              if (timeSel.options[j].text === d.periodicTime) { timeSel.selectedIndex = j; break; }
            }
          }
        }
      }

      // Duplicate file handling — editable
      if (d.dupHandle === 'overwrite') {
        var owRadio = document.querySelector('input[name="dupHandle"][value="overwrite"]');
        if (owRadio) owRadio.checked = true;
      }

      // Unzip strategy — editable
      if (d.unzipStrategy === 'keep') {
        var keepRadio = document.querySelector('input[name="unzip"][value="keep"]');
        if (keepRadio) keepRadio.checked = true;
      }
    }

    function fillStructuredEditForm() {
      var d = editTaskData;

      // Source type (connector/local) — disable switching
      var stFormTabs = document.querySelectorAll('#formStructured > .form-card .inline-tabs .inline-tab');
      if (d.sourceType === 'local' && stFormTabs[1]) {
        selectStructuredSourceTab(stFormTabs[1], 'stLocal');
      }
      stFormTabs.forEach(function(tab) {
        tab.style.pointerEvents = 'none';
        tab.style.opacity = tab.classList.contains('active') ? '1' : '0.5';
      });

      // Connector — select, trigger change, then disable
      if (d.sourceType === 'connector' && d.connectorValue) {
        var connSel = document.getElementById('stConnectorSelect');
        if (connSel) {
          connSel.value = d.connectorValue;
          onStructuredConnectorChange();

          // After connector change, fill DB/table/API in sequence
          setTimeout(function() {
            // Disable connector
            connSel.disabled = true;
            connSel.style.opacity = '0.65';
            connSel.style.cursor = 'not-allowed';

            // DB name
            if (d.dbName) {
              var dbSel = document.getElementById('stDbNameSelect');
              if (dbSel) {
                dbSel.value = d.dbName;
                onStDbNameChange();
                dbSel.disabled = true;
                dbSel.style.opacity = '0.65';
                dbSel.style.cursor = 'not-allowed';
              }
            }

            // API endpoint
            if (d.apiEndpoint) {
              var apiSel = document.getElementById('stApiEndpoint');
              if (apiSel) {
                apiSel.value = d.apiEndpoint;
                if (typeof onApiEndpointChange === 'function') onApiEndpointChange();
                apiSel.disabled = true;
                apiSel.style.opacity = '0.65';
                apiSel.style.cursor = 'not-allowed';
              }
              // Also disable method select
              var methodSel = document.getElementById('stApiMethod');
              if (methodSel) {
                methodSel.disabled = true;
                methodSel.style.opacity = '0.65';
              }
            }

            // DB table (needs another tick for options to populate)
            setTimeout(function() {
              if (d.dbTable) {
                var tblSel = document.getElementById('stDbTableSelect');
                if (tblSel) {
                  tblSel.value = d.dbTable;
                  onStDbTableChange();
                  tblSel.disabled = true;
                  tblSel.style.opacity = '0.65';
                  tblSel.style.cursor = 'not-allowed';
                }
              }

              // Catalog target — fill and disable
              if (d.target) {
                catalogConfirmedDir = d.target.dir || null;
                catalogConfirmedDb = d.target.db || null;
                catalogConfirmedTable = d.target.table || null;
                catalogTempDir = catalogConfirmedDir;
                catalogTempDb = catalogConfirmedDb;
                catalogTempTable = catalogConfirmedTable;

                var trigger = document.getElementById('catalogTriggerText');
                if (trigger) {
                  var parts = [];
                  if (catalogConfirmedDir) parts.push('⊙' + catalogConfirmedDir);
                  if (catalogConfirmedDb) parts.push('⊙' + catalogConfirmedDb);
                  if (catalogConfirmedTable) parts.push(catalogConfirmedTable);
                  trigger.textContent = parts.join(' / ');
                  trigger.className = 'trigger-value';
                }
                var triggerEl = document.getElementById('catalogTrigger');
                if (triggerEl) {
                  triggerEl.style.pointerEvents = 'none';
                  triggerEl.style.opacity = '0.65';
                  triggerEl.style.cursor = 'not-allowed';
                }

                // Target mode — disable
                if (d.targetMode) {
                  var modeRadio = document.querySelector('input[name="targetMode"][value="' + d.targetMode + '"]');
                  if (modeRadio) modeRadio.checked = true;
                  document.querySelectorAll('input[name="targetMode"]').forEach(function(r) {
                    r.disabled = true;
                    r.parentElement.style.opacity = '0.65';
                    r.parentElement.style.cursor = 'not-allowed';
                  });
                }
              }

              // Load mode — editable
              if (d.stLoadMode) {
                var modeRadio = document.querySelector('input[name="stLoadMode"][value="' + d.stLoadMode + '"]');
                if (modeRadio) {
                  modeRadio.checked = true;
                  onStLoadModeChange();
                }
                if (d.stPeriodicInterval) {
                  var intSel = document.getElementById('stPeriodicInterval');
                  if (intSel) intSel.value = d.stPeriodicInterval;
                }
              }

              // Preprocess — editable
              if (d.preprocess) {
                var ppToggle = document.getElementById('preprocessToggle');
                if (ppToggle) { ppToggle.checked = true; onPreprocessToggle(); }
              }

              // Backfill — editable
              if (d.backfill) {
                var bfToggle = document.getElementById('stBackfillToggle');
                if (bfToggle) { bfToggle.checked = true; onBackfillToggle(); }
              }

              // Trigger bottom section display
              checkShowBottomSection();

              // Sync strategy + Incremental config — needs another tick after load mode change
              setTimeout(function() {
                // 同步策略：full 全量覆盖 / incremental 增量
                var strategy = d.syncStrategy || (d.incremental ? 'incremental' : 'incremental');
                var stratEl = document.querySelector('input[name="stSyncStrategy"][value="' + strategy + '"]');
                if (stratEl) { stratEl.checked = true; if (typeof onStSyncStrategyChange === 'function') onStSyncStrategyChange(); }
                if (d.incremental && strategy === 'incremental') {
                  var incrField = document.getElementById('stIncrField');
                  if (incrField) incrField.value = d.incremental.field;
                  var lookback = document.getElementById('stLookbackWindow');
                  if (lookback) lookback.value = d.incremental.lookback;
                }
              }, 50);
            }, 50);
          }, 50);
        }
      } else if (d.sourceType === 'local') {
        // Local upload mode — fill catalog target
        if (d.target) {
          catalogConfirmedDir = d.target.dir || null;
          catalogConfirmedDb = d.target.db || null;
          catalogConfirmedTable = d.target.table || null;
          catalogTempDir = catalogConfirmedDir;
          catalogTempDb = catalogConfirmedDb;
          catalogTempTable = catalogConfirmedTable;

          var trigger = document.getElementById('catalogTriggerText');
          if (trigger) {
            var parts = [];
            if (catalogConfirmedDir) parts.push('⊙' + catalogConfirmedDir);
            if (catalogConfirmedDb) parts.push('⊙' + catalogConfirmedDb);
            if (catalogConfirmedTable) parts.push(catalogConfirmedTable);
            trigger.textContent = parts.join(' / ');
            trigger.className = 'trigger-value';
          }
          var triggerEl = document.getElementById('catalogTrigger');
          if (triggerEl) {
            triggerEl.style.pointerEvents = 'none';
            triggerEl.style.opacity = '0.65';
            triggerEl.style.cursor = 'not-allowed';
          }
          if (d.targetMode) {
            var modeRadio = document.querySelector('input[name="targetMode"][value="' + d.targetMode + '"]');
            if (modeRadio) modeRadio.checked = true;
            document.querySelectorAll('input[name="targetMode"]').forEach(function(r) {
              r.disabled = true;
              r.parentElement.style.opacity = '0.65';
              r.parentElement.style.cursor = 'not-allowed';
            });
          }
        }
      }
    }

    function saveImportEdit() {
      // Check if incremental field changed — warn about watermark reset
      if (editTaskData && editTaskData.incremental) {
        var incrField = document.getElementById('stIncrField');
        if (incrField && incrField.value && incrField.value !== editTaskData.incremental.field) {
          var confirmed = confirm('增量字段已从「' + editTaskData.incremental.field + '」变更为「' + incrField.value + '」。\n\n水位线将重置，下次载入会重新拉取数据。确认保存？');
          if (!confirmed) return;
        }
      }
      alert('载入任务配置已保存（模拟）\n\n任务：' + editTaskData.name + '\n变更将在下次调度时生效。');
      location.href = 'data-import.html';
    }

    document.addEventListener('DOMContentLoaded', function() {
      loadSavedUnstructuredConnectors();
      renderFileTypeTags();
      initEditMode();
    });

    function selectDataType(type) {
      currentDataType = type;
      document.getElementById('cardUnstructured').classList.toggle('active', type === 'unstructured');
      document.getElementById('cardStructured').classList.toggle('active', type === 'structured');
      document.getElementById('formUnstructured').style.display = type === 'unstructured' ? 'block' : 'none';
      document.getElementById('formStructured').style.display = type === 'structured' ? 'block' : 'none';
    }

    function selectSourceTab(el, id) {
      el.parentElement.querySelectorAll('.inline-tab').forEach(t => t.classList.remove('active'));
      el.classList.add('active');
      const connCatalogGroup = document.getElementById('usConnCatalogGroup');
      const localRow = document.getElementById('usLocalUploadRow');
      const fileBrowser = document.getElementById('usFileBrowserCard');
      const webPanel = document.getElementById('usWebImportPanel');
      const unzipField = document.getElementById('usUnzipField');
      const dupRow = document.getElementById('usDupRow');
      const rangeRow = document.getElementById('usRangeRow');
      const webIncrField = document.getElementById('usWebIncrField');
      if (connCatalogGroup) connCatalogGroup.style.display = id === 'connector' ? 'flex' : 'none';
      if (localRow) localRow.style.display = id === 'local' ? '' : 'none';
      // Web catalog group
      const webCatalogGroup = document.getElementById('usWebCatalogGroup');
      if (webCatalogGroup) webCatalogGroup.style.display = id === 'web' ? 'flex' : 'none';
      // File browser: only show in connector mode when connector is selected
      if (fileBrowser) {
        if (id === 'connector') {
          var sel = document.getElementById('usConnectorSelect');
          fileBrowser.style.display = (sel && sel.value) ? '' : 'none';
        } else {
          fileBrowser.style.display = 'none';
        }
      }
      if (webPanel) {
        if (id === 'web') {
          webPanel.style.display = '';
          // Only show web config card if catalog is already selected
          var webConfigCard = document.getElementById('webConfigCard');
          var webCatText = document.getElementById('usWebCatalogText');
          var hasCatalog = webCatText && webCatText.style.color === 'rgba(0,0,0,0.88)';
          if (webConfigCard) webConfigCard.style.display = hasCatalog ? '' : 'none';
        } else {
          webPanel.style.display = 'none';
        }
      }
      // Hide 解压策略/重复文件/载入范围 for web mode, show 增量策略
      if (unzipField) unzipField.style.display = id === 'web' ? 'none' : '';
      if (dupRow) dupRow.style.display = id === 'web' ? 'none' : '';
      if (rangeRow) rangeRow.style.display = id === 'web' ? 'none' : '';
      if (webIncrField) webIncrField.style.display = id === 'web' ? '' : 'none';
      updateUsRangeControls(id);
    }

    function getActiveUnstructuredSourceTab() {
      var active = document.querySelector('#formUnstructured .inline-tab.active');
      var onclick = active ? (active.getAttribute('onclick') || '') : '';
      var match = onclick.match(/'([^']+)'/);
      return match ? match[1] : 'connector';
    }

    function updateUsRangeControls(sourceId) {
      var activeSource = sourceId || getActiveUnstructuredSourceTab();
      var fileControls = document.getElementById('usFileRangeControls');
      var mailControls = document.getElementById('usMailRangeControls');
      var mailRange = document.getElementById('usMailCustomRange');
      var isMailRange = activeSource === 'connector' && usIsMailConnector();
      if (fileControls) fileControls.style.display = isMailRange ? 'none' : 'flex';
      if (mailControls) mailControls.style.display = isMailRange ? 'flex' : 'none';
      if (mailRange) mailRange.style.display = document.getElementById('usMailTimeRange')?.value === 'custom' ? 'flex' : 'none';
    }

    function onUsMailRangeChange() {
      var range = document.getElementById('usMailCustomRange');
      var value = document.getElementById('usMailTimeRange')?.value;
      if (range) range.style.display = value === 'custom' ? 'flex' : 'none';
      if (usIsRemoteMailConnector()) {
        usCurrentPath = '/';
        usSelectedItems.clear();
        usLoadRemoteMailPath('/');
      }
    }

    function onUsConnectorChange() {
      var sel = document.getElementById('usConnectorSelect');
      var fb = document.getElementById('usFileBrowserCard');
      if (fb) fb.style.display = (sel && sel.value) ? '' : 'none';
      updateUsRangeControls('connector');
      usResetBrowserForConnector();
    }

    function onLoadModeChange() {
      var isPeriodic = document.querySelector('input[name="loadMode"][value="periodic"]').checked;
      document.getElementById('periodicOptions').style.display = isPeriodic ? '' : 'none';
    }

    function onPeriodicIntervalChange() {
      var val = document.getElementById('periodicInterval').value;
      var isDayOrMore = val === '1d' || val === '7d';
      document.getElementById('periodicTime').style.display = isDayOrMore ? '' : 'none';
    }

    // === Structured form: source tab ===
    function selectStructuredSourceTab(el, id) {
      el.parentElement.querySelectorAll('.inline-tab').forEach(t => t.classList.remove('active'));
      el.classList.add('active');
      document.getElementById('stConn').style.display = id === 'stConn' ? 'block' : 'none';
      document.getElementById('stLocal').style.display = id === 'stLocal' ? 'block' : 'none';
      // Hide DB load mode when switching to local upload
      if (id === 'stLocal') {
        document.getElementById('stDbLoadMode').style.display = 'none';
        document.getElementById('fileOnlyTableSettings').style.display = 'block';
      } else {
        // Re-check if current connector is DB type
        const sel = document.getElementById('stConnectorSelect');
        const opt = sel.options[sel.selectedIndex];
        const isDb = opt && opt.getAttribute('data-type') === 'db';
        document.getElementById('stDbLoadMode').style.display = isDb ? 'block' : 'none';
        document.getElementById('fileOnlyTableSettings').style.display = isDb ? 'none' : 'block';
      }
      checkShowBottomSection();
    }

    // === Structured form: connector type detection ===
    const stMockDatabases = {
      'matrixone':  ['analytics_db', 'warehouse', 'user_center', 'moi_warehouse'],
      'mysql':      ['crm_production', 'erp_data', 'bi_reports', 'hr_system', 'user_center'],
      'mongodb':    ['iot_demo_prod', 'iot_demo_staging', 'shop_prod'],
      'hive':       ['demo_mfg_dw_ods', 'demo_mfg_dw_dwd', 'demo_mfg_dw_ads'],
      'postgresql': ['finance_dw', 'analytics_pg', 'crm_pg'],
      'sqlserver':  ['示例业务协作Quote', 'analytics_mssql', 'erp_mssql'],
      'oracle':     ['ERPPDB1', 'ods_oracle', 'erp_dw'],
      'clickhouse': ['hkex_market', 'rt_olap', 'logs_ck'],
      'doris':      ['rt_analytics', 'realtime_metrics', 'rt_dwd']
    };
    const stMockTables = {
      'analytics_db': [
        { name: 'daily_sales', rows: '125K' }, { name: 'monthly_revenue', rows: '3.6K' },
        { name: 'user_behavior', rows: '2.1M' }, { name: 'product_metrics', rows: '45K' }
      ],
      'warehouse': [
        { name: 'dim_product', rows: '8.2K' }, { name: 'dim_customer', rows: '52K' },
        { name: 'fact_orders', rows: '1.8M' }, { name: 'fact_payments', rows: '960K' }
      ],
      'crm_production': [
        { name: 'customers', rows: '89K' }, { name: 'contacts', rows: '156K' },
        { name: 'opportunities', rows: '23K' }, { name: 'activities', rows: '412K' },
        { name: 'deals', rows: '18K' }
      ],
      'erp_data': [
        { name: 'inventory', rows: '67K' }, { name: 'purchase_orders', rows: '34K' },
        { name: 'suppliers', rows: '2.1K' }, { name: 'warehouses', rows: '48' }
      ],
      'demo_mfg_dw_ods': [
        { name: 'ods_well_production_daily',  rows: '4.2M' },
        { name: 'ods_pump_failure_history',   rows: '186K' },
        { name: 'ods_drilling_logs',          rows: '12.8M' },
        { name: 'ods_field_inspection_notes', rows: '95K' }
      ],
      'demo_mfg_dw_dwd': [
        { name: 'dwd_well_kpi_hourly',        rows: '8.4M' },
        { name: 'dwd_equipment_event',        rows: '320K' }
      ],
      'demo_mfg_dw_ads': [
        { name: 'ads_field_oee_monthly',      rows: '1.2K' },
        { name: 'ads_failure_pareto_quarter', rows: '480' }
      ],
      'iot_demo_prod': [
        { name: 'sensor_readings', rows: '~90M/月' }, { name: 'pump_metadata', rows: '2.1K' },
        { name: 'crew_config', rows: '48' }, { name: 'alert_events', rows: '156K' }
      ],
      'iot_demo_staging': [
        { name: 'sensor_readings_test', rows: '1.2M' }, { name: 'pump_metadata', rows: '2.1K' }
      ],
      'shop_prod': [
        { name: 'products', rows: '156K' }, { name: 'orders', rows: '4.2M' }, { name: 'customers', rows: '320K' }
      ],
      'moi_warehouse': [
        { name: 'user_events', rows: '8.4M' }, { name: 'sessions', rows: '1.8M' }, { name: 'metrics_daily', rows: '180K' }
      ],
      'finance_dw': [
        { name: 'daily_financial_summary', rows: '21.6K' }, { name: 'gl_journal', rows: '480K' }, { name: 'ar_aging', rows: '12K' }
      ],
      'analytics_pg':[
        { name: 'event_log', rows: '2.4M' }, { name: 'user_attrs', rows: '186K' }
      ],
      'crm_pg':      [
        { name: 'leads', rows: '48K' }, { name: 'accounts', rows: '12K' }, { name: 'activities', rows: '320K' }
      ],
      '示例业务协作Quote':[
        { name: 'QUOTE_HDR', rows: '38.2K' }, { name: 'QUOTE_LINE', rows: '186K' }, { name: 'CONTACTS', rows: '8.6K' }, { name: 'USERLIST', rows: '120' }
      ],
      'analytics_mssql':[
        { name: 'fact_sales', rows: '2.8M' }, { name: 'dim_product', rows: '24K' }
      ],
      'erp_mssql':   [
        { name: 'INVENTORY', rows: '48K' }, { name: 'PO_HDR', rows: '12K' }
      ],
      'ERPPDB1':     [
        { name: 'GL_JE_LINES', rows: '480K' }, { name: 'GL_JE_HEADERS', rows: '120K' }, { name: 'AP_INVOICES_ALL', rows: '38K' }
      ],
      'ods_oracle':  [
        { name: 'ods_customer', rows: '320K' }, { name: 'ods_orders', rows: '2.4M' }
      ],
      'erp_dw':      [
        { name: 'dwd_revenue', rows: '180K' }, { name: 'dwd_cost', rows: '186K' }
      ],
      'hkex_market': [
        { name: 'tick', rows: '1.26B' }, { name: 'daily_close', rows: '1.8M' }, { name: 'minute_bar', rows: '420M' }
      ],
      'rt_olap':     [
        { name: 'user_event_5min', rows: '86M' }, { name: 'product_view_5min', rows: '24M' }
      ],
      'logs_ck':     [
        { name: 'nginx_access', rows: '4.2B' }, { name: 'app_error', rows: '12M' }
      ],
      'rt_analytics':[
        { name: 'minute_metrics', rows: '180K' }, { name: 'session_active', rows: '32K' }
      ],
      'realtime_metrics':[
        { name: 'kpi_realtime', rows: '8.6K' }
      ],
      'rt_dwd':      [
        { name: 'rt_order_flow', rows: '186K' }, { name: 'rt_user_action', rows: '2.4M' }
      ]
    };
    function onStructuredConnectorChange() {
      const sel = document.getElementById('stConnectorSelect');
      const opt = sel.options[sel.selectedIndex];
      const connType = opt.getAttribute('data-type');
      const val = sel.value;

      if (!val) {
        document.getElementById('stConnFile').style.display = 'block';
        document.getElementById('stConnDb').style.display = 'none';
        document.getElementById('stDbLoadMode').style.display = 'none';
        document.getElementById('hivePartitionCard').style.display = 'none';
        document.getElementById('preprocessCard').style.display = 'none';
        document.getElementById('fileOnlyTableSettings').style.display = 'block';
        return;
      }

      if (connType === 'db') {
        // Database connector selected
        document.getElementById('stConnFile').style.display = 'none';
        document.getElementById('stConnDb').style.display = 'block';
        document.getElementById('stConnApi') && (document.getElementById('stConnApi').style.display = 'none');
        document.getElementById('stDbLoadMode').style.display = 'block';
        document.getElementById('fileOnlyTableSettings').style.display = 'none';
        document.getElementById('hivePartitionCard').style.display = val === 'hive' ? '' : 'none';
        document.getElementById('preprocessCard').style.display = '';
        // MongoDB-specific labels
        var dbLabel = document.querySelector('#stConnDb .form-label') || document.querySelector('[for="stDbNameSelect"]');
        var tableLabel = document.getElementById('stDbTableLabel');
        if (val === 'mongodb') {
          if (tableLabel) tableLabel.innerHTML = '<span class="required">*</span> Collection：';
        } else {
          if (tableLabel) tableLabel.innerHTML = '<span class="required">*</span> 选择表：';
        }
        // Populate databases
        const dbs = stMockDatabases[val] || [];
        const dbSelect = document.getElementById('stDbNameSelect');
        dbSelect.innerHTML = '<option value="">请选择数据库</option>' + dbs.map(d => `<option value="${d}">${d}</option>`).join('');
        document.getElementById('stDbTableSection').style.display = 'none';
      } else if (connType === 'api' || connType === 'mail' || connType === 'mq' || connType === 'app' || connType === 'collab') {
        // API-style connector：含 REST/GraphQL API、邮件、消息队列、企业应用 OData、协作平台 API
        // 统一用 stConnApi 表单（endpoint 字段对应：API path / 邮箱过滤 / topic / OData entity / repo 等）
        document.getElementById('stConnFile').style.display = 'none';
        document.getElementById('stConnDb').style.display = 'none';
        document.getElementById('stConnApi').style.display = 'block';
        document.getElementById('stDbLoadMode').style.display = 'block';
        document.getElementById('fileOnlyTableSettings').style.display = 'none';
        document.getElementById('hivePartitionCard').style.display = 'none';
        document.getElementById('preprocessCard').style.display = '';
        // 根据类型调整 endpoint 标签语义
        var endpointLabel = document.getElementById('stApiEndpointLabel');
        if (endpointLabel) {
          var labels = { api: 'API Endpoint', mail: '邮箱 / 文件夹', mq: 'Topic / Queue', app: 'OData Entity / API', collab: 'Repo / Channel / Namespace' };
          endpointLabel.innerHTML = '<span class="required">*</span> ' + (labels[connType] || 'Endpoint') + '：';
        }
        showApiEndpointPreview();
      } else {
        // File connector selected
        document.getElementById('stConnFile').style.display = 'block';
        document.getElementById('stConnDb').style.display = 'none';
        document.getElementById('stConnApi') && (document.getElementById('stConnApi').style.display = 'none');
        document.getElementById('stDbLoadMode').style.display = 'none';
        document.getElementById('hivePartitionCard').style.display = 'none';
        document.getElementById('preprocessCard').style.display = 'none';
        document.getElementById('fileOnlyTableSettings').style.display = 'block';
      }
      checkShowBottomSection();
    }

    function onStDbNameChange() {
      const val = document.getElementById('stDbNameSelect').value;
      if (!val) {
        document.getElementById('stDbTableSection').style.display = 'none';
        checkShowBottomSection();
        return;
      }
      document.getElementById('stDbTableSection').style.display = 'block';
      const tables = stMockTables[val] || [];
      const sel = document.getElementById('stDbTableSelect');
      // Check if MongoDB connector
      const connSel = document.getElementById('stConnectorSelect');
      const isMongo = connSel.value === 'mongodb';
      const itemLabel = isMongo ? 'collection' : '表';
      const rowLabel = isMongo ? '文档' : '行';
      sel.innerHTML = '<option value="">请选择' + itemLabel + '</option>' + tables.map(t =>
        `<option value="${t.name}">${t.name}（${t.rows} ${rowLabel}）</option>`
      ).join('');
      checkShowBottomSection();
    }

    function onStDbTableChange() {
      checkShowBottomSection();
      // Re-render schema table when DB table/collection changes
      if (document.getElementById('bottomFormCard') && document.getElementById('bottomFormCard').style.display !== 'none') {
        renderColSchemaTable();
      }
    }

    // File type tags
    function renderFileTypeTags() {
      const container = document.getElementById('fileTypeTags');
      container.innerHTML = Array.from(activeFileTypes).map(t =>
        `<span class="checkbox-tag active" onclick="removeFileTypeTag('${t}')">${t} <span class="tag-close">✕</span></span>`
      ).join('');
      // Update the "+ 更多" count
      const remaining = activeFileTypes.size > defaultFileTypes.length ? '' : ` + ${activeFileTypes.size - defaultFileTypes.length}`;
    }

    function removeFileTypeTag(type) {
      activeFileTypes.delete(type);
      renderFileTypeTags();
    }

    function addFileTypeTag() {
      const sel = document.getElementById('addFileType');
      if (sel.value && !activeFileTypes.has(sel.value)) {
        activeFileTypes.add(sel.value);
        renderFileTypeTags();
      }
      sel.value = '';
    }

    // === Catalog modal ===
    const catalogData = {
      '默认': {
        '原始数据库': [],
        '处理数据库': []
      },
      '示例制造集团': {
        'Bronze': ['work_orders', 'wo_tasks', 'assets', 'meter_readings', 'priorities', 'statuses', 'maintenance_types', 'scheduled_maintenances', 'sites', 'users', 'parts', 'purchase_orders', 'categories', 'failure_codes', 'crews'],
        'Silver': ['sensor_readings_1min'],
        'Gold': []
      }
    };
    const catIconDir = '<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"><path d="M2 3h10v8H2z"/><path d="M2 3l2-2h4l2 2"/></svg>';
    const catIconDb = '<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"><ellipse cx="7" cy="4" rx="5" ry="2"/><path d="M2 4v3c0 1.1 2.24 2 5 2s5-.9 5-2V4"/><path d="M2 7v3c0 1.1 2.24 2 5 2s5-.9 5-2V7"/></svg>';
    const catIconTable = '<svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"><rect x="1" y="2" width="12" height="10" rx="1.5"/><path d="M1 5h12M1 8h12M5 5v7M9 5v7"/></svg>';
    let catalogTempDir = null;
    let catalogTempDb = null;
    let catalogTempTable = null;
    let catalogConfirmedDir = null;
    let catalogConfirmedDb = null;
    let catalogConfirmedTable = null;

    function openCatalogModal() {
      catalogTempDir = catalogConfirmedDir;
      catalogTempDb = catalogConfirmedDb;
      catalogTempTable = catalogConfirmedTable;
      // Hide table column in new-table mode
      const isNewMode = document.querySelector('input[name="targetMode"]:checked')?.value === 'new';
      document.getElementById('catalogColTable').style.display = isNewMode ? 'none' : '';
      renderCatalogModal();
      document.getElementById('catalogModal').classList.add('open');
    }

    function closeCatalogModal() {
      document.getElementById('catalogModal').classList.remove('open');
    }

    function renderCatalogModal() {
      // Directories
      const dirList = document.getElementById('catalogDirList');
      dirList.innerHTML = Object.keys(catalogData).map(d =>
        `<div class="cat-item${catalogTempDir === d ? ' active' : ''}" onclick="selectCatalogDir('${d}')">
          <span class="cat-icon">${catIconDir}</span><span>${d}</span><span class="cat-arrow">›</span></div>`
      ).join('');
      // Databases
      const dbList = document.getElementById('catalogDbList');
      if (catalogTempDir) {
        const dbs = Object.keys(catalogData[catalogTempDir] || {});
        dbList.innerHTML = dbs.length ? dbs.map(d =>
          `<div class="cat-item${catalogTempDb === d ? ' active' : ''}" onclick="selectCatalogDb('${d}')">
            <span class="cat-icon">${catIconDb}</span><span>${d}</span><span class="cat-arrow">›</span></div>`
        ).join('') : renderCatalogEmpty();
      } else {
        dbList.innerHTML = renderCatalogEmpty();
      }
      // Tables
      const tableList = document.getElementById('catalogTableList');
      if (catalogTempDir && catalogTempDb) {
        const tables = catalogData[catalogTempDir]?.[catalogTempDb] || [];
        tableList.innerHTML = tables.length ? tables.map(t =>
          `<div class="cat-item${catalogTempTable === t ? ' active' : ''}" onclick="selectCatalogTable('${t}')">
            <span class="cat-icon">${catIconTable}</span><span>${t}</span></div>`
        ).join('') : renderCatalogEmpty();
      } else {
        tableList.innerHTML = renderCatalogEmpty();
      }
      updateConfirmBtn();
    }

    function renderCatalogEmpty() {
      return `<div class="cat-empty">
        <svg width="48" height="48" viewBox="0 0 48 48" fill="none" stroke="currentColor" stroke-width="1.2">
          <rect x="8" y="14" width="32" height="24" rx="3"/><path d="M8 14l6-6h20l6 6"/><path d="M18 26h12"/>
        </svg>暂无数据</div>`;
    }

    function selectCatalogDir(dir) {
      catalogTempDir = dir;
      catalogTempDb = null;
      catalogTempTable = null;
      renderCatalogModal();
    }

    function selectCatalogDb(db) {
      catalogTempDb = db;
      catalogTempTable = null;
      renderCatalogModal();
    }

    function selectCatalogTable(table) {
      catalogTempTable = table;
      renderCatalogModal();
    }

    function updateConfirmBtn() {
      const btn = document.getElementById('catalogConfirmBtn');
      const isNewMode = document.querySelector('input[name="targetMode"]:checked')?.value === 'new';
      // For new table: need dir + db; for existing: need dir + db (table optional since it may be empty)
      const valid = catalogTempDir && catalogTempDb;
      btn.classList.toggle('active', !!valid);
    }

    function confirmCatalogSelection() {
      const isNewMode = document.querySelector('input[name="targetMode"]:checked')?.value === 'new';
      if (!catalogTempDir || !catalogTempDb) return;
      catalogConfirmedDir = catalogTempDir;
      catalogConfirmedDb = catalogTempDb;
      catalogConfirmedTable = catalogTempTable;
      // Update trigger display
      const trigger = document.getElementById('catalogTriggerText');
      trigger.className = 'trigger-value';
      const parts = ['⊙' + catalogConfirmedDir, '⊙' + catalogConfirmedDb];
      if (catalogConfirmedTable) parts.push(catalogConfirmedTable);
      trigger.textContent = parts.join(' / ');
      closeCatalogModal();
      checkShowBottomSection();
    }

    // === Check if bottom section (表定义 / 表映射) should be shown ===
    function isDataSourceReady() {
      // Check if structured source tab is "本地上传" (always ready) or connector is selected
      const isLocal = document.getElementById('stLocal').style.display !== 'none';
      if (isLocal) return true;
      const sel = document.getElementById('stConnectorSelect');
      if (!sel.value) return false;
      const opt = sel.options[sel.selectedIndex];
      const isDb = opt && opt.getAttribute('data-type') === 'db';
      if (isDb) {
        // DB connector needs database + table selected
        return !!document.getElementById('stDbNameSelect').value && !!document.getElementById('stDbTableSelect').value;
      }
      // File connector: just needs connector selected (file path is optional for gating)
      return true;
    }

    function isCatalogReady() {
      return !!catalogConfirmedDir && !!catalogConfirmedDb;
    }

    // Show either 表定义 or 表映射 content inside the unified card
    function showBottomContent(isNew) {
      const defContent = document.getElementById('tableDefContent');
      const mapContent = document.getElementById('colMappingContent');
      // Determine if current source is DB
      const isLocal = document.getElementById('stLocal').style.display !== 'none';
      const sel = document.getElementById('stConnectorSelect');
      const opt = sel.options[sel.selectedIndex];
      const isDb = !isLocal && opt && opt.getAttribute('data-type') === 'db';
      if (isNew) {
        defContent.style.display = '';
        mapContent.style.display = 'none';
        document.getElementById('newTableSchema').style.display = 'block';
        renderColSchemaTable();
        // PK conflict: show when any PK checkbox is checked in the schema table
        updatePkConflictVisibility();
      } else {
        defContent.style.display = 'none';
        mapContent.style.display = '';
        // Show/hide file-only mapping settings
        document.getElementById('fileOnlyMappingSettings').style.display = isDb ? 'none' : 'block';
        // PK conflict: show only when target table has PK columns
        const hasPkInTarget = mockTgtCols.some(c => c.pk);
        document.getElementById('pkConflictMapping').style.display = hasPkInTarget ? '' : 'none';
        renderColMapping();
      }
    }

    function checkShowBottomSection() {
      const ready = isDataSourceReady() && isCatalogReady();
      const mode = document.querySelector('input[name="targetMode"]:checked').value;
      const isNew = mode === 'new';
      const card = document.getElementById('bottomFormCard');
      if (ready) {
        card.style.display = '';
        document.getElementById('bottomSheetTabs').style.display = 'none';
        showBottomContent(isNew);
      } else {
        card.style.display = 'none';
      }
    }

    // === Target mode toggle ===
    function onTargetModeChange() {
      const mode = document.querySelector('input[name="targetMode"]:checked').value;
      const isNew = mode === 'new';
      document.getElementById('newTableSchema').style.display = isNew ? 'block' : 'none';
      // Hide 初次载入规则 for new table (new table is always empty)
      const initRule = document.getElementById('stInitLoadRuleDiv');
      if (initRule) initRule.style.display = isNew ? 'none' : '';
      if (isNew) {
        catalogTempTable = null;
        catalogConfirmedTable = null;
      }
      // Reset catalog trigger if mode changed
      if (catalogConfirmedDir) {
        const trigger = document.getElementById('catalogTriggerText');
        trigger.className = 'trigger-value';
        const parts = ['⊙' + catalogConfirmedDir, '⊙' + catalogConfirmedDb];
        if (!isNew && catalogConfirmedTable) parts.push(catalogConfirmedTable);
        trigger.textContent = parts.join(' / ');
      }
      checkShowBottomSection();
    }

    // === Column name toggle ===
    function onEnableColNameChange() {
      const enabled = document.getElementById('enableColName').checked;
      document.getElementById('colNameSettings').style.display = enabled ? 'inline-flex' : 'none';
      renderColSchemaTable();
    }

    function onHeaderRowChange() {
      const headerRow = parseInt(document.getElementById('headerRowInput').value) || 1;
      document.getElementById('dataStartRowInput').value = headerRow + 1;
      renderColSchemaTable();
    }

    // === Mapping section: file-only settings ===
    function onEnableColNameChangeMapping() {
      const enabled = document.getElementById('enableColNameMapping').checked;
      document.getElementById('colNameSettingsMapping').style.display = enabled ? 'inline-flex' : 'none';
      renderColMapping();
    }

    function onHeaderRowChangeMapping() {
      const headerRow = parseInt(document.getElementById('headerRowInputMapping').value) || 1;
      document.getElementById('dataStartRowInputMapping').value = headerRow + 1;
    }

    // === Column schema table ===
    const mockColSchema = [
      { idx: 'A', name: '序号', type: 'VARCHAR', len: 255, pk: false, desc: '', def: '', preview: ['1', '2', '3', '4', '5'] },
      { idx: 'B', name: '一级模块', type: 'VARCHAR', len: 256, pk: false, desc: '', def: '', preview: ['数据中心', '模型中心'] },
      { idx: 'C', name: '二级模块', type: 'VARCHAR', len: 255, pk: false, desc: '', def: '', preview: ['数据管理', '数据处理', '数据标注', '数据回流', '模型管理'] },
      { idx: 'D', name: '技术规格需求描述', type: 'VARCHAR', len: 255, pk: false, desc: '', def: '', preview: ['要求提供数据集管理和数据源管理的能力，实现高速相关数据资产的统一管理'] },
      { idx: 'E', name: '优先级', type: 'INT', len: 32, pk: false, desc: '', def: '', preview: ['P0', 'P1', 'P0', 'P2', 'P1'] },
    ];

    // MongoDB collection schemas (real 示例制造集团 data)
    const mongoColSchemas = {
      'sensor_readings': [
        { idx: '1', name: '_id', type: 'VARCHAR', len: 64, pk: true, desc: 'MongoDB ObjectId', def: '', preview: ['66a1b2c3d4e5...', '66a1b2c3d4e6...'] },
        { idx: '2', name: 'pump', type: 'VARCHAR', len: 32, pk: false, desc: '泵编号', def: '', preview: ['HP-101', 'HP-102', 'HP-103'] },
        { idx: '3', name: 'crew', type: 'VARCHAR', len: 32, pk: false, desc: '作业队', def: '', preview: ['Crew-A', 'Crew-A', 'Crew-B'] },
        { idx: '4', name: 'datetime', type: 'TIMESTAMP', len: 0, pk: false, desc: '采集时间（UTC）', def: '', preview: ['2026-04-10 14:32:15', '2026-04-10 14:32:16'] },
        { idx: '5', name: 'engine_rpm', type: 'DOUBLE', len: 0, pk: false, desc: '发动机转速', def: '', preview: ['1245.6', '1246.1', '0.0', '1198.3'] },
        { idx: '6', name: 'pump_rate', type: 'DOUBLE', len: 0, pk: false, desc: '泵速率', def: '', preview: ['3.82', '3.81', '0.00', '3.95'] },
        { idx: '7', name: 'disch_pressure', type: 'DOUBLE', len: 0, pk: false, desc: '排出压力', def: '', preview: ['8520.3', '8518.7', 'NULL', '8601.2'] },
        { idx: '8', name: 'engine_oil_pressure', type: 'DOUBLE', len: 0, pk: false, desc: '机油压力', def: '', preview: ['45.2', '45.1', '0.0', '44.8'] },
        { idx: '9', name: 'engine_coolant_temp', type: 'DOUBLE', len: 0, pk: false, desc: '冷却液温度', def: '', preview: ['88.1', '88.2', '25.3', '87.9'] },
        { idx: '10', name: 'lube_oil_pressure', type: 'DOUBLE', len: 0, pk: false, desc: '润滑油压力', def: '', preview: ['32.7', '32.6', '0.0', '33.1'] },
        { idx: '11', name: 'engine_hours', type: 'DOUBLE', len: 0, pk: false, desc: '发动机累计运行小时（累加器）', def: '', preview: ['12456.78', '12456.78', '12456.78'] },
        { idx: '12', name: 'pumping_hours', type: 'DOUBLE', len: 0, pk: false, desc: '泵累计运行小时（累加器）', def: '', preview: ['8234.56', '8234.56', '8234.56'] }
      ],
      'pump_metadata': [
        { idx: '1', name: '_id', type: 'VARCHAR', len: 64, pk: true, desc: 'MongoDB ObjectId', def: '', preview: ['66b2c3d4e5f6...'] },
        { idx: '2', name: 'pump_code', type: 'VARCHAR', len: 32, pk: false, desc: '泵编号', def: '', preview: ['HP-101', 'HP-102'] },
        { idx: '3', name: 'pump_type', type: 'VARCHAR', len: 64, pk: false, desc: '泵型号', def: '', preview: ['Quintuplex', 'Triplex'] },
        { idx: '4', name: 'manufacturer', type: 'VARCHAR', len: 128, pk: false, desc: '制造商', def: '', preview: ['Gardner Denver', 'NOV'] },
        { idx: '5', name: 'crew', type: 'VARCHAR', len: 32, pk: false, desc: '所属作业队', def: '', preview: ['Crew-A', 'Crew-B'] },
        { idx: '6', name: 'install_date', type: 'DATE', len: 0, pk: false, desc: '安装日期', def: '', preview: ['2023-06-15', '2024-01-20'] },
        { idx: '7', name: 'max_pressure', type: 'DOUBLE', len: 0, pk: false, desc: '最大额定压力', def: '', preview: ['15000', '12000'] },
        { idx: '8', name: 'status', type: 'VARCHAR', len: 16, pk: false, desc: '当前状态', def: '', preview: ['active', 'active', 'maintenance'] }
      ],
      'alert_events': [
        { idx: '1', name: '_id', type: 'VARCHAR', len: 64, pk: true, desc: 'MongoDB ObjectId', def: '', preview: ['66c3d4e5f6a7...'] },
        { idx: '2', name: 'pump', type: 'VARCHAR', len: 32, pk: false, desc: '泵编号', def: '', preview: ['HP-101', 'HP-205'] },
        { idx: '3', name: 'alert_type', type: 'VARCHAR', len: 64, pk: false, desc: '告警类型', def: '', preview: ['high_pressure', 'low_oil_pressure', 'high_temp'] },
        { idx: '4', name: 'severity', type: 'VARCHAR', len: 16, pk: false, desc: '严重程度', def: '', preview: ['critical', 'warning', 'info'] },
        { idx: '5', name: 'triggered_at', type: 'TIMESTAMP', len: 0, pk: false, desc: '触发时间', def: '', preview: ['2026-04-10 08:15:32', '2026-04-09 22:41:05'] },
        { idx: '6', name: 'resolved_at', type: 'TIMESTAMP', len: 0, pk: false, desc: '解除时间', def: 'NULL', preview: ['2026-04-10 08:22:10', 'NULL'] },
        { idx: '7', name: 'threshold', type: 'JSON', len: 0, pk: false, desc: '阈值配置（嵌套对象）', def: '', preview: ['{"field":"disch_pressure","op":">","value":12000}', '{"field":"engine_oil_pressure","op":"<","value":20}'] },
        { idx: '8', name: 'context', type: 'JSON', len: 0, pk: false, desc: '触发时上下文（嵌套对象）', def: '', preview: ['{"reading_value":12350.5,"engine_rpm":1320.0}', '{"reading_value":18.2,"engine_rpm":1180.5}'] },
        { idx: '9', name: 'acknowledged', type: 'BOOLEAN', len: 0, pk: false, desc: '是否已确认', def: 'false', preview: ['true', 'false', 'true'] }
      ],
      'crew_config': [
        { idx: '1', name: '_id', type: 'VARCHAR', len: 64, pk: true, desc: 'MongoDB ObjectId', def: '', preview: ['66d4e5f6a7b8...'] },
        { idx: '2', name: 'crew_code', type: 'VARCHAR', len: 16, pk: false, desc: '作业队编号', def: '', preview: ['Crew-A', 'Crew-B'] },
        { idx: '3', name: 'location', type: 'JSON', len: 0, pk: false, desc: '作业位置（嵌套：lat/lng/site/region）', def: '', preview: ['{"lat":25.3,"lng":49.5,"site":{"name":"Rig-7","field":"Ghawar"},"region":"Eastern"}'] },
        { idx: '4', name: 'equipment', type: 'JSON', len: 0, pk: false, desc: '设备配置（嵌套数组+对象）', def: '', preview: ['{"pumps":["HP-101","HP-102"],"sensors":{"count":12,"types":["pressure","temp","rpm"]},"maintenance":{"schedule":"weekly","last_date":"2026-04-01","provider":{"name":"示例制造集团","contract":"C-2026-001"}}}'] },
        { idx: '5', name: 'thresholds', type: 'JSON', len: 0, pk: false, desc: '告警阈值配置（多层嵌套）', def: '', preview: ['{"pressure":{"high":{"value":12000,"action":"alert"},"critical":{"value":15000,"action":"shutdown"}},"temperature":{"high":{"value":95,"action":"alert"},"critical":{"value":105,"action":"shutdown"}},"oil_pressure":{"low":{"value":20,"action":"alert"}}}'] },
        { idx: '6', name: 'shifts', type: 'JSON', len: 0, pk: false, desc: '班次配置（嵌套数组）', def: '', preview: ['[{"name":"day","start":"06:00","end":"18:00","supervisor":{"name":"Ahmed","id":"E-101"}},{"name":"night","start":"18:00","end":"06:00","supervisor":{"name":"Khalid","id":"E-205"}}]'] },
        { idx: '7', name: 'active', type: 'BOOLEAN', len: 0, pk: false, desc: '是否活跃', def: 'true', preview: ['true', 'true'] },
        { idx: '8', name: 'updated_at', type: 'TIMESTAMP', len: 0, pk: false, desc: '最后更新时间', def: '', preview: ['2026-04-10 06:00:00'] }
      ]
    };

    // Hive table schemas (示例制造集团 数据仓库) — 含推荐增量字段（partition_date / event_time / etl_load_time）
    const hiveColSchemas = {
      'ods_well_production_daily': [
        { idx: '1', name: 'well_id',         type: 'VARCHAR',   len: 32, pk: true,  desc: '井号',           def: '', preview: ['W-001', 'W-002'] },
        { idx: '2', name: 'production_date', type: 'DATE',      len: 0,  pk: true,  desc: '生产日（分区键）', def: '', preview: ['2026-05-24', '2026-05-25'] },
        { idx: '3', name: 'field_name',      type: 'VARCHAR',   len: 64, pk: false, desc: '油田名称',        def: '', preview: ['Ghawar', 'Safaniya'] },
        { idx: '4', name: 'oil_bbl',         type: 'DOUBLE',    len: 0,  pk: false, desc: '日产油（桶）',    def: '', preview: ['12450.3', '9820.1'] },
        { idx: '5', name: 'gas_mcf',         type: 'DOUBLE',    len: 0,  pk: false, desc: '日产气（千立方英尺）', def: '', preview: ['8520.0', '6105.7'] },
        { idx: '6', name: 'water_bbl',       type: 'DOUBLE',    len: 0,  pk: false, desc: '日产水（桶）',    def: '', preview: ['320.5', '480.2'] },
        { idx: '7', name: 'avg_pressure_psi',type: 'DOUBLE',    len: 0,  pk: false, desc: '平均井压 (psi)',  def: '', preview: ['3200.5', '3105.8'] },
        { idx: '8', name: 'downtime_hours',  type: 'DOUBLE',    len: 0,  pk: false, desc: '当日停机小时',    def: '0', preview: ['0.0', '2.5'] },
        { idx: '9', name: 'etl_load_time',   type: 'TIMESTAMP', len: 0,  pk: false, desc: 'ETL 入库时间',    def: '', preview: ['2026-05-25 03:15:00'] }
      ],
      'ods_pump_failure_history': [
        { idx: '1', name: 'failure_id',    type: 'BIGINT',    len: 0,   pk: true,  desc: '故障 ID',        def: '', preview: ['180001', '180002'] },
        { idx: '2', name: 'pump_code',     type: 'VARCHAR',   len: 32,  pk: false, desc: '泵编号',         def: '', preview: ['HP-101', 'HP-104'] },
        { idx: '3', name: 'failure_time',  type: 'TIMESTAMP', len: 0,   pk: false, desc: '故障发生时间',   def: '', preview: ['2026-05-24 22:15:00'] },
        { idx: '4', name: 'failure_type',  type: 'VARCHAR',   len: 64,  pk: false, desc: '故障类型',       def: '', preview: ['Seal Leak', 'Bearing Wear'] },
        { idx: '5', name: 'duration_hours',type: 'DOUBLE',    len: 0,   pk: false, desc: '停机时长（小时）', def: '', preview: ['4.2', '12.5'] },
        { idx: '6', name: 'cause_text',    type: 'TEXT',      len: 0,   pk: false, desc: '故障描述',       def: '', preview: ['Oil leak detected at main seal'] },
        { idx: '7', name: 'recorded_by',   type: 'VARCHAR',   len: 64,  pk: false, desc: '记录人',         def: '', preview: ['ahmed.k', 'sara.m'] },
        { idx: '8', name: 'etl_load_time', type: 'TIMESTAMP', len: 0,   pk: false, desc: 'ETL 入库时间',   def: '', preview: ['2026-05-25 03:15:00'] }
      ],
      'ods_drilling_logs': [
        { idx: '1', name: 'log_id',        type: 'BIGINT',    len: 0,   pk: true,  desc: '钻井日志 ID',     def: '', preview: ['12800001', '12800002'] },
        { idx: '2', name: 'well_id',       type: 'VARCHAR',   len: 32,  pk: false, desc: '井号',           def: '', preview: ['W-001', 'W-005'] },
        { idx: '3', name: 'log_time',      type: 'TIMESTAMP', len: 0,   pk: false, desc: '采样时间',       def: '', preview: ['2026-05-25 02:30:15'] },
        { idx: '4', name: 'depth_ft',      type: 'DOUBLE',    len: 0,   pk: false, desc: '当前钻深（英尺）',def: '', preview: ['8520.3', '8521.1'] },
        { idx: '5', name: 'rop_ft_hr',     type: 'DOUBLE',    len: 0,   pk: false, desc: '机械钻速 ROP',   def: '', preview: ['45.2', '43.8'] },
        { idx: '6', name: 'wob_klbs',      type: 'DOUBLE',    len: 0,   pk: false, desc: '钻压 WOB（千磅）',def: '', preview: ['32.5', '33.1'] },
        { idx: '7', name: 'rpm',           type: 'INT',       len: 0,   pk: false, desc: '转盘转速',       def: '', preview: ['85', '90'] },
        { idx: '8', name: 'mud_density',   type: 'DOUBLE',    len: 0,   pk: false, desc: '泥浆密度 (ppg)', def: '', preview: ['11.2', '11.3'] },
        { idx: '9', name: 'etl_load_time', type: 'TIMESTAMP', len: 0,   pk: false, desc: 'ETL 入库时间',   def: '', preview: ['2026-05-25 03:15:00'] }
      ],
      'ods_field_inspection_notes': [
        { idx: '1', name: 'note_id',         type: 'BIGINT',    len: 0,   pk: true,  desc: '巡检记录 ID',    def: '', preview: ['95001'] },
        { idx: '2', name: 'inspection_date', type: 'DATE',      len: 0,   pk: false, desc: '巡检日期',       def: '', preview: ['2026-05-24'] },
        { idx: '3', name: 'inspector',       type: 'VARCHAR',   len: 64,  pk: false, desc: '巡检员',         def: '', preview: ['ahmed.k'] },
        { idx: '4', name: 'site_code',       type: 'VARCHAR',   len: 32,  pk: false, desc: '站点编码',       def: '', preview: ['SITE-A', 'SITE-B'] },
        { idx: '5', name: 'note_text',       type: 'TEXT',      len: 0,   pk: false, desc: '巡检描述',       def: '', preview: ['Visible corrosion on flange'] },
        { idx: '6', name: 'severity',        type: 'VARCHAR',   len: 16,  pk: false, desc: '严重等级',       def: '', preview: ['LOW', 'HIGH'] },
        { idx: '7', name: 'photo_url',       type: 'VARCHAR',   len: 255, pk: false, desc: '照片 URL',       def: '', preview: ['s3://demo_mfg/insp/95001.jpg'] },
        { idx: '8', name: 'etl_load_time',   type: 'TIMESTAMP', len: 0,   pk: false, desc: 'ETL 入库时间',   def: '', preview: ['2026-05-25 03:15:00'] }
      ],
      'dwd_well_kpi_hourly': [
        { idx: '1', name: 'well_id',                 type: 'VARCHAR',   len: 32, pk: true,  desc: '井号',          def: '', preview: ['W-001'] },
        { idx: '2', name: 'stat_hour',               type: 'TIMESTAMP', len: 0,  pk: true,  desc: '统计小时',      def: '', preview: ['2026-05-25 02:00:00'] },
        { idx: '3', name: 'production_rate_bbl_hr', type: 'DOUBLE',    len: 0,  pk: false, desc: '产油率（桶/时）',def: '', preview: ['520.3'] },
        { idx: '4', name: 'utilization_pct',         type: 'DOUBLE',    len: 0,  pk: false, desc: '利用率 %',      def: '', preview: ['92.5'] },
        { idx: '5', name: 'mtbf_hours',              type: 'DOUBLE',    len: 0,  pk: false, desc: '小时级 MTBF',   def: '', preview: ['480.2'] },
        { idx: '6', name: 'mttr_hours',              type: 'DOUBLE',    len: 0,  pk: false, desc: '小时级 MTTR',   def: '', preview: ['3.5'] },
        { idx: '7', name: 'alarm_count',             type: 'INT',       len: 0,  pk: false, desc: '告警次数',      def: '0', preview: ['2'] },
        { idx: '8', name: 'dwh_etl_time',            type: 'TIMESTAMP', len: 0,  pk: false, desc: 'DWH 加工时间',  def: '', preview: ['2026-05-25 03:30:00'] }
      ],
      'dwd_equipment_event': [
        { idx: '1', name: 'event_id',       type: 'BIGINT',    len: 0,   pk: true,  desc: '事件 ID',        def: '', preview: ['320001'] },
        { idx: '2', name: 'equipment_code', type: 'VARCHAR',   len: 32,  pk: false, desc: '设备编码',       def: '', preview: ['HP-101'] },
        { idx: '3', name: 'event_time',     type: 'TIMESTAMP', len: 0,   pk: false, desc: '事件时间',       def: '', preview: ['2026-05-25 02:15:33'] },
        { idx: '4', name: 'event_type',     type: 'VARCHAR',   len: 32,  pk: false, desc: '事件类型',       def: '', preview: ['START', 'STOP', 'ALARM', 'MAINT'] },
        { idx: '5', name: 'duration_sec',   type: 'INT',       len: 0,   pk: false, desc: '持续时长（秒）', def: '0', preview: ['180'] },
        { idx: '6', name: 'operator',       type: 'VARCHAR',   len: 64,  pk: false, desc: '操作员',         def: '', preview: ['ahmed.k'] },
        { idx: '7', name: 'dwh_etl_time',   type: 'TIMESTAMP', len: 0,   pk: false, desc: 'DWH 加工时间',   def: '', preview: ['2026-05-25 03:30:00'] }
      ],
      'ads_field_oee_monthly': [
        { idx: '1', name: 'field_name',       type: 'VARCHAR',   len: 64, pk: true,  desc: '油田名称',         def: '', preview: ['Ghawar', 'Safaniya'] },
        { idx: '2', name: 'report_month',     type: 'DATE',      len: 0,  pk: true,  desc: '报告月份（月首）', def: '', preview: ['2026-05-01'] },
        { idx: '3', name: 'availability_pct', type: 'DOUBLE',    len: 0,  pk: false, desc: '可用率 %',         def: '', preview: ['92.3'] },
        { idx: '4', name: 'performance_pct',  type: 'DOUBLE',    len: 0,  pk: false, desc: '性能率 %',         def: '', preview: ['88.5'] },
        { idx: '5', name: 'quality_pct',      type: 'DOUBLE',    len: 0,  pk: false, desc: '质量率 %',         def: '', preview: ['99.1'] },
        { idx: '6', name: 'oee_pct',          type: 'DOUBLE',    len: 0,  pk: false, desc: 'OEE 综合效率 %',   def: '', preview: ['80.9'] },
        { idx: '7', name: 'generated_at',     type: 'TIMESTAMP', len: 0,  pk: false, desc: '报告生成时间',     def: '', preview: ['2026-06-01 04:00:00'] }
      ],
      'ads_failure_pareto_quarter': [
        { idx: '1', name: 'report_quarter',   type: 'VARCHAR',   len: 8,  pk: true,  desc: '报告季度',         def: '', preview: ['2026-Q1', '2026-Q2'] },
        { idx: '2', name: 'failure_type',     type: 'VARCHAR',   len: 64, pk: true,  desc: '故障类型',         def: '', preview: ['Seal Leak', 'Bearing Wear'] },
        { idx: '3', name: 'count',            type: 'INT',       len: 0,  pk: false, desc: '故障次数',         def: '0', preview: ['18'] },
        { idx: '4', name: 'total_hours_lost', type: 'DOUBLE',    len: 0,  pk: false, desc: '累计损失小时',     def: '', preview: ['125.5'] },
        { idx: '5', name: 'pct_of_total',     type: 'DOUBLE',    len: 0,  pk: false, desc: '占比 %',           def: '', preview: ['22.3'] },
        { idx: '6', name: 'generated_at',     type: 'TIMESTAMP', len: 0,  pk: false, desc: '报告生成时间',     def: '', preview: ['2026-07-01 04:00:00'] }
      ]
    };

    function getActiveColSchema() {
      // 1. 编辑模式：优先用任务自带的 schema（来自共享数据源 IMPORT_TASKS_DISPLAY）
      //    解决"详情显示 schema，编辑页没有"的数据源不一致问题
      if (editMode && editTaskId && window.IMPORT_TASKS_DISPLAY) {
        var taskDisp = window.IMPORT_TASKS_DISPLAY[editTaskId];
        if (taskDisp && taskDisp.schema && taskDisp.schema.length) {
          // 共享 schema 用 {name, type, pk, nullable, desc} 字段；映射到编辑器内部 {idx, name, type, len, pk, desc, def, preview}
          return taskDisp.schema.map(function(c, i) {
            // type 可能含 (N) 精度，拆出来
            var typeStr = c.type || '';
            var m = typeStr.match(/^([A-Za-z0-9_]+)(?:\((\d+(?:,\d+)?)\))?/);
            return {
              idx: String(i + 1),
              name: c.name || '',
              type: m ? m[1].toUpperCase() : typeStr,
              len: m && m[2] ? m[2] : 0,
              pk: !!c.pk,
              desc: c.desc || '',
              def: c.defaultVal || '',
              preview: c.preview || ['—', '—']
            };
          });
        }
      }
      // 2. 新建模式：按所选连接器 + 表来取
      var connSel = document.getElementById('stConnectorSelect');
      var tableSel = document.getElementById('stDbTableSelect');
      if (connSel && connSel.value === 'mongodb' && tableSel && tableSel.value && mongoColSchemas[tableSel.value]) {
        return mongoColSchemas[tableSel.value];
      }
      if (connSel && connSel.value === 'hive' && tableSel && tableSel.value && hiveColSchemas[tableSel.value]) {
        return hiveColSchemas[tableSel.value];
      }
      return mockColSchema;
    }
    const colDataTypes = ['VARCHAR', 'INT', 'BIGINT', 'FLOAT', 'DOUBLE', 'DECIMAL', 'BOOLEAN', 'DATE', 'DATETIME', 'TIMESTAMP', 'TEXT', 'BLOB', 'JSON'];

    function renderColSchemaTable() {
      const tbody = document.getElementById('colSchemaBody');
      const thead = document.getElementById('colSchemaHead');
      if (!tbody || !thead) return;
      const useHeader = document.getElementById('enableColName').checked;
      const activeSchema = getActiveColSchema();

      // Always show full columns for all import types (file and DB)
      thead.innerHTML = `<tr>
        <th class="col-check"><input type="checkbox" checked onclick="toggleAllSchemaCols(this)" style="accent-color:#1677ff;cursor:pointer"></th>
        <th class="col-idx">#</th>
        <th>列名</th>
        <th>数据类型</th>
        <th class="col-pk">主键</th>
        <th>列描述</th>
        <th>默认值</th>
        <th>行数据信息</th>
      </tr>`;

      tbody.innerHTML = activeSchema.map((col, i) => {
        const colName = useHeader ? col.name : 'column' + (i + 1);
        const lenInput = col.len ? `<input class="col-len-input" type="number" value="${col.len}" min="1">` : '';
        return `<tr>
          <td class="col-check"><input type="checkbox" checked style="accent-color:#1677ff;cursor:pointer"></td>
          <td class="col-idx">${col.idx}</td>
          <td><input class="col-name-input" type="text" value="${colName}" placeholder="请输入列名"></td>
          <td><div class="col-type-group"><select class="col-type-select">${colDataTypes.map(t => `<option${t === col.type ? ' selected' : ''}>${t}</option>`).join('')}</select>${lenInput}</div></td>
          <td class="col-pk"><input type="checkbox" ${col.pk ? 'checked' : ''} style="accent-color:#1677ff;cursor:pointer" onchange="updatePkConflictVisibility()"></td>
          <td><input class="col-desc-input" type="text" value="${col.desc}" placeholder="列描述"></td>
          <td><input class="col-default-input" type="text" value="${col.def}" placeholder="默认值"></td>
          <td class="col-preview">${col.preview.join('&nbsp;&nbsp;&nbsp;')}</td>
        </tr>`;
      }).join('');
    }

    function updatePkConflictVisibility() {
      const pkChecks = document.querySelectorAll('#colSchemaBody .col-pk input[type="checkbox"]');
      const hasPk = Array.from(pkChecks).some(cb => cb.checked);
      document.getElementById('pkConflictTableDef').style.display = hasPk ? '' : 'none';
    }

    function toggleAllSchemaCols(master) {
      document.querySelectorAll('#colSchemaBody input[type="checkbox"]').forEach(cb => {
        if (cb.closest('.col-check')) cb.checked = master.checked;
      });
    }

    // === Column mapping (existing table mode) ===
    const mockSrcCols = [
      { name: '序号', inferredType: 'INT' },
      { name: '一级模块', inferredType: 'VARCHAR' },
      { name: '二级模块', inferredType: 'VARCHAR' },
      { name: '技术规格需求描述', inferredType: 'TEXT' },
      { name: '优先级', inferredType: 'VARCHAR' },
    ];
    const mockTgtCols = [
      { name: 'dept_id', type: 'VARCHAR(50)', pk: true },
      { name: 'parent_dept_id', type: 'VARCHAR(50)', pk: true },
      { name: 'dept_name', type: 'VARCHAR(200)', pk: false },
      { name: 'depth', type: 'TINYINT(8)', pk: false },
      { name: 'created_at', type: 'TIMESTAMP(0)', pk: false },
      { name: 'updated_at', type: 'TIMESTAMP(0)', pk: false },
    ];

    let currentMappingMode = 'smart'; // 'smart' or 'sequential'

    function renderColMapping() {
      // Detect if current source is DB
      const isLocal = document.getElementById('stLocal').style.display !== 'none';
      const sel = document.getElementById('stConnectorSelect');
      const opt = sel.options[sel.selectedIndex];
      const isDb = !isLocal && opt && opt.getAttribute('data-type') === 'db';

      // Source columns (left side) — dynamic based on data source type
      const srcHead = document.getElementById('srcColTable').querySelector('thead');
      const srcBody = document.getElementById('srcColList');
      if (isDb) {
        srcHead.innerHTML = '<tr><th>列名</th><th>列类型</th><th>行数据信息</th></tr>';
        srcBody.innerHTML = mockSrcCols.map((col, i) => {
          const schema = getActiveColSchema()[i];
          const preview = schema ? schema.preview.join('&nbsp;&nbsp;') : '';
          return `<tr>
            <td>${col.name}</td>
            <td class="col-type-cell">${schema ? schema.type + '(' + schema.len + ')' : col.inferredType}</td>
            <td class="col-preview">${preview}</td>
          </tr>`;
        }).join('');
      } else {
        const useHeader = document.getElementById('enableColNameMapping').checked;
        srcHead.innerHTML = '<tr><th>列名</th><th>推荐类型</th><th>行数据信息</th></tr>';
        srcBody.innerHTML = mockSrcCols.map((col, i) => {
          const colName = useHeader ? col.name : 'column' + (i + 1);
          const schema = getActiveColSchema()[i];
          const preview = schema ? schema.preview.join('&nbsp;&nbsp;') : '';
          return `<tr>
            <td>${colName}</td>
            <td class="col-type-cell">${col.inferredType}</td>
            <td class="col-preview">${preview}</td>
          </tr>`;
        }).join('');
      }

      // Target columns (right side) - table rows with select
      const tgtBody = document.getElementById('tgtColList');
      const srcOptions = mockSrcCols.map((s, i) => `<option value="${i}">${s.name} (${s.inferredType})</option>`).join('');

      tgtBody.innerHTML = mockTgtCols.map((col, ti) => {
        const isPk = col.pk;
        return `
          <tr>
            <td>${col.name}${isPk ? ' <span class="tgt-pk-badge">PK</span>' : ''}</td>
            <td class="col-type-cell">${col.type}</td>
            <td>
              <select class="tgt-col-select" data-tgt-idx="${ti}" onchange="onMappingChange()">
                ${isPk ? '' : '<option value="">不映射</option>'}
                ${srcOptions}
              </select>
            </td>
          </tr>
        `;
      }).join('');

      // Apply auto-matching based on current mode
      applyMappingMode(currentMappingMode);
    }

    function applyMappingMode(mode) {
      const selects = document.querySelectorAll('#tgtColList .tgt-col-select');
      const usedSrc = new Set();

      if (mode === 'smart') {
        // Smart matching: match by name similarity (mock: fuzzy substring match)
        selects.forEach(sel => {
          const tgtIdx = parseInt(sel.getAttribute('data-tgt-idx'));
          const tgtCol = mockTgtCols[tgtIdx];
          const tgtName = tgtCol.name.toLowerCase();
          let bestMatch = -1;
          // Try to find a source column with similar name
          mockSrcCols.forEach((src, si) => {
            if (usedSrc.has(si)) return;
            const srcName = src.name.toLowerCase();
            // Simple similarity: check if names share common substrings
            if (srcName === tgtName || srcName.includes(tgtName) || tgtName.includes(srcName)) {
              bestMatch = si;
            }
          });
          // Fallback: if no name match found and it's a PK, assign sequentially
          if (bestMatch === -1 && tgtCol.pk && !usedSrc.has(tgtIdx) && tgtIdx < mockSrcCols.length) {
            bestMatch = tgtIdx;
          }
          if (bestMatch !== -1 && !usedSrc.has(bestMatch)) {
            sel.value = String(bestMatch);
            usedSrc.add(bestMatch);
          } else if (!tgtCol.pk) {
            sel.value = '';
          } else if (tgtCol.pk) {
            // PK must be mapped — pick first available
            for (let i = 0; i < mockSrcCols.length; i++) {
              if (!usedSrc.has(i)) { sel.value = String(i); usedSrc.add(i); break; }
            }
          }
        });
      } else {
        // Sequential matching: source col 0 → target col 0, etc.
        selects.forEach(sel => {
          const tgtIdx = parseInt(sel.getAttribute('data-tgt-idx'));
          const tgtCol = mockTgtCols[tgtIdx];
          if (tgtIdx < mockSrcCols.length && !usedSrc.has(tgtIdx)) {
            sel.value = String(tgtIdx);
            usedSrc.add(tgtIdx);
          } else if (!tgtCol.pk) {
            sel.value = '';
          } else {
            // PK must be mapped — pick first available
            for (let i = 0; i < mockSrcCols.length; i++) {
              if (!usedSrc.has(i)) { sel.value = String(i); usedSrc.add(i); break; }
            }
          }
        });
      }
      onMappingChange();
    }

    function onMappingModeChange(mode, el) {
      currentMappingMode = mode;
      el.parentElement.querySelectorAll('.inline-tab').forEach(t => t.classList.remove('active'));
      el.classList.add('active');
      applyMappingMode(mode);
    }

    function onMappingChange() {
      // Enforce: each source column can only be selected once
      const selects = document.querySelectorAll('#tgtColList .tgt-col-select');
      const usedValues = new Set();
      // Collect currently selected values
      selects.forEach(sel => {
        if (sel.value) usedValues.add(sel.value);
      });
      // Disable already-used options in other selects
      selects.forEach(sel => {
        const currentVal = sel.value;
        Array.from(sel.options).forEach(opt => {
          if (opt.value && opt.value !== currentVal) {
            opt.disabled = usedValues.has(opt.value);
          } else {
            opt.disabled = false;
          }
        });
      });
    }

    // === Hive partition sync ===
    const hivePartFieldCols = ['ods_user_id', 'event_date', 'event_type', 'platform', 'region', 'amount'];
    let hiveSelectedFields = new Set();

    function onStLoadModeChange() {
      const mode = document.querySelector('input[name="stLoadMode"]:checked').value;
      const periodicOpts = document.getElementById('stPeriodicOptions');
      const incrConfig = document.getElementById('stIncrementalConfig');
      if (periodicOpts) periodicOpts.style.display = mode === 'periodic' ? '' : 'none';
      if (incrConfig) incrConfig.style.display = (mode === 'periodic' || mode === 'realtime') ? '' : 'none';
      // Populate incremental field dropdown from current schema
      if (mode === 'periodic' || mode === 'realtime') {
        populateIncrFieldDropdown();
      }
    }

    function onStSyncStrategyChange() {
      var sel = document.querySelector('input[name="stSyncStrategy"]:checked');
      var strategy = sel ? sel.value : 'incremental';
      var fields = document.getElementById('stIncrementalFields');
      var hint = document.getElementById('stFullRefreshHint');
      var backfillRow = document.getElementById('stBackfillToggle');
      if (fields) fields.style.display = strategy === 'incremental' ? '' : 'none';
      if (hint) hint.style.display = strategy === 'full' ? '' : 'none';
      // 全量覆盖时禁用回填（每次都全量，回填没意义）
      if (backfillRow && strategy === 'full') {
        backfillRow.checked = false;
        if (typeof onBackfillToggle === 'function') onBackfillToggle();
        backfillRow.disabled = true;
      } else if (backfillRow) {
        backfillRow.disabled = false;
      }
    }

    function onBackfillToggle() {
      var on = document.getElementById('stBackfillToggle').checked;
      document.getElementById('stBackfillConfig').style.display = on ? '' : 'none';
    }

    function onBackfillStartModeChange() {
      var mode = document.getElementById('stBackfillStartMode').value;
      document.getElementById('stBackfillStartDate').style.display = mode === 'custom' ? '' : 'none';
    }

    // === Preprocess ===
    function onPreprocessToggle() {
      var on = document.getElementById('preprocessToggle').checked;
      document.getElementById('preprocessConfig').style.display = on ? '' : 'none';
      // If turned off, re-render schema from source; if on, wait for preview
      if (!on && document.getElementById('bottomFormCard').style.display !== 'none') {
        renderColSchemaTable();
      }
    }

    function onPreprocessTypeChange() {
      var type = document.querySelector('input[name="preprocessType"]:checked').value;
      var label = document.getElementById('preprocessLangLabel');
      var editor = document.getElementById('preprocessEditor');
      var templates = {
        python: '# 载入前预处理脚本\n# 输入：raw_df（源端原始数据 DataFrame）\n# 输出：返回处理后的 DataFrame\n\nimport pandas as pd\n\ndef preprocess(raw_df):\n    # 示例：1秒级传感器数据聚合为1分钟级\n    df = raw_df.copy()\n    df[\'minute\'] = df[\'datetime\'].dt.floor(\'min\')\n    \n    result = df.groupby([\'pump\', \'minute\']).agg(\n        engine_rpm=(\'engine_rpm\', \'mean\'),\n        pump_rate=(\'pump_rate\', \'mean\'),\n        disch_pressure=(\'disch_pressure\', \'mean\'),\n        engine_hours=(\'engine_hours\', \'max\'),\n        pumping_hours=(\'pumping_hours\', \'max\'),\n        readings_in_minute=(\'engine_rpm\', \'count\')\n    ).reset_index()\n    \n    return result',
        mongo_agg: '// MongoDB Aggregation Pipeline\n// 在源端 MongoDB 执行聚合，减少传输数据量\n\n[\n  { "$match": { "datetime": { "$gte": "$$START", "$lt": "$$END" } } },\n  { "$group": {\n      "_id": {\n        "pump": "$pump",\n        "minute": { "$dateTrunc": { "date": "$datetime", "unit": "minute" } }\n      },\n      "engine_rpm": { "$avg": "$engine_rpm" },\n      "pump_rate": { "$avg": "$pump_rate" },\n      "disch_pressure": { "$avg": "$disch_pressure" },\n      "engine_hours": { "$max": "$engine_hours" },\n      "pumping_hours": { "$max": "$pumping_hours" },\n      "readings_in_minute": { "$sum": 1 }\n  } },\n  { "$project": {\n      "pump": "$_id.pump",\n      "minute": "$_id.minute",\n      "engine_rpm": 1, "pump_rate": 1, "disch_pressure": 1,\n      "engine_hours": 1, "pumping_hours": 1, "readings_in_minute": 1,\n      "_id": 0\n  } }\n]',
        sql: '-- SQL 预处理（在源端数据库执行）\n-- 结果将作为载入数据写入 MOI\n\nSELECT \n  pump,\n  DATE_TRUNC(\'minute\', datetime) AS minute,\n  AVG(engine_rpm) AS engine_rpm,\n  AVG(pump_rate) AS pump_rate,\n  AVG(disch_pressure) AS disch_pressure,\n  MAX(engine_hours) AS engine_hours,\n  MAX(pumping_hours) AS pumping_hours,\n  COUNT(*) AS readings_in_minute\nFROM sensor_readings\nGROUP BY pump, DATE_TRUNC(\'minute\', datetime)'
      };
      var labels = { python: 'Python', mongo_agg: 'MongoDB Aggregation', sql: 'SQL' };
      if (label) label.textContent = labels[type] || type;
      if (editor) editor.value = templates[type] || '';
      // Hide preview when switching type
      var preview = document.getElementById('preprocessPreview');
      if (preview) preview.style.display = 'none';
    }

    var _preprocessSchema = null;

    function previewPreprocessOutput() {
      // Mock: parse output schema from the script
      var type = document.querySelector('input[name="preprocessType"]:checked').value;
      var mockSchema = [
        { idx: '1', name: 'pump', type: 'VARCHAR', len: 32, pk: false, desc: '泵编号', def: '', preview: ['HP-101', 'HP-102'] },
        { idx: '2', name: 'minute', type: 'TIMESTAMP', len: 0, pk: false, desc: '分钟时间戳', def: '', preview: ['2026-04-10 14:32:00'] },
        { idx: '3', name: 'engine_rpm', type: 'DOUBLE', len: 0, pk: false, desc: '平均发动机转速', def: '', preview: ['1245.6'] },
        { idx: '4', name: 'pump_rate', type: 'DOUBLE', len: 0, pk: false, desc: '平均泵速率', def: '', preview: ['3.82'] },
        { idx: '5', name: 'disch_pressure', type: 'DOUBLE', len: 0, pk: false, desc: '平均排出压力', def: '', preview: ['8520.3'] },
        { idx: '6', name: 'engine_hours', type: 'DOUBLE', len: 0, pk: false, desc: '发动机累计运行小时', def: '', preview: ['12456.78'] },
        { idx: '7', name: 'pumping_hours', type: 'DOUBLE', len: 0, pk: false, desc: '泵累计运行小时', def: '', preview: ['8234.56'] },
        { idx: '8', name: 'readings_in_minute', type: 'INT', len: 0, pk: false, desc: '该分钟内原始读数条数', def: '', preview: ['60', '58'] }
      ];
      _preprocessSchema = mockSchema;

      var fieldsEl = document.getElementById('preprocessSchemaFields');
      fieldsEl.innerHTML = mockSchema.map(function(f) {
        return '<span style="padding:3px 10px;background:#f0f5ff;border:1px solid #d6e4ff;border-radius:4px;font-size:12px;font-family:monospace;color:#1677ff">' + f.name + ' <span style="color:rgba(0,0,0,0.35)">' + f.type + '</span></span>';
      }).join('');
      document.getElementById('preprocessPreview').style.display = '';

      // Re-render table definition with preprocessed schema
      if (document.getElementById('bottomFormCard').style.display !== 'none') {
        renderColSchemaTable();
      }
    }

    // Extend getActiveColSchema to use preprocessed schema when available
    var _origGetActiveColSchema3 = getActiveColSchema;
    getActiveColSchema = function() {
      if (document.getElementById('preprocessToggle') && document.getElementById('preprocessToggle').checked && _preprocessSchema) {
        return _preprocessSchema;
      }
      return _origGetActiveColSchema3();
    };

    function populateIncrFieldDropdown() {
      var sel = document.getElementById('stIncrField');
      if (!sel) return;
      var schema = getActiveColSchema();
      var timeFields = schema.filter(function(col) {
        return col.type === 'TIMESTAMP' || col.type === 'DATETIME' || col.type === 'DATE'
          || col.name.toLowerCase().indexOf('date') !== -1
          || col.name.toLowerCase().indexOf('time') !== -1
          || col.name.toLowerCase().indexOf('updated') !== -1
          || col.name.toLowerCase().indexOf('modified') !== -1
          || col.name.toLowerCase().indexOf('created') !== -1;
      });
      var otherFields = schema.filter(function(col) {
        return timeFields.indexOf(col) === -1 && col.type !== 'JSON' && col.type !== 'BOOLEAN';
      });
      var html = '<option value="">请选择增量字段</option>';
      if (timeFields.length > 0) {
        html += '<optgroup label="推荐（时间戳字段）">';
        timeFields.forEach(function(f) { html += '<option value="' + f.name + '">' + f.name + ' (' + f.type + ')</option>'; });
        html += '</optgroup>';
      }
      if (otherFields.length > 0) {
        html += '<optgroup label="其他字段">';
        otherFields.forEach(function(f) { html += '<option value="' + f.name + '">' + f.name + ' (' + f.type + ')</option>'; });
        html += '</optgroup>';
      }
      sel.innerHTML = html;
      // Auto-select first time field
      if (timeFields.length > 0) sel.value = timeFields[0].name;
    }

    function onHivePartitionToggle() {
      const on = document.getElementById('hivePartitionToggle').checked;
      document.getElementById('hivePartitionOptions').style.display = on ? 'flex' : 'none';
      if (on) renderHiveFieldDropdown();
    }

    function renderHiveFieldDropdown() {
      const dd = document.getElementById('hiveFieldDropdown');
      dd.innerHTML = hivePartFieldCols.map(c =>
        `<label class="hive-field-option" onclick="event.stopPropagation()">
          <input type="checkbox" value="${c}" ${hiveSelectedFields.has(c) ? 'checked' : ''} onchange="onHiveFieldCheck(this)">
          ${c}
        </label>`
      ).join('');
    }

    function toggleHiveFieldDropdown() {
      const dd = document.getElementById('hiveFieldDropdown');
      dd.classList.toggle('open');
    }

    function onHiveFieldCheck(cb) {
      if (cb.checked) { hiveSelectedFields.add(cb.value); } else { hiveSelectedFields.delete(cb.value); }
      updateHiveFieldDisplay();
    }

    function updateHiveFieldDisplay() {
      const display = document.getElementById('hiveFieldDisplay');
      if (hiveSelectedFields.size === 0) {
        display.className = 'hive-field-placeholder';
        display.innerHTML = '请选择分区字段';
      } else {
        display.className = 'hive-field-tags';
        display.innerHTML = Array.from(hiveSelectedFields).map(f =>
          `<span class="hive-field-tag">${f}<span class="tag-x" onclick="event.stopPropagation();removeHiveField('${f}')">✕</span></span>`
        ).join('');
      }
    }

    function removeHiveField(field) {
      hiveSelectedFields.delete(field);
      updateHiveFieldDisplay();
      renderHiveFieldDropdown();
    }

    // Close dropdown when clicking outside
    document.addEventListener('click', function(e) {
      const dd = document.getElementById('hiveFieldDropdown');
      const trigger = document.getElementById('hiveFieldTrigger');
      if (dd && trigger && !trigger.contains(e.target) && !dd.contains(e.target)) {
        dd.classList.remove('open');
      }
    });

    // === File browser mock data ===
    const stMockFiles = {
      's3': [
        { name: '2024年销售数据.xlsx', size: '2.4 MB', ext: 'xlsx' },
        { name: '客户信息导出.csv', size: '856 KB', ext: 'csv' },
        { name: '产品目录_v3.xlsx', size: '1.1 MB', ext: 'xlsx' },
        { name: '月度报表.xls', size: '3.2 MB', ext: 'xls' },
        { name: 'readme.txt', size: '4 KB', ext: 'txt' },
        { name: '库存盘点.csv', size: '420 KB', ext: 'csv' },
        { name: '财务报告2024Q4.pdf', size: '5.8 MB', ext: 'pdf' },
        { name: '员工花名册.xlsx', size: '680 KB', ext: 'xlsx' },
      ],
      'oss': [
        { name: '订单明细_202401.csv', size: '12.6 MB', ext: 'csv' },
        { name: '供应商清单.xlsx', size: '340 KB', ext: 'xlsx' },
        { name: '物流数据.xls', size: '1.8 MB', ext: 'xls' },
        { name: '合同扫描件.pdf', size: '8.2 MB', ext: 'pdf' },
        { name: '渠道销售汇总.csv', size: '2.1 MB', ext: 'csv' },
      ],
      'hdfs': [
        { name: 'user_events_20240101.csv', size: '45.2 MB', ext: 'csv' },
        { name: 'transaction_log.csv', size: '128 MB', ext: 'csv' },
        { name: 'dim_store.xlsx', size: '520 KB', ext: 'xlsx' },
        { name: 'fact_sales_2024.xlsx', size: '8.6 MB', ext: 'xlsx' },
        { name: 'config.json', size: '2 KB', ext: 'json' },
        { name: 'schema_backup.sql', size: '156 KB', ext: 'sql' },
      ]
    };
    const stMockExcelSheets = {
      '2024年销售数据.xlsx': [
        { name: 'Q1销售', rows: '3,200' }, { name: 'Q2销售', rows: '4,100' },
        { name: 'Q3销售', rows: '3,800' }, { name: 'Q4销售', rows: '5,600' }, { name: '年度汇总', rows: '12' }
      ],
      '产品目录_v3.xlsx': [
        { name: '电子产品', rows: '1,250' }, { name: '家居用品', rows: '890' }, { name: '食品饮料', rows: '2,100' }
      ],
      '月度报表.xls': [
        { name: '1月', rows: '320' }, { name: '2月', rows: '280' }, { name: '3月', rows: '350' },
        { name: '4月', rows: '310' }, { name: '5月', rows: '290' }, { name: '6月', rows: '340' }
      ],
      '员工花名册.xlsx': [
        { name: '在职员工', rows: '456' }, { name: '离职员工', rows: '89' }
      ],
      '供应商清单.xlsx': [
        { name: '国内供应商', rows: '230' }, { name: '海外供应商', rows: '78' }
      ],
      '物流数据.xls': [
        { name: '发货记录', rows: '5,600' }, { name: '退货记录', rows: '320' }
      ],
      'dim_store.xlsx': [
        { name: '门店信息', rows: '1,200' }, { name: '区域划分', rows: '45' }
      ],
      'fact_sales_2024.xlsx': [
        { name: '线上销售', rows: '125,000' }, { name: '线下销售', rows: '89,000' }, { name: '退款', rows: '3,200' }
      ]
    };
    // Local upload mock Excel sheets
    const localMockExcelSheets = {
      'xlsx': [
        { name: '数据表1', rows: '1,500' }, { name: '数据表2', rows: '800' }, { name: '汇总', rows: '25' }
      ],
      'xls': [
        { name: 'Sheet1', rows: '2,000' }, { name: 'Sheet2', rows: '450' }
      ]
    };

    let selectedFile = null;
    let selectedSheets = new Set();
    let localSelectedFile = null;
    let localSelectedSheets = new Set();
    // Per-sheet target state: { sheetName: { mode:'existing'|'new', dir, db, table } }
    let sheetTargets = {};
    let editingSheetName = null; // which sheet's catalog is being edited
    let isMultiSheetMode = false;
    let activeSheetTab = null; // for table def / col mapping tabs

    const allowedExts = new Set(['csv', 'xls', 'xlsx']);

    function isExcelExt(ext) { return ext === 'xlsx' || ext === 'xls'; }

    // === File browser modal ===
    function openFileBrowser() {
      const conn = document.getElementById('stConnectorSelect').value;
      if (!conn) { alert('请先选择连接器'); return; }
      const opt = document.getElementById('stConnectorSelect').selectedOptions[0];
      if (opt.getAttribute('data-type') === 'db') return;
      const files = stMockFiles[conn] || [];
      const list = document.getElementById('fileBrowserList');
      list.innerHTML = files.map((f, i) => {
        const ok = allowedExts.has(f.ext);
        const icon = ok ? '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/></svg>' : '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="rgba(0,0,0,0.25)" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6"/></svg>';
        return `<div class="file-browser-item${!ok ? ' disabled' : ''}${selectedFile && selectedFile.name === f.name ? ' active' : ''}" onclick="${ok ? `selectBrowserFile(${i})` : ''}">
          <span class="file-icon">${icon}</span>
          <span class="file-name">${f.name}</span>
          <span class="file-ext">${f.ext.toUpperCase()}</span>
          <span class="file-size">${f.size}</span>
        </div>`;
      }).join('');
      document.getElementById('fileBrowserModal').classList.add('open');
    }

    function closeFileBrowser() {
      document.getElementById('fileBrowserModal').classList.remove('open');
    }

    let fileBrowserTempSelection = null;
    function selectBrowserFile(idx) {
      const conn = document.getElementById('stConnectorSelect').value;
      const files = stMockFiles[conn] || [];
      fileBrowserTempSelection = files[idx];
      // Update active state
      const items = document.querySelectorAll('#fileBrowserList .file-browser-item');
      items.forEach((el, i) => el.classList.toggle('active', i === idx));
      document.getElementById('fileBrowserConfirmBtn').classList.add('active');
    }

    function confirmFileBrowser() {
      if (!fileBrowserTempSelection) return;
      selectedFile = fileBrowserTempSelection;
      selectedSheets = new Set();
      sheetTargets = {};
      isMultiSheetMode = false;
      // Update trigger text — show full path: connector name / file name（size）
      const trigger = document.getElementById('fileBrowserText');
      trigger.className = 'trigger-value';
      const connSel = document.getElementById('stConnectorSelect');
      const connName = connSel.options[connSel.selectedIndex].text;
      trigger.textContent = connName + ' / ' + selectedFile.name + '（' + selectedFile.size + '）';
      closeFileBrowser();
      // Check if Excel
      if (isExcelExt(selectedFile.ext)) {
        const sheets = stMockExcelSheets[selectedFile.name] || [];
        renderExcelSheets(sheets, 'excelSheetList');
        document.getElementById('excelSheetSection').style.display = 'block';
      } else {
        document.getElementById('excelSheetSection').style.display = 'none';
      }
      updateCsvConfigBtnVisibility();
      checkShowBottomSection();
    }

    // === Excel sheet rendering ===
    function renderExcelSheets(sheets, containerId) {
      const container = document.getElementById(containerId);
      container.innerHTML = sheets.map((s, i) =>
        `<label class="checkbox-tag" onclick="toggleSheetTag(this, '${s.name}', '${containerId}')">
          <input type="checkbox" style="display:none" value="${s.name}">
          ${s.name}（${s.rows} 行）
        </label>`
      ).join('');
    }

    function toggleSheetTag(el, sheetName, containerId) {
      const isConnector = containerId === 'excelSheetList';
      const set = isConnector ? selectedSheets : localSelectedSheets;
      const cb = el.querySelector('input');
      cb.checked = !cb.checked;
      el.classList.toggle('active', cb.checked);
      if (cb.checked) { set.add(sheetName); } else { set.delete(sheetName); delete sheetTargets[sheetName]; }
      onSheetSelectionChange(isConnector);
    }

    function onSheetSelectionChange(isConnector) {
      const set = isConnector ? selectedSheets : localSelectedSheets;
      isMultiSheetMode = set.size >= 2;
      if (isMultiSheetMode) {
        document.getElementById('singleTargetSection').style.display = 'none';
        document.getElementById('multiSheetTargetSection').style.display = 'block';
        renderMultiSheetTargets(set);
      } else {
        document.getElementById('singleTargetSection').style.display = 'block';
        document.getElementById('multiSheetTargetSection').style.display = 'none';
      }
      checkShowBottomSection();
    }

    // === Multi-sheet target rendering ===
    function renderMultiSheetTargets(sheetSet) {
      const container = document.getElementById('multiSheetTargetList');
      const sheetIcon = '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="#1677ff" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 3v6"/></svg>';
      container.innerHTML = Array.from(sheetSet).map(name => {
        const t = sheetTargets[name] || { mode: 'existing', dir: null, db: null, table: null };
        sheetTargets[name] = t;
        const triggerText = t.dir ? ('⊙' + t.dir + ' / ⊙' + t.db + (t.mode === 'existing' && t.table ? ' / ' + t.table : '')) : '';
        return `<div class="sheet-target-item">
          <div class="sheet-target-name">${sheetIcon} ${name}</div>
          <div class="sheet-target-row">
            <div class="radio-group">
              <label><input type="radio" name="sheetMode_${name}" value="existing" ${t.mode === 'existing' ? 'checked' : ''} onchange="onSheetModeChange('${name}', 'existing')"> 已有表</label>
              <label><input type="radio" name="sheetMode_${name}" value="new" ${t.mode === 'new' ? 'checked' : ''} onchange="onSheetModeChange('${name}', 'new')"> 新建表</label>
            </div>
            <div class="sheet-target-trigger" onclick="openSheetCatalog('${name}')">
              <svg width="12" height="12" viewBox="0 0 14 14" fill="none" stroke="rgba(0,0,0,0.45)" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round" style="flex-shrink:0"><path d="M2 3h10v8H2z"/><path d="M2 3l2-2h4l2 2"/></svg>
              ${triggerText ? `<span class="trigger-value">${triggerText}</span>` : '<span class="trigger-placeholder">请选择载入目标</span>'}
            </div>
          </div>
          ${t.mode === 'existing' ? `<div class="sheet-target-row" style="margin-top:8px;align-items:center">
            <span style="font-size:12px;color:rgba(0,0,0,0.45);min-width:48px">初次载入</span>
            <div class="radio-group" style="gap:14px">
              <label style="font-size:13px"><input type="radio" name="sheetInit_${name}" value="append" ${(t.initRule || 'append') === 'append' ? 'checked' : ''} onchange="onSheetInitRuleChange('${name}','append')"> 在已有数据后追加</label>
              <label style="font-size:13px"><input type="radio" name="sheetInit_${name}" value="truncate" ${t.initRule === 'truncate' ? 'checked' : ''} onchange="onSheetInitRuleChange('${name}','truncate')"> 清空已有数据后追加</label>
            </div>
          </div>` : ''}
        </div>`;
      }).join('');
    }

    function onSheetModeChange(sheetName, mode) {
      if (!sheetTargets[sheetName]) sheetTargets[sheetName] = {};
      sheetTargets[sheetName].mode = mode;
      sheetTargets[sheetName].table = null;
      if (mode === 'existing' && !sheetTargets[sheetName].initRule) sheetTargets[sheetName].initRule = 'append';
      const set = selectedSheets.size >= 2 ? selectedSheets : localSelectedSheets;
      renderMultiSheetTargets(set);
      checkShowBottomSection();
    }

    function onSheetInitRuleChange(sheetName, val) {
      if (!sheetTargets[sheetName]) sheetTargets[sheetName] = {};
      sheetTargets[sheetName].initRule = val;
    }

    // === Per-sheet catalog ===
    function openSheetCatalog(sheetName) {
      editingSheetName = sheetName;
      const t = sheetTargets[sheetName] || { mode: 'existing' };
      catalogTempDir = t.dir || null;
      catalogTempDb = t.db || null;
      catalogTempTable = t.table || null;
      const isNew = t.mode === 'new';
      document.getElementById('catalogColTable').style.display = isNew ? 'none' : '';
      renderCatalogModal();
      document.getElementById('catalogModal').classList.add('open');
    }

    // Patch confirmCatalogSelection to handle multi-sheet
    const _origConfirmCatalog = confirmCatalogSelection;
    confirmCatalogSelection = function() {
      if (!catalogTempDir || !catalogTempDb) return;
      if (editingSheetName) {
        // Multi-sheet mode: save to sheetTargets
        const t = sheetTargets[editingSheetName];
        t.dir = catalogTempDir;
        t.db = catalogTempDb;
        t.table = catalogTempTable;
        editingSheetName = null;
        closeCatalogModal();
        const set = selectedSheets.size >= 2 ? selectedSheets : localSelectedSheets;
        renderMultiSheetTargets(set);
        checkShowBottomSection();
      } else {
        // Single target mode: original behavior
        catalogConfirmedDir = catalogTempDir;
        catalogConfirmedDb = catalogTempDb;
        catalogConfirmedTable = catalogTempTable;
        const trigger = document.getElementById('catalogTriggerText');
        trigger.className = 'trigger-value';
        const parts = ['⊙' + catalogConfirmedDir, '⊙' + catalogConfirmedDb];
        if (catalogConfirmedTable) parts.push(catalogConfirmedTable);
        trigger.textContent = parts.join(' / ');
        closeCatalogModal();
        checkShowBottomSection();
      }
    };

    // === Local upload simulation ===
    document.addEventListener('DOMContentLoaded', function() {
      const uploadArea = document.getElementById('stLocalUploadArea');
      if (uploadArea) {
        uploadArea.addEventListener('click', function() {
          simulateLocalUpload();
        });
      }
    });

    function simulateLocalUpload() {
      // Simulate picking a random file
      const mockLocalFiles = [
        { name: '季度销售报表.xlsx', size: '1.8 MB', ext: 'xlsx' },
        { name: '用户数据导出.csv', size: '3.2 MB', ext: 'csv' },
        { name: '库存清单.xls', size: '920 KB', ext: 'xls' },
      ];
      const file = mockLocalFiles[Math.floor(Math.random() * mockLocalFiles.length)];
      localSelectedFile = file;
      localSelectedSheets = new Set();
      sheetTargets = {};
      isMultiSheetMode = false;
      // Show file info
      document.getElementById('stLocalFileName').textContent = file.name;
      document.getElementById('stLocalFileSize').textContent = file.size;
      document.getElementById('stLocalFileInfo').style.display = 'block';
      document.getElementById('stLocalUploadArea').style.display = 'none';
      // Check Excel
      if (isExcelExt(file.ext)) {
        const sheets = localMockExcelSheets[file.ext] || [];
        renderExcelSheets(sheets, 'localExcelSheetList');
        document.getElementById('localExcelSheetSection').style.display = 'block';
      } else {
        document.getElementById('localExcelSheetSection').style.display = 'none';
      }
      // Reset multi-sheet target
      document.getElementById('singleTargetSection').style.display = 'block';
      document.getElementById('multiSheetTargetSection').style.display = 'none';
      updateCsvConfigBtnVisibility();
      checkShowBottomSection();
    }

    function clearLocalFile() {
      localSelectedFile = null;
      localSelectedSheets = new Set();
      sheetTargets = {};
      isMultiSheetMode = false;
      document.getElementById('stLocalFileInfo').style.display = 'none';
      document.getElementById('stLocalUploadArea').style.display = 'block';
      document.getElementById('localExcelSheetSection').style.display = 'none';
      document.getElementById('singleTargetSection').style.display = 'block';
      document.getElementById('multiSheetTargetSection').style.display = 'none';
      updateCsvConfigBtnVisibility();
      checkShowBottomSection();
    }

    // === Override isDataSourceReady for file connectors ===
    const _origIsDataSourceReady = isDataSourceReady;
    isDataSourceReady = function() {
      const isLocal = document.getElementById('stLocal').style.display !== 'none';
      if (isLocal) {
        if (!localSelectedFile) return false;
        if (isExcelExt(localSelectedFile.ext) && localSelectedSheets.size === 0) return false;
        return true;
      }
      const sel = document.getElementById('stConnectorSelect');
      if (!sel.value) return false;
      const opt = sel.options[sel.selectedIndex];
      const isDb = opt && opt.getAttribute('data-type') === 'db';
      if (isDb) {
        return !!document.getElementById('stDbNameSelect').value && !!document.getElementById('stDbTableSelect').value;
      }
      // File connector
      if (!selectedFile) return false;
      if (isExcelExt(selectedFile.ext) && selectedSheets.size === 0) return false;
      return true;
    };

    // === Override isCatalogReady for multi-sheet ===
    const _origIsCatalogReady = isCatalogReady;
    isCatalogReady = function() {
      if (isMultiSheetMode) {
        const set = selectedSheets.size >= 2 ? selectedSheets : localSelectedSheets;
        for (const name of set) {
          const t = sheetTargets[name];
          if (!t || !t.dir || !t.db) return false;
        }
        return true;
      }
      return !!catalogConfirmedDir && !!catalogConfirmedDb;
    };

    // === Override checkShowBottomSection — unified bottom form ===
    const _origCheckShow = checkShowBottomSection;
    checkShowBottomSection = function() {
      const ready = isDataSourceReady() && isCatalogReady();
      const card = document.getElementById('bottomFormCard');
      const tabs = document.getElementById('bottomSheetTabs');
      const defContent = document.getElementById('tableDefContent');
      const mapContent = document.getElementById('colMappingContent');

      if (!ready) {
        card.style.display = 'none';
        const lir0 = document.getElementById('stLocalInitRule');
        if (lir0) lir0.style.display = 'none';
        return;
      }

      card.style.display = '';

      if (isMultiSheetMode) {
        // Multi-sheet: show all sheet tabs, switch content per sheet's mode
        const set = selectedSheets.size >= 2 ? selectedSheets : localSelectedSheets;
        const sheets = Array.from(set);
        if (!activeSheetTab || !sheets.includes(activeSheetTab)) {
          activeSheetTab = sheets[0];
        }
        tabs.style.display = '';
        tabs.innerHTML = sheets.map(s =>
          `<div class="sheet-tab${s === activeSheetTab ? ' active' : ''}" onclick="switchSheetTab('${s}')">${s}</div>`
        ).join('');
        // Show content based on active sheet's mode
        const activeMode = sheetTargets[activeSheetTab]?.mode || 'existing';
        showBottomContent(activeMode === 'new');
        // File-only settings always visible for file imports in multi-sheet
        document.getElementById('fileOnlyTableSettings').style.display = 'block';
        document.getElementById('fileOnlyMappingSettings').style.display = 'block';
        // Hide 初次载入规则 in multi-sheet mode (每个 Sheet 内单独有)
        const initRule = document.getElementById('stInitLoadRuleDiv');
        if (initRule) initRule.style.display = 'none';
        const lirM = document.getElementById('stLocalInitRule');
        if (lirM) lirM.style.display = 'none';
      } else {
        // Single mode: no tabs, content based on targetMode radio
        tabs.style.display = 'none';
        tabs.innerHTML = '';
        activeSheetTab = null;
        const mode = document.querySelector('input[name="targetMode"]:checked')?.value || 'existing';
        const isNew = mode === 'new';
        showBottomContent(isNew);
        // File-only settings visibility
        const isLocal = document.getElementById('stLocal').style.display !== 'none';
        const sel = document.getElementById('stConnectorSelect');
        const opt = sel.options[sel.selectedIndex];
        const isDb = opt && opt.getAttribute('data-type') === 'db';
        document.getElementById('fileOnlyTableSettings').style.display = (!isLocal && isDb) ? 'none' : 'block';
        // 本地上传单文件（CSV / 单 Sheet Excel）→ 已有表：显示初次载入规则
        const lirS = document.getElementById('stLocalInitRule');
        if (lirS) lirS.style.display = (isLocal && !isNew) ? '' : 'none';
      }
    };

    // === Sheet tab switching (unified card) ===
    function switchSheetTab(sheetName) {
      activeSheetTab = sheetName;
      document.querySelectorAll('#bottomSheetTabs .sheet-tab').forEach(t =>
        t.classList.toggle('active', t.textContent === sheetName)
      );
      const mode = sheetTargets[sheetName]?.mode || 'existing';
      showBottomContent(mode === 'new');
    }

    // === Reset file state on connector change ===
    const _origOnConnChange = onStructuredConnectorChange;
    onStructuredConnectorChange = function() {
      // Reset file browser state
      selectedFile = null;
      selectedSheets = new Set();
      sheetTargets = {};
      isMultiSheetMode = false;
      fileBrowserTempSelection = null;
      csvConfigConfirmed = false;
      csvConfig = { separator: ',', delimiter: '"', escape: false };
      const trigger = document.getElementById('fileBrowserText');
      trigger.className = 'trigger-placeholder';
      trigger.textContent = '请选择文件（支持 csv、xls、xlsx）';
      document.getElementById('excelSheetSection').style.display = 'none';
      document.getElementById('singleTargetSection').style.display = 'block';
      document.getElementById('multiSheetTargetSection').style.display = 'none';
      // Call original
      _origOnConnChange();
    };

    // === Reset on source tab switch ===
    const _origSelectStructuredSourceTab = selectStructuredSourceTab;
    selectStructuredSourceTab = function(el, id) {
      // Reset all file/sheet state
      selectedFile = null;
      selectedSheets = new Set();
      localSelectedFile = null;
      localSelectedSheets = new Set();
      sheetTargets = {};
      isMultiSheetMode = false;
      activeSheetTab = null;
      csvConfigConfirmed = false;
      csvConfig = { separator: ',', delimiter: '"', escape: false };
      // Reset UI
      const trigger = document.getElementById('fileBrowserText');
      trigger.className = 'trigger-placeholder';
      trigger.textContent = '请选择文件（支持 csv、xls、xlsx）';
      document.getElementById('excelSheetSection').style.display = 'none';
      document.getElementById('localExcelSheetSection').style.display = 'none';
      document.getElementById('singleTargetSection').style.display = 'block';
      document.getElementById('multiSheetTargetSection').style.display = 'none';
      if (id === 'stLocal') {
        document.getElementById('stLocalUploadArea').style.display = 'block';
        document.getElementById('stLocalFileInfo').style.display = 'none';
      }
      _origSelectStructuredSourceTab(el, id);
    };

    // === Unstructured File Browser ===
    const usFileSystem = {
      '/': [
        { name: '产品文档', type: 'folder' },
        { name: '技术手册', type: 'folder' },
        { name: '培训资料', type: 'folder' },
        { name: '项目报告_2024Q4.pdf', type: 'file', size: '23.94 MB' },
        { name: '系统架构说明.docx', type: 'file', size: '5.12 MB' },
        { name: '数据字典_v3.xlsx', type: 'file', size: '1.87 MB' },
        { name: '会议纪要_0315.pdf', type: 'file', size: '892 KB' },
        { name: '部署指南.md', type: 'file', size: '156 KB' },
        { name: 'logo_final.png', type: 'file', size: '3.41 MB' },
        { name: '演示视频.mp4', type: 'file', size: '128.5 MB' }
      ],
      '/产品文档': [
        { name: '用户手册_v2.1.pdf', type: 'file', size: '18.32 MB' },
        { name: 'API文档.pdf', type: 'file', size: '7.65 MB' },
        { name: '发版说明', type: 'folder' },
        { name: '产品需求规格书.docx', type: 'file', size: '4.23 MB' },
        { name: '功能对比表.xlsx', type: 'file', size: '320 KB' },
        { name: '产品截图', type: 'folder' }
      ],
      '/产品文档/发版说明': [
        { name: 'v3.0_发版说明.pdf', type: 'file', size: '1.2 MB' },
        { name: 'v2.5_发版说明.pdf', type: 'file', size: '980 KB' },
        { name: 'v2.0_发版说明.pdf', type: 'file', size: '856 KB' }
      ],
      '/产品文档/产品截图': [
        { name: '首页截图.png', type: 'file', size: '2.1 MB' },
        { name: '仪表盘截图.png', type: 'file', size: '1.8 MB' }
      ],
      '/技术手册': [
        { name: '运维手册.pdf', type: 'file', size: '12.4 MB' },
        { name: '安装部署指南.pdf', type: 'file', size: '6.78 MB' },
        { name: '故障排查手册.docx', type: 'file', size: '3.45 MB' },
        { name: '性能调优指南.pdf', type: 'file', size: '8.92 MB' }
      ],
      '/培训资料': [
        { name: '新员工培训.pptx', type: 'file', size: '45.6 MB' },
        { name: '高级功能培训.pptx', type: 'file', size: '32.1 MB' },
        { name: '培训视频', type: 'folder' },
        { name: '练习数据.zip', type: 'file', size: '67.8 MB' }
      ],
      '/培训资料/培训视频': [
        { name: '入门教程.mp4', type: 'file', size: '256 MB' },
        { name: '进阶教程.mp4', type: 'file', size: '198 MB' }
      ]
    };
    const usMailFileSystems = {
      'gmail-mail': {
        '/': [
          { name: '收件箱', type: 'folder' },
          { name: '客户反馈', type: 'folder' },
          { name: '已发送', type: 'folder' }
        ],
        '/收件箱': [
          { name: '客户 A：合同条款确认', type: 'mail', sender: '公开演示环境', date: '2026-05-24 09:18' },
          { name: '客户 B：接口授权问题', type: 'mail', sender: '公开演示环境', date: '2026-05-23 16:42' },
          { name: '供应商报价单补充说明', type: 'mail', sender: '公开演示环境', date: '2026-05-22 11:05' }
        ],
        '/客户反馈': [
          { name: '知识库回答不完整的反馈', type: 'mail', sender: '公开演示环境', date: '2026-05-21 14:30' },
          { name: '多附件上传失败记录', type: 'mail', sender: '公开演示环境', date: '2026-05-20 18:16' }
        ],
        '/已发送': [
          { name: 'Re: 客户 A：合同条款确认', type: 'mail', sender: '公开演示环境', date: '2026-05-24 10:02' }
        ]
      },
      'outlook-mail': {
        '/': [
          { name: '收件箱', type: 'folder' },
          { name: '法务审阅', type: 'folder' },
          { name: '已发送', type: 'folder' }
        ],
        '/收件箱': [
          { name: 'NDA 审批：甲方合同文本', type: 'mail', sender: '公开演示环境', date: '2026-05-24 13:12' },
          { name: '采购协议变更说明', type: 'mail', sender: '公开演示环境', date: '2026-05-23 15:47' }
        ],
        '/法务审阅': [
          { name: '服务协议 v3.1 红线版', type: 'mail', sender: '公开演示环境', date: '2026-05-22 19:20' },
          { name: '数据处理附录 DPA 确认', type: 'mail', sender: '公开演示环境', date: '2026-05-21 08:56' }
        ],
        '/已发送': [
          { name: 'Re: NDA 审批：甲方合同文本', type: 'mail', sender: '公开演示环境', date: '2026-05-24 14:05' }
        ]
      },
      'wecom-mail': {
        '/': [
          { name: '收件箱', type: 'folder' },
          { name: '项目往来', type: 'folder' },
          { name: '已发送', type: 'folder' }
        ],
        '/收件箱': [
          { name: '上勘院信息提取需求补充', type: 'mail', sender: '公开演示环境', date: '2026-05-24 08:35' },
          { name: 'POC 数据口径确认', type: 'mail', sender: '公开演示环境', date: '2026-05-23 17:12' }
        ],
        '/项目往来': [
          { name: '会议纪要：邮件解析与回写边界', type: 'mail', sender: '公开演示环境', date: '2026-05-22 20:10' },
          { name: '客户原始邮件样本交付', type: 'mail', sender: '公开演示环境', date: '2026-05-21 11:40' }
        ],
        '/已发送': [
          { name: 'Re: POC 数据口径确认', type: 'mail', sender: '公开演示环境', date: '2026-05-23 18:03' }
        ]
      },
      'qq-mail': {
        '/': [
          { name: '收件箱', type: 'folder' },
          { name: '运营通知', type: 'folder' },
          { name: '已发送', type: 'folder' }
        ],
        '/收件箱': [
          { name: '活动报名名单汇总', type: 'mail', sender: '公开演示环境', date: '2026-05-24 12:20' },
          { name: '渠道合作资料', type: 'mail', sender: '公开演示环境', date: '2026-05-23 09:44' }
        ],
        '/运营通知': [
          { name: '5 月用户反馈周报', type: 'mail', sender: '公开演示环境', date: '2026-05-22 17:30' },
          { name: '线上活动素材确认', type: 'mail', sender: '公开演示环境', date: '2026-05-21 10:12' }
        ],
        '/已发送': [
          { name: 'Re: 线上活动素材确认', type: 'mail', sender: '公开演示环境', date: '2026-05-21 10:50' }
        ]
      }
    };
    const savedConnectorStorageKey = 'moi.connector.saved.v1';
    const usMailSourceLabels = {
      'gmail': 'Gmail',
      'outlook': 'Outlook',
      'wecom-mail': '企业微信邮箱',
      'qq-mail': 'QQ 邮箱',
      'imap-smtp': 'IMAP/SMTP',
      'custom-mail-api': '自定义邮件 API'
    };
    let usSavedMailConnectorsByValue = {};
    let usRemoteMailFileSystems = {};
    let usRemoteMailLoading = {};
    let usRemoteMailErrors = {};

    function usReadSavedConnectors() {
      try {
        var raw = localStorage.getItem(savedConnectorStorageKey);
        var connectors = raw ? JSON.parse(raw) : [];
        return Array.isArray(connectors) ? connectors : [];
      } catch(e) {
        return [];
      }
    }

    function usGetSavedMailValue(connector) {
      return 'saved-mail-' + connector.id;
    }

    function loadSavedUnstructuredConnectors() {
      var select = document.getElementById('usConnectorSelect');
      if (!select) return;
      Array.from(select.querySelectorAll('optgroup[data-saved-connectors="true"]')).forEach(function(group) {
        group.remove();
      });
      usSavedMailConnectorsByValue = {};

      var savedMailConnectors = usReadSavedConnectors().filter(function(connector) {
        return connector
          && connector.type === 'mail'
          && connector.usage
          && connector.usage.import
          && connector.id;
      });
      if (savedMailConnectors.length === 0) return;

      var group = document.createElement('optgroup');
      group.label = '已创建连接器';
      group.setAttribute('data-saved-connectors', 'true');
      savedMailConnectors.forEach(function(connector) {
        var value = usGetSavedMailValue(connector);
        usSavedMailConnectorsByValue[value] = connector;
        if (!usRemoteMailFileSystems[value]) usRemoteMailFileSystems[value] = {};
        var option = document.createElement('option');
        option.value = value;
        option.textContent = connector.name + '（' + (usMailSourceLabels[connector.source] || '邮箱') + '）';
        option.setAttribute('data-kind', 'mail');
        option.setAttribute('data-saved-id', connector.id);
        option.setAttribute('data-source', connector.source || '');
        group.appendChild(option);
      });
      select.appendChild(group);
    }

    let usCurrentPath = '/';
    let usSelectedItems = new Set(); // stores full paths of leaf files only

    function usGetConnectorValue() {
      var sel = document.getElementById('usConnectorSelect');
      return sel ? sel.value : '';
    }

    function usGetSavedMailConnector() {
      return usSavedMailConnectorsByValue[usGetConnectorValue()] || null;
    }

    function usIsRemoteMailConnector() {
      return !!usGetSavedMailConnector();
    }

    function usIsMailConnector() {
      var value = usGetConnectorValue();
      return !!usMailFileSystems[value] || !!usSavedMailConnectorsByValue[value];
    }

    function usGetActiveFileSystem() {
      // 编辑模式：优先用任务自带的 files 列表（来自共享数据源 IMPORT_TASKS_DISPLAY），
      // 让编辑页看到的文件清单与详情/Catalog 卷下文件保持一致
      if (editMode && editTaskId && window.IMPORT_TASKS_DISPLAY) {
        var task = window.IMPORT_TASKS_DISPLAY[editTaskId];
        if (task && task.files && task.files.length) {
          return {
            '/': task.files.map(function(f) {
              return { name: f.name, type: 'file', size: f.size || '—', mtime: f.mtime || '' };
            })
          };
        }
      }
      var value = usGetConnectorValue();
      if (usSavedMailConnectorsByValue[value]) return usRemoteMailFileSystems[value] || {};
      return usMailFileSystems[value] || usFileSystem;
    }

    function usResetBrowserForConnector() {
      usCurrentPath = '/';
      usSelectedItems.clear();
      if (usIsRemoteMailConnector()) usLoadRemoteMailPath('/');
      else usRenderFileBrowser();
    }

    function usUpdateBrowserLabels() {
      const isMail = usIsMailConnector();
      const title = document.getElementById('usFileBrowserTitle');
      const nameHead = document.getElementById('usFileNameHeader');
      const metaHead = document.getElementById('usFileMetaHeader');
      const typeHead = document.getElementById('usFileTypeHeader');
      if (title) title.textContent = isMail ? '邮件选择' : '文件选择';
      if (nameHead) nameHead.textContent = isMail ? '文件夹 / 邮件内容' : '文件名';
      if (metaHead) metaHead.textContent = isMail ? '时间' : '文件大小';
      if (typeHead) typeHead.textContent = isMail ? '发件人' : '文件类型';
    }

    function usEscapeHtml(value) {
      return String(value || '').replace(/[&<>"']/g, function(ch) {
        return ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[ch];
      });
    }

    function usEscapeJsString(value) {
      return String(value || '').replace(/\\/g, "\\\\").replace(/'/g, "\\'");
    }

    function usGetMailRangeValue() {
      var sel = document.getElementById('usMailTimeRange');
      return sel ? sel.value : '30d';
    }

    async function usLoadRemoteMailPath(path) {
      var value = usGetConnectorValue();
      var connector = usSavedMailConnectorsByValue[value];
      if (!connector) {
        usRenderFileBrowser();
        return;
      }
      var loadKey = value + '|' + path;
      usRemoteMailLoading[loadKey] = true;
      delete usRemoteMailErrors[loadKey];
      usRenderFileBrowser();
      try {
        var resp = await fetch('/api/connector/mail/list', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            source: connector.source,
            config: connector.config || {},
            usage: connector.usage || {},
            path: path,
            range: usGetMailRangeValue()
          })
        });
        var raw = await resp.text();
        var data = {};
        try { data = raw ? JSON.parse(raw) : {}; } catch(parseError) {}
        if (!resp.ok || !data.ok) {
          var message = data.message || '';
          if (!message && resp.status === 404) message = '读取接口不存在，当前本地服务可能还没有重启';
          if (!message) message = raw ? raw.slice(0, 180) : '读取邮箱数据失败';
          throw new Error((resp.ok ? '' : 'HTTP ' + resp.status + '：') + message);
        }
        if (!usRemoteMailFileSystems[value]) usRemoteMailFileSystems[value] = {};
        usRemoteMailFileSystems[value][path] = Array.isArray(data.items) ? data.items : [];
      } catch(e) {
        usRemoteMailErrors[loadKey] = e.message || '读取邮箱数据失败';
        if (!usRemoteMailFileSystems[value]) usRemoteMailFileSystems[value] = {};
        usRemoteMailFileSystems[value][path] = [];
      } finally {
        usRemoteMailLoading[loadKey] = false;
        if (usGetConnectorValue() === value && usCurrentPath === path) usRenderFileBrowser();
      }
    }

    // Get all descendant leaf files under a folder path (recursive)
    function usGetAllDescendants(folderPath) {
      const result = [];
      const children = usGetActiveFileSystem()[folderPath] || [];
      children.forEach(item => {
        const childPath = (folderPath === '/' ? '/' : folderPath + '/') + item.name;
        if (item.type === 'folder') {
          result.push(...usGetAllDescendants(childPath));
        } else {
          result.push(childPath);
        }
      });
      return result;
    }

    // Get selection state for a folder: 'all', 'some', 'none'
    function usFolderSelectionState(folderPath) {
      const descendants = usGetAllDescendants(folderPath);
      if (descendants.length === 0) return 'none';
      const selectedCount = descendants.filter(d => usSelectedItems.has(d)).length;
      if (selectedCount === 0) return 'none';
      if (selectedCount === descendants.length) return 'all';
      return 'some';
    }

    function usBuildItemKey(basePath, item) {
      if (item.path) return item.path;
      const prefix = basePath === '/' ? '/' : basePath + '/';
      if (item.type !== 'folder' && item.id) return prefix + '#' + item.id;
      return prefix + item.name;
    }

    function usRenderFileBrowser() {
      const activeFileSystem = usGetActiveFileSystem();
      const isMail = usIsMailConnector();
      const items = activeFileSystem[usCurrentPath] || [];
      const tbody = document.getElementById('usFileTableBody');
      const selectAll = document.getElementById('usFileSelectAll');
      usUpdateBrowserLabels();

      // Breadcrumb
      const bc = document.getElementById('usFileBreadcrumb');
      const parts = usCurrentPath === '/' ? [''] : usCurrentPath.split('/');
      let bcHtml = `<span style="cursor:pointer;color:#1677ff" onclick="usNavigateTo('/')">${isMail ? '全部邮件' : '全部文件'}</span>`;
      let accumulated = '';
      for (let i = 1; i < parts.length; i++) {
        accumulated += '/' + parts[i];
        const p = accumulated;
        bcHtml += `<span style="color:rgba(0,0,0,0.25)">/</span>`;
        if (i === parts.length - 1) {
          bcHtml += `<span style="color:rgba(0,0,0,0.88)">${usEscapeHtml(parts[i])}</span>`;
        } else {
          bcHtml += `<span style="cursor:pointer;color:#1677ff" onclick="usNavigateTo('${usEscapeJsString(p)}')">${usEscapeHtml(parts[i])}</span>`;
        }
      }
      bc.innerHTML = bcHtml;

      const loadKey = usGetConnectorValue() + '|' + usCurrentPath;
      if (isMail && usIsRemoteMailConnector() && usRemoteMailLoading[loadKey]) {
        tbody.innerHTML = `<tr>
          <td colspan="4" style="padding:18px 12px;text-align:center;color:rgba(0,0,0,0.55);border-bottom:1px solid #f5f5f5">正在读取真实邮箱数据...</td>
        </tr>`;
        if (selectAll) {
          selectAll.checked = false;
          selectAll.indeterminate = false;
          selectAll.disabled = true;
        }
        usUpdateSelectedCount();
        return;
      }

      if (isMail && usIsRemoteMailConnector() && usRemoteMailErrors[loadKey]) {
        tbody.innerHTML = `<tr>
          <td colspan="4" style="padding:18px 12px;text-align:left;color:#cf1322;border-bottom:1px solid #f5f5f5">
            真实邮箱数据读取失败：${usEscapeHtml(usRemoteMailErrors[loadKey])}
          </td>
        </tr>`;
        if (selectAll) {
          selectAll.checked = false;
          selectAll.indeterminate = false;
          selectAll.disabled = true;
        }
        usUpdateSelectedCount();
        return;
      }

      if (isMail && usIsRemoteMailConnector() && items.length === 0) {
        tbody.innerHTML = `<tr>
          <td colspan="4" style="padding:18px 12px;text-align:center;color:rgba(0,0,0,0.45);border-bottom:1px solid #f5f5f5">当前范围内没有读取到邮件数据</td>
        </tr>`;
        if (selectAll) {
          selectAll.checked = false;
          selectAll.indeterminate = false;
          selectAll.disabled = true;
        }
        usUpdateSelectedCount();
        return;
      }

      if (selectAll) selectAll.disabled = false;

      // Table rows
      const folderIcon = `<svg width="16" height="16" viewBox="0 0 24 24" fill="#faad14" stroke="#faad14" stroke-width="1.5"><path d="M4 20h16a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.93a2 2 0 0 1-1.66-.9l-.82-1.2A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13c0 1.1.9 2 2 2z"/></svg>`;
      const fileIcon = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="rgba(0,0,0,0.45)" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>`;
      const mailIcon = `<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="rgba(0,74,240,0.72)" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="5" width="18" height="14" rx="2"/><path d="M3 7l9 6 9-6"/></svg>`;

      tbody.innerHTML = items.map((item, idx) => {
        const isFolder = item.type === 'folder';
        const itemKey = usBuildItemKey(usCurrentPath, item);
        const isRemoteFolder = isMail && usIsRemoteMailConnector() && isFolder;
        let isChecked = false;
        let isIndeterminate = false;

        if (isFolder) {
          const state = usFolderSelectionState(itemKey);
          isChecked = state === 'all';
          isIndeterminate = state === 'some';
        } else {
          isChecked = usSelectedItems.has(itemKey);
        }

        const highlighted = isChecked || isIndeterminate;
        const ext = isFolder ? '' : (item.name.split('.').pop() || '').toUpperCase();
        const folderCount = isFolder ? (typeof item.count === 'number' ? item.count : usGetAllDescendants(itemKey).length) : 0;
        const metaText = isMail ? (isFolder ? `${folderCount} 封` : (item.date || '-')) : (isFolder ? '-' : item.size);
        const typeText = isMail
          ? (isFolder ? '邮件文件夹' : (item.sender || '-'))
          : (isFolder ? '文件夹' : (ext ? `<span style="font-size:11px;padding:1px 6px;background:#f5f5f5;border-radius:3px">${ext}</span>` : '-'));
        const escapedKey = usEscapeJsString(itemKey);
        const rowAction = isRemoteFolder ? `usNavigateInto('${escapedKey}')` : `usToggleItem('${escapedKey}', ${isFolder})`;

        return `<tr style="cursor:pointer;${highlighted ? 'background:#e6f4ff' : ''}" onclick="${rowAction}">
          <td style="width:36px;padding:9px 10px;text-align:center;border-bottom:1px solid #f5f5f5">
            <input type="checkbox" id="usCb${idx}" ${isChecked ? 'checked' : ''} ${isRemoteFolder ? 'disabled' : ''} onclick="event.stopPropagation();usToggleItem('${escapedKey}', ${isFolder})" style="accent-color:#1677ff;cursor:pointer">
          </td>
          <td style="padding:9px 10px;border-bottom:1px solid #f5f5f5">
            <span style="display:flex;align-items:center;gap:8px">
              ${isFolder ? folderIcon : (isMail ? mailIcon : fileIcon)}
              ${isFolder
                ? `<span style="color:#1677ff;cursor:pointer" onclick="event.stopPropagation();usNavigateInto('${escapedKey}')">${usEscapeHtml(item.name)}</span>`
                : `<span>${usEscapeHtml(item.name)}</span>`
              }
            </span>
          </td>
          <td style="width:120px;padding:9px 10px;text-align:left;border-bottom:1px solid #f5f5f5;color:rgba(0,0,0,0.65)">${usEscapeHtml(metaText)}</td>
          <td style="width:140px;padding:9px 10px;text-align:left;border-bottom:1px solid #f5f5f5;color:rgba(0,0,0,0.45)">${isMail ? usEscapeHtml(typeText) : typeText}</td>
        </tr>`;
      }).join('');

      // Set indeterminate state for folder checkboxes (must be done after DOM render)
      requestAnimationFrame(() => {
        items.forEach((item, idx) => {
          if (item.type === 'folder') {
            const cb = document.getElementById('usCb' + idx);
            if (cb) {
              const itemKey = usBuildItemKey(usCurrentPath, item);
              const state = usFolderSelectionState(itemKey);
              cb.indeterminate = state === 'some';
            }
          }
        });
      });

      // Select-all state: check all items in current view
      let allChecked = items.length > 0;
      let someChecked = false;
      items.forEach(item => {
        const itemKey = usBuildItemKey(usCurrentPath, item);
        if (item.type === 'folder') {
          const state = usFolderSelectionState(itemKey);
          if (state !== 'all') allChecked = false;
          if (state !== 'none') someChecked = true;
        } else {
          if (!usSelectedItems.has(itemKey)) allChecked = false;
          if (usSelectedItems.has(itemKey)) someChecked = true;
        }
      });
      selectAll.checked = allChecked;
      selectAll.indeterminate = someChecked && !allChecked;

      usUpdateSelectedCount();
    }

    function usToggleItem(itemKey, isFolder) {
      if (isFolder) {
        const state = usFolderSelectionState(itemKey);
        const descendants = usGetAllDescendants(itemKey);
        if (state === 'all') {
          // Uncheck all descendants
          descendants.forEach(d => usSelectedItems.delete(d));
        } else {
          // Check all descendants (covers 'none' and 'some')
          descendants.forEach(d => usSelectedItems.add(d));
        }
      } else {
        if (usSelectedItems.has(itemKey)) {
          usSelectedItems.delete(itemKey);
        } else {
          usSelectedItems.add(itemKey);
        }
      }
      usRenderFileBrowser();
    }

    function usToggleSelectAll(checked) {
      const items = usGetActiveFileSystem()[usCurrentPath] || [];
      items.forEach(item => {
        const key = usBuildItemKey(usCurrentPath, item);
        if (item.type === 'folder') {
          const descendants = usGetAllDescendants(key);
          descendants.forEach(d => {
            if (checked) usSelectedItems.add(d);
            else usSelectedItems.delete(d);
          });
        } else {
          if (checked) usSelectedItems.add(key);
          else usSelectedItems.delete(key);
        }
      });
      usRenderFileBrowser();
    }

    function usNavigateTo(path) {
      usCurrentPath = path;
      if (usIsRemoteMailConnector() && !usGetActiveFileSystem()[path]) usLoadRemoteMailPath(path);
      else usRenderFileBrowser();
    }

    function usNavigateInto(folderPath) {
      usCurrentPath = folderPath;
      if (usIsRemoteMailConnector() && !usGetActiveFileSystem()[folderPath]) usLoadRemoteMailPath(folderPath);
      else if (usGetActiveFileSystem()[folderPath]) usRenderFileBrowser();
    }

    function usUpdateSelectedCount() {
      const el = document.getElementById('usFileSelectedCount');
      if (usSelectedItems.size === 0) {
        el.textContent = '';
        return;
      }
      // Count unique selected files and folders at top level of selection
      // For display, count actual leaf files and distinct folder paths
      let fileCount = 0, folderPaths = new Set();
      usSelectedItems.forEach(path => {
        fileCount++;
        // Track which top-level folders contain selections
      });
      el.textContent = usIsMailConnector() ? `已选 ${fileCount} 封邮件` : `已选 ${fileCount} 个文件`;
    }

    // Initialize connector dropdown and file browser on page load
    loadSavedUnstructuredConnectors();
    usRenderFileBrowser();

    // === Unstructured Catalog Modal (目录→库→卷) ===
    const usCatalogData = {
      '默认': {
        '原始数据库': ['文档卷', '图片卷', '音视频卷'],
        '处理数据库': ['处理结果卷']
      }
    };
    const catIconVol = '<svg viewBox="0 0 14 14" fill="none" stroke="#722ed1" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"><path d="M4 12h6a2 2 0 0 0 2-2V5.5L8.5 2H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2z"/><path d="M8.5 2v3.5H12"/></svg>';
    let usTempDir = null, usTempDb = null, usTempVol = null;
    let usConfirmedDir = null, usConfirmedDb = null, usConfirmedVol = null;

    function openUsCatalogModal() {
      usTempDir = usConfirmedDir;
      usTempDb = usConfirmedDb;
      usTempVol = usConfirmedVol;
      renderUsCatalogModal();
      document.getElementById('usCatalogModal').classList.add('open');
    }
    function closeUsCatalogModal() {
      document.getElementById('usCatalogModal').classList.remove('open');
    }
    function renderUsCatalogModal() {
      const dirList = document.getElementById('usCatalogDirList');
      dirList.innerHTML = Object.keys(usCatalogData).map(d =>
        `<div class="cat-item${usTempDir === d ? ' active' : ''}" onclick="selectUsCatalogDir('${d}')">
          <span class="cat-icon">${catIconDir}</span><span>${d}</span><span class="cat-arrow">›</span></div>`
      ).join('');
      const dbList = document.getElementById('usCatalogDbList');
      if (usTempDir) {
        const dbs = Object.keys(usCatalogData[usTempDir] || {});
        dbList.innerHTML = dbs.length ? dbs.map(d =>
          `<div class="cat-item${usTempDb === d ? ' active' : ''}" onclick="selectUsCatalogDb('${d}')">
            <span class="cat-icon">${catIconDb}</span><span>${d}</span><span class="cat-arrow">›</span></div>`
        ).join('') : renderCatalogEmpty();
      } else { dbList.innerHTML = renderCatalogEmpty(); }
      const volList = document.getElementById('usCatalogVolList');
      if (usTempDir && usTempDb) {
        const vols = usCatalogData[usTempDir]?.[usTempDb] || [];
        volList.innerHTML = vols.length ? vols.map(v =>
          `<div class="cat-item${usTempVol === v ? ' active' : ''}" onclick="selectUsCatalogVol('${v}')">
            <span class="cat-icon">${catIconVol}</span><span>${v}</span></div>`
        ).join('') : renderCatalogEmpty();
      } else { volList.innerHTML = renderCatalogEmpty(); }
      updateUsConfirmBtn();
    }
    function selectUsCatalogDir(dir) { usTempDir = dir; usTempDb = null; usTempVol = null; renderUsCatalogModal(); }
    function selectUsCatalogDb(db) { usTempDb = db; usTempVol = null; renderUsCatalogModal(); }
    function selectUsCatalogVol(vol) { usTempVol = vol; renderUsCatalogModal(); }
    function updateUsConfirmBtn() {
      const btn = document.getElementById('usCatalogConfirmBtn');
      btn.classList.toggle('active', !!(usTempDir && usTempDb && usTempVol));
    }
    function confirmUsCatalogSelection() {
      if (!usTempDir || !usTempDb || !usTempVol) return;
      usConfirmedDir = usTempDir;
      usConfirmedDb = usTempDb;
      usConfirmedVol = usTempVol;
      const trigger = document.getElementById('usCatalogTriggerText');
      trigger.className = 'trigger-value';
      trigger.textContent = '⊙' + usConfirmedDir + ' / ⊙' + usConfirmedDb + ' / ' + usConfirmedVol;
      // Also update web mode catalog trigger
      const webTrigger = document.getElementById('usWebCatalogText');
      if (webTrigger) {
        webTrigger.style.color = 'rgba(0,0,0,0.88)';
        webTrigger.style.fontWeight = '500';
        webTrigger.textContent = usConfirmedDir + ' / ' + usConfirmedDb + ' / ' + usConfirmedVol;
      }
      // Show web config panel after catalog is selected in web mode
      var webPanel = document.getElementById('usWebImportPanel');
      var webConfigCard = document.getElementById('webConfigCard');
      var webCatalogGroup = document.getElementById('usWebCatalogGroup');
      if (webPanel && webCatalogGroup && webCatalogGroup.style.display !== 'none') {
        webPanel.style.display = '';
        if (webConfigCard) webConfigCard.style.display = '';
      }
      closeUsCatalogModal();
    }

    // === CSV Config ===
    let csvConfig = { separator: ',', delimiter: '"', escape: false };
    let csvConfigConfirmed = false;

    function isCsvFile() {
      const isLocal = document.getElementById('stLocal').style.display !== 'none';
      if (isLocal) return localSelectedFile && localSelectedFile.ext === 'csv';
      return selectedFile && selectedFile.ext === 'csv';
    }

    function updateCsvConfigBtnVisibility() {
      const show = isCsvFile();
      const btnDef = document.getElementById('csvConfigBtnDef');
      const btnMap = document.getElementById('csvConfigBtnMap');
      if (btnDef) btnDef.style.display = show ? '' : 'none';
      if (btnMap) btnMap.style.display = show ? '' : 'none';
      if (show && csvConfigConfirmed) {
        if (btnDef) btnDef.classList.add('configured');
        if (btnMap) btnMap.classList.add('configured');
      } else {
        if (btnDef) btnDef.classList.remove('configured');
        if (btnMap) btnMap.classList.remove('configured');
      }
    }

    function openCsvConfigModal() {
      document.getElementById('csvSeparator').value = csvConfig.separator;
      document.getElementById('csvDelimiter').value = csvConfig.delimiter;
      document.getElementById('csvEscape').checked = csvConfig.escape;
      updateCsvPreview();
      document.getElementById('csvConfigModal').classList.add('open');
    }

    function closeCsvConfigModal() {
      document.getElementById('csvConfigModal').classList.remove('open');
    }

    function confirmCsvConfig() {
      csvConfig.separator = document.getElementById('csvSeparator').value;
      csvConfig.delimiter = document.getElementById('csvDelimiter').value;
      csvConfig.escape = document.getElementById('csvEscape').checked;
      csvConfigConfirmed = true;
      updateCsvConfigBtnVisibility();
      closeCsvConfigModal();
    }

    function updateCsvPreview() {
      const sep = document.getElementById('csvSeparator').value;
      const delim = document.getElementById('csvDelimiter').value;
      const esc = document.getElementById('csvEscape').checked;

      const sepDisplay = { ',': ',', ';': ';', '\t': '\\t', '|': '|', ' ': '␣' }[sep] || sep;
      const delimDisplay = delim || '';

      // Header line: comma-separated column names
      const headerCols = ['id', 'name', 'type', 'description', 'notes'];
      const headerText = headerCols.join(sepDisplay + ' ');

      // Data line — boxes around separator, delimiter, escape chars
      let dataHtml = '';
      if (delim && esc) {
        // 021 , "MOI" , " database " , " \" database \" rocks"
        dataHtml = `<span class="val">021</span>`
          + `<span class="sep" data-label="sep">${sepDisplay}</span>`
          + `<span class="delim">${delimDisplay}</span><span class="val">MOI</span><span class="delim">${delimDisplay}</span>`
          + `<span class="sep">${sepDisplay}</span>`
          + `<span class="delim" data-label="delim">${delimDisplay}</span><span class="val"> database </span><span class="delim">${delimDisplay}</span>`
          + `<span class="sep">${sepDisplay}</span>`
          + `<span class="delim">${delimDisplay}</span><span class="esc" data-label="esc">\\${delimDisplay}</span><span class="val"> database </span><span class="esc">\\${delimDisplay}</span><span class="val"> rocks${delimDisplay}</span>`;
      } else if (delim) {
        dataHtml = `<span class="val">021</span>`
          + `<span class="sep" data-label="sep">${sepDisplay}</span>`
          + `<span class="delim">${delimDisplay}</span><span class="val">MOI</span><span class="delim">${delimDisplay}</span>`
          + `<span class="sep">${sepDisplay}</span>`
          + `<span class="delim" data-label="delim">${delimDisplay}</span><span class="val"> database </span><span class="delim">${delimDisplay}</span>`
          + `<span class="sep">${sepDisplay}</span>`
          + `<span class="delim">${delimDisplay}</span><span class="val"> database  rocks</span><span class="delim">${delimDisplay}</span>`;
      } else {
        dataHtml = `<span class="val">021</span>`
          + `<span class="sep" data-label="sep">${sepDisplay}</span>`
          + `<span class="val">MOI</span>`
          + `<span class="sep">${sepDisplay}</span>`
          + `<span class="val">database</span>`
          + `<span class="sep">${sepDisplay}</span>`
          + `<span class="val">database rocks</span>`;
      }

      const area = document.getElementById('csvPreviewArea');
      area.innerHTML = `<div class="csv-preview-header">${headerText}</div>`
        + `<span class="csv-preview-header-label">header</span>`
        + `<div class="csv-preview-data" id="csvDataRow">${dataHtml}</div>`
        + `<div class="csv-preview-labels" id="csvLabelsRow"></div>`;

      // Position labels based on actual rendered element positions
      requestAnimationFrame(() => {
        const dataRow = document.getElementById('csvDataRow');
        const labelsRow = document.getElementById('csvLabelsRow');
        if (!dataRow || !labelsRow) return;
        const rowRect = dataRow.getBoundingClientRect();
        let labelsHtml = '';

        const sepEl = dataRow.querySelector('[data-label="sep"]');
        if (sepEl) {
          const r = sepEl.getBoundingClientRect();
          const cx = r.left + r.width / 2 - rowRect.left;
          labelsHtml += `<div class="csv-label-item" style="left:${cx}px"><div class="csv-label-line"></div><div class="csv-label-bracket">Separator</div></div>`;
        }
        const delimEl = dataRow.querySelector('[data-label="delim"]');
        if (delimEl) {
          const r = delimEl.getBoundingClientRect();
          const cx = r.left + r.width / 2 - rowRect.left;
          labelsHtml += `<div class="csv-label-item" style="left:${cx}px"><div class="csv-label-line"></div><div class="csv-label-bracket">Delimiter</div></div>`;
        }
        const escEl = dataRow.querySelector('[data-label="esc"]');
        if (escEl) {
          const r = escEl.getBoundingClientRect();
          const cx = r.left + r.width / 2 - rowRect.left;
          labelsHtml += `<div class="csv-label-item" style="left:${cx}px"><div class="csv-label-line"></div><div class="csv-label-bracket">Backslash Escape</div></div>`;
        }
        labelsRow.innerHTML = labelsHtml;
      });
    }

    function createImport() {
      // Check if web mode
      var webPanel = document.getElementById('usWebImportPanel');
      if (webPanel && webPanel.style.display !== 'none') {
        var webUrl = document.getElementById('webUrl').value.trim();
        if (!webUrl) { alert('请输入网页地址'); document.getElementById('webUrl').focus(); return; }
        var depth = document.getElementById('webDepth').value;
        var pager = document.getElementById('webAutoPager').value;
        var collectType = document.getElementById('webCollectType').value;
        var interval = document.getElementById('webReqInterval').value;
        var concurrency = document.getElementById('webConcurrency').value;
        var summary = '采集任务创建成功（模拟）\n\n';
        summary += '目标地址：' + webUrl + '\n';
        summary += '采集深度：' + depth + ' 层\n';
        summary += '自动翻页：' + (pager === 'off' ? '关闭' : pager === 'max' ? '最多 ' + document.getElementById('webMaxPages').value + ' 页' : '全部') + '\n';
        summary += '采集内容：' + {all:'全部',page:'仅页面',file:'仅文件'}[collectType] + '\n';
        summary += '请求间隔：' + interval + ' 秒，并发：' + concurrency + '\n';
        if (webRegions.length) summary += '选区数量：' + webRegions.length + ' 个';
        alert(summary);
        location.href = 'data-import.html';
        return;
      }
      alert('载入任务创建成功（模拟）');
      location.href = 'data-import.html';
    }

    // === Web Import Functions ===
    var webSelectMode = false;
    var webRegions = [];

    function onWebCollectTypeChange() {
      var val = document.getElementById('webCollectType').value;
      var fileField = document.getElementById('webFileTypeField');
      if (fileField) fileField.style.display = (val === 'page') ? 'none' : '';
    }

    function onWebZoomChange() {
      var zoom = parseFloat(document.getElementById('webPreviewZoom').value);
      var frame = document.getElementById('webPreviewFrame');
      if (!frame) return;
      var inv = 1 / zoom;
      frame.style.width = (inv * 100) + '%';
      frame.style.height = (inv * 100) + '%';
      frame.style.transform = 'scale(' + zoom + ')';
      frame.style.transformOrigin = '0 0';
    }

    function onWebAutoPagerChange() {
      var val = document.getElementById('webAutoPager').value;
      var maxField = document.getElementById('webMaxPagesField');
      if (maxField) maxField.style.display = (val === 'max') ? '' : 'none';
    }

    function applyWebTemplate(tpl) {
      if (tpl === 'szse') {
        document.getElementById('webUrl').value = 'https://www.szse.cn/disclosure/listed/notice/index.html';
        document.getElementById('webCollectType').value = 'file';
        onWebCollectTypeChange();
        loadWebPreview();
      } else if (tpl === 'sse') {
        document.getElementById('webUrl').value = 'https://www.sse.com.cn/disclosure/listedinfo/announcement/';
        document.getElementById('webCollectType').value = 'file';
        onWebCollectTypeChange();
        loadWebPreview();
      } else if (tpl === 'cninfo') {
        document.getElementById('webUrl').value = 'https://www.cninfo.com.cn/new/commonUrl?url=disclosure/list/notice';
        document.getElementById('webCollectType').value = 'all';
        onWebCollectTypeChange();
        loadWebPreview();
      }
    }

    function loadWebPreview() {
      var url = document.getElementById('webUrl').value.trim();
      if (!url) { alert('请输入网页地址'); return; }
      if (!url.startsWith('http')) url = 'https://' + url;
      document.getElementById('webPreviewUrl').textContent = url;
      document.getElementById('webPreviewArea').style.display = '';
      var body = document.getElementById('webPreviewBody');
      var frame = document.getElementById('webPreviewFrame');
      // Reset zoom
      var zoom = parseFloat(document.getElementById('webPreviewZoom').value) || 0.75;
      var inv = 1 / zoom;
      frame.style.width = (inv * 100) + '%';
      frame.style.height = (inv * 100) + '%';
      frame.style.transform = 'scale(' + zoom + ')';
      frame.style.transformOrigin = '0 0';
      // Remove src if previously set, use srcdoc for loading indicator
      frame.removeAttribute('src');
      frame.srcdoc = '<div style="display:flex;align-items:center;justify-content:center;height:100%;color:rgba(0,0,0,0.35);font-size:13px;font-family:-apple-system,sans-serif"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="animation:spin 1s linear infinite;margin-right:8px"><path d="M21 12a9 9 0 1 1-6.219-8.56"/></svg>正在加载页面...</div><style>@keyframes spin{to{transform:rotate(360deg)}}</style>';
      // Clear previous selections
      webRegions = [];
      renderWebRegions();
      // Remove previous warnings
      var existingWarn = body.parentElement.querySelector('.web-proxy-warn');
      if (existingWarn) existingWarn.remove();
      // Fetch via local proxy server (node html/scripts/proxy-server.js)
      var proxyUrl = 'http://localhost:3001/proxy?url=' + encodeURIComponent(url);
      fetch(proxyUrl)
          .then(function(res) {
            if (!res.ok) throw new Error('HTTP ' + res.status);
            return res.text();
          })
          .then(function(html) {
            if (!html || html.length < 200) throw new Error('Empty response');
            frame.removeAttribute('src');
            frame.srcdoc = html;
            frame.onload = function() {
              try { setupWebPreviewInteractions(frame.contentDocument.body); } catch(e) { console.warn('Cannot access iframe:', e); }
            };
          })
          .catch(function(err) {
            console.warn('Fetch failed:', err.message);
            var warnHtml = '<div style="padding:12px 16px;background:#fff7e6;border-bottom:1px solid #ffe58f;font-size:12px;color:#d48806;display:flex;align-items:center;gap:8px">';
            warnHtml += '<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>';
            warnHtml += '无法加载页面（请确认代理服务已启动：<code style="background:#fff;padding:1px 6px;border-radius:3px;font-size:11px">node html/scripts/proxy-server.js</code>），显示模拟内容</div>';
            var previewContainer = body.parentElement;
            var warnDiv = document.createElement('div');
            warnDiv.className = 'web-proxy-warn';
            warnDiv.innerHTML = warnHtml;
            previewContainer.insertBefore(warnDiv, body);
            renderMockContent(body);
          });
    }

    function setupWebPreviewInteractions(body) {
      // Collect selectable elements at multiple granularities
      var candidates = [];
      var seen = new Set();
      function addCandidate(el, priority) {
        if (seen.has(el)) return;
        seen.add(el);
        candidates.push({el: el, priority: priority});
      }

      // Priority 1: Large semantic blocks (table, form, nav, section)
      body.querySelectorAll('table, form, nav, header, footer, section, article').forEach(function(el) {
        addCandidate(el, 1);
      });

      // Priority 2: Table columns - find tables and make each column header clickable
      body.querySelectorAll('table').forEach(function(table) {
        var ths = table.querySelectorAll('th');
        ths.forEach(function(th, colIdx) {
          // Create a virtual "column" selection by marking the th
          th.setAttribute('data-col-index', colIdx);
          th.setAttribute('data-ws-col', 'true');
          addCandidate(th, 2);
        });
      });

      // Priority 3: Groups of links (e.g. a list of <a> tags in a container)
      body.querySelectorAll('td a[href], li a[href], div > a[href]').forEach(function(a) {
        addCandidate(a, 3);
      });

      // Priority 4: Pagination
      body.querySelectorAll('[class*=page], [class*=pager], [class*=pagination]').forEach(function(el) {
        addCandidate(el, 4);
      });

      // Priority 5: Input/button for search
      body.querySelectorAll('input[type=text], input[type=search], input:not([type]), button').forEach(function(el) {
        addCandidate(el, 5);
      });

      // Priority 6: Medium divs with class/id
      body.querySelectorAll('div[class], div[id]').forEach(function(el) {
        if (el.children.length >= 2 && el.offsetHeight > 40) {
          var dominated = false;
          candidates.forEach(function(c) { if (c.el !== el && c.el.contains(el) && c.priority <= 2) dominated = true; });
          if (!dominated) addCandidate(el, 6);
        }
      });

      // Sort by priority (lower = shown first on hover)
      candidates.sort(function(a, b) { return a.priority - b.priority; });

      candidates.forEach(function(item) {
        var el = item.el;
        el.setAttribute('data-ws', guessElementType(el));

        el.addEventListener('mouseenter', function(e) {
          if (!webSelectMode) return;
          e.stopPropagation();
          body.querySelectorAll('.ws-hover').forEach(function(h) {
            if (h !== el) { h.classList.remove('ws-hover'); h.style.outline = h.classList.contains('ws-selected') ? '3px solid #1677ff' : ''; }
          });
          if (!el.classList.contains('ws-selected')) {
            el.classList.add('ws-hover');
            el.style.outline = '2px dashed #1677ff';
            el.style.outlineOffset = '2px';
          }
        });

        el.addEventListener('mouseleave', function() {
          if (!el.classList.contains('ws-selected')) {
            el.classList.remove('ws-hover');
            el.style.outline = '';
            el.style.outlineOffset = '';
          }
        });

        el.addEventListener('click', function(e) {
          if (!webSelectMode) return;
          e.preventDefault(); e.stopPropagation();

          // For table column header click: select the whole column
          var isColHeader = el.getAttribute('data-ws-col') === 'true';
          var rt = el.getAttribute('data-ws') || 'content';

          if (el.classList.contains('ws-selected')) {
            el.classList.remove('ws-selected');
            el.style.outline = ''; el.style.background = ''; el.style.outlineOffset = '';
            // If column, unhighlight column cells
            if (isColHeader) unhighlightColumn(el);
            webRegions = webRegions.filter(function(r) { return r.el !== el; });
          } else {
            el.classList.add('ws-selected');
            el.classList.remove('ws-hover');
            el.style.outline = '3px solid #1677ff';
            el.style.outlineOffset = '2px';
            el.style.background = 'rgba(22,119,255,0.04)';
            // If column header, highlight all cells in that column
            if (isColHeader) highlightColumn(el);

            var desc = el.tagName.toLowerCase();
            if (el.id) desc += '#' + el.id;
            else if (el.className && typeof el.className === 'string') desc += '.' + el.className.split(' ')[0];
            if (isColHeader) desc = '列：' + (el.textContent || '').trim().substring(0, 20);

            var labels = {content:'内容区域',file:'文件下载',pager:'翻页控件',search:'搜索区域',searchBtn:'查询按钮',exclude:'排除区域',followLinks:'跟踪链接'};
            webRegions.push({ el: el, type: rt, label: labels[rt] || '内容区域', selector: desc, isColumn: isColHeader });
          }
          renderWebRegions();
        });
      });
    }

    function highlightColumn(th) {
      var colIdx = parseInt(th.getAttribute('data-col-index'));
      var table = th.closest('table');
      if (!table) return;
      table.querySelectorAll('tr').forEach(function(tr) {
        var cells = tr.querySelectorAll('td, th');
        if (cells[colIdx]) {
          cells[colIdx].style.background = 'rgba(22,119,255,0.06)';
          cells[colIdx].classList.add('ws-col-highlight');
        }
      });
    }

    function unhighlightColumn(th) {
      var table = th.closest('table');
      if (!table) return;
      table.querySelectorAll('.ws-col-highlight').forEach(function(cell) {
        cell.style.background = '';
        cell.classList.remove('ws-col-highlight');
      });
    }

    function guessElementType(el) {
      var tag = el.tagName.toLowerCase();
      var cls = (el.className || '').toString().toLowerCase();
      // Exclude
      if (tag === 'nav' || tag === 'header' || tag === 'footer') return 'exclude';
      if (cls.indexOf('header') !== -1 || cls.indexOf('footer') !== -1) return 'exclude';
      // Search
      if (tag === 'form') return 'search';
      if (tag === 'input') return 'search';
      if (tag === 'button') return 'searchBtn';
      if (cls.indexOf('search') !== -1 || cls.indexOf('filter') !== -1) return 'search';
      // Pager
      if (cls.indexOf('page') !== -1 || cls.indexOf('pager') !== -1 || cls.indexOf('pagination') !== -1) return 'pager';
      // Table column header - check if column contains links
      if (tag === 'th' && el.getAttribute('data-ws-col')) {
        var colIdx = parseInt(el.getAttribute('data-col-index'));
        var table = el.closest('table');
        if (table) {
          var firstDataRow = table.querySelector('tbody tr') || table.querySelectorAll('tr')[1];
          if (firstDataRow) {
            var cells = firstDataRow.querySelectorAll('td');
            if (cells[colIdx] && cells[colIdx].querySelector('a[href]')) return 'followLinks';
          }
        }
        return 'content';
      }
      // Link
      if (tag === 'a' && el.href) {
        if (/\.(pdf|doc|docx|xls|xlsx|zip|rar)/i.test(el.href)) return 'file';
        return 'followLinks';
      }
      // Table with file links
      if (tag === 'table' && el.querySelector('a[href*=".pdf"], a[href*=".doc"]')) return 'file';
      // Div with links
      if (el.querySelector && el.querySelector('a[href*=".pdf"], a[href*=".doc"]')) return 'file';
      return 'content';
    }

    function renderMockContent(body) {
      var data = [
        {t:'关于全资子公司通过高新技术企业重新认定的公告',s:'64k',d:'2026-03-16'},
        {t:'关于公司及控股子公司提供担保的进展公告',s:'105k',d:'2025-12-26'},
        {t:'第六届董事会第十三次会议决议公告',s:'83k',d:'2025-10-29'},
        {t:'董事和高级管理人员持股管理制度（2025年10月）',s:'136k',d:'2025-10-29'},
        {t:'投资者关系管理制度（2025年10月）',s:'166k',d:'2025-10-29'},
        {t:'关于新增2025年度日常关联交易预计的公告',s:'100k',d:'2025-10-29'},
        {t:'2025年第三季度报告',s:'245k',d:'2025-10-28'},
        {t:'关于使用闲置自有资金进行现金管理的公告',s:'78k',d:'2025-09-15'}
      ];
      var rows='';
      data.forEach(function(a){
        rows+='<tr data-ws="content"><td style="padding:8px 10px;border-bottom:1px solid #eee;font-size:11px">300230</td><td style="padding:8px 10px;border-bottom:1px solid #eee;font-size:11px">永利股份</td><td style="padding:8px 10px;border-bottom:1px solid #eee;font-size:11px"><a href="#" style="color:#1677ff;text-decoration:none">' + a.t + '</a> <span style="color:#ff4d4f;font-size:9px;padding:1px 4px;background:#fff1f0;border-radius:2px">PDF</span>(' + a.s + ')</td><td style="padding:8px 10px;border-bottom:1px solid #eee;font-size:11px;color:#999">' + a.d + '</td></tr>';
      });
      var mockHtml = '<div style="font-family:-apple-system,SimSun,serif;padding:16px">';
      mockHtml += '<div style="display:flex;align-items:center;gap:12px;margin-bottom:16px;padding-bottom:12px;border-bottom:2px solid #1677ff">';
      mockHtml += '<span style="font-size:16px;font-weight:700;color:rgba(0,0,0,0.88)">上市公司公告</span>';
      mockHtml += '<span style="font-size:12px;color:rgba(0,0,0,0.35)">（模拟数据）</span>';
      mockHtml += '</div>';
      mockHtml += '<div style="display:flex;gap:8px;margin-bottom:14px;align-items:center" data-ws="search">';
      mockHtml += '<input type="text" placeholder="输入公司名称或代码" style="padding:4px 10px;height:30px;border:1px solid #d9d9d9;border-radius:4px;font-size:12px;flex:1;max-width:240px;outline:none">';
      mockHtml += '<button style="padding:4px 16px;height:30px;background:#1677ff;color:#fff;border:none;border-radius:4px;font-size:12px;cursor:pointer" data-ws="searchBtn">查询</button>';
      mockHtml += '</div>';
      mockHtml += '<table style="width:100%;border-collapse:collapse"><thead><tr>';
      mockHtml += '<th style="text-align:left;padding:8px 10px;background:#f7f7f7;font-size:11px;color:#666;border-bottom:2px solid #ddd">证券代码</th>';
      mockHtml += '<th style="text-align:left;padding:8px 10px;background:#f7f7f7;font-size:11px;color:#666;border-bottom:2px solid #ddd">简称</th>';
      mockHtml += '<th style="text-align:left;padding:8px 10px;background:#f7f7f7;font-size:11px;color:#666;border-bottom:2px solid #ddd">公告标题</th>';
      mockHtml += '<th style="text-align:left;padding:8px 10px;background:#f7f7f7;font-size:11px;color:#666;border-bottom:2px solid #ddd">时间</th>';
      mockHtml += '</tr></thead><tbody>' + rows + '</tbody></table>';
      mockHtml += '<div style="display:flex;justify-content:center;gap:4px;margin-top:14px;padding-top:10px;border-top:1px solid #f0f0f0" data-ws="pager">';
      mockHtml += '<span style="padding:3px 10px;border:1px solid #d9d9d9;border-radius:4px;font-size:11px;color:rgba(0,0,0,0.45);cursor:pointer">上一页</span>';
      for (var p = 1; p <= 5; p++) {
        var active = p === 1 ? 'background:#1677ff;color:#fff;border-color:#1677ff' : 'color:rgba(0,0,0,0.65)';
        mockHtml += '<span style="padding:3px 10px;border:1px solid #d9d9d9;border-radius:4px;font-size:11px;cursor:pointer;' + active + '">' + p + '</span>';
      }
      mockHtml += '<span style="padding:3px 10px;border:1px solid #d9d9d9;border-radius:4px;font-size:11px;color:rgba(0,0,0,0.65);cursor:pointer">下一页</span>';
      mockHtml += '</div></div>';

      // Write into iframe
      var frame = body.querySelector('#webPreviewFrame');
      if (frame) {
        frame.srcdoc = mockHtml;
        frame.onload = function() {
          try { setupWebPreviewInteractions(frame.contentDocument.body); } catch(e) { console.warn('Cannot access iframe:', e); }
        };
      } else {
        // Fallback: create iframe
        var zoom = parseFloat(document.getElementById('webPreviewZoom').value) || 0.75;
        var inv = 1 / zoom;
        body.innerHTML += '<iframe id="webPreviewFrame" style="width:' + (inv*100) + '%;height:' + (inv*100) + '%;border:none;transform:scale(' + zoom + ');transform-origin:0 0" sandbox="allow-same-origin"></iframe>';
        frame = body.querySelector('#webPreviewFrame');
        frame.srcdoc = mockHtml;
        frame.onload = function() {
          try { setupWebPreviewInteractions(frame.contentDocument.body); } catch(e) { console.warn('Cannot access iframe:', e); }
        };
      }
    }

    function toggleWebSelectMode() {
      webSelectMode = !webSelectMode;
      var btn = document.getElementById('webSelectBtn');
      btn.style.background = webSelectMode ? '#1677ff' : '#fff';
      btn.style.color = webSelectMode ? '#fff' : 'rgba(0,0,0,0.65)';
      btn.style.borderColor = webSelectMode ? '#1677ff' : '#d9d9d9';
      document.getElementById('webPreviewHint').style.display = webSelectMode ? '' : 'none';
    }

    function clearWebRegions() {
      webRegions.forEach(function(r) { r.el.classList.remove('ws-selected'); r.el.style.outline = ''; r.el.style.background = ''; });
      webRegions = []; renderWebRegions();
    }

    function renderWebRegions() {
      var panel = document.getElementById('webRegionsPanel');
      if (!webRegions.length) { panel.innerHTML = '<div style="font-size:11px;color:rgba(0,0,0,0.25);padding:4px 0">点击"选择区域"在预览中标记采集区域</div>'; return; }
      var html = '<div style="font-size:11px;color:rgba(0,0,0,0.35);margin-bottom:6px">已选 ' + webRegions.length + ' 个区域</div>';
      var tc = {content:'#e6f4ff;color:#1677ff',file:'#fff7e6;color:#fa8c16',pager:'#f6ffed;color:#52c41a',search:'#f5f3ff;color:#8b5cf6',searchBtn:'#f5f3ff;color:#8b5cf6',exclude:'#fff1f2;color:#ff4d4f',followLinks:'#e6fffb;color:#13c2c2',column:'#e6f4ff;color:#1677ff'};
      webRegions.forEach(function(r, i) {
        html += '<div style="display:flex;align-items:center;gap:8px;padding:5px 10px;border:1px solid #f0f0f0;border-radius:6px;margin-bottom:4px;background:#fff;flex-wrap:wrap">';
        html += '<span style="padding:1px 8px;border-radius:3px;font-size:10px;font-weight:600;background:' + tc[r.type] + '">' + r.label + '</span>';
        html += '<span style="font-family:monospace;font-size:10px;color:rgba(0,0,0,0.25);flex:1;min-width:60px">[' + r.selector + ']</span>';
        // Search region: show input for search value
        if (r.type === 'search') {
          html += '<input class="form-input" placeholder="输入搜索条件，如：金盘科技" value="' + (r.searchValue || '') + '" oninput="webRegions[' + i + '].searchValue=this.value" style="height:26px;font-size:11px;flex:1;min-width:140px;max-width:240px">';
        }
        html += '<button onclick="removeWebRegion(' + i + ')" style="border:none;background:none;cursor:pointer;color:rgba(0,0,0,0.25);font-size:12px">✕</button></div>';
      });
      panel.innerHTML = html;
      renderRulePreview();
    }

    function renderRulePreview() {
      var preview = document.getElementById('webRulePreview');
      var steps = document.getElementById('webRuleSteps');
      if (!preview || !steps) return;
      if (!webRegions.length) { preview.style.display = 'none'; return; }

      var url = document.getElementById('webUrl').value.trim() || '目标网页';
      var searchRegion = webRegions.find(function(r) { return r.type === 'search'; });
      var searchBtn = webRegions.find(function(r) { return r.type === 'searchBtn'; });
      var fileRegion = webRegions.find(function(r) { return r.type === 'file'; });
      var contentRegion = webRegions.find(function(r) { return r.type === 'content'; });
      var followRegion = webRegions.find(function(r) { return r.type === 'followLinks'; });
      var pagerRegion = webRegions.find(function(r) { return r.type === 'pager'; });
      var excludeRegions = webRegions.filter(function(r) { return r.type === 'exclude'; });

      var pagerVal = document.getElementById('webAutoPager') ? document.getElementById('webAutoPager').value : 'on';
      var depthVal = document.getElementById('webDepth') ? document.getElementById('webDepth').value : '1';
      var collectVal = document.getElementById('webCollectType') ? document.getElementById('webCollectType').value : 'all';

      var html = '';
      var stepNum = 1;

      html += '<div>' + (stepNum++) + '. 打开 <span style="color:#1677ff;word-break:break-all">' + url + '</span></div>';

      if (searchRegion) {
        var val = searchRegion.searchValue ? '<span style="color:#8b5cf6;font-weight:600">&quot;' + searchRegion.searchValue + '&quot;</span>' : '<span style="color:#ff4d4f">未填写</span>';
        html += '<div>' + (stepNum++) + '. 在搜索框中输入 ' + val;
        if (searchBtn) html += '，点击查询按钮';
        html += '</div>';
      }

      if (followRegion) {
        var colName = followRegion.selector || '链接';
        html += '<div>' + (stepNum++) + '. 逐一点击 <span style="color:#1677ff;font-weight:500">' + colName + '</span> 中的每个链接，进入详情页</div>';
        if (collectVal !== 'page') {
          html += '<div>' + (stepNum++) + '. 在详情页中查找并下载文件（PDF/Word 等）</div>';
        }
        if (collectVal !== 'file') {
          html += '<div>' + (stepNum++) + '. 采集详情页的页面内容</div>';
        }
      } else if (fileRegion) {
        html += '<div>' + (stepNum++) + '. 从页面中直接提取所有文件下载链接（PDF/Word 等）</div>';
      } else if (contentRegion) {
        html += '<div>' + (stepNum++) + '. 采集标记区域的网页内容</div>';
      }

      if (pagerRegion || pagerVal !== 'off') {
        var pagerDesc = '自动翻页';
        if (pagerVal === 'max') {
          var maxP = document.getElementById('webMaxPages') ? document.getElementById('webMaxPages').value : '50';
          pagerDesc += '（最多 ' + maxP + ' 页）';
        }
        html += '<div>' + (stepNum++) + '. ' + pagerDesc + '，对每一页重复上述采集步骤</div>';
      }

      html += '<div>' + (stepNum++) + '. 将所有采集的文件/内容保存到指定 Catalog 卷</div>';

      if (excludeRegions.length) {
        html += '<div style="color:rgba(0,0,0,0.35);margin-top:4px;font-size:11px">排除区域：' + excludeRegions.map(function(r) { return r.selector; }).join('、') + '</div>';
      }

      steps.innerHTML = html;
      preview.style.display = '';
    }

    function removeWebRegion(idx) {
      var r = webRegions[idx];
      if (r && r.el) { r.el.classList.remove('ws-selected'); r.el.style.outline = ''; r.el.style.background = ''; }
      webRegions.splice(idx, 1); renderWebRegions();
    }

    // === Sample Preview ===
    var sampleData = [
      {
        "_id": "ObjectId('66a1b2c3d4e5f6a7b8c9d0e1')",
        "pump": "HP-101",
        "crew": "Crew-A",
        "datetime": "2026-04-10T14:32:15.000Z",
        "engine_rpm": 1245.6,
        "pump_rate": 3.82,
        "disch_pressure": 8520.3,
        "engine_oil_pressure": 45.2,
        "engine_coolant_temp": 88.1,
        "lube_oil_pressure": 32.7,
        "engine_hours": 12456.78,
        "pumping_hours": 8234.56
      },
      {
        "_id": "ObjectId('66a1b2c3d4e5f6a7b8c9d0e2')",
        "pump": "HP-102",
        "crew": "Crew-A",
        "datetime": "2026-04-10T14:32:15.000Z",
        "engine_rpm": 0.0,
        "pump_rate": 0.0,
        "disch_pressure": null,
        "engine_oil_pressure": 0.0,
        "engine_coolant_temp": 25.3,
        "lube_oil_pressure": 0.0,
        "engine_hours": 9876.54,
        "pumping_hours": 6543.21
      },
      {
        "_id": "ObjectId('66a1b2c3d4e5f6a7b8c9d0e3')",
        "pump": "HP-205",
        "crew": "Crew-B",
        "datetime": "2026-04-10T14:32:16.000Z",
        "engine_rpm": 1198.3,
        "pump_rate": 3.95,
        "disch_pressure": 8601.2,
        "engine_oil_pressure": 44.8,
        "engine_coolant_temp": 87.9,
        "lube_oil_pressure": 33.1,
        "engine_hours": 15234.12,
        "pumping_hours": 10456.78
      },
      {
        "_id": "ObjectId('66c3d4e5f6a7b8c9d0e1f2a3')",
        "pump": "HP-101",
        "crew": "Crew-A",
        "alert_type": "high_pressure",
        "severity": "critical",
        "triggered_at": "2026-04-10T08:15:32.000Z",
        "resolved_at": "2026-04-10T08:22:10.000Z",
        "threshold": {
          "field": "disch_pressure",
          "op": ">",
          "value": 12000
        },
        "context": {
          "reading_value": 12350.5,
          "engine_rpm": 1320.0
        },
        "acknowledged": true
      },
      {
        "_id": "ObjectId('66c3d4e5f6a7b8c9d0e1f2a4')",
        "pump": "HP-205",
        "crew": "Crew-B",
        "alert_type": "low_oil_pressure",
        "severity": "warning",
        "triggered_at": "2026-04-09T22:41:05.000Z",
        "resolved_at": null,
        "threshold": {
          "field": "engine_oil_pressure",
          "op": "<",
          "value": 20
        },
        "context": {
          "reading_value": 18.2,
          "engine_rpm": 1180.5
        },
        "acknowledged": false
      }
    ];
    var currentSampleIdx = 0;

    function openSamplePreview() {
      currentSampleIdx = 0;
      renderSample();
      document.getElementById('samplePreviewModal').classList.add('open');
    }

    function closeSamplePreview() {
      document.getElementById('samplePreviewModal').classList.remove('open');
    }

    function nextSample() {
      currentSampleIdx = (currentSampleIdx + 1) % sampleData.length;
      renderSample();
    }

    function renderSample() {
      var el = document.getElementById('sampleContent');
      var idx = document.getElementById('sampleIndex');
      var doc = sampleData[currentSampleIdx];
      el.textContent = JSON.stringify(doc, null, 2);
      idx.textContent = '第 ' + (currentSampleIdx + 1) + ' 条 / 共 ' + sampleData.length + ' 条';
    }

    // === Flatten Control (reversible) ===
    var originalMongoSchemas = {}; // Store originals for undo

    function showFlattenControlIfNeeded() {
      var ctrl = document.getElementById('flattenControl');
      if (!ctrl) return;
      var schema = getActiveColSchema();
      var hasJson = schema.some(function(col) { return col.type === 'JSON'; });
      var hasExpanded = schema.some(function(col) { return col._expandedFrom; });
      ctrl.style.display = (hasJson || hasExpanded) ? 'flex' : 'none';

      // Update button text
      var btnArea = ctrl.querySelector('.flatten-btns');
      if (!btnArea) {
        // Replace the single button with a button group
        var oldBtn = ctrl.querySelector('button');
        if (oldBtn) oldBtn.remove();
        btnArea = document.createElement('span');
        btnArea.className = 'flatten-btns';
        btnArea.style.cssText = 'display:flex;gap:6px;margin-left:auto';
        ctrl.appendChild(btnArea);
      }
      var html = '';
      if (hasJson) {
        html += '<button style="padding:2px 10px;height:26px;border:1px solid #1677ff;border-radius:4px;background:#fff;color:#1677ff;font-size:12px;cursor:pointer;white-space:nowrap" onclick="deepFlattenAll()">全部深度打平</button>';
      }
      if (hasExpanded) {
        html += '<button style="padding:2px 10px;height:26px;border:1px solid #fa8c16;border-radius:4px;background:#fff;color:#fa8c16;font-size:12px;cursor:pointer;white-space:nowrap" onclick="undoFlattenAll()">还原全部</button>';
      }
      btnArea.innerHTML = html;

      // Update hint text
      var hint = ctrl.querySelector('span');
      if (hint) {
        if (hasJson && !hasExpanded) hint.textContent = '检测到嵌套 JSON 字段，当前已打平第一层。';
        else if (hasJson && hasExpanded) hint.textContent = '部分字段已深度打平，仍有嵌套 JSON 字段。';
        else if (!hasJson && hasExpanded) hint.textContent = '所有嵌套字段已深度打平。';
      }
    }

    function getCollectionKey() {
      var connSel = document.getElementById('stConnectorSelect');
      var tableSel = document.getElementById('stDbTableSelect');
      if (!connSel || connSel.value !== 'mongodb' || !tableSel || !tableSel.value) return null;
      return tableSel.value;
    }

    function saveOriginalSchema(key) {
      if (!originalMongoSchemas[key]) {
        originalMongoSchemas[key] = JSON.parse(JSON.stringify(mongoColSchemas[key]));
      }
    }

    function expandJsonColumn(schema, colIdx) {
      var col = schema[colIdx];
      if (col.type !== 'JSON') return schema;

      // Parse the preview JSON to discover sub-fields
      var subFields = [];
      try {
        var sample = JSON.parse(col.preview[0]);
        Object.keys(sample).forEach(function(k, ki) {
          var val = sample[k];
          var valType = 'VARCHAR';
          var valLen = 255;
          var valPreview = [];

          if (typeof val === 'number') { valType = 'DOUBLE'; valLen = 0; }
          else if (typeof val === 'boolean') { valType = 'BOOLEAN'; valLen = 0; }
          else if (typeof val === 'object' && val !== null && !Array.isArray(val)) { valType = 'JSON'; valLen = 0; }
          else if (Array.isArray(val)) { valType = 'JSON'; valLen = 0; }

          // Build preview from all sample docs
          col.preview.forEach(function(p) {
            try {
              var doc = JSON.parse(p);
              var v = doc[k];
              valPreview.push(v === null || v === undefined ? 'NULL' : (typeof v === 'object' ? JSON.stringify(v) : String(v)));
            } catch(e) { valPreview.push('—'); }
          });

          subFields.push({
            idx: col.idx + String.fromCharCode(97 + ki),
            name: col.name + '.' + k,
            type: valType,
            len: valLen,
            pk: false,
            desc: col.desc.replace(/（.*）/, '') + '.' + k,
            def: '',
            preview: valPreview,
            _expandedFrom: col.name
          });
        });
      } catch(e) {
        // Can't parse, just mark as expanded
        subFields.push({
          idx: col.idx + 'a', name: col.name + '.*', type: 'VARCHAR', len: 255,
          pk: false, desc: col.desc + '（无法解析）', def: '', preview: col.preview,
          _expandedFrom: col.name
        });
      }

      var newSchema = schema.slice();
      newSchema.splice(colIdx, 1, ...subFields);
      return newSchema;
    }

    function deepFlattenAll() {
      var key = getCollectionKey();
      if (!key) return;
      saveOriginalSchema(key);

      var schema = mongoColSchemas[key];
      var changed = true;
      var maxIter = 10; // prevent infinite loop
      while (changed && maxIter-- > 0) {
        changed = false;
        for (var i = 0; i < schema.length; i++) {
          if (schema[i].type === 'JSON') {
            schema = expandJsonColumn(schema, i);
            changed = true;
            break; // restart scan after splice
          }
        }
      }
      mongoColSchemas[key] = schema;
      renderColSchemaTable();
    }

    function flattenColumn(colIdx) {
      var key = getCollectionKey();
      if (!key) return;
      saveOriginalSchema(key);

      var schema = mongoColSchemas[key];
      if (!schema[colIdx] || schema[colIdx].type !== 'JSON') return;
      mongoColSchemas[key] = expandJsonColumn(schema, colIdx);
      renderColSchemaTable();
    }

    function collapseColumn(parentName) {
      var key = getCollectionKey();
      if (!key || !originalMongoSchemas[key]) return;

      // Find the original column
      var origCol = originalMongoSchemas[key].find(function(c) { return c.name === parentName; });
      if (!origCol) return;

      var schema = mongoColSchemas[key];
      // Remove all expanded sub-fields and insert original back
      var firstIdx = -1;
      var newSchema = [];
      schema.forEach(function(col, i) {
        if (col._expandedFrom === parentName) {
          if (firstIdx === -1) { firstIdx = i; newSchema.push(JSON.parse(JSON.stringify(origCol))); }
          // skip expanded fields
        } else {
          newSchema.push(col);
        }
      });
      mongoColSchemas[key] = newSchema;
      renderColSchemaTable();
    }

    function undoFlattenAll() {
      var key = getCollectionKey();
      if (!key || !originalMongoSchemas[key]) return;
      mongoColSchemas[key] = JSON.parse(JSON.stringify(originalMongoSchemas[key]));
      delete originalMongoSchemas[key];
      renderColSchemaTable();
    }

    // Add per-column flatten/collapse buttons for JSON and expanded columns
    var _origRenderColSchema = renderColSchemaTable;
    renderColSchemaTable = function() {
      _origRenderColSchema();
      showFlattenControlIfNeeded();

      var tbody = document.getElementById('colSchemaBody');
      if (!tbody) return;
      var rows = tbody.querySelectorAll('tr');
      var schema = getActiveColSchema();

      rows.forEach(function(row, i) {
        if (i >= schema.length) return;
        var col = schema[i];
        var previewCell = row.querySelector('.col-preview');
        if (!previewCell) return;

        if (col.type === 'JSON' && !previewCell.querySelector('.flatten-btn')) {
          var btn = document.createElement('button');
          btn.className = 'flatten-btn';
          btn.style.cssText = 'margin-left:8px;padding:1px 8px;border:1px solid #1677ff;border-radius:4px;background:#fff;color:#1677ff;font-size:11px;cursor:pointer;white-space:nowrap;flex-shrink:0';
          btn.textContent = '展开';
          btn.onclick = function() { flattenColumn(i); };
          previewCell.style.display = 'flex';
          previewCell.style.alignItems = 'center';
          previewCell.appendChild(btn);
        }

        if (col._expandedFrom && !previewCell.querySelector('.collapse-btn')) {
          // Show a collapse indicator on the first expanded field
          var isFirst = (i === 0 || !schema[i-1]._expandedFrom || schema[i-1]._expandedFrom !== col._expandedFrom);
          if (isFirst) {
            var cbtn = document.createElement('button');
            cbtn.className = 'collapse-btn';
            cbtn.style.cssText = 'margin-left:8px;padding:1px 8px;border:1px solid #fa8c16;border-radius:4px;background:#fff;color:#fa8c16;font-size:11px;cursor:pointer;white-space:nowrap;flex-shrink:0';
            cbtn.textContent = '收起 ' + col._expandedFrom;
            cbtn.onclick = function() { collapseColumn(col._expandedFrom); };
            previewCell.style.display = 'flex';
            previewCell.style.alignItems = 'center';
            previewCell.appendChild(cbtn);
          }
          // Highlight expanded rows
          row.style.background = '#fafafa';
        }
      });
    };

    // === REST API connector support ===
    var apiMockEndpoints = {
      '/work_orders': { fields: ['id','strCode','intPriorityID','intWorkOrderStatusID','intSiteID','strDescription','dtmDateCreated','dtmDateLastModified','strAssetIds','intMaintenanceTypeID','intAssignedToUserID','dblTimeEstimatedHours'], rows: '27,000', dataPath: 'data' },
      '/wo_tasks': { fields: ['id','intWorkOrderID','strDescription','intTaskType','dblTimeSpentHours','dtmDateCompleted','intAssignedToUserID','intStatusID'], rows: '318,000', dataPath: 'data' },
      '/assets': { fields: ['id','strName','strCode','intCategoryID','intParentID','strMake','strModel','strSerialNumber','intSiteID','bolIsOnline','dtmDateCreated'], rows: '2,100', dataPath: 'data' },
      '/meter_readings': { fields: ['id','intAssetID','dblMeterReading','dtmDateSubmitted','intMeterReadingUnitsID','strNotes'], rows: '45,000', dataPath: 'data' },
      '/priorities': { fields: ['id','strName','intControlID','intOrder'], rows: '5', dataPath: 'data' },
      '/statuses': { fields: ['id','strName','intControlID','intStatusType'], rows: '12', dataPath: 'data' },
      '/maintenance_types': { fields: ['id','strName','strCode','intControlID'], rows: '8', dataPath: 'data' },
      '/sites': { fields: ['id','strName','strCode','strAddress','strCity','strCountry'], rows: '3', dataPath: 'data' },
      '/users': { fields: ['id','strFullName','strEmailAddress','strUserName','bolActive'], rows: '86', dataPath: 'data' }
    };

    var apiColSchemas = {
      '/work_orders': [
        { idx: '1', name: 'id', type: 'BIGINT', len: 0, pk: true, desc: '工单 ID', preview: ['1001', '1002', '1003'] },
        { idx: '2', name: 'strCode', type: 'VARCHAR', len: 32, pk: false, desc: '工单编号', preview: ['WO-2026-0412', 'WO-2026-0411'] },
        { idx: '3', name: 'intPriorityID', type: 'INT', len: 0, pk: false, desc: '优先级 ID', preview: ['231162', '231163'] },
        { idx: '4', name: 'intWorkOrderStatusID', type: 'INT', len: 0, pk: false, desc: '状态 ID', preview: ['1', '2', '3'] },
        { idx: '5', name: 'strDescription', type: 'TEXT', len: 0, pk: false, desc: '工单描述', preview: ['Pump HP-101 vibration alarm', 'Scheduled PM for ENG-A'] },
        { idx: '6', name: 'dtmDateCreated', type: 'TIMESTAMP', len: 0, pk: false, desc: '创建时间', preview: ['2026-04-10 08:30:00', '2026-04-09 14:00:00'] },
        { idx: '7', name: 'dtmDateLastModified', type: 'TIMESTAMP', len: 0, pk: false, desc: '最后更新时间', preview: ['2026-04-11 02:15:00', '2026-04-10 16:30:00'] },
        { idx: '8', name: 'strAssetIds', type: 'VARCHAR', len: 255, pk: false, desc: '关联资产 ID（逗号分隔）', preview: ['1234,5678', '9012'] },
        { idx: '9', name: 'intMaintenanceTypeID', type: 'INT', len: 0, pk: false, desc: '维修类型 ID', preview: ['1', '2', '3'] },
        { idx: '10', name: 'intAssignedToUserID', type: 'INT', len: 0, pk: false, desc: '指派人 ID', preview: ['101', '205'] },
        { idx: '11', name: 'dblTimeEstimatedHours', type: 'DOUBLE', len: 0, pk: false, desc: '预估工时', preview: ['2.5', '4.0', '1.0'] },
        { idx: '12', name: 'intSiteID', type: 'INT', len: 0, pk: false, desc: '站点 ID', preview: ['1', '2'] }
      ],
      '/assets': [
        { idx: '1', name: 'id', type: 'BIGINT', len: 0, pk: true, desc: '资产 ID', preview: ['1234', '5678'] },
        { idx: '2', name: 'strName', type: 'VARCHAR', len: 128, pk: false, desc: '资产名称', preview: ['HP-101', 'ENG-A', 'RIG-7'] },
        { idx: '3', name: 'strCode', type: 'VARCHAR', len: 32, pk: false, desc: '资产编号', preview: ['HP-101', 'ENG-A'] },
        { idx: '4', name: 'intCategoryID', type: 'INT', len: 0, pk: false, desc: '分类 ID', preview: ['10', '20', '30'] },
        { idx: '5', name: 'intParentID', type: 'INT', len: 0, pk: false, desc: '父资产 ID', preview: ['5678', 'NULL'] },
        { idx: '6', name: 'strMake', type: 'VARCHAR', len: 64, pk: false, desc: '制造商', preview: ['Gardner Denver', 'NOV'] },
        { idx: '7', name: 'strModel', type: 'VARCHAR', len: 64, pk: false, desc: '型号', preview: ['Quintuplex', 'Triplex'] },
        { idx: '8', name: 'strSerialNumber', type: 'VARCHAR', len: 64, pk: false, desc: '序列号', preview: ['SN-2023-001', 'SN-2024-015'] },
        { idx: '9', name: 'intSiteID', type: 'INT', len: 0, pk: false, desc: '站点 ID', preview: ['1', '2'] },
        { idx: '10', name: 'bolIsOnline', type: 'BOOLEAN', len: 0, pk: false, desc: '是否在线', preview: ['true', 'true', 'false'] },
        { idx: '11', name: 'dtmDateCreated', type: 'TIMESTAMP', len: 0, pk: false, desc: '创建时间', preview: ['2023-06-15 10:00:00'] }
      ]
    };

    function showApiEndpointPreview() {
      var endpoint = (document.getElementById('stApiEndpoint') || {}).value || '';
      // Normalize: ensure leading slash
      endpoint = endpoint.trim();
      if (endpoint && endpoint.charAt(0) !== '/') endpoint = '/' + endpoint;
      var preview = document.getElementById('stApiPreview');
      var fieldsEl = document.getElementById('stApiFields');
      var countEl = document.getElementById('stApiFieldCount');
      if (!preview || !fieldsEl) return;

      if (!endpoint) { preview.style.display = 'none'; return; }

      var ep = apiMockEndpoints[endpoint];
      var fields;
      if (ep) {
        fields = ep.fields;
      } else {
        // Generate mock fields based on endpoint name
        var name = endpoint.replace(/^\//, '').replace(/[^a-zA-Z_]/g, '');
        fields = ['id', 'name', 'status', 'created_at', 'updated_at'];
        if (name) fields.splice(1, 0, name.replace(/s$/, '') + '_code');
        fields.push('description');
      }
      preview.style.display = '';
      countEl.textContent = fields.length;
      fieldsEl.innerHTML = fields.map(function(f) {
        return '<span style="padding:2px 8px;background:#f0f5ff;border:1px solid #d6e4ff;border-radius:4px;font-family:monospace;color:#1677ff">' + f + '</span>';
      }).join('');
      checkShowBottomSection();
    }

    function onApiEndpointChange() {
      showApiEndpointPreview();
      checkShowBottomSection();
    }

    // Extend getActiveColSchema to support API endpoints
    var _origGetActiveColSchema = getActiveColSchema;
    getActiveColSchema = function() {
      var connSel = document.getElementById('stConnectorSelect');
      if (connSel && connSel.value === 'rest-api') {
        var endpoint = (document.getElementById('stApiEndpoint') || {}).value || '';
        if (apiColSchemas[endpoint]) return apiColSchemas[endpoint];
      }
      return _origGetActiveColSchema();
    };

    // Extend isDataSourceReady for API connector
    var _origIsDataSourceReady2 = isDataSourceReady;
    isDataSourceReady = function() {
      var connSel = document.getElementById('stConnectorSelect');
      if (connSel && connSel.value === 'rest-api') {
        var endpoint = (document.getElementById('stApiEndpoint') || {}).value || '';
        return !!endpoint;
      }
      return _origIsDataSourceReady2();
    };
