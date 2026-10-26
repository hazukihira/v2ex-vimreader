// 后端 HTTP API 封装与 SSE 解析；?mock=1 时整体切换到 mock 数据。

const search = new URLSearchParams(location.search);
export const isMock = search.has('mock');

let mockPromise = null;
function getMock() {
  if (!mockPromise) mockPromise = import('./mock.js').then((mod) => mod.createMockApi());
  return mockPromise;
}

export class ApiError extends Error {
  constructor(message, { status = 0, code = '' } = {}) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

function buildQuery(params) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params || {})) {
    if (value == null || value === '') continue;
    query.set(key, String(value));
  }
  const text = query.toString();
  return text ? '?' + text : '';
}

async function request(path, options = {}) {
  const { method = 'GET', body, headers, signal, timeoutMs = 45000 } = options;
  const controller = new AbortController();
  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);
  const onAbort = () => controller.abort();
  if (signal) {
    if (signal.aborted) controller.abort();
    else signal.addEventListener('abort', onAbort, { once: true });
  }
  try {
    const init = {
      method,
      signal: controller.signal,
      cache: 'no-store',
      headers: { Accept: 'application/json', ...(headers || {}) },
    };
    if (body !== undefined) {
      init.headers['Content-Type'] = 'application/json';
      init.body = JSON.stringify(body);
    }
    const response = await fetch(path, init);
    if (response.status === 204) return null;
    const contentType = response.headers.get('content-type') || '';
    const payload = contentType.includes('json')
      ? await response.json().catch(() => null)
      : await response.text();
    if (!response.ok) {
      const detail = payload && payload.error ? payload.error : null;
      const message = (detail && detail.message) || '请求失败（HTTP ' + response.status + '）';
      throw new ApiError(message, { status: response.status, code: (detail && detail.code) || '' });
    }
    return payload;
  } catch (error) {
    if (error instanceof ApiError) throw error;
    if (error && error.name === 'AbortError' && !timedOut) throw new ApiError('请求已取消', { code: 'aborted' });
    if (error && error.name === 'AbortError') throw new ApiError('请求超时，按 r 重试', { code: 'timeout' });
    throw new ApiError('网络请求失败，按 r 重试', { code: 'network' });
  } finally {
    clearTimeout(timer);
    if (signal) signal.removeEventListener('abort', onAbort);
  }
}

function parseEventBlock(block) {
  let eventName = 'message';
  const dataLines = [];
  for (const rawLine of block.split('\n')) {
    const line = rawLine.replace(/\r$/, '');
    if (!line || line.startsWith(':')) continue;
    if (line.startsWith('event:')) eventName = line.slice(6).trim();
    else if (line.startsWith('data:')) dataLines.push(line.slice(5).replace(/^ /, ''));
  }
  return dataLines.length ? { eventName, dataText: dataLines.join('\n') } : null;
}

function readEventData(eventName, dataText) {
  if (dataText === '[DONE]') return { kind: 'done', data: {} };
  try {
    const parsed = JSON.parse(dataText);
    if (typeof parsed === 'string') return { kind: eventName, delta: parsed };
    if (parsed && typeof parsed === 'object') {
      if (typeof parsed.error === 'string' && parsed.error) {
        return { kind: 'error', message: parsed.error };
      }
      if (eventName === 'error') return { kind: 'error', message: parsed.message || parsed.error || dataText };
      if (eventName === 'done' || parsed.done === true) return { kind: 'done', data: parsed };
      return { kind: 'delta', delta: parsed.delta ?? parsed.text ?? parsed.content ?? '', model: parsed.model };
    }
  } catch (e) {
    /* 非 JSON 的 data 行按原文处理 */
  }
  if (eventName === 'error') return { kind: 'error', message: dataText };
  if (eventName === 'done') return { kind: 'done', data: {} };
  return { kind: 'delta', delta: dataText };
}

async function readSummaryStream(response, onDelta) {
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let content = '';
  let model = '';
  let finished = false;

  const consume = (block) => {
    const event = parseEventBlock(block);
    if (!event) return;
    const parsed = readEventData(event.eventName, event.dataText);
    if (parsed.kind === 'error') throw new ApiError(parsed.message || 'AI 总结失败', { code: 'summary_error' });
    if (parsed.kind === 'done') {
      if (parsed.data) {
        if (typeof parsed.data.content === 'string' && parsed.data.content) content = parsed.data.content;
        if (parsed.data.model) model = parsed.data.model;
      }
      finished = true;
      return;
    }
    if (parsed.model) model = parsed.model;
    if (parsed.delta) {
      content += parsed.delta;
      onDelta?.(parsed.delta, content);
    }
  };

  while (true) {
    const chunk = await reader.read();
    if (chunk.done) break;
    buffer += decoder.decode(chunk.value, { stream: true });
    let index;
    while ((index = buffer.indexOf('\n\n')) >= 0) {
      consume(buffer.slice(0, index));
      buffer = buffer.slice(index + 2);
    }
  }
  if (buffer.trim()) consume(buffer);
  return { status: finished ? 'ready' : content ? 'ready' : 'running', content, model };
}

