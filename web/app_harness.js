/*
 * naming conventions:
 * - fns that draw stuff are named renderXXX(), under the `=== rendering` header
 * - fns that makes api calls are named apiXXX(), under the `=== api calls` header
 * */

import { gfm, gfmHtml, micromark } from "/lib/markdown/markdown.js";

// mermaid utils
window.mermaid.initialize({ startOnLoad: false, securityLevel: "strict", suppressErrorRendering: true, theme: document.documentElement.dataset.theme === "dark" ? "dark" : "default" });
let mermaidID = 0;
function _renderMermaid(source, diagram) {
	const theme = document.documentElement.dataset.theme === "dark" ? "dark" : "default", renderID = diagram.renderID = (diagram.renderID || 0) + 1, id = `ahweb-mermaid-${++mermaidID}`;
	window.mermaid.initialize({ startOnLoad: false, securityLevel: "strict", suppressErrorRendering: true, theme });
	window.mermaid.render(id, source).then(({ svg }) => {
		if (diagram.renderID === renderID) diagram.innerHTML = svg;
	}).catch((err) => {
		document.getElementById(id)?.remove();
		document.getElementById(`d${id}`)?.remove();
		document.getElementById(`i${id}`)?.remove();
		if (diagram.renderID === renderID) diagram.textContent = `Could not render diagram: ${err.message}`;
	});
}
new MutationObserver(() => {
	document.querySelectorAll("#messages .mermaid-code").forEach((wrapper) => {
		const code = wrapper.querySelector("code.language-mermaid"), diagram = wrapper.querySelector(".mermaid-diagram");
		if (code && diagram && !diagram.hidden) _renderMermaid(code.textContent, diagram);
	});
}).observe(document.documentElement, { attributes: true, attributeFilter: ["data-theme"] });

// actual utils
const $ = (s) => document.querySelector(s);
const CACHE = { chats: {} };
const enc = encodeURIComponent;
let folder = "",
	sessid = null,
	lastSeq = 0,
	nextStartFrom = null,
	seen = new Set();
let chatPinned = true;
let tools = new Map();
let eventsCard = null,
	eventsList = null,
	eventsLabel = null;
let models = {},
	config = {};

const eleDir = $("#directory"),
	eleNoticeArea = $("#notice-area"),
	eleNoticeList = $("#notice-list"),
	eleChatStatus = $("#chat-status"),
	eleUsage = $("#usage"),
	eleQueued = $("#queued-prompts"),
	eleChat = $("#messages"),
	eleSess = $("#chats-list"),
	eleProjects = $("#projects-list"),
	eleWorktrees = $("#worktrees-list"),
	eleBreadcrumbProject = $('[data-breadcrumb="project"]'),
	eleBreadcrumbWorktree = $('[data-breadcrumb="worktree"]'),
	eleFolder = $("#session-folder"),
	eleCount = $("#session-count"),
	elePrompt = $("#prompt"),
	eleImageInput = $("#prompt-images"),
	eleImagePreview = $("#image-preview"),
	eleModel = $("#model"),
	eleModelList = $("#model-list"), // used to support auto complete
	eleReasoning = $("#reasoning"),
	eleHarness = $("#harness"),
	eleRefresh = $("#refresh");

eleChat.addEventListener("scroll", () => {
	chatPinned =
		eleChat.scrollHeight - eleChat.scrollTop - eleChat.clientHeight < 80;
});
const realFolder = {};
let promptImages = [];
let projectRoots = [],
	selectedProject = "",
	worktreePaths = [],
	selectedWorktree = "";

// === utils + state actions
function localISO(timestamp) {
	if (!Number.isFinite(timestamp) || timestamp <= 0) return "";
	function _pad(value) {
		return String(value).padStart(2, "0");
	}
	const date = new Date(timestamp);
	if (Number.isNaN(date.getTime())) return "";
	const offset = -date.getTimezoneOffset();
	const sign = offset < 0 ? "-" : "+";
	const minutes = Math.abs(offset);
	const zone = `${sign}${_pad(Math.floor(minutes / 60))}:${_pad(minutes % 60)}`;
	return `${date.getFullYear()}-${_pad(date.getMonth() + 1)}-${_pad(date.getDate())}T${_pad(date.getHours())}:${_pad(date.getMinutes())}:${_pad(date.getSeconds())}${zone}`;
}

function clearQueuedPrompts() {
	eleQueued.textContent = "";
	eleQueued.hidden = true;
}

const notices = new Map();
function setNotice(id, message) {
	if (message) notices.set(id, message);
	else notices.delete(id);
	eleNoticeList.replaceChildren();
	for (const [key, text] of notices) {
		const item = document.createElement("article"),
			content = document.createElement("span"),
			dismiss = document.createElement("button");
		item.className = key === "error" ? "notice-item error" : "notice-item";
		item.setAttribute("role", key === "error" ? "alert" : "status");
		content.textContent = text;
		dismiss.type = "button";
		dismiss.className = "outline secondary project-remove";
		dismiss.setAttribute("aria-label", "Dismiss notice");
		dismiss.innerHTML = '<i data-lucide="x"></i>';
		dismiss.addEventListener("click", () => setNotice(key, ""));
		item.append(content, dismiss);
		eleNoticeList.append(item);
	}
	eleNoticeArea.hidden = notices.size === 0;
	if (window.lucide) window.lucide.createIcons();
}

function setError(message) {
	setNotice("error", message || "");
}

function cachePath(folder) {
	return realFolder[folder] || folder;
}

function chatInfo(f, id) {
	return CACHE.chats[cachePath(f)]?.[id];
}

function currentHarness() {
	return (
		(sessid && chatInfo(folder, sessid)?.harness) || eleHarness.value || "pi"
	);
}

