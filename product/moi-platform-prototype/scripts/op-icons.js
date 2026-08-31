/*
 * op-icons.js — 列表操作统一图标库（与告警页图标一致）
 * 用法：
 *   1) 页面引入 <script src="../scripts/op-icons.js"></script>（在 common.js 之后）
 *   2) JS 渲染操作列时直接用 OPS.edit / OPS.del ... 拼 <button class="op-btn" title="编辑">...
 *   3) 静态/已有文字按钮的列表：渲染后调用 OPS.iconify(容器) 自动把 .action-btn-sm 文字按钮转成图标按钮
 * 图标为线条 SVG（stroke 1.8, viewBox 0 0 24 24），与 MOI 图标规范一致。
 */
(function () {
  var A = 'viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"';
  function svg(inner) { return '<svg ' + A + '>' + inner + '</svg>'; }

  var OPS = {
    perm:     svg('<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>'),
    edit:     svg('<path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5z"/>'),
    del:      svg('<path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>'),
    job:      svg('<path d="M9 5H7a2 2 0 0 0-2 2v12a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V7a2 2 0 0 0-2-2h-2"/><rect x="9" y="3" width="6" height="4" rx="1"/><path d="M9 12h6M9 16h4"/>'),
    run:      svg('<path d="M6 3l14 9-14 9V3z"/>'),
    stop:     svg('<rect x="5" y="5" width="14" height="14" rx="2"/>'),
    log:      svg('<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6M9 13h6M9 17h6"/>'),
    view:     svg('<path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7z"/><circle cx="12" cy="12" r="3"/>'),
    download: svg('<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4M7 10l5 5 5-5M12 15V3"/>'),
    lineage:  svg('<circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="M15.4 6.5 8.6 10.5M8.6 13.5l6.8 4"/>'),
    copy:     svg('<rect x="9" y="9" width="12" height="12" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/>'),
    inherit:  svg('<path d="M6 3v12M18 9v6a2 2 0 0 1-2 2H8M6 15l-3-3M6 15l3-3"/><circle cx="18" cy="6" r="3"/>'),
    disable:  svg('<circle cx="12" cy="12" r="9"/><path d="M5.6 5.6l12.8 12.8"/>'),
    enable:   svg('<circle cx="12" cy="12" r="9"/><path d="M8 12l3 3 5-6"/>'),
    role:     svg('<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/>'),
    subscribe:  svg('<path d="M12 5v14M5 12h14"/>'),
    unsubscribe: svg('<path d="M5 12h14"/>'),
    select:   svg('<path d="M20 6L9 17l-5-5"/>')
  };

  // 文字标签 → 图标 key（未命中的标签保持文字，不误转）
  var LABEL_MAP = {
    '权限': 'perm', '权限查看': 'perm',
    '编辑': 'edit', '修改': 'edit', '修改信息': 'edit', '重命名': 'edit',
    '删除': 'del',
    '作业': 'job',
    '运行': 'run', '停止': 'stop',
    '日志': 'log', '查看日志': 'log', '查看告警记录': 'log', '记录': 'log', '操作日志': 'log',
    '详情': 'view', '查看': 'view',
    '下载': 'download', '血缘': 'lineage', '复制': 'copy',
    '继承': 'inherit', '禁用': 'disable', '启用': 'enable',
    '修改角色': 'role', '订阅': 'subscribe', '取消订阅': 'unsubscribe',
    '选择数据': 'select'
  };

  // 把容器内的 .action-btn-sm 文字按钮转成 .op-btn 图标按钮（幂等，保留 onclick / disabled / danger）
  OPS.iconify = function (root) {
    if (!root) return;
    var btns = root.querySelectorAll('button.action-btn-sm');
    for (var i = 0; i < btns.length; i++) {
      var btn = btns[i];
      if (btn.getAttribute('data-op-icon')) continue;
      var label = (btn.textContent || '').trim();
      var key = LABEL_MAP[label];
      if (!key || !OPS[key]) continue; // 未识别的标签保持原样
      var danger = btn.classList.contains('danger');
      btn.className = 'op-btn' + (danger ? ' danger' : '');
      btn.setAttribute('data-tip', label); // hover 即时文字气泡（样式在 common.css）
      btn.removeAttribute('title');
      btn.setAttribute('aria-label', label);
      btn.innerHTML = OPS[key];
      btn.setAttribute('data-op-icon', key);
    }
  };

  // 便捷生成一个图标按钮 HTML 串（供 JS 渲染操作列直接使用）
  // opts（可选）：{ disabled: true, tip: '禁用原因' } — disabled 时用 tip 覆盖气泡文案说明原因
  OPS.btn = function (key, title, onclick, danger, opts) {
    if (!OPS[key]) return '';
    opts = opts || {};
    var tip = opts.disabled && opts.tip ? opts.tip : title;
    return '<button class="op-btn' + (danger ? ' danger' : '') + '" data-tip="' + tip + '" aria-label="' + title
      + (opts.disabled ? '" disabled>' : '" onclick="' + (onclick || '').replace(/"/g, '&quot;') + '">') + OPS[key] + '</button>';
  };

  window.OPS = OPS;
})();
