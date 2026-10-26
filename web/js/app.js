// 入口：路由、状态机、键盘流程与各视图装配。

import { api, isMock, ApiError } from './api.js?v=20261026';
import { store } from './store.js?v=20261026';
import { KeyEngine } from './vim.js?v=20261026';
import * as R from './render.js?v=20261026';
import { initLightbox, isLightboxOpen, openLightbox } from './lightbox.js?v=20261026';
import {
  isOverlayOpen, openHelp, openCommandPalette, openMenu, openBlocksManager, openNodePicker, openNodeManager, openLogin, toast,
} from './overlays.js?v=20261026';

const $ = function (selector) { return document.querySelector(selector); };
const ROW_H = 27;

const state = {
  boot: null,
  nodes: [],
  allNodes: null,
  nodeMap: new Map(),
  followed: [],
  config: {},
  archive: { from: 0, to: 0 },
  blocks: { topics: new Set(), users: new Set() },
  login: { hasCookie: false, loggedIn: false, user: '', checkedAt: 0 },
  route: null,
  viewRoute: null,
  list: null,
  topic: null,
  entries: [],
  replies: [],
  rowEls: [],
  replyEls: [],
  selectable: [],
  selection: -1,
  topicSelection: -1,
  filter: '',
  summary: null,
  summaryOpen: false,
  summaryTopicId: 0,
  summarySource: 'none',
  streaming: false,
  summarySeq: 0,
  status: null,
  loading: 0,
  read: new Set(),
  error: null,
  reqSeq: 0,
  lastListRoute: null,
  history: { items: [], index: -1 },
  clockTimer: 0,
  statusTimer: 0,
  nodesFetching: false,
  nodeManagerBusy: false,
};

const engine = new KeyEngine();

const HELP_GROUPS = [
  {
    title: '通用',
    rows: [
      ['?', '帮助'],
      [':', '命令面板'],
      ['Esc', '关闭浮层 / 取消'],
      ['\\', '切换侧栏 / 抽屉'],
      ['H Ctrl-L', '历史后退 / 前进'],
      ['g l', '登录 V2EX（粘贴 Cookie）'],
      ['gg G', '跳到首 / 尾'],
    ],
  },
  {
    title: '列表',
    rows: [
      ['j k', '下 / 上移动'],
      ['Ctrl-d Ctrl-u', '半页下 / 上'],
      ['Ctrl-f Ctrl-b', '整页下 / 上'],
      ['Enter l o', '打开主题'],
      ['h q', '返回'],
      ['[ ]', '上一页 / 下一页'],
      ['1-9 + 动作', '数字前缀（5j / 3]）'],
      ['/', '过滤当前列表'],
      ['n N', '下一个 / 上一个匹配'],
      ['r R', '刷新 / 强制刷新'],
      ['y', '复制原帖链接'],
      ['b', '屏蔽菜单'],
      ['x', '屏蔽当前主题'],
      ['u', '屏蔽当前作者'],
      ['m', '标记已读 / 未读'],
    ],
  },
  {
    title: '主题',
    rows: [
      ['j k', '滚动内容，读完切换楼层'],
      ['Ctrl-d Ctrl-u', '半页滚动（含主楼）'],
      ['Ctrl-f Ctrl-b', '整页滚动'],
      ['[ ] n p', '楼层上一页 / 下一页'],
      ['t', '最新楼层页'],
      ['s', 'AI 总结（流式）'],
      ['i', '查看当前楼层图片'],
      ['w', '宽屏 / 收束楼层内容'],
      ['Enter o', '打开楼层图片 / 链接'],
      ['b u x', '屏蔽菜单 / 作者 / 主题'],
      ['y', '复制原帖链接'],
      ['q h', '返回列表'],
      ['gg G', '顶部 / 底部'],
    ],
  },
  {
    title: '命令面板',
    rows: [
      [':home :recent :new', '切换列表'],
      [':node <名称>', '打开节点'],
      [':day <YYYY-MM-DD>', '按天查看'],
      [':topic <id>', '打开主题'],
      [':theme dark|light', '切换主题'],
      [':wide', '宽屏 / 收束楼层内容（也可按 w）'],
      [':refresh', '刷新当前视图'],
      [':block-user <名字>', '屏蔽用户'],
      [':block-topic <id>', '屏蔽主题'],
      [':blocks', '管理屏蔽列表'],
      [':nodes', '管理常用节点（侧栏）'],
      [':sync', '触发后台同步'],
      [':clear-cache', '清除本地缓存'],
    ],
  },
];

const COMMANDS = [
  { name: 'home', desc: '首页列表' },
  { name: 'recent', desc: '最近主题（按最后回复）' },
  { name: 'new', desc: '最新（本地归档，按创建时间）' },
  { name: 'node', desc: 'node <名称> [页码] 打开节点' },
  { name: 'day', desc: 'day <YYYY-MM-DD> 按天浏览归档' },
  { name: 'topic', desc: 'topic <id> [页码] 打开主题' },
  { name: 'refresh', desc: '重新加载当前视图' },
  { name: 'theme', desc: 'theme dark|light 切换主题' },
  { name: 'wide', desc: 'wide 宽屏 / 收束楼层内容' },
  { name: 'block-user', desc: 'block-user <用户名> 屏蔽用户' },
  { name: 'block-topic', desc: 'block-topic <id> 屏蔽主题' },
  { name: 'unblock-user', desc: 'unblock-user <用户名> 解除屏蔽' },
  { name: 'unblock-topic', desc: 'unblock-topic <id> 解除屏蔽' },
  { name: 'blocks', desc: '管理屏蔽列表' },
  { name: 'nodes', desc: '管理常用节点（侧栏搜索 / 多选 / 保存）' },
  { name: 'sync', desc: '触发后台同步' },
  { name: 'clear-cache', desc: '清除已读、位置、过滤词' },
  { name: 'login', desc: '登录 V2EX（粘贴 Cookie）' },
  { name: 'logout', desc: '退出登录，清除本地 Cookie' },
  { name: 'help', desc: '显示键盘快捷键' },
];

/* ---------- 工具 ---------- */

function pad2(value) {
  return String(value).padStart(2, '0');
}

function clamp(value, min, max) {
  return Math.min(max, Math.max(min, value));
}

function todayLocalDate() {
  const date = new Date();
  return date.getFullYear() + '-' + pad2(date.getMonth() + 1) + '-' + pad2(date.getDate());
}

function normalizeLogin(source) {
  const login = source && typeof source === 'object' ? source : {};
  return {
    hasCookie: login.hasCookie === true,
    loggedIn: login.loggedIn === true,
    user: typeof login.user === 'string' ? login.user : '',
    checkedAt: Number(login.checkedAt) || 0,
  };
}

function isEditable(element) {
  if (!element) return false;
  const tag = element.tagName;
  return tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT' || element.isContentEditable === true;
}