function updateChat(realpath, updatedChat) {
	const { sessid: id, events, resetEvents, ...fields } = updatedChat;
	if (!id) return null;
	realpath = cachePath(realpath);
	let chats = CACHE.chats[realpath];
	if (!chats) {
		chats = {};
		CACHE.chats[realpath] = chats;
	}
	let chat = chats[id];
	if (!chat) {
		chat = chats[id] = {
			sessid: id,
			events: [],
			stats: null,
			queued: "",
			busy: false,
			tstamp: null,
		};
	}
	if (Array.isArray(events)) {
		if (resetEvents || events[0]?.seq === 0) chat.events = [];
		for (const event of events)
			if (Number.isInteger(event?.seq) && event.seq >= 0)
				chat.events[event.seq] = event;
	}

	Object.assign(chat, fields);
	if (realpath === cachePath(folder)) renderSessCard(chat);
	renderProjectLists();
	return chat;
}

function nextEventSeq(events) {
	let max = -1;
	for (const event of events)
		if (Number.isInteger(event?.seq) && event.seq > max) max = event.seq;
	return max + 1;
}

// === polling stuff
const polling = {};
function stopPollSess(f, id) {
	f = cachePath(f);
	const sessions = polling[f];
	if (!sessions || !Object.hasOwn(sessions, id)) return;
	clearTimeout(sessions[id]);
	delete sessions[id];
	if (!Object.keys(sessions).length) delete polling[f];
}

function startPollSess(f, id) {
	f = cachePath(f);
	let sessions = polling[f];
	if (!sessions) {
		sessions = {};
		polling[f] = sessions;
	}
	if (Object.hasOwn(sessions, id)) return;

	const _doPoll = async () => {
		try {
			const chat = chatInfo(f, id);
			const start = nextEventSeq(chat?.events || []);
			const harness = chat?.harness || "pi";
			const response = await fetch(
				`/api/events?folder=${enc(f)}&sessid=${enc(id)}&harness=${enc(harness)}&start_from=${start}&opened=${f === cachePath(folder) && id === sessid ? 1 : 0}`,
			);
			if (!response.ok)
				throw new Error(`events request failed: ${response.status}`);
			const body = await response.json();
			const updated = updateChat(f, { ...body, sessid: id });
			if (f === cachePath(folder) && id === sessid) {
				renderBusy(updated.busy);
				renderEvents(body.events || []);
				renderQueuedPrompts(updated.queued);
				renderStats(updated.stats);
			}
			if (f === cachePath(folder)) renderSessCard(updated);
		} catch {
			// Keep polling after transient errors if the session is still active.
		}
		if (!Object.hasOwn(sessions, id)) return;
		const chat = chatInfo(f, id);
		if (chat?.busy || chat?.queued) sessions[id] = setTimeout(_doPoll, 1000);
		else {
			delete sessions[id];
			if (!Object.keys(sessions).length) delete polling[f];
		}
	};
	sessions[id] = setTimeout(_doPoll, 0);
}

// === rendering
function renderStats(stats) {
	function _formatTokens(n) {
		if (!Number.isFinite(n)) return "?";
		if (n < 1000) return String(n);
		if (n < 10000) return `${(n / 1000).toFixed(1)}k`;
		if (n < 1e6) return `${Math.round(n / 1000)}k`;
		return `${(n / 1e6).toFixed(1)}M`;
	}
	const parts = [""];
	const inputTokens =
		Number(stats?.full_input_tokens || 0) +
		Number(stats?.cache_read_tokens || 0);
	if (inputTokens) parts.push(`in ${_formatTokens(inputTokens)}`);
	if (stats?.full_output_tokens)
		parts.push(`out ${_formatTokens(Number(stats.full_output_tokens))}`);
	const window = Number(stats?.context_window),
		tokens = Number(stats?.context_tokens);
	if (window)
		parts.push(
			`${Number.isFinite(tokens) ? ((tokens / window) * 100).toFixed(1) : "?"}%/${_formatTokens(window)}`,
		);
	if (typeof stats?.full_cost === "number")
		parts.push(`$${stats.full_cost.toFixed(4)}`);
	eleUsage.textContent = parts.join("\u00a0\u00a0\u00a0");
}

function renderSessCard(chat) {
	const button = [...eleSess.querySelectorAll("button[data-sessid]")].find(
		(button) => button.dataset.sessid === chat.sessid,
	);
	if (!button) return false;
	button.querySelector(".session-name").textContent =
		`${chat.harness ? `${chat.harness}. ` : ""}${chat.name || chat.sessid.slice(0, 8)}`;
	let status = button.parentElement.querySelector(".status");
	const pending = Number(chat.tstamp) > Number(chat.last_opened);
	if (!chat.busy && !pending) {
		status?.remove();
		return true;
	}
	if (!status) {
		status = document.createElement("span");
		status.setAttribute("role", "img");
		button.parentElement.append(status);
	}
	status.className = `status${chat.busy ? " busy" : ""}`;
	status.setAttribute("aria-label", chat.busy ? "Busy" : "Pending message");
	return true;
}

