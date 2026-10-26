// ?mock=1 使用的离线数据，覆盖列表、受限节点、图片/视频正文、AI 总结流式与屏蔽过滤。

const REPLY_AUTHORS = [
  'Livid', 'sayo', 'est', 'binux', 'lepture', 'cloverstd', 'shoaly', 'bytewind',
  'spencer', 'stardust', 'dian', 'chengfu', 'xxl', 'yfchen', 'dreamer', 'huangj',
  'kalman', 'moonlight', 'nerd', 'ocean', 'spamking',
];

const NODES = [
  { name: 'tech', title: '科技', titleAlt: 'Technology', parent: '', topicsCount: 596, restricted: false },
  { name: 'qna', title: '问与答', titleAlt: 'Questions and Answers', parent: '', topicsCount: 1204, restricted: false },
  { name: 'programmer', title: '程序员', titleAlt: 'Programmer', parent: 'tech', topicsCount: 842, restricted: false },
  { name: 'create', title: '分享创造', titleAlt: 'Create', parent: '', topicsCount: 377, restricted: false },
  { name: 'ideas', title: '奇思妙想', titleAlt: 'Ideas', parent: '', topicsCount: 128, restricted: false },
  { name: 'jobs', title: '酷工作', titleAlt: 'Jobs', parent: '', topicsCount: 934, restricted: false },
  { name: 'apple', title: 'Apple', titleAlt: '', parent: '', topicsCount: 512, restricted: false },
  { name: 'android', title: 'Android', titleAlt: '', parent: '', topicsCount: 288, restricted: false },
  { name: 'ios', title: 'iOS', titleAlt: '', parent: '', topicsCount: 264, restricted: false },
  { name: 'python', title: 'Python', titleAlt: '', parent: 'programmer', topicsCount: 318, restricted: false },
  { name: 'javascript', title: 'JavaScript', titleAlt: '', parent: 'programmer', topicsCount: 355, restricted: false },
  { name: 'golang', title: 'Go 编程语言', titleAlt: 'Golang', parent: 'programmer', topicsCount: 214, restricted: false },
  { name: 'linux', title: 'Linux', titleAlt: '', parent: 'tech', topicsCount: 402, restricted: false },
  { name: 'windows', title: 'Windows', titleAlt: '', parent: 'tech', topicsCount: 356, restricted: false },
  { name: 'hardware', title: '硬件', titleAlt: 'Hardware', parent: 'tech', topicsCount: 189, restricted: false },
  { name: 'design', title: '设计', titleAlt: 'Design', parent: '', topicsCount: 142, restricted: false },
  { name: 'share', title: '分享发现', titleAlt: 'Share', parent: '', topicsCount: 712, restricted: false },
  { name: 'deals', title: '优惠信息', titleAlt: 'Deals', parent: '', topicsCount: 486, restricted: false },
  { name: 'travel', title: '旅行', titleAlt: 'Travel', parent: '', topicsCount: 176, restricted: false },
  { name: 'food', title: '美食', titleAlt: 'Food', parent: '', topicsCount: 151, restricted: false },
  { name: 'movie', title: '电影', titleAlt: 'Movie', parent: '', topicsCount: 233, restricted: false },
  { name: 'music', title: '音乐', titleAlt: 'Music', parent: '', topicsCount: 168, restricted: false },
  { name: 'book', title: '书籍', titleAlt: 'Book', parent: '', topicsCount: 97, restricted: false },
  { name: 'play', title: '游戏', titleAlt: 'Play', parent: '', topicsCount: 201, restricted: false },
  { name: 'career', title: '职场话题', titleAlt: 'Career', parent: '', topicsCount: 425, restricted: false },
  { name: 'photography', title: '摄影', titleAlt: 'Photography', parent: '', topicsCount: 43, restricted: false },
  { name: 'nsfw', title: 'NSFW', titleAlt: '', parent: '', topicsCount: 0, restricted: true },
  { name: 'sandbox', title: '沙盒', titleAlt: 'Sandbox', parent: '', topicsCount: 0, restricted: true },
  { name: 'chamber', title: 'Chamber', titleAlt: '', parent: '', topicsCount: 0, restricted: true },
];

const NODE_TITLES = {};
for (const node of NODES) NODE_TITLES[node.name] = node.title;

const FOLLOWED = ['tech', 'qna', 'programmer', 'create', 'apple', 'share', 'jobs', 'deals', 'windows', 'linux', 'golang'];

