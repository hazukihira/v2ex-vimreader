# V2EX 调研核查笔记（独立复核）

- 核查对象：[docs/RESEARCH.md](RESEARCH.md)（211 行，2026-10-26 草稿）
- 核查时间：2026-10-26 16:59–17:26（UTC+8）
- 核查方式：匿名 curl 实测（User-Agent 带项目标识、无 Cookie、请求间隔 ≥1.2s，多数 1.8–2s，分 15 轮、单轮 ≤6 个 URL）；官方帮助页与站内主题原文；GitHub Search API；中文法律公开资料。所有 URL 均为当日可访问。
- 范围声明：本文只做事实核查、资料补充与实现风险提示，**不构成法律意见**。

## 0. 五条关键修正

1. **受限节点不止 3 个，302 目标有 3 种**。实测 8 个受限节点：nsfw、sandbox、chamber、cheap → /signin；flamewar（水深火热）、flood（水）、monthly → /；survey（调查）→ /restricted（该页再 302 到 /signin）。另有反例 /go/ohno 匿名 200。把「302 到 /signin」当成唯一受限信号会漏判，把「302 到 /」当成节点不存在也会误判（/go/不存在的节点 返回的是 200 + 「节点未找到」页）。
2. **受限节点内容仍能通过 V2EX 自己的公开通道匿名读取**：/feed/nsfw.json、/feed/nsfw.xml、/feed/flamewar.json、/feed/flood.json、/feed/sandbox.json、/feed/chamber.json 均 200；/api/topics/show.json?node_name=nsfw（以及 flamewar、sandbox）均 200 且返回正文。也就是说「本项目不绕过登录墙」不能只靠「不主动访问受限节点页面」，必须在解析层显式过滤受限节点。
3. **/api 不是重定向到 API 2.0 文档**，实测 302 → /go/api（讨论节点）。API 2.0 文档在 /help/api；旧 API 1.0 文档仍挂在 /p/7v9TEc53（页面日期 2014-08-28），其写的限流是 **120 次/IP/小时**，而实测响应头是 **600**，/help/api 也写 600——旧文档的 120 已过时。
4. **feed 的覆盖不是「最多 477 天」**。JSON Feed 只保留「最近 50 条」，没有时间窗口：/feed/tech.json 50 条覆盖 481 天（2025-07-02 → 2026-10-23），而冷门节点 /feed/api.json 50 条覆盖约 4.6 年（2022-03-14 → 2026-10-03）。「477 天」只是 tech 节点当时的样本，不是上限。
5. **「没有官方 SLA」无法证实**：没有找到 SLA 或可用性承诺的原文，也没有找到「不保证可用」的表述。建议改写为「未见公开 SLA/可用性承诺」。

## 1. 结论核对表

