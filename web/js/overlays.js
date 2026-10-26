// 浮层：帮助、命令面板、屏蔽菜单/列表、节点选择、Toast。

let stack = [];
let docHandlerInstalled = false;

function ensureDocHandler() {
  if (docHandlerInstalled) return;
  docHandlerInstalled = true;
  document.addEventListener('keydown', function (event) {
    if (!stack.length) return;
    const top = stack[stack.length - 1];
    if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      if (top.onEscape && top.onEscape(event) === true) return;
      top.close();
      return;
    }
    if (event.key === 'Tab') {
      if (top.onTab && top.onTab(event)) {
        event.preventDefault();
        return;
      }
      trapTab(event, top.panel);
    }
  }, true);
}

function trapTab(event, panel) {
  const items = Array.from(panel.querySelectorAll('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'))
    .filter(function (node) { return node.offsetParent !== null && !node.disabled; });
  if (!items.length) return;
  event.preventDefault();
  const current = items.indexOf(document.activeElement);
  const delta = event.shiftKey ? -1 : 1;
  const next = (current + delta + items.length) % items.length;
  items[next].focus({ preventScroll: true });
}

function closeAll() {
  while (stack.length) stack[stack.length - 1].close();
}

function openOverlay(options) {
  ensureDocHandler();
  closeAll();
  const wrapper = document.createElement('div');
  wrapper.className = 'overlay' + (options.className ? ' ' + options.className : '');
  wrapper.setAttribute('role', 'dialog');
  wrapper.setAttribute('aria-modal', 'true');
  wrapper.setAttribute('aria-label', options.title || '对话框');

  const panel = document.createElement('div');
  panel.className = 'overlay-panel';
  panel.tabIndex = -1;

  const head = document.createElement('div');
  head.className = 'overlay-head';
  const heading = document.createElement('h2');
  heading.textContent = options.title || '';
  head.append(heading);
  if (options.sub) {
    const sub = document.createElement('span');
    sub.className = 'sub';
    sub.textContent = options.sub;
    head.append(sub);
  }
  const closeBtn = document.createElement('button');
  closeBtn.className = 'icon-btn';
  closeBtn.type = 'button';
  closeBtn.title = '关闭 (Esc)';
  closeBtn.setAttribute('aria-label', '关闭');
  closeBtn.textContent = '✕';
  head.append(closeBtn);

  const body = document.createElement('div');
  body.className = 'overlay-body';

  panel.append(head, body);
  wrapper.append(panel);
  document.getElementById('overlay-root').append(wrapper);

  let previousFocus = document.activeElement;
  const entry = {
    wrapper: wrapper,
    panel: panel,
    body: body,
    close: function () {
      if (!entry.closed) {
        entry.closed = true;
        if (options.documentKey) document.removeEventListener('keydown', keyHandler);
        wrapper.remove();
        const idx = stack.indexOf(entry);
        if (idx >= 0) stack.splice(idx, 1);
        if (previousFocus && previousFocus.focus) previousFocus.focus({ preventScroll: true });
        previousFocus = null;
      }
    },
    onKey: options.onKey || null,
    onTab: options.onTab || null,
    onEscape: options.onEscape || null,
  };

  const keyHandler = function (event) {
    if (entry.closed) return;
    if (entry.onKey && entry.onKey(event) === true) {
      event.preventDefault();
      event.stopPropagation();
    }
  };
  // 焦点可能落在浮层之外（例如按钮被禁用、重渲染），管理器用 document 级监听兜底
  if (options.documentKey) document.addEventListener('keydown', keyHandler);
  else wrapper.addEventListener('keydown', keyHandler);
  wrapper.addEventListener('mousedown', function (event) {
    if (event.target === wrapper) entry.close();
  });
  closeBtn.addEventListener('click', function () { entry.close(); });

  stack.push(entry);
  const built = options.build ? options.build(body, entry) : null;
  const focusTarget = (typeof options.focus === 'string' ? body.querySelector(options.focus) : null) || built || panel;
  if (focusTarget && focusTarget.focus) focusTarget.focus({ preventScroll: true });
  return entry;
}

export function isOverlayOpen() {
  return stack.length > 0;
}

export function closeOverlay() {
  if (stack.length) stack[stack.length - 1].close();
}

/* ---------- toast ---------- */

let toastNode = null;
let toastTimer = 0;

export function toast(message, type, ms) {
  const host = document.getElementById('toast-root');
  if (!host) return null;
  if (toastTimer) {
    clearTimeout(toastTimer);
    toastTimer = 0;
  }
  if (toastNode) toastNode.remove();
  const node = document.createElement('div');
  node.className = 'toast' + (type ? ' ' + type : '');
  node.textContent = message;
  host.append(node);
  toastNode = node;
  const timeout = ms == null ? (type === 'error' ? 3800 : 2600) : ms;
  toastTimer = setTimeout(function () {
    toastTimer = 0;
    node.classList.add('hide');
    setTimeout(function () {
      node.remove();
      if (toastNode === node) toastNode = null;
    }, 220);
  }, timeout);
  return node;
}