const ENTRIES = [
  ['programmer', '整理了 12 个 Windows 下顺手的命令行工具，附配置文件', 'sayo', 132],
  ['tech', '换了 4K 显示器之后字太小，大家系统缩放都设多少？', 'binux', 26],
  ['qna', '有没有靠谱的本地电子书管理方案？Calibre 太重了', 'lepture', 8],
  ['windows', 'Windows 11 24H2 升级后输入法偶发卡顿，有一样的吗', 'spencer', 17],
  ['linux', 'Arch 滚挂了，有没有推荐的恢复流程', 'ocean', 12],
  ['tech', '实测 2.5G 网口的 NAS 在弱电箱里温度有点高', 'spamking', 3],
  ['apple', 'M4 Mac mini 做开发机三个月：散热、编译和外接屏', 'est', 44],
  ['share', '写了一个只做一件事的小工具：把剪贴板里的链接自动转 Markdown', 'dreamer', 21],
  ['golang', 'Go 1.27 的 range-over-func 在实际项目里的两种用法', 'kalman', 9],
  ['qna', '通勤 40 分钟，有什么办法把碎片时间用起来', 'moonlight', 33],
  ['create', '做了一个纯前端的 v2ex 阅读器，无框架无构建', 'sayo', 58],
  ['jobs', '远程岗位：找一位熟悉 Go 的后端，长期合作', 'nerd', 14],
  ['deals', '京东的机械键盘最近有好价吗，预算 400 以内', 'dian', 11],
  ['javascript', '原生 ES Modules 写到一定规模后，你们怎么组织目录', 'bytewind', 19],
  ['tech', '家用 NAS 备份方案：3-2-1 原则在小规模下的取舍', 'chengfu', 27],
  ['windows', 'PowerToys 里最常用的三个功能，说说你的', 'huangj', 16],
  ['programmer', '代码评审里最难说服人的是什么？我的经验是命名', 'yfchen', 41],
  ['apple', 'iOS 18 的专注模式自定义，思路分享一下', 'stardust', 7],
  ['qna', '想给父母换一台简单的笔记本，预算 3500，求推荐', 'cloverstd', 22],
  ['share', '把每天的屏幕时间导成 CSV，做了个可视化，看了一周有点震撼', 'spencer', 15],
  ['linux', 'systemd 的 timer 替代 crontab 之后，日志好查多了', 'ocean', 6],
  ['create', '用 SQLite 当本地缓存写爬虫，几个容易踩的坑', 'kalman', 13],
  ['tech', '路由器刷 OpenWrt 之后 IPv6 老掉线，怎么排查', 'binux', 10],
  ['qna', '午休在工位趴着睡还是去车里睡？', 'moonlight', 24],
  ['programmer', '你所在的团队还写单元测试吗，覆盖率大概多少', 'est', 36],
  ['jobs', '面试被问到「为什么想换工作」，怎么说才不算踩雷', 'huangj', 29],
  ['golang', 'modernc.org/sqlite 在 Windows 上的编译时间到底有多长', 'nerd', 5],
  ['apple', 'Mac 上替代 Homebrew 的方案有没有更省心的', 'lepture', 18],
  ['windows', '资源管理器右键菜单卡顿，清理无效项的一个办法', 'spamking', 2],
  ['share', '把自己三年的 RSS 阅读记录做成了年度报告', 'dreamer', 12],
  ['deals', '双十一前的显示器价格走势，整理了一张表', 'dian', 9],
  ['idea', '一个想法：把 V2EX 的收藏做成可搜索的本地知识库', 'sayo', 31],
  ['ideas', '如果让 AI 帮你读长帖，你最想要什么形式的摘要', 'moonlight', 20],
  ['tech', 'Thunderbolt 5 的扩展坞什么时候能降到合理价位', 'est', 8],
  ['programmer', '从 Java 转 Go 半年，最大的不适应是错误处理', 'yfchen', 23],
  ['qna', '有什么适合睡前读的纸质书，不要太烧脑', 'moonlight', 17],
  ['apple', 'AirPods Pro 的降噪在地铁里好像变弱了', 'stardust', 11],
  ['share', '自建了一个只给自己用的搜索页，聚合了 8 个来源', 'dreamer', 26],
  ['jobs', '四线城市远程岗，薪资按一线打八折能接受吗', 'nerd', 34],
  ['create', '做了个极简的番茄钟，单文件 HTML，没有依赖', 'bytewind', 14],
  ['linux', '固态硬盘健康度 87%，还能当系统盘用多久', 'ocean', 7],
  ['windows', 'Windows 上有没有轻量的截图+标注工具推荐', 'huangj', 25],
  ['tech', '自建 DNS 之后广告少了，但有些站打不开了', 'binux', 13],
  ['qna', '第一次带团队，怎么处理技术分歧', 'chengfu', 39],
  ['travel', '十一后错峰去云南，路线和住宿记录', 'lepture', 28],
  ['food', '在家复刻兰州牛肉面，汤底的几个关键点', 'moonlight', 16],
  ['movie', '今年到目前为止最喜欢的一部电影，说说理由', 'spencer', 21],
  ['music', '写代码时的歌单，纯音乐还是有人声', 'dian', 12],
  ['book', '最近读完的三本书，一本推荐两本劝退', 'stardust', 19],
  ['play', 'Steam  Deck 上适合通勤玩的游戏', 'bytewind', 9],
  ['career', '工作五年后，如何判断该不该转管理', 'yfchen', 32],
  ['photography', '手机摄影的构图习惯，你们有刻意练习吗', 'dreamer', 6],
  ['idea', '想法：把 RSS 和本地归档做成同一条时间线', 'kalman', 4],
  ['apple', 'Apple Watch 的睡眠监测数据准吗，和手环比', 'moonlight', 8],
  ['programmer', '文档写到什么程度算够，团队里的分歧点', 'est', 15],
];

