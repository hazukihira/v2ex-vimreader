package service

import "errors"

// Service level sentinel errors mapped to HTTP status codes by the API layer.
var (
	ErrBadRequest        = errors.New("bad request")
	ErrLoginRequired     = errors.New("this page requires a V2EX login")
	ErrNotFound          = errors.New("not found")
	ErrUnavailable       = errors.New("v2ex is temporarily unavailable")
	ErrLoginFailed       = errors.New("cookie 无效或已过期，请重新从浏览器复制 v2ex.com 的 Cookie")
	ErrChromeUnavailable = errors.New("无法从本机 Chrome 导入 Cookie")
	ErrInvalidNode       = errors.New("节点不存在")
)