| # | RESEARCH.md 结论（节/行） | verdict | 证据 |
|---|---------------------------|---------|------|
| 1 | 未登录不能看全部帖子；公开节点/帖子/楼层可匿名阅读（§0 行 12、§2） | verified | 首页、/go/tech、/t/1 匿名 200；/t/1 两页共 178 层读完 |
| 2 | /go/nsfw、/go/sandbox、/go/chamber 302 → /signin（§2 行 48、附录 203–204） | verified | 实测三条均 302，Location: /signin |
| 3 | /t/1250875（NSFW 主题）302 → /signin?next=（§2 行 49、附录 205） | verified | 302，Location: /signin?next=/t/1250875 |
| 4 | 受限节点清单「nsfw、sandbox、chamber」且可能变化（§11 行 187–188） | **partially / 不完整** | 另实测 flamewar、flood、monthly、survey、cheap 也受限；/go/ohno 可访问 |
| 5 | 受限节点统一 302 到登录页 | **wrong（不统一）** | flamewar/flood/monthly → /；survey → /restricted；cheap/nsfw/sandbox/chamber → /signin |
| 6 | /feed/nsfw.json、/api/topics/show.json?node_name=nsfw 匿名 200（§2 行 50） | verified，且范围更大 | 另测 /feed/{flamewar,flood,sandbox,chamber}.json 与 .xml 均 200；旧 API 同样 200 |
| 7 | /recent、/index.xml、/read.xml 200 且不含受限节点内容（§2 行 51） | verified | 5 个受限节点 feed 各 50 条（共 250 条 ID）与 index/read/recent 交集均为 0 |
| 8 | 节点可见性官方说明 t/1185649（included_in_homepage=0 → 不上首页/不进 /recent、/index.xml、/read.xml、需登录；§2 行 53–55） | verified | 原文与文档描述一致，见 2.1 |
| 9 | 无官方 SLA、页面结构会变（§3 行 61） | unverifiable（后半句属合理判断） | 未见 SLA 原文；结构确实多年多次改版（旧客户端多为 2018 年前停更） |
| 10 | API 默认 600 次/IP/小时（/help/api 原文，§3 行 62、§4.1 行 86） | verified | /help/api「默认情况下，每个 IP 每小时…600 次」；实测 x-rate-limit-limit: 600 |
| 11 | 旧 API 1.0 文档也是 600 | **wrong / 需注明** | /p/7v9TEc53（2014）写 120；实测响应头 600。RESEARCH.md 的「实测 v1 也是 600」成立，但旧文档写 120 |
| 12 | CDN 缓存的 API 请求只有第一次消耗配额（§3 行 62） | verified | /help/api 与 /p/7v9TEc53 均有同义原文；实测 cf-cache-status: HIT |
| 13 | robots.txt 仅 Disallow /backstage/、/signin、/signout、/settings；bingbot Crawl-delay 120（§3 行 65–66） | verified | robots.txt 原文一字不差 |
| 14 | 帖文版权归作者，仅本地个人阅读、不重新分发（§3 行 67） | verified（法律默认）但需补一句 | 《著作权法》第 11 条「著作权属于作者」；未找到 V2EX 公开服务条款/内容许可协议（/terms 302 → /d/terms，是词典页） |
| 15 | 绕过登录/破解反爬/批量抓取有法律风险（§3 行 68） | verified（有公开判例与检察案例） | 见 2.10；注意「仅违反 robots + 公开数据」与「破解加密/非公开数据」在刑事上被区分 |
| 16 | 官方 API 2.0 需 Personal Access Token，Bearer，最多 10 个，30/60/90/180 天（§4.1 行 80） | verified | /help/api 与 /help/personal-access-token 原文 |
| 17 | API 2.0 端点只有 nodes/topics/replies 4 类（§4.1 行 82–85） | **partially（不完整）** | 还有 notifications、DELETE notifications/:id、member、token、POST tokens、POST set-sticky、POST boost |
| 18 | 未带 token 请求 /api/v2/nodes/all 返回 401 Token not found（§4.1 行 87） | verified | 401 {"message": "Token not found", "success": false}；带假 token 同样 401 |
| 19 | 节点 JSON Feed：JSON Feed 1.0、固定 50 条、按创建时间倒序、含正文（§4.2 行 91–93） | verified | 实测 50 条；date_published 倒序，且与 topic.created 完全相等（t/1251278） |
| 20 | 实测最早可到 477 天前（§4.2 行 93） | **partially（样本值，不是上限）** | tech 实测 481 天；/feed/api.json 覆盖 1664 天 |
| 21 | feed 带 ETag、Cache-Control max-age=1800、If-None-Match 304（§4.2 行 94） | verified | ETag W/"c499…"、Cache-Control: public, max-age=1800、cdn-cache-control: 120；304 Not Modified 复现 |
| 22 | /index.xml、/read.xml 各 50 条最新主题、按创建时间倒序、含正文 HTML、带 ETag（§4.2 行 95–96） | verified（304 未复现） | 各 50 条，published 倒序，content 为 HTML；ETag 存在；因内容每分钟变化，条件请求返回 200 新 ETag（JSON feed 的 304 
| 23 | 旧 API 1.0 匿名可用（§4.3 行 100） | verified | latest/hot/nodes/all/site/info/site/stats/topics/show 均 200 |
| 24 | 「/api 文档入口已重定向到 API 2.0」（§4.3 行 101） | **wrong** | /api 302 → /go/api（讨论节点）；API 2.0 文档是 /help/api，旧文档是 /p/7v9TEc53 |
| 25 | 旧 API 的分页参数 p 被忽略（§4.3 行 101–102） | verified（对 latest/index/feed 验证） | /api/topics/latest.json p=2 与 p=3 完全同批；/index.xml?p=2 与 /feed/tech.json?p=2 返回同一份最新内容 |
| 26 | 无官方「全站导出」接口（§4 行 74、104） | verified | /help/api 端点表无导出；旧 API 仅有 latest/hot 快照；全站导出只能靠长期本地归档 |
| 27 | 主题页 100 层/页；JSON-LD 的 commentCount 可算总页数（§5 行 115） | verified | /t/1 p1=100 层（1–100），p2=78 层（101–178）；JSON-LD commentCount=178、datePublished=2010-04-25T14:56:27Z |
| 28 | 超出页码被夹到最后一页（附录 198） | verified | /t/1?p=999 返回 101–178 层，与 p=2 相同 |
| 29 | /recent 行内时间是最后回复而非创建时间（§6 行 130） | verified（有异常行） | 1251929 创建 15:30:29、显示 16:09:15 = last_touched；推广位首行 1249854 显示 14:18:47 而 last_touched 为 16:28:42，说明不能把显示时间当排序/时间字段 |
| 30 | /recent 顺序被顶帖打乱（§6 行 131） | verified | 第 1 页混有 1249321 等老主题；显示时间并不单调 |
| 31 | 节点页按最后回复排序、最多 30 页（§6 行 132） | verified | /go/tech?p=30 = 「Topics 581 to 600 of 596」，p=31 与 p=30 返回同一批主题；页脚最后回复人 |
| 32 | feed 只给最新 50 条、无分页参数（§6 行 133） | verified | 见 #20/#25 |
| 33 | /recent 22 条/页（附录 196） | verified | 实测单页 22 个 topic-link / 22 行；分页页码上限 41731 |
| 34 | /go/<node> 20 条/页（附录 197） | verified（末页会少） | /go/tech 第 30 页「Topics 581 to 600 of 596」，实际渲染 16 行（部分主题不可见/已删），第 1 页 20 条 |
| 35 | /planes 1379 个节点链接（附录 207） | verified | /planes 1379 个唯一 /go/ 链接；/api/nodes/all.json 也是 1379 个节点 |
| 36 | 图片带/不带 Referer 都 200、无热链限制（§7 行 148–149） | **partially** | 只验证了 2 个域名：i.v2ex.co（1,030,961B image/png，带与不带 Referer 均 200）、raw.githubusercontent.com（200）；其余图床未测，不能写成绝对结论 |
| 37 | /read.xml 中存在 YouTube iframe（embedded_video_wrapper）（§7 行 152） | verified | read.xml 中 embedded_video_wrapper 出现 1 次、youtube 出现 3 次 |
| 38 | 视频「B 站等多数是可点击链接」（§7 行 153） | unverifiable | 本次未抽样统计，建议删除或标注未验证 |
| 39 | 匿名 API 没有 CORS，浏览器直连不行（§1 行 36） | verified（有据） | 约 100 个响应里没有任何 Access-Control-Allow-Origin 头 |
| 40 | 主题 ID 单调递增（§5 行 111） | verified | 站点 topic_max=1251977（/api/site/stats.json），/t/1=1，最新主题 1251970+；旧 API latest 按 created 倒序 |
| 41 | 后端内存 20–40MB、单文件 15–20MB 等性能数字（§8 行 163–164） | unverifiable | 属于预估，文中也写了「构建完成后实测补充」，核查时无实测 |

## 2. 官方资料摘录与实测证据

### 2.1 节点可见性：官方原文