/* ---------- 帮助 ---------- */

export function openHelp(groups, note) {
  return openOverlay({
    title: '键盘快捷键',
    sub: 'Esc / ? 关闭',
    className: 'wide',
    build: function (body) {
      const root = document.createElement('div');
      root.className = 'help';
      for (const group of groups) {
        const section = document.createElement('section');
        const heading = document.createElement('h3');
        heading.textContent = group.title;
        const list = document.createElement('dl');
        for (const row of group.rows) {
          const dt = document.createElement('dt');
          for (const part of String(row[0]).split(/\s+/)) {
            if (!part) continue;
            const kbd = document.createElement('kbd');
            kbd.textContent = part;
            dt.append(kbd, ' ');
          }
          const dd = document.createElement('dd');
          dd.textContent = row[1];
          list.append(dt, dd);
        }
        section.append(heading, list);
        root.append(section);
      }
      if (note) {
        const extra = document.createElement('p');
        extra.className = 'help-note';
        extra.textContent = note;
        root.append(extra);
      }
      body.append(root);
      return null;
    },
    onKey: function (event) {
      if (event.key === '?' || event.key === 'q') {
        closeOverlay();
        return true;
      }
      return scrollKey(event, stack[stack.length - 1]);
    },
  });
}

function scrollKey(event, entry) {
  let delta = 0;
  if (event.key === 'j' || event.key === 'ArrowDown' || event.key === 'PageDown') delta = event.key === 'PageDown' ? 240 : 44;
  else if (event.key === 'k' || event.key === 'ArrowUp' || event.key === 'PageUp') delta = event.key === 'PageUp' ? -240 : -44;
  else if (event.key === ' ') delta = 240;
  if (!delta || !entry) return false;
  entry.body.scrollTop += delta;
  return true;
}

/* ---------- 命令面板 ---------- */

export function openCommandPalette(commands, onRun) {
  let filtered = commands.slice();
  let selected = 0;
  let listEl = null;
  let inputEl = null;

  const renderList = function () {
    listEl.textContent = '';
    if (!filtered.length) {
      const empty = document.createElement('li');
      empty.className = 'palette-empty';
      empty.textContent = '没有匹配的命令';
      listEl.append(empty);
      return;
    }
    selected = Math.min(selected, filtered.length - 1);
    filtered.forEach(function (command, idx) {
      const item = document.createElement('li');
      item.className = 'palette-item' + (idx === selected ? ' selected' : '');
      const name = document.createElement('span');
      name.className = 'pname';
      name.textContent = ':' + command.name;
      const desc = document.createElement('span');
      desc.className = 'pdesc';
      desc.textContent = command.desc || command.usage || '';
      item.append(name, desc);
      item.addEventListener('mouseenter', function () { selected = idx; renderList(); });
      item.addEventListener('click', function () {
        entry.close();
        onRun(command.name);
      });
      listEl.append(item);
    });
  };

  const refresh = function () {
    const query = inputEl.value.trim().toLowerCase();
    const head = query.split(/\s+/)[0];
    filtered = commands.filter(function (command) {
      if (!head) return true;
      return command.name.startsWith(head) || (command.desc || '').toLowerCase().includes(head);
    });
    if (!filtered.length) filtered = [];
    selected = 0;
    renderList();
  };

  const entry = openOverlay({
    title: '命令',
    sub: 'Enter 执行 · Tab 补全 · Esc 取消',
    className: 'narrow',
    focus: 'input',
    build: function (body) {
      inputEl = document.createElement('input');
      inputEl.className = 'palette-input';
      inputEl.type = 'text';
      inputEl.spellcheck = false;
      inputEl.autocomplete = 'off';
      inputEl.placeholder = '例如 node tech、day 2026-10-26、theme light、sync';
      inputEl.addEventListener('input', refresh);
      listEl = document.createElement('ul');
      listEl.className = 'palette-list';
      body.append(inputEl, listEl);
      refresh();
      return inputEl;
    },
    onKey: function (event) {
      if (event.ctrlKey && event.key.toLowerCase() === 'n') {
        selected = Math.min(filtered.length - 1, selected + 1);
        renderList();
        return true;
      }
      if (event.ctrlKey && event.key.toLowerCase() === 'p') {
        selected = Math.max(0, selected - 1);
        renderList();
        return true;
      }
      const handler = {
        ArrowDown: function () { selected = Math.min(filtered.length - 1, selected + 1); renderList(); return true; },
        ArrowUp: function () { selected = Math.max(0, selected - 1); renderList(); return true; },
        down: function () { selected = Math.min(filtered.length - 1, selected + 1); renderList(); return true; },
        up: function () { selected = Math.max(0, selected - 1); renderList(); return true; },
        Enter: function () {
          const raw = inputEl.value.trim();
          entry.close();
          if (raw) onRun(raw);
          else if (filtered[selected]) onRun(filtered[selected].name);
          return true;
        },
      }[event.key];
      return handler ? handler() : false;
    },
    onTab: function (event) {
      if (event.key !== 'Tab') return false;
      const command = filtered[selected];
      if (!command) return true;
      inputEl.value = command.name + ' ';
      refresh();
      inputEl.setSelectionRange(inputEl.value.length, inputEl.value.length);
      return true;
    },
  });
  return entry;
}