function renderSessList(f) {
	const path = cachePath(f),
		list = Object.values(CACHE.chats[path] || {});
	const open = new Map(
		[...eleSess.querySelectorAll("details[data-parent]")].map((d) => [
			d.dataset.parent,
			d.open,
		]),
	);
	eleFolder.textContent = path;
	eleFolder.title = path;
	eleCount.textContent = `${list.length} chat${list.length === 1 ? "" : "s"}`;
	eleSess.replaceChildren();
	const fresh = document.createElement("button");
	fresh.className = "outline secondary";
	fresh.textContent = "+ New chat";
	fresh.addEventListener("click", () => {
		location.hash = "#chat";
		apiLoadSess(folder, null);
		elePrompt.focus();
	});
	eleSess.append(fresh);
	const _sort = (a, b) => {
		const at = Number(a.tstamp) || 0,
			bt = Number(b.tstamp) || 0;
		if (!at) return bt ? -1 : 0;
		if (!bt) return 1;
		return bt - at;
	};
	const byID = new Map(list.map((s) => [s.sessid, s]));
	const _root = (s) => {
		let root = s;
		const seen = new Set([s.sessid]);
		while (byID.has(root.parent_sessid)) {
			if (seen.has(root.parent_sessid)) return s;
			seen.add(root.parent_sessid);
			root = byID.get(root.parent_sessid);
		}
		return root;
	};
	const groups = new Map();
	for (const s of list) {
		const root = _root(s);
		if (!groups.has(root.sessid)) groups.set(root.sessid, []);
		if (root.sessid !== s.sessid) groups.get(root.sessid).push(s);
	}
	const _card = (s) => {
		const row = document.createElement("div"),
			b = document.createElement("button");
		row.className = "session-row";
		b.className = s.sessid === sessid ? "" : "outline";
		b.title = s.sessid;
		b.dataset.sessid = s.sessid;
		const name = document.createElement("span");
		name.className = "session-name";
		name.textContent = `${s.harness ? `${s.harness}. ` : ""}${s.name || s.sessid.slice(0, 8)}`;
		b.append(name);
		const pending = Number(s.tstamp) > Number(s.last_opened);
		if (s.busy || pending) {
			const status = document.createElement("span");
			status.className = `status${s.busy ? " busy" : ""}`;
			status.setAttribute("role", "img");
			status.setAttribute("aria-label", s.busy ? "Busy" : "Pending message");
			row.append(status);
		}
		b.addEventListener("click", () => {
			location.hash = "#chat";
			apiLoadSess(folder, s.sessid);
			elePrompt.focus();
		});
		row.prepend(b);
		return row;
	};
	for (const s of list
		.filter((item) => _root(item).sessid === item.sessid)
		.sort(_sort)) {
		eleSess.append(_card(s));
		const children = (groups.get(s.sessid) || []).sort(_sort);
		if (!children.length) continue;
		const details = document.createElement("details");
		details.dataset.parent = s.sessid;
		details.open = open.get(s.sessid) ?? false;
		const summary = document.createElement("summary");
		summary.textContent = `${children.length} child chat${children.length === 1 ? "" : "s"}`;
		details.append(summary);
		for (const child of children) {
			const row = document.createElement("div");
			row.className = "session-child";
			row.append(_card(child));
			details.append(row);
		}
		eleSess.append(details);
	}
	if (!list.length)
		eleSess.append(document.createTextNode("No chats in this worktree"));
	if (window.lucide) window.lucide.createIcons();
}

function renderModelList() {
	const q = eleModel.value.trim().toLowerCase();
	eleModelList.replaceChildren(new Option("Default model", ""));
	for (const m of Object.keys(models[currentHarness()] || {}))
		if (m.toLowerCase().includes(q)) eleModelList.append(new Option(m, m));
}

function renderReasoningList(useSaved = false) {
	const levels = models[currentHarness()]?.[eleModel.value.trim()] || [];
	const saved = config[`${currentHarness()}_default_reasoning`] || "";
	const preferred = useSaved ? saved : eleReasoning.value;
	const selected = levels.includes(preferred)
		? preferred
		: levels.includes(saved)
			? saved
			: levels[0] || "";
	eleReasoning.replaceChildren();
	for (const level of levels) eleReasoning.append(new Option(level, level));
	eleReasoning.value = selected;
	eleReasoning.disabled = levels.length === 0;
}

function renderQueuedPrompts(queue) {
	const text = typeof queue === "string" ? queue : "";
	eleQueued.textContent = text;
	eleQueued.hidden = !text;
}

