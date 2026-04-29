const listEl = document.getElementById("list");
const form = document.getElementById("form");
const logEl = document.getElementById("log");
const btnBackup = document.getElementById("btnBackup");
const btnPasteRestore = document.getElementById("btnPasteRestore");
const fileRestore = document.getElementById("fileRestore");
const paste = document.getElementById("paste");
const apiKeyEl = document.getElementById("apiKey");
const btnSaveKey = document.getElementById("btnSaveKey");
const forkInput = document.getElementById("forkInput");
const btnFork = document.getElementById("btnFork");

const KEY_STORE = "agentpark_api_key";
apiKeyEl.value = sessionStorage.getItem(KEY_STORE) || "";

function headersJSON() {
  const h = { "Content-Type": "application/json" };
  const k = sessionStorage.getItem(KEY_STORE)?.trim();
  if (k) h.Authorization = `Bearer ${k}`;
  return h;
}

function log(msg) {
  logEl.textContent = typeof msg === "string" ? msg : JSON.stringify(msg, null, 2);
}

btnSaveKey.addEventListener("click", () => {
  sessionStorage.setItem(KEY_STORE, apiKeyEl.value.trim());
  log("已保存 API Key（仅本标签页 sessionStorage）");
  refresh().catch((e) => log(String(e)));
});

async function refresh() {
  const res = await fetch("/api/v1/agents", { headers: headersJSON() });
  if (res.status === 401 || res.status === 403) {
    log("需要 API Key：请在上方填写并保存。");
    return;
  }
  const data = await res.json();
  listEl.innerHTML = "";
  if (!data.length) {
    listEl.innerHTML = '<li class="empty">暂无 Agent，请新建、恢复备份或从分享导入。</li>';
    return;
  }
  for (const a of data) {
    const li = document.createElement("li");
    const origin = a.origin || "generic";
    const ver = a.version != null ? `v${a.version}` : "";
    const ext = a.external_id ? ` · ext:${a.external_id.slice(0, 8)}…` : "";
    li.innerHTML = `
      <div class="meta">
        <div class="name"></div>
        <div class="origin"></div>
        <div class="sys"></div>
      </div>
      <div class="actions-row">
        <button class="share" type="button" data-id="${a.id}">生成分享链接</button>
        <button class="del" type="button" data-id="${a.id}">删除</button>
      </div>
    `;
    li.querySelector(".name").textContent = `${a.name} ${ver}`;
    li.querySelector(".origin").textContent = `${origin}${ext}`;
    li.querySelector(".sys").textContent = a.system || "（无系统提示）";
    li.querySelector(".del").addEventListener("click", async () => {
      await fetch(`/api/v1/agents/${encodeURIComponent(a.id)}`, {
        method: "DELETE",
        headers: headersJSON(),
      });
      await refresh();
      log(`已删除: ${a.name}`);
    });
    li.querySelector(".share").addEventListener("click", async () => {
      const r = await fetch(`/api/v1/agents/${encodeURIComponent(a.id)}/shares`, {
        method: "POST",
        headers: headersJSON(),
        body: "{}",
      });
      const out = await r.json().catch(() => ({}));
      if (!r.ok) {
        log(out);
        return;
      }
      log(out);
      if (out.share_url) {
        try {
          await navigator.clipboard.writeText(out.share_url);
          log({ ...out, note: "分享 URL 已复制到剪贴板" });
        } catch {
          log(out);
        }
      }
    });
    listEl.appendChild(li);
  }
}

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const fd = new FormData(form);
  const name = fd.get("name")?.toString().trim();
  const system = fd.get("system")?.toString() ?? "";
  const origin = fd.get("origin")?.toString() || "generic";
  const external_id = fd.get("external_id")?.toString().trim() ?? "";
  const body = { name, system, origin };
  if (external_id) body.external_id = external_id;
  const res = await fetch("/api/v1/agents", {
    method: "POST",
    headers: headersJSON(),
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    log(await res.text());
    return;
  }
  form.reset();
  await refresh();
  log("已添加 Agent");
});

btnFork.addEventListener("click", async () => {
  let raw = forkInput.value.trim();
  if (!raw) return;
  const m = raw.match(/shares\/([a-f0-9]+)/i);
  if (m) raw = m[1];
  const res = await fetch("/api/v1/agents/fork", {
    method: "POST",
    headers: headersJSON(),
    body: JSON.stringify({ share_token: raw }),
  });
  const out = await res.json().catch(() => ({}));
  if (!res.ok) {
    log(out.detail || out);
    return;
  }
  forkInput.value = "";
  await refresh();
  log({ imported: out });
});

btnBackup.addEventListener("click", async () => {
  const res = await fetch("/api/v1/backup", { headers: headersJSON() });
  const json = await res.json();
  const blob = new Blob([JSON.stringify(json, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  const ts = json.exported_at || json.created_at;
  a.download = `agentpark-backup-${ts?.slice(0, 19).replace(/[:T]/g, "-") || "demo"}.json`;
  a.click();
  URL.revokeObjectURL(url);
  log("已下载备份 JSON（含 schema: agentpark.workspace.v1）");
});

async function doRestore(text) {
  let data;
  try {
    data = JSON.parse(text);
  } catch {
    log("JSON 解析失败");
    return;
  }
  const res = await fetch("/api/v1/restore", {
    method: "POST",
    headers: headersJSON(),
    body: JSON.stringify(data),
  });
  const out = await res.json().catch(() => ({}));
  if (!res.ok) {
    log(out);
    return;
  }
  await refresh();
  log(out);
}

btnPasteRestore.addEventListener("click", () => doRestore(paste.value.trim()));

fileRestore.addEventListener("change", async () => {
  const f = fileRestore.files?.[0];
  fileRestore.value = "";
  if (!f) return;
  const text = await f.text();
  await doRestore(text);
});

refresh().catch((e) => log(String(e)));
