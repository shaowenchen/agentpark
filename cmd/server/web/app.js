const KEY_STORE = "agentpark_api_key";

const app = document.getElementById("app");

function headersJSON() {
  const h = { "Content-Type": "application/json" };
  const k = sessionStorage.getItem(KEY_STORE)?.trim();
  if (k) h.Authorization = `Bearer ${k}`;
  return h;
}

function getKey() {
  return sessionStorage.getItem(KEY_STORE)?.trim() || "";
}

function setHash(path) {
  location.hash = path.startsWith("#") ? path : "#" + path;
}

function parseRoute() {
  const raw = (location.hash || "#/").replace(/^#/, "") || "/";
  const segs = raw.split("/").filter(Boolean);
  if (segs.length === 0 || segs[0] === "home") return { view: "home" };
  if (segs[0] === "store") return { view: "store" };
  if (segs[0] === "register") return { view: "register" };
  if (segs[0] === "manage") return { view: "manage" };
  if (segs[0] === "agent" && segs[1]) return { view: "agent", id: decodeURIComponent(segs[1]) };
  return { view: "home" };
}

function esc(s) {
  const d = document.createElement("div");
  d.textContent = s ?? "";
  return d.innerHTML;
}

function originEmoji(origin) {
  if (origin === "openclaw") return "🦞";
  if (origin === "hermes") return "⚡";
  return "◆";
}

async function fetchCatalog() {
  const r = await fetch("/api/v1/catalog/agents");
  if (!r.ok) return [];
  return r.json();
}

async function fetchAgentDetail(id) {
  let r = await fetch(`/api/v1/public/catalog/agents/${encodeURIComponent(id)}`);
  if (r.ok) return { agent: await r.json(), from: "catalog" };
  r = await fetch(`/api/v1/agents/${encodeURIComponent(id)}`, { headers: headersJSON() });
  if (r.ok) return { agent: await r.json(), from: "mine" };
  return null;
}

function shell(route, innerHTML) {
  const tabHome = route.view === "home" ? "is-active" : "";
  const tabStore = route.view === "store" ? "is-active" : "";
  return `
    <header class="mega">
      <div class="mega-inner">
        <p class="mega-kicker">Agent 目录与备份</p>
        <h1 class="mega-title">AgentPark</h1>
        <p class="mega-lead">把 OpenClaw、Hermes 等 Agent 快照同步到云端目录；支持分享链接与一键恢复到本地。</p>
        <div class="mega-cta">
          <button type="button" class="btn btn-primary btn-lg" id="ctaBackup">开始备份</button>
          <button type="button" class="btn btn-secondary btn-lg" id="ctaBrowse">浏览商店</button>
        </div>
      </div>
    </header>
    <div class="toplinks">
      <a href="#/register">注册密钥</a>
      <a href="#/manage">我的 Agents</a>
    </div>
    <nav class="tabs" aria-label="主导览">
      <button type="button" class="tab ${tabHome}" data-tab="home">首页</button>
      <button type="button" class="tab ${tabStore}" data-tab="store">商店</button>
    </nav>
    <div class="shell"><div class="shell-inner">${innerHTML}</div></div>
    <footer class="store-ft">
      <code>POST /api/v1/auth/register</code> · <code>GET /api/v1/catalog/agents</code> · <code>POST /api/v1/catalog/agents/{id}/install</code>
    </footer>
  `;
}

function bindShellNav() {
  document.getElementById("ctaBackup")?.addEventListener("click", () => {
    if (getKey()) setHash("/manage");
    else setHash("/register");
  });
  document.getElementById("ctaBrowse")?.addEventListener("click", () => setHash("/store"));
  document.querySelectorAll(".tab").forEach((btn) => {
    btn.addEventListener("click", () => {
      const t = btn.getAttribute("data-tab");
      setHash(t === "store" ? "/store" : "/home");
    });
  });
}

function renderHome(catalog) {
  const hot = catalog.slice(0, 6);
  const tiles = hot
    .map(
      (a) => `
    <button type="button" class="tile" data-agent-id="${encodeURIComponent(a.id)}">
      <span class="tile-icon">${originEmoji(a.origin)}</span>
      <span class="tile-name">${esc(a.name)}</span>
      <span class="tile-desc">${esc(a.description || a.system || "查看详情")}</span>
      <span class="tile-meta">${esc(a.origin || "generic")}</span>
    </button>`
    )
    .join("");
  return `
    <h2 class="section-title">热门 Agent</h2>
    <p class="section-desc">来自商店目录的精选示例，点击进入详情并安装到你的空间。</p>
    <div class="row-scroll">
      ${tiles || '<p class="empty" style="min-width:100%">暂无目录数据</p>'}
    </div>
  `;
}

function renderStore(catalog) {
  if (!catalog.length) return '<p class="empty">商店暂无条目</p>';
  return `
    <h2 class="section-title">全部 Agent</h2>
    <p class="section-desc">以下为商店内可浏览的 Agent，点击卡片查看详情。</p>
    <div class="grid">
      ${catalog
        .map(
          (a) => `
        <button type="button" class="tile" data-agent-id="${encodeURIComponent(a.id)}">
          <span class="tile-icon">${originEmoji(a.origin)}</span>
          <span class="tile-name">${esc(a.name)}</span>
          <span class="tile-desc">${esc(a.description || a.system || "")}</span>
          <span class="tile-meta">${esc(a.origin || "generic")} · v${a.version ?? 1}</span>
        </button>`
        )
        .join("")}
    </div>
  `;
}

function renderRegister() {
  return `
    <div class="panel">
      <h2>注册访问密钥</h2>
      <p>无需用户名与密码。点击下方按钮将生成<strong>随机 API Key</strong>，请立即复制保存；之后用该密钥管理你的 Agent 目录。</p>
      <div class="key-box" id="regKeyOut">（尚未生成）</div>
      <button type="button" class="btn btn-primary btn-block" id="btnReg">生成我的密钥</button>
      <p style="margin-top:1rem;font-size:0.82rem;color:var(--muted2)">生成后会自动写入本会话，并跳转到「我的 Agents」。</p>
    </div>
  `;
}

function renderManage() {
  return `
    <div class="split">
      <div class="panel" style="margin:0">
        <h2 style="margin-top:0">密钥</h2>
        <p style="font-size:0.88rem;color:var(--muted)">请求 API 时携带 <code>Authorization: Bearer &lt;密钥&gt;</code></p>
        <div class="field">
          <label for="apiKey">API Key</label>
          <input type="password" id="apiKey" autocomplete="off" placeholder="粘贴已有密钥" />
        </div>
        <button type="button" class="btn btn-secondary btn-block" id="btnSaveKey">保存密钥</button>
        <hr style="border:none;border-top:1px solid var(--border);margin:1.25rem 0" />
        <h2 style="margin:0 0 0.5rem">发布 Agent</h2>
        <form id="form">
          <div class="field"><label>名称</label><input name="name" required placeholder="名称" /></div>
          <div class="field"><label>来源</label>
            <select name="origin">
              <option value="generic">generic</option>
              <option value="openclaw">openclaw</option>
              <option value="hermes">hermes</option>
            </select>
          </div>
          <div class="field"><label>系统提示</label><textarea name="system" rows="3"></textarea></div>
          <div class="field"><label>外部 ID（可选）</label><input name="external_id" placeholder="同步幂等" /></div>
          <button type="submit" class="btn btn-primary btn-block">上架</button>
        </form>
      </div>
      <div>
        <h2 class="section-title">我的目录</h2>
        <p class="section-desc">备份与恢复作用于当前密钥对应的空间。</p>
        <div id="myList" class="grid manage-grid"></div>
        <div style="margin-top:1rem;display:flex;flex-wrap:wrap;gap:0.5rem">
          <button type="button" class="btn btn-secondary" id="btnBackup">导出备份</button>
          <label class="btn btn-ghost" style="cursor:pointer">
            <input type="file" id="fileRestore" accept="application/json,.json" hidden />
            从文件恢复
          </label>
        </div>
        <hr style="border:none;border-top:1px solid var(--border);margin:1.25rem 0" />
        <h2 class="section-title" style="margin:0 0 0.35rem">从分享安装</h2>
        <p class="section-desc">粘贴 token 或完整分享 URL。</p>
        <div class="field" style="display:flex;gap:0.5rem;flex-wrap:wrap;align-items:flex-end">
          <div style="flex:1;min-width:200px">
            <label for="forkInput">链接 / token</label>
            <input type="text" id="forkInput" placeholder="…/public/shares/xxx" />
          </div>
          <button type="button" class="btn btn-secondary" id="btnFork">安装</button>
        </div>
        <div class="field" style="margin-top:1rem">
          <label>粘贴备份 JSON</label>
          <textarea id="paste" rows="4" placeholder="{...}"></textarea>
        </div>
        <button type="button" class="btn btn-primary" id="btnPasteRestore">从剪贴板恢复</button>
        <pre class="log" id="log"></pre>
      </div>
    </div>
  `;
}

function logInto(el, msg) {
  if (!el) return;
  el.textContent = typeof msg === "string" ? msg : JSON.stringify(msg, null, 2);
}

async function refreshMyList(container, logEl) {
  const res = await fetch("/api/v1/agents", { headers: headersJSON() });
  if (res.status === 401 || res.status === 403) {
    container.innerHTML = '<p class="empty">请先保存有效 API Key，或完成注册。</p>';
    return;
  }
  const data = await res.json();
  if (!data.length) {
    container.innerHTML = '<p class="empty">目录为空</p>';
    return;
  }
  container.innerHTML = data
    .map(
      (a) => `
    <div class="own-card">
      <button type="button" class="own-head" data-agent-id="${encodeURIComponent(a.id)}">
        <span class="tile-icon">${originEmoji(a.origin)}</span>
        <span>
          <span class="tile-name" style="display:block">${esc(a.name)}</span>
          <span class="tile-desc" style="margin-top:0.25rem;display:block">${esc(a.system || "")}</span>
          <span class="tile-meta" style="margin-top:0.35rem;display:block">v${a.version ?? 1}</span>
        </span>
      </button>
      <div class="manage-actions">
        <button type="button" class="btn btn-secondary" data-share="${encodeURIComponent(a.id)}">分享</button>
        <button type="button" class="btn btn-danger" data-del="${encodeURIComponent(a.id)}">删除</button>
      </div>
    </div>`
    )
    .join("");

  container.querySelectorAll(".own-head").forEach((btn) => {
    btn.addEventListener("click", () => {
      const id = decodeURIComponent(btn.getAttribute("data-agent-id"));
      setHash(`/agent/${encodeURIComponent(id)}`);
    });
  });
  container.querySelectorAll("[data-del]").forEach((btn) => {
    btn.addEventListener("click", async (e) => {
      e.stopPropagation();
      const id = decodeURIComponent(btn.getAttribute("data-del"));
      await fetch(`/api/v1/agents/${encodeURIComponent(id)}`, { method: "DELETE", headers: headersJSON() });
      await refreshMyList(container, logEl);
      logInto(logEl, "已删除");
    });
  });
  container.querySelectorAll("[data-share]").forEach((btn) => {
    btn.addEventListener("click", async (e) => {
      e.stopPropagation();
      const id = decodeURIComponent(btn.getAttribute("data-share"));
      const r = await fetch(`/api/v1/agents/${encodeURIComponent(id)}/shares`, {
        method: "POST",
        headers: headersJSON(),
        body: "{}",
      });
      const out = await r.json().catch(() => ({}));
      if (out.share_url) {
        try {
          await navigator.clipboard.writeText(out.share_url);
          logInto(logEl, { ...out, note: "已复制分享链接" });
        } catch {
          logInto(logEl, out);
        }
      } else logInto(logEl, out);
    });
  });
}

function bindManage() {
  const logEl = document.getElementById("log");
  const listEl = document.getElementById("myList");
  const apiKeyEl = document.getElementById("apiKey");
  apiKeyEl.value = getKey();
  document.getElementById("btnSaveKey")?.addEventListener("click", () => {
    sessionStorage.setItem(KEY_STORE, apiKeyEl.value.trim());
    refreshMyList(listEl, logEl);
    logInto(logEl, "已保存");
  });
  document.getElementById("form")?.addEventListener("submit", async (e) => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const body = {
      name: fd.get("name")?.toString().trim(),
      system: fd.get("system")?.toString() ?? "",
      origin: fd.get("origin")?.toString() || "generic",
    };
    const ext = fd.get("external_id")?.toString().trim();
    if (ext) body.external_id = ext;
    const res = await fetch("/api/v1/agents", { method: "POST", headers: headersJSON(), body: JSON.stringify(body) });
    if (!res.ok) {
      logInto(logEl, await res.text());
      return;
    }
    e.target.reset();
    await refreshMyList(listEl, logEl);
    logInto(logEl, "已上架");
  });
  document.getElementById("btnBackup")?.addEventListener("click", async () => {
    const res = await fetch("/api/v1/backup", { headers: headersJSON() });
    const json = await res.json();
    const blob = new Blob([JSON.stringify(json, null, 2)], { type: "application/json" });
    const u = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = u;
    const ts = json.exported_at || json.created_at;
    a.download = `agentpark-${ts?.slice(0, 19).replace(/[:T]/g, "-") || "bak"}.json`;
    a.click();
    URL.revokeObjectURL(u);
    logInto(logEl, "已导出");
  });
  document.getElementById("fileRestore")?.addEventListener("change", async (ev) => {
    const f = ev.target.files?.[0];
    ev.target.value = "";
    if (!f) return;
    const text = await f.text();
    await doRestore(text, logEl, listEl);
  });
  document.getElementById("btnPasteRestore")?.addEventListener("click", () => {
    doRestore(document.getElementById("paste").value.trim(), logEl, listEl);
  });
  document.getElementById("btnFork")?.addEventListener("click", async () => {
    let raw = document.getElementById("forkInput").value.trim();
    if (!raw) return;
    const m = raw.match(/shares\/([a-f0-9]+)/i);
    if (m) raw = m[1];
    const res = await fetch("/api/v1/agents/fork", {
      method: "POST",
      headers: headersJSON(),
      body: JSON.stringify({ share_token: raw }),
    });
    const text = await res.text();
    let out;
    try {
      out = text ? JSON.parse(text) : {};
    } catch {
      out = { raw: text };
    }
    if (!res.ok) {
      logInto(logEl, out);
      return;
    }
    document.getElementById("forkInput").value = "";
    await refreshMyList(listEl, logEl);
    logInto(logEl, { installed: out });
  });
  refreshMyList(listEl, logEl);
}