async function streamSummary(id, { onDelta, signal, refresh } = {}) {
  const init = {
    method: 'POST',
    headers: { Accept: 'text/event-stream' },
    signal,
    cache: 'no-store',
  };
  if (refresh) {
    init.headers['Content-Type'] = 'application/json';
    init.body = JSON.stringify({ refresh: true });
  }
  const response = await fetch('/api/topic/' + encodeURIComponent(id) + '/summary', init);
  if (response.status === 501) {
    throw new ApiError('AI 总结未启用（后端 config.toml 的 [ai] 段）', { status: 501, code: 'ai_disabled' });
  }
  if (!response.ok) {
    let message = 'AI 总结失败（HTTP ' + response.status + '）';
    try {
      const payload = await response.json();
      if (payload && payload.error && payload.error.message) message = payload.error.message;
    } catch (e) {}
    throw new ApiError(message, { status: response.status, code: 'summary_error' });
  }
  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    const payload = await response.json();
    const content = (payload && payload.content) || '';
    if (content) onDelta?.(content, content);
    return { status: 'ready', content, model: (payload && payload.model) || '' };
  }
  if (!response.body) {
    const text = await response.text();
    return { status: 'ready', content: text, model: '' };
  }
  return readSummaryStream(response, onDelta);
}

export const api = {
  isMock,

  bootstrap() {
    if (isMock) return getMock().then((mock) => mock.bootstrap());
    return request('/api/bootstrap');
  },

  list(params) {
    if (isMock) return getMock().then((mock) => mock.list(params));
    return request('/api/list' + buildQuery(params));
  },

  topic(id, page = 1, refresh = false) {
    if (isMock) return getMock().then((mock) => mock.topic(id, page, refresh));
    return request('/api/topic/' + encodeURIComponent(id) + buildQuery({ page, refresh: refresh ? 1 : '' }));
  },

  replies(id, page = 1) {
    if (isMock) return getMock().then((mock) => mock.replies(id, page));
    return request('/api/topic/' + encodeURIComponent(id) + '/replies' + buildQuery({ page }));
  },

  refreshTopic(id, page = 1) {
    if (isMock) return getMock().then((mock) => mock.topic(id, page, true));
    return request('/api/topic/' + encodeURIComponent(id) + '/refresh', { method: 'POST', body: { page } });
  },

  summaryGet(id) {
    if (isMock) return getMock().then((mock) => mock.summaryGet(id));
    return request('/api/topic/' + encodeURIComponent(id) + '/summary');
  },

  summaryStream(id, options) {
    if (isMock) return getMock().then((mock) => mock.summaryStream(id, options));
    return streamSummary(id, options);
  },

  blocks() {
    if (isMock) return getMock().then((mock) => mock.blocks());
    return request('/api/blocks');
  },

  addBlock(kind, value) {
    if (isMock) return getMock().then((mock) => mock.addBlock(kind, value));
    return request('/api/blocks', { method: 'POST', body: { kind, value } });
  },

  removeBlock(kind, value) {
    if (isMock) return getMock().then((mock) => mock.removeBlock(kind, value));
    return request('/api/blocks' + buildQuery({ kind, value }), { method: 'DELETE' });
  },

  nodes() {
    if (isMock) return getMock().then((mock) => mock.nodes());
    return request('/api/nodes', { timeoutMs: 60000 });
  },

  followedNodes() {
    if (isMock) return getMock().then((mock) => mock.followedNodes());
    return request('/api/followed/nodes', { timeoutMs: 20000 }).catch((error) => {
      if (error && error.status === 404) {
        throw new ApiError('后端尚未支持节点管理接口（GET /api/followed/nodes 404）', { status: 404, code: 'followed_unsupported' });
      }
      throw error;
    });
  },

  saveFollowedNodes(nodes) {
    if (isMock) return getMock().then((mock) => mock.saveFollowedNodes(nodes));
    return request('/api/followed/nodes', { method: 'PUT', body: { nodes: nodes }, timeoutMs: 30000 }).catch((error) => {
      if (error && error.status === 404) {
        throw new ApiError('后端尚未支持节点管理接口（PUT /api/followed/nodes 404）', { status: 404, code: 'followed_unsupported' });
      }
      throw error;
    });
  },

  status() {
    if (isMock) return getMock().then((mock) => mock.status());
    return request('/api/status', { timeoutMs: 15000 });
  },

  login(cookie) {
    if (isMock) return getMock().then((mock) => mock.login(cookie));
    return request('/api/login', { method: 'POST', body: { cookie }, timeoutMs: 30000 }).catch((error) => {
      if (error && (error.status === 404 || error.status === 501 || error.status === 405)) {
        throw new ApiError('后端尚未支持登录接口，等新版后端就绪后再试', { status: error.status, code: 'login_unsupported' });
      }
      throw error;
    });
  },

  loginChrome() {
    if (isMock) return getMock().then((mock) => mock.loginChrome());
    return request('/api/login/chrome', { method: 'POST', body: {}, timeoutMs: 150000 }).catch((error) => {
      if (error && (error.status === 404 || error.status === 405 || error.status === 501)) {
        throw new ApiError('后端尚未支持 Chrome 导入，稍后重启新版', { status: error.status, code: 'chrome_unsupported' });
      }
      throw error;
    });
  },

  logout() {
    if (isMock) return getMock().then((mock) => mock.logout());
    return request('/api/login', { method: 'DELETE', timeoutMs: 15000 }).catch((error) => {
      if (error && (error.status === 404 || error.status === 501 || error.status === 405)) {
        throw new ApiError('后端尚未支持登录接口', { status: error.status, code: 'login_unsupported' });
      }
      throw error;
    });
  },

  sync(target = 'all') {
    if (isMock) return getMock().then((mock) => mock.sync(target));
    return request('/api/sync', { method: 'POST', body: { target } });
  },

  search(term) {
    if (isMock) return getMock().then((mock) => mock.search(term));
    return request('/api/search' + buildQuery({ q: term }));
  },
};
