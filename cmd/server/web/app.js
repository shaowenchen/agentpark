const listEl = document.getElementById("list");
const form = document.getElementById("form");
const logEl = document.getElementById("log");
const btnBackup = document.getElementById("btnBackup");
const btnPasteRestore = document.getElementById("btnPasteRestore");
const fileRestore = document.getElementById("fileRestore");
const paste = document.getElementById("paste");

function log(msg) {
  logEl.textContent = typeof msg === "string" ? msg : JSON.stringify(msg, null, 2);
}

async function refresh() {
  const res = await fetch("/api/agents");
  const data = await res.json();
  listEl.innerHTML = "";
  if (!data.length) {
    listEl.innerHTML = '<li class="empty">暂无 Agent，请新建或恢复备份。</li>';
    return;
  }
  for (const a of data) {
    const li = document.createElement("li");
    li.innerHTML = `
      <div class="meta">
        <div class="name"></div>
        <div class="sys"></div>
      </div>
      <button class="del" type="button" data-id="${a.id}">删除</button>
    `;
    li.querySelector(".name").textContent = a.name;
    li.querySelector(".sys").textContent = a.system || "（无系统提示）";
    li.querySelector(".del").addEventListener("click", async () => {
      await fetch(`/api/agents/${encodeURIComponent(a.id)}`, { method: "DELETE" });
      await refresh();
      log(`已删除: ${a.name}`);
    });
    listEl.appendChild(li);
  }
}

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const fd = new FormData(form);
  const name = fd.get("name")?.toString().trim();
  const system = fd.get("system")?.toString() ?? "";
  const res = await fetch("/api/agents", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, system }),
  });
  if (!res.ok) {
    log(await res.text());
    return;
  }
  form.reset();
  await refresh();
  log("已添加 Agent");
});

btnBackup.addEventListener("click", async () => {
  const res = await fetch("/api/backup");
  const json = await res.json();
  const blob = new Blob([JSON.stringify(json, null, 2)], { type: "application/json" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `agentpark-backup-${json.created_at?.slice(0, 19).replace(/[:T]/g, "-") || "demo"}.json`;
  a.click();
  URL.revokeObjectURL(url);
  log("已下载备份 JSON");
});

async function doRestore(text) {
  let data;
  try {
    data = JSON.parse(text);
  } catch {
    log("JSON 解析失败");
    return;
  }
  const res = await fetch("/api/restore", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
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
