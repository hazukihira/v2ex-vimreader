# V2EX 轻量阅读器：前期调研

调研时间：2026-10-26。所有结论都基于当天对 v2ex.com 的实测（HTTP 状态码、响应头、
页面结构、feed 内容）以及官方帮助页；不是二手转述。项目为个人阅读用途，不做全网批量爬取、
不绕过登录墙、不重新分发内容。

## 0. 结论速览

| # | 问题 | 结论 |
|---|------|------|
| 1 | 技术栈 | Go 单二进制（内嵌 SQLite + 静态前端）+ 原生 JS/CSS，无框架、无 Node 运行时；Windows/macOS/Linux 一套代码 |
| 2 | 未登录能否看全部帖子 | 不能。公开节点/帖子/楼层可匿名阅读；受限节点会跳登录/受限页（实测 8 个，见 §2） |
| 3 | 代价与要求 | 无官方 SLA、页面结构会变；要克制抓取频率、标识 UA、尊重 robots、不能绕过登录；内容版权归作者，仅本地个人阅读 |
| 4 | 官方接口 | API 2.0 需要 Personal Access Token（登录后创建；读接口也强制 token），限 600 次/IP/小时；另有公开 JSON Feed / Atom Feed 可匿名读，带 ETag。无"全站导出"接口 |
| 5 | 增量抓取 | 可以。feed 支持 `If-None-Match`（实测 304）；feed 固定最近 50 条、无时间窗口；主题 ID 单调递增；回复 100 层/页，可用 JSON-LD 的 `commentCount` 只补最后一页 |
| 6 | 按时间排序分割 | 可以，基于本地归档按创建时间（UTC+8 自然日）分页。站点没有公开的"按创建时间翻页"列表，所以任意历史日期能否看到取决于本地归档覆盖范围 |
| 7 | 图片/视频 | 图片可直接外链（实测带/不带 Referer 都 200），懒加载 + 点击看原图；视频多为外链或 iframe（YouTube），做点击后才加载的白名单嵌入 |
| 8 | 性能 | 后端常驻约 20-40MB、单文件约 15-20MB；前端无框架，首屏 < 100KB；本地 SQLite 缓存优先渲染，网络抓取后台进行 |
| 9 | 屏蔽用户/帖子 | 本地 blocks 表（user/topic），服务端过滤 + 前端高亮/隐藏，快捷键可操作 |
| 10 | 拓展性 | AI 总结走 OpenAI Chat Completions 兼容配置（base_url/model/api_key/自定义 Header/prompt），支持 SSE 流式；HTTP JSON API 便于二次开发 |

## 1. 技术栈选型

**选定：Go 1.27 单二进制 + 内嵌 SQLite（`modernc.org/sqlite`，纯 Go 无需 CGO）+ 原生 ES Modules 前端。**

原因：

- 阅读器本质是"抓取 + 解析 + 本地缓存 + 渲染"，Go 的 `net/http` + `goquery` 足够，编译后
  是单个 exe，双击即用，Windows 用户不需要装运行时。
- SQLite 用纯 Go 驱动，免 CGO，跨平台交叉编译简单；WAL 模式并发读性能足够，数据库就是本地缓存。
- 前端不用 React/Vue：列表每页 20-25 行、楼层每页 100 层，原生 DOM 足够；无构建步骤意味着
  开源贡献者只要装 Go 就能改前端，也不用 Node 工具链。
- 对比：
  - Node/Next.js 全栈：运行时内存 80-200MB，安装 Node 是额外门槛，偏离"轻量"目标。
  - Electron/Tauri 桌面壳：Electron 内存占用高；Tauri 依赖 WebView2，打包链复杂。浏览器页面已经够用。
  - 纯静态页面直连 V2EX：匿名 API 没有 CORS、无法解析 HTML 页面、无法本地持久化，行不通。

关键依赖：`github.com/PuerkitoBio/goquery`（HTML 解析）、`github.com/microcosm-cc/bluemonday`
（正文消毒）、`modernc.org/sqlite`、`github.com/BurntSushi/toml`（配置）。

## 2. 未登录能看到哪些帖子

