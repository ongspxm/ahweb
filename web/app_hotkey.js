const dialog = document.querySelector("#actions-dialog"),
	hintsToggle = document.querySelector("#hotkey-hints-toggle"),
	hintsLegend = document.querySelector("#hotkey-hints-legend"),
	list = document.querySelector("#actions-list"),
	leaderInput = document.querySelector("#hotkey-leader"),
	storageKey = "hotkey-leader";
let leader = "Ctrl+'",
	waiting = false;
try {
	leader = localStorage.getItem(storageKey) || leader;
} catch {}
leaderInput.value = leader;

function _toggleHotkeyHints(force) {
	const enabled = force ?? !document.body.classList.contains("show-hotkeys");
	document.body.classList.toggle("show-hotkeys", enabled);
	hintsToggle.setAttribute("aria-pressed", String(enabled));
	hintsToggle.title = enabled ? "Hide hotkey hints" : "Show hotkey hints";
	hintsToggle.setAttribute("aria-label", hintsToggle.title);
	hintsLegend.hidden = !enabled;
}

const actions = [
	[
		"New chat",
		"Ctrl+Enter",
		() =>
			[...document.querySelectorAll("#chats-list > button")]
				.find((b) => b.textContent.includes("New chat"))
				?.click(),
		"#chats-list > button",
	],
	[
		"Stop chat",
		"Ctrl+ ",
		() => document.querySelector("#stop")?.click(),
		"#stop",
	],
	[
		"Go to prompt",
		"h",
		() => document.querySelector("#prompt")?.focus(),
		"#prompt-wrap",
	],
	[
		"Open Files",
		"j",
		() => document.querySelector('[data-workspace-tab="tab-files"]')?.click(),
		'[data-workspace-tab="tab-files"]',
	],
	[
		"Open Git",
		"k",
		() => document.querySelector('[data-workspace-tab="tab-git"]')?.click(),
		'[data-workspace-tab="tab-git"]',
	],
	[
		"Open CLI",
		"l",
		() => {
			document.querySelector('[data-workspace-tab="tab-cli"]')?.click();
			setTimeout(
				() =>
					document
						.querySelector("#cli-terminal .xterm-helper-textarea")
						?.focus(),
				0,
			);
		},
		'[data-workspace-tab="tab-cli"]',
	],
	[
		"Open Status",
		null,
		() => document.querySelector('[data-workspace-tab="tab-status"]')?.click(),
	],
	[
		"CLI font up",
		"g",
		() => document.querySelector('#cli-keys [data-font-size="1"]')?.click(),
		'#cli-keys [data-font-size="1"]',
	],
	[
		"CLI font down",
		"f",
		() => document.querySelector('#cli-keys [data-font-size="-1"]')?.click(),
		'#cli-keys [data-font-size="-1"]',
	],
	[
		"Refresh workspace",
		"r",
		() => document.querySelector("#workspace-refresh")?.click(),
		"#workspace-refresh",
	],
	[
		"Go to project",
		"p",
		() =>
			(
				document.querySelector(
					"#projects-list .project-row > button:not(.outline)",
				) || document.querySelector("#projects-list .project-row > button")
			)?.focus(),
		"#projects-panel > summary",
	],
	[
		"Go to worktree",
		"w",
		() =>
			(
				document.querySelector(
					"#worktrees-list .worktree-row > button:not(.outline)",
				) || document.querySelector("#worktrees-list .worktree-row > button")
			)?.focus(),
		"#worktrees-panel > summary",
	],
	[
		"Go to chat",
		"c",
		() =>
			(
				document.querySelector(
					"#chats-list .session-row > button:not(.outline)",
				) ||
				document.querySelector("#chats-list .session-row > button[data-sessid]")
			)?.focus(),
		"#chats-panel > summary",
	],
	...["Add project", "Swap panel layout", "Switch theme"].map((name, i) => [
		name,
		String(i + 1),
		() =>
			document
				.querySelectorAll(
					"#panel-sessions .title-row > div:last-child > button",
				)
				[i]?.click(),
		["#add-project", "#layout-toggle", "#theme"][i],
	]),
	["Toggle hotkey hints", null, () => _toggleHotkeyHints()],
];
function _renderHotkeyHints() {
	hintsLegend.textContent = `Press ${leader}, then the shown key.`;
	for (const [, key, , selectors] of actions) {
		if (!selectors) continue;
		const label =
			key === "Ctrl+ "
				? "(ctrl+space)"
				: key.includes("Ctrl+")
					? `(${key.toLowerCase()})`
					: key;
		for (const selector of Array.isArray(selectors) ? selectors : [selectors])
			document.querySelector(selector)?.setAttribute("data-hotkey", label);
	}
}
_renderHotkeyHints();
new MutationObserver(_renderHotkeyHints).observe(
	document.querySelector("#chats-list"),
	{ childList: true },
);

for (const [name, key, run] of actions) {
	const button = document.createElement("button"),
		hint = document.createElement("small");
	button.type = "button";
	button.className = "action-row outline secondary";
	button.textContent = name;
	if (key) {
		hint.textContent =
			key === "Ctrl+ "
				? "(ctrl+space)"
				: key.includes("Ctrl+")
					? `(${key.toLowerCase()})`
					: key;
		button.append(hint);
	}
	button.addEventListener("click", () => {
		waiting = false;
		dialog.close();
		run();
	});
	list.append(button);
}

function _combo(e) {
	return `${e.ctrlKey ? "Ctrl+" : ""}${e.altKey ? "Alt+" : ""}${e.metaKey ? "Meta+" : ""}${e.shiftKey ? "Shift+" : ""}${e.key.length === 1 ? e.key.toLowerCase() : e.key}`;
}
document
	.querySelector("#actions-close")
	.addEventListener("click", () => dialog.close());
hintsToggle.addEventListener("click", () => _toggleHotkeyHints());
document.querySelector("#actions-toggle").addEventListener("click", () => {
	waiting = false;
	dialog.showModal();
});
dialog.addEventListener("close", () => {
	waiting = false;
});

document.addEventListener(
	"keydown",
	(e) => {
		if (e.key === "Escape") {
			_toggleHotkeyHints(false);
			return;
		}
		if (e.key === "Tab") return;
		const combo = _combo(e);
		if (document.activeElement === leaderInput) {
			if (["Control", "Shift", "Alt", "Meta"].includes(e.key)) return;
			e.preventDefault();
			if (!e.ctrlKey && !e.altKey && !e.metaKey) return;
			leader = combo;
			leaderInput.value = leader;
			try {
				localStorage.setItem(storageKey, leader);
			} catch {}
			leaderInput.blur();
			_renderHotkeyHints();
			return;
		}
		if (combo === leader) {
			e.preventDefault();
			waiting = true;
			if (!dialog.open) dialog.showModal();
			return;
		}
		const action = waiting
			? actions.find(([, key]) => key === e.key)
			: actions.find(([, key]) => key === combo && key.includes("Ctrl+"));
		if (waiting) {
			waiting = false;
			dialog.close();
		}
		if (!action) return;
		e.preventDefault();
		action[2]();
	},
	true,
);