async function doRestore(text, logEl, listEl) {
  let data;
  try {
    data = JSON.parse(text);
  } catch {
    logInto(logEl, "JSON 无效");
    return;
  }
  const res = await fetch("/api/v1/restore", { method: "POST", headers: headersJSON(), body: JSON.stringify(data) });
  const out = await res.json().catch(() => ({}));
  if (!res.ok) {
    logInto(logEl, out);
    return;
  }
  await refreshMyList(listEl, logEl);
  logInto(logEl, out);
}

function bindRegister() {
  document.getElementById("btnReg")?.addEventListener("click", async () => {
    const r = await fetch("/api/v1/auth/register", { method: "POST" });
    const out = await r.json().catch(() => ({}));
    const box = document.getElementById("regKeyOut");
    if (!r.ok) {
      box.textContent = JSON.stringify(out);
      return;
    }
    box.textContent = out.api_key || "";
    sessionStorage.setItem(KEY_STORE, (out.api_key || "").trim());
    try {
      await navigator.clipboard.writeText(out.api_key);
    } catch (_) {}
    setTimeout(() => setHash("/manage"), 800);
  });
}

async function renderAgentDetail(id) {
  const data = await fetchAgentDetail(id);
  if (!data) {
    return `<div class="detail"><p class="empty">未找到该 Agent，或无权访问。</p>
      <button type="button" class="btn btn-secondary" data-nav="back">返回</button></div>`;
  }
  const { agent: a, from } = data;
  const installBtn =
    from === "catalog"
      ? `<button type="button" class="btn btn-primary" id="btnInstall">安装到我的空间</button>`
      : "";
  const forkHint =
    from === "mine"
      ? `<p class="section-desc">这是当前密钥空间内的 Agent。</p>`
      : `<p class="section-desc">商店条目，安装后会复制到你的空间。</p>`;
  return `
    <div class="detail">
      <div class="detail-back"><button type="button" class="btn btn-ghost" data-nav="back">← 返回</button></div>
      <div class="detail-hero">
        <h1 class="detail-title">${esc(a.name)}</h1>
        <div class="detail-badges">
          <span class="badge">${esc(a.origin || "generic")}</span>
          <span class="badge">v${a.version ?? 1}</span>
        </div>
      </div>
      ${forkHint}
      <h3 class="section-title" style="margin-top:0.5rem">系统提示</h3>
      <div class="detail-body">${esc(a.system || "（无）")}</div>
      ${
        a.description
          ? `<h3 class="section-title" style="margin-top:1rem">简介</h3><div class="detail-body">${esc(a.description)}</div>`
          : ""
      }
      <div class="detail-actions">
        ${installBtn}
        <button type="button" class="btn btn-secondary" data-nav="store">去商店</button>
      </div>
    </div>
  `;
}

