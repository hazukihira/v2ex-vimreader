# HTTP API 契约（v1）

服务默认监听 `http://127.0.0.1:7788`，所有接口返回 UTF-8 JSON，错误统一为
`{"error":{"code":"...","message":"..."}}`，HTTP 状态码 4xx/5xx。

时间字段一律是 Unix 秒（整数，UTC）；`createdKnown` 为 false 时表示该主题的创建时间尚未取到。
HTML 内容字段（`contentHtml`）已在服务端消毒，前端可直接插入 DOM；图片、iframe 已做白名单处理。

## 通用对象

```
Node        { name, title, titleAlt, parent, topicsCount, restricted }
TopicBrief  { id, node, nodeTitle, title, author, replies, createdAt, createdKnown,
              lastTouchedAt, pinned }
Reply       { floor, author, createdAt, contentHtml, isOp }
```

## GET /api/bootstrap

首屏数据。

```json
{
  "version": "0.1.0",
  "nodes": [{"name":"tech","title":"科技","titleAlt":"Technology","parent":"","topicsCount":596,"restricted":false}],
  "followedNodes": ["tech", "qna"],
  "config": {"aiEnabled": false, "aiModel": "", "theme": "dark", "siteTitle": "v2ex-vimreader"},
  "status": {"db": {"topics": 0, "replies": 0, "lists": 0}, "fetcher": {"requests": 0, "rateRemaining": 0, "lastError": ""}},
  "archive": {"from": 0, "to": 0},
  "login": {"hasCookie": false, "loggedIn": false, "user": "", "checkedAt": 0}
}
```

`login` 字段只表示本地是否保存了 Cookie 及其校验结果，**不会返回 Cookie 内容**。

## GET /api/list

列表页。站点列表按需抓取并落库；`new`、`day` 来自本地归档（数据库分页）。

参数：
- `kind`：`home` | `node` | `recent` | `new` | `day`
- `node`：kind=node 时必填，节点名（如 `tech`）
- `page`：从 1 开始
- `date`：kind=day 时必填，`YYYY-MM-DD`（按本地时区 UTC+8 计算自然日）
- `refresh`：`1` 时强制刷新远端页面（无视缓存 TTL）

响应：
```json
{
  "kind": "node", "node": "tech", "page": 1, "pages": 30,
  "entries": [{"id":1251278,"node":"tech","nodeTitle":"科技","title":"...","author":"x",
               "replies":6,"createdAt":0,"createdKnown":false,"lastTouchedAt":0,"pinned":false}],
  "fetchedAt": 1793000000, "stale": false,
  "notice": "节点主题需要登录后查看"
}
```

说明：
- `home`：V2EX 首页（`/?tab=all`），只有第一页。
- `node`：`/go/<node>` 第 1..30 页（站点最多 30 页）。未登录时受限节点返回 200 + `notice` + 空
  `entries`；登录后正常返回内容。名单在配置 `sync.restricted_nodes`（默认 8 个），节点目录里
  也会标记 `restricted`。
- `recent`：`/recent`，按最后回复时间排序，可分很多页。
- `new`：本地归档中按创建时间倒序，本站分页。
- `day`：本地归档中某一天创建的主题，本站分页；`archive` 字段给出覆盖范围。
- 受限节点（NSFW、Chamber 等）会返回 `notice`，`entries` 为空。

## GET /api/topic/{id}

打开主题：返回主题正文 + 某一页楼层。受限节点/主题返回 403 `login_required`。

参数：`page`（默认 1）、`refresh`（1 强制抓取远端）。
回复按 100 层/页抓取，页数由站点 JSON-LD 的 `commentCount` 推算。

```json
{
  "topic": {"id":1,"node":"babel","nodeTitle":"Project Babel","title":"...","author":"Livid",
            "createdAt":1272207387,"replies":178,"views":102364,"contentHtml":"<p>...</p>",
            "fetchedAt":1793000000,"source":"html"},
  "page": 1, "pages": 2, "pageSize": 100,
  "replies": [{"floor":1,"author":"Jay","createdAt":1272217077,"contentHtml":"...","isOp":false}],
  "summary": {"status":"none", "content":"", "model":"", "createdAt":0}
}
```

## GET /api/topic/{id}/replies?page=N

只取楼层页（同样会按需抓取并落库）。`page` 会被夹在 `[1, pages]` 内；
`isOp` 表示该楼层作者是否等于楼主。

