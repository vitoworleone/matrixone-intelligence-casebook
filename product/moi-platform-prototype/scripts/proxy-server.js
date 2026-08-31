// Local dev server: static files + web page proxy with JS rendering
// Usage: node html/scripts/proxy-server.js
// Open:  http://localhost:3001/data-connection/data-import-create.html

const http = require('http');
const https = require('https');
const net = require('net');
const tls = require('tls');
const url = require('url');
const fs = require('fs');
const path = require('path');
const { TextDecoder } = require('util');

const PORT = Number(process.env.PORT || 3001);
const STATIC_ROOT = path.join(__dirname, '..');
const CHROME_PATH = '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const TEST_TIMEOUT_MS = 12000;

const MIME = {
  '.html':'text/html;charset=utf-8', '.css':'text/css', '.js':'application/javascript',
  '.json':'application/json', '.png':'image/png', '.jpg':'image/jpeg',
  '.svg':'image/svg+xml', '.ico':'image/x-icon', '.gif':'image/gif'
};

let puppeteer, browser;
try {
  puppeteer = require('puppeteer-core');
} catch(e) {
  console.warn('puppeteer-core not installed, using simple fetch mode');
}

async function fetchWithBrowser(targetUrl) {
  if (!browser) {
    browser = await puppeteer.launch({
      executablePath: CHROME_PATH,
      headless: 'new',
      args: ['--no-sandbox', '--disable-setuid-sandbox']
    });
  }
  const page = await browser.newPage();
  await page.setViewport({ width: 1280, height: 900 });
  await page.setUserAgent('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36');
  try {
    await page.goto(targetUrl, { waitUntil: 'networkidle2', timeout: 15000 });
    // Wait a bit more for dynamic content
    await new Promise(r => setTimeout(r, 2000));
    const html = await page.content();
    await page.close();
    return html;
  } catch(e) {
    await page.close();
    throw e;
  }
}

