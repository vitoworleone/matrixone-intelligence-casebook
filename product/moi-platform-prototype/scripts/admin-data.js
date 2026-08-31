/**
 * 管理后台共享数据
 * - window.ADMIN_USERS：业务用户（客户账号），与「管理员账户」区分
 * - window.USER_TAGS：用户分组标签（多对多关系，uids 存储该标签下的用户 id）
 *
 * 多页共享：user-management.html 维护数据；billing-management.html 读取使用。
 * 同会话内的修改通过 window 全局对象立即可见（同一标签页跳转生效）。
 */
(function() {
  if (window.ADMIN_USERS) return; // 防止重复加载

  window.ADMIN_USERS = [
    { id: 'U10000001', name: '演示用户甲',         phone: '13901234567',     email: '公开演示环境', balance: 1286,  org: 'MOI·研发',  registeredAt: '2025-08-12 09:14:32', status: true,  lastLogin: '2026-04-09 15:32' },
    { id: 'U10000002', name: '演示用户乙',         phone: '13802345678',     email: '公开演示环境', balance: 8420,  org: 'MOI·运营',  registeredAt: '2025-09-05 14:08:51', status: true,  lastLogin: '2026-04-09 14:08' },
    { id: 'U10000003', name: '演示用户丙',         phone: '13703456789',     email: '公开演示环境',        balance: 53210, org: '示例教育集团·AI 中台',  registeredAt: '2025-11-20 16:42:07', status: true,  lastLogin: '2026-04-09 11:24' },
    { id: 'U10000004', name: '演示用户丁',           phone: '13604567890',     email: '公开演示环境',        balance: 2150,  org: '示例教育集团·编校',     registeredAt: '2026-01-10 10:25:18', status: true,  lastLogin: '2026-04-08 17:05' },
    { id: 'U10000005', name: '演示用户戊',           phone: '13505678901',     email: '公开演示环境',     balance: 17600, org: '示例医药企业',        registeredAt: '2025-10-28 11:36:44', status: true,  lastLogin: '2026-04-09 09:50' },
    { id: 'U10000006', name: 'Sarah Johnson',  phone: '+1-415-555-0142', email: '公开演示环境',               balance: 24560, org: 'Acme Inc.',      registeredAt: '2025-12-03 22:51:09', status: true,  lastLogin: '2026-04-09 02:14' },
    { id: 'U10000007', name: '演示用户己',         phone: '13406789012',     email: '公开演示环境',   balance: 980,   org: 'MOI·渠道',  registeredAt: '2026-02-14 08:47:23', status: true,  lastLogin: '2026-04-08 22:36' },
    { id: 'U10000008', name: '赵敏',           phone: '13307890123',     email: '公开演示环境',        balance: 0,     org: '个人开发者',     registeredAt: '2026-02-01 19:32:56', status: false, lastLogin: '2026-03-15 10:21' },
    { id: 'U10000009', name: 'Carlos Rivera',  phone: '+34-612-345-678', email: '公开演示环境',              balance: 6420,  org: 'Búho Studio',    registeredAt: '2026-03-18 03:14:02', status: true,  lastLogin: '2026-04-07 08:42' },
    { id: 'U10000010', name: '演示只读 A',           phone: '13208901234',     email: '公开演示环境',      balance: 350,   org: '个人开发者',     registeredAt: '2026-04-02 13:58:41', status: true,  lastLogin: '2026-04-09 13:58' }
  ];

  window.USER_TAGS = [
    { id: 'tag-mo',       name: 'MOI·全员',  color: '#722ed1', desc: '内部员工',          uids: ['U10000001','U10000002','U10000007'] },
    { id: 'tag-fltrp',    name: '示例教育集团·全部',    color: '#1677ff', desc: '示例教育集团所有账户',    uids: ['U10000003','U10000004'] },
    { id: 'tag-poc',      name: 'PoC 客户',       color: '#fa8c16', desc: '试用阶段企业客户',  uids: ['U10000003','U10000005','U10000006'] },
    { id: 'tag-personal', name: '个人开发者',     color: '#13c2c2', desc: '免费/低消费用户',   uids: ['U10000008','U10000010'] },
    { id: 'tag-overseas', name: '海外用户',       color: '#eb2f96', desc: '$ 区域账号',        uids: ['U10000006','U10000009'] },
    { id: 'tag-vip',      name: 'VIP 用户',       color: '#fa541c', desc: '余额 > 10,000 cr', uids: ['U10000003','U10000005','U10000006'] }
  ];

  // 工具方法
  window.getUserById = function(uid) {
    return window.ADMIN_USERS.find(function(u) { return u.id === uid; }) || null;
  };
  window.getTagsForUser = function(uid) {
    return window.USER_TAGS.filter(function(t) { return t.uids.indexOf(uid) !== -1; });
  };
  window.getTagById = function(tid) {
    return window.USER_TAGS.find(function(t) { return t.id === tid; }) || null;
  };
})();
