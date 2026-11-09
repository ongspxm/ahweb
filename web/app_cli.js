import { FitAddon, PredictiveEchoAddon, Terminal } from "/lib/xterm/xterm.js";

const terminalHost = document.querySelector("#cli-terminal");
const cliPane = document.querySelector("#tab-cli");
const cliStatus = document.querySelector("#cli-status");

// hide dot when cli tab not active or when socket state changes
const _syncCliStatus = () => {
	cliStatus.hidden = active?.socket?.readyState !== WebSocket.OPEN;
};
new MutationObserver(_syncCliStatus).observe(cliPane, {
	attributes: true,
	attributeFilter: ["hidden"],
});

const folderInput = document.querySelector("#directory");
const encoder = new TextEncoder();
const states = new Map();
let active = null;
const keyBar = document.querySelector("#cli-keys");
const modifiers = new Set();
const _withModifiers = (data, key) => {
	const activeModifiers = new Set(modifiers);
	modifiers.clear();
	for (const button of keyBar.querySelectorAll("[data-modifier]"))
		button.setAttribute("aria-pressed", "false");
	const modifier =
		1 +
		Number(activeModifiers.has("shift")) +
		2 * Number(activeModifiers.has("alt")) +
		4 * Number(activeModifiers.has("ctrl"));
	const arrows = { left: "D", down: "B", up: "A", right: "C" };
	if (arrows[key])
		return modifier === 1
			? `\x1b[${arrows[key]}`
			: `\x1b[1;${modifier}${arrows[key]}`;
	if (key === "pageup" || key === "pagedown") {
		const code = key === "pageup" ? 5 : 6;
		return modifier === 1 ? `\x1b[${code}~` : `\x1b[${code};${modifier}~`;
	}
	if (activeModifiers.has("ctrl") && data.length === 1) {
		const c = data.toUpperCase();
		data =
			c >= "@" && c <= "_"
				? String.fromCharCode(c.charCodeAt(0) & 31)
				: c === "?"
					? "\x7f"
					: data;
	} else if (activeModifiers.has("shift") && data.length === 1)
		data = data.toUpperCase();
	return `${activeModifiers.has("alt") ? "\x1b" : ""}${data}`;
};

keyBar.addEventListener("click", (e) => {
	const button = e.target.closest("button");
	if (!button) return;
	if (button.dataset.fontSize) {
		if (active) {
			const terminal = active.terminal;
			terminal.options.fontSize = Math.max(
				4,
				Math.min(
					32,
					terminal.options.fontSize + Number(button.dataset.fontSize),
				),
			);
			_fitTerminal(active);
			active.predictor.refreshFont();
		}
		return;
	}
	if (button.dataset.modifier) {
		const modifier = button.dataset.modifier;
		if (modifiers.has(modifier)) modifiers.delete(modifier);
		else modifiers.add(modifier);
		button.setAttribute("aria-pressed", String(modifiers.has(modifier)));
		active?.terminal.textarea?.focus();
		return;
	}
	const key = button.dataset.key;
	const data = { escape: "\x1b", tab: "\t", enter: "\r" }[key] || "";
	const sequence = _withModifiers(data, key);
	active?.send(sequence);
	active?.terminal.textarea?.focus();
});
const empty = document.createElement("div");
empty.textContent = "Select a worktree to open its CLI.";
terminalHost.append(empty);