/* ---------- 菜单 ---------- */

export function openMenu(title, items) {
  let selected = 0;
  let listEl = null;
  let entryRef = null;

  const render = function () {
    listEl.textContent = '';
    items.forEach(function (item, idx) {
      const row = document.createElement('li');
      row.className = 'menu-item' + (idx === selected ? ' selected' : '');
      const key = document.createElement('span');
      key.className = 'mkey';
      key.textContent = item.key || '';
      const label = document.createElement('span');
      label.className = 'mlabel';
      label.textContent = item.label;
      row.append(key, label);
      if (item.hint) {
        const hint = document.createElement('span');
        hint.className = 'mhint';
        hint.textContent = item.hint;
        row.append(hint);
      }
      row.addEventListener('mouseenter', function () { selected = idx; render(); });
      row.addEventListener('click', function () {
        entryRef.close();
        item.run();
      });
      listEl.append(row);
    });
  };

  const entry = openOverlay({
    title: title || '菜单',
    className: 'narrow',
    build: function (body) {
      listEl = document.createElement('ul');
      listEl.className = 'menu';
      body.append(listEl);
      render();
      return listEl.querySelector('.menu-item');
    },
    onKey: function (event) {
      if (event.key === 'j' || event.key === 'ArrowDown') {
        selected = Math.min(items.length - 1, selected + 1);
        render();
        return true;
      }
      if (event.key === 'k' || event.key === 'ArrowUp') {
        selected = Math.max(0, selected - 1);
        render();
        return true;
      }
      if (event.key === 'Enter') {
        const item = items[selected];
        if (!item) return true;
        entry.close();
        item.run();
        return true;
      }
      return false;
    },
  });
  entryRef = entry;
  return entry;
}

/* ---------- 屏蔽列表 ---------- */

export function openBlocksManager(items, onRemove) {
  let selected = 0;
  let listEl = null;
  const state = { items: items.slice() };

  const render = function () {
    listEl.textContent = '';
    if (!state.items.length) {
      const empty = document.createElement('p');
      empty.className = 'muted';
      empty.textContent = '还没有屏蔽任何主题或用户。列表里按 b 打开屏蔽菜单，或用 :block-user 命令。';
      listEl.append(empty);
      return;
    }
    const kinds = [
      ['user', '屏蔽的用户'],
      ['topic', '屏蔽的主题'],
    ];
    for (const pair of kinds) {
      const group = state.items.filter(function (item) { return item.kind === pair[0]; });
      if (!group.length) continue;
      const split = document.createElement('div');
      split.className = 'blocks-split';
      split.textContent = pair[1];
      listEl.append(split);
      for (const item of group) {
        const idx = state.items.indexOf(item);
        const row = document.createElement('li');
        row.className = 'block-row' + (idx === selected ? ' selected' : '');
        const kindEl = document.createElement('span');
        kindEl.className = 'bkind';
        kindEl.textContent = item.kind === 'user' ? '用户' : '主题';
        const value = document.createElement('span');
        value.className = 'bval';
        value.textContent = String(item.value);
        const remove = document.createElement('a');
        remove.className = 'bremove';
        remove.href = '#';
        remove.textContent = '解除';
        remove.addEventListener('click', function (event) {
          event.preventDefault();
          removeAt(idx);
        });
        row.append(kindEl, value, remove);
        row.addEventListener('mouseenter', function () { selected = idx; render(); });
        listEl.append(row);
      }
    }
  };

  const removeAt = function (idx) {
    const item = state.items[idx];
    if (!item) return;
    onRemove(item.kind, item.value).then(function (ok) {
      if (!ok) return;
      state.items = state.items.filter(function (entry) { return entry !== item; });
      selected = Math.max(0, Math.min(selected, state.items.length - 1));
      render();
    });
  };

  const entry = openOverlay({
    title: '屏蔽列表',
    sub: 'j/k 选择 · Enter 解除 · Esc 关闭',
    className: 'narrow',
    build: function (body) {
      listEl = document.createElement('ul');
      listEl.className = 'blocks-list';
      body.append(listEl);
      render();
      return listEl;
    },
    onKey: function (event) {
      if (event.key === 'j' || event.key === 'ArrowDown') {
        selected = Math.min(state.items.length - 1, selected + 1);
        render();
        return true;
      }
      if (event.key === 'k' || event.key === 'ArrowUp') {
        selected = Math.max(0, selected - 1);
        render();
        return true;
      }
      if (event.key === 'Enter' || event.key === 'Delete' || event.key === 'Backspace') {
        removeAt(selected);
        return true;
      }
      return false;
    },
  });
  return entry;
}

