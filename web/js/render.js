// 列表 / 主题 / 楼层的 DOM 渲染与时间格式化。

import { decorateContent } from './lightbox.js?v=20261030';

export function h(tag, attrs) {
  const node = document.createElement(tag);
  if (attrs) {
    for (const key of Object.keys(attrs)) {
      const value = attrs[key];
      if (value == null || value === false) continue;
      if (key === 'class') node.className = value;
      else if (key === 'text') node.textContent = value;
      else if (key === 'html') node.innerHTML = value;
      else if (key === 'dataset') Object.assign(node.dataset, value);
      else if (key === 'style') node.style.cssText = value;
      else if (key.startsWith('on') && typeof value === 'function') node.addEventListener(key.slice(2).toLowerCase(), value);
      else node.setAttribute(key, value === true ? '' : String(value));
    }
  }
  for (let i = 2; i < arguments.length; i++) append(node, arguments[i]);
  return node;
}

function append(node, child) {
  if (child == null || child === false) return;
  if (Array.isArray(child)) {
    for (const item of child) append(node, item);
  } else if (child instanceof Node) {
    node.append(child);
  } else {
    node.append(document.createTextNode(String(child)));
  }
}

export function clear(node) {
  while (node.firstChild) node.firstChild.remove();
}

export function pad2(value) {
  return String(value).padStart(2, '0');
}

export function fmtDateTime(sec) {
  if (!sec) return '';
  const date = new Date(sec * 1000);
  return date.getFullYear() + '-' + pad2(date.getMonth() + 1) + '-' + pad2(date.getDate()) +
    ' ' + pad2(date.getHours()) + ':' + pad2(date.getMinutes());
}

export function fmtShortDate(sec) {
  if (!sec) return '';
  const date = new Date(sec * 1000);
  const now = new Date();
  if (date.getFullYear() === now.getFullYear()) return pad2(date.getMonth() + 1) + '-' + pad2(date.getDate());
  return date.getFullYear() + '-' + pad2(date.getMonth() + 1) + '-' + pad2(date.getDate());
}

export function fmtClock(sec) {
  const date = sec ? new Date(sec * 1000) : new Date();
  return pad2(date.getHours()) + ':' + pad2(date.getMinutes());
}

export function fmtTime(sec, known) {
  if (!sec) return known === false ? '时间未知' : '';
  const now = Math.floor(Date.now() / 1000);
  const diff = now - sec;
  if (diff < 0) return fmtShortDate(sec);
  if (diff < 60) return '刚刚';
  if (diff < 3600) return Math.floor(diff / 60) + ' 分钟前';
  if (diff < 86400) return Math.floor(diff / 3600) + ' 小时前';
  if (diff < 86400 * 7) return Math.floor(diff / 86400) + ' 天前';
  return fmtShortDate(sec);
}

export function topicLink(id) {
  return 'https://www.v2ex.com/t/' + id;
}

// 后端不同版本对总回复数的字段名不一致：优先 repliesCount，其次 replies。
export function replyCount(topic) {
  if (!topic) return 0;
  const raw = topic.repliesCount != null ? topic.repliesCount : topic.replies;
  const count = Number(raw);
  return Number.isFinite(count) && count > 0 ? count : 0;
}

export function fmtReplyCount(value) {
  const count = Number(value);
  if (!Number.isFinite(count) || count <= 0) return '';
  return String(count);
}

export function listKindLabel(kind, node, nodeTitle, date) {
  if (kind === 'home') return '首页';
  if (kind === 'recent') return '最近';
  if (kind === 'new') return '最新（本地归档）';
  if (kind === 'node') return '节点 ' + node + (nodeTitle ? ' · ' + nodeTitle : '');
  if (kind === 'day') return '按天 ' + (date || '');
  return kind;
}

function entryMatches(entry, filter) {
  if (!filter) return true;
  const needle = filter.toLowerCase();
  return String(entry.title || '').toLowerCase().includes(needle) ||
    String(entry.author || '').toLowerCase().includes(needle) ||
    String(entry.node || '').toLowerCase().includes(needle) ||
    String(entry.nodeTitle || '').toLowerCase().includes(needle);
}

export function computeMatches(entries, filter) {
  const matches = [];
  entries.forEach(function (entry, index) {
    if (entryMatches(entry, filter)) matches.push(index);
  });
  return matches;
}

