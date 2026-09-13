// baaz interceptor: hand browser downloads to the local baaz daemon.
//
// Uses downloads.onDeterminingFilename because doing nothing lets the
// browser download proceed untouched — so every failure path (daemon dead,
// native host missing, timeout) degrades to a normal browser download.

const HOST = "com.shahriyar.baaz";
const NATIVE_TIMEOUT_MS = 3000;

const DEFAULTS = { enabled: true };

function getSettings() {
  return chrome.storage.local.get(DEFAULTS);
}

function sendNative(msg) {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error("native host timeout")), NATIVE_TIMEOUT_MS);
    try {
      chrome.runtime.sendNativeMessage(HOST, msg, (reply) => {
        clearTimeout(timer);
        if (chrome.runtime.lastError) reject(new Error(chrome.runtime.lastError.message));
        else resolve(reply);
      });
    } catch (e) {
      clearTimeout(timer);
      reject(e);
    }
  });
}

async function cookieHeaderFor(url) {
  try {
    const cookies = await chrome.cookies.getAll({ url });
    return cookies.map((c) => `${c.name}=${c.value}`).join("; ");
  } catch {
    return "";
  }
}

function basename(path) {
  return path.split(/[\\/]/).pop() || "";
}

async function flagFallback() {
  try {
    await chrome.action.setBadgeText({ text: "!" });
    await chrome.action.setBadgeBackgroundColor({ color: "#cc3333" });
    setTimeout(() => chrome.action.setBadgeText({ text: "" }), 15000);
  } catch {
    // no-op
  }
}

async function intercept(item) {
  const settings = await getSettings();
  if (!settings.enabled) return;

  const url = item.finalUrl || item.url;
  if (!/^https?:/i.test(url)) return; // blob:, data:, filesystem:, chrome: stay in the browser
  if (item.byExtensionId) return; // another extension's download

  // Size and policy (intercept on/off, min size) are the daemon's call —
  // a "rejected" reply means "let the browser download it", silently.
  const fileSize = item.fileSize > 0 ? item.fileSize : (item.totalBytes > 0 ? item.totalBytes : 0);

  let reply;
  try {
    reply = await sendNative({
      type: "add",
      url,
      fileSize,
      filename: basename(item.filename),
      cookies: await cookieHeaderFor(url),
      referrer: item.referrer || "",
      userAgent: navigator.userAgent,
    });
  } catch (e) {
    console.warn("baaz unreachable, falling back to browser download:", e.message);
    flagFallback();
    return;
  }
  if (!reply || !reply.ok) {
    if (reply && reply.rejected) return; // intercept off or below min size
    console.warn("baaz error:", reply && reply.error);
    flagFallback();
    return;
  }

  // Daemon owns it now — remove the browser's copy. If a tiny file already
  // finished (race), keep the browser file and cancel the daemon job instead.
  const [current] = await chrome.downloads.search({ id: item.id });
  if (current && current.state === "complete") {
    sendNative({ type: "cancel", id: reply.id }).catch(() => {});
    return;
  }
  try {
    await chrome.downloads.cancel(item.id);
    await chrome.downloads.erase({ id: item.id });
  } catch (e) {
    console.warn("could not cancel browser download:", e.message);
  }
}

chrome.downloads.onDeterminingFilename.addListener((item, suggest) => {
  suggest(); // pass-through; the async work below decides interception
  intercept(item).catch((e) => console.error("baaz intercept error:", e));
});

// Explicit user click on the in-page video button: bypasses intercept/min-size
// policy — the user asked for exactly this file.
chrome.runtime.onMessage.addListener((msg, sender, sendResponse) => {
  if (!msg || msg.type !== "baaz-grab") return;
  (async () => {
    try {
      const reply = await sendNative({
        type: "grab",
        url: msg.url,
        filename: msg.filename || "",
        cookies: await cookieHeaderFor(msg.url),
        referrer: msg.referrer || "",
        userAgent: navigator.userAgent,
      });
      sendResponse({ ok: !!(reply && reply.ok) });
    } catch (e) {
      console.warn("baaz grab failed:", e.message);
      sendResponse({ ok: false });
    }
  })();
  return true; // async sendResponse
});
