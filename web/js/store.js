// localStorage 持久化：已读、主题、侧栏宽度、列表位置与过滤词。

const PREFIX = 'v2r:';
const READ_LIMIT = 6000;

function readRaw(key, fallback) {
  try {
    const raw = localStorage.getItem(PREFIX + key);
    if (raw == null) return fallback;
    return JSON.parse(raw);
  } catch (e) {
    return fallback;
  }
}

function writeRaw(key, value) {
  try {
    localStorage.setItem(PREFIX + key, JSON.stringify(value));
  } catch (e) {
    /* 存储满了或隐私模式，忽略 */
  }
}

export const store = {
  theme() {
    const value = readRaw('theme', 'dark');
    return value === 'light' ? 'light' : 'dark';
  },

  setTheme(value) {
    writeRaw('theme', value === 'light' ? 'light' : 'dark');
  },

  readSet() {
    const obj = readRaw('read', {});
    return new Set(Object.keys(obj || {}));
  },

  setRead(id, on) {
    const obj = readRaw('read', {}) || {};
    if (on) {
      obj[String(id)] = Math.floor(Date.now() / 1000);
      const keys = Object.keys(obj);
      if (keys.length > READ_LIMIT) {
        keys.sort((a, b) => (obj[a] || 0) - (obj[b] || 0));
        for (const key of keys.slice(0, keys.length - READ_LIMIT)) delete obj[key];
      }
    } else {
      delete obj[String(id)];
    }
    writeRaw('read', obj);
  },

  position(key) {
    const value = readRaw('pos:' + key, null);
    if (!value || typeof value !== 'object') return null;
    const index = Number.isFinite(value.i) ? value.i : -1;
    const scroll = Number.isFinite(value.s) ? Math.max(0, value.s) : 0;
    return { index, scroll };
  },

  savePosition(key, index, scroll) {
    writeRaw('pos:' + key, { i: index, s: Math.max(0, Math.round(scroll || 0)) });
  },

  filter(key) {
    const value = readRaw('filter:' + key, '');
    return typeof value === 'string' ? value : '';
  },

  saveFilter(key, term) {
    if (term) writeRaw('filter:' + key, term);
    else {
      try {
        localStorage.removeItem(PREFIX + 'filter:' + key);
      } catch (e) {}
    }
  },

  sidebarWidth() {
    const value = readRaw('sidebarWidth', 220);
    return Number.isFinite(value) ? Math.min(340, Math.max(180, value)) : 220;
  },

  saveSidebarWidth(px) {
    writeRaw('sidebarWidth', Math.round(px));
  },

  sidebarCollapsed() {
    return readRaw('sidebarCollapsed', false) === true;
  },

  saveSidebarCollapsed(collapsed) {
    writeRaw('sidebarCollapsed', collapsed === true);
  },

  clearCache() {
    const keepKeys = [PREFIX + 'theme', PREFIX + 'favorites', PREFIX + 'queue'];
    const keep = {};
    try {
      for (const key of keepKeys) {
        const value = localStorage.getItem(key);
        if (value != null) keep[key] = value;
      }
      const keys = [];
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key && key.startsWith(PREFIX) && !keepKeys.includes(key)) keys.push(key);
      }
      for (const key of keys) localStorage.removeItem(key);
      for (const key of keepKeys) {
        if (keep[key] != null) localStorage.setItem(key, keep[key]);
      }
    } catch (e) {}
  },
};