function archiveText(archive) {
  if (!archive || (!archive.from && !archive.to)) return '';
  const from = archive.from ? fmtShortDate(archive.from) : '?';
  const to = archive.to ? fmtShortDate(archive.to) : '?';
  return '归档 ' + from + ' ~ ' + to;
}

function addToolbar(wrap, ctx) {
  const toolbar = h('div', { class: 'list-toolbar' });
  toolbar.append(h('span', { class: 'ltitle', text: listKindLabel(ctx.kind, ctx.node, ctx.nodeTitle, ctx.date) }));
  if (ctx.entries && ctx.entries.length) {
    toolbar.append(h('span', { class: 'sep', text: '·' }));
    toolbar.append(h('span', { text: ctx.entries.length + ' 条' }));
  }
  if (ctx.pages > 1) {
    toolbar.append(h('span', { class: 'sep', text: '·' }));
    toolbar.append(h('span', { class: 'num', text: '第 ' + ctx.page + '/' + ctx.pages + ' 页' }));
  }
  if (ctx.stale) toolbar.append(h('span', { class: 'stale', text: '缓存数据' }));
  if (ctx.fetchedAt) {
    toolbar.append(h('span', { class: 'muted', text: '更新于 ' + fmtClock(ctx.fetchedAt), title: fmtDateTime(ctx.fetchedAt) }));
  }
  const spacer = h('span', { class: 'spacer' });
  toolbar.append(spacer);

  if (ctx.kind === 'day') {
    const input = h('input', { type: 'date', id: 'day-date', value: ctx.date || '', title: '选择日期' });
    toolbar.append(h('span', { class: 'muted', text: '日期' }), input);
    if (ctx.archive) toolbar.append(h('span', { class: 'day-archive', text: archiveText(ctx.archive) }));
  } else if (ctx.kind === 'new' || ctx.kind === 'home' || ctx.kind === 'recent') {
    if (ctx.archive) toolbar.append(h('span', { class: 'day-archive', text: archiveText(ctx.archive) }));
  }
  wrap.append(toolbar);
}

function addNotice(wrap, notice, loginHint) {
  if (!notice && !loginHint) return;
  const box = h('div', { class: 'notice', text: notice || '' });
  if (loginHint) box.append(h('span', { class: 'notice-login', text: loginHint }));
  wrap.append(box);
}

function addEmpty(wrap, ctx) {
  const box = h('div', { class: 'empty' });
  let title = '这里没有内容';
  const paragraphs = [];
  if (ctx.notice) {
    title = '受限节点不可见';
    paragraphs.push('该节点需要登录 V2EX 才能浏览，阅读器不模拟登录，也不抓取受限内容。');
    paragraphs.push('可以按 ? 查看其他可用节点，或用 :node <名称> 切换。');
  } else if (ctx.kind === 'day') {
    title = '这一天没有归档主题';
    paragraphs.push('按天视图依赖本地归档：只包含开始同步之后 feed 收录的主题。');
    if (ctx.archive && ctx.archive.from) {
      paragraphs.push('当前归档覆盖 ' + fmtShortDate(ctx.archive.from) + ' ~ ' + fmtShortDate(ctx.archive.to || Math.floor(Date.now() / 1000)) + '，换一个日期试试。');
    } else {
      paragraphs.push('归档还是空的，先执行一次同步（:sync）。');
    }
  } else if (ctx.kind === 'new') {
    title = '本地归档还没有主题';
    paragraphs.push('「最新」按创建时间排序，数据来自后台增量同步。用 :sync 手动触发一次同步。');
  } else if (ctx.kind === 'node') {
    title = '该节点暂时没有可读主题';
    paragraphs.push('可能是节点清空、页面结构变化，或本地缓存过期。按 R 强制刷新试试。');
  } else if (ctx.filter) {
    title = '过滤后没有匹配的主题';
    paragraphs.push('按 Esc 清除过滤词，或换一个关键词。');
  } else {
    paragraphs.push('按 r 刷新，或切换其他节点。');
  }
  box.append(h('h3', { text: title }));
  for (const text of paragraphs) box.append(h('p', { text }));
  if (ctx.notice) box.append(h('p', { class: 'muted', text: '后端提示：' + ctx.notice }));
  wrap.append(box);
}