function bindAgentDetail(id, fromGuess) {
  document.getElementById("btnInstall")?.addEventListener("click", async () => {
    if (!getKey()) {
      alert("请先注册或粘贴密钥到「我的 Agents」");
      setHash("/register");
      return;
    }
    const r = await fetch(`/api/v1/catalog/agents/${encodeURIComponent(id)}/install`, {
      method: "POST",
      headers: headersJSON(),
    });
    const t = await r.text();
    let out;
    try {
      out = JSON.parse(t);
    } catch {
      out = t;
    }
    if (!r.ok) {
      alert(typeof out === "string" ? out : JSON.stringify(out));
      return;
    }
    setHash("/manage");
  });
  document.querySelectorAll("[data-nav]").forEach((b) => {
    b.addEventListener("click", () => {
      const v = b.getAttribute("data-nav");
      if (v === "back") history.back();
      else if (v === "store") setHash("/store");
    });
  });
}

function bindTileNav(root) {
  root.querySelectorAll(".tile[data-agent-id]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const id = decodeURIComponent(btn.getAttribute("data-agent-id"));
      setHash(`/agent/${encodeURIComponent(id)}`);
    });
  });
}

async function render() {
  const route = parseRoute();
  let inner = "";
  let catalog = [];

  if (route.view === "home" || route.view === "store") {
    catalog = await fetchCatalog();
  }

  if (route.view === "home") inner = renderHome(catalog);
  else if (route.view === "store") inner = renderStore(catalog);
  else if (route.view === "register") inner = renderRegister();
  else if (route.view === "manage") inner = renderManage();
  else if (route.view === "agent") inner = await renderAgentDetail(route.id);
  else inner = renderHome(catalog);

  const hideTabs = route.view === "register" || route.view === "manage";
  const html = hideTabs
    ? `
    <header class="mega">
      <div class="mega-inner">
        <p class="mega-kicker">Agent 目录与备份</p>
        <h1 class="mega-title">AgentPark</h1>
        <p class="mega-lead">把 OpenClaw、Hermes 等 Agent 快照同步到云端目录；支持分享链接与一键恢复到本地。</p>
        <div class="mega-cta">
          <button type="button" class="btn btn-primary btn-lg" id="ctaBackup">开始备份</button>
          <button type="button" class="btn btn-secondary btn-lg" id="ctaBrowse">浏览商店</button>
        </div>
      </div>
    </header>
    <div class="toplinks">
      <a href="#/">首页</a>
      <a href="#/store">商店</a>
      <a href="#/register">注册密钥</a>
      <a href="#/manage">我的 Agents</a>
    </div>
    <div class="shell"><div class="shell-inner">${inner}</div></div>
    <footer class="store-ft">
      <code>POST /api/v1/auth/register</code> · <code>GET /api/v1/catalog/agents</code>
    </footer>`
    : route.view === "agent"
      ? `
    <header class="mega">
      <div class="mega-inner">
        <p class="mega-kicker">Agent 目录与备份</p>
        <h1 class="mega-title">AgentPark</h1>
        <p class="mega-lead">把 OpenClaw、Hermes 等 Agent 快照同步到云端目录；支持分享链接与一键恢复到本地。</p>
        <div class="mega-cta">
          <button type="button" class="btn btn-primary btn-lg" id="ctaBackup">开始备份</button>
          <button type="button" class="btn btn-secondary btn-lg" id="ctaBrowse">浏览商店</button>
        </div>
      </div>
    </header>
    <div class="toplinks">
      <a href="#/">首页</a>
      <a href="#/store">商店</a>
      <a href="#/register">注册密钥</a>
      <a href="#/manage">我的 Agents</a>
    </div>
    <div class="shell"><div class="shell-inner">${inner}</div></div>
    <footer class="store-ft"><code>GET /api/v1/public/catalog/agents/{id}</code></footer>`
      : shell(route, inner);

  app.innerHTML = html;

  document.getElementById("ctaBackup")?.addEventListener("click", () => {
    if (getKey()) setHash("/manage");
    else setHash("/register");
  });
  document.getElementById("ctaBrowse")?.addEventListener("click", () => setHash("/store"));

  if (!hideTabs && route.view !== "agent") {
    bindShellNav();
    bindTileNav(app);
  }
  if (route.view === "register") {
    bindRegister();
  }
  if (route.view === "manage") bindManage();
  if (route.view === "agent") {
    bindAgentDetail(route.id);
  }
}

window.addEventListener("hashchange", () => render());
if (!location.hash) location.hash = "#/";
render();
