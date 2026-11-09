const themeBtn = document.querySelector("#theme");
function paintTheme() {
	const dark = document.documentElement.dataset.theme === "dark";
	const label = dark ? "Switch to light theme" : "Switch to dark theme";
	const icon = document.createElement("i");
	icon.dataset.lucide = dark ? "sun" : "moon";
	themeBtn.replaceChildren(icon);
	themeBtn.title = label;
	themeBtn.setAttribute("aria-label", label);
	if (window.lucide) window.lucide.createIcons();
}
themeBtn.addEventListener("click", () => {
	const next =
		document.documentElement.dataset.theme === "dark" ? "light" : "dark";
	document.documentElement.dataset.theme = next;
	try {
		localStorage.setItem("theme", next);
	} catch {}
	paintTheme();
});
paintTheme();