export function renderList(view, ctx) {
  clear(view);
  const wrap = h('div', { class: 'list-view' });
  addToolbar(wrap, ctx);
  addNotice(wrap, ctx.notice, ctx.loginHint);

  const entries = ctx.entries || [];
  const filter = ctx.filter || '';
  const matches = computeMatches(entries, filter);
  const matchSet = new Set(matches);
  const rowsEl = h('div', { class: 'list-rows', role: 'list' });
  const rows = [];

  entries.forEach(function (entry, index) {
    const read = ctx.readSet && (ctx.readSet.has(entry.id) || ctx.readSet.has(String(entry.id)));
    const row = h('a', {
      class: 'row' + (index === ctx.selected ? ' current' : '') + (read ? ' read' : ''),
      href: '#/topic/' + entry.id,
      'data-i': index,
      role: 'listitem',
      title: entry.title || '',
    });
    // 固定数字列在最左：所有回复数右对齐成一列，标题永远从同一 x 开始；未知（<=0 / 缺失）显示 -
    const replies = entry.replies == null ? NaN : Number(entry.replies);
    const repliesKnown = Number.isFinite(replies) && replies > 0;
    row.append(h('span', {
      class: 'col num replies' + (repliesKnown ? '' : ' unknown'),
      text: repliesKnown ? String(replies) : '-',
      title: repliesKnown ? '回复数 ' + replies : '回复数未知',
    }));
    const titleCell = h('span', { class: 'title-cell' });
    titleCell.append(h('span', { class: 'title', text: entry.title || '(无标题)' }));
    if (entry.pinned) titleCell.append(h('span', { class: 'pin-tag', text: '置顶' }));
    row.append(titleCell);
    const nodeLabel = entry.node || '';
    const nodeTitle = entry.nodeTitle || '';
    row.append(h('span', { class: 'col node', text: nodeLabel, title: nodeTitle ? '节点 ' + nodeLabel + ' · ' + nodeTitle : '节点 ' + nodeLabel }));
    row.append(h('span', { class: 'col author', text: entry.author || '', title: '作者 ' + (entry.author || '') }));
    const touched = entry.lastTouchedAt || entry.createdAt || 0;
    row.append(h('span', {
      class: 'col num time',
      text: fmtTime(touched, entry.createdKnown),
      title: (entry.createdAt ? '创建 ' + fmtDateTime(entry.createdAt) : '创建时间未知') + (touched ? ' · 最后回复 ' + fmtDateTime(touched) : ''),
    }));
    if (filter && !matchSet.has(index)) row.classList.add('filtered');
    rows.push(row);
    rowsEl.append(row);
  });

  if (!entries.length) {
    addEmpty(wrap, ctx);
  } else {
    wrap.append(rowsEl);
  }

  const hint = h('div', { class: 'list-hint' });
  const hintRow = h('div', { class: 'hint-row' });
  const hints = ctx.kind === 'day'
    ? [['j/k', '选择'], ['Enter', '打开'], ['[ ]', '翻页'], ['/', '过滤'], ['?', '帮助']]
    : [['j/k', '选择'], ['Enter', '打开'], ['[ ]', '翻页'], ['/ 过滤', ''], ['y 复制', ''], ['b 屏蔽', ''], ['?', '帮助']];
  for (const pair of hints) {
    const item = h('span');
    const kbd = h('kbd', { text: pair[0] });
    item.append(kbd);
    if (pair[1]) item.append(' ', pair[1]);
    hintRow.append(item);
  }
  hint.append(hintRow);
  if (ctx.pages > 1) {
    const pageRow = h('div', { class: 'hint-row', style: 'margin-top:4px' });
    pageRow.append(h('span', { text: '第 ' + ctx.page + ' / ' + ctx.pages + ' 页' }));
    if (ctx.page > 1) {
      pageRow.append(h('span', {}, h('kbd', { text: '[' }), ' 上一页'));
    }
    if (ctx.page < ctx.pages) {
      pageRow.append(h('span', {}, h('kbd', { text: ']' }), ' 下一页'));
    }
    hint.append(pageRow);
  }
  wrap.append(hint);
  view.append(wrap);
  return { rows: rows, matches: matches, entries: entries };
}