/* ---------- 节点选择 ---------- */

export function openNodePicker(nodes, onPick, options) {
  const settings = options || {};
  const loggedIn = settings.loggedIn === true;
  let filtered = nodes.slice();
  let selected = 0;
  let listEl = null;
  let inputEl = null;
  let entryRef = null;

  const render = function () {
    listEl.textContent = '';
    if (!filtered.length) {
      const empty = document.createElement('p');
      empty.className = 'muted';
      empty.textContent = '没有匹配的节点';
      listEl.append(empty);
      return;
    }
    selected = Math.min(selected, filtered.length - 1);
    filtered.slice(0, 120).forEach(function (node, idx) {
      const row = document.createElement('li');
      row.className = 'node-pick-row' + (idx === selected ? ' selected' : '');
      const name = document.createElement('span');
      name.className = 'np-name';
      name.textContent = node.name;
      const title = document.createElement('span');
      title.className = 'np-title';
      title.textContent = node.title || '';
      row.append(name, title);
      if (node.titleAlt) {
        const alt = document.createElement('span');
        alt.className = 'np-alt';
        alt.textContent = node.titleAlt;
        row.append(alt);
      }
      if (node.restricted) {
        const flag = document.createElement('span');
        flag.className = 'np-restricted' + (loggedIn ? '' : ' np-lock');
        flag.textContent = loggedIn ? '受限' : '锁';
        flag.title = loggedIn ? '限登录用户访问的节点' : '需要登录 V2EX 后可见（按 g l 配置 Cookie）';
        row.append(flag);
      }
      row.addEventListener('mouseenter', function () { selected = idx; render(); });
      row.addEventListener('click', function () {
        entryRef.close();
        onPick(node.name);
      });
      listEl.append(row);
    });
  };

  const entry = openOverlay({
    title: '节点目录',
    sub: 'Enter 打开 · Esc 关闭',
    className: 'wide',
    focus: 'input',
    build: function (body) {
      inputEl = document.createElement('input');
      inputEl.className = 'node-picker-input';
      inputEl.type = 'search';
      inputEl.placeholder = '搜索节点名或描述…';
      inputEl.autocomplete = 'off';
      inputEl.spellcheck = false;
      inputEl.addEventListener('input', function () {
        const query = inputEl.value.trim().toLowerCase();
        filtered = nodes.filter(function (node) {
          if (!query) return true;
          return node.name.toLowerCase().includes(query) ||
            (node.title || '').toLowerCase().includes(query) ||
            (node.titleAlt || '').toLowerCase().includes(query);
        });
        selected = 0;
        render();
      });
      listEl = document.createElement('ul');
      listEl.className = 'node-picker-list';
      body.append(inputEl, listEl);
      render();
      return inputEl;
    },
    onKey: function (event) {
      if (event.key === 'ArrowDown') { selected = Math.min(filtered.length - 1, selected + 1); render(); return true; }
      if (event.key === 'ArrowUp') { selected = Math.max(0, selected - 1); render(); return true; }
      if (event.key === 'Enter') {
        const node = filtered[selected];
        if (!node) return true;
        entry.close();
        onPick(node.name);
        return true;
      }
      return false;
    },
  });
  entryRef = entry;
  return entry;
}
/* ---------- 常用节点管理 ---------- */

const NODE_MANAGER_MAX_ROWS = 400;

function kbdHint(keys, label) {
  const span = document.createElement('span');
  const kbd = document.createElement('kbd');
  kbd.textContent = keys;
  span.append(kbd, ' ', label);
  return span;
}

