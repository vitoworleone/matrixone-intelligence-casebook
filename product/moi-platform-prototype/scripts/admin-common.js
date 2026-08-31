// Admin sidebar builder — accordion style
(function() {
  var path = window.location.pathname;
  var hash = window.location.hash.replace('#', '');

  var menu = [
    { id: 'overview', label: '总览', href: 'index.html', icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>' },
    { id: 'aistudio', label: 'AI Studio', icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="3"/><path d="M8 12h8M12 8v8"/></svg>',
      page: 'aistudio-management.html',
      children: [
        { id: 'as-models', label: 'AI 服务', hash: 'models' },
        { id: 'as-compute', label: '计算资源', hash: 'compute' }
      ]
    },
    { id: 'taas', label: 'Genesis', icon: '<svg viewBox="0 0 48 48" fill="none" stroke="currentColor"><ellipse cx="24" cy="24" rx="17" ry="8" stroke-width="2.6" transform="rotate(-45 24 24)"/><path d="M15 33 L15 15 L33 33 L33 15" stroke-width="3.8" stroke-linecap="round" stroke-linejoin="round"/><circle cx="15" cy="15" r="3.6" fill="currentColor" stroke="none"/><circle cx="33" cy="33" r="3.6" fill="currentColor" stroke="none"/></svg>',
      page: 'taas-management.html',
      children: [
        { id: 'taas-dashboard', label: '运营看板', hash: 'dashboard' },
        { id: 'taas-channels', label: '渠道管理', hash: 'channels' },
        { id: 'taas-models', label: '模型管理', hash: 'models' },
        { id: 'taas-plans', label: '企业方案', hash: 'plans' },
        { id: 'taas-tokens', label: '密钥管理', hash: 'tokens' },
        { id: 'taas-usage', label: '用量统计', hash: 'usage' },
        { id: 'taas-logs', label: '请求日志', hash: 'logs' },
        { id: 'taas-notifications', label: '告警通知', hash: 'notifications' },
        { id: 'taas-audit', label: '操作审计', hash: 'audit' }
      ]
    },
    { id: 'billing', label: '计费中心', icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="5" width="20" height="14" rx="2"/><line x1="2" y1="10" x2="22" y2="10"/><line x1="6" y1="15" x2="10" y2="15"/></svg>',
      page: 'billing-management.html',
      children: [
        { id: 'billing-overview', label: '费用总览', hash: 'overview' },
        { id: 'billing-usage', label: '用量明细', hash: 'usage' },
        { id: 'billing-transactions', label: '收支明细', hash: 'transactions' },
        { id: 'billing-pricing', label: '价格设置', hash: 'pricing' }
      ]
    },
    { id: 'users', label: '用户管理', href: 'user-management.html', icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>' },
    { id: 'operations', label: '产品运营', icon: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 20V10M18 20V4M6 20v-4"/></svg>',
      page: 'operations-management.html',
      children: [
        { id: 'op-feedback', label: '用户反馈', hash: 'feedback' },
        { id: 'op-rewards', label: '奖励中心', hash: 'rewards' }
      ]
    }
  ];

  function isOnPage(href) {
    if (!href) return false;
    return path.indexOf(href) !== -1 || (href === 'index.html' && (path.endsWith('/admin/') || path.endsWith('/admin')));
  }

  function buildSidebar() {
    var sidebar = document.getElementById('adminSidebar');
    if (!sidebar) return;

    var html = '<div class="admin-sidebar-title">'
      + '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>'
      + '管理后台</div>';

    menu.forEach(function(item) {
      if (item.children) {
        // Has sub-menu — check if any child is active
        var pageStem = (item.page || '').replace('.html', '');
        var isOnGroupPage = pageStem && path.indexOf(pageStem) !== -1;
        var isExpanded = isOnGroupPage;
        var activeChildId = '';
        if (isOnGroupPage) {
          var defaultHash = item.children[0].hash;
          item.children.forEach(function(child) {
            if (hash === child.hash || (!hash && child.hash === defaultHash)) activeChildId = child.id;
          });
        }

        html += '<div class="admin-nav-parent' + (isExpanded ? ' expanded' : '') + '" data-group="' + item.id + '" data-page="' + (item.page || '') + '" onclick="toggleAdminGroup(\'' + item.id + '\')">'
          + item.icon + ' ' + item.label
          + '</div>';
        html += '<div class="admin-nav-children" id="adminGroup-' + item.id + '" style="' + (isExpanded ? '' : 'display:none') + '">';
        item.children.forEach(function(child) {
          var isActive = activeChildId === child.id;
          html += '<a class="admin-nav-child' + (isActive ? ' active' : '') + '" href="' + item.page + '#' + child.hash + '" onclick="activateChild(this,\'' + child.hash + '\',\'' + item.page + '\');return false;">' + child.label + '</a>';
        });
        html += '</div>';
      } else {
        // No sub-menu — direct link
        var isActive = isOnPage(item.href);
        html += '<a class="admin-nav-item' + (isActive ? ' active' : '') + '" href="' + item.href + '">'
          + item.icon + ' ' + item.label + '</a>';
      }
    });

    html += '<div class="admin-sidebar-bottom"></div>';
    sidebar.innerHTML = html;
  }

  // Toggle accordion group
  window.toggleAdminGroup = function(groupId) {
    var children = document.getElementById('adminGroup-' + groupId);
    var parent = document.querySelector('[data-group="' + groupId + '"]');
    if (!children || !parent) return;

    var isOpen = children.style.display !== 'none';

    // Close all groups
    document.querySelectorAll('.admin-nav-children').forEach(function(el) { el.style.display = 'none'; });
    document.querySelectorAll('.admin-nav-parent').forEach(function(el) { el.classList.remove('expanded'); });
    // Deactivate all direct nav items
    document.querySelectorAll('.admin-nav-item').forEach(function(el) { el.classList.remove('active'); });

    if (!isOpen) {
      children.style.display = '';
      parent.classList.add('expanded');
      // 跳转到该 group 的承载页（取该 group 第一个子项作为默认 hash）
      var page = parent.getAttribute('data-page');
      var firstChild = children.querySelector('.admin-nav-child');
      var defaultHash = firstChild ? firstChild.getAttribute('href').split('#')[1] : '';
      if (page && path.indexOf(page.replace('.html','')) === -1) {
        window.location.href = page + '#' + defaultHash;
      }
    }
  };

  // Activate a child menu item — same page (hash change) vs jumping to another page
  window.activateChild = function(el, hash, page) {
    if (page && path.indexOf(page.replace('.html','')) === -1) {
      window.location.href = page + '#' + hash;
      return;
    }
    window.location.hash = hash;
    document.querySelectorAll('.admin-nav-child').forEach(function(c) { c.classList.remove('active'); });
    el.classList.add('active');
  };

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', buildSidebar);
  } else {
    buildSidebar();
  }
})();