export function renderSummaryMarkdown(text) {
  const root = h('div', { class: 'md-summary' });
  const lines = String(text || '').split('\n');
  let paragraph = [];
  let list = null;
  let listOrdered = false;

  const flushParagraph = function () {
    if (!paragraph.length) return;
    const p = h('p');
    appendInline(p, paragraph.join(' '));
    root.append(p);
    paragraph = [];
  };
  const closeList = function () {
    list = null;
  };

  for (const rawLine of lines) {
    const line = rawLine.replace(/\s+$/, '');
    if (!line.trim()) {
      flushParagraph();
      closeList();
      continue;
    }
    const heading = line.match(/^(#{1,6})\s+(.*)$/);
    if (heading) {
      flushParagraph();
      closeList();
      const level = heading[1].length;
      const node = h(level <= 2 ? 'h3' : 'h4');
      appendInline(node, heading[2]);
      root.append(node);
      continue;
    }
    const ordered = line.match(/^\s*(\d+)[.)]\s+(.*)$/);
    const bullet = line.match(/^\s*[-*+]\s+(.*)$/);
    if (ordered || bullet) {
      flushParagraph();
      const orderedList = Boolean(ordered);
      if (!list || listOrdered !== orderedList) {
        list = h(orderedList ? 'ol' : 'ul');
        listOrdered = orderedList;
        root.append(list);
      }
      const item = h('li');
      appendInline(item, ordered ? ordered[2] : bullet[1]);
      list.append(item);
      continue;
    }
    if (line.startsWith('>')) {
      flushParagraph();
      closeList();
      const quote = h('blockquote');
      appendInline(quote, line.replace(/^>\s?/, ''));
      root.append(quote);
      continue;
    }
    paragraph.push(line.trim());
  }
  flushParagraph();
  return root;
}

function appendInline(parent, text) {
  const parts = String(text).split(/(\*\*[^*]+\*\*|\x60[^\x60]+\x60)/g);
  for (const part of parts) {
    if (!part) continue;
    if (part.startsWith('**') && part.endsWith('**')) {
      parent.append(h('strong', { text: part.slice(2, -2) }));
    } else if (part.startsWith('\x60') && part.endsWith('\x60')) {
      parent.append(h('code', { text: part.slice(1, -1) }));
    } else {
      parent.append(document.createTextNode(part));
    }
  }
}

function buildTopicHead(ctx) {
  const topic = ctx.topic;
  const head = h('div', { class: 'topic-head' });
  const main = h('div', { class: 'head-main' });
  const crumb = h('div', { class: 'crumb' });
  crumb.append(
    h('a', { class: 'node-tag', href: '#/node/' + topic.node, text: '#' + topic.node }),
    h('span', { class: 'sep', text: '›' }),
    h('span', { text: topic.nodeTitle || topic.node || '' })
  );
  const title = h('h1', { class: 'topic-title', text: topic.title || '' });
  const meta = h('div', { class: 'topic-meta' });
  meta.append(
    h('span', { class: 'author', text: '楼主 ' + (topic.author || '') }),
    h('span', { class: 'num', text: topic.createdAt ? fmtDateTime(topic.createdAt) : '创建时间未知' }),
    h('span', { class: 'num', text: '回复 ' + replyCount(topic) }),
    h('span', { class: 'num', text: topic.views > 0 ? '浏览 ' + topic.views : '' }),
    h('span', { class: 'num', text: ctx.read ? '已读' : '未读' })
  );
  main.append(crumb, title, meta);
  head.append(main);

  const actions = h('div', { class: 'topic-actions' });
  const button = function (act, label, key, title) {
    const btn = h('button', { class: 'tbtn', type: 'button', 'data-act': act, title: title });
    btn.append(label);
    if (key) btn.append(h('em', { text: key }));
    return btn;
  };
  actions.append(
    button('refresh', '刷新', 'r', '重新加载（r）'),
    button('wide', '宽屏', 'w', '宽屏 / 收束楼层内容（w）'),
    button('summary', 'AI 总结', 's', '生成/查看 AI 总结（s）'),
    button('block', '屏蔽', 'b', '屏蔽菜单（b）'),
    button('copy', '复制链接', 'y', '复制原帖链接（y）')
  );
  head.append(actions);
  return head;
}

export function renderTopic(view, ctx) {
  clear(view);
  const topic = ctx.topic;
  const wrap = h('div', { class: 'topic-view' });
  wrap.append(buildTopicHead(ctx));

  if (ctx.blockedTopic) {
    wrap.append(h('div', { class: 'notice', text: '这个主题已被你屏蔽，仍然显示内容；在屏蔽列表里可以解除。' }));
  }

  const opBlock = h('article', { class: 'op-block', id: 'op-block' });
  const opHead = h('div', { class: 'op-head' });
  opHead.append(
    h('span', { class: 'author', text: (topic.author || '') }),
    h('span', { class: 'badge op-badge', text: '楼主' }),
    h('span', { class: 'num', text: topic.createdAt ? fmtDateTime(topic.createdAt) : '创建时间未知' })
  );
  const opBody = h('div', { class: 'content' });
  opBody.innerHTML = topic.contentHtml || '<p class="muted">（正文为空）</p>';
  decorateContent(opBody);
  opBlock.append(opHead, opBody);
  wrap.append(opBlock);

  const summaryPanel = h('section', { class: 'summary-panel hidden', id: 'summary-panel' });
  wrap.append(summaryPanel);

  const repliesEl = h('div', { class: 'replies' });
  const replyEls = [];
  (ctx.replies || []).forEach(function (reply, index) {
    const article = h('article', {
      class: 'reply' + (reply.isOp ? ' by-op' : '') + (index === ctx.selected ? ' current' : ''),
      id: 'floor-' + reply.floor,
      'data-floor': reply.floor,
      'data-i': index,
    });
    const meta = h('div', { class: 'r-meta' });
    meta.append(h('span', { class: 'floor', text: '#' + reply.floor }));
    const authorLine = h('span', { class: 'r-author-line' });
    authorLine.append(h('span', { class: 'author', text: reply.author || '', title: reply.author || '' }));
    if (reply.isOp) authorLine.append(h('span', { class: 'op-badge', text: 'OP', title: '楼主回复' }));
    meta.append(authorLine);
    meta.append(h('span', {
      class: 'r-time',
      text: reply.createdAt ? fmtTime(reply.createdAt) : '',
      title: reply.createdAt ? fmtDateTime(reply.createdAt) : '',
    }));
    const body = h('div', { class: 'r-body content' });
    body.innerHTML = reply.contentHtml || '';
    decorateContent(body);
    article.append(meta, body);
    repliesEl.append(article);
    replyEls.push(article);
  });
  if (!replyEls.length) {
    const total = replyCount(topic);
    const text = total === 0
      ? '还没有回复。'
      : '这一页没有楼层（可能页码超出范围，按 t 跳到最新页）。';
    repliesEl.append(h('p', { class: 'muted', text: text }));
  }
  wrap.append(repliesEl);
  wrap.append(buildTopicFoot(ctx));
  view.append(wrap);
  return { replyEls: replyEls, summaryPanel: summaryPanel };
}

function buildTopicFoot(ctx) {
  const foot = h('div', { class: 'topic-foot' });
  const pages = ctx.pages || 1;
  const page = ctx.page || 1;
  const prev = h('button', { class: 'page-btn', type: 'button', 'data-page': String(page - 1), text: '‹ 上一页 [', title: '上一页（[ 或 p）' });
  const next = h('button', { class: 'page-btn', type: 'button', 'data-page': String(page + 1), text: '下一页 ] ›', title: '下一页（] 或 n）' });
  prev.disabled = page <= 1;
  next.disabled = page >= pages;
  foot.append(
    h('span', { class: 'num', text: '楼层第 ' + page + ' / ' + pages + ' 页' }),
    prev,
    next,
    h('button', { class: 'page-btn', type: 'button', 'data-page': 'last', text: '最新页 t', title: '跳到最新页（t）' }),
    h('span', { class: 'muted', text: '共 ' + replyCount(ctx.topic) + ' 条回复' })
  );
  return foot;
}

export function paintTopicSelection(view, index) {
  const op = view.querySelector('#op-block');
  if (op) op.classList.toggle('current', index < 0);
  const nodes = view.querySelectorAll('.reply');
  nodes.forEach(function (node, i) { node.classList.toggle('current', i === index); });
}

export function renderFollowedNodes(container, nodes, followed, activeNode) {
  clear(container);
  if (!followed.length) {
    container.append(h('p', { class: 'muted', style: 'font-size:11.5px;padding:2px 4px' }, '没有关注节点，点上方「管理」添加。'));
    return;
  }
  for (const name of followed) {
    const node = nodes.get(name);
    const link = h('a', { class: 'node-link' + (name === activeNode ? ' active' : ''), href: '#/node/' + name, title: node ? node.title || name : name });
    link.append(h('span', { class: 'n-name', text: name }));
    container.append(link);
  }
}
