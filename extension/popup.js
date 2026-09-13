const DEFAULTS = { enabled: true, minSizeMB: 5 };

const enabledEl = document.getElementById("enabled");
const minSizeEl = document.getElementById("minSize");
const dotEl = document.getElementById("dot");
const statusEl = document.getElementById("status");

chrome.storage.local.get(DEFAULTS).then((s) => {
  enabledEl.checked = s.enabled;
  minSizeEl.value = s.minSizeMB;
});

enabledEl.addEventListener("change", () =>
  chrome.storage.local.set({ enabled: enabledEl.checked }));
minSizeEl.addEventListener("change", () =>
  chrome.storage.local.set({ minSizeMB: Math.max(0, Number(minSizeEl.value) || 0) }));

chrome.runtime.sendNativeMessage("com.shahriyar.dm", { type: "ping" }, (reply) => {
  if (chrome.runtime.lastError || !reply || !reply.ok) {
    dotEl.className = "down";
    statusEl.textContent = "daemon unreachable";
    return;
  }
  dotEl.className = "up";
  const n = reply.active || 0;
  statusEl.textContent = n ? `${n} active` : "idle";
  chrome.action.setBadgeText({ text: "" });
});
