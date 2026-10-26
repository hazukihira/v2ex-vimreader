# 架构与核心逻辑

## 一句话

一个单进程 Go 服务：抓 V2EX 页面/feed → 解析成结构体 → 写入 SQLite → 通过 REST API 提供给一个原生 ES Module 前端；前端只负责路由、键盘和渲染。

## 目录

```text
main.go                 进程入口：flag、配置、SQLite、HTTP 服务、嵌入 web、优雅退出
config.toml             本地配置（gitignore），Cookie / AI Key 只在这里或环境变量
data/                   本地 SQLite 与归档（gitignore）
internal/config/        TOML + 环境变量 + 默认值，data_dir 解析
internal/v2ex/          HTTP 客户端、限速、条件请求、HTML/feed 解析器
internal/store/         SQLite schema 与读写：主题、楼层、列表缓存、meta
internal/service/       业务编排：列表/主题/feed 同步/AI/登录/节点管理
internal/api/           REST 路由、DTO 转换、静态文件、错误映射
internal/ai/            OpenAI Chat Completions 兼容客户端（含 SSE 流式）
internal/chromeimport/  从本机 Chrome DevTools 读取 v2ex.com Cookie
web/                    原生前端；无构建步骤
  js/app.js             路由、状态机、键盘流程、各视图装配
  js/vim.js             按键序列引擎（数字前缀、gg、组合键）
  js/render.js          DOM 渲染：列表、主题、楼层、侧栏、主题头
  js/overlays.js        帮助、命令面板、节点管理、登录、Toast
  js/api.js             fetch 封装与 mock
  js/store.js           localStorage：主题、位置、已读、过滤、侧栏宽度
  js/lightbox.js        图片灯箱
  css/app.css           全部样式与主题变量
docs/API.md             HTTP 接口说明
```

## 请求链路

1. 浏览器打开 `http://127.0.0.1:7788/`，Go 返回内嵌的 `index.html` 和静态资源。
2. 前端 `app.js` 解析 hash 路由（列表 / 节点 / 按天 / 主题），调用 `/api/*`。
3. `internal/api` 把请求交给 `internal/service`。
4. service 先查 SQLite；缓存过期、强制刷新或数据缺失时，调用 `internal/v2ex` 抓页面。
5. v2ex 解析器用 goquery 提取标题、作者、时间、回复数、浏览数、正文和楼层，写回 SQLite。
6. service 转成 DTO，API 返回 JSON；前端 render.js 重新画 DOM。

## 数据与缓存

- `topics` / `replies`：主题和楼层；回复按 100 层/页存储。
- `lists`：首页、节点、最近等列表的缓存（延迟刷新）。
- `meta`：登录 Cookie、登录用户、常用节点等键值。
- 已读状态、当前选中位置、过滤词在浏览器 localStorage，不进 SQLite。
- 主题详情缓存默认 5 分钟；列表缓存约 2 分钟；`R` 强制刷新。

## 同步

后台按 token bucket 限速（默认 1 req/s，burst 2）：

- 站点 feed：`/index.xml`、`/read.xml`；
- 节点 feed：`/feed/<node>.json`；
- 用 ETag / If-Modified-Since 做条件请求，304 不重复解析；
- 主题按 ID 去重，只补新楼层；受限节点匿名时直接跳过。

## 登录与 AI

- 登录：`POST /api/login` 校验 Cookie 后写入 meta；之后抓取请求由后端注入 Cookie。`POST /api/login/chrome` 通过本机 Chrome CDP 读取 Cookie。
- AI：`POST /api/summary` → `internal/ai` 调 OpenAI 兼容接口，支持自定义 Header 和 SSE；结果按主题缓存。

## 阅读体验的核心

- 楼层是 DOM 中的 article，`state.topicSelection` 表示当前项；`j/k` 在“当前楼层内部滚动”和“切换楼层”之间二选一。
- 页面滚动做 90–180ms 缓动；单次按键最多推进约 1/6 屏，避免长主楼/回复之间出现断裂式跳跃。
- `#focus-cursor` 是 nvim 风格的单条光标，只表示键盘焦点；楼层高亮不依赖整行底色。
- 列表/侧栏用固定列宽保证数字和节点名对齐；`w` 在 1080px 居中与铺满之间切换。

## 改代码时先看哪里

- 抓取或页面结构变了：`internal/v2ex/parse_*.go` 和 `testdata/`。
- 缓存/同步不更新：`internal/service/feeds.go`、`internal/service/lists.go`。
- 页面不显示字段：`internal/service/dto.go` → `internal/api` → `web/js/render.js`。
- 键盘行为：`web/js/vim.js` + `web/js/app.js` 的 listBindings/topicBindings。
- 样式与主题：`web/css/app.css` 顶部变量区。