function renderEvents(list) {
	function _images(parent, e) {
		for (const src of e.images || []) {
			if (typeof src !== "string" || !src.startsWith("data:image/")) continue;
			const img = document.createElement("img");
			img.src = src;
			img.alt = "Chat image";
			img.loading = "lazy";
			img.className = "chat-image";
			parent.append(img);
		}
	}
	function _messageMeta(e) {
		return [localISO(Number(e.timestamp)), e.model || ""]
			.filter(Boolean)
			.join(" · ");
	}

	function _card(role, meta) {
		const a = document.createElement("article");
		a.classList.add(
			role === "YOU"
				? "you"
				: role === "SYSTEM"
					? "system"
					: role === "ERROR"
						? "error"
						: "assistant",
		);
		if (role === "THINKING") a.classList.add("thinking");
		const h = document.createElement(
			role === "THINKING" ? "summary" : "header",
		);
		h.className = "row";
		const label = document.createElement("div");
		const strong = document.createElement("strong");
		strong.textContent = role;
		const small = document.createElement("small");
		small.textContent = meta || "";
		label.append(strong, document.createTextNode(" "), small);
		h.append(label);
		const body = document.createElement("div");
		if (role === "THINKING") {
			const details = document.createElement("details");
			details.append(h, body);
			a.append(details);
		} else a.append(h, body);
		eleChat.append(a);
		return body;
	}

	function _toolKey(e) {
		return e.tool_call_id || `seq:${e.seq}`;
	}

	function _toolStart(e) {
		const k = _toolKey(e),
			detail = (e.content || "").trim(),
			name = e.tool_name || "tool";
		const old = tools.get(k);
		if (old) {
			if (detail) {
				old.pre.textContent = detail;
				if (!old.hasTitleDetail) {
					old.titleDetail.textContent = `  ${detail}`;
					old.title.title = detail;
					old.hasTitleDetail = true;
				}
			}
			return;
		}
		_ensureEvents();
		const li = document.createElement("div");
		li.className = "tool";
		const row = document.createElement("summary");
		row.className = "tool-row";
		const title = document.createElement("span");
		title.className = "t";
		const mark = document.createElement("span");
		mark.textContent = "? ";
		const strong = document.createElement("strong");
		strong.textContent = name;
		title.append(mark, strong);
		const titleDetail = document.createElement("span");
		titleDetail.className = "tool-detail";
		titleDetail.textContent = detail ? `  ${detail}` : "";
		title.append(titleDetail);
		title.title = detail || name;
		row.append(title);
		const det = document.createElement("details");
		const pre = document.createElement("pre");
		pre.textContent = detail;
		det.append(row, pre);
		li.append(det);
		eventsList.append(li);
		tools.set(k, {
			li,
			mark,
			pre,
			title,
			titleDetail,
			hasTitleDetail: !!detail,
		});
		_updateEventsLabel();
	}

	function _toolUpdate(e, state = "") {
		const k = _toolKey(e);
		if (!tools.has(k)) _toolStart({ ...e, content: "" });
		const item = tools.get(k),
			detail = (e.content || "").trim();
		if (detail) item.pre.textContent = detail;
		_images(item.li, e);
		if (state) {
			item.mark.textContent = state === "FAILED" ? "✕ " : "✓ ";
			item.title.classList.toggle("failed", state === "FAILED");
			item.title.classList.toggle("success", state !== "FAILED");
		}
	}

	function _ensureEvents() {
		if (eventsCard) return;
		eventsCard = document.createElement("article");
		eventsCard.className = "events";
		const wrap = document.createElement("details");
		const head = document.createElement("summary");
		head.className = "events-head";
		eventsLabel = document.createElement("small");
		eventsLabel.textContent = "ACTIVITY";
		head.append(eventsLabel);
		eventsList = document.createElement("div");
		eventsList.className = "tools";
		wrap.append(head, eventsList);
		eventsCard.append(wrap);
		eleChat.append(eventsCard);
		_updateEventsLabel();
	}

	function _updateEventsLabel() {
		if (!eventsLabel || !eventsList) return;
		const n = eventsList.querySelectorAll(
			":scope > .tool:not(.thinking)",
		).length;
		const thinking = eventsList.querySelectorAll(":scope > .thinking").length;
		eventsLabel.textContent =
			[
				thinking && `${thinking} thinking`,
				n && `${n} tool${n === 1 ? "" : "s"}`,
			]
				.filter(Boolean)
				.join(" · ") || "ACTIVITY";
	}

	function _closeEvents() {
		eventsCard = eventsList = eventsLabel = null;
	}

	function _thinking(e) {
		_ensureEvents();
		const item = document.createElement("div");
		item.className = "tool thinking";
		const det = document.createElement("details");
		const head = document.createElement("summary");
		head.className = "tool-row";
		const title = document.createElement("strong");
		title.textContent = "Thinking";
		const meta = document.createElement("small");
		meta.textContent = ` . ${_messageMeta(e)}`;
		head.append(title, meta);
		const body = document.createElement("div");
		body.textContent = e.content || "";
		det.append(head, body);
		item.append(det);
		eventsList.append(item);
		_updateEventsLabel();
	}

	function _renderEvent(e) {
		if (typeof e.seq === "number") {
			if (seen.has(e.seq)) return;
			seen.add(e.seq);
			if (e.seq > lastSeq) lastSeq = e.seq;
		}
		if (e.type === "user") {
			setError("");
			clearQueuedPrompts();
			const b = _card("YOU", _messageMeta(e));
			b.textContent = e.content || "";
			_images(b, e);
			_closeEvents();
		} else if (e.type === "thinking") {
			_thinking(e);
		} else if (e.type === "assistant") {
			_closeEvents();
			const b = _card("ASSISTANT", _messageMeta(e));
			b.innerHTML = micromark(e.content || "", {
				extensions: [gfm()],
				htmlExtensions: [gfmHtml()],
			});

            // rendering mermaid only for assistant msg
			b.querySelectorAll("pre > code.language-mermaid").forEach((code) => {
				const pre = code.parentElement, wrapper = document.createElement("div"), toggle = document.createElement("button"), diagram = document.createElement("div");
				wrapper.className = "mermaid-code";
				toggle.className = "outline secondary mermaid-toggle";
				toggle.type = "button";
				toggle.textContent = "Diagram";
				diagram.className = "mermaid-diagram";
				diagram.hidden = true;
				let rendered = false;
				toggle.addEventListener("click", () => {
					if (!pre.hidden) {
						pre.hidden = true;
						diagram.hidden = false;
						toggle.textContent = "Text";
						if (!rendered) {
							_renderMermaid(code.textContent, diagram);
							rendered = true;
						}
					} else {
						pre.hidden = false;
						diagram.hidden = true;
						toggle.textContent = "Diagram";
					}
				});
				pre.replaceWith(wrapper);
				wrapper.append(toggle, pre, diagram);
			});
			_images(b, e);
		} else if (e.type === "tool_start") _toolStart(e);
		else if (e.type === "tool_end")
			_toolUpdate(e, e.tool_failed ? "FAILED" : "DONE");
		else if (e.type === "error") {
			const b = _card("ERROR", "");
			b.textContent = e.content;
			setError(e.content);
		}
	}

	for (const e of list) _renderEvent(e);
	if (chatPinned) eleChat.scrollTop = eleChat.scrollHeight;
}

function renderBusy(busy) {
	eleChatStatus.className = `status${busy ? " busy" : ""}`;
	eleChatStatus.hidden = !busy;
}

function renderHarnessDetails() {
	const harness = currentHarness();
	eleModel.value = config[`${harness}_default_model`] || "";
	renderModelList();
	renderReasoningList(true);
}

function renderReset() {
	renderBusy(false);
	chatPinned = true;
	eleChat.replaceChildren();
	tools = new Map();
	lastSeq = 0;
	nextStartFrom = null;
	seen = new Set();
	clearQueuedPrompts();
	eventsCard = eventsList = eventsLabel = null;
	eleUsage.textContent = "";
}