实测（2026-10-26，无 Cookie）：

| 目标 | 结果 |
|------|------|
| 首页、节点列表页 `/go/tech`、主题页 `/t/1` | 200，完整 HTML，含全部楼层（100 层/页） |
| `/recent`、`/index.xml`、`/read.xml` | 200，且不含受限节点内容 |

受限节点不是只有三个。独立复核（见 [research-notes.md](research-notes.md) §2.2）实测了 8 个，
且匿名访问表现为三种 302 目标：

| 节点 | 匿名响应 | 跳转到 |
|------|----------|--------|
| `nsfw`、`sandbox`、`chamber`、`cheap` | 302 | `/signin` |
| `flamewar`（水深火热）、`flood`（水）、`monthly` | 302 | `/`（首页） |
| `survey`（调查） | 302 | `/restricted`（再转 `/signin`） |

反例：`/go/ohno` 匿名 200（同为 chamber 子节点但未受限）；`/go/不存在的节点` 也是 200，
只是页面显示"节点未找到"，所以**不能把 302 当成唯一的受限信号**。

V2EX 官方在 2026-01-14 的主题里说明了节点可见性规则（[t/1185649](https://www.v2ex.com/t/1185649)）：
当节点的 `included_in_homepage` 被管理员设为 0 时，节点内容不会进入首页、`/recent`、
`/index.xml` 或 `/read.xml`，且节点内主题需要登录才能查看。实测与这条说明一致。

**重要：受限内容仍能通过官方通道匿名读到。** 独立复核实测 `/feed/{nsfw,flamewar,flood,sandbox,chamber}.json`、
对应 `.xml`、以及旧接口 `/api/topics/show.json?node_name=nsfw` 均返回 200 且带正文。
所以"不绕过登录墙"不能只靠"不抓页面"，必须在解析/同步层按节点名字显式过滤：

- 维护受限节点清单（当前 8 个，见配置项 `sync.restricted_nodes`）；
- 抓节点 feed 前先判断受限，受限节点直接跳过；
- 站点级 feed 写入前再按节点名兜底过滤；
- 节点目录里把这些节点标记为受限，UI 点击时提示"需要登录后查看"。

**登录与边界**：项目默认匿名，不模拟登录、不自动盗用浏览器会话。用户如果愿意，可以在网页登录面板
粘贴自己的 V2EX Cookie（或 `-cookie` / `scripts/import-chrome-cookie.mjs` 从本机 Chrome 导入），
Cookie 只保存在本机 SQLite，登录后受限节点的列表、主题与楼层按普通节点读取；退出登录立即清除。
这只是"使用自己的账号看自己有权限看的内容"，不是绕过登录墙。

**结论：未登录可以看绝大多数公开帖子与全部楼层，但受限节点（NSFW 等 8 个）不行；匿名模式下项目在数据层主动过滤这些节点的 feed，可选使用用户自己的 Cookie 登录后正常读取。**

## 3. 代价与要求

1. **无 SLA、会变**：HTML 结构、feed 路径都可能改版；解析器要写成容错 + 单测 + 失败提示。
2. **请求预算**：官方 API 文档写明默认 600 次/IP/小时；"能被 CDN 缓存的 API 请求只有第一次消耗配额"
   （`/help/api` 原文）。HTML 页面没有公开限额，但 Cloudflare 在前面，必须克制。本项目默认全局
   1 req/s、单并发、缓存 TTL + 本地优先渲染。
3. **robots.txt**（实测）：只禁止 `/backstage/`、`/signin`、`/signout`、`/settings`，
   内容页面不在禁止范围；`bingbot` 的 Crawl-delay 是 120。
4. **版权**：帖子内容版权归作者，本项目只做本地缓存/阅读，不提供导出再分发功能；README 写明用途。
5. **法律边界**：绕过登录、破解反爬、批量抓取个人信息存在法律风险（社区有相关讨论）。
   本项目默认不抓受限节点、不模拟登录、不并发轰炸。
6. **开源注意**：UA 带项目标识；限速可配置；README 声明非官方、与 V2EX 无关。
7. **robots 不等于授权**：robots.txt 只说明抓取路径；旧 API 文档的公平使用规则明确反对把 API
   输出用于填充商业/个人网站的内容。公开部署本项目需要另行评估。
8. **Cloudflare 与错误语义**：站点在 Cloudflare 后面，高频/异常访问可能返回 403 或 520；
   解析器要区分 403/404/302/200（200 也可能是"节点未找到"），网络层做指数退避并给出可重试提示。
9. **302 不一定是登录墙**：受限节点可能跳到 `/signin`、`/` 或 `/restricted`，三种都要识别，
   不能只把 `/signin` 当受限信号。

## 4. V2EX 是否提供帖子接口

**有，但分两类，都不能"全站导出"。**

### 4.1 官方 API 2.0（需要 Personal Access Token）

`/help/api` 原文（2026-08-03 更新）：

- Header `Authorization: Bearer <token>`；token 在登录后创建（最多 10 个，30/60/90/180 天有效）。
- 相关端点：
  - `GET /api/v2/nodes/:node_name`
  - `GET /api/v2/nodes/:node_name/topics?p=N`
  - `GET /api/v2/topics/:topic_id`
  - `GET /api/v2/topics/:topic_id/replies?p=N`
- 默认每个 IP 每小时 600 次，响应头有 `X-Rate-Limit-Limit/Reset/Remaining`（实测 v1 接口也是 600；
  文档示例里的 120 是旧值）。
- 读接口也需要 token：实测无 token 与假 token 请求 `/api/v2/nodes/all`、`/api/v2/topics/1`、
  `/api/v2/nodes/tech/topics?p=1` 均 401 `{"message":"Token not found"}`，不存在匿名读。
- 端点表还包括 notifications、member、token、POST tokens、set-sticky、boost；
  token 创建/管理页是 [`/settings/tokens`](https://www.v2ex.com/settings/tokens)（匿名 302 到登录页），
  令牌只会在创建后 10 分钟内完整显示一次，最长 180 天有效。

### 4.2 公开 Feed（匿名即可，适合增量）

- 节点 JSON Feed：`https://www.v2ex.com/feed/<node>.json` —— JSON Feed 1.0，固定最近 50 条、
  **没有时间窗口**（跨度取决于节点活跃度：tech 样本覆盖 481 天，冷门节点 api 可达 1664 天），
  按 `date_published` 倒序，实测 `date_published == topic.created`；
  字段含 `author`、`url`、`title`、`date_published`、`content_html`（完整正文）。
  响应带 `ETag`、`Cache-Control: public, max-age=1800`，`If-None-Match` 实测返回 304。
- 站点 Atom Feed：`/index.xml`、`/read.xml` —— 各 50 条最新主题，两者实测 49/50 重合，
  按一份处理即可；按创建时间倒序，含标题、作者、发布时间、正文 HTML。
  ETag 存在，但内容每分钟在变，按 5 分钟以上间隔轮询比较合理。

### 4.3 旧的 API 1.0（不建议依赖）

`/api` 这个地址 302 到 `/go/api`（一个讨论节点），不是文档入口；API 2.0 文档在
[/help/api](https://www.v2ex.com/help/api)，旧 API 1.0 的文档还留在
[/p/7v9TEc53](https://www.v2ex.com/p/7v9TEc53)（2014 年，写的是 120 次/小时，已过时，实测与
`/help/api` 都是 600）。

旧接口 `/api/topics/latest.json`、`/api/topics/hot.json`、`/api/topics/show.json?id=...`、
`/api/nodes/all.json`、`/api/site/info.json` 目前匿名可用，但实测**分页参数 `p` 被忽略**
（p=1/2/3/10 返回同一份数据），且 `latest.json` 有 5 天 CDN 缓存、数据滞后于站点最新主题。
它们只能当一次性快照/兜底，项目不拿它做主数据源。旧文档还写明了 API 公平使用规则：
鼓励学术研究、手机应用和浏览器扩展，反对把 API 输出用于填充商业/个人网站的内容。

**结论：没有"抓取全部帖子"的官方接口；能稳定匿名使用的是 feed + HTML 页面，配合本地缓存做增量。**

## 5. 增量抓取

可行，设计如下：

- **新主题发现**：轮询 `/index.xml`、`/read.xml` 和关注节点的 `/feed/<node>.json`。带
  `If-None-Match`，未变时 304、几乎零流量；变了只 upsert 最新 50 条，按主题 ID 去重。
- **列表页**：用户打开列表时才抓对应 HTML 页（首页 / 节点 / `/recent`），解析后写
  `list_pages` 快照并带 TTL；滚动翻页再抓下一页。不预先扫全站。
- **主题正文**：feed 里已带正文的直接可用；不在 feed 里的打开时抓一次。
- **楼层**：主题页 100 层/页；页面内嵌 JSON-LD 的 `commentCount` 就是总回复数，可算总页数。
  首次打开抓第 1 页，用户翻页/跳最新再抓对应页；再次打开只重抓最后一页找新增。
  每页楼层按 `(topic_id, floor)` 主键 upsert，天然幂等。
- **去重**：所有表以站点 ID 为主键（topics.id、replies(topic_id,floor)），不存在"全量查重"。
- **限速**：全局令牌桶 + 单/双并发 + 重试退避，抓取记录曝光在状态栏。

**做不到的部分**：公开接口没有"按创建时间翻页"的历史列表（见下节），所以没法一次性补全
任意历史日期的归档；归档从首次运行开始按 feed 增量累积。

## 6. 按时间排序 / 只某一天

**能做，但基于本地归档。**

实测限制：

- `/recent`（Recent Topics）22 条/页、约 41731 页；行内时间是**最后活动时间**而非创建时间
  （实测主题 1251929 创建于 15:30，列表显示 16:09 最后回复），顺序也被顶帖打乱
  （第 1 页混着老主题 1249321；推广位首行甚至可能不一致，不能当排序字段）。
- 节点页 `/go/<node>` 20 条/页，按最后回复排序，且最多翻 30 页（p=31 与 p=30 返回同一批）。
- feed 只给最新 50 条，没有分页参数（`/index.xml?p=2` 实测返回同一份内容）。
- 本来设想用 `/recent` 分页 + 二分定位某一天，实测排序规则不成立，放弃。

因此"按天看"的实现：

1. 后台增量同步把 feed 里的新主题（含创建时间）持续写入本地 SQLite；
2. "按天"视图按 `created_at` 的 UTC+8 自然日分组，本地分页，秒开；
3. 页面显示归档覆盖范围（最早/最晚日期），没覆盖到的日期会提示"归档未覆盖"；
4. 可选扩展：登录后配置 Personal Access Token，用 API 2.0 的节点分页接口慢慢补历史（不默认开启）。

这个方案的回答是：**能按时间分割浏览，但"任意历史某天"需要本地归档覆盖到那一天**，
不是立刻能看全站任意历史。这是匿名 + 克制抓取的取舍。

## 7. 图片与视频

- 正文里的图片基本是外链图床（实测样本：`raw.githubusercontent.com`、`i.v2ex.co`、
  `i.imgur.com`、腾讯云 COS、个人域名）。对 `i.v2ex.co` 与 `raw.githubusercontent.com`
  实测带/不带 `Referer` 都返回 200；其它图床未逐一验证，实现上按"可能失败"处理（`onerror` 占位）。
- 方案：`<img loading="lazy" referrerpolicy="no-referrer">` 直接展示，限制最大高度；
  点击打开内置灯箱，灯箱里提供"查看原图"链接（新标签页），同时保留点击图片直接打开原图的路径。
- 视频：V2EX 正文支持嵌入（实测 `/read.xml` 里有 YouTube iframe：`embedded_video_wrapper`）。
  B 站等站点多数是普通链接（未系统验证）。方案：白名单 iframe（youtube.com / player.bilibili.com）
  默认显示播放占位卡片，点击后才创建 iframe（省流量、避免第三方跟踪）；其它视频站渲染成链接卡片，
  点击到原站看。
- 后续可选：图片代理缓存（隐私/弱网），默认不做，避免额外磁盘与带宽。

## 8. 性能与小占用

- 架构层面：单进程、无 Node 运行时、无前端框架、无 CDN；SQLite 只在本地读写。
- 交互层面：列表/主题优先读本地缓存立即渲染，网络请求后台补；带 TTL，重复打开不重复抓。
- 网络层面：ETag 条件请求、全局令牌桶、单并发、失败退避；前端页面隐藏时暂停轮询。
- 预期指标（预估，后续实测补充）：常驻内存 20-40MB；数据库每月增量 < 50MB；首屏资源 < 100KB；
  冷启动打开列表 < 300ms（本地缓存命中）。
- 明确不做：全站预抓取、图片预下载、后台定时全量扫描。

## 9. 屏蔽用户 / 帖子

- SQLite `blocks` 表：`(kind, value)`，kind=user/topic。
- 服务端在列表、主题楼层、搜索结果里过滤（或返回标记由前端决定显示"已屏蔽"占位）。
- 前端快捷键：`b` 打开屏蔽菜单、`x` 屏蔽当前帖子、`u` 屏蔽当前作者；命令面板可解除。
- 屏蔽是本地行为，不影响 V2EX 账号（本项目不需要登录）。

## 10. 扩展性（AI 总结等）

- 配置化 AI：`[ai]` 段配置 `base_url`、`model`、`api_key`/`api_key_env`、自定义
  `[ai.headers`（中转站常用 `X-Title`、`HTTP-Referer`）、prompt 模板；请求走 OpenAI
  `/chat/completions` 兼容格式，SSE 流式输出到前端。
- 总结输入 = 标题 + 节点 + 楼主正文 + 前 N 层回复（按 `max_input_chars` 截断），结果缓存进
  `summaries` 表，重复查看不重复计费。
- 其它扩展方向：本地搜索（SQLite FTS5）、收藏/稍后读、RSS 输出、翻译、按关键词高亮。
- HTTP JSON API 独立于前端，方便脚本或其它客户端复用。

## 11. 待办与风险

- 解析器依赖页面结构，需要 fixture 单测；页面改版时给出明确错误提示而不是崩溃。
- 受限节点清单（8 个：`nsfw`、`sandbox`、`chamber`、`cheap`、`flamewar`、`flood`、
  `monthly`、`survey`）可在 `sync.restricted_nodes` 配置；运行时遇到 302（到
  `/signin`、`/` 或 `/restricted`）会自动标记。节点 feed 与站点 feed 写入前都做节点名过滤，
  避免通过公开 feed 意外摄入受限内容。
- 可选 Cookie 登录已实现（网页登录面板 / `-cookie` / Chrome 导入脚本），Cookie 只存本机；
  Cookie 过期时状态栏会提示，需要重新导入。

## 附录：关键实测记录（2026-10-26）

```
GET /api/topics/latest.json            200  x-rate-limit-limit: 600  x-rate-limit-remaining: 597
GET /api/v2/nodes/all                  401  {"message":"Token not found","success":false}
GET /recent                           200  22 条/页，行内时间=最后回复时间
GET /go/tech                          200  20 条/页，最多第 30 页
GET /t/1?p=1                          200  100 层/页；p=2 为 101-178 层；超出页码被夹到最后一页
GET /t/1 里 JSON-LD                    commentCount: 178, datePublished: 2010-04-25T14:56:27Z
GET /feed/tech.json                   200  50 条，按 date_published 倒序，ETag
GET /feed/tech.json If-None-Match     304  （第二次请求）
GET /index.xml                        200  50 条，ETag
GET /go/nsfw                          302  Location: /signin
GET /go/sandbox                       302  Location: /signin
GET /go/chamber                       302  Location: /signin
GET /go/cheap                         302  Location: /signin
GET /go/flamewar                      302  Location: /
GET /go/flood                         302  Location: /
GET /go/monthly                       302  Location: /
GET /go/survey                        302  Location: /restricted
GET /go/ohno                          200  （受限父节点下但本身公开）
GET /t/1250875 (NSFW)                 302  Location: /signin?next=/t/1250875
GET /feed/nsfw.json                   200  公开 feed 可匿名读到受限节点内容（项目显式过滤）
GET /robots.txt                       200  仅 Disallow /backstage/ /signin /signout /settings
GET /planes                           200  1379 个节点链接（节点目录页）

请求均使用 `v2ex-vimreader/0.1` 或浏览器 UA，请求间隔 >= 1s，未使用 Cookie。
```

官方页面：[API 2.0 文档](https://www.v2ex.com/help/api)、[节点可见性变更说明](https://www.v2ex.com/t/1185649)、
[robots.txt](https://www.v2ex.com/robots.txt)。

## 12. 独立复核、先例与法律资料

本节由另一位调研者独立复核后补充，完整证据见 [docs/research-notes.md](research-notes.md)（41 条核对表、
20 条修改建议、实测请求记录与未证实清单）。要点：

**先例项目**（按 star 排序，取与本项目相关的）：

- [livid/v2ex-gae](https://github.com/livid/v2ex-gae)（Python，官方旧站源码，已归档）：
  可参考数据模型与权限语义；
- [coolpace/V2EX_Polish](https://github.com/coolpace/V2EX_Polish)、
  [sciooga/v2ex-plus](https://github.com/sciooga/v2ex-plus)：浏览器扩展，直接在原站 DOM 上增强，
  不建爬虫、成本最低，但无法离线归档；
- [gexiao/sov2ex](https://github.com/gexiao/sov2ex)（Go）：自建索引的站内搜索，可参考索引设计；
  全站抓取/索引的合规风险较高，本项目不做；
- [oldshensheep/v2ex_scrapy](https://github.com/oldshensheep/v2ex_scrapy)：3.7GB 全站 SQLite 爬虫，
  README 明确"爬快了封 IP、并发 3 可长期爬、部分帖需 Cookie、含水深火热"——反面教材，
  本项目不做全站批量抓取；
- [kaolengmian7/v2ex-tui](https://github.com/kaolengmian7/v2ex-tui)、
  [six-ddc/v2ex-tui](https://github.com/six-ddc/v2ex-tui)（Go）：与本次技术栈最接近的终端客户端，
  可参考网络/解析/键盘交互；
- [DIYgod/RSSHub](https://github.com/DIYgod/RSSHub) 的 `/v2ex/topics/:type` 路由直接调旧 API，
  说明轻量聚合可行，但其公开部署形态与官方 API 公平使用规则存在张力。

**法律资料**（只引公开资料，不构成法律意见）：

- 《著作权法》（2020 修正）：第 10 条第（十二）项信息网络传播权、第 11 条"著作权属于作者"、
  第 24 条第（一）项个人学习/研究/欣赏的合理使用。
- 《刑法》第 285 条：非法获取计算机信息系统数据罪。
- [检察日报 2025-06-03（最高检网站转载）](https://www.spp.gov.cn/spp/llyj/202506/t20250603_697192.shtml)：
  "行为人单纯违反 robots 协议、仅爬取公开数据不宜入罪"，但"破解加密算法、突破安全防护措施
  获取非公开数据"可能构成犯罪。
- [北京知识产权法院 2023-03 文章](https://bjzcfy.bjcourt.gov.cn/article/detail/2023/03/id/7214062.shtml)：
  绕过访问权限/技术措施抓取非公开数据可能构成不正当竞争；实质替代数据来源网站亦属"搭便车"。
- V2EX 没有公开的服务条款/EULA（`/terms` 302 到词典页），帖子/回复的著作权默认归作者，
  不能假定平台许可再分发；About 页也明确"绝对不会全文转载任何文章"。

**本项目据此坚持的边界**：只读公开页面、本地缓存、个人阅读；不登录、不抓受限节点、不并发轰炸、
不提供导出/公开镜像/二次分发；公开部署或对外提供服务不在本项目目标内。

**尚未证实的点**（见 research-notes 附录 B）：无官方 SLA；Atom 304 未复现（内容每分钟变）；
feed 对超长文是否截断；除两个图床外是否有热链限制；B 站是否多为链接；`m.v2ex.com` 是否可用；
API 2.0 的节点分页是否有 30 页上限。