const RICH_TOPIC_ID = 1251000;
const DAY_MS = 86400;

function nowSec() {
  return Math.floor(Date.now() / 1000);
}

function startOfLocalDay(sec) {
  const date = new Date(sec * 1000);
  date.setHours(0, 0, 0, 0);
  return Math.floor(date.getTime() / 1000);
}

function localDate(sec) {
  const date = new Date(sec * 1000);
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return date.getFullYear() + '-' + month + '-' + day;
}

function archiveRange() {
  const today = startOfLocalDay(nowSec());
  return { from: today - 3 * DAY_MS, to: nowSec() };
}

function makeEntries() {
  const base = nowSec() - 600;
  return ENTRIES.map(function (row, index) {
    const node = row[0];
    const title = row[1];
    const author = row[2];
    const replies = row[3];
    const id = 1251000 + index * 3;
    const createdAt = base - index * 7200 - (index % 5) * 240;
    const lastTouchedAt = Math.min(nowSec() - 60, createdAt + Math.min(replies, 60) * 420 + (index % 7) * 60);
    return {
      id: id,
      node: node,
      nodeTitle: NODE_TITLES[node] || node,
      title: title,
      author: author,
      replies: replies,
      createdAt: createdAt,
      createdKnown: true,
      lastTouchedAt: lastTouchedAt,
      pinned: index === 0 || index === 5,
    };
  });
}

const ALL = makeEntries();
const ENTRY_BY_ID = new Map(ALL.map(function (entry) { return [entry.id, entry]; }));

const RICH_OP_CONTENT =
  '<p>最近把主力工作流整个搬到了 Windows，顺手整理了一份命令行工具清单。都是日常真的在用的，不是收藏夹里吃灰的那种。</p>' +
  '<p>先放一张桌面截图：</p>' +
  '<p><img src="/assets/mock-shot-1.svg" alt="mock 桌面截图"></p>' +
  '<p>启动耗时对比也一并贴出来：</p>' +
  '<p><img src="/assets/mock-shot-2.svg" alt="mock 启动耗时对比"></p>' +
  '<h3>清单</h3>' +
  '<ul>' +
  '<li><strong>fzf</strong>：历史命令和文件跳转，配合 ripgrep 很快</li>' +
  '<li><strong>ripgrep</strong>：搜索代码，默认忽略 .gitignore</li>' +
  '<li><strong>zoxide</strong>：目录跳转，用一周就回不去了</li>' +
  '<li><strong>bat</strong>：带高亮的 cat，看配置文件很舒服</li>' +
  '<li><strong>starship</strong>：跨 shell 的提示符，配置文件就一个 toml</li>' +
  '</ul>' +
  '<h3>安装方式</h3>' +
  '<pre><code>winget install junegunn.fzf\nwinget install BurntSushi.ripgrep.MSVC\nwinget install ajeetdsouza.zoxide\nwinget install sharkdp.bat\nwinget install Starship.Starship</code></pre>' +
  '<p>PowerShell 的 profile 里加两行：</p>' +
  '<pre><code>Invoke-Expression (&amp;starship init powershell)\nInvoke-Expression (&amp;zoxide init powershell | Out-String)</code></pre>' +
  '<p>之前录过一段演示，可以直接看：</p>' +
  '<iframe src="https://www.youtube.com/embed/dQw4w9WgXcQ" width="640" height="360" allowfullscreen title="demo"></iframe>' +
  '<p>B 站也有同步的一份：<a href="https://www.bilibili.com/video/BV1xx411c7mD">BV1xx411c7mD</a>，字幕稍微慢几天。</p>' +
  '<blockquote>注意：不要把 PowerShell 的 ExecutionPolicy 直接改成 Bypass，profile 脚本签名校验还是建议保留。</blockquote>' +
  '<p>完整的 dotfiles 放在 <a href="https://github.com/example/dotfiles">github.com/example/dotfiles</a>，配置项都写了注释。</p>';