const _stateFor = (folder) => {
	let state = states.get(folder);
	if (state) return state;
	const host = document.createElement("div");
	host.style.cssText = "width:100%;height:100%";
	host.hidden = true;
	terminalHost.append(host);
	const terminal = new Terminal({
		allowProposedApi: true,
		cursorBlink: true,
		scrollback: 5000,
		fontSize: 13,
	});
	const fit = new FitAddon();
	terminal.loadAddon(fit);
	terminal.open(host);
	const predictor = new PredictiveEchoAddon();
	terminal.loadAddon(predictor);
	terminal.textarea?.setAttribute("autocomplete", "off");
	terminal.textarea?.setAttribute("autocapitalize", "off");
	terminal.textarea?.setAttribute("autocorrect", "off");
	if (terminal.textarea) terminal.textarea.spellcheck = false;
	const _send = (data) => {
		if (state.socket?.readyState !== WebSocket.OPEN) return;
		if (data === "\x7f") predictor.predictBackspace();
		else if (Array.from(data).length === 1 && data.codePointAt(0) >= 0x20)
			predictor.predictChar(data);
		else predictor.clearPredictions();
		state.socket.send(encoder.encode(data));
	};
	state = {
		host,
		terminal,
		fit,
		predictor,
		send: _send,
		socket: null,
		reconnectTimer: null,
		reconnectDelay: 1000,
		offset: 0,
	};
	terminal.onData((data) => {
		if (modifiers.size && data.length === 1) data = _withModifiers(data);
		_send(data);
	});
	states.set(folder, state);
	return state;
};

const _fitTerminal = (state) => {
	try {
		state.fit.fit();
	} catch {
		return;
	}
	if (state.socket?.readyState === WebSocket.OPEN)
		state.socket.send(
			JSON.stringify({
				type: "resize",
				cols: state.terminal.cols,
				rows: state.terminal.rows,
			}),
		);
};

const _connect = (folder, state) => {
	if (state.socket) return;
	clearTimeout(state.reconnectTimer);
	state.reconnectTimer = null;
	const url = new URL("/api/cli", location.href);
	url.protocol = url.protocol.replace("http", "ws");
	url.searchParams.set("folder", folder);
	url.searchParams.set("offset", String(state.offset));
	state.socket = new WebSocket(url);
	const socket = state.socket;
	socket.binaryType = "arraybuffer";
	socket.onopen = () => {
		_syncCliStatus();
		state.reconnectDelay = 1000;
		if (active === state && terminalHost.getClientRects().length)
			_fitTerminal(state);
	};
	socket.onmessage = ({ data }) => {
		if (typeof data === "string") {
			try {
				const message = JSON.parse(data);
				if (message.type === "offset") state.offset = message.offset;
			} catch {}
			return;
		}
		const bytes = new Uint8Array(data);
		state.offset += bytes.length;
		state.terminal.write(bytes);
	};
	socket.onclose = () => {
		if (state.socket !== socket) return;
		state.socket = null;
		_syncCliStatus();
		const delay = state.reconnectDelay;
		state.reconnectDelay = Math.min(delay * 2, 30000);
		state.reconnectTimer = setTimeout(() => {
			state.reconnectTimer = null;
			_connect(folder, state);
		}, delay);
	};
};

const _sync = () => {
	const folder = folderInput.value.trim();
	if (!folder) {
		if (active) active.host.hidden = true;
		active = null;
		empty.hidden = false;
		return;
	}
	const state = _stateFor(folder);
	if (active !== state) {
		if (active) active.host.hidden = true;
		active = state;
	}
	const visible = !!terminalHost.getClientRects().length;
	state.host.hidden = !visible;
	empty.hidden = true;
	if (visible) _connect(folder, state);
	if (visible) requestAnimationFrame(() => _fitTerminal(state));
};

new ResizeObserver(_sync).observe(terminalHost);
folderInput.addEventListener("change", _sync);
_sync();
window.addEventListener("redraw-cli", () => {
	const folder = folderInput.value.trim();
	const state = states.get(folder);
	if (!state) return;
	state.predictor.clearPredictions();
	state.terminal.reset();
	state.offset = 0;
	clearTimeout(state.reconnectTimer);
	state.reconnectTimer = null;
	if (state.socket) {
		const socket = state.socket;
		state.socket = null;
		socket.onopen = socket.onmessage = socket.onerror = socket.onclose = null;
		socket.close();
	}
	_syncCliStatus();
	_connect(folder, state);
});