[V2EX 主题 t/1185649](https://www.v2ex.com/t/1185649)（Livid，2026-01-14，标题「20260114 - 修复了节点 visibility 的一些问题」）：

> 如果一个节点的 included_in_homepage 属性被管理员设置为 0，那么以下行为是设计本意：
> - 内容不会出现 V2EX 首页的任何部分：节点内容不会进入到首页右侧的最热主题或者首页最热 tab
> - 节点内的主题需要用户登录才能查看
> - 节点内的主题不会进入到 /read.xml 或者 /index.xml 这两个订阅源
> - 用户在节点内的回复不会显示在用户的个人主页上
>
> Supplement 1 · Jan 14：节点内的主题不会进入到 /recent

同一事件的 [t/1185368](https://www.v2ex.com/t/1185368) 里，Livid 对「为什么有的节点会提示需要登录」的回复（#26）：

> 节点内如果包含成人内容或者灌水内容，就不会再出现在首页。

#28：

> 如果一个节点创建者，创建完节点之后，自己在节点里发的第一个主题就是成人内容，那么意图是很明确的，那么这个节点自然也会被打上不上首页的标记。

#34：

> 另外两个相关的更新：1. 明确的 NSFW 内容不会在出现在首页及任何订阅源里：https://www.v2ex.com/t/1185649

节点自身的说明（来自未登录可读的 /api/nodes/all.json 的 header 字段）：

- chamber：「一些不适合出现在首页上的信息，请发布到这里。如果你要给另外一位会员留言，也可以使用这个节点 @ 对方用户名。」
- flamewar（水深火热）：「这个节点的存在，只是为了将一类信息放到同一个节点。**如果一个主题里吵起来了而被移动到这里，管理员没有时间去细看里面谁对谁错**…如果一个账号持续以激烈的语言攻击其他账号，并且被管理员看到，那么会被彻底禁用。」
- sandbox：官方 [帮助页 /help/node](https://www.v2ex.com/help/node) 只说「发布到这个节点的内容不会出现在首页上」，**没有提需要登录**——帮助页 2021-10-23 最后更新，早于 2026 年 1 月的 visibility 改动，属于文档滞后。
- monthly：「本节点内的主题不会出现在首页和 RSS 中。」
- cheap（穷）：「所有和『穷』特别有关的主题归类在这个节点，并且不会上首页」。

### 2.2 受限节点实测清单（2026-10-26，无 Cookie）

| 节点 | 地址 | 匿名响应 | Location | 备注 |
|------|------|----------|----------|------|
| NSFW | /go/nsfw | 302 | /signin | 主题页 /t/1250875 → /signin?next=/t/1250875 |
| 沙盒 | /go/sandbox | 302 | /signin | 主题 /t/1251928 → /signin?next=… |
| Chamber | /go/chamber | 302 | /signin | 主题 /t/1251085 → /signin?next=… |
| 水深火热 | /go/flamewar | 302 | / | 主题 /t/1251948 → /restricted |
| 水 | /go/flood | 302 | / | |
| 每个月都会出现的那种主题 | /go/monthly | 302 | / | |
| 调查 | /go/survey | 302 | /restricted | /restricted 本身再 302 → /signin?next=%2Frestricted |
| 穷 | /go/cheap | 302 | /signin | |
| 请不要再发这样的文章 | /go/ohno | **200** | — | 反例：同为 chamber 子节点但可匿名访问 |
| 把妹 | /go/pua | 200 | — | 用户自建节点，未受限 |
| 发疯 | /go/crazy | 200 | — | 用户自建节点，未受限 |
| 不存在的节点 | /go/zzzz-no-such-node-xyz | **200** | — | 页面标题「节点未找到」，不是 302 |

结论：受限与否是**逐节点**的管理员标记（included_in_homepage=0），不是按父节点/内容类型自动继承；匿名访问时表现为三种 302 目标。实现里建议把「302 到 /signin、/、/restricted」都当作受限信号，并用「200 + 节点未找到」区分不存在的节点。

### 2.3 受限内容仍可通过官方通道匿名读取（重要）

以下均为无 Cookie 200，且内容真实（抽查标题/正文属于对应节点）：

| 通道 | 示例 | 状态 | 说明 |
|------|------|------|------|
| 节点 JSON Feed | /feed/nsfw.json、/feed/flamewar.json、/feed/flood.json、/feed/sandbox.json、/feed/chamber.json | 200 | 各 50 条；nsfw 最早 2026-01-15，flamewar 最新 2026-10-26 |
| 节点 Atom Feed | /feed/nsfw.xml | 200 | Content-Type: application/atom+xml，ETag，Cache-Control: public, max-age=600 |
| 旧 API 1.0 | /api/topics/show.json?node_name=nsfw（以及 flamewar、sandbox） | 200 | 各返回 10 条，含 title/content/content_rendered |
| 站点级 Atom | /index.xml、/read.xml | 200 | 实测不含受限节点主题（250 条受限 feed ID 与两者交集为 0） |
| /recent | /recent | 200 | 同上，交集为 0 |

**对设计的直接影响**：RESEARCH.md 第 50 行「这是绕过站点登录墙，本项目主动不使用」需要升级为「显式过滤」。建议：维护受限节点清单（当前 8 个）+ 运行时 302 探测 + 解析 /feed 与旧 API 时按 node.name 二次过滤；否则同步逻辑一旦抓了节点 feed/旧 API，就在事实上绕过了登录墙。

### 2.4 官方 API 2.0

文档：[https://www.v2ex.com/help/api](https://www.v2ex.com/help/api)，页面原文「本页最后更新时间 2026-08-03 12:55:25 +0000」。

原文要点：

> API 2.0 Beta 是我们正在持续更新中的新接口，会提供一系列通过 Personal Access Token 访问 V2EX 功能的新方式。
>
> Personal Access Token 可以在 Authorization header 中使用，例子如下：Authorization: Bearer bd1f2c67-…
>
> 所有 2.0 的 RESTful 的 API 接口都位于下面的这个前缀下：https://www.v2ex.com/api/v2/
>
> 默认情况下，每个 IP 每小时可以发起的 API 请求数被限制在 600 次。你可以在 API 返回结果的 HTTP 头部找到 Rate Limit 信息：
> X-Rate-Limit-Limit: 120
> X-Rate-Limit-Reset: 1409479200
> X-Rate-Limit-Remaining: 116
>
> 对于能够被 CDN 缓存的 API 请求，只有第一次请求时，才会消耗 Rate Limit 配额。

（注意：文档示例里的 120 是旧示例值；正文和实测都是 600。）

端点表（文档原文，已补全）：

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | /api/v2/notifications | 最新提醒（p 分页） |
| DELETE | /api/v2/notifications/:id | 删除提醒 |
| GET | /api/v2/member | 自己的 Profile |
| GET | /api/v2/token | 当前令牌 |
| POST | /api/v2/tokens | 创建新令牌（scope: everything/regular；expiration: 2592000/5184000/7776000/15552000 秒；最多 10 个） |
| GET | /api/v2/nodes/:node_name | 节点 |
| GET | /api/v2/nodes/:node_name/topics?p=N | 节点主题 |
| GET | /api/v2/topics/:topic_id | 主题 |
| GET | /api/v2/topics/:topic_id/replies?p=N | 回复 |
| POST | /api/v2/topics/:topic_id/set-sticky?duration=15min/1hr/8hr | 置顶自己的主题 |
| POST | /api/v2/topics/:topic_id/boost | 放首页（费用/资格见文档） |

令牌相关：

- 令牌创建/管理页：https://www.v2ex.com/settings/tokens（匿名 302 → /signin?next=%2Fsettings%2Ftokens）。
- [帮助页 /help/personal-access-token](https://www.v2ex.com/help/personal-access-token)：「Personal Access Token 只会在创建之后的 10 分钟内可以被完整地显示或者下载为 token.txt 文件」「你可以创建至多 180 天有效期的令牌」。
- 过期错误示例：{"success": false, "message": "Token expired"}。

实测：

- 无 token：GET /api/v2/nodes/all → 401，{"message": "Token not found", "success": false}，且响应头带 x-rate-limit-limit: 600、x-rate-limit-remaining、x-rate-limit-reset。
- 假 token：同样 401（53 字节）。
- /api/v2/topics/1、/api/v2/nodes/tech/topics?p=1 均 401 → 2.0 的读接口也强制 token，不存在匿名读。

### 2.5 旧 API 1.0（仍在线，文档停在 2014）

- 官方文档页：[https://www.v2ex.com/p/7v9TEc53](https://www.v2ex.com/p/7v9TEc53)（Livid，2014-08-28，2,882,065 views）。文档记录端点：/api/topics/hot.json（10 条）、/api/topics/latest.json、/api/nodes/show.json?name=、/api/members/show.json?username=/id=。
- 文档也写限流 120 次/IP/小时；实测响应头 600，与 /help/api 的 600 一致。**旧文档的 120 不要写进实现默认值**。
- 文档承诺：「V2EX 不会删除或是更改已经发布的 API 接口的 URI 及字段名」。
- 文档的「API 公平使用规则」原文：

> 我们鼓励将 V2EX API 用于学术研究、手机应用及浏览器扩展
> 我们反对将 API 输出的结果用于填充你的商业或是个人网站的内容。……只是不希望 V2EX 的数据被用在垃圾站和 content farm

- 未列入旧文档但实测可用（匿名 200）：
  - /api/topics/latest.json（本次 37 条，字段：node、member、last_reply_by、last_touched、title、url、created、deleted、content、content_rendered、last_modified、replies、id；按 created 倒序；**不含分页**，p=2 与 p=3 返回同一批）
  - /api/topics/hot.json（10 条，按「热」排序，不按 created）
  - /api/topics/show.json?id=1 或 ?node_name=tech|nsfw|flamewar|sandbox（10 条/节点）
  - /api/nodes/all.json（1379 个节点，含 parent_node_name/header/footer/topics）
  - /api/site/info.json、/api/site/stats.json（topic_max=1251977、member_max=858623）
- /api 入口：302 → /go/api（讨论节点，页面右侧有 RSS/JSON Feed 订阅说明），**不是** API 文档。
- /api/topics/latest.json 的 Cache-Control: public, max-age=432000（5 天）、cf-cache-status: HIT，数据可能落后于站点最新主题（实测 latest.json 首条 1251969，站点时点最新已到 1251975），只适合当快照。

### 2.6 公开 Feed

节点级 Feed 是页面官方提供的（/go/<node> 右侧「节点订阅方式」直接给 /feed/<node>.xml 与 /feed/<node>.json），不是隐藏接口。

JSON Feed（/feed/<node>.json）：

- 顶层：version=https://jsonfeed.org/version/1、title、description、home_page_url、feed_url、icon、favicon、items。
- item：author（{url,name,avatar}）、url、title、id、date_published、content_html。
- 实测 50 条固定；date_published 倒序；**date_published 与 topic.created 完全相等**（抽查 t/1251278：feed 2026-10-23T07:17:21+00:00 = created 1792739841）。
- 覆盖跨度取决于节点活跃度：/feed/tech.json 481 天；/feed/api.json 1664 天；/feed/flamewar.json 12 天；/feed/nsfw.json 280 天。
- 响应头：ETag、Cache-Control: public, max-age=1800、cdn-cache-control: public, max-age=120；If-None-Match 复现 304（tech 与 nsfw 均 304）。

站点 Atom（/index.xml、/read.xml）：

- 各 50 条，entry 字段：title、link（…/t/<id>#replyN）、id（tag:…）、published、updated、author/name、content（CDATA 的正文 HTML）。
- published 倒序；updated 是最后回复时间（与 published 不同）。
- 实测两者高度重合：50 条里 49 条相同，只是抓取时刻不同；**不要把它们当成两套不同数据源**。
- 响应头：ETag、Cache-Control: public, max-age=300、cdn-cache-control: public, max-age=60。
- 条件请求：ETag 存在，但内容每分钟都在变，本次两次 2 秒间隔的请求拿到不同 ETag（200），未复现 304；JSON feed 的 304 已复现，说明机制没问题，是更新频率问题。

### 2.7 robots.txt 与抓取礼仪

[https://www.v2ex.com/robots.txt](https://www.v2ex.com/robots.txt) 原文（209 字节，一字不差）：

    User-agent: bingbot
    Crawl-delay: 120
    Disallow: /backstage/
    Disallow: /signin
    Disallow: /signout
    Disallow: /settings

    User-agent: *
    Disallow: /backstage/
    Disallow: /signin
    Disallow: /signout
    Disallow: /settings

解读：内容页面（/t/、/go/、/feed/、/api/）不在 Disallow 内；但 robots 只是爬虫协议，不等于官方授权。官方对批量使用最直接的表态是旧 API 文档的「API 公平使用规则」（见 2.5）。

反爬/封禁的公开证据：

- [t/247459](https://www.v2ex.com/t/247459)（2015-12-31）楼主多线程爬虫被自动封 IP，回复：「后台程序自动封的，访问频率过高自动屏蔽对应 IP」「你的 IP 被自动封禁了 我们的运作并没有受到影响 我们甚至根本就没关注到你的行为 你就被系统干掉了」。
- [oldshensheep/v2ex_scrapy](https://github.com/oldshensheep/v2ex_scrapy) README（2023）：「爬取花了几十小时，因为爬快了会封禁 IP，并且我也没使用代理池。并发数设置为 3 基本上可以一直爬。」「服务器可能返回 404/403/302/200，如果是 404 说明帖子被删除了，如果是 403 说明是爬虫被限制了，302 一般是跳转到登陆页面，有的也是跳转到主页，200 返回正常页面。」——与本次实测的 302 目标完全吻合。
- 响应头可见 Cloudflare（Server: cloudflare、CF-RAY、cf-cache-status），HTML 页面还有 V2EX 自己的缓存头（如 v2ex-cache-recent: HIT）。
- 未证实：RSSHub 因 Cloudflare WAF 对机房 IP 返回 403 的博客（检索到摘要，原文未打开，见附录 B）。

### 2.8 HTML 页面结构实测

- 主题页 /t/1：第 1 页 100 层（1–100），第 2 页 78 层（101–178）；?p=999 被夹到最后一页；分页用 link rel="next" href="/t/1?p=2"；页面内嵌 JSON-LD（DiscussionForumPosting）含 commentCount=178、datePublished=2010-04-25T14:56:27Z、interactionStatistic（ViewAction 102368 / LikeAction 71 / ReplyAction 178）。
- 列表页 /recent：单页 22 行，结构 div.cell.item → a.topic-link id=topic-link-<id>，时间在 span[title] 里（标题属性是绝对时间 +08:00，显示为「2h 41m ago」）；页码上限 41731；显示时间是「最后活动时间」，不是创建时间。
- 节点页 /go/tech：20 条/页、按最后回复排序、页脚「Topics 581 to 600 of 596」、最多 30 页（p=31 与 p=30 同一批主题）。
- 错误页：/go/不存在 → 200「节点未找到」；/t/不存在 → 404「Topic Not Found」+ JS 跳 /404。
- 页面版本：VERSION 3.9.8.5（页脚）。

### 2.9 其它入口（是否有别的公开接口）

| 入口 | 实测 | 说明 |
|------|------|------|
| /feed/<node>.json / .xml | 200 | 官方节点订阅方式，见 2.6 |
| /index.xml /read.xml | 200 | 站点级 Atom |
| /api/v2/* | 401 | 需 token |
| /api/topics/*、/api/nodes/*、/api/members/*、/api/site/* | 200 | 旧 API 1.0，匿名 |
| /amp/t/<id> | 200 | AMP 版本（如 /amp/t/1） |
| 页面内嵌 JSON-LD | 有 | 主题正文/评论/计数，可省去部分 HTML 解析 |
| /sitemap.xml | 404 | 首次 522（Cloudflare 源站错误），复测 404 |
| /feed/all.json、/feed/index.json | 404 | 无全站 JSON feed |
| m.v2ex.com | TLS 握手失败 | 本次网络下不可用，不能据此断定站点不存在 |
| /search?q=… | 302 → /go/search | 匿名没有站内搜索页，/go/search 是节点 |

### 2.10 版权与法律边界（只引资料，不作法律意见）

V2EX 侧：

- [About 页](https://www.v2ex.com/about)：「这里绝对不会全文转载任何文章，而只会以链接方式分享」「这里尊重原创」。
- 未找到公开的服务条款/EULA：/terms 302 → /d/terms（英汉词典页），/privacy 是 Google Sign-In 隐私政策（2026-08-28 生效）。[t/30248](https://www.v2ex.com/t/30248)（2012）讨论过加 EULA，Livid 回复「我赞同你的考虑…能否给我一个你比较认可的 EULA 作为范例呢？」——之后未见平台许可协议。
- 结论：帖子/回复的著作权默认归作者（《著作权法》第 11 条），平台未授予再分发许可。个人本地缓存阅读与「公开发布/镜像/二次分发」是两回事。
- Feed 的 Atom 里有 rights 元素「Copyright © 2010-2018, V2EX」，这是 V2EX 对 feed 本身的版权声明，不等于对用户内容的授权。

法律资料（中国法语境）：

- 《著作权法》（2020 修正）[gov.cn 全文](https://www.gov.cn/guoqing/2021-10/29/content_5647633.htm)：第 10 条第（十二）项「信息网络传播权，即以有线或者无线方式向公众提供，使公众可以在其选定的时间和地点获得作品的权利」；第 11 条「著作权属于作者」；第 24 条第（一）项「为个人学习、研究或者欣赏，使用他人已经发表的作品」可不经许可、不付报酬（需指明作者/作品名，且不影响正常使用、不损害合法权益）。
- 《刑法》第 285 条（非法获取计算机信息系统数据、非法控制计算机信息系统罪）原文可见 [青海司法行政网](https://qh.12348.gov.cn/pub/qhpfw/sfxzyw/jrbb/202302/t20230221_100555.html)：「违反国家规定，侵入前款规定以外的计算机信息系统或者采用其他技术手段，获取该计算机信息系统中存储、处理或者传输的数据，或者对该计算机信息系统实施非法控制，情节严重的，处三年以下有期徒刑或者拘役…」。
- [检察日报（最高检网站转载，2025-06-03）](https://www.spp.gov.cn/spp/llyj/202506/t20250603_697192.shtml)：对「AI 平台破解 App 加密接口、伪造请求获取非公开数据」案，检察机关明确「**行为人单纯违反 robots 协议、仅爬取公开数据不宜入罪**」，但「通过技术手段破解加密算法，突破计算机信息系统安全防护措施…获取了非公开数据」构成非法获取计算机信息系统数据罪；2024 年判决：单位罚金 20 万元，6 名被告有期徒刑三年至一年九个月、缓刑五年至二年。
- [北京知识产权法院 兰国红《爬取数据，存在哪些法律风险》（2023-03）](https://bjzcfy.bjcourt.gov.cn/article/detail/2023/03/id/7214062.shtml)：大众点评诉百度案、鹰击系统爬微博案；「绕过被爬取对象设定的访问权限或者破坏网站所设定的技术措施爬取非公开数据的行为，会干扰被爬取方的正常运行，给被爬取方造成损害，也属于不正当竞争行为」；「如果爬取的内容达到实质替代数据来源网站的程度，则是一种『搭便车』行为」。
- [最高人民法院网 酷米客案（2019-12-04）](http://wlf.court.gov.cn/news/view-13.html)：爬虫盗用实时公交数据用于同类 App，构成不正当竞争，判赔 50 万元。
- [《厘定边界合理规制网络爬虫行为》（检察日报，2025-11-29 转载）](https://finance.sina.com.cn/jjxw/2025-11-29/doc-infyzkhq6707232.shtml)：讨论「形式+实质」的刑事违法性判断，认为 robots 协议的法律效力有争议，主张以技术障碍标准判断。

把这些放到本项目：**只读公开页面 → 本地缓存 → 个人阅读**，风险最低；**登录/带 Cookie 抓取、绕过 302 登录墙、批量抓取用户主页**，风险明显上升（对应 285 条与不正当竞争的讨论）；**公开部署/提供镜像或再分发正文**，还叠加著作权/信息网络传播权问题。项目当前「不登录、不抓受限节点、不并发轰炸、不提供导出再分发」的方向与这些资料的边界一致，但需要在代码层落实（见 2.3）。

## 3. 限制与风险补充（实现相关）

1. **feed 窗口会丢主题**：站点级 /index.xml 只有最新 50 条。实测 50 条只覆盖约 2.7 小时（06:11–08:55 UTC 的发帖窗口）；离线超过这个窗口，没被节点 feed 覆盖的主题就永久缺失。建议：轮询 /index.xml + /read.xml（两者去重后实际按一份处理）+ 已关注节点 feed；把「归档可能不完整」做成可见状态，而不是假装全量。
2. **无「按创建时间翻页」的历史列表**：/recent 按最后回复（bump）排序、41731 页；节点页最多 30 页（≈600 条）；feed 固定 50 条。结论与 RESEARCH.md 一致：历史只能靠本地长期归档，或登录后用 API 2.0 的节点分页（p 参数）慢慢补，但 token 限 600/h，且 API 2.0 是否也有 30 页上限未验证（见附录 B）。
3. **302/404/403/520 语义**：403 多为被反爬限制；302 可能是登录页、主页或 /restricted（受限）；200 也可能是「节点未找到」。解析器要区分这些，不能把 302 一律当登录。
4. **ETag 缓存**：JSON feed 的 ETag 稳定、304 可用；Atom 更新快，304 命中率取决于轮询间隔。建议轮询间隔 ≥5 分钟（Atom max-age=300），节点 JSON feed 可 30 分钟（max-age=1800）。
5. **受限节点过滤必须在数据层**：见 2.3。建议维护 node 名单（nsfw、chamber、sandbox、flamewar、flood、monthly、survey、cheap），并结合 /api/nodes/all.json 的 parent_node_name（chamber 子节点里除 ohno 外均受限）做兜底，但**不要只靠父节点推断**（ohno 是反例）。
6. **旧 API 的稳定性未知**：文档承诺不改 URI/字段，但 2014 年后未更新；/api/topics/latest.json 有 5 天 CDN 缓存、数据滞后。可以当补充/兜底，不宜作为主数据源。
7. **Cloudflare**：Server: cloudflare；机器 IP/高频访问可能被 403 或 520；需要在 UI 上给出「被限制/稍后重试」的明确提示，并指数退避。
8. **图片**：外链图床多样（raw.githubusercontent.com、i.v2ex.co、i.imgur.com、腾讯云 COS、个人域名等），只验证了两个域名无热链限制；实现应按「可能失败」处理（onerror 占位），并设 referrerpolicy=no-referrer。
9. **版权**：本地缓存可以，公开部署要谨慎；不要在 repo 里提交抓取到的正文/数据库；README 写明用途与非官方。
10. **法律**：不要把「robots 未禁止」当作「允许」；官方 API 公平使用规则明确反对把 API 输出填充商业/个人网站内容，公开服务形态尤其要注意。

## 4. 先例项目列表

GitHub 搜索（2026-10-26，GitHub Search API，关键词 v2ex reader/client/api/name/description/topic，去重 114 个仓库），按 star 取相关项目：

| 项目 | 语言 | ★ | 最近推送 | 可借鉴 / 需注意 |
|------|------|---|----------|-----------------|
| [livid/v2ex-gae](https://github.com/livid/v2ex-gae) | Python | 3082 | 2017-10（已归档） | 官方旧站源码（Project Babel 2/GAE）。可借鉴数据模型与权限语义；不能直接复用（Python 2/GAE）。注意 About 页链接的 livid/v2ex 仓库已 404 |
| [singro/v2ex](https://github.com/singro/v2ex) | Objective-C | 1946 | 2018-02 | 老牌 iOS 客户端。API 1.0 时代的写法；接口/展示已过时，只作 UI 参考 |
| [Finb/V2ex-Swift](https://github.com/Finb/V2ex-Swift) | Swift | 1558 | 2026-09 | 仍在维护的 iOS 客户端。看它如何做主题/回复/节点缓存与登录态；注意它可能仍依赖旧 API |
| [coolpace/V2EX_Polish](https://github.com/coolpace/V2EX_Polish) | TypeScript | 1216 | 2026-10 | 浏览器扩展，直接增强原站 DOM。借鉴：不建爬虫、成本最低；避免：只在浏览器里可用，无法离线归档 |
| [guozhigq/flutter_v2ex](https://github.com/guozhigq/flutter_v2ex) | Dart | 1023 | 2026-10 | Flutter 客户端（Android/iOS）。移动端交互与 Material You 参考 |
| [liaoliao666/v2ex](https://github.com/liaoliao666/v2ex) | TypeScript | 798 | 2026-09 | Expo/React Native 原生客户端。移动端信息架构参考 |
| [sciooga/v2ex-plus](https://github.com/sciooga/v2ex-plus) | JavaScript | 717 | 2026-05 | 老牌浏览器扩展。DOM 侧增强思路 |
| [isaced/V2exOS](https://github.com/isaced/V2exOS) | Swift | 683 | 2026-10 | SwiftUI 多端（macOS/iOS/tvOS）。离线/缓存与多端同步参考 |
| [gexiao/sov2ex](https://github.com/gexiao/sov2ex) | Go | 503 | 2025-11 | V2EX 站内搜索引擎，自建索引、Go 实现。借鉴：索引/查询设计；避免：全站抓取/索引的合规风险，本项目不做全站索引 |
| [djyde/V2EX-API](https://github.com/djyde/V2EX-API) | — | 375 | 2017-04 | 非官方 API 列表（旧接口字段说明）。查历史接口很有用；内容已旧 |
| [funnyzak/react-native-v2ex](https://github.com/funnyzak/react-native-v2ex) | TypeScript | 295 | 2026-10 | RN 客户端，「数据完全基于 V2EX 开放 API」。可看 API 封装 |
| [oldshensheep/v2ex_scrapy](https://github.com/oldshensheep/v2ex_scrapy) | Python | 278 | 2023-07 | Scrapy 全站爬虫 + 3.7GB SQLite。README 明确「不建议自行运行」「爬快了会封 IP、并发 3 可长期爬」「部分帖子和信息需登录（Cookie）」「包含水深火热的帖子」。借鉴：404/403/302/200 语义、表结构；避免：全站批量 |
| [DIYgod/RSSHub](https://github.com/DIYgod/RSSHub) | TypeScript | 46669 | 2026-10 | /v2ex/topics/:type 路由直接调 /api/topics/{hot,latest}.json。借鉴：轻量聚合；注意：官方 API 公平使用规则反对把 API 输出填充个人网站内容，公开部署尤其要谨慎 |
| [kaolengmian7/v2ex-tui](https://github.com/kaolengmian7/v2ex-tui) | Go | 247 | 2025-01 | Go 终端客户端。Go 网络/解析/键盘交互参考（与本项目技术栈最接近） |
| [six-ddc/v2ex-tui](https://github.com/six-ddc/v2ex-tui) | Go | 175 | 2026-01 | 另一个 Go TUI 客户端，结构更简单，适合快速读源码 |

补充：v2ex/launcher 不是 V2EX 客户端（同名组织下的 macOS 工具），不要误列；V2EX 官方没有公开的移动客户端源码（About 页给的是旧站 Project Babel 2）。

## 5. 对 docs/RESEARCH.md 的具体修改建议

1. 行 12（结论表第 2 行）：把「受限节点（实测 nsfw、sandbox、chamber）」改为「nsfw、sandbox、chamber、cheap → /signin；flamewar、flood、monthly → /；survey → /restricted」。
2. 行 14（结论表第 4 行）：补「旧 API 1.0 文档写 120 次/小时（2014 年，已过时），实际响应头 600；/help/api（2.0）写 600」。
3. 行 15（结论表第 5 行）：把「feed 按创建时间倒序给最新 50 条」补成「feed 只保留最近 50 条，无时间窗口；跨度取决于节点活跃度（tech 481 天、api 节点 1664 天）」。
4. 行 48–49：补全受限节点清单与三种 302 目标；加一句「/go/不存在的节点返回 200 节点未找到，不是 302」。
5. 行 50：把「本项目主动不使用」升级为「解析层显式过滤」：/feed/<node>.json、/feed/<node>.xml、/api/topics/show.json?node_name= 对受限节点均匿名 200，仅靠不抓页面不能避免绕墙。
6. 行 53–55：核对无误；建议引 t/1185368 的 #26/#28/#34 作为补充（管理员如何判定「成人/灌水节点」）。
7. 行 62–63：补「Cloudflare 在前；403 表示被反爬限制（v2ex_scrapy README）；HTML 页面无公开限额，但有自动封 IP 实践（t/247459）」；把「600 次」注明出自 /help/api（2.0）。
8. 行 65–66：robots 描述正确；建议补一句「robots 未禁止内容页 ≠ 官方授权，官方 API 公平使用规则反对把数据用于填充商业/个人网站」。
9. 行 67：版权建议改为「帖子/回复版权默认归作者（著作权法第 11 条）；未找到 V2EX 公开服务条款或内容许可（/terms 重定向到词典页），因此不能假定平台许可再分发；About 页明确『绝对不会全文转载任何文章』」。
10. 行 68：法律风险已有讨论，建议引两条最贴近的公开资料：检察日报 2025-06-03（区分「仅违反 robots + 公开数据」与「破解加密/非公开数据」）、北京知产法院 2023-03（绕过访问权限/技术措施抓非公开数据＝不正当竞争）；并注明「不构成法律意见」。
11. 行 84–87：API 2.0 端点表补全（notifications、member、token、POST tokens、set-sticky、boost）；注明读接口也需 token（匿名与假 token 均 401）；token 创建页 /settings/tokens。
12. 行 91–96：把「最早可到 477 天前」改成「样本：tech 481 天；冷门节点可达数年（api 节点 1664 天）」；补「/index.xml 与 /read.xml 高度重合（50 条中 49 条相同），只按一份处理」；补 Atom ETag 存在但因内容每分钟变化，本次未复现 304。
13. 行 100–102：把「/api 文档入口已重定向到 API 2.0」改为「/api 302 → /go/api（讨论节点）；API 2.0 文档是 /help/api；旧 API 文档是 /p/7v9TEc53」；补「p 被忽略」只对 latest/index/feed 验证；补 hot.json 不按时间排序；补 site/info、site/stats、nodes/all、topics/show 可用。
14. 行 130–133：补「/recent 22 条/页、约 41731 页」「显示时间是最后活动时间，但推广位首行可能不一致（1249854）」；节点页 30 页上限可引 p=31==p=30。
15. 行 148–149：把「带/不带 Referer 都 200，没有热链限制」限定为「实测样本：i.v2ex.co、raw.githubusercontent.com」。
16. 行 152–153：「/read.xml 有 YouTube iframe（embedded_video_wrapper）」成立；「B 站等多数是可点击链接」未验证，建议删或标未证实。
17. 行 163–164：性能数字标注「预估，未实测」，避免和已核实事实混在一起。
18. 行 186–189：受限节点清单更新 + 运行时探测策略（302 目标三类 + 200 节点未找到）+ 用 /api/nodes/all.json 缓存节点元数据。
19. 建议新增一小节「受限节点与内容过滤」：名单、302 识别、feed/旧 API 过滤、图片外链过滤。
20. 建议在附录补：所有实测的 UA、请求间隔、受限节点响应表，以及 403/520 的处理约定。

## 附录 A：关键请求记录（2026-10-26，UTC+8，匿名、无 Cookie）

| 时间窗口 | 请求 | 状态 | 关键结果 |
|----------|------|------|----------|
| 16:59–17:00 | /t/1185649、/help/api、/robots.txt、/feed/tech.json、/index.xml、/read.xml、/api/topics/latest.json、/api/v2/nodes/all、/go/{nsfw,chamber,sandbox,tech}、/api、/help/faq、/feed/nsfw.json、/recent、/about | 200/302/401 | 见正文 |
| 17:02–17:03 | feed 与 index 条件请求、/api/v2/topics/1、/api/v2/nodes/tech/topics、latest?p=2/3、topics/show、nodes/all、/planes、/sitemap.xml、/feed/all.json、/go/api、/t/1、/t/1?p=2、/t/1250875 | 200/304/401/404/522 | feed 304；sitemap 522 |
| 17:04–17:05 | /、/faq、/privacy、/help、/go/tech?p=30/31、/feed/tech.xml、/feed/{sandbox,chamber}.json、/t/1251969、带假 token 的 /api/v2/nodes/all | 200/401 | p31==p30 |
| 17:05–17:07 | /help/personal-access-token、/help/node、/t/1185368、/go/flamewar、/api/topics/show?id=1251929/1249854、/settings/tokens、/feed/api.json | 200/302 | flamewar → / |
| 17:07–17:09 | /go/{flood,survey,monthly,ohno,cheap}、/feed/flamewar.json、/t/1?p=999、/index.xml?p=2、/feed/tech.json?p=2、/t/1251928、/t/1251085 | 200/302 | flood/survey/monthly/cheap 受限；ohno 200 |
| 17:09 | /restricted、/go/zzzz-no-such-node-xyz、/t/99999999、/search?q=爬虫 | 302/200/404 | 见 2.2/2.9 |
| 17:10–17:12 | /t/247459、/t/30248、/go/guide、/go/v2ex、/api/topics/hot.json、/go/{pua,celibacy,crazy} | 200/302（个别超时） | 反爬/版权讨论 |
| 17:13–17:15 | /d/terms、/d/privacy、/go/guide（重试）、/go/v2ex（重试）、/t/752745、/api/topics/show?node_name=tech | 200/302 | 无公开 ToS |
| 17:16–17:20 | /feed/nsfw.xml、/sitemap.xml（复测 404）、/t/1251948、/t/1251957、/amp/t/1、/api/site/info.json、/api/site/stats.json、/api/nodes/show.json?name=tech、/api/members/show.json?username=Livid | 200/302/404 | flamewar 主题 → /restricted |
| 17:21–17:26 | /api/topics/show.json?id=1251278、图片带/不带 Referer 测试（i.v2ex.co、raw.githubusercontent.com） | 200 | 图片均 200 |

（另有 GitHub Search API 与法律网站访问，未计入 v2ex.com 请求；对 v2ex.com 全程无登录、无 Cookie、未触发 403/520。）

## 附录 B：未证实 / 待确认清单

1. **「无官方 SLA」**：未见 SLA 或可用性承诺，也未见明确否认；建议写成「未见公开 SLA」。
2. **Atom 条件请求 304**：/index.xml、/read.xml 的 ETag 存在，但因内容每分钟变化，本次未复现 304；JSON feed 的 304 已复现。
3. **feed 的 content_html 是否永远是完整首帖**：抽样是完整正文；超长帖是否截断未验证。
4. **图片热链**：只验证 i.v2ex.co 与 raw.githubusercontent.com；其他图床（imgur、腾讯云 COS、个人域名）未逐一验证。
5. **B 站等视频「多数是可点击链接」**：本次未抽样统计，未证实。
6. **RSSHub 因 Cloudflare WAF 对机房 IP 返回 403**：只检索到博客摘要（blog.hoopan.net/853.html），原文未能打开，未证实。
7. **m.v2ex.com 不存在/不可用**：本次 TLS 握手失败（schannel），可能只是网络问题，未证实。
8. **API 2.0 节点主题的分页上限**：/help/api 只写 p 参数，未验证是否有 30 页上限（HTML 节点页是 30 页）。
9. **受限节点完整名单**：本次确认 8 个 + 反例 ohno；站点不公开 included_in_homepage，无法穷举（1379 个节点无法逐个探测，且不应为此扫描全站）。实现上以运行时探测 + 名单兜底。
10. **/recent 页码上限 41731 与每页 22 条的换算不一致**：仅记录现象，未深究原因，不建议依赖页码推算总量。
11. **livid/v2ex 源码仓库**：About 页链接现在 404（livid/v2ex-gae 已归档），无法核对最新服务端实现。