function renderProjectLists() {
	eleProjects.replaceChildren();
	for (const root of projectRoots) {
		const row = document.createElement("div");
		row.className = "project-row";
		const b = document.createElement("button");
		b.className = root === selectedProject ? "" : "outline";
		b.textContent = root;
		b.title = root;
		b.addEventListener("click", () => apiSelectProject(root));
		const remove = document.createElement("button");
		remove.className = "outline secondary project-remove";
		remove.type = "button";
		remove.innerHTML = '<i data-lucide="x"></i>';
		remove.title = `Remove ${root}`;
		remove.setAttribute("aria-label", `Remove project ${root}`);
		remove.addEventListener("click", async () => {
			if (
				!confirm(
					`Remove this project from ahweb? This won’t delete its files.\n\n${root}`,
				)
			)
				return;
			try {
				const r = await fetch(`/api/delproject?folder=${enc(root)}`, {
					method: "POST",
				});
				const body = await r.json();
				if (!r.ok) throw new Error(body.error || "Could not remove project");
				setError("");
				await apiLoadProjects();
			} catch (err) {
				setError(err.message);
			}
		});
		row.append(b, remove);
		eleProjects.append(row);
	}
	if (!projectRoots.length)
		eleProjects.append(document.createTextNode("No projects yet"));
	eleWorktrees.replaceChildren();
	for (const worktree of worktreePaths) {
		const row = document.createElement("div");
		row.className = "worktree-row";
		const b = document.createElement("button");
		b.className = worktree.dirname === selectedWorktree ? "" : "outline";
		b.textContent = worktree.branch || worktree.dirname;
		b.title = worktree.dirname;
		b.addEventListener("click", () => apiSelectWorktree(worktree.dirname));
		row.append(b);
		const chats = Object.values(CACHE.chats[cachePath(worktree.dirname)] || {});
		const busy = chats.some((chat) => chat.busy),
			pending = chats.some(
				(chat) =>
					!chat.parent_sessid &&
					Number(chat.tstamp) > Number(chat.last_opened),
			);
		if (busy || pending) {
			const status = document.createElement("span");
			status.className = `status${busy ? " busy" : ""}`;
			status.setAttribute("role", "img");
			status.setAttribute("aria-label", busy ? "Busy" : "Pending message");
			row.append(status);
		}
		eleWorktrees.append(row);
	}
	if (selectedProject && !worktreePaths.length)
		eleWorktrees.append(document.createTextNode("No worktrees"));
	const _basename = (path) => path.split(/[\\/]/).filter(Boolean).at(-1) || path;
	eleBreadcrumbProject.textContent = selectedProject ? _basename(selectedProject) : "(empty)";
	const selected = worktreePaths.find((worktree) => worktree.dirname === selectedWorktree);
	eleBreadcrumbWorktree.textContent = selectedWorktree
		? selected?.branch || _basename(selectedWorktree)
		: "(empty)";
	if (window.lucide) window.lucide.createIcons();
}

// === api calls
function apiLoadConfig(harnesses) {
	const keys = harnesses.flatMap((h) => [
		`${h}_default_model`,
		`${h}_default_reasoning`,
	]);
	Promise.all(
		keys.map((key) =>
			fetch(`/api/config?key=${key}`).then((r) => {
				if (!r.ok) throw new Error("Could not load config");
				return r.json();
			}),
		),
	)
		.then((values) => {
			keys.forEach((key, i) => {
				config[key] = values[i] || "";
			});
			renderHarnessDetails();
		})
		.catch((err) => setError(err.message));
}

async function apiSaveModelSetting() {
	const value = eleModel.value.trim();
	const key = `${currentHarness()}_default_model`;
	try {
		const params = new URLSearchParams({ key, value });
		const r = await fetch(`/api/config?${params}`, { method: "POST" });
		const saved = await r.json();
		if (!r.ok) throw new Error(saved.error || "Could not save config");
		config[key] = typeof saved === "string" ? saved : value;
	} catch (err) {
		setError(err.message);
	}
}

async function apiSaveReasoningSetting() {
	const value = eleReasoning.value;
	const key = `${currentHarness()}_default_reasoning`;
	try {
		const params = new URLSearchParams({ key, value });
		const r = await fetch(`/api/config?${params}`, { method: "POST" });
		const saved = await r.json();
		if (!r.ok) throw new Error(saved.error || "Could not save config");
		config[key] = typeof saved === "string" ? saved : value;
	} catch (err) {
		setError(err.message);
	}
}

function apiLoadModels() {
	eleModel.disabled = true;
	fetch("/api/models")
		.then((r) => {
			if (!r.ok) throw new Error("Could not load models");
			return r.json();
		})
		.then((body) => {
			models = body;
			const harnesses = Object.keys(body),
				selected = eleHarness.value;
			eleHarness.replaceChildren(...harnesses.map((h) => new Option(h, h)));
			eleHarness.value = harnesses.includes(selected)
				? selected
				: harnesses[0] || "";
			apiLoadConfig(harnesses);
			renderHarnessDetails();
		})
		.catch((err) => {
			models = {};
			renderHarnessDetails();
			setError(err.message);
		})
		.finally(() => {
			eleModel.disabled = false;
		});
}
window.apiLoadModels = apiLoadModels;
{
	const dialog = $("#cleanup-dialog"),
		daysInput = $("#cleanup-days"),
		results = $("#cleanup-results"),
		previewButton = $("#cleanup-preview"),
		runButton = $("#cleanup-run");
	let preview = null;
	const _reset = (clear = false) => {
		preview = null;
		if (clear) results.replaceChildren();
		runButton.disabled = true;
	};
	const _render = (counts) => {
		results.replaceChildren();
		const rows = Object.entries(counts || {});
		if (!rows.length) {
			results.textContent = "No managed project folders";
			return;
		}
		const table = document.createElement("table");
		table.innerHTML =
			"<thead><tr><th>Folder</th><th>Delete</th></tr></thead><tbody></tbody>";
		const body = table.tBodies[0];
		for (const [folder, count] of rows) {
			const row = body.insertRow();
			row.insertCell().textContent = folder;
			row.insertCell().textContent = String(count);
		}
		results.append(table);
	};
	const _request = async (params, confirm, plan = "") => {
		const query = new URLSearchParams({
			...params,
			confirm: confirm ? "1" : "0",
		});
		if (plan) query.set("plan", plan);
		const r = await fetch(`/cleanup?${query}`, { method: "POST" });
		const body = await r.json();
		if (!r.ok || r.status === 207) {
			const err = new Error(body.error || "Could not delete old sessions");
			err.counts = r.status === 207 ? body.data || null : null;
			throw err;
		}
		return { body: body.data, plan: body.planid };
	};
	const _preview = async () => {
		const days = daysInput.valueAsNumber;
		if (!Number.isInteger(days) || days < 0)
			return setError("Days must be a nonnegative integer");
		previewButton.disabled = runButton.disabled = true;
		try {
			const params = { days: String(days) };
			const res = await _request(params, false);
			preview = { params, plan: res.plan };
			_render(res.body);
			setError("");
			runButton.disabled = !res.plan;
		} catch (err) {
			setError(err.message);
		} finally {
			previewButton.disabled = false;
		}
	};
	const _run = async () => {
		if (!preview) return;
		previewButton.disabled = runButton.disabled = true;
		try {
			const res = await _request({}, true, preview.plan);
			preview = null;
			_render(res.body);
			setError("");
			dialog.close();
			setNotice("cleanup", "Old sessions deleted");
			await apiLoadProjects();
		} catch (err) {
			preview = null;
			if (err.counts) _render(err.counts);
			setError(err.message);
		} finally {
			previewButton.disabled = false;
			runButton.disabled = !preview?.plan;
		}
	};
	$("#cleanup-sessions").addEventListener("click", () => {
		_reset(true);
		dialog.showModal();
	});
	$("#cleanup-close").addEventListener("click", () => dialog.close());
	previewButton.addEventListener("click", _preview);
	runButton.addEventListener("click", _run);
	daysInput.addEventListener("input", () => _reset());
	dialog.addEventListener("close", () => _reset());
}
async function apiLoadProjects() {
	try {
		const r = await fetch("/api/config"),
			body = await r.json();
		if (!r.ok) throw new Error(body.error || "Could not load projects");
		projectRoots = (body.git_root_real_dirs || "")
			.split(/[\r\n]+/)
			.filter(Boolean);
		renderProjectLists();
		if (projectRoots.length) {
			await apiSelectProject(
				projectRoots.includes(selectedProject)
					? selectedProject
					: projectRoots[0],
			);
		} else {
			selectedProject = selectedWorktree = folder = "";
			worktreePaths = [];
			renderProjectLists();
			eleDir.value = "";
			eleSess.replaceChildren();
			eleSess.removeAttribute("hx-get");
			eleCount.textContent = eleFolder.textContent = "";
			apiLoadSess("", "");
			window.apiLoadFiles("");
			window.apiLoadGit("");
		}
	} catch (err) {
		setError(err.message);
	}
}

