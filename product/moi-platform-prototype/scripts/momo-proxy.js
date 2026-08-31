#!/usr/bin/env node
/*
 * momo-proxy.js —— MOMO 本地 CORS 转发代理（零依赖，仅用 Node 内置模块）
 *
 * 作用：有些大模型网关（如公司的 TaaS）不允许浏览器跨域(CORS)，浏览器里的 MOMO 直接调用会 "Failed to fetch"。
 *      本代理在本机做一次"服务端转发"：浏览器 → 本代理（同机、本代理回 CORS 头）→ 上游网关（服务端到服务端，无 CORS）。
 *
 * 启动：  node html/scripts/momo-proxy.js          （默认端口 8788，仅监听 127.0.0.1）
 *        端口可改：  MOMO_PROXY_PORT=9000 node html/scripts/momo-proxy.js
 *        可选白名单（只允许转发到这些 host，更安全）：
 *                  MOMO_PROXY_ALLOW=api-taas.moi.matrixorigin.cn node html/scripts/momo-proxy.js
 *
 * 在 MOMO「模型设置 · API 地址」里填（把真实上游 URL 直接拼在代理地址后面）：
 *        http://localhost:8788/https://api-taas.moi.matrixorigin.cn/v1
 *
 * 关于密钥：本代理不读、不存任何 Key。Key 由浏览器在每次请求时通过 Authorization 头带上，代理只原样转发。
 *          因此本文件可以安全提交 Git，里面没有任何机密。
 */
'use strict';
var http = require('http');
var https = require('https');
var URL = require('url').URL;

var PORT = parseInt(process.env.MOMO_PROXY_PORT || '8788', 10);
var ALLOW = (process.env.MOMO_PROXY_ALLOW || '').split(',').map(function (s) { return s.trim(); }).filter(Boolean);

function corsHeaders(req) {
  return {
    'Access-Control-Allow-Origin': req.headers.origin || '*',
    'Access-Control-Allow-Methods': 'GET,POST,PUT,DELETE,OPTIONS',
    'Access-Control-Allow-Headers': req.headers['access-control-request-headers'] || 'Authorization,Content-Type,X-API-Key',
    'Access-Control-Max-Age': '86400'
  };
}

function handler(req, res) {
  var cors = corsHeaders(req);

  // 预检请求
  if (req.method === 'OPTIONS') { res.writeHead(204, cors); res.end(); return; }

  // 目标 = 路径里"/"之后的整段（即完整上游 URL）
  var target = req.url.slice(1);
  try { if (/^https?%3a/i.test(target)) target = decodeURIComponent(target); } catch (e) {}

  if (!/^https?:\/\//i.test(target)) {
    res.writeHead(400, Object.assign({ 'Content-Type': 'application/json; charset=utf-8' }, cors));
    res.end(JSON.stringify({ error: '地址格式应为 http://localhost:' + PORT + '/<完整上游URL>，例如 http://localhost:' + PORT + '/https://api-taas.moi.matrixorigin.cn/v1/chat/completions' }));
    return;
  }

  var upstream;
  try { upstream = new URL(target); } catch (e) {
    res.writeHead(400, cors); res.end('bad target url'); return;
  }
  if (ALLOW.length && ALLOW.indexOf(upstream.host) === -1) {
    res.writeHead(403, cors); res.end('host not allowed (set MOMO_PROXY_ALLOW)'); return;
  }

  var lib = upstream.protocol === 'https:' ? https : http;
  var headers = Object.assign({}, req.headers);
  delete headers.host; delete headers.origin; delete headers.referer; delete headers['accept-encoding'];
  headers.host = upstream.host;

  var upReq = lib.request(upstream, { method: req.method, headers: headers }, function (upRes) {
    // 上游若自带 CORS 头，会和本代理这份重复（浏览器报 "multiple values"）。先剔除上游所有 access-control-* 头，只保留本代理的单值。
    var outHeaders = {};
    Object.keys(upRes.headers).forEach(function (k) {
      if (k.toLowerCase().indexOf('access-control-') === 0) return;
      outHeaders[k] = upRes.headers[k];
    });
    Object.keys(cors).forEach(function (k) { outHeaders[k] = cors[k]; });
    res.writeHead(upRes.statusCode || 502, outHeaders);
    upRes.pipe(res); // 流式（SSE）原样透传
  });
  upReq.on('error', function (e) {
    res.writeHead(502, Object.assign({ 'Content-Type': 'application/json; charset=utf-8' }, cors));
    res.end(JSON.stringify({ error: '上游请求失败: ' + e.message }));
  });
  req.pipe(upReq); // 转发请求体
}

// 同时监听 IPv4(127.0.0.1) 和 IPv6(::1) 回环，避免 localhost 解析到哪个都打不通
function boot(host, label) {
  var s = http.createServer(handler);
  s.on('error', function (e) { console.log('   ⚠️ ' + label + ' 监听失败: ' + e.message); });
  s.listen(PORT, host, function () { console.log('   ✅ ' + label + ' 就绪'); });
}

console.log('MOMO 代理启动中（端口 ' + PORT + '）…');
boot('127.0.0.1', 'IPv4 127.0.0.1:' + PORT);
boot('::1', 'IPv6 [::1]:' + PORT);
console.log('在 MOMO「API 地址」里填：http://localhost:' + PORT + '/<完整上游URL>');
console.log('例如：http://localhost:' + PORT + '/https://api-taas.moi.matrixorigin.cn/v1');
if (ALLOW.length) console.log('仅允许转发到：' + ALLOW.join(', '));
