const $ = (selector) => document.querySelector(selector);

const workspaceTabs = $("#workspace-tabs");
const isMobile = matchMedia("(max-width: 600px)").matches;
const PANELS = {
	sess: "sessions",
	chat: "chats",
	"tab-files": "tab-files",
	"tab-cli": "tab-cli",
	"tab-git": "tab-git",
	"tab-status": "tab-status",
};
const _resizeViewport = () =>
	document.body.style.setProperty(
		"--app-height",
		`${window.visualViewport?.height ?? window.innerHeight}px`,
	);
_resizeViewport();
window.addEventListener("resize", _resizeViewport);
window.visualViewport?.addEventListener("resize", _resizeViewport);
for (const [id, label] of [
	["sessions", "Sess"],
	["chats", "Chat"],
]) {
	const button = document.createElement("button");
	button.dataset.mobilePanel = id;
	button.setAttribute("aria-controls", `panel-${id}`);
	button.textContent = label;
	workspaceTabs.insertBefore(
		button,
		workspaceTabs.querySelector("[data-workspace-tab]"),
	);
}

workspaceTabs.addEventListener("click", (e) => {
	const tab =
		e.target.closest("#workspace-tabs > [data-mobile-panel]") ||
		e.target.closest("#workspace-tabs > [data-workspace-tab]");
	if (!tab) return;
	const panel = tab.dataset.mobilePanel || tab.dataset.workspaceTab;
	location.hash = Object.keys(PANELS).find((hash) => PANELS[hash] === panel);
});

let desktopTab = ["#tab-files", "#tab-git", "#tab-status"].includes(
	location.hash,
)
	? location.hash.slice(1)
	: "tab-cli";

const _activateTab = () => {
	const hash = location.hash.slice(1) || (isMobile ? "sess" : "tab-cli");
	if (["tab-files", "tab-git", "tab-cli", "tab-status"].includes(hash))
		desktopTab = hash;
	const panel = PANELS[!isMobile && hash === "chat" ? desktopTab : hash];
	if (!panel) return;
	document.body.dataset.mobilePanel = panel;
	for (const tab of workspaceTabs.querySelectorAll(
		"[data-mobile-panel], [data-workspace-tab]",
	))
		tab.setAttribute(
			"aria-selected",
			String((tab.dataset.mobilePanel || tab.dataset.workspaceTab) === panel),
		);
	for (const pane of document.querySelectorAll(
		"#panel-workspace > [role=tabpanel]",
	))
		pane.hidden = pane.id !== panel;
};
window.addEventListener("hashchange", _activateTab);
_activateTab();

const layoutToggle = $("#layout-toggle");
layoutToggle.addEventListener("click", () => {
	const chatFirst = document.body.classList.toggle("chat_first");
	try {
		localStorage.setItem("chat_first", String(chatFirst));
	} catch {}
});

for (const [index, resizer] of document
	.querySelectorAll(".panel-resizer")
	.entries()) {
	const _panel = () => {
		const chatFirst = document.body.classList.contains("chat_first");
		return $((index === 0) === chatFirst ? "#panel-chats" : "#panel-sessions");
	};
	const _property = () =>
		_panel().id === "panel-chats" ? "--chats-width" : "--sessions-width";
	try {
		const stored = localStorage.getItem(_property());
		const saved = stored === null ? NaN : Number(stored);
		if (saved >= 0) document.body.style.setProperty(_property(), `${saved}px`);
	} catch {}

	const _resize = (width, save = false) => {
		const panel = _panel();
		const property = _property();
		const otherWidth = $(
			panel.id === "panel-chats" ? "#panel-sessions" : "#panel-chats",
		).getBoundingClientRect().width;
		const maxWidth = window.innerWidth - 12 - otherWidth;
		width = Math.max(0, Math.min(width, maxWidth));
		document.body.style.setProperty(property, `${width}px`);
		resizer.setAttribute("aria-valuemin", 0);
		resizer.setAttribute("aria-valuemax", Math.floor(maxWidth));
		resizer.setAttribute("aria-valuenow", Math.round(width));
		if (save)
			try {
				localStorage.setItem(property, String(Math.round(width)));
			} catch {}
	};
	if (!isMobile) _resize(_panel().getBoundingClientRect().width);
	resizer.addEventListener("pointerdown", (e) => {
		if (e.button !== 0) return;
		resizer.setPointerCapture(e.pointerId);
		const startX = e.clientX;
		const startWidth = _panel().getBoundingClientRect().width;
		const _move = (moveEvent) =>
			_resize(
				startWidth + (index === 0 ? 1 : -1) * (moveEvent.clientX - startX),
			);
		const _up = () => {
			_resize(_panel().getBoundingClientRect().width, true);
			resizer.removeEventListener("pointermove", _move);
			resizer.removeEventListener("pointerup", _up);
			resizer.removeEventListener("pointercancel", _up);
		};
		resizer.addEventListener("pointermove", _move);
		resizer.addEventListener("pointerup", _up);
		resizer.addEventListener("pointercancel", _up);
	});
	resizer.addEventListener("keydown", (e) => {
		const direction =
			e.key === "ArrowRight" ? 1 : e.key === "ArrowLeft" ? -1 : 0;
		if (!direction) return;
		e.preventDefault();
		_resize(
			_panel().getBoundingClientRect().width +
				direction * (index === 0 ? 1 : -1) * 16,
			true,
		);
	});
}

for (const resizer of document.querySelectorAll(".split-resizer")) {
	const top = resizer.previousElementSibling;
	const bottom = resizer.nextElementSibling;
	if (!top || !bottom) continue;
	const key = `split-height:${top.id}`;
	try {
		const saved = localStorage.getItem(key);
		if (saved !== null && !Number.isNaN(Number(saved))) {
			top.style.flex = "0 0 auto";
			top.style.height = `${Number(saved)}px`;
		}
	} catch {}

	const _resize = (height, save = false) => {
		const containerHeight =
			resizer.parentElement.getBoundingClientRect().height;
		const max = Math.max(0, containerHeight - 6);
		height = Math.max(0, Math.min(height, max));
		top.style.flex = "0 0 auto";
		top.style.height = `${height}px`;
		resizer.setAttribute("aria-valuemin", 0);
		resizer.setAttribute("aria-valuemax", Math.floor(max));
		resizer.setAttribute("aria-valuenow", Math.round(height));
		if (save)
			try {
				localStorage.setItem(key, String(Math.round(height)));
			} catch {}
	};

	resizer.addEventListener("pointerdown", (e) => {
		if (e.button !== 0) return;
		resizer.setPointerCapture(e.pointerId);
		const startY = e.clientY;
		const startHeight = top.getBoundingClientRect().height;
		const _move = (moveEvent) =>
			_resize(startHeight + (moveEvent.clientY - startY));
		const _up = () => {
			_resize(top.getBoundingClientRect().height, true);
			resizer.removeEventListener("pointermove", _move);
			resizer.removeEventListener("pointerup", _up);
			resizer.removeEventListener("pointercancel", _up);
		};
		resizer.addEventListener("pointermove", _move);
		resizer.addEventListener("pointerup", _up);
		resizer.addEventListener("pointercancel", _up);
	});
	resizer.addEventListener("keydown", (e) => {
		const direction = e.key === "ArrowDown" ? 1 : e.key === "ArrowUp" ? -1 : 0;
		if (!direction) return;
		e.preventDefault();
		_resize(top.getBoundingClientRect().height + direction * 16, true);
	});
}
