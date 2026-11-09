(this.driver = this.driver || {}),
	(this.driver.js = ((e) => {
		Object.defineProperty(e, Symbol.toStringTag, { value: `Module` });
		const t = [`pointerdown`, `mousedown`, `pointerup`, `mouseup`, `click`],
			n = new WeakMap();
		function r(e, r, a) {
			i(e);
			const o = (t) => {
				const n = t.target;
				e.contains(n) &&
					((!a || a(n)) &&
						(t.preventDefault(),
						t.stopPropagation(),
						t.stopImmediatePropagation()),
					t.type === `click` && r?.(t));
			};
			for (const e of t) document.addEventListener(e, o, !0);
			n.set(e, o);
		}
		function i(e) {
			const r = n.get(e);
			if (r) {
				for (const e of t) document.removeEventListener(e, r, !0);
				n.delete(e);
			}
		}
		function a(e, t) {
			const n = e.wrapper.getBoundingClientRect();
			return {
				width: n.width + t,
				height: n.height + t,
				realWidth: n.width,
				realHeight: n.height,
			};
		}
		function o(e, t) {
			const {
				elementDimensions: n,
				popoverDimensions: r,
				popoverPadding: i,
				popoverArrowDimensions: a,
			} = t;
			return e === `start`
				? Math.max(
						Math.min(n.top - i, window.innerHeight - r.realHeight - a.width),
						a.width,
					)
				: e === `end`
					? Math.max(
							Math.min(
								n.top - r?.realHeight + n.height + i,
								window.innerHeight - r?.realHeight - a.width,
							),
							a.width,
						)
					: e === `center`
						? Math.max(
								Math.min(
									n.top + n.height / 2 - r?.realHeight / 2,
									window.innerHeight - r?.realHeight - a.width,
								),
								a.width,
							)
						: 0;
		}
		function s(e, t) {
			const {
				elementDimensions: n,
				popoverDimensions: r,
				popoverPadding: i,
				popoverArrowDimensions: a,
			} = t;
			return e === `start`
				? Math.max(
						Math.min(n.left - i, window.innerWidth - r.realWidth - a.width),
						a.width,
					)
				: e === `end`
					? Math.max(
							Math.min(
								n.left - r?.realWidth + n.width + i,
								window.innerWidth - r?.realWidth - a.width,
							),
							a.width,
						)
					: e === `center`
						? Math.max(
								Math.min(
									n.left + n.width / 2 - r?.realWidth / 2,
									window.innerWidth - r?.realWidth - a.width,
								),
								a.width,
							)
						: 0;
		}
		function c(e, t, n) {
			let { align: r, side: i } = n,
				c = n.centered ? `over` : i,
				l = n.padding,
				u = a(e, n.offset),
				d = e.arrow.getBoundingClientRect(),
				p = t.getBoundingClientRect(),
				m = p.top - u.height,
				h = m >= 0,
				g = window.innerHeight - (p.bottom + u.height),
				_ = g >= 0,
				v = p.left - u.width,
				y = v >= 0,
				b = window.innerWidth - (p.right + u.width),
				x = b >= 0,
				S = !h && !_ && !y && !x,
				C = c;
			if (
				(c === `top` && h
					? (x = y = _ = !1)
					: c === `bottom` && _
						? (x = y = h = !1)
						: c === `left` && y
							? (x = h = _ = !1)
							: c === `right` && x && (y = h = _ = !1),
				c === `over`)
			) {
				const t = window.innerWidth / 2 - u.realWidth / 2,
					n = window.innerHeight / 2 - u.realHeight / 2;
				(e.wrapper.style.left = `${t}px`),
					(e.wrapper.style.right = `auto`),
					(e.wrapper.style.top = `${n}px`),
					(e.wrapper.style.bottom = `auto`);
			} else if (S) {
				const t = window.innerWidth / 2 - u?.realWidth / 2;
				(e.wrapper.style.left = `${t}px`),
					(e.wrapper.style.right = `auto`),
					(e.wrapper.style.bottom = `10px`),
					(e.wrapper.style.top = `auto`);
			} else if (y) {
				const t = Math.min(v, window.innerWidth - u?.realWidth - d.width),
					n = o(r, {
						elementDimensions: p,
						popoverDimensions: u,
						popoverPadding: l,
						popoverArrowDimensions: d,
					});
				(e.wrapper.style.left = `${t}px`),
					(e.wrapper.style.top = `${n}px`),
					(e.wrapper.style.bottom = `auto`),
					(e.wrapper.style.right = `auto`),
					(C = `left`);
			} else if (x) {
				const t = Math.min(b, window.innerWidth - u?.realWidth - d.width),
					n = o(r, {
						elementDimensions: p,
						popoverDimensions: u,
						popoverPadding: l,
						popoverArrowDimensions: d,
					});
				(e.wrapper.style.right = `${t}px`),
					(e.wrapper.style.top = `${n}px`),
					(e.wrapper.style.bottom = `auto`),
					(e.wrapper.style.left = `auto`),
					(C = `right`);
			} else if (h) {
				const t = Math.min(m, window.innerHeight - u.realHeight - d.width),
					n = s(r, {
						elementDimensions: p,
						popoverDimensions: u,
						popoverPadding: l,
						popoverArrowDimensions: d,
					});
				(e.wrapper.style.top = `${t}px`),
					(e.wrapper.style.left = `${n}px`),
					(e.wrapper.style.bottom = `auto`),
					(e.wrapper.style.right = `auto`),
					(C = `top`);
			} else if (_) {
				const t = Math.min(g, window.innerHeight - u?.realHeight - d.width),
					n = s(r, {
						elementDimensions: p,
						popoverDimensions: u,
						popoverPadding: l,
						popoverArrowDimensions: d,
					});
				(e.wrapper.style.left = `${n}px`),
					(e.wrapper.style.bottom = `${t}px`),
					(e.wrapper.style.top = `auto`),
					(e.wrapper.style.right = `auto`),
					(C = `bottom`);
			}
			f(e, S ? `over` : C, r, t),
				[...e.wrapper.classList]
					.filter(
						(e) =>
							e.startsWith(`driver-popover-side-`) ||
							e.startsWith(`driver-popover-align-`),
					)
					.forEach((t) => e.wrapper.classList.remove(t)),
				e.wrapper.classList.add(`driver-popover-side-${C}`),
				e.wrapper.classList.add(`driver-popover-align-${r}`);
		}
		function l(e, t, n, r, i, a = 10) {
			const o = r - n;
			return e <= n && t >= r
				? i === `start`
					? 15 + a / 2
					: i === `end`
						? o - 15 - a / 2
						: o / 2
				: (Math.min(Math.max(e, n), r) + Math.min(Math.max(t, n), r)) / 2 - n;
		}
		function u(e, t, n = 10) {
			const r = t - 15 - n;
			if (r < 15) return Math.max(0, (t - n) / 2);
			const i = e - n / 2;
			return Math.min(Math.max(i, 15), r);
		}
		function d(e, t, n) {
			return e === `left` || e === `right`
				? t.bottom > n.top && t.top < n.bottom
					? e
					: t.bottom <= n.top
						? `bottom`
						: `top`
				: t.right > n.left && t.left < n.right
					? e
					: t.right <= n.left
						? `right`
						: `left`;
		}
		function f(e, t, n, r) {
			const i = e.arrow;
			if (
				((i.className = `driver-popover-arrow`),
				(i.style.top = ``),
				(i.style.right = ``),
				(i.style.bottom = ``),
				(i.style.left = ``),
				t === `over`)
			) {
				i.classList.add(`driver-popover-arrow-none`);
				return;
			}
			const a = r.getBoundingClientRect(),
				o = e.wrapper.getBoundingClientRect(),
				s = d(t, a, o);
			i.classList.add(`driver-popover-arrow-side-${s}`);
			const c = i.getBoundingClientRect().width || 10;
			if (s === `left` || s === `right`) {
				const e = l(a.top, a.bottom, o.top, o.bottom, n, c);
				i.style.top = `${u(e, o.height, c)}px`;
			} else {
				const e = l(a.left, a.right, o.left, o.right, n, c);
				i.style.left = `${u(e, o.width, c)}px`;
			}
		}
		function p(e) {
			return typeof e == `function`
				? e()
				: typeof e == `string`
					? document.querySelector(e)
					: e;
		}
		function m(e) {
			const t = window.getComputedStyle(e);
			return [t.overflow, t.overflowX, t.overflowY].some(
				(e) => e === `auto` || e === `scroll`,
			);
		}
		function h(e, t, n, r) {
			return (e /= r / 2) < 1
				? (n / 2) * e * e + t
				: (-n / 2) * (--e * (e - 2) - 1) + t;
		}
		function g(e) {
			const t = `a[href]:not([disabled]), button:not([disabled]), textarea:not([disabled]), input[type="text"]:not([disabled]), input[type="radio"]:not([disabled]), input[type="checkbox"]:not([disabled]), select:not([disabled])`;
			return e
				.flatMap((e) => {
					const n = e.matches(t),
						r = Array.from(e.querySelectorAll(t));
					return [...(n ? [e] : []), ...r];
				})
				.filter((e) => getComputedStyle(e).pointerEvents !== `none` && x(e));
		}
		function _(e, t) {
			if (!e || y(e)) return;
			const n = e.offsetHeight > window.innerHeight;
			e.scrollIntoView({
				behavior: !t || v(e) ? `auto` : `smooth`,
				inline: `center`,
				block: n ? `start` : `center`,
			});
		}
		function v(e) {
			if (!e || !e.parentElement) return;
			const t = e.parentElement;
			return t.scrollHeight > t.clientHeight;
		}
		function y(e) {
			const t = e.getBoundingClientRect();
			return (
				t.top >= 0 &&
				t.left >= 0 &&
				t.bottom <=
					(window.innerHeight || document.documentElement.clientHeight) &&
				t.right <=
					(window.innerWidth || document.documentElement.clientWidth) &&
				!b(e, t)
			);
		}
		function b(e, t) {
			let n = e;
			for (; n; ) {
				if (getComputedStyle(n).position === `fixed`) return !1;
				const e = n.parentElement;
				if (!e || e === document.body || e === document.documentElement)
					return !1;
				if (m(e)) {
					const n = e.getBoundingClientRect(),
						r = n.top + e.clientTop,
						i = n.left + e.clientLeft;
					if (
						t.top < r ||
						t.left < i ||
						t.bottom > r + (e.clientHeight || n.height) ||
						t.right > i + (e.clientWidth || n.width)
					)
						return !0;
				}
				n = e;
			}
			return !1;
		}
		function x(e) {
			return !!(e.offsetWidth || e.offsetHeight || e.getClientRects().length);
		}
		const S = new WeakSet();
		function C(e) {
			const t = e.closest(`dialog[open]`);
			if (!t) return document.body;
			if (!(`showPopover` in t)) return t;
			let n = t.querySelector(`:scope > .driver-layer`);
			return (
				n ||
					((n = document.createElement(`div`)),
					(n.className = `driver-layer`),
					n.setAttribute(`popover`, `manual`),
					t.appendChild(n)),
				S.has(t) ||
					(S.add(t),
					t.addEventListener(`close`, () => {
						t.querySelector(
							`:scope > .driver-layer:popover-open`,
						)?.hidePopover();
					})),
				n.matches(`:popover-open`) || n.showPopover(),
				n
			);
		}
		function w(e) {
			const t = e.parentElement;
			e.remove(),
				t?.classList.contains(`driver-layer`) &&
					!t.childElementCount &&
					t.remove();
		}
		function T(e) {
			e && (e.wrapper.style.display = `none`);
		}
		function ee(e, t) {
			const n = D();
			C(e).appendChild(n.wrapper);
			const {
				title: i,
				description: a,
				showButtons: o,
				disableButtons: s,
				showProgress: l,
				nextBtnText: u,
				prevBtnText: d,
				progressText: f,
			} = t;
			(n.nextButton.innerHTML = u),
				(n.previousButton.innerHTML = d),
				(n.progress.innerHTML = f),
				n.closeButton.setAttribute(`aria-label`, t.closeBtnLabel),
				t.doneButton && n.nextButton.classList.add(`driver-popover-done-btn`),
				i
					? ((n.title.innerHTML = i), (n.title.style.display = `block`))
					: (n.title.style.display = `none`),
				a
					? ((n.description.innerHTML = a),
						(n.description.style.display = `block`))
					: (n.description.style.display = `none`);
			const p = o.includes(`next`) || o.includes(`previous`) || l;
			(n.closeButton.style.display = o.includes(`close`) ? `block` : `none`),
				p
					? ((n.footer.style.display = `flex`),
						(n.progress.style.display = l ? `block` : `none`),
						(n.nextButton.style.display = o.includes(`next`)
							? `block`
							: `none`),
						(n.previousButton.style.display = o.includes(`previous`)
							? `block`
							: `none`))
					: (n.footer.style.display = `none`),
				s.includes(`next`) &&
					((n.nextButton.disabled = !0),
					n.nextButton.classList.add(`driver-popover-btn-disabled`)),
				s.includes(`previous`) &&
					((n.previousButton.disabled = !0),
					n.previousButton.classList.add(`driver-popover-btn-disabled`)),
				s.includes(`close`) &&
					((n.closeButton.disabled = !0),
					n.closeButton.classList.add(`driver-popover-btn-disabled`));
			const m = n.wrapper;
			(m.style.display = `block`),
				(m.style.left = ``),
				(m.style.top = ``),
				(m.style.bottom = ``),
				(m.style.right = ``),
				(m.id = `driver-popover-content`),
				m.setAttribute(`role`, `dialog`),
				m.setAttribute(`aria-labelledby`, `driver-popover-title`),
				m.setAttribute(`aria-describedby`, `driver-popover-description`);
			const h = n.arrow;
			(h.className = `driver-popover-arrow`),
				(m.className = `driver-popover ${t.popoverClass || ``}`.trim()),
				r(
					n.wrapper,
					(e) => {
						const n =
							e.target instanceof Element ? e.target : e.target?.parentElement;
						if (n) {
							if (n.closest(`.driver-popover-next-btn`))
								return t.onNextClick?.();
							if (n.closest(`.driver-popover-prev-btn`))
								return t.onPrevClick?.();
							if (n.closest(`.driver-popover-close-btn`))
								return t.onCloseClick?.();
						}
					},
					(e) =>
						n.description.contains(e) || n.title.contains(e)
							? !1
							: !!e.closest(
									`.driver-popover-prev-btn, .driver-popover-next-btn, .driver-popover-close-btn`,
								),
				),
				t.onRender?.(n),
				c(n, e, t.position),
				E(n, e, t.position),
				_(m, t.smoothScroll);
			const v = g([m, e]);
			return v.length > 0 && v[0].focus(), n;
		}
		function E(e, t, n) {
			e.wrapper.querySelectorAll(`img`).forEach((r) => {
				if (r.complete) return;
				const i = () => c(e, t, n);
				r.addEventListener(`load`, i, { once: !0 }),
					r.addEventListener(`error`, i, { once: !0 });
			});
		}
		function D() {
			const e = document.createElement(`div`);
			e.classList.add(`driver-popover`);
			const t = document.createElement(`div`);
			t.classList.add(`driver-popover-arrow`);
			const n = document.createElement(`div`);
			(n.id = `driver-popover-title`),
				n.classList.add(`driver-popover-title`),
				(n.style.display = `none`),
				(n.innerText = `Popover Title`);
			const r = document.createElement(`div`);
			(r.id = `driver-popover-description`),
				r.classList.add(`driver-popover-description`),
				(r.style.display = `none`),
				(r.innerText = `Popover description is here`);
			const i = document.createElement(`button`);
			(i.type = `button`),
				i.classList.add(`driver-popover-close-btn`),
				(i.innerHTML = `&times;`);
			const a = document.createElement(`div`);
			a.classList.add(`driver-popover-footer`);
			const o = document.createElement(`span`);
			o.classList.add(`driver-popover-progress-text`), (o.innerText = ``);
			const s = document.createElement(`span`);
			s.classList.add(`driver-popover-navigation-btns`);
			const c = document.createElement(`button`);
			(c.type = `button`),
				c.classList.add(`driver-popover-prev-btn`, `driver-popover-footer-btn`),
				(c.innerHTML = `Previous`);
			const l = document.createElement(`button`);
			return (
				(l.type = `button`),
				l.classList.add(`driver-popover-next-btn`, `driver-popover-footer-btn`),
				(l.innerHTML = `Next`),
				s.appendChild(c),
				s.appendChild(l),
				a.appendChild(o),
				a.appendChild(s),
				e.appendChild(i),
				e.appendChild(t),
				e.appendChild(n),
				e.appendChild(r),
				e.appendChild(a),
				{
					wrapper: e,
					arrow: t,
					title: n,
					description: r,
					footer: a,
					previousButton: c,
					nextButton: l,
					closeButton: i,
					footerButtons: s,
					progress: o,
				}
			);
		}
		function O(e) {
			e && (i(e.wrapper), w(e.wrapper));
		}
		function k(e, t) {
			const n = window.innerWidth,
				r = window.innerHeight,
				i = t.padding,
				a = t.radius,
				o = e.width + i * 2,
				s = e.height + i * 2,
				c = Math.min(a, o / 2, s / 2),
				l = Math.floor(Math.max(c, 0)),
				u = e.x - i + l,
				d = e.y - i,
				f = o - l * 2,
				p = s - l * 2;
			return `M${n},0L0,0L0,${r}L${n},${r}L${n},0Z
    M${u},${d} h${f} a${l},${l} 0 0 1 ${l},${l} v${p} a${l},${l} 0 0 1 -${l},${l} h-${f} a${l},${l} 0 0 1 -${l},-${l} v-${p} a${l},${l} 0 0 1 ${l},-${l} z`;
		}
		function te(e, t, n, r, i) {
			let a = e.getState(`__activeStagePosition`),
				o = a || r.getBoundingClientRect(),
				s = i.getBoundingClientRect();
			(a = {
				x: h(t, o.x, s.x - o.x, n),
				y: h(t, o.y, s.y - o.y, n),
				width: h(t, o.width, s.width - o.width, n),
				height: h(t, o.height, s.height - o.height, n),
			}),
				M(e, a),
				e.setState(`__activeStagePosition`, a);
		}
		function A(e, t) {
			if (!t) return;
			const n = t.getBoundingClientRect(),
				r = { x: n.x, y: n.y, width: n.width, height: n.height };
			e.setState(`__activeStagePosition`, r), M(e, r);
		}
		function j(e) {
			const t = e.getState(`__activeStagePosition`),
				n = e.getState(`__overlaySvg`);
			if (!t) return;
			if (!n) {
				console.warn(`No stage svg found.`);
				return;
			}
			const r = window.innerWidth,
				i = window.innerHeight;
			n.setAttribute(`viewBox`, `0 0 ${r} ${i}`);
		}
		function ne(e, t) {
			const n = re(e, t);
			document.body.appendChild(n),
				r(n, (t) => {
					t.target.tagName === `path` && e.emit(`overlayClick`);
				}),
				e.setState(`__overlaySvg`, n);
		}
		function M(e, t) {
			const n = e.getState(`__overlaySvg`);
			if (!n) {
				ne(e, t);
				return;
			}
			const r = n.firstElementChild;
			if (r?.tagName !== `path`)
				throw Error(`no path element found in stage svg`);
			r.setAttribute(`d`, k(t, N(e)));
		}
		function N(e) {
			return e.getState(`activeElement`)?.id === `driver-dummy-element`
				? { padding: 0, radius: 0 }
				: {
						padding: e.getConfig(`stagePadding`) || 0,
						radius: e.getConfig(`stageRadius`) || 0,
					};
		}
		function re(e, t) {
			const n = window.innerWidth,
				r = window.innerHeight,
				i = document.createElementNS(`http://www.w3.org/2000/svg`, `svg`);
			i.classList.add(`driver-overlay`, `driver-overlay-animated`),
				i.setAttribute(`viewBox`, `0 0 ${n} ${r}`),
				i.setAttribute(`xmlSpace`, `preserve`),
				i.setAttribute(`xmlnsXlink`, `http://www.w3.org/1999/xlink`),
				i.setAttribute(`version`, `1.1`),
				i.setAttribute(`preserveAspectRatio`, `xMinYMin slice`),
				(i.style.fillRule = `evenodd`),
				(i.style.clipRule = `evenodd`),
				(i.style.strokeLinejoin = `round`),
				(i.style.strokeMiterlimit = `2`),
				(i.style.zIndex = `10000`),
				(i.style.position = `fixed`),
				(i.style.top = `0`),
				(i.style.left = `0`),
				(i.style.width = `100%`),
				(i.style.height = `100%`);
			const a = document.createElementNS(`http://www.w3.org/2000/svg`, `path`);
			return (
				a.setAttribute(`d`, k(t, N(e))),
				(a.style.fill = e.getConfig(`overlayColor`) || `rgb(0,0,0)`),
				(a.style.opacity = `${e.getConfig(`overlayOpacity`)}`),
				(a.style.pointerEvents = `auto`),
				(a.style.cursor = `auto`),
				i.appendChild(a),
				i
			);
		}
		function P(e) {
			const t = e.getState(`__overlaySvg`);
			t && (i(t), t.remove());
		}
		const F = `{{current}} of {{total}}`;
		function I(e, t) {
			return !(t.skipMissingElement ?? e.getConfig(`skipMissingElement`)) ||
				!t.element
				? !1
				: !p(t.element);
		}
		function L(e, t, n) {
			const r = e.getConfig(`steps`) || [];
			for (let i = t; i >= 0 && i < r.length; i += n) if (!I(e, r[i])) return i;
		}
		function R(e, t) {
			const n = e.getState(`activeIndex`),
				r = n !== void 0 && L(e, n + 1, 1) === void 0,
				i = t?.popover?.onDoneClick || e.getConfig(`onDoneClick`);
			return r && i ? i : t?.popover?.onNextClick || e.getConfig(`onNextClick`);
		}
		function z(e, t) {
			return t?.popover?.onPrevClick || e.getConfig(`onPrevClick`);
		}
		function ie(e, t) {
			return t?.popover?.onCloseClick || e.getConfig(`onCloseClick`);
		}
		function B(e, t, n) {
			const r = e.getConfig(`steps`) || [],
				i = r[t],
				a = i.popover || {},
				o = L(e, t + 1, 1) !== void 0,
				s = L(e, t - 1, -1) !== void 0,
				c = a.doneBtnText || e.getConfig(`doneBtnText`) || `Done`,
				l = e.getConfig(`allowClose`),
				u =
					a.showProgress === void 0
						? e.getConfig(`showProgress`)
						: a.showProgress,
				d = (a.progressText || e.getConfig(`progressText`) || F)
					.replace(`{{current}}`, `${t + 1}`)
					.replace(`{{total}}`, `${r.length}`),
				f = a.showButtons || e.getConfig(`showButtons`),
				p = [`next`, `previous`, ...(l ? [`close`] : [])].filter(
					(e) => !f?.length || f.includes(e),
				),
				m = a.onNextClick || e.getConfig(`onNextClick`),
				h = a.onPrevClick || e.getConfig(`onPrevClick`),
				g = a.onCloseClick || e.getConfig(`onCloseClick`);
			return {
				...i,
				popover: {
					showButtons: p,
					nextBtnText: o ? void 0 : c,
					disableButtons: [
						...(e.getConfig(`disableButtons`) || []),
						...(s ? [] : [`previous`]),
					],
					showProgress: u,
					onNextClick: m || n.onNextClick,
					onPrevClick: h || n.onPrevClick,
					onCloseClick: g || n.onCloseClick,
					...a,
					progressText: d,
				},
			};
		}
		function V(e, t) {
			return {
				showButtons:
					t?.popover?.showButtons || e.getConfig(`showButtons`) || [],
				disableButtons:
					t?.popover?.disableButtons || e.getConfig(`disableButtons`) || [],
			};
		}
		function H(e, t, n) {
			const r = e.getConfig(`stagePadding`) || 0;
			return {
				side: n.popover?.side || `bottom`,
				align: n.popover?.align || `start`,
				offset: r + (e.getConfig(`popoverOffset`) || 0),
				padding: r,
				centered: t.id === `driver-dummy-element`,
			};
		}
		function U(e, t, n) {
			const r = n.popover || {},
				i = e.getState(`activeIndex`),
				a = i !== void 0 && L(e, i + 1, 1) === void 0;
			return {
				title: r.title,
				description: r.description,
				...V(e, n),
				showProgress: r.showProgress || e.getConfig(`showProgress`) || !1,
				progressText: r.progressText ?? (e.getConfig(`progressText`) || F),
				nextBtnText: r.nextBtnText ?? (e.getConfig(`nextBtnText`) || `Next`),
				prevBtnText:
					r.prevBtnText ?? (e.getConfig(`prevBtnText`) || `Previous`),
				closeBtnLabel:
					r.closeBtnLabel ?? (e.getConfig(`closeBtnLabel`) || `Close`),
				doneButton: a,
				popoverClass: r.popoverClass || e.getConfig(`popoverClass`) || ``,
				smoothScroll: e.getConfig(`smoothScroll`),
				onNextClick: () => {
					a && e.setState(`__destroyReason`, `done`);
					const r = R(e, n);
					return r ? r(t, n, e.getHookOpts()) : e.emit(`nextClick`);
				},
				onPrevClick: () => {
					const r = z(e, n);
					return r ? r(t, n, e.getHookOpts()) : e.emit(`prevClick`);
				},
				onCloseClick: () => {
					e.setState(`__destroyReason`, `close`);
					const r = ie(e, n);
					return r ? r(t, n, e.getHookOpts()) : e.emit(`closeClick`);
				},
				onRender: (t) => {
					e.setState(`popover`, t),
						(r.onPopoverRender || e.getConfig(`onPopoverRender`))?.(
							t,
							e.getHookOpts(),
						);
				},
				position: H(e, t, n),
			};
		}
		function W(e, t, n) {
			O(e.getState(`popover`)), ee(t, U(e, t, n));
		}
		function G(e, t, n) {
			const r = e.getState(`popover`);
			r && c(r, t, H(e, t, n));
		}
		const K = {
				"aria-haspopup": `dialog`,
				"aria-expanded": `true`,
				"aria-controls": `driver-popover-content`,
			},
			q = new WeakMap();
		function J(e) {
			if (!q.has(e)) {
				const t = Object.fromEntries(
					Object.keys(K).map((t) => [t, e.getAttribute(t)]),
				);
				q.set(e, t);
			}
			for (const [t, n] of Object.entries(K)) e.setAttribute(t, n);
		}
		function Y(e) {
			const t = q.get(e);
			if (t) {
				for (const [n, r] of Object.entries(t))
					r === null ? e.removeAttribute(n) : e.setAttribute(n, r);
				q.delete(e);
			}
		}
		function ae() {
			const e = document.getElementById(`driver-dummy-element`);
			if (e) return e;
			const t = document.createElement(`div`);
			return (
				(t.id = `driver-dummy-element`),
				(t.style.width = `0`),
				(t.style.height = `0`),
				(t.style.pointerEvents = `none`),
				(t.style.opacity = `0`),
				(t.style.position = `fixed`),
				(t.style.top = `50%`),
				(t.style.left = `50%`),
				document.body.appendChild(t),
				t
			);
		}
		function X(e, t) {
			let n = p(t.element);
			n || (n = ae()), se(e, n, t);
		}
		function oe(e) {
			const t = e.getState(`__activeElement`),
				n = e.getState(`__activeStep`);
			t && (A(e, t), j(e), G(e, t, n));
		}
		function se(e, t, n) {
			const r = e.getConfig(`duration`) || 400,
				i = Date.now(),
				a = e.getState(`__activeStep`),
				o = e.getState(`__activeElement`) || t,
				s = !o || o === t,
				c = t.id === `driver-dummy-element`,
				l = o.id === `driver-dummy-element`,
				u = e.getConfig(`animate`),
				d = n.onHighlightStarted || e.getConfig(`onHighlightStarted`),
				f = n?.onHighlighted || e.getConfig(`onHighlighted`),
				p = a?.onDeselected || e.getConfig(`onDeselected`),
				h = e.getHookOpts();
			a && a !== n && p && p(l ? void 0 : o, a, h),
				d && d(c ? void 0 : t, n, h);
			let g = !s && u,
				v = !1;
			T(e.getState(`popover`)),
				e.setState(`previousStep`, a),
				e.setState(`previousElement`, o),
				e.setState(`activeStep`, n),
				e.setState(`activeElement`, t);
			const y = () => {
				if (e.getState(`__transitionCallback`) !== y) return;
				const s = Date.now() - i,
					l = r - s <= r / 2;
				n.popover && l && !v && g && (W(e, t, n), (v = !0)),
					e.getConfig(`animate`) && s < r
						? te(e, s, r, o, t)
						: (A(e, t),
							f && f(c ? void 0 : t, n, e.getHookOpts()),
							e.setState(`__transitionCallback`, void 0),
							e.setState(`__previousStep`, a),
							e.setState(`__previousElement`, o),
							e.setState(`__activeStep`, n),
							e.setState(`__activeElement`, t)),
					window.requestAnimationFrame(y);
			};
			e.setState(`__transitionCallback`, y),
				window.requestAnimationFrame(y),
				document
					.querySelectorAll(`.driver-active-element-parent`)
					.forEach((e) => {
						e.classList.remove(
							`driver-active-element-parent`,
							`driver-active-element-parent-no-scroll`,
						);
					}),
				_(t, e.getConfig(`smoothScroll`)),
				!g && n.popover && W(e, t, n),
				document.querySelectorAll(`.driver-active-element`).forEach((e) => {
					e.classList.remove(`driver-active-element`, `driver-no-interaction`),
						Y(e);
				}),
				(n.disableActiveInteraction ??
					e.getConfig(`disableActiveInteraction`)) &&
					t.classList.add(`driver-no-interaction`);
			const b = t.parentElement;
			b &&
				b !== document.body &&
				(b.classList.add(`driver-active-element-parent`),
				m(b) && b.classList.add(`driver-active-element-parent-no-scroll`)),
				t.classList.add(`driver-active-element`),
				t.id !== `driver-dummy-element` && J(t);
		}
		function ce() {
			document.getElementById(`driver-dummy-element`)?.remove(),
				document.querySelectorAll(`.driver-active-element`).forEach((e) => {
					const t = e.parentElement;
					t &&
						t !== document.body &&
						t.classList.remove(
							`driver-active-element-parent`,
							`driver-active-element-parent-no-scroll`,
						),
						e.classList.remove(
							`driver-active-element`,
							`driver-no-interaction`,
						),
						Y(e);
				});
		}
		function Z(e) {
			const t = e.getState(`__resizeTimeout`);
			t && window.cancelAnimationFrame(t),
				e.setState(
					`__resizeTimeout`,
					window.requestAnimationFrame(() => oe(e)),
				);
		}
		function le(e, t) {
			if (!e.getState(`isInitialized`) || !(t.key === `Tab` || t.keyCode === 9))
				return;
			const n = e.getState(`__activeElement`),
				r = e.getState(`popover`)?.wrapper,
				i = g([...(r ? [r] : []), ...(n ? [n] : [])]),
				a = i[0],
				o = i[i.length - 1];
			t.preventDefault(),
				t.shiftKey
					? (i[i.indexOf(document.activeElement) - 1] || o)?.focus()
					: (i[i.indexOf(document.activeElement) + 1] || a)?.focus();
		}
		function ue(e, t) {
			(e.getConfig(`allowKeyboardControl`) ?? !0) &&
				(t.key === `Escape`
					? e.emit(`escapePress`)
					: t.key === `ArrowRight`
						? e.emit(`arrowRightPress`)
						: t.key === `ArrowLeft` && e.emit(`arrowLeftPress`));
		}
		function de(e, t) {
			const n = e.getState(`__activeElement`),
				r = t.target;
			!n || !r || !n.contains(r) || e.emit(`activeElementClick`);
		}
		function fe(e) {
			const t = (t) => ue(e, t),
				n = (t) => le(e, t),
				r = () => Z(e),
				i = () => Z(e),
				a = (t) => de(e, t);
			e.setState(`__events`, {
				onKeyup: t,
				onKeydown: n,
				onResize: r,
				onScroll: i,
				onClick: a,
			}),
				window.addEventListener(`keyup`, t, !1),
				window.addEventListener(`keydown`, n, !1),
				window.addEventListener(`resize`, r),
				window.addEventListener(`scroll`, i),
				document.addEventListener(`click`, a, !1);
		}
		function pe(e) {
			const t = e.getState(`__events`);
			t &&
				(window.removeEventListener(`keyup`, t.onKeyup),
				window.removeEventListener(`keydown`, t.onKeydown),
				window.removeEventListener(`resize`, t.onResize),
				window.removeEventListener(`scroll`, t.onScroll),
				document.removeEventListener(`click`, t.onClick, !1));
		}
		function Q() {
			let e = {};
			function t(t = {}) {
				e = {
					animate: !0,
					duration: 400,
					allowClose: !0,
					allowScroll: !0,
					overlayClickBehavior: `close`,
					overlayOpacity: 0.7,
					smoothScroll: !1,
					disableActiveInteraction: !1,
					advanceOnClick: !1,
					skipMissingElement: !1,
					waitForElement: 0,
					showProgress: !1,
					stagePadding: 10,
					stageRadius: 5,
					popoverOffset: 10,
					showButtons: [`next`, `previous`, `close`],
					disableButtons: [],
					overlayColor: `#000`,
					...t,
				};
			}
			return t(), { getConfig: (t) => (t ? e[t] : e), configure: t };
		}
		function me() {
			let e = {},
				t = (t) => (t ? e[t] : e),
				n = (t, n) => {
					e[t] = n;
				};
			function r() {
				e = {};
			}
			return { getState: t, setState: n, resetState: r };
		}
		function he() {
			let e = {};
			function t(t, n) {
				e[t] = n;
			}
			function n(t) {
				e[t]?.();
			}
			function r() {
				e = {};
			}
			return { listen: t, emit: n, reset: r };
		}
		function ge(e = {}) {
			const t = Q();
			t.configure(e);
			let n = me(),
				r = he(),
				i;
			return {
				getConfig: t.getConfig,
				setConfig: t.configure,
				getState: n.getState,
				setState: n.setState,
				resetState: n.resetState,
				listen: r.listen,
				emit: r.emit,
				resetEmitter: r.reset,
				getDriver: () => i,
				setDriver: (e) => {
					i = e;
				},
				getHookOpts: (e, r) => {
					const a = e || n.getState();
					return {
						config: t.getConfig(),
						state: a,
						driver: i,
						index: a.activeIndex,
						...(r ? { reason: r } : {}),
					};
				},
			};
		}
		let $;
		function _e(e = {}) {
			const t = ge(e),
				n = () => y(`replaced`, !1);
			function r(e = `close`) {
				t.getConfig(`allowClose`) && y(e);
			}
			function i() {
				const e = t.getState(`activeIndex`);
				e !== void 0 &&
					L(t, e + 1, 1) === void 0 &&
					t.setState(`__destroyReason`, `done`);
			}
			function a() {
				const { disableButtons: e } = V(t, t.getState(`__activeStep`));
				e.includes(`close`) || r(`escape`);
			}
			function o(e) {
				const { showButtons: n, disableButtons: r } = V(
					t,
					t.getState(`__activeStep`),
				);
				return n.includes(e) && !r.includes(e);
			}
			function s() {
				const e = t.getConfig(`overlayClickBehavior`);
				if (t.getConfig(`allowClose`) && e === `close`) {
					y(`overlay`);
					return;
				}
				if (typeof e == `function`) {
					t.setState(`__destroyReason`, `overlay`);
					const n = t.getState(`__activeStep`);
					e(t.getState(`__activeElement`), n, t.getHookOpts());
					return;
				}
				if (e === `nextStep`) {
					const e = t.getState(`activeStep`),
						n = t.getState(`activeElement`),
						r = R(t, e);
					if (r) {
						i(), r(n, e, t.getHookOpts());
						return;
					}
					c();
				}
			}
			function c() {
				const e = t.getState(`activeIndex`),
					n = t.getConfig(`steps`) || [];
				if (e === void 0) return;
				const r = e + 1;
				n[r] ? v(r) : y(`done`);
			}
			function l() {
				const e = t.getState(`activeIndex`),
					n = t.getConfig(`steps`) || [];
				if (e === void 0) return;
				const r = e - 1;
				n[r] ? v(r) : y(`api`);
			}
			function u(e) {
				(t.getConfig(`steps`) || [])[e] ? v(e) : y(`api`);
			}
			function d() {
				if (t.getState(`__transitionCallback`)) return;
				const e = t.getState(`__activeStep`);
				if (!e || !(e.advanceOnClick ?? t.getConfig(`advanceOnClick`))) return;
				const n = t.getState(`__activeElement`),
					r = R(t, e);
				if (r) {
					i(), r(n, e, t.getHookOpts());
					return;
				}
				c();
			}
			function f() {
				if (t.getState(`__transitionCallback`)) return;
				const e = t.getState(`activeIndex`),
					n = t.getState(`__activeStep`),
					r = t.getState(`__activeElement`);
				if (
					e === void 0 ||
					n === void 0 ||
					!(t.getConfig(`steps`) || [])[e - 1] ||
					!o(`previous`)
				)
					return;
				const i = z(t, n);
				if (i) return i(r, n, t.getHookOpts());
				l();
			}
			function m() {
				if (t.getState(`__transitionCallback`)) return;
				const e = t.getState(`activeIndex`),
					n = t.getState(`__activeStep`),
					r = t.getState(`__activeElement`);
				if (e === void 0 || n === void 0 || !o(`next`)) return;
				const a = R(t, n);
				if (a) return i(), a(r, n, t.getHookOpts());
				c();
			}
			function h() {
				t.getState(`isInitialized`) ||
					($ && $ !== n && $(),
					($ = n),
					t.setState(`isInitialized`, !0),
					document.body.classList.add(
						`driver-active`,
						t.getConfig(`animate`) ? `driver-fade` : `driver-simple`,
					),
					t.getConfig(`allowScroll`) ||
						document.body.classList.add(`driver-no-scroll`),
					document.body.style.setProperty(
						`--driver-animation-duration`,
						`${t.getConfig(`duration`) || 400}ms`,
					),
					fe(t),
					t.listen(`overlayClick`, s),
					t.listen(`activeElementClick`, d),
					t.listen(`escapePress`, a),
					t.listen(`closeClick`, r),
					t.listen(`arrowLeftPress`, f),
					t.listen(`arrowRightPress`, m));
			}
			function g() {
				const e = t.getState(`__pendingWaitCancel`);
				e && (t.setState(`__pendingWaitCancel`, void 0), e());
			}
			function _(e, n, r) {
				const i = () => {
						a.disconnect(),
							window.clearTimeout(o),
							t.setState(`__pendingWaitCancel`, void 0),
							r();
					},
					a = new MutationObserver(() => {
						p(e.element) && i();
					}),
					o = window.setTimeout(i, n);
				t.setState(`__pendingWaitCancel`, () => {
					a.disconnect(), window.clearTimeout(o);
				}),
					a.observe(document.documentElement, {
						childList: !0,
						subtree: !0,
						attributes: !0,
					});
			}
			function v(e = 0, n = !1) {
				g(), t.setState(`__destroyReason`, void 0);
				const r = t.getConfig(`steps`);
				if (!r) {
					console.error(`No steps to drive through`), y(`api`);
					return;
				}
				if (!r[e]) {
					y(`api`);
					return;
				}
				const i = r[e],
					a = i.waitForElement ?? t.getConfig(`waitForElement`) ?? 0;
				if (!n && a > 0 && i.element && !p(i.element)) {
					T(t.getState(`popover`)), _(i, a, () => v(e, !0));
					return;
				}
				if (I(t, i)) {
					const n = t.getState(`activeIndex`),
						i = typeof n == `number` && e < n ? -1 : 1;
					r[e + i] ? v(e + i) : i === 1 && y(`done`);
					return;
				}
				t.setState(`__activeOnDestroyed`, document.activeElement),
					t.setState(`activeIndex`, e);
				const o = r[e + 1];
				X(
					t,
					B(t, e, {
						onNextClick: () => {
							o ? v(e + 1) : y(`done`);
						},
						onPrevClick: () => {
							v(e - 1);
						},
						onCloseClick: () => {
							y(`close`);
						},
					}),
				);
			}
			function y(e, r = !0) {
				if (!t.getState(`isInitialized`)) return;
				e === `api` && (e = t.getState(`__destroyReason`) ?? `api`);
				const i = t.getState(`__activeElement`),
					a = t.getState(`__activeStep`),
					o = t.getState(`__activeOnDestroyed`),
					s = t.getConfig(`onDestroyStarted`);
				if (r && s) {
					t.setState(`__destroyReason`, e),
						s(
							!i || i?.id === `driver-dummy-element` ? void 0 : i,
							a,
							t.getHookOpts(void 0, e),
						);
					return;
				}
				$ === n && ($ = void 0);
				const c = a?.onDeselected || t.getConfig(`onDeselected`),
					l = t.getConfig(`onDestroyed`);
				document.body.classList.remove(
					`driver-active`,
					`driver-fade`,
					`driver-simple`,
					`driver-no-scroll`,
				),
					document.body.style.removeProperty(`--driver-animation-duration`),
					g(),
					pe(t),
					O(t.getState(`popover`)),
					ce(),
					P(t),
					t.resetEmitter();
				const u = t.getState();
				if ((t.resetState(), i && a)) {
					const n = i.id === `driver-dummy-element`;
					c && c(n ? void 0 : i, a, t.getHookOpts(u, e)),
						l && l(n ? void 0 : i, a, t.getHookOpts(u, e));
				}
				o && o.focus();
			}
			const b = {
				isActive: () => t.getState(`isInitialized`) || !1,
				refresh: () => Z(t),
				drive: (e = 0) => {
					h(), v(e);
				},
				setConfig: t.setConfig,
				setSteps: (e) => {
					g(), t.resetState(), t.setConfig({ ...t.getConfig(), steps: e });
				},
				getConfig: t.getConfig,
				getState: t.getState,
				getActiveIndex: () => t.getState(`activeIndex`),
				isFirstStep: () => {
					const e = t.getState(`activeIndex`);
					return e !== void 0 && L(t, e - 1, -1) === void 0;
				},
				isLastStep: () => {
					const e = t.getState(`activeIndex`);
					return e !== void 0 && L(t, e + 1, 1) === void 0;
				},
				getActiveStep: () => t.getState(`activeStep`),
				getActiveElement: () => t.getState(`activeElement`),
				getPreviousElement: () => t.getState(`previousElement`),
				getPreviousStep: () => t.getState(`previousStep`),
				getNextStep: () => {
					const e = t.getConfig(`steps`) || [],
						n = t.getState(`activeIndex`);
					if (n === void 0) return;
					const r = L(t, n + 1, 1);
					return r === void 0 ? void 0 : e[r];
				},
				moveNext: c,
				movePrevious: l,
				moveTo: u,
				hasNextStep: () => {
					const e = t.getState(`activeIndex`);
					return e !== void 0 && L(t, e + 1, 1) !== void 0;
				},
				hasPreviousStep: () => {
					const e = t.getState(`activeIndex`);
					return e !== void 0 && L(t, e - 1, -1) !== void 0;
				},
				highlight: (e) => {
					h(),
						t.setState(`__destroyReason`, void 0),
						X(t, {
							...e,
							popover: e.popover
								? {
										showButtons: [],
										showProgress: !1,
										progressText: ``,
										...e.popover,
									}
								: void 0,
						});
				},
				destroy: () => {
					y(`api`, !1);
				},
			};
			return t.setDriver(b), b;
		}
		return (e.driver = _e), e;
	})({}));