async function apiSelectProject(root) {
	selectedProject = root;
	selectedWorktree = "";
	window.apiLoadFiles("");
	window.apiLoadGit("");
	worktreePaths = [];
	renderProjectLists();
	try {
		const r = await fetch(`/api/worktrees?folder=${enc(root)}`),
			body = await r.json();
		if (!r.ok) throw new Error(body.error || "Could not load worktrees");
		worktreePaths = body.worktrees || [];
		selectedWorktree = worktreePaths.some(
			(worktree) => worktree.dirname === folder,
		)
			? folder
			: worktreePaths[0]?.dirname || "";
		apiLoadSess(selectedWorktree || root, "");

		// here i want to load chats from all worktrees so that i can
		// show correct status for that worktree. also keep listening to busy chats
		await Promise.allSettled(
			worktreePaths
				.filter((worktree) => worktree.dirname !== selectedWorktree)
				.map(async ({ dirname }) => {
					const r = await fetch(`/api/list?folder=${enc(dirname)}`),
						body = await r.json();
					if (!r.ok)
						throw new Error(
							body.error || `Could not load chats for ${dirname}`,
						);
					const realpath = body.folder || dirname;
					realFolder[dirname] = realpath;
					let chats = CACHE.chats[realpath];
					if (!chats) chats = CACHE.chats[realpath] = {};
					for (const { last_timestamp, ...session } of body.sessions || []) {
						let chat = chats[session.sessid];
						if (!chat) {
							chat = chats[session.sessid] = { events: [], stats: null };
						}
						Object.assign(chat, session, { tstamp: last_timestamp });
						if (session.busy) startPollSess(realpath, session.sessid);
					}
				}),
		);
		renderProjectLists();
		if (matchMedia("(max-width: 600px)").matches)
			sideSections.forEach((section, i) => {
				section.open = i === 1;
			});
		if (selectedWorktree) apiLoadSessList(selectedWorktree);
	} catch (err) {
		setError(err.message);
	}
}

function apiSelectWorktree(path) {
	selectedWorktree = path;
	apiLoadSess(path, "");
	apiLoadSessList(path);
	if (matchMedia("(max-width: 600px)").matches)
		sideSections.forEach((section, i) => {
			section.open = i === 2;
		});
}

function apiLoadSessList(f) {
	folder = (f || selectedWorktree || eleDir.value).trim();
	selectedWorktree = folder;
	eleDir.value = folder;
	eleDir.dispatchEvent(new Event("change"));
	renderProjectLists();
	eleSess.setAttribute("hx-get", `/api/list?folder=${enc(folder)}`);
	htmx.process(eleSess);
	if (CACHE.chats[cachePath(folder)]) renderSessList(folder);
	setError("");
	htmx.trigger(eleSess, "refresh");
	window.apiLoadFiles(folder);
	window.apiLoadGit(folder);
}

function apiLoadSess(f, id) {
	// Render cached events, then refresh from the cache tail.
	f = (f || eleDir.value).trim();
	folder = f;
	sessid = id;
	const url = new URL(location.href);
	if (id) url.searchParams.set("sessid", id);
	else url.searchParams.delete("sessid");
	history.replaceState(history.state, "", url);
	renderReset();
	if (id) {
		const harness = chatInfo(folder, id)?.harness || "pi";
		eleHarness.value = harness;
		eleHarness.disabled = true;
		eleChat.setAttribute(
			"hx-get",
			`/api/events?folder=${enc(folder)}&sessid=${enc(id)}&harness=${enc(harness)}`,
		);
	} else {
		eleHarness.disabled = false;
		eleChat.removeAttribute("hx-get");
	}
	renderHarnessDetails();
	htmx.process(eleChat);
	for (const button of eleSess.querySelectorAll("button[data-sessid]"))
		button.classList.toggle("outline", button.dataset.sessid !== id);
	if (!id) {
		setError("");
		return;
	}
	const chat = chatInfo(folder, id),
		hit = chat?.events;
	nextStartFrom = hit?.length ? Math.max(0, nextEventSeq(hit) - 2) : 0;
	if (hit?.length) {
		renderEvents(hit.filter(Boolean));
		eleChat.scrollTop = eleChat.scrollHeight;
	}
	renderBusy(chatInfo(folder, id)?.busy);
	setError("");
	htmx.trigger(eleChat, "refresh");
}