async function copyText(text) {
  try {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch (e) {
    /* 继续走兜底方案 */
  }
  try {
    const area = document.createElement('textarea');
    area.value = text;
    area.setAttribute('readonly', '');
    area.style.position = 'fixed';
    area.style.opacity = '0';
    document.body.append(area);
    area.select();
    const ok = document.execCommand('copy');
    area.remove();
    return ok;
  } catch (e) {
    return false;
  }
}

/* ---------- 主题与侧栏 ---------- */

function applyTheme(theme) {
  const value = theme === 'light' ? 'light' : 'dark';
  document.documentElement.dataset.theme = value;
  store.setTheme(value);
}

function setTheme(theme) {
  applyTheme(theme);
  toast('主题：' + (theme === 'light' ? '浅色' : '暗色'), 'ok');
}

// 楼层阅读宽度：默认居中窄栏，可切换为铺满横向空间，偏好写入 localStorage。
function applyWideReading(on) {
  document.documentElement.classList.toggle('topic-wide', Boolean(on));
}

function wideReadingOn() {
  return document.documentElement.classList.contains('topic-wide');
}

function toggleTopicWide() {
  const next = !wideReadingOn();
  applyWideReading(next);
  try { localStorage.setItem('v2r:wide', next ? '1' : '0'); } catch (e) { /* ignore */ }
  toast(next ? '宽屏阅读：楼层铺满横向空间' : '收束阅读：楼层恢复居中窄栏', 'ok');
  requestAnimationFrame(updateFocusCursor);
}

function setSidebarWidth(px) {
  const value = clamp(Math.round(px), 160, 380);
  document.documentElement.style.setProperty('--sidebar-w', value + 'px');
  return value;
}

function setDrawer(open) {
  document.body.classList.toggle('drawer-open', Boolean(open));
  const backdrop = $('#sidebar-backdrop');
  if (backdrop) backdrop.classList.toggle('hidden', !open);
}

function closeDrawer() {
  if (document.body.classList.contains('drawer-open')) setDrawer(false);
}

function toggleSidebar() {
  if (window.matchMedia('(max-width: 900px)').matches) {
    setDrawer(!document.body.classList.contains('drawer-open'));
    return;
  }
  const collapsed = document.body.classList.toggle('sidebar-collapsed');
  store.saveSidebarCollapsed(collapsed);
}

/* ---------- 状态消息 ---------- */

function setMsg(text, type) {
  const node = $('#st-msg');
  if (!node) return;
  node.textContent = text || '';
  node.className = 'st-msg' + (type ? ' ' + type : '');
}

function beginLoad() {
  state.loading += 1;
  const bar = $('#progress');
  bar.classList.remove('done');
  bar.classList.add('active');
}

function endLoad() {
  state.loading = Math.max(0, state.loading - 1);
  if (state.loading > 0) return;
  const bar = $('#progress');
  bar.classList.add('done');
  setTimeout(function () {
    bar.classList.remove('active', 'done');
  }, 260);
}

/* ---------- 路由 ---------- */

function routeToHash(route) {
  if (!route) return '#/home';
  if (route.view === 'topic') {
    return '#/topic/' + encodeURIComponent(route.id) + (route.page > 1 ? '?page=' + route.page : '');
  }
  const kind = route.kind || 'home';
  if (kind === 'node') {
    return '#/node/' + encodeURIComponent(route.node || '') + (route.page > 1 ? '?page=' + route.page : '');
  }
  if (kind === 'day') {
    return '#/day/' + (route.date || todayLocalDate()) + (route.page > 1 ? '?page=' + route.page : '');
  }
  return '#/' + kind + (route.page > 1 ? '?page=' + route.page : '');
}

function routeKeyOf(route) {
  if (!route) return '';
  if (route.view === 'topic') return 'topic:' + route.id + ':p' + (route.page || 1);
  return 'list:' + (route.kind || 'home') + ':' + (route.node || '') + ':' + (route.date || '') + ':p' + (route.page || 1);
}

function sameRoute(a, b) {
  return routeKeyOf(a) === routeKeyOf(b);
}

function parseHash(hash) {
  const raw = String(hash || '').replace(/^#/, '');
  if (!raw || raw === '/') return null;
  const questionIndex = raw.indexOf('?');
  const path = questionIndex >= 0 ? raw.slice(0, questionIndex) : raw;
  const query = new URLSearchParams(questionIndex >= 0 ? raw.slice(questionIndex + 1) : '');
  const page = Math.max(1, Number(query.get('page')) || 1);
  const parts = path.split('/').filter(Boolean);
  if (!parts.length) return null;
  const first = parts[0];
  if (first === 'topic' && parts[1]) {
    return { view: 'topic', id: Number(parts[1]) || parts[1], page: page };
  }
  if (first === 'node' && parts[1]) {
    return { view: 'list', kind: 'node', node: decodeURIComponent(parts[1]), date: '', page: page };
  }
  if (first === 'day') {
    return { view: 'list', kind: 'day', node: '', date: parts[1] || todayLocalDate(), page: page };
  }
  if (first === 'home' || first === 'recent' || first === 'new') {
    return { view: 'list', kind: first, node: '', date: '', page: page };
  }
  return null;
}

function routeFromLocation() {
  return parseHash(location.hash);
}

function setHashSilent(hash) {
  if (location.hash === hash) return;
  try {
    history.replaceState(null, '', hash);
  } catch (e) {
    location.replace(hash);
  }
}

function pushHistory(hash) {
  if (!hash) return;
  const items = state.history.items;
  if (items[state.history.index] === hash) return;
  items.splice(state.history.index + 1);
  items.push(hash);
  state.history.index = items.length - 1;
  if (items.length > 120) {
    state.history.items = items.slice(items.length - 120);
    state.history.index = state.history.items.length - 1;
  }
}

async function onHashChange() {
  const route = routeFromLocation();
  if (!route) {
    setHashSilent(routeToHash({ view: 'list', kind: 'home', page: 1 }));
    return;
  }
  if (state.route && sameRoute(route, state.route)) return;
  capturePosition();
  pushHistory(location.hash);
  await show(route);
}

function navigate(route, options) {
  const opts = options || {};
  const hash = routeToHash(route);
  if (opts.silent) {
    setHashSilent(hash);
    if (opts.record !== false) pushHistory(hash);
    return show(route);
  }
  if (location.hash === hash) return show(route);
  location.hash = hash;
  return Promise.resolve();
}

function goHistory(delta) {
  const target = state.history.index + delta;
  if (target < 0) {
    setMsg('没有更早的历史', 'error');
    return;
  }
  if (target >= state.history.items.length) {
    setMsg('没有更新的历史', 'error');
    return;
  }
  capturePosition();
  state.history.index = target;
  const route = parseHash(state.history.items[target]) || { view: 'list', kind: 'home', page: 1 };
  navigate(route, { silent: true, record: false });
}

function goBack() {
  goHistory(-1);
}

// 主题视图的 q/h 固定回到打开它之前的列表，而不是楼层页历史。
function returnToList() {
  if (state.lastListRoute) {
    const target = state.lastListRoute;
    const hash = routeToHash(target);
    if (location.hash === hash) {
      show(target);
    } else {
      location.hash = hash;
    }
    return;
  }
  navigate({ view: 'list', kind: 'home', node: '', date: '', page: 1 });
}

/* ---------- 位置记忆 ---------- */

function positionKey(route) {
  return routeKeyOf(route);
}

function listFilterKey(route) {
  return 'list:' + (route.kind || 'home') + ':' + (route.node || '') + ':' + (route.date || '');
}

function capturePosition() {
  const route = state.route;
  if (!route) return;
  if (route.view === 'list') {
    store.savePosition(positionKey(route), state.selection, $('#main').scrollTop);
  } else {
    store.savePosition(positionKey(route), state.topicSelection, $('#main').scrollTop);
  }
}

/* ---------- 加载 ---------- */

function nodeTitle(name) {
  const node = state.nodeMap.get(name);
  return node ? node.title || '' : '';
}

function normalizeBlocks(data) {
  const topics = new Set((data && data.topics ? data.topics : []).map(Number));
  const users = new Set((data && data.users ? data.users : []).map(String));
  return { topics: topics, users: users };
}

function filterEntries(entries) {
  return (entries || []).filter(function (entry) {
    return !state.blocks.topics.has(Number(entry.id)) && !state.blocks.users.has(String(entry.author));
  });
}

function filterReplies(replies) {
  return (replies || []).filter(function (reply) {
    return !state.blocks.users.has(String(reply.author));
  });
}

async function loadBootstrap() {
  const boot = await api.bootstrap();
  state.boot = boot || {};
  state.nodes = boot.nodes || [];
  state.allNodes = state.nodes.length ? state.nodes.slice() : null;
  state.nodeMap = new Map(state.nodes.map(function (node) { return [node.name, node]; }));
  // followedNodes 为 null 表示“用户显式清空”或“后端未配置”，此时保持为空，不要用默认节点兜底
  state.followed = Array.isArray(boot.followedNodes) ? boot.followedNodes.slice() : []
  state.config = boot.config || {};
  state.archive = boot.archive || { from: 0, to: 0 };
  state.read = new Set(Array.from(store.readSet()).map(Number));

  const bootStatus = boot.status || {};
  if (bootStatus.db) {
    state.status = bootStatus;
  } else {
    state.status = {
      db: { topics: bootStatus.dbTopics || 0, replies: bootStatus.dbReplies || 0, lists: 0 },
      fetcher: { requests: 0, rateRemaining: bootStatus.rateRemaining, lastError: '' },
      sync: { lastSiteFeed: bootStatus.lastFeedSync || 0, lastNodeFeeds: 0, running: false },
      archive: state.archive,
    };
  }
  state.login = normalizeLogin((bootStatus && bootStatus.login) || boot.login || state.login);
  R.renderFollowedNodes($('#followed-nodes'), state.nodeMap, state.followed, '');
  updateChrome();
}

async function loadBlocks() {
  try {
    state.blocks = normalizeBlocks(await api.blocks());
  } catch (error) {
    setMsg('屏蔽列表加载失败，暂时按未屏蔽处理', 'error');
  }
}

function renderLoading(text) {
  const view = $('#view');
  R.clear(view);
  const box = R.h('div', { class: 'loading' });
  box.append(R.h('span', { class: 'spinner' }), R.h('span', { text: text || '加载中…' }));
  view.append(box);
}

async function show(route) {
  closeDrawer();
  state.route = route;
  state.viewRoute = route;
  state.error = null;
  state.selection = -1;
  state.topicSelection = -1;
  if (route.view === 'list') {
    state.lastListRoute = Object.assign({}, route);
    state.topic = null;
    state.replies = [];
    state.replyEls = [];
    state.summaryPanel = null;
  } else {
    state.list = null;
    state.entries = [];
    state.rowEls = [];
    state.selectable = [];
    state.summaryPanel = null;
  }
  state.summary = null;
  state.summarySource = 'none';
  state.summaryOpen = false;
  state.streaming = false;
  state.summarySeq += 1;
  setMsg('');
  renderLoading(route.view === 'list' ? '加载列表…' : '加载主题…');
  updateChrome();
  updateEngine();
  if (route.view === 'list') {
    await loadList(route, { restore: true });
  } else {
    await loadTopic(route, { restore: true });
  }
  updateFocusCursor();
}

async function loadList(route, options) {
  const opts = options || {};
  const seq = ++state.reqSeq;
  state.viewRoute = route;
  beginLoad();
  try {
    const data = await api.list({
      kind: route.kind,
      node: route.node,
      page: route.page,
      date: route.date,
      refresh: opts.force ? 1 : '',
    });
    if (seq !== state.reqSeq) return false;
    state.list = data;
    if (data.archive) state.archive = data.archive;
    state.entries = filterEntries(data.entries || []);
    state.filter = store.filter(listFilterKey(route));
    syncFilterInput();
    renderListView(route);

    if (opts.restore) {
      const saved = store.position(positionKey(route));
      state.selection = saved ? clampSelection(saved.index) : clampSelection(-1);
      paintListSelection(false);
      const scroll = saved ? saved.scroll : 0;
      requestAnimationFrame(function () { $('#main').scrollTop = scroll; });
    } else {
      state.selection = clampSelection(state.selection);
      paintListSelection(false);
      if (opts.scrollTop) requestAnimationFrame(function () { $('#main').scrollTop = 0; });
    }

    if (opts.adopt) {
      state.route = route;
      state.lastListRoute = Object.assign({}, route);
      const hash = routeToHash(route);
      setHashSilent(hash);
      pushHistory(hash);
      updateChrome();
      updateEngine();
    }
    return true;
  } catch (error) {
    if (error.code !== 'aborted' && seq === state.reqSeq) handleLoadError(error);
    return false;
  } finally {
    if (seq === state.reqSeq) endLoad();
  }
}

async function loadTopic(route, options) {
  const opts = options || {};
  const seq = ++state.reqSeq;
  state.viewRoute = route;
  beginLoad();
  try {
    const sameTopic = state.topic && String(state.topic.topic.id) === String(route.id);
    let data;
    if (opts.repliesOnly && sameTopic) {
      const pageData = await api.replies(route.id, route.page);
      data = {
        topic: state.topic.topic,
        page: pageData.page,
        pages: pageData.pages,
        pageSize: pageData.pageSize,
        replies: pageData.replies,
        summary: state.topic.summary,
      };
    } else {
      data = await api.topic(route.id, route.page, Boolean(opts.force));
    }
    if (seq !== state.reqSeq) return false;

    if (!sameTopic || !opts.repliesOnly) {
      state.summaryStreamSeq = (state.summaryStreamSeq || 0) + 1;
      state.summarySeq += 1;
      state.streaming = false;
      state.summary = data.summary || { status: 'none', content: '', model: '', createdAt: 0 };
      state.summarySource = state.summary.status === 'ready' ? 'cache' : 'none';
      state.summaryOpen = false;
      state.summaryTopicId = Number(data.topic.id);
      state.topic = data;
      state.topicSelection = -1;
    } else {
      state.topic = data;
      state.topicSelection = -1;
    }

    markRead(route.id, true, true);
    renderTopicView();

    if (opts.adopt) {
      state.route = route;
      const hash = routeToHash(route);
      setHashSilent(hash);
      pushHistory(hash);
      updateChrome();
      updateEngine();
    }

    const main = $('#main');
    if (opts.restore) {
      const saved = store.position(positionKey(route));
      requestAnimationFrame(function () { main.scrollTop = saved ? saved.scroll : 0; });
    } else if (opts.jumpBottom) {
      requestAnimationFrame(function () {
        main.scrollTop = main.scrollHeight;
        if (state.replies.length) setFloorSelection(state.replies.length - 1);
      });
    } else {
      requestAnimationFrame(function () { main.scrollTop = 0; });
    }
    return true;
  } catch (error) {
    if (error.code !== 'aborted' && seq === state.reqSeq) handleLoadError(error);
    return false;
  } finally {
    if (seq === state.reqSeq) endLoad();
  }
}

async function reloadCurrent(opts) {
  const route = state.route;
  if (!route) return;
  const force = Boolean(opts && opts.force);
  if (route.view === 'list') {
    await loadList(route, { restore: false, force: force });
  } else {
    await loadTopic(route, { restore: false, force: force });
  }
}

function friendlyError(error) {
  if (!error) return '加载失败';
  if (error.code === 'login_required' || error.status === 403) {
    return '该内容需要登录 V2EX 才能查看（阅读器不模拟登录）';
  }
  if (error.code === 'not_found' || error.status === 404) return '内容不存在，可能已被删除';
  if (error.code === 'ai_disabled') return 'AI 总结未启用（后端 config.toml 的 [ai] 段）';
  if (error.code === 'login_failed') return 'Cookie 无效或已过期，请重新从 v2ex.com 复制完整 Cookie';
  if (error.code === 'login_unsupported') return '后端尚未支持登录接口，等新版后端就绪后再试';
  return error.message || '加载失败';
}

function handleLoadError(error) {
  if (!error) return;
  if (error.code === 'aborted') return;
  const message = friendlyError(error);
  state.error = message;
  setMsg(message.includes('按 r 重试') ? message : message + ' · 按 r 重试', 'error');
  toast(message, 'error');
  const view = $('#view');
  const hasContent = view && (view.querySelector('.list-view .row') || view.querySelector('.topic-view .op-block'));
  if (hasContent) return;
  const box = R.h('div', { class: 'error-box' });
  const heading = state.route && state.route.view === 'topic' ? '无法打开主题' : '加载列表失败';
  box.append(
    R.h('h3', { text: heading }),
    R.h('p', { text: message }),
    R.h('p', { class: 'muted' }, '按 ', R.h('kbd', { text: 'r' }), ' 重试，或按 ', R.h('kbd', { text: ':' }), ' 执行 refresh。')
  );
  if (view) {
    R.clear(view);
    view.append(box);
  }
  updateChrome();
}

/* ---------- 列表视图 ---------- */

function renderListView(route) {
  const target = route || state.viewRoute || state.route;
  const data = state.list || {};
  const nodeInfo = state.nodeMap.get(target.node);
  const restrictedNode = Boolean(nodeInfo && nodeInfo.restricted) || /登录/.test(String(data.notice || ''));
  const needsLoginHint = restrictedNode && !state.login.loggedIn && !String(data.notice || '').includes('g l');
  const loginHint = target.kind === 'node' && needsLoginHint
    ? '未登录：在 v2ex.com 登录后按 g l 粘贴 Cookie，即可查看此节点'
    : '';
  const handled = R.renderList($('#view'), {
    entries: state.entries,
    selected: state.selection,
    filter: state.filter,
    readSet: state.read,
    kind: target.kind,
    node: target.node,
    nodeTitle: nodeTitle(target.node),
    date: target.date,
    page: data.page || target.page || 1,
    pages: data.pages || 1,
    notice: data.notice || '',
    stale: Boolean(data.stale),
    fetchedAt: data.fetchedAt,
    archive: data.archive || state.archive,
    loginHint: loginHint,
  });
  state.rowEls = handled.rows;
  state.selectable = handled.matches;
  updateFilterInfo();
  updateStatusContext();
}

function clampSelection(index) {
  const selectable = state.selectable;
  if (!selectable.length) return -1;
  if (index == null || index < 0) return selectable[0];
  if (selectable.includes(index)) return index;
  const next = selectable.find(function (item) { return item >= index; });
  return next != null ? next : selectable[selectable.length - 1];
}

function paintListSelection(scroll) {
  if (!state.rowEls) return;
  state.rowEls.forEach(function (row, index) {
    row.classList.toggle('current', index === state.selection);
  });
  const row = state.rowEls[state.selection];
  if (row && scroll !== false) row.scrollIntoView({ block: 'nearest' });
  updateStatusContext();
}

function setListSelection(index) {
  if (!state.selectable.includes(index)) return;
  state.selection = index;
  paintListSelection(true);
}

function moveList(delta) {
  if (!state.selectable.length) return;
  let position = state.selectable.indexOf(state.selection);
  if (position < 0) position = 0;
  position = clamp(position + delta, 0, state.selectable.length - 1);
  setListSelection(state.selectable[position]);
}

function visibleRows() {
  return Math.max(1, Math.floor(($('#main').clientHeight - 60) / ROW_H));
}

function openSelectedTopic() {
  const entry = state.entries[state.selection];
  if (!entry) {
    toast('没有选中的主题', 'error');
    return;
  }
  navigate({ view: 'topic', id: entry.id, page: 1 });
}

function currentListPage() {
  return (state.list && state.list.page) || (state.viewRoute && state.viewRoute.page) || 1;
}

async function gotoListPage(target) {
  const pages = (state.list && state.list.pages) || 1;
  const page = clamp(Number(target) || 1, 1, pages);
  if (page === currentListPage()) {
    toast(page === 1 && target < 1 ? '已经是第一页' : '已经是最后一页');
    return;
  }
  capturePosition();
  await loadList(Object.assign({}, state.route, { page: page }), { restore: false, adopt: true, scrollTop: true });
}

function syncFilterInput() {
  const input = $('#filter-input');
  if (input) input.value = state.filter;
}

function updateFilterInfo() {
  const info = $('#filter-info');
  if (!info) return;
  if (state.route && state.route.view !== 'list') {
    info.textContent = '';
    return;
  }
  if (state.filter) {
    info.textContent = state.selectable.length + ' / ' + state.entries.length + ' 条匹配';
  } else {
    info.textContent = state.entries.length ? state.entries.length + ' 条' : '';
  }
}

function applyFilter(term) {
  if (!state.rowEls) return;
  state.filter = term;
  state.selectable = R.computeMatches(state.entries, term);
  const matchSet = new Set(state.selectable);
  state.rowEls.forEach(function (row, index) {
    row.classList.toggle('filtered', Boolean(term) && !matchSet.has(index));
  });
  if (!matchSet.has(state.selection)) {
    state.selection = state.selectable.length ? state.selectable[0] : -1;
  }
  paintListSelection(Boolean(term));
  updateFilterInfo();
  if (state.route) store.saveFilter(listFilterKey(state.route), term);
}

function focusFilter() {
  if (state.route && state.route.view !== 'list') {
    toast('过滤只在列表视图可用');
    return;
  }
  const input = $('#filter-input');
  if (!input) return;
  input.focus();
  input.select();
}

function jumpMatch(direction) {
  if (!state.filter) {
    toast('没有过滤词，按 / 输入关键词');
    return;
  }
  if (!state.selectable.length) {
    toast('没有匹配的主题');
    return;
  }
  let position = state.selectable.indexOf(state.selection);
  if (position < 0) position = direction > 0 ? -1 : 0;
  const next = (position + direction + state.selectable.length) % state.selectable.length;
  setListSelection(state.selectable[next]);
}

/* ---------- 主题视图 ---------- */

function renderTopicView() {
  const data = state.topic;
  if (!data) return;
  const topic = data.topic;
  state.replies = filterReplies(data.replies || []);
  const handled = R.renderTopic($('#view'), {
    topic: topic,
    replies: state.replies,
    page: data.page,
    pages: data.pages,
    pageSize: data.pageSize,
    selected: state.topicSelection,
    read: state.read.has(Number(topic.id)),
    blockedTopic: state.blocks.topics.has(Number(topic.id)),
  });
  state.replyEls = handled.replyEls;
  state.summaryPanel = handled.summaryPanel;
  paintSummary();
  R.paintTopicSelection($('#view'), state.topicSelection);
  updateStatusContext();
  updateFocusCursor();
}

function currentFloorPage() {
  return (state.topic && state.topic.page) || 1;
}

async function gotoFloorPage(target, options) {
  const opts = options || {};
  const pages = (state.topic && state.topic.pages) || 1;
  const page = target === 'last' ? pages : clamp(Number(target) || 1, 1, pages);
  if (page === currentFloorPage()) {
    if (page === pages) {
      toast('已经是最后一页');
    } else {
      toast('已经是第一页');
    }
    return;
  }
  capturePosition();
  await loadTopic(Object.assign({}, state.route, { page: page }), {
    repliesOnly: true,
    adopt: true,
    jumpBottom: Boolean(opts.jumpBottom),
  });
}

const SCROLL_EDGE = 8;

function mainViewport() {
  const main = $('#main');
  const rect = main.getBoundingClientRect();
  return { main: main, top: rect.top, bottom: rect.bottom, height: rect.height };
}

function topicItemEls() {
  const items = [];
  const op = document.querySelector('#op-block');
  if (op) items.push({ index: -1, el: op });
  state.replyEls.forEach(function (el, index) {
    items.push({ index: index, el: el });
  });
  return items;
}

function topicItemEl(index) {
  if (index < 0) return document.querySelector('#op-block');
  return state.replyEls[index] || null;
}

function scrollTopicStep(delta) {
  if (!delta) return false;
  const main = $('#main');
  const before = main.scrollTop;
  main.scrollTop = Math.max(0, main.scrollTop + delta);
  return main.scrollTop !== before;
}

function setFloorSelection(index) {
  state.topicSelection = index;
  R.paintTopicSelection($('#view'), index);
  updateStatusContext();
  updateFocusCursor();
}

// nvim 风格的单条光标：按当前 OP/楼层定位（长项滚出视口时贴视口上沿）。
function updateFocusCursor() {
  const cursor = document.getElementById('focus-cursor');
  const main = $('#main');
  if (!cursor || !main) return;
  if (!state.route || state.route.view !== 'topic' || !state.topic) {
    cursor.hidden = true;
    return;
  }
  const item = topicItemEl(state.topicSelection);
  if (!item) {
    cursor.hidden = true;
    return;
  }
  const mainRect = main.getBoundingClientRect();
  const itemRect = item.getBoundingClientRect();
  if (itemRect.bottom < mainRect.top + 4 || itemRect.top > mainRect.bottom - 4) {
    cursor.hidden = true;
    return;
  }
  const anchor = state.topicSelection < 0 ? item.querySelector('.op-head .author') : item.querySelector('.floor');
  const rect = (anchor || item).getBoundingClientRect();
  let left = rect.left - 10;
  let top = rect.top + 1;
  if (left < mainRect.left + 4) left = mainRect.left + 4;
  if (top < mainRect.top + 8) top = mainRect.top + 8;
  if (top > mainRect.bottom - 26) top = mainRect.bottom - 26;
  cursor.hidden = false;
  cursor.style.transform = 'translate3d(' + Math.round(left) + 'px,' + Math.round(top) + 'px,0)';
}

// 视口提前量：光标移到视口 38%~62% 之外时，页面才跟着滚动，给下一楼留预览空间。
const TOP_LEAD = 0.38;
const BOTTOM_LEAD = 0.62;

function keepTopicItemInBand(el) {
  if (!el) return;
  const vp = mainViewport();
  const rect = el.getBoundingClientRect();
  if (rect.height > vp.height * 0.6) {
    const topTarget = vp.top + vp.height * 0.12;
    if (rect.top > vp.top + vp.height * 0.2) {
      scrollTopicStep(rect.top - topTarget);
    } else if (rect.top < vp.top + SCROLL_EDGE) {
      scrollTopicStep(rect.top - (vp.top + SCROLL_EDGE));
    }
    return;
  }
  const topLimit = vp.top + vp.height * TOP_LEAD;
  const bottomLimit = vp.top + vp.height * BOTTOM_LEAD;
  if (rect.bottom > bottomLimit) {
    scrollTopicStep(rect.bottom - bottomLimit);
  } else if (rect.top < topLimit) {
    scrollTopicStep(rect.top - topLimit);
  }
}

// j/k：当前项还有内容留在视口外时先滚动，读完才切换 OP / 下一层
function stepTopicCursor(direction) {
  const count = state.replies.length;
  const current = state.topicSelection;
  const el = topicItemEl(current);
  if (!el) {
    setFloorSelection(-1);
    return;
  }
  const vp = mainViewport();
  const rect = el.getBoundingClientRect();
  if (direction > 0) {
    if (rect.bottom > vp.bottom - SCROLL_EDGE) {
      scrollTopicStep(Math.max(24, Math.min(64, rect.bottom - (vp.bottom - SCROLL_EDGE))));
      return;
    }
    if (current >= count - 1) {
      toast(count ? '已经是最后一个楼层' : '没有更多楼层');
      return;
    }
    setFloorSelection(current + 1);
    keepTopicItemInBand(topicItemEl(current + 1));
    return;
  }
  if (rect.top < vp.top + SCROLL_EDGE) {
    scrollTopicStep(-Math.max(24, Math.min(64, (vp.top + SCROLL_EDGE) - rect.top)));
    return;
  }
  if (current < 0) {
    toast('已经是顶部');
    return;
  }
  setFloorSelection(current - 1);
  keepTopicItemInBand(topicItemEl(current - 1));
}

function moveFloor(delta) {
  const steps = Math.max(1, Math.abs(delta || 1));
  const direction = delta < 0 ? -1 : 1;
  for (let i = 0; i < steps; i += 1) stepTopicCursor(direction);
}

// Ctrl-d/u/f/b 与 space：滚动后把选中项放到视口中线附近，保持阅读提前量。
function syncFloorSelectionToViewport() {
  const items = topicItemEls();
  if (!items.length) return;
  const vp = mainViewport();
  const focusY = vp.top + vp.height * 0.5;
  let containing = null;
  let nearest = null;
  let nearestDistance = Infinity;
  items.forEach(function (item) {
    const rect = item.el.getBoundingClientRect();
    if (rect.top <= focusY && rect.bottom >= focusY) containing = item;
    const itemCenter = (rect.top + rect.bottom) / 2;
    const distance = Math.abs(itemCenter - focusY);
    if (distance < nearestDistance) {
      nearestDistance = distance;
      nearest = item;
    }
  });
  const target = containing || nearest;
  if (target && target.index !== state.topicSelection) setFloorSelection(target.index);
}

function scrollTopicViewport(delta) {
  scrollTopicStep(delta);
  syncFloorSelectionToViewport();
}

function openCurrentImage() {
  if (!state.topic) return;
  let scope = null;
  if (state.topicSelection >= 0 && state.replyEls[state.topicSelection]) {
    scope = state.replyEls[state.topicSelection].querySelector('.r-body');
  } else {
    scope = $('#op-block .content');
  }
  const images = scope ? Array.from(scope.querySelectorAll('img.content-img')) : [];
  if (!images.length) {
    toast('当前楼层没有图片', 'error');
    return;
  }
  openLightbox(images.map(function (img) {
    return { src: img.currentSrc || img.src, alt: img.alt || '帖子图片' };
  }), 0);
}

function openCurrentTarget() {
  const reply = state.topicSelection >= 0 ? state.replies[state.topicSelection] : null;
  const scope = reply && state.replyEls[state.topicSelection]
    ? state.replyEls[state.topicSelection].querySelector('.r-body')
    : $('#op-block .content');
  if (!scope) return;
  const images = scope.querySelectorAll('img.content-img');
  if (images.length) {
    openCurrentImage();
    return;
  }
  const link = scope.querySelector('a[href]');
  if (link) {
    window.open(link.href, '_blank', 'noopener,noreferrer');
    return;
  }
  toast('当前楼层没有可打开的图片或链接', 'error');
}

/* ---------- AI 总结 ---------- */

function scheduleSummaryPaint() {
  if (state.summaryPaintQueued) return;
  state.summaryPaintQueued = true;
  requestAnimationFrame(function () {
    state.summaryPaintQueued = false;
    paintSummary();
  });
}

function summaryStatusLabel(summary) {
  if (state.streaming) return '生成中…';
  if (summary.status === 'ready') {
    const parts = [state.summarySource === 'cache' ? '缓存' : '已生成'];
    if (summary.model) parts.push(summary.model);
    if (state.summarySource === 'cache' && summary.createdAt) parts.push(R.fmtClock(summary.createdAt));
    return parts.join(' · ');
  }
  if (summary.status === 'error') return '生成失败';
  return '';
}

function closeSummary() {
  state.summaryOpen = false;
  paintSummary();
}

function paintSummary() {
  const panel = state.summaryPanel || document.getElementById('summary-panel');
  if (!panel) return;
  const summary = state.summary || { status: 'none', content: '' };
  const visible = state.summaryOpen && (summary.status !== 'none' || state.streaming);
  panel.classList.toggle('hidden', !visible);
  if (!visible) return;

  R.clear(panel);
  const head = R.h('div', { class: 'summary-head' });
  head.append(R.h('span', { class: 'sm-title', text: 'AI 总结' }));
  const statusText = summaryStatusLabel(summary);
  if (statusText) head.append(R.h('span', { class: 'sm-status', text: statusText, title: statusText }));
  head.append(R.h('span', { class: 'spacer' }));

  if (summary.status === 'ready' && summary.content) {
    const copy = R.h('button', { class: 'sm-btn', type: 'button', text: '复制', title: '复制总结全文' });
    copy.addEventListener('click', function () {
      copyText(summary.content).then(function (ok) {
        toast(ok ? '已复制 AI 总结' : '复制失败，可手动选中文本复制', ok ? 'ok' : 'error');
      });
    });
    head.append(copy);
  }
  if (!state.streaming && summary.status !== 'none') {
    const regen = R.h('button', { class: 'sm-btn', type: 'button', text: '重新生成', title: '重新生成总结' });
    regen.addEventListener('click', function () { generateSummary(true); });
    head.append(regen);
  }
  const close = R.h('button', { class: 'icon-btn', type: 'button', title: '关闭总结 (Esc)', text: '✕' });
  close.addEventListener('click', closeSummary);
  head.append(close);
  panel.append(head);

  const body = R.h('div', { class: 'summary-body' });
  if (summary.status === 'error') {
    body.append(R.h('p', { class: 'summary-error', text: summary.error || 'AI 总结失败。' }));
    const retry = R.h('button', { class: 'sm-btn', type: 'button', text: '重试', title: '重新生成总结' });
    retry.addEventListener('click', function () { generateSummary(true); });
    body.append(retry);
  } else if (summary.content) {
    body.append(R.renderSummaryMarkdown(summary.content));
    if (state.streaming) body.append(R.h('span', { class: 'cursor' }));
  } else if (state.streaming) {
    body.append(R.h('p', { class: 'summary-loading' }, R.h('span', { class: 'spinner' }), '正在生成…'));
  } else {
    body.append(R.h('p', { class: 'summary-hint', text: '还没有总结，按 s 生成。' }));
  }
  panel.append(body);
  if (state.streaming) body.scrollTop = body.scrollHeight;
}

async function toggleSummary() {
  if (!state.topic) return;
  if (state.summaryOpen && !state.streaming) {
    state.summaryOpen = false;
    paintSummary();
    return;
  }
  state.summaryOpen = true;
  paintSummary();
  if (state.streaming) return;
  if (state.summary && state.summary.status === 'ready' && state.summary.content) return;
  if (state.config && state.config.aiEnabled === false) {
    state.summaryOpen = false;
    paintSummary();
    toast('AI 总结未启用：在后端 config.toml 的 [ai] 段配置后可用', 'error');
    setMsg('AI 总结未启用（后端 [ai] 配置）', 'error');
    return;
  }
  await generateSummary();
}

async function generateSummary(force) {
  if (!state.topic) return;
  const topicId = state.topic.topic.id;
  const seq = ++state.summarySeq;
  state.streaming = true;
  state.summarySource = 'stream';
  state.summaryTopicId = Number(topicId);
  state.summary = { status: 'running', content: '', model: '', createdAt: 0 };
  state.summaryOpen = true;
  paintSummary();
  try {
    const result = await api.summaryStream(topicId, {
      refresh: Boolean(force),
      onDelta: function (delta, full) {
        if (seq !== state.summarySeq) return;
        state.summary = { status: 'running', content: full, model: (state.summary && state.summary.model) || '', createdAt: 0 };
        scheduleSummaryPaint();
      },
    });
    if (seq !== state.summarySeq) return;
    const content = (result && result.content) || (state.summary && state.summary.content) || '';
    state.summary = { status: 'ready', content: content, model: (result && result.model) || '', createdAt: Math.floor(Date.now() / 1000) };
    state.summarySource = 'stream';
  } catch (error) {
    if (seq !== state.summarySeq) return;
    if (error.code === 'ai_disabled') {
      state.summaryOpen = false;
      state.summary = { status: 'none', content: '', model: '' };
      toast('AI 总结未启用：在后端 config.toml 的 [ai] 段配置后可用', 'error');
    } else {
      state.summary = { status: 'error', content: '', error: error.message };
      toast('AI 总结失败：' + error.message, 'error');
    }
  } finally {
    if (seq === state.summarySeq) {
      state.streaming = false;
      paintSummary();
    }
  }
}

/* ---------- 已读 / 屏蔽 ---------- */

function markRead(id, on, quiet) {
  const key = Number(id);
  if (on) state.read.add(key);
  else state.read.delete(key);
  store.setRead(key, on);
  if (state.route && state.route.view === 'list' && state.rowEls.length) {
    const entry = state.entries.find(function (item) { return Number(item.id) === key; });
    if (entry) {
      const row = state.rowEls[state.entries.indexOf(entry)];
      if (row) row.classList.toggle('read', Boolean(on));
    }
  }
  if (!quiet) toast(on ? '已标记为已读' : '已标记为未读', 'ok');
}

function currentTarget() {
  if (!state.route) return null;
  if (state.route.view === 'list') {
    const entry = state.entries[state.selection];
    if (!entry) return null;
    return {
      topicId: Number(entry.id),
      author: String(entry.author || ''),
      title: entry.title || '',
      floor: null,
      opAuthor: String(entry.author || ''),
    };
  }
  if (state.route.view === 'topic' && state.topic) {
    const topic = state.topic.topic;
    const reply = state.topicSelection >= 0 ? state.replies[state.topicSelection] : null;
    return {
      topicId: Number(topic.id),
      author: String((reply && reply.author) || topic.author || ''),
      title: topic.title || '',
      floor: reply ? reply.floor : null,
      opAuthor: String(topic.author || ''),
    };
  }
  return null;
}

function afterBlocksChanged() {
  if (!state.route) return;
  if (state.route.view === 'list' && state.list) {
    const previous = state.selection;
    state.entries = filterEntries(state.list.entries || []);
    renderListView(state.route);
    state.selection = clampSelection(previous);
    paintListSelection(false);
  } else if (state.route.view === 'topic' && state.topic) {
    renderTopicView();
    if (state.blocks.topics.has(Number(state.topic.topic.id))) {
      toast('当前主题已被屏蔽', 'ok');
    }
  }
}

async function addBlock(kind, value, label) {
  try {
    await api.addBlock(kind, value);
    if (kind === 'topic') state.blocks.topics.add(Number(value));
    else state.blocks.users.add(String(value));
    toast('已屏蔽' + label, 'ok');
    afterBlocksChanged();
    return true;
  } catch (error) {
    toast('屏蔽失败：' + error.message, 'error');
    return false;
  }
}

async function removeBlock(kind, value) {
  try {
    await api.removeBlock(kind, value);
    if (kind === 'topic') state.blocks.topics.delete(Number(value));
    else state.blocks.users.delete(String(value));
    toast('已解除屏蔽', 'ok');
    afterBlocksChanged();
    return true;
  } catch (error) {
    toast('解除失败：' + error.message, 'error');
    return false;
  }
}

async function blockCurrentTopic() {
  const target = currentTarget();
  if (!target) {
    toast('当前没有可屏蔽的主题', 'error');
    return;
  }
  const watchingTopic = state.route && state.route.view === 'topic' && Number(state.route.id) === target.topicId;
  const ok = await addBlock('topic', target.topicId, '主题 #' + target.topicId + '（' + target.title.slice(0, 20) + '…）');
  if (ok && watchingTopic) returnToList();
}

async function blockCurrentAuthor() {
  const target = currentTarget();
  if (!target || !target.author) {
    toast('当前没有可屏蔽的作者', 'error');
    return;
  }
  await addBlock('user', target.author, '用户 ' + target.author);
}

function openBlockMenu() {
  const target = currentTarget();
  if (!target) {
    toast('当前没有可屏蔽的内容', 'error');
    return;
  }
  const items = [];
  const viewingTopic = state.route && state.route.view === 'topic';
  items.push({
    key: 'x',
    label: '屏蔽主题 #' + target.topicId,
    hint: (target.title || '').slice(0, 22),
    run: function () { blockCurrentTopic(); },
  });
  if (viewingTopic) {
    items.push({ key: 'u', label: '屏蔽楼主 ' + target.opAuthor, run: function () { blockUser(target.opAuthor); } });
    if (target.floor && target.author !== target.opAuthor) {
      items.push({
        key: 'u',
        label: '屏蔽 #' + target.floor + ' 作者 ' + target.author,
        run: function () { blockUser(target.author); },
      });
    }
  } else {
    items.push({ key: 'u', label: '屏蔽作者 ' + target.author, run: function () { blockUser(target.author); } });
  }
  items.push({
    key: 'm',
    label: '标记为' + (state.read.has(Number(target.topicId)) ? '未读' : '已读'),
    run: function () { markRead(target.topicId, !state.read.has(Number(target.topicId))); },
  });
  items.push({
    key: 'B',
    label: '管理屏蔽列表…',
    hint: state.blocks.topics.size + state.blocks.users.size + ' 条',
    run: openBlocksManagerView,
  });
  openMenu('屏蔽', items);
}

function blockUser(name) {
  if (!name) return;
  addBlock('user', name, '用户 ' + name);
}

function openBlocksManagerView() {
  const items = [];
  for (const user of state.blocks.users) items.push({ kind: 'user', value: user });
  for (const topic of state.blocks.topics) items.push({ kind: 'topic', value: topic });
  openBlocksManager(items, removeBlock);
}

function toggleReadCurrent() {
  const target = currentTarget();
  if (!target) {
    toast('没有选中的主题', 'error');
    return;
  }
  markRead(target.topicId, !state.read.has(Number(target.topicId)));
}

function copyCurrentLink() {
  const target = currentTarget();
  if (!target) {
    toast('没有可复制的链接', 'error');
    return;
  }
  const link = R.topicLink(target.topicId);
  copyText(link).then(function (ok) {
    if (ok) toast('已复制 ' + link, 'ok');
    else toast('复制失败：' + link, 'error');
  });
}

/* ---------- 节点目录 / 常用节点管理 ---------- */

async function ensureNodeCatalog() {
  if (state.nodesFull && state.allNodes && state.allNodes.length) return state.allNodes;
  const data = await api.nodes();
  if (data && Array.isArray(data.nodes) && data.nodes.length) {
    state.allNodes = data.nodes;
    state.nodesFull = true;
    for (const node of data.nodes) {
      if (!state.nodeMap.has(node.name)) state.nodeMap.set(node.name, node);
    }
  }
  return state.allNodes || state.nodes;
}

async function openNodesPicker() {
  if (state.nodesFetching) return;
  let nodes = state.allNodes || state.nodes;
  if (!state.nodesFull) {
    state.nodesFetching = true;
    setMsg('正在加载节点目录…');
    try {
      nodes = await ensureNodeCatalog();
    } catch (error) {
      toast('节点目录加载失败：' + error.message, 'error');
    } finally {
      state.nodesFetching = false;
      setMsg('');
    }
  }
  if (!nodes || !nodes.length) {
    toast('没有可用的节点目录', 'error');
    return;
  }
  openNodePicker(nodes, function (name) {
    navigate({ view: 'list', kind: 'node', node: name, page: 1 });
  }, { loggedIn: state.login.loggedIn });
}

// PUT /api/followed/nodes 成功后同步 state 并即时重画侧栏，无需整页刷新
async function applyFollowedNodes(names) {
  const data = await api.saveFollowedNodes(names);
  const followed = data && Array.isArray(data.followedNodes) ? data.followedNodes.slice() : names.slice();
  state.followed = followed;
  for (const node of (data && Array.isArray(data.nodes) ? data.nodes : [])) {
    state.nodeMap.set(node.name, Object.assign({}, state.nodeMap.get(node.name) || {}, node));
    if (state.allNodes && !state.allNodes.some(function (item) { return item.name === node.name; })) {
      state.allNodes.push(node);
    }
  }
  updateChrome();
  toast('常用节点已保存：' + followed.length + ' 个', 'ok');
  return { followedNodes: followed, nodes: (data && data.nodes) || [] };
}

async function openNodeManagerView() {
  if (state.nodeManagerBusy) return;
  state.nodeManagerBusy = true;
  setMsg('正在加载节点目录…');
  try {
    const catalog = await ensureNodeCatalog();
    if (!catalog || !catalog.length) {
      toast('没有可用的节点目录', 'error');
      return;
    }
    let followed = state.followed.slice();
    let backendReady = true;
    try {
      const data = await api.followedNodes();
      if (data && Array.isArray(data.followedNodes)) followed = data.followedNodes.slice();
      for (const node of (data && Array.isArray(data.nodes) ? data.nodes : [])) {
        state.nodeMap.set(node.name, Object.assign({}, state.nodeMap.get(node.name) || {}, node));
      }
    } catch (error) {
      if (error && (error.status === 404 || error.code === 'followed_unsupported')) {
        backendReady = false;
      } else {
        throw error;
      }
    }
    openNodeManager({
      nodes: catalog,
      followed: followed,
      backendReady: backendReady,
      onSave: applyFollowedNodes,
    });
  } catch (error) {
    toast('节点管理加载失败：' + error.message, 'error');
  } finally {
    state.nodeManagerBusy = false;
    setMsg('');
  }
}

/* ---------- 命令 ---------- */

function openPalette() {
  openCommandPalette(COMMANDS, function (line) { runCommand(line); });
}

async function triggerSync(target) {
  try {
    await api.sync(target || 'all');
    toast('已触发后台同步', 'ok');
    setTimeout(function () {
      if (!document.hidden) refreshStatus();
    }, 1500);
  } catch (error) {
    toast('同步失败：' + error.message, 'error');
  }
}

async function runCommand(line) {
  const text = String(line || '').trim().replace(/^:/, '');
  if (!text) return;
  const parts = text.split(/\s+/);
  const name = parts.shift().toLowerCase();
  const rest = parts;
  switch (name) {
    case 'home':
      return navigate({ view: 'list', kind: 'home', node: '', date: '', page: 1 });
    case 'recent':
      return navigate({ view: 'list', kind: 'recent', node: '', date: '', page: Math.max(1, Number(rest[0]) || 1) });
    case 'new':
      return navigate({ view: 'list', kind: 'new', node: '', date: '', page: Math.max(1, Number(rest[0]) || 1) });
    case 'node': {
      if (!rest[0]) {
        await openNodesPicker();
        return;
      }
      return navigate({ view: 'list', kind: 'node', node: rest[0], date: '', page: Math.max(1, Number(rest[1]) || 1) });
    }
    case 'day': {
      const date = rest[0] || todayLocalDate();
      if (!/^\d{4}-\d{2}-\d{2}$/.test(date)) {
        toast('日期格式应为 YYYY-MM-DD', 'error');
        return;
      }
      return navigate({ view: 'list', kind: 'day', node: '', date: date, page: 1 });
    }
    case 'topic': {
      const id = Number(rest[0]);
      if (!id) {
        toast('用法：:topic <id>', 'error');
        return;
      }
      return navigate({ view: 'topic', id: id, page: Math.max(1, Number(rest[1]) || 1) });
    }
    case 'refresh':
      return reloadCurrent({ force: false });
    case 'wide':
      return toggleTopicWide();
    case 'theme': {
      const theme = rest[0];
      if (theme === 'dark' || theme === 'light') setTheme(theme);
      else setTheme(document.documentElement.dataset.theme === 'light' ? 'dark' : 'light');
      return;
    }
    case 'block-user': {
      const user = rest.join(' ').trim();
      if (!user) {
        toast('用法：:block-user <用户名>', 'error');
        return;
      }
      return addBlock('user', user, '用户 ' + user);
    }
    case 'block-topic': {
      const id = Number(rest[0]);
      if (!id) {
        toast('用法：:block-topic <id>', 'error');
        return;
      }
      return addBlock('topic', id, '主题 #' + id);
    }
    case 'unblock-user': {
      const user = rest.join(' ').trim();
      if (!user) {
        toast('用法：:unblock-user <用户名>', 'error');
        return;
      }
      return removeBlock('user', user);
    }
    case 'unblock-topic': {
      const id = Number(rest[0]);
      if (!id) {
        toast('用法：:unblock-topic <id>', 'error');
        return;
      }
      return removeBlock('topic', id);
    }
    case 'blocks':
      openBlocksManagerView();
      return;
    case 'nodes':
      await openNodeManagerView();
      return;
    case 'sync':
      return triggerSync(rest[0] || 'all');
    case 'clear-cache':
      store.clearCache();
      state.read = new Set();
      state.filter = '';
      syncFilterInput();
      toast('已清除本地缓存（已读、位置、过滤词）', 'ok');
      renderListView();
      return;
    case 'login':
      openLoginPanel();
      return;
    case 'logout':
      return doLogout();
    case 'help':
      openHelpOverlay();
      return;
    case 'mock':
      toast('mock 模式：在地址后加 ?mock=1 打开', 'ok');
      return;
    default:
      toast('未知命令：' + name + '，按 : 查看可用命令', 'error');
  }
}

/* ---------- 状态栏 / 顶栏 ---------- */

function contextLabel() {
  const route = state.route;
  if (!route) return '加载中';
  if (route.view === 'topic') return '主题 #' + route.id;
  return R.listKindLabel(route.kind, route.node, nodeTitle(route.node), route.date);
}

function pageInfo() {
  const route = state.route;
  if (!route) return '';
  if (route.view === 'list') {
    const data = state.list;
    if (!data) return '';
    return '第 ' + (data.page || 1) + '/' + (data.pages || 1) + ' 页';
  }
  const data = state.topic;
  if (!data) return '';
  const page = data.page || 1;
  const pages = data.pages || 1;
  const size = data.pageSize || 100;
  const total = R.replyCount(data.topic);
  const from = total ? (page - 1) * size + 1 : 0;
  const to = Math.min(total, page * size);
  return '楼层 ' + from + '–' + to + ' / ' + total + ' · 页 ' + page + '/' + pages;
}

function hints() {
  if (!state.route) return '';
  if (state.route.view === 'list') {
    return 'j/k 选择 · Enter 打开 · [ ] 翻页 · / 过滤 · y 复制 · b 屏蔽 · ? 帮助';
  }
  return 'j/k 楼层 · [ ] 翻页 · t 最新 · s 总结 · i 图片 · b 屏蔽 · q 返回 · ? 帮助';
}

function updateStatusContext() {
  const context = $('#st-context');
  if (context) {
    context.textContent = '';
    const label = contextLabel();
    const strong = document.createElement('b');
    strong.textContent = label;
    context.append(strong);
    if (state.filter && state.route && state.route.view === 'list') {
      context.append(document.createTextNode(' · 过滤「' + state.filter + '」'));
    }
    if (state.route && state.route.view === 'topic' && state.topic) {
      const reply = state.topicSelection >= 0 ? state.replies[state.topicSelection] : null;
      context.append(document.createTextNode(' · ' + (reply ? '#' + reply.floor : '主楼')));
    }
  }
  const page = $('#st-page');
  if (page) page.textContent = pageInfo();
  const hint = $('#st-hint');
  if (hint) hint.textContent = hints();
}

function updateSyncUi() {
  const wrap = $('#sync-state');
  const text = $('#sync-text');
  const rate = $('#rate-text');
  const status = state.status;
  if (!wrap || !text) return;
  let kind = '';
  let label = '等待同步';
  let title = '';
  if (status) {
    const sync = status.sync || {};
    const fetcher = status.fetcher || {};
    if (sync.running) {
      kind = 'busy';
      label = '同步中';
    } else if (fetcher.lastError) {
      kind = 'error';
      label = '同步异常';
      title = fetcher.lastError;
    } else if (sync.lastSiteFeed) {
      kind = 'ok';
      label = '已同步 ' + R.fmtTime(sync.lastSiteFeed);
      title = '站点 feed：' + R.fmtDateTime(sync.lastSiteFeed);
    }
    if (rate) {
      if (fetcher.rateRemaining == null || fetcher.rateRemaining === 0) {
        rate.textContent = '';
      } else {
        rate.textContent = '配额 ' + fetcher.rateRemaining;
        rate.title = 'V2EX API 剩余配额（默认 600 次/小时）';
        rate.classList.toggle('warn', fetcher.rateRemaining < 60);
      }
    }
  }
  wrap.className = 'sync' + (kind ? ' ' + kind : '');
  wrap.title = title || '同步状态';
  text.textContent = label;
}

function updateArchiveFooter() {
  const foot = $('#side-archive');
  if (!foot) return;
  const archive = state.archive || {};
  const status = state.status || {};
  const db = status.db || {};
  const lines = [];
  if (archive.from || archive.to) {
    lines.push('归档 ' + (archive.from ? R.fmtShortDate(archive.from) : '?') + ' ~ ' + (archive.to ? R.fmtShortDate(archive.to) : '?'));
  } else {
    lines.push('归档：等待首次同步');
  }
  if (db.topics) lines.push('本地 ' + db.topics + ' 主题 · ' + (db.replies || 0) + ' 楼层');
  foot.textContent = lines.join('\n');
}

function updateChrome() {
  const route = state.route;
  document.querySelectorAll('#tabs .tab').forEach(function (tab) {
    const active = Boolean(route && route.view === 'list' && route.kind === tab.dataset.tab);
    tab.classList.toggle('active', active);
    tab.setAttribute('aria-current', active ? 'page' : 'false');
  });
  const activeNode = route && route.view === 'list' && route.kind === 'node' ? route.node : '';
  R.renderFollowedNodes($('#followed-nodes'), state.nodeMap, state.followed, activeNode);
  updateSyncUi();
  updateArchiveFooter();
  updateLoginUi();
  const label = contextLabel();
  document.title = (route && route.view === 'topic' ? label + ' · ' : '') + 'v2ex-vimreader';
  updateStatusContext();
}

/* ---------- 登录 ---------- */

function updateLoginUi() {
  const button = $('#login-state');
  if (!button) return;
  const login = state.login || {};
  button.classList.toggle('on', login.loggedIn === true);
  button.textContent = login.loggedIn
    ? '已登录' + (login.user ? ' ' + login.user : '')
    : (login.hasCookie ? 'Cookie 未验证' : '未登录');
  button.title = login.loggedIn
    ? 'V2EX 已登录' + (login.user ? '：' + login.user : '') + '（g l 管理登录）'
    : '登录 V2EX（g l）：粘贴 Cookie 后可查看受限节点';
}

function refreshAfterLoginChange() {
  const route = state.route;
  if (!route) return;
  const nodeInfo = route.view === 'list' && route.kind === 'node' ? state.nodeMap.get(route.node) : null;
  const restrictedNode = Boolean(nodeInfo && nodeInfo.restricted);
  const emptyNotice = route.view === 'list' && route.kind === 'node' && state.list
    && (state.list.entries || []).length === 0 && Boolean(state.list.notice);
  const loginError = Boolean(state.error && String(state.error).includes('登录'));
  if (restrictedNode || emptyNotice || loginError) {
    reloadCurrent({ force: true });
  }
}

function openLoginPanel() {
  openLogin({
    login: state.login,
    getLogin: function () { return state.login; },
    onSubmit: async function (cookie) {
      const result = await api.login(cookie);
      state.login = normalizeLogin(result);
      updateLoginUi();
      updateChrome();
      toast(state.login.loggedIn ? '登录成功' + (state.login.user ? '：' + state.login.user : '') : 'Cookie 已保存', 'ok');
      refreshStatus();
      refreshAfterLoginChange();
      return state.login;
    },
    onLogout: async function () {
      await api.logout();
      state.login = { hasCookie: false, loggedIn: false, user: '', checkedAt: Math.floor(Date.now() / 1000) };
      updateLoginUi();
      updateChrome();
      toast('已退出登录，本地 Cookie 已清除', 'ok');
      refreshStatus();
      refreshAfterLoginChange();
      return state.login;
    },
    onChromeImport: async function () {
      const result = await api.loginChrome();
      state.login = normalizeLogin(result);
      updateLoginUi();
      updateChrome();
      toast('已从 Chrome 导入：' + (state.login.user || 'Cookie'), 'ok');
      refreshStatus();
      refreshAfterLoginChange();
      return state.login;
    },
  });
}

async function doLogout() {
  try {
    await api.logout();
    state.login = { hasCookie: false, loggedIn: false, user: '', checkedAt: Math.floor(Date.now() / 1000) };
    updateLoginUi();
    updateChrome();
    toast('已退出登录，本地 Cookie 已清除', 'ok');
    refreshStatus();
    refreshAfterLoginChange();
  } catch (error) {
    toast('退出登录失败：' + error.message, 'error');
  }
}

/* ---------- 定时器 ---------- */

function updateClock() {
  const node = $('#st-clock');
  if (node) node.textContent = R.fmtClock();
}

async function refreshStatus() {
  try {
    const status = await api.status();
    if (!status) return;
    state.status = status;
    if (status.login) state.login = normalizeLogin(status.login);
    if (status.archive && (status.archive.from || status.archive.to)) state.archive = status.archive;
    updateSyncUi();
    updateArchiveFooter();
    updateLoginUi();
  } catch (error) {
    const wrap = $('#sync-state');
    if (wrap) {
      wrap.className = 'sync error';
      const text = $('#sync-text');
      if (text) text.textContent = '状态不可用';
      wrap.title = error.message;
    }
  }
}

function startTimers() {
  updateClock();
  state.clockTimer = setInterval(function () {
    if (document.hidden) return;
    updateClock();
    refreshStatus();
  }, 30000);
  document.addEventListener('visibilitychange', function () {
    if (document.hidden) {
      capturePosition();
    } else {
      updateClock();
      refreshStatus();
    }
  });
  window.addEventListener('pagehide', capturePosition);
  window.addEventListener('beforeunload', capturePosition);
}

/* ---------- 键绑定 ---------- */

function listBindings() {
  return {
    'j': function (count) { moveList(count); },
    'k': function (count) { moveList(-count); },
    'g g': function () {
      if (!state.selectable.length) return;
      state.selection = state.selectable[0];
      paintListSelection(false);
      $('#main').scrollTop = 0;
    },
    'G': function () {
      if (!state.selectable.length) return;
      state.selection = state.selectable[state.selectable.length - 1];
      paintListSelection(true);
    },
    'ctrl+d': function () { moveList(Math.max(4, Math.floor(visibleRows() / 2))); },
    'ctrl+u': function () { moveList(-Math.max(4, Math.floor(visibleRows() / 2))); },
    'ctrl+f': function () { moveList(Math.max(8, visibleRows())); },
    'ctrl+b': function () { moveList(-Math.max(8, visibleRows())); },
    'enter': openSelectedTopic,
    'l': openSelectedTopic,
    'o': openSelectedTopic,
    'h': goBack,
    'q': goBack,
    'H': function () { goHistory(-1); },
    'ctrl+l': function () { goHistory(1); },
    'g l': openLoginPanel,
    'r': function () { reloadCurrent({ force: false }); },
    'R': function () { reloadCurrent({ force: true }); },
    '[': function (count) { gotoListPage(currentListPage() - count); },
    ']': function (count) { gotoListPage(currentListPage() + count); },
    '/': focusFilter,
    'n': function () { jumpMatch(1); },
    'N': function () { jumpMatch(-1); },
    'y': copyCurrentLink,
    'b': openBlockMenu,
    'x': blockCurrentTopic,
    'u': blockCurrentAuthor,
    'm': toggleReadCurrent,
    '?': openHelpOverlay,
    ':': openPalette,
    '\\': toggleSidebar,
  };
}

function topicBindings() {
  return {
    'j': function (count) { moveFloor(count); },
    'k': function (count) { moveFloor(-count); },
    'g g': function () { setFloorSelection(-1); $('#main').scrollTop = 0; },
    'G': function () {
      const main = $('#main');
      main.scrollTop = main.scrollHeight;
      if (state.replies.length) setFloorSelection(state.replies.length - 1);
    },
    'ctrl+d': function () { scrollTopicViewport(Math.floor($('#main').clientHeight / 2)); },
    'ctrl+u': function () { scrollTopicViewport(-Math.floor($('#main').clientHeight / 2)); },
    'ctrl+f': function () { scrollTopicViewport($('#main').clientHeight - 24); },
    'ctrl+b': function () { scrollTopicViewport(-($('#main').clientHeight - 24)); },
    'space': function () { scrollTopicViewport(Math.floor($('#main').clientHeight * 0.8)); },
    '[': function (count) { gotoFloorPage(currentFloorPage() - count); },
    ']': function (count) { gotoFloorPage(currentFloorPage() + count); },
    'n': function (count) { gotoFloorPage(currentFloorPage() + count); },
    'p': function (count) { gotoFloorPage(currentFloorPage() - count); },
    't': function () { gotoFloorPage('last', { jumpBottom: true }); },
    'q': returnToList,
    'h': returnToList,
    'H': function () { goHistory(-1); },
    'ctrl+l': function () { goHistory(1); },
    'g l': openLoginPanel,
    'r': function () { reloadCurrent({ force: false }); },
    'R': function () { reloadCurrent({ force: true }); },
    'enter': openCurrentTarget,
    'o': openCurrentTarget,
    'i': openCurrentImage,
    'w': toggleTopicWide,
    's': toggleSummary,
    'b': openBlockMenu,
    'x': blockCurrentTopic,
    'u': blockCurrentAuthor,
    'y': copyCurrentLink,
    '?': openHelpOverlay,
    ':': openPalette,
    '\\': toggleSidebar,
  };
}

function updateEngine() {
  const route = state.route;
  if (route && route.view === 'topic') engine.setBindings(topicBindings());
  else engine.setBindings(listBindings());
}

function openHelpOverlay() {
  openHelp(HELP_GROUPS, '提示：数字前缀可用，如 5j 移动 5 行、3] 翻 3 页。命令面板输入 : 打开，g l（先按 g 再按 l）打开登录面板。侧栏「管理」或 :nodes 管理常用节点。');
}

function handleEscape() {
  if (state.route && state.route.view === 'topic' && state.summaryOpen) {
    state.summaryOpen = false;
    paintSummary();
    return;
  }
  if (state.filter) {
    applyFilter('');
    syncFilterInput();
    setMsg('已清除过滤');
    return;
  }
  setMsg('');
}

/* ---------- 事件绑定 ---------- */

function bindShell() {
  document.getElementById('sidebar-toggle').addEventListener('click', toggleSidebar);
  document.getElementById('help-btn').addEventListener('click', openHelpOverlay);
  document.getElementById('login-state').addEventListener('click', openLoginPanel);
  document.getElementById('nodes-btn').addEventListener('click', openNodesPicker);
  const manageNodesBtn = document.getElementById('nodes-manage-btn');
  if (manageNodesBtn) manageNodesBtn.addEventListener('click', openNodeManagerView);
  document.getElementById('sidebar-backdrop').addEventListener('click', closeDrawer);
  document.getElementById('main').addEventListener('scroll', function () {
    if (state.route && state.route.view === 'topic') requestAnimationFrame(updateFocusCursor);
  });
  window.addEventListener('resize', function () {
    if (state.route && state.route.view === 'topic') requestAnimationFrame(updateFocusCursor);
  });


  const resize = $('#sidebar-resize');
  let dragging = false;
  let startX = 0;
  let startWidth = 0;
  resize.addEventListener('pointerdown', function (event) {
    dragging = true;
    startX = event.clientX;
    startWidth = $('#sidebar').offsetWidth;
    resize.setPointerCapture(event.pointerId);
    document.body.style.cursor = 'col-resize';
  });
  resize.addEventListener('pointermove', function (event) {
    if (!dragging) return;
    setSidebarWidth(startWidth + (event.clientX - startX));
  });
  resize.addEventListener('pointerup', function (event) {
    if (!dragging) return;
    dragging = false;
    setSidebarWidth($('#sidebar').offsetWidth);
    store.saveSidebarWidth($('#sidebar').offsetWidth);
    try { resize.releasePointerCapture(event.pointerId); } catch (e) {}
    document.body.style.cursor = '';
  });

  const filterInput = $('#filter-input');
  filterInput.addEventListener('input', function () {
    applyFilter(filterInput.value.trim());
  });
  filterInput.addEventListener('keydown', function (event) {
    if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      if (filterInput.value) {
        filterInput.value = '';
        applyFilter('');
        syncFilterInput();
      } else {
        filterInput.blur();
      }
    } else if (event.key === 'Enter') {
      event.preventDefault();
      filterInput.blur();
    }
  });
  $('#filter-clear').addEventListener('click', function () {
    filterInput.value = '';
    applyFilter('');
    syncFilterInput();
  });

  $('#followed-nodes').addEventListener('click', function (event) {
    if (event.target.closest('a.node-link')) closeDrawer();
  });

  $('#view').addEventListener('click', function (event) {
    const replyRow = event.target.closest ? event.target.closest('.reply[data-i]') : null;
    if (replyRow && state.route && state.route.view === 'topic'
      && !event.target.closest('a, img, video, iframe, button, .video-card')) {
      setFloorSelection(Number(replyRow.dataset.i));
      return;
    }
    const pageButton = event.target.closest('[data-page]');
    if (pageButton && !pageButton.disabled) {
      const value = pageButton.dataset.page;
      if (state.route && state.route.view === 'topic') {
        gotoFloorPage(value === 'last' ? 'last' : Number(value), { jumpBottom: value === 'last' });
      }
      return;
    }
    const action = event.target.closest('[data-act]');
    if (!action) return;
    const act = action.dataset.act;
    if (act === 'refresh') reloadCurrent({ force: false });
    else if (act === 'summary') toggleSummary();
    else if (act === 'block') openBlockMenu();
    else if (act === 'copy') copyCurrentLink();
    else if (act === 'wide') toggleTopicWide();
  });

  $('#view').addEventListener('change', function (event) {
    const input = event.target.closest('#day-date');
    if (!input || !state.route || state.route.kind !== 'day') return;
    const value = input.value;
    if (!/^\d{4}-\d{2}-\d{2}$/.test(value || '')) return;
    navigate({ view: 'list', kind: 'day', node: '', date: value, page: 1 });
  });

  document.addEventListener('keydown', onGlobalKey);
  window.addEventListener('hashchange', onHashChange);
}

function onGlobalKey(event) {
  if (isLightboxOpen() || isOverlayOpen()) return;
  if (event.defaultPrevented) return;
  if (isEditable(event.target)) return;
  if (event.key === 'Escape') {
    handleEscape();
    return;
  }
  const handled = engine.handle(event);
  if (handled) {
    event.preventDefault();
    event.stopPropagation();
  }
}

/* ---------- 启动 ---------- */

async function boot() {
  initLightbox();
  applyTheme(store.theme());
  try { applyWideReading(localStorage.getItem('v2r:wide') === '1'); } catch (e) { /* ignore */ }
  setSidebarWidth(store.sidebarWidth());
  if (store.sidebarCollapsed()) document.body.classList.add('sidebar-collapsed');
  if (isMock) $('#mock-badge').classList.remove('hidden');
  bindShell();

  try {
    await loadBootstrap();
  } catch (error) {
    setMsg('初始化失败：' + error.message, 'error');
    toast('后端不可用：' + error.message, 'error');
  }
  try {
    await loadBlocks();
  } catch (e) {
    /* loadBlocks 内部已提示 */
  }

  updateEngine();
  const initial = routeFromLocation() || { view: 'list', kind: 'home', node: '', date: '', page: 1 };
  const hash = routeToHash(initial);
  if (location.hash !== hash) setHashSilent(hash);
  pushHistory(hash);
  await show(initial);
  startTimers();
  updateChrome();
}

boot();