const GENERIC_OP = [
  '<p>标题里的问题最近碰到了，搜了一圈没找到太好的答案，来问问大家的实际经验。</p>',
  '<p>我目前的情况是：日常使用强度中等，试过几种方案，各有利弊。想听听不同的思路。</p>',
  '<p>补充一下环境信息，方便判断：系统是最新的稳定版，硬件不算新，预算控制在合理范围内。</p>',
  '<p>欢迎分享你的做法，有具体配置或命令更好，我会整理到主楼。</p>',
].join('');

function opContent(entry) {
  if (entry.id === RICH_TOPIC_ID) return RICH_OP_CONTENT;
  return '<p>' + entry.title + '</p>' + GENERIC_OP;
}

function replyHtml(topic, floor, author) {
  if (author === topic.author) {
    if (floor % 2 === 0) return '<p>谢谢反馈，已经更新到主楼。后续有变化也会在这里补充。</p>';
    return '<p>补充一下前面没说清楚的地方：具体配置我放到主楼的代码块里了，可以直接复制。</p>';
  }
  switch (floor % 12) {
    case 0:
      return '<p>可以试试这个思路，我这边实测有效：</p><pre><code>Get-ChildItem -Recurse | Where-Object { $_.Length -gt 50MB } | Sort-Object Length -Descending | Select-Object -First 20 FullName, @{n="MB";e={[math]::Round($_.Length/1MB,1)}}</code></pre>';
    case 1:
      return '<p>同感。这个问题我折腾了一周，最后发现是驱动版本的问题，回退之后就没再出现过。</p>';
    case 2:
      return '<p>楼主这个清单挺实用的，补充一个 <strong>fd</strong>，比 find 好记，和 ripgrep 是同一个作者。</p>';
    case 3:
      return '<p>贴一下我的实际效果，供参考：</p><p><img src="/assets/mock-shot-2.svg" alt="mock 效果截图"></p>';
    case 4:
      return '<blockquote>不要把 ExecutionPolicy 直接改成 Bypass</blockquote><p>同意，安全边界还是别自己拆掉。用 CurrentUser 的 RemoteSigned 就够了。</p>';
    case 5:
      return '<p>相关问题我之前录过一期视频，B 站：<a href="https://www.bilibili.com/video/BV1xx411c7mD">这里</a>，讲得比较浅，适合入门。</p>';
    case 6:
      return '<p>先说结论：值得折腾，但别一次全上。先换终端和字体，再慢慢加工具，不然配置会变成另一个负担。</p>';
    case 7:
      return '<p>我的环境比较旧，试了一下有两个工具装不上，后来用 scoop 装的，版本会新一点。</p>';
    case 8:
      return '<p>顺便问一下楼主，starship 在 PowerShell 里启动大概多慢？我这边冷启动能感觉到卡顿。</p>';
    case 9:
      return '<p>补充几个容易踩的坑：</p><ul><li>路径里带空格要引号包起来</li><li>PowerShell 5.1 和 7 的语法有差异</li><li>别在 profile 里做网络请求</li></ul>';
    case 10:
      return '<p>这个我熟，之前写过一篇笔记，核心就是先把默认终端换成 Windows Terminal，不然体验提升有限。</p>';
    default:
      return '<p>感谢分享，收藏了，周末照着配置一遍。</p>';
  }
}

