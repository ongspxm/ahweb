const statusView = document.querySelector("#server-status");

function renderStatus(status) {
	statusView.replaceChildren();
	const version = document.createElement("p");
	version.textContent = `Version: ${status.version || "unknown"}`;
	const update = document.createElement("p");
	update.textContent =
		status.update ||
		`To update, run ${status.update_command || "go install github.com/ongspxm/ahweb@latest"}.`;
	statusView.append(version, update);

	const pollStatus = document.createElement("p");
	pollStatus.textContent = status.poll_status || "poll.session 0";

	const harnesses = document.createElement("section"),
		harnessTitle = document.createElement("h3"),
		harnessRows = document.createElement("div");
	harnessTitle.textContent = "Harnesses";
	harnessRows.className = "status-harnesses";
	for (const harness of status.harnesses || []) {
		const row = document.createElement("div"),
			icon = document.createElement("i"),
			name = document.createElement("strong"),
			state = document.createElement("span");
		row.className = "status-harness-row";
		icon.dataset.lucide = harness.available ? "check" : "x";
		icon.setAttribute("role", "img");
		icon.setAttribute(
			"aria-label",
			harness.available ? "Available" : "Unavailable",
		);
		icon.className = harness.available ? "status-ok" : "status-error";
		name.textContent = harness.name;
		state.textContent = harness.available
			? "available"
			: `unavailable${harness.reason ? ` — ${harness.reason}` : ""}`;
		row.append(icon, name, state);
		harnessRows.append(row);
	}
	if (!(status.harnesses || []).length) {
		const empty = document.createElement("p");
		empty.className = "status-empty";
		empty.textContent = "No harnesses configured";
		harnessRows.append(empty);
	}
	harnesses.append(harnessTitle, harnessRows);
	statusView.append(harnesses);
	statusView.append(pollStatus);

	const projects = document.createElement("section"),
		projectsTitle = document.createElement("h3"),
		projectList = status.projects || [],
		worktreeCount = projectList.reduce(
			(total, project) => total + (project.worktrees || []).length,
			0,
		);
	projectsTitle.textContent = `Projects (${projectList.length} roots, ${worktreeCount} worktrees)`;
	projects.append(projectsTitle);
	if (projectList.length) {
		const table = document.createElement("table"),
			head = table.createTHead().insertRow();
		table.className = "status-projects";
		for (const label of ["Root", "Worktrees", "Sessions"]) {
			const cell = document.createElement("th");
			cell.scope = "col";
			cell.textContent = label;
			head.append(cell);
		}
		const body = table.createTBody();
		for (const project of projectList) {
			const worktrees = project.worktrees || [],
				sessions = worktrees.reduce(
					(total, worktree) => total + worktree.sessions,
					0,
				),
				row = body.insertRow(),
				cell = row.insertCell();
			cell.colSpan = 3;
			cell.className = "status-project-root";
			const details = document.createElement("details"),
				summary = document.createElement("summary");
			summary.className = "status-project-summary";
			for (const value of [project.root, worktrees.length, sessions]) {
				const item = document.createElement("span");
				item.textContent = value;
				summary.append(item);
			}
			details.append(summary);
			if (project.error) {
				const error = document.createElement("p");
				error.className = "status-project-error";
				error.textContent = project.error;
				details.append(error);
			}
			if (worktrees.length) {
				const nested = document.createElement("table"),
					nestedHead = nested.createTHead().insertRow(),
					nestedBody = nested.createTBody();
				nested.className = "status-worktrees";
				for (const label of ["Path", "Branch", "Sessions"]) {
					const heading = document.createElement("th");
					heading.scope = "col";
					heading.textContent = label;
					nestedHead.append(heading);
				}
				for (const worktree of worktrees) {
					const worktreeRow = nestedBody.insertRow(),
						path = worktreeRow.insertCell(),
						branch = worktreeRow.insertCell(),
						count = worktreeRow.insertCell();
					path.textContent = worktree.path;
					branch.textContent = worktree.branch || "";
					count.textContent = worktree.sessions;
				}
				details.append(nested);
			}
			cell.append(details);
		}
		projects.append(table);
	} else {
		const empty = document.createElement("p");
		empty.className = "status-empty";
		empty.textContent = "No projects configured";
		projects.append(empty);
	}
	statusView.append(projects);
	if (window.lucide) window.lucide.createIcons();
}

window.addEventListener("server-status", (event) => renderStatus(event.detail));
window.addEventListener("server-status-error", (event) => {
	statusView.textContent = event.detail || "Could not load server status";
});
async function apiLoadStatus() {
	try {
		const response = await fetch("/api/status");
		if (!response.ok) throw new Error("Could not load server status");
		const body = await response.json();
		window.dispatchEvent(new CustomEvent("server-status", { detail: body }));
		return body;
	} catch (err) {
		window.dispatchEvent(
			new CustomEvent("server-status-error", { detail: err.message }),
		);
	}
}
window.apiLoadStatus = apiLoadStatus;

document.querySelector("#status-refresh").addEventListener("click", () => {
	window.apiLoadStatus()?.then?.(() => window.apiLoadModels?.());
});

apiLoadStatus();
