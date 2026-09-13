// baaz video grabber: hover any <video> and a download control appears.
// Direct http(s) sources download in one click; on known media sites
// (YouTube etc., where the player only has a blob stream) the page URL is
// sent instead and a quality menu is offered — the daemon runs yt-dlp.
//
// Visibility is decided by one watcher loop over the cursor position
// (elementsFromPoint pierces player overlays): over video or control =
// visible, over neither = hidden, idle 2.5s = faded like native controls.

(() => {
  let wrap = null;
  let menu = null;
  let currentVideo = null;
  let pageMode = false;
  let pageURL = ""; // what a pageMode grab downloads (watch link or page URL)
  let lastX = -1, lastY = -1, lastMove = 0;

  // Keep in sync with mediaHosts in internal/downloader/ytdlp.go.
  const MEDIA_HOSTS = [
    "youtube.com", "youtu.be", "vimeo.com", "twitch.tv", "tiktok.com",
    "x.com", "twitter.com", "instagram.com", "facebook.com",
    "dailymotion.com", "soundcloud.com",
  ];

  const QUALITIES = [
    { key: "best", label: "Best quality" },
    { key: "1080", label: "1080p" },
    { key: "720", label: "720p" },
    { key: "480", label: "480p" },
    { key: "audio", label: "Audio only · mp3" },
  ];

  const ICON = `<svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3v12"/><path d="m7 11 5 5 5-5"/><path d="M5 21h14"/></svg>`;

  function isMediaPage() {
    const h = location.hostname.replace(/^www\./, "");
    return MEDIA_HOSTS.some((m) => h === m || h.endsWith("." + m));
  }

  function videoURL(v) {
    const src = v.currentSrc || v.src || "";
    if (/^https?:/i.test(src)) return src;
    for (const s of v.querySelectorAll("source")) {
      if (/^https?:/i.test(s.src)) return s.src;
    }
    return "";
  }

  // On feed/home pages the address bar is useless (youtube.com/) — the
  // hovered preview's own watch link lives in an anchor under the cursor
  // or wrapping the video. Only fall back to the page URL when it points
  // at an actual video page.
  const VIDEO_PATH = /watch\?|\/shorts\/|\/reel(s)?\/|\/videos?\/|\/clip\/|\/status\/|youtu\.be\//;
  function mediaTargetURL(stack, v) {
    const cands = [];
    for (const el of stack) {
      if (el instanceof HTMLAnchorElement && el.href) cands.push(el.href);
    }
    const a = v.closest ? v.closest("a[href]") : null;
    if (a && a.href) cands.push(a.href);
    for (const href of cands) {
      try {
        const u = new URL(href, location.href);
        if (/^https?:$/.test(u.protocol) && VIDEO_PATH.test(u.pathname + u.search)) return u.href;
      } catch { /* skip */ }
    }
    if (VIDEO_PATH.test(location.pathname + location.search)) return location.href;
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

  function ensureUI() {
    if (wrap) return;
    wrap = document.createElement("div");
    Object.assign(wrap.style, {
      position: "fixed",
      zIndex: "2147483647",
      display: "none",
      fontFamily: "system-ui, -apple-system, 'Segoe UI', sans-serif",
      WebkitFontSmoothing: "antialiased",
    });

    const btn = document.createElement("div");
    btn.id = "baaz-btn";
    btn.innerHTML = ICON + "<span style='margin-left:6px'>Download</span>";
    Object.assign(btn.style, {
      display: "flex",
      alignItems: "center",
      padding: "7px 14px",
      background: "rgba(17, 20, 26, .78)",
      backdropFilter: "blur(10px)",
      color: "#fff",
      fontSize: "12.5px",
      fontWeight: "600",
      letterSpacing: ".2px",
      borderRadius: "10px",
      border: "1px solid rgba(255,255,255,.14)",
      boxShadow: "0 4px 14px rgba(0,0,0,.35)",
      cursor: "pointer",
      userSelect: "none",
      transition: "background .12s",
    });
    btn.addEventListener("mouseenter", () => { btn.style.background = "rgba(35, 40, 50, .92)"; });
    btn.addEventListener("mouseleave", () => { btn.style.background = "rgba(17, 20, 26, .78)"; });
    btn.addEventListener("click", (e) => {
      e.stopPropagation();
      e.preventDefault();
      if (pageMode) toggleMenu();
      else grab("");
    });
    wrap.appendChild(btn);

    menu = document.createElement("div");
    Object.assign(menu.style, {
      display: "none",
      // right-aligned under the button so it opens leftward, staying on
      // the video instead of spilling past its right edge
      position: "absolute",
      right: "0",
      top: "calc(100% + 6px)",
      minWidth: "170px",
      background: "rgba(17, 20, 26, .92)",
      backdropFilter: "blur(12px)",
      border: "1px solid rgba(255,255,255,.14)",
      borderRadius: "12px",
      boxShadow: "0 8px 24px rgba(0,0,0,.45)",
      overflow: "hidden",
      padding: "5px",
    });
    for (const q of QUALITIES) {
      const item = document.createElement("div");
      item.textContent = q.label;
      Object.assign(item.style, {
        padding: "8px 12px",
        color: "#e8eaed",
        fontSize: "12.5px",
        borderRadius: "8px",
        cursor: "pointer",
      });
      item.addEventListener("mouseenter", () => { item.style.background = "rgba(79,140,255,.22)"; });
      item.addEventListener("mouseleave", () => { item.style.background = "transparent"; });
      item.addEventListener("click", (e) => {
        e.stopPropagation();
        e.preventDefault();
        menu.style.display = "none";
        grab(q.key);
      });
      menu.appendChild(item);
    }
    wrap.appendChild(menu);
    document.documentElement.appendChild(wrap);
  }

  function toggleMenu() {
    menu.style.display = menu.style.display === "none" ? "block" : "none";
  }

  function setLabel(html) {
    const btn = wrap.querySelector("#baaz-btn");
    if (btn) btn.innerHTML = html;
  }

  function grab(format) {
    if (!currentVideo) return;
    const url = pageMode ? pageURL : videoURL(currentVideo);
    if (!url) return;
    setLabel(ICON + "<span style='margin-left:6px'>Sending…</span>");
    chrome.runtime.sendMessage(
      {
        type: "baaz-grab",
        url,
        format,
        filename: pageMode ? "" : filenameFor(url),
        referrer: location.href,
      },
      (reply) => {
        const ok = reply && reply.ok;
        setLabel(ICON + `<span style='margin-left:6px'>${ok ? "Added ✓" : "Failed ✗"}</span>`);
        setTimeout(() => setLabel(ICON + "<span style='margin-left:6px'>Download</span>"), 2200);
      }
    );
  }

  function showFor(v, stack) {
    const url = videoURL(v);
    pageMode = !url;
    if (!url) {
      if (!isMediaPage()) return;
      pageURL = mediaTargetURL(stack || [], v);
      if (!pageURL) return; // homepage preview with no resolvable watch link
    }
    const r = v.getBoundingClientRect();
    if (r.width < 160 || r.height < 90) return; // skip thumbnails
    currentVideo = v;
    ensureUI();
    wrap.style.left = Math.max(6, r.right - 130) + "px";
    wrap.style.top = Math.max(6, r.top + 10) + "px";
    wrap.style.display = "block";
  }

  function hideUI() {
    if (!wrap) return;
    wrap.style.display = "none";
    menu.style.display = "none";
  }

  function cursorStack() {
    if (lastX < 0) return [];
    try {
      return document.elementsFromPoint(lastX, lastY);
    } catch {
      return [];
    }
  }

  document.addEventListener("mousemove", (e) => {
    lastX = e.clientX;
    lastY = e.clientY;
    lastMove = Date.now();
  }, true);

  // The watcher: sole owner of show/hide, immune to event-timing races.
  setInterval(() => {
    const stack = cursorStack();
    const overControl = wrap && stack.some((el) => wrap.contains(el));
    const video = stack.find((el) => el instanceof HTMLVideoElement);
    const menuOpen = menu && menu.style.display !== "none";
    const visible = wrap && wrap.style.display !== "none";

    if (overControl) return; // never yank the control out from under the cursor
    if (video) {
      showFor(video, stack);
      if (visible && !menuOpen && Date.now() - lastMove > 2500) hideUI(); // idle fade
      return;
    }
    if (visible && !menuOpen) hideUI(); // over neither video nor control
  }, 250);

  window.addEventListener("scroll", hideUI, true);
  document.addEventListener("mouseleave", hideUI);
})();
