const eleFileList = document.querySelector("#file-list"),
	eleUploadForm = document.querySelector("#file-upload"),
	eleUploadFile = document.querySelector("#file-upload-file"),
	eleUploadPath = document.querySelector("#file-upload-path"),
	eleUploadStatus = document.querySelector("#file-upload-status"),
	eleFileView = document.querySelector("#file-view");
let fileListRequest = 0,
	fileViewRequest = 0,
	activeFileRoot = "";

window.apiLoadFiles = async (root) => {
	activeFileRoot = root;
	const request = ++fileListRequest;
	fileViewRequest++;
	eleFileList.replaceChildren();
	const _empty = (icon, message) => {
		const state = document.createElement("div"),
			glyph = document.createElement("i"),
			label = document.createElement("span");
		state.className = "file-empty";
		glyph.dataset.lucide = icon;
		glyph.setAttribute("aria-hidden", "true");
		label.textContent = message;
		state.append(glyph, label);
		eleFileView.replaceChildren(state);
		if (window.lucide) window.lucide.createIcons();
	};
	_empty(
		root ? "file-search" : "folder-open",
		root ? "Choose a file to preview" : "Select a worktree to browse files",
	);
	if (!root) return;

	async function _loadDirectory(path, target) {
		try {
			const params = new URLSearchParams({ folder: root, path });
			const r = await fetch(`/api/file?${params}`),
				body = await r.json();
			if (!r.ok) throw new Error(body.error || "Could not list folder");
			if (request !== fileListRequest) return;
			target.replaceChildren();
			for (const dir of body.dirs || []) {
				const details = document.createElement("details"),
					summary = document.createElement("summary"),
					icon = document.createElement("i"),
					label = document.createElement("span"),
					children = document.createElement("div");
				icon.dataset.lucide = "folder";
				icon.setAttribute("aria-hidden", "true");
				label.textContent = `${dir.split("/").at(-1)}/`;
				summary.title = dir;
				summary.append(icon, label);
				children.className = "file-children";
				details.append(summary, children);
				details.addEventListener("toggle", async () => {
					if (!details.open || details.dataset.loaded) return;
					details.dataset.loaded = "true";
					if (!(await _loadDirectory(dir, children)))
						delete details.dataset.loaded;
				});
				target.append(details);
			}
			for (const file of body.files || []) {
				const button = document.createElement("button"),
					icon = document.createElement("i"),
					label = document.createElement("span");
				button.className = "file-entry";
				button.type = "button";
				button.setAttribute("aria-pressed", "false");
				icon.dataset.lucide = "file";
				icon.setAttribute("aria-hidden", "true");
				label.textContent = file.split("/").at(-1);
				button.title = file;
				button.append(icon, label);
				button.addEventListener("click", () => {
					eleFileList
						.querySelectorAll(".file-entry[aria-pressed='true']")
						.forEach((selected) => {
							selected.setAttribute("aria-pressed", "false");
						});
					button.setAttribute("aria-pressed", "true");
					_loadFile(file);
				});
				target.append(button);
			}
			if (!(body.dirs || []).length && !(body.files || []).length) {
				const empty = document.createElement("div");
				empty.className = "file-list-empty";
				empty.textContent = "Empty folder";
				target.append(empty);
			}
			if (window.lucide) window.lucide.createIcons();
			return true;
		} catch (err) {
			if (request === fileListRequest) target.textContent = err.message;
		}
	}

	async function _loadFile(path) {
		const selection = ++fileViewRequest;
		eleFileView.textContent = "Loading…";
		try {
			const params = new URLSearchParams({ folder: root, path });
			const url = `/api/file?${params}`,
				r = await fetch(url);
			if (!r.ok) {
				const body = await r.json();
				throw new Error(body.error || "Could not load file");
			}
			const type = (r.headers.get("content-type") || "")
				.split(";")[0]
				.toLowerCase();
			if (selection !== fileViewRequest) return;
			const header = document.createElement("div"),
				link = document.createElement("a"),
				icon = document.createElement("i"),
				title = document.createElement("strong");
			header.className = "file-title";
			link.href = url;
			link.target = "_blank";
			link.rel = "noopener noreferrer";
			link.className = "file-raw-link";
			icon.dataset.lucide = "file";
			icon.setAttribute("aria-hidden", "true");
			link.append(icon);
			title.textContent = path.split("/").at(-1);
			header.append(link, title);
			eleFileView.replaceChildren(header);
			if (window.lucide) window.lucide.createIcons();
			if (
				type.startsWith("text/") ||
				/^application\/(json|javascript|xml|yaml|x-yaml|toml)$/.test(type)
			) {
				const pre = document.createElement("pre");
				pre.className = "file-text";
				const text = await r.text();
				if (selection !== fileViewRequest) return;
				text.split("\n").forEach((line) => {
					const row = document.createElement("span");
					row.className = "file-line";
					row.textContent = line.replace(/\r$/, "");
					pre.append(row);
				});
				eleFileView.append(pre);
				return;
			}
			if (
				!type.startsWith("image/") &&
				!type.startsWith("video/") &&
				!type.startsWith("audio/") &&
				type !== "application/pdf"
			) {
				await r.body?.cancel();
				if (selection !== fileViewRequest) return;
				eleFileView.append(document.createTextNode("Binary can't display"));
				return;
			}
			await r.body?.cancel();
			if (selection !== fileViewRequest) return;
			const media = document.createElement(
				type.startsWith("image/")
					? "img"
					: type.startsWith("video/")
						? "video"
						: type.startsWith("audio/")
							? "audio"
							: "iframe",
			);
			if (
				media instanceof HTMLVideoElement ||
				media instanceof HTMLAudioElement
			)
				media.controls = true;
			if (media instanceof HTMLImageElement) media.alt = path.split("/").at(-1);
			if (media instanceof HTMLIFrameElement) media.title = title.textContent;
			media.src = url;
			eleFileView.append(media);
		} catch (err) {
			if (selection === fileViewRequest) eleFileView.textContent = err.message;
		}
	}

	await _loadDirectory(".", eleFileList);
};