document.body.addEventListener("htmx:configRequest", (e) => {
	if (e.target === eleChat) {
		const start = nextStartFrom ?? lastSeq;
		e.detail.parameters.start_from = start;
		e.detail.parameters.opened = sessid ? 1 : 0;
		nextStartFrom = null;
	}
});

document.body.addEventListener("htmx:afterRequest", (e) => {
	const q = e.detail.requestConfig?.parameters || {};
	if (e.target === eleSess) {
		const listedFolder = q.folder || folder;
		const xhr = e.detail.xhr;
		if (!xhr || !xhr.response) return;
		let body;
		try {
			body = JSON.parse(xhr.response);
		} catch {
			return;
		}
		const realpath = body.folder || listedFolder;
		const oldChats = CACHE.chats[listedFolder];
		realFolder[listedFolder] = realpath;
		let chats = CACHE.chats[realpath];
		if (!chats) {
			chats = {};
			CACHE.chats[realpath] = chats;
		}
		if (oldChats && oldChats !== chats) {
			Object.assign(chats, oldChats);
			delete CACHE.chats[listedFolder];
		}
		const sessions = body.sessions || [],
			listed = new Set(sessions.map((s) => s.sessid));
		for (const id of Object.keys(chats))
			if (!listed.has(id)) {
				const chat = chats[id];
				if (chat.busy || chat.queued) continue;
				stopPollSess(realpath, id);
				delete chats[id];
			}
		for (const session of sessions) {
			const { last_timestamp, ...fields } = session;
			let chat = chats[session.sessid];
			if (!chat) {
				chat = chats[session.sessid] = {
					sessid: session.sessid,
					events: [],
					stats: null,
					queued: "",
					busy: false,
					tstamp: null,
				};
			}
			Object.assign(chat, fields, { tstamp: last_timestamp });
		}
		for (const session of sessions)
			if (session.busy) startPollSess(realpath, session.sessid);
		renderProjectLists();
		if (listedFolder !== folder) return;
		renderSessList(listedFolder);
	} else if (e.target === eleChat) {
		const responseFolder = q.folder || folder;
		const responseSession = q.sessid || sessid;
		const xhr = e.detail.xhr;
		if (!xhr || !xhr.response) return;
		let body;
		try {
			body = JSON.parse(xhr.response);
		} catch {
			return;
		}
		if (!responseSession) return;
		const chat = updateChat(responseFolder, {
			...body,
			sessid: responseSession,
			resetEvents: Number(q.start_from) === -1,
		});
		if (responseFolder === folder && responseSession === sessid)
			startPollSess(responseFolder, responseSession);
		if (responseFolder !== folder || responseSession !== sessid) return;
		if (Number(q.start_from) === -1) renderReset();
		renderEvents(body.events || []);
		renderBusy(chat.busy);
		renderQueuedPrompts(chat.queued);
		renderStats(chat.stats);
	}
});

const projectDialog = $("#project-dialog"),
	projectPath = $("#project-path"),
	projectBrowser = $("#project-browser");
$("#add-project").addEventListener("click", () => {
	projectPath.value = folder || ".";
	projectDialog.showModal();
});
$("#close-project").addEventListener("click", () => projectDialog.close());
$("#browse-project").addEventListener("click", async () => {
	try {
		const params = new URLSearchParams({
			folder: projectPath.value.trim(),
			path: ".",
		});
		const r = await fetch(`/api/file?${params}`);
		const body = await r.json();
		if (!r.ok) throw new Error(body.error || "Could not list folders");
		projectBrowser.replaceChildren();
		for (const rel of body.dirs || []) {
			const b = document.createElement("button");
			b.className = "outline";
			b.textContent = `${rel.split("/").at(-1)}/`;
			b.addEventListener("click", () => {
				const base = projectPath.value.replace(/\/+$/, "");
				projectPath.value = `${base}/${rel.split("/").at(-1)}`;
				$("#browse-project").click();
			});
			projectBrowser.append(b);
		}
		if (!(body.dirs || []).length)
			projectBrowser.textContent = "No folders here";
	} catch (err) {
		setError(err.message);
	}
});
$("#save-project").addEventListener("click", async () => {
	try {
		const r = await fetch(
			`/api/addproject?folder=${enc(projectPath.value.trim())}`,
			{ method: "POST" },
		);
		const body = await r.json();
		if (!r.ok) throw new Error(body.error || "Could not add project");
		projectDialog.close();
		setError("");
		await apiLoadProjects();
		await apiSelectProject(body.folder);
	} catch (err) {
		setError(err.message);
	}
});
const sideSections = [...document.querySelectorAll(".side-section")];
if (matchMedia("(max-width: 600px)").matches)
	sideSections.forEach((section, i) => {
		section.open = i === 0;
	});
for (const section of sideSections)
	section.addEventListener("toggle", () => {
		if (matchMedia("(max-width: 600px)").matches && section.open)
			for (const other of sideSections)
				if (other !== section) other.open = false;
	});
for (const [crumb, section] of [
	[eleBreadcrumbProject, sideSections[0]],
	[eleBreadcrumbWorktree, sideSections[1]],
])
	crumb.addEventListener("click", () => {
		location.hash = "#sess";
		sideSections.forEach((item) => (item.open = item === section));
	});
eleRefresh.addEventListener("click", () => {
	if (!sessid) return;
	nextStartFrom = -1;
	setError("");
	htmx.trigger(eleChat, "refresh");
});
document.querySelector("#workspace-refresh").addEventListener("click", () => {
	window.dispatchEvent(new Event("redraw-cli"));
	window.apiLoadFiles(folder);
	window.apiLoadGit(folder);
	window.apiLoadStatus();
	apiLoadModels();
});
function growPrompt() {
	elePrompt.style.height = "auto";
	elePrompt.style.height = `${elePrompt.scrollHeight}px`;
}
elePrompt.addEventListener("input", growPrompt);
growPrompt();
elePrompt.addEventListener("keydown", (e) => {
	if (
		e.key === "Enter" &&
		!e.shiftKey &&
		!matchMedia("(max-width: 600px)").matches
	) {
		e.preventDefault();
		e.target.form.requestSubmit();
	}
});
eleModel.addEventListener("input", () => {
	renderModelList();
	renderReasoningList();
});
$("#add-image").addEventListener("click", () => eleImageInput.click());
eleModel.addEventListener("change", apiSaveModelSetting);
eleReasoning.addEventListener("change", apiSaveReasoningSetting);
eleHarness.addEventListener("change", renderHarnessDetails);