function makeReplies(topic, page) {
  const pageSize = 100;
  const total = topic.repliesCount;
  const pages = Math.max(1, Math.ceil(total / pageSize));
  const from = (page - 1) * pageSize + 1;
  const to = Math.min(total, page * pageSize);
  const replies = [];
  for (let floor = from; floor <= to; floor++) {
    let author = REPLY_AUTHORS[(floor * 7 + topic.id) % REPLY_AUTHORS.length];
    if (floor % 17 === 0) author = topic.author;
    if (floor === 7 || floor === 55) author = 'spamking';
    replies.push({
      floor: floor,
      author: author,
      createdAt: topic.createdAt + floor * 260,
      contentHtml: replyHtml(topic, floor, author),
      isOp: author === topic.author,
    });
  }
  return { page: page, pages: pages, pageSize: pageSize, replies: replies };
}

function makeTopic(id) {
  const entry = ENTRY_BY_ID.get(Number(id));
  const fallback = {
    id: Number(id),
    node: 'tech',
    nodeTitle: '科技',
    title: '主题 #' + id,
    author: 'sayo',
    replies: 0,
    createdAt: nowSec() - 7200,
    createdKnown: true,
  };
  const source = entry || fallback;
  return {
    id: source.id,
    node: source.node,
    nodeTitle: source.nodeTitle,
    title: source.title,
    author: source.author,
    createdAt: source.createdAt,
    repliesCount: source.replies,
    views: 1200 + (source.id % 8000),
    contentHtml: opContent(source),
    fetchedAt: nowSec(),
    source: 'mock',
  };
}

function topicPage(id, page) {
  const topic = makeTopic(id);
  const pages = Math.max(1, Math.ceil(topic.repliesCount / 100));
  const current = Math.min(pages, Math.max(1, Number(page) || 1));
  const data = makeReplies(topic, current);
  return {
    topic: topic,
    page: current,
    pages: pages,
    pageSize: 100,
    replies: data.replies,
    summary: summaryFor(id),
  };
}

const summaries = new Map();
const status = { syncingUntil: 0 };

function summaryFor(id) {
  const cached = summaries.get(Number(id));
  if (cached) return { status: 'ready', content: cached.content, model: cached.model, createdAt: cached.createdAt };
  return { status: 'none', content: '', model: '', createdAt: 0 };
}

function summaryText(id) {
  const topic = makeTopic(id);
  const lines = [
    '## 讨论概要',
    '',
    '围绕「' + topic.title + '」这个主题，' + topic.repliesCount + ' 条回复里主要讨论了三个方面：',
    '',
    '- **工具与方案选择**：多数回复给出了具体工具和配置，偏向可复制粘贴的命令行方案',
    '- **环境差异**：Windows 10/11 与 PowerShell 5.1/7 的行为差异被多次提到',
    '- **踩坑提醒**：权限策略、路径空格、启动速度是出现频率最高的三个问题',
    '',
    '## 高赞观点',
    '',
    '1. 先换终端和字体，再逐步加工具，不要一次全上',
    '2. 配置里不要放网络请求，冷启动会明显变慢',
    '3. 用包管理器统一安装，手动下载的版本很难维护',
    '',
    '## 结论',
    '',
    '如果你只想做一件事：先把常用工具装好并写进 profile，再花半小时把键位调顺，收益已经足够。剩下的配置可以慢慢迭代。',
  ];
  return lines.join('\n');
}

function sleep(ms) {
  return new Promise(function (resolve) { setTimeout(resolve, ms); });
}

function delay(value, ms) {
  const wait = ms == null ? 120 + Math.floor(Math.random() * 160) : ms;
  return sleep(wait).then(function () { return value; });
}

function restrictedNode(name) {
  const node = NODES.find(function (item) { return item.name === name; });
  return node ? node.restricted : false;
}