```json
{"page":2,"pages":2,"pageSize":100,
 "replies":[{"floor":101,"author":"...","createdAt":0,"contentHtml":"...","isOp":false}]}
```

## POST /api/topic/{id}/refresh

强制重新抓取主题页（`{"page":1}` 可指定页），返回与 GET /api/topic/{id} 相同结构。

## GET /api/topic/{id}/summary

读取已缓存的 AI 总结：`{"status":"none|ready|running","content":"", "model":"", "createdAt":0}`

## POST /api/topic/{id}/summary

生成 AI 总结。`Accept: text/event-stream` 时以 SSE 流式返回，每行一个 JSON：

```
data: {"delta":"增量文本"}
data: {"done":true,"model":"deepseek-chat"}
data: {"error":"失败原因"}
```

否则等待完成后返回与 GET 相同的 JSON。请求体可选 `{"refresh":true}` 强制重新生成；
AI 未启用返回 501。

## GET /api/blocks

```json
{"topics":[123,456],"users":["someone"]}
```

## POST /api/blocks

`{"kind":"topic"|"user","value":123|"username"}`，成功返回 204。

## DELETE /api/blocks?kind=topic&value=123

成功返回 204。

## GET /api/nodes

节点目录（来自 /planes，首次访问时抓取并缓存）：`{"nodes":[...], "fetchedAt":0}`。

## GET /api/status

```json
{"db":{"topics":0,"replies":0,"lists":0},"fetcher":{"requests":0,"rateRemaining":0,
 "lastError":""},"sync":{"lastSiteFeed":0,"lastNodeFeeds":0,"running":false},
 "archive":{"from":0,"to":0},
 "login":{"hasCookie":false,"loggedIn":false,"user":"","checkedAt":0}}
```

## POST /api/sync

手动触发同步：`{"target":"site"|"followed"|"all"}`，立即返回 `202 {"started":true}`。

## POST /api/login

保存并校验 V2EX Cookie（从浏览器开发者工具复制 v2ex.com 的 Cookie 字符串）：

```json
{"cookie": "PB3_SESSION=...; A2=...; V2EX_LANG=zhcn"}
```

成功返回 `{"hasCookie":true,"loggedIn":true,"user":"example_user","checkedAt":1793009999}`；
无效/过期返回 400 `{"error":{"code":"login_failed","message":"cookie 无效或已过期..."}}`。
Cookie 只保存在本机 SQLite，不会通过任何接口返回。

## DELETE /api/login

退出登录：清空本地保存的 Cookie，返回 `{"hasCookie":false,"loggedIn":false,...}`。

## POST /api/login/chrome

从本机正在运行的 Chrome（DevTools 协议）读取 v2ex.com 的 Cookie 并登录，无请求体。
仅允许本机请求。Chrome 需要开启远程调试（`chrome://inspect/#remote-debugging`），首次连接可能要在
Chrome 里点 Allow，接口最长等待约 2 分钟。

- 成功：200，返回同 POST /api/login 的 LoginStatus；
- Chrome 未开启远程调试 / 连接失败：502 `{"error":{"code":"chrome_unavailable","message":"..."}}`；
- Chrome 里没有 v2ex.com 登录态：502，message 提示先登录 v2ex.com；
- 非本机请求：403。

## GET /api/followed/nodes

返回侧栏当前节点：`{"followedNodes":["tech","qna"],"nodes":[{"name":"tech","title":"科技","restricted":false}]}`。
列表优先使用本机数据库里的用户自定义值，没有自定义时用 `sync.followed_nodes` 配置默认值。

## PUT /api/followed/nodes

整体替换侧栏节点：`{"nodes":["tech","qna","create"]}`（允许空数组）。成功返回与 GET 相同结构，
保存到本机 SQLite；非法/不存在的节点返回 400 `invalid_node`。保存后后台同步会按新列表抓取节点 feed。

## GET /api/search?q=...

本地全文/模糊搜索（标题、正文、楼层），用于扩展功能。
```json
{"topics":[{"id":1,"title":"...","node":"babel","author":"Livid","replies":178,"createdAt":0}]}
```

## 静态资源

- `/` → `web/index.html`
- `/css/*`、`/js/*` → 前端静态资源
- 未匹配的非 `/api/` 路径返回 index.html（前端路由）
