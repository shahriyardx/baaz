const DEFAULTS = { enabled: true };

const enabledEl = document.getElementById("enabled");
const dotEl = document.getElementById("dot");
const statusEl = document.getElementById("status");

chrome.storage.local.get(DEFAULTS).then((s) => {
  enabledEl.checked = s.enabled;
});

enabledEl.addEventListener("change", () =>
  chrome.storage.local.set({ enabled: enabledEl.checked }));

chrome.runtime.sendNativeMessage("com.shahriyar.dm", { type: "ping" }, (reply) => {
  if (chrome.runtime.lastError || !reply || !reply.ok) {
    dotEl.className = "down";
    statusEl.textContent = "daemon unreachable";
    return;
  }
  const n = reply.active || 0;
  if (reply.intercept === false) {
    dotEl.className = "down";
    statusEl.textContent = "intercept off (dm on)";
  } else {
    dotEl.className = "up";
    statusEl.textContent = n ? `${n} active` : "idle";
  }
  chrome.action.setBadgeText({ text: "" });
});