function listPage(kind, node, page, date) {
  const pageSize = 20;
  let source = [];
  let pages = 1;
  let notice = '';

  if (kind === 'home') {
    source = ALL.slice().sort(function (a, b) { return b.lastTouchedAt - a.lastTouchedAt; }).slice(0, 20);
    pages = 1;
  } else if (kind === 'recent') {
    source = ALL.slice().sort(function (a, b) { return b.lastTouchedAt - a.lastTouchedAt; });
    pages = Math.max(1, Math.ceil(source.length / pageSize));
  } else if (kind === 'new') {
    source = ALL.slice().sort(function (a, b) { return b.createdAt - a.createdAt; });
    pages = Math.max(1, Math.ceil(source.length / pageSize));
  } else if (kind === 'node') {
    if (restrictedNode(node)) {
      if (loginState.loggedIn) {
        source = restrictedEntries();
        pages = Math.max(1, Math.ceil(source.length / pageSize));
      } else {
        notice = '节点 ' + node + ' 需要登录后才能查看（在 v2ex.com 登录后按 g l 粘贴 Cookie）';
        source = [];
        pages = 1;
      }
    } else {
      source = ALL.filter(function (entry) { return entry.node === node; }).sort(function (a, b) { return b.lastTouchedAt - a.lastTouchedAt; });
      pages = Math.max(1, Math.ceil(source.length / pageSize));
    }
  } else if (kind === 'day') {
    source = ALL.filter(function (entry) { return localDate(entry.createdAt) === date; }).sort(function (a, b) { return b.createdAt - a.createdAt; });
    pages = Math.max(1, Math.ceil(source.length / pageSize));
  }

  const start = (page - 1) * pageSize;
  const entries = kind === 'home' ? source : source.slice(start, start + pageSize);
  return {
    kind: kind,
    node: node || '',
    page: page,
    pages: pages,
    entries: entries.map(function (entry) { return Object.assign({}, entry); }),
    fetchedAt: nowSec(),
    stale: false,
    notice: notice,
    archive: archiveRange(),
  };
}

const blocked = {
  topics: new Set(),
  users: new Set(['spamking']),
};

const loginState = { hasCookie: false, loggedIn: false, user: '', checkedAt: 0 };

function serializeBlocks() {
  return { topics: Array.from(blocked.topics), users: Array.from(blocked.users) };
}

function serializeLogin() {
  return {
    hasCookie: loginState.hasCookie,
    loggedIn: loginState.loggedIn,
    user: loginState.user,
    checkedAt: loginState.checkedAt,
  };
}

function restrictedEntries() {
  return ALL.slice(0, 8).map(function (entry, index) {
    return {
      id: 1259000 + index,
      node: 'nsfw',
      nodeTitle: 'NSFW',
      title: '[受限节点] ' + entry.title,
      author: entry.author,
      replies: entry.replies,
      createdAt: entry.createdAt,
      createdKnown: true,
      lastTouchedAt: entry.lastTouchedAt,
      pinned: false,
    };
  });
}

function followedInfos() {
  return FOLLOWED.map(function (name) {
    const node = NODES.find(function (item) { return item.name === name; });
    return node
      ? Object.assign({}, node)
      : { name: name, title: name, titleAlt: '', parent: '', topicsCount: 0, restricted: false };
  });
}

