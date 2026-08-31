/* ============================================================
   官网共享导航栏
   引入方式：
     <link rel="stylesheet" href="nav.css">
     <div id="navMount"></div>
     <script src="nav.js"></script>
   登录入口可通过 window.NAV_LOGIN_FROM = 'taas' 传 from 标记，
   或自动从 pathname 推断（含 taas 时自动加 ?from=taas）
   ============================================================ */
(function(){
  // ===== 用户状态 =====
  function isLoggedIn() { return localStorage.getItem('moi_user') !== null; }
  function getUser() { var u = localStorage.getItem('moi_user'); return u ? JSON.parse(u) : null; }
  function logout() { localStorage.removeItem('moi_user'); renderNav(); if (typeof renderHero === 'function') renderHero(); }
  // 暴露给页面用
  window.isLoggedIn = isLoggedIn;
  window.getUser = getUser;
  window.logout = logout;

  // ===== 登录链接（自动加 from 参数） =====
  function loginUrl(){
    var from = window.NAV_LOGIN_FROM;
    if (!from && /taas/i.test(location.pathname)) from = 'taas';
    return 'login.html' + (from ? '?from=' + from : '');
  }

  // ===== 主导航菜单（中部）· 公开演示版不跳转到外部站点 =====
  var MO = '#';
  var ARROW_R = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="5" y1="12" x2="19" y2="12"/><polyline points="12 5 19 12 12 19"/></svg>';
  var ARROW_TR = '<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" style="margin-left:3px;vertical-align:-1px"><path d="M7 17L17 7M9 7h8v8"/></svg>';

  function _capItem(href, name, desc, target){
    var blocked = /^(?:https?:|mailto:|#)/i.test(href);
    var t = target && !blocked ? ' target="_blank" rel="noopener"' : '';
    var action = blocked ? ' onclick="return false;" aria-disabled="true"' : '';
    return '<a class="dd-cap-item" href="' + (blocked ? '#' : href) + '"' + t + action + '>'
      + '<div class="dd-cap-name">' + name + '</div>'
      + '<div class="dd-cap-desc">' + desc + '</div>'
      + '</a>';
  }
  function _sideItem(icon, name, desc, href, target){
    var blocked = /^(?:https?:|mailto:|#)/i.test(href);
    var t = target && !blocked ? ' target="_blank" rel="noopener"' : '';
    var arrow = target && !blocked ? ARROW_TR : '';
    var action = blocked ? ' onclick="return false;" aria-disabled="true"' : '';
    return '<a class="dd-side-item" href="' + (blocked ? '#' : href) + '"' + t + action + '>'
      + '<div class="dd-side-icon">' + icon + '</div>'
      + '<div class="dd-side-text">'
      +   '<div class="dd-side-name">' + name + arrow + '</div>'
      +   '<div class="dd-side-desc">' + desc + '</div>'
      + '</div>'
      + '</a>';
  }
  function _industryItem(icon, name){
    return '<a class="dd-ind-item" href="#"><span class="dd-ind-icon">' + icon + '</span>' + name + '</a>';
  }

  // ---- 产品 dropdown：MOI 主项 + 6 能力 + 右侧基础设施 3 项 ----
  var PRODUCT_HTML = ''
    + '<div class="nav-dropdown dd-product">'
    +   '<div class="dd-product-main">'
    +     '<a class="dd-feat-card" href="../index.html">'
    +       '<div class="dd-feat-icon"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 1.98-3A2.5 2.5 0 0 1 9.5 2z"/><path d="M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-1.98-3A2.5 2.5 0 0 0 14.5 2z"/></svg></div>'
    +       '<div class="dd-feat-text"><div class="dd-feat-name">MatrixOne Intelligence</div><div class="dd-feat-desc">AI 原生数据智能平台 — 从数据底座到智能应用的一站式方案</div></div>'
    +       '<div class="dd-feat-arrow">' + ARROW_R + '</div>'
    +     '</a>'
    +     '<div class="dd-cap-grid">'
    +       _capItem('#', 'Data Governance', '统一数据治理、血缘追溯与合规管理')
    +     _capItem('#', 'Data Warehousing', '云原生 HTAP 数仓，一库支撑所有负载')
    +     _capItem('#', 'Data Pipeline', '50+ 连接器，可视化 ETL 编排')
    +     _capItem('#', 'Document Intelligence', '智能文档解析、抽取与知识库构建')
    +     _capItem('#', 'AI Agent Builder', '构建生产级 AI Agent，集成 RAG 与工具调用')
    +     _capItem('#', 'Enterprise Data Agent', '自然语言驱动的跨源数据分析智能体')
    +     '</div>'
    +   '</div>'
    +   '<div class="dd-product-side">'
    +     '<div class="dd-side-title">基础设施</div>'
    +   _sideItem('<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z"/><path d="M12 15l-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z"/></svg>', 'Astra', '工程化的 Agent Harness', MO + '/astra', true)
    +   _sideItem('<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9.5 2A2.5 2.5 0 0 1 12 4.5v15a2.5 2.5 0 0 1-4.96.44 2.5 2.5 0 0 1-2.96-3.08 3 3 0 0 1-.34-5.58 2.5 2.5 0 0 1 1.32-4.24 2.5 2.5 0 0 1 1.98-3A2.5 2.5 0 0 1 9.5 2z"/><path d="M14.5 2A2.5 2.5 0 0 0 12 4.5v15a2.5 2.5 0 0 0 4.96.44 2.5 2.5 0 0 0 2.96-3.08 3 3 0 0 0 .34-5.58 2.5 2.5 0 0 0-1.32-4.24 2.5 2.5 0 0 0-1.98-3A2.5 2.5 0 0 0 14.5 2z"/></svg>', 'Memoria', 'AI Agent 持久记忆', 'https://thememoria.ai', true)
    +   '</div>'
    + '</div>';

  // ---- 解决方案 dropdown：AI Factory banner + 3 列 ----
  var SOLUTION_HTML = ''
    + '<div class="nav-dropdown dd-solution">'
    +   '<a class="dd-banner" href="' + MO + '/whitepaper" target="_blank" rel="noopener">'
    +     '<div class="dd-banner-icon"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 20a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8l-7-5-7 5v7"/><path d="M17 18h1"/><path d="M12 18h1"/><path d="M7 18h1"/></svg></div>'
    +     '<div class="dd-banner-text">'
    +       '<div class="dd-banner-title">AI Factory <span class="dd-banner-tag">与 NVIDIA 联合推出</span></div>'
    +       '<div class="dd-banner-desc">从算力基础设施到智能应用的端到端企业 AI 工厂解决方案</div>'
    +     '</div>'
    +     '<div class="dd-banner-arrow">了解更多 ' + ARROW_R + '</div>'
    +   '</a>'
    +   '<div class="dd-3col">'
    +     '<div>'
    +       '<div class="dd-group-title">AI 数据能力</div>'
    +       _sideItem('<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>', '文档智能', '智能解析、提取与理解', MO + '/solution/document-intelligence', true)
    +     _sideItem('<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>', 'Agentic RAG', 'Agent 驱动的检索增强', MO + '/solution/agentic-rag', true)
    +     _sideItem('<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2"/><rect x="9" y="9" width="6" height="6"/><line x1="9" y1="2" x2="9" y2="4"/><line x1="15" y1="2" x2="15" y2="4"/><line x1="9" y1="20" x2="9" y2="22"/><line x1="15" y1="20" x2="15" y2="22"/><line x1="20" y1="9" x2="22" y2="9"/><line x1="20" y1="14" x2="22" y2="14"/><line x1="2" y1="9" x2="4" y2="9"/><line x1="2" y1="14" x2="4" y2="14"/></svg>', '大模型微调', '定制化模型训练管线', MO + '/solution/llm-finetuning', true)
    +     _sideItem('<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>', '多模态数据 ETL', '文本、图像、音视频统一处理', MO + '/solution/data-etl', true)
    +     _sideItem('<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 22 8.5 22 15.5 12 22 2 15.5 2 8.5 12 2"/><line x1="12" y1="22" x2="12" y2="15.5"/><polyline points="22 8.5 12 15.5 2 8.5"/></svg>', 'AI 数据底座', '统一的 AI 数据基础设施', MO + '/solution/ai-data-foundation', true)
    +     '</div>'
    +     '<div>'
    +       '<div class="dd-group-title">AI 智能体应用</div>'
    +       _sideItem('<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 6 13.5 15.5 8.5 10.5 1 18"/><polyline points="17 6 23 6 23 12"/></svg>', 'AI 投资情报', '多源数据驱动投研决策', '#')
    +     _sideItem('<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><polyline points="9 15 11 17 15 13"/></svg>', '智能标书', '自动化标书生成与审核', '#')
    +     _sideItem('<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>', '合同审核', 'AI 风险识别与条款分析', '#')
    +     _sideItem('<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>', '智能问数', '自然语言数据查询', '#')
    +     _sideItem('<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="8" y1="9" x2="16" y2="9"/><line x1="8" y1="13" x2="16" y2="13"/><line x1="8" y1="17" x2="12" y2="17"/></svg>', '智能填表', '自动提取填充表单', '#')
    +     '</div>'
    +     '<div>'
    +       '<div class="dd-group-title">行业方案</div>'
    +       _industryItem('<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 21h18M5 21V7l7-4 7 4v14M9 9h.01M9 12h.01M9 15h.01M15 9h.01M15 12h.01M15 15h.01"/></svg>', '金融服务')
    +     _industryItem('<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 22V11l5-3v14M7 22V8l5-3v17M12 22V5l5-3v20M17 22V2l5 3v17"/></svg>', '制造业')
    +     _industryItem('<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>', '零售消费')
    +     _industryItem('<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 22h20M3 22V10l9-7 9 7v12M9 22v-6h6v6"/></svg>', '政务与公共服务')
    +     _industryItem('<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>', '医疗健康')
    +     _industryItem('<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="3" width="16" height="16" rx="2"/><path d="M4 11h16M9 3v8M15 3v8"/><circle cx="9" cy="22" r="1"/><circle cx="15" cy="22" r="1"/></svg>', '通信与交通')
    +     '</div>'
    +   '</div>'
    + '</div>';

  // ---- 客户 dropdown：为什么选择我们 + 客户案例 + 4 case ----
  var CUSTOMER_HTML = ''
    + '<div class="nav-dropdown dd-customer">'
    +   '<div class="dd-cust-left">'
    +     '<div class="dd-group-title">为什么选择我们</div>'
    +     _sideItem('<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>', 'AI 战略转化为业务 ROI', '让 AI 投资产生可量化的业务回报', '#')
    +   _sideItem('<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>', '企业数据与知识飞轮', '一个平台集中、沉淀、激活全部企业数据与知识', '#')
    +   _sideItem('<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>', '高性价比可落地方案', '以传统方案几分之一的成本交付企业级 AI 能力', '#')
    +   '</div>'
    +   '<div class="dd-cust-right">'
    +     '<a class="dd-feat-card dd-feat-card-sm" href="' + MO + '/customers" target="_blank" rel="noopener">'
    +       '<div class="dd-feat-icon"><svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg></div>'
    +       '<div class="dd-feat-text"><div class="dd-feat-name">客户案例</div><div class="dd-feat-desc">了解各行业领先企业的成功实践</div></div>'
    +       '<div class="dd-feat-arrow">' + ARROW_R + '</div>'
    +     '</a>'
    +     '<div class="dd-case-grid">'
    +       '<a class="dd-case-item" href="' + MO + '/case/health-supplement-ai-platform" target="_blank" rel="noopener">示例零售集团 — 企业 AI 基座 <span class="dd-case-tag" style="background:#fff7e6;color:#fa8c16">零售</span></a>'
    +     '<a class="dd-case-item" href="' + MO + '/case/extreme-vision-feature-platform" target="_blank" rel="noopener">示例视觉智能企业 — AI 特征平台 <span class="dd-case-tag" style="background:#f9f0ff;color:#722ed1">AI</span></a>'
    +     '<a class="dd-case-item" href="' + MO + '/case/stone-castle-investment" target="_blank" rel="noopener">示例投资机构 — 投研智能 <span class="dd-case-tag" style="background:#f0f5ff;color:#1677ff">金融</span></a>'
    +     '<a class="dd-case-item" href="' + MO + '/case/smart-city-traffic-platform" target="_blank" rel="noopener">示例智慧城市平台 — 政务语料治理 <span class="dd-case-tag" style="background:#f6ffed;color:#52c41a">政务</span></a>'
    +     '</div>'
    +   '</div>'
    + '</div>';

  // ---- 资源 dropdown：学习 + 内容 + 社区 3 列 ----
  var RESOURCE_HTML = ''
    + '<div class="nav-dropdown dd-resource">'
    +   '<div>'
    +     '<div class="dd-group-title">学习</div>'
    +     _sideItem('<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M2 3h6a4 4 0 0 1 4 4v14a3 3 0 0 0-3-3H2z"/><path d="M22 3h-6a4 4 0 0 0-4 4v14a3 3 0 0 1 3-3h7z"/></svg>', '文档中心', '产品文档与 API 参考', 'https://docs.moi-demo.cn', true)
    +   '</div>'
    +   '<div>'
    +     '<div class="dd-group-title">内容</div>'
    +     _sideItem('<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>', '博客', '技术深度解读与最佳实践', MO + '/blog', true)
    +   _sideItem('<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 2v7.31"/><path d="M14 9.3V1.99"/><path d="M8.5 2h7"/><path d="M14 9.3a6.5 6.5 0 1 1-4 0"/></svg>', '研究', '前沿技术研究报告', MO + '/research', true)
    +   _sideItem('<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="12" y1="18" x2="12" y2="12"/><line x1="9" y1="15" x2="12" y2="18"/><line x1="15" y1="15" x2="12" y2="18"/></svg>', '白皮书', '深度行业洞察与方案', MO + '/whitepaper', true)
    +   '</div>'
    +   '<div>'
    +     '<div class="dd-group-title">社区</div>'
    +     _sideItem('<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"/></svg>', 'GitHub', '开源项目与代码贡献', 'https://github.com/moi-demo/matrixone', true)
    +   '</div>'
    + '</div>';

  // ---- 公司 dropdown：3 项横排 ----
  var COMPANY_HTML = ''
    + '<div class="nav-dropdown dd-company">'
    +   _sideItem('<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><path d="M20 8v6M23 11h-6"/></svg>', '关于我们', '了解 MOI 的愿景与使命', MO, true)
    + _sideItem('<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M9 12l2 2 4-4"/><path d="M5 7c0-1.1.9-2 2-2h10a2 2 0 0 1 2 2v12l-7-3-7 3z"/></svg>', '生态合作', '携手共建 AI 数据生态', MO + '/partnership', true)
    + _sideItem('<svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="7" width="20" height="14" rx="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>', '加入我们', '探索职业机会，共创未来', 'https://www.zhipin.com/gongsi/job/59a68fecfe392c0d1nd93N-4EVU~.html', true)
    + '</div>';

  var MENU_HTML = ''
    + '<div class="nav-dropdown-wrap"><span class="nav-menu-item">产品 <span class="caret">▾</span></span>' + PRODUCT_HTML + '</div>'
    + '<div class="nav-dropdown-wrap"><span class="nav-menu-item">解决方案 <span class="caret">▾</span></span>' + SOLUTION_HTML + '</div>'
    + '<div class="nav-dropdown-wrap"><span class="nav-menu-item">客户 <span class="caret">▾</span></span>' + CUSTOMER_HTML + '</div>'
    + '<div class="nav-dropdown-wrap"><span class="nav-menu-item">资源 <span class="caret">▾</span></span>' + RESOURCE_HTML + '</div>'
    + '<div class="nav-dropdown-wrap"><span class="nav-menu-item">公司 <span class="caret">▾</span></span>' + COMPANY_HTML + '</div>';

  var GLOBE_SVG = '<svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><path d="M12 2a14.5 14.5 0 0 0 0 20 14.5 14.5 0 0 0 0-20"/><path d="M2 12h20"/></svg>';

  // ===== 右侧（按登录态切换） =====
  function rightHtml(){
    if (isLoggedIn()){
      var user = getUser();
      var email = user.email || (user.username ? user.username + '@moi-demo.cn' : (user.phone || '') + '@moi-demo.cn');
      var first = (user.name || 'U').charAt(0).toUpperCase();
      return ''
        + '<button class="nav-globe-btn" title="切换语言">'+GLOBE_SVG+'</button>'
        + '<button class="btn-trial-nav" onclick="location.href=\'../index.html\'">工作台</button>'
        + '<div class="user-avatar-wrap" id="avatarWrap">'
        +   '<span class="user-avatar-nav" id="avatarBtn" title="'+user.name+'">'+first+'</span>'
        +   '<div class="user-popover" id="userPopover">'
        +     '<div class="popover-header">'
        +       '<div class="popover-avatar">'+first+'</div>'
        +       '<div class="popover-info">'
        +         '<div class="popover-name">'+user.name+'</div>'
        +         '<div class="popover-email">'+email+'</div>'
        +       '</div>'
        +     '</div>'
        +     '<div class="popover-menu">'
        +       '<button class="popover-item" onclick="window.open(\'../account/account.html\',\'_blank\')">账号管理</button>'
        +       '<button class="popover-item" onclick="window.open(\'../account/billing.html\',\'_blank\')">计费中心</button>'
        +       '<button class="popover-item danger" onclick="logout()">退出登录</button>'
        +     '</div>'
        +   '</div>'
        + '</div>';
    } else {
      var url = loginUrl();
      return ''
        + '<button class="nav-globe-btn" title="切换语言">'+GLOBE_SVG+'</button>'
        + '<button class="btn-trial-nav" onclick="location.href=\''+url+'\'">免费试用</button>';
    }
  }

  function bindPopover(){
    var btn = document.getElementById('avatarBtn');
    var pop = document.getElementById('userPopover');
    if (!btn || !pop) return;
    btn.onclick = function(e){ e.stopPropagation(); pop.classList.toggle('show'); };
    document.addEventListener('click', function(e){
      if (pop.classList.contains('show') && !pop.contains(e.target) && e.target !== btn) {
        pop.classList.remove('show');
      }
    });
  }

  function renderNav(){
    var mount = document.getElementById('navMount');
    if (!mount) return;
    mount.innerHTML = ''
      + '<nav class="nav">'
      +   '<div class="nav-logo" role="img" aria-label="MatrixOne Intelligence">'
      +     '<img src="../images/logo-moi.svg" alt="MatrixOne Intelligence">'
      +   '</div>'
      +   '<div class="nav-menu">' + MENU_HTML + '</div>'
      +   '<div class="nav-right" id="navRight">' + rightHtml() + '</div>'
      + '</nav>';
    mount.onclick = function(e) {
      var link = e.target.closest('a[href^="#/"]');
      if (link) e.preventDefault();
    };
    bindPopover();
  }

  window.renderNav = renderNav;
  // 自动渲染
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', renderNav);
  } else {
    renderNav();
  }
})();
