var Qe = Object.defineProperty;
var Ye = (t, e) => {
	for (var n in e) Qe(t, n, { get: e[n], enumerable: !0 });
};
var Bn = document.createElement("i");
function Pt(t) {
	const e = "&" + t + ";";
	Bn.innerHTML = e;
	const n = Bn.textContent;
	return (n.charCodeAt(n.length - 1) === 59 && t !== "semi") || n === e
		? !1
		: n;
}
function V(t, e, n, r) {
	let l = t.length,
		u = 0,
		i;
	if (
		(e < 0 ? (e = -e > l ? 0 : l + e) : (e = e > l ? l : e),
		(n = n > 0 ? n : 0),
		r.length < 1e4)
	)
		(i = Array.from(r)), i.unshift(e, n), t.splice(...i);
	else
		for (n && t.splice(e, n); u < r.length; )
			(i = r.slice(u, u + 1e4)),
				i.unshift(e, 0),
				t.splice(...i),
				(u += 1e4),
				(e += 1e4);
}
function j(t, e) {
	return t.length > 0 ? (V(t, t.length, 0, e), t) : e;
}
var Kt = {}.hasOwnProperty;
function Bt(t) {
	let e = {},
		n = -1;
	for (; ++n < t.length; ) $e(e, t[n]);
	return e;
}
function $e(t, e) {
	let n;
	for (n in e) {
		let l = (Kt.call(t, n) ? t[n] : void 0) || (t[n] = {}),
			u = e[n],
			i;
		if (u)
			for (i in u) {
				Kt.call(l, i) || (l[i] = []);
				const o = u[i];
				Ze(l[i], Array.isArray(o) ? o : o ? [o] : []);
			}
	}
}
function Ze(t, e) {
	let n = -1,
		r = [];
	for (; ++n < e.length; ) (e[n].add === "after" ? t : r).push(e[n]);
	V(t, 0, 0, r);
}
function Mt(t) {
	let e = {},
		n = -1;
	for (; ++n < t.length; ) Ge(e, t[n]);
	return e;
}
function Ge(t, e) {
	let n;
	for (n in e) {
		let l = (Kt.call(t, n) ? t[n] : void 0) || (t[n] = {}),
			u = e[n],
			i;
		if (u) for (i in u) l[i] = u[i];
	}
}
function Mn(t, e) {
	const n = Number.parseInt(t, e);
	return n < 9 ||
		n === 11 ||
		(n > 13 && n < 32) ||
		(n > 126 && n < 160) ||
		(n > 55295 && n < 57344) ||
		(n > 64975 && n < 65008) ||
		(n & 65535) === 65535 ||
		(n & 65535) === 65534 ||
		n > 1114111
		? "\uFFFD"
		: String.fromCodePoint(n);
}
var Je = { '"': "quot", "&": "amp", "<": "lt", ">": "gt" };
function Rt(t) {
	return t.replace(/["&<>]/g, e);
	function e(n) {
		return "&" + Je[n] + ";";
	}
}
function G(t) {
	return t
		.replace(/[\t\n\r ]+/g, " ")
		.replace(/^ | $/g, "")
		.toLowerCase()
		.toUpperCase();
}
var U = pt(/[A-Za-z]/),
	q = pt(/[\dA-Za-z]/),
	Rn = pt(/[#-'*+\--9=?A-Z^-~]/);
function ft(t) {
	return t !== null && (t < 32 || t === 127);
}
var It = pt(/\d/),
	Nn = pt(/[\dA-Fa-f]/),
	Hn = pt(/[!-/:-@[-`{-~]/);
function w(t) {
	return t !== null && t < -2;
}
function O(t) {
	return t !== null && (t < 0 || t === 32);
}
function F(t) {
	return t === -2 || t === -1 || t === 32;
}
var zt = pt(/\p{P}|\p{S}/u),
	mt = pt(/\s/);
function pt(t) {
	return e;
	function e(n) {
		return n !== null && n > -1 && t.test(String.fromCharCode(n));
	}
}
function rt(t, e) {
	const n = Rt(Xe(t || ""));
	if (!e) return n;
	const r = n.indexOf(":"),
		l = n.indexOf("?"),
		u = n.indexOf("#"),
		i = n.indexOf("/");
	return r < 0 ||
		(i > -1 && r > i) ||
		(l > -1 && r > l) ||
		(u > -1 && r > u) ||
		e.test(n.slice(0, r))
		? n
		: "";
}
function Xe(t) {
	let e = [],
		n = -1,
		r = 0,
		l = 0;
	for (; ++n < t.length; ) {
		let u = t.charCodeAt(n),
			i = "";
		if (u === 37 && q(t.charCodeAt(n + 1)) && q(t.charCodeAt(n + 2))) l = 2;
		else if (u < 128)
			/[!#$&-;=?-Z_a-z~]/.test(String.fromCharCode(u)) ||
				(i = String.fromCharCode(u));
		else if (u > 55295 && u < 57344) {
			const o = t.charCodeAt(n + 1);
			u < 56320 && o > 56319 && o < 57344
				? ((i = String.fromCharCode(u, o)), (l = 1))
				: (i = "\uFFFD");
		} else i = String.fromCharCode(u);
		i &&
			(e.push(t.slice(r, n), encodeURIComponent(i)), (r = n + l + 1), (i = "")),
			l && ((n += l), (l = 0));
	}
	return e.join("") + t.slice(r);
}
var qn = {}.hasOwnProperty,
	Vn = /^(https?|ircs?|mailto|xmpp)$/i,
	Ke = /^https?$/i;
function jn(t) {
	let e = t || {},
		n = !0,
		r = {},
		l = [[]],
		u = [],
		i = [],
		a = Mt([
			{
				enter: {
					blockQuote: _,
					codeFenced: tt,
					codeFencedFenceInfo: L,
					codeFencedFenceMeta: L,
					codeIndented: lt,
					codeText: Re,
					content: Ie,
					definition: de,
					definitionDestinationString: we,
					definitionLabelString: L,
					definitionTitleString: L,
					emphasis: Be,
					htmlFlow: Pe,
					htmlText: Tn,
					image: J,
					label: L,
					link: bt,
					listItemMarker: Q,
					listItemValue: k,
					listOrdered: R,
					listUnordered: N,
					paragraph: W,
					reference: L,
					resource: Ct,
					resourceDestinationString: ot,
					resourceTitleString: L,
					setextHeading: Fe,
					strong: Me,
				},
				exit: {
					atxHeading: Le,
					atxHeadingSequence: ze,
					autolinkEmail: Ue,
					autolinkProtocol: We,
					blockQuote: H,
					characterEscapeValue: Et,
					characterReferenceMarkerHexadecimal: Dn,
					characterReferenceMarkerNumeric: Dn,
					characterReferenceValue: je,
					codeFenced: f,
					codeFencedFence: X,
					codeFencedFenceInfo: h,
					codeFencedFenceMeta: P,
					codeFlowValue: Oe,
					codeIndented: f,
					codeText: Ne,
					codeTextData: Et,
					data: Et,
					definition: Ee,
					definitionDestinationString: ye,
					definitionLabelString: Se,
					definitionTitleString: Ce,
					emphasis: He,
					hardBreakEscape: An,
					hardBreakTrailing: An,
					htmlFlow: Ln,
					htmlFlowData: Et,
					htmlText: Ln,
					htmlTextData: Et,
					image: Ot,
					label: _t,
					labelText: Xt,
					lineEnding: _e,
					link: Ot,
					listOrdered: M,
					listUnordered: I,
					paragraph: K,
					reference: P,
					referenceString: Y,
					resource: P,
					resourceDestinationString: dt,
					resourceTitleString: St,
					setextHeading: De,
					setextHeadingLineSequence: Te,
					setextHeadingText: Ae,
					strong: qe,
					thematicBreak: Ve,
				},
			},
			...(e.htmlExtensions || []),
		]),
		c = { definitions: r, tightStack: i },
		m = {
			buffer: L,
			encode: b,
			getData: E,
			lineEndingIfNeeded: T,
			options: e,
			raw: y,
			resume: d,
			setData: x,
			tag: C,
		},
		s = e.defaultLineEnding;
	return g;
	function g(S) {
		let z = -1,
			$ = 0,
			v = [],
			nt = [],
			at = [];
		for (; ++z < S.length; )
			!s &&
				(S[z][1].type === "lineEnding" || S[z][1].type === "lineEndingBlank") &&
				(s = S[z][2].sliceSerialize(S[z][1])),
				(S[z][1].type === "listOrdered" || S[z][1].type === "listUnordered") &&
					(S[z][0] === "enter" ? v.push(z) : p(S.slice(v.pop(), z))),
				S[z][1].type === "definition" &&
					(S[z][0] === "enter"
						? ((at = j(at, S.slice($, z))), ($ = z))
						: ((nt = j(nt, S.slice($, z + 1))), ($ = z + 1)));
		(nt = j(nt, at)), (nt = j(nt, S.slice($))), (z = -1);
		const et = nt;
		for (a.enter.null && a.enter.null.call(m); ++z < S.length; ) {
			const _n = a[et[z][0]],
				On = et[z][1].type,
				Pn = _n[On];
			qn.call(_n, On) &&
				Pn &&
				Pn.call({ sliceSerialize: et[z][2].sliceSerialize, ...m }, et[z][1]);
		}
		return a.exit.null && a.exit.null.call(m), l[0].join("");
	}
	function p(S) {
		let z = S.length,
			$ = 0,
			v = 0,
			nt = !1,
			at;
		for (; ++$ < z; ) {
			const et = S[$];
			if (et[1]._container) (at = void 0), et[0] === "enter" ? v++ : v--;
			else
				switch (et[1].type) {
					case "listItemPrefix": {
						et[0] === "exit" && (at = !0);
						break;
					}
					case "linePrefix":
						break;
					case "lineEndingBlank": {
						et[0] === "enter" && !v && (at ? (at = void 0) : (nt = !0));
						break;
					}
					default:
						at = void 0;
				}
		}
		S[0][1]._loose = nt;
	}
	function x(S, z) {
		c[S] = z;
	}
	function E(S) {
		return c[S];
	}
	function L() {
		l.push([]);
	}
	function d() {
		return l.pop().join("");
	}
	function C(S) {
		n && (x("lastWasTag", !0), l[l.length - 1].push(S));
	}
	function y(S) {
		x("lastWasTag"), l[l.length - 1].push(S);
	}
	function B() {
		y(
			s ||
				`
`,
		);
	}
	function T() {
		const S = l[l.length - 1],
			z = S[S.length - 1],
			$ = z ? z.charCodeAt(z.length - 1) : null;
		$ === 10 || $ === 13 || $ === null || B();
	}
	function b(S) {
		return E("ignoreEncode") ? S : Rt(S);
	}
	function P() {
		d();
	}
	function R(S) {
		i.push(!S._loose), T(), C("<ol"), x("expectFirstItem", !0);
	}
	function N(S) {
		i.push(!S._loose), T(), C("<ul"), x("expectFirstItem", !0);
	}
	function k(S) {
		if (E("expectFirstItem")) {
			const z = Number.parseInt(this.sliceSerialize(S), 10);
			z !== 1 && C(' start="' + b(String(z)) + '"');
		}
	}
	function Q() {
		E("expectFirstItem") ? C(">") : D(),
			T(),
			C("<li>"),
			x("expectFirstItem"),
			x("lastWasTag");
	}
	function M() {
		D(), i.pop(), B(), C("</ol>");
	}
	function I() {
		D(), i.pop(), B(), C("</ul>");
	}
	function D() {
		E("lastWasTag") && !E("slurpAllLineEndings") && T(),
			C("</li>"),
			x("slurpAllLineEndings");
	}
	function _() {
		i.push(!1), T(), C("<blockquote>");
	}
	function H() {
		i.pop(), T(), C("</blockquote>"), x("slurpAllLineEndings");
	}
	function W() {
		i[i.length - 1] || (T(), C("<p>")), x("slurpAllLineEndings");
	}
	function K() {
		i[i.length - 1] ? x("slurpAllLineEndings", !0) : C("</p>");
	}
	function tt() {
		T(), C("<pre><code"), x("fencesCount", 0);
	}
	function h() {
		const S = d();
		C(' class="language-' + S + '"');
	}
	function X() {
		const S = E("fencesCount") || 0;
		S || (C(">"), x("slurpOneLineEnding", !0)), x("fencesCount", S + 1);
	}
	function lt() {
		T(), C("<pre><code>");
	}
	function f() {
		const S = E("fencesCount");
		S !== void 0 && S < 2 && c.tightStack.length > 0 && !E("lastWasTag") && B(),
			E("flowCodeSeenData") && T(),
			C("</code></pre>"),
			S !== void 0 && S < 2 && T(),
			x("flowCodeSeenData"),
			x("fencesCount"),
			x("slurpOneLineEnding");
	}
	function J() {
		u.push({ image: !0 }), (n = void 0);
	}
	function bt() {
		u.push({});
	}
	function Xt(S) {
		u[u.length - 1].labelId = this.sliceSerialize(S);
	}
	function _t() {
		u[u.length - 1].label = d();
	}
	function Y(S) {
		u[u.length - 1].referenceId = this.sliceSerialize(S);
	}
	function Ct() {
		L(), (u[u.length - 1].destination = "");
	}
	function ot() {
		L(), x("ignoreEncode", !0);
	}
	function dt() {
		(u[u.length - 1].destination = d()), x("ignoreEncode");
	}
	function St() {
		u[u.length - 1].title = d();
	}
	function Ot() {
		let S = u.length - 1,
			z = u[S],
			$ = z.referenceId || z.labelId,
			v = z.destination === void 0 ? r[G($)] : z;
		for (n = !0; S--; )
			if (u[S].image) {
				n = void 0;
				break;
			}
		z.image
			? (C(
					'<img src="' +
						rt(v.destination, e.allowDangerousProtocol ? void 0 : Ke) +
						'" alt="',
				),
				y(z.label),
				C('"'))
			: C(
					'<a href="' +
						rt(v.destination, e.allowDangerousProtocol ? void 0 : Vn) +
						'"',
				),
			C(v.title ? ' title="' + v.title + '"' : ""),
			z.image ? C(" />") : (C(">"), y(z.label), C("</a>")),
			u.pop();
	}
	function de() {
		L(), u.push({});
	}
	function Se(S) {
		d(), (u[u.length - 1].labelId = this.sliceSerialize(S));
	}
	function we() {
		L(), x("ignoreEncode", !0);
	}
	function ye() {
		(u[u.length - 1].destination = d()), x("ignoreEncode");
	}
	function Ce() {
		u[u.length - 1].title = d();
	}
	function Ee() {
		const S = u[u.length - 1],
			z = G(S.labelId);
		d(), qn.call(r, z) || (r[z] = u[u.length - 1]), u.pop();
	}
	function Ie() {
		x("slurpAllLineEndings", !0);
	}
	function ze(S) {
		E("headingRank") ||
			(x("headingRank", this.sliceSerialize(S).length),
			T(),
			C("<h" + E("headingRank") + ">"));
	}
	function Fe() {
		L(), x("slurpAllLineEndings");
	}
	function Ae() {
		x("slurpAllLineEndings", !0);
	}
	function Le() {
		C("</h" + E("headingRank") + ">"), x("headingRank");
	}
	function Te(S) {
		x("headingRank", this.sliceSerialize(S).charCodeAt(0) === 61 ? 1 : 2);
	}
	function De() {
		const S = d();
		T(),
			C("<h" + E("headingRank") + ">"),
			y(S),
			C("</h" + E("headingRank") + ">"),
			x("slurpAllLineEndings"),
			x("headingRank");
	}
	function Et(S) {
		y(b(this.sliceSerialize(S)));
	}
	function _e(S) {
		if (!E("slurpAllLineEndings")) {
			if (E("slurpOneLineEnding")) {
				x("slurpOneLineEnding");
				return;
			}
			if (E("inCodeText")) {
				y(" ");
				return;
			}
			y(b(this.sliceSerialize(S)));
		}
	}
	function Oe(S) {
		y(b(this.sliceSerialize(S))), x("flowCodeSeenData", !0);
	}
	function An() {
		C("<br />");
	}
	function Pe() {
		T(), Tn();
	}
	function Ln() {
		x("ignoreEncode");
	}
	function Tn() {
		e.allowDangerousHtml && x("ignoreEncode", !0);
	}
	function Be() {
		C("<em>");
	}
	function Me() {
		C("<strong>");
	}
	function Re() {
		x("inCodeText", !0), C("<code>");
	}
	function Ne() {
		x("inCodeText"), C("</code>");
	}
	function He() {
		C("</em>");
	}
	function qe() {
		C("</strong>");
	}
	function Ve() {
		T(), C("<hr />");
	}
	function Dn(S) {
		x("characterReferenceType", S.type);
	}
	function je(S) {
		const z = this.sliceSerialize(S),
			$ = E("characterReferenceType")
				? Mn(
						z,
						E("characterReferenceType") === "characterReferenceMarkerNumeric"
							? 10
							: 16,
					)
				: Pt(z);
		y(b($)), x("characterReferenceType");
	}
	function We(S) {
		const z = this.sliceSerialize(S);
		C('<a href="' + rt(z, e.allowDangerousProtocol ? void 0 : Vn) + '">'),
			y(b(z)),
			C("</a>");
	}
	function Ue(S) {
		const z = this.sliceSerialize(S);
		C('<a href="' + rt("mailto:" + z) + '">'), y(b(z)), C("</a>");
	}
}
function A(t, e, n, r) {
	let l = r ? r - 1 : 1 / 0,
		u = 0;
	return i;
	function i(a) {
		return F(a) ? (t.enter(n), o(a)) : e(a);
	}
	function o(a) {
		return F(a) && u++ < l ? (t.consume(a), o) : (t.exit(n), e(a));
	}
}
function vt(t, e, n, r, l, u) {
	let i = 0;
	return o;
	function o(m) {
		return u > 0 && F(m) ? (t.enter(r), a(m)) : c(m);
	}
	function a(m) {
		return F(m) && i < u ? (t.consume(m), i++, a) : (t.exit(r), c(m));
	}
	function c(m) {
		return i >= l ? e(m) : n(m);
	}
}
var Wn = { tokenize: ve };
function ve(t) {
	let e = t.attempt(this.parser.constructs.contentInitial, r, l),
		n;
	return e;
	function r(o) {
		if (o === null) {
			t.consume(o);
			return;
		}
		return (
			t.enter("lineEnding"),
			t.consume(o),
			t.exit("lineEnding"),
			A(t, e, "linePrefix")
		);
	}
	function l(o) {
		return t.enter("paragraph"), u(o);
	}
	function u(o) {
		const a = t.enter("chunkText", { contentType: "text", previous: n });
		return n && (n.next = a), (n = a), i(o);
	}
	function i(o) {
		if (o === null) {
			t.exit("chunkText"), t.exit("paragraph"), t.consume(o);
			return;
		}
		return w(o) ? (t.consume(o), t.exit("chunkText"), u) : (t.consume(o), i);
	}
}
var Qn = { tokenize: tr },
	Un = { tokenize: nr };
function tr(t) {
	let e = this,
		n = [],
		r = 0,
		l,
		u,
		i;
	return o;
	function o(y) {
		if (r < n.length) {
			const B = n[r];
			return (e.containerState = B[1]), t.attempt(B[0].continuation, a, c)(y);
		}
		return c(y);
	}
	function a(y) {
		if ((r++, e.containerState._closeFlow)) {
			(e.containerState._closeFlow = void 0), l && C();
			let B = e.events.length,
				T = B,
				b;
			for (; T--; )
				if (e.events[T][0] === "exit" && e.events[T][1].type === "chunkFlow") {
					b = e.events[T][1].end;
					break;
				}
			d(r);
			let P = B;
			for (; P < e.events.length; ) (e.events[P][1].end = { ...b }), P++;
			return (
				V(e.events, T + 1, 0, e.events.slice(B)), (e.events.length = P), c(y)
			);
		}
		return o(y);
	}
	function c(y) {
		if (r === n.length) {
			if (!l) return g(y);
			if (l.currentConstruct && l.currentConstruct.concrete) return x(y);
			e.interrupt = !!(l.currentConstruct && !l._gfmTableDynamicInterruptHack);
		}
		return (e.containerState = {}), t.check(Un, m, s)(y);
	}
	function m(y) {
		return l && C(), d(r), g(y);
	}
	function s(y) {
		return (
			(e.parser.lazy[e.now().line] = r !== n.length), (i = e.now().offset), x(y)
		);
	}
	function g(y) {
		return (e.containerState = {}), t.attempt(Un, p, x)(y);
	}
	function p(y) {
		return r++, n.push([e.currentConstruct, e.containerState]), g(y);
	}
	function x(y) {
		if (y === null) {
			l && C(), d(0), t.consume(y);
			return;
		}
		return (
			(l = l || e.parser.flow(e.now())),
			t.enter("chunkFlow", { _tokenizer: l, contentType: "flow", previous: u }),
			E(y)
		);
	}
	function E(y) {
		if (y === null) {
			L(t.exit("chunkFlow"), !0), d(0), t.consume(y);
			return;
		}
		return w(y)
			? (t.consume(y),
				L(t.exit("chunkFlow")),
				(r = 0),
				(e.interrupt = void 0),
				o)
			: (t.consume(y), E);
	}
	function L(y, B) {
		const T = e.sliceStream(y);
		if (
			(B && T.push(null),
			(y.previous = u),
			u && (u.next = y),
			(u = y),
			l.defineSkip(y.start),
			l.write(T),
			e.parser.lazy[y.start.line])
		) {
			let b = l.events.length;
			for (; b--; )
				if (
					l.events[b][1].start.offset < i &&
					(!l.events[b][1].end || l.events[b][1].end.offset > i)
				)
					return;
			let P = e.events.length,
				R = P,
				N,
				k;
			for (; R--; )
				if (e.events[R][0] === "exit" && e.events[R][1].type === "chunkFlow") {
					if (N) {
						k = e.events[R][1].end;
						break;
					}
					N = !0;
				}
			for (d(r), b = P; b < e.events.length; )
				(e.events[b][1].end = { ...k }), b++;
			V(e.events, R + 1, 0, e.events.slice(P)), (e.events.length = b);
		}
	}
	function d(y) {
		let B = n.length;
		for (; B-- > y; ) {
			const T = n[B];
			(e.containerState = T[1]), T[0].exit.call(e, t);
		}
		n.length = y;
	}
	function C() {
		l.write([null]),
			(u = void 0),
			(l = void 0),
			(e.containerState._closeFlow = void 0);
	}
}
function nr(t, e, n) {
	return A(
		t,
		t.attempt(this.parser.constructs.document, e, n),
		"linePrefix",
		this.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 4,
	);
}
function wt(t) {
	if (t === null || O(t) || mt(t)) return 1;
	if (zt(t)) return 2;
}
function ht(t, e, n) {
	let r = [],
		l = -1;
	for (; ++l < t.length; ) {
		const u = t[l].resolveAll;
		u && !r.includes(u) && ((e = u(e, n)), r.push(u));
	}
	return e;
}
var Ft = { name: "attention", resolveAll: er, tokenize: rr };
function er(t, e) {
	let n = -1,
		r;
	for (; ++n < t.length; )
		if (
			t[n][0] === "enter" &&
			t[n][1].type === "attentionSequence" &&
			t[n][1]._close
		) {
			let l = n;
			for (; l--; )
				if (
					t[l][0] === "exit" &&
					t[l][1].type === "attentionSequence" &&
					t[l][1]._open &&
					e.sliceSerialize(t[l][1]).charCodeAt(0) ===
						e.sliceSerialize(t[n][1]).charCodeAt(0)
				) {
					if (
						(t[l][1]._close || t[n][1]._open) &&
						(t[n][1].end.offset - t[n][1].start.offset) % 3 &&
						!(
							(t[l][1].end.offset -
								t[l][1].start.offset +
								t[n][1].end.offset -
								t[n][1].start.offset) %
							3
						)
					)
						continue;
					const u =
							t[l][1].end.offset - t[l][1].start.offset > 1 &&
							t[n][1].end.offset - t[n][1].start.offset > 1
								? 2
								: 1,
						i = { ...t[l][1].end },
						o = { ...t[n][1].start };
					Yn(i, -u), Yn(o, u);
					const a = {
							type: u > 1 ? "strongSequence" : "emphasisSequence",
							start: i,
							end: { ...t[l][1].end },
						},
						c = {
							type: u > 1 ? "strongSequence" : "emphasisSequence",
							start: { ...t[n][1].start },
							end: o,
						},
						m = {
							type: u > 1 ? "strongText" : "emphasisText",
							start: { ...t[l][1].end },
							end: { ...t[n][1].start },
						},
						s = {
							type: u > 1 ? "strong" : "emphasis",
							start: { ...a.start },
							end: { ...c.end },
						};
					(t[l][1].end = { ...a.start }),
						(t[n][1].start = { ...c.end }),
						(r = []),
						t[l][1].end.offset - t[l][1].start.offset &&
							(r = j(r, [
								["enter", t[l][1], e],
								["exit", t[l][1], e],
							])),
						(r = j(r, [
							["enter", s, e],
							["enter", a, e],
							["exit", a, e],
							["enter", m, e],
						])),
						(r = j(
							r,
							ht(e.parser.constructs.insideSpan.null, t.slice(l + 1, n), e),
						)),
						(r = j(r, [
							["exit", m, e],
							["enter", c, e],
							["exit", c, e],
							["exit", s, e],
						]));
					let g = 0;
					t[n][1].end.offset - t[n][1].start.offset &&
						((g = 2),
						(r = j(r, [
							["enter", t[n][1], e],
							["exit", t[n][1], e],
						]))),
						V(t, l - 1, n - l + 3, r),
						(n = l + r.length - g - 2);
					break;
				}
		}
	for (n = -1; ++n < t.length; )
		t[n][1].type === "attentionSequence" && (t[n][1].type = "data");
	return t;
}
function rr(t, e) {
	let n = this.parser.constructs.attentionMarkers.null,
		r = this.previous,
		l = wt(r),
		u;
	return i;
	function i(a) {
		return (u = a), t.enter("attentionSequence"), o(a);
	}
	function o(a) {
		if (a === u) return t.consume(a), o;
		const c = t.exit("attentionSequence"),
			m = wt(a),
			s = !m || (m === 2 && l) || (n.includes(a) && a !== 42 && a !== 95),
			g = !l || (l === 2 && m) || (n.includes(r) && r !== 42 && r !== 95);
		return (
			(c._open = !!(u === 42 ? s : s && (l || !g))),
			(c._close = !!(u === 42 ? g : g && (m || !s))),
			e(a)
		);
	}
}
function Yn(t, e) {
	(t.column += e), (t.offset += e), (t._bufferIndex += e);
}
var tn = { name: "autolink", tokenize: ir };
function ir(t, e, n) {
	let r = 0;
	return l;
	function l(p) {
		return (
			t.enter("autolink"),
			t.enter("autolinkMarker"),
			t.consume(p),
			t.exit("autolinkMarker"),
			t.enter("autolinkProtocol"),
			u
		);
	}
	function u(p) {
		return U(p) ? (t.consume(p), i) : p === 64 ? n(p) : c(p);
	}
	function i(p) {
		return p === 43 || p === 45 || p === 46 || q(p) ? ((r = 1), o(p)) : c(p);
	}
	function o(p) {
		return p === 58
			? (t.consume(p), (r = 0), a)
			: (p === 43 || p === 45 || p === 46 || q(p)) && r++ < 32
				? (t.consume(p), o)
				: ((r = 0), c(p));
	}
	function a(p) {
		return p === 62
			? (t.exit("autolinkProtocol"),
				t.enter("autolinkMarker"),
				t.consume(p),
				t.exit("autolinkMarker"),
				t.exit("autolink"),
				e)
			: p === null || p === 32 || p === 60 || ft(p)
				? n(p)
				: (t.consume(p), a);
	}
	function c(p) {
		return p === 64 ? (t.consume(p), m) : Rn(p) ? (t.consume(p), c) : n(p);
	}
	function m(p) {
		return q(p) ? s(p) : n(p);
	}
	function s(p) {
		return p === 46
			? (t.consume(p), (r = 0), m)
			: p === 62
				? ((t.exit("autolinkProtocol").type = "autolinkEmail"),
					t.enter("autolinkMarker"),
					t.consume(p),
					t.exit("autolinkMarker"),
					t.exit("autolink"),
					e)
				: g(p);
	}
	function g(p) {
		if ((p === 45 || q(p)) && r++ < 63) {
			const x = p === 45 ? g : s;
			return t.consume(p), x;
		}
		return n(p);
	}
}
var it = { partial: !0, tokenize: ur };
function ur(t, e, n) {
	return r;
	function r(u) {
		return F(u) ? A(t, l, "linePrefix")(u) : l(u);
	}
	function l(u) {
		return u === null || w(u) ? e(u) : n(u);
	}
}
var Nt = {
	continuation: { tokenize: or },
	exit: ar,
	name: "blockQuote",
	tokenize: lr,
};
function lr(t, e, n) {
	const r = this;
	return l;
	function l(i) {
		if (i === 62) {
			const o = r.containerState;
			return (
				o.open || (t.enter("blockQuote", { _container: !0 }), (o.open = !0)),
				t.enter("blockQuotePrefix"),
				t.enter("blockQuoteMarker"),
				t.consume(i),
				t.exit("blockQuoteMarker"),
				u
			);
		}
		return n(i);
	}
	function u(i) {
		return F(i)
			? (t.enter("blockQuotePrefixWhitespace"),
				t.consume(i),
				t.exit("blockQuotePrefixWhitespace"),
				t.exit("blockQuotePrefix"),
				e)
			: (t.exit("blockQuotePrefix"), e(i));
	}
}
function or(t, e, n) {
	const r = this;
	return l;
	function l(i) {
		return F(i)
			? A(
					t,
					u,
					"linePrefix",
					r.parser.constructs.disable.null.includes("codeIndented")
						? void 0
						: 4,
				)(i)
			: u(i);
	}
	function u(i) {
		return t.attempt(Nt, e, n)(i);
	}
}
function ar(t) {
	t.exit("blockQuote");
}
var Ht = { name: "characterEscape", tokenize: sr };
function sr(t, e, n) {
	return r;
	function r(u) {
		return (
			t.enter("characterEscape"),
			t.enter("escapeMarker"),
			t.consume(u),
			t.exit("escapeMarker"),
			l
		);
	}
	function l(u) {
		return Hn(u)
			? (t.enter("characterEscapeValue"),
				t.consume(u),
				t.exit("characterEscapeValue"),
				t.exit("characterEscape"),
				e)
			: n(u);
	}
}
var qt = { name: "characterReference", tokenize: mr };
function mr(t, e, n) {
	let r = this,
		l = 0,
		u,
		i;
	return o;
	function o(s) {
		return (
			t.enter("characterReference"),
			t.enter("characterReferenceMarker"),
			t.consume(s),
			t.exit("characterReferenceMarker"),
			a
		);
	}
	function a(s) {
		return s === 35
			? (t.enter("characterReferenceMarkerNumeric"),
				t.consume(s),
				t.exit("characterReferenceMarkerNumeric"),
				c)
			: (t.enter("characterReferenceValue"), (u = 31), (i = q), m(s));
	}
	function c(s) {
		return s === 88 || s === 120
			? (t.enter("characterReferenceMarkerHexadecimal"),
				t.consume(s),
				t.exit("characterReferenceMarkerHexadecimal"),
				t.enter("characterReferenceValue"),
				(u = 6),
				(i = Nn),
				m)
			: (t.enter("characterReferenceValue"), (u = 7), (i = It), m(s));
	}
	function m(s) {
		if (s === 59 && l) {
			const g = t.exit("characterReferenceValue");
			return i === q && !Pt(r.sliceSerialize(g))
				? n(s)
				: (t.enter("characterReferenceMarker"),
					t.consume(s),
					t.exit("characterReferenceMarker"),
					t.exit("characterReference"),
					e);
		}
		return i(s) && l++ < u ? (t.consume(s), m) : n(s);
	}
}
var At = { partial: !0, tokenize: pr };
function pr(t, e, n) {
	const r = this;
	return l;
	function l(i) {
		return i === null
			? n(i)
			: (t.enter("lineEnding"), t.consume(i), t.exit("lineEnding"), u);
	}
	function u(i) {
		return r.parser.lazy[r.now().line] ? n(i) : e(i);
	}
}
var Vt = { concrete: !0, name: "codeFenced", tokenize: hr };
function hr(t, e, n) {
	let r = this,
		l = { partial: !0, tokenize: T },
		u = 0,
		i = 0,
		o;
	return a;
	function a(b) {
		return c(b);
	}
	function c(b) {
		const P = r.events[r.events.length - 1];
		return (
			(u =
				P && P[1].type === "linePrefix"
					? P[2].sliceSerialize(P[1], !0).length
					: 0),
			(o = b),
			t.enter("codeFenced"),
			t.enter("codeFencedFence"),
			t.enter("codeFencedFenceSequence"),
			m(b)
		);
	}
	function m(b) {
		return b === o
			? (i++, t.consume(b), m)
			: i < 3
				? n(b)
				: (t.exit("codeFencedFenceSequence"),
					F(b) ? A(t, s, "whitespace")(b) : s(b));
	}
	function s(b) {
		return b === null || w(b)
			? (t.exit("codeFencedFence"), r.interrupt ? e(b) : t.check(At, E, B)(b))
			: (t.enter("codeFencedFenceInfo"),
				t.enter("chunkString", { contentType: "string" }),
				g(b));
	}
	function g(b) {
		return b === null || w(b)
			? (t.exit("chunkString"), t.exit("codeFencedFenceInfo"), s(b))
			: F(b)
				? (t.exit("chunkString"),
					t.exit("codeFencedFenceInfo"),
					A(t, p, "whitespace")(b))
				: b === 96 && b === o
					? n(b)
					: (t.consume(b), g);
	}
	function p(b) {
		return b === null || w(b)
			? s(b)
			: (t.enter("codeFencedFenceMeta"),
				t.enter("chunkString", { contentType: "string" }),
				x(b));
	}
	function x(b) {
		return b === null || w(b)
			? (t.exit("chunkString"), t.exit("codeFencedFenceMeta"), s(b))
			: b === 96 && b === o
				? n(b)
				: (t.consume(b), x);
	}
	function E(b) {
		return t.attempt(l, B, L)(b);
	}
	function L(b) {
		return t.enter("lineEnding"), t.consume(b), t.exit("lineEnding"), d;
	}
	function d(b) {
		return u > 0 && F(b) ? A(t, C, "linePrefix", u + 1)(b) : C(b);
	}
	function C(b) {
		return b === null || w(b)
			? t.check(At, E, B)(b)
			: (t.enter("codeFlowValue"), y(b));
	}
	function y(b) {
		return b === null || w(b)
			? (t.exit("codeFlowValue"), C(b))
			: (t.consume(b), y);
	}
	function B(b) {
		return t.exit("codeFenced"), e(b);
	}
	function T(b, P, R) {
		let N = 0;
		return k;
		function k(_) {
			return b.enter("lineEnding"), b.consume(_), b.exit("lineEnding"), Q;
		}
		function Q(_) {
			return (
				b.enter("codeFencedFence"),
				F(_)
					? A(
							b,
							M,
							"linePrefix",
							r.parser.constructs.disable.null.includes("codeIndented")
								? void 0
								: 4,
						)(_)
					: M(_)
			);
		}
		function M(_) {
			return _ === o ? (b.enter("codeFencedFenceSequence"), I(_)) : R(_);
		}
		function I(_) {
			return _ === o
				? (N++, b.consume(_), I)
				: N >= i
					? (b.exit("codeFencedFenceSequence"),
						F(_) ? A(b, D, "whitespace")(_) : D(_))
					: R(_);
		}
		function D(_) {
			return _ === null || w(_) ? (b.exit("codeFencedFence"), P(_)) : R(_);
		}
	}
}
var Lt = { name: "codeIndented", tokenize: cr },
	fr = { partial: !0, tokenize: gr };
function cr(t, e, n) {
	return r;
	function r(o) {
		return t.enter("codeIndented"), vt(t, l, n, "linePrefix", 4, 4)(o);
	}
	function l(o) {
		return o === null
			? i(o)
			: w(o)
				? t.attempt(fr, l, i)(o)
				: (t.enter("codeFlowValue"), u(o));
	}
	function u(o) {
		return o === null || w(o)
			? (t.exit("codeFlowValue"), l(o))
			: (t.consume(o), u);
	}
	function i(o) {
		return t.exit("codeIndented"), e(o);
	}
}
function gr(t, e, n) {
	const r = this;
	return l;
	function l(i) {
		return r.parser.lazy[r.now().line]
			? n(i)
			: w(i)
				? (t.enter("lineEnding"), t.consume(i), t.exit("lineEnding"), l)
				: vt(t, e, u, "linePrefix", 4, 4)(i);
	}
	function u(i) {
		return w(i) ? l(i) : n(i);
	}
}
var nn = { name: "codeText", previous: kr, resolve: xr, tokenize: br };
function xr(t) {
	let e = t.length - 4,
		n = 3,
		r,
		l;
	if (
		(t[n][1].type === "lineEnding" || t[n][1].type === "space") &&
		(t[e][1].type === "lineEnding" || t[e][1].type === "space")
	) {
		for (r = n; ++r < e; )
			if (t[r][1].type === "codeTextData") {
				(t[n][1].type = "codeTextPadding"),
					(t[e][1].type = "codeTextPadding"),
					(n += 2),
					(e -= 2);
				break;
			}
	}
	for (r = n - 1, e++; ++r <= e; )
		l === void 0
			? r !== e && t[r][1].type !== "lineEnding" && (l = r)
			: (r === e || t[r][1].type === "lineEnding") &&
				((t[l][1].type = "codeTextData"),
				r !== l + 2 &&
					((t[l][1].end = t[r - 1][1].end),
					t.splice(l + 2, r - l - 2),
					(e -= r - l - 2),
					(r = l + 2)),
				(l = void 0));
	return t;
}
function kr(t) {
	return (
		t !== 96 ||
		this.events[this.events.length - 1][1].type === "characterEscape"
	);
}
function br(t, e, n) {
	let l = 0,
		u,
		i;
	return o;
	function o(g) {
		return t.enter("codeText"), t.enter("codeTextSequence"), a(g);
	}
	function a(g) {
		return g === 96
			? (t.consume(g), l++, a)
			: (t.exit("codeTextSequence"), c(g));
	}
	function c(g) {
		return g === null
			? n(g)
			: g === 32
				? (t.enter("space"), t.consume(g), t.exit("space"), c)
				: g === 96
					? ((i = t.enter("codeTextSequence")), (u = 0), s(g))
					: w(g)
						? (t.enter("lineEnding"), t.consume(g), t.exit("lineEnding"), c)
						: (t.enter("codeTextData"), m(g));
	}
	function m(g) {
		return g === null || g === 32 || g === 96 || w(g)
			? (t.exit("codeTextData"), c(g))
			: (t.consume(g), m);
	}
	function s(g) {
		return g === 96
			? (t.consume(g), u++, s)
			: u === l
				? (t.exit("codeTextSequence"), t.exit("codeText"), e(g))
				: ((i.type = "codeTextData"), m(g));
	}
}
var jt = class {
	constructor(e) {
		(this.left = e ? [...e] : []), (this.right = []);
	}
	get(e) {
		if (e < 0 || e >= this.left.length + this.right.length)
			throw new RangeError(
				"Cannot access index `" +
					e +
					"` in a splice buffer of size `" +
					(this.left.length + this.right.length) +
					"`",
			);
		return e < this.left.length
			? this.left[e]
			: this.right[this.right.length - e + this.left.length - 1];
	}
	get length() {
		return this.left.length + this.right.length;
	}
	shift() {
		return this.setCursor(0), this.right.pop();
	}
	slice(e, n) {
		const r = n ?? Number.POSITIVE_INFINITY;
		return r < this.left.length
			? this.left.slice(e, r)
			: e > this.left.length
				? this.right
						.slice(
							this.right.length - r + this.left.length,
							this.right.length - e + this.left.length,
						)
						.reverse()
				: this.left
						.slice(e)
						.concat(
							this.right
								.slice(this.right.length - r + this.left.length)
								.reverse(),
						);
	}
	splice(e, n, r) {
		const l = n || 0;
		this.setCursor(Math.trunc(e));
		const u = this.right.splice(
			this.right.length - l,
			Number.POSITIVE_INFINITY,
		);
		return r && Tt(this.left, r), u.reverse();
	}
	pop() {
		return this.setCursor(Number.POSITIVE_INFINITY), this.left.pop();
	}
	push(e) {
		this.setCursor(Number.POSITIVE_INFINITY), this.left.push(e);
	}
	pushMany(e) {
		this.setCursor(Number.POSITIVE_INFINITY), Tt(this.left, e);
	}
	unshift(e) {
		this.setCursor(0), this.right.push(e);
	}
	unshiftMany(e) {
		this.setCursor(0), Tt(this.right, e.reverse());
	}
	setCursor(e) {
		if (
			!(
				e === this.left.length ||
				(e > this.left.length && this.right.length === 0) ||
				(e < 0 && this.left.length === 0)
			)
		)
			if (e < this.left.length) {
				const n = this.left.splice(e, Number.POSITIVE_INFINITY);
				Tt(this.right, n.reverse());
			} else {
				const n = this.right.splice(
					this.left.length + this.right.length - e,
					Number.POSITIVE_INFINITY,
				);
				Tt(this.left, n.reverse());
			}
	}
};
function Tt(t, e) {
	let n = 0;
	if (e.length < 1e4) t.push(...e);
	else for (; n < e.length; ) t.push(...e.slice(n, n + 1e4)), (n += 1e4);
}
function Wt(t) {
	let e = {},
		n = -1,
		r,
		l,
		u,
		i,
		o,
		a,
		c,
		m = new jt(t);
	for (; ++n < m.length; ) {
		for (; n in e; ) n = e[n];
		if (
			((r = m.get(n)),
			n &&
				r[1].type === "chunkFlow" &&
				m.get(n - 1)[1].type === "listItemPrefix" &&
				((a = r[1]._tokenizer.events),
				(u = 0),
				u < a.length && a[u][1].type === "lineEndingBlank" && (u += 2),
				u < a.length && a[u][1].type === "content"))
		)
			for (; ++u < a.length && a[u][1].type !== "content"; )
				a[u][1].type === "chunkText" &&
					((a[u][1]._isInFirstContentOfListItem = !0), u++);
		if (r[0] === "enter")
			r[1].contentType && (Object.assign(e, dr(m, n)), (n = e[n]), (c = !0));
		else if (r[1]._container) {
			for (u = n, l = void 0; u--; )
				if (
					((i = m.get(u)),
					i[1].type === "lineEnding" || i[1].type === "lineEndingBlank")
				)
					i[0] === "enter" &&
						(l && (m.get(l)[1].type = "lineEndingBlank"),
						(i[1].type = "lineEnding"),
						(l = u));
				else if (
					!(i[1].type === "linePrefix" || i[1].type === "listItemIndent")
				)
					break;
			l &&
				((r[1].end = { ...m.get(l)[1].start }),
				(o = m.slice(l, n)),
				o.unshift(r),
				m.splice(l, n - l + 1, o));
		}
	}
	return V(t, 0, Number.POSITIVE_INFINITY, m.slice(0)), !c;
}
function dr(t, e) {
	let n = t.get(e)[1],
		r = t.get(e)[2],
		l = e - 1,
		u = [],
		i = n._tokenizer;
	i ||
		((i = r.parser[n.contentType](n.start)),
		n._contentTypeTextTrailing && (i._contentTypeTextTrailing = !0));
	let o = i.events,
		a = [],
		c = {},
		m,
		s,
		g = -1,
		p = n,
		x = 0,
		E = 0,
		L = [E];
	for (; p; ) {
		for (; t.get(++l)[1] !== p; );
		u.push(l),
			p._tokenizer ||
				((m = r.sliceStream(p)),
				p.next || m.push(null),
				s && i.defineSkip(p.start),
				p._isInFirstContentOfListItem &&
					(i._gfmTasklistFirstContentOfListItem = !0),
				i.write(m),
				p._isInFirstContentOfListItem &&
					(i._gfmTasklistFirstContentOfListItem = void 0)),
			(s = p),
			(p = p.next);
	}
	for (p = n; ++g < o.length; )
		o[g][0] === "exit" &&
			o[g - 1][0] === "enter" &&
			o[g][1].type === o[g - 1][1].type &&
			o[g][1].start.line !== o[g][1].end.line &&
			((E = g + 1),
			L.push(E),
			(p._tokenizer = void 0),
			(p.previous = void 0),
			(p = p.next));
	for (
		i.events = [],
			p ? ((p._tokenizer = void 0), (p.previous = void 0)) : L.pop(),
			g = L.length;
		g--;
	) {
		const d = o.slice(L[g], L[g + 1]),
			C = u.pop();
		a.push([C, C + d.length - 1]), t.splice(C, 2, d);
	}
	for (a.reverse(), g = -1; ++g < a.length; )
		(c[x + a[g][0]] = x + a[g][1]), (x += a[g][1] - a[g][0] - 1);
	return c;
}
var en = { resolve: wr, tokenize: yr },
	Sr = { partial: !0, tokenize: Cr };
function wr(t) {
	return Wt(t), t;
}
function yr(t, e) {
	let n;
	return r;
	function r(o) {
		return (
			t.enter("content"),
			(n = t.enter("chunkContent", { contentType: "content" })),
			l(o)
		);
	}
	function l(o) {
		return o === null ? u(o) : w(o) ? t.check(Sr, i, u)(o) : (t.consume(o), l);
	}
	function u(o) {
		return t.exit("chunkContent"), t.exit("content"), e(o);
	}
	function i(o) {
		return (
			t.consume(o),
			t.exit("chunkContent"),
			(n.next = t.enter("chunkContent", {
				contentType: "content",
				previous: n,
			})),
			(n = n.next),
			l
		);
	}
}
function Cr(t, e, n) {
	const r = this;
	return l;
	function l(i) {
		return (
			t.exit("chunkContent"),
			t.enter("lineEnding"),
			t.consume(i),
			t.exit("lineEnding"),
			A(t, u, "linePrefix")
		);
	}
	function u(i) {
		if (i === null || w(i)) return n(i);
		const o = r.events[r.events.length - 1];
		return !r.parser.constructs.disable.null.includes("codeIndented") &&
			o &&
			o[1].type === "linePrefix" &&
			o[2].sliceSerialize(o[1], !0).length >= 4
			? e(i)
			: t.interrupt(r.parser.constructs.flow, n, e)(i);
	}
}
function Ut(t, e, n, r, l, u, i, o, a) {
	let c = a || Number.POSITIVE_INFINITY,
		m = 0;
	return s;
	function s(d) {
		return d === 60
			? (t.enter(r), t.enter(l), t.enter(u), t.consume(d), t.exit(u), g)
			: d === null || d === 32 || d === 41 || ft(d)
				? n(d)
				: (t.enter(r),
					t.enter(i),
					t.enter(o),
					t.enter("chunkString", { contentType: "string" }),
					E(d));
	}
	function g(d) {
		return d === 62
			? (t.enter(u), t.consume(d), t.exit(u), t.exit(l), t.exit(r), e)
			: (t.enter(o), t.enter("chunkString", { contentType: "string" }), p(d));
	}
	function p(d) {
		return d === 62
			? (t.exit("chunkString"), t.exit(o), g(d))
			: d === null || d === 60 || w(d)
				? n(d)
				: (t.consume(d), d === 92 ? x : p);
	}
	function x(d) {
		return d === 60 || d === 62 || d === 92 ? (t.consume(d), p) : p(d);
	}
	function E(d) {
		return !m && (d === null || d === 41 || O(d))
			? (t.exit("chunkString"), t.exit(o), t.exit(i), t.exit(r), e(d))
			: m < c && d === 40
				? (t.consume(d), m++, E)
				: d === 41
					? (t.consume(d), m--, E)
					: d === null || d === 32 || d === 40 || ft(d)
						? n(d)
						: (t.consume(d), d === 92 ? L : E);
	}
	function L(d) {
		return d === 40 || d === 41 || d === 92 ? (t.consume(d), E) : E(d);
	}
}
function Qt(t, e, n, r, l, u) {
	let i = this,
		o = 0,
		a;
	return c;
	function c(p) {
		return t.enter(r), t.enter(l), t.consume(p), t.exit(l), t.enter(u), m;
	}
	function m(p) {
		return o > 999 ||
			p === null ||
			p === 91 ||
			(p === 93 && !a) ||
			(p === 94 && !o && "_hiddenFootnoteSupport" in i.parser.constructs)
			? n(p)
			: p === 93
				? (t.exit(u), t.enter(l), t.consume(p), t.exit(l), t.exit(r), e)
				: w(p)
					? (t.enter("lineEnding"), t.consume(p), t.exit("lineEnding"), m)
					: (t.enter("chunkString", { contentType: "string" }), s(p));
	}
	function s(p) {
		return p === null || p === 91 || p === 93 || w(p) || o++ > 999
			? (t.exit("chunkString"), m(p))
			: (t.consume(p), a || (a = !F(p)), p === 92 ? g : s);
	}
	function g(p) {
		return p === 91 || p === 92 || p === 93 ? (t.consume(p), o++, s) : s(p);
	}
}
function Yt(t, e, n, r, l, u) {
	let i;
	return o;
	function o(g) {
		return g === 34 || g === 39 || g === 40
			? (t.enter(r),
				t.enter(l),
				t.consume(g),
				t.exit(l),
				(i = g === 40 ? 41 : g),
				a)
			: n(g);
	}
	function a(g) {
		return g === i
			? (t.enter(l), t.consume(g), t.exit(l), t.exit(r), e)
			: (t.enter(u), c(g));
	}
	function c(g) {
		return g === i
			? (t.exit(u), a(i))
			: g === null
				? n(g)
				: w(g)
					? (t.enter("lineEnding"),
						t.consume(g),
						t.exit("lineEnding"),
						A(t, c, "linePrefix"))
					: (t.enter("chunkString", { contentType: "string" }), m(g));
	}
	function m(g) {
		return g === i || g === null || w(g)
			? (t.exit("chunkString"), c(g))
			: (t.consume(g), g === 92 ? s : m);
	}
	function s(g) {
		return g === i || g === 92 ? (t.consume(g), m) : m(g);
	}
}
function ct(t, e) {
	let n;
	return r;
	function r(l) {
		return w(l)
			? (t.enter("lineEnding"), t.consume(l), t.exit("lineEnding"), (n = !0), r)
			: F(l)
				? A(t, r, n ? "linePrefix" : "lineSuffix")(l)
				: e(l);
	}
}
var rn = { name: "definition", tokenize: Ir },
	Er = { partial: !0, tokenize: zr };
function Ir(t, e, n) {
	let r = this,
		l;
	return u;
	function u(p) {
		return t.enter("definition"), i(p);
	}
	function i(p) {
		return Qt.call(
			r,
			t,
			o,
			n,
			"definitionLabel",
			"definitionLabelMarker",
			"definitionLabelString",
		)(p);
	}
	function o(p) {
		return (
			(l = G(r.sliceSerialize(r.events[r.events.length - 1][1]).slice(1, -1))),
			p === 58
				? (t.enter("definitionMarker"),
					t.consume(p),
					t.exit("definitionMarker"),
					a)
				: n(p)
		);
	}
	function a(p) {
		return O(p) ? ct(t, c)(p) : c(p);
	}
	function c(p) {
		return Ut(
			t,
			m,
			n,
			"definitionDestination",
			"definitionDestinationLiteral",
			"definitionDestinationLiteralMarker",
			"definitionDestinationRaw",
			"definitionDestinationString",
		)(p);
	}
	function m(p) {
		return t.attempt(Er, s, s)(p);
	}
	function s(p) {
		return F(p) ? A(t, g, "whitespace")(p) : g(p);
	}
	function g(p) {
		return p === null || w(p)
			? (t.exit("definition"), r.parser.defined.push(l), e(p))
			: n(p);
	}
}
function zr(t, e, n) {
	return r;
	function r(o) {
		return O(o) ? ct(t, l)(o) : n(o);
	}
	function l(o) {
		return Yt(
			t,
			u,
			n,
			"definitionTitle",
			"definitionTitleMarker",
			"definitionTitleString",
		)(o);
	}
	function u(o) {
		return F(o) ? A(t, i, "whitespace")(o) : i(o);
	}
	function i(o) {
		return o === null || w(o) ? e(o) : n(o);
	}
}
var un = { name: "hardBreakEscape", tokenize: Fr };
function Fr(t, e, n) {
	return r;
	function r(u) {
		return t.enter("hardBreakEscape"), t.consume(u), l;
	}
	function l(u) {
		return w(u) ? (t.exit("hardBreakEscape"), e(u)) : n(u);
	}
}
var ln = { name: "headingAtx", resolve: Ar, tokenize: Lr };
function Ar(t, e) {
	let n = t.length - 2,
		r = 3;
	if (
		(t[r][1].type === "whitespace" && (r += 2),
		n - 2 > r && t[n][1].type === "whitespace" && (n -= 2),
		t[n][1].type === "atxHeadingSequence" &&
			(r === n - 1 || (n - 4 > r && t[n - 2][1].type === "whitespace")) &&
			(n -= r + 1 === n ? 2 : 4),
		n > r)
	) {
		const l = {
				type: "atxHeadingText",
				start: t[r][1].start,
				end: t[n][1].end,
			},
			u = {
				type: "chunkText",
				start: t[r][1].start,
				end: t[n][1].end,
				contentType: "text",
			};
		V(t, r, n - r + 1, [
			["enter", l, e],
			["enter", u, e],
			["exit", u, e],
			["exit", l, e],
		]);
	}
	return t;
}
function Lr(t, e, n) {
	let r = 0;
	return l;
	function l(m) {
		return t.enter("atxHeading"), u(m);
	}
	function u(m) {
		return t.enter("atxHeadingSequence"), i(m);
	}
	function i(m) {
		return m === 35 && r++ < 6
			? (t.consume(m), i)
			: m === null || O(m)
				? (t.exit("atxHeadingSequence"), o(m))
				: n(m);
	}
	function o(m) {
		return m === 35
			? (t.enter("atxHeadingSequence"), a(m))
			: m === null || w(m)
				? (t.exit("atxHeading"), e(m))
				: F(m)
					? A(t, o, "whitespace")(m)
					: (t.enter("atxHeadingText"), c(m));
	}
	function a(m) {
		return m === 35 ? (t.consume(m), a) : (t.exit("atxHeadingSequence"), o(m));
	}
	function c(m) {
		return m === null || m === 35 || O(m)
			? (t.exit("atxHeadingText"), o(m))
			: (t.consume(m), c);
	}
}
var $n = [
		"address",
		"article",
		"aside",
		"base",
		"basefont",
		"blockquote",
		"body",
		"caption",
		"center",
		"col",
		"colgroup",
		"dd",
		"details",
		"dialog",
		"dir",
		"div",
		"dl",
		"dt",
		"fieldset",
		"figcaption",
		"figure",
		"footer",
		"form",
		"frame",
		"frameset",
		"h1",
		"h2",
		"h3",
		"h4",
		"h5",
		"h6",
		"head",
		"header",
		"hr",
		"html",
		"iframe",
		"legend",
		"li",
		"link",
		"main",
		"menu",
		"menuitem",
		"nav",
		"noframes",
		"ol",
		"optgroup",
		"option",
		"p",
		"param",
		"search",
		"section",
		"summary",
		"table",
		"tbody",
		"td",
		"tfoot",
		"th",
		"thead",
		"title",
		"tr",
		"track",
		"ul",
	],
	on = ["pre", "script", "style", "textarea"];
var an = { concrete: !0, name: "htmlFlow", resolveTo: Dr, tokenize: _r },
	Tr = { partial: !0, tokenize: Or };
function Dr(t) {
	let e = t.length;
	for (; e-- && !(t[e][0] === "enter" && t[e][1].type === "htmlFlow"); );
	return (
		e > 1 &&
			t[e - 2][1].type === "linePrefix" &&
			((t[e][1].start = t[e - 2][1].start),
			(t[e + 1][1].start = t[e - 2][1].start),
			t.splice(e - 2, 2)),
		t
	);
}
function _r(t, e, n) {
	let r = this,
		l,
		u,
		i,
		o,
		a;
	return c;
	function c(f) {
		return m(f);
	}
	function m(f) {
		return t.enter("htmlFlow"), t.enter("htmlFlowData"), t.consume(f), s;
	}
	function s(f) {
		return f === 33
			? (t.consume(f), g)
			: f === 47
				? (t.consume(f), (u = !0), E)
				: f === 63
					? (t.consume(f), (l = 3), r.interrupt ? e : h)
					: U(f)
						? (t.consume(f), (i = String.fromCharCode(f)), L)
						: n(f);
	}
	function g(f) {
		return f === 45
			? (t.consume(f), (l = 2), p)
			: f === 91
				? (t.consume(f), (l = 5), (o = 0), x)
				: U(f)
					? (t.consume(f), (l = 4), r.interrupt ? e : h)
					: n(f);
	}
	function p(f) {
		return f === 45 ? (t.consume(f), r.interrupt ? e : h) : n(f);
	}
	function x(f) {
		const J = "CDATA[";
		return f === J.charCodeAt(o++)
			? (t.consume(f), o === J.length ? (r.interrupt ? e : M) : x)
			: n(f);
	}
	function E(f) {
		return U(f) ? (t.consume(f), (i = String.fromCharCode(f)), L) : n(f);
	}
	function L(f) {
		if (f === null || f === 47 || f === 62 || O(f)) {
			const J = f === 47,
				bt = i.toLowerCase();
			return !J && !u && on.includes(bt)
				? ((l = 1), r.interrupt ? e(f) : M(f))
				: $n.includes(i.toLowerCase())
					? ((l = 6), J ? (t.consume(f), d) : r.interrupt ? e(f) : M(f))
					: ((l = 7),
						r.interrupt && !r.parser.lazy[r.now().line]
							? n(f)
							: u
								? C(f)
								: y(f));
		}
		return f === 45 || q(f)
			? (t.consume(f), (i += String.fromCharCode(f)), L)
			: n(f);
	}
	function d(f) {
		return f === 62 ? (t.consume(f), r.interrupt ? e : M) : n(f);
	}
	function C(f) {
		return F(f) ? (t.consume(f), C) : k(f);
	}
	function y(f) {
		return f === 47
			? (t.consume(f), k)
			: f === 58 || f === 95 || U(f)
				? (t.consume(f), B)
				: F(f)
					? (t.consume(f), y)
					: k(f);
	}
	function B(f) {
		return f === 45 || f === 46 || f === 58 || f === 95 || q(f)
			? (t.consume(f), B)
			: T(f);
	}
	function T(f) {
		return f === 61 ? (t.consume(f), b) : F(f) ? (t.consume(f), T) : y(f);
	}
	function b(f) {
		return f === null || f === 60 || f === 61 || f === 62 || f === 96
			? n(f)
			: f === 34 || f === 39
				? (t.consume(f), (a = f), P)
				: F(f)
					? (t.consume(f), b)
					: R(f);
	}
	function P(f) {
		return f === a
			? (t.consume(f), (a = null), N)
			: f === null || w(f)
				? n(f)
				: (t.consume(f), P);
	}
	function R(f) {
		return f === null ||
			f === 34 ||
			f === 39 ||
			f === 47 ||
			f === 60 ||
			f === 61 ||
			f === 62 ||
			f === 96 ||
			O(f)
			? T(f)
			: (t.consume(f), R);
	}
	function N(f) {
		return f === 47 || f === 62 || F(f) ? y(f) : n(f);
	}
	function k(f) {
		return f === 62 ? (t.consume(f), Q) : n(f);
	}
	function Q(f) {
		return f === null || w(f) ? M(f) : F(f) ? (t.consume(f), Q) : n(f);
	}
	function M(f) {
		return f === 45 && l === 2
			? (t.consume(f), H)
			: f === 60 && l === 1
				? (t.consume(f), W)
				: f === 62 && l === 4
					? (t.consume(f), X)
					: f === 63 && l === 3
						? (t.consume(f), h)
						: f === 93 && l === 5
							? (t.consume(f), tt)
							: w(f) && (l === 6 || l === 7)
								? (t.exit("htmlFlowData"), t.check(Tr, lt, I)(f))
								: f === null || w(f)
									? (t.exit("htmlFlowData"), I(f))
									: (t.consume(f), M);
	}
	function I(f) {
		return t.check(At, D, lt)(f);
	}
	function D(f) {
		return t.enter("lineEnding"), t.consume(f), t.exit("lineEnding"), _;
	}
	function _(f) {
		return f === null || w(f) ? I(f) : (t.enter("htmlFlowData"), M(f));
	}
	function H(f) {
		return f === 45 ? (t.consume(f), h) : M(f);
	}
	function W(f) {
		return f === 47 ? (t.consume(f), (i = ""), K) : M(f);
	}
	function K(f) {
		if (f === 62) {
			const J = i.toLowerCase();
			return on.includes(J) ? (t.consume(f), X) : M(f);
		}
		return U(f) && i.length < 8
			? (t.consume(f), (i += String.fromCharCode(f)), K)
			: M(f);
	}
	function tt(f) {
		return f === 93 ? (t.consume(f), h) : M(f);
	}
	function h(f) {
		return f === 62
			? (t.consume(f), X)
			: f === 45 && l === 2
				? (t.consume(f), h)
				: M(f);
	}
	function X(f) {
		return f === null || w(f)
			? (t.exit("htmlFlowData"), lt(f))
			: (t.consume(f), X);
	}
	function lt(f) {
		return t.exit("htmlFlow"), e(f);
	}
}
function Or(t, e, n) {
	return r;
	function r(l) {
		return (
			t.enter("lineEnding"),
			t.consume(l),
			t.exit("lineEnding"),
			t.attempt(it, e, n)
		);
	}
}
var sn = { name: "htmlText", tokenize: Pr };
function Pr(t, e, n) {
	let r = this,
		l,
		u,
		i;
	return o;
	function o(h) {
		return t.enter("htmlText"), t.enter("htmlTextData"), t.consume(h), a;
	}
	function a(h) {
		return h === 33
			? (t.consume(h), c)
			: h === 47
				? (t.consume(h), T)
				: h === 63
					? (t.consume(h), y)
					: U(h)
						? (t.consume(h), R)
						: n(h);
	}
	function c(h) {
		return h === 45
			? (t.consume(h), m)
			: h === 91
				? (t.consume(h), (u = 0), x)
				: U(h)
					? (t.consume(h), C)
					: n(h);
	}
	function m(h) {
		return h === 45 ? (t.consume(h), p) : n(h);
	}
	function s(h) {
		return h === null
			? n(h)
			: h === 45
				? (t.consume(h), g)
				: w(h)
					? ((i = s), W(h))
					: (t.consume(h), s);
	}
	function g(h) {
		return h === 45 ? (t.consume(h), p) : s(h);
	}
	function p(h) {
		return h === 62 ? H(h) : h === 45 ? g(h) : s(h);
	}
	function x(h) {
		const X = "CDATA[";
		return h === X.charCodeAt(u++)
			? (t.consume(h), u === X.length ? E : x)
			: n(h);
	}
	function E(h) {
		return h === null
			? n(h)
			: h === 93
				? (t.consume(h), L)
				: w(h)
					? ((i = E), W(h))
					: (t.consume(h), E);
	}
	function L(h) {
		return h === 93 ? (t.consume(h), d) : E(h);
	}
	function d(h) {
		return h === 62 ? H(h) : h === 93 ? (t.consume(h), d) : E(h);
	}
	function C(h) {
		return h === null || h === 62
			? H(h)
			: w(h)
				? ((i = C), W(h))
				: (t.consume(h), C);
	}
	function y(h) {
		return h === null
			? n(h)
			: h === 63
				? (t.consume(h), B)
				: w(h)
					? ((i = y), W(h))
					: (t.consume(h), y);
	}
	function B(h) {
		return h === 62 ? H(h) : y(h);
	}
	function T(h) {
		return U(h) ? (t.consume(h), b) : n(h);
	}
	function b(h) {
		return h === 45 || q(h) ? (t.consume(h), b) : P(h);
	}
	function P(h) {
		return w(h) ? ((i = P), W(h)) : F(h) ? (t.consume(h), P) : H(h);
	}
	function R(h) {
		return h === 45 || q(h)
			? (t.consume(h), R)
			: h === 47 || h === 62 || O(h)
				? N(h)
				: n(h);
	}
	function N(h) {
		return h === 47
			? (t.consume(h), H)
			: h === 58 || h === 95 || U(h)
				? (t.consume(h), k)
				: w(h)
					? ((i = N), W(h))
					: F(h)
						? (t.consume(h), N)
						: H(h);
	}
	function k(h) {
		return h === 45 || h === 46 || h === 58 || h === 95 || q(h)
			? (t.consume(h), k)
			: Q(h);
	}
	function Q(h) {
		return h === 61
			? (t.consume(h), M)
			: w(h)
				? ((i = Q), W(h))
				: F(h)
					? (t.consume(h), Q)
					: N(h);
	}
	function M(h) {
		return h === null || h === 60 || h === 61 || h === 62 || h === 96
			? n(h)
			: h === 34 || h === 39
				? (t.consume(h), (l = h), I)
				: w(h)
					? ((i = M), W(h))
					: F(h)
						? (t.consume(h), M)
						: (t.consume(h), D);
	}
	function I(h) {
		return h === l
			? (t.consume(h), (l = void 0), _)
			: h === null
				? n(h)
				: w(h)
					? ((i = I), W(h))
					: (t.consume(h), I);
	}
	function D(h) {
		return h === null ||
			h === 34 ||
			h === 39 ||
			h === 60 ||
			h === 61 ||
			h === 96
			? n(h)
			: h === 47 || h === 62 || O(h)
				? N(h)
				: (t.consume(h), D);
	}
	function _(h) {
		return h === 47 || h === 62 || O(h) ? N(h) : n(h);
	}
	function H(h) {
		return h === 62
			? (t.consume(h), t.exit("htmlTextData"), t.exit("htmlText"), e)
			: n(h);
	}
	function W(h) {
		return (
			t.exit("htmlTextData"),
			t.enter("lineEnding"),
			t.consume(h),
			t.exit("lineEnding"),
			K
		);
	}
	function K(h) {
		return F(h)
			? A(
					t,
					tt,
					"linePrefix",
					r.parser.constructs.disable.null.includes("codeIndented")
						? void 0
						: 4,
				)(h)
			: tt(h);
	}
	function tt(h) {
		return t.enter("htmlTextData"), i(h);
	}
}
var gt = { name: "labelEnd", resolveAll: Nr, resolveTo: Hr, tokenize: qr },
	Br = { tokenize: Vr },
	Mr = { tokenize: jr },
	Rr = { tokenize: Wr };
function Nr(t) {
	let e = -1,
		n = [];
	for (; ++e < t.length; ) {
		const r = t[e][1];
		if (
			(n.push(t[e]),
			r.type === "labelImage" ||
				r.type === "labelLink" ||
				r.type === "labelEnd")
		) {
			const l = r.type === "labelImage" ? 4 : 2;
			(r.type = "data"), (e += l);
		}
	}
	return t.length !== n.length && V(t, 0, t.length, n), t;
}
function Hr(t, e) {
	let n = t.length,
		r = 0,
		l,
		u,
		i;
	for (; n--; ) {
		const m = t[n][1];
		if (l) {
			if (m.type === "link" || (m.type === "labelLink" && m._inactive)) break;
			t[n][0] === "enter" && m.type === "labelLink" && (m._inactive = !0);
		} else if (u) {
			if (
				t[n][0] === "enter" &&
				(m.type === "labelImage" || m.type === "labelLink") &&
				!m._balanced &&
				((l = n), m.type !== "labelLink")
			) {
				r = 2;
				break;
			}
		} else m.type === "labelEnd" && (u = n);
	}
	const o = {
			type: t[l][1].type === "labelLink" ? "link" : "image",
			start: { ...t[l][1].start },
			end: { ...t[t.length - 1][1].end },
		},
		a = { type: "label", start: { ...t[l][1].start }, end: { ...t[u][1].end } },
		c = {
			type: "labelText",
			start: { ...t[l + r + 2][1].end },
			end: { ...t[u - 2][1].start },
		};
	return (
		(i = [
			["enter", o, e],
			["enter", a, e],
		]),
		(i = j(i, t.slice(l + 1, l + r + 3))),
		(i = j(i, [["enter", c, e]])),
		(i = j(
			i,
			ht(e.parser.constructs.insideSpan.null, t.slice(l + r + 4, u - 3), e),
		)),
		(i = j(i, [["exit", c, e], t[u - 2], t[u - 1], ["exit", a, e]])),
		(i = j(i, t.slice(u + 1))),
		(i = j(i, [["exit", o, e]])),
		V(t, l, t.length, i),
		t
	);
}
function qr(t, e, n) {
	let r = this,
		l = r._labelStarts,
		u,
		i;
	if (l) {
		for (; l.length > 0 && l[l.length - 1]._balanced; ) l.pop();
		u = l[l.length - 1];
	}
	return o;
	function o(g) {
		return u
			? u._inactive
				? s(g)
				: ((i = r.parser.defined.includes(
						G(r.sliceSerialize({ start: u.end, end: r.now() })),
					)),
					t.enter("labelEnd"),
					t.enter("labelMarker"),
					t.consume(g),
					t.exit("labelMarker"),
					t.exit("labelEnd"),
					a)
			: n(g);
	}
	function a(g) {
		return g === 40
			? t.attempt(Br, m, i ? m : s)(g)
			: g === 91
				? t.attempt(Mr, m, i ? c : s)(g)
				: i
					? m(g)
					: s(g);
	}
	function c(g) {
		return t.attempt(Rr, m, s)(g);
	}
	function m(g) {
		return l.pop(), e(g);
	}
	function s(g) {
		return (u._balanced = !0), n(g);
	}
}
function Vr(t, e, n) {
	return r;
	function r(s) {
		return (
			t.enter("resource"),
			t.enter("resourceMarker"),
			t.consume(s),
			t.exit("resourceMarker"),
			l
		);
	}
	function l(s) {
		return O(s) ? ct(t, u)(s) : u(s);
	}
	function u(s) {
		return s === 41
			? m(s)
			: Ut(
					t,
					i,
					o,
					"resourceDestination",
					"resourceDestinationLiteral",
					"resourceDestinationLiteralMarker",
					"resourceDestinationRaw",
					"resourceDestinationString",
					32,
				)(s);
	}
	function i(s) {
		return O(s) ? ct(t, a)(s) : m(s);
	}
	function o(s) {
		return n(s);
	}
	function a(s) {
		return s === 34 || s === 39 || s === 40
			? Yt(
					t,
					c,
					n,
					"resourceTitle",
					"resourceTitleMarker",
					"resourceTitleString",
				)(s)
			: m(s);
	}
	function c(s) {
		return O(s) ? ct(t, m)(s) : m(s);
	}
	function m(s) {
		return s === 41
			? (t.enter("resourceMarker"),
				t.consume(s),
				t.exit("resourceMarker"),
				t.exit("resource"),
				e)
			: n(s);
	}
}
function jr(t, e, n) {
	const r = this;
	return l;
	function l(o) {
		return Qt.call(
			r,
			t,
			u,
			i,
			"reference",
			"referenceMarker",
			"referenceString",
		)(o);
	}
	function u(o) {
		return r.parser.defined.includes(
			G(r.sliceSerialize(r.events[r.events.length - 1][1]).slice(1, -1)),
		)
			? e(o)
			: n(o);
	}
	function i(o) {
		return n(o);
	}
}
function Wr(t, e, n) {
	return r;
	function r(u) {
		return (
			t.enter("reference"),
			t.enter("referenceMarker"),
			t.consume(u),
			t.exit("referenceMarker"),
			l
		);
	}
	function l(u) {
		return u === 93
			? (t.enter("referenceMarker"),
				t.consume(u),
				t.exit("referenceMarker"),
				t.exit("reference"),
				e)
			: n(u);
	}
}
var mn = { name: "labelStartImage", resolveAll: gt.resolveAll, tokenize: Ur };
function Ur(t, e, n) {
	let r = this,
		l;
	return u;
	function u(a) {
		return (
			t.enter("labelImage"),
			t.enter("labelImageMarker"),
			t.consume(a),
			t.exit("labelImageMarker"),
			i
		);
	}
	function i(a) {
		return a === 91
			? (t.enter("labelMarker"),
				t.consume(a),
				t.exit("labelMarker"),
				(l = t.exit("labelImage")),
				o)
			: n(a);
	}
	function o(a) {
		return a === 94 && "_hiddenFootnoteSupport" in r.parser.constructs
			? n(a)
			: ((r._labelStarts = r._labelStarts || []), r._labelStarts.push(l), e(a));
	}
}
var pn = { name: "labelStartLink", resolveAll: gt.resolveAll, tokenize: Qr };
function Qr(t, e, n) {
	let r = this,
		l;
	return u;
	function u(o) {
		return (
			t.enter("labelLink"),
			t.enter("labelMarker"),
			t.consume(o),
			t.exit("labelMarker"),
			(l = t.exit("labelLink")),
			i
		);
	}
	function i(o) {
		return o === 94 && "_hiddenFootnoteSupport" in r.parser.constructs
			? n(o)
			: ((r._labelStarts = r._labelStarts || []), r._labelStarts.push(l), e(o));
	}
}
var Dt = { name: "lineEnding", tokenize: Yr };
function Yr(t, e) {
	return n;
	function n(r) {
		return (
			t.enter("lineEnding"),
			t.consume(r),
			t.exit("lineEnding"),
			A(t, e, "linePrefix")
		);
	}
}
var xt = { name: "thematicBreak", tokenize: $r };
function $r(t, e, n) {
	let r = 0,
		l;
	return u;
	function u(c) {
		return t.enter("thematicBreak"), i(c);
	}
	function i(c) {
		return (l = c), o(c);
	}
	function o(c) {
		return c === l
			? (t.enter("thematicBreakSequence"), a(c))
			: r >= 3 && (c === null || w(c))
				? (t.exit("thematicBreak"), e(c))
				: n(c);
	}
	function a(c) {
		return c === l
			? (t.consume(c), r++, a)
			: (t.exit("thematicBreakSequence"),
				F(c) ? A(t, o, "whitespace")(c) : o(c));
	}
}
var Z = {
		continuation: { tokenize: Xr },
		exit: vr,
		name: "list",
		tokenize: Jr,
	},
	Zr = { partial: !0, tokenize: ti },
	Gr = { partial: !0, tokenize: Kr };
function Jr(t, e, n) {
	let r = this,
		l = r.events[r.events.length - 1],
		u =
			l && l[1].type === "linePrefix"
				? l[2].sliceSerialize(l[1], !0).length
				: 0,
		i = 0;
	return o;
	function o(p) {
		const x =
			r.containerState.type ||
			(p === 42 || p === 43 || p === 45 ? "listUnordered" : "listOrdered");
		if (
			x === "listUnordered"
				? !r.containerState.marker || p === r.containerState.marker
				: It(p)
		) {
			if (
				(r.containerState.type ||
					((r.containerState.type = x), t.enter(x, { _container: !0 })),
				x === "listUnordered")
			)
				return (
					t.enter("listItemPrefix"),
					p === 42 || p === 45 ? t.check(xt, n, c)(p) : c(p)
				);
			if (!r.interrupt || p === 49)
				return t.enter("listItemPrefix"), t.enter("listItemValue"), a(p);
		}
		return n(p);
	}
	function a(p) {
		return It(p) && ++i < 10
			? (t.consume(p), a)
			: (!r.interrupt || i < 2) &&
					(r.containerState.marker
						? p === r.containerState.marker
						: p === 41 || p === 46)
				? (t.exit("listItemValue"), c(p))
				: n(p);
	}
	function c(p) {
		return (
			t.enter("listItemMarker"),
			t.consume(p),
			t.exit("listItemMarker"),
			(r.containerState.marker = r.containerState.marker || p),
			t.check(it, r.interrupt ? n : m, t.attempt(Zr, g, s))
		);
	}
	function m(p) {
		return (r.containerState.initialBlankLine = !0), u++, g(p);
	}
	function s(p) {
		return F(p)
			? (t.enter("listItemPrefixWhitespace"),
				t.consume(p),
				t.exit("listItemPrefixWhitespace"),
				g)
			: n(p);
	}
	function g(p) {
		return (
			(r.containerState.size =
				u + r.sliceSerialize(t.exit("listItemPrefix"), !0).length),
			e(p)
		);
	}
}
function Xr(t, e, n) {
	const r = this;
	return (r.containerState._closeFlow = void 0), t.check(it, l, u);
	function l(o) {
		return (
			(r.containerState.furtherBlankLines =
				r.containerState.furtherBlankLines ||
				r.containerState.initialBlankLine),
			A(t, e, "listItemIndent", r.containerState.size + 1)(o)
		);
	}
	function u(o) {
		return r.containerState.furtherBlankLines || !F(o)
			? ((r.containerState.furtherBlankLines = void 0),
				(r.containerState.initialBlankLine = void 0),
				i(o))
			: ((r.containerState.furtherBlankLines = void 0),
				(r.containerState.initialBlankLine = void 0),
				t.attempt(Gr, e, i)(o));
	}
	function i(o) {
		return (
			(r.containerState._closeFlow = !0),
			(r.interrupt = void 0),
			A(
				t,
				t.attempt(Z, e, n),
				"linePrefix",
				r.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 4,
			)(o)
		);
	}
}
function Kr(t, e, n) {
	const r = this;
	return A(t, l, "listItemIndent", r.containerState.size + 1);
	function l(u) {
		const i = r.events[r.events.length - 1];
		return i &&
			i[1].type === "listItemIndent" &&
			i[2].sliceSerialize(i[1], !0).length === r.containerState.size
			? e(u)
			: n(u);
	}
}
function vr(t) {
	t.exit(this.containerState.type);
}
function ti(t, e, n) {
	const r = this;
	return A(
		t,
		l,
		"listItemPrefixWhitespace",
		r.parser.constructs.disable.null.includes("codeIndented") ? void 0 : 5,
	);
	function l(u) {
		const i = r.events[r.events.length - 1];
		return !F(u) && i && i[1].type === "listItemPrefixWhitespace" ? e(u) : n(u);
	}
}
var $t = class {
	constructor() {
		(this.index = new Map()), (this.map = []);
	}
	add(e, n, r) {
		Zn(this, e, n, r, !1);
	}
	addBefore(e, n, r) {
		Zn(this, e, n, r, !0);
	}
	consume(e) {
		if ((this.map.sort((u, i) => u[0] - i[0]), this.map.length === 0)) return;
		let n = this.map.length,
			r = [];
		for (; n > 0; )
			(n -= 1),
				r.push(e.slice(this.map[n][0] + this.map[n][1]), this.map[n][2]),
				(e.length = this.map[n][0]);
		r.push(e.slice()), (e.length = 0);
		let l = r.pop();
		for (; l; ) {
			for (const u of l) e.push(u);
			l = r.pop();
		}
		(this.map.length = 0), this.index.clear();
	}
};
function Zn(t, e, n, r, l) {
	if (n === 0 && r.length === 0) return;
	const u = t.index.get(e);
	if (u) {
		(u[1] += n), l ? (r.push(...u[2]), (u[2] = r)) : u[2].push(...r);
		return;
	}
	const i = [e, n, r];
	t.map.push(i), t.index.set(e, i);
}
var Zt = { name: "setextUnderline", resolveTo: ni, tokenize: ei };
function ni(t, e) {
	let n = new $t(),
		r = t.length,
		l,
		u,
		i;
	for (; r--; )
		if (t[r][0] === "enter") {
			if (t[r][1].type === "content") {
				l = r;
				break;
			}
			t[r][1].type === "paragraph" && (u = r);
		} else
			t[r][1].type === "content" && n.add(r, 1, []),
				!i && t[r][1].type === "definition" && (i = r);
	const o = {
		type: "setextHeading",
		start: { ...t[l][1].start },
		end: { ...t[t.length - 1][1].end },
	};
	return (
		(t[u][1].type = "setextHeadingText"),
		i
			? (n.add(u, 0, [["enter", o, e]]),
				n.add(i + 1, 0, [["exit", t[l][1], e]]),
				(t[l][1].end = { ...t[i][1].end }))
			: (t[l][1] = o),
		n.add(t.length, 0, [["exit", o, e]]),
		n.consume(t),
		t
	);
}
function ei(t, e, n) {
	let r = this,
		l;
	return u;
	function u(c) {
		let m = r.events.length,
			s;
		for (; m--; )
			if (
				r.events[m][1].type !== "lineEnding" &&
				r.events[m][1].type !== "linePrefix" &&
				r.events[m][1].type !== "content"
			) {
				s = r.events[m][1].type === "paragraph";
				break;
			}
		return !r.parser.lazy[r.now().line] && (r.interrupt || s)
			? (t.enter("setextHeadingLine"), (l = c), i(c))
			: n(c);
	}
	function i(c) {
		return t.enter("setextHeadingLineSequence"), o(c);
	}
	function o(c) {
		return c === l
			? (t.consume(c), o)
			: (t.exit("setextHeadingLineSequence"),
				F(c) ? A(t, a, "lineSuffix")(c) : a(c));
	}
	function a(c) {
		return c === null || w(c) ? (t.exit("setextHeadingLine"), e(c)) : n(c);
	}
}
var Gn = { tokenize: ri };
function ri(t) {
	const e = this,
		n = t.attempt(
			it,
			r,
			t.attempt(
				this.parser.constructs.flowInitial,
				l,
				A(
					t,
					t.attempt(this.parser.constructs.flow, l, t.attempt(en, l)),
					"linePrefix",
				),
			),
		);
	return n;
	function r(u) {
		if (u === null) {
			t.consume(u);
			return;
		}
		return (
			t.enter("lineEndingBlank"),
			t.consume(u),
			t.exit("lineEndingBlank"),
			(e.currentConstruct = void 0),
			n
		);
	}
	function l(u) {
		if (u === null) {
			t.consume(u);
			return;
		}
		return (
			t.enter("lineEnding"),
			t.consume(u),
			t.exit("lineEnding"),
			(e.currentConstruct = void 0),
			n
		);
	}
}
var Jn = { resolveAll: te() },
	Xn = vn("string"),
	Kn = vn("text");
function vn(t) {
	return { resolveAll: te(t === "text" ? ii : void 0), tokenize: e };
	function e(n) {
		const r = this,
			l = this.parser.constructs[t],
			u = n.attempt(l, i, o);
		return i;
		function i(m) {
			return c(m) ? u(m) : o(m);
		}
		function o(m) {
			if (m === null) {
				n.consume(m);
				return;
			}
			return n.enter("data"), n.consume(m), a;
		}
		function a(m) {
			return c(m) ? (n.exit("data"), u(m)) : (n.consume(m), a);
		}
		function c(m) {
			if (m === null) return !0;
			let s = l[m],
				g = -1;
			if (s)
				for (; ++g < s.length; ) {
					const p = s[g];
					if (!p.previous || p.previous.call(r, r.previous)) return !0;
				}
			return !1;
		}
	}
}
function te(t) {
	return e;
	function e(n, r) {
		let l = -1,
			u;
		for (; ++l <= n.length; )
			u === void 0
				? n[l] && n[l][1].type === "data" && ((u = l), l++)
				: (!n[l] || n[l][1].type !== "data") &&
					(l !== u + 2 &&
						((n[u][1].end = n[l - 1][1].end),
						n.splice(u + 2, l - u - 2),
						(l = u + 2)),
					(u = void 0));
		return t ? t(n, r) : n;
	}
}
function ii(t, e) {
	let n = 0;
	for (; ++n <= t.length; )
		if (
			(n === t.length || t[n][1].type === "lineEnding") &&
			t[n - 1][1].type === "data"
		) {
			let r = t[n - 1][1],
				l = e.sliceStream(r),
				u = l.length,
				i = -1,
				o = 0,
				a;
			for (; u--; ) {
				const c = l[u];
				if (typeof c == "string") {
					for (i = c.length; c.charCodeAt(i - 1) === 32; ) o++, i--;
					if (i) break;
					i = -1;
				} else if (c === -2) (a = !0), o++;
				else if (c !== -1) {
					u++;
					break;
				}
			}
			if ((e._contentTypeTextTrailing && n === t.length && (o = 0), o)) {
				const c = {
					type:
						n === t.length || a || o < 2 ? "lineSuffix" : "hardBreakTrailing",
					start: {
						_bufferIndex: u ? i : r.start._bufferIndex + i,
						_index: r.start._index + u,
						line: r.end.line,
						column: r.end.column - o,
						offset: r.end.offset - o,
					},
					end: { ...r.end },
				};
				(r.end = { ...c.start }),
					r.start.offset === r.end.offset
						? Object.assign(r, c)
						: (t.splice(n, 0, ["enter", c, e], ["exit", c, e]), (n += 2));
			}
			n++;
		}
	return t;
}
var hn = {};
Ye(hn, {
	attentionMarkers: () => hi,
	contentInitial: () => li,
	disable: () => fi,
	document: () => ui,
	flow: () => ai,
	flowInitial: () => oi,
	insideSpan: () => pi,
	string: () => si,
	text: () => mi,
});
var ui = {
		42: Z,
		43: Z,
		45: Z,
		48: Z,
		49: Z,
		50: Z,
		51: Z,
		52: Z,
		53: Z,
		54: Z,
		55: Z,
		56: Z,
		57: Z,
		62: Nt,
	},
	li = { 91: rn },
	oi = { [-2]: Lt, [-1]: Lt, 32: Lt },
	ai = {
		35: ln,
		42: xt,
		45: [Zt, xt],
		60: an,
		61: Zt,
		95: xt,
		96: Vt,
		126: Vt,
	},
	si = { 38: qt, 92: Ht },
	mi = {
		[-5]: Dt,
		[-4]: Dt,
		[-3]: Dt,
		33: mn,
		38: qt,
		42: Ft,
		60: [tn, sn],
		91: pn,
		92: [un, Ht],
		93: gt,
		95: Ft,
		96: nn,
	},
	pi = { null: [Ft, Jn] },
	hi = { null: [42, 95] },
	fi = { null: [] };
function ne(t, e, n) {
	let r = {
			_bufferIndex: -1,
			_index: 0,
			line: (n && n.line) || 1,
			column: (n && n.column) || 1,
			offset: (n && n.offset) || 0,
		},
		l = {},
		u = [],
		i = [],
		o = [],
		a = !0,
		c = {
			attempt: N(P),
			check: N(R),
			consume: B,
			enter: T,
			exit: b,
			interrupt: N(R, { interrupt: !0 }),
		},
		m = {
			code: null,
			containerState: {},
			defineSkip: d,
			events: [],
			now: L,
			parser: t,
			previous: null,
			sliceSerialize: x,
			sliceStream: E,
			write: p,
		},
		s = e.tokenize.call(m, c),
		g;
	return e.resolveAll && u.push(e), m;
	function p(I) {
		return (
			(i = j(i, I)),
			C(),
			i[i.length - 1] !== null
				? []
				: (k(e, 0), (m.events = ht(u, m.events, m)), m.events)
		);
	}
	function x(I, D) {
		return gi(E(I), D);
	}
	function E(I) {
		return ci(i, I);
	}
	function L() {
		const { _bufferIndex: I, _index: D, line: _, column: H, offset: W } = r;
		return { _bufferIndex: I, _index: D, line: _, column: H, offset: W };
	}
	function d(I) {
		(l[I.line] = I.column), M();
	}
	function C() {
		let I;
		for (; r._index < i.length; ) {
			const D = i[r._index];
			if (typeof D == "string")
				for (
					I = r._index, r._bufferIndex < 0 && (r._bufferIndex = 0);
					r._index === I && r._bufferIndex < D.length;
				)
					y(D.charCodeAt(r._bufferIndex));
			else y(D);
		}
	}
	function y(I) {
		(a = void 0), (g = I), (s = s(I));
	}
	function B(I) {
		w(I)
			? (r.line++, (r.column = 1), (r.offset += I === -3 ? 2 : 1), M())
			: I !== -1 && (r.column++, r.offset++),
			r._bufferIndex < 0
				? r._index++
				: (r._bufferIndex++,
					r._bufferIndex === i[r._index].length &&
						((r._bufferIndex = -1), r._index++)),
			(m.previous = I),
			(a = !0);
	}
	function T(I, D) {
		const _ = D || {};
		return (
			(_.type = I),
			(_.start = L()),
			m.events.push(["enter", _, m]),
			o.push(_),
			_
		);
	}
	function b(I) {
		const D = o.pop();
		return (D.end = L()), m.events.push(["exit", D, m]), D;
	}
	function P(I, D) {
		k(I, D.from);
	}
	function R(I, D) {
		D.restore();
	}
	function N(I, D) {
		return _;
		function _(H, W, K) {
			let tt, h, X, lt;
			return Array.isArray(H) ? J(H) : "tokenize" in H ? J([H]) : f(H);
			function f(Y) {
				return Ct;
				function Ct(ot) {
					const dt = ot !== null && Y[ot],
						St = ot !== null && Y.null,
						Ot = [
							...(Array.isArray(dt) ? dt : dt ? [dt] : []),
							...(Array.isArray(St) ? St : St ? [St] : []),
						];
					return J(Ot)(ot);
				}
			}
			function J(Y) {
				return (tt = Y), (h = 0), Y.length === 0 ? K : bt(Y[h]);
			}
			function bt(Y) {
				return Ct;
				function Ct(ot) {
					return (
						(lt = Q()),
						(X = Y),
						Y.partial || (m.currentConstruct = Y),
						Y.name && m.parser.constructs.disable.null.includes(Y.name)
							? _t(ot)
							: Y.tokenize.call(
									D ? Object.assign(Object.create(m), D) : m,
									c,
									Xt,
									_t,
								)(ot)
					);
				}
			}
			function Xt(Y) {
				return (a = !0), I(X, lt), W;
			}
			function _t(Y) {
				return (a = !0), lt.restore(), ++h < tt.length ? bt(tt[h]) : K;
			}
		}
	}
	function k(I, D) {
		I.resolveAll && !u.includes(I) && u.push(I),
			I.resolve &&
				V(m.events, D, m.events.length - D, I.resolve(m.events.slice(D), m)),
			I.resolveTo && (m.events = I.resolveTo(m.events, m));
	}
	function Q() {
		const I = L(),
			D = m.previous,
			_ = m.currentConstruct,
			H = m.events.length,
			W = Array.from(o);
		return { from: H, restore: K };
		function K() {
			(r = I),
				(m.previous = D),
				(m.currentConstruct = _),
				(m.events.length = H),
				(o = W),
				M();
		}
	}
	function M() {
		r.line in l &&
			r.column < 2 &&
			((r.column = l[r.line]), (r.offset += l[r.line] - 1));
	}
}
function ci(t, e) {
	let n = e.start._index,
		r = e.start._bufferIndex,
		l = e.end._index,
		u = e.end._bufferIndex,
		i;
	if (n === l) i = [t[n].slice(r, u)];
	else {
		if (((i = t.slice(n, l)), r > -1)) {
			const o = i[0];
			typeof o == "string" ? (i[0] = o.slice(r)) : i.shift();
		}
		u > 0 && i.push(t[l].slice(0, u));
	}
	return i;
}
function gi(t, e) {
	let n = -1,
		r = [],
		l;
	for (; ++n < t.length; ) {
		let u = t[n],
			i;
		if (typeof u == "string") i = u;
		else
			switch (u) {
				case -5: {
					i = "\r";
					break;
				}
				case -4: {
					i = `
`;
					break;
				}
				case -3: {
					i = `\r
`;
					break;
				}
				case -2: {
					i = e ? " " : "	";
					break;
				}
				case -1: {
					if (!e && l) continue;
					i = " ";
					break;
				}
				default:
					i = String.fromCharCode(u);
			}
		(l = u === -2), r.push(i);
	}
	return r.join("");
}
function ee(t) {
	const r = {
		constructs: Bt([hn, ...((t || {}).extensions || [])]),
		content: l(Wn),
		defined: [],
		document: l(Qn),
		flow: l(Gn),
		lazy: {},
		string: l(Xn),
		text: l(Kn),
	};
	return r;
	function l(u) {
		return i;
		function i(o) {
			return ne(r, u, o);
		}
	}
}
function re(t) {
	for (; !Wt(t); );
	return t;
}
var ie = /[\0\t\n\r]/g;
function ue() {
	let t = 1,
		e = "",
		n = !0,
		r;
	return l;
	function l(u, i, o) {
		let a = [],
			c,
			m,
			s,
			g,
			p;
		for (
			u =
				e +
				(typeof u == "string"
					? u.toString()
					: new TextDecoder(i || void 0).decode(u)),
				s = 0,
				e = "",
				n && (u.charCodeAt(0) === 65279 && s++, (n = void 0));
			s < u.length;
		) {
			if (
				((ie.lastIndex = s),
				(c = ie.exec(u)),
				(g = c && c.index !== void 0 ? c.index : u.length),
				(p = u.charCodeAt(g)),
				!c)
			) {
				e = u.slice(s);
				break;
			}
			if (p === 10 && s === g && r) a.push(-3), (r = void 0);
			else
				switch (
					(r && (a.push(-5), (r = void 0)),
					s < g && (a.push(u.slice(s, g)), (t += g - s)),
					p)
				) {
					case 0: {
						a.push(65533), t++;
						break;
					}
					case 9: {
						for (m = Math.ceil(t / 4) * 4, a.push(-2); t++ < m; ) a.push(-1);
						break;
					}
					case 10: {
						a.push(-4), (t = 1);
						break;
					}
					default:
						(r = !0), (t = 1);
				}
			s = g + 1;
		}
		return o && (r && a.push(-5), e && a.push(e), a.push(null)), a;
	}
}
function xi(t, e, n) {
	return (
		typeof e != "string" && ((n = e), (e = void 0)),
		jn(n)(
			re(
				ee(n)
					.document()
					.write(ue()(t, e, !0)),
			),
		)
	);
}
var ki = { tokenize: yi, partial: !0 },
	le = { tokenize: Ci, partial: !0 },
	oe = { tokenize: Ei, partial: !0 },
	ae = { tokenize: Ii, partial: !0 },
	bi = { tokenize: zi, partial: !0 },
	se = { name: "wwwAutolink", tokenize: Si, previous: pe },
	me = { name: "protocolAutolink", tokenize: wi, previous: he },
	st = { name: "emailAutolink", tokenize: di, previous: fe },
	ut = {};
function cn() {
	return { text: ut };
}
var kt = 48;
for (; kt < 123; )
	(ut[kt] = st), kt++, kt === 58 ? (kt = 65) : kt === 91 && (kt = 97);
ut[43] = st;
ut[45] = st;
ut[46] = st;
ut[95] = st;
ut[72] = [st, me];
ut[104] = [st, me];
ut[87] = [st, se];
ut[119] = [st, se];
function di(t, e, n) {
	let r = this,
		l,
		u;
	return i;
	function i(s) {
		return !fn(s) || !fe.call(r, r.previous) || gn(r.events)
			? n(s)
			: (t.enter("literalAutolink"), t.enter("literalAutolinkEmail"), o(s));
	}
	function o(s) {
		return fn(s) ? (t.consume(s), o) : s === 64 ? (t.consume(s), a) : n(s);
	}
	function a(s) {
		return s === 46
			? t.check(bi, m, c)(s)
			: s === 45 || s === 95 || q(s)
				? ((u = !0), t.consume(s), a)
				: m(s);
	}
	function c(s) {
		return t.consume(s), (l = !0), a;
	}
	function m(s) {
		return u && l && U(r.previous)
			? (t.exit("literalAutolinkEmail"), t.exit("literalAutolink"), e(s))
			: n(s);
	}
}
function Si(t, e, n) {
	const r = this;
	return l;
	function l(i) {
		return (i !== 87 && i !== 119) || !pe.call(r, r.previous) || gn(r.events)
			? n(i)
			: (t.enter("literalAutolink"),
				t.enter("literalAutolinkWww"),
				t.check(ki, t.attempt(le, t.attempt(oe, u), n), n)(i));
	}
	function u(i) {
		return t.exit("literalAutolinkWww"), t.exit("literalAutolink"), e(i);
	}
}
function wi(t, e, n) {
	let r = this,
		l = "",
		u = !1;
	return i;
	function i(s) {
		return (s === 72 || s === 104) && he.call(r, r.previous) && !gn(r.events)
			? (t.enter("literalAutolink"),
				t.enter("literalAutolinkHttp"),
				(l += String.fromCodePoint(s)),
				t.consume(s),
				o)
			: n(s);
	}
	function o(s) {
		if (U(s) && l.length < 5)
			return (l += String.fromCodePoint(s)), t.consume(s), o;
		if (s === 58) {
			const g = l.toLowerCase();
			if (g === "http" || g === "https") return t.consume(s), a;
		}
		return n(s);
	}
	function a(s) {
		return s === 47 ? (t.consume(s), u ? c : ((u = !0), a)) : n(s);
	}
	function c(s) {
		return s === null || ft(s) || O(s) || mt(s) || zt(s)
			? n(s)
			: t.attempt(le, t.attempt(oe, m), n)(s);
	}
	function m(s) {
		return t.exit("literalAutolinkHttp"), t.exit("literalAutolink"), e(s);
	}
}
function yi(t, e, n) {
	let r = 0;
	return l;
	function l(i) {
		return (i === 87 || i === 119) && r < 3
			? (r++, t.consume(i), l)
			: i === 46 && r === 3
				? (t.consume(i), u)
				: n(i);
	}
	function u(i) {
		return i === null ? n(i) : e(i);
	}
}
function Ci(t, e, n) {
	let r, l, u;
	return i;
	function i(c) {
		return c === 46 || c === 95
			? t.check(ae, a, o)(c)
			: c === null || O(c) || mt(c) || (c !== 45 && zt(c))
				? a(c)
				: ((u = !0), t.consume(c), i);
	}
	function o(c) {
		return c === 95 ? (r = !0) : ((l = r), (r = void 0)), t.consume(c), i;
	}
	function a(c) {
		return l || r || !u ? n(c) : e(c);
	}
}
function Ei(t, e) {
	let n = 0,
		r = 0;
	return l;
	function l(i) {
		return i === 40
			? (n++, t.consume(i), l)
			: i === 41 && r < n
				? u(i)
				: i === 33 ||
						i === 34 ||
						i === 38 ||
						i === 39 ||
						i === 41 ||
						i === 42 ||
						i === 44 ||
						i === 46 ||
						i === 58 ||
						i === 59 ||
						i === 60 ||
						i === 63 ||
						i === 93 ||
						i === 95 ||
						i === 126
					? t.check(ae, e, u)(i)
					: i === null || O(i) || mt(i)
						? e(i)
						: (t.consume(i), l);
	}
	function u(i) {
		return i === 41 && r++, t.consume(i), l;
	}
}
function Ii(t, e, n) {
	return r;
	function r(o) {
		return o === 33 ||
			o === 34 ||
			o === 39 ||
			o === 41 ||
			o === 42 ||
			o === 44 ||
			o === 46 ||
			o === 58 ||
			o === 59 ||
			o === 63 ||
			o === 95 ||
			o === 126
			? (t.consume(o), r)
			: o === 38
				? (t.consume(o), u)
				: o === 93
					? (t.consume(o), l)
					: o === 60 || o === null || O(o) || mt(o)
						? e(o)
						: n(o);
	}
	function l(o) {
		return o === null || o === 40 || o === 91 || O(o) || mt(o) ? e(o) : r(o);
	}
	function u(o) {
		return U(o) ? i(o) : n(o);
	}
	function i(o) {
		return o === 59 ? (t.consume(o), r) : U(o) ? (t.consume(o), i) : n(o);
	}
}
function zi(t, e, n) {
	return r;
	function r(u) {
		return t.consume(u), l;
	}
	function l(u) {
		return q(u) ? n(u) : e(u);
	}
}
function pe(t) {
	return (
		t === null ||
		t === 40 ||
		t === 42 ||
		t === 95 ||
		t === 91 ||
		t === 93 ||
		t === 126 ||
		O(t)
	);
}
function he(t) {
	return !U(t);
}
function fe(t) {
	return !(t === 47 || fn(t));
}
function fn(t) {
	return t === 43 || t === 45 || t === 46 || t === 95 || q(t);
}
function gn(t) {
	let e = t.length,
		n = !1;
	for (; e--; ) {
		const r = t[e][1];
		if ((r.type === "labelLink" || r.type === "labelImage") && !r._balanced) {
			n = !0;
			break;
		}
		if (r._gfmAutolinkLiteralWalkedInto) {
			n = !1;
			break;
		}
	}
	return (
		t.length > 0 &&
			!n &&
			(t[t.length - 1][1]._gfmAutolinkLiteralWalkedInto = !0),
		n
	);
}
function xn() {
	return {
		exit: {
			literalAutolinkEmail: Ai,
			literalAutolinkHttp: Li,
			literalAutolinkWww: Fi,
		},
	};
}
function Fi(t) {
	kn.call(this, t, "http://");
}
function Ai(t) {
	kn.call(this, t, "mailto:");
}
function Li(t) {
	kn.call(this, t);
}
function kn(t, e) {
	const n = this.sliceSerialize(t);
	this.tag('<a href="' + rt((e || "") + n) + '">'),
		this.raw(this.encode(n)),
		this.tag("</a>");
}
var Ti = { tokenize: Ri, partial: !0 };
function bn() {
	return {
		document: {
			91: {
				name: "gfmFootnoteDefinition",
				tokenize: Pi,
				continuation: { tokenize: Bi },
				exit: Mi,
			},
		},
		text: {
			91: { name: "gfmFootnoteCall", tokenize: Oi },
			93: {
				name: "gfmPotentialFootnoteCall",
				add: "after",
				tokenize: Di,
				resolveTo: _i,
			},
		},
	};
}
function Di(t, e, n) {
	let r = this,
		l = r.events.length,
		u = r.parser.gfmFootnotes || (r.parser.gfmFootnotes = []),
		i;
	for (; l--; ) {
		const a = r.events[l][1];
		if (a.type === "labelImage") {
			i = a;
			break;
		}
		if (
			a.type === "gfmFootnoteCall" ||
			a.type === "labelLink" ||
			a.type === "label" ||
			a.type === "image" ||
			a.type === "link"
		)
			break;
	}
	return o;
	function o(a) {
		if (!i || !i._balanced) return n(a);
		const c = G(r.sliceSerialize({ start: i.end, end: r.now() }));
		return c.codePointAt(0) !== 94 || !u.includes(c.slice(1))
			? n(a)
			: (t.enter("gfmFootnoteCallLabelMarker"),
				t.consume(a),
				t.exit("gfmFootnoteCallLabelMarker"),
				e(a));
	}
}
function _i(t, e) {
	let n = t.length,
		r;
	for (; n--; )
		if (t[n][1].type === "labelImage" && t[n][0] === "enter") {
			r = t[n][1];
			break;
		}
	(t[n + 1][1].type = "data"),
		(t[n + 3][1].type = "gfmFootnoteCallLabelMarker");
	const l = {
			type: "gfmFootnoteCall",
			start: Object.assign({}, t[n + 3][1].start),
			end: Object.assign({}, t[t.length - 1][1].end),
		},
		u = {
			type: "gfmFootnoteCallMarker",
			start: Object.assign({}, t[n + 3][1].end),
			end: Object.assign({}, t[n + 3][1].end),
		};
	u.end.column++, u.end.offset++, u.end._bufferIndex++;
	const i = {
			type: "gfmFootnoteCallString",
			start: Object.assign({}, u.end),
			end: Object.assign({}, t[t.length - 1][1].start),
		},
		o = {
			type: "chunkString",
			contentType: "string",
			start: Object.assign({}, i.start),
			end: Object.assign({}, i.end),
		},
		a = [
			t[n + 1],
			t[n + 2],
			["enter", l, e],
			t[n + 3],
			t[n + 4],
			["enter", u, e],
			["exit", u, e],
			["enter", i, e],
			["enter", o, e],
			["exit", o, e],
			["exit", i, e],
			t[t.length - 2],
			t[t.length - 1],
			["exit", l, e],
		];
	return t.splice(n, t.length - n + 1, ...a), t;
}
function Oi(t, e, n) {
	let r = this,
		l = r.parser.gfmFootnotes || (r.parser.gfmFootnotes = []),
		u = 0,
		i;
	return o;
	function o(s) {
		return (
			t.enter("gfmFootnoteCall"),
			t.enter("gfmFootnoteCallLabelMarker"),
			t.consume(s),
			t.exit("gfmFootnoteCallLabelMarker"),
			a
		);
	}
	function a(s) {
		return s !== 94
			? n(s)
			: (t.enter("gfmFootnoteCallMarker"),
				t.consume(s),
				t.exit("gfmFootnoteCallMarker"),
				t.enter("gfmFootnoteCallString"),
				(t.enter("chunkString").contentType = "string"),
				c);
	}
	function c(s) {
		if (u > 999 || (s === 93 && !i) || s === null || s === 91 || O(s))
			return n(s);
		if (s === 93) {
			t.exit("chunkString");
			const g = t.exit("gfmFootnoteCallString");
			return l.includes(G(r.sliceSerialize(g)))
				? (t.enter("gfmFootnoteCallLabelMarker"),
					t.consume(s),
					t.exit("gfmFootnoteCallLabelMarker"),
					t.exit("gfmFootnoteCall"),
					e)
				: n(s);
		}
		return O(s) || (i = !0), u++, t.consume(s), s === 92 ? m : c;
	}
	function m(s) {
		return s === 91 || s === 92 || s === 93 ? (t.consume(s), u++, c) : c(s);
	}
}
function Pi(t, e, n) {
	let r = this,
		l = r.parser.gfmFootnotes || (r.parser.gfmFootnotes = []),
		u,
		i = 0,
		o;
	return a;
	function a(x) {
		return (
			(t.enter("gfmFootnoteDefinition")._container = !0),
			t.enter("gfmFootnoteDefinitionLabel"),
			t.enter("gfmFootnoteDefinitionLabelMarker"),
			t.consume(x),
			t.exit("gfmFootnoteDefinitionLabelMarker"),
			c
		);
	}
	function c(x) {
		return x === 94
			? (t.enter("gfmFootnoteDefinitionMarker"),
				t.consume(x),
				t.exit("gfmFootnoteDefinitionMarker"),
				t.enter("gfmFootnoteDefinitionLabelString"),
				(t.enter("chunkString").contentType = "string"),
				m)
			: n(x);
	}
	function m(x) {
		if (i > 999 || (x === 93 && !o) || x === null || x === 91 || O(x))
			return n(x);
		if (x === 93) {
			t.exit("chunkString");
			const E = t.exit("gfmFootnoteDefinitionLabelString");
			return (
				(u = G(r.sliceSerialize(E))),
				t.enter("gfmFootnoteDefinitionLabelMarker"),
				t.consume(x),
				t.exit("gfmFootnoteDefinitionLabelMarker"),
				t.exit("gfmFootnoteDefinitionLabel"),
				g
			);
		}
		return O(x) || (o = !0), i++, t.consume(x), x === 92 ? s : m;
	}
	function s(x) {
		return x === 91 || x === 92 || x === 93 ? (t.consume(x), i++, m) : m(x);
	}
	function g(x) {
		return x === 58
			? (t.enter("definitionMarker"),
				t.consume(x),
				t.exit("definitionMarker"),
				l.includes(u) || l.push(u),
				A(t, p, "gfmFootnoteDefinitionWhitespace"))
			: n(x);
	}
	function p(x) {
		return e(x);
	}
}
function Bi(t, e, n) {
	return t.check(it, e, t.attempt(Ti, e, n));
}
function Mi(t) {
	t.exit("gfmFootnoteDefinition");
}
function Ri(t, e, n) {
	const r = this;
	return A(t, l, "gfmFootnoteDefinitionIndent", 5);
	function l(u) {
		const i = r.events[r.events.length - 1];
		return i &&
			i[1].type === "gfmFootnoteDefinitionIndent" &&
			i[2].sliceSerialize(i[1], !0).length === 4
			? e(u)
			: n(u);
	}
}
var Ni = {}.hasOwnProperty,
	Hi = {};
function ce(t, e) {
	return "Back to reference " + (t + 1) + (e > 1 ? "-" + e : "");
}
function dn(t) {
	const e = t || Hi,
		n = e.label || "Footnotes",
		r = e.labelTagName || "h2",
		l =
			e.labelAttributes === null || e.labelAttributes === void 0
				? 'class="sr-only"'
				: e.labelAttributes,
		u = e.backLabel || ce,
		i =
			e.clobberPrefix === null || e.clobberPrefix === void 0
				? "user-content-"
				: e.clobberPrefix;
	return {
		enter: {
			gfmFootnoteDefinition() {
				this.getData("tightStack").push(!1);
			},
			gfmFootnoteDefinitionLabelString() {
				this.buffer();
			},
			gfmFootnoteCallString() {
				this.buffer();
			},
		},
		exit: {
			gfmFootnoteDefinition() {
				let o = this.getData("gfmFootnoteDefinitions"),
					a = this.getData("gfmFootnoteDefinitionStack"),
					c = this.getData("tightStack"),
					m = a.pop(),
					s = this.resume();
				o || this.setData("gfmFootnoteDefinitions", (o = {})),
					Ni.call(o, m) || (o[m] = s),
					c.pop(),
					this.setData("slurpOneLineEnding", !0),
					this.setData("lastWasTag");
			},
			gfmFootnoteDefinitionLabelString(o) {
				let a = this.getData("gfmFootnoteDefinitionStack");
				a || this.setData("gfmFootnoteDefinitionStack", (a = [])),
					a.push(G(this.sliceSerialize(o))),
					this.resume(),
					this.buffer();
			},
			gfmFootnoteCallString(o) {
				let a = this.getData("gfmFootnoteCallOrder"),
					c = this.getData("gfmFootnoteCallCounts"),
					m = G(this.sliceSerialize(o)),
					s;
				this.resume(),
					a || this.setData("gfmFootnoteCallOrder", (a = [])),
					c || this.setData("gfmFootnoteCallCounts", (c = {}));
				const g = a.indexOf(m),
					p = rt(m.toLowerCase());
				g === -1
					? (a.push(m), (c[m] = 1), (s = a.length))
					: (c[m]++, (s = g + 1));
				const x = c[m];
				this.tag(
					'<sup><a href="#' +
						i +
						"fn-" +
						p +
						'" id="' +
						i +
						"fnref-" +
						p +
						(x > 1 ? "-" + x : "") +
						'" data-footnote-ref="" aria-describedby="footnote-label">' +
						String(s) +
						"</a></sup>",
				);
			},
			null() {
				let o = this.getData("gfmFootnoteCallOrder") || [],
					a = this.getData("gfmFootnoteCallCounts") || {},
					c = this.getData("gfmFootnoteDefinitions") || {},
					m = -1;
				for (
					o.length > 0 &&
					(this.lineEndingIfNeeded(),
					this.tag(
						'<section data-footnotes="" class="footnotes"><' +
							r +
							' id="footnote-label"' +
							(l ? " " + l : "") +
							">",
					),
					this.raw(this.encode(n)),
					this.tag("</" + r + ">"),
					this.lineEndingIfNeeded(),
					this.tag("<ol>"));
					++m < o.length;
				) {
					let s = o[m],
						g = rt(s.toLowerCase()),
						p = 0,
						x = [];
					for (; ++p <= a[s]; )
						x.push(
							'<a href="#' +
								i +
								"fnref-" +
								g +
								(p > 1 ? "-" + p : "") +
								'" data-footnote-backref="" aria-label="' +
								this.encode(typeof u == "string" ? u : u(m, p)) +
								'" class="data-footnote-backref">\u21A9' +
								(p > 1 ? "<sup>" + p + "</sup>" : "") +
								"</a>",
						);
					let E = x.join(" "),
						L = !1;
					this.lineEndingIfNeeded(),
						this.tag('<li id="' + i + "fn-" + g + '">'),
						this.lineEndingIfNeeded(),
						this.tag(
							c[s].replace(
								/<\/p>(?:\r?\n|\r)?$/,
								(d) => ((L = !0), " " + E + d),
							),
						),
						L || (this.lineEndingIfNeeded(), this.tag(E)),
						this.lineEndingIfNeeded(),
						this.tag("</li>");
				}
				o.length > 0 &&
					(this.lineEndingIfNeeded(),
					this.tag("</ol>"),
					this.lineEndingIfNeeded(),
					this.tag("</section>"));
			},
		},
	};
}
function Sn() {
	return {
		enter: {
			strikethrough() {
				this.tag("<del>");
			},
		},
		exit: {
			strikethrough() {
				this.tag("</del>");
			},
		},
	};
}
function wn(t) {
	let n = (t || {}).singleTilde,
		r = { name: "strikethrough", tokenize: u, resolveAll: l };
	return (
		n == null && (n = !0),
		{
			text: { 126: r },
			insideSpan: { null: [r] },
			attentionMarkers: { null: [126] },
		}
	);
	function l(i, o) {
		let a = -1;
		for (; ++a < i.length; )
			if (
				i[a][0] === "enter" &&
				i[a][1].type === "strikethroughSequenceTemporary" &&
				i[a][1]._close
			) {
				let c = a;
				for (; c--; )
					if (
						i[c][0] === "exit" &&
						i[c][1].type === "strikethroughSequenceTemporary" &&
						i[c][1]._open &&
						i[a][1].end.offset - i[a][1].start.offset ===
							i[c][1].end.offset - i[c][1].start.offset
					) {
						(i[a][1].type = "strikethroughSequence"),
							(i[c][1].type = "strikethroughSequence");
						const m = {
								type: "strikethrough",
								start: Object.assign({}, i[c][1].start),
								end: Object.assign({}, i[a][1].end),
							},
							s = {
								type: "strikethroughText",
								start: Object.assign({}, i[c][1].end),
								end: Object.assign({}, i[a][1].start),
							},
							g = [
								["enter", m, o],
								["enter", i[c][1], o],
								["exit", i[c][1], o],
								["enter", s, o],
							],
							p = o.parser.constructs.insideSpan.null;
						p && V(g, g.length, 0, ht(p, i.slice(c + 1, a), o)),
							V(g, g.length, 0, [
								["exit", s, o],
								["enter", i[a][1], o],
								["exit", i[a][1], o],
								["exit", m, o],
							]),
							V(i, c - 1, a - c + 3, g),
							(a = c + g.length - 2);
						break;
					}
			}
		for (a = -1; ++a < i.length; )
			i[a][1].type === "strikethroughSequenceTemporary" &&
				(i[a][1].type = "data");
		return i;
	}
	function u(i, o, a) {
		let c = this.previous,
			m = this.events,
			s = 0;
		return g;
		function g(x) {
			return c === 126 && m[m.length - 1][1].type !== "characterEscape"
				? a(x)
				: (i.enter("strikethroughSequenceTemporary"), p(x));
		}
		function p(x) {
			const E = wt(c);
			if (x === 126) return s > 1 ? a(x) : (i.consume(x), s++, p);
			if (s < 2 && !n) return a(x);
			const L = i.exit("strikethroughSequenceTemporary"),
				d = wt(x);
			return (
				(L._open = !d || (d === 2 && !!E)),
				(L._close = !E || (E === 2 && !!d)),
				o(x)
			);
		}
	}
}
var yn = {
	none: "",
	left: ' align="left"',
	right: ' align="right"',
	center: ' align="center"',
};
function Cn() {
	return {
		enter: {
			table(t) {
				const e = t._align;
				this.lineEndingIfNeeded(),
					this.tag("<table>"),
					this.setData("tableAlign", e);
			},
			tableBody() {
				this.tag("<tbody>");
			},
			tableData() {
				const t = this.getData("tableAlign"),
					e = this.getData("tableColumn"),
					n = yn[t[e]];
				n === void 0
					? this.buffer()
					: (this.lineEndingIfNeeded(), this.tag("<td" + n + ">"));
			},
			tableHead() {
				this.lineEndingIfNeeded(), this.tag("<thead>");
			},
			tableHeader() {
				const t = this.getData("tableAlign"),
					e = this.getData("tableColumn"),
					n = yn[t[e]];
				this.lineEndingIfNeeded(), this.tag("<th" + n + ">");
			},
			tableRow() {
				this.setData("tableColumn", 0),
					this.lineEndingIfNeeded(),
					this.tag("<tr>");
			},
		},
		exit: {
			codeTextData(t) {
				let e = this.sliceSerialize(t);
				this.getData("tableAlign") && (e = e.replace(/\\([\\|])/g, qi)),
					this.raw(this.encode(e));
			},
			table() {
				this.setData("tableAlign"),
					this.setData("slurpAllLineEndings"),
					this.lineEndingIfNeeded(),
					this.tag("</table>");
			},
			tableBody() {
				this.lineEndingIfNeeded(), this.tag("</tbody>");
			},
			tableData() {
				const t = this.getData("tableAlign"),
					e = this.getData("tableColumn");
				e in t
					? (this.tag("</td>"), this.setData("tableColumn", e + 1))
					: this.resume();
			},
			tableHead() {
				this.lineEndingIfNeeded(), this.tag("</thead>");
			},
			tableHeader() {
				const t = this.getData("tableColumn");
				this.tag("</th>"), this.setData("tableColumn", t + 1);
			},
			tableRow() {
				let t = this.getData("tableAlign"),
					e = this.getData("tableColumn");
				for (; e < t.length; )
					this.lineEndingIfNeeded(), this.tag("<td" + yn[t[e]] + "></td>"), e++;
				this.setData("tableColumn", e),
					this.lineEndingIfNeeded(),
					this.tag("</tr>");
			},
		},
	};
}
function qi(t, e) {
	return e === "|" ? e : t;
}
var Gt = class {
	constructor() {
		(this.map = []), (this.index = new Map());
	}
	add(e, n, r) {
		Vi(this, e, n, r);
	}
	consume(e) {
		if ((this.map.sort((u, i) => u[0] - i[0]), this.map.length === 0)) return;
		let n = this.map.length,
			r = [];
		for (; n > 0; )
			(n -= 1),
				r.push(e.slice(this.map[n][0] + this.map[n][1]), this.map[n][2]),
				(e.length = this.map[n][0]);
		r.push(e.slice()), (e.length = 0);
		let l = r.pop();
		for (; l; ) {
			for (const u of l) e.push(u);
			l = r.pop();
		}
		(this.map.length = 0), this.index.clear();
	}
};
function Vi(t, e, n, r) {
	if (n === 0 && r.length === 0) return;
	const l = t.index.get(e);
	if (l) {
		(l[1] += n), l[2].push(...r);
		return;
	}
	const u = [e, n, r];
	t.map.push(u), t.index.set(e, u);
}
function ge(t, e) {
	let n = !1,
		r = [];
	for (; e < t.length; ) {
		const l = t[e];
		if (n) {
			if (l[0] === "enter")
				l[1].type === "tableContent" &&
					r.push(t[e + 1][1].type === "tableDelimiterMarker" ? "left" : "none");
			else if (l[1].type === "tableContent") {
				if (t[e - 1][1].type === "tableDelimiterMarker") {
					const u = r.length - 1;
					r[u] = r[u] === "left" ? "center" : "right";
				}
			} else if (l[1].type === "tableDelimiterRow") break;
		} else l[0] === "enter" && l[1].type === "tableDelimiterRow" && (n = !0);
		e += 1;
	}
	return r;
}
function En() {
	return { flow: { null: { name: "table", tokenize: ji, resolveAll: Wi } } };
}
function ji(t, e, n) {
	let r = this,
		l = 0,
		u = 0,
		i;
	return o;
	function o(k) {
		let Q = r.events.length - 1;
		for (; Q > -1; ) {
			const { type: D } = r.events[Q][1];
			if (D === "lineEnding" || D === "linePrefix") Q--;
			else break;
		}
		const M = Q > -1 ? r.events[Q][1].type : null,
			I = M === "tableHead" || M === "tableRow" ? b : a;
		return I === b && r.parser.lazy[r.now().line] ? n(k) : I(k);
	}
	function a(k) {
		return t.enter("tableHead"), t.enter("tableRow"), c(k);
	}
	function c(k) {
		return k === 124 || ((i = !0), (u += 1)), m(k);
	}
	function m(k) {
		return k === null
			? n(k)
			: w(k)
				? u > 1
					? ((u = 0),
						(r.interrupt = !0),
						t.exit("tableRow"),
						t.enter("lineEnding"),
						t.consume(k),
						t.exit("lineEnding"),
						p)
					: n(k)
				: F(k)
					? A(t, m, "whitespace")(k)
					: ((u += 1),
						i && ((i = !1), (l += 1)),
						k === 124
							? (t.enter("tableCellDivider"),
								t.consume(k),
								t.exit("tableCellDivider"),
								(i = !0),
								m)
							: (t.enter("data"), s(k)));
	}
	function s(k) {
		return k === null || k === 124 || O(k)
			? (t.exit("data"), m(k))
			: (t.consume(k), k === 92 ? g : s);
	}
	function g(k) {
		return k === 92 || k === 124 ? (t.consume(k), s) : s(k);
	}
	function p(k) {
		return (
			(r.interrupt = !1),
			r.parser.lazy[r.now().line]
				? n(k)
				: (t.enter("tableDelimiterRow"),
					(i = !1),
					F(k)
						? A(
								t,
								x,
								"linePrefix",
								r.parser.constructs.disable.null.includes("codeIndented")
									? void 0
									: 4,
							)(k)
						: x(k))
		);
	}
	function x(k) {
		return k === 45 || k === 58
			? L(k)
			: k === 124
				? ((i = !0),
					t.enter("tableCellDivider"),
					t.consume(k),
					t.exit("tableCellDivider"),
					E)
				: T(k);
	}
	function E(k) {
		return F(k) ? A(t, L, "whitespace")(k) : L(k);
	}
	function L(k) {
		return k === 58
			? ((u += 1),
				(i = !0),
				t.enter("tableDelimiterMarker"),
				t.consume(k),
				t.exit("tableDelimiterMarker"),
				d)
			: k === 45
				? ((u += 1), d(k))
				: k === null || w(k)
					? B(k)
					: T(k);
	}
	function d(k) {
		return k === 45 ? (t.enter("tableDelimiterFiller"), C(k)) : T(k);
	}
	function C(k) {
		return k === 45
			? (t.consume(k), C)
			: k === 58
				? ((i = !0),
					t.exit("tableDelimiterFiller"),
					t.enter("tableDelimiterMarker"),
					t.consume(k),
					t.exit("tableDelimiterMarker"),
					y)
				: (t.exit("tableDelimiterFiller"), y(k));
	}
	function y(k) {
		return F(k) ? A(t, B, "whitespace")(k) : B(k);
	}
	function B(k) {
		return k === 124
			? x(k)
			: k === null || w(k)
				? !i || l !== u
					? T(k)
					: (t.exit("tableDelimiterRow"), t.exit("tableHead"), e(k))
				: T(k);
	}
	function T(k) {
		return n(k);
	}
	function b(k) {
		return t.enter("tableRow"), P(k);
	}
	function P(k) {
		return k === 124
			? (t.enter("tableCellDivider"),
				t.consume(k),
				t.exit("tableCellDivider"),
				P)
			: k === null || w(k)
				? (t.exit("tableRow"), e(k))
				: F(k)
					? A(t, P, "whitespace")(k)
					: (t.enter("data"), R(k));
	}
	function R(k) {
		return k === null || k === 124 || O(k)
			? (t.exit("data"), P(k))
			: (t.consume(k), k === 92 ? N : R);
	}
	function N(k) {
		return k === 92 || k === 124 ? (t.consume(k), R) : R(k);
	}
}
function Wi(t, e) {
	let n = -1,
		r = !0,
		l = 0,
		u = [0, 0, 0, 0],
		i = [0, 0, 0, 0],
		o = !1,
		a = 0,
		c,
		m,
		s,
		g = new Gt();
	for (; ++n < t.length; ) {
		const p = t[n],
			x = p[1];
		p[0] === "enter"
			? x.type === "tableHead"
				? ((o = !1),
					a !== 0 && (xe(g, e, a, c, m), (m = void 0), (a = 0)),
					(c = {
						type: "table",
						start: Object.assign({}, x.start),
						end: Object.assign({}, x.end),
					}),
					g.add(n, 0, [["enter", c, e]]))
				: x.type === "tableRow" || x.type === "tableDelimiterRow"
					? ((r = !0),
						(s = void 0),
						(u = [0, 0, 0, 0]),
						(i = [0, n + 1, 0, 0]),
						o &&
							((o = !1),
							(m = {
								type: "tableBody",
								start: Object.assign({}, x.start),
								end: Object.assign({}, x.end),
							}),
							g.add(n, 0, [["enter", m, e]])),
						(l = x.type === "tableDelimiterRow" ? 2 : m ? 3 : 1))
					: l &&
							(x.type === "data" ||
								x.type === "tableDelimiterMarker" ||
								x.type === "tableDelimiterFiller")
						? ((r = !1),
							i[2] === 0 &&
								(u[1] !== 0 &&
									((i[0] = i[1]),
									(s = Jt(g, e, u, l, void 0, s)),
									(u = [0, 0, 0, 0])),
								(i[2] = n)))
						: x.type === "tableCellDivider" &&
							(r
								? (r = !1)
								: (u[1] !== 0 &&
										((i[0] = i[1]), (s = Jt(g, e, u, l, void 0, s))),
									(u = i),
									(i = [u[1], n, 0, 0])))
			: x.type === "tableHead"
				? ((o = !0), (a = n))
				: x.type === "tableRow" || x.type === "tableDelimiterRow"
					? ((a = n),
						u[1] !== 0
							? ((i[0] = i[1]), (s = Jt(g, e, u, l, n, s)))
							: i[1] !== 0 && (s = Jt(g, e, i, l, n, s)),
						(l = 0))
					: l &&
						(x.type === "data" ||
							x.type === "tableDelimiterMarker" ||
							x.type === "tableDelimiterFiller") &&
						(i[3] = n);
	}
	for (
		a !== 0 && xe(g, e, a, c, m), g.consume(e.events), n = -1;
		++n < e.events.length;
	) {
		const p = e.events[n];
		p[0] === "enter" &&
			p[1].type === "table" &&
			(p[1]._align = ge(e.events, n));
	}
	return t;
}
function Jt(t, e, n, r, l, u) {
	const i = r === 1 ? "tableHeader" : r === 2 ? "tableDelimiter" : "tableData",
		o = "tableContent";
	n[0] !== 0 &&
		((u.end = Object.assign({}, yt(e.events, n[0]))),
		t.add(n[0], 0, [["exit", u, e]]));
	const a = yt(e.events, n[1]);
	if (
		((u = { type: i, start: Object.assign({}, a), end: Object.assign({}, a) }),
		t.add(n[1], 0, [["enter", u, e]]),
		n[2] !== 0)
	) {
		const c = yt(e.events, n[2]),
			m = yt(e.events, n[3]),
			s = { type: o, start: Object.assign({}, c), end: Object.assign({}, m) };
		if ((t.add(n[2], 0, [["enter", s, e]]), r !== 2)) {
			const g = e.events[n[2]],
				p = e.events[n[3]];
			if (
				((g[1].end = Object.assign({}, p[1].end)),
				(g[1].type = "chunkText"),
				(g[1].contentType = "text"),
				n[3] > n[2] + 1)
			) {
				const x = n[2] + 1,
					E = n[3] - n[2] - 1;
				t.add(x, E, []);
			}
		}
		t.add(n[3] + 1, 0, [["exit", s, e]]);
	}
	return (
		l !== void 0 &&
			((u.end = Object.assign({}, yt(e.events, l))),
			t.add(l, 0, [["exit", u, e]]),
			(u = void 0)),
		u
	);
}
function xe(t, e, n, r, l) {
	const u = [],
		i = yt(e.events, n);
	l && ((l.end = Object.assign({}, i)), u.push(["exit", l, e])),
		(r.end = Object.assign({}, i)),
		u.push(["exit", r, e]),
		t.add(n + 1, 0, u);
}
function yt(t, e) {
	const n = t[e],
		r = n[0] === "enter" ? "start" : "end";
	return n[1][r];
}
var be =
		/<(\/?)(iframe|noembed|noframes|plaintext|script|style|title|textarea|xmp)(?=[\t\n\f\r />])/gi,
	Ui = new RegExp("^" + be.source, "i");
function In() {
	return {
		exit: {
			htmlFlowData(t) {
				ke.call(this, t, be);
			},
			htmlTextData(t) {
				ke.call(this, t, Ui);
			},
		},
	};
}
function ke(t, e) {
	let n = this.sliceSerialize(t);
	this.options.allowDangerousHtml && (n = n.replace(e, "&lt;$1$2")),
		this.raw(this.encode(n));
}
function zn() {
	return {
		enter: {
			taskListCheck() {
				this.tag('<input type="checkbox" disabled="" ');
			},
		},
		exit: {
			taskListCheck() {
				this.tag("/>");
			},
			taskListCheckValueChecked() {
				this.tag('checked="" ');
			},
		},
	};
}
var Qi = { name: "tasklistCheck", tokenize: Yi };
function Fn() {
	return { text: { 91: Qi } };
}
function Yi(t, e, n) {
	const r = this;
	return l;
	function l(a) {
		return r.previous !== null || !r._gfmTasklistFirstContentOfListItem
			? n(a)
			: (t.enter("taskListCheck"),
				t.enter("taskListCheckMarker"),
				t.consume(a),
				t.exit("taskListCheckMarker"),
				u);
	}
	function u(a) {
		return O(a)
			? (t.enter("taskListCheckValueUnchecked"),
				t.consume(a),
				t.exit("taskListCheckValueUnchecked"),
				i)
			: a === 88 || a === 120
				? (t.enter("taskListCheckValueChecked"),
					t.consume(a),
					t.exit("taskListCheckValueChecked"),
					i)
				: n(a);
	}
	function i(a) {
		return a === 93
			? (t.enter("taskListCheckMarker"),
				t.consume(a),
				t.exit("taskListCheckMarker"),
				t.exit("taskListCheck"),
				o)
			: n(a);
	}
	function o(a) {
		return w(a) ? e(a) : F(a) ? t.check({ tokenize: $i }, e, n)(a) : n(a);
	}
}
function $i(t, e, n) {
	return A(t, r, "whitespace");
	function r(l) {
		return l === null ? n(l) : e(l);
	}
}
function Zi(t) {
	return Bt([cn(), bn(), wn(t), En(), Fn()]);
}
function Gi(t) {
	return Mt([xn(), dn(t), Sn(), Cn(), In(), zn()]);
}
export { Zi as gfm, Gi as gfmHtml, xi as micromark };