// 侧栏「常用节点」管理器：搜索即时过滤、Space/Enter 切换、Ctrl-S 保存、Esc 取消。
export function openNodeManager(options) {
  const settings = options || {};
  const nodes = (settings.nodes || []).slice();
  const backendReady = settings.backendReady !== false;
  const opened = (settings.followed || []).slice();
  let saved = opened.slice();
  let picked = new Set(opened);
  let query = '';
  let onlyPicked = false;
  let cursor = 0;
  let visible = [];
  let rowEls = [];
  let dirty = false;
  let saving = false;
  let gPending = false;
  let searchEl = null;
  let listEl = null;
  let countEl = null;
  let feedbackEl = null;
  let filterBtn = null;
  let clearBtn = null;
  let revertBtn = null;
  let saveBtn = null;

  const lower = function (value) { return String(value || '').toLowerCase(); };

  const isDirty = function () {
    if (picked.size !== saved.length) return true;
    for (const name of saved) if (!picked.has(name)) return true;
    return false;
  };

  const collect = function () {
    const out = [];
    for (const name of opened) if (picked.has(name)) out.push(name);
    for (const node of nodes) if (picked.has(node.name) && out.indexOf(node.name) < 0) out.push(node.name);
    for (const name of picked) if (out.indexOf(name) < 0) out.push(name);
    return out;
  };

  const setFeedback = function (text, type) {
    if (!feedbackEl) return;
    feedbackEl.textContent = text || '';
    feedbackEl.className = 'nm-feedback' + (type ? ' ' + type : '');
  };

  const updateStatus = function () {
    dirty = isDirty();
    if (countEl) {
      countEl.textContent = '已选 ' + picked.size + ' 个' + (dirty ? ' · 未保存' : '');
      countEl.classList.toggle('dirty', dirty);
    }
    if (filterBtn) filterBtn.textContent = '只看已选' + (picked.size ? ' (' + picked.size + ')' : '');
    if (saveBtn) saveBtn.disabled = saving || !dirty || !backendReady;
    if (clearBtn) clearBtn.disabled = saving || picked.size === 0;
    if (revertBtn) revertBtn.disabled = saving || !dirty;
  };

  // 已保存（打开时已勾选）的节点置顶，组内保持节点目录顺序；其余按目录顺序跟随
  const visibleNodes = function () {
    const needle = lower(query);
    const pinned = new Set(saved);
    const ordered = [];
    for (const node of nodes) if (pinned.has(node.name)) ordered.push(node);
    for (const node of nodes) if (!pinned.has(node.name)) ordered.push(node);
    return ordered.filter(function (node) {
      if (onlyPicked && !picked.has(node.name)) return false;
      if (needle && (lower(node.name) + ' ' + lower(node.title) + ' ' + lower(node.titleAlt)).indexOf(needle) < 0) return false;
      return true;
    });
  };

  // 只按需滚动最小距离：选中行在列表下方时对齐到列表底边，在上方时对齐到顶边
  const renderSelection = function () {
    rowEls.forEach(function (row, index) { row.classList.toggle('selected', index === cursor); });
    const current = rowEls[cursor];
    if (!current || !listEl) return;
    const rowRect = current.getBoundingClientRect();
    const listRect = listEl.getBoundingClientRect();
    if (rowRect.top < listRect.top) listEl.scrollTop -= listRect.top - rowRect.top;
    else if (rowRect.bottom > listRect.bottom) listEl.scrollTop += rowRect.bottom - listRect.bottom;
  };

  const renderList = function (options) {
    const opts = options || {};
    const anchorName = opts.keepName && visible[cursor] ? visible[cursor].name : '';
    visible = visibleNodes();
    const lastIndex = Math.max(0, Math.min(visible.length, NODE_MANAGER_MAX_ROWS) - 1);
    if (anchorName) {
      const anchorIndex = visible.findIndex(function (node) { return node.name === anchorName; });
      cursor = anchorIndex >= 0 ? Math.min(anchorIndex, lastIndex) : Math.min(cursor, lastIndex);
    } else {
      cursor = Math.min(cursor, lastIndex);
    }
    listEl.textContent = '';
    rowEls = [];
    if (!visible.length) {
      const empty = document.createElement('li');
      empty.className = 'nm-empty';
      empty.textContent = onlyPicked
        ? (picked.size ? '没有匹配的已选节点' : '还没有选中任何节点')
        : '没有匹配的节点';
      listEl.append(empty);
      return;
    }
    const limit = Math.min(visible.length, NODE_MANAGER_MAX_ROWS);
    for (let index = 0; index < limit; index += 1) {
      const node = visible[index];
      const on = picked.has(node.name);
      const row = document.createElement('li');
      row.className = 'nm-row' + (index === cursor ? ' selected' : '') + (on ? ' on' : '');
      const mark = document.createElement('span');
      mark.className = 'nm-mark';
      mark.textContent = on ? '✓' : '·';
      const name = document.createElement('span');
      name.className = 'nm-name';
      name.textContent = node.name;
      const title = document.createElement('span');
      title.className = 'nm-title';
      title.textContent = node.title || '';
      row.append(mark, name, title);
      if (node.titleAlt) {
        const alt = document.createElement('span');
        alt.className = 'nm-alt';
        alt.textContent = node.titleAlt;
        row.append(alt);
      }
      if (node.restricted) {
        const flag = document.createElement('span');
        flag.className = 'nm-restricted';
        flag.textContent = '锁';
        flag.title = '受限节点，需要登录 V2EX 后可见';
        row.append(flag);
      }
      row.addEventListener('click', function () { toggleIndex(index); });
      listEl.append(row);
      rowEls.push(row);
    }
    if (visible.length > limit) {
      const tail = document.createElement('li');
      tail.className = 'nm-empty';
      tail.textContent = '还有 ' + (visible.length - limit) + ' 个节点，继续输入缩小范围';
      listEl.append(tail);
    }
    renderSelection();
  };

  const toggleIndex = function (index) {
    const node = visible[index];
    if (!node || saving) return;
    if (picked.has(node.name)) picked.delete(node.name);
    else picked.add(node.name);
    const row = rowEls[index];
    if (row) {
      const on = picked.has(node.name);
      row.classList.toggle('on', on);
      const mark = row.querySelector('.nm-mark');
      if (mark) mark.textContent = on ? '✓' : '·';
    }
    updateStatus();
    if (onlyPicked && !picked.has(node.name)) renderList();
  };

  const shownCount = function () { return Math.min(visible.length, NODE_MANAGER_MAX_ROWS); };

  const move = function (delta) {
    if (!visible.length) return;
    cursor = Math.max(0, Math.min(shownCount() - 1, cursor + delta));
    renderSelection();
  };

  const revert = function () {
    if (saving || !dirty) return;
    picked = new Set(saved);
    cursor = 0;
    renderList();
    updateStatus();
    setFeedback('已还原到上次保存的列表。');
  };

  const clear = function () {
    if (saving || !picked.size) return;
    picked = new Set();
    cursor = 0;
    renderList();
    updateStatus();
    setFeedback('已清空选择，Ctrl-S 保存后生效。');
  };

  const save = async function () {
    if (saving || !backendReady) return;
    if (!dirty) {
      setFeedback('没有修改需要保存。');
      return;
    }
    saving = true;
    updateStatus();
    setFeedback('保存中…');
    try {
      const result = await settings.onSave(collect());
      const followed = result && Array.isArray(result.followedNodes) ? result.followedNodes : collect();
      saved = followed.slice();
      picked = new Set(saved);
      setFeedback('已保存，侧栏已更新（' + saved.length + ' 个）。', 'ok');
    } catch (error) {
      setFeedback('保存失败：' + ((error && error.message) || '未知错误') + '，按 Ctrl-S 重试。', 'error');
    } finally {
      saving = false;
      updateStatus();
      renderList({ keepName: true });
    }
  };

  const makeButton = function (label, title, onClick, className) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = className || 'nm-btn';
    button.textContent = label;
    button.title = title || '';
    button.addEventListener('click', onClick);
    return button;
  };

  const focusList = function () {
    if (listEl) listEl.focus({ preventScroll: true });
  };

  const onSearch = function () {
    query = searchEl.value.trim();
    cursor = 0;
    renderList();
  };

  const toggleOnlyPicked = function () {
    onlyPicked = !onlyPicked;
    filterBtn.classList.toggle('active', onlyPicked);
    cursor = 0;
    renderList();
  };

  const entry = openOverlay({
    title: '常用节点',
    sub: 'j/k 移动 · Space/Enter 选中 · / 搜索 · Ctrl-S 保存 · Esc 取消',
    className: 'wide node-manager',
    focus: '.nm-list',
    documentKey: true,
    build: function (body) {
      const toolbar = document.createElement('div');
      toolbar.className = 'nm-toolbar';
      searchEl = document.createElement('input');
      searchEl.className = 'nm-search';
      searchEl.type = 'search';
      searchEl.placeholder = '搜索节点名 / 描述，输入即时过滤';
      searchEl.autocomplete = 'off';
      searchEl.spellcheck = false;
      searchEl.setAttribute('aria-label', '搜索节点');
      searchEl.addEventListener('input', onSearch);
      filterBtn = makeButton('只看已选', '只显示已选中的节点', toggleOnlyPicked, 'nm-filter-btn');
      countEl = document.createElement('span');
      countEl.className = 'nm-count';
      const actions = document.createElement('div');
      actions.className = 'nm-actions';
      clearBtn = makeButton('清空', '取消全部选中，Ctrl-S 保存后生效', clear);
      revertBtn = makeButton('还原', '还原到上次保存的列表', revert);
      saveBtn = makeButton('保存', '保存并刷新侧栏（Ctrl-S）', save, 'nm-btn primary');
      actions.append(clearBtn, revertBtn, saveBtn);
      toolbar.append(searchEl, filterBtn, countEl, actions);

      listEl = document.createElement('ul');
      listEl.className = 'nm-list';
      listEl.tabIndex = 0;
      listEl.setAttribute('aria-label', '节点列表');

      feedbackEl = document.createElement('div');
      feedbackEl.className = 'nm-feedback';
      feedbackEl.setAttribute('role', 'status');
      feedbackEl.setAttribute('aria-live', 'polite');

      const foot = document.createElement('div');
      foot.className = 'nm-foot';
      foot.append(
        kbdHint('↓↑ / j k', '移动'),
        kbdHint('Space / Enter', '选中/取消'),
        kbdHint('gg / G', '首/尾'),
        kbdHint('Tab', '切换搜索/列表'),
        kbdHint('Ctrl-S', '保存'),
        kbdHint('Esc', '取消')
      );

      body.append(toolbar, listEl, feedbackEl, foot);
      if (!backendReady) setFeedback('后端节点接口未就绪（404），改动无法保存。', 'error');
      renderList();
      updateStatus();
      return listEl;
    },
    onTab: function (event) {
      if (event.shiftKey) return false;
      if (event.target === searchEl) {
        listEl.focus({ preventScroll: true });
        return true;
      }
      if (event.target === listEl) {
        searchEl.focus({ preventScroll: true });
        return true;
      }
      return false;
    },
    onKey: function (event) {
      const inSearch = event.target === searchEl;
      if ((event.ctrlKey || event.metaKey) && (event.key === 's' || event.key === 'S')) {
        save();
        return true;
      }
      if (event.key === 'ArrowDown' || (!inSearch && event.key === 'j')) {
        gPending = false;
        move(1);
        return true;
      }
      if (event.key === 'ArrowUp' || (!inSearch && event.key === 'k')) {
        gPending = false;
        move(-1);
        return true;
      }
      if (!inSearch && event.key === 'g') {
        if (gPending) {
          gPending = false;
          cursor = 0;
          renderSelection();
        } else {
          gPending = true;
          setTimeout(function () { gPending = false; }, 700);
        }
        return true;
      }
      if (!inSearch && event.key === 'G') {
        gPending = false;
        cursor = Math.max(0, shownCount() - 1);
        renderSelection();
        return true;
      }
      if (!inSearch && event.key === ' ') {
        toggleIndex(cursor);
        return true;
      }
      if (event.key === 'Enter') {
        if (inSearch) {
          focusList();
          return true;
        }
        toggleIndex(cursor);
        return true;
      }
      if (!inSearch && event.key === '/') {
        searchEl.focus({ preventScroll: true });
        searchEl.select();
        return true;
      }
      return false;
    },
    onEscape: function () {
      if (searchEl && document.activeElement === searchEl) {
        focusList();
        return true;
      }
      return false;
    },
  });
  return entry;
}

