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

  // Built from what the video actually offers, not a fixed list: showing 4K
  // on a 720p clip promises a download that cannot happen. The daemon asks
  // yt-dlp and the menu is rebuilt from the answer.

  // Whether a file plays is a property of its codec, not its resolution.
  // H.264 opens in everything; VP9 and AV1 are fine in Chrome, VLC, IINA and
  // recent Macs, but older QuickTime refuses them. So the note names the
  // codec and leaves the judgement to whoever knows their own player,
  // instead of asserting "needs VLC" from the resolution alone.
  function labelFor(q) {
    const p = q.label + "p";
    if (!q.codec || q.codec === "h264") return p;
    if (q.codec === "vp9") return p + " · VP9";
    if (q.codec === "av1") return p + " · AV1";
    return p;
  }

  // Built from what the video reports rather than an expected ladder:
  // YouTube goes 2160/1440/1080/720/480, Facebook exposes no sizes at all,
  // and hardcoding either leaves the other wrong. Anything under 240p is a
  // thumbnail strip, not a choice.
  function qualitiesFor(qualities) {
    if (!qualities || !qualities.length) return null;
    const usable = qualities.filter((q) => q.label >= 240).slice(0, 6);
    if (!usable.length) return null;
    return [
      { key: "best", label: "Best · plays anywhere" },
      ...usable.map((q) => ({ key: String(q.label), label: labelFor(q) })),
      { key: "audio", label: "Audio only · mp3" },
    ];
  }

  const ONLY_BEST = [
    { key: "best", label: "Best quality" },
    { key: "audio", label: "Audio only · mp3" },
  ];

  const FALLBACK_QUALITIES = [
    { key: "best", label: "Best · plays anywhere" },
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
    // Sits on top of someone else's video, so it stays small and dark and
    // gets out of the way: no shadow bloom, a hairline border, and a lift on
    // hover rather than a colour change shouting for attention.
    const REST = "rgba(12, 14, 18, .72)";
    const HOVER = "rgba(22, 26, 33, .92)";
    Object.assign(btn.style, {
      display: "flex",
      alignItems: "center",
      gap: "6px",
      padding: "6px 11px",
      background: REST,
      backdropFilter: "blur(14px) saturate(140%)",
      WebkitBackdropFilter: "blur(14px) saturate(140%)",
      color: "#fff",
      fontSize: "12px",
      fontWeight: "560",
      letterSpacing: "-.01em",
      lineHeight: "1",
      borderRadius: "8px",
      border: "1px solid rgba(255,255,255,.12)",
      boxShadow: "0 2px 8px rgba(0,0,0,.28)",
      cursor: "pointer",
      userSelect: "none",
      transition: "background .14s ease, transform .14s ease, box-shadow .14s ease",
    });
    btn.addEventListener("mouseenter", () => {
      btn.style.background = HOVER;
      btn.style.transform = "translateY(-1px)";
      btn.style.boxShadow = "0 4px 14px rgba(0,0,0,.34)";
    });
    btn.addEventListener("mouseleave", () => {
      btn.style.background = REST;
      btn.style.transform = "none";
      btn.style.boxShadow = "0 2px 8px rgba(0,0,0,.28)";
    });
    btn.addEventListener("mousedown", () => { btn.style.transform = "translateY(0) scale(.97)"; });
    btn.addEventListener("mouseup", () => { btn.style.transform = "translateY(-1px)"; });
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
      minWidth: "164px",
      background: "rgba(14, 17, 22, .94)",
      backdropFilter: "blur(20px) saturate(140%)",
      WebkitBackdropFilter: "blur(20px) saturate(140%)",
      border: "1px solid rgba(255,255,255,.1)",
      borderRadius: "10px",
      boxShadow: "0 10px 30px rgba(0,0,0,.4), 0 1px 0 rgba(255,255,255,.05) inset",
      overflow: "hidden",
      padding: "4px",
    });
    // Filled once the daemon reports what this video offers.
    renderMenu(null);
    wrap.appendChild(menu);
    document.documentElement.appendChild(wrap);
  }

  // null = still asking. An empty menu with a note beats a menu of
  // resolutions the video does not have.
  function renderMenu(items) {
    menu.textContent = "";
    if (!items) {
      const note = document.createElement("div");
      note.textContent = "Checking qualities…";
      Object.assign(note.style, {
        padding: "7px 10px", color: "rgba(255,255,255,.5)",
        fontSize: "12px", fontWeight: "500",
      });
      menu.appendChild(note);
      return;
    }
    for (const q of items) {
      const item = document.createElement("div");
      item.textContent = q.label;
      Object.assign(item.style, {
        display: "flex", alignItems: "center", padding: "7px 10px",
        color: "rgba(255,255,255,.88)", fontSize: "12px", fontWeight: "500",
        letterSpacing: "-.01em", borderRadius: "6px", cursor: "pointer",
        transition: "background .1s ease, color .1s ease",
      });
      item.addEventListener("mouseenter", () => {
        item.style.background = "rgba(79,140,255,.9)";
        item.style.color = "#fff";
      });
      item.addEventListener("mouseleave", () => {
        item.style.background = "transparent";
        item.style.color = "rgba(255,255,255,.88)";
      });
      item.addEventListener("click", (e) => {
        e.stopPropagation();
        e.preventDefault();
        menu.style.display = "none";
        grab(q.key);
      });
      menu.appendChild(item);
    }
  }

  // Asked once per video. A failure falls back to the fixed list rather than
  // leaving the menu empty — a slow answer should not cost the download.
  // Answers are cached per URL. The first version tracked only the most
  // recent request, so a hover moving between videos orphaned the reply in
  // flight — and because it also refused to ask twice for the same URL, the
  // menu stayed on "Checking qualities…" for good.
  const formatsCache = new Map();
  const formatsPending = new Set();

  function menuItemsFor(url) {
    const qualities = formatsCache.get(url);
    if (qualities === undefined) return null;   // still asking
    if (qualities === null) return FALLBACK_QUALITIES; // lookup failed: guess
    // Answered, but with no resolutions to choose between. Facebook is the
    // common case: its formats are named "sd" and "hd" and carry no height
    // at all. Offering 1080p there would promise something that does not
    // exist, which is the whole bug this set out to fix.
    return qualitiesFor(qualities) || ONLY_BEST;
  }

  function loadFormats(url) {
    if (!url || formatsCache.has(url) || formatsPending.has(url)) return;
    formatsPending.add(url);
    chrome.runtime.sendMessage({ type: "baaz-formats", url }, (reply) => {
      formatsPending.delete(url);
      // An unreachable daemon or an unreadable link both land here; caching
      // an empty list means the fallback menu is shown rather than nothing.
      // null distinguishes a failed lookup from one that succeeded with
      // nothing to offer; the two deserve different menus.
      formatsCache.set(url, reply && reply.ok ? (reply.qualities || []) : null);
      // Only redraw if this is still the video under the cursor.
      if (pageMode && pageURL === url && menu && menu.style.display !== "none") {
        renderMenu(menuItemsFor(url));
      }
    });
  }

  function toggleMenu() {
    const opening = menu.style.display === "none";
    menu.style.display = opening ? "block" : "none";
    if (!opening) return;
    renderMenu(menuItemsFor(pageURL));
    loadFormats(pageURL);
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
    const idle = Date.now() - lastMove > 2500;
    if (video) {
      // Only a moving cursor shows the button; once idle-faded it stays
      // hidden until the mouse moves again — re-showing and fading in the
      // same loop made it blink 4x a second under a resting cursor.
      if (!idle) showFor(video, stack);
      else if (visible && !menuOpen) hideUI();
      return;
    }
    if (visible && !menuOpen) hideUI(); // over neither video nor control
  }, 250);

  window.addEventListener("scroll", hideUI, true);
  document.addEventListener("mouseleave", hideUI);
})();
