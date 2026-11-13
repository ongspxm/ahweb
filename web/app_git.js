const eleGitFiles = document.querySelector("#git-files"),
	eleGitDiff = document.querySelector("#git-diff");
let gitListRequest = 0,
	gitDiffRequest = 0;

window.apiLoadGit = async (folder) => {
	const request = ++gitListRequest;
	gitDiffRequest++;
	eleGitFiles.replaceChildren();
	eleGitDiff.replaceChildren();
	const _message = (target, message) => {
		const item = document.createElement("div");
		item.className = "git-empty";
		item.textContent = message;
		target.append(item);
	};
	if (!folder) {
		_message(eleGitFiles, "Select a worktree to view Git changes");
		return;
	}
	_message(eleGitFiles, "Loading changes…");
	try {
		const params = new URLSearchParams({ folder });
		const response = await fetch(`/api/git?${params}`);
		const body = await response.json();
		if (!response.ok)
			throw new Error(body.error || "Could not load Git status");
		if (request !== gitListRequest) return;
		eleGitFiles.replaceChildren();
		const patchLink = document.createElement("a"),
			patchParams = new URLSearchParams({ folder, file: "." });
		patchLink.className = "git-patch-link";
		patchLink.href = `/api/git?${patchParams}`;
		patchLink.target = "_blank";
		patchLink.rel = "noopener";
		patchLink.textContent = "Open full patch in new tab";
		eleGitFiles.append(patchLink);
		if (!(body.files || []).length) {
			_message(eleGitFiles, "Working tree clean");
			return;
		}
		for (const file of body.files) {
			const button = document.createElement("button"),
				status = document.createElement("code"),
				path = document.createElement("span"),
				code = file.status || "";
			button.type = "button";
			button.className = "git-entry";
			button.setAttribute("aria-pressed", "false");
			button.title = file.path;
			status.className = "git-status";
			status.textContent = code;
			status.dataset.kind =
				code.includes("?") || code.includes("A")
					? "added"
					: code.includes("D")
						? "deleted"
						: code.includes("R") || code.includes("C")
							? "renamed"
							: "modified";
			path.textContent = file.path;
			button.append(status, path);
			button.addEventListener("click", () => {
				eleGitFiles
					.querySelectorAll(".git-entry[aria-pressed='true']")
					.forEach((selected) => {
						selected.setAttribute("aria-pressed", "false");
					});
				button.setAttribute("aria-pressed", "true");
				_loadDiff(file.path);
			});
			eleGitFiles.append(button);
		}
	} catch (err) {
		if (request === gitListRequest) {
			eleGitFiles.replaceChildren();
			_message(eleGitFiles, err.message);
		}
	}

	async function _loadDiff(file) {
		const selection = ++gitDiffRequest;
		eleGitDiff.replaceChildren();
		const title = document.createElement("strong");
		title.textContent = file;
		eleGitDiff.append(title);
		try {
			const params = new URLSearchParams({ folder, file });
			const response = await fetch(`/api/git?${params}`);
			const body = await response.json();
			if (!response.ok) throw new Error(body.error || "Could not load diff");
			if (selection !== gitDiffRequest || request !== gitListRequest) return;
			if (!body.diff) {
				_message(eleGitDiff, "No diff available for this file");
				return;
			}
			const pre = document.createElement("pre");
			pre.className = "git-diff-text";
			for (const line of body.diff.split("\n")) {
				const row = document.createElement("span");
				row.className = "git-diff-line";
				row.textContent = line;
				if (line.startsWith("+") && !line.startsWith("+++"))
					row.classList.add("added");
				else if (line.startsWith("-") && !line.startsWith("---"))
					row.classList.add("deleted");
				pre.append(row);
			}
			eleGitDiff.append(pre);
		} catch (err) {
			if (selection === gitDiffRequest) _message(eleGitDiff, err.message);
		}
	}
};