export function createMockApi() {
  return {
    bootstrap: function () {
      const range = archiveRange();
      return delay({
        version: '0.1.0-mock',
        nodes: NODES.map(function (node) { return Object.assign({}, node); }),
        followedNodes: FOLLOWED.slice(),
        config: { aiEnabled: true, aiModel: 'mock-summary', theme: 'dark', siteTitle: 'v2ex-vimreader' },
        status: { dbTopics: ALL.length, dbReplies: 5280, lastFeedSync: nowSec() - 240, rateRemaining: 592, login: serializeLogin() },
        archive: range,
      });
    },

    list: function (params) {
      const options = params || {};
      const kind = options.kind || 'home';
      const page = Math.max(1, Number(options.page) || 1);
      const node = options.node || '';
      const date = options.date || '';
      return delay(listPage(kind, node, page, date));
    },

    topic: function (id, page) {
      return delay(topicPage(id, page));
    },

    replies: function (id, page) {
      const topic = makeTopic(id);
      return delay(makeReplies(topic, Math.min(Math.max(1, Number(page) || 1), Math.max(1, Math.ceil(topic.repliesCount / 100)))));
    },

    summaryGet: function (id) {
      return delay(summaryFor(id), 80);
    },

    summaryStream: async function (id, options) {
      const settings = options || {};
      if (settings.refresh) summaries.delete(Number(id));
      const cached = summaries.get(Number(id));
      if (cached) {
        await sleep(120);
        settings.onDelta?.(cached.content, cached.content);
        return { status: 'ready', content: cached.content, model: cached.model };
      }
      const text = summaryText(id);
      await sleep(260);
      let sent = 0;
      while (sent < text.length) {
        const step = 6 + Math.floor(Math.random() * 8);
        const delta = text.slice(sent, sent + step);
        sent += step;
        settings.onDelta?.(delta, text.slice(0, sent));
        await sleep(24);
      }
      const model = 'mock-summary';
      summaries.set(Number(id), { content: text, model: model, createdAt: nowSec() });
      return { status: 'ready', content: text, model: model };
    },

    blocks: function () {
      return delay(serializeBlocks(), 60);
    },

    addBlock: function (kind, value) {
      if (kind === 'topic') blocked.topics.add(Number(value));
      else blocked.users.add(String(value));
      return delay(null, 60);
    },

    removeBlock: function (kind, value) {
      if (kind === 'topic') blocked.topics.delete(Number(value));
      else blocked.users.delete(String(value));
      return delay(null, 60);
    },

    nodes: function () {
      return delay({ nodes: NODES.map(function (node) { return Object.assign({}, node); }), fetchedAt: nowSec() }, 220);
    },

    followedNodes: function () {
      return delay({ followedNodes: FOLLOWED.slice(), nodes: followedInfos() }, 80);
    },

    saveFollowedNodes: function (names) {
      const list = Array.isArray(names)
        ? names.map(function (name) { return String(name || '').trim().toLowerCase(); }).filter(Boolean)
        : [];
      const unknown = list.filter(function (name) { return !NODE_TITLES[name]; });
      if (unknown.length) {
        return sleep(90).then(function () {
          const error = new Error('节点名不合法：' + unknown.join(', '));
          error.code = 'invalid_node';
          error.status = 400;
          throw error;
        });
      }
      FOLLOWED.length = 0;
      for (const name of list) if (FOLLOWED.indexOf(name) < 0) FOLLOWED.push(name);
      return delay({ followedNodes: FOLLOWED.slice(), nodes: followedInfos() }, 160);
    },

    status: function () {
      const range = archiveRange();
      return delay({
        db: { topics: ALL.length, replies: 5280, lists: 12 },
        fetcher: { requests: 214, rateRemaining: 592, lastError: '' },
        sync: {
          lastSiteFeed: nowSec() - 240,
          lastNodeFeeds: nowSec() - 900,
          running: status.syncingUntil > nowSec(),
        },
        archive: range,
        login: serializeLogin(),
      }, 90);
    },

    login: function (cookie) {
      const value = String(cookie || '').trim();
      if (!value || value.length < 8) {
        return sleep(90).then(function () {
          const error = new Error('Cookie 无效或过短（mock 模式至少 8 个字符）');
          error.code = 'login_failed';
          error.status = 400;
          throw error;
        });
      }
      loginState.hasCookie = true;
      loginState.loggedIn = true;
      loginState.user = 'mock-user';
      loginState.checkedAt = nowSec();
      return delay({ hasCookie: true, loggedIn: true, user: 'mock-user', message: '登录成功（mock）' }, 240);
    },

    loginChrome: function () {
      const mode = new URLSearchParams(location.search).get('mockchrome');
      if (mode === 'fail') {
        return sleep(700).then(function () {
          const error = new Error('无法连接 Chrome DevTools（mock 失败分支）');
          error.code = 'chrome_unavailable';
          error.status = 502;
          throw error;
        });
      }
      return delay(null, 900).then(function () {
        loginState.hasCookie = true;
        loginState.loggedIn = true;
        loginState.user = 'demo_user';
        loginState.checkedAt = nowSec();
        return { hasCookie: true, loggedIn: true, user: 'demo_user', checkedAt: loginState.checkedAt };
      });
    },

    logout: function () {
      loginState.hasCookie = false;
      loginState.loggedIn = false;
      loginState.user = '';
      loginState.checkedAt = nowSec();
      return delay({ hasCookie: false, loggedIn: false }, 140);
    },

    sync: function () {
      status.syncingUntil = nowSec() + 2;
      return delay({ started: true }, 120);
    },

    search: function (term) {
      const query = String(term || '').toLowerCase();
      const topics = ALL.filter(function (entry) {
        return !query || entry.title.toLowerCase().includes(query) || entry.author.toLowerCase().includes(query);
      }).map(function (entry) {
        return { id: entry.id, title: entry.title, node: entry.node, author: entry.author, replies: entry.replies, createdAt: entry.createdAt };
      });
      return delay({ topics: topics.slice(0, 50) }, 120);
    },
  };
}