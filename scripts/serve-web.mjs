#!/usr/bin/env node
// 本地静态开发服务器：无依赖，只服务 web/，SPA 回退到 index.html。
//
//   node scripts/serve-web.mjs [--port 7801] [--host 127.0.0.1]
//
// 环境变量 PORT / HOST 同样可用。生产环境由 Go 服务的 embed 提供静态资源。

import { createServer } from 'node:http';
import { createReadStream } from 'node:fs';
import { stat } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const webRoot = path.resolve(here, '..', 'web');

const MIME = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.mjs': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.gif': 'image/gif',
  '.webp': 'image/webp',
  '.ico': 'image/x-icon',
  '.woff2': 'font/woff2',
  '.txt': 'text/plain; charset=utf-8',
  '.map': 'application/json; charset=utf-8',
};

function argValue(flag, fallback) {
  const index = process.argv.indexOf(flag);
  if (index >= 0 && process.argv[index + 1]) return process.argv[index + 1];
  return fallback;
}

const port = Number(argValue('--port', process.env.PORT || 7801));
const host = argValue('--host', process.env.HOST || '127.0.0.1');

function contentType(file) {
  return MIME[path.extname(file).toLowerCase()] || 'application/octet-stream';
}

function resolvePath(urlPath) {
  const decoded = decodeURIComponent(urlPath.split('?')[0]);
  const relative = decoded.replace(/^\/+/, '');
  const absolute = path.resolve(webRoot, relative);
  const inside = absolute === webRoot || absolute.startsWith(webRoot + path.sep);
  return inside ? absolute : null;
}

async function sendFile(response, file, method) {
  const info = await stat(file);
  if (info.isDirectory()) {
    return sendFile(response, path.join(file, 'index.html'), method);
  }
  response.writeHead(200, {
    'Content-Type': contentType(file),
    'Content-Length': info.size,
    'Cache-Control': 'no-store',
    'X-Content-Type-Options': 'nosniff',
  });
  if (method === 'HEAD') {
    response.end();
    return;
  }
  createReadStream(file).pipe(response);
}

const server = createServer(async (request, response) => {
  const method = request.method || 'GET';
  if (method !== 'GET' && method !== 'HEAD') {
    response.writeHead(405, { Allow: 'GET, HEAD', 'Content-Type': 'text/plain; charset=utf-8' });
    response.end('Method Not Allowed');
    return;
  }

  const started = Date.now();
  let urlPath = '/';
  try {
    urlPath = new URL(request.url || '/', 'http://' + (request.headers.host || host)).pathname;
  } catch (error) {
    urlPath = '/';
  }

  let status = 200;
  try {
    const file = resolvePath(urlPath);
    if (!file) {
      status = 403;
      response.writeHead(403, { 'Content-Type': 'text/plain; charset=utf-8' });
      response.end('Forbidden');
    } else if (urlPath === '/' || urlPath === '/index.html') {
      await sendFile(response, path.join(webRoot, 'index.html'), method);
    } else {
      try {
        const info = await stat(file);
        if (info.isFile()) {
          await sendFile(response, file, method);
        } else {
          await sendFile(response, path.join(webRoot, 'index.html'), method);
        }
      } catch (error) {
        if (error.code === 'ENOENT') {
          // 带扩展名但文件不存在 → 404；否则视为前端路由，回退 index.html。
          if (path.extname(urlPath)) {
            status = 404;
            response.writeHead(404, { 'Content-Type': 'text/plain; charset=utf-8' });
            response.end('Not Found: ' + urlPath);
          } else {
            await sendFile(response, path.join(webRoot, 'index.html'), method);
          }
        } else {
          throw error;
        }
      }
    }
  } catch (error) {
    status = 500;
    response.writeHead(500, { 'Content-Type': 'text/plain; charset=utf-8' });
    response.end('Server Error: ' + error.message);
  } finally {
    const elapsed = Date.now() - started;
    process.stdout.write(method + ' ' + status + ' ' + urlPath + ' ' + elapsed + 'ms\n');
  }
});

server.listen(port, host, function () {
  process.stdout.write('v2ex-vimreader web dev server\n');
  process.stdout.write('  root: ' + webRoot + '\n');
  process.stdout.write('  url:  http://' + host + ':' + port + '/  (mock: /?mock=1)\n');
});

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, function () {
    server.close(function () {
      process.exit(0);
    });
  });
}
