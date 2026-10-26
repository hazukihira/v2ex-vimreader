// 按键序列引擎：支持数字前缀（5j）、多键序列（gg）、Ctrl 组合与超时重置。

const SEQUENCE_TIMEOUT = 900;

function normalize(event) {
  const key = event.key;
  if (!key) return null;
  if (event.metaKey || event.altKey) return null;
  if (event.ctrlKey && key.length === 1) return 'ctrl+' + key.toLowerCase();
  switch (key) {
    case 'Escape': return 'esc';
    case 'Enter': return 'enter';
    case ' ': return 'space';
    case 'Tab': return 'tab';
    case 'Backspace': return 'backspace';
    case 'Delete': return 'delete';
    case 'ArrowDown': return 'down';
    case 'ArrowUp': return 'up';
    case 'ArrowLeft': return 'left';
    case 'ArrowRight': return 'right';
    case 'PageDown': return 'pagedown';
    case 'PageUp': return 'pageup';
    case 'Home': return 'home-key';
    case 'End': return 'end-key';
    default: break;
  }
  return key.length === 1 ? key : null;
}

export class KeyEngine {
  constructor() {
    this.map = new Map();
    this.prefix = '';
    this.countText = '';
    this.timer = 0;
  }

  setBindings(bindings) {
    this.map = new Map(Object.entries(bindings || {}));
    this.reset();
  }

  hasBinding(sequence) {
    return this.map.has(sequence);
  }

  reset() {
    this.prefix = '';
    this.countText = '';
    if (this.timer) {
      clearTimeout(this.timer);
      this.timer = 0;
    }
  }

  armTimeout() {
    if (this.timer) clearTimeout(this.timer);
    this.timer = setTimeout(() => {
      this.prefix = '';
      this.countText = '';
      this.timer = 0;
    }, SEQUENCE_TIMEOUT);
  }

  handle(event) {
    const token = normalize(event);
    if (!token) return false;

    if (!this.prefix) {
      if (token === '0' && this.countText) {
        this.countText += token;
        this.armTimeout();
        return true;
      }
      if (/^[1-9]$/.test(token)) {
        this.countText += token;
        this.armTimeout();
        return true;
      }
    }

    const sequence = this.prefix ? this.prefix + ' ' + token : token;
    if (this.map.has(sequence)) {
      const handler = this.map.get(sequence);
      const count = parseInt(this.countText || '1', 10) || 1;
      this.reset();
      handler(count, sequence);
      return true;
    }

    let isPrefix = false;
    for (const binding of this.map.keys()) {
      if (binding.startsWith(sequence + ' ')) {
        isPrefix = true;
        break;
      }
    }
    if (isPrefix) {
      this.prefix = sequence;
      this.armTimeout();
      return true;
    }

    const hadPrefix = Boolean(this.prefix);
    this.reset();
    if (hadPrefix && this.map.has(token)) {
      this.map.get(token)(1, token);
      return true;
    }
    return false;
  }
}
