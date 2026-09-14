const DEFAULTS = { enabled: true };

const els = {
  enabled: document.getElementById("enabled"),
  dot: document.getElementById("dot"),
  status: document.getElementById("statusText"),
  live: document.getElementById("live"),
  count: document.getElementById("activeCount"),
  cardSub: document.getElementById("cardSub"),
  version: document.getElementById("version"),
  getapp: document.getElementById("getapp"),
};

els.version.textContent = "v" + chrome.runtime.getManifest().version;

chrome.storage.local.get(DEFAULTS).then((s) => {
  els.enabled.checked = s.enabled;
  reflectSwitch(s.enabled);
});

els.enabled.addEventListener("change", () => {
  chrome.storage.local.set({ enabled: els.enabled.checked });
  reflectSwitch(els.enabled.checked);
});

function reflectSwitch(on) {
  els.cardSub.textContent = on
    ? "Chrome hands files to Baaz"
    : "Chrome downloads normally";
}

function setStatus(kind, text) {
  els.dot.className = "dot" + (kind ? " " + kind : "");
  els.status.textContent = text;
}

chrome.runtime.sendNativeMessage("com.shahriyar.baaz", { type: "ping" }, (reply) => {
  if (chrome.runtime.lastError || !reply || !reply.ok) {
    // Installed from the store without the app, or the app is not running.
    // Either way, point somewhere useful rather than just reporting it.
    setStatus("down", "not connected");
    els.getapp.classList.add("show");
    return;
  }
  const n = reply.active || 0;
  // The header reports the connection, the banner reports activity. Saying
  // "downloading" in both just repeats itself.
  if (reply.intercept === false) {
    // The app's own switch is off, which outranks the extension's.
    setStatus("", "paused in app");
  } else {
    setStatus(n ? "busy" : "up", "connected");
  }
  if (n > 0) {
    els.count.textContent = String(n);
    els.live.classList.add("show");
  }
  chrome.action.setBadgeText({ text: "" });
});