function simpleFetch(targetUrl) {
  return new Promise((resolve, reject) => {
    const client = targetUrl.startsWith('https') ? https : http;
    client.get(targetUrl, {
      headers: { 'User-Agent': 'Mozilla/5.0', 'Accept-Encoding': 'identity' }
    }, resp => {
      if (resp.statusCode >= 300 && resp.statusCode < 400 && resp.headers.location) {
        let loc = resp.headers.location;
        if (!loc.startsWith('http')) { const u = new URL(targetUrl); loc = u.origin + loc; }
        return simpleFetch(loc).then(resolve).catch(reject);
      }
      const chunks = [];
      resp.on('data', c => chunks.push(c));
      resp.on('end', () => {
        const buf = Buffer.concat(chunks);
        const ct = resp.headers['content-type'] || '';
        let charset = 'utf-8';
        const m = ct.match(/charset=([^\s;]+)/i);
        if (m) charset = m[1].toLowerCase();
        const peek = buf.toString('ascii', 0, Math.min(buf.length, 2000));
        const metaM = peek.match(/charset=["']?([^"'\s;>]+)/i);
        if (metaM) charset = metaM[1].toLowerCase();
        try {
          const { TextDecoder } = require('util');
          resolve(new TextDecoder(charset.replace('gb2312','gbk')).decode(buf));
        } catch(e) { resolve(buf.toString('utf-8')); }
      });
    }).on('error', reject);
  });
}

function rewriteUrls(html, baseUrl) {
  try {
    const u = new URL(baseUrl);
    const origin = u.origin;
    const basePath = baseUrl.replace(/\/[^\/]*$/, '/');
    html = html.replace(/(href|src|action)=(["'])(?!http|\/\/|#|javascript:|data:|mailto:)([^"']*)\2/gi,
      (m, attr, q, p) => attr + '=' + q + (p.startsWith('/') ? origin + p : basePath + p) + q);
    html = html.replace(/url\((?!["']?(?:http|\/\/|data:))["']?([^"')]+)["']?\)/gi,
      (m, p) => 'url(' + (p.startsWith('/') ? origin + p : basePath + p) + ')');
  } catch(e) {}
  return html;
}

function sendJson(res, statusCode, payload) {
  res.writeHead(statusCode, { 'Content-Type': 'application/json; charset=utf-8' });
  res.end(JSON.stringify(payload));
}

function readJsonBody(req) {
  return new Promise((resolve, reject) => {
    let body = '';
    req.on('data', chunk => {
      body += chunk;
      if (body.length > 1024 * 1024) {
        reject(new Error('请求体过大'));
        req.destroy();
      }
    });
    req.on('end', () => {
      if (!body) { resolve({}); return; }
      try { resolve(JSON.parse(body)); }
      catch(e) { reject(new Error('请求体不是合法 JSON')); }
    });
    req.on('error', reject);
  });
}

function requireFields(config, fields) {
  const missing = fields.filter(field => !String(config[field] || '').trim());
  if (missing.length) throw new Error('缺少必要配置：' + missing.join('、'));
}

function normalizeEndpoint(baseUrl, endpoint) {
  const base = String(baseUrl || '').trim();
  if (!base) throw new Error('缺少 Base URL');
  if (!endpoint) return base;
  return new URL(endpoint, base.endsWith('/') ? base : base + '/').toString();
}

function httpRequest(targetUrl, options = {}) {
  return new Promise((resolve, reject) => {
    const parsed = new URL(targetUrl);
    const client = parsed.protocol === 'https:' ? https : http;
    const body = options.body || '';
    const req = client.request(parsed, {
      method: options.method || 'GET',
      headers: Object.assign({
        'User-Agent': 'MOI-Connector-Test/1.0',
        'Accept': 'application/json, text/plain, */*'
      }, options.headers || {}, body ? { 'Content-Length': Buffer.byteLength(body) } : {}),
      timeout: options.timeout || TEST_TIMEOUT_MS
    }, resp => {
      const chunks = [];
      resp.on('data', chunk => chunks.push(chunk));
      resp.on('end', () => {
        const text = Buffer.concat(chunks).toString('utf8');
        resolve({ statusCode: resp.statusCode || 0, headers: resp.headers, body: text });
      });
    });
    req.on('timeout', () => req.destroy(new Error('连接超时')));
    req.on('error', reject);
    if (body) req.write(body);
    req.end();
  });
}

async function postForm(targetUrl, form) {
  const body = new URLSearchParams(form).toString();
  const resp = await httpRequest(targetUrl, {
    method: 'POST',
    body,
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  });
  if (resp.statusCode < 200 || resp.statusCode >= 300) {
    throw new Error('令牌接口返回 HTTP ' + resp.statusCode + ': ' + resp.body.slice(0, 180));
  }
  try { return JSON.parse(resp.body); }
  catch(e) { throw new Error('令牌接口返回内容不是 JSON'); }
}

async function getGmailToken(config) {
  requireFields(config, ['clientId', 'clientSecret', 'refreshToken']);
  const token = await postForm('https://oauth2.googleapis.com/token', {
    client_id: config.clientId,
    client_secret: config.clientSecret,
    refresh_token: config.refreshToken,
    grant_type: 'refresh_token'
  });
  if (!token.access_token) throw new Error('Google OAuth 未返回 access_token');
  return token.access_token;
}

async function testGmail(config) {
  requireFields(config, ['mailboxAddress']);
  const token = await getGmailToken(config);
  const user = encodeURIComponent(config.mailboxAddress || 'me');
  const resp = await httpRequest('https://gmail.googleapis.com/gmail/v1/users/' + user + '/profile', {
    headers: { Authorization: 'Bearer ' + token }
  });
  if (resp.statusCode < 200 || resp.statusCode >= 300) {
    throw new Error('Gmail 返回 HTTP ' + resp.statusCode + ': ' + resp.body.slice(0, 180));
  }
  return 'Gmail API 验证通过，邮箱 ' + config.mailboxAddress + ' 可访问';
}

async function getGraphToken(config) {
  requireFields(config, ['tenantId', 'clientId', 'clientSecret']);
  const token = await postForm('https://login.microsoftonline.com/' + encodeURIComponent(config.tenantId) + '/oauth2/v2.0/token', {
    client_id: config.clientId,
    client_secret: config.clientSecret,
    grant_type: 'client_credentials',
    scope: 'https://graph.microsoft.com/.default'
  });
  if (!token.access_token) throw new Error('Microsoft OAuth 未返回 access_token');
  return token.access_token;
}

async function testOutlook(config) {
  requireFields(config, ['mailboxAddress']);
  const token = await getGraphToken(config);
  const mailbox = encodeURIComponent(config.mailboxAddress);
  const resp = await httpRequest('https://graph.microsoft.com/v1.0/users/' + mailbox + '/mailFolders?$top=1', {
    headers: { Authorization: 'Bearer ' + token }
  });
  if (resp.statusCode < 200 || resp.statusCode >= 300) {
    throw new Error('Microsoft Graph 返回 HTTP ' + resp.statusCode + ': ' + resp.body.slice(0, 180));
  }
  return 'Microsoft Graph 验证通过，邮箱 ' + config.mailboxAddress + ' 可访问';
}

function parseJsonHttpResponse(resp, serviceName) {
  if (resp.statusCode < 200 || resp.statusCode >= 300) {
    throw new Error(serviceName + ' 返回 HTTP ' + resp.statusCode + ': ' + resp.body.slice(0, 180));
  }
  try { return JSON.parse(resp.body || '{}'); }
  catch(e) { throw new Error(serviceName + ' 返回内容不是 JSON'); }
}

function formatMailDate(value) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value).slice(0, 32);
  const pad = n => String(n).padStart(2, '0');
  return date.getFullYear() + '-' + pad(date.getMonth() + 1) + '-' + pad(date.getDate())
    + ' ' + pad(date.getHours()) + ':' + pad(date.getMinutes());
}

function decodeMimeWords(value) {
  return String(value || '').replace(/=\?([^?]+)\?([BQbq])\?([^?]+)\?=/g, (match, charset, encoding, text) => {
    try {
      let buf;
      if (encoding.toUpperCase() === 'B') {
        buf = Buffer.from(text, 'base64');
      } else {
        const qp = text.replace(/_/g, ' ').replace(/=([0-9A-F]{2})/gi, (_, h) => String.fromCharCode(parseInt(h, 16)));
        buf = Buffer.from(qp, 'binary');
      }
      const normalized = charset.toLowerCase().replace('gb2312', 'gbk');
      return new TextDecoder(normalized).decode(buf);
    } catch(e) {
      return match;
    }
  }).trim();
}

function parseMailHeaders(headerText) {
  const unfolded = String(headerText || '').replace(/\r?\n[ \t]+/g, ' ');
  const headers = {};
  unfolded.split(/\r?\n/).forEach(line => {
    const idx = line.indexOf(':');
    if (idx <= 0) return;
    headers[line.slice(0, idx).toLowerCase()] = decodeMimeWords(line.slice(idx + 1).trim());
  });
  return headers;
}

function displaySender(value) {
  const decoded = decodeMimeWords(value || '');
  const mail = decoded.match(/<([^>]+)>/);
  return mail ? mail[1] : decoded || '-';
}

function displaySubject(value) {
  return decodeMimeWords(value || '').trim() || '（无主题）';
}

function rangeToDays(range) {
  switch (range) {
    case '7d': return 7;
    case '30d': return 30;
    case '90d': return 90;
    case '180d': return 180;
    default: return 0;
  }
}

function rangeToSinceDate(range) {
  const days = rangeToDays(range);
  if (!days) return null;
  const date = new Date();
  date.setDate(date.getDate() - days);
  return date;
}

function rangeToGmailQuery(range) {
  const days = rangeToDays(range);
  return days ? 'newer_than:' + days + 'd' : '';
}

function formatImapDate(date) {
  const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
  return date.getDate() + '-' + months[date.getMonth()] + '-' + date.getFullYear();
}

const gmailSystemLabelNames = {
  INBOX: '收件箱',
  SENT: '已发送',
  DRAFT: '草稿箱',
  SPAM: '垃圾邮件',
  TRASH: '已删除',
  IMPORTANT: '重要邮件',
  STARRED: '星标邮件',
  CATEGORY_PERSONAL: '个人',
  CATEGORY_SOCIAL: '社交',
  CATEGORY_PROMOTIONS: '推广',
  CATEGORY_UPDATES: '更新',
  CATEGORY_FORUMS: '论坛'
};

function gmailLabelDisplayName(label) {
  return gmailSystemLabelNames[label.id] || label.name || label.id;
}

async function listGmailMail(payload) {
  const config = payload.config || {};
  requireFields(config, ['mailboxAddress']);
  const token = await getGmailToken(config);
  const user = encodeURIComponent(config.mailboxAddress || 'me');
  const labelResp = await httpRequest('https://gmail.googleapis.com/gmail/v1/users/' + user + '/labels', {
    headers: { Authorization: 'Bearer ' + token }
  });
  const labelData = parseJsonHttpResponse(labelResp, 'Gmail');
  const labels = (labelData.labels || []).filter(label => {
    if (['CHAT', 'UNREAD'].includes(label.id)) return false;
    return label.type === 'user' || ['INBOX', 'SENT', 'DRAFT', 'IMPORTANT', 'STARRED'].includes(label.id);
  });
  const pathName = decodeURIComponent(String(payload.path || '/').replace(/^\//, ''));
  if (!pathName) {
    return labels.map(label => ({
      name: gmailLabelDisplayName(label).replace(/\//g, '／'),
      type: 'folder',
      count: label.messagesTotal || 0
    }));
  }

  const label = labels.find(item => gmailLabelDisplayName(item).replace(/\//g, '／') === pathName || item.name === pathName || item.id === pathName);
  if (!label) throw new Error('Gmail 中未找到邮件标签：' + pathName);
  const query = rangeToGmailQuery(payload.range);
  const params = new URLSearchParams({ maxResults: '20', labelIds: label.id });
  if (query) params.set('q', query);
  const listResp = await httpRequest('https://gmail.googleapis.com/gmail/v1/users/' + user + '/messages?' + params.toString(), {
    headers: { Authorization: 'Bearer ' + token }
  });
  const listData = parseJsonHttpResponse(listResp, 'Gmail');
  const messages = listData.messages || [];
  const items = await Promise.all(messages.map(async message => {
    const metaResp = await httpRequest('https://gmail.googleapis.com/gmail/v1/users/' + user + '/messages/' + encodeURIComponent(message.id) + '?format=metadata&metadataHeaders=Subject&metadataHeaders=From&metadataHeaders=Date', {
      headers: { Authorization: 'Bearer ' + token }
    });
    const meta = parseJsonHttpResponse(metaResp, 'Gmail');
    const headers = {};
    (((meta.payload || {}).headers) || []).forEach(h => { headers[h.name.toLowerCase()] = h.value; });
    return {
      name: displaySubject(headers.subject),
      type: 'mail',
      sender: displaySender(headers.from),
      date: formatMailDate(headers.date || Number(meta.internalDate || 0)),
      id: message.id
    };
  }));
  return items;
}

async function listOutlookMail(payload) {
  const config = payload.config || {};
  requireFields(config, ['mailboxAddress']);
  const token = await getGraphToken(config);
  const mailbox = encodeURIComponent(config.mailboxAddress);
  const folderResp = await httpRequest('https://graph.microsoft.com/v1.0/users/' + mailbox + '/mailFolders?$top=50', {
    headers: { Authorization: 'Bearer ' + token }
  });
  const folderData = parseJsonHttpResponse(folderResp, 'Microsoft Graph');
  const folders = folderData.value || [];
  const pathName = decodeURIComponent(String(payload.path || '/').replace(/^\//, ''));
  if (!pathName) {
    return folders.map(folder => ({
      name: folder.displayName || folder.id,
      type: 'folder',
      count: folder.totalItemCount || 0
    }));
  }

  const folder = folders.find(item => item.displayName === pathName || item.id === pathName);
  if (!folder) throw new Error('Outlook 中未找到邮件文件夹：' + pathName);
  const params = new URLSearchParams({
    '$top': '20',
    '$select': 'subject,from,receivedDateTime,hasAttachments',
    '$orderby': 'receivedDateTime desc'
  });
  const since = rangeToSinceDate(payload.range);
  if (since) params.set('$filter', 'receivedDateTime ge ' + since.toISOString());
  const msgResp = await httpRequest('https://graph.microsoft.com/v1.0/users/' + mailbox + '/mailFolders/' + encodeURIComponent(folder.id) + '/messages?' + params.toString(), {
    headers: { Authorization: 'Bearer ' + token }
  });
  const msgData = parseJsonHttpResponse(msgResp, 'Microsoft Graph');
  return (msgData.value || []).map(message => ({
    name: displaySubject(message.subject),
    type: 'mail',
    sender: (((message.from || {}).emailAddress || {}).address) || (((message.from || {}).emailAddress || {}).name) || '-',
    date: formatMailDate(message.receivedDateTime),
    id: message.id,
    hasAttachments: !!message.hasAttachments
  }));
}

function escapeImapString(value) {
  return '"' + String(value || '').replace(/\\/g, '\\\\').replace(/"/g, '\\"') + '"';
}

function waitForSocketLine(socket, matcher, timeoutMs) {
  return new Promise((resolve, reject) => {
    let buf = '';
    const timer = setTimeout(() => cleanup(new Error('等待服务器响应超时')), timeoutMs || TEST_TIMEOUT_MS);
    function cleanup(err, value) {
      clearTimeout(timer);
      socket.off('data', onData);
      socket.off('error', onError);
      if (err) reject(err);
      else resolve(value);
    }
    function onError(err) { cleanup(err); }
    function onData(chunk) {
      buf += chunk.toString('utf8');
      if (matcher(buf)) cleanup(null, buf);
    }
    socket.on('data', onData);
    socket.on('error', onError);
  });
}

function openTcpConnection(host, port, secure) {
  return new Promise((resolve, reject) => {
    const socket = secure
      ? tls.connect({ host, port, servername: host, rejectUnauthorized: false })
      : net.connect({ host, port });
    const timer = setTimeout(() => {
      socket.destroy();
      reject(new Error('连接超时'));
    }, TEST_TIMEOUT_MS);
    const readyEvent = secure ? 'secureConnect' : 'connect';
    socket.once(readyEvent, () => {
      clearTimeout(timer);
      resolve(socket);
    });
    socket.once('error', err => {
      clearTimeout(timer);
      reject(err);
    });
  });
}

async function upgradeStartTls(socket, host, command) {
  socket.write(command + '\r\n');
  const resp = await waitForSocketLine(socket, text => /(^|\r?\n)(A001 OK|220 )/i.test(text), TEST_TIMEOUT_MS);
  if (!/(A001 OK|220 )/i.test(resp)) throw new Error('服务器拒绝 STARTTLS: ' + resp.slice(0, 160));
  return tls.connect({ socket, servername: host, rejectUnauthorized: false });
}

async function testImap(config, defaults = {}) {
  const host = config.imapHost || defaults.imapHost;
  const port = Number(config.imapPort || defaults.imapPort || 993);
  const encryption = config.encryption || defaults.encryption || 'ssl';
  const user = config.user || config.mailboxUser || config.mailboxAddress;
  const password = config.password;
  if (!host) throw new Error('缺少 IMAP 主机');
  if (!user || !password) throw new Error('缺少 IMAP 用户名或授权码');
  let socket = await openTcpConnection(host, port, encryption === 'ssl');
  try {
    await waitForSocketLine(socket, text => /\* OK|\* PREAUTH/i.test(text), TEST_TIMEOUT_MS);
    if (encryption === 'starttls') {
      socket = await upgradeStartTls(socket, host, 'A001 STARTTLS');
      await new Promise(resolve => socket.once('secureConnect', resolve));
    }
    socket.write('A002 LOGIN ' + escapeImapString(user) + ' ' + escapeImapString(password) + '\r\n');
    const loginResp = await waitForSocketLine(socket, text => /(^|\r?\n)A002 (OK|NO|BAD)/i.test(text), TEST_TIMEOUT_MS);
    if (!/(^|\r?\n)A002 OK/i.test(loginResp)) throw new Error('IMAP 登录失败: ' + loginResp.slice(0, 180));
    socket.write('A003 LOGOUT\r\n');
    return 'IMAP 登录成功，邮箱账号可访问';
  } finally {
    socket.destroy();
  }
}

async function openLoggedInImap(config, defaults = {}) {
  const host = config.imapHost || defaults.imapHost;
  const port = Number(config.imapPort || defaults.imapPort || 993);
  const encryption = config.encryption || defaults.encryption || 'ssl';
  const user = config.user || config.mailboxUser || config.mailboxAddress;
  const password = config.password;
  if (!host) throw new Error('缺少 IMAP 主机');
  if (!user || !password) throw new Error('缺少 IMAP 用户名或授权码');
  let socket = await openTcpConnection(host, port, encryption === 'ssl');
  await waitForSocketLine(socket, text => /\* OK|\* PREAUTH/i.test(text), TEST_TIMEOUT_MS);
  if (encryption === 'starttls') {
    socket = await upgradeStartTls(socket, host, 'A001 STARTTLS');
    await new Promise(resolve => socket.once('secureConnect', resolve));
  }
  socket.write('A002 LOGIN ' + escapeImapString(user) + ' ' + escapeImapString(password) + '\r\n');
  const loginResp = await waitForSocketLine(socket, text => /(^|\r?\n)A002 (OK|NO|BAD)/i.test(text), TEST_TIMEOUT_MS);
  if (!/(^|\r?\n)A002 OK/i.test(loginResp)) {
    socket.destroy();
    throw new Error('IMAP 登录失败: ' + loginResp.slice(0, 180));
  }
  return socket;
}

function imapCommand(socket, tag, command) {
  socket.write(tag + ' ' + command + '\r\n');
  return waitForSocketLine(socket, text => new RegExp('(^|\\r?\\n)' + tag + ' (OK|NO|BAD)', 'i').test(text), TEST_TIMEOUT_MS);
}

function parseImapMailboxList(text) {
  const names = [];
  String(text || '').split(/\r?\n/).forEach(line => {
    if (!/^\* LIST/i.test(line)) return;
    const quoted = Array.from(line.matchAll(/"((?:[^"\\]|\\.)*)"/g)).map(match => match[1].replace(/\\"/g, '"').replace(/\\\\/g, '\\'));
    if (quoted.length) names.push(quoted[quoted.length - 1]);
  });
  return Array.from(new Set(names)).filter(Boolean);
}

function displayImapFolderName(name) {
  const base = String(name || '').split(/[\/.]/).pop() || name;
  const normalized = {
    INBOX: '收件箱',
    Sent: '已发送',
    'Sent Messages': '已发送',
    Drafts: '草稿箱',
    Trash: '已删除',
    Junk: '垃圾邮件',
    Spam: '垃圾邮件'
  };
  return normalized[base] || base;
}

async function listImapMail(payload, defaults = {}) {
  const socket = await openLoggedInImap(payload.config || {}, defaults);
  try {
    const listResp = await imapCommand(socket, 'A003', 'LIST "" "*"');
    const mailboxes = parseImapMailboxList(listResp);
    const pathName = decodeURIComponent(String(payload.path || '/').replace(/^\//, ''));
    if (!pathName) {
      return mailboxes.slice(0, 30).map(name => ({
        name: displayImapFolderName(name),
        rawName: name,
        type: 'folder'
      }));
    }

    const mailbox = mailboxes.find(name => displayImapFolderName(name) === pathName || name === pathName) || pathName;
    const selectResp = await imapCommand(socket, 'A004', 'SELECT ' + escapeImapString(mailbox));
    if (!/(^|\r?\n)A004 OK/i.test(selectResp)) throw new Error('无法打开 IMAP 邮箱文件夹：' + pathName);
    const since = rangeToSinceDate(payload.range);
    const searchResp = await imapCommand(socket, 'A005', since ? ('SEARCH SINCE ' + formatImapDate(since)) : 'SEARCH ALL');
    const searchLine = (searchResp.match(/\* SEARCH([^\r\n]*)/i) || [])[1] || '';
    const ids = searchLine.trim().split(/\s+/).filter(Boolean).slice(-20).reverse();
    const items = [];
    for (let i = 0; i < ids.length; i++) {
      const tag = 'A' + String(100 + i);
      const fetchResp = await imapCommand(socket, tag, 'FETCH ' + ids[i] + ' BODY.PEEK[HEADER.FIELDS (SUBJECT FROM DATE)]');
      const headers = parseMailHeaders(fetchResp);
      items.push({
        name: displaySubject(headers.subject),
        type: 'mail',
        sender: displaySender(headers.from),
        date: formatMailDate(headers.date),
        id: ids[i]
      });
    }
    return items;
  } finally {
    try { socket.write('A999 LOGOUT\r\n'); } catch(e) {}
    socket.destroy();
  }
}

async function testSmtp(config, defaults = {}) {
  const host = config.smtpHost || defaults.smtpHost;
  const port = Number(config.smtpPort || defaults.smtpPort || 465);
  if (!host) return null;
  const encryption = config.encryption || defaults.encryption || 'ssl';
  let socket = await openTcpConnection(host, port, encryption === 'ssl');
  try {
    await waitForSocketLine(socket, text => /^220|\r?\n220/i.test(text), TEST_TIMEOUT_MS);
    socket.write('EHLO moi.local\r\n');
    await waitForSocketLine(socket, text => /(^|\r?\n)250 /i.test(text), TEST_TIMEOUT_MS);
    if (encryption === 'starttls') {
      socket = await upgradeStartTls(socket, host, 'STARTTLS');
      await new Promise(resolve => socket.once('secureConnect', resolve));
      socket.write('EHLO moi.local\r\n');
      await waitForSocketLine(socket, text => /(^|\r?\n)250 /i.test(text), TEST_TIMEOUT_MS);
    }
    if (config.user && config.password) {
      socket.write('AUTH LOGIN\r\n');
      await waitForSocketLine(socket, text => /(^|\r?\n)334 /i.test(text), TEST_TIMEOUT_MS);
      socket.write(Buffer.from(config.user).toString('base64') + '\r\n');
      await waitForSocketLine(socket, text => /(^|\r?\n)334 /i.test(text), TEST_TIMEOUT_MS);
      socket.write(Buffer.from(config.password).toString('base64') + '\r\n');
      const authResp = await waitForSocketLine(socket, text => /(^|\r?\n)(235|535|454|530) /i.test(text), TEST_TIMEOUT_MS);
      if (!/(^|\r?\n)235 /i.test(authResp)) throw new Error('SMTP 认证失败: ' + authResp.slice(0, 180));
    }
    socket.write('QUIT\r\n');
    return 'SMTP 握手' + (config.user && config.password ? '和认证' : '') + '成功';
  } finally {
    socket.destroy();
  }
}

async function testImapSmtp(config, defaults, usage = {}) {
  const messages = [await testImap(config, defaults)];
  const shouldTestSmtp = usage.export || config.smtpHost || config.smtpPort;
  if (shouldTestSmtp) {
    const smtpMessage = await testSmtp(config, defaults);
    if (smtpMessage) messages.push(smtpMessage);
  }
  return messages.join('；');
}

async function testWeComMail(config, usage) {
  return testImapSmtp(config, { imapHost: 'imap.exmail.qq.com', imapPort: 993, smtpHost: 'smtp.exmail.qq.com', smtpPort: 465, encryption: 'ssl' }, usage);
}

async function testCustomMailApi(config) {
  const headers = {};
  if (config.authMode === 'apikey') {
    requireFields(config, ['apiKeyName', 'apiKeyValue']);
    if (config.apiKeyLocation === 'query') {
      const target = new URL(normalizeEndpoint(config.baseUrl, config.messagesEndpoint || '/'));
      target.searchParams.set(config.apiKeyName, config.apiKeyValue);
      return testHttpEndpoint(target.toString(), headers);
    }
    headers[config.apiKeyName] = config.apiKeyValue;
  } else if (config.authMode === 'bearer') {
    requireFields(config, ['bearerToken']);
    headers.Authorization = 'Bearer ' + config.bearerToken;
  } else if (config.authMode === 'oauth2') {
    requireFields(config, ['clientId', 'clientSecret', 'tokenUrl']);
    const token = await postForm(config.tokenUrl, {
      client_id: config.clientId,
      client_secret: config.clientSecret,
      grant_type: 'client_credentials'
    });
    if (!token.access_token) throw new Error('OAuth 令牌接口未返回 access_token');
    headers.Authorization = 'Bearer ' + token.access_token;
  }
  return testHttpEndpoint(normalizeEndpoint(config.baseUrl, config.messagesEndpoint || '/'), headers);
}

function pickDataPath(data, pathExpr) {
  if (!pathExpr) {
    if (Array.isArray(data)) return data;
    if (Array.isArray(data.items)) return data.items;
    if (Array.isArray(data.messages)) return data.messages;
    if (Array.isArray(data.results)) return data.results;
    if (data.data) return pickDataPath(data.data, '');
    return [];
  }
  const value = String(pathExpr).split('.').filter(Boolean).reduce((acc, key) => {
    if (acc == null) return undefined;
    return acc[key];
  }, data);
  return Array.isArray(value) ? value : [];
}

async function buildCustomMailApiRequest(config, endpoint) {
  const headers = {};
  let targetUrl = normalizeEndpoint(config.baseUrl, endpoint || config.messagesEndpoint || '/');
  if (config.authMode === 'apikey') {
    requireFields(config, ['apiKeyName', 'apiKeyValue']);
    if (config.apiKeyLocation === 'query') {
      const target = new URL(targetUrl);
      target.searchParams.set(config.apiKeyName, config.apiKeyValue);
      targetUrl = target.toString();
    } else {
      headers[config.apiKeyName] = config.apiKeyValue;
    }
  } else if (config.authMode === 'bearer') {
    requireFields(config, ['bearerToken']);
    headers.Authorization = 'Bearer ' + config.bearerToken;
  } else if (config.authMode === 'oauth2') {
    requireFields(config, ['clientId', 'clientSecret', 'tokenUrl']);
    const token = await postForm(config.tokenUrl, {
      client_id: config.clientId,
      client_secret: config.clientSecret,
      grant_type: 'client_credentials'
    });
    if (!token.access_token) throw new Error('OAuth 令牌接口未返回 access_token');
    headers.Authorization = 'Bearer ' + token.access_token;
  }
  return { targetUrl, headers };
}

async function listCustomMailApi(payload) {
  const config = payload.config || {};
  requireFields(config, ['baseUrl']);
  const request = await buildCustomMailApiRequest(config, config.messagesEndpoint || '/');
  const resp = await httpRequest(request.targetUrl, { headers: request.headers });
  const data = parseJsonHttpResponse(resp, '自定义邮件 API');
  const rows = pickDataPath(data, config.dataPath).slice(0, 50);
  return rows.map((item, index) => {
    const sender = item.sender || item.from || item.fromAddress || item.mailFrom || item.author || '-';
    return {
      name: displaySubject(item.subject || item.title || item.name || item.snippet || item.summary || ('邮件 #' + (index + 1))),
      type: 'mail',
      sender: typeof sender === 'string' ? displaySender(sender) : (sender.address || sender.email || sender.name || '-'),
      date: formatMailDate(item.date || item.receivedAt || item.receivedDateTime || item.createdAt || item.timestamp),
      id: item.id || item.messageId || item.uid || String(index + 1),
      hasAttachments: !!(item.hasAttachments || item.attachments)
    };
  });
}

async function testHttpEndpoint(targetUrl, headers) {
  const resp = await httpRequest(targetUrl, { headers });
  if (resp.statusCode < 200 || resp.statusCode >= 400) {
    throw new Error('接口返回 HTTP ' + resp.statusCode + ': ' + resp.body.slice(0, 180));
  }
  return 'HTTP 接口验证通过，返回 HTTP ' + resp.statusCode;
}

async function testConnector(payload) {
  const config = payload.config || {};
  const usage = payload.usage || {};
  switch (payload.source) {
    case 'gmail': return testGmail(config);
    case 'outlook': return testOutlook(config);
    case 'wecom-mail': return testWeComMail(config, usage);
    case 'qq-mail': return testImapSmtp(config, { imapHost: 'imap.qq.com', imapPort: 993, smtpHost: 'smtp.qq.com', smtpPort: 465, encryption: 'ssl' }, usage);
    case 'imap-smtp': return testImapSmtp(config, {}, usage);
    case 'custom-mail-api': return testCustomMailApi(config);
    case 'rest-api': return testCustomMailApi(Object.assign({ messagesEndpoint: '/' }, config));
    default:
      throw Object.assign(new Error('真实连接测试暂未支持该连接器类型：' + (payload.source || '未知')), { statusCode: 501 });
  }
}

async function listMailConnector(payload) {
  switch (payload.source) {
    case 'gmail': return listGmailMail(payload);
    case 'outlook': return listOutlookMail(payload);
    case 'wecom-mail': return listImapMail(payload, { imapHost: 'imap.exmail.qq.com', imapPort: 993, encryption: 'ssl' });
    case 'qq-mail': return listImapMail(payload, { imapHost: 'imap.qq.com', imapPort: 993, encryption: 'ssl' });
    case 'imap-smtp': return listImapMail(payload, {});
    case 'custom-mail-api': return listCustomMailApi(payload);
    default:
      throw Object.assign(new Error('真实邮箱读取暂未支持该连接器类型：' + (payload.source || '未知')), { statusCode: 501 });
  }
}

http.createServer(async (req, res) => {
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,OPTIONS');
  if (req.method === 'OPTIONS') { res.writeHead(200); res.end(); return; }

  const parsed = url.parse(req.url, true);

  if (parsed.pathname === '/api/connector/test' && req.method === 'POST') {
    try {
      const payload = await readJsonBody(req);
      const message = await testConnector(payload);
      sendJson(res, 200, { ok: true, message });
    } catch(e) {
      const statusCode = e.statusCode || 400;
      sendJson(res, statusCode, { ok: false, message: e.message || '连接测试失败' });
    }
    return;
  }

  if (parsed.pathname === '/api/connector/mail/list' && req.method === 'POST') {
    try {
      const payload = await readJsonBody(req);
      const items = await listMailConnector(payload);
      sendJson(res, 200, { ok: true, path: payload.path || '/', items });
    } catch(e) {
      const statusCode = e.statusCode || 400;
      sendJson(res, statusCode, { ok: false, message: e.message || '读取邮箱数据失败' });
    }
    return;
  }

  // Proxy endpoint
  if (parsed.pathname === '/proxy' && parsed.query.url) {
    const targetUrl = parsed.query.url;
    const useRender = parsed.query.render !== '0'; // default: use browser rendering
    console.log('Proxy:', targetUrl, useRender ? '(rendered)' : '(simple)');
    try {
      let html;
      if (useRender && puppeteer) {
        html = await fetchWithBrowser(targetUrl);
      } else {
        html = await simpleFetch(targetUrl);
      }
      html = html.replace(/<script[\s\S]*?<\/script>/gi, '');
      html = rewriteUrls(html, targetUrl);
      // Inject <base> tag so relative URLs resolve correctly
      const baseTag = '<base href="' + targetUrl.replace(/\/[^\/]*$/, '/') + '">';
      if (/<head[^>]*>/i.test(html)) {
        html = html.replace(/<head[^>]*>/i, '$&' + baseTag);
      } else {
        html = baseTag + html;
      }
      // Neutralize position:fixed to prevent overlays escaping the preview
      html = html.replace(/position\s*:\s*fixed/gi, 'position:absolute');
      html = html.replace(/z-index\s*:\s*(\d{4,})/gi, 'z-index:1');
      res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
      res.end(html);
    } catch(e) {
      console.error('Proxy error:', e.message);
      res.writeHead(500);
      res.end(e.message);
    }
    return;
  }

  // Static files
  const filePath = path.join(STATIC_ROOT, parsed.pathname === '/' ? 'index.html' : parsed.pathname);
  fs.readFile(filePath, (err, data) => {
    if (err) { res.writeHead(404); res.end('Not found'); return; }
    const ext = path.extname(filePath).toLowerCase();
    res.writeHead(200, { 'Content-Type': MIME[ext] || 'application/octet-stream' });
    res.end(data);
  });
}).listen(PORT, () => {
  console.log('Server: http://localhost:' + PORT);
  console.log('Open:   http://localhost:' + PORT + '/data-connection/data-import-create.html');
  console.log('Mode:   ' + (puppeteer ? 'Browser rendering (Puppeteer)' : 'Simple fetch'));
});

process.on('SIGINT', async () => {
  if (browser) await browser.close();
  process.exit();
});
