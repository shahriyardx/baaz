const DEFAULTS = { enabled: true };

const enabledEl = document.getElementById("enabled");
const dotEl = document.getElementById("dot");
const statusEl = document.getElementById("status");
const activeEl = document.getElementById("active");
const modeEl = document.getElementById("mode");
const versionEl = document.getElementById("version");

versionEl.textContent = "v" + chrome.runtime.getManifest().version;

chrome.storage.local.get(DEFAULTS).then((s) => {
  enabledEl.checked = s.enabled;
});

enabledEl.addEventListener("change", () =>
  chrome.storage.local.set({ enabled: enabledEl.checked }));

chrome.runtime.sendNativeMessage("com.shahriyar.baaz", { type: "ping" }, (reply) => {
  if (chrome.runtime.lastError || !reply || !reply.ok) {
    dotEl.className = "dot down";
    statusEl.textContent = "daemon offline";
    activeEl.textContent = "–";
    modeEl.textContent = "–";
    return;
  }
  const n = reply.active || 0;
  activeEl.textContent = String(n);
  modeEl.textContent = reply.intercept === false ? "off" : "on";
  modeEl.style.color = reply.intercept === false ? "var(--bad)" : "var(--ok)";
  dotEl.className = "dot up";
  statusEl.textContent = n ? "downloading" : "idle";
  chrome.action.setBadgeText({ text: "" });
});
