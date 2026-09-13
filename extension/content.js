// baaz video grabber: hover any <video> with a real http(s) source and a
// small download button appears (IDM-style). blob:/MSE streams (YouTube etc.)
// get no button — their bytes never exist at a downloadable URL.

(() => {
  let btn = null;
  let currentVideo = null;
  let hideTimer = 0;

  function videoURL(v) {
    const src = v.currentSrc || v.src || "";
    if (/^https?:/i.test(src)) return src;
    for (const s of v.querySelectorAll("source")) {
      if (/^https?:/i.test(s.src)) return s.src;
    }
    return "";
  }

  function filenameFor(url) {
    try {
      const base = new URL(url).pathname.split("/").pop() || "";
      if (/\.[a-z0-9]{2,5}$/i.test(base)) return decodeURIComponent(base);
    } catch { /* fall through */ }
    const title = (document.title || "video").replace(/[\\/:*?"<>|]+/g, " ").trim().slice(0, 80);
    return title + ".mp4";
  }

  function ensureButton() {
    if (btn) return btn;
    btn = document.createElement("div");
    btn.textContent = "⬇ baaz";
    Object.assign(btn.style, {
      position: "fixed",
      zIndex: "2147483647",
      padding: "4px 10px",
      background: "rgba(20,20,20,.85)",
      color: "#fff",
      font: "12px system-ui, sans-serif",
      borderRadius: "14px",
      cursor: "pointer",
      userSelect: "none",
      display: "none",
      boxShadow: "0 1px 4px rgba(0,0,0,.4)",
    });
    btn.addEventListener("mouseenter", () => clearTimeout(hideTimer));
    btn.addEventListener("mouseleave", scheduleHide);
    btn.addEventListener("click", (e) => {
      e.stopPropagation();
      e.preventDefault();
      if (!currentVideo) return;
      const url = videoURL(currentVideo);
      if (!url) return;
      btn.textContent = "…";
      chrome.runtime.sendMessage(
        { type: "baaz-grab", url, filename: filenameFor(url), referrer: location.href },
        (reply) => {
          btn.textContent = reply && reply.ok ? "✓ baaz" : "✗ baaz";
          setTimeout(() => { if (btn) btn.textContent = "⬇ baaz"; }, 2000);
        }
      );
    });
    document.documentElement.appendChild(btn);
    return btn;
  }

  function showFor(v) {
    const url = videoURL(v);
    if (!url) return; // blob:/MSE stream — nothing at a URL to download
    currentVideo = v;
    const b = ensureButton();
    const r = v.getBoundingClientRect();
    if (r.width < 120 || r.height < 60) return; // skip thumbnail-sized players
    b.style.left = Math.max(4, r.right - 84) + "px";
    b.style.top = Math.max(4, r.top + 8) + "px";
    b.style.display = "block";
  }

  function scheduleHide() {
    clearTimeout(hideTimer);
    hideTimer = setTimeout(() => { if (btn) btn.style.display = "none"; }, 400);
  }

  document.addEventListener("mouseover", (e) => {
    const v = e.target instanceof HTMLVideoElement
      ? e.target
      : (e.target instanceof Element ? e.target.closest("video") : null);
    if (v) {
      clearTimeout(hideTimer);
      showFor(v);
    } else if (btn && e.target !== btn) {
      scheduleHide();
    }
  }, true);

  window.addEventListener("scroll", () => { if (btn) btn.style.display = "none"; }, true);
})();
