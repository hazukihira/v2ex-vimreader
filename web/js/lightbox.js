// 图片灯箱、懒加载、点击后才加载的视频嵌入。

let box = null;
let imageEl = null;
let captionEl = null;
let countEl = null;
let origEl = null;
let images = [];
let index = 0;
let isOpen = false;
let lastFocus = null;

function safeUrl(raw, base) {
  try {
    return new URL(raw, base || location.href);
  } catch (e) {
    return null;
  }
}

function parseYouTube(raw) {
  const url = safeUrl(raw);
  if (!url) return null;
  const host = url.hostname.replace(/^www\./, '').replace(/^m\./, '');
  if (host === 'youtube.com' || host === 'youtube-nocookie.com') {
    let id = url.searchParams.get('v');
    if (!id) {
      const match = url.pathname.match(/^\/(?:embed|shorts|live|v)\/([\w-]{6,})/);
      if (match) id = match[1];
    }
    if(safeId(id)) return {
      site: 'YouTube',
      label: 'YouTube 视频',
      embed: 'https://www.youtube.com/embed/' + id,
      href: 'https://www.youtube.com/watch?v=' + id,
    };
    return null;
  }
  if (host === 'youtu.be') {
    const id = url.pathname.slice(1).split('/')[0];
    if (safeId(id)) return {
      site: 'YouTube',
      label: 'YouTube 视频',
      embed: 'https://www.youtube.com/embed/' + id,
      href: 'https://www.youtube.com/watch?v=' + id,
    };
  }
  return null;
}

function safeId(id) {
  return typeof id === 'string' && /^[\w-]{6,}$/.test(id);
}

function parseBilibili(raw) {
  const url = safeUrl(raw);
  if (!url) return null;
  const host = url.hostname.replace(/^www\./, '');
  if (host === 'bilibili.com' || host === 'm.bilibili.com') {
    const match = url.pathname.match(/\/video\/(BV[\w]+)/);
    if (match) {
      return {
        site: '哔哩哔哩',
        label: 'B 站视频 · ' + match[1],
        embed: 'https://player.bilibili.com/player.html?bvid=' + match[1] + '&autoplay=0&high_quality=1',
        href: url.href,
      };
    }
  }
  if (host === 'b23.tv') {
    return { site: '哔哩哔哩', label: 'B 站视频（短链接）', embed: '', href: url.href };
  }
  return null;
}

function videoInfo(raw) {
  return parseYouTube(raw) || parseBilibili(raw);
}

function makeVideoCard(info) {
  const card = document.createElement('div');
  card.className = 'video-card';
  card.setAttribute('role', 'button');
  card.tabIndex = 0;
  const play = document.createElement('span');
  play.className = 'vc-play';
  play.textContent = '▶';
  const body = document.createElement('span');
  body.className = 'vc-body';
  const title = document.createElement('b');
  title.textContent = info.site + ' 视频';
  const hint = document.createElement('span');
  hint.textContent = info.embed ? '点击加载播放器' : '点击打开原站观看';
  body.append(title, hint);
  const ext = document.createElement('a');
  ext.className = 'vc-ext';
  ext.href = info.href;
  ext.target = '_blank';
  ext.rel = 'noopener noreferrer';
  ext.textContent = '↗';
  ext.title = '在新标签页打开';
  ext.addEventListener('click', function (event) { event.stopPropagation(); });
  card.append(play, body, ext);

  const activate = function () {
    if (info.embed) {
      const frame = document.createElement('iframe');
      frame.className = 'video-embed';
      frame.src = info.embed;
      frame.title = info.site + ' 播放器';
      frame.allow = 'accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; fullscreen';
      frame.allowFullscreen = true;
      frame.referrerPolicy = 'no-referrer';
      card.replaceWith(frame);
    } else {
      window.open(info.href, '_blank', 'noopener,noreferrer');
    }
  };
  card.addEventListener('click', function (event) {
    if (event.target.closest('.vc-ext')) return;
    activate();
  });
  card.addEventListener('keydown', function (event) {
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      activate();
    }
  });
  return card;
}

function replaceIframe(iframe) {
  const src = iframe.getAttribute('src') || '';
  const info = videoInfo(src);
  const replacement = info
    ? makeVideoCard(info)
    : makeVideoCard({ site: '嵌入内容', label: '嵌入内容', embed: '', href: src || '#' });
  iframe.replaceWith(replacement);
}

