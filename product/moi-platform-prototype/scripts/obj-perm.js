// ============================================================
// ObjPerm — 对象权限配置通用组件（原型）
// 各对象列表操作列的「权限」入口共用：按「对象权限 × 角色」配置对象级授权，
// 全局权限（生效范围 = 全部）来的角色只读。用法：ObjPerm.open('载入任务', '任务名')
// ============================================================
window.ObjPerm = (function () {
  var PERM_SETS = {
    '连接器': [['C3', '连接器查询', '查看该连接器详情'], ['C4', '连接器修改', '修改该连接器'], ['C5', '连接器删除', '删除该连接器'], ['C6', '连接器使用', '使用该连接器做数据载入或导出']],
    '载入任务': [['L3', '载入任务查询', '查看该载入任务详情'], ['L4', '载入任务修改', '修改、暂停、恢复、重试该载入任务'], ['L5', '载入任务删除', '删除该载入任务']],
    '导出任务': [['E3', '导出任务查询', '查看该导出任务详情'], ['E4', '导出任务修改', '修改、暂停、恢复、重试该导出任务'], ['E5', '导出任务删除', '删除该导出任务']],
    '工作流': [['W2', '运行工作流', '运行该工作流、重试失败文件'], ['W4', '工作流查询', '查看该工作流详情（含分支对比、作业）'], ['W5', '工作流停止', '停止该工作流'], ['W6', '工作流修改', '修改该工作流（包括分支）'], ['W7', '工作流删除', '删除该工作流（包括分支，不含删除数据）']],
    '计算资源': [['CR2', '查看计算资源', '查看该计算资源详情'], ['CR3', '启停计算资源', '调整该计算资源的弹性伸缩 / 自动暂停'], ['CR4', '修改计算资源', '修改该计算资源'], ['CR5', '删除计算资源', '删除该计算资源']],
    '数据看板': [['DS3', '看板查询', '查看该看板的当前与任意历史时点快照、图表原始数据（快照对所有可见者一致——授予本权限即授予快照数据的可见性）；明细下探 / 上钻下卷等触发查询的实时分析另按操作者当前角色实时校验数据权限'], ['DS4', '修改看板', '修改该看板与其图表配置（含刷新周期），以及手动刷新（以刷新者当前角色执行 SQL 并校验数据权限）'], ['DS5', '删除看板', '删除该看板']],
    '知识库': [['K2', '查看知识库', '查看该知识库及其配置详情'], ['K3', '修改知识库', '修改该知识库'], ['K4', '删除知识库', '删除该知识库'], ['K5', '使用知识库', '在对话中使用该知识库（能否看到数据仍取决于数据权限）']],
    '告警规则': [['A2', '查看告警规则', '查看该告警规则详情'], ['A3', '修改告警规则', '修改、启用、禁用该告警规则'], ['A4', '删除告警规则', '删除该告警规则'], ['A9', '查看告警记录', '查看该告警规则相关的告警记录']],
    '通知对象': [['A6', '查看通知对象', '查看该通知对象详情'], ['A7', '修改通知对象', '修改、启用、禁用该通知对象'], ['A8', '删除通知对象', '删除该通知对象']],
    '发布': [['PS3', '修改发布', '修改该发布'], ['PS4', '删除发布', '删除该发布']],
    '目录': [['DC3', '修改目录', '修改该目录信息'], ['DC4', '删除目录', '删除该目录'], ['DB1', '创建库 / CREATE DATABASE', '在该目录下创建一个库'], ['DB2', '查看库列表 / SHOW DATABASES', '查看该目录下的库列表']],
    '库': [['DB3', '修改库 / ALTER DATABASE', '修改该库信息'], ['DB4', '删除库 / DROP DATABASE', '删除该库'], ['DT1', '创建表 / CREATE TABLE', '在该库中创建一张表'], ['DT2', '查看表列表 / SHOW TABLES', '查看该库中的表列表（含视图）'], ['DV1', '创建卷', '在该库下创建一个卷'], ['DV2', '查看卷列表', '查看该库下的卷列表'], ['MD1', '查看模型列表', '查看该库下的模型列表'], ['OP1', '查看算子列表', '查看该库下的算子列表']],
    '表': [['DT3', '修改表 / ALTER TABLE', '修改该表的表名 / 备注 / 定义'], ['DT4', '删除表 / DROP TABLE', '删除该表'], ['DT8', '表查询 / SELECT', '对该表执行 SELECT'], ['DT9', '表写入 / INSERT', '对该表执行 INSERT'], ['DT10', '表更新 / UPDATE', '对该表执行 UPDATE'], ['DT11', '表删除 / DELETE', '对该表执行 DELETE'], ['DT12', '表清除 / TRUNCATE', '对该表执行 TRUNCATE'], ['DT13', '表引用 / REFERENCE', '允许将该表引用为外键约束的唯一 / 主键表'], ['DT14', '表索引 / INDEX', '创建或删除该表的索引']],
    '卷': [['DV3', '修改卷', '修改该卷信息'], ['DV4', '删除卷', '删除该卷'], ['DV5', '卷查询', '查看该卷中的数据（文件列表 / 详情 / 预览 / 下载）'], ['DV6', '卷写入', '向该卷写入、修改或删除数据']],
    '模型': [['MD3', '修改模型', '修改该模型'], ['MD4', '删除模型', '删除该模型'], ['MD5', '设为默认模型', '将该模型设为默认']],
    '算子': [['OP3', '修改算子', '修改该算子（代码 / 参数定义，须先取消发布）'], ['OP4', '删除算子', '删除该算子'], ['OP5', '发布算子', '把该算子发布为可调用的 API 服务'], ['OP6', '使用算子', '在工作区中选择该算子、通过访问凭证调用其 API 服务']]
  };
  var ALL_ROLES = ['超级管理员', '工作区管理员', '数据开发者', '智能体开发者', '只读成员', '部门经理'];
  var GLOBAL_ROLES = ['超级管理员', '工作区管理员']; // mock：拥有全局权限（生效范围 = 全部）的角色
  var ROLE_DESCS = { '超级管理员': '系统内置，拥有所有权限', '工作区管理员': '管理工作区资源和成员', '数据开发者': '数据模式完整权限', '智能体开发者': '智能体模式完整权限', '只读成员': '仅可查看', '部门经理': '继承数据开发者权限' };

  var store = {};   // objType||objName -> { code: [{role, global}] }
  var cur = { type: null, name: null };
  var addCode = null;

  function esc(s) { return String(s == null ? '' : s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;').replace(/'/g, '&#39;'); }

  function ensureDom() {
    if (document.getElementById('opmModal')) return;
    var css = '.opm-ov{display:none;position:fixed;inset:0;background:rgba(0,0,0,0.45);z-index:400;align-items:center;justify-content:center}'
      + '.opm-ov.show{display:flex}'
      + '.opm-box{background:#fff;border-radius:12px;box-shadow:0 6px 30px rgba(0,0,0,0.12);display:flex;flex-direction:column;max-height:85vh}'
      + '.opm-head{display:flex;align-items:center;justify-content:space-between;padding:16px 22px;border-bottom:1px solid #f0f0f0}'
      + '.opm-x{border:none;background:none;font-size:16px;color:rgba(0,0,0,0.45);cursor:pointer}';
    var st = document.createElement('style'); st.textContent = css; document.head.appendChild(st);
    var d1 = document.createElement('div');
    d1.className = 'opm-ov'; d1.id = 'opmModal';
    d1.onclick = function (e) { if (e.target === d1) close(); };
    d1.innerHTML = '<div class="opm-box" style="width:760px;max-width:92vw">'
      + '<div class="opm-head"><span id="opmTitle" style="font-size:15px;font-weight:600">权限设置</span><button class="opm-x" onclick="ObjPerm.close()">✕</button></div>'
      + '<div id="opmBody" style="padding:6px 20px 16px;max-height:60vh;overflow:auto"></div>'
      + '<div style="padding:12px 20px 16px;display:flex;justify-content:space-between;align-items:center;gap:16px;border-top:1px solid #f0f0f0">'
      + '<span style="font-size:12px;color:rgba(0,0,0,0.4)">带「全局」标记的角色来自全局权限（生效范围 = 全部），请在「用户权限 › 角色权限」中调整；此处仅管理该对象的对象级授权。</span>'
      + '<button style="height:32px;padding:0 18px;flex-shrink:0;border:none;border-radius:6px;background:#1677ff;color:#fff;font-size:13px;cursor:pointer" onclick="ObjPerm.close()">完成</button>'
      + '</div></div>';
    document.body.appendChild(d1);
    var d2 = document.createElement('div');
    d2.className = 'opm-ov'; d2.id = 'opmRoleModal'; d2.style.zIndex = '410';
    d2.onclick = function (e) { if (e.target === d2) closeRole(); };
    d2.innerHTML = '<div class="opm-box" style="width:460px;max-width:90vw">'
      + '<div class="opm-head"><span id="opmRoleTitle" style="font-size:15px;font-weight:600">添加角色</span><button class="opm-x" onclick="ObjPerm.closeRole()">✕</button></div>'
      + '<div id="opmRoleList" style="padding:16px 20px 8px;max-height:52vh;overflow:auto"></div>'
      + '<div style="padding:12px 20px 16px;display:flex;justify-content:flex-end;gap:10px;border-top:1px solid #f0f0f0">'
      + '<button style="height:32px;padding:0 16px;border:1px solid #d9d9d9;border-radius:6px;background:#fff;cursor:pointer;font-size:13px" onclick="ObjPerm.closeRole()">取消</button>'
      + '<button style="height:32px;padding:0 18px;border:none;border-radius:6px;background:#1677ff;color:#fff;font-size:13px;cursor:pointer" onclick="ObjPerm.confirmAdd()">添 加</button>'
      + '</div></div>';
    document.body.appendChild(d2);
  }

  function key() { return cur.type + '||' + cur.name; }
  function initData() {
    var k = key();
    if (store[k]) return store[k];
    var d = {};
    (PERM_SETS[cur.type] || []).forEach(function (p) {
      d[p[0]] = GLOBAL_ROLES.map(function (r) { return { role: r, global: true }; });
    });
    store[k] = d;
    return d;
  }

  function render() {
    var data = initData();
    var perms = PERM_SETS[cur.type] || [];
    var h = '<table style="width:100%;border-collapse:collapse">';
    h += '<thead><tr><th style="text-align:left;padding:10px 12px;border-bottom:1px solid #f0f0f0;font-size:12px;color:rgba(0,0,0,0.45);width:220px">权限</th><th style="text-align:left;padding:10px 12px;border-bottom:1px solid #f0f0f0;font-size:12px;color:rgba(0,0,0,0.45)">角色</th></tr></thead><tbody>';
    perms.forEach(function (p) {
      var code = p[0];
      h += '<tr><td style="padding:14px 12px;border-bottom:1px solid #f7f7f7;vertical-align:top">';
      h += '<div style="font-size:13px;font-weight:600;color:rgba(0,0,0,0.82)">' + code + '-' + p[1] + '</div>';
      h += '<div style="font-size:11px;color:rgba(0,0,0,0.4);margin-top:2px">' + p[2] + '</div></td>';
      h += '<td style="padding:12px;border-bottom:1px solid #f7f7f7"><div style="display:flex;flex-wrap:wrap;gap:8px;align-items:center">';
      (data[code] || []).forEach(function (g) {
        var check = '<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="#52c41a" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M8 12l3 3 5-6"/></svg>';
        if (g.global) {
          h += '<span title="通过全局权限（生效范围 = 全部）获得，请在「角色权限」中调整，不能在此移除" style="display:inline-flex;align-items:center;gap:6px;border:1px solid #e8e9ec;background:#fafafa;border-radius:8px;padding:5px 10px;font-size:12.5px;color:rgba(0,0,0,0.65)">' + check + esc(g.role) + '<span style="font-size:10px;color:#1677ff;background:#eef4ff;border-radius:6px;padding:0 5px">全局</span></span>';
        } else {
          h += '<span style="display:inline-flex;align-items:center;gap:6px;border:1px solid #d6e4ff;background:#f0f5ff;border-radius:8px;padding:5px 10px;font-size:12.5px;color:rgba(0,0,0,0.75)">' + check + esc(g.role) + '<span style="cursor:pointer;color:rgba(0,0,0,0.3);font-size:13px;line-height:1" title="移除该对象授权" onclick="ObjPerm.removeRole(\'' + code + '\',\'' + esc(g.role) + '\')">✕</span></span>';
        }
      });
      h += '<button title="为该权限添加角色（对象级授权）" style="width:30px;height:30px;border:1px solid #e5e6eb;border-radius:8px;background:#fff;cursor:pointer;font-size:15px;color:rgba(0,0,0,0.55)" onclick="ObjPerm.addRole(\'' + code + '\')">＋</button>';
      h += '</div></td></tr>';
    });
    h += '</tbody></table>';
    document.getElementById('opmBody').innerHTML = h;
  }

  function open(objType, objName) {
    if (!PERM_SETS[objType]) { alert('未定义「' + objType + '」的对象权限集'); return; }
    ensureDom();
    cur = { type: objType, name: objName };
    document.getElementById('opmTitle').textContent = '权限设置 · ' + objName;
    render();
    document.getElementById('opmModal').classList.add('show');
  }
  function close() { var m = document.getElementById('opmModal'); if (m) m.classList.remove('show'); }
  function closeRole() { var m = document.getElementById('opmRoleModal'); if (m) m.classList.remove('show'); addCode = null; }

  function addRole(code) {
    var data = initData();
    var used = (data[code] || []).map(function (g) { return g.role; });
    var avail = ALL_ROLES.filter(function (r) { return used.indexOf(r) === -1; });
    if (!avail.length) { alert('所有角色均已拥有该权限'); return; }
    addCode = code;
    var p = (PERM_SETS[cur.type] || []).find(function (x) { return x[0] === code; });
    document.getElementById('opmRoleTitle').textContent = '添加角色 · ' + code + '-' + (p ? p[1] : '');
    document.getElementById('opmRoleList').innerHTML = avail.map(function (r) {
      return '<label style="display:flex;align-items:center;gap:10px;padding:10px 14px;border:1px solid #f0f0f0;border-radius:8px;margin-bottom:8px;cursor:pointer" onmouseover="this.style.borderColor=\'#b9ccff\'" onmouseout="this.style.borderColor=\'#f0f0f0\'">'
        + '<input type="checkbox" value="' + esc(r) + '" style="width:15px;height:15px">'
        + '<span style="flex:1"><span style="font-size:13px;font-weight:600;color:rgba(0,0,0,0.82)">' + esc(r) + '</span>'
        + '<span style="font-size:12px;color:rgba(0,0,0,0.4);margin-left:10px">' + (ROLE_DESCS[r] || '') + '</span></span></label>';
    }).join('');
    document.getElementById('opmRoleModal').classList.add('show');
  }
  function confirmAdd() {
    var checked = [].slice.call(document.querySelectorAll('#opmRoleList input:checked')).map(function (c) { return c.value; });
    if (!checked.length) { alert('请选择至少一个角色'); return; }
    var data = initData();
    checked.forEach(function (r) { data[addCode].push({ role: r, global: false }); });
    closeRole();
    render();
  }
  function removeRole(code, role) {
    var data = initData();
    var g = (data[code] || []).find(function (x) { return x.role === role; });
    if (!g || g.global) return;
    if (!confirm('移除角色「' + role + '」对该对象的「' + code + '」授权？')) return;
    data[code] = data[code].filter(function (x) { return x.role !== role; });
    render();
  }

  return { open: open, close: close, closeRole: closeRole, addRole: addRole, confirmAdd: confirmAdd, removeRole: removeRole };
})();