// chunk for UI for image for prompts
async function addPromptImages(files) {
	for (const file of files) {
		if (!file?.type.startsWith("image/")) continue;
		try {
			const uri = await new Promise((resolve, reject) => {
				const reader = new FileReader();
				reader.onload = () => resolve(reader.result);
				reader.onerror = () => reject(new Error("Could not read image"));
				reader.readAsDataURL(file);
			});
			promptImages.push({ file, uri });
		} catch (err) {
			setError(err.message);
		}
	}
	const _render = () => {
		eleImagePreview.replaceChildren();
		promptImages.forEach(({ file, uri }, index) => {
			const item = document.createElement("span");
			item.className = "image-preview-item";
			const image = document.createElement("img");
			image.src = uri;
			image.alt = file.name;
			const remove = document.createElement("button");
			remove.className = "secondary outline";
			remove.textContent = "×";
			remove.setAttribute("aria-label", `Remove ${file.name}`);
			remove.addEventListener("click", () => {
				promptImages.splice(index, 1);
				_render();
			});
			item.append(image, remove);
			eleImagePreview.append(item);
		});
		eleImagePreview.hidden = promptImages.length === 0;
	};
	_render();
	if (promptImages.length) setError("");
}

eleImageInput.addEventListener("change", async () => {
	await addPromptImages(eleImageInput.files || []);
	eleImageInput.value = "";
});
elePrompt.addEventListener("paste", (e) => {
	const files = [...(e.clipboardData?.items || [])]
		.filter((item) => item.kind === "file" && item.type.startsWith("image/"))
		.map((item) => item.getAsFile())
		.filter(Boolean);
	if (!files.length) return;
	e.preventDefault();
	addPromptImages(files);
});

$("#prompt-form").addEventListener("submit", async (e) => {
	e.preventDefault();
	const msg = elePrompt.value.trim();
	folder = eleDir.value.trim();
	if (!msg || !folder) return;
	const model = eleModel.value.trim();
	if (model && !Object.hasOwn(models[currentHarness()] || {}, model)) {
		setError("Unknown model \u2013 pick from list or clear for Default");
		return;
	}
	setError("");
	const send = $("#send"),
		sendIcon = send.innerHTML;
	send.disabled = true;
	send.innerHTML = '<i data-lucide="loader-circle" aria-label="Sending"></i>';
	if (window.lucide) window.lucide.createIcons();
	elePrompt.value = "";
	growPrompt();
	const isNew = !sessid;
	try {
		const r = await fetch("/api/prompt", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				folder,
				sessid: sessid || "",
				prompt: msg,
				images: promptImages.map(({ uri }) => uri),
				model,
				reasoning: eleReasoning.value,
				harness: sessid
					? chatInfo(folder, sessid)?.harness || "pi"
					: currentHarness(),
			}),
		});
		const b = await r.json();
		if (!r.ok) throw new Error(b.error || "Request failed");
		promptImages = [];
		eleImagePreview.replaceChildren();
		eleImagePreview.hidden = true;
		const chat = updateChat(folder, { ...b, busy: true });
		renderBusy(chat.busy);
		renderQueuedPrompts(chat.queued);
		if (isNew) {
			apiLoadSess(folder, b.sessid);
			apiLoadSessList(folder);
		} else {
			htmx.trigger(eleChat, "refresh");
		}
	} catch (err) {
		elePrompt.value = msg;
		growPrompt();
		setError(err.message);
	} finally {
		send.disabled = false;
		send.innerHTML = sendIcon;
	}
});

$("#stop").addEventListener("click", async (e) => {
	if (!sessid) return;
	const f = folder,
		id = sessid,
		harness = chatInfo(folder, sessid)?.harness || "",
		button = e.currentTarget;
	stopPollSess(f, id);
	button.disabled = true;
	const stopIcon = button.innerHTML;
	button.innerHTML =
		'<i data-lucide="loader-circle" aria-label="Stopping"></i>';
	if (window.lucide) window.lucide.createIcons();
	try {
		const r = await fetch(
			`/api/stop?folder=${enc(f)}&sessid=${enc(id)}&harness=${enc(harness)}`,
			{
				method: "POST",
			},
		);
		const b = await r.json();
		if (!r.ok) throw new Error(b.error || "Request failed");
		const chat = updateChat(f, { ...b, sessid: id });
		if (f === folder && id === sessid) {
			if (chat.queued) elePrompt.value = chat.queued;
			growPrompt();
			renderQueuedPrompts("");
			htmx.trigger(eleChat, "refresh");
		}
		apiLoadSessList(f);
	} catch (err) {
		startPollSess(f, id);
		setError(err.message);
	} finally {
		button.disabled = false;
		button.innerHTML = stopIcon;
	}
});

const collapseBtn = $("#collapse-all");
collapseBtn.addEventListener("click", () => {
	const blocks = [
		...eleChat.querySelectorAll(
			"article.thinking details, article.events > details",
		),
	];
	const open = blocks.some((block) => block.open);
	for (const block of blocks) block.open = !open;
	const label = open ? "Expand all" : "Collapse all";
	collapseBtn.title = label;
	collapseBtn.setAttribute("aria-label", label);
	const icon = collapseBtn.querySelector("i");
	icon.dataset.lucide = open ? "maximize-2" : "minimize-2";
	if (window.lucide) window.lucide.createIcons();
});

apiLoadModels();
window.apiLoadStatus?.()?.then?.(() => apiLoadModels());
setInterval(() => {
	if (folder) htmx.trigger(eleSess, "refresh");
}, 30_000);
apiLoadProjects();