function rewriteLinks(root) {
  const anchors = root.querySelectorAll('a[href]');
  for (const anchor of anchors) {
    anchor.target = '_blank';
    anchor.rel = 'noopener noreferrer';
    const info = videoInfo(anchor.getAttribute('href'));
    if (info) {
      const card = makeVideoCard(info);
      anchor.replaceWith(card);
    }
  }
}

export function decorateContent(root) {
  if (!root) return;
  root.setAttribute('data-content', '1');
  for (const img of root.querySelectorAll('img')) {
    img.loading = 'lazy';
    img.decoding = 'async';
    img.referrerPolicy = 'no-referrer';
    img.classList.add('content-img');
    if (!img.alt) img.alt = '帖子图片';
    img.title = '点击查看大图';
  }
  for (const iframe of Array.from(root.querySelectorAll('iframe'))) replaceIframe(iframe);
  rewriteLinks(root);
}

function galleryFor(img) {
  const holder = img.closest('[data-content]');
  const nodes = holder ? Array.from(holder.querySelectorAll('img.content-img')) : [img];
  return nodes.map(function (node) {
    return { src: node.currentSrc || node.src, alt: node.alt || '帖子图片' };
  });
}

export function initLightbox() {
  box = document.getElementById('lightbox');
  imageEl = document.getElementById('lb-img');
  captionEl = document.getElementById('lb-caption');
  countEl = document.getElementById('lb-count');
  origEl = document.getElementById('lb-orig');

  document.getElementById('lb-close').addEventListener('click', closeLightbox);
  document.getElementById('lb-prev').addEventListener('click', function () { step(-1); });
  document.getElementById('lb-next').addEventListener('click', function () { step(1); });
  imageEl.addEventListener('click', function (event) {
    if (images.length < 2) return;
    const rect = imageEl.getBoundingClientRect();
    step(event.clientX < rect.left + rect.width / 2 ? -1 : 1);
  });

  document.addEventListener('click', function (event) {
    const img = event.target.closest ? event.target.closest('img.content-img') : null;
    if (!img) return;
    event.preventDefault();
    event.stopPropagation();
    openLightbox(galleryFor(img), Math.max(0, galleryFor(img).findIndex(function (item) {
      return item.src === (img.currentSrc || img.src);
    })));
  });

  document.addEventListener('keydown', function (event) {
    if (!isOpen) return;
    if (event.key === 'Escape') {
      event.preventDefault();
      event.stopPropagation();
      closeLightbox();
    } else if (event.key === 'ArrowLeft') {
      event.preventDefault();
      step(-1);
    } else if (event.key === 'ArrowRight') {
      event.preventDefault();
      step(1);
    }
  }, true);
}

export function isLightboxOpen() {
  return isOpen;
}

export function openLightbox(list, startIndex) {
  if (!box || !list || !list.length) return;
  images = list.slice();
  index = Math.min(Math.max(0, startIndex || 0), images.length - 1);
  lastFocus = document.activeElement;
  isOpen = true;
  box.classList.remove('hidden');
  box.setAttribute('aria-hidden', 'false');
  render();
  document.getElementById('lb-close').focus({ preventScroll: true });
}

export function closeLightbox() {
  if (!isOpen) return;
  isOpen = false;
  box.classList.add('hidden');
  box.setAttribute('aria-hidden', 'true');
  imageEl.removeAttribute('src');
  if (lastFocus && lastFocus.focus) lastFocus.focus({ preventScroll: true });
  lastFocus = null;
}

function step(delta) {
  if (!images.length) return;
  index = (index + delta + images.length) % images.length;
  render();
}

function render() {
  const item = images[index];
  if (!item) return;
  imageEl.src = item.src;
  imageEl.alt = item.alt || '';
  captionEl.textContent = item.alt || '';
  countEl.textContent = (index + 1) + ' / ' + images.length;
  origEl.href = item.src;
  origEl.textContent = item.src.startsWith('data:') ? '打开原图' : '打开原图 ↗';
  const multiple = images.length > 1;
  document.getElementById('lb-prev').classList.toggle('hidden', !multiple);
  document.getElementById('lb-next').classList.toggle('hidden', !multiple);
  countEl.classList.toggle('hidden', !multiple);
}

export function currentGallery() {
  return images.slice();
}

export function galleryIndexOf(url) {
  return images.findIndex(function (item) { return item.src === url; });
}

export function openGalleryAt(url) {
  const idx = galleryIndexOf(url);
  if (idx >= 0) {
    imageEl.src = url;
    index = idx;
    render();
  }
}