eleUploadFile.addEventListener("change", () => {
	if (eleUploadFile.files[0]) eleUploadPath.value = eleUploadFile.files[0].name;
});

eleUploadForm.addEventListener("submit", async (event) => {
	event.preventDefault();
	const root = activeFileRoot,
		file = eleUploadFile.files[0],
		path = eleUploadPath.value.trim();
	if (!root) {
		eleUploadStatus.textContent = "Select a worktree first";
		return;
	}
	if (!file || !path) {
		eleUploadStatus.textContent = "Choose a file and enter its destination path";
		return;
	}

    // double upload to chk whether to replace
	const params = new URLSearchParams({ folder: root, path });
	let overwrite = false;
	try {
		const existing = await fetch(`/api/file?${params}`);
		if (existing.ok) {
			const isDirectory = !existing.headers.has("content-disposition");
			await existing.body?.cancel();
			if (isDirectory)
				throw new Error("Destination is a folder");
			if (!window.confirm(`Replace ${path}?`)) return;
			overwrite = true;
		} else if (existing.status === 404) {
			await existing.body?.cancel();
		} else {
			const body = await existing.json();
			throw new Error(body.error || "Could not check destination");
		}

		eleUploadStatus.textContent = "Uploading…";
		const body = new FormData();
		body.append("file", file);
		body.append("path", path);
		if (overwrite) body.append("overwrite", "true");
		const response = await fetch(
			`/api/file?folder=${encodeURIComponent(root)}`,
			{ method: "POST", body },
		);
		const result = await response.json();
		if (!response.ok) throw new Error(result.error || "Upload failed");
		eleUploadStatus.textContent = `Uploaded ${result.path}`;
		eleUploadFile.value = "";
		if (activeFileRoot === root) await window.apiLoadFiles(root);
	} catch (err) {
		eleUploadStatus.textContent = err.message;
	}
});
