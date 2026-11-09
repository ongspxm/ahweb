const driver = window.driver?.js?.driver,
	dialog = document.querySelector("#actions-dialog"),
	list = document.querySelector("#actions-list"),
	isMobile = matchMedia("(max-width: 600px)").matches;

const _drive = (steps, onDone) =>
	driver({
		overlayColor: getComputedStyle(document.documentElement)
			.getPropertyValue("--pico-del-color")
			.trim(),
		showProgress: true,
		steps: isMobile
			? steps.filter(
					({ element }) =>
						element !== "#layout-toggle" && !element.includes("resizer"),
				)
			: steps,
		onHighlightStarted: (_element, step) => {
			if (!isMobile) return;
			const panel = [
				"#cli-btns",
				"#cli-keys",
				"#workspace-tabs",
				"#workspace-refresh",
			].includes(step.element)
				? "tab-cli"
				: [
							"#prompt-controls",
							"#prompt",
							"#panel-chats",
							"#stop",
							"#collapse-all",
							"#prompt-actions",
						].includes(step.element)
					? "chats"
					: "sessions";
			document
				.querySelector(
					`[data-mobile-panel="${panel}"], [data-workspace-tab="${panel}"]`,
				)
				?.click();
			const section =
				step.element === "#chats-list > button"
					? "#chats-panel"
					: ["#projects-panel", "#worktrees-panel", "#chats-panel"].includes(
								step.element,
							)
						? step.element
						: null;
			if (section) document.querySelector(section).open = true;
		},
		onDestroyed: (_element, _step, { reason }) => {
			if (reason === "done") onDone?.();
		},
	}).drive();
const _readCompletedTours = () => {
	try {
		const completed = JSON.parse(
			localStorage.getItem("ahweb-onboarding-items") || "[]",
		);
		return Array.isArray(completed) ? completed : [];
	} catch {
		return [];
	}
};
const _markTourDone = (id) => {
	const completed = _readCompletedTours();
	if (!completed.includes(id)) completed.push(id);
	try {
		localStorage.setItem("ahweb-onboarding-items", JSON.stringify(completed));
	} catch {}
	checklist?.complete(id);
};
const tours = [
	[
		"Basic tour",
		() =>
			_drive(
				[
					{
						element: "#add-project",
						popover: {
							title: "Add a project",
							description:
								"Add a project folder here if you haven’t added one yet.",
						},
					},
					{
						element: "#projects-panel",
						popover: {
							title: "Select a project",
							description: "Choose the project you want to work in.",
						},
					},
					{
						element: "#worktrees-panel",
						popover: {
							title: "Select a worktree",
							description: "Choose the worktree for this chat.",
						},
					},
					{
						element: "#chats-panel",
						popover: {
							title: "Select a chat",
							description:
								"Choose a chat. A blinking dot means it’s working; a solid dot means it has unread messages.",
						},
					},
					{
						element: "#chats-list > button",
						popover: {
							title: "Create a new session",
							description:
								"Click + New chat to start a session in this worktree.",
						},
					},
					{
						element: "#prompt-controls",
						popover: {
							title: "Select your harness and model",
							description:
								"Choose which harness and model to use for this chat.",
						},
					},
					{
						element: "#prompt",
						popover: {
							title: "Write your first message",
							description: "Enter a prompt here, then press Send.",
						},
					},
					{
						element: "#panel-chats",
						popover: {
							title: "See the conversation",
							description:
								"Your conversation appears here. New messages show below.",
						},
					},
					{
						element: "#stop",
						popover: {
							title: "Stop your chat",
							description: "Stop the harness here.",
						},
					},
					{
						element: "#collapse-all",
						popover: {
							title: "Expand for more tool details",
							description:
								"Expand or collapse tool details in the conversation.",
						},
					},
					{
						element: "#actions-toggle",
						popover: {
							title: "Find the tours again",
							description: "Tours are also available from the Actions panel.",
						},
					},
				],
				() => _markTourDone("basic-tour"),
			),
	],
	[
		"Other features",
		() => {
			const _start = () =>
				_drive(
					[
						{
							element: "#cli-keys",
							popover: {
								title: "CLI font size",
								description:
									"Re-render the CLI tab with bigger (F+) and smaller (F-) fonts.",
							},
						},
						{
							element: "#workspace-tabs",
							popover: {
								title: "Workspace tabs",
								description: "Switch between Files, Git, and CLI here.",
							},
						},
						{
							element: "#workspace-refresh",
							popover: {
								title: "Refresh workspace",
								description: "Refresh the Files, Git, and CLI views.",
							},
						},
						{
							element: "#prompt-actions",
							popover: {
								title: "Chat controls",
								description:
									"Attach images, send or stop a chat, refresh, and control message details here.",
							},
						},
						{
							element: "#hotkey-hints-toggle",
							popover: {
								title: "Hotkey hints",
								description:
									"Show shortcuts beside the controls that support them.",
							},
						},
						{
							element: "#actions-toggle",
							popover: {
								title: "Actions and shortcuts",
								description:
									"Browse shortcuts and change the leader key in Actions.",
							},
						},
						{
							element: "#theme",
							popover: {
								title: "Change theme",
								description: "Switch between the available themes.",
							},
						},
						{
							element: "#layout-toggle",
							popover: {
								title: "Swap panel layout",
								description:
									"Change which side the chat and workspace panels appear on.",
							},
						},
						{
							element: "#brand-icon",
							popover: {
								title: "Identify this instance",
								description:
									"This icon is based on the hostname, so you can tell which instance you’re using.",
							},
						},
						{
							element: "#panel-sessions + .panel-resizer",
							popover: {
								title: "Resize sessions and workspace panels",
								description: "Drag this divider to change the panel widths.",
							},
						},
						{
							element: "#panel-workspace + .panel-resizer",
							popover: {
								title: "Resize workspace and chat panels",
								description: "Drag this divider to change the panel widths.",
							},
						},
					],
					() => _markTourDone("other-features"),
				);
			const _waitForCLI = () => {
				if (location.hash !== "#tab-cli") return;
				window.removeEventListener("hashchange", _waitForCLI);
				_start();
			};
			if (location.hash === "#tab-cli") _start();
			else {
				window.addEventListener("hashchange", _waitForCLI);
				document.querySelector('[data-workspace-tab="tab-cli"]').click();
			}
		},
	],
];

const checklist = window.driverChecklist?.checklist({
	id: "ahweb-onboarding",
	title: "Get started",
	description: "Take these tours to learn AhWeb.",
	completeTitle: "Onboarding complete",
	completeDescription: "You’re all set.",
	launcher: false,
	completed: _readCompletedTours(),
	items: [
		{
			id: "basic-tour",
			title: tours[0][0],
			action: () => tours[0][1](),
			completeWhen: () => _readCompletedTours().includes("basic-tour"),
		},
		{
			id: "other-features",
			title: tours[1][0],
			action: () => tours[1][1](),
			completeWhen: () => _readCompletedTours().includes("other-features"),
		},
	],
});

if (!driver || !checklist)
	console.error("Driver.js onboarding resources failed to load.");
else {
	const button = document.createElement("button"),
		hint = document.createElement("small");
	button.type = "button";
	button.className = "action-row outline secondary";
	button.textContent = "Onboarding checklist";
	hint.textContent = "Tours";
	button.append(hint);
	button.addEventListener("click", () => {
		dialog.close();
		checklist.show();
		checklist.open();
	});
	list.append(button);
}

if (
	driver &&
	checklist &&
	checklist.getProgress().done < checklist.getProgress().total
) {
	checklist.show();
	checklist.open();
}
