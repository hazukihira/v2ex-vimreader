#!/usr/bin/env node
// Import v2ex.com cookies from the running Chrome via the DevTools protocol.
//
// Requirements: Chrome started with the default profile and remote debugging
// enabled (chrome://inspect/#remote-debugging). A one-time "Allow" prompt may
// appear inside Chrome when this script connects.
//
// Usage: node scripts/import-chrome-cookie.mjs
//        VIMREADER_URL=http://127.0.0.1:7788 node scripts/import-chrome-cookie.mjs
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

const APP = process.env.VIMREADER_URL || "http://127.0.0.1:7788";

function chromeUserDataDir() {
  if (process.platform === "win32") {
    return path.join(process.env.LOCALAPPDATA || "", "Google", "Chrome", "User Data");
  }
  if (process.platform === "darwin") {
    return path.join(os.homedir(), "Library", "Application Support", "Google", "Chrome");
  }
  return path.join(os.homedir(), ".config", "google-chrome");
}

function readDevToolsPort() {
  const file = path.join(chromeUserDataDir(), "DevToolsActivePort");
  const lines = fs.readFileSync(file, "utf8").trim().split(/\r?\n/);
  if (lines.length < 2) throw new Error("DevToolsActivePort is incomplete");
  return { port: lines[0].trim(), wsPath: lines[1].trim() };
}

function cdp(ws, method, params, timeoutMs = 120000) {
  return new Promise((resolve, reject) => {
    const id = cdp.seq = (cdp.seq || 0) + 1;
    const timer = setTimeout(() => { cleanup(); reject(new Error("CDP timeout (did you click Allow in Chrome?)")); }, timeoutMs);
    function cleanup() { clearTimeout(timer); ws.removeEventListener("message", onMessage); }
    function onMessage(event) {
      let message;
      try { message = JSON.parse(event.data); } catch { return; }
      if (message.id !== id) return;
      cleanup();
      if (message.error) reject(new Error(message.error.message || "CDP error"));
      else resolve(message.result);
    }
    ws.addEventListener("message", onMessage);
    ws.send(JSON.stringify({ id, method, params }));
  });
}

function cookieHeader(cookies) {
  const byName = new Map();
  for (const cookie of cookies) {
    if (!/(^|\.)v2ex\.com$/.test(cookie.domain)) continue;
    if (!byName.has(cookie.name)) byName.set(cookie.name, cookie.value);
  }
  if (byName.size === 0) return "";
  return [...byName.entries()].map(([name, value]) => name + "=" + value).join("; ");
}

async function main() {
  const { port, wsPath } = readDevToolsPort();
  const wsUrl = "ws://127.0.0.1:" + port + wsPath;
  console.log("connecting to Chrome DevTools:", wsUrl);
  const ws = new WebSocket(wsUrl);
  await new Promise((resolve, reject) => {
    ws.addEventListener("open", resolve, { once: true });
    ws.addEventListener("error", () => reject(new Error("cannot connect to Chrome DevTools (is remote debugging enabled?)")), { once: true });
  });
  try {
    const result = await cdp(ws, "Storage.getCookies", {});
    const header = cookieHeader(result.cookies || []);
    if (!header) {
      console.error("Chrome 里没有 v2ex.com 的 Cookie，请先在 Chrome 登录 v2ex.com");
      process.exitCode = 2;
      return;
    }
    const response = await fetch(APP + "/api/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ cookie: header }),
    });
    const body = await response.text();
    if (!response.ok) {
      console.error("导入失败:", response.status, body);
      process.exitCode = 1;
      return;
    }
    const status = JSON.parse(body);
    console.log("已导入并登录:", status.user || "(未知用户)", "| 状态:", JSON.stringify(status));
  } finally {
    ws.close();
  }
}

main().catch((error) => {
  console.error(error.message);
  console.error("如果 Chrome 没有开启远程调试，请在 chrome://inspect/#remote-debugging 打开，或直接在网页登录面板手动粘贴 Cookie。");
  process.exit(1);
});
