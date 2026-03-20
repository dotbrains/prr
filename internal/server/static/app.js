/**
 * prr serve — vanilla JS SPA
 *
 * Structure:
 *   1. Router         — hash-based navigation
 *   2. State          — cached review data for palette
 *   3. API            — fetch wrapper
 *   4. Pages          — dashboard + detail renderers
 *   5. Components     — reusable HTML builders
 *   6. Event Handlers — delegated clicks, keyboard shortcuts
 *   7. Command Palette— ⌘K quick-nav
 *   8. Utilities      — escaping, formatting, helpers
 */

const app = document.getElementById("app");

// ─── 1. Router ───────────────────────────────────────────────

const routes = [
  { pattern: /^#\/$/, handler: () => renderDashboard() },
  { pattern: /^#\/review\/(.+)$/, handler: (m) => renderDetail(decodeURIComponent(m[1])) },
];

function route() {
  const hash = location.hash || "#/";
  for (const r of routes) {
    const m = hash.match(r.pattern);
    if (m) return r.handler(m);
  }
  renderDashboard();
}

window.addEventListener("hashchange", route);
window.addEventListener("DOMContentLoaded", route);

// ─── 2. State ────────────────────────────────────────────────

let cachedReviews = []; // populated by dashboard fetch, used by palette

// ─── 3. API ──────────────────────────────────────────────────

async function api(path) {
  const res = await fetch(`/api/${path}`);
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.json();
}

// ─── 4. Pages ────────────────────────────────────────────────

async function renderDashboard() {
  app.innerHTML = '<div class="loading">Loading reviews</div>';
  try {
    const reviews = await api("reviews");
    cachedReviews = reviews;
    updateNavCount(reviews.length);
    if (!reviews.length) {
      app.innerHTML = emptyState("No reviews yet", "Run <code>prr &lt;PR_NUMBER&gt;</code> to generate your first review.");
      return;
    }
    app.innerHTML = `
      <div class="search-bar">
        <input class="search-input" type="text" placeholder="Search reviews…" data-action="search">
        <span class="search-hint">⌘K</span>
      </div>
      <div class="review-list">${reviews.map(reviewCard).join("")}</div>`;
  } catch (err) {
    app.innerHTML = emptyState("Error loading reviews", esc(err.message));
  }
}

async function renderDetail(name) {
  app.innerHTML = '<div class="loading">Loading review</div>';
  try {
    const meta = await api(`reviews/${encodeURIComponent(name)}`);
    app.innerHTML = detailPage(meta, name);
  } catch (err) {
    app.innerHTML = `<a href="#/" class="back-link">← Back</a>${emptyState("Error loading review", esc(err.message))}`;
  }
}

// ─── 5. Components ───────────────────────────────────────────

function emptyState(title, body) {
  return `<div class="empty-state"><h2>${title}</h2><p>${body}</p></div>`;
}

const mergeIconSm = '<svg class="review-card-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="4" cy="4" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="4" cy="12" r="2"/><path d="M4 6v4M12 10c0-4-8-4-8 0"/></svg>';

function reviewCard(r) {
  const title = r.pr_number > 0 ? `PR #${r.pr_number}` : formatReviewName(r.name);
  const searchText = [title, r.repo_slug, r.agent_name, r.summary].filter(Boolean).join(" ").toLowerCase();
  const icon = r.pr_number > 0 ? mergeIconSm : "";
  return `
    <a href="#/review/${encodeURIComponent(r.name)}" class="review-card" data-search="${escAttr(searchText)}">
      <div class="review-card-header">
        <span class="review-card-title">${icon}${esc(title)}</span>
        <div class="badges">${severityBadges(r.stats)}</div>
      </div>
      <div class="review-card-meta">
        <span>${agentLabel(r)}</span>
        ${r.repo_slug ? `<span>${esc(r.repo_slug)}</span>` : ""}
        <span>${relativeDate(r.created_at)}</span>
      </div>
      ${r.summary ? `<div class="review-card-summary">${esc(r.summary)}</div>` : ""}
    </a>`;
}

function detailPage(meta, name) {
  const title = meta.pr_number > 0 ? `PR #${meta.pr_number}` : formatReviewName(name);
  const comments = meta.comments || [];
  const stats = countBy(comments, "severity");
  const byFile = groupBy(comments, "file");
  const fileNames = Object.keys(byFile).sort();
  fileNames.forEach(f => byFile[f].sort((a, b) => a.start_line - b.start_line));

  const sevs = ["critical", "suggestion", "nit", "praise"].filter(s => stats[s] > 0);

  const total = comments.length;
  const ghBase = meta.repo_slug ? `https://github.com/${meta.repo_slug}` : "";
  const ghPrUrl = (ghBase && meta.pr_number > 0) ? `${ghBase}/pull/${meta.pr_number}` : "";
  const ghCommitUrl = (ghBase && meta.head_sha) ? `${ghBase}/commit/${meta.head_sha}` : "";

  // Store ghPrUrl on window so fileSection can reference it.
  window._prr_ghPrUrl = ghPrUrl;
  window._prr_headSha = meta.head_sha || "";

  return `
    <div class="detail-topbar">
      <a href="#/" class="back-link">← All Reviews</a>
      <div class="detail-topbar-right">
        ${ghPrUrl ? `<a href="${ghPrUrl}" target="_blank" rel="noopener" class="gh-link">View PR on GitHub ↗</a>` : ""}
      </div>
    </div>
    <div class="detail-header">
      <div class="detail-title-row">
        ${meta.pr_number > 0 ? '<svg class="detail-title-icon" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="4" cy="4" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="4" cy="12" r="2"/><path d="M4 6v4M12 10c0-4-8-4-8 0"/></svg>' : ''}
        <h1>${esc(title)}</h1>
        <span class="detail-comment-count">${total} comment${total !== 1 ? "s" : ""}</span>
      </div>
      <div class="detail-meta">
        <span>${agentLabel(meta)}</span>
        ${ghBase ? `<a href="${ghBase}" target="_blank" rel="noopener" class="meta-link">${esc(meta.repo_slug)}</a>` : ""}
        ${ghCommitUrl ? `<a href="${ghCommitUrl}" target="_blank" rel="noopener" class="meta-link">${esc(meta.head_sha.slice(0, 8))}</a>` : ""}
        <span>${formatDate(meta.created_at)}</span>
      </div>
    </div>
    <div class="detail-layout">
      <div class="detail-sidebar">
        ${severityToggles(sevs, stats)}
        ${meta.summary ? `<div class="summary-section"><h2>Summary</h2><p>${esc(meta.summary)}</p></div>` : ""}
        ${fileNames.length ? fileNav(fileNames, byFile) : ""}
      </div>
      <div class="detail-content">
        ${fileNames.map(f => fileSection(f, byFile[f])).join("")}
      </div>
    </div>`;
}

function severityToggles(sevs, stats) {
  if (!sevs.length) return "";
  const total = sevs.reduce((n, s) => n + (stats[s] || 0), 0);
  return `<div class="severity-toggles">
    <span class="severity-toggles-label">Filter ${total} comments</span>
    <div class="severity-toggles-buttons">
      ${sevs.map(s =>
        `<button class="severity-toggle badge-${s} active" data-sev="${s}">${stats[s]} ${s}</button>`
      ).join("")}
    </div>
  </div>`;
}

function fileNav(fileNames, byFile) {
  const items = fileNames.map(f => `
    <div class="file-nav-item" data-scroll-to="${fileAnchor(f)}">
      <span>${esc(f.split("/").pop())}</span>
      <span class="file-nav-count">${byFile[f].length}</span>
    </div>`).join("");
  return `<div class="file-nav"><h3>Files (${fileNames.length})</h3>${items}</div>`;
}

function fileSection(fileName, comments) {
  const ghPrUrl = window._prr_ghPrUrl || "";
  const sha = window._prr_headSha || "";
  // Link to the file in the PR's "Files changed" tab.
  const fileUrl = (ghPrUrl && sha)
    ? `${ghPrUrl}/files#diff-${sha}-${fileName}` : "";
  // Simpler fallback: just link to the blob.
  const blobUrl = (ghPrUrl && !fileUrl) ? "" : "";
  const linkBtn = ghPrUrl
    ? `<a href="${ghPrUrl}/files#diff" target="_blank" rel="noopener" class="file-header-link" title="View in PR">↗</a>`
    : "";

  return `
    <div class="file-section" id="${fileAnchor(fileName)}">
      <div class="file-header">${esc(fileName)}${linkBtn}</div>
      ${comments.map(commentCard).join("")}
    </div>`;
}

function commentCard(c) {
  const line = formatLineRange(c.start_line, c.end_line);
  const mdLine = (c.start_line === c.end_line || !c.end_line)
    ? `Line ${c.start_line}` : `Lines ${c.start_line}-${c.end_line}`;
  const markdown = `**[${c.severity}]** ${c.file} — ${mdLine}\n\n${c.body}`;

  // Build GitHub link to highlighted lines in the PR.
  const ghPrUrl = window._prr_ghPrUrl || "";
  const lineRef = ghPrUrl ? buildGhLineUrl(ghPrUrl, c.file, c.start_line, c.end_line) : "";
  const lineHtml = lineRef
    ? `<a href="${lineRef}" target="_blank" rel="noopener" class="comment-line comment-line-link">${line}</a>`
    : `<span class="comment-line">${line}</span>`;

  return `
    <div class="comment" data-severity="${c.severity}">
      <button class="copy-btn" data-copy-md="${escAttr(markdown)}" title="Copy as markdown">⎘ Copy</button>
      <div class="comment-header">
        <span class="badge badge-${c.severity}">${c.severity}</span>
        ${lineHtml}
      </div>
      <div class="comment-body">${formatBody(c.body)}</div>
      ${verificationBadge(c.verification)}
    </div>`;
}

function verificationBadge(v) {
  if (!v) return "";
  const icons = { verified: "✓", inaccurate: "✗", uncertain: "⚠" };
  const reason = v.reason ? ": " + formatBody(v.reason) : "";
  return `<div class="verification verification-${v.verdict}">${icons[v.verdict] || ""} ${esc(v.verdict)}${reason}</div>`;
}

function severityBadges(stats) {
  return ["critical", "suggestion", "nit", "praise"]
    .filter(s => stats[s] > 0)
    .map(s => `<span class="badge badge-${s}">${stats[s]} ${s}</span>`)
    .join("");
}

function agentLabel(r) {
  return esc(r.agent_name) + (r.model ? ` (${esc(r.model)})` : "");
}

// ─── 6. Event Handlers (delegated) ───────────────────────────

document.addEventListener("click", (e) => {
  // File nav: scroll to section and reveal its comments.
  const navItem = e.target.closest("[data-scroll-to]");
  if (navItem) {
    const section = document.getElementById(navItem.dataset.scrollTo);
    if (section) {
      // Temporarily reveal all comments in this file section.
      section.querySelectorAll(".comment.sev-hidden").forEach(c => c.classList.remove("sev-hidden"));
      // Reset the severity toggles to "all active" since the filter state
      // no longer matches what's visible.
      document.querySelectorAll(".severity-toggle").forEach(t => t.classList.add("active"));
      section.scrollIntoView({ behavior: "smooth", block: "start" });
    }
    return;
  }

  // Copy button: write markdown to clipboard.
  const copyBtn = e.target.closest(".copy-btn");
  if (copyBtn) {
    navigator.clipboard.writeText(copyBtn.dataset.copyMd).then(() => {
      copyBtn.classList.add("copied");
      copyBtn.textContent = "✓ Copied";
      setTimeout(() => {
        copyBtn.classList.remove("copied");
        copyBtn.textContent = "⎘ Copy";
      }, 1500);
    });
    return;
  }

  // Severity toggle: show/hide comments by severity.
  const sevToggle = e.target.closest(".severity-toggle");
  if (sevToggle) {
    sevToggle.classList.toggle("active");
    const activeSevs = new Set(
      [...document.querySelectorAll(".severity-toggle.active")].map(b => b.dataset.sev)
    );
    document.querySelectorAll(".comment[data-severity]").forEach(el => {
      el.classList.toggle("sev-hidden", !activeSevs.has(el.dataset.severity));
    });
    return;
  }

  // Close palette on overlay click.
  if (e.target.classList.contains("palette-overlay")) {
    closePalette();
  }
});

// Dashboard search: filter cards as you type.
document.addEventListener("input", (e) => {
  if (e.target.dataset.action !== "search") return;
  const q = e.target.value.toLowerCase();
  document.querySelectorAll(".review-card").forEach(card => {
    const text = card.dataset.search || "";
    card.classList.toggle("hidden", q && !text.includes(q));
  });
});

// Keyboard shortcuts.
document.addEventListener("keydown", (e) => {
  const paletteOpen = !!document.querySelector(".palette-overlay");

  // ⌘K / Ctrl+K: toggle command palette.
  if ((e.metaKey || e.ctrlKey) && e.key === "k") {
    e.preventDefault();
    togglePalette();
    return;
  }

  // When palette is open, handle all nav keys globally (fallback if input
  // doesn't have focus for any reason).
  if (paletteOpen) {
    if (e.key === "Escape") {
      closePalette();
      return;
    }
    if (e.key === "ArrowDown" || e.key === "ArrowUp" || e.key === "Enter") {
      onPaletteKeydown(e);
      return;
    }
    // Any other key: ensure the palette input has focus so typing works.
    const input = document.querySelector(".palette-input");
    if (input && document.activeElement !== input) input.focus();
  }
});

// ─── 7. Command Palette ──────────────────────────────────────

let paletteSelectedIdx = 0;
let paletteItems = [];    // unified list: commands + reviews
let paletteAllItems = []; // full unfiltered list

function togglePalette() {
  document.querySelector(".palette-overlay") ? closePalette() : openPalette();
}

function isDetailPage() {
  return (location.hash || "").startsWith("#/review/");
}

function getDetailCommands() {
  const cmds = [];
  const ghPrUrl = window._prr_ghPrUrl || "";

  cmds.push({ type: "cmd", icon: "←", title: "Go to dashboard", action: () => { location.hash = "#/"; } });

  if (ghPrUrl) {
    cmds.push({ type: "cmd", icon: "↗", title: "Open PR on GitHub", action: () => { window.open(ghPrUrl, "_blank"); } });
  }

  // Severity filter commands — only for severities present in this review.
  // Show filled circle for currently active severities, empty for inactive.
  const sevToggles = [...document.querySelectorAll(".severity-toggle")];
  const activeSevs = new Set(sevToggles.filter(t => t.classList.contains("active")).map(t => t.dataset.sev));
  const allSevs = sevToggles.map(t => t.dataset.sev);
  const sevLabels = { critical: "critical", suggestion: "suggestions", nit: "nits", praise: "praise" };
  for (const sev of allSevs) {
    const isActive = activeSevs.has(sev);
    cmds.push({ type: "cmd", icon: isActive ? "●" : "○", title: `Show only ${sevLabels[sev] || sev}`, action: () => filterToSeverity(sev) });
  }
  if (allSevs.length > 1) {
    const allActive = activeSevs.size === allSevs.length;
    cmds.push({ type: "cmd", icon: allActive ? "●" : "○", title: "Show all severities", action: () => filterToSeverity(null) });
  }

  // Copy all comments as markdown.
  cmds.push({ type: "cmd", icon: "⎘", title: "Copy all comments as markdown", action: copyAllComments });

  return cmds;
}

function filterToSeverity(sev) {
  document.querySelectorAll(".severity-toggle").forEach(t => {
    if (sev === null) {
      t.classList.add("active");
    } else {
      t.classList.toggle("active", t.dataset.sev === sev);
    }
  });
  const activeSevs = new Set(
    [...document.querySelectorAll(".severity-toggle.active")].map(b => b.dataset.sev)
  );
  document.querySelectorAll(".comment[data-severity]").forEach(el => {
    el.classList.toggle("sev-hidden", !activeSevs.has(el.dataset.severity));
  });
  // Show feedback.
  const visible = document.querySelectorAll(".comment:not(.sev-hidden)").length;
  const label = sev === null ? "Showing all severities" : `Showing ${sev} only`;
  showToast(`${label} (${visible} comment${visible !== 1 ? "s" : ""})`);
  // Scroll to the filter bar so the user sees the state change.
  document.querySelector(".severity-toggles")?.scrollIntoView({ behavior: "smooth", block: "nearest" });
}

function copyAllComments() {
  const blocks = [];
  document.querySelectorAll(".comment:not(.sev-hidden)").forEach(el => {
    const btn = el.querySelector(".copy-btn");
    if (btn && btn.dataset.copyMd) blocks.push(btn.dataset.copyMd);
  });
  if (blocks.length) {
    navigator.clipboard.writeText(blocks.join("\n\n---\n\n"));
    showToast(`Copied ${blocks.length} comment${blocks.length !== 1 ? "s" : ""} to clipboard`);
  }
}

function showToast(message) {
  // Remove existing toast.
  document.querySelector(".toast")?.remove();
  const toast = document.createElement("div");
  toast.className = "toast";
  toast.textContent = message;
  document.body.appendChild(toast);
  // Trigger reflow then add visible class for animation.
  toast.offsetHeight;
  toast.classList.add("toast-visible");
  setTimeout(() => {
    toast.classList.remove("toast-visible");
    setTimeout(() => toast.remove(), 300);
  }, 1800);
}

async function openPalette() {
  if (!cachedReviews.length) {
    try { cachedReviews = await api("reviews"); } catch { /* ignore */ }
  }

  // Build items: commands first (on detail page), then reviews.
  const cmds = isDetailPage() ? getDetailCommands() : [];
  // Exclude the current review from the list if we're on its detail page.
  const currentName = isDetailPage()
    ? decodeURIComponent((location.hash.match(/^#\/review\/(.+)$/) || [])[1] || "")
    : "";
  const reviewItems = cachedReviews
    .filter(r => r.name !== currentName)
    .map(r => ({ type: "review", review: r }));
  paletteAllItems = [...cmds, ...reviewItems];
  paletteItems = paletteAllItems;
  paletteSelectedIdx = 0;

  const placeholder = isDetailPage() ? "Type a command or search reviews…" : "Search reviews…";

  const overlay = document.createElement("div");
  overlay.className = "palette-overlay";
  overlay.innerHTML = `
    <div class="palette">
      <input class="palette-input" type="text" placeholder="${placeholder}">
      <div class="palette-results">${renderPaletteItems(paletteItems, paletteSelectedIdx)}</div>
    </div>`;
  document.body.appendChild(overlay);

  const input = overlay.querySelector(".palette-input");
  input.addEventListener("input", () => onPaletteInput(input.value));
  setTimeout(() => input.focus(), 50);
}

function closePalette() {
  document.querySelector(".palette-overlay")?.remove();
}

function onPaletteInput(query) {
  const q = query.toLowerCase();
  paletteItems = paletteAllItems.filter(item => {
    if (!q) return true;
    if (item.type === "cmd") return item.title.toLowerCase().includes(q);
    const r = item.review;
    const text = [r.name, r.repo_slug, r.agent_name, r.summary, r.pr_number > 0 ? `PR #${r.pr_number}` : ""].join(" ").toLowerCase();
    return text.includes(q);
  });
  paletteSelectedIdx = 0;
  updatePaletteResults();
}

function onPaletteKeydown(e) {
  if (e.key === "ArrowDown") {
    e.preventDefault();
    paletteSelectedIdx = Math.min(paletteSelectedIdx + 1, paletteItems.length - 1);
    updatePaletteResults();
  } else if (e.key === "ArrowUp") {
    e.preventDefault();
    paletteSelectedIdx = Math.max(paletteSelectedIdx - 1, 0);
    updatePaletteResults();
  } else if (e.key === "Enter" && paletteItems[paletteSelectedIdx]) {
    e.preventDefault();
    executePaletteItem(paletteItems[paletteSelectedIdx]);
  }
}

function executePaletteItem(item) {
  closePalette();
  if (item.type === "cmd") {
    item.action();
  } else {
    location.hash = `#/review/${encodeURIComponent(item.review.name)}`;
  }
}

function updatePaletteResults() {
  const el = document.querySelector(".palette-results");
  if (!el) return;
  el.innerHTML = renderPaletteItems(paletteItems, paletteSelectedIdx);
  // Scroll the selected item into view.
  const selected = el.querySelector(".palette-item.selected");
  if (selected) selected.scrollIntoView({ block: "nearest" });
}

function renderPaletteItems(items, selectedIdx) {
  if (!items.length) return '<div class="palette-empty">No results</div>';
  return items.slice(0, 20).map((item, i) => {
    const sel = i === selectedIdx ? " selected" : "";
    if (item.type === "cmd") {
      return `<div class="palette-item${sel}" data-palette-idx="${i}">
        <span class="palette-item-icon">${item.icon}</span>
        <span class="palette-item-title">${esc(item.title)}</span>
        <span class="palette-item-meta">Command</span>
      </div>`;
    }
    const r = item.review;
    const title = r.pr_number > 0 ? `PR #${r.pr_number}` : formatReviewName(r.name);
    const icon = r.pr_number > 0
      ? `<svg class="palette-item-svg" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="4" cy="4" r="2"/><circle cx="12" cy="12" r="2"/><circle cx="4" cy="12" r="2"/><path d="M4 6v4M12 10c0-4-8-4-8 0"/></svg>`
      : `<span class="palette-item-icon">─</span>`;
    return `<div class="palette-item${sel}" data-palette-idx="${i}">
      ${icon}
      <span class="palette-item-title">${esc(title)}</span>
      <span class="palette-item-meta">${agentLabel(r)} · ${relativeDate(r.created_at)}</span>
    </div>`;
  }).join("");
}

// Palette click handler.
document.addEventListener("click", (e) => {
  const item = e.target.closest(".palette-item");
  if (item) {
    const idx = parseInt(item.dataset.paletteIdx, 10);
    if (paletteItems[idx]) executePaletteItem(paletteItems[idx]);
  }
});

// ─── 8. Utilities ────────────────────────────────────────────

function esc(str) {
  if (!str) return "";
  const el = document.createElement("span");
  el.textContent = str;
  return el.innerHTML;
}

function escAttr(str) {
  if (!str) return "";
  return str
    .replace(/&/g, "&amp;")
    .replace(/"/g, "&quot;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

function formatDate(iso) {
  if (!iso) return "";
  try {
    const d = new Date(iso);
    return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" }) +
      " " + d.toLocaleTimeString("en-US", { hour: "2-digit", minute: "2-digit" });
  } catch { return iso; }
}

function relativeDate(iso) {
  if (!iso) return "";
  try {
    const d = new Date(iso);
    const now = Date.now();
    const diffS = Math.floor((now - d.getTime()) / 1000);
    if (diffS < 60) return "just now";
    if (diffS < 3600) return `${Math.floor(diffS / 60)}m ago`;
    if (diffS < 86400) return `${Math.floor(diffS / 3600)}h ago`;
    if (diffS < 604800) return `${Math.floor(diffS / 86400)}d ago`;
    return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
  } catch { return iso; }
}

function formatBody(text) {
  if (!text) return "";
  let html = esc(text);
  html = html.replace(/```(\w*)\n([\s\S]*?)```/g, (_, _lang, code) => `<pre><code>${code}</code></pre>`);
  html = html.replace(/`([^`]+)`/g, "<code>$1</code>");
  return html;
}

function formatLineRange(start, end) {
  return (start === end || !end) ? `L${start}` : `L${start}-${end}`;
}

function formatReviewName(name) {
  return name.replace(/^review-/, "").replace(/-\d{8}-\d{6}$/, "");
}

function fileAnchor(path) {
  return "file-" + path.replace(/[^a-zA-Z0-9]/g, "-");
}

function buildGhLineUrl(prUrl, file, startLine, endLine) {
  // GitHub PR files tab line anchors use the format:
  // /pull/N/files#diff-<sha256-of-path>R<line> or R<start>-R<end>
  // Since we can't compute the SHA256 in vanilla JS without crypto API complexity,
  // use the simpler blob URL: /blob/<sha>/<file>#L<start>-L<end>
  const ghBase = prUrl.replace(/\/pull\/\d+$/, "");
  const sha = window._prr_headSha;
  if (!sha) return "";
  const anchor = (startLine === endLine || !endLine)
    ? `L${startLine}` : `L${startLine}-L${endLine}`;
  return `${ghBase}/blob/${sha}/${file}#${anchor}`;
}

function groupBy(arr, key) {
  const m = {};
  for (const item of arr) (m[item[key]] ??= []).push(item);
  return m;
}

function countBy(arr, key) {
  const m = {};
  for (const item of arr) m[item[key]] = (m[item[key]] || 0) + 1;
  return m;
}

function updateNavCount(n) {
  const el = document.getElementById("nav-review-count");
  if (el) el.textContent = n > 0 ? `${n} review${n !== 1 ? "s" : ""}` : "";
}