/* ---------- 登录 ---------- */

function normalizeLoginState(source) {
  const login = source && typeof source === 'object' ? source : {};
  return {
    hasCookie: login.hasCookie === true,
    loggedIn: login.loggedIn === true,
    user: typeof login.user === 'string' ? login.user : '',
    checkedAt: Number(login.checkedAt) || 0,
  };
}

export function openLogin(options) {
  const settings = options || {};
  let current = normalizeLoginState(settings.login);
  let textareaEl = null;
  let feedbackEl = null;
  let statusValueEl = null;
  let submitBtn = null;
  let logoutBtn = null;
  let chromeBtn = null;
  let busy = false;

  const readCurrent = function () {
    if (typeof settings.getLogin === 'function') return normalizeLoginState(settings.getLogin());
    return current;
  };

  const setFeedback = function (text, type) {
    if (!feedbackEl) return;
    feedbackEl.textContent = text || '';
    feedbackEl.className = 'login-feedback' + (type ? ' ' + type : '');
  };

  const refresh = function () {
    const login = normalizeLoginState(current);
    if (statusValueEl) {
      statusValueEl.textContent = login.loggedIn
        ? '已登录' + (login.user ? ' ' + login.user : '')
        : (login.hasCookie ? '已保存 Cookie（未验证）' : '未登录');
      statusValueEl.className = 'ls-value' + (login.loggedIn ? ' on' : '');
    }
    if (logoutBtn) logoutBtn.classList.toggle('hidden', !(login.hasCookie || login.loggedIn));
    if (submitBtn) submitBtn.textContent = login.loggedIn ? '更新 Cookie' : '保存并验证';
  };

  const submit = async function () {
    if (!textareaEl || !submitBtn || submitBtn.disabled) return;
    const cookie = textareaEl.value.trim();
    if (!cookie) {
      setFeedback('请先粘贴 v2ex.com 的 Cookie 字符串', 'error');
      textareaEl.focus();
      return;
    }
    submitBtn.disabled = true;
    setFeedback('验证中…');
    try {
      const result = await settings.onSubmit(cookie);
      current = normalizeLoginState(result && typeof result === 'object' ? result : readCurrent());
      refresh();
      textareaEl.value = '';
      setFeedback('登录成功，受限节点现在可以访问了。', 'ok');
    } catch (error) {
      setFeedback((error && error.message) || '登录失败，请检查 Cookie 是否完整', 'error');
    } finally {
      submitBtn.disabled = false;
    }
  };

  const logout = async function () {
    if (!logoutBtn || logoutBtn.disabled) return;
    logoutBtn.disabled = true;
    setFeedback('正在退出…');
    try {
      const result = await settings.onLogout();
      current = normalizeLoginState(result && typeof result === 'object' ? result : readCurrent());
      refresh();
      setFeedback('已退出登录，本地 Cookie 已清除。', 'ok');
    } catch (error) {
      setFeedback((error && error.message) || '退出失败', 'error');
    } finally {
      logoutBtn.disabled = false;
    }
  };

  const setBusy = function (value) {
    busy = value;
    if (submitBtn) submitBtn.disabled = value;
    if (chromeBtn) chromeBtn.disabled = value;
    if (logoutBtn) logoutBtn.disabled = value;
    if (chromeBtn) chromeBtn.textContent = value ? '正在连接 Chrome…' : '从 Chrome 导入';
  };

  const chromeHint = function (error) {
    const code = error && error.code;
    if (code === 'chrome_unavailable') return '（确认 Chrome 已开启远程调试并已登录 v2ex.com；弹出「允许调试」时点 Allow）';
    if (code === 'chrome_unsupported' || (error && (error.status === 404 || error.status === 405))) return '（后端尚未支持，稍后重启新版）';
    if (code === 'login_failed') return '（Chrome 可能还没有登录 v2ex.com）';
    if (error && error.status === 403) return '（只允许本机请求）';
    return '';
  };

  const importChrome = async function () {
    if (busy) return;
    setBusy(true);
    setFeedback('正在连接 Chrome…如弹出「允许调试」请点 Allow（最长约 120 秒）');
    try {
      const result = await settings.onChromeImport();
      current = normalizeLoginState(result && typeof result === 'object' ? result : readCurrent());
      refresh();
      setFeedback('已从 Chrome 导入：' + (current.user || 'Cookie'), 'ok');
    } catch (error) {
      setFeedback(((error && error.message) || '从 Chrome 导入失败') + chromeHint(error), 'error');
    } finally {
      setBusy(false);
    }
  };

  const entry = openOverlay({
    title: '登录 V2EX',
    sub: 'Cookie 只保存在本地',
    className: 'narrow',
    build: function (body) {
      const status = document.createElement('div');
      status.className = 'login-status';
      const label = document.createElement('span');
      label.className = 'ls-label';
      label.textContent = '状态';
      statusValueEl = document.createElement('span');
      statusValueEl.className = 'ls-value';
      status.append(label, statusValueEl);

      const hint = document.createElement('p');
      hint.className = 'login-hint';
      hint.innerHTML = '在浏览器登录 <code>v2ex.com</code> 后，打开开发者工具 Console 执行 <code>copy(document.cookie)</code>，把整段 Cookie 粘贴到下方；也可以在 Application → Cookies 中复制。本机 Chrome 已登录时，可直接点下方「从 Chrome 导入」。';

      textareaEl = document.createElement('textarea');
      textareaEl.className = 'login-textarea';
      textareaEl.spellcheck = false;
      textareaEl.placeholder = 'A2=...; PB3_SESSION=...; V2EX_LANG=zhcn';
      textareaEl.setAttribute('aria-label', 'V2EX Cookie');

      const actions = document.createElement('div');
      actions.className = 'login-actions';
      submitBtn = document.createElement('button');
      submitBtn.type = 'button';
      submitBtn.className = 'login-btn primary';
      submitBtn.textContent = '保存并验证';
      submitBtn.addEventListener('click', submit);
      chromeBtn = document.createElement('button');
      chromeBtn.type = 'button';
      chromeBtn.className = 'login-btn';
      chromeBtn.textContent = '从 Chrome 导入';
      chromeBtn.title = '调用后端连接本机 Chrome DevTools 读取 v2ex.com Cookie，可能需要点 Allow';
      chromeBtn.addEventListener('click', importChrome);
      logoutBtn = document.createElement('button');
      logoutBtn.type = 'button';
      logoutBtn.className = 'login-btn danger';
      logoutBtn.textContent = '退出登录';
      logoutBtn.addEventListener('click', logout);
      const spacer = document.createElement('span');
      spacer.className = 'spacer';
      const closeBtn = document.createElement('button');
      closeBtn.type = 'button';
      closeBtn.className = 'login-btn';
      closeBtn.textContent = '关闭';
      closeBtn.addEventListener('click', function () { entry.close(); });
      actions.append(submitBtn, chromeBtn, logoutBtn, spacer, closeBtn);

      feedbackEl = document.createElement('div');
      feedbackEl.className = 'login-feedback';
      feedbackEl.setAttribute('role', 'status');
      feedbackEl.setAttribute('aria-live', 'polite');

      const security = document.createElement('p');
      security.className = 'login-security';
      security.textContent = '安全提示：Cookie 只保存在本地服务的数据库/配置里，仅用于抓取你已登录可见的页面，不会上传到任何第三方。退出登录会立即清除。';

      body.append(status, hint, textareaEl, actions, feedbackEl, security);
      refresh();
      return textareaEl;
    },
    onKey: function (event) {
      if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') {
        submit();
        return true;
      }
      return false;
    },
  });
  return entry;
}

