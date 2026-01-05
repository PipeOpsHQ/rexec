var Ve = Object.defineProperty;
var Xe = (ne, B, I) => B in ne ? Ve(ne, B, { enumerable: !0, configurable: !0, writable: !0, value: I }) : ne[B] = I;
var oe = (ne, B, I) => Xe(ne, typeof B != "symbol" ? B + "" : B, I);
var Ae = { exports: {} }, $e;
function Je() {
  return $e || ($e = 1, (function(ne, B) {
    (function(I, $) {
      ne.exports = $();
    })(globalThis, (() => (() => {
      var I = { 4567: function(T, t, a) {
        var c = this && this.__decorate || function(s, i, u, p) {
          var l, m = arguments.length, _ = m < 3 ? i : p === null ? p = Object.getOwnPropertyDescriptor(i, u) : p;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") _ = Reflect.decorate(s, i, u, p);
          else for (var v = s.length - 1; v >= 0; v--) (l = s[v]) && (_ = (m < 3 ? l(_) : m > 3 ? l(i, u, _) : l(i, u)) || _);
          return m > 3 && _ && Object.defineProperty(i, u, _), _;
        }, h = this && this.__param || function(s, i) {
          return function(u, p) {
            i(u, p, s);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.AccessibilityManager = void 0;
        const r = a(9042), d = a(9924), f = a(844), g = a(4725), n = a(2585), e = a(3656);
        let o = t.AccessibilityManager = class extends f.Disposable {
          constructor(s, i, u, p) {
            super(), this._terminal = s, this._coreBrowserService = u, this._renderService = p, this._rowColumns = /* @__PURE__ */ new WeakMap(), this._liveRegionLineCount = 0, this._charsToConsume = [], this._charsToAnnounce = "", this._accessibilityContainer = this._coreBrowserService.mainDocument.createElement("div"), this._accessibilityContainer.classList.add("xterm-accessibility"), this._rowContainer = this._coreBrowserService.mainDocument.createElement("div"), this._rowContainer.setAttribute("role", "list"), this._rowContainer.classList.add("xterm-accessibility-tree"), this._rowElements = [];
            for (let l = 0; l < this._terminal.rows; l++) this._rowElements[l] = this._createAccessibilityTreeNode(), this._rowContainer.appendChild(this._rowElements[l]);
            if (this._topBoundaryFocusListener = (l) => this._handleBoundaryFocus(l, 0), this._bottomBoundaryFocusListener = (l) => this._handleBoundaryFocus(l, 1), this._rowElements[0].addEventListener("focus", this._topBoundaryFocusListener), this._rowElements[this._rowElements.length - 1].addEventListener("focus", this._bottomBoundaryFocusListener), this._refreshRowsDimensions(), this._accessibilityContainer.appendChild(this._rowContainer), this._liveRegion = this._coreBrowserService.mainDocument.createElement("div"), this._liveRegion.classList.add("live-region"), this._liveRegion.setAttribute("aria-live", "assertive"), this._accessibilityContainer.appendChild(this._liveRegion), this._liveRegionDebouncer = this.register(new d.TimeBasedDebouncer(this._renderRows.bind(this))), !this._terminal.element) throw new Error("Cannot enable accessibility before Terminal.open");
            this._terminal.element.insertAdjacentElement("afterbegin", this._accessibilityContainer), this.register(this._terminal.onResize(((l) => this._handleResize(l.rows)))), this.register(this._terminal.onRender(((l) => this._refreshRows(l.start, l.end)))), this.register(this._terminal.onScroll((() => this._refreshRows()))), this.register(this._terminal.onA11yChar(((l) => this._handleChar(l)))), this.register(this._terminal.onLineFeed((() => this._handleChar(`
`)))), this.register(this._terminal.onA11yTab(((l) => this._handleTab(l)))), this.register(this._terminal.onKey(((l) => this._handleKey(l.key)))), this.register(this._terminal.onBlur((() => this._clearLiveRegion()))), this.register(this._renderService.onDimensionsChange((() => this._refreshRowsDimensions()))), this.register((0, e.addDisposableDomListener)(document, "selectionchange", (() => this._handleSelectionChange()))), this.register(this._coreBrowserService.onDprChange((() => this._refreshRowsDimensions()))), this._refreshRows(), this.register((0, f.toDisposable)((() => {
              this._accessibilityContainer.remove(), this._rowElements.length = 0;
            })));
          }
          _handleTab(s) {
            for (let i = 0; i < s; i++) this._handleChar(" ");
          }
          _handleChar(s) {
            this._liveRegionLineCount < 21 && (this._charsToConsume.length > 0 ? this._charsToConsume.shift() !== s && (this._charsToAnnounce += s) : this._charsToAnnounce += s, s === `
` && (this._liveRegionLineCount++, this._liveRegionLineCount === 21 && (this._liveRegion.textContent += r.tooMuchOutput)));
          }
          _clearLiveRegion() {
            this._liveRegion.textContent = "", this._liveRegionLineCount = 0;
          }
          _handleKey(s) {
            this._clearLiveRegion(), /\p{Control}/u.test(s) || this._charsToConsume.push(s);
          }
          _refreshRows(s, i) {
            this._liveRegionDebouncer.refresh(s, i, this._terminal.rows);
          }
          _renderRows(s, i) {
            const u = this._terminal.buffer, p = u.lines.length.toString();
            for (let l = s; l <= i; l++) {
              const m = u.lines.get(u.ydisp + l), _ = [], v = (m == null ? void 0 : m.translateToString(!0, void 0, void 0, _)) || "", C = (u.ydisp + l + 1).toString(), w = this._rowElements[l];
              w && (v.length === 0 ? (w.innerText = " ", this._rowColumns.set(w, [0, 1])) : (w.textContent = v, this._rowColumns.set(w, _)), w.setAttribute("aria-posinset", C), w.setAttribute("aria-setsize", p));
            }
            this._announceCharacters();
          }
          _announceCharacters() {
            this._charsToAnnounce.length !== 0 && (this._liveRegion.textContent += this._charsToAnnounce, this._charsToAnnounce = "");
          }
          _handleBoundaryFocus(s, i) {
            const u = s.target, p = this._rowElements[i === 0 ? 1 : this._rowElements.length - 2];
            if (u.getAttribute("aria-posinset") === (i === 0 ? "1" : `${this._terminal.buffer.lines.length}`) || s.relatedTarget !== p) return;
            let l, m;
            if (i === 0 ? (l = u, m = this._rowElements.pop(), this._rowContainer.removeChild(m)) : (l = this._rowElements.shift(), m = u, this._rowContainer.removeChild(l)), l.removeEventListener("focus", this._topBoundaryFocusListener), m.removeEventListener("focus", this._bottomBoundaryFocusListener), i === 0) {
              const _ = this._createAccessibilityTreeNode();
              this._rowElements.unshift(_), this._rowContainer.insertAdjacentElement("afterbegin", _);
            } else {
              const _ = this._createAccessibilityTreeNode();
              this._rowElements.push(_), this._rowContainer.appendChild(_);
            }
            this._rowElements[0].addEventListener("focus", this._topBoundaryFocusListener), this._rowElements[this._rowElements.length - 1].addEventListener("focus", this._bottomBoundaryFocusListener), this._terminal.scrollLines(i === 0 ? -1 : 1), this._rowElements[i === 0 ? 1 : this._rowElements.length - 2].focus(), s.preventDefault(), s.stopImmediatePropagation();
          }
          _handleSelectionChange() {
            var v, C;
            if (this._rowElements.length === 0) return;
            const s = document.getSelection();
            if (!s) return;
            if (s.isCollapsed) return void (this._rowContainer.contains(s.anchorNode) && this._terminal.clearSelection());
            if (!s.anchorNode || !s.focusNode) return void console.error("anchorNode and/or focusNode are null");
            let i = { node: s.anchorNode, offset: s.anchorOffset }, u = { node: s.focusNode, offset: s.focusOffset };
            if ((i.node.compareDocumentPosition(u.node) & Node.DOCUMENT_POSITION_PRECEDING || i.node === u.node && i.offset > u.offset) && ([i, u] = [u, i]), i.node.compareDocumentPosition(this._rowElements[0]) & (Node.DOCUMENT_POSITION_CONTAINED_BY | Node.DOCUMENT_POSITION_FOLLOWING) && (i = { node: this._rowElements[0].childNodes[0], offset: 0 }), !this._rowContainer.contains(i.node)) return;
            const p = this._rowElements.slice(-1)[0];
            if (u.node.compareDocumentPosition(p) & (Node.DOCUMENT_POSITION_CONTAINED_BY | Node.DOCUMENT_POSITION_PRECEDING) && (u = { node: p, offset: (C = (v = p.textContent) == null ? void 0 : v.length) != null ? C : 0 }), !this._rowContainer.contains(u.node)) return;
            const l = ({ node: w, offset: S }) => {
              const b = w instanceof Text ? w.parentNode : w;
              let x = parseInt(b == null ? void 0 : b.getAttribute("aria-posinset"), 10) - 1;
              if (isNaN(x)) return console.warn("row is invalid. Race condition?"), null;
              const A = this._rowColumns.get(b);
              if (!A) return console.warn("columns is null. Race condition?"), null;
              let P = S < A.length ? A[S] : A.slice(-1)[0] + 1;
              return P >= this._terminal.cols && (++x, P = 0), { row: x, column: P };
            }, m = l(i), _ = l(u);
            if (m && _) {
              if (m.row > _.row || m.row === _.row && m.column >= _.column) throw new Error("invalid range");
              this._terminal.select(m.column, m.row, (_.row - m.row) * this._terminal.cols - m.column + _.column);
            }
          }
          _handleResize(s) {
            this._rowElements[this._rowElements.length - 1].removeEventListener("focus", this._bottomBoundaryFocusListener);
            for (let i = this._rowContainer.children.length; i < this._terminal.rows; i++) this._rowElements[i] = this._createAccessibilityTreeNode(), this._rowContainer.appendChild(this._rowElements[i]);
            for (; this._rowElements.length > s; ) this._rowContainer.removeChild(this._rowElements.pop());
            this._rowElements[this._rowElements.length - 1].addEventListener("focus", this._bottomBoundaryFocusListener), this._refreshRowsDimensions();
          }
          _createAccessibilityTreeNode() {
            const s = this._coreBrowserService.mainDocument.createElement("div");
            return s.setAttribute("role", "listitem"), s.tabIndex = -1, this._refreshRowDimensions(s), s;
          }
          _refreshRowsDimensions() {
            if (this._renderService.dimensions.css.cell.height) {
              this._accessibilityContainer.style.width = `${this._renderService.dimensions.css.canvas.width}px`, this._rowElements.length !== this._terminal.rows && this._handleResize(this._terminal.rows);
              for (let s = 0; s < this._terminal.rows; s++) this._refreshRowDimensions(this._rowElements[s]);
            }
          }
          _refreshRowDimensions(s) {
            s.style.height = `${this._renderService.dimensions.css.cell.height}px`;
          }
        };
        t.AccessibilityManager = o = c([h(1, n.IInstantiationService), h(2, g.ICoreBrowserService), h(3, g.IRenderService)], o);
      }, 3614: (T, t) => {
        function a(d) {
          return d.replace(/\r?\n/g, "\r");
        }
        function c(d, f) {
          return f ? "\x1B[200~" + d + "\x1B[201~" : d;
        }
        function h(d, f, g, n) {
          d = c(d = a(d), g.decPrivateModes.bracketedPasteMode && n.rawOptions.ignoreBracketedPasteMode !== !0), g.triggerDataEvent(d, !0), f.value = "";
        }
        function r(d, f, g) {
          const n = g.getBoundingClientRect(), e = d.clientX - n.left - 10, o = d.clientY - n.top - 10;
          f.style.width = "20px", f.style.height = "20px", f.style.left = `${e}px`, f.style.top = `${o}px`, f.style.zIndex = "1000", f.focus();
        }
        Object.defineProperty(t, "__esModule", { value: !0 }), t.rightClickHandler = t.moveTextAreaUnderMouseCursor = t.paste = t.handlePasteEvent = t.copyHandler = t.bracketTextForPaste = t.prepareTextForTerminal = void 0, t.prepareTextForTerminal = a, t.bracketTextForPaste = c, t.copyHandler = function(d, f) {
          d.clipboardData && d.clipboardData.setData("text/plain", f.selectionText), d.preventDefault();
        }, t.handlePasteEvent = function(d, f, g, n) {
          d.stopPropagation(), d.clipboardData && h(d.clipboardData.getData("text/plain"), f, g, n);
        }, t.paste = h, t.moveTextAreaUnderMouseCursor = r, t.rightClickHandler = function(d, f, g, n, e) {
          r(d, f, g), e && n.rightClickSelect(d), f.value = n.selectionText, f.select();
        };
      }, 7239: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.ColorContrastCache = void 0;
        const c = a(1505);
        t.ColorContrastCache = class {
          constructor() {
            this._color = new c.TwoKeyMap(), this._css = new c.TwoKeyMap();
          }
          setCss(h, r, d) {
            this._css.set(h, r, d);
          }
          getCss(h, r) {
            return this._css.get(h, r);
          }
          setColor(h, r, d) {
            this._color.set(h, r, d);
          }
          getColor(h, r) {
            return this._color.get(h, r);
          }
          clear() {
            this._color.clear(), this._css.clear();
          }
        };
      }, 3656: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.addDisposableDomListener = void 0, t.addDisposableDomListener = function(a, c, h, r) {
          a.addEventListener(c, h, r);
          let d = !1;
          return { dispose: () => {
            d || (d = !0, a.removeEventListener(c, h, r));
          } };
        };
      }, 3551: function(T, t, a) {
        var c = this && this.__decorate || function(o, s, i, u) {
          var p, l = arguments.length, m = l < 3 ? s : u === null ? u = Object.getOwnPropertyDescriptor(s, i) : u;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") m = Reflect.decorate(o, s, i, u);
          else for (var _ = o.length - 1; _ >= 0; _--) (p = o[_]) && (m = (l < 3 ? p(m) : l > 3 ? p(s, i, m) : p(s, i)) || m);
          return l > 3 && m && Object.defineProperty(s, i, m), m;
        }, h = this && this.__param || function(o, s) {
          return function(i, u) {
            s(i, u, o);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.Linkifier = void 0;
        const r = a(3656), d = a(8460), f = a(844), g = a(2585), n = a(4725);
        let e = t.Linkifier = class extends f.Disposable {
          get currentLink() {
            return this._currentLink;
          }
          constructor(o, s, i, u, p) {
            super(), this._element = o, this._mouseService = s, this._renderService = i, this._bufferService = u, this._linkProviderService = p, this._linkCacheDisposables = [], this._isMouseOut = !0, this._wasResized = !1, this._activeLine = -1, this._onShowLinkUnderline = this.register(new d.EventEmitter()), this.onShowLinkUnderline = this._onShowLinkUnderline.event, this._onHideLinkUnderline = this.register(new d.EventEmitter()), this.onHideLinkUnderline = this._onHideLinkUnderline.event, this.register((0, f.getDisposeArrayDisposable)(this._linkCacheDisposables)), this.register((0, f.toDisposable)((() => {
              var l;
              this._lastMouseEvent = void 0, (l = this._activeProviderReplies) == null || l.clear();
            }))), this.register(this._bufferService.onResize((() => {
              this._clearCurrentLink(), this._wasResized = !0;
            }))), this.register((0, r.addDisposableDomListener)(this._element, "mouseleave", (() => {
              this._isMouseOut = !0, this._clearCurrentLink();
            }))), this.register((0, r.addDisposableDomListener)(this._element, "mousemove", this._handleMouseMove.bind(this))), this.register((0, r.addDisposableDomListener)(this._element, "mousedown", this._handleMouseDown.bind(this))), this.register((0, r.addDisposableDomListener)(this._element, "mouseup", this._handleMouseUp.bind(this)));
          }
          _handleMouseMove(o) {
            this._lastMouseEvent = o;
            const s = this._positionFromMouseEvent(o, this._element, this._mouseService);
            if (!s) return;
            this._isMouseOut = !1;
            const i = o.composedPath();
            for (let u = 0; u < i.length; u++) {
              const p = i[u];
              if (p.classList.contains("xterm")) break;
              if (p.classList.contains("xterm-hover")) return;
            }
            this._lastBufferCell && s.x === this._lastBufferCell.x && s.y === this._lastBufferCell.y || (this._handleHover(s), this._lastBufferCell = s);
          }
          _handleHover(o) {
            if (this._activeLine !== o.y || this._wasResized) return this._clearCurrentLink(), this._askForLink(o, !1), void (this._wasResized = !1);
            this._currentLink && this._linkAtPosition(this._currentLink.link, o) || (this._clearCurrentLink(), this._askForLink(o, !0));
          }
          _askForLink(o, s) {
            var u, p;
            this._activeProviderReplies && s || ((u = this._activeProviderReplies) == null || u.forEach(((l) => {
              l == null || l.forEach(((m) => {
                m.link.dispose && m.link.dispose();
              }));
            })), this._activeProviderReplies = /* @__PURE__ */ new Map(), this._activeLine = o.y);
            let i = !1;
            for (const [l, m] of this._linkProviderService.linkProviders.entries()) s ? (p = this._activeProviderReplies) != null && p.get(l) && (i = this._checkLinkProviderResult(l, o, i)) : m.provideLinks(o.y, ((_) => {
              var C, w;
              if (this._isMouseOut) return;
              const v = _ == null ? void 0 : _.map(((S) => ({ link: S })));
              (C = this._activeProviderReplies) == null || C.set(l, v), i = this._checkLinkProviderResult(l, o, i), ((w = this._activeProviderReplies) == null ? void 0 : w.size) === this._linkProviderService.linkProviders.length && this._removeIntersectingLinks(o.y, this._activeProviderReplies);
            }));
          }
          _removeIntersectingLinks(o, s) {
            const i = /* @__PURE__ */ new Set();
            for (let u = 0; u < s.size; u++) {
              const p = s.get(u);
              if (p) for (let l = 0; l < p.length; l++) {
                const m = p[l], _ = m.link.range.start.y < o ? 0 : m.link.range.start.x, v = m.link.range.end.y > o ? this._bufferService.cols : m.link.range.end.x;
                for (let C = _; C <= v; C++) {
                  if (i.has(C)) {
                    p.splice(l--, 1);
                    break;
                  }
                  i.add(C);
                }
              }
            }
          }
          _checkLinkProviderResult(o, s, i) {
            var l;
            if (!this._activeProviderReplies) return i;
            const u = this._activeProviderReplies.get(o);
            let p = !1;
            for (let m = 0; m < o; m++) this._activeProviderReplies.has(m) && !this._activeProviderReplies.get(m) || (p = !0);
            if (!p && u) {
              const m = u.find(((_) => this._linkAtPosition(_.link, s)));
              m && (i = !0, this._handleNewLink(m));
            }
            if (this._activeProviderReplies.size === this._linkProviderService.linkProviders.length && !i) for (let m = 0; m < this._activeProviderReplies.size; m++) {
              const _ = (l = this._activeProviderReplies.get(m)) == null ? void 0 : l.find(((v) => this._linkAtPosition(v.link, s)));
              if (_) {
                i = !0, this._handleNewLink(_);
                break;
              }
            }
            return i;
          }
          _handleMouseDown() {
            this._mouseDownLink = this._currentLink;
          }
          _handleMouseUp(o) {
            if (!this._currentLink) return;
            const s = this._positionFromMouseEvent(o, this._element, this._mouseService);
            s && this._mouseDownLink === this._currentLink && this._linkAtPosition(this._currentLink.link, s) && this._currentLink.link.activate(o, this._currentLink.link.text);
          }
          _clearCurrentLink(o, s) {
            this._currentLink && this._lastMouseEvent && (!o || !s || this._currentLink.link.range.start.y >= o && this._currentLink.link.range.end.y <= s) && (this._linkLeave(this._element, this._currentLink.link, this._lastMouseEvent), this._currentLink = void 0, (0, f.disposeArray)(this._linkCacheDisposables));
          }
          _handleNewLink(o) {
            if (!this._lastMouseEvent) return;
            const s = this._positionFromMouseEvent(this._lastMouseEvent, this._element, this._mouseService);
            s && this._linkAtPosition(o.link, s) && (this._currentLink = o, this._currentLink.state = { decorations: { underline: o.link.decorations === void 0 || o.link.decorations.underline, pointerCursor: o.link.decorations === void 0 || o.link.decorations.pointerCursor }, isHovered: !0 }, this._linkHover(this._element, o.link, this._lastMouseEvent), o.link.decorations = {}, Object.defineProperties(o.link.decorations, { pointerCursor: { get: () => {
              var i, u;
              return (u = (i = this._currentLink) == null ? void 0 : i.state) == null ? void 0 : u.decorations.pointerCursor;
            }, set: (i) => {
              var u;
              (u = this._currentLink) != null && u.state && this._currentLink.state.decorations.pointerCursor !== i && (this._currentLink.state.decorations.pointerCursor = i, this._currentLink.state.isHovered && this._element.classList.toggle("xterm-cursor-pointer", i));
            } }, underline: { get: () => {
              var i, u;
              return (u = (i = this._currentLink) == null ? void 0 : i.state) == null ? void 0 : u.decorations.underline;
            }, set: (i) => {
              var u, p, l;
              (u = this._currentLink) != null && u.state && ((l = (p = this._currentLink) == null ? void 0 : p.state) == null ? void 0 : l.decorations.underline) !== i && (this._currentLink.state.decorations.underline = i, this._currentLink.state.isHovered && this._fireUnderlineEvent(o.link, i));
            } } }), this._linkCacheDisposables.push(this._renderService.onRenderedViewportChange(((i) => {
              if (!this._currentLink) return;
              const u = i.start === 0 ? 0 : i.start + 1 + this._bufferService.buffer.ydisp, p = this._bufferService.buffer.ydisp + 1 + i.end;
              if (this._currentLink.link.range.start.y >= u && this._currentLink.link.range.end.y <= p && (this._clearCurrentLink(u, p), this._lastMouseEvent)) {
                const l = this._positionFromMouseEvent(this._lastMouseEvent, this._element, this._mouseService);
                l && this._askForLink(l, !1);
              }
            }))));
          }
          _linkHover(o, s, i) {
            var u;
            (u = this._currentLink) != null && u.state && (this._currentLink.state.isHovered = !0, this._currentLink.state.decorations.underline && this._fireUnderlineEvent(s, !0), this._currentLink.state.decorations.pointerCursor && o.classList.add("xterm-cursor-pointer")), s.hover && s.hover(i, s.text);
          }
          _fireUnderlineEvent(o, s) {
            const i = o.range, u = this._bufferService.buffer.ydisp, p = this._createLinkUnderlineEvent(i.start.x - 1, i.start.y - u - 1, i.end.x, i.end.y - u - 1, void 0);
            (s ? this._onShowLinkUnderline : this._onHideLinkUnderline).fire(p);
          }
          _linkLeave(o, s, i) {
            var u;
            (u = this._currentLink) != null && u.state && (this._currentLink.state.isHovered = !1, this._currentLink.state.decorations.underline && this._fireUnderlineEvent(s, !1), this._currentLink.state.decorations.pointerCursor && o.classList.remove("xterm-cursor-pointer")), s.leave && s.leave(i, s.text);
          }
          _linkAtPosition(o, s) {
            const i = o.range.start.y * this._bufferService.cols + o.range.start.x, u = o.range.end.y * this._bufferService.cols + o.range.end.x, p = s.y * this._bufferService.cols + s.x;
            return i <= p && p <= u;
          }
          _positionFromMouseEvent(o, s, i) {
            const u = i.getCoords(o, s, this._bufferService.cols, this._bufferService.rows);
            if (u) return { x: u[0], y: u[1] + this._bufferService.buffer.ydisp };
          }
          _createLinkUnderlineEvent(o, s, i, u, p) {
            return { x1: o, y1: s, x2: i, y2: u, cols: this._bufferService.cols, fg: p };
          }
        };
        t.Linkifier = e = c([h(1, n.IMouseService), h(2, n.IRenderService), h(3, g.IBufferService), h(4, n.ILinkProviderService)], e);
      }, 9042: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.tooMuchOutput = t.promptLabel = void 0, t.promptLabel = "Terminal input", t.tooMuchOutput = "Too much output to announce, navigate to rows manually to read";
      }, 3730: function(T, t, a) {
        var c = this && this.__decorate || function(n, e, o, s) {
          var i, u = arguments.length, p = u < 3 ? e : s === null ? s = Object.getOwnPropertyDescriptor(e, o) : s;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") p = Reflect.decorate(n, e, o, s);
          else for (var l = n.length - 1; l >= 0; l--) (i = n[l]) && (p = (u < 3 ? i(p) : u > 3 ? i(e, o, p) : i(e, o)) || p);
          return u > 3 && p && Object.defineProperty(e, o, p), p;
        }, h = this && this.__param || function(n, e) {
          return function(o, s) {
            e(o, s, n);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.OscLinkProvider = void 0;
        const r = a(511), d = a(2585);
        let f = t.OscLinkProvider = class {
          constructor(n, e, o) {
            this._bufferService = n, this._optionsService = e, this._oscLinkService = o;
          }
          provideLinks(n, e) {
            var v;
            const o = this._bufferService.buffer.lines.get(n - 1);
            if (!o) return void e(void 0);
            const s = [], i = this._optionsService.rawOptions.linkHandler, u = new r.CellData(), p = o.getTrimmedLength();
            let l = -1, m = -1, _ = !1;
            for (let C = 0; C < p; C++) if (m !== -1 || o.hasContent(C)) {
              if (o.loadCell(C, u), u.hasExtendedAttrs() && u.extended.urlId) {
                if (m === -1) {
                  m = C, l = u.extended.urlId;
                  continue;
                }
                _ = u.extended.urlId !== l;
              } else m !== -1 && (_ = !0);
              if (_ || m !== -1 && C === p - 1) {
                const w = (v = this._oscLinkService.getLinkData(l)) == null ? void 0 : v.uri;
                if (w) {
                  const S = { start: { x: m + 1, y: n }, end: { x: C + (_ || C !== p - 1 ? 0 : 1), y: n } };
                  let b = !1;
                  if (!(i != null && i.allowNonHttpProtocols)) try {
                    const x = new URL(w);
                    ["http:", "https:"].includes(x.protocol) || (b = !0);
                  } catch (x) {
                    b = !0;
                  }
                  b || s.push({ text: w, range: S, activate: (x, A) => i ? i.activate(x, A, S) : g(0, A), hover: (x, A) => {
                    var P;
                    return (P = i == null ? void 0 : i.hover) == null ? void 0 : P.call(i, x, A, S);
                  }, leave: (x, A) => {
                    var P;
                    return (P = i == null ? void 0 : i.leave) == null ? void 0 : P.call(i, x, A, S);
                  } });
                }
                _ = !1, u.hasExtendedAttrs() && u.extended.urlId ? (m = C, l = u.extended.urlId) : (m = -1, l = -1);
              }
            }
            e(s);
          }
        };
        function g(n, e) {
          if (confirm(`Do you want to navigate to ${e}?

WARNING: This link could potentially be dangerous`)) {
            const o = window.open();
            if (o) {
              try {
                o.opener = null;
              } catch (s) {
              }
              o.location.href = e;
            } else console.warn("Opening link blocked as opener could not be cleared");
          }
        }
        t.OscLinkProvider = f = c([h(0, d.IBufferService), h(1, d.IOptionsService), h(2, d.IOscLinkService)], f);
      }, 6193: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.RenderDebouncer = void 0, t.RenderDebouncer = class {
          constructor(a, c) {
            this._renderCallback = a, this._coreBrowserService = c, this._refreshCallbacks = [];
          }
          dispose() {
            this._animationFrame && (this._coreBrowserService.window.cancelAnimationFrame(this._animationFrame), this._animationFrame = void 0);
          }
          addRefreshCallback(a) {
            return this._refreshCallbacks.push(a), this._animationFrame || (this._animationFrame = this._coreBrowserService.window.requestAnimationFrame((() => this._innerRefresh()))), this._animationFrame;
          }
          refresh(a, c, h) {
            this._rowCount = h, a = a !== void 0 ? a : 0, c = c !== void 0 ? c : this._rowCount - 1, this._rowStart = this._rowStart !== void 0 ? Math.min(this._rowStart, a) : a, this._rowEnd = this._rowEnd !== void 0 ? Math.max(this._rowEnd, c) : c, this._animationFrame || (this._animationFrame = this._coreBrowserService.window.requestAnimationFrame((() => this._innerRefresh())));
          }
          _innerRefresh() {
            if (this._animationFrame = void 0, this._rowStart === void 0 || this._rowEnd === void 0 || this._rowCount === void 0) return void this._runRefreshCallbacks();
            const a = Math.max(this._rowStart, 0), c = Math.min(this._rowEnd, this._rowCount - 1);
            this._rowStart = void 0, this._rowEnd = void 0, this._renderCallback(a, c), this._runRefreshCallbacks();
          }
          _runRefreshCallbacks() {
            for (const a of this._refreshCallbacks) a(0);
            this._refreshCallbacks = [];
          }
        };
      }, 3236: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.Terminal = void 0;
        const c = a(3614), h = a(3656), r = a(3551), d = a(9042), f = a(3730), g = a(1680), n = a(3107), e = a(5744), o = a(2950), s = a(1296), i = a(428), u = a(4269), p = a(5114), l = a(8934), m = a(3230), _ = a(9312), v = a(4725), C = a(6731), w = a(8055), S = a(8969), b = a(8460), x = a(844), A = a(6114), P = a(8437), k = a(2584), M = a(7399), y = a(5941), L = a(9074), R = a(2585), D = a(5435), F = a(4567), U = a(779);
        class K extends S.CoreTerminal {
          get onFocus() {
            return this._onFocus.event;
          }
          get onBlur() {
            return this._onBlur.event;
          }
          get onA11yChar() {
            return this._onA11yCharEmitter.event;
          }
          get onA11yTab() {
            return this._onA11yTabEmitter.event;
          }
          get onWillOpen() {
            return this._onWillOpen.event;
          }
          constructor(O = {}) {
            super(O), this.browser = A, this._keyDownHandled = !1, this._keyDownSeen = !1, this._keyPressHandled = !1, this._unprocessedDeadKey = !1, this._accessibilityManager = this.register(new x.MutableDisposable()), this._onCursorMove = this.register(new b.EventEmitter()), this.onCursorMove = this._onCursorMove.event, this._onKey = this.register(new b.EventEmitter()), this.onKey = this._onKey.event, this._onRender = this.register(new b.EventEmitter()), this.onRender = this._onRender.event, this._onSelectionChange = this.register(new b.EventEmitter()), this.onSelectionChange = this._onSelectionChange.event, this._onTitleChange = this.register(new b.EventEmitter()), this.onTitleChange = this._onTitleChange.event, this._onBell = this.register(new b.EventEmitter()), this.onBell = this._onBell.event, this._onFocus = this.register(new b.EventEmitter()), this._onBlur = this.register(new b.EventEmitter()), this._onA11yCharEmitter = this.register(new b.EventEmitter()), this._onA11yTabEmitter = this.register(new b.EventEmitter()), this._onWillOpen = this.register(new b.EventEmitter()), this._setup(), this._decorationService = this._instantiationService.createInstance(L.DecorationService), this._instantiationService.setService(R.IDecorationService, this._decorationService), this._linkProviderService = this._instantiationService.createInstance(U.LinkProviderService), this._instantiationService.setService(v.ILinkProviderService, this._linkProviderService), this._linkProviderService.registerLinkProvider(this._instantiationService.createInstance(f.OscLinkProvider)), this.register(this._inputHandler.onRequestBell((() => this._onBell.fire()))), this.register(this._inputHandler.onRequestRefreshRows(((E, H) => this.refresh(E, H)))), this.register(this._inputHandler.onRequestSendFocus((() => this._reportFocus()))), this.register(this._inputHandler.onRequestReset((() => this.reset()))), this.register(this._inputHandler.onRequestWindowsOptionsReport(((E) => this._reportWindowsOptions(E)))), this.register(this._inputHandler.onColor(((E) => this._handleColorEvent(E)))), this.register((0, b.forwardEvent)(this._inputHandler.onCursorMove, this._onCursorMove)), this.register((0, b.forwardEvent)(this._inputHandler.onTitleChange, this._onTitleChange)), this.register((0, b.forwardEvent)(this._inputHandler.onA11yChar, this._onA11yCharEmitter)), this.register((0, b.forwardEvent)(this._inputHandler.onA11yTab, this._onA11yTabEmitter)), this.register(this._bufferService.onResize(((E) => this._afterResize(E.cols, E.rows)))), this.register((0, x.toDisposable)((() => {
              var E, H;
              this._customKeyEventHandler = void 0, (H = (E = this.element) == null ? void 0 : E.parentNode) == null || H.removeChild(this.element);
            })));
          }
          _handleColorEvent(O) {
            if (this._themeService) for (const E of O) {
              let H, N = "";
              switch (E.index) {
                case 256:
                  H = "foreground", N = "10";
                  break;
                case 257:
                  H = "background", N = "11";
                  break;
                case 258:
                  H = "cursor", N = "12";
                  break;
                default:
                  H = "ansi", N = "4;" + E.index;
              }
              switch (E.type) {
                case 0:
                  const G = w.color.toColorRGB(H === "ansi" ? this._themeService.colors.ansi[E.index] : this._themeService.colors[H]);
                  this.coreService.triggerDataEvent(`${k.C0.ESC}]${N};${(0, y.toRgbString)(G)}${k.C1_ESCAPED.ST}`);
                  break;
                case 1:
                  if (H === "ansi") this._themeService.modifyColors(((j) => j.ansi[E.index] = w.channels.toColor(...E.color)));
                  else {
                    const j = H;
                    this._themeService.modifyColors(((ie) => ie[j] = w.channels.toColor(...E.color)));
                  }
                  break;
                case 2:
                  this._themeService.restoreColor(E.index);
              }
            }
          }
          _setup() {
            super._setup(), this._customKeyEventHandler = void 0;
          }
          get buffer() {
            return this.buffers.active;
          }
          focus() {
            this.textarea && this.textarea.focus({ preventScroll: !0 });
          }
          _handleScreenReaderModeOptionChange(O) {
            O ? !this._accessibilityManager.value && this._renderService && (this._accessibilityManager.value = this._instantiationService.createInstance(F.AccessibilityManager, this)) : this._accessibilityManager.clear();
          }
          _handleTextAreaFocus(O) {
            this.coreService.decPrivateModes.sendFocus && this.coreService.triggerDataEvent(k.C0.ESC + "[I"), this.element.classList.add("focus"), this._showCursor(), this._onFocus.fire();
          }
          blur() {
            var O;
            return (O = this.textarea) == null ? void 0 : O.blur();
          }
          _handleTextAreaBlur() {
            this.textarea.value = "", this.refresh(this.buffer.y, this.buffer.y), this.coreService.decPrivateModes.sendFocus && this.coreService.triggerDataEvent(k.C0.ESC + "[O"), this.element.classList.remove("focus"), this._onBlur.fire();
          }
          _syncTextArea() {
            if (!this.textarea || !this.buffer.isCursorInViewport || this._compositionHelper.isComposing || !this._renderService) return;
            const O = this.buffer.ybase + this.buffer.y, E = this.buffer.lines.get(O);
            if (!E) return;
            const H = Math.min(this.buffer.x, this.cols - 1), N = this._renderService.dimensions.css.cell.height, G = E.getWidth(H), j = this._renderService.dimensions.css.cell.width * G, ie = this.buffer.y * this._renderService.dimensions.css.cell.height, V = H * this._renderService.dimensions.css.cell.width;
            this.textarea.style.left = V + "px", this.textarea.style.top = ie + "px", this.textarea.style.width = j + "px", this.textarea.style.height = N + "px", this.textarea.style.lineHeight = N + "px", this.textarea.style.zIndex = "-5";
          }
          _initGlobal() {
            this._bindKeys(), this.register((0, h.addDisposableDomListener)(this.element, "copy", ((E) => {
              this.hasSelection() && (0, c.copyHandler)(E, this._selectionService);
            })));
            const O = (E) => (0, c.handlePasteEvent)(E, this.textarea, this.coreService, this.optionsService);
            this.register((0, h.addDisposableDomListener)(this.textarea, "paste", O)), this.register((0, h.addDisposableDomListener)(this.element, "paste", O)), A.isFirefox ? this.register((0, h.addDisposableDomListener)(this.element, "mousedown", ((E) => {
              E.button === 2 && (0, c.rightClickHandler)(E, this.textarea, this.screenElement, this._selectionService, this.options.rightClickSelectsWord);
            }))) : this.register((0, h.addDisposableDomListener)(this.element, "contextmenu", ((E) => {
              (0, c.rightClickHandler)(E, this.textarea, this.screenElement, this._selectionService, this.options.rightClickSelectsWord);
            }))), A.isLinux && this.register((0, h.addDisposableDomListener)(this.element, "auxclick", ((E) => {
              E.button === 1 && (0, c.moveTextAreaUnderMouseCursor)(E, this.textarea, this.screenElement);
            })));
          }
          _bindKeys() {
            this.register((0, h.addDisposableDomListener)(this.textarea, "keyup", ((O) => this._keyUp(O)), !0)), this.register((0, h.addDisposableDomListener)(this.textarea, "keydown", ((O) => this._keyDown(O)), !0)), this.register((0, h.addDisposableDomListener)(this.textarea, "keypress", ((O) => this._keyPress(O)), !0)), this.register((0, h.addDisposableDomListener)(this.textarea, "compositionstart", (() => this._compositionHelper.compositionstart()))), this.register((0, h.addDisposableDomListener)(this.textarea, "compositionupdate", ((O) => this._compositionHelper.compositionupdate(O)))), this.register((0, h.addDisposableDomListener)(this.textarea, "compositionend", (() => this._compositionHelper.compositionend()))), this.register((0, h.addDisposableDomListener)(this.textarea, "input", ((O) => this._inputEvent(O)), !0)), this.register(this.onRender((() => this._compositionHelper.updateCompositionElements())));
          }
          open(O) {
            var H, N, G;
            if (!O) throw new Error("Terminal requires a parent element.");
            if (O.isConnected || this._logService.debug("Terminal.open was called on an element that was not attached to the DOM"), ((H = this.element) == null ? void 0 : H.ownerDocument.defaultView) && this._coreBrowserService) return void (this.element.ownerDocument.defaultView !== this._coreBrowserService.window && (this._coreBrowserService.window = this.element.ownerDocument.defaultView));
            this._document = O.ownerDocument, this.options.documentOverride && this.options.documentOverride instanceof Document && (this._document = this.optionsService.rawOptions.documentOverride), this.element = this._document.createElement("div"), this.element.dir = "ltr", this.element.classList.add("terminal"), this.element.classList.add("xterm"), O.appendChild(this.element);
            const E = this._document.createDocumentFragment();
            this._viewportElement = this._document.createElement("div"), this._viewportElement.classList.add("xterm-viewport"), E.appendChild(this._viewportElement), this._viewportScrollArea = this._document.createElement("div"), this._viewportScrollArea.classList.add("xterm-scroll-area"), this._viewportElement.appendChild(this._viewportScrollArea), this.screenElement = this._document.createElement("div"), this.screenElement.classList.add("xterm-screen"), this.register((0, h.addDisposableDomListener)(this.screenElement, "mousemove", ((j) => this.updateCursorStyle(j)))), this._helperContainer = this._document.createElement("div"), this._helperContainer.classList.add("xterm-helpers"), this.screenElement.appendChild(this._helperContainer), E.appendChild(this.screenElement), this.textarea = this._document.createElement("textarea"), this.textarea.classList.add("xterm-helper-textarea"), this.textarea.setAttribute("aria-label", d.promptLabel), A.isChromeOS || this.textarea.setAttribute("aria-multiline", "false"), this.textarea.setAttribute("autocorrect", "off"), this.textarea.setAttribute("autocapitalize", "off"), this.textarea.setAttribute("spellcheck", "false"), this.textarea.tabIndex = 0, this._coreBrowserService = this.register(this._instantiationService.createInstance(p.CoreBrowserService, this.textarea, (N = O.ownerDocument.defaultView) != null ? N : window, ((G = this._document) != null ? G : typeof window != "undefined") ? window.document : null)), this._instantiationService.setService(v.ICoreBrowserService, this._coreBrowserService), this.register((0, h.addDisposableDomListener)(this.textarea, "focus", ((j) => this._handleTextAreaFocus(j)))), this.register((0, h.addDisposableDomListener)(this.textarea, "blur", (() => this._handleTextAreaBlur()))), this._helperContainer.appendChild(this.textarea), this._charSizeService = this._instantiationService.createInstance(i.CharSizeService, this._document, this._helperContainer), this._instantiationService.setService(v.ICharSizeService, this._charSizeService), this._themeService = this._instantiationService.createInstance(C.ThemeService), this._instantiationService.setService(v.IThemeService, this._themeService), this._characterJoinerService = this._instantiationService.createInstance(u.CharacterJoinerService), this._instantiationService.setService(v.ICharacterJoinerService, this._characterJoinerService), this._renderService = this.register(this._instantiationService.createInstance(m.RenderService, this.rows, this.screenElement)), this._instantiationService.setService(v.IRenderService, this._renderService), this.register(this._renderService.onRenderedViewportChange(((j) => this._onRender.fire(j)))), this.onResize(((j) => this._renderService.resize(j.cols, j.rows))), this._compositionView = this._document.createElement("div"), this._compositionView.classList.add("composition-view"), this._compositionHelper = this._instantiationService.createInstance(o.CompositionHelper, this.textarea, this._compositionView), this._helperContainer.appendChild(this._compositionView), this._mouseService = this._instantiationService.createInstance(l.MouseService), this._instantiationService.setService(v.IMouseService, this._mouseService), this.linkifier = this.register(this._instantiationService.createInstance(r.Linkifier, this.screenElement)), this.element.appendChild(E);
            try {
              this._onWillOpen.fire(this.element);
            } catch (j) {
            }
            this._renderService.hasRenderer() || this._renderService.setRenderer(this._createRenderer()), this.viewport = this._instantiationService.createInstance(g.Viewport, this._viewportElement, this._viewportScrollArea), this.viewport.onRequestScrollLines(((j) => this.scrollLines(j.amount, j.suppressScrollEvent, 1))), this.register(this._inputHandler.onRequestSyncScrollBar((() => this.viewport.syncScrollArea()))), this.register(this.viewport), this.register(this.onCursorMove((() => {
              this._renderService.handleCursorMove(), this._syncTextArea();
            }))), this.register(this.onResize((() => this._renderService.handleResize(this.cols, this.rows)))), this.register(this.onBlur((() => this._renderService.handleBlur()))), this.register(this.onFocus((() => this._renderService.handleFocus()))), this.register(this._renderService.onDimensionsChange((() => this.viewport.syncScrollArea()))), this._selectionService = this.register(this._instantiationService.createInstance(_.SelectionService, this.element, this.screenElement, this.linkifier)), this._instantiationService.setService(v.ISelectionService, this._selectionService), this.register(this._selectionService.onRequestScrollLines(((j) => this.scrollLines(j.amount, j.suppressScrollEvent)))), this.register(this._selectionService.onSelectionChange((() => this._onSelectionChange.fire()))), this.register(this._selectionService.onRequestRedraw(((j) => this._renderService.handleSelectionChanged(j.start, j.end, j.columnSelectMode)))), this.register(this._selectionService.onLinuxMouseSelection(((j) => {
              this.textarea.value = j, this.textarea.focus(), this.textarea.select();
            }))), this.register(this._onScroll.event(((j) => {
              this.viewport.syncScrollArea(), this._selectionService.refresh();
            }))), this.register((0, h.addDisposableDomListener)(this._viewportElement, "scroll", (() => this._selectionService.refresh()))), this.register(this._instantiationService.createInstance(n.BufferDecorationRenderer, this.screenElement)), this.register((0, h.addDisposableDomListener)(this.element, "mousedown", ((j) => this._selectionService.handleMouseDown(j)))), this.coreMouseService.areMouseEventsActive ? (this._selectionService.disable(), this.element.classList.add("enable-mouse-events")) : this._selectionService.enable(), this.options.screenReaderMode && (this._accessibilityManager.value = this._instantiationService.createInstance(F.AccessibilityManager, this)), this.register(this.optionsService.onSpecificOptionChange("screenReaderMode", ((j) => this._handleScreenReaderModeOptionChange(j)))), this.options.overviewRulerWidth && (this._overviewRulerRenderer = this.register(this._instantiationService.createInstance(e.OverviewRulerRenderer, this._viewportElement, this.screenElement))), this.optionsService.onSpecificOptionChange("overviewRulerWidth", ((j) => {
              !this._overviewRulerRenderer && j && this._viewportElement && this.screenElement && (this._overviewRulerRenderer = this.register(this._instantiationService.createInstance(e.OverviewRulerRenderer, this._viewportElement, this.screenElement)));
            })), this._charSizeService.measure(), this.refresh(0, this.rows - 1), this._initGlobal(), this.bindMouse();
          }
          _createRenderer() {
            return this._instantiationService.createInstance(s.DomRenderer, this, this._document, this.element, this.screenElement, this._viewportElement, this._helperContainer, this.linkifier);
          }
          bindMouse() {
            const O = this, E = this.element;
            function H(j) {
              const ie = O._mouseService.getMouseReportCoords(j, O.screenElement);
              if (!ie) return !1;
              let V, ae;
              switch (j.overrideType || j.type) {
                case "mousemove":
                  ae = 32, j.buttons === void 0 ? (V = 3, j.button !== void 0 && (V = j.button < 3 ? j.button : 3)) : V = 1 & j.buttons ? 0 : 4 & j.buttons ? 1 : 2 & j.buttons ? 2 : 3;
                  break;
                case "mouseup":
                  ae = 0, V = j.button < 3 ? j.button : 3;
                  break;
                case "mousedown":
                  ae = 1, V = j.button < 3 ? j.button : 3;
                  break;
                case "wheel":
                  if (O._customWheelEventHandler && O._customWheelEventHandler(j) === !1 || O.viewport.getLinesScrolled(j) === 0) return !1;
                  ae = j.deltaY < 0 ? 0 : 1, V = 4;
                  break;
                default:
                  return !1;
              }
              return !(ae === void 0 || V === void 0 || V > 4) && O.coreMouseService.triggerMouseEvent({ col: ie.col, row: ie.row, x: ie.x, y: ie.y, button: V, action: ae, ctrl: j.ctrlKey, alt: j.altKey, shift: j.shiftKey });
            }
            const N = { mouseup: null, wheel: null, mousedrag: null, mousemove: null }, G = { mouseup: (j) => (H(j), j.buttons || (this._document.removeEventListener("mouseup", N.mouseup), N.mousedrag && this._document.removeEventListener("mousemove", N.mousedrag)), this.cancel(j)), wheel: (j) => (H(j), this.cancel(j, !0)), mousedrag: (j) => {
              j.buttons && H(j);
            }, mousemove: (j) => {
              j.buttons || H(j);
            } };
            this.register(this.coreMouseService.onProtocolChange(((j) => {
              j ? (this.optionsService.rawOptions.logLevel === "debug" && this._logService.debug("Binding to mouse events:", this.coreMouseService.explainEvents(j)), this.element.classList.add("enable-mouse-events"), this._selectionService.disable()) : (this._logService.debug("Unbinding from mouse events."), this.element.classList.remove("enable-mouse-events"), this._selectionService.enable()), 8 & j ? N.mousemove || (E.addEventListener("mousemove", G.mousemove), N.mousemove = G.mousemove) : (E.removeEventListener("mousemove", N.mousemove), N.mousemove = null), 16 & j ? N.wheel || (E.addEventListener("wheel", G.wheel, { passive: !1 }), N.wheel = G.wheel) : (E.removeEventListener("wheel", N.wheel), N.wheel = null), 2 & j ? N.mouseup || (N.mouseup = G.mouseup) : (this._document.removeEventListener("mouseup", N.mouseup), N.mouseup = null), 4 & j ? N.mousedrag || (N.mousedrag = G.mousedrag) : (this._document.removeEventListener("mousemove", N.mousedrag), N.mousedrag = null);
            }))), this.coreMouseService.activeProtocol = this.coreMouseService.activeProtocol, this.register((0, h.addDisposableDomListener)(E, "mousedown", ((j) => {
              if (j.preventDefault(), this.focus(), this.coreMouseService.areMouseEventsActive && !this._selectionService.shouldForceSelection(j)) return H(j), N.mouseup && this._document.addEventListener("mouseup", N.mouseup), N.mousedrag && this._document.addEventListener("mousemove", N.mousedrag), this.cancel(j);
            }))), this.register((0, h.addDisposableDomListener)(E, "wheel", ((j) => {
              if (!N.wheel) {
                if (this._customWheelEventHandler && this._customWheelEventHandler(j) === !1) return !1;
                if (!this.buffer.hasScrollback) {
                  const ie = this.viewport.getLinesScrolled(j);
                  if (ie === 0) return;
                  const V = k.C0.ESC + (this.coreService.decPrivateModes.applicationCursorKeys ? "O" : "[") + (j.deltaY < 0 ? "A" : "B");
                  let ae = "";
                  for (let ce = 0; ce < Math.abs(ie); ce++) ae += V;
                  return this.coreService.triggerDataEvent(ae, !0), this.cancel(j, !0);
                }
                return this.viewport.handleWheel(j) ? this.cancel(j) : void 0;
              }
            }), { passive: !1 })), this.register((0, h.addDisposableDomListener)(E, "touchstart", ((j) => {
              if (!this.coreMouseService.areMouseEventsActive) return this.viewport.handleTouchStart(j), this.cancel(j);
            }), { passive: !0 })), this.register((0, h.addDisposableDomListener)(E, "touchmove", ((j) => {
              if (!this.coreMouseService.areMouseEventsActive) return this.viewport.handleTouchMove(j) ? void 0 : this.cancel(j);
            }), { passive: !1 }));
          }
          refresh(O, E) {
            var H;
            (H = this._renderService) == null || H.refreshRows(O, E);
          }
          updateCursorStyle(O) {
            var E;
            (E = this._selectionService) != null && E.shouldColumnSelect(O) ? this.element.classList.add("column-select") : this.element.classList.remove("column-select");
          }
          _showCursor() {
            this.coreService.isCursorInitialized || (this.coreService.isCursorInitialized = !0, this.refresh(this.buffer.y, this.buffer.y));
          }
          scrollLines(O, E, H = 0) {
            var N;
            H === 1 ? (super.scrollLines(O, E, H), this.refresh(0, this.rows - 1)) : (N = this.viewport) == null || N.scrollLines(O);
          }
          paste(O) {
            (0, c.paste)(O, this.textarea, this.coreService, this.optionsService);
          }
          attachCustomKeyEventHandler(O) {
            this._customKeyEventHandler = O;
          }
          attachCustomWheelEventHandler(O) {
            this._customWheelEventHandler = O;
          }
          registerLinkProvider(O) {
            return this._linkProviderService.registerLinkProvider(O);
          }
          registerCharacterJoiner(O) {
            if (!this._characterJoinerService) throw new Error("Terminal must be opened first");
            const E = this._characterJoinerService.register(O);
            return this.refresh(0, this.rows - 1), E;
          }
          deregisterCharacterJoiner(O) {
            if (!this._characterJoinerService) throw new Error("Terminal must be opened first");
            this._characterJoinerService.deregister(O) && this.refresh(0, this.rows - 1);
          }
          get markers() {
            return this.buffer.markers;
          }
          registerMarker(O) {
            return this.buffer.addMarker(this.buffer.ybase + this.buffer.y + O);
          }
          registerDecoration(O) {
            return this._decorationService.registerDecoration(O);
          }
          hasSelection() {
            return !!this._selectionService && this._selectionService.hasSelection;
          }
          select(O, E, H) {
            this._selectionService.setSelection(O, E, H);
          }
          getSelection() {
            return this._selectionService ? this._selectionService.selectionText : "";
          }
          getSelectionPosition() {
            if (this._selectionService && this._selectionService.hasSelection) return { start: { x: this._selectionService.selectionStart[0], y: this._selectionService.selectionStart[1] }, end: { x: this._selectionService.selectionEnd[0], y: this._selectionService.selectionEnd[1] } };
          }
          clearSelection() {
            var O;
            (O = this._selectionService) == null || O.clearSelection();
          }
          selectAll() {
            var O;
            (O = this._selectionService) == null || O.selectAll();
          }
          selectLines(O, E) {
            var H;
            (H = this._selectionService) == null || H.selectLines(O, E);
          }
          _keyDown(O) {
            if (this._keyDownHandled = !1, this._keyDownSeen = !0, this._customKeyEventHandler && this._customKeyEventHandler(O) === !1) return !1;
            const E = this.browser.isMac && this.options.macOptionIsMeta && O.altKey;
            if (!E && !this._compositionHelper.keydown(O)) return this.options.scrollOnUserInput && this.buffer.ybase !== this.buffer.ydisp && this.scrollToBottom(), !1;
            E || O.key !== "Dead" && O.key !== "AltGraph" || (this._unprocessedDeadKey = !0);
            const H = (0, M.evaluateKeyboardEvent)(O, this.coreService.decPrivateModes.applicationCursorKeys, this.browser.isMac, this.options.macOptionIsMeta);
            if (this.updateCursorStyle(O), H.type === 3 || H.type === 2) {
              const N = this.rows - 1;
              return this.scrollLines(H.type === 2 ? -N : N), this.cancel(O, !0);
            }
            return H.type === 1 && this.selectAll(), !!this._isThirdLevelShift(this.browser, O) || (H.cancel && this.cancel(O, !0), !H.key || !!(O.key && !O.ctrlKey && !O.altKey && !O.metaKey && O.key.length === 1 && O.key.charCodeAt(0) >= 65 && O.key.charCodeAt(0) <= 90) || (this._unprocessedDeadKey ? (this._unprocessedDeadKey = !1, !0) : (H.key !== k.C0.ETX && H.key !== k.C0.CR || (this.textarea.value = ""), this._onKey.fire({ key: H.key, domEvent: O }), this._showCursor(), this.coreService.triggerDataEvent(H.key, !0), !this.optionsService.rawOptions.screenReaderMode || O.altKey || O.ctrlKey ? this.cancel(O, !0) : void (this._keyDownHandled = !0))));
          }
          _isThirdLevelShift(O, E) {
            const H = O.isMac && !this.options.macOptionIsMeta && E.altKey && !E.ctrlKey && !E.metaKey || O.isWindows && E.altKey && E.ctrlKey && !E.metaKey || O.isWindows && E.getModifierState("AltGraph");
            return E.type === "keypress" ? H : H && (!E.keyCode || E.keyCode > 47);
          }
          _keyUp(O) {
            this._keyDownSeen = !1, this._customKeyEventHandler && this._customKeyEventHandler(O) === !1 || ((function(E) {
              return E.keyCode === 16 || E.keyCode === 17 || E.keyCode === 18;
            })(O) || this.focus(), this.updateCursorStyle(O), this._keyPressHandled = !1);
          }
          _keyPress(O) {
            let E;
            if (this._keyPressHandled = !1, this._keyDownHandled || this._customKeyEventHandler && this._customKeyEventHandler(O) === !1) return !1;
            if (this.cancel(O), O.charCode) E = O.charCode;
            else if (O.which === null || O.which === void 0) E = O.keyCode;
            else {
              if (O.which === 0 || O.charCode === 0) return !1;
              E = O.which;
            }
            return !(!E || (O.altKey || O.ctrlKey || O.metaKey) && !this._isThirdLevelShift(this.browser, O) || (E = String.fromCharCode(E), this._onKey.fire({ key: E, domEvent: O }), this._showCursor(), this.coreService.triggerDataEvent(E, !0), this._keyPressHandled = !0, this._unprocessedDeadKey = !1, 0));
          }
          _inputEvent(O) {
            if (O.data && O.inputType === "insertText" && (!O.composed || !this._keyDownSeen) && !this.optionsService.rawOptions.screenReaderMode) {
              if (this._keyPressHandled) return !1;
              this._unprocessedDeadKey = !1;
              const E = O.data;
              return this.coreService.triggerDataEvent(E, !0), this.cancel(O), !0;
            }
            return !1;
          }
          resize(O, E) {
            O !== this.cols || E !== this.rows ? super.resize(O, E) : this._charSizeService && !this._charSizeService.hasValidSize && this._charSizeService.measure();
          }
          _afterResize(O, E) {
            var H, N;
            (H = this._charSizeService) == null || H.measure(), (N = this.viewport) == null || N.syncScrollArea(!0);
          }
          clear() {
            var O;
            if (this.buffer.ybase !== 0 || this.buffer.y !== 0) {
              this.buffer.clearAllMarkers(), this.buffer.lines.set(0, this.buffer.lines.get(this.buffer.ybase + this.buffer.y)), this.buffer.lines.length = 1, this.buffer.ydisp = 0, this.buffer.ybase = 0, this.buffer.y = 0;
              for (let E = 1; E < this.rows; E++) this.buffer.lines.push(this.buffer.getBlankLine(P.DEFAULT_ATTR_DATA));
              this._onScroll.fire({ position: this.buffer.ydisp, source: 0 }), (O = this.viewport) == null || O.reset(), this.refresh(0, this.rows - 1);
            }
          }
          reset() {
            var E, H;
            this.options.rows = this.rows, this.options.cols = this.cols;
            const O = this._customKeyEventHandler;
            this._setup(), super.reset(), (E = this._selectionService) == null || E.reset(), this._decorationService.reset(), (H = this.viewport) == null || H.reset(), this._customKeyEventHandler = O, this.refresh(0, this.rows - 1);
          }
          clearTextureAtlas() {
            var O;
            (O = this._renderService) == null || O.clearTextureAtlas();
          }
          _reportFocus() {
            var O;
            (O = this.element) != null && O.classList.contains("focus") ? this.coreService.triggerDataEvent(k.C0.ESC + "[I") : this.coreService.triggerDataEvent(k.C0.ESC + "[O");
          }
          _reportWindowsOptions(O) {
            if (this._renderService) switch (O) {
              case D.WindowsOptionsReportType.GET_WIN_SIZE_PIXELS:
                const E = this._renderService.dimensions.css.canvas.width.toFixed(0), H = this._renderService.dimensions.css.canvas.height.toFixed(0);
                this.coreService.triggerDataEvent(`${k.C0.ESC}[4;${H};${E}t`);
                break;
              case D.WindowsOptionsReportType.GET_CELL_SIZE_PIXELS:
                const N = this._renderService.dimensions.css.cell.width.toFixed(0), G = this._renderService.dimensions.css.cell.height.toFixed(0);
                this.coreService.triggerDataEvent(`${k.C0.ESC}[6;${G};${N}t`);
            }
          }
          cancel(O, E) {
            if (this.options.cancelEvents || E) return O.preventDefault(), O.stopPropagation(), !1;
          }
        }
        t.Terminal = K;
      }, 9924: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.TimeBasedDebouncer = void 0, t.TimeBasedDebouncer = class {
          constructor(a, c = 1e3) {
            this._renderCallback = a, this._debounceThresholdMS = c, this._lastRefreshMs = 0, this._additionalRefreshRequested = !1;
          }
          dispose() {
            this._refreshTimeoutID && clearTimeout(this._refreshTimeoutID);
          }
          refresh(a, c, h) {
            this._rowCount = h, a = a !== void 0 ? a : 0, c = c !== void 0 ? c : this._rowCount - 1, this._rowStart = this._rowStart !== void 0 ? Math.min(this._rowStart, a) : a, this._rowEnd = this._rowEnd !== void 0 ? Math.max(this._rowEnd, c) : c;
            const r = Date.now();
            if (r - this._lastRefreshMs >= this._debounceThresholdMS) this._lastRefreshMs = r, this._innerRefresh();
            else if (!this._additionalRefreshRequested) {
              const d = r - this._lastRefreshMs, f = this._debounceThresholdMS - d;
              this._additionalRefreshRequested = !0, this._refreshTimeoutID = window.setTimeout((() => {
                this._lastRefreshMs = Date.now(), this._innerRefresh(), this._additionalRefreshRequested = !1, this._refreshTimeoutID = void 0;
              }), f);
            }
          }
          _innerRefresh() {
            if (this._rowStart === void 0 || this._rowEnd === void 0 || this._rowCount === void 0) return;
            const a = Math.max(this._rowStart, 0), c = Math.min(this._rowEnd, this._rowCount - 1);
            this._rowStart = void 0, this._rowEnd = void 0, this._renderCallback(a, c);
          }
        };
      }, 1680: function(T, t, a) {
        var c = this && this.__decorate || function(o, s, i, u) {
          var p, l = arguments.length, m = l < 3 ? s : u === null ? u = Object.getOwnPropertyDescriptor(s, i) : u;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") m = Reflect.decorate(o, s, i, u);
          else for (var _ = o.length - 1; _ >= 0; _--) (p = o[_]) && (m = (l < 3 ? p(m) : l > 3 ? p(s, i, m) : p(s, i)) || m);
          return l > 3 && m && Object.defineProperty(s, i, m), m;
        }, h = this && this.__param || function(o, s) {
          return function(i, u) {
            s(i, u, o);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.Viewport = void 0;
        const r = a(3656), d = a(4725), f = a(8460), g = a(844), n = a(2585);
        let e = t.Viewport = class extends g.Disposable {
          constructor(o, s, i, u, p, l, m, _) {
            super(), this._viewportElement = o, this._scrollArea = s, this._bufferService = i, this._optionsService = u, this._charSizeService = p, this._renderService = l, this._coreBrowserService = m, this.scrollBarWidth = 0, this._currentRowHeight = 0, this._currentDeviceCellHeight = 0, this._lastRecordedBufferLength = 0, this._lastRecordedViewportHeight = 0, this._lastRecordedBufferHeight = 0, this._lastTouchY = 0, this._lastScrollTop = 0, this._wheelPartialScroll = 0, this._refreshAnimationFrame = null, this._ignoreNextScrollEvent = !1, this._smoothScrollState = { startTime: 0, origin: -1, target: -1 }, this._onRequestScrollLines = this.register(new f.EventEmitter()), this.onRequestScrollLines = this._onRequestScrollLines.event, this.scrollBarWidth = this._viewportElement.offsetWidth - this._scrollArea.offsetWidth || 15, this.register((0, r.addDisposableDomListener)(this._viewportElement, "scroll", this._handleScroll.bind(this))), this._activeBuffer = this._bufferService.buffer, this.register(this._bufferService.buffers.onBufferActivate(((v) => this._activeBuffer = v.activeBuffer))), this._renderDimensions = this._renderService.dimensions, this.register(this._renderService.onDimensionsChange(((v) => this._renderDimensions = v))), this._handleThemeChange(_.colors), this.register(_.onChangeColors(((v) => this._handleThemeChange(v)))), this.register(this._optionsService.onSpecificOptionChange("scrollback", (() => this.syncScrollArea()))), setTimeout((() => this.syncScrollArea()));
          }
          _handleThemeChange(o) {
            this._viewportElement.style.backgroundColor = o.background.css;
          }
          reset() {
            this._currentRowHeight = 0, this._currentDeviceCellHeight = 0, this._lastRecordedBufferLength = 0, this._lastRecordedViewportHeight = 0, this._lastRecordedBufferHeight = 0, this._lastTouchY = 0, this._lastScrollTop = 0, this._coreBrowserService.window.requestAnimationFrame((() => this.syncScrollArea()));
          }
          _refresh(o) {
            if (o) return this._innerRefresh(), void (this._refreshAnimationFrame !== null && this._coreBrowserService.window.cancelAnimationFrame(this._refreshAnimationFrame));
            this._refreshAnimationFrame === null && (this._refreshAnimationFrame = this._coreBrowserService.window.requestAnimationFrame((() => this._innerRefresh())));
          }
          _innerRefresh() {
            if (this._charSizeService.height > 0) {
              this._currentRowHeight = this._renderDimensions.device.cell.height / this._coreBrowserService.dpr, this._currentDeviceCellHeight = this._renderDimensions.device.cell.height, this._lastRecordedViewportHeight = this._viewportElement.offsetHeight;
              const s = Math.round(this._currentRowHeight * this._lastRecordedBufferLength) + (this._lastRecordedViewportHeight - this._renderDimensions.css.canvas.height);
              this._lastRecordedBufferHeight !== s && (this._lastRecordedBufferHeight = s, this._scrollArea.style.height = this._lastRecordedBufferHeight + "px");
            }
            const o = this._bufferService.buffer.ydisp * this._currentRowHeight;
            this._viewportElement.scrollTop !== o && (this._ignoreNextScrollEvent = !0, this._viewportElement.scrollTop = o), this._refreshAnimationFrame = null;
          }
          syncScrollArea(o = !1) {
            if (this._lastRecordedBufferLength !== this._bufferService.buffer.lines.length) return this._lastRecordedBufferLength = this._bufferService.buffer.lines.length, void this._refresh(o);
            this._lastRecordedViewportHeight === this._renderService.dimensions.css.canvas.height && this._lastScrollTop === this._activeBuffer.ydisp * this._currentRowHeight && this._renderDimensions.device.cell.height === this._currentDeviceCellHeight || this._refresh(o);
          }
          _handleScroll(o) {
            if (this._lastScrollTop = this._viewportElement.scrollTop, !this._viewportElement.offsetParent) return;
            if (this._ignoreNextScrollEvent) return this._ignoreNextScrollEvent = !1, void this._onRequestScrollLines.fire({ amount: 0, suppressScrollEvent: !0 });
            const s = Math.round(this._lastScrollTop / this._currentRowHeight) - this._bufferService.buffer.ydisp;
            this._onRequestScrollLines.fire({ amount: s, suppressScrollEvent: !0 });
          }
          _smoothScroll() {
            if (this._isDisposed || this._smoothScrollState.origin === -1 || this._smoothScrollState.target === -1) return;
            const o = this._smoothScrollPercent();
            this._viewportElement.scrollTop = this._smoothScrollState.origin + Math.round(o * (this._smoothScrollState.target - this._smoothScrollState.origin)), o < 1 ? this._coreBrowserService.window.requestAnimationFrame((() => this._smoothScroll())) : this._clearSmoothScrollState();
          }
          _smoothScrollPercent() {
            return this._optionsService.rawOptions.smoothScrollDuration && this._smoothScrollState.startTime ? Math.max(Math.min((Date.now() - this._smoothScrollState.startTime) / this._optionsService.rawOptions.smoothScrollDuration, 1), 0) : 1;
          }
          _clearSmoothScrollState() {
            this._smoothScrollState.startTime = 0, this._smoothScrollState.origin = -1, this._smoothScrollState.target = -1;
          }
          _bubbleScroll(o, s) {
            const i = this._viewportElement.scrollTop + this._lastRecordedViewportHeight;
            return !(s < 0 && this._viewportElement.scrollTop !== 0 || s > 0 && i < this._lastRecordedBufferHeight) || (o.cancelable && o.preventDefault(), !1);
          }
          handleWheel(o) {
            const s = this._getPixelsScrolled(o);
            return s !== 0 && (this._optionsService.rawOptions.smoothScrollDuration ? (this._smoothScrollState.startTime = Date.now(), this._smoothScrollPercent() < 1 ? (this._smoothScrollState.origin = this._viewportElement.scrollTop, this._smoothScrollState.target === -1 ? this._smoothScrollState.target = this._viewportElement.scrollTop + s : this._smoothScrollState.target += s, this._smoothScrollState.target = Math.max(Math.min(this._smoothScrollState.target, this._viewportElement.scrollHeight), 0), this._smoothScroll()) : this._clearSmoothScrollState()) : this._viewportElement.scrollTop += s, this._bubbleScroll(o, s));
          }
          scrollLines(o) {
            if (o !== 0) if (this._optionsService.rawOptions.smoothScrollDuration) {
              const s = o * this._currentRowHeight;
              this._smoothScrollState.startTime = Date.now(), this._smoothScrollPercent() < 1 ? (this._smoothScrollState.origin = this._viewportElement.scrollTop, this._smoothScrollState.target = this._smoothScrollState.origin + s, this._smoothScrollState.target = Math.max(Math.min(this._smoothScrollState.target, this._viewportElement.scrollHeight), 0), this._smoothScroll()) : this._clearSmoothScrollState();
            } else this._onRequestScrollLines.fire({ amount: o, suppressScrollEvent: !1 });
          }
          _getPixelsScrolled(o) {
            if (o.deltaY === 0 || o.shiftKey) return 0;
            let s = this._applyScrollModifier(o.deltaY, o);
            return o.deltaMode === WheelEvent.DOM_DELTA_LINE ? s *= this._currentRowHeight : o.deltaMode === WheelEvent.DOM_DELTA_PAGE && (s *= this._currentRowHeight * this._bufferService.rows), s;
          }
          getBufferElements(o, s) {
            var _;
            let i, u = "";
            const p = [], l = s != null ? s : this._bufferService.buffer.lines.length, m = this._bufferService.buffer.lines;
            for (let v = o; v < l; v++) {
              const C = m.get(v);
              if (!C) continue;
              const w = (_ = m.get(v + 1)) == null ? void 0 : _.isWrapped;
              if (u += C.translateToString(!w), !w || v === m.length - 1) {
                const S = document.createElement("div");
                S.textContent = u, p.push(S), u.length > 0 && (i = S), u = "";
              }
            }
            return { bufferElements: p, cursorElement: i };
          }
          getLinesScrolled(o) {
            if (o.deltaY === 0 || o.shiftKey) return 0;
            let s = this._applyScrollModifier(o.deltaY, o);
            return o.deltaMode === WheelEvent.DOM_DELTA_PIXEL ? (s /= this._currentRowHeight + 0, this._wheelPartialScroll += s, s = Math.floor(Math.abs(this._wheelPartialScroll)) * (this._wheelPartialScroll > 0 ? 1 : -1), this._wheelPartialScroll %= 1) : o.deltaMode === WheelEvent.DOM_DELTA_PAGE && (s *= this._bufferService.rows), s;
          }
          _applyScrollModifier(o, s) {
            const i = this._optionsService.rawOptions.fastScrollModifier;
            return i === "alt" && s.altKey || i === "ctrl" && s.ctrlKey || i === "shift" && s.shiftKey ? o * this._optionsService.rawOptions.fastScrollSensitivity * this._optionsService.rawOptions.scrollSensitivity : o * this._optionsService.rawOptions.scrollSensitivity;
          }
          handleTouchStart(o) {
            this._lastTouchY = o.touches[0].pageY;
          }
          handleTouchMove(o) {
            const s = this._lastTouchY - o.touches[0].pageY;
            return this._lastTouchY = o.touches[0].pageY, s !== 0 && (this._viewportElement.scrollTop += s, this._bubbleScroll(o, s));
          }
        };
        t.Viewport = e = c([h(2, n.IBufferService), h(3, n.IOptionsService), h(4, d.ICharSizeService), h(5, d.IRenderService), h(6, d.ICoreBrowserService), h(7, d.IThemeService)], e);
      }, 3107: function(T, t, a) {
        var c = this && this.__decorate || function(n, e, o, s) {
          var i, u = arguments.length, p = u < 3 ? e : s === null ? s = Object.getOwnPropertyDescriptor(e, o) : s;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") p = Reflect.decorate(n, e, o, s);
          else for (var l = n.length - 1; l >= 0; l--) (i = n[l]) && (p = (u < 3 ? i(p) : u > 3 ? i(e, o, p) : i(e, o)) || p);
          return u > 3 && p && Object.defineProperty(e, o, p), p;
        }, h = this && this.__param || function(n, e) {
          return function(o, s) {
            e(o, s, n);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.BufferDecorationRenderer = void 0;
        const r = a(4725), d = a(844), f = a(2585);
        let g = t.BufferDecorationRenderer = class extends d.Disposable {
          constructor(n, e, o, s, i) {
            super(), this._screenElement = n, this._bufferService = e, this._coreBrowserService = o, this._decorationService = s, this._renderService = i, this._decorationElements = /* @__PURE__ */ new Map(), this._altBufferIsActive = !1, this._dimensionsChanged = !1, this._container = document.createElement("div"), this._container.classList.add("xterm-decoration-container"), this._screenElement.appendChild(this._container), this.register(this._renderService.onRenderedViewportChange((() => this._doRefreshDecorations()))), this.register(this._renderService.onDimensionsChange((() => {
              this._dimensionsChanged = !0, this._queueRefresh();
            }))), this.register(this._coreBrowserService.onDprChange((() => this._queueRefresh()))), this.register(this._bufferService.buffers.onBufferActivate((() => {
              this._altBufferIsActive = this._bufferService.buffer === this._bufferService.buffers.alt;
            }))), this.register(this._decorationService.onDecorationRegistered((() => this._queueRefresh()))), this.register(this._decorationService.onDecorationRemoved(((u) => this._removeDecoration(u)))), this.register((0, d.toDisposable)((() => {
              this._container.remove(), this._decorationElements.clear();
            })));
          }
          _queueRefresh() {
            this._animationFrame === void 0 && (this._animationFrame = this._renderService.addRefreshCallback((() => {
              this._doRefreshDecorations(), this._animationFrame = void 0;
            })));
          }
          _doRefreshDecorations() {
            for (const n of this._decorationService.decorations) this._renderDecoration(n);
            this._dimensionsChanged = !1;
          }
          _renderDecoration(n) {
            this._refreshStyle(n), this._dimensionsChanged && this._refreshXPosition(n);
          }
          _createElement(n) {
            var s, i;
            const e = this._coreBrowserService.mainDocument.createElement("div");
            e.classList.add("xterm-decoration"), e.classList.toggle("xterm-decoration-top-layer", ((s = n == null ? void 0 : n.options) == null ? void 0 : s.layer) === "top"), e.style.width = `${Math.round((n.options.width || 1) * this._renderService.dimensions.css.cell.width)}px`, e.style.height = (n.options.height || 1) * this._renderService.dimensions.css.cell.height + "px", e.style.top = (n.marker.line - this._bufferService.buffers.active.ydisp) * this._renderService.dimensions.css.cell.height + "px", e.style.lineHeight = `${this._renderService.dimensions.css.cell.height}px`;
            const o = (i = n.options.x) != null ? i : 0;
            return o && o > this._bufferService.cols && (e.style.display = "none"), this._refreshXPosition(n, e), e;
          }
          _refreshStyle(n) {
            const e = n.marker.line - this._bufferService.buffers.active.ydisp;
            if (e < 0 || e >= this._bufferService.rows) n.element && (n.element.style.display = "none", n.onRenderEmitter.fire(n.element));
            else {
              let o = this._decorationElements.get(n);
              o || (o = this._createElement(n), n.element = o, this._decorationElements.set(n, o), this._container.appendChild(o), n.onDispose((() => {
                this._decorationElements.delete(n), o.remove();
              }))), o.style.top = e * this._renderService.dimensions.css.cell.height + "px", o.style.display = this._altBufferIsActive ? "none" : "block", n.onRenderEmitter.fire(o);
            }
          }
          _refreshXPosition(n, e = n.element) {
            var s;
            if (!e) return;
            const o = (s = n.options.x) != null ? s : 0;
            (n.options.anchor || "left") === "right" ? e.style.right = o ? o * this._renderService.dimensions.css.cell.width + "px" : "" : e.style.left = o ? o * this._renderService.dimensions.css.cell.width + "px" : "";
          }
          _removeDecoration(n) {
            var e;
            (e = this._decorationElements.get(n)) == null || e.remove(), this._decorationElements.delete(n), n.dispose();
          }
        };
        t.BufferDecorationRenderer = g = c([h(1, f.IBufferService), h(2, r.ICoreBrowserService), h(3, f.IDecorationService), h(4, r.IRenderService)], g);
      }, 5871: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.ColorZoneStore = void 0, t.ColorZoneStore = class {
          constructor() {
            this._zones = [], this._zonePool = [], this._zonePoolIndex = 0, this._linePadding = { full: 0, left: 0, center: 0, right: 0 };
          }
          get zones() {
            return this._zonePool.length = Math.min(this._zonePool.length, this._zones.length), this._zones;
          }
          clear() {
            this._zones.length = 0, this._zonePoolIndex = 0;
          }
          addDecoration(a) {
            if (a.options.overviewRulerOptions) {
              for (const c of this._zones) if (c.color === a.options.overviewRulerOptions.color && c.position === a.options.overviewRulerOptions.position) {
                if (this._lineIntersectsZone(c, a.marker.line)) return;
                if (this._lineAdjacentToZone(c, a.marker.line, a.options.overviewRulerOptions.position)) return void this._addLineToZone(c, a.marker.line);
              }
              if (this._zonePoolIndex < this._zonePool.length) return this._zonePool[this._zonePoolIndex].color = a.options.overviewRulerOptions.color, this._zonePool[this._zonePoolIndex].position = a.options.overviewRulerOptions.position, this._zonePool[this._zonePoolIndex].startBufferLine = a.marker.line, this._zonePool[this._zonePoolIndex].endBufferLine = a.marker.line, void this._zones.push(this._zonePool[this._zonePoolIndex++]);
              this._zones.push({ color: a.options.overviewRulerOptions.color, position: a.options.overviewRulerOptions.position, startBufferLine: a.marker.line, endBufferLine: a.marker.line }), this._zonePool.push(this._zones[this._zones.length - 1]), this._zonePoolIndex++;
            }
          }
          setPadding(a) {
            this._linePadding = a;
          }
          _lineIntersectsZone(a, c) {
            return c >= a.startBufferLine && c <= a.endBufferLine;
          }
          _lineAdjacentToZone(a, c, h) {
            return c >= a.startBufferLine - this._linePadding[h || "full"] && c <= a.endBufferLine + this._linePadding[h || "full"];
          }
          _addLineToZone(a, c) {
            a.startBufferLine = Math.min(a.startBufferLine, c), a.endBufferLine = Math.max(a.endBufferLine, c);
          }
        };
      }, 5744: function(T, t, a) {
        var c = this && this.__decorate || function(i, u, p, l) {
          var m, _ = arguments.length, v = _ < 3 ? u : l === null ? l = Object.getOwnPropertyDescriptor(u, p) : l;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") v = Reflect.decorate(i, u, p, l);
          else for (var C = i.length - 1; C >= 0; C--) (m = i[C]) && (v = (_ < 3 ? m(v) : _ > 3 ? m(u, p, v) : m(u, p)) || v);
          return _ > 3 && v && Object.defineProperty(u, p, v), v;
        }, h = this && this.__param || function(i, u) {
          return function(p, l) {
            u(p, l, i);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.OverviewRulerRenderer = void 0;
        const r = a(5871), d = a(4725), f = a(844), g = a(2585), n = { full: 0, left: 0, center: 0, right: 0 }, e = { full: 0, left: 0, center: 0, right: 0 }, o = { full: 0, left: 0, center: 0, right: 0 };
        let s = t.OverviewRulerRenderer = class extends f.Disposable {
          get _width() {
            return this._optionsService.options.overviewRulerWidth || 0;
          }
          constructor(i, u, p, l, m, _, v) {
            var w;
            super(), this._viewportElement = i, this._screenElement = u, this._bufferService = p, this._decorationService = l, this._renderService = m, this._optionsService = _, this._coreBrowserService = v, this._colorZoneStore = new r.ColorZoneStore(), this._shouldUpdateDimensions = !0, this._shouldUpdateAnchor = !0, this._lastKnownBufferLength = 0, this._canvas = this._coreBrowserService.mainDocument.createElement("canvas"), this._canvas.classList.add("xterm-decoration-overview-ruler"), this._refreshCanvasDimensions(), (w = this._viewportElement.parentElement) == null || w.insertBefore(this._canvas, this._viewportElement);
            const C = this._canvas.getContext("2d");
            if (!C) throw new Error("Ctx cannot be null");
            this._ctx = C, this._registerDecorationListeners(), this._registerBufferChangeListeners(), this._registerDimensionChangeListeners(), this.register((0, f.toDisposable)((() => {
              var S;
              (S = this._canvas) == null || S.remove();
            })));
          }
          _registerDecorationListeners() {
            this.register(this._decorationService.onDecorationRegistered((() => this._queueRefresh(void 0, !0)))), this.register(this._decorationService.onDecorationRemoved((() => this._queueRefresh(void 0, !0))));
          }
          _registerBufferChangeListeners() {
            this.register(this._renderService.onRenderedViewportChange((() => this._queueRefresh()))), this.register(this._bufferService.buffers.onBufferActivate((() => {
              this._canvas.style.display = this._bufferService.buffer === this._bufferService.buffers.alt ? "none" : "block";
            }))), this.register(this._bufferService.onScroll((() => {
              this._lastKnownBufferLength !== this._bufferService.buffers.normal.lines.length && (this._refreshDrawHeightConstants(), this._refreshColorZonePadding());
            })));
          }
          _registerDimensionChangeListeners() {
            this.register(this._renderService.onRender((() => {
              this._containerHeight && this._containerHeight === this._screenElement.clientHeight || (this._queueRefresh(!0), this._containerHeight = this._screenElement.clientHeight);
            }))), this.register(this._optionsService.onSpecificOptionChange("overviewRulerWidth", (() => this._queueRefresh(!0)))), this.register(this._coreBrowserService.onDprChange((() => this._queueRefresh(!0)))), this._queueRefresh(!0);
          }
          _refreshDrawConstants() {
            const i = Math.floor(this._canvas.width / 3), u = Math.ceil(this._canvas.width / 3);
            e.full = this._canvas.width, e.left = i, e.center = u, e.right = i, this._refreshDrawHeightConstants(), o.full = 0, o.left = 0, o.center = e.left, o.right = e.left + e.center;
          }
          _refreshDrawHeightConstants() {
            n.full = Math.round(2 * this._coreBrowserService.dpr);
            const i = this._canvas.height / this._bufferService.buffer.lines.length, u = Math.round(Math.max(Math.min(i, 12), 6) * this._coreBrowserService.dpr);
            n.left = u, n.center = u, n.right = u;
          }
          _refreshColorZonePadding() {
            this._colorZoneStore.setPadding({ full: Math.floor(this._bufferService.buffers.active.lines.length / (this._canvas.height - 1) * n.full), left: Math.floor(this._bufferService.buffers.active.lines.length / (this._canvas.height - 1) * n.left), center: Math.floor(this._bufferService.buffers.active.lines.length / (this._canvas.height - 1) * n.center), right: Math.floor(this._bufferService.buffers.active.lines.length / (this._canvas.height - 1) * n.right) }), this._lastKnownBufferLength = this._bufferService.buffers.normal.lines.length;
          }
          _refreshCanvasDimensions() {
            this._canvas.style.width = `${this._width}px`, this._canvas.width = Math.round(this._width * this._coreBrowserService.dpr), this._canvas.style.height = `${this._screenElement.clientHeight}px`, this._canvas.height = Math.round(this._screenElement.clientHeight * this._coreBrowserService.dpr), this._refreshDrawConstants(), this._refreshColorZonePadding();
          }
          _refreshDecorations() {
            this._shouldUpdateDimensions && this._refreshCanvasDimensions(), this._ctx.clearRect(0, 0, this._canvas.width, this._canvas.height), this._colorZoneStore.clear();
            for (const u of this._decorationService.decorations) this._colorZoneStore.addDecoration(u);
            this._ctx.lineWidth = 1;
            const i = this._colorZoneStore.zones;
            for (const u of i) u.position !== "full" && this._renderColorZone(u);
            for (const u of i) u.position === "full" && this._renderColorZone(u);
            this._shouldUpdateDimensions = !1, this._shouldUpdateAnchor = !1;
          }
          _renderColorZone(i) {
            this._ctx.fillStyle = i.color, this._ctx.fillRect(o[i.position || "full"], Math.round((this._canvas.height - 1) * (i.startBufferLine / this._bufferService.buffers.active.lines.length) - n[i.position || "full"] / 2), e[i.position || "full"], Math.round((this._canvas.height - 1) * ((i.endBufferLine - i.startBufferLine) / this._bufferService.buffers.active.lines.length) + n[i.position || "full"]));
          }
          _queueRefresh(i, u) {
            this._shouldUpdateDimensions = i || this._shouldUpdateDimensions, this._shouldUpdateAnchor = u || this._shouldUpdateAnchor, this._animationFrame === void 0 && (this._animationFrame = this._coreBrowserService.window.requestAnimationFrame((() => {
              this._refreshDecorations(), this._animationFrame = void 0;
            })));
          }
        };
        t.OverviewRulerRenderer = s = c([h(2, g.IBufferService), h(3, g.IDecorationService), h(4, d.IRenderService), h(5, g.IOptionsService), h(6, d.ICoreBrowserService)], s);
      }, 2950: function(T, t, a) {
        var c = this && this.__decorate || function(n, e, o, s) {
          var i, u = arguments.length, p = u < 3 ? e : s === null ? s = Object.getOwnPropertyDescriptor(e, o) : s;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") p = Reflect.decorate(n, e, o, s);
          else for (var l = n.length - 1; l >= 0; l--) (i = n[l]) && (p = (u < 3 ? i(p) : u > 3 ? i(e, o, p) : i(e, o)) || p);
          return u > 3 && p && Object.defineProperty(e, o, p), p;
        }, h = this && this.__param || function(n, e) {
          return function(o, s) {
            e(o, s, n);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CompositionHelper = void 0;
        const r = a(4725), d = a(2585), f = a(2584);
        let g = t.CompositionHelper = class {
          get isComposing() {
            return this._isComposing;
          }
          constructor(n, e, o, s, i, u) {
            this._textarea = n, this._compositionView = e, this._bufferService = o, this._optionsService = s, this._coreService = i, this._renderService = u, this._isComposing = !1, this._isSendingComposition = !1, this._compositionPosition = { start: 0, end: 0 }, this._dataAlreadySent = "";
          }
          compositionstart() {
            this._isComposing = !0, this._compositionPosition.start = this._textarea.value.length, this._compositionView.textContent = "", this._dataAlreadySent = "", this._compositionView.classList.add("active");
          }
          compositionupdate(n) {
            this._compositionView.textContent = n.data, this.updateCompositionElements(), setTimeout((() => {
              this._compositionPosition.end = this._textarea.value.length;
            }), 0);
          }
          compositionend() {
            this._finalizeComposition(!0);
          }
          keydown(n) {
            if (this._isComposing || this._isSendingComposition) {
              if (n.keyCode === 229 || n.keyCode === 16 || n.keyCode === 17 || n.keyCode === 18) return !1;
              this._finalizeComposition(!1);
            }
            return n.keyCode !== 229 || (this._handleAnyTextareaChanges(), !1);
          }
          _finalizeComposition(n) {
            if (this._compositionView.classList.remove("active"), this._isComposing = !1, n) {
              const e = { start: this._compositionPosition.start, end: this._compositionPosition.end };
              this._isSendingComposition = !0, setTimeout((() => {
                if (this._isSendingComposition) {
                  let o;
                  this._isSendingComposition = !1, e.start += this._dataAlreadySent.length, o = this._isComposing ? this._textarea.value.substring(e.start, e.end) : this._textarea.value.substring(e.start), o.length > 0 && this._coreService.triggerDataEvent(o, !0);
                }
              }), 0);
            } else {
              this._isSendingComposition = !1;
              const e = this._textarea.value.substring(this._compositionPosition.start, this._compositionPosition.end);
              this._coreService.triggerDataEvent(e, !0);
            }
          }
          _handleAnyTextareaChanges() {
            const n = this._textarea.value;
            setTimeout((() => {
              if (!this._isComposing) {
                const e = this._textarea.value, o = e.replace(n, "");
                this._dataAlreadySent = o, e.length > n.length ? this._coreService.triggerDataEvent(o, !0) : e.length < n.length ? this._coreService.triggerDataEvent(`${f.C0.DEL}`, !0) : e.length === n.length && e !== n && this._coreService.triggerDataEvent(e, !0);
              }
            }), 0);
          }
          updateCompositionElements(n) {
            if (this._isComposing) {
              if (this._bufferService.buffer.isCursorInViewport) {
                const e = Math.min(this._bufferService.buffer.x, this._bufferService.cols - 1), o = this._renderService.dimensions.css.cell.height, s = this._bufferService.buffer.y * this._renderService.dimensions.css.cell.height, i = e * this._renderService.dimensions.css.cell.width;
                this._compositionView.style.left = i + "px", this._compositionView.style.top = s + "px", this._compositionView.style.height = o + "px", this._compositionView.style.lineHeight = o + "px", this._compositionView.style.fontFamily = this._optionsService.rawOptions.fontFamily, this._compositionView.style.fontSize = this._optionsService.rawOptions.fontSize + "px";
                const u = this._compositionView.getBoundingClientRect();
                this._textarea.style.left = i + "px", this._textarea.style.top = s + "px", this._textarea.style.width = Math.max(u.width, 1) + "px", this._textarea.style.height = Math.max(u.height, 1) + "px", this._textarea.style.lineHeight = u.height + "px";
              }
              n || setTimeout((() => this.updateCompositionElements(!0)), 0);
            }
          }
        };
        t.CompositionHelper = g = c([h(2, d.IBufferService), h(3, d.IOptionsService), h(4, d.ICoreService), h(5, r.IRenderService)], g);
      }, 9806: (T, t) => {
        function a(c, h, r) {
          const d = r.getBoundingClientRect(), f = c.getComputedStyle(r), g = parseInt(f.getPropertyValue("padding-left")), n = parseInt(f.getPropertyValue("padding-top"));
          return [h.clientX - d.left - g, h.clientY - d.top - n];
        }
        Object.defineProperty(t, "__esModule", { value: !0 }), t.getCoords = t.getCoordsRelativeToElement = void 0, t.getCoordsRelativeToElement = a, t.getCoords = function(c, h, r, d, f, g, n, e, o) {
          if (!g) return;
          const s = a(c, h, r);
          return s ? (s[0] = Math.ceil((s[0] + (o ? n / 2 : 0)) / n), s[1] = Math.ceil(s[1] / e), s[0] = Math.min(Math.max(s[0], 1), d + (o ? 1 : 0)), s[1] = Math.min(Math.max(s[1], 1), f), s) : void 0;
        };
      }, 9504: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.moveToCellSequence = void 0;
        const c = a(2584);
        function h(e, o, s, i) {
          const u = e - r(e, s), p = o - r(o, s), l = Math.abs(u - p) - (function(m, _, v) {
            let C = 0;
            const w = m - r(m, v), S = _ - r(_, v);
            for (let b = 0; b < Math.abs(w - S); b++) {
              const x = d(m, _) === "A" ? -1 : 1, A = v.buffer.lines.get(w + x * b);
              A != null && A.isWrapped && C++;
            }
            return C;
          })(e, o, s);
          return n(l, g(d(e, o), i));
        }
        function r(e, o) {
          let s = 0, i = o.buffer.lines.get(e), u = i == null ? void 0 : i.isWrapped;
          for (; u && e >= 0 && e < o.rows; ) s++, i = o.buffer.lines.get(--e), u = i == null ? void 0 : i.isWrapped;
          return s;
        }
        function d(e, o) {
          return e > o ? "A" : "B";
        }
        function f(e, o, s, i, u, p) {
          let l = e, m = o, _ = "";
          for (; l !== s || m !== i; ) l += u ? 1 : -1, u && l > p.cols - 1 ? (_ += p.buffer.translateBufferLineToString(m, !1, e, l), l = 0, e = 0, m++) : !u && l < 0 && (_ += p.buffer.translateBufferLineToString(m, !1, 0, e + 1), l = p.cols - 1, e = l, m--);
          return _ + p.buffer.translateBufferLineToString(m, !1, e, l);
        }
        function g(e, o) {
          const s = o ? "O" : "[";
          return c.C0.ESC + s + e;
        }
        function n(e, o) {
          e = Math.floor(e);
          let s = "";
          for (let i = 0; i < e; i++) s += o;
          return s;
        }
        t.moveToCellSequence = function(e, o, s, i) {
          const u = s.buffer.x, p = s.buffer.y;
          if (!s.buffer.hasScrollback) return (function(_, v, C, w, S, b) {
            return h(v, w, S, b).length === 0 ? "" : n(f(_, v, _, v - r(v, S), !1, S).length, g("D", b));
          })(u, p, 0, o, s, i) + h(p, o, s, i) + (function(_, v, C, w, S, b) {
            let x;
            x = h(v, w, S, b).length > 0 ? w - r(w, S) : v;
            const A = w, P = (function(k, M, y, L, R, D) {
              let F;
              return F = h(y, L, R, D).length > 0 ? L - r(L, R) : M, k < y && F <= L || k >= y && F < L ? "C" : "D";
            })(_, v, C, w, S, b);
            return n(f(_, x, C, A, P === "C", S).length, g(P, b));
          })(u, p, e, o, s, i);
          let l;
          if (p === o) return l = u > e ? "D" : "C", n(Math.abs(u - e), g(l, i));
          l = p > o ? "D" : "C";
          const m = Math.abs(p - o);
          return n((function(_, v) {
            return v.cols - _;
          })(p > o ? e : u, s) + (m - 1) * s.cols + 1 + ((p > o ? u : e) - 1), g(l, i));
        };
      }, 1296: function(T, t, a) {
        var c = this && this.__decorate || function(b, x, A, P) {
          var k, M = arguments.length, y = M < 3 ? x : P === null ? P = Object.getOwnPropertyDescriptor(x, A) : P;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") y = Reflect.decorate(b, x, A, P);
          else for (var L = b.length - 1; L >= 0; L--) (k = b[L]) && (y = (M < 3 ? k(y) : M > 3 ? k(x, A, y) : k(x, A)) || y);
          return M > 3 && y && Object.defineProperty(x, A, y), y;
        }, h = this && this.__param || function(b, x) {
          return function(A, P) {
            x(A, P, b);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.DomRenderer = void 0;
        const r = a(3787), d = a(2550), f = a(2223), g = a(6171), n = a(6052), e = a(4725), o = a(8055), s = a(8460), i = a(844), u = a(2585), p = "xterm-dom-renderer-owner-", l = "xterm-rows", m = "xterm-fg-", _ = "xterm-bg-", v = "xterm-focus", C = "xterm-selection";
        let w = 1, S = t.DomRenderer = class extends i.Disposable {
          constructor(b, x, A, P, k, M, y, L, R, D, F, U, K) {
            super(), this._terminal = b, this._document = x, this._element = A, this._screenElement = P, this._viewportElement = k, this._helperContainer = M, this._linkifier2 = y, this._charSizeService = R, this._optionsService = D, this._bufferService = F, this._coreBrowserService = U, this._themeService = K, this._terminalClass = w++, this._rowElements = [], this._selectionRenderModel = (0, n.createSelectionRenderModel)(), this.onRequestRedraw = this.register(new s.EventEmitter()).event, this._rowContainer = this._document.createElement("div"), this._rowContainer.classList.add(l), this._rowContainer.style.lineHeight = "normal", this._rowContainer.setAttribute("aria-hidden", "true"), this._refreshRowElements(this._bufferService.cols, this._bufferService.rows), this._selectionContainer = this._document.createElement("div"), this._selectionContainer.classList.add(C), this._selectionContainer.setAttribute("aria-hidden", "true"), this.dimensions = (0, g.createRenderDimensions)(), this._updateDimensions(), this.register(this._optionsService.onOptionChange((() => this._handleOptionsChanged()))), this.register(this._themeService.onChangeColors(((q) => this._injectCss(q)))), this._injectCss(this._themeService.colors), this._rowFactory = L.createInstance(r.DomRendererRowFactory, document), this._element.classList.add(p + this._terminalClass), this._screenElement.appendChild(this._rowContainer), this._screenElement.appendChild(this._selectionContainer), this.register(this._linkifier2.onShowLinkUnderline(((q) => this._handleLinkHover(q)))), this.register(this._linkifier2.onHideLinkUnderline(((q) => this._handleLinkLeave(q)))), this.register((0, i.toDisposable)((() => {
              this._element.classList.remove(p + this._terminalClass), this._rowContainer.remove(), this._selectionContainer.remove(), this._widthCache.dispose(), this._themeStyleElement.remove(), this._dimensionsStyleElement.remove();
            }))), this._widthCache = new d.WidthCache(this._document, this._helperContainer), this._widthCache.setFont(this._optionsService.rawOptions.fontFamily, this._optionsService.rawOptions.fontSize, this._optionsService.rawOptions.fontWeight, this._optionsService.rawOptions.fontWeightBold), this._setDefaultSpacing();
          }
          _updateDimensions() {
            const b = this._coreBrowserService.dpr;
            this.dimensions.device.char.width = this._charSizeService.width * b, this.dimensions.device.char.height = Math.ceil(this._charSizeService.height * b), this.dimensions.device.cell.width = this.dimensions.device.char.width + Math.round(this._optionsService.rawOptions.letterSpacing), this.dimensions.device.cell.height = Math.floor(this.dimensions.device.char.height * this._optionsService.rawOptions.lineHeight), this.dimensions.device.char.left = 0, this.dimensions.device.char.top = 0, this.dimensions.device.canvas.width = this.dimensions.device.cell.width * this._bufferService.cols, this.dimensions.device.canvas.height = this.dimensions.device.cell.height * this._bufferService.rows, this.dimensions.css.canvas.width = Math.round(this.dimensions.device.canvas.width / b), this.dimensions.css.canvas.height = Math.round(this.dimensions.device.canvas.height / b), this.dimensions.css.cell.width = this.dimensions.css.canvas.width / this._bufferService.cols, this.dimensions.css.cell.height = this.dimensions.css.canvas.height / this._bufferService.rows;
            for (const A of this._rowElements) A.style.width = `${this.dimensions.css.canvas.width}px`, A.style.height = `${this.dimensions.css.cell.height}px`, A.style.lineHeight = `${this.dimensions.css.cell.height}px`, A.style.overflow = "hidden";
            this._dimensionsStyleElement || (this._dimensionsStyleElement = this._document.createElement("style"), this._screenElement.appendChild(this._dimensionsStyleElement));
            const x = `${this._terminalSelector} .${l} span { display: inline-block; height: 100%; vertical-align: top;}`;
            this._dimensionsStyleElement.textContent = x, this._selectionContainer.style.height = this._viewportElement.style.height, this._screenElement.style.width = `${this.dimensions.css.canvas.width}px`, this._screenElement.style.height = `${this.dimensions.css.canvas.height}px`;
          }
          _injectCss(b) {
            this._themeStyleElement || (this._themeStyleElement = this._document.createElement("style"), this._screenElement.appendChild(this._themeStyleElement));
            let x = `${this._terminalSelector} .${l} { color: ${b.foreground.css}; font-family: ${this._optionsService.rawOptions.fontFamily}; font-size: ${this._optionsService.rawOptions.fontSize}px; font-kerning: none; white-space: pre}`;
            x += `${this._terminalSelector} .${l} .xterm-dim { color: ${o.color.multiplyOpacity(b.foreground, 0.5).css};}`, x += `${this._terminalSelector} span:not(.xterm-bold) { font-weight: ${this._optionsService.rawOptions.fontWeight};}${this._terminalSelector} span.xterm-bold { font-weight: ${this._optionsService.rawOptions.fontWeightBold};}${this._terminalSelector} span.xterm-italic { font-style: italic;}`;
            const A = `blink_underline_${this._terminalClass}`, P = `blink_bar_${this._terminalClass}`, k = `blink_block_${this._terminalClass}`;
            x += `@keyframes ${A} { 50% {  border-bottom-style: hidden; }}`, x += `@keyframes ${P} { 50% {  box-shadow: none; }}`, x += `@keyframes ${k} { 0% {  background-color: ${b.cursor.css};  color: ${b.cursorAccent.css}; } 50% {  background-color: inherit;  color: ${b.cursor.css}; }}`, x += `${this._terminalSelector} .${l}.${v} .xterm-cursor.xterm-cursor-blink.xterm-cursor-underline { animation: ${A} 1s step-end infinite;}${this._terminalSelector} .${l}.${v} .xterm-cursor.xterm-cursor-blink.xterm-cursor-bar { animation: ${P} 1s step-end infinite;}${this._terminalSelector} .${l}.${v} .xterm-cursor.xterm-cursor-blink.xterm-cursor-block { animation: ${k} 1s step-end infinite;}${this._terminalSelector} .${l} .xterm-cursor.xterm-cursor-block { background-color: ${b.cursor.css}; color: ${b.cursorAccent.css};}${this._terminalSelector} .${l} .xterm-cursor.xterm-cursor-block:not(.xterm-cursor-blink) { background-color: ${b.cursor.css} !important; color: ${b.cursorAccent.css} !important;}${this._terminalSelector} .${l} .xterm-cursor.xterm-cursor-outline { outline: 1px solid ${b.cursor.css}; outline-offset: -1px;}${this._terminalSelector} .${l} .xterm-cursor.xterm-cursor-bar { box-shadow: ${this._optionsService.rawOptions.cursorWidth}px 0 0 ${b.cursor.css} inset;}${this._terminalSelector} .${l} .xterm-cursor.xterm-cursor-underline { border-bottom: 1px ${b.cursor.css}; border-bottom-style: solid; height: calc(100% - 1px);}`, x += `${this._terminalSelector} .${C} { position: absolute; top: 0; left: 0; z-index: 1; pointer-events: none;}${this._terminalSelector}.focus .${C} div { position: absolute; background-color: ${b.selectionBackgroundOpaque.css};}${this._terminalSelector} .${C} div { position: absolute; background-color: ${b.selectionInactiveBackgroundOpaque.css};}`;
            for (const [M, y] of b.ansi.entries()) x += `${this._terminalSelector} .${m}${M} { color: ${y.css}; }${this._terminalSelector} .${m}${M}.xterm-dim { color: ${o.color.multiplyOpacity(y, 0.5).css}; }${this._terminalSelector} .${_}${M} { background-color: ${y.css}; }`;
            x += `${this._terminalSelector} .${m}${f.INVERTED_DEFAULT_COLOR} { color: ${o.color.opaque(b.background).css}; }${this._terminalSelector} .${m}${f.INVERTED_DEFAULT_COLOR}.xterm-dim { color: ${o.color.multiplyOpacity(o.color.opaque(b.background), 0.5).css}; }${this._terminalSelector} .${_}${f.INVERTED_DEFAULT_COLOR} { background-color: ${b.foreground.css}; }`, this._themeStyleElement.textContent = x;
          }
          _setDefaultSpacing() {
            const b = this.dimensions.css.cell.width - this._widthCache.get("W", !1, !1);
            this._rowContainer.style.letterSpacing = `${b}px`, this._rowFactory.defaultSpacing = b;
          }
          handleDevicePixelRatioChange() {
            this._updateDimensions(), this._widthCache.clear(), this._setDefaultSpacing();
          }
          _refreshRowElements(b, x) {
            for (let A = this._rowElements.length; A <= x; A++) {
              const P = this._document.createElement("div");
              this._rowContainer.appendChild(P), this._rowElements.push(P);
            }
            for (; this._rowElements.length > x; ) this._rowContainer.removeChild(this._rowElements.pop());
          }
          handleResize(b, x) {
            this._refreshRowElements(b, x), this._updateDimensions(), this.handleSelectionChanged(this._selectionRenderModel.selectionStart, this._selectionRenderModel.selectionEnd, this._selectionRenderModel.columnSelectMode);
          }
          handleCharSizeChanged() {
            this._updateDimensions(), this._widthCache.clear(), this._setDefaultSpacing();
          }
          handleBlur() {
            this._rowContainer.classList.remove(v), this.renderRows(0, this._bufferService.rows - 1);
          }
          handleFocus() {
            this._rowContainer.classList.add(v), this.renderRows(this._bufferService.buffer.y, this._bufferService.buffer.y);
          }
          handleSelectionChanged(b, x, A) {
            if (this._selectionContainer.replaceChildren(), this._rowFactory.handleSelectionChanged(b, x, A), this.renderRows(0, this._bufferService.rows - 1), !b || !x) return;
            this._selectionRenderModel.update(this._terminal, b, x, A);
            const P = this._selectionRenderModel.viewportStartRow, k = this._selectionRenderModel.viewportEndRow, M = this._selectionRenderModel.viewportCappedStartRow, y = this._selectionRenderModel.viewportCappedEndRow;
            if (M >= this._bufferService.rows || y < 0) return;
            const L = this._document.createDocumentFragment();
            if (A) {
              const R = b[0] > x[0];
              L.appendChild(this._createSelectionElement(M, R ? x[0] : b[0], R ? b[0] : x[0], y - M + 1));
            } else {
              const R = P === M ? b[0] : 0, D = M === k ? x[0] : this._bufferService.cols;
              L.appendChild(this._createSelectionElement(M, R, D));
              const F = y - M - 1;
              if (L.appendChild(this._createSelectionElement(M + 1, 0, this._bufferService.cols, F)), M !== y) {
                const U = k === y ? x[0] : this._bufferService.cols;
                L.appendChild(this._createSelectionElement(y, 0, U));
              }
            }
            this._selectionContainer.appendChild(L);
          }
          _createSelectionElement(b, x, A, P = 1) {
            const k = this._document.createElement("div"), M = x * this.dimensions.css.cell.width;
            let y = this.dimensions.css.cell.width * (A - x);
            return M + y > this.dimensions.css.canvas.width && (y = this.dimensions.css.canvas.width - M), k.style.height = P * this.dimensions.css.cell.height + "px", k.style.top = b * this.dimensions.css.cell.height + "px", k.style.left = `${M}px`, k.style.width = `${y}px`, k;
          }
          handleCursorMove() {
          }
          _handleOptionsChanged() {
            this._updateDimensions(), this._injectCss(this._themeService.colors), this._widthCache.setFont(this._optionsService.rawOptions.fontFamily, this._optionsService.rawOptions.fontSize, this._optionsService.rawOptions.fontWeight, this._optionsService.rawOptions.fontWeightBold), this._setDefaultSpacing();
          }
          clear() {
            for (const b of this._rowElements) b.replaceChildren();
          }
          renderRows(b, x) {
            const A = this._bufferService.buffer, P = A.ybase + A.y, k = Math.min(A.x, this._bufferService.cols - 1), M = this._optionsService.rawOptions.cursorBlink, y = this._optionsService.rawOptions.cursorStyle, L = this._optionsService.rawOptions.cursorInactiveStyle;
            for (let R = b; R <= x; R++) {
              const D = R + A.ydisp, F = this._rowElements[R], U = A.lines.get(D);
              if (!F || !U) break;
              F.replaceChildren(...this._rowFactory.createRow(U, D, D === P, y, L, k, M, this.dimensions.css.cell.width, this._widthCache, -1, -1));
            }
          }
          get _terminalSelector() {
            return `.${p}${this._terminalClass}`;
          }
          _handleLinkHover(b) {
            this._setCellUnderline(b.x1, b.x2, b.y1, b.y2, b.cols, !0);
          }
          _handleLinkLeave(b) {
            this._setCellUnderline(b.x1, b.x2, b.y1, b.y2, b.cols, !1);
          }
          _setCellUnderline(b, x, A, P, k, M) {
            A < 0 && (b = 0), P < 0 && (x = 0);
            const y = this._bufferService.rows - 1;
            A = Math.max(Math.min(A, y), 0), P = Math.max(Math.min(P, y), 0), k = Math.min(k, this._bufferService.cols);
            const L = this._bufferService.buffer, R = L.ybase + L.y, D = Math.min(L.x, k - 1), F = this._optionsService.rawOptions.cursorBlink, U = this._optionsService.rawOptions.cursorStyle, K = this._optionsService.rawOptions.cursorInactiveStyle;
            for (let q = A; q <= P; ++q) {
              const O = q + L.ydisp, E = this._rowElements[q], H = L.lines.get(O);
              if (!E || !H) break;
              E.replaceChildren(...this._rowFactory.createRow(H, O, O === R, U, K, D, F, this.dimensions.css.cell.width, this._widthCache, M ? q === A ? b : 0 : -1, M ? (q === P ? x : k) - 1 : -1));
            }
          }
        };
        t.DomRenderer = S = c([h(7, u.IInstantiationService), h(8, e.ICharSizeService), h(9, u.IOptionsService), h(10, u.IBufferService), h(11, e.ICoreBrowserService), h(12, e.IThemeService)], S);
      }, 3787: function(T, t, a) {
        var c = this && this.__decorate || function(l, m, _, v) {
          var C, w = arguments.length, S = w < 3 ? m : v === null ? v = Object.getOwnPropertyDescriptor(m, _) : v;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") S = Reflect.decorate(l, m, _, v);
          else for (var b = l.length - 1; b >= 0; b--) (C = l[b]) && (S = (w < 3 ? C(S) : w > 3 ? C(m, _, S) : C(m, _)) || S);
          return w > 3 && S && Object.defineProperty(m, _, S), S;
        }, h = this && this.__param || function(l, m) {
          return function(_, v) {
            m(_, v, l);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.DomRendererRowFactory = void 0;
        const r = a(2223), d = a(643), f = a(511), g = a(2585), n = a(8055), e = a(4725), o = a(4269), s = a(6171), i = a(3734);
        let u = t.DomRendererRowFactory = class {
          constructor(l, m, _, v, C, w, S) {
            this._document = l, this._characterJoinerService = m, this._optionsService = _, this._coreBrowserService = v, this._coreService = C, this._decorationService = w, this._themeService = S, this._workCell = new f.CellData(), this._columnSelectMode = !1, this.defaultSpacing = 0;
          }
          handleSelectionChanged(l, m, _) {
            this._selectionStart = l, this._selectionEnd = m, this._columnSelectMode = _;
          }
          createRow(l, m, _, v, C, w, S, b, x, A, P) {
            const k = [], M = this._characterJoinerService.getJoinedCharacters(m), y = this._themeService.colors;
            let L, R = l.getNoBgTrimmedLength();
            _ && R < w + 1 && (R = w + 1);
            let D = 0, F = "", U = 0, K = 0, q = 0, O = !1, E = 0, H = !1, N = 0;
            const G = [], j = A !== -1 && P !== -1;
            for (let ie = 0; ie < R; ie++) {
              l.loadCell(ie, this._workCell);
              let V = this._workCell.getWidth();
              if (V === 0) continue;
              let ae = !1, ce = ie, ee = this._workCell;
              if (M.length > 0 && ie === M[0][0]) {
                ae = !0;
                const te = M.shift();
                ee = new o.JoinedCellData(this._workCell, l.translateToString(!0, te[0], te[1]), te[1] - te[0]), ce = te[1] - 1, V = ee.getWidth();
              }
              const _e = this._isCellInSelection(ie, m), ge = _ && ie === w, Z = j && ie >= A && ie <= P;
              let X = !1;
              this._decorationService.forEachDecorationAtCell(ie, m, void 0, ((te) => {
                X = !0;
              }));
              let J = ee.getChars() || d.WHITESPACE_CELL_CHAR;
              if (J === " " && (ee.isUnderline() || ee.isOverline()) && (J = " "), N = V * b - x.get(J, ee.isBold(), ee.isItalic()), L) {
                if (D && (_e && H || !_e && !H && ee.bg === U) && (_e && H && y.selectionForeground || ee.fg === K) && ee.extended.ext === q && Z === O && N === E && !ge && !ae && !X) {
                  ee.isInvisible() ? F += d.WHITESPACE_CELL_CHAR : F += J, D++;
                  continue;
                }
                D && (L.textContent = F), L = this._document.createElement("span"), D = 0, F = "";
              } else L = this._document.createElement("span");
              if (U = ee.bg, K = ee.fg, q = ee.extended.ext, O = Z, E = N, H = _e, ae && w >= ie && w <= ce && (w = ie), !this._coreService.isCursorHidden && ge && this._coreService.isCursorInitialized) {
                if (G.push("xterm-cursor"), this._coreBrowserService.isFocused) S && G.push("xterm-cursor-blink"), G.push(v === "bar" ? "xterm-cursor-bar" : v === "underline" ? "xterm-cursor-underline" : "xterm-cursor-block");
                else if (C) switch (C) {
                  case "outline":
                    G.push("xterm-cursor-outline");
                    break;
                  case "block":
                    G.push("xterm-cursor-block");
                    break;
                  case "bar":
                    G.push("xterm-cursor-bar");
                    break;
                  case "underline":
                    G.push("xterm-cursor-underline");
                }
              }
              if (ee.isBold() && G.push("xterm-bold"), ee.isItalic() && G.push("xterm-italic"), ee.isDim() && G.push("xterm-dim"), F = ee.isInvisible() ? d.WHITESPACE_CELL_CHAR : ee.getChars() || d.WHITESPACE_CELL_CHAR, ee.isUnderline() && (G.push(`xterm-underline-${ee.extended.underlineStyle}`), F === " " && (F = " "), !ee.isUnderlineColorDefault())) if (ee.isUnderlineColorRGB()) L.style.textDecorationColor = `rgb(${i.AttributeData.toColorRGB(ee.getUnderlineColor()).join(",")})`;
              else {
                let te = ee.getUnderlineColor();
                this._optionsService.rawOptions.drawBoldTextInBrightColors && ee.isBold() && te < 8 && (te += 8), L.style.textDecorationColor = y.ansi[te].css;
              }
              ee.isOverline() && (G.push("xterm-overline"), F === " " && (F = " ")), ee.isStrikethrough() && G.push("xterm-strikethrough"), Z && (L.style.textDecoration = "underline");
              let z = ee.getFgColor(), Q = ee.getFgColorMode(), he = ee.getBgColor(), re = ee.getBgColorMode();
              const fe = !!ee.isInverse();
              if (fe) {
                const te = z;
                z = he, he = te;
                const ve = Q;
                Q = re, re = ve;
              }
              let de, ue, le, se = !1;
              switch (this._decorationService.forEachDecorationAtCell(ie, m, void 0, ((te) => {
                te.options.layer !== "top" && se || (te.backgroundColorRGB && (re = 50331648, he = te.backgroundColorRGB.rgba >> 8 & 16777215, de = te.backgroundColorRGB), te.foregroundColorRGB && (Q = 50331648, z = te.foregroundColorRGB.rgba >> 8 & 16777215, ue = te.foregroundColorRGB), se = te.options.layer === "top");
              })), !se && _e && (de = this._coreBrowserService.isFocused ? y.selectionBackgroundOpaque : y.selectionInactiveBackgroundOpaque, he = de.rgba >> 8 & 16777215, re = 50331648, se = !0, y.selectionForeground && (Q = 50331648, z = y.selectionForeground.rgba >> 8 & 16777215, ue = y.selectionForeground)), se && G.push("xterm-decoration-top"), re) {
                case 16777216:
                case 33554432:
                  le = y.ansi[he], G.push(`xterm-bg-${he}`);
                  break;
                case 50331648:
                  le = n.channels.toColor(he >> 16, he >> 8 & 255, 255 & he), this._addStyle(L, `background-color:#${p((he >>> 0).toString(16), "0", 6)}`);
                  break;
                default:
                  fe ? (le = y.foreground, G.push(`xterm-bg-${r.INVERTED_DEFAULT_COLOR}`)) : le = y.background;
              }
              switch (de || ee.isDim() && (de = n.color.multiplyOpacity(le, 0.5)), Q) {
                case 16777216:
                case 33554432:
                  ee.isBold() && z < 8 && this._optionsService.rawOptions.drawBoldTextInBrightColors && (z += 8), this._applyMinimumContrast(L, le, y.ansi[z], ee, de, void 0) || G.push(`xterm-fg-${z}`);
                  break;
                case 50331648:
                  const te = n.channels.toColor(z >> 16 & 255, z >> 8 & 255, 255 & z);
                  this._applyMinimumContrast(L, le, te, ee, de, ue) || this._addStyle(L, `color:#${p(z.toString(16), "0", 6)}`);
                  break;
                default:
                  this._applyMinimumContrast(L, le, y.foreground, ee, de, ue) || fe && G.push(`xterm-fg-${r.INVERTED_DEFAULT_COLOR}`);
              }
              G.length && (L.className = G.join(" "), G.length = 0), ge || ae || X ? L.textContent = F : D++, N !== this.defaultSpacing && (L.style.letterSpacing = `${N}px`), k.push(L), ie = ce;
            }
            return L && D && (L.textContent = F), k;
          }
          _applyMinimumContrast(l, m, _, v, C, w) {
            if (this._optionsService.rawOptions.minimumContrastRatio === 1 || (0, s.treatGlyphAsBackgroundColor)(v.getCode())) return !1;
            const S = this._getContrastCache(v);
            let b;
            if (C || w || (b = S.getColor(m.rgba, _.rgba)), b === void 0) {
              const x = this._optionsService.rawOptions.minimumContrastRatio / (v.isDim() ? 2 : 1);
              b = n.color.ensureContrastRatio(C || m, w || _, x), S.setColor((C || m).rgba, (w || _).rgba, b != null ? b : null);
            }
            return !!b && (this._addStyle(l, `color:${b.css}`), !0);
          }
          _getContrastCache(l) {
            return l.isDim() ? this._themeService.colors.halfContrastCache : this._themeService.colors.contrastCache;
          }
          _addStyle(l, m) {
            l.setAttribute("style", `${l.getAttribute("style") || ""}${m};`);
          }
          _isCellInSelection(l, m) {
            const _ = this._selectionStart, v = this._selectionEnd;
            return !(!_ || !v) && (this._columnSelectMode ? _[0] <= v[0] ? l >= _[0] && m >= _[1] && l < v[0] && m <= v[1] : l < _[0] && m >= _[1] && l >= v[0] && m <= v[1] : m > _[1] && m < v[1] || _[1] === v[1] && m === _[1] && l >= _[0] && l < v[0] || _[1] < v[1] && m === v[1] && l < v[0] || _[1] < v[1] && m === _[1] && l >= _[0]);
          }
        };
        function p(l, m, _) {
          for (; l.length < _; ) l = m + l;
          return l;
        }
        t.DomRendererRowFactory = u = c([h(1, e.ICharacterJoinerService), h(2, g.IOptionsService), h(3, e.ICoreBrowserService), h(4, g.ICoreService), h(5, g.IDecorationService), h(6, e.IThemeService)], u);
      }, 2550: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.WidthCache = void 0, t.WidthCache = class {
          constructor(a, c) {
            this._flat = new Float32Array(256), this._font = "", this._fontSize = 0, this._weight = "normal", this._weightBold = "bold", this._measureElements = [], this._container = a.createElement("div"), this._container.classList.add("xterm-width-cache-measure-container"), this._container.setAttribute("aria-hidden", "true"), this._container.style.whiteSpace = "pre", this._container.style.fontKerning = "none";
            const h = a.createElement("span");
            h.classList.add("xterm-char-measure-element");
            const r = a.createElement("span");
            r.classList.add("xterm-char-measure-element"), r.style.fontWeight = "bold";
            const d = a.createElement("span");
            d.classList.add("xterm-char-measure-element"), d.style.fontStyle = "italic";
            const f = a.createElement("span");
            f.classList.add("xterm-char-measure-element"), f.style.fontWeight = "bold", f.style.fontStyle = "italic", this._measureElements = [h, r, d, f], this._container.appendChild(h), this._container.appendChild(r), this._container.appendChild(d), this._container.appendChild(f), c.appendChild(this._container), this.clear();
          }
          dispose() {
            this._container.remove(), this._measureElements.length = 0, this._holey = void 0;
          }
          clear() {
            this._flat.fill(-9999), this._holey = /* @__PURE__ */ new Map();
          }
          setFont(a, c, h, r) {
            a === this._font && c === this._fontSize && h === this._weight && r === this._weightBold || (this._font = a, this._fontSize = c, this._weight = h, this._weightBold = r, this._container.style.fontFamily = this._font, this._container.style.fontSize = `${this._fontSize}px`, this._measureElements[0].style.fontWeight = `${h}`, this._measureElements[1].style.fontWeight = `${r}`, this._measureElements[2].style.fontWeight = `${h}`, this._measureElements[3].style.fontWeight = `${r}`, this.clear());
          }
          get(a, c, h) {
            let r = 0;
            if (!c && !h && a.length === 1 && (r = a.charCodeAt(0)) < 256) {
              if (this._flat[r] !== -9999) return this._flat[r];
              const g = this._measure(a, 0);
              return g > 0 && (this._flat[r] = g), g;
            }
            let d = a;
            c && (d += "B"), h && (d += "I");
            let f = this._holey.get(d);
            if (f === void 0) {
              let g = 0;
              c && (g |= 1), h && (g |= 2), f = this._measure(a, g), f > 0 && this._holey.set(d, f);
            }
            return f;
          }
          _measure(a, c) {
            const h = this._measureElements[c];
            return h.textContent = a.repeat(32), h.offsetWidth / 32;
          }
        };
      }, 2223: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.TEXT_BASELINE = t.DIM_OPACITY = t.INVERTED_DEFAULT_COLOR = void 0;
        const c = a(6114);
        t.INVERTED_DEFAULT_COLOR = 257, t.DIM_OPACITY = 0.5, t.TEXT_BASELINE = c.isFirefox || c.isLegacyEdge ? "bottom" : "ideographic";
      }, 6171: (T, t) => {
        function a(h) {
          return 57508 <= h && h <= 57558;
        }
        function c(h) {
          return h >= 128512 && h <= 128591 || h >= 127744 && h <= 128511 || h >= 128640 && h <= 128767 || h >= 9728 && h <= 9983 || h >= 9984 && h <= 10175 || h >= 65024 && h <= 65039 || h >= 129280 && h <= 129535 || h >= 127462 && h <= 127487;
        }
        Object.defineProperty(t, "__esModule", { value: !0 }), t.computeNextVariantOffset = t.createRenderDimensions = t.treatGlyphAsBackgroundColor = t.allowRescaling = t.isEmoji = t.isRestrictedPowerlineGlyph = t.isPowerlineGlyph = t.throwIfFalsy = void 0, t.throwIfFalsy = function(h) {
          if (!h) throw new Error("value must not be falsy");
          return h;
        }, t.isPowerlineGlyph = a, t.isRestrictedPowerlineGlyph = function(h) {
          return 57520 <= h && h <= 57527;
        }, t.isEmoji = c, t.allowRescaling = function(h, r, d, f) {
          return r === 1 && d > Math.ceil(1.5 * f) && h !== void 0 && h > 255 && !c(h) && !a(h) && !(function(g) {
            return 57344 <= g && g <= 63743;
          })(h);
        }, t.treatGlyphAsBackgroundColor = function(h) {
          return a(h) || (function(r) {
            return 9472 <= r && r <= 9631;
          })(h);
        }, t.createRenderDimensions = function() {
          return { css: { canvas: { width: 0, height: 0 }, cell: { width: 0, height: 0 } }, device: { canvas: { width: 0, height: 0 }, cell: { width: 0, height: 0 }, char: { width: 0, height: 0, left: 0, top: 0 } } };
        }, t.computeNextVariantOffset = function(h, r, d = 0) {
          return (h - (2 * Math.round(r) - d)) % (2 * Math.round(r));
        };
      }, 6052: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.createSelectionRenderModel = void 0;
        class a {
          constructor() {
            this.clear();
          }
          clear() {
            this.hasSelection = !1, this.columnSelectMode = !1, this.viewportStartRow = 0, this.viewportEndRow = 0, this.viewportCappedStartRow = 0, this.viewportCappedEndRow = 0, this.startCol = 0, this.endCol = 0, this.selectionStart = void 0, this.selectionEnd = void 0;
          }
          update(h, r, d, f = !1) {
            if (this.selectionStart = r, this.selectionEnd = d, !r || !d || r[0] === d[0] && r[1] === d[1]) return void this.clear();
            const g = h.buffers.active.ydisp, n = r[1] - g, e = d[1] - g, o = Math.max(n, 0), s = Math.min(e, h.rows - 1);
            o >= h.rows || s < 0 ? this.clear() : (this.hasSelection = !0, this.columnSelectMode = f, this.viewportStartRow = n, this.viewportEndRow = e, this.viewportCappedStartRow = o, this.viewportCappedEndRow = s, this.startCol = r[0], this.endCol = d[0]);
          }
          isCellSelected(h, r, d) {
            return !!this.hasSelection && (d -= h.buffer.active.viewportY, this.columnSelectMode ? this.startCol <= this.endCol ? r >= this.startCol && d >= this.viewportCappedStartRow && r < this.endCol && d <= this.viewportCappedEndRow : r < this.startCol && d >= this.viewportCappedStartRow && r >= this.endCol && d <= this.viewportCappedEndRow : d > this.viewportStartRow && d < this.viewportEndRow || this.viewportStartRow === this.viewportEndRow && d === this.viewportStartRow && r >= this.startCol && r < this.endCol || this.viewportStartRow < this.viewportEndRow && d === this.viewportEndRow && r < this.endCol || this.viewportStartRow < this.viewportEndRow && d === this.viewportStartRow && r >= this.startCol);
          }
        }
        t.createSelectionRenderModel = function() {
          return new a();
        };
      }, 456: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.SelectionModel = void 0, t.SelectionModel = class {
          constructor(a) {
            this._bufferService = a, this.isSelectAllActive = !1, this.selectionStartLength = 0;
          }
          clearSelection() {
            this.selectionStart = void 0, this.selectionEnd = void 0, this.isSelectAllActive = !1, this.selectionStartLength = 0;
          }
          get finalSelectionStart() {
            return this.isSelectAllActive ? [0, 0] : this.selectionEnd && this.selectionStart && this.areSelectionValuesReversed() ? this.selectionEnd : this.selectionStart;
          }
          get finalSelectionEnd() {
            if (this.isSelectAllActive) return [this._bufferService.cols, this._bufferService.buffer.ybase + this._bufferService.rows - 1];
            if (this.selectionStart) {
              if (!this.selectionEnd || this.areSelectionValuesReversed()) {
                const a = this.selectionStart[0] + this.selectionStartLength;
                return a > this._bufferService.cols ? a % this._bufferService.cols == 0 ? [this._bufferService.cols, this.selectionStart[1] + Math.floor(a / this._bufferService.cols) - 1] : [a % this._bufferService.cols, this.selectionStart[1] + Math.floor(a / this._bufferService.cols)] : [a, this.selectionStart[1]];
              }
              if (this.selectionStartLength && this.selectionEnd[1] === this.selectionStart[1]) {
                const a = this.selectionStart[0] + this.selectionStartLength;
                return a > this._bufferService.cols ? [a % this._bufferService.cols, this.selectionStart[1] + Math.floor(a / this._bufferService.cols)] : [Math.max(a, this.selectionEnd[0]), this.selectionEnd[1]];
              }
              return this.selectionEnd;
            }
          }
          areSelectionValuesReversed() {
            const a = this.selectionStart, c = this.selectionEnd;
            return !(!a || !c) && (a[1] > c[1] || a[1] === c[1] && a[0] > c[0]);
          }
          handleTrim(a) {
            return this.selectionStart && (this.selectionStart[1] -= a), this.selectionEnd && (this.selectionEnd[1] -= a), this.selectionEnd && this.selectionEnd[1] < 0 ? (this.clearSelection(), !0) : (this.selectionStart && this.selectionStart[1] < 0 && (this.selectionStart[1] = 0), !1);
          }
        };
      }, 428: function(T, t, a) {
        var c = this && this.__decorate || function(s, i, u, p) {
          var l, m = arguments.length, _ = m < 3 ? i : p === null ? p = Object.getOwnPropertyDescriptor(i, u) : p;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") _ = Reflect.decorate(s, i, u, p);
          else for (var v = s.length - 1; v >= 0; v--) (l = s[v]) && (_ = (m < 3 ? l(_) : m > 3 ? l(i, u, _) : l(i, u)) || _);
          return m > 3 && _ && Object.defineProperty(i, u, _), _;
        }, h = this && this.__param || function(s, i) {
          return function(u, p) {
            i(u, p, s);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CharSizeService = void 0;
        const r = a(2585), d = a(8460), f = a(844);
        let g = t.CharSizeService = class extends f.Disposable {
          get hasValidSize() {
            return this.width > 0 && this.height > 0;
          }
          constructor(s, i, u) {
            super(), this._optionsService = u, this.width = 0, this.height = 0, this._onCharSizeChange = this.register(new d.EventEmitter()), this.onCharSizeChange = this._onCharSizeChange.event;
            try {
              this._measureStrategy = this.register(new o(this._optionsService));
            } catch (p) {
              this._measureStrategy = this.register(new e(s, i, this._optionsService));
            }
            this.register(this._optionsService.onMultipleOptionChange(["fontFamily", "fontSize"], (() => this.measure())));
          }
          measure() {
            const s = this._measureStrategy.measure();
            s.width === this.width && s.height === this.height || (this.width = s.width, this.height = s.height, this._onCharSizeChange.fire());
          }
        };
        t.CharSizeService = g = c([h(2, r.IOptionsService)], g);
        class n extends f.Disposable {
          constructor() {
            super(...arguments), this._result = { width: 0, height: 0 };
          }
          _validateAndSet(i, u) {
            i !== void 0 && i > 0 && u !== void 0 && u > 0 && (this._result.width = i, this._result.height = u);
          }
        }
        class e extends n {
          constructor(i, u, p) {
            super(), this._document = i, this._parentElement = u, this._optionsService = p, this._measureElement = this._document.createElement("span"), this._measureElement.classList.add("xterm-char-measure-element"), this._measureElement.textContent = "W".repeat(32), this._measureElement.setAttribute("aria-hidden", "true"), this._measureElement.style.whiteSpace = "pre", this._measureElement.style.fontKerning = "none", this._parentElement.appendChild(this._measureElement);
          }
          measure() {
            return this._measureElement.style.fontFamily = this._optionsService.rawOptions.fontFamily, this._measureElement.style.fontSize = `${this._optionsService.rawOptions.fontSize}px`, this._validateAndSet(Number(this._measureElement.offsetWidth) / 32, Number(this._measureElement.offsetHeight)), this._result;
          }
        }
        class o extends n {
          constructor(i) {
            super(), this._optionsService = i, this._canvas = new OffscreenCanvas(100, 100), this._ctx = this._canvas.getContext("2d");
            const u = this._ctx.measureText("W");
            if (!("width" in u && "fontBoundingBoxAscent" in u && "fontBoundingBoxDescent" in u)) throw new Error("Required font metrics not supported");
          }
          measure() {
            this._ctx.font = `${this._optionsService.rawOptions.fontSize}px ${this._optionsService.rawOptions.fontFamily}`;
            const i = this._ctx.measureText("W");
            return this._validateAndSet(i.width, i.fontBoundingBoxAscent + i.fontBoundingBoxDescent), this._result;
          }
        }
      }, 4269: function(T, t, a) {
        var c = this && this.__decorate || function(o, s, i, u) {
          var p, l = arguments.length, m = l < 3 ? s : u === null ? u = Object.getOwnPropertyDescriptor(s, i) : u;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") m = Reflect.decorate(o, s, i, u);
          else for (var _ = o.length - 1; _ >= 0; _--) (p = o[_]) && (m = (l < 3 ? p(m) : l > 3 ? p(s, i, m) : p(s, i)) || m);
          return l > 3 && m && Object.defineProperty(s, i, m), m;
        }, h = this && this.__param || function(o, s) {
          return function(i, u) {
            s(i, u, o);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CharacterJoinerService = t.JoinedCellData = void 0;
        const r = a(3734), d = a(643), f = a(511), g = a(2585);
        class n extends r.AttributeData {
          constructor(s, i, u) {
            super(), this.content = 0, this.combinedData = "", this.fg = s.fg, this.bg = s.bg, this.combinedData = i, this._width = u;
          }
          isCombined() {
            return 2097152;
          }
          getWidth() {
            return this._width;
          }
          getChars() {
            return this.combinedData;
          }
          getCode() {
            return 2097151;
          }
          setFromCharData(s) {
            throw new Error("not implemented");
          }
          getAsCharData() {
            return [this.fg, this.getChars(), this.getWidth(), this.getCode()];
          }
        }
        t.JoinedCellData = n;
        let e = t.CharacterJoinerService = class qe {
          constructor(s) {
            this._bufferService = s, this._characterJoiners = [], this._nextCharacterJoinerId = 0, this._workCell = new f.CellData();
          }
          register(s) {
            const i = { id: this._nextCharacterJoinerId++, handler: s };
            return this._characterJoiners.push(i), i.id;
          }
          deregister(s) {
            for (let i = 0; i < this._characterJoiners.length; i++) if (this._characterJoiners[i].id === s) return this._characterJoiners.splice(i, 1), !0;
            return !1;
          }
          getJoinedCharacters(s) {
            if (this._characterJoiners.length === 0) return [];
            const i = this._bufferService.buffer.lines.get(s);
            if (!i || i.length === 0) return [];
            const u = [], p = i.translateToString(!0);
            let l = 0, m = 0, _ = 0, v = i.getFg(0), C = i.getBg(0);
            for (let w = 0; w < i.getTrimmedLength(); w++) if (i.loadCell(w, this._workCell), this._workCell.getWidth() !== 0) {
              if (this._workCell.fg !== v || this._workCell.bg !== C) {
                if (w - l > 1) {
                  const S = this._getJoinedRanges(p, _, m, i, l);
                  for (let b = 0; b < S.length; b++) u.push(S[b]);
                }
                l = w, _ = m, v = this._workCell.fg, C = this._workCell.bg;
              }
              m += this._workCell.getChars().length || d.WHITESPACE_CELL_CHAR.length;
            }
            if (this._bufferService.cols - l > 1) {
              const w = this._getJoinedRanges(p, _, m, i, l);
              for (let S = 0; S < w.length; S++) u.push(w[S]);
            }
            return u;
          }
          _getJoinedRanges(s, i, u, p, l) {
            const m = s.substring(i, u);
            let _ = [];
            try {
              _ = this._characterJoiners[0].handler(m);
            } catch (v) {
              console.error(v);
            }
            for (let v = 1; v < this._characterJoiners.length; v++) try {
              const C = this._characterJoiners[v].handler(m);
              for (let w = 0; w < C.length; w++) qe._mergeRanges(_, C[w]);
            } catch (C) {
              console.error(C);
            }
            return this._stringRangesToCellRanges(_, p, l), _;
          }
          _stringRangesToCellRanges(s, i, u) {
            let p = 0, l = !1, m = 0, _ = s[p];
            if (_) {
              for (let v = u; v < this._bufferService.cols; v++) {
                const C = i.getWidth(v), w = i.getString(v).length || d.WHITESPACE_CELL_CHAR.length;
                if (C !== 0) {
                  if (!l && _[0] <= m && (_[0] = v, l = !0), _[1] <= m) {
                    if (_[1] = v, _ = s[++p], !_) break;
                    _[0] <= m ? (_[0] = v, l = !0) : l = !1;
                  }
                  m += w;
                }
              }
              _ && (_[1] = this._bufferService.cols);
            }
          }
          static _mergeRanges(s, i) {
            let u = !1;
            for (let p = 0; p < s.length; p++) {
              const l = s[p];
              if (u) {
                if (i[1] <= l[0]) return s[p - 1][1] = i[1], s;
                if (i[1] <= l[1]) return s[p - 1][1] = Math.max(i[1], l[1]), s.splice(p, 1), s;
                s.splice(p, 1), p--;
              } else {
                if (i[1] <= l[0]) return s.splice(p, 0, i), s;
                if (i[1] <= l[1]) return l[0] = Math.min(i[0], l[0]), s;
                i[0] < l[1] && (l[0] = Math.min(i[0], l[0]), u = !0);
              }
            }
            return u ? s[s.length - 1][1] = i[1] : s.push(i), s;
          }
        };
        t.CharacterJoinerService = e = c([h(0, g.IBufferService)], e);
      }, 5114: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CoreBrowserService = void 0;
        const c = a(844), h = a(8460), r = a(3656);
        class d extends c.Disposable {
          constructor(n, e, o) {
            super(), this._textarea = n, this._window = e, this.mainDocument = o, this._isFocused = !1, this._cachedIsFocused = void 0, this._screenDprMonitor = new f(this._window), this._onDprChange = this.register(new h.EventEmitter()), this.onDprChange = this._onDprChange.event, this._onWindowChange = this.register(new h.EventEmitter()), this.onWindowChange = this._onWindowChange.event, this.register(this.onWindowChange(((s) => this._screenDprMonitor.setWindow(s)))), this.register((0, h.forwardEvent)(this._screenDprMonitor.onDprChange, this._onDprChange)), this._textarea.addEventListener("focus", (() => this._isFocused = !0)), this._textarea.addEventListener("blur", (() => this._isFocused = !1));
          }
          get window() {
            return this._window;
          }
          set window(n) {
            this._window !== n && (this._window = n, this._onWindowChange.fire(this._window));
          }
          get dpr() {
            return this.window.devicePixelRatio;
          }
          get isFocused() {
            return this._cachedIsFocused === void 0 && (this._cachedIsFocused = this._isFocused && this._textarea.ownerDocument.hasFocus(), queueMicrotask((() => this._cachedIsFocused = void 0))), this._cachedIsFocused;
          }
        }
        t.CoreBrowserService = d;
        class f extends c.Disposable {
          constructor(n) {
            super(), this._parentWindow = n, this._windowResizeListener = this.register(new c.MutableDisposable()), this._onDprChange = this.register(new h.EventEmitter()), this.onDprChange = this._onDprChange.event, this._outerListener = () => this._setDprAndFireIfDiffers(), this._currentDevicePixelRatio = this._parentWindow.devicePixelRatio, this._updateDpr(), this._setWindowResizeListener(), this.register((0, c.toDisposable)((() => this.clearListener())));
          }
          setWindow(n) {
            this._parentWindow = n, this._setWindowResizeListener(), this._setDprAndFireIfDiffers();
          }
          _setWindowResizeListener() {
            this._windowResizeListener.value = (0, r.addDisposableDomListener)(this._parentWindow, "resize", (() => this._setDprAndFireIfDiffers()));
          }
          _setDprAndFireIfDiffers() {
            this._parentWindow.devicePixelRatio !== this._currentDevicePixelRatio && this._onDprChange.fire(this._parentWindow.devicePixelRatio), this._updateDpr();
          }
          _updateDpr() {
            var n;
            this._outerListener && ((n = this._resolutionMediaMatchList) == null || n.removeListener(this._outerListener), this._currentDevicePixelRatio = this._parentWindow.devicePixelRatio, this._resolutionMediaMatchList = this._parentWindow.matchMedia(`screen and (resolution: ${this._parentWindow.devicePixelRatio}dppx)`), this._resolutionMediaMatchList.addListener(this._outerListener));
          }
          clearListener() {
            this._resolutionMediaMatchList && this._outerListener && (this._resolutionMediaMatchList.removeListener(this._outerListener), this._resolutionMediaMatchList = void 0, this._outerListener = void 0);
          }
        }
      }, 779: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.LinkProviderService = void 0;
        const c = a(844);
        class h extends c.Disposable {
          constructor() {
            super(), this.linkProviders = [], this.register((0, c.toDisposable)((() => this.linkProviders.length = 0)));
          }
          registerLinkProvider(d) {
            return this.linkProviders.push(d), { dispose: () => {
              const f = this.linkProviders.indexOf(d);
              f !== -1 && this.linkProviders.splice(f, 1);
            } };
          }
        }
        t.LinkProviderService = h;
      }, 8934: function(T, t, a) {
        var c = this && this.__decorate || function(g, n, e, o) {
          var s, i = arguments.length, u = i < 3 ? n : o === null ? o = Object.getOwnPropertyDescriptor(n, e) : o;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") u = Reflect.decorate(g, n, e, o);
          else for (var p = g.length - 1; p >= 0; p--) (s = g[p]) && (u = (i < 3 ? s(u) : i > 3 ? s(n, e, u) : s(n, e)) || u);
          return i > 3 && u && Object.defineProperty(n, e, u), u;
        }, h = this && this.__param || function(g, n) {
          return function(e, o) {
            n(e, o, g);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.MouseService = void 0;
        const r = a(4725), d = a(9806);
        let f = t.MouseService = class {
          constructor(g, n) {
            this._renderService = g, this._charSizeService = n;
          }
          getCoords(g, n, e, o, s) {
            return (0, d.getCoords)(window, g, n, e, o, this._charSizeService.hasValidSize, this._renderService.dimensions.css.cell.width, this._renderService.dimensions.css.cell.height, s);
          }
          getMouseReportCoords(g, n) {
            const e = (0, d.getCoordsRelativeToElement)(window, g, n);
            if (this._charSizeService.hasValidSize) return e[0] = Math.min(Math.max(e[0], 0), this._renderService.dimensions.css.canvas.width - 1), e[1] = Math.min(Math.max(e[1], 0), this._renderService.dimensions.css.canvas.height - 1), { col: Math.floor(e[0] / this._renderService.dimensions.css.cell.width), row: Math.floor(e[1] / this._renderService.dimensions.css.cell.height), x: Math.floor(e[0]), y: Math.floor(e[1]) };
          }
        };
        t.MouseService = f = c([h(0, r.IRenderService), h(1, r.ICharSizeService)], f);
      }, 3230: function(T, t, a) {
        var c = this && this.__decorate || function(s, i, u, p) {
          var l, m = arguments.length, _ = m < 3 ? i : p === null ? p = Object.getOwnPropertyDescriptor(i, u) : p;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") _ = Reflect.decorate(s, i, u, p);
          else for (var v = s.length - 1; v >= 0; v--) (l = s[v]) && (_ = (m < 3 ? l(_) : m > 3 ? l(i, u, _) : l(i, u)) || _);
          return m > 3 && _ && Object.defineProperty(i, u, _), _;
        }, h = this && this.__param || function(s, i) {
          return function(u, p) {
            i(u, p, s);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.RenderService = void 0;
        const r = a(6193), d = a(4725), f = a(8460), g = a(844), n = a(7226), e = a(2585);
        let o = t.RenderService = class extends g.Disposable {
          get dimensions() {
            return this._renderer.value.dimensions;
          }
          constructor(s, i, u, p, l, m, _, v) {
            super(), this._rowCount = s, this._charSizeService = p, this._renderer = this.register(new g.MutableDisposable()), this._pausedResizeTask = new n.DebouncedIdleTask(), this._observerDisposable = this.register(new g.MutableDisposable()), this._isPaused = !1, this._needsFullRefresh = !1, this._isNextRenderRedrawOnly = !0, this._needsSelectionRefresh = !1, this._canvasWidth = 0, this._canvasHeight = 0, this._selectionState = { start: void 0, end: void 0, columnSelectMode: !1 }, this._onDimensionsChange = this.register(new f.EventEmitter()), this.onDimensionsChange = this._onDimensionsChange.event, this._onRenderedViewportChange = this.register(new f.EventEmitter()), this.onRenderedViewportChange = this._onRenderedViewportChange.event, this._onRender = this.register(new f.EventEmitter()), this.onRender = this._onRender.event, this._onRefreshRequest = this.register(new f.EventEmitter()), this.onRefreshRequest = this._onRefreshRequest.event, this._renderDebouncer = new r.RenderDebouncer(((C, w) => this._renderRows(C, w)), _), this.register(this._renderDebouncer), this.register(_.onDprChange((() => this.handleDevicePixelRatioChange()))), this.register(m.onResize((() => this._fullRefresh()))), this.register(m.buffers.onBufferActivate((() => {
              var C;
              return (C = this._renderer.value) == null ? void 0 : C.clear();
            }))), this.register(u.onOptionChange((() => this._handleOptionsChanged()))), this.register(this._charSizeService.onCharSizeChange((() => this.handleCharSizeChanged()))), this.register(l.onDecorationRegistered((() => this._fullRefresh()))), this.register(l.onDecorationRemoved((() => this._fullRefresh()))), this.register(u.onMultipleOptionChange(["customGlyphs", "drawBoldTextInBrightColors", "letterSpacing", "lineHeight", "fontFamily", "fontSize", "fontWeight", "fontWeightBold", "minimumContrastRatio", "rescaleOverlappingGlyphs"], (() => {
              this.clear(), this.handleResize(m.cols, m.rows), this._fullRefresh();
            }))), this.register(u.onMultipleOptionChange(["cursorBlink", "cursorStyle"], (() => this.refreshRows(m.buffer.y, m.buffer.y, !0)))), this.register(v.onChangeColors((() => this._fullRefresh()))), this._registerIntersectionObserver(_.window, i), this.register(_.onWindowChange(((C) => this._registerIntersectionObserver(C, i))));
          }
          _registerIntersectionObserver(s, i) {
            if ("IntersectionObserver" in s) {
              const u = new s.IntersectionObserver(((p) => this._handleIntersectionChange(p[p.length - 1])), { threshold: 0 });
              u.observe(i), this._observerDisposable.value = (0, g.toDisposable)((() => u.disconnect()));
            }
          }
          _handleIntersectionChange(s) {
            this._isPaused = s.isIntersecting === void 0 ? s.intersectionRatio === 0 : !s.isIntersecting, this._isPaused || this._charSizeService.hasValidSize || this._charSizeService.measure(), !this._isPaused && this._needsFullRefresh && (this._pausedResizeTask.flush(), this.refreshRows(0, this._rowCount - 1), this._needsFullRefresh = !1);
          }
          refreshRows(s, i, u = !1) {
            this._isPaused ? this._needsFullRefresh = !0 : (u || (this._isNextRenderRedrawOnly = !1), this._renderDebouncer.refresh(s, i, this._rowCount));
          }
          _renderRows(s, i) {
            this._renderer.value && (s = Math.min(s, this._rowCount - 1), i = Math.min(i, this._rowCount - 1), this._renderer.value.renderRows(s, i), this._needsSelectionRefresh && (this._renderer.value.handleSelectionChanged(this._selectionState.start, this._selectionState.end, this._selectionState.columnSelectMode), this._needsSelectionRefresh = !1), this._isNextRenderRedrawOnly || this._onRenderedViewportChange.fire({ start: s, end: i }), this._onRender.fire({ start: s, end: i }), this._isNextRenderRedrawOnly = !0);
          }
          resize(s, i) {
            this._rowCount = i, this._fireOnCanvasResize();
          }
          _handleOptionsChanged() {
            this._renderer.value && (this.refreshRows(0, this._rowCount - 1), this._fireOnCanvasResize());
          }
          _fireOnCanvasResize() {
            this._renderer.value && (this._renderer.value.dimensions.css.canvas.width === this._canvasWidth && this._renderer.value.dimensions.css.canvas.height === this._canvasHeight || this._onDimensionsChange.fire(this._renderer.value.dimensions));
          }
          hasRenderer() {
            return !!this._renderer.value;
          }
          setRenderer(s) {
            this._renderer.value = s, this._renderer.value && (this._renderer.value.onRequestRedraw(((i) => this.refreshRows(i.start, i.end, !0))), this._needsSelectionRefresh = !0, this._fullRefresh());
          }
          addRefreshCallback(s) {
            return this._renderDebouncer.addRefreshCallback(s);
          }
          _fullRefresh() {
            this._isPaused ? this._needsFullRefresh = !0 : this.refreshRows(0, this._rowCount - 1);
          }
          clearTextureAtlas() {
            var s, i;
            this._renderer.value && ((i = (s = this._renderer.value).clearTextureAtlas) == null || i.call(s), this._fullRefresh());
          }
          handleDevicePixelRatioChange() {
            this._charSizeService.measure(), this._renderer.value && (this._renderer.value.handleDevicePixelRatioChange(), this.refreshRows(0, this._rowCount - 1));
          }
          handleResize(s, i) {
            this._renderer.value && (this._isPaused ? this._pausedResizeTask.set((() => {
              var u;
              return (u = this._renderer.value) == null ? void 0 : u.handleResize(s, i);
            })) : this._renderer.value.handleResize(s, i), this._fullRefresh());
          }
          handleCharSizeChanged() {
            var s;
            (s = this._renderer.value) == null || s.handleCharSizeChanged();
          }
          handleBlur() {
            var s;
            (s = this._renderer.value) == null || s.handleBlur();
          }
          handleFocus() {
            var s;
            (s = this._renderer.value) == null || s.handleFocus();
          }
          handleSelectionChanged(s, i, u) {
            var p;
            this._selectionState.start = s, this._selectionState.end = i, this._selectionState.columnSelectMode = u, (p = this._renderer.value) == null || p.handleSelectionChanged(s, i, u);
          }
          handleCursorMove() {
            var s;
            (s = this._renderer.value) == null || s.handleCursorMove();
          }
          clear() {
            var s;
            (s = this._renderer.value) == null || s.clear();
          }
        };
        t.RenderService = o = c([h(2, e.IOptionsService), h(3, d.ICharSizeService), h(4, e.IDecorationService), h(5, e.IBufferService), h(6, d.ICoreBrowserService), h(7, d.IThemeService)], o);
      }, 9312: function(T, t, a) {
        var c = this && this.__decorate || function(_, v, C, w) {
          var S, b = arguments.length, x = b < 3 ? v : w === null ? w = Object.getOwnPropertyDescriptor(v, C) : w;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") x = Reflect.decorate(_, v, C, w);
          else for (var A = _.length - 1; A >= 0; A--) (S = _[A]) && (x = (b < 3 ? S(x) : b > 3 ? S(v, C, x) : S(v, C)) || x);
          return b > 3 && x && Object.defineProperty(v, C, x), x;
        }, h = this && this.__param || function(_, v) {
          return function(C, w) {
            v(C, w, _);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.SelectionService = void 0;
        const r = a(9806), d = a(9504), f = a(456), g = a(4725), n = a(8460), e = a(844), o = a(6114), s = a(4841), i = a(511), u = a(2585), p = " ", l = new RegExp(p, "g");
        let m = t.SelectionService = class extends e.Disposable {
          constructor(_, v, C, w, S, b, x, A, P) {
            super(), this._element = _, this._screenElement = v, this._linkifier = C, this._bufferService = w, this._coreService = S, this._mouseService = b, this._optionsService = x, this._renderService = A, this._coreBrowserService = P, this._dragScrollAmount = 0, this._enabled = !0, this._workCell = new i.CellData(), this._mouseDownTimeStamp = 0, this._oldHasSelection = !1, this._oldSelectionStart = void 0, this._oldSelectionEnd = void 0, this._onLinuxMouseSelection = this.register(new n.EventEmitter()), this.onLinuxMouseSelection = this._onLinuxMouseSelection.event, this._onRedrawRequest = this.register(new n.EventEmitter()), this.onRequestRedraw = this._onRedrawRequest.event, this._onSelectionChange = this.register(new n.EventEmitter()), this.onSelectionChange = this._onSelectionChange.event, this._onRequestScrollLines = this.register(new n.EventEmitter()), this.onRequestScrollLines = this._onRequestScrollLines.event, this._mouseMoveListener = (k) => this._handleMouseMove(k), this._mouseUpListener = (k) => this._handleMouseUp(k), this._coreService.onUserInput((() => {
              this.hasSelection && this.clearSelection();
            })), this._trimListener = this._bufferService.buffer.lines.onTrim(((k) => this._handleTrim(k))), this.register(this._bufferService.buffers.onBufferActivate(((k) => this._handleBufferActivate(k)))), this.enable(), this._model = new f.SelectionModel(this._bufferService), this._activeSelectionMode = 0, this.register((0, e.toDisposable)((() => {
              this._removeMouseDownListeners();
            })));
          }
          reset() {
            this.clearSelection();
          }
          disable() {
            this.clearSelection(), this._enabled = !1;
          }
          enable() {
            this._enabled = !0;
          }
          get selectionStart() {
            return this._model.finalSelectionStart;
          }
          get selectionEnd() {
            return this._model.finalSelectionEnd;
          }
          get hasSelection() {
            const _ = this._model.finalSelectionStart, v = this._model.finalSelectionEnd;
            return !(!_ || !v || _[0] === v[0] && _[1] === v[1]);
          }
          get selectionText() {
            const _ = this._model.finalSelectionStart, v = this._model.finalSelectionEnd;
            if (!_ || !v) return "";
            const C = this._bufferService.buffer, w = [];
            if (this._activeSelectionMode === 3) {
              if (_[0] === v[0]) return "";
              const S = _[0] < v[0] ? _[0] : v[0], b = _[0] < v[0] ? v[0] : _[0];
              for (let x = _[1]; x <= v[1]; x++) {
                const A = C.translateBufferLineToString(x, !0, S, b);
                w.push(A);
              }
            } else {
              const S = _[1] === v[1] ? v[0] : void 0;
              w.push(C.translateBufferLineToString(_[1], !0, _[0], S));
              for (let b = _[1] + 1; b <= v[1] - 1; b++) {
                const x = C.lines.get(b), A = C.translateBufferLineToString(b, !0);
                x != null && x.isWrapped ? w[w.length - 1] += A : w.push(A);
              }
              if (_[1] !== v[1]) {
                const b = C.lines.get(v[1]), x = C.translateBufferLineToString(v[1], !0, 0, v[0]);
                b && b.isWrapped ? w[w.length - 1] += x : w.push(x);
              }
            }
            return w.map(((S) => S.replace(l, " "))).join(o.isWindows ? `\r
` : `
`);
          }
          clearSelection() {
            this._model.clearSelection(), this._removeMouseDownListeners(), this.refresh(), this._onSelectionChange.fire();
          }
          refresh(_) {
            this._refreshAnimationFrame || (this._refreshAnimationFrame = this._coreBrowserService.window.requestAnimationFrame((() => this._refresh()))), o.isLinux && _ && this.selectionText.length && this._onLinuxMouseSelection.fire(this.selectionText);
          }
          _refresh() {
            this._refreshAnimationFrame = void 0, this._onRedrawRequest.fire({ start: this._model.finalSelectionStart, end: this._model.finalSelectionEnd, columnSelectMode: this._activeSelectionMode === 3 });
          }
          _isClickInSelection(_) {
            const v = this._getMouseBufferCoords(_), C = this._model.finalSelectionStart, w = this._model.finalSelectionEnd;
            return !!(C && w && v) && this._areCoordsInSelection(v, C, w);
          }
          isCellInSelection(_, v) {
            const C = this._model.finalSelectionStart, w = this._model.finalSelectionEnd;
            return !(!C || !w) && this._areCoordsInSelection([_, v], C, w);
          }
          _areCoordsInSelection(_, v, C) {
            return _[1] > v[1] && _[1] < C[1] || v[1] === C[1] && _[1] === v[1] && _[0] >= v[0] && _[0] < C[0] || v[1] < C[1] && _[1] === C[1] && _[0] < C[0] || v[1] < C[1] && _[1] === v[1] && _[0] >= v[0];
          }
          _selectWordAtCursor(_, v) {
            var S, b;
            const C = (b = (S = this._linkifier.currentLink) == null ? void 0 : S.link) == null ? void 0 : b.range;
            if (C) return this._model.selectionStart = [C.start.x - 1, C.start.y - 1], this._model.selectionStartLength = (0, s.getRangeLength)(C, this._bufferService.cols), this._model.selectionEnd = void 0, !0;
            const w = this._getMouseBufferCoords(_);
            return !!w && (this._selectWordAt(w, v), this._model.selectionEnd = void 0, !0);
          }
          selectAll() {
            this._model.isSelectAllActive = !0, this.refresh(), this._onSelectionChange.fire();
          }
          selectLines(_, v) {
            this._model.clearSelection(), _ = Math.max(_, 0), v = Math.min(v, this._bufferService.buffer.lines.length - 1), this._model.selectionStart = [0, _], this._model.selectionEnd = [this._bufferService.cols, v], this.refresh(), this._onSelectionChange.fire();
          }
          _handleTrim(_) {
            this._model.handleTrim(_) && this.refresh();
          }
          _getMouseBufferCoords(_) {
            const v = this._mouseService.getCoords(_, this._screenElement, this._bufferService.cols, this._bufferService.rows, !0);
            if (v) return v[0]--, v[1]--, v[1] += this._bufferService.buffer.ydisp, v;
          }
          _getMouseEventScrollAmount(_) {
            let v = (0, r.getCoordsRelativeToElement)(this._coreBrowserService.window, _, this._screenElement)[1];
            const C = this._renderService.dimensions.css.canvas.height;
            return v >= 0 && v <= C ? 0 : (v > C && (v -= C), v = Math.min(Math.max(v, -50), 50), v /= 50, v / Math.abs(v) + Math.round(14 * v));
          }
          shouldForceSelection(_) {
            return o.isMac ? _.altKey && this._optionsService.rawOptions.macOptionClickForcesSelection : _.shiftKey;
          }
          handleMouseDown(_) {
            if (this._mouseDownTimeStamp = _.timeStamp, (_.button !== 2 || !this.hasSelection) && _.button === 0) {
              if (!this._enabled) {
                if (!this.shouldForceSelection(_)) return;
                _.stopPropagation();
              }
              _.preventDefault(), this._dragScrollAmount = 0, this._enabled && _.shiftKey ? this._handleIncrementalClick(_) : _.detail === 1 ? this._handleSingleClick(_) : _.detail === 2 ? this._handleDoubleClick(_) : _.detail === 3 && this._handleTripleClick(_), this._addMouseDownListeners(), this.refresh(!0);
            }
          }
          _addMouseDownListeners() {
            this._screenElement.ownerDocument && (this._screenElement.ownerDocument.addEventListener("mousemove", this._mouseMoveListener), this._screenElement.ownerDocument.addEventListener("mouseup", this._mouseUpListener)), this._dragScrollIntervalTimer = this._coreBrowserService.window.setInterval((() => this._dragScroll()), 50);
          }
          _removeMouseDownListeners() {
            this._screenElement.ownerDocument && (this._screenElement.ownerDocument.removeEventListener("mousemove", this._mouseMoveListener), this._screenElement.ownerDocument.removeEventListener("mouseup", this._mouseUpListener)), this._coreBrowserService.window.clearInterval(this._dragScrollIntervalTimer), this._dragScrollIntervalTimer = void 0;
          }
          _handleIncrementalClick(_) {
            this._model.selectionStart && (this._model.selectionEnd = this._getMouseBufferCoords(_));
          }
          _handleSingleClick(_) {
            if (this._model.selectionStartLength = 0, this._model.isSelectAllActive = !1, this._activeSelectionMode = this.shouldColumnSelect(_) ? 3 : 0, this._model.selectionStart = this._getMouseBufferCoords(_), !this._model.selectionStart) return;
            this._model.selectionEnd = void 0;
            const v = this._bufferService.buffer.lines.get(this._model.selectionStart[1]);
            v && v.length !== this._model.selectionStart[0] && v.hasWidth(this._model.selectionStart[0]) === 0 && this._model.selectionStart[0]++;
          }
          _handleDoubleClick(_) {
            this._selectWordAtCursor(_, !0) && (this._activeSelectionMode = 1);
          }
          _handleTripleClick(_) {
            const v = this._getMouseBufferCoords(_);
            v && (this._activeSelectionMode = 2, this._selectLineAt(v[1]));
          }
          shouldColumnSelect(_) {
            return _.altKey && !(o.isMac && this._optionsService.rawOptions.macOptionClickForcesSelection);
          }
          _handleMouseMove(_) {
            if (_.stopImmediatePropagation(), !this._model.selectionStart) return;
            const v = this._model.selectionEnd ? [this._model.selectionEnd[0], this._model.selectionEnd[1]] : null;
            if (this._model.selectionEnd = this._getMouseBufferCoords(_), !this._model.selectionEnd) return void this.refresh(!0);
            this._activeSelectionMode === 2 ? this._model.selectionEnd[1] < this._model.selectionStart[1] ? this._model.selectionEnd[0] = 0 : this._model.selectionEnd[0] = this._bufferService.cols : this._activeSelectionMode === 1 && this._selectToWordAt(this._model.selectionEnd), this._dragScrollAmount = this._getMouseEventScrollAmount(_), this._activeSelectionMode !== 3 && (this._dragScrollAmount > 0 ? this._model.selectionEnd[0] = this._bufferService.cols : this._dragScrollAmount < 0 && (this._model.selectionEnd[0] = 0));
            const C = this._bufferService.buffer;
            if (this._model.selectionEnd[1] < C.lines.length) {
              const w = C.lines.get(this._model.selectionEnd[1]);
              w && w.hasWidth(this._model.selectionEnd[0]) === 0 && this._model.selectionEnd[0] < this._bufferService.cols && this._model.selectionEnd[0]++;
            }
            v && v[0] === this._model.selectionEnd[0] && v[1] === this._model.selectionEnd[1] || this.refresh(!0);
          }
          _dragScroll() {
            if (this._model.selectionEnd && this._model.selectionStart && this._dragScrollAmount) {
              this._onRequestScrollLines.fire({ amount: this._dragScrollAmount, suppressScrollEvent: !1 });
              const _ = this._bufferService.buffer;
              this._dragScrollAmount > 0 ? (this._activeSelectionMode !== 3 && (this._model.selectionEnd[0] = this._bufferService.cols), this._model.selectionEnd[1] = Math.min(_.ydisp + this._bufferService.rows, _.lines.length - 1)) : (this._activeSelectionMode !== 3 && (this._model.selectionEnd[0] = 0), this._model.selectionEnd[1] = _.ydisp), this.refresh();
            }
          }
          _handleMouseUp(_) {
            const v = _.timeStamp - this._mouseDownTimeStamp;
            if (this._removeMouseDownListeners(), this.selectionText.length <= 1 && v < 500 && _.altKey && this._optionsService.rawOptions.altClickMovesCursor) {
              if (this._bufferService.buffer.ybase === this._bufferService.buffer.ydisp) {
                const C = this._mouseService.getCoords(_, this._element, this._bufferService.cols, this._bufferService.rows, !1);
                if (C && C[0] !== void 0 && C[1] !== void 0) {
                  const w = (0, d.moveToCellSequence)(C[0] - 1, C[1] - 1, this._bufferService, this._coreService.decPrivateModes.applicationCursorKeys);
                  this._coreService.triggerDataEvent(w, !0);
                }
              }
            } else this._fireEventIfSelectionChanged();
          }
          _fireEventIfSelectionChanged() {
            const _ = this._model.finalSelectionStart, v = this._model.finalSelectionEnd, C = !(!_ || !v || _[0] === v[0] && _[1] === v[1]);
            C ? _ && v && (this._oldSelectionStart && this._oldSelectionEnd && _[0] === this._oldSelectionStart[0] && _[1] === this._oldSelectionStart[1] && v[0] === this._oldSelectionEnd[0] && v[1] === this._oldSelectionEnd[1] || this._fireOnSelectionChange(_, v, C)) : this._oldHasSelection && this._fireOnSelectionChange(_, v, C);
          }
          _fireOnSelectionChange(_, v, C) {
            this._oldSelectionStart = _, this._oldSelectionEnd = v, this._oldHasSelection = C, this._onSelectionChange.fire();
          }
          _handleBufferActivate(_) {
            this.clearSelection(), this._trimListener.dispose(), this._trimListener = _.activeBuffer.lines.onTrim(((v) => this._handleTrim(v)));
          }
          _convertViewportColToCharacterIndex(_, v) {
            let C = v;
            for (let w = 0; v >= w; w++) {
              const S = _.loadCell(w, this._workCell).getChars().length;
              this._workCell.getWidth() === 0 ? C-- : S > 1 && v !== w && (C += S - 1);
            }
            return C;
          }
          setSelection(_, v, C) {
            this._model.clearSelection(), this._removeMouseDownListeners(), this._model.selectionStart = [_, v], this._model.selectionStartLength = C, this.refresh(), this._fireEventIfSelectionChanged();
          }
          rightClickSelect(_) {
            this._isClickInSelection(_) || (this._selectWordAtCursor(_, !1) && this.refresh(!0), this._fireEventIfSelectionChanged());
          }
          _getWordAt(_, v, C = !0, w = !0) {
            if (_[0] >= this._bufferService.cols) return;
            const S = this._bufferService.buffer, b = S.lines.get(_[1]);
            if (!b) return;
            const x = S.translateBufferLineToString(_[1], !1);
            let A = this._convertViewportColToCharacterIndex(b, _[0]), P = A;
            const k = _[0] - A;
            let M = 0, y = 0, L = 0, R = 0;
            if (x.charAt(A) === " ") {
              for (; A > 0 && x.charAt(A - 1) === " "; ) A--;
              for (; P < x.length && x.charAt(P + 1) === " "; ) P++;
            } else {
              let U = _[0], K = _[0];
              b.getWidth(U) === 0 && (M++, U--), b.getWidth(K) === 2 && (y++, K++);
              const q = b.getString(K).length;
              for (q > 1 && (R += q - 1, P += q - 1); U > 0 && A > 0 && !this._isCharWordSeparator(b.loadCell(U - 1, this._workCell)); ) {
                b.loadCell(U - 1, this._workCell);
                const O = this._workCell.getChars().length;
                this._workCell.getWidth() === 0 ? (M++, U--) : O > 1 && (L += O - 1, A -= O - 1), A--, U--;
              }
              for (; K < b.length && P + 1 < x.length && !this._isCharWordSeparator(b.loadCell(K + 1, this._workCell)); ) {
                b.loadCell(K + 1, this._workCell);
                const O = this._workCell.getChars().length;
                this._workCell.getWidth() === 2 ? (y++, K++) : O > 1 && (R += O - 1, P += O - 1), P++, K++;
              }
            }
            P++;
            let D = A + k - M + L, F = Math.min(this._bufferService.cols, P - A + M + y - L - R);
            if (v || x.slice(A, P).trim() !== "") {
              if (C && D === 0 && b.getCodePoint(0) !== 32) {
                const U = S.lines.get(_[1] - 1);
                if (U && b.isWrapped && U.getCodePoint(this._bufferService.cols - 1) !== 32) {
                  const K = this._getWordAt([this._bufferService.cols - 1, _[1] - 1], !1, !0, !1);
                  if (K) {
                    const q = this._bufferService.cols - K.start;
                    D -= q, F += q;
                  }
                }
              }
              if (w && D + F === this._bufferService.cols && b.getCodePoint(this._bufferService.cols - 1) !== 32) {
                const U = S.lines.get(_[1] + 1);
                if (U != null && U.isWrapped && U.getCodePoint(0) !== 32) {
                  const K = this._getWordAt([0, _[1] + 1], !1, !1, !0);
                  K && (F += K.length);
                }
              }
              return { start: D, length: F };
            }
          }
          _selectWordAt(_, v) {
            const C = this._getWordAt(_, v);
            if (C) {
              for (; C.start < 0; ) C.start += this._bufferService.cols, _[1]--;
              this._model.selectionStart = [C.start, _[1]], this._model.selectionStartLength = C.length;
            }
          }
          _selectToWordAt(_) {
            const v = this._getWordAt(_, !0);
            if (v) {
              let C = _[1];
              for (; v.start < 0; ) v.start += this._bufferService.cols, C--;
              if (!this._model.areSelectionValuesReversed()) for (; v.start + v.length > this._bufferService.cols; ) v.length -= this._bufferService.cols, C++;
              this._model.selectionEnd = [this._model.areSelectionValuesReversed() ? v.start : v.start + v.length, C];
            }
          }
          _isCharWordSeparator(_) {
            return _.getWidth() !== 0 && this._optionsService.rawOptions.wordSeparator.indexOf(_.getChars()) >= 0;
          }
          _selectLineAt(_) {
            const v = this._bufferService.buffer.getWrappedRangeForLine(_), C = { start: { x: 0, y: v.first }, end: { x: this._bufferService.cols - 1, y: v.last } };
            this._model.selectionStart = [0, v.first], this._model.selectionEnd = void 0, this._model.selectionStartLength = (0, s.getRangeLength)(C, this._bufferService.cols);
          }
        };
        t.SelectionService = m = c([h(3, u.IBufferService), h(4, u.ICoreService), h(5, g.IMouseService), h(6, u.IOptionsService), h(7, g.IRenderService), h(8, g.ICoreBrowserService)], m);
      }, 4725: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.ILinkProviderService = t.IThemeService = t.ICharacterJoinerService = t.ISelectionService = t.IRenderService = t.IMouseService = t.ICoreBrowserService = t.ICharSizeService = void 0;
        const c = a(8343);
        t.ICharSizeService = (0, c.createDecorator)("CharSizeService"), t.ICoreBrowserService = (0, c.createDecorator)("CoreBrowserService"), t.IMouseService = (0, c.createDecorator)("MouseService"), t.IRenderService = (0, c.createDecorator)("RenderService"), t.ISelectionService = (0, c.createDecorator)("SelectionService"), t.ICharacterJoinerService = (0, c.createDecorator)("CharacterJoinerService"), t.IThemeService = (0, c.createDecorator)("ThemeService"), t.ILinkProviderService = (0, c.createDecorator)("LinkProviderService");
      }, 6731: function(T, t, a) {
        var c = this && this.__decorate || function(m, _, v, C) {
          var w, S = arguments.length, b = S < 3 ? _ : C === null ? C = Object.getOwnPropertyDescriptor(_, v) : C;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") b = Reflect.decorate(m, _, v, C);
          else for (var x = m.length - 1; x >= 0; x--) (w = m[x]) && (b = (S < 3 ? w(b) : S > 3 ? w(_, v, b) : w(_, v)) || b);
          return S > 3 && b && Object.defineProperty(_, v, b), b;
        }, h = this && this.__param || function(m, _) {
          return function(v, C) {
            _(v, C, m);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.ThemeService = t.DEFAULT_ANSI_COLORS = void 0;
        const r = a(7239), d = a(8055), f = a(8460), g = a(844), n = a(2585), e = d.css.toColor("#ffffff"), o = d.css.toColor("#000000"), s = d.css.toColor("#ffffff"), i = d.css.toColor("#000000"), u = { css: "rgba(255, 255, 255, 0.3)", rgba: 4294967117 };
        t.DEFAULT_ANSI_COLORS = Object.freeze((() => {
          const m = [d.css.toColor("#2e3436"), d.css.toColor("#cc0000"), d.css.toColor("#4e9a06"), d.css.toColor("#c4a000"), d.css.toColor("#3465a4"), d.css.toColor("#75507b"), d.css.toColor("#06989a"), d.css.toColor("#d3d7cf"), d.css.toColor("#555753"), d.css.toColor("#ef2929"), d.css.toColor("#8ae234"), d.css.toColor("#fce94f"), d.css.toColor("#729fcf"), d.css.toColor("#ad7fa8"), d.css.toColor("#34e2e2"), d.css.toColor("#eeeeec")], _ = [0, 95, 135, 175, 215, 255];
          for (let v = 0; v < 216; v++) {
            const C = _[v / 36 % 6 | 0], w = _[v / 6 % 6 | 0], S = _[v % 6];
            m.push({ css: d.channels.toCss(C, w, S), rgba: d.channels.toRgba(C, w, S) });
          }
          for (let v = 0; v < 24; v++) {
            const C = 8 + 10 * v;
            m.push({ css: d.channels.toCss(C, C, C), rgba: d.channels.toRgba(C, C, C) });
          }
          return m;
        })());
        let p = t.ThemeService = class extends g.Disposable {
          get colors() {
            return this._colors;
          }
          constructor(m) {
            super(), this._optionsService = m, this._contrastCache = new r.ColorContrastCache(), this._halfContrastCache = new r.ColorContrastCache(), this._onChangeColors = this.register(new f.EventEmitter()), this.onChangeColors = this._onChangeColors.event, this._colors = { foreground: e, background: o, cursor: s, cursorAccent: i, selectionForeground: void 0, selectionBackgroundTransparent: u, selectionBackgroundOpaque: d.color.blend(o, u), selectionInactiveBackgroundTransparent: u, selectionInactiveBackgroundOpaque: d.color.blend(o, u), ansi: t.DEFAULT_ANSI_COLORS.slice(), contrastCache: this._contrastCache, halfContrastCache: this._halfContrastCache }, this._updateRestoreColors(), this._setTheme(this._optionsService.rawOptions.theme), this.register(this._optionsService.onSpecificOptionChange("minimumContrastRatio", (() => this._contrastCache.clear()))), this.register(this._optionsService.onSpecificOptionChange("theme", (() => this._setTheme(this._optionsService.rawOptions.theme))));
          }
          _setTheme(m = {}) {
            const _ = this._colors;
            if (_.foreground = l(m.foreground, e), _.background = l(m.background, o), _.cursor = l(m.cursor, s), _.cursorAccent = l(m.cursorAccent, i), _.selectionBackgroundTransparent = l(m.selectionBackground, u), _.selectionBackgroundOpaque = d.color.blend(_.background, _.selectionBackgroundTransparent), _.selectionInactiveBackgroundTransparent = l(m.selectionInactiveBackground, _.selectionBackgroundTransparent), _.selectionInactiveBackgroundOpaque = d.color.blend(_.background, _.selectionInactiveBackgroundTransparent), _.selectionForeground = m.selectionForeground ? l(m.selectionForeground, d.NULL_COLOR) : void 0, _.selectionForeground === d.NULL_COLOR && (_.selectionForeground = void 0), d.color.isOpaque(_.selectionBackgroundTransparent) && (_.selectionBackgroundTransparent = d.color.opacity(_.selectionBackgroundTransparent, 0.3)), d.color.isOpaque(_.selectionInactiveBackgroundTransparent) && (_.selectionInactiveBackgroundTransparent = d.color.opacity(_.selectionInactiveBackgroundTransparent, 0.3)), _.ansi = t.DEFAULT_ANSI_COLORS.slice(), _.ansi[0] = l(m.black, t.DEFAULT_ANSI_COLORS[0]), _.ansi[1] = l(m.red, t.DEFAULT_ANSI_COLORS[1]), _.ansi[2] = l(m.green, t.DEFAULT_ANSI_COLORS[2]), _.ansi[3] = l(m.yellow, t.DEFAULT_ANSI_COLORS[3]), _.ansi[4] = l(m.blue, t.DEFAULT_ANSI_COLORS[4]), _.ansi[5] = l(m.magenta, t.DEFAULT_ANSI_COLORS[5]), _.ansi[6] = l(m.cyan, t.DEFAULT_ANSI_COLORS[6]), _.ansi[7] = l(m.white, t.DEFAULT_ANSI_COLORS[7]), _.ansi[8] = l(m.brightBlack, t.DEFAULT_ANSI_COLORS[8]), _.ansi[9] = l(m.brightRed, t.DEFAULT_ANSI_COLORS[9]), _.ansi[10] = l(m.brightGreen, t.DEFAULT_ANSI_COLORS[10]), _.ansi[11] = l(m.brightYellow, t.DEFAULT_ANSI_COLORS[11]), _.ansi[12] = l(m.brightBlue, t.DEFAULT_ANSI_COLORS[12]), _.ansi[13] = l(m.brightMagenta, t.DEFAULT_ANSI_COLORS[13]), _.ansi[14] = l(m.brightCyan, t.DEFAULT_ANSI_COLORS[14]), _.ansi[15] = l(m.brightWhite, t.DEFAULT_ANSI_COLORS[15]), m.extendedAnsi) {
              const v = Math.min(_.ansi.length - 16, m.extendedAnsi.length);
              for (let C = 0; C < v; C++) _.ansi[C + 16] = l(m.extendedAnsi[C], t.DEFAULT_ANSI_COLORS[C + 16]);
            }
            this._contrastCache.clear(), this._halfContrastCache.clear(), this._updateRestoreColors(), this._onChangeColors.fire(this.colors);
          }
          restoreColor(m) {
            this._restoreColor(m), this._onChangeColors.fire(this.colors);
          }
          _restoreColor(m) {
            if (m !== void 0) switch (m) {
              case 256:
                this._colors.foreground = this._restoreColors.foreground;
                break;
              case 257:
                this._colors.background = this._restoreColors.background;
                break;
              case 258:
                this._colors.cursor = this._restoreColors.cursor;
                break;
              default:
                this._colors.ansi[m] = this._restoreColors.ansi[m];
            }
            else for (let _ = 0; _ < this._restoreColors.ansi.length; ++_) this._colors.ansi[_] = this._restoreColors.ansi[_];
          }
          modifyColors(m) {
            m(this._colors), this._onChangeColors.fire(this.colors);
          }
          _updateRestoreColors() {
            this._restoreColors = { foreground: this._colors.foreground, background: this._colors.background, cursor: this._colors.cursor, ansi: this._colors.ansi.slice() };
          }
        };
        function l(m, _) {
          if (m !== void 0) try {
            return d.css.toColor(m);
          } catch (v) {
          }
          return _;
        }
        t.ThemeService = p = c([h(0, n.IOptionsService)], p);
      }, 6349: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CircularList = void 0;
        const c = a(8460), h = a(844);
        class r extends h.Disposable {
          constructor(f) {
            super(), this._maxLength = f, this.onDeleteEmitter = this.register(new c.EventEmitter()), this.onDelete = this.onDeleteEmitter.event, this.onInsertEmitter = this.register(new c.EventEmitter()), this.onInsert = this.onInsertEmitter.event, this.onTrimEmitter = this.register(new c.EventEmitter()), this.onTrim = this.onTrimEmitter.event, this._array = new Array(this._maxLength), this._startIndex = 0, this._length = 0;
          }
          get maxLength() {
            return this._maxLength;
          }
          set maxLength(f) {
            if (this._maxLength === f) return;
            const g = new Array(f);
            for (let n = 0; n < Math.min(f, this.length); n++) g[n] = this._array[this._getCyclicIndex(n)];
            this._array = g, this._maxLength = f, this._startIndex = 0;
          }
          get length() {
            return this._length;
          }
          set length(f) {
            if (f > this._length) for (let g = this._length; g < f; g++) this._array[g] = void 0;
            this._length = f;
          }
          get(f) {
            return this._array[this._getCyclicIndex(f)];
          }
          set(f, g) {
            this._array[this._getCyclicIndex(f)] = g;
          }
          push(f) {
            this._array[this._getCyclicIndex(this._length)] = f, this._length === this._maxLength ? (this._startIndex = ++this._startIndex % this._maxLength, this.onTrimEmitter.fire(1)) : this._length++;
          }
          recycle() {
            if (this._length !== this._maxLength) throw new Error("Can only recycle when the buffer is full");
            return this._startIndex = ++this._startIndex % this._maxLength, this.onTrimEmitter.fire(1), this._array[this._getCyclicIndex(this._length - 1)];
          }
          get isFull() {
            return this._length === this._maxLength;
          }
          pop() {
            return this._array[this._getCyclicIndex(this._length-- - 1)];
          }
          splice(f, g, ...n) {
            if (g) {
              for (let e = f; e < this._length - g; e++) this._array[this._getCyclicIndex(e)] = this._array[this._getCyclicIndex(e + g)];
              this._length -= g, this.onDeleteEmitter.fire({ index: f, amount: g });
            }
            for (let e = this._length - 1; e >= f; e--) this._array[this._getCyclicIndex(e + n.length)] = this._array[this._getCyclicIndex(e)];
            for (let e = 0; e < n.length; e++) this._array[this._getCyclicIndex(f + e)] = n[e];
            if (n.length && this.onInsertEmitter.fire({ index: f, amount: n.length }), this._length + n.length > this._maxLength) {
              const e = this._length + n.length - this._maxLength;
              this._startIndex += e, this._length = this._maxLength, this.onTrimEmitter.fire(e);
            } else this._length += n.length;
          }
          trimStart(f) {
            f > this._length && (f = this._length), this._startIndex += f, this._length -= f, this.onTrimEmitter.fire(f);
          }
          shiftElements(f, g, n) {
            if (!(g <= 0)) {
              if (f < 0 || f >= this._length) throw new Error("start argument out of range");
              if (f + n < 0) throw new Error("Cannot shift elements in list beyond index 0");
              if (n > 0) {
                for (let o = g - 1; o >= 0; o--) this.set(f + o + n, this.get(f + o));
                const e = f + g + n - this._length;
                if (e > 0) for (this._length += e; this._length > this._maxLength; ) this._length--, this._startIndex++, this.onTrimEmitter.fire(1);
              } else for (let e = 0; e < g; e++) this.set(f + e + n, this.get(f + e));
            }
          }
          _getCyclicIndex(f) {
            return (this._startIndex + f) % this._maxLength;
          }
        }
        t.CircularList = r;
      }, 1439: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.clone = void 0, t.clone = function a(c, h = 5) {
          if (typeof c != "object") return c;
          const r = Array.isArray(c) ? [] : {};
          for (const d in c) r[d] = h <= 1 ? c[d] : c[d] && a(c[d], h - 1);
          return r;
        };
      }, 8055: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.contrastRatio = t.toPaddedHex = t.rgba = t.rgb = t.css = t.color = t.channels = t.NULL_COLOR = void 0;
        let a = 0, c = 0, h = 0, r = 0;
        var d, f, g, n, e;
        function o(i) {
          const u = i.toString(16);
          return u.length < 2 ? "0" + u : u;
        }
        function s(i, u) {
          return i < u ? (u + 0.05) / (i + 0.05) : (i + 0.05) / (u + 0.05);
        }
        t.NULL_COLOR = { css: "#00000000", rgba: 0 }, (function(i) {
          i.toCss = function(u, p, l, m) {
            return m !== void 0 ? `#${o(u)}${o(p)}${o(l)}${o(m)}` : `#${o(u)}${o(p)}${o(l)}`;
          }, i.toRgba = function(u, p, l, m = 255) {
            return (u << 24 | p << 16 | l << 8 | m) >>> 0;
          }, i.toColor = function(u, p, l, m) {
            return { css: i.toCss(u, p, l, m), rgba: i.toRgba(u, p, l, m) };
          };
        })(d || (t.channels = d = {})), (function(i) {
          function u(p, l) {
            return r = Math.round(255 * l), [a, c, h] = e.toChannels(p.rgba), { css: d.toCss(a, c, h, r), rgba: d.toRgba(a, c, h, r) };
          }
          i.blend = function(p, l) {
            if (r = (255 & l.rgba) / 255, r === 1) return { css: l.css, rgba: l.rgba };
            const m = l.rgba >> 24 & 255, _ = l.rgba >> 16 & 255, v = l.rgba >> 8 & 255, C = p.rgba >> 24 & 255, w = p.rgba >> 16 & 255, S = p.rgba >> 8 & 255;
            return a = C + Math.round((m - C) * r), c = w + Math.round((_ - w) * r), h = S + Math.round((v - S) * r), { css: d.toCss(a, c, h), rgba: d.toRgba(a, c, h) };
          }, i.isOpaque = function(p) {
            return (255 & p.rgba) == 255;
          }, i.ensureContrastRatio = function(p, l, m) {
            const _ = e.ensureContrastRatio(p.rgba, l.rgba, m);
            if (_) return d.toColor(_ >> 24 & 255, _ >> 16 & 255, _ >> 8 & 255);
          }, i.opaque = function(p) {
            const l = (255 | p.rgba) >>> 0;
            return [a, c, h] = e.toChannels(l), { css: d.toCss(a, c, h), rgba: l };
          }, i.opacity = u, i.multiplyOpacity = function(p, l) {
            return r = 255 & p.rgba, u(p, r * l / 255);
          }, i.toColorRGB = function(p) {
            return [p.rgba >> 24 & 255, p.rgba >> 16 & 255, p.rgba >> 8 & 255];
          };
        })(f || (t.color = f = {})), (function(i) {
          let u, p;
          try {
            const l = document.createElement("canvas");
            l.width = 1, l.height = 1;
            const m = l.getContext("2d", { willReadFrequently: !0 });
            m && (u = m, u.globalCompositeOperation = "copy", p = u.createLinearGradient(0, 0, 1, 1));
          } catch (l) {
          }
          i.toColor = function(l) {
            if (l.match(/#[\da-f]{3,8}/i)) switch (l.length) {
              case 4:
                return a = parseInt(l.slice(1, 2).repeat(2), 16), c = parseInt(l.slice(2, 3).repeat(2), 16), h = parseInt(l.slice(3, 4).repeat(2), 16), d.toColor(a, c, h);
              case 5:
                return a = parseInt(l.slice(1, 2).repeat(2), 16), c = parseInt(l.slice(2, 3).repeat(2), 16), h = parseInt(l.slice(3, 4).repeat(2), 16), r = parseInt(l.slice(4, 5).repeat(2), 16), d.toColor(a, c, h, r);
              case 7:
                return { css: l, rgba: (parseInt(l.slice(1), 16) << 8 | 255) >>> 0 };
              case 9:
                return { css: l, rgba: parseInt(l.slice(1), 16) >>> 0 };
            }
            const m = l.match(/rgba?\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(,\s*(0|1|\d?\.(\d+))\s*)?\)/);
            if (m) return a = parseInt(m[1]), c = parseInt(m[2]), h = parseInt(m[3]), r = Math.round(255 * (m[5] === void 0 ? 1 : parseFloat(m[5]))), d.toColor(a, c, h, r);
            if (!u || !p) throw new Error("css.toColor: Unsupported css format");
            if (u.fillStyle = p, u.fillStyle = l, typeof u.fillStyle != "string") throw new Error("css.toColor: Unsupported css format");
            if (u.fillRect(0, 0, 1, 1), [a, c, h, r] = u.getImageData(0, 0, 1, 1).data, r !== 255) throw new Error("css.toColor: Unsupported css format");
            return { rgba: d.toRgba(a, c, h, r), css: l };
          };
        })(g || (t.css = g = {})), (function(i) {
          function u(p, l, m) {
            const _ = p / 255, v = l / 255, C = m / 255;
            return 0.2126 * (_ <= 0.03928 ? _ / 12.92 : Math.pow((_ + 0.055) / 1.055, 2.4)) + 0.7152 * (v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)) + 0.0722 * (C <= 0.03928 ? C / 12.92 : Math.pow((C + 0.055) / 1.055, 2.4));
          }
          i.relativeLuminance = function(p) {
            return u(p >> 16 & 255, p >> 8 & 255, 255 & p);
          }, i.relativeLuminance2 = u;
        })(n || (t.rgb = n = {})), (function(i) {
          function u(l, m, _) {
            const v = l >> 24 & 255, C = l >> 16 & 255, w = l >> 8 & 255;
            let S = m >> 24 & 255, b = m >> 16 & 255, x = m >> 8 & 255, A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            for (; A < _ && (S > 0 || b > 0 || x > 0); ) S -= Math.max(0, Math.ceil(0.1 * S)), b -= Math.max(0, Math.ceil(0.1 * b)), x -= Math.max(0, Math.ceil(0.1 * x)), A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            return (S << 24 | b << 16 | x << 8 | 255) >>> 0;
          }
          function p(l, m, _) {
            const v = l >> 24 & 255, C = l >> 16 & 255, w = l >> 8 & 255;
            let S = m >> 24 & 255, b = m >> 16 & 255, x = m >> 8 & 255, A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            for (; A < _ && (S < 255 || b < 255 || x < 255); ) S = Math.min(255, S + Math.ceil(0.1 * (255 - S))), b = Math.min(255, b + Math.ceil(0.1 * (255 - b))), x = Math.min(255, x + Math.ceil(0.1 * (255 - x))), A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            return (S << 24 | b << 16 | x << 8 | 255) >>> 0;
          }
          i.blend = function(l, m) {
            if (r = (255 & m) / 255, r === 1) return m;
            const _ = m >> 24 & 255, v = m >> 16 & 255, C = m >> 8 & 255, w = l >> 24 & 255, S = l >> 16 & 255, b = l >> 8 & 255;
            return a = w + Math.round((_ - w) * r), c = S + Math.round((v - S) * r), h = b + Math.round((C - b) * r), d.toRgba(a, c, h);
          }, i.ensureContrastRatio = function(l, m, _) {
            const v = n.relativeLuminance(l >> 8), C = n.relativeLuminance(m >> 8);
            if (s(v, C) < _) {
              if (C < v) {
                const b = u(l, m, _), x = s(v, n.relativeLuminance(b >> 8));
                if (x < _) {
                  const A = p(l, m, _);
                  return x > s(v, n.relativeLuminance(A >> 8)) ? b : A;
                }
                return b;
              }
              const w = p(l, m, _), S = s(v, n.relativeLuminance(w >> 8));
              if (S < _) {
                const b = u(l, m, _);
                return S > s(v, n.relativeLuminance(b >> 8)) ? w : b;
              }
              return w;
            }
          }, i.reduceLuminance = u, i.increaseLuminance = p, i.toChannels = function(l) {
            return [l >> 24 & 255, l >> 16 & 255, l >> 8 & 255, 255 & l];
          };
        })(e || (t.rgba = e = {})), t.toPaddedHex = o, t.contrastRatio = s;
      }, 8969: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CoreTerminal = void 0;
        const c = a(844), h = a(2585), r = a(4348), d = a(7866), f = a(744), g = a(7302), n = a(6975), e = a(8460), o = a(1753), s = a(1480), i = a(7994), u = a(9282), p = a(5435), l = a(5981), m = a(2660);
        let _ = !1;
        class v extends c.Disposable {
          get onScroll() {
            return this._onScrollApi || (this._onScrollApi = this.register(new e.EventEmitter()), this._onScroll.event(((w) => {
              var S;
              (S = this._onScrollApi) == null || S.fire(w.position);
            }))), this._onScrollApi.event;
          }
          get cols() {
            return this._bufferService.cols;
          }
          get rows() {
            return this._bufferService.rows;
          }
          get buffers() {
            return this._bufferService.buffers;
          }
          get options() {
            return this.optionsService.options;
          }
          set options(w) {
            for (const S in w) this.optionsService.options[S] = w[S];
          }
          constructor(w) {
            super(), this._windowsWrappingHeuristics = this.register(new c.MutableDisposable()), this._onBinary = this.register(new e.EventEmitter()), this.onBinary = this._onBinary.event, this._onData = this.register(new e.EventEmitter()), this.onData = this._onData.event, this._onLineFeed = this.register(new e.EventEmitter()), this.onLineFeed = this._onLineFeed.event, this._onResize = this.register(new e.EventEmitter()), this.onResize = this._onResize.event, this._onWriteParsed = this.register(new e.EventEmitter()), this.onWriteParsed = this._onWriteParsed.event, this._onScroll = this.register(new e.EventEmitter()), this._instantiationService = new r.InstantiationService(), this.optionsService = this.register(new g.OptionsService(w)), this._instantiationService.setService(h.IOptionsService, this.optionsService), this._bufferService = this.register(this._instantiationService.createInstance(f.BufferService)), this._instantiationService.setService(h.IBufferService, this._bufferService), this._logService = this.register(this._instantiationService.createInstance(d.LogService)), this._instantiationService.setService(h.ILogService, this._logService), this.coreService = this.register(this._instantiationService.createInstance(n.CoreService)), this._instantiationService.setService(h.ICoreService, this.coreService), this.coreMouseService = this.register(this._instantiationService.createInstance(o.CoreMouseService)), this._instantiationService.setService(h.ICoreMouseService, this.coreMouseService), this.unicodeService = this.register(this._instantiationService.createInstance(s.UnicodeService)), this._instantiationService.setService(h.IUnicodeService, this.unicodeService), this._charsetService = this._instantiationService.createInstance(i.CharsetService), this._instantiationService.setService(h.ICharsetService, this._charsetService), this._oscLinkService = this._instantiationService.createInstance(m.OscLinkService), this._instantiationService.setService(h.IOscLinkService, this._oscLinkService), this._inputHandler = this.register(new p.InputHandler(this._bufferService, this._charsetService, this.coreService, this._logService, this.optionsService, this._oscLinkService, this.coreMouseService, this.unicodeService)), this.register((0, e.forwardEvent)(this._inputHandler.onLineFeed, this._onLineFeed)), this.register(this._inputHandler), this.register((0, e.forwardEvent)(this._bufferService.onResize, this._onResize)), this.register((0, e.forwardEvent)(this.coreService.onData, this._onData)), this.register((0, e.forwardEvent)(this.coreService.onBinary, this._onBinary)), this.register(this.coreService.onRequestScrollToBottom((() => this.scrollToBottom()))), this.register(this.coreService.onUserInput((() => this._writeBuffer.handleUserInput()))), this.register(this.optionsService.onMultipleOptionChange(["windowsMode", "windowsPty"], (() => this._handleWindowsPtyOptionChange()))), this.register(this._bufferService.onScroll(((S) => {
              this._onScroll.fire({ position: this._bufferService.buffer.ydisp, source: 0 }), this._inputHandler.markRangeDirty(this._bufferService.buffer.scrollTop, this._bufferService.buffer.scrollBottom);
            }))), this.register(this._inputHandler.onScroll(((S) => {
              this._onScroll.fire({ position: this._bufferService.buffer.ydisp, source: 0 }), this._inputHandler.markRangeDirty(this._bufferService.buffer.scrollTop, this._bufferService.buffer.scrollBottom);
            }))), this._writeBuffer = this.register(new l.WriteBuffer(((S, b) => this._inputHandler.parse(S, b)))), this.register((0, e.forwardEvent)(this._writeBuffer.onWriteParsed, this._onWriteParsed));
          }
          write(w, S) {
            this._writeBuffer.write(w, S);
          }
          writeSync(w, S) {
            this._logService.logLevel <= h.LogLevelEnum.WARN && !_ && (this._logService.warn("writeSync is unreliable and will be removed soon."), _ = !0), this._writeBuffer.writeSync(w, S);
          }
          input(w, S = !0) {
            this.coreService.triggerDataEvent(w, S);
          }
          resize(w, S) {
            isNaN(w) || isNaN(S) || (w = Math.max(w, f.MINIMUM_COLS), S = Math.max(S, f.MINIMUM_ROWS), this._bufferService.resize(w, S));
          }
          scroll(w, S = !1) {
            this._bufferService.scroll(w, S);
          }
          scrollLines(w, S, b) {
            this._bufferService.scrollLines(w, S, b);
          }
          scrollPages(w) {
            this.scrollLines(w * (this.rows - 1));
          }
          scrollToTop() {
            this.scrollLines(-this._bufferService.buffer.ydisp);
          }
          scrollToBottom() {
            this.scrollLines(this._bufferService.buffer.ybase - this._bufferService.buffer.ydisp);
          }
          scrollToLine(w) {
            const S = w - this._bufferService.buffer.ydisp;
            S !== 0 && this.scrollLines(S);
          }
          registerEscHandler(w, S) {
            return this._inputHandler.registerEscHandler(w, S);
          }
          registerDcsHandler(w, S) {
            return this._inputHandler.registerDcsHandler(w, S);
          }
          registerCsiHandler(w, S) {
            return this._inputHandler.registerCsiHandler(w, S);
          }
          registerOscHandler(w, S) {
            return this._inputHandler.registerOscHandler(w, S);
          }
          _setup() {
            this._handleWindowsPtyOptionChange();
          }
          reset() {
            this._inputHandler.reset(), this._bufferService.reset(), this._charsetService.reset(), this.coreService.reset(), this.coreMouseService.reset();
          }
          _handleWindowsPtyOptionChange() {
            let w = !1;
            const S = this.optionsService.rawOptions.windowsPty;
            S && S.buildNumber !== void 0 && S.buildNumber !== void 0 ? w = S.backend === "conpty" && S.buildNumber < 21376 : this.optionsService.rawOptions.windowsMode && (w = !0), w ? this._enableWindowsWrappingHeuristics() : this._windowsWrappingHeuristics.clear();
          }
          _enableWindowsWrappingHeuristics() {
            if (!this._windowsWrappingHeuristics.value) {
              const w = [];
              w.push(this.onLineFeed(u.updateWindowsModeWrappedState.bind(null, this._bufferService))), w.push(this.registerCsiHandler({ final: "H" }, (() => ((0, u.updateWindowsModeWrappedState)(this._bufferService), !1)))), this._windowsWrappingHeuristics.value = (0, c.toDisposable)((() => {
                for (const S of w) S.dispose();
              }));
            }
          }
        }
        t.CoreTerminal = v;
      }, 8460: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.runAndSubscribe = t.forwardEvent = t.EventEmitter = void 0, t.EventEmitter = class {
          constructor() {
            this._listeners = [], this._disposed = !1;
          }
          get event() {
            return this._event || (this._event = (a) => (this._listeners.push(a), { dispose: () => {
              if (!this._disposed) {
                for (let c = 0; c < this._listeners.length; c++) if (this._listeners[c] === a) return void this._listeners.splice(c, 1);
              }
            } })), this._event;
          }
          fire(a, c) {
            const h = [];
            for (let r = 0; r < this._listeners.length; r++) h.push(this._listeners[r]);
            for (let r = 0; r < h.length; r++) h[r].call(void 0, a, c);
          }
          dispose() {
            this.clearListeners(), this._disposed = !0;
          }
          clearListeners() {
            this._listeners && (this._listeners.length = 0);
          }
        }, t.forwardEvent = function(a, c) {
          return a(((h) => c.fire(h)));
        }, t.runAndSubscribe = function(a, c) {
          return c(void 0), a(((h) => c(h)));
        };
      }, 5435: function(T, t, a) {
        var c = this && this.__decorate || function(M, y, L, R) {
          var D, F = arguments.length, U = F < 3 ? y : R === null ? R = Object.getOwnPropertyDescriptor(y, L) : R;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") U = Reflect.decorate(M, y, L, R);
          else for (var K = M.length - 1; K >= 0; K--) (D = M[K]) && (U = (F < 3 ? D(U) : F > 3 ? D(y, L, U) : D(y, L)) || U);
          return F > 3 && U && Object.defineProperty(y, L, U), U;
        }, h = this && this.__param || function(M, y) {
          return function(L, R) {
            y(L, R, M);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.InputHandler = t.WindowsOptionsReportType = void 0;
        const r = a(2584), d = a(7116), f = a(2015), g = a(844), n = a(482), e = a(8437), o = a(8460), s = a(643), i = a(511), u = a(3734), p = a(2585), l = a(1480), m = a(6242), _ = a(6351), v = a(5941), C = { "(": 0, ")": 1, "*": 2, "+": 3, "-": 1, ".": 2 }, w = 131072;
        function S(M, y) {
          if (M > 24) return y.setWinLines || !1;
          switch (M) {
            case 1:
              return !!y.restoreWin;
            case 2:
              return !!y.minimizeWin;
            case 3:
              return !!y.setWinPosition;
            case 4:
              return !!y.setWinSizePixels;
            case 5:
              return !!y.raiseWin;
            case 6:
              return !!y.lowerWin;
            case 7:
              return !!y.refreshWin;
            case 8:
              return !!y.setWinSizeChars;
            case 9:
              return !!y.maximizeWin;
            case 10:
              return !!y.fullscreenWin;
            case 11:
              return !!y.getWinState;
            case 13:
              return !!y.getWinPosition;
            case 14:
              return !!y.getWinSizePixels;
            case 15:
              return !!y.getScreenSizePixels;
            case 16:
              return !!y.getCellSizePixels;
            case 18:
              return !!y.getWinSizeChars;
            case 19:
              return !!y.getScreenSizeChars;
            case 20:
              return !!y.getIconTitle;
            case 21:
              return !!y.getWinTitle;
            case 22:
              return !!y.pushTitle;
            case 23:
              return !!y.popTitle;
            case 24:
              return !!y.setWinLines;
          }
          return !1;
        }
        var b;
        (function(M) {
          M[M.GET_WIN_SIZE_PIXELS = 0] = "GET_WIN_SIZE_PIXELS", M[M.GET_CELL_SIZE_PIXELS = 1] = "GET_CELL_SIZE_PIXELS";
        })(b || (t.WindowsOptionsReportType = b = {}));
        let x = 0;
        class A extends g.Disposable {
          getAttrData() {
            return this._curAttrData;
          }
          constructor(y, L, R, D, F, U, K, q, O = new f.EscapeSequenceParser()) {
            super(), this._bufferService = y, this._charsetService = L, this._coreService = R, this._logService = D, this._optionsService = F, this._oscLinkService = U, this._coreMouseService = K, this._unicodeService = q, this._parser = O, this._parseBuffer = new Uint32Array(4096), this._stringDecoder = new n.StringToUtf32(), this._utf8Decoder = new n.Utf8ToUtf32(), this._workCell = new i.CellData(), this._windowTitle = "", this._iconName = "", this._windowTitleStack = [], this._iconNameStack = [], this._curAttrData = e.DEFAULT_ATTR_DATA.clone(), this._eraseAttrDataInternal = e.DEFAULT_ATTR_DATA.clone(), this._onRequestBell = this.register(new o.EventEmitter()), this.onRequestBell = this._onRequestBell.event, this._onRequestRefreshRows = this.register(new o.EventEmitter()), this.onRequestRefreshRows = this._onRequestRefreshRows.event, this._onRequestReset = this.register(new o.EventEmitter()), this.onRequestReset = this._onRequestReset.event, this._onRequestSendFocus = this.register(new o.EventEmitter()), this.onRequestSendFocus = this._onRequestSendFocus.event, this._onRequestSyncScrollBar = this.register(new o.EventEmitter()), this.onRequestSyncScrollBar = this._onRequestSyncScrollBar.event, this._onRequestWindowsOptionsReport = this.register(new o.EventEmitter()), this.onRequestWindowsOptionsReport = this._onRequestWindowsOptionsReport.event, this._onA11yChar = this.register(new o.EventEmitter()), this.onA11yChar = this._onA11yChar.event, this._onA11yTab = this.register(new o.EventEmitter()), this.onA11yTab = this._onA11yTab.event, this._onCursorMove = this.register(new o.EventEmitter()), this.onCursorMove = this._onCursorMove.event, this._onLineFeed = this.register(new o.EventEmitter()), this.onLineFeed = this._onLineFeed.event, this._onScroll = this.register(new o.EventEmitter()), this.onScroll = this._onScroll.event, this._onTitleChange = this.register(new o.EventEmitter()), this.onTitleChange = this._onTitleChange.event, this._onColor = this.register(new o.EventEmitter()), this.onColor = this._onColor.event, this._parseStack = { paused: !1, cursorStartX: 0, cursorStartY: 0, decodedLength: 0, position: 0 }, this._specialColors = [256, 257, 258], this.register(this._parser), this._dirtyRowTracker = new P(this._bufferService), this._activeBuffer = this._bufferService.buffer, this.register(this._bufferService.buffers.onBufferActivate(((E) => this._activeBuffer = E.activeBuffer))), this._parser.setCsiHandlerFallback(((E, H) => {
              this._logService.debug("Unknown CSI code: ", { identifier: this._parser.identToString(E), params: H.toArray() });
            })), this._parser.setEscHandlerFallback(((E) => {
              this._logService.debug("Unknown ESC code: ", { identifier: this._parser.identToString(E) });
            })), this._parser.setExecuteHandlerFallback(((E) => {
              this._logService.debug("Unknown EXECUTE code: ", { code: E });
            })), this._parser.setOscHandlerFallback(((E, H, N) => {
              this._logService.debug("Unknown OSC code: ", { identifier: E, action: H, data: N });
            })), this._parser.setDcsHandlerFallback(((E, H, N) => {
              H === "HOOK" && (N = N.toArray()), this._logService.debug("Unknown DCS code: ", { identifier: this._parser.identToString(E), action: H, payload: N });
            })), this._parser.setPrintHandler(((E, H, N) => this.print(E, H, N))), this._parser.registerCsiHandler({ final: "@" }, ((E) => this.insertChars(E))), this._parser.registerCsiHandler({ intermediates: " ", final: "@" }, ((E) => this.scrollLeft(E))), this._parser.registerCsiHandler({ final: "A" }, ((E) => this.cursorUp(E))), this._parser.registerCsiHandler({ intermediates: " ", final: "A" }, ((E) => this.scrollRight(E))), this._parser.registerCsiHandler({ final: "B" }, ((E) => this.cursorDown(E))), this._parser.registerCsiHandler({ final: "C" }, ((E) => this.cursorForward(E))), this._parser.registerCsiHandler({ final: "D" }, ((E) => this.cursorBackward(E))), this._parser.registerCsiHandler({ final: "E" }, ((E) => this.cursorNextLine(E))), this._parser.registerCsiHandler({ final: "F" }, ((E) => this.cursorPrecedingLine(E))), this._parser.registerCsiHandler({ final: "G" }, ((E) => this.cursorCharAbsolute(E))), this._parser.registerCsiHandler({ final: "H" }, ((E) => this.cursorPosition(E))), this._parser.registerCsiHandler({ final: "I" }, ((E) => this.cursorForwardTab(E))), this._parser.registerCsiHandler({ final: "J" }, ((E) => this.eraseInDisplay(E, !1))), this._parser.registerCsiHandler({ prefix: "?", final: "J" }, ((E) => this.eraseInDisplay(E, !0))), this._parser.registerCsiHandler({ final: "K" }, ((E) => this.eraseInLine(E, !1))), this._parser.registerCsiHandler({ prefix: "?", final: "K" }, ((E) => this.eraseInLine(E, !0))), this._parser.registerCsiHandler({ final: "L" }, ((E) => this.insertLines(E))), this._parser.registerCsiHandler({ final: "M" }, ((E) => this.deleteLines(E))), this._parser.registerCsiHandler({ final: "P" }, ((E) => this.deleteChars(E))), this._parser.registerCsiHandler({ final: "S" }, ((E) => this.scrollUp(E))), this._parser.registerCsiHandler({ final: "T" }, ((E) => this.scrollDown(E))), this._parser.registerCsiHandler({ final: "X" }, ((E) => this.eraseChars(E))), this._parser.registerCsiHandler({ final: "Z" }, ((E) => this.cursorBackwardTab(E))), this._parser.registerCsiHandler({ final: "`" }, ((E) => this.charPosAbsolute(E))), this._parser.registerCsiHandler({ final: "a" }, ((E) => this.hPositionRelative(E))), this._parser.registerCsiHandler({ final: "b" }, ((E) => this.repeatPrecedingCharacter(E))), this._parser.registerCsiHandler({ final: "c" }, ((E) => this.sendDeviceAttributesPrimary(E))), this._parser.registerCsiHandler({ prefix: ">", final: "c" }, ((E) => this.sendDeviceAttributesSecondary(E))), this._parser.registerCsiHandler({ final: "d" }, ((E) => this.linePosAbsolute(E))), this._parser.registerCsiHandler({ final: "e" }, ((E) => this.vPositionRelative(E))), this._parser.registerCsiHandler({ final: "f" }, ((E) => this.hVPosition(E))), this._parser.registerCsiHandler({ final: "g" }, ((E) => this.tabClear(E))), this._parser.registerCsiHandler({ final: "h" }, ((E) => this.setMode(E))), this._parser.registerCsiHandler({ prefix: "?", final: "h" }, ((E) => this.setModePrivate(E))), this._parser.registerCsiHandler({ final: "l" }, ((E) => this.resetMode(E))), this._parser.registerCsiHandler({ prefix: "?", final: "l" }, ((E) => this.resetModePrivate(E))), this._parser.registerCsiHandler({ final: "m" }, ((E) => this.charAttributes(E))), this._parser.registerCsiHandler({ final: "n" }, ((E) => this.deviceStatus(E))), this._parser.registerCsiHandler({ prefix: "?", final: "n" }, ((E) => this.deviceStatusPrivate(E))), this._parser.registerCsiHandler({ intermediates: "!", final: "p" }, ((E) => this.softReset(E))), this._parser.registerCsiHandler({ intermediates: " ", final: "q" }, ((E) => this.setCursorStyle(E))), this._parser.registerCsiHandler({ final: "r" }, ((E) => this.setScrollRegion(E))), this._parser.registerCsiHandler({ final: "s" }, ((E) => this.saveCursor(E))), this._parser.registerCsiHandler({ final: "t" }, ((E) => this.windowOptions(E))), this._parser.registerCsiHandler({ final: "u" }, ((E) => this.restoreCursor(E))), this._parser.registerCsiHandler({ intermediates: "'", final: "}" }, ((E) => this.insertColumns(E))), this._parser.registerCsiHandler({ intermediates: "'", final: "~" }, ((E) => this.deleteColumns(E))), this._parser.registerCsiHandler({ intermediates: '"', final: "q" }, ((E) => this.selectProtected(E))), this._parser.registerCsiHandler({ intermediates: "$", final: "p" }, ((E) => this.requestMode(E, !0))), this._parser.registerCsiHandler({ prefix: "?", intermediates: "$", final: "p" }, ((E) => this.requestMode(E, !1))), this._parser.setExecuteHandler(r.C0.BEL, (() => this.bell())), this._parser.setExecuteHandler(r.C0.LF, (() => this.lineFeed())), this._parser.setExecuteHandler(r.C0.VT, (() => this.lineFeed())), this._parser.setExecuteHandler(r.C0.FF, (() => this.lineFeed())), this._parser.setExecuteHandler(r.C0.CR, (() => this.carriageReturn())), this._parser.setExecuteHandler(r.C0.BS, (() => this.backspace())), this._parser.setExecuteHandler(r.C0.HT, (() => this.tab())), this._parser.setExecuteHandler(r.C0.SO, (() => this.shiftOut())), this._parser.setExecuteHandler(r.C0.SI, (() => this.shiftIn())), this._parser.setExecuteHandler(r.C1.IND, (() => this.index())), this._parser.setExecuteHandler(r.C1.NEL, (() => this.nextLine())), this._parser.setExecuteHandler(r.C1.HTS, (() => this.tabSet())), this._parser.registerOscHandler(0, new m.OscHandler(((E) => (this.setTitle(E), this.setIconName(E), !0)))), this._parser.registerOscHandler(1, new m.OscHandler(((E) => this.setIconName(E)))), this._parser.registerOscHandler(2, new m.OscHandler(((E) => this.setTitle(E)))), this._parser.registerOscHandler(4, new m.OscHandler(((E) => this.setOrReportIndexedColor(E)))), this._parser.registerOscHandler(8, new m.OscHandler(((E) => this.setHyperlink(E)))), this._parser.registerOscHandler(10, new m.OscHandler(((E) => this.setOrReportFgColor(E)))), this._parser.registerOscHandler(11, new m.OscHandler(((E) => this.setOrReportBgColor(E)))), this._parser.registerOscHandler(12, new m.OscHandler(((E) => this.setOrReportCursorColor(E)))), this._parser.registerOscHandler(104, new m.OscHandler(((E) => this.restoreIndexedColor(E)))), this._parser.registerOscHandler(110, new m.OscHandler(((E) => this.restoreFgColor(E)))), this._parser.registerOscHandler(111, new m.OscHandler(((E) => this.restoreBgColor(E)))), this._parser.registerOscHandler(112, new m.OscHandler(((E) => this.restoreCursorColor(E)))), this._parser.registerEscHandler({ final: "7" }, (() => this.saveCursor())), this._parser.registerEscHandler({ final: "8" }, (() => this.restoreCursor())), this._parser.registerEscHandler({ final: "D" }, (() => this.index())), this._parser.registerEscHandler({ final: "E" }, (() => this.nextLine())), this._parser.registerEscHandler({ final: "H" }, (() => this.tabSet())), this._parser.registerEscHandler({ final: "M" }, (() => this.reverseIndex())), this._parser.registerEscHandler({ final: "=" }, (() => this.keypadApplicationMode())), this._parser.registerEscHandler({ final: ">" }, (() => this.keypadNumericMode())), this._parser.registerEscHandler({ final: "c" }, (() => this.fullReset())), this._parser.registerEscHandler({ final: "n" }, (() => this.setgLevel(2))), this._parser.registerEscHandler({ final: "o" }, (() => this.setgLevel(3))), this._parser.registerEscHandler({ final: "|" }, (() => this.setgLevel(3))), this._parser.registerEscHandler({ final: "}" }, (() => this.setgLevel(2))), this._parser.registerEscHandler({ final: "~" }, (() => this.setgLevel(1))), this._parser.registerEscHandler({ intermediates: "%", final: "@" }, (() => this.selectDefaultCharset())), this._parser.registerEscHandler({ intermediates: "%", final: "G" }, (() => this.selectDefaultCharset()));
            for (const E in d.CHARSETS) this._parser.registerEscHandler({ intermediates: "(", final: E }, (() => this.selectCharset("(" + E))), this._parser.registerEscHandler({ intermediates: ")", final: E }, (() => this.selectCharset(")" + E))), this._parser.registerEscHandler({ intermediates: "*", final: E }, (() => this.selectCharset("*" + E))), this._parser.registerEscHandler({ intermediates: "+", final: E }, (() => this.selectCharset("+" + E))), this._parser.registerEscHandler({ intermediates: "-", final: E }, (() => this.selectCharset("-" + E))), this._parser.registerEscHandler({ intermediates: ".", final: E }, (() => this.selectCharset("." + E))), this._parser.registerEscHandler({ intermediates: "/", final: E }, (() => this.selectCharset("/" + E)));
            this._parser.registerEscHandler({ intermediates: "#", final: "8" }, (() => this.screenAlignmentPattern())), this._parser.setErrorHandler(((E) => (this._logService.error("Parsing error: ", E), E))), this._parser.registerDcsHandler({ intermediates: "$", final: "q" }, new _.DcsHandler(((E, H) => this.requestStatusString(E, H))));
          }
          _preserveStack(y, L, R, D) {
            this._parseStack.paused = !0, this._parseStack.cursorStartX = y, this._parseStack.cursorStartY = L, this._parseStack.decodedLength = R, this._parseStack.position = D;
          }
          _logSlowResolvingAsync(y) {
            this._logService.logLevel <= p.LogLevelEnum.WARN && Promise.race([y, new Promise(((L, R) => setTimeout((() => R("#SLOW_TIMEOUT")), 5e3)))]).catch(((L) => {
              if (L !== "#SLOW_TIMEOUT") throw L;
              console.warn("async parser handler taking longer than 5000 ms");
            }));
          }
          _getCurrentLinkId() {
            return this._curAttrData.extended.urlId;
          }
          parse(y, L) {
            let R, D = this._activeBuffer.x, F = this._activeBuffer.y, U = 0;
            const K = this._parseStack.paused;
            if (K) {
              if (R = this._parser.parse(this._parseBuffer, this._parseStack.decodedLength, L)) return this._logSlowResolvingAsync(R), R;
              D = this._parseStack.cursorStartX, F = this._parseStack.cursorStartY, this._parseStack.paused = !1, y.length > w && (U = this._parseStack.position + w);
            }
            if (this._logService.logLevel <= p.LogLevelEnum.DEBUG && this._logService.debug("parsing data" + (typeof y == "string" ? ` "${y}"` : ` "${Array.prototype.map.call(y, ((E) => String.fromCharCode(E))).join("")}"`), typeof y == "string" ? y.split("").map(((E) => E.charCodeAt(0))) : y), this._parseBuffer.length < y.length && this._parseBuffer.length < w && (this._parseBuffer = new Uint32Array(Math.min(y.length, w))), K || this._dirtyRowTracker.clearRange(), y.length > w) for (let E = U; E < y.length; E += w) {
              const H = E + w < y.length ? E + w : y.length, N = typeof y == "string" ? this._stringDecoder.decode(y.substring(E, H), this._parseBuffer) : this._utf8Decoder.decode(y.subarray(E, H), this._parseBuffer);
              if (R = this._parser.parse(this._parseBuffer, N)) return this._preserveStack(D, F, N, E), this._logSlowResolvingAsync(R), R;
            }
            else if (!K) {
              const E = typeof y == "string" ? this._stringDecoder.decode(y, this._parseBuffer) : this._utf8Decoder.decode(y, this._parseBuffer);
              if (R = this._parser.parse(this._parseBuffer, E)) return this._preserveStack(D, F, E, 0), this._logSlowResolvingAsync(R), R;
            }
            this._activeBuffer.x === D && this._activeBuffer.y === F || this._onCursorMove.fire();
            const q = this._dirtyRowTracker.end + (this._bufferService.buffer.ybase - this._bufferService.buffer.ydisp), O = this._dirtyRowTracker.start + (this._bufferService.buffer.ybase - this._bufferService.buffer.ydisp);
            O < this._bufferService.rows && this._onRequestRefreshRows.fire(Math.min(O, this._bufferService.rows - 1), Math.min(q, this._bufferService.rows - 1));
          }
          print(y, L, R) {
            let D, F;
            const U = this._charsetService.charset, K = this._optionsService.rawOptions.screenReaderMode, q = this._bufferService.cols, O = this._coreService.decPrivateModes.wraparound, E = this._coreService.modes.insertMode, H = this._curAttrData;
            let N = this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y);
            this._dirtyRowTracker.markDirty(this._activeBuffer.y), this._activeBuffer.x && R - L > 0 && N.getWidth(this._activeBuffer.x - 1) === 2 && N.setCellFromCodepoint(this._activeBuffer.x - 1, 0, 1, H);
            let G = this._parser.precedingJoinState;
            for (let j = L; j < R; ++j) {
              if (D = y[j], D < 127 && U) {
                const ce = U[String.fromCharCode(D)];
                ce && (D = ce.charCodeAt(0));
              }
              const ie = this._unicodeService.charProperties(D, G);
              F = l.UnicodeService.extractWidth(ie);
              const V = l.UnicodeService.extractShouldJoin(ie), ae = V ? l.UnicodeService.extractWidth(G) : 0;
              if (G = ie, K && this._onA11yChar.fire((0, n.stringFromCodePoint)(D)), this._getCurrentLinkId() && this._oscLinkService.addLineToLink(this._getCurrentLinkId(), this._activeBuffer.ybase + this._activeBuffer.y), this._activeBuffer.x + F - ae > q) {
                if (O) {
                  const ce = N;
                  let ee = this._activeBuffer.x - ae;
                  for (this._activeBuffer.x = ae, this._activeBuffer.y++, this._activeBuffer.y === this._activeBuffer.scrollBottom + 1 ? (this._activeBuffer.y--, this._bufferService.scroll(this._eraseAttrData(), !0)) : (this._activeBuffer.y >= this._bufferService.rows && (this._activeBuffer.y = this._bufferService.rows - 1), this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y).isWrapped = !0), N = this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y), ae > 0 && N instanceof e.BufferLine && N.copyCellsFrom(ce, ee, 0, ae, !1); ee < q; ) ce.setCellFromCodepoint(ee++, 0, 1, H);
                } else if (this._activeBuffer.x = q - 1, F === 2) continue;
              }
              if (V && this._activeBuffer.x) {
                const ce = N.getWidth(this._activeBuffer.x - 1) ? 1 : 2;
                N.addCodepointToCell(this._activeBuffer.x - ce, D, F);
                for (let ee = F - ae; --ee >= 0; ) N.setCellFromCodepoint(this._activeBuffer.x++, 0, 0, H);
              } else if (E && (N.insertCells(this._activeBuffer.x, F - ae, this._activeBuffer.getNullCell(H)), N.getWidth(q - 1) === 2 && N.setCellFromCodepoint(q - 1, s.NULL_CELL_CODE, s.NULL_CELL_WIDTH, H)), N.setCellFromCodepoint(this._activeBuffer.x++, D, F, H), F > 0) for (; --F; ) N.setCellFromCodepoint(this._activeBuffer.x++, 0, 0, H);
            }
            this._parser.precedingJoinState = G, this._activeBuffer.x < q && R - L > 0 && N.getWidth(this._activeBuffer.x) === 0 && !N.hasContent(this._activeBuffer.x) && N.setCellFromCodepoint(this._activeBuffer.x, 0, 1, H), this._dirtyRowTracker.markDirty(this._activeBuffer.y);
          }
          registerCsiHandler(y, L) {
            return y.final !== "t" || y.prefix || y.intermediates ? this._parser.registerCsiHandler(y, L) : this._parser.registerCsiHandler(y, ((R) => !S(R.params[0], this._optionsService.rawOptions.windowOptions) || L(R)));
          }
          registerDcsHandler(y, L) {
            return this._parser.registerDcsHandler(y, new _.DcsHandler(L));
          }
          registerEscHandler(y, L) {
            return this._parser.registerEscHandler(y, L);
          }
          registerOscHandler(y, L) {
            return this._parser.registerOscHandler(y, new m.OscHandler(L));
          }
          bell() {
            return this._onRequestBell.fire(), !0;
          }
          lineFeed() {
            return this._dirtyRowTracker.markDirty(this._activeBuffer.y), this._optionsService.rawOptions.convertEol && (this._activeBuffer.x = 0), this._activeBuffer.y++, this._activeBuffer.y === this._activeBuffer.scrollBottom + 1 ? (this._activeBuffer.y--, this._bufferService.scroll(this._eraseAttrData())) : this._activeBuffer.y >= this._bufferService.rows ? this._activeBuffer.y = this._bufferService.rows - 1 : this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y).isWrapped = !1, this._activeBuffer.x >= this._bufferService.cols && this._activeBuffer.x--, this._dirtyRowTracker.markDirty(this._activeBuffer.y), this._onLineFeed.fire(), !0;
          }
          carriageReturn() {
            return this._activeBuffer.x = 0, !0;
          }
          backspace() {
            var y;
            if (!this._coreService.decPrivateModes.reverseWraparound) return this._restrictCursor(), this._activeBuffer.x > 0 && this._activeBuffer.x--, !0;
            if (this._restrictCursor(this._bufferService.cols), this._activeBuffer.x > 0) this._activeBuffer.x--;
            else if (this._activeBuffer.x === 0 && this._activeBuffer.y > this._activeBuffer.scrollTop && this._activeBuffer.y <= this._activeBuffer.scrollBottom && ((y = this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y)) != null && y.isWrapped)) {
              this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y).isWrapped = !1, this._activeBuffer.y--, this._activeBuffer.x = this._bufferService.cols - 1;
              const L = this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y);
              L.hasWidth(this._activeBuffer.x) && !L.hasContent(this._activeBuffer.x) && this._activeBuffer.x--;
            }
            return this._restrictCursor(), !0;
          }
          tab() {
            if (this._activeBuffer.x >= this._bufferService.cols) return !0;
            const y = this._activeBuffer.x;
            return this._activeBuffer.x = this._activeBuffer.nextStop(), this._optionsService.rawOptions.screenReaderMode && this._onA11yTab.fire(this._activeBuffer.x - y), !0;
          }
          shiftOut() {
            return this._charsetService.setgLevel(1), !0;
          }
          shiftIn() {
            return this._charsetService.setgLevel(0), !0;
          }
          _restrictCursor(y = this._bufferService.cols - 1) {
            this._activeBuffer.x = Math.min(y, Math.max(0, this._activeBuffer.x)), this._activeBuffer.y = this._coreService.decPrivateModes.origin ? Math.min(this._activeBuffer.scrollBottom, Math.max(this._activeBuffer.scrollTop, this._activeBuffer.y)) : Math.min(this._bufferService.rows - 1, Math.max(0, this._activeBuffer.y)), this._dirtyRowTracker.markDirty(this._activeBuffer.y);
          }
          _setCursor(y, L) {
            this._dirtyRowTracker.markDirty(this._activeBuffer.y), this._coreService.decPrivateModes.origin ? (this._activeBuffer.x = y, this._activeBuffer.y = this._activeBuffer.scrollTop + L) : (this._activeBuffer.x = y, this._activeBuffer.y = L), this._restrictCursor(), this._dirtyRowTracker.markDirty(this._activeBuffer.y);
          }
          _moveCursor(y, L) {
            this._restrictCursor(), this._setCursor(this._activeBuffer.x + y, this._activeBuffer.y + L);
          }
          cursorUp(y) {
            const L = this._activeBuffer.y - this._activeBuffer.scrollTop;
            return L >= 0 ? this._moveCursor(0, -Math.min(L, y.params[0] || 1)) : this._moveCursor(0, -(y.params[0] || 1)), !0;
          }
          cursorDown(y) {
            const L = this._activeBuffer.scrollBottom - this._activeBuffer.y;
            return L >= 0 ? this._moveCursor(0, Math.min(L, y.params[0] || 1)) : this._moveCursor(0, y.params[0] || 1), !0;
          }
          cursorForward(y) {
            return this._moveCursor(y.params[0] || 1, 0), !0;
          }
          cursorBackward(y) {
            return this._moveCursor(-(y.params[0] || 1), 0), !0;
          }
          cursorNextLine(y) {
            return this.cursorDown(y), this._activeBuffer.x = 0, !0;
          }
          cursorPrecedingLine(y) {
            return this.cursorUp(y), this._activeBuffer.x = 0, !0;
          }
          cursorCharAbsolute(y) {
            return this._setCursor((y.params[0] || 1) - 1, this._activeBuffer.y), !0;
          }
          cursorPosition(y) {
            return this._setCursor(y.length >= 2 ? (y.params[1] || 1) - 1 : 0, (y.params[0] || 1) - 1), !0;
          }
          charPosAbsolute(y) {
            return this._setCursor((y.params[0] || 1) - 1, this._activeBuffer.y), !0;
          }
          hPositionRelative(y) {
            return this._moveCursor(y.params[0] || 1, 0), !0;
          }
          linePosAbsolute(y) {
            return this._setCursor(this._activeBuffer.x, (y.params[0] || 1) - 1), !0;
          }
          vPositionRelative(y) {
            return this._moveCursor(0, y.params[0] || 1), !0;
          }
          hVPosition(y) {
            return this.cursorPosition(y), !0;
          }
          tabClear(y) {
            const L = y.params[0];
            return L === 0 ? delete this._activeBuffer.tabs[this._activeBuffer.x] : L === 3 && (this._activeBuffer.tabs = {}), !0;
          }
          cursorForwardTab(y) {
            if (this._activeBuffer.x >= this._bufferService.cols) return !0;
            let L = y.params[0] || 1;
            for (; L--; ) this._activeBuffer.x = this._activeBuffer.nextStop();
            return !0;
          }
          cursorBackwardTab(y) {
            if (this._activeBuffer.x >= this._bufferService.cols) return !0;
            let L = y.params[0] || 1;
            for (; L--; ) this._activeBuffer.x = this._activeBuffer.prevStop();
            return !0;
          }
          selectProtected(y) {
            const L = y.params[0];
            return L === 1 && (this._curAttrData.bg |= 536870912), L !== 2 && L !== 0 || (this._curAttrData.bg &= -536870913), !0;
          }
          _eraseInBufferLine(y, L, R, D = !1, F = !1) {
            const U = this._activeBuffer.lines.get(this._activeBuffer.ybase + y);
            U.replaceCells(L, R, this._activeBuffer.getNullCell(this._eraseAttrData()), F), D && (U.isWrapped = !1);
          }
          _resetBufferLine(y, L = !1) {
            const R = this._activeBuffer.lines.get(this._activeBuffer.ybase + y);
            R && (R.fill(this._activeBuffer.getNullCell(this._eraseAttrData()), L), this._bufferService.buffer.clearMarkers(this._activeBuffer.ybase + y), R.isWrapped = !1);
          }
          eraseInDisplay(y, L = !1) {
            let R;
            switch (this._restrictCursor(this._bufferService.cols), y.params[0]) {
              case 0:
                for (R = this._activeBuffer.y, this._dirtyRowTracker.markDirty(R), this._eraseInBufferLine(R++, this._activeBuffer.x, this._bufferService.cols, this._activeBuffer.x === 0, L); R < this._bufferService.rows; R++) this._resetBufferLine(R, L);
                this._dirtyRowTracker.markDirty(R);
                break;
              case 1:
                for (R = this._activeBuffer.y, this._dirtyRowTracker.markDirty(R), this._eraseInBufferLine(R, 0, this._activeBuffer.x + 1, !0, L), this._activeBuffer.x + 1 >= this._bufferService.cols && (this._activeBuffer.lines.get(R + 1).isWrapped = !1); R--; ) this._resetBufferLine(R, L);
                this._dirtyRowTracker.markDirty(0);
                break;
              case 2:
                for (R = this._bufferService.rows, this._dirtyRowTracker.markDirty(R - 1); R--; ) this._resetBufferLine(R, L);
                this._dirtyRowTracker.markDirty(0);
                break;
              case 3:
                const D = this._activeBuffer.lines.length - this._bufferService.rows;
                D > 0 && (this._activeBuffer.lines.trimStart(D), this._activeBuffer.ybase = Math.max(this._activeBuffer.ybase - D, 0), this._activeBuffer.ydisp = Math.max(this._activeBuffer.ydisp - D, 0), this._onScroll.fire(0));
            }
            return !0;
          }
          eraseInLine(y, L = !1) {
            switch (this._restrictCursor(this._bufferService.cols), y.params[0]) {
              case 0:
                this._eraseInBufferLine(this._activeBuffer.y, this._activeBuffer.x, this._bufferService.cols, this._activeBuffer.x === 0, L);
                break;
              case 1:
                this._eraseInBufferLine(this._activeBuffer.y, 0, this._activeBuffer.x + 1, !1, L);
                break;
              case 2:
                this._eraseInBufferLine(this._activeBuffer.y, 0, this._bufferService.cols, !0, L);
            }
            return this._dirtyRowTracker.markDirty(this._activeBuffer.y), !0;
          }
          insertLines(y) {
            this._restrictCursor();
            let L = y.params[0] || 1;
            if (this._activeBuffer.y > this._activeBuffer.scrollBottom || this._activeBuffer.y < this._activeBuffer.scrollTop) return !0;
            const R = this._activeBuffer.ybase + this._activeBuffer.y, D = this._bufferService.rows - 1 - this._activeBuffer.scrollBottom, F = this._bufferService.rows - 1 + this._activeBuffer.ybase - D + 1;
            for (; L--; ) this._activeBuffer.lines.splice(F - 1, 1), this._activeBuffer.lines.splice(R, 0, this._activeBuffer.getBlankLine(this._eraseAttrData()));
            return this._dirtyRowTracker.markRangeDirty(this._activeBuffer.y, this._activeBuffer.scrollBottom), this._activeBuffer.x = 0, !0;
          }
          deleteLines(y) {
            this._restrictCursor();
            let L = y.params[0] || 1;
            if (this._activeBuffer.y > this._activeBuffer.scrollBottom || this._activeBuffer.y < this._activeBuffer.scrollTop) return !0;
            const R = this._activeBuffer.ybase + this._activeBuffer.y;
            let D;
            for (D = this._bufferService.rows - 1 - this._activeBuffer.scrollBottom, D = this._bufferService.rows - 1 + this._activeBuffer.ybase - D; L--; ) this._activeBuffer.lines.splice(R, 1), this._activeBuffer.lines.splice(D, 0, this._activeBuffer.getBlankLine(this._eraseAttrData()));
            return this._dirtyRowTracker.markRangeDirty(this._activeBuffer.y, this._activeBuffer.scrollBottom), this._activeBuffer.x = 0, !0;
          }
          insertChars(y) {
            this._restrictCursor();
            const L = this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y);
            return L && (L.insertCells(this._activeBuffer.x, y.params[0] || 1, this._activeBuffer.getNullCell(this._eraseAttrData())), this._dirtyRowTracker.markDirty(this._activeBuffer.y)), !0;
          }
          deleteChars(y) {
            this._restrictCursor();
            const L = this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y);
            return L && (L.deleteCells(this._activeBuffer.x, y.params[0] || 1, this._activeBuffer.getNullCell(this._eraseAttrData())), this._dirtyRowTracker.markDirty(this._activeBuffer.y)), !0;
          }
          scrollUp(y) {
            let L = y.params[0] || 1;
            for (; L--; ) this._activeBuffer.lines.splice(this._activeBuffer.ybase + this._activeBuffer.scrollTop, 1), this._activeBuffer.lines.splice(this._activeBuffer.ybase + this._activeBuffer.scrollBottom, 0, this._activeBuffer.getBlankLine(this._eraseAttrData()));
            return this._dirtyRowTracker.markRangeDirty(this._activeBuffer.scrollTop, this._activeBuffer.scrollBottom), !0;
          }
          scrollDown(y) {
            let L = y.params[0] || 1;
            for (; L--; ) this._activeBuffer.lines.splice(this._activeBuffer.ybase + this._activeBuffer.scrollBottom, 1), this._activeBuffer.lines.splice(this._activeBuffer.ybase + this._activeBuffer.scrollTop, 0, this._activeBuffer.getBlankLine(e.DEFAULT_ATTR_DATA));
            return this._dirtyRowTracker.markRangeDirty(this._activeBuffer.scrollTop, this._activeBuffer.scrollBottom), !0;
          }
          scrollLeft(y) {
            if (this._activeBuffer.y > this._activeBuffer.scrollBottom || this._activeBuffer.y < this._activeBuffer.scrollTop) return !0;
            const L = y.params[0] || 1;
            for (let R = this._activeBuffer.scrollTop; R <= this._activeBuffer.scrollBottom; ++R) {
              const D = this._activeBuffer.lines.get(this._activeBuffer.ybase + R);
              D.deleteCells(0, L, this._activeBuffer.getNullCell(this._eraseAttrData())), D.isWrapped = !1;
            }
            return this._dirtyRowTracker.markRangeDirty(this._activeBuffer.scrollTop, this._activeBuffer.scrollBottom), !0;
          }
          scrollRight(y) {
            if (this._activeBuffer.y > this._activeBuffer.scrollBottom || this._activeBuffer.y < this._activeBuffer.scrollTop) return !0;
            const L = y.params[0] || 1;
            for (let R = this._activeBuffer.scrollTop; R <= this._activeBuffer.scrollBottom; ++R) {
              const D = this._activeBuffer.lines.get(this._activeBuffer.ybase + R);
              D.insertCells(0, L, this._activeBuffer.getNullCell(this._eraseAttrData())), D.isWrapped = !1;
            }
            return this._dirtyRowTracker.markRangeDirty(this._activeBuffer.scrollTop, this._activeBuffer.scrollBottom), !0;
          }
          insertColumns(y) {
            if (this._activeBuffer.y > this._activeBuffer.scrollBottom || this._activeBuffer.y < this._activeBuffer.scrollTop) return !0;
            const L = y.params[0] || 1;
            for (let R = this._activeBuffer.scrollTop; R <= this._activeBuffer.scrollBottom; ++R) {
              const D = this._activeBuffer.lines.get(this._activeBuffer.ybase + R);
              D.insertCells(this._activeBuffer.x, L, this._activeBuffer.getNullCell(this._eraseAttrData())), D.isWrapped = !1;
            }
            return this._dirtyRowTracker.markRangeDirty(this._activeBuffer.scrollTop, this._activeBuffer.scrollBottom), !0;
          }
          deleteColumns(y) {
            if (this._activeBuffer.y > this._activeBuffer.scrollBottom || this._activeBuffer.y < this._activeBuffer.scrollTop) return !0;
            const L = y.params[0] || 1;
            for (let R = this._activeBuffer.scrollTop; R <= this._activeBuffer.scrollBottom; ++R) {
              const D = this._activeBuffer.lines.get(this._activeBuffer.ybase + R);
              D.deleteCells(this._activeBuffer.x, L, this._activeBuffer.getNullCell(this._eraseAttrData())), D.isWrapped = !1;
            }
            return this._dirtyRowTracker.markRangeDirty(this._activeBuffer.scrollTop, this._activeBuffer.scrollBottom), !0;
          }
          eraseChars(y) {
            this._restrictCursor();
            const L = this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y);
            return L && (L.replaceCells(this._activeBuffer.x, this._activeBuffer.x + (y.params[0] || 1), this._activeBuffer.getNullCell(this._eraseAttrData())), this._dirtyRowTracker.markDirty(this._activeBuffer.y)), !0;
          }
          repeatPrecedingCharacter(y) {
            const L = this._parser.precedingJoinState;
            if (!L) return !0;
            const R = y.params[0] || 1, D = l.UnicodeService.extractWidth(L), F = this._activeBuffer.x - D, U = this._activeBuffer.lines.get(this._activeBuffer.ybase + this._activeBuffer.y).getString(F), K = new Uint32Array(U.length * R);
            let q = 0;
            for (let E = 0; E < U.length; ) {
              const H = U.codePointAt(E) || 0;
              K[q++] = H, E += H > 65535 ? 2 : 1;
            }
            let O = q;
            for (let E = 1; E < R; ++E) K.copyWithin(O, 0, q), O += q;
            return this.print(K, 0, O), !0;
          }
          sendDeviceAttributesPrimary(y) {
            return y.params[0] > 0 || (this._is("xterm") || this._is("rxvt-unicode") || this._is("screen") ? this._coreService.triggerDataEvent(r.C0.ESC + "[?1;2c") : this._is("linux") && this._coreService.triggerDataEvent(r.C0.ESC + "[?6c")), !0;
          }
          sendDeviceAttributesSecondary(y) {
            return y.params[0] > 0 || (this._is("xterm") ? this._coreService.triggerDataEvent(r.C0.ESC + "[>0;276;0c") : this._is("rxvt-unicode") ? this._coreService.triggerDataEvent(r.C0.ESC + "[>85;95;0c") : this._is("linux") ? this._coreService.triggerDataEvent(y.params[0] + "c") : this._is("screen") && this._coreService.triggerDataEvent(r.C0.ESC + "[>83;40003;0c")), !0;
          }
          _is(y) {
            return (this._optionsService.rawOptions.termName + "").indexOf(y) === 0;
          }
          setMode(y) {
            for (let L = 0; L < y.length; L++) switch (y.params[L]) {
              case 4:
                this._coreService.modes.insertMode = !0;
                break;
              case 20:
                this._optionsService.options.convertEol = !0;
            }
            return !0;
          }
          setModePrivate(y) {
            for (let L = 0; L < y.length; L++) switch (y.params[L]) {
              case 1:
                this._coreService.decPrivateModes.applicationCursorKeys = !0;
                break;
              case 2:
                this._charsetService.setgCharset(0, d.DEFAULT_CHARSET), this._charsetService.setgCharset(1, d.DEFAULT_CHARSET), this._charsetService.setgCharset(2, d.DEFAULT_CHARSET), this._charsetService.setgCharset(3, d.DEFAULT_CHARSET);
                break;
              case 3:
                this._optionsService.rawOptions.windowOptions.setWinLines && (this._bufferService.resize(132, this._bufferService.rows), this._onRequestReset.fire());
                break;
              case 6:
                this._coreService.decPrivateModes.origin = !0, this._setCursor(0, 0);
                break;
              case 7:
                this._coreService.decPrivateModes.wraparound = !0;
                break;
              case 12:
                this._optionsService.options.cursorBlink = !0;
                break;
              case 45:
                this._coreService.decPrivateModes.reverseWraparound = !0;
                break;
              case 66:
                this._logService.debug("Serial port requested application keypad."), this._coreService.decPrivateModes.applicationKeypad = !0, this._onRequestSyncScrollBar.fire();
                break;
              case 9:
                this._coreMouseService.activeProtocol = "X10";
                break;
              case 1e3:
                this._coreMouseService.activeProtocol = "VT200";
                break;
              case 1002:
                this._coreMouseService.activeProtocol = "DRAG";
                break;
              case 1003:
                this._coreMouseService.activeProtocol = "ANY";
                break;
              case 1004:
                this._coreService.decPrivateModes.sendFocus = !0, this._onRequestSendFocus.fire();
                break;
              case 1005:
                this._logService.debug("DECSET 1005 not supported (see #2507)");
                break;
              case 1006:
                this._coreMouseService.activeEncoding = "SGR";
                break;
              case 1015:
                this._logService.debug("DECSET 1015 not supported (see #2507)");
                break;
              case 1016:
                this._coreMouseService.activeEncoding = "SGR_PIXELS";
                break;
              case 25:
                this._coreService.isCursorHidden = !1;
                break;
              case 1048:
                this.saveCursor();
                break;
              case 1049:
                this.saveCursor();
              case 47:
              case 1047:
                this._bufferService.buffers.activateAltBuffer(this._eraseAttrData()), this._coreService.isCursorInitialized = !0, this._onRequestRefreshRows.fire(0, this._bufferService.rows - 1), this._onRequestSyncScrollBar.fire();
                break;
              case 2004:
                this._coreService.decPrivateModes.bracketedPasteMode = !0;
            }
            return !0;
          }
          resetMode(y) {
            for (let L = 0; L < y.length; L++) switch (y.params[L]) {
              case 4:
                this._coreService.modes.insertMode = !1;
                break;
              case 20:
                this._optionsService.options.convertEol = !1;
            }
            return !0;
          }
          resetModePrivate(y) {
            for (let L = 0; L < y.length; L++) switch (y.params[L]) {
              case 1:
                this._coreService.decPrivateModes.applicationCursorKeys = !1;
                break;
              case 3:
                this._optionsService.rawOptions.windowOptions.setWinLines && (this._bufferService.resize(80, this._bufferService.rows), this._onRequestReset.fire());
                break;
              case 6:
                this._coreService.decPrivateModes.origin = !1, this._setCursor(0, 0);
                break;
              case 7:
                this._coreService.decPrivateModes.wraparound = !1;
                break;
              case 12:
                this._optionsService.options.cursorBlink = !1;
                break;
              case 45:
                this._coreService.decPrivateModes.reverseWraparound = !1;
                break;
              case 66:
                this._logService.debug("Switching back to normal keypad."), this._coreService.decPrivateModes.applicationKeypad = !1, this._onRequestSyncScrollBar.fire();
                break;
              case 9:
              case 1e3:
              case 1002:
              case 1003:
                this._coreMouseService.activeProtocol = "NONE";
                break;
              case 1004:
                this._coreService.decPrivateModes.sendFocus = !1;
                break;
              case 1005:
                this._logService.debug("DECRST 1005 not supported (see #2507)");
                break;
              case 1006:
              case 1016:
                this._coreMouseService.activeEncoding = "DEFAULT";
                break;
              case 1015:
                this._logService.debug("DECRST 1015 not supported (see #2507)");
                break;
              case 25:
                this._coreService.isCursorHidden = !0;
                break;
              case 1048:
                this.restoreCursor();
                break;
              case 1049:
              case 47:
              case 1047:
                this._bufferService.buffers.activateNormalBuffer(), y.params[L] === 1049 && this.restoreCursor(), this._coreService.isCursorInitialized = !0, this._onRequestRefreshRows.fire(0, this._bufferService.rows - 1), this._onRequestSyncScrollBar.fire();
                break;
              case 2004:
                this._coreService.decPrivateModes.bracketedPasteMode = !1;
            }
            return !0;
          }
          requestMode(y, L) {
            const R = this._coreService.decPrivateModes, { activeProtocol: D, activeEncoding: F } = this._coreMouseService, U = this._coreService, { buffers: K, cols: q } = this._bufferService, { active: O, alt: E } = K, H = this._optionsService.rawOptions, N = (V) => V ? 1 : 2, G = y.params[0];
            return j = G, ie = L ? G === 2 ? 4 : G === 4 ? N(U.modes.insertMode) : G === 12 ? 3 : G === 20 ? N(H.convertEol) : 0 : G === 1 ? N(R.applicationCursorKeys) : G === 3 ? H.windowOptions.setWinLines ? q === 80 ? 2 : q === 132 ? 1 : 0 : 0 : G === 6 ? N(R.origin) : G === 7 ? N(R.wraparound) : G === 8 ? 3 : G === 9 ? N(D === "X10") : G === 12 ? N(H.cursorBlink) : G === 25 ? N(!U.isCursorHidden) : G === 45 ? N(R.reverseWraparound) : G === 66 ? N(R.applicationKeypad) : G === 67 ? 4 : G === 1e3 ? N(D === "VT200") : G === 1002 ? N(D === "DRAG") : G === 1003 ? N(D === "ANY") : G === 1004 ? N(R.sendFocus) : G === 1005 ? 4 : G === 1006 ? N(F === "SGR") : G === 1015 ? 4 : G === 1016 ? N(F === "SGR_PIXELS") : G === 1048 ? 1 : G === 47 || G === 1047 || G === 1049 ? N(O === E) : G === 2004 ? N(R.bracketedPasteMode) : 0, U.triggerDataEvent(`${r.C0.ESC}[${L ? "" : "?"}${j};${ie}$y`), !0;
            var j, ie;
          }
          _updateAttrColor(y, L, R, D, F) {
            return L === 2 ? (y |= 50331648, y &= -16777216, y |= u.AttributeData.fromColorRGB([R, D, F])) : L === 5 && (y &= -50331904, y |= 33554432 | 255 & R), y;
          }
          _extractColor(y, L, R) {
            const D = [0, 0, -1, 0, 0, 0];
            let F = 0, U = 0;
            do {
              if (D[U + F] = y.params[L + U], y.hasSubParams(L + U)) {
                const K = y.getSubParams(L + U);
                let q = 0;
                do
                  D[1] === 5 && (F = 1), D[U + q + 1 + F] = K[q];
                while (++q < K.length && q + U + 1 + F < D.length);
                break;
              }
              if (D[1] === 5 && U + F >= 2 || D[1] === 2 && U + F >= 5) break;
              D[1] && (F = 1);
            } while (++U + L < y.length && U + F < D.length);
            for (let K = 2; K < D.length; ++K) D[K] === -1 && (D[K] = 0);
            switch (D[0]) {
              case 38:
                R.fg = this._updateAttrColor(R.fg, D[1], D[3], D[4], D[5]);
                break;
              case 48:
                R.bg = this._updateAttrColor(R.bg, D[1], D[3], D[4], D[5]);
                break;
              case 58:
                R.extended = R.extended.clone(), R.extended.underlineColor = this._updateAttrColor(R.extended.underlineColor, D[1], D[3], D[4], D[5]);
            }
            return U;
          }
          _processUnderline(y, L) {
            L.extended = L.extended.clone(), (!~y || y > 5) && (y = 1), L.extended.underlineStyle = y, L.fg |= 268435456, y === 0 && (L.fg &= -268435457), L.updateExtended();
          }
          _processSGR0(y) {
            y.fg = e.DEFAULT_ATTR_DATA.fg, y.bg = e.DEFAULT_ATTR_DATA.bg, y.extended = y.extended.clone(), y.extended.underlineStyle = 0, y.extended.underlineColor &= -67108864, y.updateExtended();
          }
          charAttributes(y) {
            if (y.length === 1 && y.params[0] === 0) return this._processSGR0(this._curAttrData), !0;
            const L = y.length;
            let R;
            const D = this._curAttrData;
            for (let F = 0; F < L; F++) R = y.params[F], R >= 30 && R <= 37 ? (D.fg &= -50331904, D.fg |= 16777216 | R - 30) : R >= 40 && R <= 47 ? (D.bg &= -50331904, D.bg |= 16777216 | R - 40) : R >= 90 && R <= 97 ? (D.fg &= -50331904, D.fg |= 16777224 | R - 90) : R >= 100 && R <= 107 ? (D.bg &= -50331904, D.bg |= 16777224 | R - 100) : R === 0 ? this._processSGR0(D) : R === 1 ? D.fg |= 134217728 : R === 3 ? D.bg |= 67108864 : R === 4 ? (D.fg |= 268435456, this._processUnderline(y.hasSubParams(F) ? y.getSubParams(F)[0] : 1, D)) : R === 5 ? D.fg |= 536870912 : R === 7 ? D.fg |= 67108864 : R === 8 ? D.fg |= 1073741824 : R === 9 ? D.fg |= 2147483648 : R === 2 ? D.bg |= 134217728 : R === 21 ? this._processUnderline(2, D) : R === 22 ? (D.fg &= -134217729, D.bg &= -134217729) : R === 23 ? D.bg &= -67108865 : R === 24 ? (D.fg &= -268435457, this._processUnderline(0, D)) : R === 25 ? D.fg &= -536870913 : R === 27 ? D.fg &= -67108865 : R === 28 ? D.fg &= -1073741825 : R === 29 ? D.fg &= 2147483647 : R === 39 ? (D.fg &= -67108864, D.fg |= 16777215 & e.DEFAULT_ATTR_DATA.fg) : R === 49 ? (D.bg &= -67108864, D.bg |= 16777215 & e.DEFAULT_ATTR_DATA.bg) : R === 38 || R === 48 || R === 58 ? F += this._extractColor(y, F, D) : R === 53 ? D.bg |= 1073741824 : R === 55 ? D.bg &= -1073741825 : R === 59 ? (D.extended = D.extended.clone(), D.extended.underlineColor = -1, D.updateExtended()) : R === 100 ? (D.fg &= -67108864, D.fg |= 16777215 & e.DEFAULT_ATTR_DATA.fg, D.bg &= -67108864, D.bg |= 16777215 & e.DEFAULT_ATTR_DATA.bg) : this._logService.debug("Unknown SGR attribute: %d.", R);
            return !0;
          }
          deviceStatus(y) {
            switch (y.params[0]) {
              case 5:
                this._coreService.triggerDataEvent(`${r.C0.ESC}[0n`);
                break;
              case 6:
                const L = this._activeBuffer.y + 1, R = this._activeBuffer.x + 1;
                this._coreService.triggerDataEvent(`${r.C0.ESC}[${L};${R}R`);
            }
            return !0;
          }
          deviceStatusPrivate(y) {
            if (y.params[0] === 6) {
              const L = this._activeBuffer.y + 1, R = this._activeBuffer.x + 1;
              this._coreService.triggerDataEvent(`${r.C0.ESC}[?${L};${R}R`);
            }
            return !0;
          }
          softReset(y) {
            return this._coreService.isCursorHidden = !1, this._onRequestSyncScrollBar.fire(), this._activeBuffer.scrollTop = 0, this._activeBuffer.scrollBottom = this._bufferService.rows - 1, this._curAttrData = e.DEFAULT_ATTR_DATA.clone(), this._coreService.reset(), this._charsetService.reset(), this._activeBuffer.savedX = 0, this._activeBuffer.savedY = this._activeBuffer.ybase, this._activeBuffer.savedCurAttrData.fg = this._curAttrData.fg, this._activeBuffer.savedCurAttrData.bg = this._curAttrData.bg, this._activeBuffer.savedCharset = this._charsetService.charset, this._coreService.decPrivateModes.origin = !1, !0;
          }
          setCursorStyle(y) {
            const L = y.params[0] || 1;
            switch (L) {
              case 1:
              case 2:
                this._optionsService.options.cursorStyle = "block";
                break;
              case 3:
              case 4:
                this._optionsService.options.cursorStyle = "underline";
                break;
              case 5:
              case 6:
                this._optionsService.options.cursorStyle = "bar";
            }
            const R = L % 2 == 1;
            return this._optionsService.options.cursorBlink = R, !0;
          }
          setScrollRegion(y) {
            const L = y.params[0] || 1;
            let R;
            return (y.length < 2 || (R = y.params[1]) > this._bufferService.rows || R === 0) && (R = this._bufferService.rows), R > L && (this._activeBuffer.scrollTop = L - 1, this._activeBuffer.scrollBottom = R - 1, this._setCursor(0, 0)), !0;
          }
          windowOptions(y) {
            if (!S(y.params[0], this._optionsService.rawOptions.windowOptions)) return !0;
            const L = y.length > 1 ? y.params[1] : 0;
            switch (y.params[0]) {
              case 14:
                L !== 2 && this._onRequestWindowsOptionsReport.fire(b.GET_WIN_SIZE_PIXELS);
                break;
              case 16:
                this._onRequestWindowsOptionsReport.fire(b.GET_CELL_SIZE_PIXELS);
                break;
              case 18:
                this._bufferService && this._coreService.triggerDataEvent(`${r.C0.ESC}[8;${this._bufferService.rows};${this._bufferService.cols}t`);
                break;
              case 22:
                L !== 0 && L !== 2 || (this._windowTitleStack.push(this._windowTitle), this._windowTitleStack.length > 10 && this._windowTitleStack.shift()), L !== 0 && L !== 1 || (this._iconNameStack.push(this._iconName), this._iconNameStack.length > 10 && this._iconNameStack.shift());
                break;
              case 23:
                L !== 0 && L !== 2 || this._windowTitleStack.length && this.setTitle(this._windowTitleStack.pop()), L !== 0 && L !== 1 || this._iconNameStack.length && this.setIconName(this._iconNameStack.pop());
            }
            return !0;
          }
          saveCursor(y) {
            return this._activeBuffer.savedX = this._activeBuffer.x, this._activeBuffer.savedY = this._activeBuffer.ybase + this._activeBuffer.y, this._activeBuffer.savedCurAttrData.fg = this._curAttrData.fg, this._activeBuffer.savedCurAttrData.bg = this._curAttrData.bg, this._activeBuffer.savedCharset = this._charsetService.charset, !0;
          }
          restoreCursor(y) {
            return this._activeBuffer.x = this._activeBuffer.savedX || 0, this._activeBuffer.y = Math.max(this._activeBuffer.savedY - this._activeBuffer.ybase, 0), this._curAttrData.fg = this._activeBuffer.savedCurAttrData.fg, this._curAttrData.bg = this._activeBuffer.savedCurAttrData.bg, this._charsetService.charset = this._savedCharset, this._activeBuffer.savedCharset && (this._charsetService.charset = this._activeBuffer.savedCharset), this._restrictCursor(), !0;
          }
          setTitle(y) {
            return this._windowTitle = y, this._onTitleChange.fire(y), !0;
          }
          setIconName(y) {
            return this._iconName = y, !0;
          }
          setOrReportIndexedColor(y) {
            const L = [], R = y.split(";");
            for (; R.length > 1; ) {
              const D = R.shift(), F = R.shift();
              if (/^\d+$/.exec(D)) {
                const U = parseInt(D);
                if (k(U)) if (F === "?") L.push({ type: 0, index: U });
                else {
                  const K = (0, v.parseColor)(F);
                  K && L.push({ type: 1, index: U, color: K });
                }
              }
            }
            return L.length && this._onColor.fire(L), !0;
          }
          setHyperlink(y) {
            const L = y.split(";");
            return !(L.length < 2) && (L[1] ? this._createHyperlink(L[0], L[1]) : !L[0] && this._finishHyperlink());
          }
          _createHyperlink(y, L) {
            this._getCurrentLinkId() && this._finishHyperlink();
            const R = y.split(":");
            let D;
            const F = R.findIndex(((U) => U.startsWith("id=")));
            return F !== -1 && (D = R[F].slice(3) || void 0), this._curAttrData.extended = this._curAttrData.extended.clone(), this._curAttrData.extended.urlId = this._oscLinkService.registerLink({ id: D, uri: L }), this._curAttrData.updateExtended(), !0;
          }
          _finishHyperlink() {
            return this._curAttrData.extended = this._curAttrData.extended.clone(), this._curAttrData.extended.urlId = 0, this._curAttrData.updateExtended(), !0;
          }
          _setOrReportSpecialColor(y, L) {
            const R = y.split(";");
            for (let D = 0; D < R.length && !(L >= this._specialColors.length); ++D, ++L) if (R[D] === "?") this._onColor.fire([{ type: 0, index: this._specialColors[L] }]);
            else {
              const F = (0, v.parseColor)(R[D]);
              F && this._onColor.fire([{ type: 1, index: this._specialColors[L], color: F }]);
            }
            return !0;
          }
          setOrReportFgColor(y) {
            return this._setOrReportSpecialColor(y, 0);
          }
          setOrReportBgColor(y) {
            return this._setOrReportSpecialColor(y, 1);
          }
          setOrReportCursorColor(y) {
            return this._setOrReportSpecialColor(y, 2);
          }
          restoreIndexedColor(y) {
            if (!y) return this._onColor.fire([{ type: 2 }]), !0;
            const L = [], R = y.split(";");
            for (let D = 0; D < R.length; ++D) if (/^\d+$/.exec(R[D])) {
              const F = parseInt(R[D]);
              k(F) && L.push({ type: 2, index: F });
            }
            return L.length && this._onColor.fire(L), !0;
          }
          restoreFgColor(y) {
            return this._onColor.fire([{ type: 2, index: 256 }]), !0;
          }
          restoreBgColor(y) {
            return this._onColor.fire([{ type: 2, index: 257 }]), !0;
          }
          restoreCursorColor(y) {
            return this._onColor.fire([{ type: 2, index: 258 }]), !0;
          }
          nextLine() {
            return this._activeBuffer.x = 0, this.index(), !0;
          }
          keypadApplicationMode() {
            return this._logService.debug("Serial port requested application keypad."), this._coreService.decPrivateModes.applicationKeypad = !0, this._onRequestSyncScrollBar.fire(), !0;
          }
          keypadNumericMode() {
            return this._logService.debug("Switching back to normal keypad."), this._coreService.decPrivateModes.applicationKeypad = !1, this._onRequestSyncScrollBar.fire(), !0;
          }
          selectDefaultCharset() {
            return this._charsetService.setgLevel(0), this._charsetService.setgCharset(0, d.DEFAULT_CHARSET), !0;
          }
          selectCharset(y) {
            return y.length !== 2 ? (this.selectDefaultCharset(), !0) : (y[0] === "/" || this._charsetService.setgCharset(C[y[0]], d.CHARSETS[y[1]] || d.DEFAULT_CHARSET), !0);
          }
          index() {
            return this._restrictCursor(), this._activeBuffer.y++, this._activeBuffer.y === this._activeBuffer.scrollBottom + 1 ? (this._activeBuffer.y--, this._bufferService.scroll(this._eraseAttrData())) : this._activeBuffer.y >= this._bufferService.rows && (this._activeBuffer.y = this._bufferService.rows - 1), this._restrictCursor(), !0;
          }
          tabSet() {
            return this._activeBuffer.tabs[this._activeBuffer.x] = !0, !0;
          }
          reverseIndex() {
            if (this._restrictCursor(), this._activeBuffer.y === this._activeBuffer.scrollTop) {
              const y = this._activeBuffer.scrollBottom - this._activeBuffer.scrollTop;
              this._activeBuffer.lines.shiftElements(this._activeBuffer.ybase + this._activeBuffer.y, y, 1), this._activeBuffer.lines.set(this._activeBuffer.ybase + this._activeBuffer.y, this._activeBuffer.getBlankLine(this._eraseAttrData())), this._dirtyRowTracker.markRangeDirty(this._activeBuffer.scrollTop, this._activeBuffer.scrollBottom);
            } else this._activeBuffer.y--, this._restrictCursor();
            return !0;
          }
          fullReset() {
            return this._parser.reset(), this._onRequestReset.fire(), !0;
          }
          reset() {
            this._curAttrData = e.DEFAULT_ATTR_DATA.clone(), this._eraseAttrDataInternal = e.DEFAULT_ATTR_DATA.clone();
          }
          _eraseAttrData() {
            return this._eraseAttrDataInternal.bg &= -67108864, this._eraseAttrDataInternal.bg |= 67108863 & this._curAttrData.bg, this._eraseAttrDataInternal;
          }
          setgLevel(y) {
            return this._charsetService.setgLevel(y), !0;
          }
          screenAlignmentPattern() {
            const y = new i.CellData();
            y.content = 4194373, y.fg = this._curAttrData.fg, y.bg = this._curAttrData.bg, this._setCursor(0, 0);
            for (let L = 0; L < this._bufferService.rows; ++L) {
              const R = this._activeBuffer.ybase + this._activeBuffer.y + L, D = this._activeBuffer.lines.get(R);
              D && (D.fill(y), D.isWrapped = !1);
            }
            return this._dirtyRowTracker.markAllDirty(), this._setCursor(0, 0), !0;
          }
          requestStatusString(y, L) {
            const R = this._bufferService.buffer, D = this._optionsService.rawOptions;
            return ((F) => (this._coreService.triggerDataEvent(`${r.C0.ESC}${F}${r.C0.ESC}\\`), !0))(y === '"q' ? `P1$r${this._curAttrData.isProtected() ? 1 : 0}"q` : y === '"p' ? 'P1$r61;1"p' : y === "r" ? `P1$r${R.scrollTop + 1};${R.scrollBottom + 1}r` : y === "m" ? "P1$r0m" : y === " q" ? `P1$r${{ block: 2, underline: 4, bar: 6 }[D.cursorStyle] - (D.cursorBlink ? 1 : 0)} q` : "P0$r");
          }
          markRangeDirty(y, L) {
            this._dirtyRowTracker.markRangeDirty(y, L);
          }
        }
        t.InputHandler = A;
        let P = class {
          constructor(M) {
            this._bufferService = M, this.clearRange();
          }
          clearRange() {
            this.start = this._bufferService.buffer.y, this.end = this._bufferService.buffer.y;
          }
          markDirty(M) {
            M < this.start ? this.start = M : M > this.end && (this.end = M);
          }
          markRangeDirty(M, y) {
            M > y && (x = M, M = y, y = x), M < this.start && (this.start = M), y > this.end && (this.end = y);
          }
          markAllDirty() {
            this.markRangeDirty(0, this._bufferService.rows - 1);
          }
        };
        function k(M) {
          return 0 <= M && M < 256;
        }
        P = c([h(0, p.IBufferService)], P);
      }, 844: (T, t) => {
        function a(c) {
          for (const h of c) h.dispose();
          c.length = 0;
        }
        Object.defineProperty(t, "__esModule", { value: !0 }), t.getDisposeArrayDisposable = t.disposeArray = t.toDisposable = t.MutableDisposable = t.Disposable = void 0, t.Disposable = class {
          constructor() {
            this._disposables = [], this._isDisposed = !1;
          }
          dispose() {
            this._isDisposed = !0;
            for (const c of this._disposables) c.dispose();
            this._disposables.length = 0;
          }
          register(c) {
            return this._disposables.push(c), c;
          }
          unregister(c) {
            const h = this._disposables.indexOf(c);
            h !== -1 && this._disposables.splice(h, 1);
          }
        }, t.MutableDisposable = class {
          constructor() {
            this._isDisposed = !1;
          }
          get value() {
            return this._isDisposed ? void 0 : this._value;
          }
          set value(c) {
            var h;
            this._isDisposed || c === this._value || ((h = this._value) == null || h.dispose(), this._value = c);
          }
          clear() {
            this.value = void 0;
          }
          dispose() {
            var c;
            this._isDisposed = !0, (c = this._value) == null || c.dispose(), this._value = void 0;
          }
        }, t.toDisposable = function(c) {
          return { dispose: c };
        }, t.disposeArray = a, t.getDisposeArrayDisposable = function(c) {
          return { dispose: () => a(c) };
        };
      }, 1505: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.FourKeyMap = t.TwoKeyMap = void 0;
        class a {
          constructor() {
            this._data = {};
          }
          set(h, r, d) {
            this._data[h] || (this._data[h] = {}), this._data[h][r] = d;
          }
          get(h, r) {
            return this._data[h] ? this._data[h][r] : void 0;
          }
          clear() {
            this._data = {};
          }
        }
        t.TwoKeyMap = a, t.FourKeyMap = class {
          constructor() {
            this._data = new a();
          }
          set(c, h, r, d, f) {
            this._data.get(c, h) || this._data.set(c, h, new a()), this._data.get(c, h).set(r, d, f);
          }
          get(c, h, r, d) {
            var f;
            return (f = this._data.get(c, h)) == null ? void 0 : f.get(r, d);
          }
          clear() {
            this._data.clear();
          }
        };
      }, 6114: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.isChromeOS = t.isLinux = t.isWindows = t.isIphone = t.isIpad = t.isMac = t.getSafariVersion = t.isSafari = t.isLegacyEdge = t.isFirefox = t.isNode = void 0, t.isNode = typeof process != "undefined" && "title" in process;
        const a = t.isNode ? "node" : navigator.userAgent, c = t.isNode ? "node" : navigator.platform;
        t.isFirefox = a.includes("Firefox"), t.isLegacyEdge = a.includes("Edge"), t.isSafari = /^((?!chrome|android).)*safari/i.test(a), t.getSafariVersion = function() {
          if (!t.isSafari) return 0;
          const h = a.match(/Version\/(\d+)/);
          return h === null || h.length < 2 ? 0 : parseInt(h[1]);
        }, t.isMac = ["Macintosh", "MacIntel", "MacPPC", "Mac68K"].includes(c), t.isIpad = c === "iPad", t.isIphone = c === "iPhone", t.isWindows = ["Windows", "Win16", "Win32", "WinCE"].includes(c), t.isLinux = c.indexOf("Linux") >= 0, t.isChromeOS = /\bCrOS\b/.test(a);
      }, 6106: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.SortedList = void 0;
        let a = 0;
        t.SortedList = class {
          constructor(c) {
            this._getKey = c, this._array = [];
          }
          clear() {
            this._array.length = 0;
          }
          insert(c) {
            this._array.length !== 0 ? (a = this._search(this._getKey(c)), this._array.splice(a, 0, c)) : this._array.push(c);
          }
          delete(c) {
            if (this._array.length === 0) return !1;
            const h = this._getKey(c);
            if (h === void 0 || (a = this._search(h), a === -1) || this._getKey(this._array[a]) !== h) return !1;
            do
              if (this._array[a] === c) return this._array.splice(a, 1), !0;
            while (++a < this._array.length && this._getKey(this._array[a]) === h);
            return !1;
          }
          *getKeyIterator(c) {
            if (this._array.length !== 0 && (a = this._search(c), !(a < 0 || a >= this._array.length) && this._getKey(this._array[a]) === c)) do
              yield this._array[a];
            while (++a < this._array.length && this._getKey(this._array[a]) === c);
          }
          forEachByKey(c, h) {
            if (this._array.length !== 0 && (a = this._search(c), !(a < 0 || a >= this._array.length) && this._getKey(this._array[a]) === c)) do
              h(this._array[a]);
            while (++a < this._array.length && this._getKey(this._array[a]) === c);
          }
          values() {
            return [...this._array].values();
          }
          _search(c) {
            let h = 0, r = this._array.length - 1;
            for (; r >= h; ) {
              let d = h + r >> 1;
              const f = this._getKey(this._array[d]);
              if (f > c) r = d - 1;
              else {
                if (!(f < c)) {
                  for (; d > 0 && this._getKey(this._array[d - 1]) === c; ) d--;
                  return d;
                }
                h = d + 1;
              }
            }
            return h;
          }
        };
      }, 7226: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.DebouncedIdleTask = t.IdleTaskQueue = t.PriorityTaskQueue = void 0;
        const c = a(6114);
        class h {
          constructor() {
            this._tasks = [], this._i = 0;
          }
          enqueue(f) {
            this._tasks.push(f), this._start();
          }
          flush() {
            for (; this._i < this._tasks.length; ) this._tasks[this._i]() || this._i++;
            this.clear();
          }
          clear() {
            this._idleCallback && (this._cancelCallback(this._idleCallback), this._idleCallback = void 0), this._i = 0, this._tasks.length = 0;
          }
          _start() {
            this._idleCallback || (this._idleCallback = this._requestCallback(this._process.bind(this)));
          }
          _process(f) {
            this._idleCallback = void 0;
            let g = 0, n = 0, e = f.timeRemaining(), o = 0;
            for (; this._i < this._tasks.length; ) {
              if (g = Date.now(), this._tasks[this._i]() || this._i++, g = Math.max(1, Date.now() - g), n = Math.max(g, n), o = f.timeRemaining(), 1.5 * n > o) return e - g < -20 && console.warn(`task queue exceeded allotted deadline by ${Math.abs(Math.round(e - g))}ms`), void this._start();
              e = o;
            }
            this.clear();
          }
        }
        class r extends h {
          _requestCallback(f) {
            return setTimeout((() => f(this._createDeadline(16))));
          }
          _cancelCallback(f) {
            clearTimeout(f);
          }
          _createDeadline(f) {
            const g = Date.now() + f;
            return { timeRemaining: () => Math.max(0, g - Date.now()) };
          }
        }
        t.PriorityTaskQueue = r, t.IdleTaskQueue = !c.isNode && "requestIdleCallback" in window ? class extends h {
          _requestCallback(d) {
            return requestIdleCallback(d);
          }
          _cancelCallback(d) {
            cancelIdleCallback(d);
          }
        } : r, t.DebouncedIdleTask = class {
          constructor() {
            this._queue = new t.IdleTaskQueue();
          }
          set(d) {
            this._queue.clear(), this._queue.enqueue(d);
          }
          flush() {
            this._queue.flush();
          }
        };
      }, 9282: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.updateWindowsModeWrappedState = void 0;
        const c = a(643);
        t.updateWindowsModeWrappedState = function(h) {
          const r = h.buffer.lines.get(h.buffer.ybase + h.buffer.y - 1), d = r == null ? void 0 : r.get(h.cols - 1), f = h.buffer.lines.get(h.buffer.ybase + h.buffer.y);
          f && d && (f.isWrapped = d[c.CHAR_DATA_CODE_INDEX] !== c.NULL_CELL_CODE && d[c.CHAR_DATA_CODE_INDEX] !== c.WHITESPACE_CELL_CODE);
        };
      }, 3734: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.ExtendedAttrs = t.AttributeData = void 0;
        class a {
          constructor() {
            this.fg = 0, this.bg = 0, this.extended = new c();
          }
          static toColorRGB(r) {
            return [r >>> 16 & 255, r >>> 8 & 255, 255 & r];
          }
          static fromColorRGB(r) {
            return (255 & r[0]) << 16 | (255 & r[1]) << 8 | 255 & r[2];
          }
          clone() {
            const r = new a();
            return r.fg = this.fg, r.bg = this.bg, r.extended = this.extended.clone(), r;
          }
          isInverse() {
            return 67108864 & this.fg;
          }
          isBold() {
            return 134217728 & this.fg;
          }
          isUnderline() {
            return this.hasExtendedAttrs() && this.extended.underlineStyle !== 0 ? 1 : 268435456 & this.fg;
          }
          isBlink() {
            return 536870912 & this.fg;
          }
          isInvisible() {
            return 1073741824 & this.fg;
          }
          isItalic() {
            return 67108864 & this.bg;
          }
          isDim() {
            return 134217728 & this.bg;
          }
          isStrikethrough() {
            return 2147483648 & this.fg;
          }
          isProtected() {
            return 536870912 & this.bg;
          }
          isOverline() {
            return 1073741824 & this.bg;
          }
          getFgColorMode() {
            return 50331648 & this.fg;
          }
          getBgColorMode() {
            return 50331648 & this.bg;
          }
          isFgRGB() {
            return (50331648 & this.fg) == 50331648;
          }
          isBgRGB() {
            return (50331648 & this.bg) == 50331648;
          }
          isFgPalette() {
            return (50331648 & this.fg) == 16777216 || (50331648 & this.fg) == 33554432;
          }
          isBgPalette() {
            return (50331648 & this.bg) == 16777216 || (50331648 & this.bg) == 33554432;
          }
          isFgDefault() {
            return (50331648 & this.fg) == 0;
          }
          isBgDefault() {
            return (50331648 & this.bg) == 0;
          }
          isAttributeDefault() {
            return this.fg === 0 && this.bg === 0;
          }
          getFgColor() {
            switch (50331648 & this.fg) {
              case 16777216:
              case 33554432:
                return 255 & this.fg;
              case 50331648:
                return 16777215 & this.fg;
              default:
                return -1;
            }
          }
          getBgColor() {
            switch (50331648 & this.bg) {
              case 16777216:
              case 33554432:
                return 255 & this.bg;
              case 50331648:
                return 16777215 & this.bg;
              default:
                return -1;
            }
          }
          hasExtendedAttrs() {
            return 268435456 & this.bg;
          }
          updateExtended() {
            this.extended.isEmpty() ? this.bg &= -268435457 : this.bg |= 268435456;
          }
          getUnderlineColor() {
            if (268435456 & this.bg && ~this.extended.underlineColor) switch (50331648 & this.extended.underlineColor) {
              case 16777216:
              case 33554432:
                return 255 & this.extended.underlineColor;
              case 50331648:
                return 16777215 & this.extended.underlineColor;
              default:
                return this.getFgColor();
            }
            return this.getFgColor();
          }
          getUnderlineColorMode() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? 50331648 & this.extended.underlineColor : this.getFgColorMode();
          }
          isUnderlineColorRGB() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? (50331648 & this.extended.underlineColor) == 50331648 : this.isFgRGB();
          }
          isUnderlineColorPalette() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? (50331648 & this.extended.underlineColor) == 16777216 || (50331648 & this.extended.underlineColor) == 33554432 : this.isFgPalette();
          }
          isUnderlineColorDefault() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? (50331648 & this.extended.underlineColor) == 0 : this.isFgDefault();
          }
          getUnderlineStyle() {
            return 268435456 & this.fg ? 268435456 & this.bg ? this.extended.underlineStyle : 1 : 0;
          }
          getUnderlineVariantOffset() {
            return this.extended.underlineVariantOffset;
          }
        }
        t.AttributeData = a;
        class c {
          get ext() {
            return this._urlId ? -469762049 & this._ext | this.underlineStyle << 26 : this._ext;
          }
          set ext(r) {
            this._ext = r;
          }
          get underlineStyle() {
            return this._urlId ? 5 : (469762048 & this._ext) >> 26;
          }
          set underlineStyle(r) {
            this._ext &= -469762049, this._ext |= r << 26 & 469762048;
          }
          get underlineColor() {
            return 67108863 & this._ext;
          }
          set underlineColor(r) {
            this._ext &= -67108864, this._ext |= 67108863 & r;
          }
          get urlId() {
            return this._urlId;
          }
          set urlId(r) {
            this._urlId = r;
          }
          get underlineVariantOffset() {
            const r = (3758096384 & this._ext) >> 29;
            return r < 0 ? 4294967288 ^ r : r;
          }
          set underlineVariantOffset(r) {
            this._ext &= 536870911, this._ext |= r << 29 & 3758096384;
          }
          constructor(r = 0, d = 0) {
            this._ext = 0, this._urlId = 0, this._ext = r, this._urlId = d;
          }
          clone() {
            return new c(this._ext, this._urlId);
          }
          isEmpty() {
            return this.underlineStyle === 0 && this._urlId === 0;
          }
        }
        t.ExtendedAttrs = c;
      }, 9092: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.Buffer = t.MAX_BUFFER_SIZE = void 0;
        const c = a(6349), h = a(7226), r = a(3734), d = a(8437), f = a(4634), g = a(511), n = a(643), e = a(4863), o = a(7116);
        t.MAX_BUFFER_SIZE = 4294967295, t.Buffer = class {
          constructor(s, i, u) {
            this._hasScrollback = s, this._optionsService = i, this._bufferService = u, this.ydisp = 0, this.ybase = 0, this.y = 0, this.x = 0, this.tabs = {}, this.savedY = 0, this.savedX = 0, this.savedCurAttrData = d.DEFAULT_ATTR_DATA.clone(), this.savedCharset = o.DEFAULT_CHARSET, this.markers = [], this._nullCell = g.CellData.fromCharData([0, n.NULL_CELL_CHAR, n.NULL_CELL_WIDTH, n.NULL_CELL_CODE]), this._whitespaceCell = g.CellData.fromCharData([0, n.WHITESPACE_CELL_CHAR, n.WHITESPACE_CELL_WIDTH, n.WHITESPACE_CELL_CODE]), this._isClearing = !1, this._memoryCleanupQueue = new h.IdleTaskQueue(), this._memoryCleanupPosition = 0, this._cols = this._bufferService.cols, this._rows = this._bufferService.rows, this.lines = new c.CircularList(this._getCorrectBufferLength(this._rows)), this.scrollTop = 0, this.scrollBottom = this._rows - 1, this.setupTabStops();
          }
          getNullCell(s) {
            return s ? (this._nullCell.fg = s.fg, this._nullCell.bg = s.bg, this._nullCell.extended = s.extended) : (this._nullCell.fg = 0, this._nullCell.bg = 0, this._nullCell.extended = new r.ExtendedAttrs()), this._nullCell;
          }
          getWhitespaceCell(s) {
            return s ? (this._whitespaceCell.fg = s.fg, this._whitespaceCell.bg = s.bg, this._whitespaceCell.extended = s.extended) : (this._whitespaceCell.fg = 0, this._whitespaceCell.bg = 0, this._whitespaceCell.extended = new r.ExtendedAttrs()), this._whitespaceCell;
          }
          getBlankLine(s, i) {
            return new d.BufferLine(this._bufferService.cols, this.getNullCell(s), i);
          }
          get hasScrollback() {
            return this._hasScrollback && this.lines.maxLength > this._rows;
          }
          get isCursorInViewport() {
            const s = this.ybase + this.y - this.ydisp;
            return s >= 0 && s < this._rows;
          }
          _getCorrectBufferLength(s) {
            if (!this._hasScrollback) return s;
            const i = s + this._optionsService.rawOptions.scrollback;
            return i > t.MAX_BUFFER_SIZE ? t.MAX_BUFFER_SIZE : i;
          }
          fillViewportRows(s) {
            if (this.lines.length === 0) {
              s === void 0 && (s = d.DEFAULT_ATTR_DATA);
              let i = this._rows;
              for (; i--; ) this.lines.push(this.getBlankLine(s));
            }
          }
          clear() {
            this.ydisp = 0, this.ybase = 0, this.y = 0, this.x = 0, this.lines = new c.CircularList(this._getCorrectBufferLength(this._rows)), this.scrollTop = 0, this.scrollBottom = this._rows - 1, this.setupTabStops();
          }
          resize(s, i) {
            const u = this.getNullCell(d.DEFAULT_ATTR_DATA);
            let p = 0;
            const l = this._getCorrectBufferLength(i);
            if (l > this.lines.maxLength && (this.lines.maxLength = l), this.lines.length > 0) {
              if (this._cols < s) for (let _ = 0; _ < this.lines.length; _++) p += +this.lines.get(_).resize(s, u);
              let m = 0;
              if (this._rows < i) for (let _ = this._rows; _ < i; _++) this.lines.length < i + this.ybase && (this._optionsService.rawOptions.windowsMode || this._optionsService.rawOptions.windowsPty.backend !== void 0 || this._optionsService.rawOptions.windowsPty.buildNumber !== void 0 ? this.lines.push(new d.BufferLine(s, u)) : this.ybase > 0 && this.lines.length <= this.ybase + this.y + m + 1 ? (this.ybase--, m++, this.ydisp > 0 && this.ydisp--) : this.lines.push(new d.BufferLine(s, u)));
              else for (let _ = this._rows; _ > i; _--) this.lines.length > i + this.ybase && (this.lines.length > this.ybase + this.y + 1 ? this.lines.pop() : (this.ybase++, this.ydisp++));
              if (l < this.lines.maxLength) {
                const _ = this.lines.length - l;
                _ > 0 && (this.lines.trimStart(_), this.ybase = Math.max(this.ybase - _, 0), this.ydisp = Math.max(this.ydisp - _, 0), this.savedY = Math.max(this.savedY - _, 0)), this.lines.maxLength = l;
              }
              this.x = Math.min(this.x, s - 1), this.y = Math.min(this.y, i - 1), m && (this.y += m), this.savedX = Math.min(this.savedX, s - 1), this.scrollTop = 0;
            }
            if (this.scrollBottom = i - 1, this._isReflowEnabled && (this._reflow(s, i), this._cols > s)) for (let m = 0; m < this.lines.length; m++) p += +this.lines.get(m).resize(s, u);
            this._cols = s, this._rows = i, this._memoryCleanupQueue.clear(), p > 0.1 * this.lines.length && (this._memoryCleanupPosition = 0, this._memoryCleanupQueue.enqueue((() => this._batchedMemoryCleanup())));
          }
          _batchedMemoryCleanup() {
            let s = !0;
            this._memoryCleanupPosition >= this.lines.length && (this._memoryCleanupPosition = 0, s = !1);
            let i = 0;
            for (; this._memoryCleanupPosition < this.lines.length; ) if (i += this.lines.get(this._memoryCleanupPosition++).cleanupMemory(), i > 100) return !0;
            return s;
          }
          get _isReflowEnabled() {
            const s = this._optionsService.rawOptions.windowsPty;
            return s && s.buildNumber ? this._hasScrollback && s.backend === "conpty" && s.buildNumber >= 21376 : this._hasScrollback && !this._optionsService.rawOptions.windowsMode;
          }
          _reflow(s, i) {
            this._cols !== s && (s > this._cols ? this._reflowLarger(s, i) : this._reflowSmaller(s, i));
          }
          _reflowLarger(s, i) {
            const u = (0, f.reflowLargerGetLinesToRemove)(this.lines, this._cols, s, this.ybase + this.y, this.getNullCell(d.DEFAULT_ATTR_DATA));
            if (u.length > 0) {
              const p = (0, f.reflowLargerCreateNewLayout)(this.lines, u);
              (0, f.reflowLargerApplyNewLayout)(this.lines, p.layout), this._reflowLargerAdjustViewport(s, i, p.countRemoved);
            }
          }
          _reflowLargerAdjustViewport(s, i, u) {
            const p = this.getNullCell(d.DEFAULT_ATTR_DATA);
            let l = u;
            for (; l-- > 0; ) this.ybase === 0 ? (this.y > 0 && this.y--, this.lines.length < i && this.lines.push(new d.BufferLine(s, p))) : (this.ydisp === this.ybase && this.ydisp--, this.ybase--);
            this.savedY = Math.max(this.savedY - u, 0);
          }
          _reflowSmaller(s, i) {
            const u = this.getNullCell(d.DEFAULT_ATTR_DATA), p = [];
            let l = 0;
            for (let m = this.lines.length - 1; m >= 0; m--) {
              let _ = this.lines.get(m);
              if (!_ || !_.isWrapped && _.getTrimmedLength() <= s) continue;
              const v = [_];
              for (; _.isWrapped && m > 0; ) _ = this.lines.get(--m), v.unshift(_);
              const C = this.ybase + this.y;
              if (C >= m && C < m + v.length) continue;
              const w = v[v.length - 1].getTrimmedLength(), S = (0, f.reflowSmallerGetNewLineLengths)(v, this._cols, s), b = S.length - v.length;
              let x;
              x = this.ybase === 0 && this.y !== this.lines.length - 1 ? Math.max(0, this.y - this.lines.maxLength + b) : Math.max(0, this.lines.length - this.lines.maxLength + b);
              const A = [];
              for (let R = 0; R < b; R++) {
                const D = this.getBlankLine(d.DEFAULT_ATTR_DATA, !0);
                A.push(D);
              }
              A.length > 0 && (p.push({ start: m + v.length + l, newLines: A }), l += A.length), v.push(...A);
              let P = S.length - 1, k = S[P];
              k === 0 && (P--, k = S[P]);
              let M = v.length - b - 1, y = w;
              for (; M >= 0; ) {
                const R = Math.min(y, k);
                if (v[P] === void 0) break;
                if (v[P].copyCellsFrom(v[M], y - R, k - R, R, !0), k -= R, k === 0 && (P--, k = S[P]), y -= R, y === 0) {
                  M--;
                  const D = Math.max(M, 0);
                  y = (0, f.getWrappedLineTrimmedLength)(v, D, this._cols);
                }
              }
              for (let R = 0; R < v.length; R++) S[R] < s && v[R].setCell(S[R], u);
              let L = b - x;
              for (; L-- > 0; ) this.ybase === 0 ? this.y < i - 1 ? (this.y++, this.lines.pop()) : (this.ybase++, this.ydisp++) : this.ybase < Math.min(this.lines.maxLength, this.lines.length + l) - i && (this.ybase === this.ydisp && this.ydisp++, this.ybase++);
              this.savedY = Math.min(this.savedY + b, this.ybase + i - 1);
            }
            if (p.length > 0) {
              const m = [], _ = [];
              for (let P = 0; P < this.lines.length; P++) _.push(this.lines.get(P));
              const v = this.lines.length;
              let C = v - 1, w = 0, S = p[w];
              this.lines.length = Math.min(this.lines.maxLength, this.lines.length + l);
              let b = 0;
              for (let P = Math.min(this.lines.maxLength - 1, v + l - 1); P >= 0; P--) if (S && S.start > C + b) {
                for (let k = S.newLines.length - 1; k >= 0; k--) this.lines.set(P--, S.newLines[k]);
                P++, m.push({ index: C + 1, amount: S.newLines.length }), b += S.newLines.length, S = p[++w];
              } else this.lines.set(P, _[C--]);
              let x = 0;
              for (let P = m.length - 1; P >= 0; P--) m[P].index += x, this.lines.onInsertEmitter.fire(m[P]), x += m[P].amount;
              const A = Math.max(0, v + l - this.lines.maxLength);
              A > 0 && this.lines.onTrimEmitter.fire(A);
            }
          }
          translateBufferLineToString(s, i, u = 0, p) {
            const l = this.lines.get(s);
            return l ? l.translateToString(i, u, p) : "";
          }
          getWrappedRangeForLine(s) {
            let i = s, u = s;
            for (; i > 0 && this.lines.get(i).isWrapped; ) i--;
            for (; u + 1 < this.lines.length && this.lines.get(u + 1).isWrapped; ) u++;
            return { first: i, last: u };
          }
          setupTabStops(s) {
            for (s != null ? this.tabs[s] || (s = this.prevStop(s)) : (this.tabs = {}, s = 0); s < this._cols; s += this._optionsService.rawOptions.tabStopWidth) this.tabs[s] = !0;
          }
          prevStop(s) {
            for (s == null && (s = this.x); !this.tabs[--s] && s > 0; ) ;
            return s >= this._cols ? this._cols - 1 : s < 0 ? 0 : s;
          }
          nextStop(s) {
            for (s == null && (s = this.x); !this.tabs[++s] && s < this._cols; ) ;
            return s >= this._cols ? this._cols - 1 : s < 0 ? 0 : s;
          }
          clearMarkers(s) {
            this._isClearing = !0;
            for (let i = 0; i < this.markers.length; i++) this.markers[i].line === s && (this.markers[i].dispose(), this.markers.splice(i--, 1));
            this._isClearing = !1;
          }
          clearAllMarkers() {
            this._isClearing = !0;
            for (let s = 0; s < this.markers.length; s++) this.markers[s].dispose(), this.markers.splice(s--, 1);
            this._isClearing = !1;
          }
          addMarker(s) {
            const i = new e.Marker(s);
            return this.markers.push(i), i.register(this.lines.onTrim(((u) => {
              i.line -= u, i.line < 0 && i.dispose();
            }))), i.register(this.lines.onInsert(((u) => {
              i.line >= u.index && (i.line += u.amount);
            }))), i.register(this.lines.onDelete(((u) => {
              i.line >= u.index && i.line < u.index + u.amount && i.dispose(), i.line > u.index && (i.line -= u.amount);
            }))), i.register(i.onDispose((() => this._removeMarker(i)))), i;
          }
          _removeMarker(s) {
            this._isClearing || this.markers.splice(this.markers.indexOf(s), 1);
          }
        };
      }, 8437: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.BufferLine = t.DEFAULT_ATTR_DATA = void 0;
        const c = a(3734), h = a(511), r = a(643), d = a(482);
        t.DEFAULT_ATTR_DATA = Object.freeze(new c.AttributeData());
        let f = 0;
        class g {
          constructor(e, o, s = !1) {
            this.isWrapped = s, this._combined = {}, this._extendedAttrs = {}, this._data = new Uint32Array(3 * e);
            const i = o || h.CellData.fromCharData([0, r.NULL_CELL_CHAR, r.NULL_CELL_WIDTH, r.NULL_CELL_CODE]);
            for (let u = 0; u < e; ++u) this.setCell(u, i);
            this.length = e;
          }
          get(e) {
            const o = this._data[3 * e + 0], s = 2097151 & o;
            return [this._data[3 * e + 1], 2097152 & o ? this._combined[e] : s ? (0, d.stringFromCodePoint)(s) : "", o >> 22, 2097152 & o ? this._combined[e].charCodeAt(this._combined[e].length - 1) : s];
          }
          set(e, o) {
            this._data[3 * e + 1] = o[r.CHAR_DATA_ATTR_INDEX], o[r.CHAR_DATA_CHAR_INDEX].length > 1 ? (this._combined[e] = o[1], this._data[3 * e + 0] = 2097152 | e | o[r.CHAR_DATA_WIDTH_INDEX] << 22) : this._data[3 * e + 0] = o[r.CHAR_DATA_CHAR_INDEX].charCodeAt(0) | o[r.CHAR_DATA_WIDTH_INDEX] << 22;
          }
          getWidth(e) {
            return this._data[3 * e + 0] >> 22;
          }
          hasWidth(e) {
            return 12582912 & this._data[3 * e + 0];
          }
          getFg(e) {
            return this._data[3 * e + 1];
          }
          getBg(e) {
            return this._data[3 * e + 2];
          }
          hasContent(e) {
            return 4194303 & this._data[3 * e + 0];
          }
          getCodePoint(e) {
            const o = this._data[3 * e + 0];
            return 2097152 & o ? this._combined[e].charCodeAt(this._combined[e].length - 1) : 2097151 & o;
          }
          isCombined(e) {
            return 2097152 & this._data[3 * e + 0];
          }
          getString(e) {
            const o = this._data[3 * e + 0];
            return 2097152 & o ? this._combined[e] : 2097151 & o ? (0, d.stringFromCodePoint)(2097151 & o) : "";
          }
          isProtected(e) {
            return 536870912 & this._data[3 * e + 2];
          }
          loadCell(e, o) {
            return f = 3 * e, o.content = this._data[f + 0], o.fg = this._data[f + 1], o.bg = this._data[f + 2], 2097152 & o.content && (o.combinedData = this._combined[e]), 268435456 & o.bg && (o.extended = this._extendedAttrs[e]), o;
          }
          setCell(e, o) {
            2097152 & o.content && (this._combined[e] = o.combinedData), 268435456 & o.bg && (this._extendedAttrs[e] = o.extended), this._data[3 * e + 0] = o.content, this._data[3 * e + 1] = o.fg, this._data[3 * e + 2] = o.bg;
          }
          setCellFromCodepoint(e, o, s, i) {
            268435456 & i.bg && (this._extendedAttrs[e] = i.extended), this._data[3 * e + 0] = o | s << 22, this._data[3 * e + 1] = i.fg, this._data[3 * e + 2] = i.bg;
          }
          addCodepointToCell(e, o, s) {
            let i = this._data[3 * e + 0];
            2097152 & i ? this._combined[e] += (0, d.stringFromCodePoint)(o) : 2097151 & i ? (this._combined[e] = (0, d.stringFromCodePoint)(2097151 & i) + (0, d.stringFromCodePoint)(o), i &= -2097152, i |= 2097152) : i = o | 4194304, s && (i &= -12582913, i |= s << 22), this._data[3 * e + 0] = i;
          }
          insertCells(e, o, s) {
            if ((e %= this.length) && this.getWidth(e - 1) === 2 && this.setCellFromCodepoint(e - 1, 0, 1, s), o < this.length - e) {
              const i = new h.CellData();
              for (let u = this.length - e - o - 1; u >= 0; --u) this.setCell(e + o + u, this.loadCell(e + u, i));
              for (let u = 0; u < o; ++u) this.setCell(e + u, s);
            } else for (let i = e; i < this.length; ++i) this.setCell(i, s);
            this.getWidth(this.length - 1) === 2 && this.setCellFromCodepoint(this.length - 1, 0, 1, s);
          }
          deleteCells(e, o, s) {
            if (e %= this.length, o < this.length - e) {
              const i = new h.CellData();
              for (let u = 0; u < this.length - e - o; ++u) this.setCell(e + u, this.loadCell(e + o + u, i));
              for (let u = this.length - o; u < this.length; ++u) this.setCell(u, s);
            } else for (let i = e; i < this.length; ++i) this.setCell(i, s);
            e && this.getWidth(e - 1) === 2 && this.setCellFromCodepoint(e - 1, 0, 1, s), this.getWidth(e) !== 0 || this.hasContent(e) || this.setCellFromCodepoint(e, 0, 1, s);
          }
          replaceCells(e, o, s, i = !1) {
            if (i) for (e && this.getWidth(e - 1) === 2 && !this.isProtected(e - 1) && this.setCellFromCodepoint(e - 1, 0, 1, s), o < this.length && this.getWidth(o - 1) === 2 && !this.isProtected(o) && this.setCellFromCodepoint(o, 0, 1, s); e < o && e < this.length; ) this.isProtected(e) || this.setCell(e, s), e++;
            else for (e && this.getWidth(e - 1) === 2 && this.setCellFromCodepoint(e - 1, 0, 1, s), o < this.length && this.getWidth(o - 1) === 2 && this.setCellFromCodepoint(o, 0, 1, s); e < o && e < this.length; ) this.setCell(e++, s);
          }
          resize(e, o) {
            if (e === this.length) return 4 * this._data.length * 2 < this._data.buffer.byteLength;
            const s = 3 * e;
            if (e > this.length) {
              if (this._data.buffer.byteLength >= 4 * s) this._data = new Uint32Array(this._data.buffer, 0, s);
              else {
                const i = new Uint32Array(s);
                i.set(this._data), this._data = i;
              }
              for (let i = this.length; i < e; ++i) this.setCell(i, o);
            } else {
              this._data = this._data.subarray(0, s);
              const i = Object.keys(this._combined);
              for (let p = 0; p < i.length; p++) {
                const l = parseInt(i[p], 10);
                l >= e && delete this._combined[l];
              }
              const u = Object.keys(this._extendedAttrs);
              for (let p = 0; p < u.length; p++) {
                const l = parseInt(u[p], 10);
                l >= e && delete this._extendedAttrs[l];
              }
            }
            return this.length = e, 4 * s * 2 < this._data.buffer.byteLength;
          }
          cleanupMemory() {
            if (4 * this._data.length * 2 < this._data.buffer.byteLength) {
              const e = new Uint32Array(this._data.length);
              return e.set(this._data), this._data = e, 1;
            }
            return 0;
          }
          fill(e, o = !1) {
            if (o) for (let s = 0; s < this.length; ++s) this.isProtected(s) || this.setCell(s, e);
            else {
              this._combined = {}, this._extendedAttrs = {};
              for (let s = 0; s < this.length; ++s) this.setCell(s, e);
            }
          }
          copyFrom(e) {
            this.length !== e.length ? this._data = new Uint32Array(e._data) : this._data.set(e._data), this.length = e.length, this._combined = {};
            for (const o in e._combined) this._combined[o] = e._combined[o];
            this._extendedAttrs = {};
            for (const o in e._extendedAttrs) this._extendedAttrs[o] = e._extendedAttrs[o];
            this.isWrapped = e.isWrapped;
          }
          clone() {
            const e = new g(0);
            e._data = new Uint32Array(this._data), e.length = this.length;
            for (const o in this._combined) e._combined[o] = this._combined[o];
            for (const o in this._extendedAttrs) e._extendedAttrs[o] = this._extendedAttrs[o];
            return e.isWrapped = this.isWrapped, e;
          }
          getTrimmedLength() {
            for (let e = this.length - 1; e >= 0; --e) if (4194303 & this._data[3 * e + 0]) return e + (this._data[3 * e + 0] >> 22);
            return 0;
          }
          getNoBgTrimmedLength() {
            for (let e = this.length - 1; e >= 0; --e) if (4194303 & this._data[3 * e + 0] || 50331648 & this._data[3 * e + 2]) return e + (this._data[3 * e + 0] >> 22);
            return 0;
          }
          copyCellsFrom(e, o, s, i, u) {
            const p = e._data;
            if (u) for (let m = i - 1; m >= 0; m--) {
              for (let _ = 0; _ < 3; _++) this._data[3 * (s + m) + _] = p[3 * (o + m) + _];
              268435456 & p[3 * (o + m) + 2] && (this._extendedAttrs[s + m] = e._extendedAttrs[o + m]);
            }
            else for (let m = 0; m < i; m++) {
              for (let _ = 0; _ < 3; _++) this._data[3 * (s + m) + _] = p[3 * (o + m) + _];
              268435456 & p[3 * (o + m) + 2] && (this._extendedAttrs[s + m] = e._extendedAttrs[o + m]);
            }
            const l = Object.keys(e._combined);
            for (let m = 0; m < l.length; m++) {
              const _ = parseInt(l[m], 10);
              _ >= o && (this._combined[_ - o + s] = e._combined[_]);
            }
          }
          translateToString(e, o, s, i) {
            o = o != null ? o : 0, s = s != null ? s : this.length, e && (s = Math.min(s, this.getTrimmedLength())), i && (i.length = 0);
            let u = "";
            for (; o < s; ) {
              const p = this._data[3 * o + 0], l = 2097151 & p, m = 2097152 & p ? this._combined[o] : l ? (0, d.stringFromCodePoint)(l) : r.WHITESPACE_CELL_CHAR;
              if (u += m, i) for (let _ = 0; _ < m.length; ++_) i.push(o);
              o += p >> 22 || 1;
            }
            return i && i.push(o), u;
          }
        }
        t.BufferLine = g;
      }, 4841: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.getRangeLength = void 0, t.getRangeLength = function(a, c) {
          if (a.start.y > a.end.y) throw new Error(`Buffer range end (${a.end.x}, ${a.end.y}) cannot be before start (${a.start.x}, ${a.start.y})`);
          return c * (a.end.y - a.start.y) + (a.end.x - a.start.x + 1);
        };
      }, 4634: (T, t) => {
        function a(c, h, r) {
          if (h === c.length - 1) return c[h].getTrimmedLength();
          const d = !c[h].hasContent(r - 1) && c[h].getWidth(r - 1) === 1, f = c[h + 1].getWidth(0) === 2;
          return d && f ? r - 1 : r;
        }
        Object.defineProperty(t, "__esModule", { value: !0 }), t.getWrappedLineTrimmedLength = t.reflowSmallerGetNewLineLengths = t.reflowLargerApplyNewLayout = t.reflowLargerCreateNewLayout = t.reflowLargerGetLinesToRemove = void 0, t.reflowLargerGetLinesToRemove = function(c, h, r, d, f) {
          const g = [];
          for (let n = 0; n < c.length - 1; n++) {
            let e = n, o = c.get(++e);
            if (!o.isWrapped) continue;
            const s = [c.get(n)];
            for (; e < c.length && o.isWrapped; ) s.push(o), o = c.get(++e);
            if (d >= n && d < e) {
              n += s.length - 1;
              continue;
            }
            let i = 0, u = a(s, i, h), p = 1, l = 0;
            for (; p < s.length; ) {
              const _ = a(s, p, h), v = _ - l, C = r - u, w = Math.min(v, C);
              s[i].copyCellsFrom(s[p], l, u, w, !1), u += w, u === r && (i++, u = 0), l += w, l === _ && (p++, l = 0), u === 0 && i !== 0 && s[i - 1].getWidth(r - 1) === 2 && (s[i].copyCellsFrom(s[i - 1], r - 1, u++, 1, !1), s[i - 1].setCell(r - 1, f));
            }
            s[i].replaceCells(u, r, f);
            let m = 0;
            for (let _ = s.length - 1; _ > 0 && (_ > i || s[_].getTrimmedLength() === 0); _--) m++;
            m > 0 && (g.push(n + s.length - m), g.push(m)), n += s.length - 1;
          }
          return g;
        }, t.reflowLargerCreateNewLayout = function(c, h) {
          const r = [];
          let d = 0, f = h[d], g = 0;
          for (let n = 0; n < c.length; n++) if (f === n) {
            const e = h[++d];
            c.onDeleteEmitter.fire({ index: n - g, amount: e }), n += e - 1, g += e, f = h[++d];
          } else r.push(n);
          return { layout: r, countRemoved: g };
        }, t.reflowLargerApplyNewLayout = function(c, h) {
          const r = [];
          for (let d = 0; d < h.length; d++) r.push(c.get(h[d]));
          for (let d = 0; d < r.length; d++) c.set(d, r[d]);
          c.length = h.length;
        }, t.reflowSmallerGetNewLineLengths = function(c, h, r) {
          const d = [], f = c.map(((o, s) => a(c, s, h))).reduce(((o, s) => o + s));
          let g = 0, n = 0, e = 0;
          for (; e < f; ) {
            if (f - e < r) {
              d.push(f - e);
              break;
            }
            g += r;
            const o = a(c, n, h);
            g > o && (g -= o, n++);
            const s = c[n].getWidth(g - 1) === 2;
            s && g--;
            const i = s ? r - 1 : r;
            d.push(i), e += i;
          }
          return d;
        }, t.getWrappedLineTrimmedLength = a;
      }, 5295: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.BufferSet = void 0;
        const c = a(8460), h = a(844), r = a(9092);
        class d extends h.Disposable {
          constructor(g, n) {
            super(), this._optionsService = g, this._bufferService = n, this._onBufferActivate = this.register(new c.EventEmitter()), this.onBufferActivate = this._onBufferActivate.event, this.reset(), this.register(this._optionsService.onSpecificOptionChange("scrollback", (() => this.resize(this._bufferService.cols, this._bufferService.rows)))), this.register(this._optionsService.onSpecificOptionChange("tabStopWidth", (() => this.setupTabStops())));
          }
          reset() {
            this._normal = new r.Buffer(!0, this._optionsService, this._bufferService), this._normal.fillViewportRows(), this._alt = new r.Buffer(!1, this._optionsService, this._bufferService), this._activeBuffer = this._normal, this._onBufferActivate.fire({ activeBuffer: this._normal, inactiveBuffer: this._alt }), this.setupTabStops();
          }
          get alt() {
            return this._alt;
          }
          get active() {
            return this._activeBuffer;
          }
          get normal() {
            return this._normal;
          }
          activateNormalBuffer() {
            this._activeBuffer !== this._normal && (this._normal.x = this._alt.x, this._normal.y = this._alt.y, this._alt.clearAllMarkers(), this._alt.clear(), this._activeBuffer = this._normal, this._onBufferActivate.fire({ activeBuffer: this._normal, inactiveBuffer: this._alt }));
          }
          activateAltBuffer(g) {
            this._activeBuffer !== this._alt && (this._alt.fillViewportRows(g), this._alt.x = this._normal.x, this._alt.y = this._normal.y, this._activeBuffer = this._alt, this._onBufferActivate.fire({ activeBuffer: this._alt, inactiveBuffer: this._normal }));
          }
          resize(g, n) {
            this._normal.resize(g, n), this._alt.resize(g, n), this.setupTabStops(g);
          }
          setupTabStops(g) {
            this._normal.setupTabStops(g), this._alt.setupTabStops(g);
          }
        }
        t.BufferSet = d;
      }, 511: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CellData = void 0;
        const c = a(482), h = a(643), r = a(3734);
        class d extends r.AttributeData {
          constructor() {
            super(...arguments), this.content = 0, this.fg = 0, this.bg = 0, this.extended = new r.ExtendedAttrs(), this.combinedData = "";
          }
          static fromCharData(g) {
            const n = new d();
            return n.setFromCharData(g), n;
          }
          isCombined() {
            return 2097152 & this.content;
          }
          getWidth() {
            return this.content >> 22;
          }
          getChars() {
            return 2097152 & this.content ? this.combinedData : 2097151 & this.content ? (0, c.stringFromCodePoint)(2097151 & this.content) : "";
          }
          getCode() {
            return this.isCombined() ? this.combinedData.charCodeAt(this.combinedData.length - 1) : 2097151 & this.content;
          }
          setFromCharData(g) {
            this.fg = g[h.CHAR_DATA_ATTR_INDEX], this.bg = 0;
            let n = !1;
            if (g[h.CHAR_DATA_CHAR_INDEX].length > 2) n = !0;
            else if (g[h.CHAR_DATA_CHAR_INDEX].length === 2) {
              const e = g[h.CHAR_DATA_CHAR_INDEX].charCodeAt(0);
              if (55296 <= e && e <= 56319) {
                const o = g[h.CHAR_DATA_CHAR_INDEX].charCodeAt(1);
                56320 <= o && o <= 57343 ? this.content = 1024 * (e - 55296) + o - 56320 + 65536 | g[h.CHAR_DATA_WIDTH_INDEX] << 22 : n = !0;
              } else n = !0;
            } else this.content = g[h.CHAR_DATA_CHAR_INDEX].charCodeAt(0) | g[h.CHAR_DATA_WIDTH_INDEX] << 22;
            n && (this.combinedData = g[h.CHAR_DATA_CHAR_INDEX], this.content = 2097152 | g[h.CHAR_DATA_WIDTH_INDEX] << 22);
          }
          getAsCharData() {
            return [this.fg, this.getChars(), this.getWidth(), this.getCode()];
          }
        }
        t.CellData = d;
      }, 643: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.WHITESPACE_CELL_CODE = t.WHITESPACE_CELL_WIDTH = t.WHITESPACE_CELL_CHAR = t.NULL_CELL_CODE = t.NULL_CELL_WIDTH = t.NULL_CELL_CHAR = t.CHAR_DATA_CODE_INDEX = t.CHAR_DATA_WIDTH_INDEX = t.CHAR_DATA_CHAR_INDEX = t.CHAR_DATA_ATTR_INDEX = t.DEFAULT_EXT = t.DEFAULT_ATTR = t.DEFAULT_COLOR = void 0, t.DEFAULT_COLOR = 0, t.DEFAULT_ATTR = 256 | t.DEFAULT_COLOR << 9, t.DEFAULT_EXT = 0, t.CHAR_DATA_ATTR_INDEX = 0, t.CHAR_DATA_CHAR_INDEX = 1, t.CHAR_DATA_WIDTH_INDEX = 2, t.CHAR_DATA_CODE_INDEX = 3, t.NULL_CELL_CHAR = "", t.NULL_CELL_WIDTH = 1, t.NULL_CELL_CODE = 0, t.WHITESPACE_CELL_CHAR = " ", t.WHITESPACE_CELL_WIDTH = 1, t.WHITESPACE_CELL_CODE = 32;
      }, 4863: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.Marker = void 0;
        const c = a(8460), h = a(844);
        class r {
          get id() {
            return this._id;
          }
          constructor(f) {
            this.line = f, this.isDisposed = !1, this._disposables = [], this._id = r._nextId++, this._onDispose = this.register(new c.EventEmitter()), this.onDispose = this._onDispose.event;
          }
          dispose() {
            this.isDisposed || (this.isDisposed = !0, this.line = -1, this._onDispose.fire(), (0, h.disposeArray)(this._disposables), this._disposables.length = 0);
          }
          register(f) {
            return this._disposables.push(f), f;
          }
        }
        t.Marker = r, r._nextId = 1;
      }, 7116: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.DEFAULT_CHARSET = t.CHARSETS = void 0, t.CHARSETS = {}, t.DEFAULT_CHARSET = t.CHARSETS.B, t.CHARSETS[0] = { "`": "◆", a: "▒", b: "␉", c: "␌", d: "␍", e: "␊", f: "°", g: "±", h: "␤", i: "␋", j: "┘", k: "┐", l: "┌", m: "└", n: "┼", o: "⎺", p: "⎻", q: "─", r: "⎼", s: "⎽", t: "├", u: "┤", v: "┴", w: "┬", x: "│", y: "≤", z: "≥", "{": "π", "|": "≠", "}": "£", "~": "·" }, t.CHARSETS.A = { "#": "£" }, t.CHARSETS.B = void 0, t.CHARSETS[4] = { "#": "£", "@": "¾", "[": "ij", "\\": "½", "]": "|", "{": "¨", "|": "f", "}": "¼", "~": "´" }, t.CHARSETS.C = t.CHARSETS[5] = { "[": "Ä", "\\": "Ö", "]": "Å", "^": "Ü", "`": "é", "{": "ä", "|": "ö", "}": "å", "~": "ü" }, t.CHARSETS.R = { "#": "£", "@": "à", "[": "°", "\\": "ç", "]": "§", "{": "é", "|": "ù", "}": "è", "~": "¨" }, t.CHARSETS.Q = { "@": "à", "[": "â", "\\": "ç", "]": "ê", "^": "î", "`": "ô", "{": "é", "|": "ù", "}": "è", "~": "û" }, t.CHARSETS.K = { "@": "§", "[": "Ä", "\\": "Ö", "]": "Ü", "{": "ä", "|": "ö", "}": "ü", "~": "ß" }, t.CHARSETS.Y = { "#": "£", "@": "§", "[": "°", "\\": "ç", "]": "é", "`": "ù", "{": "à", "|": "ò", "}": "è", "~": "ì" }, t.CHARSETS.E = t.CHARSETS[6] = { "@": "Ä", "[": "Æ", "\\": "Ø", "]": "Å", "^": "Ü", "`": "ä", "{": "æ", "|": "ø", "}": "å", "~": "ü" }, t.CHARSETS.Z = { "#": "£", "@": "§", "[": "¡", "\\": "Ñ", "]": "¿", "{": "°", "|": "ñ", "}": "ç" }, t.CHARSETS.H = t.CHARSETS[7] = { "@": "É", "[": "Ä", "\\": "Ö", "]": "Å", "^": "Ü", "`": "é", "{": "ä", "|": "ö", "}": "å", "~": "ü" }, t.CHARSETS["="] = { "#": "ù", "@": "à", "[": "é", "\\": "ç", "]": "ê", "^": "î", _: "è", "`": "ô", "{": "ä", "|": "ö", "}": "ü", "~": "û" };
      }, 2584: (T, t) => {
        var a, c, h;
        Object.defineProperty(t, "__esModule", { value: !0 }), t.C1_ESCAPED = t.C1 = t.C0 = void 0, (function(r) {
          r.NUL = "\0", r.SOH = "", r.STX = "", r.ETX = "", r.EOT = "", r.ENQ = "", r.ACK = "", r.BEL = "\x07", r.BS = "\b", r.HT = "	", r.LF = `
`, r.VT = "\v", r.FF = "\f", r.CR = "\r", r.SO = "", r.SI = "", r.DLE = "", r.DC1 = "", r.DC2 = "", r.DC3 = "", r.DC4 = "", r.NAK = "", r.SYN = "", r.ETB = "", r.CAN = "", r.EM = "", r.SUB = "", r.ESC = "\x1B", r.FS = "", r.GS = "", r.RS = "", r.US = "", r.SP = " ", r.DEL = "";
        })(a || (t.C0 = a = {})), (function(r) {
          r.PAD = "", r.HOP = "", r.BPH = "", r.NBH = "", r.IND = "", r.NEL = "", r.SSA = "", r.ESA = "", r.HTS = "", r.HTJ = "", r.VTS = "", r.PLD = "", r.PLU = "", r.RI = "", r.SS2 = "", r.SS3 = "", r.DCS = "", r.PU1 = "", r.PU2 = "", r.STS = "", r.CCH = "", r.MW = "", r.SPA = "", r.EPA = "", r.SOS = "", r.SGCI = "", r.SCI = "", r.CSI = "", r.ST = "", r.OSC = "", r.PM = "", r.APC = "";
        })(c || (t.C1 = c = {})), (function(r) {
          r.ST = `${a.ESC}\\`;
        })(h || (t.C1_ESCAPED = h = {}));
      }, 7399: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.evaluateKeyboardEvent = void 0;
        const c = a(2584), h = { 48: ["0", ")"], 49: ["1", "!"], 50: ["2", "@"], 51: ["3", "#"], 52: ["4", "$"], 53: ["5", "%"], 54: ["6", "^"], 55: ["7", "&"], 56: ["8", "*"], 57: ["9", "("], 186: [";", ":"], 187: ["=", "+"], 188: [",", "<"], 189: ["-", "_"], 190: [".", ">"], 191: ["/", "?"], 192: ["`", "~"], 219: ["[", "{"], 220: ["\\", "|"], 221: ["]", "}"], 222: ["'", '"'] };
        t.evaluateKeyboardEvent = function(r, d, f, g) {
          const n = { type: 0, cancel: !1, key: void 0 }, e = (r.shiftKey ? 1 : 0) | (r.altKey ? 2 : 0) | (r.ctrlKey ? 4 : 0) | (r.metaKey ? 8 : 0);
          switch (r.keyCode) {
            case 0:
              r.key === "UIKeyInputUpArrow" ? n.key = d ? c.C0.ESC + "OA" : c.C0.ESC + "[A" : r.key === "UIKeyInputLeftArrow" ? n.key = d ? c.C0.ESC + "OD" : c.C0.ESC + "[D" : r.key === "UIKeyInputRightArrow" ? n.key = d ? c.C0.ESC + "OC" : c.C0.ESC + "[C" : r.key === "UIKeyInputDownArrow" && (n.key = d ? c.C0.ESC + "OB" : c.C0.ESC + "[B");
              break;
            case 8:
              n.key = r.ctrlKey ? "\b" : c.C0.DEL, r.altKey && (n.key = c.C0.ESC + n.key);
              break;
            case 9:
              if (r.shiftKey) {
                n.key = c.C0.ESC + "[Z";
                break;
              }
              n.key = c.C0.HT, n.cancel = !0;
              break;
            case 13:
              n.key = r.altKey ? c.C0.ESC + c.C0.CR : c.C0.CR, n.cancel = !0;
              break;
            case 27:
              n.key = c.C0.ESC, r.altKey && (n.key = c.C0.ESC + c.C0.ESC), n.cancel = !0;
              break;
            case 37:
              if (r.metaKey) break;
              e ? (n.key = c.C0.ESC + "[1;" + (e + 1) + "D", n.key === c.C0.ESC + "[1;3D" && (n.key = c.C0.ESC + (f ? "b" : "[1;5D"))) : n.key = d ? c.C0.ESC + "OD" : c.C0.ESC + "[D";
              break;
            case 39:
              if (r.metaKey) break;
              e ? (n.key = c.C0.ESC + "[1;" + (e + 1) + "C", n.key === c.C0.ESC + "[1;3C" && (n.key = c.C0.ESC + (f ? "f" : "[1;5C"))) : n.key = d ? c.C0.ESC + "OC" : c.C0.ESC + "[C";
              break;
            case 38:
              if (r.metaKey) break;
              e ? (n.key = c.C0.ESC + "[1;" + (e + 1) + "A", f || n.key !== c.C0.ESC + "[1;3A" || (n.key = c.C0.ESC + "[1;5A")) : n.key = d ? c.C0.ESC + "OA" : c.C0.ESC + "[A";
              break;
            case 40:
              if (r.metaKey) break;
              e ? (n.key = c.C0.ESC + "[1;" + (e + 1) + "B", f || n.key !== c.C0.ESC + "[1;3B" || (n.key = c.C0.ESC + "[1;5B")) : n.key = d ? c.C0.ESC + "OB" : c.C0.ESC + "[B";
              break;
            case 45:
              r.shiftKey || r.ctrlKey || (n.key = c.C0.ESC + "[2~");
              break;
            case 46:
              n.key = e ? c.C0.ESC + "[3;" + (e + 1) + "~" : c.C0.ESC + "[3~";
              break;
            case 36:
              n.key = e ? c.C0.ESC + "[1;" + (e + 1) + "H" : d ? c.C0.ESC + "OH" : c.C0.ESC + "[H";
              break;
            case 35:
              n.key = e ? c.C0.ESC + "[1;" + (e + 1) + "F" : d ? c.C0.ESC + "OF" : c.C0.ESC + "[F";
              break;
            case 33:
              r.shiftKey ? n.type = 2 : r.ctrlKey ? n.key = c.C0.ESC + "[5;" + (e + 1) + "~" : n.key = c.C0.ESC + "[5~";
              break;
            case 34:
              r.shiftKey ? n.type = 3 : r.ctrlKey ? n.key = c.C0.ESC + "[6;" + (e + 1) + "~" : n.key = c.C0.ESC + "[6~";
              break;
            case 112:
              n.key = e ? c.C0.ESC + "[1;" + (e + 1) + "P" : c.C0.ESC + "OP";
              break;
            case 113:
              n.key = e ? c.C0.ESC + "[1;" + (e + 1) + "Q" : c.C0.ESC + "OQ";
              break;
            case 114:
              n.key = e ? c.C0.ESC + "[1;" + (e + 1) + "R" : c.C0.ESC + "OR";
              break;
            case 115:
              n.key = e ? c.C0.ESC + "[1;" + (e + 1) + "S" : c.C0.ESC + "OS";
              break;
            case 116:
              n.key = e ? c.C0.ESC + "[15;" + (e + 1) + "~" : c.C0.ESC + "[15~";
              break;
            case 117:
              n.key = e ? c.C0.ESC + "[17;" + (e + 1) + "~" : c.C0.ESC + "[17~";
              break;
            case 118:
              n.key = e ? c.C0.ESC + "[18;" + (e + 1) + "~" : c.C0.ESC + "[18~";
              break;
            case 119:
              n.key = e ? c.C0.ESC + "[19;" + (e + 1) + "~" : c.C0.ESC + "[19~";
              break;
            case 120:
              n.key = e ? c.C0.ESC + "[20;" + (e + 1) + "~" : c.C0.ESC + "[20~";
              break;
            case 121:
              n.key = e ? c.C0.ESC + "[21;" + (e + 1) + "~" : c.C0.ESC + "[21~";
              break;
            case 122:
              n.key = e ? c.C0.ESC + "[23;" + (e + 1) + "~" : c.C0.ESC + "[23~";
              break;
            case 123:
              n.key = e ? c.C0.ESC + "[24;" + (e + 1) + "~" : c.C0.ESC + "[24~";
              break;
            default:
              if (!r.ctrlKey || r.shiftKey || r.altKey || r.metaKey) if (f && !g || !r.altKey || r.metaKey) !f || r.altKey || r.ctrlKey || r.shiftKey || !r.metaKey ? r.key && !r.ctrlKey && !r.altKey && !r.metaKey && r.keyCode >= 48 && r.key.length === 1 ? n.key = r.key : r.key && r.ctrlKey && (r.key === "_" && (n.key = c.C0.US), r.key === "@" && (n.key = c.C0.NUL)) : r.keyCode === 65 && (n.type = 1);
              else {
                const o = h[r.keyCode], s = o == null ? void 0 : o[r.shiftKey ? 1 : 0];
                if (s) n.key = c.C0.ESC + s;
                else if (r.keyCode >= 65 && r.keyCode <= 90) {
                  const i = r.ctrlKey ? r.keyCode - 64 : r.keyCode + 32;
                  let u = String.fromCharCode(i);
                  r.shiftKey && (u = u.toUpperCase()), n.key = c.C0.ESC + u;
                } else if (r.keyCode === 32) n.key = c.C0.ESC + (r.ctrlKey ? c.C0.NUL : " ");
                else if (r.key === "Dead" && r.code.startsWith("Key")) {
                  let i = r.code.slice(3, 4);
                  r.shiftKey || (i = i.toLowerCase()), n.key = c.C0.ESC + i, n.cancel = !0;
                }
              }
              else r.keyCode >= 65 && r.keyCode <= 90 ? n.key = String.fromCharCode(r.keyCode - 64) : r.keyCode === 32 ? n.key = c.C0.NUL : r.keyCode >= 51 && r.keyCode <= 55 ? n.key = String.fromCharCode(r.keyCode - 51 + 27) : r.keyCode === 56 ? n.key = c.C0.DEL : r.keyCode === 219 ? n.key = c.C0.ESC : r.keyCode === 220 ? n.key = c.C0.FS : r.keyCode === 221 && (n.key = c.C0.GS);
          }
          return n;
        };
      }, 482: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.Utf8ToUtf32 = t.StringToUtf32 = t.utf32ToString = t.stringFromCodePoint = void 0, t.stringFromCodePoint = function(a) {
          return a > 65535 ? (a -= 65536, String.fromCharCode(55296 + (a >> 10)) + String.fromCharCode(a % 1024 + 56320)) : String.fromCharCode(a);
        }, t.utf32ToString = function(a, c = 0, h = a.length) {
          let r = "";
          for (let d = c; d < h; ++d) {
            let f = a[d];
            f > 65535 ? (f -= 65536, r += String.fromCharCode(55296 + (f >> 10)) + String.fromCharCode(f % 1024 + 56320)) : r += String.fromCharCode(f);
          }
          return r;
        }, t.StringToUtf32 = class {
          constructor() {
            this._interim = 0;
          }
          clear() {
            this._interim = 0;
          }
          decode(a, c) {
            const h = a.length;
            if (!h) return 0;
            let r = 0, d = 0;
            if (this._interim) {
              const f = a.charCodeAt(d++);
              56320 <= f && f <= 57343 ? c[r++] = 1024 * (this._interim - 55296) + f - 56320 + 65536 : (c[r++] = this._interim, c[r++] = f), this._interim = 0;
            }
            for (let f = d; f < h; ++f) {
              const g = a.charCodeAt(f);
              if (55296 <= g && g <= 56319) {
                if (++f >= h) return this._interim = g, r;
                const n = a.charCodeAt(f);
                56320 <= n && n <= 57343 ? c[r++] = 1024 * (g - 55296) + n - 56320 + 65536 : (c[r++] = g, c[r++] = n);
              } else g !== 65279 && (c[r++] = g);
            }
            return r;
          }
        }, t.Utf8ToUtf32 = class {
          constructor() {
            this.interim = new Uint8Array(3);
          }
          clear() {
            this.interim.fill(0);
          }
          decode(a, c) {
            const h = a.length;
            if (!h) return 0;
            let r, d, f, g, n = 0, e = 0, o = 0;
            if (this.interim[0]) {
              let u = !1, p = this.interim[0];
              p &= (224 & p) == 192 ? 31 : (240 & p) == 224 ? 15 : 7;
              let l, m = 0;
              for (; (l = 63 & this.interim[++m]) && m < 4; ) p <<= 6, p |= l;
              const _ = (224 & this.interim[0]) == 192 ? 2 : (240 & this.interim[0]) == 224 ? 3 : 4, v = _ - m;
              for (; o < v; ) {
                if (o >= h) return 0;
                if (l = a[o++], (192 & l) != 128) {
                  o--, u = !0;
                  break;
                }
                this.interim[m++] = l, p <<= 6, p |= 63 & l;
              }
              u || (_ === 2 ? p < 128 ? o-- : c[n++] = p : _ === 3 ? p < 2048 || p >= 55296 && p <= 57343 || p === 65279 || (c[n++] = p) : p < 65536 || p > 1114111 || (c[n++] = p)), this.interim.fill(0);
            }
            const s = h - 4;
            let i = o;
            for (; i < h; ) {
              for (; !(!(i < s) || 128 & (r = a[i]) || 128 & (d = a[i + 1]) || 128 & (f = a[i + 2]) || 128 & (g = a[i + 3])); ) c[n++] = r, c[n++] = d, c[n++] = f, c[n++] = g, i += 4;
              if (r = a[i++], r < 128) c[n++] = r;
              else if ((224 & r) == 192) {
                if (i >= h) return this.interim[0] = r, n;
                if (d = a[i++], (192 & d) != 128) {
                  i--;
                  continue;
                }
                if (e = (31 & r) << 6 | 63 & d, e < 128) {
                  i--;
                  continue;
                }
                c[n++] = e;
              } else if ((240 & r) == 224) {
                if (i >= h) return this.interim[0] = r, n;
                if (d = a[i++], (192 & d) != 128) {
                  i--;
                  continue;
                }
                if (i >= h) return this.interim[0] = r, this.interim[1] = d, n;
                if (f = a[i++], (192 & f) != 128) {
                  i--;
                  continue;
                }
                if (e = (15 & r) << 12 | (63 & d) << 6 | 63 & f, e < 2048 || e >= 55296 && e <= 57343 || e === 65279) continue;
                c[n++] = e;
              } else if ((248 & r) == 240) {
                if (i >= h) return this.interim[0] = r, n;
                if (d = a[i++], (192 & d) != 128) {
                  i--;
                  continue;
                }
                if (i >= h) return this.interim[0] = r, this.interim[1] = d, n;
                if (f = a[i++], (192 & f) != 128) {
                  i--;
                  continue;
                }
                if (i >= h) return this.interim[0] = r, this.interim[1] = d, this.interim[2] = f, n;
                if (g = a[i++], (192 & g) != 128) {
                  i--;
                  continue;
                }
                if (e = (7 & r) << 18 | (63 & d) << 12 | (63 & f) << 6 | 63 & g, e < 65536 || e > 1114111) continue;
                c[n++] = e;
              }
            }
            return n;
          }
        };
      }, 225: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.UnicodeV6 = void 0;
        const c = a(1480), h = [[768, 879], [1155, 1158], [1160, 1161], [1425, 1469], [1471, 1471], [1473, 1474], [1476, 1477], [1479, 1479], [1536, 1539], [1552, 1557], [1611, 1630], [1648, 1648], [1750, 1764], [1767, 1768], [1770, 1773], [1807, 1807], [1809, 1809], [1840, 1866], [1958, 1968], [2027, 2035], [2305, 2306], [2364, 2364], [2369, 2376], [2381, 2381], [2385, 2388], [2402, 2403], [2433, 2433], [2492, 2492], [2497, 2500], [2509, 2509], [2530, 2531], [2561, 2562], [2620, 2620], [2625, 2626], [2631, 2632], [2635, 2637], [2672, 2673], [2689, 2690], [2748, 2748], [2753, 2757], [2759, 2760], [2765, 2765], [2786, 2787], [2817, 2817], [2876, 2876], [2879, 2879], [2881, 2883], [2893, 2893], [2902, 2902], [2946, 2946], [3008, 3008], [3021, 3021], [3134, 3136], [3142, 3144], [3146, 3149], [3157, 3158], [3260, 3260], [3263, 3263], [3270, 3270], [3276, 3277], [3298, 3299], [3393, 3395], [3405, 3405], [3530, 3530], [3538, 3540], [3542, 3542], [3633, 3633], [3636, 3642], [3655, 3662], [3761, 3761], [3764, 3769], [3771, 3772], [3784, 3789], [3864, 3865], [3893, 3893], [3895, 3895], [3897, 3897], [3953, 3966], [3968, 3972], [3974, 3975], [3984, 3991], [3993, 4028], [4038, 4038], [4141, 4144], [4146, 4146], [4150, 4151], [4153, 4153], [4184, 4185], [4448, 4607], [4959, 4959], [5906, 5908], [5938, 5940], [5970, 5971], [6002, 6003], [6068, 6069], [6071, 6077], [6086, 6086], [6089, 6099], [6109, 6109], [6155, 6157], [6313, 6313], [6432, 6434], [6439, 6440], [6450, 6450], [6457, 6459], [6679, 6680], [6912, 6915], [6964, 6964], [6966, 6970], [6972, 6972], [6978, 6978], [7019, 7027], [7616, 7626], [7678, 7679], [8203, 8207], [8234, 8238], [8288, 8291], [8298, 8303], [8400, 8431], [12330, 12335], [12441, 12442], [43014, 43014], [43019, 43019], [43045, 43046], [64286, 64286], [65024, 65039], [65056, 65059], [65279, 65279], [65529, 65531]], r = [[68097, 68099], [68101, 68102], [68108, 68111], [68152, 68154], [68159, 68159], [119143, 119145], [119155, 119170], [119173, 119179], [119210, 119213], [119362, 119364], [917505, 917505], [917536, 917631], [917760, 917999]];
        let d;
        t.UnicodeV6 = class {
          constructor() {
            if (this.version = "6", !d) {
              d = new Uint8Array(65536), d.fill(1), d[0] = 0, d.fill(0, 1, 32), d.fill(0, 127, 160), d.fill(2, 4352, 4448), d[9001] = 2, d[9002] = 2, d.fill(2, 11904, 42192), d[12351] = 1, d.fill(2, 44032, 55204), d.fill(2, 63744, 64256), d.fill(2, 65040, 65050), d.fill(2, 65072, 65136), d.fill(2, 65280, 65377), d.fill(2, 65504, 65511);
              for (let f = 0; f < h.length; ++f) d.fill(0, h[f][0], h[f][1] + 1);
            }
          }
          wcwidth(f) {
            return f < 32 ? 0 : f < 127 ? 1 : f < 65536 ? d[f] : (function(g, n) {
              let e, o = 0, s = n.length - 1;
              if (g < n[0][0] || g > n[s][1]) return !1;
              for (; s >= o; ) if (e = o + s >> 1, g > n[e][1]) o = e + 1;
              else {
                if (!(g < n[e][0])) return !0;
                s = e - 1;
              }
              return !1;
            })(f, r) ? 0 : f >= 131072 && f <= 196605 || f >= 196608 && f <= 262141 ? 2 : 1;
          }
          charProperties(f, g) {
            let n = this.wcwidth(f), e = n === 0 && g !== 0;
            if (e) {
              const o = c.UnicodeService.extractWidth(g);
              o === 0 ? e = !1 : o > n && (n = o);
            }
            return c.UnicodeService.createPropertyValue(0, n, e);
          }
        };
      }, 5981: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.WriteBuffer = void 0;
        const c = a(8460), h = a(844);
        class r extends h.Disposable {
          constructor(f) {
            super(), this._action = f, this._writeBuffer = [], this._callbacks = [], this._pendingData = 0, this._bufferOffset = 0, this._isSyncWriting = !1, this._syncCalls = 0, this._didUserInput = !1, this._onWriteParsed = this.register(new c.EventEmitter()), this.onWriteParsed = this._onWriteParsed.event;
          }
          handleUserInput() {
            this._didUserInput = !0;
          }
          writeSync(f, g) {
            if (g !== void 0 && this._syncCalls > g) return void (this._syncCalls = 0);
            if (this._pendingData += f.length, this._writeBuffer.push(f), this._callbacks.push(void 0), this._syncCalls++, this._isSyncWriting) return;
            let n;
            for (this._isSyncWriting = !0; n = this._writeBuffer.shift(); ) {
              this._action(n);
              const e = this._callbacks.shift();
              e && e();
            }
            this._pendingData = 0, this._bufferOffset = 2147483647, this._isSyncWriting = !1, this._syncCalls = 0;
          }
          write(f, g) {
            if (this._pendingData > 5e7) throw new Error("write data discarded, use flow control to avoid losing data");
            if (!this._writeBuffer.length) {
              if (this._bufferOffset = 0, this._didUserInput) return this._didUserInput = !1, this._pendingData += f.length, this._writeBuffer.push(f), this._callbacks.push(g), void this._innerWrite();
              setTimeout((() => this._innerWrite()));
            }
            this._pendingData += f.length, this._writeBuffer.push(f), this._callbacks.push(g);
          }
          _innerWrite(f = 0, g = !0) {
            const n = f || Date.now();
            for (; this._writeBuffer.length > this._bufferOffset; ) {
              const e = this._writeBuffer[this._bufferOffset], o = this._action(e, g);
              if (o) {
                const i = (u) => Date.now() - n >= 12 ? setTimeout((() => this._innerWrite(0, u))) : this._innerWrite(n, u);
                return void o.catch(((u) => (queueMicrotask((() => {
                  throw u;
                })), Promise.resolve(!1)))).then(i);
              }
              const s = this._callbacks[this._bufferOffset];
              if (s && s(), this._bufferOffset++, this._pendingData -= e.length, Date.now() - n >= 12) break;
            }
            this._writeBuffer.length > this._bufferOffset ? (this._bufferOffset > 50 && (this._writeBuffer = this._writeBuffer.slice(this._bufferOffset), this._callbacks = this._callbacks.slice(this._bufferOffset), this._bufferOffset = 0), setTimeout((() => this._innerWrite()))) : (this._writeBuffer.length = 0, this._callbacks.length = 0, this._pendingData = 0, this._bufferOffset = 0), this._onWriteParsed.fire();
          }
        }
        t.WriteBuffer = r;
      }, 5941: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.toRgbString = t.parseColor = void 0;
        const a = /^([\da-f])\/([\da-f])\/([\da-f])$|^([\da-f]{2})\/([\da-f]{2})\/([\da-f]{2})$|^([\da-f]{3})\/([\da-f]{3})\/([\da-f]{3})$|^([\da-f]{4})\/([\da-f]{4})\/([\da-f]{4})$/, c = /^[\da-f]+$/;
        function h(r, d) {
          const f = r.toString(16), g = f.length < 2 ? "0" + f : f;
          switch (d) {
            case 4:
              return f[0];
            case 8:
              return g;
            case 12:
              return (g + g).slice(0, 3);
            default:
              return g + g;
          }
        }
        t.parseColor = function(r) {
          if (!r) return;
          let d = r.toLowerCase();
          if (d.indexOf("rgb:") === 0) {
            d = d.slice(4);
            const f = a.exec(d);
            if (f) {
              const g = f[1] ? 15 : f[4] ? 255 : f[7] ? 4095 : 65535;
              return [Math.round(parseInt(f[1] || f[4] || f[7] || f[10], 16) / g * 255), Math.round(parseInt(f[2] || f[5] || f[8] || f[11], 16) / g * 255), Math.round(parseInt(f[3] || f[6] || f[9] || f[12], 16) / g * 255)];
            }
          } else if (d.indexOf("#") === 0 && (d = d.slice(1), c.exec(d) && [3, 6, 9, 12].includes(d.length))) {
            const f = d.length / 3, g = [0, 0, 0];
            for (let n = 0; n < 3; ++n) {
              const e = parseInt(d.slice(f * n, f * n + f), 16);
              g[n] = f === 1 ? e << 4 : f === 2 ? e : f === 3 ? e >> 4 : e >> 8;
            }
            return g;
          }
        }, t.toRgbString = function(r, d = 16) {
          const [f, g, n] = r;
          return `rgb:${h(f, d)}/${h(g, d)}/${h(n, d)}`;
        };
      }, 5770: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.PAYLOAD_LIMIT = void 0, t.PAYLOAD_LIMIT = 1e7;
      }, 6351: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.DcsHandler = t.DcsParser = void 0;
        const c = a(482), h = a(8742), r = a(5770), d = [];
        t.DcsParser = class {
          constructor() {
            this._handlers = /* @__PURE__ */ Object.create(null), this._active = d, this._ident = 0, this._handlerFb = () => {
            }, this._stack = { paused: !1, loopPosition: 0, fallThrough: !1 };
          }
          dispose() {
            this._handlers = /* @__PURE__ */ Object.create(null), this._handlerFb = () => {
            }, this._active = d;
          }
          registerHandler(g, n) {
            this._handlers[g] === void 0 && (this._handlers[g] = []);
            const e = this._handlers[g];
            return e.push(n), { dispose: () => {
              const o = e.indexOf(n);
              o !== -1 && e.splice(o, 1);
            } };
          }
          clearHandler(g) {
            this._handlers[g] && delete this._handlers[g];
          }
          setHandlerFallback(g) {
            this._handlerFb = g;
          }
          reset() {
            if (this._active.length) for (let g = this._stack.paused ? this._stack.loopPosition - 1 : this._active.length - 1; g >= 0; --g) this._active[g].unhook(!1);
            this._stack.paused = !1, this._active = d, this._ident = 0;
          }
          hook(g, n) {
            if (this.reset(), this._ident = g, this._active = this._handlers[g] || d, this._active.length) for (let e = this._active.length - 1; e >= 0; e--) this._active[e].hook(n);
            else this._handlerFb(this._ident, "HOOK", n);
          }
          put(g, n, e) {
            if (this._active.length) for (let o = this._active.length - 1; o >= 0; o--) this._active[o].put(g, n, e);
            else this._handlerFb(this._ident, "PUT", (0, c.utf32ToString)(g, n, e));
          }
          unhook(g, n = !0) {
            if (this._active.length) {
              let e = !1, o = this._active.length - 1, s = !1;
              if (this._stack.paused && (o = this._stack.loopPosition - 1, e = n, s = this._stack.fallThrough, this._stack.paused = !1), !s && e === !1) {
                for (; o >= 0 && (e = this._active[o].unhook(g), e !== !0); o--) if (e instanceof Promise) return this._stack.paused = !0, this._stack.loopPosition = o, this._stack.fallThrough = !1, e;
                o--;
              }
              for (; o >= 0; o--) if (e = this._active[o].unhook(!1), e instanceof Promise) return this._stack.paused = !0, this._stack.loopPosition = o, this._stack.fallThrough = !0, e;
            } else this._handlerFb(this._ident, "UNHOOK", g);
            this._active = d, this._ident = 0;
          }
        };
        const f = new h.Params();
        f.addParam(0), t.DcsHandler = class {
          constructor(g) {
            this._handler = g, this._data = "", this._params = f, this._hitLimit = !1;
          }
          hook(g) {
            this._params = g.length > 1 || g.params[0] ? g.clone() : f, this._data = "", this._hitLimit = !1;
          }
          put(g, n, e) {
            this._hitLimit || (this._data += (0, c.utf32ToString)(g, n, e), this._data.length > r.PAYLOAD_LIMIT && (this._data = "", this._hitLimit = !0));
          }
          unhook(g) {
            let n = !1;
            if (this._hitLimit) n = !1;
            else if (g && (n = this._handler(this._data, this._params), n instanceof Promise)) return n.then(((e) => (this._params = f, this._data = "", this._hitLimit = !1, e)));
            return this._params = f, this._data = "", this._hitLimit = !1, n;
          }
        };
      }, 2015: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.EscapeSequenceParser = t.VT500_TRANSITION_TABLE = t.TransitionTable = void 0;
        const c = a(844), h = a(8742), r = a(6242), d = a(6351);
        class f {
          constructor(o) {
            this.table = new Uint8Array(o);
          }
          setDefault(o, s) {
            this.table.fill(o << 4 | s);
          }
          add(o, s, i, u) {
            this.table[s << 8 | o] = i << 4 | u;
          }
          addMany(o, s, i, u) {
            for (let p = 0; p < o.length; p++) this.table[s << 8 | o[p]] = i << 4 | u;
          }
        }
        t.TransitionTable = f;
        const g = 160;
        t.VT500_TRANSITION_TABLE = (function() {
          const e = new f(4095), o = Array.apply(null, Array(256)).map(((m, _) => _)), s = (m, _) => o.slice(m, _), i = s(32, 127), u = s(0, 24);
          u.push(25), u.push.apply(u, s(28, 32));
          const p = s(0, 14);
          let l;
          for (l in e.setDefault(1, 0), e.addMany(i, 0, 2, 0), p) e.addMany([24, 26, 153, 154], l, 3, 0), e.addMany(s(128, 144), l, 3, 0), e.addMany(s(144, 152), l, 3, 0), e.add(156, l, 0, 0), e.add(27, l, 11, 1), e.add(157, l, 4, 8), e.addMany([152, 158, 159], l, 0, 7), e.add(155, l, 11, 3), e.add(144, l, 11, 9);
          return e.addMany(u, 0, 3, 0), e.addMany(u, 1, 3, 1), e.add(127, 1, 0, 1), e.addMany(u, 8, 0, 8), e.addMany(u, 3, 3, 3), e.add(127, 3, 0, 3), e.addMany(u, 4, 3, 4), e.add(127, 4, 0, 4), e.addMany(u, 6, 3, 6), e.addMany(u, 5, 3, 5), e.add(127, 5, 0, 5), e.addMany(u, 2, 3, 2), e.add(127, 2, 0, 2), e.add(93, 1, 4, 8), e.addMany(i, 8, 5, 8), e.add(127, 8, 5, 8), e.addMany([156, 27, 24, 26, 7], 8, 6, 0), e.addMany(s(28, 32), 8, 0, 8), e.addMany([88, 94, 95], 1, 0, 7), e.addMany(i, 7, 0, 7), e.addMany(u, 7, 0, 7), e.add(156, 7, 0, 0), e.add(127, 7, 0, 7), e.add(91, 1, 11, 3), e.addMany(s(64, 127), 3, 7, 0), e.addMany(s(48, 60), 3, 8, 4), e.addMany([60, 61, 62, 63], 3, 9, 4), e.addMany(s(48, 60), 4, 8, 4), e.addMany(s(64, 127), 4, 7, 0), e.addMany([60, 61, 62, 63], 4, 0, 6), e.addMany(s(32, 64), 6, 0, 6), e.add(127, 6, 0, 6), e.addMany(s(64, 127), 6, 0, 0), e.addMany(s(32, 48), 3, 9, 5), e.addMany(s(32, 48), 5, 9, 5), e.addMany(s(48, 64), 5, 0, 6), e.addMany(s(64, 127), 5, 7, 0), e.addMany(s(32, 48), 4, 9, 5), e.addMany(s(32, 48), 1, 9, 2), e.addMany(s(32, 48), 2, 9, 2), e.addMany(s(48, 127), 2, 10, 0), e.addMany(s(48, 80), 1, 10, 0), e.addMany(s(81, 88), 1, 10, 0), e.addMany([89, 90, 92], 1, 10, 0), e.addMany(s(96, 127), 1, 10, 0), e.add(80, 1, 11, 9), e.addMany(u, 9, 0, 9), e.add(127, 9, 0, 9), e.addMany(s(28, 32), 9, 0, 9), e.addMany(s(32, 48), 9, 9, 12), e.addMany(s(48, 60), 9, 8, 10), e.addMany([60, 61, 62, 63], 9, 9, 10), e.addMany(u, 11, 0, 11), e.addMany(s(32, 128), 11, 0, 11), e.addMany(s(28, 32), 11, 0, 11), e.addMany(u, 10, 0, 10), e.add(127, 10, 0, 10), e.addMany(s(28, 32), 10, 0, 10), e.addMany(s(48, 60), 10, 8, 10), e.addMany([60, 61, 62, 63], 10, 0, 11), e.addMany(s(32, 48), 10, 9, 12), e.addMany(u, 12, 0, 12), e.add(127, 12, 0, 12), e.addMany(s(28, 32), 12, 0, 12), e.addMany(s(32, 48), 12, 9, 12), e.addMany(s(48, 64), 12, 0, 11), e.addMany(s(64, 127), 12, 12, 13), e.addMany(s(64, 127), 10, 12, 13), e.addMany(s(64, 127), 9, 12, 13), e.addMany(u, 13, 13, 13), e.addMany(i, 13, 13, 13), e.add(127, 13, 0, 13), e.addMany([27, 156, 24, 26], 13, 14, 0), e.add(g, 0, 2, 0), e.add(g, 8, 5, 8), e.add(g, 6, 0, 6), e.add(g, 11, 0, 11), e.add(g, 13, 13, 13), e;
        })();
        class n extends c.Disposable {
          constructor(o = t.VT500_TRANSITION_TABLE) {
            super(), this._transitions = o, this._parseStack = { state: 0, handlers: [], handlerPos: 0, transition: 0, chunkPos: 0 }, this.initialState = 0, this.currentState = this.initialState, this._params = new h.Params(), this._params.addParam(0), this._collect = 0, this.precedingJoinState = 0, this._printHandlerFb = (s, i, u) => {
            }, this._executeHandlerFb = (s) => {
            }, this._csiHandlerFb = (s, i) => {
            }, this._escHandlerFb = (s) => {
            }, this._errorHandlerFb = (s) => s, this._printHandler = this._printHandlerFb, this._executeHandlers = /* @__PURE__ */ Object.create(null), this._csiHandlers = /* @__PURE__ */ Object.create(null), this._escHandlers = /* @__PURE__ */ Object.create(null), this.register((0, c.toDisposable)((() => {
              this._csiHandlers = /* @__PURE__ */ Object.create(null), this._executeHandlers = /* @__PURE__ */ Object.create(null), this._escHandlers = /* @__PURE__ */ Object.create(null);
            }))), this._oscParser = this.register(new r.OscParser()), this._dcsParser = this.register(new d.DcsParser()), this._errorHandler = this._errorHandlerFb, this.registerEscHandler({ final: "\\" }, (() => !0));
          }
          _identifier(o, s = [64, 126]) {
            let i = 0;
            if (o.prefix) {
              if (o.prefix.length > 1) throw new Error("only one byte as prefix supported");
              if (i = o.prefix.charCodeAt(0), i && 60 > i || i > 63) throw new Error("prefix must be in range 0x3c .. 0x3f");
            }
            if (o.intermediates) {
              if (o.intermediates.length > 2) throw new Error("only two bytes as intermediates are supported");
              for (let p = 0; p < o.intermediates.length; ++p) {
                const l = o.intermediates.charCodeAt(p);
                if (32 > l || l > 47) throw new Error("intermediate must be in range 0x20 .. 0x2f");
                i <<= 8, i |= l;
              }
            }
            if (o.final.length !== 1) throw new Error("final must be a single byte");
            const u = o.final.charCodeAt(0);
            if (s[0] > u || u > s[1]) throw new Error(`final must be in range ${s[0]} .. ${s[1]}`);
            return i <<= 8, i |= u, i;
          }
          identToString(o) {
            const s = [];
            for (; o; ) s.push(String.fromCharCode(255 & o)), o >>= 8;
            return s.reverse().join("");
          }
          setPrintHandler(o) {
            this._printHandler = o;
          }
          clearPrintHandler() {
            this._printHandler = this._printHandlerFb;
          }
          registerEscHandler(o, s) {
            const i = this._identifier(o, [48, 126]);
            this._escHandlers[i] === void 0 && (this._escHandlers[i] = []);
            const u = this._escHandlers[i];
            return u.push(s), { dispose: () => {
              const p = u.indexOf(s);
              p !== -1 && u.splice(p, 1);
            } };
          }
          clearEscHandler(o) {
            this._escHandlers[this._identifier(o, [48, 126])] && delete this._escHandlers[this._identifier(o, [48, 126])];
          }
          setEscHandlerFallback(o) {
            this._escHandlerFb = o;
          }
          setExecuteHandler(o, s) {
            this._executeHandlers[o.charCodeAt(0)] = s;
          }
          clearExecuteHandler(o) {
            this._executeHandlers[o.charCodeAt(0)] && delete this._executeHandlers[o.charCodeAt(0)];
          }
          setExecuteHandlerFallback(o) {
            this._executeHandlerFb = o;
          }
          registerCsiHandler(o, s) {
            const i = this._identifier(o);
            this._csiHandlers[i] === void 0 && (this._csiHandlers[i] = []);
            const u = this._csiHandlers[i];
            return u.push(s), { dispose: () => {
              const p = u.indexOf(s);
              p !== -1 && u.splice(p, 1);
            } };
          }
          clearCsiHandler(o) {
            this._csiHandlers[this._identifier(o)] && delete this._csiHandlers[this._identifier(o)];
          }
          setCsiHandlerFallback(o) {
            this._csiHandlerFb = o;
          }
          registerDcsHandler(o, s) {
            return this._dcsParser.registerHandler(this._identifier(o), s);
          }
          clearDcsHandler(o) {
            this._dcsParser.clearHandler(this._identifier(o));
          }
          setDcsHandlerFallback(o) {
            this._dcsParser.setHandlerFallback(o);
          }
          registerOscHandler(o, s) {
            return this._oscParser.registerHandler(o, s);
          }
          clearOscHandler(o) {
            this._oscParser.clearHandler(o);
          }
          setOscHandlerFallback(o) {
            this._oscParser.setHandlerFallback(o);
          }
          setErrorHandler(o) {
            this._errorHandler = o;
          }
          clearErrorHandler() {
            this._errorHandler = this._errorHandlerFb;
          }
          reset() {
            this.currentState = this.initialState, this._oscParser.reset(), this._dcsParser.reset(), this._params.reset(), this._params.addParam(0), this._collect = 0, this.precedingJoinState = 0, this._parseStack.state !== 0 && (this._parseStack.state = 2, this._parseStack.handlers = []);
          }
          _preserveStack(o, s, i, u, p) {
            this._parseStack.state = o, this._parseStack.handlers = s, this._parseStack.handlerPos = i, this._parseStack.transition = u, this._parseStack.chunkPos = p;
          }
          parse(o, s, i) {
            let u, p = 0, l = 0, m = 0;
            if (this._parseStack.state) if (this._parseStack.state === 2) this._parseStack.state = 0, m = this._parseStack.chunkPos + 1;
            else {
              if (i === void 0 || this._parseStack.state === 1) throw this._parseStack.state = 1, new Error("improper continuation due to previous async handler, giving up parsing");
              const _ = this._parseStack.handlers;
              let v = this._parseStack.handlerPos - 1;
              switch (this._parseStack.state) {
                case 3:
                  if (i === !1 && v > -1) {
                    for (; v >= 0 && (u = _[v](this._params), u !== !0); v--) if (u instanceof Promise) return this._parseStack.handlerPos = v, u;
                  }
                  this._parseStack.handlers = [];
                  break;
                case 4:
                  if (i === !1 && v > -1) {
                    for (; v >= 0 && (u = _[v](), u !== !0); v--) if (u instanceof Promise) return this._parseStack.handlerPos = v, u;
                  }
                  this._parseStack.handlers = [];
                  break;
                case 6:
                  if (p = o[this._parseStack.chunkPos], u = this._dcsParser.unhook(p !== 24 && p !== 26, i), u) return u;
                  p === 27 && (this._parseStack.transition |= 1), this._params.reset(), this._params.addParam(0), this._collect = 0;
                  break;
                case 5:
                  if (p = o[this._parseStack.chunkPos], u = this._oscParser.end(p !== 24 && p !== 26, i), u) return u;
                  p === 27 && (this._parseStack.transition |= 1), this._params.reset(), this._params.addParam(0), this._collect = 0;
              }
              this._parseStack.state = 0, m = this._parseStack.chunkPos + 1, this.precedingJoinState = 0, this.currentState = 15 & this._parseStack.transition;
            }
            for (let _ = m; _ < s; ++_) {
              switch (p = o[_], l = this._transitions.table[this.currentState << 8 | (p < 160 ? p : g)], l >> 4) {
                case 2:
                  for (let b = _ + 1; ; ++b) {
                    if (b >= s || (p = o[b]) < 32 || p > 126 && p < g) {
                      this._printHandler(o, _, b), _ = b - 1;
                      break;
                    }
                    if (++b >= s || (p = o[b]) < 32 || p > 126 && p < g) {
                      this._printHandler(o, _, b), _ = b - 1;
                      break;
                    }
                    if (++b >= s || (p = o[b]) < 32 || p > 126 && p < g) {
                      this._printHandler(o, _, b), _ = b - 1;
                      break;
                    }
                    if (++b >= s || (p = o[b]) < 32 || p > 126 && p < g) {
                      this._printHandler(o, _, b), _ = b - 1;
                      break;
                    }
                  }
                  break;
                case 3:
                  this._executeHandlers[p] ? this._executeHandlers[p]() : this._executeHandlerFb(p), this.precedingJoinState = 0;
                  break;
                case 0:
                  break;
                case 1:
                  if (this._errorHandler({ position: _, code: p, currentState: this.currentState, collect: this._collect, params: this._params, abort: !1 }).abort) return;
                  break;
                case 7:
                  const v = this._csiHandlers[this._collect << 8 | p];
                  let C = v ? v.length - 1 : -1;
                  for (; C >= 0 && (u = v[C](this._params), u !== !0); C--) if (u instanceof Promise) return this._preserveStack(3, v, C, l, _), u;
                  C < 0 && this._csiHandlerFb(this._collect << 8 | p, this._params), this.precedingJoinState = 0;
                  break;
                case 8:
                  do
                    switch (p) {
                      case 59:
                        this._params.addParam(0);
                        break;
                      case 58:
                        this._params.addSubParam(-1);
                        break;
                      default:
                        this._params.addDigit(p - 48);
                    }
                  while (++_ < s && (p = o[_]) > 47 && p < 60);
                  _--;
                  break;
                case 9:
                  this._collect <<= 8, this._collect |= p;
                  break;
                case 10:
                  const w = this._escHandlers[this._collect << 8 | p];
                  let S = w ? w.length - 1 : -1;
                  for (; S >= 0 && (u = w[S](), u !== !0); S--) if (u instanceof Promise) return this._preserveStack(4, w, S, l, _), u;
                  S < 0 && this._escHandlerFb(this._collect << 8 | p), this.precedingJoinState = 0;
                  break;
                case 11:
                  this._params.reset(), this._params.addParam(0), this._collect = 0;
                  break;
                case 12:
                  this._dcsParser.hook(this._collect << 8 | p, this._params);
                  break;
                case 13:
                  for (let b = _ + 1; ; ++b) if (b >= s || (p = o[b]) === 24 || p === 26 || p === 27 || p > 127 && p < g) {
                    this._dcsParser.put(o, _, b), _ = b - 1;
                    break;
                  }
                  break;
                case 14:
                  if (u = this._dcsParser.unhook(p !== 24 && p !== 26), u) return this._preserveStack(6, [], 0, l, _), u;
                  p === 27 && (l |= 1), this._params.reset(), this._params.addParam(0), this._collect = 0, this.precedingJoinState = 0;
                  break;
                case 4:
                  this._oscParser.start();
                  break;
                case 5:
                  for (let b = _ + 1; ; b++) if (b >= s || (p = o[b]) < 32 || p > 127 && p < g) {
                    this._oscParser.put(o, _, b), _ = b - 1;
                    break;
                  }
                  break;
                case 6:
                  if (u = this._oscParser.end(p !== 24 && p !== 26), u) return this._preserveStack(5, [], 0, l, _), u;
                  p === 27 && (l |= 1), this._params.reset(), this._params.addParam(0), this._collect = 0, this.precedingJoinState = 0;
              }
              this.currentState = 15 & l;
            }
          }
        }
        t.EscapeSequenceParser = n;
      }, 6242: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.OscHandler = t.OscParser = void 0;
        const c = a(5770), h = a(482), r = [];
        t.OscParser = class {
          constructor() {
            this._state = 0, this._active = r, this._id = -1, this._handlers = /* @__PURE__ */ Object.create(null), this._handlerFb = () => {
            }, this._stack = { paused: !1, loopPosition: 0, fallThrough: !1 };
          }
          registerHandler(d, f) {
            this._handlers[d] === void 0 && (this._handlers[d] = []);
            const g = this._handlers[d];
            return g.push(f), { dispose: () => {
              const n = g.indexOf(f);
              n !== -1 && g.splice(n, 1);
            } };
          }
          clearHandler(d) {
            this._handlers[d] && delete this._handlers[d];
          }
          setHandlerFallback(d) {
            this._handlerFb = d;
          }
          dispose() {
            this._handlers = /* @__PURE__ */ Object.create(null), this._handlerFb = () => {
            }, this._active = r;
          }
          reset() {
            if (this._state === 2) for (let d = this._stack.paused ? this._stack.loopPosition - 1 : this._active.length - 1; d >= 0; --d) this._active[d].end(!1);
            this._stack.paused = !1, this._active = r, this._id = -1, this._state = 0;
          }
          _start() {
            if (this._active = this._handlers[this._id] || r, this._active.length) for (let d = this._active.length - 1; d >= 0; d--) this._active[d].start();
            else this._handlerFb(this._id, "START");
          }
          _put(d, f, g) {
            if (this._active.length) for (let n = this._active.length - 1; n >= 0; n--) this._active[n].put(d, f, g);
            else this._handlerFb(this._id, "PUT", (0, h.utf32ToString)(d, f, g));
          }
          start() {
            this.reset(), this._state = 1;
          }
          put(d, f, g) {
            if (this._state !== 3) {
              if (this._state === 1) for (; f < g; ) {
                const n = d[f++];
                if (n === 59) {
                  this._state = 2, this._start();
                  break;
                }
                if (n < 48 || 57 < n) return void (this._state = 3);
                this._id === -1 && (this._id = 0), this._id = 10 * this._id + n - 48;
              }
              this._state === 2 && g - f > 0 && this._put(d, f, g);
            }
          }
          end(d, f = !0) {
            if (this._state !== 0) {
              if (this._state !== 3) if (this._state === 1 && this._start(), this._active.length) {
                let g = !1, n = this._active.length - 1, e = !1;
                if (this._stack.paused && (n = this._stack.loopPosition - 1, g = f, e = this._stack.fallThrough, this._stack.paused = !1), !e && g === !1) {
                  for (; n >= 0 && (g = this._active[n].end(d), g !== !0); n--) if (g instanceof Promise) return this._stack.paused = !0, this._stack.loopPosition = n, this._stack.fallThrough = !1, g;
                  n--;
                }
                for (; n >= 0; n--) if (g = this._active[n].end(!1), g instanceof Promise) return this._stack.paused = !0, this._stack.loopPosition = n, this._stack.fallThrough = !0, g;
              } else this._handlerFb(this._id, "END", d);
              this._active = r, this._id = -1, this._state = 0;
            }
          }
        }, t.OscHandler = class {
          constructor(d) {
            this._handler = d, this._data = "", this._hitLimit = !1;
          }
          start() {
            this._data = "", this._hitLimit = !1;
          }
          put(d, f, g) {
            this._hitLimit || (this._data += (0, h.utf32ToString)(d, f, g), this._data.length > c.PAYLOAD_LIMIT && (this._data = "", this._hitLimit = !0));
          }
          end(d) {
            let f = !1;
            if (this._hitLimit) f = !1;
            else if (d && (f = this._handler(this._data), f instanceof Promise)) return f.then(((g) => (this._data = "", this._hitLimit = !1, g)));
            return this._data = "", this._hitLimit = !1, f;
          }
        };
      }, 8742: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.Params = void 0;
        const a = 2147483647;
        class c {
          static fromArray(r) {
            const d = new c();
            if (!r.length) return d;
            for (let f = Array.isArray(r[0]) ? 1 : 0; f < r.length; ++f) {
              const g = r[f];
              if (Array.isArray(g)) for (let n = 0; n < g.length; ++n) d.addSubParam(g[n]);
              else d.addParam(g);
            }
            return d;
          }
          constructor(r = 32, d = 32) {
            if (this.maxLength = r, this.maxSubParamsLength = d, d > 256) throw new Error("maxSubParamsLength must not be greater than 256");
            this.params = new Int32Array(r), this.length = 0, this._subParams = new Int32Array(d), this._subParamsLength = 0, this._subParamsIdx = new Uint16Array(r), this._rejectDigits = !1, this._rejectSubDigits = !1, this._digitIsSub = !1;
          }
          clone() {
            const r = new c(this.maxLength, this.maxSubParamsLength);
            return r.params.set(this.params), r.length = this.length, r._subParams.set(this._subParams), r._subParamsLength = this._subParamsLength, r._subParamsIdx.set(this._subParamsIdx), r._rejectDigits = this._rejectDigits, r._rejectSubDigits = this._rejectSubDigits, r._digitIsSub = this._digitIsSub, r;
          }
          toArray() {
            const r = [];
            for (let d = 0; d < this.length; ++d) {
              r.push(this.params[d]);
              const f = this._subParamsIdx[d] >> 8, g = 255 & this._subParamsIdx[d];
              g - f > 0 && r.push(Array.prototype.slice.call(this._subParams, f, g));
            }
            return r;
          }
          reset() {
            this.length = 0, this._subParamsLength = 0, this._rejectDigits = !1, this._rejectSubDigits = !1, this._digitIsSub = !1;
          }
          addParam(r) {
            if (this._digitIsSub = !1, this.length >= this.maxLength) this._rejectDigits = !0;
            else {
              if (r < -1) throw new Error("values lesser than -1 are not allowed");
              this._subParamsIdx[this.length] = this._subParamsLength << 8 | this._subParamsLength, this.params[this.length++] = r > a ? a : r;
            }
          }
          addSubParam(r) {
            if (this._digitIsSub = !0, this.length) if (this._rejectDigits || this._subParamsLength >= this.maxSubParamsLength) this._rejectSubDigits = !0;
            else {
              if (r < -1) throw new Error("values lesser than -1 are not allowed");
              this._subParams[this._subParamsLength++] = r > a ? a : r, this._subParamsIdx[this.length - 1]++;
            }
          }
          hasSubParams(r) {
            return (255 & this._subParamsIdx[r]) - (this._subParamsIdx[r] >> 8) > 0;
          }
          getSubParams(r) {
            const d = this._subParamsIdx[r] >> 8, f = 255 & this._subParamsIdx[r];
            return f - d > 0 ? this._subParams.subarray(d, f) : null;
          }
          getSubParamsAll() {
            const r = {};
            for (let d = 0; d < this.length; ++d) {
              const f = this._subParamsIdx[d] >> 8, g = 255 & this._subParamsIdx[d];
              g - f > 0 && (r[d] = this._subParams.slice(f, g));
            }
            return r;
          }
          addDigit(r) {
            let d;
            if (this._rejectDigits || !(d = this._digitIsSub ? this._subParamsLength : this.length) || this._digitIsSub && this._rejectSubDigits) return;
            const f = this._digitIsSub ? this._subParams : this.params, g = f[d - 1];
            f[d - 1] = ~g ? Math.min(10 * g + r, a) : r;
          }
        }
        t.Params = c;
      }, 5741: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.AddonManager = void 0, t.AddonManager = class {
          constructor() {
            this._addons = [];
          }
          dispose() {
            for (let a = this._addons.length - 1; a >= 0; a--) this._addons[a].instance.dispose();
          }
          loadAddon(a, c) {
            const h = { instance: c, dispose: c.dispose, isDisposed: !1 };
            this._addons.push(h), c.dispose = () => this._wrappedAddonDispose(h), c.activate(a);
          }
          _wrappedAddonDispose(a) {
            if (a.isDisposed) return;
            let c = -1;
            for (let h = 0; h < this._addons.length; h++) if (this._addons[h] === a) {
              c = h;
              break;
            }
            if (c === -1) throw new Error("Could not dispose an addon that has not been loaded");
            a.isDisposed = !0, a.dispose.apply(a.instance), this._addons.splice(c, 1);
          }
        };
      }, 8771: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.BufferApiView = void 0;
        const c = a(3785), h = a(511);
        t.BufferApiView = class {
          constructor(r, d) {
            this._buffer = r, this.type = d;
          }
          init(r) {
            return this._buffer = r, this;
          }
          get cursorY() {
            return this._buffer.y;
          }
          get cursorX() {
            return this._buffer.x;
          }
          get viewportY() {
            return this._buffer.ydisp;
          }
          get baseY() {
            return this._buffer.ybase;
          }
          get length() {
            return this._buffer.lines.length;
          }
          getLine(r) {
            const d = this._buffer.lines.get(r);
            if (d) return new c.BufferLineApiView(d);
          }
          getNullCell() {
            return new h.CellData();
          }
        };
      }, 3785: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.BufferLineApiView = void 0;
        const c = a(511);
        t.BufferLineApiView = class {
          constructor(h) {
            this._line = h;
          }
          get isWrapped() {
            return this._line.isWrapped;
          }
          get length() {
            return this._line.length;
          }
          getCell(h, r) {
            if (!(h < 0 || h >= this._line.length)) return r ? (this._line.loadCell(h, r), r) : this._line.loadCell(h, new c.CellData());
          }
          translateToString(h, r, d) {
            return this._line.translateToString(h, r, d);
          }
        };
      }, 8285: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.BufferNamespaceApi = void 0;
        const c = a(8771), h = a(8460), r = a(844);
        class d extends r.Disposable {
          constructor(g) {
            super(), this._core = g, this._onBufferChange = this.register(new h.EventEmitter()), this.onBufferChange = this._onBufferChange.event, this._normal = new c.BufferApiView(this._core.buffers.normal, "normal"), this._alternate = new c.BufferApiView(this._core.buffers.alt, "alternate"), this._core.buffers.onBufferActivate((() => this._onBufferChange.fire(this.active)));
          }
          get active() {
            if (this._core.buffers.active === this._core.buffers.normal) return this.normal;
            if (this._core.buffers.active === this._core.buffers.alt) return this.alternate;
            throw new Error("Active buffer is neither normal nor alternate");
          }
          get normal() {
            return this._normal.init(this._core.buffers.normal);
          }
          get alternate() {
            return this._alternate.init(this._core.buffers.alt);
          }
        }
        t.BufferNamespaceApi = d;
      }, 7975: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.ParserApi = void 0, t.ParserApi = class {
          constructor(a) {
            this._core = a;
          }
          registerCsiHandler(a, c) {
            return this._core.registerCsiHandler(a, ((h) => c(h.toArray())));
          }
          addCsiHandler(a, c) {
            return this.registerCsiHandler(a, c);
          }
          registerDcsHandler(a, c) {
            return this._core.registerDcsHandler(a, ((h, r) => c(h, r.toArray())));
          }
          addDcsHandler(a, c) {
            return this.registerDcsHandler(a, c);
          }
          registerEscHandler(a, c) {
            return this._core.registerEscHandler(a, c);
          }
          addEscHandler(a, c) {
            return this.registerEscHandler(a, c);
          }
          registerOscHandler(a, c) {
            return this._core.registerOscHandler(a, c);
          }
          addOscHandler(a, c) {
            return this.registerOscHandler(a, c);
          }
        };
      }, 7090: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.UnicodeApi = void 0, t.UnicodeApi = class {
          constructor(a) {
            this._core = a;
          }
          register(a) {
            this._core.unicodeService.register(a);
          }
          get versions() {
            return this._core.unicodeService.versions;
          }
          get activeVersion() {
            return this._core.unicodeService.activeVersion;
          }
          set activeVersion(a) {
            this._core.unicodeService.activeVersion = a;
          }
        };
      }, 744: function(T, t, a) {
        var c = this && this.__decorate || function(e, o, s, i) {
          var u, p = arguments.length, l = p < 3 ? o : i === null ? i = Object.getOwnPropertyDescriptor(o, s) : i;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") l = Reflect.decorate(e, o, s, i);
          else for (var m = e.length - 1; m >= 0; m--) (u = e[m]) && (l = (p < 3 ? u(l) : p > 3 ? u(o, s, l) : u(o, s)) || l);
          return p > 3 && l && Object.defineProperty(o, s, l), l;
        }, h = this && this.__param || function(e, o) {
          return function(s, i) {
            o(s, i, e);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.BufferService = t.MINIMUM_ROWS = t.MINIMUM_COLS = void 0;
        const r = a(8460), d = a(844), f = a(5295), g = a(2585);
        t.MINIMUM_COLS = 2, t.MINIMUM_ROWS = 1;
        let n = t.BufferService = class extends d.Disposable {
          get buffer() {
            return this.buffers.active;
          }
          constructor(e) {
            super(), this.isUserScrolling = !1, this._onResize = this.register(new r.EventEmitter()), this.onResize = this._onResize.event, this._onScroll = this.register(new r.EventEmitter()), this.onScroll = this._onScroll.event, this.cols = Math.max(e.rawOptions.cols || 0, t.MINIMUM_COLS), this.rows = Math.max(e.rawOptions.rows || 0, t.MINIMUM_ROWS), this.buffers = this.register(new f.BufferSet(e, this));
          }
          resize(e, o) {
            this.cols = e, this.rows = o, this.buffers.resize(e, o), this._onResize.fire({ cols: e, rows: o });
          }
          reset() {
            this.buffers.reset(), this.isUserScrolling = !1;
          }
          scroll(e, o = !1) {
            const s = this.buffer;
            let i;
            i = this._cachedBlankLine, i && i.length === this.cols && i.getFg(0) === e.fg && i.getBg(0) === e.bg || (i = s.getBlankLine(e, o), this._cachedBlankLine = i), i.isWrapped = o;
            const u = s.ybase + s.scrollTop, p = s.ybase + s.scrollBottom;
            if (s.scrollTop === 0) {
              const l = s.lines.isFull;
              p === s.lines.length - 1 ? l ? s.lines.recycle().copyFrom(i) : s.lines.push(i.clone()) : s.lines.splice(p + 1, 0, i.clone()), l ? this.isUserScrolling && (s.ydisp = Math.max(s.ydisp - 1, 0)) : (s.ybase++, this.isUserScrolling || s.ydisp++);
            } else {
              const l = p - u + 1;
              s.lines.shiftElements(u + 1, l - 1, -1), s.lines.set(p, i.clone());
            }
            this.isUserScrolling || (s.ydisp = s.ybase), this._onScroll.fire(s.ydisp);
          }
          scrollLines(e, o, s) {
            const i = this.buffer;
            if (e < 0) {
              if (i.ydisp === 0) return;
              this.isUserScrolling = !0;
            } else e + i.ydisp >= i.ybase && (this.isUserScrolling = !1);
            const u = i.ydisp;
            i.ydisp = Math.max(Math.min(i.ydisp + e, i.ybase), 0), u !== i.ydisp && (o || this._onScroll.fire(i.ydisp));
          }
        };
        t.BufferService = n = c([h(0, g.IOptionsService)], n);
      }, 7994: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CharsetService = void 0, t.CharsetService = class {
          constructor() {
            this.glevel = 0, this._charsets = [];
          }
          reset() {
            this.charset = void 0, this._charsets = [], this.glevel = 0;
          }
          setgLevel(a) {
            this.glevel = a, this.charset = this._charsets[a];
          }
          setgCharset(a, c) {
            this._charsets[a] = c, this.glevel === a && (this.charset = c);
          }
        };
      }, 1753: function(T, t, a) {
        var c = this && this.__decorate || function(i, u, p, l) {
          var m, _ = arguments.length, v = _ < 3 ? u : l === null ? l = Object.getOwnPropertyDescriptor(u, p) : l;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") v = Reflect.decorate(i, u, p, l);
          else for (var C = i.length - 1; C >= 0; C--) (m = i[C]) && (v = (_ < 3 ? m(v) : _ > 3 ? m(u, p, v) : m(u, p)) || v);
          return _ > 3 && v && Object.defineProperty(u, p, v), v;
        }, h = this && this.__param || function(i, u) {
          return function(p, l) {
            u(p, l, i);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CoreMouseService = void 0;
        const r = a(2585), d = a(8460), f = a(844), g = { NONE: { events: 0, restrict: () => !1 }, X10: { events: 1, restrict: (i) => i.button !== 4 && i.action === 1 && (i.ctrl = !1, i.alt = !1, i.shift = !1, !0) }, VT200: { events: 19, restrict: (i) => i.action !== 32 }, DRAG: { events: 23, restrict: (i) => i.action !== 32 || i.button !== 3 }, ANY: { events: 31, restrict: (i) => !0 } };
        function n(i, u) {
          let p = (i.ctrl ? 16 : 0) | (i.shift ? 4 : 0) | (i.alt ? 8 : 0);
          return i.button === 4 ? (p |= 64, p |= i.action) : (p |= 3 & i.button, 4 & i.button && (p |= 64), 8 & i.button && (p |= 128), i.action === 32 ? p |= 32 : i.action !== 0 || u || (p |= 3)), p;
        }
        const e = String.fromCharCode, o = { DEFAULT: (i) => {
          const u = [n(i, !1) + 32, i.col + 32, i.row + 32];
          return u[0] > 255 || u[1] > 255 || u[2] > 255 ? "" : `\x1B[M${e(u[0])}${e(u[1])}${e(u[2])}`;
        }, SGR: (i) => {
          const u = i.action === 0 && i.button !== 4 ? "m" : "M";
          return `\x1B[<${n(i, !0)};${i.col};${i.row}${u}`;
        }, SGR_PIXELS: (i) => {
          const u = i.action === 0 && i.button !== 4 ? "m" : "M";
          return `\x1B[<${n(i, !0)};${i.x};${i.y}${u}`;
        } };
        let s = t.CoreMouseService = class extends f.Disposable {
          constructor(i, u) {
            super(), this._bufferService = i, this._coreService = u, this._protocols = {}, this._encodings = {}, this._activeProtocol = "", this._activeEncoding = "", this._lastEvent = null, this._onProtocolChange = this.register(new d.EventEmitter()), this.onProtocolChange = this._onProtocolChange.event;
            for (const p of Object.keys(g)) this.addProtocol(p, g[p]);
            for (const p of Object.keys(o)) this.addEncoding(p, o[p]);
            this.reset();
          }
          addProtocol(i, u) {
            this._protocols[i] = u;
          }
          addEncoding(i, u) {
            this._encodings[i] = u;
          }
          get activeProtocol() {
            return this._activeProtocol;
          }
          get areMouseEventsActive() {
            return this._protocols[this._activeProtocol].events !== 0;
          }
          set activeProtocol(i) {
            if (!this._protocols[i]) throw new Error(`unknown protocol "${i}"`);
            this._activeProtocol = i, this._onProtocolChange.fire(this._protocols[i].events);
          }
          get activeEncoding() {
            return this._activeEncoding;
          }
          set activeEncoding(i) {
            if (!this._encodings[i]) throw new Error(`unknown encoding "${i}"`);
            this._activeEncoding = i;
          }
          reset() {
            this.activeProtocol = "NONE", this.activeEncoding = "DEFAULT", this._lastEvent = null;
          }
          triggerMouseEvent(i) {
            if (i.col < 0 || i.col >= this._bufferService.cols || i.row < 0 || i.row >= this._bufferService.rows || i.button === 4 && i.action === 32 || i.button === 3 && i.action !== 32 || i.button !== 4 && (i.action === 2 || i.action === 3) || (i.col++, i.row++, i.action === 32 && this._lastEvent && this._equalEvents(this._lastEvent, i, this._activeEncoding === "SGR_PIXELS")) || !this._protocols[this._activeProtocol].restrict(i)) return !1;
            const u = this._encodings[this._activeEncoding](i);
            return u && (this._activeEncoding === "DEFAULT" ? this._coreService.triggerBinaryEvent(u) : this._coreService.triggerDataEvent(u, !0)), this._lastEvent = i, !0;
          }
          explainEvents(i) {
            return { down: !!(1 & i), up: !!(2 & i), drag: !!(4 & i), move: !!(8 & i), wheel: !!(16 & i) };
          }
          _equalEvents(i, u, p) {
            if (p) {
              if (i.x !== u.x || i.y !== u.y) return !1;
            } else if (i.col !== u.col || i.row !== u.row) return !1;
            return i.button === u.button && i.action === u.action && i.ctrl === u.ctrl && i.alt === u.alt && i.shift === u.shift;
          }
        };
        t.CoreMouseService = s = c([h(0, r.IBufferService), h(1, r.ICoreService)], s);
      }, 6975: function(T, t, a) {
        var c = this && this.__decorate || function(s, i, u, p) {
          var l, m = arguments.length, _ = m < 3 ? i : p === null ? p = Object.getOwnPropertyDescriptor(i, u) : p;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") _ = Reflect.decorate(s, i, u, p);
          else for (var v = s.length - 1; v >= 0; v--) (l = s[v]) && (_ = (m < 3 ? l(_) : m > 3 ? l(i, u, _) : l(i, u)) || _);
          return m > 3 && _ && Object.defineProperty(i, u, _), _;
        }, h = this && this.__param || function(s, i) {
          return function(u, p) {
            i(u, p, s);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CoreService = void 0;
        const r = a(1439), d = a(8460), f = a(844), g = a(2585), n = Object.freeze({ insertMode: !1 }), e = Object.freeze({ applicationCursorKeys: !1, applicationKeypad: !1, bracketedPasteMode: !1, origin: !1, reverseWraparound: !1, sendFocus: !1, wraparound: !0 });
        let o = t.CoreService = class extends f.Disposable {
          constructor(s, i, u) {
            super(), this._bufferService = s, this._logService = i, this._optionsService = u, this.isCursorInitialized = !1, this.isCursorHidden = !1, this._onData = this.register(new d.EventEmitter()), this.onData = this._onData.event, this._onUserInput = this.register(new d.EventEmitter()), this.onUserInput = this._onUserInput.event, this._onBinary = this.register(new d.EventEmitter()), this.onBinary = this._onBinary.event, this._onRequestScrollToBottom = this.register(new d.EventEmitter()), this.onRequestScrollToBottom = this._onRequestScrollToBottom.event, this.modes = (0, r.clone)(n), this.decPrivateModes = (0, r.clone)(e);
          }
          reset() {
            this.modes = (0, r.clone)(n), this.decPrivateModes = (0, r.clone)(e);
          }
          triggerDataEvent(s, i = !1) {
            if (this._optionsService.rawOptions.disableStdin) return;
            const u = this._bufferService.buffer;
            i && this._optionsService.rawOptions.scrollOnUserInput && u.ybase !== u.ydisp && this._onRequestScrollToBottom.fire(), i && this._onUserInput.fire(), this._logService.debug(`sending data "${s}"`, (() => s.split("").map(((p) => p.charCodeAt(0))))), this._onData.fire(s);
          }
          triggerBinaryEvent(s) {
            this._optionsService.rawOptions.disableStdin || (this._logService.debug(`sending binary "${s}"`, (() => s.split("").map(((i) => i.charCodeAt(0))))), this._onBinary.fire(s));
          }
        };
        t.CoreService = o = c([h(0, g.IBufferService), h(1, g.ILogService), h(2, g.IOptionsService)], o);
      }, 9074: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.DecorationService = void 0;
        const c = a(8055), h = a(8460), r = a(844), d = a(6106);
        let f = 0, g = 0;
        class n extends r.Disposable {
          get decorations() {
            return this._decorations.values();
          }
          constructor() {
            super(), this._decorations = new d.SortedList(((s) => s == null ? void 0 : s.marker.line)), this._onDecorationRegistered = this.register(new h.EventEmitter()), this.onDecorationRegistered = this._onDecorationRegistered.event, this._onDecorationRemoved = this.register(new h.EventEmitter()), this.onDecorationRemoved = this._onDecorationRemoved.event, this.register((0, r.toDisposable)((() => this.reset())));
          }
          registerDecoration(s) {
            if (s.marker.isDisposed) return;
            const i = new e(s);
            if (i) {
              const u = i.marker.onDispose((() => i.dispose()));
              i.onDispose((() => {
                i && (this._decorations.delete(i) && this._onDecorationRemoved.fire(i), u.dispose());
              })), this._decorations.insert(i), this._onDecorationRegistered.fire(i);
            }
            return i;
          }
          reset() {
            for (const s of this._decorations.values()) s.dispose();
            this._decorations.clear();
          }
          *getDecorationsAtCell(s, i, u) {
            var m, _, v;
            let p = 0, l = 0;
            for (const C of this._decorations.getKeyIterator(i)) p = (m = C.options.x) != null ? m : 0, l = p + ((_ = C.options.width) != null ? _ : 1), s >= p && s < l && (!u || ((v = C.options.layer) != null ? v : "bottom") === u) && (yield C);
          }
          forEachDecorationAtCell(s, i, u, p) {
            this._decorations.forEachByKey(i, ((l) => {
              var m, _, v;
              f = (m = l.options.x) != null ? m : 0, g = f + ((_ = l.options.width) != null ? _ : 1), s >= f && s < g && (!u || ((v = l.options.layer) != null ? v : "bottom") === u) && p(l);
            }));
          }
        }
        t.DecorationService = n;
        class e extends r.Disposable {
          get isDisposed() {
            return this._isDisposed;
          }
          get backgroundColorRGB() {
            return this._cachedBg === null && (this.options.backgroundColor ? this._cachedBg = c.css.toColor(this.options.backgroundColor) : this._cachedBg = void 0), this._cachedBg;
          }
          get foregroundColorRGB() {
            return this._cachedFg === null && (this.options.foregroundColor ? this._cachedFg = c.css.toColor(this.options.foregroundColor) : this._cachedFg = void 0), this._cachedFg;
          }
          constructor(s) {
            super(), this.options = s, this.onRenderEmitter = this.register(new h.EventEmitter()), this.onRender = this.onRenderEmitter.event, this._onDispose = this.register(new h.EventEmitter()), this.onDispose = this._onDispose.event, this._cachedBg = null, this._cachedFg = null, this.marker = s.marker, this.options.overviewRulerOptions && !this.options.overviewRulerOptions.position && (this.options.overviewRulerOptions.position = "full");
          }
          dispose() {
            this._onDispose.fire(), super.dispose();
          }
        }
      }, 4348: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.InstantiationService = t.ServiceCollection = void 0;
        const c = a(2585), h = a(8343);
        class r {
          constructor(...f) {
            this._entries = /* @__PURE__ */ new Map();
            for (const [g, n] of f) this.set(g, n);
          }
          set(f, g) {
            const n = this._entries.get(f);
            return this._entries.set(f, g), n;
          }
          forEach(f) {
            for (const [g, n] of this._entries.entries()) f(g, n);
          }
          has(f) {
            return this._entries.has(f);
          }
          get(f) {
            return this._entries.get(f);
          }
        }
        t.ServiceCollection = r, t.InstantiationService = class {
          constructor() {
            this._services = new r(), this._services.set(c.IInstantiationService, this);
          }
          setService(d, f) {
            this._services.set(d, f);
          }
          getService(d) {
            return this._services.get(d);
          }
          createInstance(d, ...f) {
            const g = (0, h.getServiceDependencies)(d).sort(((o, s) => o.index - s.index)), n = [];
            for (const o of g) {
              const s = this._services.get(o.id);
              if (!s) throw new Error(`[createInstance] ${d.name} depends on UNKNOWN service ${o.id}.`);
              n.push(s);
            }
            const e = g.length > 0 ? g[0].index : f.length;
            if (f.length !== e) throw new Error(`[createInstance] First service dependency of ${d.name} at position ${e + 1} conflicts with ${f.length} static arguments`);
            return new d(...f, ...n);
          }
        };
      }, 7866: function(T, t, a) {
        var c = this && this.__decorate || function(e, o, s, i) {
          var u, p = arguments.length, l = p < 3 ? o : i === null ? i = Object.getOwnPropertyDescriptor(o, s) : i;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") l = Reflect.decorate(e, o, s, i);
          else for (var m = e.length - 1; m >= 0; m--) (u = e[m]) && (l = (p < 3 ? u(l) : p > 3 ? u(o, s, l) : u(o, s)) || l);
          return p > 3 && l && Object.defineProperty(o, s, l), l;
        }, h = this && this.__param || function(e, o) {
          return function(s, i) {
            o(s, i, e);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.traceCall = t.setTraceLogger = t.LogService = void 0;
        const r = a(844), d = a(2585), f = { trace: d.LogLevelEnum.TRACE, debug: d.LogLevelEnum.DEBUG, info: d.LogLevelEnum.INFO, warn: d.LogLevelEnum.WARN, error: d.LogLevelEnum.ERROR, off: d.LogLevelEnum.OFF };
        let g, n = t.LogService = class extends r.Disposable {
          get logLevel() {
            return this._logLevel;
          }
          constructor(e) {
            super(), this._optionsService = e, this._logLevel = d.LogLevelEnum.OFF, this._updateLogLevel(), this.register(this._optionsService.onSpecificOptionChange("logLevel", (() => this._updateLogLevel()))), g = this;
          }
          _updateLogLevel() {
            this._logLevel = f[this._optionsService.rawOptions.logLevel];
          }
          _evalLazyOptionalParams(e) {
            for (let o = 0; o < e.length; o++) typeof e[o] == "function" && (e[o] = e[o]());
          }
          _log(e, o, s) {
            this._evalLazyOptionalParams(s), e.call(console, (this._optionsService.options.logger ? "" : "xterm.js: ") + o, ...s);
          }
          trace(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.TRACE && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.trace.bind(this._optionsService.options.logger)) != null ? i : console.log, e, o);
          }
          debug(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.DEBUG && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.debug.bind(this._optionsService.options.logger)) != null ? i : console.log, e, o);
          }
          info(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.INFO && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.info.bind(this._optionsService.options.logger)) != null ? i : console.info, e, o);
          }
          warn(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.WARN && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.warn.bind(this._optionsService.options.logger)) != null ? i : console.warn, e, o);
          }
          error(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.ERROR && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.error.bind(this._optionsService.options.logger)) != null ? i : console.error, e, o);
          }
        };
        t.LogService = n = c([h(0, d.IOptionsService)], n), t.setTraceLogger = function(e) {
          g = e;
        }, t.traceCall = function(e, o, s) {
          if (typeof s.value != "function") throw new Error("not supported");
          const i = s.value;
          s.value = function(...u) {
            if (g.logLevel !== d.LogLevelEnum.TRACE) return i.apply(this, u);
            g.trace(`GlyphRenderer#${i.name}(${u.map(((l) => JSON.stringify(l))).join(", ")})`);
            const p = i.apply(this, u);
            return g.trace(`GlyphRenderer#${i.name} return`, p), p;
          };
        };
      }, 7302: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.OptionsService = t.DEFAULT_OPTIONS = void 0;
        const c = a(8460), h = a(844), r = a(6114);
        t.DEFAULT_OPTIONS = { cols: 80, rows: 24, cursorBlink: !1, cursorStyle: "block", cursorWidth: 1, cursorInactiveStyle: "outline", customGlyphs: !0, drawBoldTextInBrightColors: !0, documentOverride: null, fastScrollModifier: "alt", fastScrollSensitivity: 5, fontFamily: "courier-new, courier, monospace", fontSize: 15, fontWeight: "normal", fontWeightBold: "bold", ignoreBracketedPasteMode: !1, lineHeight: 1, letterSpacing: 0, linkHandler: null, logLevel: "info", logger: null, scrollback: 1e3, scrollOnUserInput: !0, scrollSensitivity: 1, screenReaderMode: !1, smoothScrollDuration: 0, macOptionIsMeta: !1, macOptionClickForcesSelection: !1, minimumContrastRatio: 1, disableStdin: !1, allowProposedApi: !1, allowTransparency: !1, tabStopWidth: 8, theme: {}, rescaleOverlappingGlyphs: !1, rightClickSelectsWord: r.isMac, windowOptions: {}, windowsMode: !1, windowsPty: {}, wordSeparator: " ()[]{}',\"`", altClickMovesCursor: !0, convertEol: !1, termName: "xterm", cancelEvents: !1, overviewRulerWidth: 0 };
        const d = ["normal", "bold", "100", "200", "300", "400", "500", "600", "700", "800", "900"];
        class f extends h.Disposable {
          constructor(n) {
            super(), this._onOptionChange = this.register(new c.EventEmitter()), this.onOptionChange = this._onOptionChange.event;
            const e = { ...t.DEFAULT_OPTIONS };
            for (const o in n) if (o in e) try {
              const s = n[o];
              e[o] = this._sanitizeAndValidateOption(o, s);
            } catch (s) {
              console.error(s);
            }
            this.rawOptions = e, this.options = { ...e }, this._setupOptions(), this.register((0, h.toDisposable)((() => {
              this.rawOptions.linkHandler = null, this.rawOptions.documentOverride = null;
            })));
          }
          onSpecificOptionChange(n, e) {
            return this.onOptionChange(((o) => {
              o === n && e(this.rawOptions[n]);
            }));
          }
          onMultipleOptionChange(n, e) {
            return this.onOptionChange(((o) => {
              n.indexOf(o) !== -1 && e();
            }));
          }
          _setupOptions() {
            const n = (o) => {
              if (!(o in t.DEFAULT_OPTIONS)) throw new Error(`No option with key "${o}"`);
              return this.rawOptions[o];
            }, e = (o, s) => {
              if (!(o in t.DEFAULT_OPTIONS)) throw new Error(`No option with key "${o}"`);
              s = this._sanitizeAndValidateOption(o, s), this.rawOptions[o] !== s && (this.rawOptions[o] = s, this._onOptionChange.fire(o));
            };
            for (const o in this.rawOptions) {
              const s = { get: n.bind(this, o), set: e.bind(this, o) };
              Object.defineProperty(this.options, o, s);
            }
          }
          _sanitizeAndValidateOption(n, e) {
            switch (n) {
              case "cursorStyle":
                if (e || (e = t.DEFAULT_OPTIONS[n]), !/* @__PURE__ */ (function(o) {
                  return o === "block" || o === "underline" || o === "bar";
                })(e)) throw new Error(`"${e}" is not a valid value for ${n}`);
                break;
              case "wordSeparator":
                e || (e = t.DEFAULT_OPTIONS[n]);
                break;
              case "fontWeight":
              case "fontWeightBold":
                if (typeof e == "number" && 1 <= e && e <= 1e3) break;
                e = d.includes(e) ? e : t.DEFAULT_OPTIONS[n];
                break;
              case "cursorWidth":
                e = Math.floor(e);
              case "lineHeight":
              case "tabStopWidth":
                if (e < 1) throw new Error(`${n} cannot be less than 1, value: ${e}`);
                break;
              case "minimumContrastRatio":
                e = Math.max(1, Math.min(21, Math.round(10 * e) / 10));
                break;
              case "scrollback":
                if ((e = Math.min(e, 4294967295)) < 0) throw new Error(`${n} cannot be less than 0, value: ${e}`);
                break;
              case "fastScrollSensitivity":
              case "scrollSensitivity":
                if (e <= 0) throw new Error(`${n} cannot be less than or equal to 0, value: ${e}`);
                break;
              case "rows":
              case "cols":
                if (!e && e !== 0) throw new Error(`${n} must be numeric, value: ${e}`);
                break;
              case "windowsPty":
                e = e != null ? e : {};
            }
            return e;
          }
        }
        t.OptionsService = f;
      }, 2660: function(T, t, a) {
        var c = this && this.__decorate || function(f, g, n, e) {
          var o, s = arguments.length, i = s < 3 ? g : e === null ? e = Object.getOwnPropertyDescriptor(g, n) : e;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") i = Reflect.decorate(f, g, n, e);
          else for (var u = f.length - 1; u >= 0; u--) (o = f[u]) && (i = (s < 3 ? o(i) : s > 3 ? o(g, n, i) : o(g, n)) || i);
          return s > 3 && i && Object.defineProperty(g, n, i), i;
        }, h = this && this.__param || function(f, g) {
          return function(n, e) {
            g(n, e, f);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.OscLinkService = void 0;
        const r = a(2585);
        let d = t.OscLinkService = class {
          constructor(f) {
            this._bufferService = f, this._nextId = 1, this._entriesWithId = /* @__PURE__ */ new Map(), this._dataByLinkId = /* @__PURE__ */ new Map();
          }
          registerLink(f) {
            const g = this._bufferService.buffer;
            if (f.id === void 0) {
              const u = g.addMarker(g.ybase + g.y), p = { data: f, id: this._nextId++, lines: [u] };
              return u.onDispose((() => this._removeMarkerFromLink(p, u))), this._dataByLinkId.set(p.id, p), p.id;
            }
            const n = f, e = this._getEntryIdKey(n), o = this._entriesWithId.get(e);
            if (o) return this.addLineToLink(o.id, g.ybase + g.y), o.id;
            const s = g.addMarker(g.ybase + g.y), i = { id: this._nextId++, key: this._getEntryIdKey(n), data: n, lines: [s] };
            return s.onDispose((() => this._removeMarkerFromLink(i, s))), this._entriesWithId.set(i.key, i), this._dataByLinkId.set(i.id, i), i.id;
          }
          addLineToLink(f, g) {
            const n = this._dataByLinkId.get(f);
            if (n && n.lines.every(((e) => e.line !== g))) {
              const e = this._bufferService.buffer.addMarker(g);
              n.lines.push(e), e.onDispose((() => this._removeMarkerFromLink(n, e)));
            }
          }
          getLinkData(f) {
            var g;
            return (g = this._dataByLinkId.get(f)) == null ? void 0 : g.data;
          }
          _getEntryIdKey(f) {
            return `${f.id};;${f.uri}`;
          }
          _removeMarkerFromLink(f, g) {
            const n = f.lines.indexOf(g);
            n !== -1 && (f.lines.splice(n, 1), f.lines.length === 0 && (f.data.id !== void 0 && this._entriesWithId.delete(f.key), this._dataByLinkId.delete(f.id)));
          }
        };
        t.OscLinkService = d = c([h(0, r.IBufferService)], d);
      }, 8343: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.createDecorator = t.getServiceDependencies = t.serviceRegistry = void 0;
        const a = "di$target", c = "di$dependencies";
        t.serviceRegistry = /* @__PURE__ */ new Map(), t.getServiceDependencies = function(h) {
          return h[c] || [];
        }, t.createDecorator = function(h) {
          if (t.serviceRegistry.has(h)) return t.serviceRegistry.get(h);
          const r = function(d, f, g) {
            if (arguments.length !== 3) throw new Error("@IServiceName-decorator can only be used to decorate a parameter");
            (function(n, e, o) {
              e[a] === e ? e[c].push({ id: n, index: o }) : (e[c] = [{ id: n, index: o }], e[a] = e);
            })(r, d, g);
          };
          return r.toString = () => h, t.serviceRegistry.set(h, r), r;
        };
      }, 2585: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.IDecorationService = t.IUnicodeService = t.IOscLinkService = t.IOptionsService = t.ILogService = t.LogLevelEnum = t.IInstantiationService = t.ICharsetService = t.ICoreService = t.ICoreMouseService = t.IBufferService = void 0;
        const c = a(8343);
        var h;
        t.IBufferService = (0, c.createDecorator)("BufferService"), t.ICoreMouseService = (0, c.createDecorator)("CoreMouseService"), t.ICoreService = (0, c.createDecorator)("CoreService"), t.ICharsetService = (0, c.createDecorator)("CharsetService"), t.IInstantiationService = (0, c.createDecorator)("InstantiationService"), (function(r) {
          r[r.TRACE = 0] = "TRACE", r[r.DEBUG = 1] = "DEBUG", r[r.INFO = 2] = "INFO", r[r.WARN = 3] = "WARN", r[r.ERROR = 4] = "ERROR", r[r.OFF = 5] = "OFF";
        })(h || (t.LogLevelEnum = h = {})), t.ILogService = (0, c.createDecorator)("LogService"), t.IOptionsService = (0, c.createDecorator)("OptionsService"), t.IOscLinkService = (0, c.createDecorator)("OscLinkService"), t.IUnicodeService = (0, c.createDecorator)("UnicodeService"), t.IDecorationService = (0, c.createDecorator)("DecorationService");
      }, 1480: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.UnicodeService = void 0;
        const c = a(8460), h = a(225);
        class r {
          static extractShouldJoin(f) {
            return (1 & f) != 0;
          }
          static extractWidth(f) {
            return f >> 1 & 3;
          }
          static extractCharKind(f) {
            return f >> 3;
          }
          static createPropertyValue(f, g, n = !1) {
            return (16777215 & f) << 3 | (3 & g) << 1 | (n ? 1 : 0);
          }
          constructor() {
            this._providers = /* @__PURE__ */ Object.create(null), this._active = "", this._onChange = new c.EventEmitter(), this.onChange = this._onChange.event;
            const f = new h.UnicodeV6();
            this.register(f), this._active = f.version, this._activeProvider = f;
          }
          dispose() {
            this._onChange.dispose();
          }
          get versions() {
            return Object.keys(this._providers);
          }
          get activeVersion() {
            return this._active;
          }
          set activeVersion(f) {
            if (!this._providers[f]) throw new Error(`unknown Unicode version "${f}"`);
            this._active = f, this._activeProvider = this._providers[f], this._onChange.fire(f);
          }
          register(f) {
            this._providers[f.version] = f;
          }
          wcwidth(f) {
            return this._activeProvider.wcwidth(f);
          }
          getStringCellWidth(f) {
            let g = 0, n = 0;
            const e = f.length;
            for (let o = 0; o < e; ++o) {
              let s = f.charCodeAt(o);
              if (55296 <= s && s <= 56319) {
                if (++o >= e) return g + this.wcwidth(s);
                const p = f.charCodeAt(o);
                56320 <= p && p <= 57343 ? s = 1024 * (s - 55296) + p - 56320 + 65536 : g += this.wcwidth(p);
              }
              const i = this.charProperties(s, n);
              let u = r.extractWidth(i);
              r.extractShouldJoin(i) && (u -= r.extractWidth(n)), g += u, n = i;
            }
            return g;
          }
          charProperties(f, g) {
            return this._activeProvider.charProperties(f, g);
          }
        }
        t.UnicodeService = r;
      } }, $ = {};
      function W(T) {
        var t = $[T];
        if (t !== void 0) return t.exports;
        var a = $[T] = { exports: {} };
        return I[T].call(a.exports, a, a.exports, W), a.exports;
      }
      var Y = {};
      return (() => {
        var T = Y;
        Object.defineProperty(T, "__esModule", { value: !0 }), T.Terminal = void 0;
        const t = W(9042), a = W(3236), c = W(844), h = W(5741), r = W(8285), d = W(7975), f = W(7090), g = ["cols", "rows"];
        class n extends c.Disposable {
          constructor(o) {
            super(), this._core = this.register(new a.Terminal(o)), this._addonManager = this.register(new h.AddonManager()), this._publicOptions = { ...this._core.options };
            const s = (u) => this._core.options[u], i = (u, p) => {
              this._checkReadonlyOptions(u), this._core.options[u] = p;
            };
            for (const u in this._core.options) {
              const p = { get: s.bind(this, u), set: i.bind(this, u) };
              Object.defineProperty(this._publicOptions, u, p);
            }
          }
          _checkReadonlyOptions(o) {
            if (g.includes(o)) throw new Error(`Option "${o}" can only be set in the constructor`);
          }
          _checkProposedApi() {
            if (!this._core.optionsService.rawOptions.allowProposedApi) throw new Error("You must set the allowProposedApi option to true to use proposed API");
          }
          get onBell() {
            return this._core.onBell;
          }
          get onBinary() {
            return this._core.onBinary;
          }
          get onCursorMove() {
            return this._core.onCursorMove;
          }
          get onData() {
            return this._core.onData;
          }
          get onKey() {
            return this._core.onKey;
          }
          get onLineFeed() {
            return this._core.onLineFeed;
          }
          get onRender() {
            return this._core.onRender;
          }
          get onResize() {
            return this._core.onResize;
          }
          get onScroll() {
            return this._core.onScroll;
          }
          get onSelectionChange() {
            return this._core.onSelectionChange;
          }
          get onTitleChange() {
            return this._core.onTitleChange;
          }
          get onWriteParsed() {
            return this._core.onWriteParsed;
          }
          get element() {
            return this._core.element;
          }
          get parser() {
            return this._parser || (this._parser = new d.ParserApi(this._core)), this._parser;
          }
          get unicode() {
            return this._checkProposedApi(), new f.UnicodeApi(this._core);
          }
          get textarea() {
            return this._core.textarea;
          }
          get rows() {
            return this._core.rows;
          }
          get cols() {
            return this._core.cols;
          }
          get buffer() {
            return this._buffer || (this._buffer = this.register(new r.BufferNamespaceApi(this._core))), this._buffer;
          }
          get markers() {
            return this._checkProposedApi(), this._core.markers;
          }
          get modes() {
            const o = this._core.coreService.decPrivateModes;
            let s = "none";
            switch (this._core.coreMouseService.activeProtocol) {
              case "X10":
                s = "x10";
                break;
              case "VT200":
                s = "vt200";
                break;
              case "DRAG":
                s = "drag";
                break;
              case "ANY":
                s = "any";
            }
            return { applicationCursorKeysMode: o.applicationCursorKeys, applicationKeypadMode: o.applicationKeypad, bracketedPasteMode: o.bracketedPasteMode, insertMode: this._core.coreService.modes.insertMode, mouseTrackingMode: s, originMode: o.origin, reverseWraparoundMode: o.reverseWraparound, sendFocusMode: o.sendFocus, wraparoundMode: o.wraparound };
          }
          get options() {
            return this._publicOptions;
          }
          set options(o) {
            for (const s in o) this._publicOptions[s] = o[s];
          }
          blur() {
            this._core.blur();
          }
          focus() {
            this._core.focus();
          }
          input(o, s = !0) {
            this._core.input(o, s);
          }
          resize(o, s) {
            this._verifyIntegers(o, s), this._core.resize(o, s);
          }
          open(o) {
            this._core.open(o);
          }
          attachCustomKeyEventHandler(o) {
            this._core.attachCustomKeyEventHandler(o);
          }
          attachCustomWheelEventHandler(o) {
            this._core.attachCustomWheelEventHandler(o);
          }
          registerLinkProvider(o) {
            return this._core.registerLinkProvider(o);
          }
          registerCharacterJoiner(o) {
            return this._checkProposedApi(), this._core.registerCharacterJoiner(o);
          }
          deregisterCharacterJoiner(o) {
            this._checkProposedApi(), this._core.deregisterCharacterJoiner(o);
          }
          registerMarker(o = 0) {
            return this._verifyIntegers(o), this._core.registerMarker(o);
          }
          registerDecoration(o) {
            var s, i, u;
            return this._checkProposedApi(), this._verifyPositiveIntegers((s = o.x) != null ? s : 0, (i = o.width) != null ? i : 0, (u = o.height) != null ? u : 0), this._core.registerDecoration(o);
          }
          hasSelection() {
            return this._core.hasSelection();
          }
          select(o, s, i) {
            this._verifyIntegers(o, s, i), this._core.select(o, s, i);
          }
          getSelection() {
            return this._core.getSelection();
          }
          getSelectionPosition() {
            return this._core.getSelectionPosition();
          }
          clearSelection() {
            this._core.clearSelection();
          }
          selectAll() {
            this._core.selectAll();
          }
          selectLines(o, s) {
            this._verifyIntegers(o, s), this._core.selectLines(o, s);
          }
          dispose() {
            super.dispose();
          }
          scrollLines(o) {
            this._verifyIntegers(o), this._core.scrollLines(o);
          }
          scrollPages(o) {
            this._verifyIntegers(o), this._core.scrollPages(o);
          }
          scrollToTop() {
            this._core.scrollToTop();
          }
          scrollToBottom() {
            this._core.scrollToBottom();
          }
          scrollToLine(o) {
            this._verifyIntegers(o), this._core.scrollToLine(o);
          }
          clear() {
            this._core.clear();
          }
          write(o, s) {
            this._core.write(o, s);
          }
          writeln(o, s) {
            this._core.write(o), this._core.write(`\r
`, s);
          }
          paste(o) {
            this._core.paste(o);
          }
          refresh(o, s) {
            this._verifyIntegers(o, s), this._core.refresh(o, s);
          }
          reset() {
            this._core.reset();
          }
          clearTextureAtlas() {
            this._core.clearTextureAtlas();
          }
          loadAddon(o) {
            this._addonManager.loadAddon(this, o);
          }
          static get strings() {
            return t;
          }
          _verifyIntegers(...o) {
            for (const s of o) if (s === 1 / 0 || isNaN(s) || s % 1 != 0) throw new Error("This API only accepts integers");
          }
          _verifyPositiveIntegers(...o) {
            for (const s of o) if (s && (s === 1 / 0 || isNaN(s) || s % 1 != 0 || s < 0)) throw new Error("This API only accepts positive integers");
          }
        }
        T.Terminal = n;
      })(), Y;
    })()));
  })(Ae)), Ae.exports;
}
var Ye = Je(), De = { exports: {} }, We;
function Ze() {
  return We || (We = 1, (function(ne, B) {
    (function(I, $) {
      ne.exports = $();
    })(self, (() => (() => {
      var I = {};
      return (() => {
        var $ = I;
        Object.defineProperty($, "__esModule", { value: !0 }), $.FitAddon = void 0, $.FitAddon = class {
          activate(W) {
            this._terminal = W;
          }
          dispose() {
          }
          fit() {
            const W = this.proposeDimensions();
            if (!W || !this._terminal || isNaN(W.cols) || isNaN(W.rows)) return;
            const Y = this._terminal._core;
            this._terminal.rows === W.rows && this._terminal.cols === W.cols || (Y._renderService.clear(), this._terminal.resize(W.cols, W.rows));
          }
          proposeDimensions() {
            if (!this._terminal || !this._terminal.element || !this._terminal.element.parentElement) return;
            const W = this._terminal._core, Y = W._renderService.dimensions;
            if (Y.css.cell.width === 0 || Y.css.cell.height === 0) return;
            const T = this._terminal.options.scrollback === 0 ? 0 : W.viewport.scrollBarWidth, t = window.getComputedStyle(this._terminal.element.parentElement), a = parseInt(t.getPropertyValue("height")), c = Math.max(0, parseInt(t.getPropertyValue("width"))), h = window.getComputedStyle(this._terminal.element), r = a - (parseInt(h.getPropertyValue("padding-top")) + parseInt(h.getPropertyValue("padding-bottom"))), d = c - (parseInt(h.getPropertyValue("padding-right")) + parseInt(h.getPropertyValue("padding-left"))) - T;
            return { cols: Math.max(2, Math.floor(d / Y.css.cell.width)), rows: Math.max(1, Math.floor(r / Y.css.cell.height)) };
          }
        };
      })(), I;
    })()));
  })(De)), De.exports;
}
var Qe = Ze(), ke = { exports: {} }, Ue;
function et() {
  return Ue || (Ue = 1, (function(ne, B) {
    (function(I, $) {
      ne.exports = $();
    })(self, (() => (() => {
      var I = { 6: (T, t) => {
        function a(h) {
          try {
            const r = new URL(h), d = r.password && r.username ? `${r.protocol}//${r.username}:${r.password}@${r.host}` : r.username ? `${r.protocol}//${r.username}@${r.host}` : `${r.protocol}//${r.host}`;
            return h.toLocaleLowerCase().startsWith(d.toLocaleLowerCase());
          } catch (r) {
            return !1;
          }
        }
        Object.defineProperty(t, "__esModule", { value: !0 }), t.LinkComputer = t.WebLinkProvider = void 0, t.WebLinkProvider = class {
          constructor(h, r, d, f = {}) {
            this._terminal = h, this._regex = r, this._handler = d, this._options = f;
          }
          provideLinks(h, r) {
            const d = c.computeLink(h, this._regex, this._terminal, this._handler);
            r(this._addCallbacks(d));
          }
          _addCallbacks(h) {
            return h.map(((r) => (r.leave = this._options.leave, r.hover = (d, f) => {
              if (this._options.hover) {
                const { range: g } = r;
                this._options.hover(d, f, g);
              }
            }, r)));
          }
        };
        class c {
          static computeLink(r, d, f, g) {
            const n = new RegExp(d.source, (d.flags || "") + "g"), [e, o] = c._getWindowedLineStrings(r - 1, f), s = e.join("");
            let i;
            const u = [];
            for (; i = n.exec(s); ) {
              const p = i[0];
              if (!a(p)) continue;
              const [l, m] = c._mapStrIdx(f, o, 0, i.index), [_, v] = c._mapStrIdx(f, l, m, p.length);
              if (l === -1 || m === -1 || _ === -1 || v === -1) continue;
              const C = { start: { x: m + 1, y: l + 1 }, end: { x: v, y: _ + 1 } };
              u.push({ range: C, text: p, activate: g });
            }
            return u;
          }
          static _getWindowedLineStrings(r, d) {
            let f, g = r, n = r, e = 0, o = "";
            const s = [];
            if (f = d.buffer.active.getLine(r)) {
              const i = f.translateToString(!0);
              if (f.isWrapped && i[0] !== " ") {
                for (e = 0; (f = d.buffer.active.getLine(--g)) && e < 2048 && (o = f.translateToString(!0), e += o.length, s.push(o), f.isWrapped && o.indexOf(" ") === -1); ) ;
                s.reverse();
              }
              for (s.push(i), e = 0; (f = d.buffer.active.getLine(++n)) && f.isWrapped && e < 2048 && (o = f.translateToString(!0), e += o.length, s.push(o), o.indexOf(" ") === -1); ) ;
            }
            return [s, g];
          }
          static _mapStrIdx(r, d, f, g) {
            const n = r.buffer.active, e = n.getNullCell();
            let o = f;
            for (; g; ) {
              const s = n.getLine(d);
              if (!s) return [-1, -1];
              for (let i = o; i < s.length; ++i) {
                s.getCell(i, e);
                const u = e.getChars();
                if (e.getWidth() && (g -= u.length || 1, i === s.length - 1 && u === "")) {
                  const p = n.getLine(d + 1);
                  p && p.isWrapped && (p.getCell(0, e), e.getWidth() === 2 && (g += 1));
                }
                if (g < 0) return [d, i];
              }
              d++, o = 0;
            }
            return [d, o];
          }
        }
        t.LinkComputer = c;
      } }, $ = {};
      function W(T) {
        var t = $[T];
        if (t !== void 0) return t.exports;
        var a = $[T] = { exports: {} };
        return I[T](a, a.exports, W), a.exports;
      }
      var Y = {};
      return (() => {
        var T = Y;
        Object.defineProperty(T, "__esModule", { value: !0 }), T.WebLinksAddon = void 0;
        const t = W(6), a = /(https?|HTTPS?):[/]{2}[^\s"'!*(){}|\\\^<>`]*[^\s"':,.!?{}|\\\^~\[\]`()<>]/;
        function c(h, r) {
          const d = window.open();
          if (d) {
            try {
              d.opener = null;
            } catch (f) {
            }
            d.location.href = r;
          } else console.warn("Opening link blocked as opener could not be cleared");
        }
        T.WebLinksAddon = class {
          constructor(h = c, r = {}) {
            this._handler = h, this._options = r;
          }
          activate(h) {
            this._terminal = h;
            const r = this._options, d = r.urlRegex || a;
            this._linkProvider = this._terminal.registerLinkProvider(new t.WebLinkProvider(this._terminal, d, this._handler, r));
          }
          dispose() {
            var h;
            (h = this._linkProvider) == null || h.dispose();
          }
        };
      })(), Y;
    })()));
  })(ke)), ke.exports;
}
var tt = et(), xe = { exports: {} }, it = xe.exports, Ne;
function st() {
  return Ne || (Ne = 1, (function(ne, B) {
    (function(I, $) {
      ne.exports = $();
    })(it, (() => (() => {
      var I = { 433: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.UnicodeV11 = void 0;
        const c = a(938), h = [[768, 879], [1155, 1161], [1425, 1469], [1471, 1471], [1473, 1474], [1476, 1477], [1479, 1479], [1536, 1541], [1552, 1562], [1564, 1564], [1611, 1631], [1648, 1648], [1750, 1757], [1759, 1764], [1767, 1768], [1770, 1773], [1807, 1807], [1809, 1809], [1840, 1866], [1958, 1968], [2027, 2035], [2045, 2045], [2070, 2073], [2075, 2083], [2085, 2087], [2089, 2093], [2137, 2139], [2259, 2306], [2362, 2362], [2364, 2364], [2369, 2376], [2381, 2381], [2385, 2391], [2402, 2403], [2433, 2433], [2492, 2492], [2497, 2500], [2509, 2509], [2530, 2531], [2558, 2558], [2561, 2562], [2620, 2620], [2625, 2626], [2631, 2632], [2635, 2637], [2641, 2641], [2672, 2673], [2677, 2677], [2689, 2690], [2748, 2748], [2753, 2757], [2759, 2760], [2765, 2765], [2786, 2787], [2810, 2815], [2817, 2817], [2876, 2876], [2879, 2879], [2881, 2884], [2893, 2893], [2902, 2902], [2914, 2915], [2946, 2946], [3008, 3008], [3021, 3021], [3072, 3072], [3076, 3076], [3134, 3136], [3142, 3144], [3146, 3149], [3157, 3158], [3170, 3171], [3201, 3201], [3260, 3260], [3263, 3263], [3270, 3270], [3276, 3277], [3298, 3299], [3328, 3329], [3387, 3388], [3393, 3396], [3405, 3405], [3426, 3427], [3530, 3530], [3538, 3540], [3542, 3542], [3633, 3633], [3636, 3642], [3655, 3662], [3761, 3761], [3764, 3772], [3784, 3789], [3864, 3865], [3893, 3893], [3895, 3895], [3897, 3897], [3953, 3966], [3968, 3972], [3974, 3975], [3981, 3991], [3993, 4028], [4038, 4038], [4141, 4144], [4146, 4151], [4153, 4154], [4157, 4158], [4184, 4185], [4190, 4192], [4209, 4212], [4226, 4226], [4229, 4230], [4237, 4237], [4253, 4253], [4448, 4607], [4957, 4959], [5906, 5908], [5938, 5940], [5970, 5971], [6002, 6003], [6068, 6069], [6071, 6077], [6086, 6086], [6089, 6099], [6109, 6109], [6155, 6158], [6277, 6278], [6313, 6313], [6432, 6434], [6439, 6440], [6450, 6450], [6457, 6459], [6679, 6680], [6683, 6683], [6742, 6742], [6744, 6750], [6752, 6752], [6754, 6754], [6757, 6764], [6771, 6780], [6783, 6783], [6832, 6846], [6912, 6915], [6964, 6964], [6966, 6970], [6972, 6972], [6978, 6978], [7019, 7027], [7040, 7041], [7074, 7077], [7080, 7081], [7083, 7085], [7142, 7142], [7144, 7145], [7149, 7149], [7151, 7153], [7212, 7219], [7222, 7223], [7376, 7378], [7380, 7392], [7394, 7400], [7405, 7405], [7412, 7412], [7416, 7417], [7616, 7673], [7675, 7679], [8203, 8207], [8234, 8238], [8288, 8292], [8294, 8303], [8400, 8432], [11503, 11505], [11647, 11647], [11744, 11775], [12330, 12333], [12441, 12442], [42607, 42610], [42612, 42621], [42654, 42655], [42736, 42737], [43010, 43010], [43014, 43014], [43019, 43019], [43045, 43046], [43204, 43205], [43232, 43249], [43263, 43263], [43302, 43309], [43335, 43345], [43392, 43394], [43443, 43443], [43446, 43449], [43452, 43453], [43493, 43493], [43561, 43566], [43569, 43570], [43573, 43574], [43587, 43587], [43596, 43596], [43644, 43644], [43696, 43696], [43698, 43700], [43703, 43704], [43710, 43711], [43713, 43713], [43756, 43757], [43766, 43766], [44005, 44005], [44008, 44008], [44013, 44013], [64286, 64286], [65024, 65039], [65056, 65071], [65279, 65279], [65529, 65531]], r = [[66045, 66045], [66272, 66272], [66422, 66426], [68097, 68099], [68101, 68102], [68108, 68111], [68152, 68154], [68159, 68159], [68325, 68326], [68900, 68903], [69446, 69456], [69633, 69633], [69688, 69702], [69759, 69761], [69811, 69814], [69817, 69818], [69821, 69821], [69837, 69837], [69888, 69890], [69927, 69931], [69933, 69940], [70003, 70003], [70016, 70017], [70070, 70078], [70089, 70092], [70191, 70193], [70196, 70196], [70198, 70199], [70206, 70206], [70367, 70367], [70371, 70378], [70400, 70401], [70459, 70460], [70464, 70464], [70502, 70508], [70512, 70516], [70712, 70719], [70722, 70724], [70726, 70726], [70750, 70750], [70835, 70840], [70842, 70842], [70847, 70848], [70850, 70851], [71090, 71093], [71100, 71101], [71103, 71104], [71132, 71133], [71219, 71226], [71229, 71229], [71231, 71232], [71339, 71339], [71341, 71341], [71344, 71349], [71351, 71351], [71453, 71455], [71458, 71461], [71463, 71467], [71727, 71735], [71737, 71738], [72148, 72151], [72154, 72155], [72160, 72160], [72193, 72202], [72243, 72248], [72251, 72254], [72263, 72263], [72273, 72278], [72281, 72283], [72330, 72342], [72344, 72345], [72752, 72758], [72760, 72765], [72767, 72767], [72850, 72871], [72874, 72880], [72882, 72883], [72885, 72886], [73009, 73014], [73018, 73018], [73020, 73021], [73023, 73029], [73031, 73031], [73104, 73105], [73109, 73109], [73111, 73111], [73459, 73460], [78896, 78904], [92912, 92916], [92976, 92982], [94031, 94031], [94095, 94098], [113821, 113822], [113824, 113827], [119143, 119145], [119155, 119170], [119173, 119179], [119210, 119213], [119362, 119364], [121344, 121398], [121403, 121452], [121461, 121461], [121476, 121476], [121499, 121503], [121505, 121519], [122880, 122886], [122888, 122904], [122907, 122913], [122915, 122916], [122918, 122922], [123184, 123190], [123628, 123631], [125136, 125142], [125252, 125258], [917505, 917505], [917536, 917631], [917760, 917999]], d = [[4352, 4447], [8986, 8987], [9001, 9002], [9193, 9196], [9200, 9200], [9203, 9203], [9725, 9726], [9748, 9749], [9800, 9811], [9855, 9855], [9875, 9875], [9889, 9889], [9898, 9899], [9917, 9918], [9924, 9925], [9934, 9934], [9940, 9940], [9962, 9962], [9970, 9971], [9973, 9973], [9978, 9978], [9981, 9981], [9989, 9989], [9994, 9995], [10024, 10024], [10060, 10060], [10062, 10062], [10067, 10069], [10071, 10071], [10133, 10135], [10160, 10160], [10175, 10175], [11035, 11036], [11088, 11088], [11093, 11093], [11904, 11929], [11931, 12019], [12032, 12245], [12272, 12283], [12288, 12329], [12334, 12350], [12353, 12438], [12443, 12543], [12549, 12591], [12593, 12686], [12688, 12730], [12736, 12771], [12784, 12830], [12832, 12871], [12880, 19903], [19968, 42124], [42128, 42182], [43360, 43388], [44032, 55203], [63744, 64255], [65040, 65049], [65072, 65106], [65108, 65126], [65128, 65131], [65281, 65376], [65504, 65510]], f = [[94176, 94179], [94208, 100343], [100352, 101106], [110592, 110878], [110928, 110930], [110948, 110951], [110960, 111355], [126980, 126980], [127183, 127183], [127374, 127374], [127377, 127386], [127488, 127490], [127504, 127547], [127552, 127560], [127568, 127569], [127584, 127589], [127744, 127776], [127789, 127797], [127799, 127868], [127870, 127891], [127904, 127946], [127951, 127955], [127968, 127984], [127988, 127988], [127992, 128062], [128064, 128064], [128066, 128252], [128255, 128317], [128331, 128334], [128336, 128359], [128378, 128378], [128405, 128406], [128420, 128420], [128507, 128591], [128640, 128709], [128716, 128716], [128720, 128722], [128725, 128725], [128747, 128748], [128756, 128762], [128992, 129003], [129293, 129393], [129395, 129398], [129402, 129442], [129445, 129450], [129454, 129482], [129485, 129535], [129648, 129651], [129656, 129658], [129664, 129666], [129680, 129685], [131072, 196605], [196608, 262141]];
        let g;
        function n(e, o) {
          let s, i = 0, u = o.length - 1;
          if (e < o[0][0] || e > o[u][1]) return !1;
          for (; u >= i; ) if (s = i + u >> 1, e > o[s][1]) i = s + 1;
          else {
            if (!(e < o[s][0])) return !0;
            u = s - 1;
          }
          return !1;
        }
        t.UnicodeV11 = class {
          constructor() {
            if (this.version = "11", !g) {
              g = new Uint8Array(65536), g.fill(1), g[0] = 0, g.fill(0, 1, 32), g.fill(0, 127, 160);
              for (let e = 0; e < h.length; ++e) g.fill(0, h[e][0], h[e][1] + 1);
              for (let e = 0; e < d.length; ++e) g.fill(2, d[e][0], d[e][1] + 1);
            }
          }
          wcwidth(e) {
            return e < 32 ? 0 : e < 127 ? 1 : e < 65536 ? g[e] : n(e, r) ? 0 : n(e, f) ? 2 : 1;
          }
          charProperties(e, o) {
            let s = this.wcwidth(e), i = s === 0 && o !== 0;
            if (i) {
              const u = c.UnicodeService.extractWidth(o);
              u === 0 ? i = !1 : u > s && (s = u);
            }
            return c.UnicodeService.createPropertyValue(0, s, i);
          }
        };
      }, 345: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.runAndSubscribe = t.forwardEvent = t.EventEmitter = void 0, t.EventEmitter = class {
          constructor() {
            this._listeners = [], this._disposed = !1;
          }
          get event() {
            return this._event || (this._event = (a) => (this._listeners.push(a), { dispose: () => {
              if (!this._disposed) {
                for (let c = 0; c < this._listeners.length; c++) if (this._listeners[c] === a) return void this._listeners.splice(c, 1);
              }
            } })), this._event;
          }
          fire(a, c) {
            const h = [];
            for (let r = 0; r < this._listeners.length; r++) h.push(this._listeners[r]);
            for (let r = 0; r < h.length; r++) h[r].call(void 0, a, c);
          }
          dispose() {
            this.clearListeners(), this._disposed = !0;
          }
          clearListeners() {
            this._listeners && (this._listeners.length = 0);
          }
        }, t.forwardEvent = function(a, c) {
          return a(((h) => c.fire(h)));
        }, t.runAndSubscribe = function(a, c) {
          return c(void 0), a(((h) => c(h)));
        };
      }, 490: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.UnicodeV6 = void 0;
        const c = a(938), h = [[768, 879], [1155, 1158], [1160, 1161], [1425, 1469], [1471, 1471], [1473, 1474], [1476, 1477], [1479, 1479], [1536, 1539], [1552, 1557], [1611, 1630], [1648, 1648], [1750, 1764], [1767, 1768], [1770, 1773], [1807, 1807], [1809, 1809], [1840, 1866], [1958, 1968], [2027, 2035], [2305, 2306], [2364, 2364], [2369, 2376], [2381, 2381], [2385, 2388], [2402, 2403], [2433, 2433], [2492, 2492], [2497, 2500], [2509, 2509], [2530, 2531], [2561, 2562], [2620, 2620], [2625, 2626], [2631, 2632], [2635, 2637], [2672, 2673], [2689, 2690], [2748, 2748], [2753, 2757], [2759, 2760], [2765, 2765], [2786, 2787], [2817, 2817], [2876, 2876], [2879, 2879], [2881, 2883], [2893, 2893], [2902, 2902], [2946, 2946], [3008, 3008], [3021, 3021], [3134, 3136], [3142, 3144], [3146, 3149], [3157, 3158], [3260, 3260], [3263, 3263], [3270, 3270], [3276, 3277], [3298, 3299], [3393, 3395], [3405, 3405], [3530, 3530], [3538, 3540], [3542, 3542], [3633, 3633], [3636, 3642], [3655, 3662], [3761, 3761], [3764, 3769], [3771, 3772], [3784, 3789], [3864, 3865], [3893, 3893], [3895, 3895], [3897, 3897], [3953, 3966], [3968, 3972], [3974, 3975], [3984, 3991], [3993, 4028], [4038, 4038], [4141, 4144], [4146, 4146], [4150, 4151], [4153, 4153], [4184, 4185], [4448, 4607], [4959, 4959], [5906, 5908], [5938, 5940], [5970, 5971], [6002, 6003], [6068, 6069], [6071, 6077], [6086, 6086], [6089, 6099], [6109, 6109], [6155, 6157], [6313, 6313], [6432, 6434], [6439, 6440], [6450, 6450], [6457, 6459], [6679, 6680], [6912, 6915], [6964, 6964], [6966, 6970], [6972, 6972], [6978, 6978], [7019, 7027], [7616, 7626], [7678, 7679], [8203, 8207], [8234, 8238], [8288, 8291], [8298, 8303], [8400, 8431], [12330, 12335], [12441, 12442], [43014, 43014], [43019, 43019], [43045, 43046], [64286, 64286], [65024, 65039], [65056, 65059], [65279, 65279], [65529, 65531]], r = [[68097, 68099], [68101, 68102], [68108, 68111], [68152, 68154], [68159, 68159], [119143, 119145], [119155, 119170], [119173, 119179], [119210, 119213], [119362, 119364], [917505, 917505], [917536, 917631], [917760, 917999]];
        let d;
        t.UnicodeV6 = class {
          constructor() {
            if (this.version = "6", !d) {
              d = new Uint8Array(65536), d.fill(1), d[0] = 0, d.fill(0, 1, 32), d.fill(0, 127, 160), d.fill(2, 4352, 4448), d[9001] = 2, d[9002] = 2, d.fill(2, 11904, 42192), d[12351] = 1, d.fill(2, 44032, 55204), d.fill(2, 63744, 64256), d.fill(2, 65040, 65050), d.fill(2, 65072, 65136), d.fill(2, 65280, 65377), d.fill(2, 65504, 65511);
              for (let f = 0; f < h.length; ++f) d.fill(0, h[f][0], h[f][1] + 1);
            }
          }
          wcwidth(f) {
            return f < 32 ? 0 : f < 127 ? 1 : f < 65536 ? d[f] : (function(g, n) {
              let e, o = 0, s = n.length - 1;
              if (g < n[0][0] || g > n[s][1]) return !1;
              for (; s >= o; ) if (e = o + s >> 1, g > n[e][1]) o = e + 1;
              else {
                if (!(g < n[e][0])) return !0;
                s = e - 1;
              }
              return !1;
            })(f, r) ? 0 : f >= 131072 && f <= 196605 || f >= 196608 && f <= 262141 ? 2 : 1;
          }
          charProperties(f, g) {
            let n = this.wcwidth(f), e = n === 0 && g !== 0;
            if (e) {
              const o = c.UnicodeService.extractWidth(g);
              o === 0 ? e = !1 : o > n && (n = o);
            }
            return c.UnicodeService.createPropertyValue(0, n, e);
          }
        };
      }, 938: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.UnicodeService = void 0;
        const c = a(345), h = a(490);
        class r {
          static extractShouldJoin(f) {
            return (1 & f) != 0;
          }
          static extractWidth(f) {
            return f >> 1 & 3;
          }
          static extractCharKind(f) {
            return f >> 3;
          }
          static createPropertyValue(f, g, n = !1) {
            return (16777215 & f) << 3 | (3 & g) << 1 | (n ? 1 : 0);
          }
          constructor() {
            this._providers = /* @__PURE__ */ Object.create(null), this._active = "", this._onChange = new c.EventEmitter(), this.onChange = this._onChange.event;
            const f = new h.UnicodeV6();
            this.register(f), this._active = f.version, this._activeProvider = f;
          }
          dispose() {
            this._onChange.dispose();
          }
          get versions() {
            return Object.keys(this._providers);
          }
          get activeVersion() {
            return this._active;
          }
          set activeVersion(f) {
            if (!this._providers[f]) throw new Error(`unknown Unicode version "${f}"`);
            this._active = f, this._activeProvider = this._providers[f], this._onChange.fire(f);
          }
          register(f) {
            this._providers[f.version] = f;
          }
          wcwidth(f) {
            return this._activeProvider.wcwidth(f);
          }
          getStringCellWidth(f) {
            let g = 0, n = 0;
            const e = f.length;
            for (let o = 0; o < e; ++o) {
              let s = f.charCodeAt(o);
              if (55296 <= s && s <= 56319) {
                if (++o >= e) return g + this.wcwidth(s);
                const p = f.charCodeAt(o);
                56320 <= p && p <= 57343 ? s = 1024 * (s - 55296) + p - 56320 + 65536 : g += this.wcwidth(p);
              }
              const i = this.charProperties(s, n);
              let u = r.extractWidth(i);
              r.extractShouldJoin(i) && (u -= r.extractWidth(n)), g += u, n = i;
            }
            return g;
          }
          charProperties(f, g) {
            return this._activeProvider.charProperties(f, g);
          }
        }
        t.UnicodeService = r;
      } }, $ = {};
      function W(T) {
        var t = $[T];
        if (t !== void 0) return t.exports;
        var a = $[T] = { exports: {} };
        return I[T](a, a.exports, W), a.exports;
      }
      var Y = {};
      return (() => {
        var T = Y;
        Object.defineProperty(T, "__esModule", { value: !0 }), T.Unicode11Addon = void 0;
        const t = W(433);
        T.Unicode11Addon = class {
          activate(a) {
            a.unicode.register(new t.UnicodeV11());
          }
          dispose() {
          }
        };
      })(), Y;
    })()));
  })(xe)), xe.exports;
}
var rt = st(), Te = { exports: {} }, ze;
function nt() {
  return ze || (ze = 1, (function(ne, B) {
    (function(I, $) {
      ne.exports = $();
    })(self, (() => (() => {
      var I = { 965: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.GlyphRenderer = void 0;
        const c = a(374), h = a(509), r = a(855), d = a(859), f = a(381), g = 11, n = g * Float32Array.BYTES_PER_ELEMENT;
        let e, o = 0, s = 0, i = 0;
        class u extends d.Disposable {
          constructor(l, m, _, v) {
            super(), this._terminal = l, this._gl = m, this._dimensions = _, this._optionsService = v, this._activeBuffer = 0, this._vertices = { count: 0, attributes: new Float32Array(0), attributesBuffers: [new Float32Array(0), new Float32Array(0)] };
            const C = this._gl;
            h.TextureAtlas.maxAtlasPages === void 0 && (h.TextureAtlas.maxAtlasPages = Math.min(32, (0, c.throwIfFalsy)(C.getParameter(C.MAX_TEXTURE_IMAGE_UNITS))), h.TextureAtlas.maxTextureSize = (0, c.throwIfFalsy)(C.getParameter(C.MAX_TEXTURE_SIZE))), this._program = (0, c.throwIfFalsy)((0, f.createProgram)(C, `#version 300 es
layout (location = 0) in vec2 a_unitquad;
layout (location = 1) in vec2 a_cellpos;
layout (location = 2) in vec2 a_offset;
layout (location = 3) in vec2 a_size;
layout (location = 4) in float a_texpage;
layout (location = 5) in vec2 a_texcoord;
layout (location = 6) in vec2 a_texsize;

uniform mat4 u_projection;
uniform vec2 u_resolution;

out vec2 v_texcoord;
flat out int v_texpage;

void main() {
  vec2 zeroToOne = (a_offset / u_resolution) + a_cellpos + (a_unitquad * a_size);
  gl_Position = u_projection * vec4(zeroToOne, 0.0, 1.0);
  v_texpage = int(a_texpage);
  v_texcoord = a_texcoord + a_unitquad * a_texsize;
}`, (function(P) {
              let k = "";
              for (let M = 1; M < P; M++) k += ` else if (v_texpage == ${M}) { outColor = texture(u_texture[${M}], v_texcoord); }`;
              return `#version 300 es
precision lowp float;

in vec2 v_texcoord;
flat in int v_texpage;

uniform sampler2D u_texture[${P}];

out vec4 outColor;

void main() {
  if (v_texpage == 0) {
    outColor = texture(u_texture[0], v_texcoord);
  } ${k}
}`;
            })(h.TextureAtlas.maxAtlasPages))), this.register((0, d.toDisposable)((() => C.deleteProgram(this._program)))), this._projectionLocation = (0, c.throwIfFalsy)(C.getUniformLocation(this._program, "u_projection")), this._resolutionLocation = (0, c.throwIfFalsy)(C.getUniformLocation(this._program, "u_resolution")), this._textureLocation = (0, c.throwIfFalsy)(C.getUniformLocation(this._program, "u_texture")), this._vertexArrayObject = C.createVertexArray(), C.bindVertexArray(this._vertexArrayObject);
            const w = new Float32Array([0, 0, 1, 0, 0, 1, 1, 1]), S = C.createBuffer();
            this.register((0, d.toDisposable)((() => C.deleteBuffer(S)))), C.bindBuffer(C.ARRAY_BUFFER, S), C.bufferData(C.ARRAY_BUFFER, w, C.STATIC_DRAW), C.enableVertexAttribArray(0), C.vertexAttribPointer(0, 2, this._gl.FLOAT, !1, 0, 0);
            const b = new Uint8Array([0, 1, 2, 3]), x = C.createBuffer();
            this.register((0, d.toDisposable)((() => C.deleteBuffer(x)))), C.bindBuffer(C.ELEMENT_ARRAY_BUFFER, x), C.bufferData(C.ELEMENT_ARRAY_BUFFER, b, C.STATIC_DRAW), this._attributesBuffer = (0, c.throwIfFalsy)(C.createBuffer()), this.register((0, d.toDisposable)((() => C.deleteBuffer(this._attributesBuffer)))), C.bindBuffer(C.ARRAY_BUFFER, this._attributesBuffer), C.enableVertexAttribArray(2), C.vertexAttribPointer(2, 2, C.FLOAT, !1, n, 0), C.vertexAttribDivisor(2, 1), C.enableVertexAttribArray(3), C.vertexAttribPointer(3, 2, C.FLOAT, !1, n, 2 * Float32Array.BYTES_PER_ELEMENT), C.vertexAttribDivisor(3, 1), C.enableVertexAttribArray(4), C.vertexAttribPointer(4, 1, C.FLOAT, !1, n, 4 * Float32Array.BYTES_PER_ELEMENT), C.vertexAttribDivisor(4, 1), C.enableVertexAttribArray(5), C.vertexAttribPointer(5, 2, C.FLOAT, !1, n, 5 * Float32Array.BYTES_PER_ELEMENT), C.vertexAttribDivisor(5, 1), C.enableVertexAttribArray(6), C.vertexAttribPointer(6, 2, C.FLOAT, !1, n, 7 * Float32Array.BYTES_PER_ELEMENT), C.vertexAttribDivisor(6, 1), C.enableVertexAttribArray(1), C.vertexAttribPointer(1, 2, C.FLOAT, !1, n, 9 * Float32Array.BYTES_PER_ELEMENT), C.vertexAttribDivisor(1, 1), C.useProgram(this._program);
            const A = new Int32Array(h.TextureAtlas.maxAtlasPages);
            for (let P = 0; P < h.TextureAtlas.maxAtlasPages; P++) A[P] = P;
            C.uniform1iv(this._textureLocation, A), C.uniformMatrix4fv(this._projectionLocation, !1, f.PROJECTION_MATRIX), this._atlasTextures = [];
            for (let P = 0; P < h.TextureAtlas.maxAtlasPages; P++) {
              const k = new f.GLTexture((0, c.throwIfFalsy)(C.createTexture()));
              this.register((0, d.toDisposable)((() => C.deleteTexture(k.texture)))), C.activeTexture(C.TEXTURE0 + P), C.bindTexture(C.TEXTURE_2D, k.texture), C.texParameteri(C.TEXTURE_2D, C.TEXTURE_WRAP_S, C.CLAMP_TO_EDGE), C.texParameteri(C.TEXTURE_2D, C.TEXTURE_WRAP_T, C.CLAMP_TO_EDGE), C.texImage2D(C.TEXTURE_2D, 0, C.RGBA, 1, 1, 0, C.RGBA, C.UNSIGNED_BYTE, new Uint8Array([255, 0, 0, 255])), this._atlasTextures[P] = k;
            }
            C.enable(C.BLEND), C.blendFunc(C.SRC_ALPHA, C.ONE_MINUS_SRC_ALPHA), this.handleResize();
          }
          beginFrame() {
            return !this._atlas || this._atlas.beginFrame();
          }
          updateCell(l, m, _, v, C, w, S, b, x) {
            this._updateCell(this._vertices.attributes, l, m, _, v, C, w, S, b, x);
          }
          _updateCell(l, m, _, v, C, w, S, b, x, A) {
            o = (_ * this._terminal.cols + m) * g, v !== r.NULL_CELL_CODE && v !== void 0 ? this._atlas && (e = b && b.length > 1 ? this._atlas.getRasterizedGlyphCombinedChar(b, C, w, S, !1) : this._atlas.getRasterizedGlyph(v, C, w, S, !1), s = Math.floor((this._dimensions.device.cell.width - this._dimensions.device.char.width) / 2), C !== A && e.offset.x > s ? (i = e.offset.x - s, l[o] = -(e.offset.x - i) + this._dimensions.device.char.left, l[o + 1] = -e.offset.y + this._dimensions.device.char.top, l[o + 2] = (e.size.x - i) / this._dimensions.device.canvas.width, l[o + 3] = e.size.y / this._dimensions.device.canvas.height, l[o + 4] = e.texturePage, l[o + 5] = e.texturePositionClipSpace.x + i / this._atlas.pages[e.texturePage].canvas.width, l[o + 6] = e.texturePositionClipSpace.y, l[o + 7] = e.sizeClipSpace.x - i / this._atlas.pages[e.texturePage].canvas.width, l[o + 8] = e.sizeClipSpace.y) : (l[o] = -e.offset.x + this._dimensions.device.char.left, l[o + 1] = -e.offset.y + this._dimensions.device.char.top, l[o + 2] = e.size.x / this._dimensions.device.canvas.width, l[o + 3] = e.size.y / this._dimensions.device.canvas.height, l[o + 4] = e.texturePage, l[o + 5] = e.texturePositionClipSpace.x, l[o + 6] = e.texturePositionClipSpace.y, l[o + 7] = e.sizeClipSpace.x, l[o + 8] = e.sizeClipSpace.y), this._optionsService.rawOptions.rescaleOverlappingGlyphs && (0, c.allowRescaling)(v, x, e.size.x, this._dimensions.device.cell.width) && (l[o + 2] = (this._dimensions.device.cell.width - 1) / this._dimensions.device.canvas.width)) : l.fill(0, o, o + g - 1 - 2);
          }
          clear() {
            const l = this._terminal, m = l.cols * l.rows * g;
            this._vertices.count !== m ? this._vertices.attributes = new Float32Array(m) : this._vertices.attributes.fill(0);
            let _ = 0;
            for (; _ < this._vertices.attributesBuffers.length; _++) this._vertices.count !== m ? this._vertices.attributesBuffers[_] = new Float32Array(m) : this._vertices.attributesBuffers[_].fill(0);
            this._vertices.count = m, _ = 0;
            for (let v = 0; v < l.rows; v++) for (let C = 0; C < l.cols; C++) this._vertices.attributes[_ + 9] = C / l.cols, this._vertices.attributes[_ + 10] = v / l.rows, _ += g;
          }
          handleResize() {
            const l = this._gl;
            l.useProgram(this._program), l.viewport(0, 0, l.canvas.width, l.canvas.height), l.uniform2f(this._resolutionLocation, l.canvas.width, l.canvas.height), this.clear();
          }
          render(l) {
            if (!this._atlas) return;
            const m = this._gl;
            m.useProgram(this._program), m.bindVertexArray(this._vertexArrayObject), this._activeBuffer = (this._activeBuffer + 1) % 2;
            const _ = this._vertices.attributesBuffers[this._activeBuffer];
            let v = 0;
            for (let C = 0; C < l.lineLengths.length; C++) {
              const w = C * this._terminal.cols * g, S = this._vertices.attributes.subarray(w, w + l.lineLengths[C] * g);
              _.set(S, v), v += S.length;
            }
            m.bindBuffer(m.ARRAY_BUFFER, this._attributesBuffer), m.bufferData(m.ARRAY_BUFFER, _.subarray(0, v), m.STREAM_DRAW);
            for (let C = 0; C < this._atlas.pages.length; C++) this._atlas.pages[C].version !== this._atlasTextures[C].version && this._bindAtlasPageTexture(m, this._atlas, C);
            m.drawElementsInstanced(m.TRIANGLE_STRIP, 4, m.UNSIGNED_BYTE, 0, v / g);
          }
          setAtlas(l) {
            this._atlas = l;
            for (const m of this._atlasTextures) m.version = -1;
          }
          _bindAtlasPageTexture(l, m, _) {
            l.activeTexture(l.TEXTURE0 + _), l.bindTexture(l.TEXTURE_2D, this._atlasTextures[_].texture), l.texParameteri(l.TEXTURE_2D, l.TEXTURE_WRAP_S, l.CLAMP_TO_EDGE), l.texParameteri(l.TEXTURE_2D, l.TEXTURE_WRAP_T, l.CLAMP_TO_EDGE), l.texImage2D(l.TEXTURE_2D, 0, l.RGBA, l.RGBA, l.UNSIGNED_BYTE, m.pages[_].canvas), l.generateMipmap(l.TEXTURE_2D), this._atlasTextures[_].version = m.pages[_].version;
          }
          setDimensions(l) {
            this._dimensions = l;
          }
        }
        t.GlyphRenderer = u;
      }, 742: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.RectangleRenderer = void 0;
        const c = a(374), h = a(859), r = a(310), d = a(381), f = 8 * Float32Array.BYTES_PER_ELEMENT;
        class g {
          constructor() {
            this.attributes = new Float32Array(160), this.count = 0;
          }
        }
        let n = 0, e = 0, o = 0, s = 0, i = 0, u = 0, p = 0;
        class l extends h.Disposable {
          constructor(_, v, C, w) {
            super(), this._terminal = _, this._gl = v, this._dimensions = C, this._themeService = w, this._vertices = new g(), this._verticesCursor = new g();
            const S = this._gl;
            this._program = (0, c.throwIfFalsy)((0, d.createProgram)(S, `#version 300 es
layout (location = 0) in vec2 a_position;
layout (location = 1) in vec2 a_size;
layout (location = 2) in vec4 a_color;
layout (location = 3) in vec2 a_unitquad;

uniform mat4 u_projection;

out vec4 v_color;

void main() {
  vec2 zeroToOne = a_position + (a_unitquad * a_size);
  gl_Position = u_projection * vec4(zeroToOne, 0.0, 1.0);
  v_color = a_color;
}`, `#version 300 es
precision lowp float;

in vec4 v_color;

out vec4 outColor;

void main() {
  outColor = v_color;
}`)), this.register((0, h.toDisposable)((() => S.deleteProgram(this._program)))), this._projectionLocation = (0, c.throwIfFalsy)(S.getUniformLocation(this._program, "u_projection")), this._vertexArrayObject = S.createVertexArray(), S.bindVertexArray(this._vertexArrayObject);
            const b = new Float32Array([0, 0, 1, 0, 0, 1, 1, 1]), x = S.createBuffer();
            this.register((0, h.toDisposable)((() => S.deleteBuffer(x)))), S.bindBuffer(S.ARRAY_BUFFER, x), S.bufferData(S.ARRAY_BUFFER, b, S.STATIC_DRAW), S.enableVertexAttribArray(3), S.vertexAttribPointer(3, 2, this._gl.FLOAT, !1, 0, 0);
            const A = new Uint8Array([0, 1, 2, 3]), P = S.createBuffer();
            this.register((0, h.toDisposable)((() => S.deleteBuffer(P)))), S.bindBuffer(S.ELEMENT_ARRAY_BUFFER, P), S.bufferData(S.ELEMENT_ARRAY_BUFFER, A, S.STATIC_DRAW), this._attributesBuffer = (0, c.throwIfFalsy)(S.createBuffer()), this.register((0, h.toDisposable)((() => S.deleteBuffer(this._attributesBuffer)))), S.bindBuffer(S.ARRAY_BUFFER, this._attributesBuffer), S.enableVertexAttribArray(0), S.vertexAttribPointer(0, 2, S.FLOAT, !1, f, 0), S.vertexAttribDivisor(0, 1), S.enableVertexAttribArray(1), S.vertexAttribPointer(1, 2, S.FLOAT, !1, f, 2 * Float32Array.BYTES_PER_ELEMENT), S.vertexAttribDivisor(1, 1), S.enableVertexAttribArray(2), S.vertexAttribPointer(2, 4, S.FLOAT, !1, f, 4 * Float32Array.BYTES_PER_ELEMENT), S.vertexAttribDivisor(2, 1), this._updateCachedColors(w.colors), this.register(this._themeService.onChangeColors(((k) => {
              this._updateCachedColors(k), this._updateViewportRectangle();
            })));
          }
          renderBackgrounds() {
            this._renderVertices(this._vertices);
          }
          renderCursor() {
            this._renderVertices(this._verticesCursor);
          }
          _renderVertices(_) {
            const v = this._gl;
            v.useProgram(this._program), v.bindVertexArray(this._vertexArrayObject), v.uniformMatrix4fv(this._projectionLocation, !1, d.PROJECTION_MATRIX), v.bindBuffer(v.ARRAY_BUFFER, this._attributesBuffer), v.bufferData(v.ARRAY_BUFFER, _.attributes, v.DYNAMIC_DRAW), v.drawElementsInstanced(this._gl.TRIANGLE_STRIP, 4, v.UNSIGNED_BYTE, 0, _.count);
          }
          handleResize() {
            this._updateViewportRectangle();
          }
          setDimensions(_) {
            this._dimensions = _;
          }
          _updateCachedColors(_) {
            this._bgFloat = this._colorToFloat32Array(_.background), this._cursorFloat = this._colorToFloat32Array(_.cursor);
          }
          _updateViewportRectangle() {
            this._addRectangleFloat(this._vertices.attributes, 0, 0, 0, this._terminal.cols * this._dimensions.device.cell.width, this._terminal.rows * this._dimensions.device.cell.height, this._bgFloat);
          }
          updateBackgrounds(_) {
            const v = this._terminal, C = this._vertices;
            let w, S, b, x, A, P, k, M, y, L, R, D = 1;
            for (w = 0; w < v.rows; w++) {
              for (b = -1, x = 0, A = 0, P = !1, S = 0; S < v.cols; S++) k = (w * v.cols + S) * r.RENDER_MODEL_INDICIES_PER_CELL, M = _.cells[k + r.RENDER_MODEL_BG_OFFSET], y = _.cells[k + r.RENDER_MODEL_FG_OFFSET], L = !!(67108864 & y), (M !== x || y !== A && (P || L)) && ((x !== 0 || P && A !== 0) && (R = 8 * D++, this._updateRectangle(C, R, A, x, b, S, w)), b = S, x = M, A = y, P = L);
              (x !== 0 || P && A !== 0) && (R = 8 * D++, this._updateRectangle(C, R, A, x, b, v.cols, w));
            }
            C.count = D;
          }
          updateCursor(_) {
            const v = this._verticesCursor, C = _.cursor;
            if (!C || C.style === "block") return void (v.count = 0);
            let w, S = 0;
            C.style !== "bar" && C.style !== "outline" || (w = 8 * S++, this._addRectangleFloat(v.attributes, w, C.x * this._dimensions.device.cell.width, C.y * this._dimensions.device.cell.height, C.style === "bar" ? C.dpr * C.cursorWidth : C.dpr, this._dimensions.device.cell.height, this._cursorFloat)), C.style !== "underline" && C.style !== "outline" || (w = 8 * S++, this._addRectangleFloat(v.attributes, w, C.x * this._dimensions.device.cell.width, (C.y + 1) * this._dimensions.device.cell.height - C.dpr, C.width * this._dimensions.device.cell.width, C.dpr, this._cursorFloat)), C.style === "outline" && (w = 8 * S++, this._addRectangleFloat(v.attributes, w, C.x * this._dimensions.device.cell.width, C.y * this._dimensions.device.cell.height, C.width * this._dimensions.device.cell.width, C.dpr, this._cursorFloat), w = 8 * S++, this._addRectangleFloat(v.attributes, w, (C.x + C.width) * this._dimensions.device.cell.width - C.dpr, C.y * this._dimensions.device.cell.height, C.dpr, this._dimensions.device.cell.height, this._cursorFloat)), v.count = S;
          }
          _updateRectangle(_, v, C, w, S, b, x) {
            if (67108864 & C) switch (50331648 & C) {
              case 16777216:
              case 33554432:
                n = this._themeService.colors.ansi[255 & C].rgba;
                break;
              case 50331648:
                n = (16777215 & C) << 8;
                break;
              default:
                n = this._themeService.colors.foreground.rgba;
            }
            else switch (50331648 & w) {
              case 16777216:
              case 33554432:
                n = this._themeService.colors.ansi[255 & w].rgba;
                break;
              case 50331648:
                n = (16777215 & w) << 8;
                break;
              default:
                n = this._themeService.colors.background.rgba;
            }
            _.attributes.length < v + 4 && (_.attributes = (0, d.expandFloat32Array)(_.attributes, this._terminal.rows * this._terminal.cols * 8)), e = S * this._dimensions.device.cell.width, o = x * this._dimensions.device.cell.height, s = (n >> 24 & 255) / 255, i = (n >> 16 & 255) / 255, u = (n >> 8 & 255) / 255, p = 1, this._addRectangle(_.attributes, v, e, o, (b - S) * this._dimensions.device.cell.width, this._dimensions.device.cell.height, s, i, u, p);
          }
          _addRectangle(_, v, C, w, S, b, x, A, P, k) {
            _[v] = C / this._dimensions.device.canvas.width, _[v + 1] = w / this._dimensions.device.canvas.height, _[v + 2] = S / this._dimensions.device.canvas.width, _[v + 3] = b / this._dimensions.device.canvas.height, _[v + 4] = x, _[v + 5] = A, _[v + 6] = P, _[v + 7] = k;
          }
          _addRectangleFloat(_, v, C, w, S, b, x) {
            _[v] = C / this._dimensions.device.canvas.width, _[v + 1] = w / this._dimensions.device.canvas.height, _[v + 2] = S / this._dimensions.device.canvas.width, _[v + 3] = b / this._dimensions.device.canvas.height, _[v + 4] = x[0], _[v + 5] = x[1], _[v + 6] = x[2], _[v + 7] = x[3];
          }
          _colorToFloat32Array(_) {
            return new Float32Array([(_.rgba >> 24 & 255) / 255, (_.rgba >> 16 & 255) / 255, (_.rgba >> 8 & 255) / 255, (255 & _.rgba) / 255]);
          }
        }
        t.RectangleRenderer = l;
      }, 310: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.RenderModel = t.COMBINED_CHAR_BIT_MASK = t.RENDER_MODEL_EXT_OFFSET = t.RENDER_MODEL_FG_OFFSET = t.RENDER_MODEL_BG_OFFSET = t.RENDER_MODEL_INDICIES_PER_CELL = void 0;
        const c = a(296);
        t.RENDER_MODEL_INDICIES_PER_CELL = 4, t.RENDER_MODEL_BG_OFFSET = 1, t.RENDER_MODEL_FG_OFFSET = 2, t.RENDER_MODEL_EXT_OFFSET = 3, t.COMBINED_CHAR_BIT_MASK = 2147483648, t.RenderModel = class {
          constructor() {
            this.cells = new Uint32Array(0), this.lineLengths = new Uint32Array(0), this.selection = (0, c.createSelectionRenderModel)();
          }
          resize(h, r) {
            const d = h * r * t.RENDER_MODEL_INDICIES_PER_CELL;
            d !== this.cells.length && (this.cells = new Uint32Array(d), this.lineLengths = new Uint32Array(r));
          }
          clear() {
            this.cells.fill(0, 0), this.lineLengths.fill(0, 0);
          }
        };
      }, 666: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.JoinedCellData = t.WebglRenderer = void 0;
        const c = a(820), h = a(274), r = a(627), d = a(457), f = a(56), g = a(374), n = a(345), e = a(859), o = a(147), s = a(782), i = a(855), u = a(965), p = a(742), l = a(310), m = a(733);
        class _ extends e.Disposable {
          constructor(S, b, x, A, P, k, M, y, L) {
            super(), this._terminal = S, this._characterJoinerService = b, this._charSizeService = x, this._coreBrowserService = A, this._coreService = P, this._decorationService = k, this._optionsService = M, this._themeService = y, this._cursorBlinkStateManager = new e.MutableDisposable(), this._charAtlasDisposable = this.register(new e.MutableDisposable()), this._observerDisposable = this.register(new e.MutableDisposable()), this._model = new l.RenderModel(), this._workCell = new s.CellData(), this._workCell2 = new s.CellData(), this._rectangleRenderer = this.register(new e.MutableDisposable()), this._glyphRenderer = this.register(new e.MutableDisposable()), this._onChangeTextureAtlas = this.register(new n.EventEmitter()), this.onChangeTextureAtlas = this._onChangeTextureAtlas.event, this._onAddTextureAtlasCanvas = this.register(new n.EventEmitter()), this.onAddTextureAtlasCanvas = this._onAddTextureAtlasCanvas.event, this._onRemoveTextureAtlasCanvas = this.register(new n.EventEmitter()), this.onRemoveTextureAtlasCanvas = this._onRemoveTextureAtlasCanvas.event, this._onRequestRedraw = this.register(new n.EventEmitter()), this.onRequestRedraw = this._onRequestRedraw.event, this._onContextLoss = this.register(new n.EventEmitter()), this.onContextLoss = this._onContextLoss.event, this.register(this._themeService.onChangeColors((() => this._handleColorChange()))), this._cellColorResolver = new h.CellColorResolver(this._terminal, this._optionsService, this._model.selection, this._decorationService, this._coreBrowserService, this._themeService), this._core = this._terminal._core, this._renderLayers = [new m.LinkRenderLayer(this._core.screenElement, 2, this._terminal, this._core.linkifier, this._coreBrowserService, M, this._themeService)], this.dimensions = (0, g.createRenderDimensions)(), this._devicePixelRatio = this._coreBrowserService.dpr, this._updateDimensions(), this._updateCursorBlink(), this.register(M.onOptionChange((() => this._handleOptionsChanged()))), this._canvas = this._coreBrowserService.mainDocument.createElement("canvas");
            const R = { antialias: !1, depth: !1, preserveDrawingBuffer: L };
            if (this._gl = this._canvas.getContext("webgl2", R), !this._gl) throw new Error("WebGL2 not supported " + this._gl);
            this.register((0, c.addDisposableDomListener)(this._canvas, "webglcontextlost", ((D) => {
              console.log("webglcontextlost event received"), D.preventDefault(), this._contextRestorationTimeout = setTimeout((() => {
                this._contextRestorationTimeout = void 0, console.warn("webgl context not restored; firing onContextLoss"), this._onContextLoss.fire(D);
              }), 3e3);
            }))), this.register((0, c.addDisposableDomListener)(this._canvas, "webglcontextrestored", ((D) => {
              console.warn("webglcontextrestored event received"), clearTimeout(this._contextRestorationTimeout), this._contextRestorationTimeout = void 0, (0, r.removeTerminalFromCache)(this._terminal), this._initializeWebGLState(), this._requestRedrawViewport();
            }))), this._observerDisposable.value = (0, f.observeDevicePixelDimensions)(this._canvas, this._coreBrowserService.window, ((D, F) => this._setCanvasDevicePixelDimensions(D, F))), this.register(this._coreBrowserService.onWindowChange(((D) => {
              this._observerDisposable.value = (0, f.observeDevicePixelDimensions)(this._canvas, D, ((F, U) => this._setCanvasDevicePixelDimensions(F, U)));
            }))), this._core.screenElement.appendChild(this._canvas), [this._rectangleRenderer.value, this._glyphRenderer.value] = this._initializeWebGLState(), this._isAttached = this._coreBrowserService.window.document.body.contains(this._core.screenElement), this.register((0, e.toDisposable)((() => {
              var D;
              for (const F of this._renderLayers) F.dispose();
              (D = this._canvas.parentElement) == null || D.removeChild(this._canvas), (0, r.removeTerminalFromCache)(this._terminal);
            })));
          }
          get textureAtlas() {
            var S;
            return (S = this._charAtlas) == null ? void 0 : S.pages[0].canvas;
          }
          _handleColorChange() {
            this._refreshCharAtlas(), this._clearModel(!0);
          }
          handleDevicePixelRatioChange() {
            this._devicePixelRatio !== this._coreBrowserService.dpr && (this._devicePixelRatio = this._coreBrowserService.dpr, this.handleResize(this._terminal.cols, this._terminal.rows));
          }
          handleResize(S, b) {
            var x, A, P, k;
            this._updateDimensions(), this._model.resize(this._terminal.cols, this._terminal.rows);
            for (const M of this._renderLayers) M.resize(this._terminal, this.dimensions);
            this._canvas.width = this.dimensions.device.canvas.width, this._canvas.height = this.dimensions.device.canvas.height, this._canvas.style.width = `${this.dimensions.css.canvas.width}px`, this._canvas.style.height = `${this.dimensions.css.canvas.height}px`, this._core.screenElement.style.width = `${this.dimensions.css.canvas.width}px`, this._core.screenElement.style.height = `${this.dimensions.css.canvas.height}px`, (x = this._rectangleRenderer.value) == null || x.setDimensions(this.dimensions), (A = this._rectangleRenderer.value) == null || A.handleResize(), (P = this._glyphRenderer.value) == null || P.setDimensions(this.dimensions), (k = this._glyphRenderer.value) == null || k.handleResize(), this._refreshCharAtlas(), this._clearModel(!1);
          }
          handleCharSizeChanged() {
            this.handleResize(this._terminal.cols, this._terminal.rows);
          }
          handleBlur() {
            var S;
            for (const b of this._renderLayers) b.handleBlur(this._terminal);
            (S = this._cursorBlinkStateManager.value) == null || S.pause(), this._requestRedrawViewport();
          }
          handleFocus() {
            var S;
            for (const b of this._renderLayers) b.handleFocus(this._terminal);
            (S = this._cursorBlinkStateManager.value) == null || S.resume(), this._requestRedrawViewport();
          }
          handleSelectionChanged(S, b, x) {
            for (const A of this._renderLayers) A.handleSelectionChanged(this._terminal, S, b, x);
            this._model.selection.update(this._core, S, b, x), this._requestRedrawViewport();
          }
          handleCursorMove() {
            var S;
            for (const b of this._renderLayers) b.handleCursorMove(this._terminal);
            (S = this._cursorBlinkStateManager.value) == null || S.restartBlinkAnimation();
          }
          _handleOptionsChanged() {
            this._updateDimensions(), this._refreshCharAtlas(), this._updateCursorBlink();
          }
          _initializeWebGLState() {
            return this._rectangleRenderer.value = new p.RectangleRenderer(this._terminal, this._gl, this.dimensions, this._themeService), this._glyphRenderer.value = new u.GlyphRenderer(this._terminal, this._gl, this.dimensions, this._optionsService), this.handleCharSizeChanged(), [this._rectangleRenderer.value, this._glyphRenderer.value];
          }
          _refreshCharAtlas() {
            var b;
            if (this.dimensions.device.char.width <= 0 && this.dimensions.device.char.height <= 0) return void (this._isAttached = !1);
            const S = (0, r.acquireTextureAtlas)(this._terminal, this._optionsService.rawOptions, this._themeService.colors, this.dimensions.device.cell.width, this.dimensions.device.cell.height, this.dimensions.device.char.width, this.dimensions.device.char.height, this._coreBrowserService.dpr);
            this._charAtlas !== S && (this._onChangeTextureAtlas.fire(S.pages[0].canvas), this._charAtlasDisposable.value = (0, e.getDisposeArrayDisposable)([(0, n.forwardEvent)(S.onAddTextureAtlasCanvas, this._onAddTextureAtlasCanvas), (0, n.forwardEvent)(S.onRemoveTextureAtlasCanvas, this._onRemoveTextureAtlasCanvas)])), this._charAtlas = S, this._charAtlas.warmUp(), (b = this._glyphRenderer.value) == null || b.setAtlas(this._charAtlas);
          }
          _clearModel(S) {
            var b;
            this._model.clear(), S && ((b = this._glyphRenderer.value) == null || b.clear());
          }
          clearTextureAtlas() {
            var S;
            (S = this._charAtlas) == null || S.clearTexture(), this._clearModel(!0), this._requestRedrawViewport();
          }
          clear() {
            var S;
            this._clearModel(!0);
            for (const b of this._renderLayers) b.reset(this._terminal);
            (S = this._cursorBlinkStateManager.value) == null || S.restartBlinkAnimation(), this._updateCursorBlink();
          }
          registerCharacterJoiner(S) {
            return -1;
          }
          deregisterCharacterJoiner(S) {
            return !1;
          }
          renderRows(S, b) {
            if (!this._isAttached) {
              if (!(this._coreBrowserService.window.document.body.contains(this._core.screenElement) && this._charSizeService.width && this._charSizeService.height)) return;
              this._updateDimensions(), this._refreshCharAtlas(), this._isAttached = !0;
            }
            for (const x of this._renderLayers) x.handleGridChanged(this._terminal, S, b);
            this._glyphRenderer.value && this._rectangleRenderer.value && (this._glyphRenderer.value.beginFrame() ? (this._clearModel(!0), this._updateModel(0, this._terminal.rows - 1)) : this._updateModel(S, b), this._rectangleRenderer.value.renderBackgrounds(), this._glyphRenderer.value.render(this._model), this._cursorBlinkStateManager.value && !this._cursorBlinkStateManager.value.isCursorVisible || this._rectangleRenderer.value.renderCursor());
          }
          _updateCursorBlink() {
            this._terminal.options.cursorBlink ? this._cursorBlinkStateManager.value = new d.CursorBlinkStateManager((() => {
              this._requestRedrawCursor();
            }), this._coreBrowserService) : this._cursorBlinkStateManager.clear(), this._requestRedrawCursor();
          }
          _updateModel(S, b) {
            const x = this._core;
            let A, P, k, M, y, L, R, D, F, U, K, q, O, E, H = this._workCell;
            S = C(S, x.rows - 1, 0), b = C(b, x.rows - 1, 0);
            const N = this._terminal.buffer.active.baseY + this._terminal.buffer.active.cursorY, G = N - x.buffer.ydisp, j = Math.min(this._terminal.buffer.active.cursorX, x.cols - 1);
            let ie = -1;
            const V = this._coreService.isCursorInitialized && !this._coreService.isCursorHidden && (!this._cursorBlinkStateManager.value || this._cursorBlinkStateManager.value.isCursorVisible);
            this._model.cursor = void 0;
            let ae = !1;
            for (P = S; P <= b; P++) for (k = P + x.buffer.ydisp, M = x.buffer.lines.get(k), this._model.lineLengths[P] = 0, y = this._characterJoinerService.getJoinedCharacters(k), O = 0; O < x.cols; O++) if (A = this._cellColorResolver.result.bg, M.loadCell(O, H), O === 0 && (A = this._cellColorResolver.result.bg), L = !1, R = O, y.length > 0 && O === y[0][0] && (L = !0, D = y.shift(), H = new v(H, M.translateToString(!0, D[0], D[1]), D[1] - D[0]), R = D[1] - 1), F = H.getChars(), U = H.getCode(), q = (P * x.cols + O) * l.RENDER_MODEL_INDICIES_PER_CELL, this._cellColorResolver.resolve(H, O, k, this.dimensions.device.cell.width), V && k === N && (O === j && (this._model.cursor = { x: j, y: G, width: H.getWidth(), style: this._coreBrowserService.isFocused ? x.options.cursorStyle || "block" : x.options.cursorInactiveStyle, cursorWidth: x.options.cursorWidth, dpr: this._devicePixelRatio }, ie = j + H.getWidth() - 1), O >= j && O <= ie && (this._coreBrowserService.isFocused && (x.options.cursorStyle || "block") === "block" || this._coreBrowserService.isFocused === !1 && x.options.cursorInactiveStyle === "block") && (this._cellColorResolver.result.fg = 50331648 | this._themeService.colors.cursorAccent.rgba >> 8 & 16777215, this._cellColorResolver.result.bg = 50331648 | this._themeService.colors.cursor.rgba >> 8 & 16777215)), U !== i.NULL_CELL_CODE && (this._model.lineLengths[P] = O + 1), (this._model.cells[q] !== U || this._model.cells[q + l.RENDER_MODEL_BG_OFFSET] !== this._cellColorResolver.result.bg || this._model.cells[q + l.RENDER_MODEL_FG_OFFSET] !== this._cellColorResolver.result.fg || this._model.cells[q + l.RENDER_MODEL_EXT_OFFSET] !== this._cellColorResolver.result.ext) && (ae = !0, F.length > 1 && (U |= l.COMBINED_CHAR_BIT_MASK), this._model.cells[q] = U, this._model.cells[q + l.RENDER_MODEL_BG_OFFSET] = this._cellColorResolver.result.bg, this._model.cells[q + l.RENDER_MODEL_FG_OFFSET] = this._cellColorResolver.result.fg, this._model.cells[q + l.RENDER_MODEL_EXT_OFFSET] = this._cellColorResolver.result.ext, K = H.getWidth(), this._glyphRenderer.value.updateCell(O, P, U, this._cellColorResolver.result.bg, this._cellColorResolver.result.fg, this._cellColorResolver.result.ext, F, K, A), L)) for (H = this._workCell, O++; O < R; O++) E = (P * x.cols + O) * l.RENDER_MODEL_INDICIES_PER_CELL, this._glyphRenderer.value.updateCell(O, P, i.NULL_CELL_CODE, 0, 0, 0, i.NULL_CELL_CHAR, 0, 0), this._model.cells[E] = i.NULL_CELL_CODE, this._model.cells[E + l.RENDER_MODEL_BG_OFFSET] = this._cellColorResolver.result.bg, this._model.cells[E + l.RENDER_MODEL_FG_OFFSET] = this._cellColorResolver.result.fg, this._model.cells[E + l.RENDER_MODEL_EXT_OFFSET] = this._cellColorResolver.result.ext;
            ae && this._rectangleRenderer.value.updateBackgrounds(this._model), this._rectangleRenderer.value.updateCursor(this._model);
          }
          _updateDimensions() {
            this._charSizeService.width && this._charSizeService.height && (this.dimensions.device.char.width = Math.floor(this._charSizeService.width * this._devicePixelRatio), this.dimensions.device.char.height = Math.ceil(this._charSizeService.height * this._devicePixelRatio), this.dimensions.device.cell.height = Math.floor(this.dimensions.device.char.height * this._optionsService.rawOptions.lineHeight), this.dimensions.device.char.top = this._optionsService.rawOptions.lineHeight === 1 ? 0 : Math.round((this.dimensions.device.cell.height - this.dimensions.device.char.height) / 2), this.dimensions.device.cell.width = this.dimensions.device.char.width + Math.round(this._optionsService.rawOptions.letterSpacing), this.dimensions.device.char.left = Math.floor(this._optionsService.rawOptions.letterSpacing / 2), this.dimensions.device.canvas.height = this._terminal.rows * this.dimensions.device.cell.height, this.dimensions.device.canvas.width = this._terminal.cols * this.dimensions.device.cell.width, this.dimensions.css.canvas.height = Math.round(this.dimensions.device.canvas.height / this._devicePixelRatio), this.dimensions.css.canvas.width = Math.round(this.dimensions.device.canvas.width / this._devicePixelRatio), this.dimensions.css.cell.height = this.dimensions.device.cell.height / this._devicePixelRatio, this.dimensions.css.cell.width = this.dimensions.device.cell.width / this._devicePixelRatio);
          }
          _setCanvasDevicePixelDimensions(S, b) {
            this._canvas.width === S && this._canvas.height === b || (this._canvas.width = S, this._canvas.height = b, this._requestRedrawViewport());
          }
          _requestRedrawViewport() {
            this._onRequestRedraw.fire({ start: 0, end: this._terminal.rows - 1 });
          }
          _requestRedrawCursor() {
            const S = this._terminal.buffer.active.cursorY;
            this._onRequestRedraw.fire({ start: S, end: S });
          }
        }
        t.WebglRenderer = _;
        class v extends o.AttributeData {
          constructor(S, b, x) {
            super(), this.content = 0, this.combinedData = "", this.fg = S.fg, this.bg = S.bg, this.combinedData = b, this._width = x;
          }
          isCombined() {
            return 2097152;
          }
          getWidth() {
            return this._width;
          }
          getChars() {
            return this.combinedData;
          }
          getCode() {
            return 2097151;
          }
          setFromCharData(S) {
            throw new Error("not implemented");
          }
          getAsCharData() {
            return [this.fg, this.getChars(), this.getWidth(), this.getCode()];
          }
        }
        function C(w, S, b = 0) {
          return Math.max(Math.min(w, S), b);
        }
        t.JoinedCellData = v;
      }, 381: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.GLTexture = t.expandFloat32Array = t.createShader = t.createProgram = t.PROJECTION_MATRIX = void 0;
        const c = a(374);
        function h(r, d, f) {
          const g = (0, c.throwIfFalsy)(r.createShader(d));
          if (r.shaderSource(g, f), r.compileShader(g), r.getShaderParameter(g, r.COMPILE_STATUS)) return g;
          console.error(r.getShaderInfoLog(g)), r.deleteShader(g);
        }
        t.PROJECTION_MATRIX = new Float32Array([2, 0, 0, 0, 0, -2, 0, 0, 0, 0, 1, 0, -1, 1, 0, 1]), t.createProgram = function(r, d, f) {
          const g = (0, c.throwIfFalsy)(r.createProgram());
          if (r.attachShader(g, (0, c.throwIfFalsy)(h(r, r.VERTEX_SHADER, d))), r.attachShader(g, (0, c.throwIfFalsy)(h(r, r.FRAGMENT_SHADER, f))), r.linkProgram(g), r.getProgramParameter(g, r.LINK_STATUS)) return g;
          console.error(r.getProgramInfoLog(g)), r.deleteProgram(g);
        }, t.createShader = h, t.expandFloat32Array = function(r, d) {
          const f = Math.min(2 * r.length, d), g = new Float32Array(f);
          for (let n = 0; n < r.length; n++) g[n] = r[n];
          return g;
        }, t.GLTexture = class {
          constructor(r) {
            this.texture = r, this.version = -1;
          }
        };
      }, 592: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.BaseRenderLayer = void 0;
        const c = a(627), h = a(237), r = a(374), d = a(859);
        class f extends d.Disposable {
          constructor(n, e, o, s, i, u, p, l) {
            super(), this._container = e, this._alpha = i, this._coreBrowserService = u, this._optionsService = p, this._themeService = l, this._deviceCharWidth = 0, this._deviceCharHeight = 0, this._deviceCellWidth = 0, this._deviceCellHeight = 0, this._deviceCharLeft = 0, this._deviceCharTop = 0, this._canvas = this._coreBrowserService.mainDocument.createElement("canvas"), this._canvas.classList.add(`xterm-${o}-layer`), this._canvas.style.zIndex = s.toString(), this._initCanvas(), this._container.appendChild(this._canvas), this.register(this._themeService.onChangeColors(((m) => {
              this._refreshCharAtlas(n, m), this.reset(n);
            }))), this.register((0, d.toDisposable)((() => {
              this._canvas.remove();
            })));
          }
          _initCanvas() {
            this._ctx = (0, r.throwIfFalsy)(this._canvas.getContext("2d", { alpha: this._alpha })), this._alpha || this._clearAll();
          }
          handleBlur(n) {
          }
          handleFocus(n) {
          }
          handleCursorMove(n) {
          }
          handleGridChanged(n, e, o) {
          }
          handleSelectionChanged(n, e, o, s = !1) {
          }
          _setTransparency(n, e) {
            if (e === this._alpha) return;
            const o = this._canvas;
            this._alpha = e, this._canvas = this._canvas.cloneNode(), this._initCanvas(), this._container.replaceChild(this._canvas, o), this._refreshCharAtlas(n, this._themeService.colors), this.handleGridChanged(n, 0, n.rows - 1);
          }
          _refreshCharAtlas(n, e) {
            this._deviceCharWidth <= 0 && this._deviceCharHeight <= 0 || (this._charAtlas = (0, c.acquireTextureAtlas)(n, this._optionsService.rawOptions, e, this._deviceCellWidth, this._deviceCellHeight, this._deviceCharWidth, this._deviceCharHeight, this._coreBrowserService.dpr), this._charAtlas.warmUp());
          }
          resize(n, e) {
            this._deviceCellWidth = e.device.cell.width, this._deviceCellHeight = e.device.cell.height, this._deviceCharWidth = e.device.char.width, this._deviceCharHeight = e.device.char.height, this._deviceCharLeft = e.device.char.left, this._deviceCharTop = e.device.char.top, this._canvas.width = e.device.canvas.width, this._canvas.height = e.device.canvas.height, this._canvas.style.width = `${e.css.canvas.width}px`, this._canvas.style.height = `${e.css.canvas.height}px`, this._alpha || this._clearAll(), this._refreshCharAtlas(n, this._themeService.colors);
          }
          _fillBottomLineAtCells(n, e, o = 1) {
            this._ctx.fillRect(n * this._deviceCellWidth, (e + 1) * this._deviceCellHeight - this._coreBrowserService.dpr - 1, o * this._deviceCellWidth, this._coreBrowserService.dpr);
          }
          _clearAll() {
            this._alpha ? this._ctx.clearRect(0, 0, this._canvas.width, this._canvas.height) : (this._ctx.fillStyle = this._themeService.colors.background.css, this._ctx.fillRect(0, 0, this._canvas.width, this._canvas.height));
          }
          _clearCells(n, e, o, s) {
            this._alpha ? this._ctx.clearRect(n * this._deviceCellWidth, e * this._deviceCellHeight, o * this._deviceCellWidth, s * this._deviceCellHeight) : (this._ctx.fillStyle = this._themeService.colors.background.css, this._ctx.fillRect(n * this._deviceCellWidth, e * this._deviceCellHeight, o * this._deviceCellWidth, s * this._deviceCellHeight));
          }
          _fillCharTrueColor(n, e, o, s) {
            this._ctx.font = this._getFont(n, !1, !1), this._ctx.textBaseline = h.TEXT_BASELINE, this._clipCell(o, s, e.getWidth()), this._ctx.fillText(e.getChars(), o * this._deviceCellWidth + this._deviceCharLeft, s * this._deviceCellHeight + this._deviceCharTop + this._deviceCharHeight);
          }
          _clipCell(n, e, o) {
            this._ctx.beginPath(), this._ctx.rect(n * this._deviceCellWidth, e * this._deviceCellHeight, o * this._deviceCellWidth, this._deviceCellHeight), this._ctx.clip();
          }
          _getFont(n, e, o) {
            return `${o ? "italic" : ""} ${e ? n.options.fontWeightBold : n.options.fontWeight} ${n.options.fontSize * this._coreBrowserService.dpr}px ${n.options.fontFamily}`;
          }
        }
        t.BaseRenderLayer = f;
      }, 733: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.LinkRenderLayer = void 0;
        const c = a(197), h = a(237), r = a(592);
        class d extends r.BaseRenderLayer {
          constructor(g, n, e, o, s, i, u) {
            super(e, g, "link", n, !0, s, i, u), this.register(o.onShowLinkUnderline(((p) => this._handleShowLinkUnderline(p)))), this.register(o.onHideLinkUnderline(((p) => this._handleHideLinkUnderline(p))));
          }
          resize(g, n) {
            super.resize(g, n), this._state = void 0;
          }
          reset(g) {
            this._clearCurrentLink();
          }
          _clearCurrentLink() {
            if (this._state) {
              this._clearCells(this._state.x1, this._state.y1, this._state.cols - this._state.x1, 1);
              const g = this._state.y2 - this._state.y1 - 1;
              g > 0 && this._clearCells(0, this._state.y1 + 1, this._state.cols, g), this._clearCells(0, this._state.y2, this._state.x2, 1), this._state = void 0;
            }
          }
          _handleShowLinkUnderline(g) {
            if (g.fg === h.INVERTED_DEFAULT_COLOR ? this._ctx.fillStyle = this._themeService.colors.background.css : g.fg !== void 0 && (0, c.is256Color)(g.fg) ? this._ctx.fillStyle = this._themeService.colors.ansi[g.fg].css : this._ctx.fillStyle = this._themeService.colors.foreground.css, g.y1 === g.y2) this._fillBottomLineAtCells(g.x1, g.y1, g.x2 - g.x1);
            else {
              this._fillBottomLineAtCells(g.x1, g.y1, g.cols - g.x1);
              for (let n = g.y1 + 1; n < g.y2; n++) this._fillBottomLineAtCells(0, n, g.cols);
              this._fillBottomLineAtCells(0, g.y2, g.x2);
            }
            this._state = g;
          }
          _handleHideLinkUnderline(g) {
            this._clearCurrentLink();
          }
        }
        t.LinkRenderLayer = d;
      }, 820: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.addDisposableDomListener = void 0, t.addDisposableDomListener = function(a, c, h, r) {
          a.addEventListener(c, h, r);
          let d = !1;
          return { dispose: () => {
            d || (d = !0, a.removeEventListener(c, h, r));
          } };
        };
      }, 274: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CellColorResolver = void 0;
        const c = a(855), h = a(160), r = a(374);
        let d, f = 0, g = 0, n = !1, e = !1, o = !1, s = 0;
        t.CellColorResolver = class {
          constructor(i, u, p, l, m, _) {
            this._terminal = i, this._optionService = u, this._selectionRenderModel = p, this._decorationService = l, this._coreBrowserService = m, this._themeService = _, this.result = { fg: 0, bg: 0, ext: 0 };
          }
          resolve(i, u, p, l) {
            if (this.result.bg = i.bg, this.result.fg = i.fg, this.result.ext = 268435456 & i.bg ? i.extended.ext : 0, g = 0, f = 0, e = !1, n = !1, o = !1, d = this._themeService.colors, s = 0, i.getCode() !== c.NULL_CELL_CODE && i.extended.underlineStyle === 4) {
              const m = Math.max(1, Math.floor(this._optionService.rawOptions.fontSize * this._coreBrowserService.dpr / 15));
              s = u * l % (2 * Math.round(m));
            }
            if (this._decorationService.forEachDecorationAtCell(u, p, "bottom", ((m) => {
              m.backgroundColorRGB && (g = m.backgroundColorRGB.rgba >> 8 & 16777215, e = !0), m.foregroundColorRGB && (f = m.foregroundColorRGB.rgba >> 8 & 16777215, n = !0);
            })), o = this._selectionRenderModel.isCellSelected(this._terminal, u, p), o) {
              if (67108864 & this.result.fg || (50331648 & this.result.bg) != 0) {
                if (67108864 & this.result.fg) switch (50331648 & this.result.fg) {
                  case 16777216:
                  case 33554432:
                    g = this._themeService.colors.ansi[255 & this.result.fg].rgba;
                    break;
                  case 50331648:
                    g = (16777215 & this.result.fg) << 8 | 255;
                    break;
                  default:
                    g = this._themeService.colors.foreground.rgba;
                }
                else switch (50331648 & this.result.bg) {
                  case 16777216:
                  case 33554432:
                    g = this._themeService.colors.ansi[255 & this.result.bg].rgba;
                    break;
                  case 50331648:
                    g = (16777215 & this.result.bg) << 8 | 255;
                }
                g = h.rgba.blend(g, 4294967040 & (this._coreBrowserService.isFocused ? d.selectionBackgroundOpaque : d.selectionInactiveBackgroundOpaque).rgba | 128) >> 8 & 16777215;
              } else g = (this._coreBrowserService.isFocused ? d.selectionBackgroundOpaque : d.selectionInactiveBackgroundOpaque).rgba >> 8 & 16777215;
              if (e = !0, d.selectionForeground && (f = d.selectionForeground.rgba >> 8 & 16777215, n = !0), (0, r.treatGlyphAsBackgroundColor)(i.getCode())) {
                if (67108864 & this.result.fg && (50331648 & this.result.bg) == 0) f = (this._coreBrowserService.isFocused ? d.selectionBackgroundOpaque : d.selectionInactiveBackgroundOpaque).rgba >> 8 & 16777215;
                else {
                  if (67108864 & this.result.fg) switch (50331648 & this.result.bg) {
                    case 16777216:
                    case 33554432:
                      f = this._themeService.colors.ansi[255 & this.result.bg].rgba;
                      break;
                    case 50331648:
                      f = (16777215 & this.result.bg) << 8 | 255;
                  }
                  else switch (50331648 & this.result.fg) {
                    case 16777216:
                    case 33554432:
                      f = this._themeService.colors.ansi[255 & this.result.fg].rgba;
                      break;
                    case 50331648:
                      f = (16777215 & this.result.fg) << 8 | 255;
                      break;
                    default:
                      f = this._themeService.colors.foreground.rgba;
                  }
                  f = h.rgba.blend(f, 4294967040 & (this._coreBrowserService.isFocused ? d.selectionBackgroundOpaque : d.selectionInactiveBackgroundOpaque).rgba | 128) >> 8 & 16777215;
                }
                n = !0;
              }
            }
            this._decorationService.forEachDecorationAtCell(u, p, "top", ((m) => {
              m.backgroundColorRGB && (g = m.backgroundColorRGB.rgba >> 8 & 16777215, e = !0), m.foregroundColorRGB && (f = m.foregroundColorRGB.rgba >> 8 & 16777215, n = !0);
            })), e && (g = o ? -16777216 & i.bg & -134217729 | g | 50331648 : -16777216 & i.bg | g | 50331648), n && (f = -16777216 & i.fg & -67108865 | f | 50331648), 67108864 & this.result.fg && (e && !n && (f = (50331648 & this.result.bg) == 0 ? -134217728 & this.result.fg | 16777215 & d.background.rgba >> 8 | 50331648 : -134217728 & this.result.fg | 67108863 & this.result.bg, n = !0), !e && n && (g = (50331648 & this.result.fg) == 0 ? -67108864 & this.result.bg | 16777215 & d.foreground.rgba >> 8 | 50331648 : -67108864 & this.result.bg | 67108863 & this.result.fg, e = !0)), d = void 0, this.result.bg = e ? g : this.result.bg, this.result.fg = n ? f : this.result.fg, this.result.ext &= 536870911, this.result.ext |= s << 29 & 3758096384;
          }
        };
      }, 627: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.removeTerminalFromCache = t.acquireTextureAtlas = void 0;
        const c = a(509), h = a(197), r = [];
        t.acquireTextureAtlas = function(d, f, g, n, e, o, s, i) {
          const u = (0, h.generateConfig)(n, e, o, s, f, g, i);
          for (let m = 0; m < r.length; m++) {
            const _ = r[m], v = _.ownedBy.indexOf(d);
            if (v >= 0) {
              if ((0, h.configEquals)(_.config, u)) return _.atlas;
              _.ownedBy.length === 1 ? (_.atlas.dispose(), r.splice(m, 1)) : _.ownedBy.splice(v, 1);
              break;
            }
          }
          for (let m = 0; m < r.length; m++) {
            const _ = r[m];
            if ((0, h.configEquals)(_.config, u)) return _.ownedBy.push(d), _.atlas;
          }
          const p = d._core, l = { atlas: new c.TextureAtlas(document, u, p.unicodeService), config: u, ownedBy: [d] };
          return r.push(l), l.atlas;
        }, t.removeTerminalFromCache = function(d) {
          for (let f = 0; f < r.length; f++) {
            const g = r[f].ownedBy.indexOf(d);
            if (g !== -1) {
              r[f].ownedBy.length === 1 ? (r[f].atlas.dispose(), r.splice(f, 1)) : r[f].ownedBy.splice(g, 1);
              break;
            }
          }
        };
      }, 197: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.is256Color = t.configEquals = t.generateConfig = void 0;
        const c = a(160);
        t.generateConfig = function(h, r, d, f, g, n, e) {
          const o = { foreground: n.foreground, background: n.background, cursor: c.NULL_COLOR, cursorAccent: c.NULL_COLOR, selectionForeground: c.NULL_COLOR, selectionBackgroundTransparent: c.NULL_COLOR, selectionBackgroundOpaque: c.NULL_COLOR, selectionInactiveBackgroundTransparent: c.NULL_COLOR, selectionInactiveBackgroundOpaque: c.NULL_COLOR, ansi: n.ansi.slice(), contrastCache: n.contrastCache, halfContrastCache: n.halfContrastCache };
          return { customGlyphs: g.customGlyphs, devicePixelRatio: e, letterSpacing: g.letterSpacing, lineHeight: g.lineHeight, deviceCellWidth: h, deviceCellHeight: r, deviceCharWidth: d, deviceCharHeight: f, fontFamily: g.fontFamily, fontSize: g.fontSize, fontWeight: g.fontWeight, fontWeightBold: g.fontWeightBold, allowTransparency: g.allowTransparency, drawBoldTextInBrightColors: g.drawBoldTextInBrightColors, minimumContrastRatio: g.minimumContrastRatio, colors: o };
        }, t.configEquals = function(h, r) {
          for (let d = 0; d < h.colors.ansi.length; d++) if (h.colors.ansi[d].rgba !== r.colors.ansi[d].rgba) return !1;
          return h.devicePixelRatio === r.devicePixelRatio && h.customGlyphs === r.customGlyphs && h.lineHeight === r.lineHeight && h.letterSpacing === r.letterSpacing && h.fontFamily === r.fontFamily && h.fontSize === r.fontSize && h.fontWeight === r.fontWeight && h.fontWeightBold === r.fontWeightBold && h.allowTransparency === r.allowTransparency && h.deviceCharWidth === r.deviceCharWidth && h.deviceCharHeight === r.deviceCharHeight && h.drawBoldTextInBrightColors === r.drawBoldTextInBrightColors && h.minimumContrastRatio === r.minimumContrastRatio && h.colors.foreground.rgba === r.colors.foreground.rgba && h.colors.background.rgba === r.colors.background.rgba;
        }, t.is256Color = function(h) {
          return (50331648 & h) == 16777216 || (50331648 & h) == 33554432;
        };
      }, 237: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.TEXT_BASELINE = t.DIM_OPACITY = t.INVERTED_DEFAULT_COLOR = void 0;
        const c = a(399);
        t.INVERTED_DEFAULT_COLOR = 257, t.DIM_OPACITY = 0.5, t.TEXT_BASELINE = c.isFirefox || c.isLegacyEdge ? "bottom" : "ideographic";
      }, 457: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CursorBlinkStateManager = void 0, t.CursorBlinkStateManager = class {
          constructor(a, c) {
            this._renderCallback = a, this._coreBrowserService = c, this.isCursorVisible = !0, this._coreBrowserService.isFocused && this._restartInterval();
          }
          get isPaused() {
            return !(this._blinkStartTimeout || this._blinkInterval);
          }
          dispose() {
            this._blinkInterval && (this._coreBrowserService.window.clearInterval(this._blinkInterval), this._blinkInterval = void 0), this._blinkStartTimeout && (this._coreBrowserService.window.clearTimeout(this._blinkStartTimeout), this._blinkStartTimeout = void 0), this._animationFrame && (this._coreBrowserService.window.cancelAnimationFrame(this._animationFrame), this._animationFrame = void 0);
          }
          restartBlinkAnimation() {
            this.isPaused || (this._animationTimeRestarted = Date.now(), this.isCursorVisible = !0, this._animationFrame || (this._animationFrame = this._coreBrowserService.window.requestAnimationFrame((() => {
              this._renderCallback(), this._animationFrame = void 0;
            }))));
          }
          _restartInterval(a = 600) {
            this._blinkInterval && (this._coreBrowserService.window.clearInterval(this._blinkInterval), this._blinkInterval = void 0), this._blinkStartTimeout = this._coreBrowserService.window.setTimeout((() => {
              if (this._animationTimeRestarted) {
                const c = 600 - (Date.now() - this._animationTimeRestarted);
                if (this._animationTimeRestarted = void 0, c > 0) return void this._restartInterval(c);
              }
              this.isCursorVisible = !1, this._animationFrame = this._coreBrowserService.window.requestAnimationFrame((() => {
                this._renderCallback(), this._animationFrame = void 0;
              })), this._blinkInterval = this._coreBrowserService.window.setInterval((() => {
                if (this._animationTimeRestarted) {
                  const c = 600 - (Date.now() - this._animationTimeRestarted);
                  return this._animationTimeRestarted = void 0, void this._restartInterval(c);
                }
                this.isCursorVisible = !this.isCursorVisible, this._animationFrame = this._coreBrowserService.window.requestAnimationFrame((() => {
                  this._renderCallback(), this._animationFrame = void 0;
                }));
              }), 600);
            }), a);
          }
          pause() {
            this.isCursorVisible = !0, this._blinkInterval && (this._coreBrowserService.window.clearInterval(this._blinkInterval), this._blinkInterval = void 0), this._blinkStartTimeout && (this._coreBrowserService.window.clearTimeout(this._blinkStartTimeout), this._blinkStartTimeout = void 0), this._animationFrame && (this._coreBrowserService.window.cancelAnimationFrame(this._animationFrame), this._animationFrame = void 0);
          }
          resume() {
            this.pause(), this._animationTimeRestarted = void 0, this._restartInterval(), this.restartBlinkAnimation();
          }
        };
      }, 860: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.tryDrawCustomChar = t.powerlineDefinitions = t.boxDrawingDefinitions = t.blockElementDefinitions = void 0;
        const c = a(374);
        t.blockElementDefinitions = { "▀": [{ x: 0, y: 0, w: 8, h: 4 }], "▁": [{ x: 0, y: 7, w: 8, h: 1 }], "▂": [{ x: 0, y: 6, w: 8, h: 2 }], "▃": [{ x: 0, y: 5, w: 8, h: 3 }], "▄": [{ x: 0, y: 4, w: 8, h: 4 }], "▅": [{ x: 0, y: 3, w: 8, h: 5 }], "▆": [{ x: 0, y: 2, w: 8, h: 6 }], "▇": [{ x: 0, y: 1, w: 8, h: 7 }], "█": [{ x: 0, y: 0, w: 8, h: 8 }], "▉": [{ x: 0, y: 0, w: 7, h: 8 }], "▊": [{ x: 0, y: 0, w: 6, h: 8 }], "▋": [{ x: 0, y: 0, w: 5, h: 8 }], "▌": [{ x: 0, y: 0, w: 4, h: 8 }], "▍": [{ x: 0, y: 0, w: 3, h: 8 }], "▎": [{ x: 0, y: 0, w: 2, h: 8 }], "▏": [{ x: 0, y: 0, w: 1, h: 8 }], "▐": [{ x: 4, y: 0, w: 4, h: 8 }], "▔": [{ x: 0, y: 0, w: 8, h: 1 }], "▕": [{ x: 7, y: 0, w: 1, h: 8 }], "▖": [{ x: 0, y: 4, w: 4, h: 4 }], "▗": [{ x: 4, y: 4, w: 4, h: 4 }], "▘": [{ x: 0, y: 0, w: 4, h: 4 }], "▙": [{ x: 0, y: 0, w: 4, h: 8 }, { x: 0, y: 4, w: 8, h: 4 }], "▚": [{ x: 0, y: 0, w: 4, h: 4 }, { x: 4, y: 4, w: 4, h: 4 }], "▛": [{ x: 0, y: 0, w: 4, h: 8 }, { x: 4, y: 0, w: 4, h: 4 }], "▜": [{ x: 0, y: 0, w: 8, h: 4 }, { x: 4, y: 0, w: 4, h: 8 }], "▝": [{ x: 4, y: 0, w: 4, h: 4 }], "▞": [{ x: 4, y: 0, w: 4, h: 4 }, { x: 0, y: 4, w: 4, h: 4 }], "▟": [{ x: 4, y: 0, w: 4, h: 8 }, { x: 0, y: 4, w: 8, h: 4 }], "🭰": [{ x: 1, y: 0, w: 1, h: 8 }], "🭱": [{ x: 2, y: 0, w: 1, h: 8 }], "🭲": [{ x: 3, y: 0, w: 1, h: 8 }], "🭳": [{ x: 4, y: 0, w: 1, h: 8 }], "🭴": [{ x: 5, y: 0, w: 1, h: 8 }], "🭵": [{ x: 6, y: 0, w: 1, h: 8 }], "🭶": [{ x: 0, y: 1, w: 8, h: 1 }], "🭷": [{ x: 0, y: 2, w: 8, h: 1 }], "🭸": [{ x: 0, y: 3, w: 8, h: 1 }], "🭹": [{ x: 0, y: 4, w: 8, h: 1 }], "🭺": [{ x: 0, y: 5, w: 8, h: 1 }], "🭻": [{ x: 0, y: 6, w: 8, h: 1 }], "🭼": [{ x: 0, y: 0, w: 1, h: 8 }, { x: 0, y: 7, w: 8, h: 1 }], "🭽": [{ x: 0, y: 0, w: 1, h: 8 }, { x: 0, y: 0, w: 8, h: 1 }], "🭾": [{ x: 7, y: 0, w: 1, h: 8 }, { x: 0, y: 0, w: 8, h: 1 }], "🭿": [{ x: 7, y: 0, w: 1, h: 8 }, { x: 0, y: 7, w: 8, h: 1 }], "🮀": [{ x: 0, y: 0, w: 8, h: 1 }, { x: 0, y: 7, w: 8, h: 1 }], "🮁": [{ x: 0, y: 0, w: 8, h: 1 }, { x: 0, y: 2, w: 8, h: 1 }, { x: 0, y: 4, w: 8, h: 1 }, { x: 0, y: 7, w: 8, h: 1 }], "🮂": [{ x: 0, y: 0, w: 8, h: 2 }], "🮃": [{ x: 0, y: 0, w: 8, h: 3 }], "🮄": [{ x: 0, y: 0, w: 8, h: 5 }], "🮅": [{ x: 0, y: 0, w: 8, h: 6 }], "🮆": [{ x: 0, y: 0, w: 8, h: 7 }], "🮇": [{ x: 6, y: 0, w: 2, h: 8 }], "🮈": [{ x: 5, y: 0, w: 3, h: 8 }], "🮉": [{ x: 3, y: 0, w: 5, h: 8 }], "🮊": [{ x: 2, y: 0, w: 6, h: 8 }], "🮋": [{ x: 1, y: 0, w: 7, h: 8 }], "🮕": [{ x: 0, y: 0, w: 2, h: 2 }, { x: 4, y: 0, w: 2, h: 2 }, { x: 2, y: 2, w: 2, h: 2 }, { x: 6, y: 2, w: 2, h: 2 }, { x: 0, y: 4, w: 2, h: 2 }, { x: 4, y: 4, w: 2, h: 2 }, { x: 2, y: 6, w: 2, h: 2 }, { x: 6, y: 6, w: 2, h: 2 }], "🮖": [{ x: 2, y: 0, w: 2, h: 2 }, { x: 6, y: 0, w: 2, h: 2 }, { x: 0, y: 2, w: 2, h: 2 }, { x: 4, y: 2, w: 2, h: 2 }, { x: 2, y: 4, w: 2, h: 2 }, { x: 6, y: 4, w: 2, h: 2 }, { x: 0, y: 6, w: 2, h: 2 }, { x: 4, y: 6, w: 2, h: 2 }], "🮗": [{ x: 0, y: 2, w: 8, h: 2 }, { x: 0, y: 6, w: 8, h: 2 }] };
        const h = { "░": [[1, 0, 0, 0], [0, 0, 0, 0], [0, 0, 1, 0], [0, 0, 0, 0]], "▒": [[1, 0], [0, 0], [0, 1], [0, 0]], "▓": [[0, 1], [1, 1], [1, 0], [1, 1]] };
        t.boxDrawingDefinitions = { "─": { 1: "M0,.5 L1,.5" }, "━": { 3: "M0,.5 L1,.5" }, "│": { 1: "M.5,0 L.5,1" }, "┃": { 3: "M.5,0 L.5,1" }, "┌": { 1: "M0.5,1 L.5,.5 L1,.5" }, "┏": { 3: "M0.5,1 L.5,.5 L1,.5" }, "┐": { 1: "M0,.5 L.5,.5 L.5,1" }, "┓": { 3: "M0,.5 L.5,.5 L.5,1" }, "└": { 1: "M.5,0 L.5,.5 L1,.5" }, "┗": { 3: "M.5,0 L.5,.5 L1,.5" }, "┘": { 1: "M.5,0 L.5,.5 L0,.5" }, "┛": { 3: "M.5,0 L.5,.5 L0,.5" }, "├": { 1: "M.5,0 L.5,1 M.5,.5 L1,.5" }, "┣": { 3: "M.5,0 L.5,1 M.5,.5 L1,.5" }, "┤": { 1: "M.5,0 L.5,1 M.5,.5 L0,.5" }, "┫": { 3: "M.5,0 L.5,1 M.5,.5 L0,.5" }, "┬": { 1: "M0,.5 L1,.5 M.5,.5 L.5,1" }, "┳": { 3: "M0,.5 L1,.5 M.5,.5 L.5,1" }, "┴": { 1: "M0,.5 L1,.5 M.5,.5 L.5,0" }, "┻": { 3: "M0,.5 L1,.5 M.5,.5 L.5,0" }, "┼": { 1: "M0,.5 L1,.5 M.5,0 L.5,1" }, "╋": { 3: "M0,.5 L1,.5 M.5,0 L.5,1" }, "╴": { 1: "M.5,.5 L0,.5" }, "╸": { 3: "M.5,.5 L0,.5" }, "╵": { 1: "M.5,.5 L.5,0" }, "╹": { 3: "M.5,.5 L.5,0" }, "╶": { 1: "M.5,.5 L1,.5" }, "╺": { 3: "M.5,.5 L1,.5" }, "╷": { 1: "M.5,.5 L.5,1" }, "╻": { 3: "M.5,.5 L.5,1" }, "═": { 1: (n, e) => `M0,${0.5 - e} L1,${0.5 - e} M0,${0.5 + e} L1,${0.5 + e}` }, "║": { 1: (n, e) => `M${0.5 - n},0 L${0.5 - n},1 M${0.5 + n},0 L${0.5 + n},1` }, "╒": { 1: (n, e) => `M.5,1 L.5,${0.5 - e} L1,${0.5 - e} M.5,${0.5 + e} L1,${0.5 + e}` }, "╓": { 1: (n, e) => `M${0.5 - n},1 L${0.5 - n},.5 L1,.5 M${0.5 + n},.5 L${0.5 + n},1` }, "╔": { 1: (n, e) => `M1,${0.5 - e} L${0.5 - n},${0.5 - e} L${0.5 - n},1 M1,${0.5 + e} L${0.5 + n},${0.5 + e} L${0.5 + n},1` }, "╕": { 1: (n, e) => `M0,${0.5 - e} L.5,${0.5 - e} L.5,1 M0,${0.5 + e} L.5,${0.5 + e}` }, "╖": { 1: (n, e) => `M${0.5 + n},1 L${0.5 + n},.5 L0,.5 M${0.5 - n},.5 L${0.5 - n},1` }, "╗": { 1: (n, e) => `M0,${0.5 + e} L${0.5 - n},${0.5 + e} L${0.5 - n},1 M0,${0.5 - e} L${0.5 + n},${0.5 - e} L${0.5 + n},1` }, "╘": { 1: (n, e) => `M.5,0 L.5,${0.5 + e} L1,${0.5 + e} M.5,${0.5 - e} L1,${0.5 - e}` }, "╙": { 1: (n, e) => `M1,.5 L${0.5 - n},.5 L${0.5 - n},0 M${0.5 + n},.5 L${0.5 + n},0` }, "╚": { 1: (n, e) => `M1,${0.5 - e} L${0.5 + n},${0.5 - e} L${0.5 + n},0 M1,${0.5 + e} L${0.5 - n},${0.5 + e} L${0.5 - n},0` }, "╛": { 1: (n, e) => `M0,${0.5 + e} L.5,${0.5 + e} L.5,0 M0,${0.5 - e} L.5,${0.5 - e}` }, "╜": { 1: (n, e) => `M0,.5 L${0.5 + n},.5 L${0.5 + n},0 M${0.5 - n},.5 L${0.5 - n},0` }, "╝": { 1: (n, e) => `M0,${0.5 - e} L${0.5 - n},${0.5 - e} L${0.5 - n},0 M0,${0.5 + e} L${0.5 + n},${0.5 + e} L${0.5 + n},0` }, "╞": { 1: (n, e) => `M.5,0 L.5,1 M.5,${0.5 - e} L1,${0.5 - e} M.5,${0.5 + e} L1,${0.5 + e}` }, "╟": { 1: (n, e) => `M${0.5 - n},0 L${0.5 - n},1 M${0.5 + n},0 L${0.5 + n},1 M${0.5 + n},.5 L1,.5` }, "╠": { 1: (n, e) => `M${0.5 - n},0 L${0.5 - n},1 M1,${0.5 + e} L${0.5 + n},${0.5 + e} L${0.5 + n},1 M1,${0.5 - e} L${0.5 + n},${0.5 - e} L${0.5 + n},0` }, "╡": { 1: (n, e) => `M.5,0 L.5,1 M0,${0.5 - e} L.5,${0.5 - e} M0,${0.5 + e} L.5,${0.5 + e}` }, "╢": { 1: (n, e) => `M0,.5 L${0.5 - n},.5 M${0.5 - n},0 L${0.5 - n},1 M${0.5 + n},0 L${0.5 + n},1` }, "╣": { 1: (n, e) => `M${0.5 + n},0 L${0.5 + n},1 M0,${0.5 + e} L${0.5 - n},${0.5 + e} L${0.5 - n},1 M0,${0.5 - e} L${0.5 - n},${0.5 - e} L${0.5 - n},0` }, "╤": { 1: (n, e) => `M0,${0.5 - e} L1,${0.5 - e} M0,${0.5 + e} L1,${0.5 + e} M.5,${0.5 + e} L.5,1` }, "╥": { 1: (n, e) => `M0,.5 L1,.5 M${0.5 - n},.5 L${0.5 - n},1 M${0.5 + n},.5 L${0.5 + n},1` }, "╦": { 1: (n, e) => `M0,${0.5 - e} L1,${0.5 - e} M0,${0.5 + e} L${0.5 - n},${0.5 + e} L${0.5 - n},1 M1,${0.5 + e} L${0.5 + n},${0.5 + e} L${0.5 + n},1` }, "╧": { 1: (n, e) => `M.5,0 L.5,${0.5 - e} M0,${0.5 - e} L1,${0.5 - e} M0,${0.5 + e} L1,${0.5 + e}` }, "╨": { 1: (n, e) => `M0,.5 L1,.5 M${0.5 - n},.5 L${0.5 - n},0 M${0.5 + n},.5 L${0.5 + n},0` }, "╩": { 1: (n, e) => `M0,${0.5 + e} L1,${0.5 + e} M0,${0.5 - e} L${0.5 - n},${0.5 - e} L${0.5 - n},0 M1,${0.5 - e} L${0.5 + n},${0.5 - e} L${0.5 + n},0` }, "╪": { 1: (n, e) => `M.5,0 L.5,1 M0,${0.5 - e} L1,${0.5 - e} M0,${0.5 + e} L1,${0.5 + e}` }, "╫": { 1: (n, e) => `M0,.5 L1,.5 M${0.5 - n},0 L${0.5 - n},1 M${0.5 + n},0 L${0.5 + n},1` }, "╬": { 1: (n, e) => `M0,${0.5 + e} L${0.5 - n},${0.5 + e} L${0.5 - n},1 M1,${0.5 + e} L${0.5 + n},${0.5 + e} L${0.5 + n},1 M0,${0.5 - e} L${0.5 - n},${0.5 - e} L${0.5 - n},0 M1,${0.5 - e} L${0.5 + n},${0.5 - e} L${0.5 + n},0` }, "╱": { 1: "M1,0 L0,1" }, "╲": { 1: "M0,0 L1,1" }, "╳": { 1: "M1,0 L0,1 M0,0 L1,1" }, "╼": { 1: "M.5,.5 L0,.5", 3: "M.5,.5 L1,.5" }, "╽": { 1: "M.5,.5 L.5,0", 3: "M.5,.5 L.5,1" }, "╾": { 1: "M.5,.5 L1,.5", 3: "M.5,.5 L0,.5" }, "╿": { 1: "M.5,.5 L.5,1", 3: "M.5,.5 L.5,0" }, "┍": { 1: "M.5,.5 L.5,1", 3: "M.5,.5 L1,.5" }, "┎": { 1: "M.5,.5 L1,.5", 3: "M.5,.5 L.5,1" }, "┑": { 1: "M.5,.5 L.5,1", 3: "M.5,.5 L0,.5" }, "┒": { 1: "M.5,.5 L0,.5", 3: "M.5,.5 L.5,1" }, "┕": { 1: "M.5,.5 L.5,0", 3: "M.5,.5 L1,.5" }, "┖": { 1: "M.5,.5 L1,.5", 3: "M.5,.5 L.5,0" }, "┙": { 1: "M.5,.5 L.5,0", 3: "M.5,.5 L0,.5" }, "┚": { 1: "M.5,.5 L0,.5", 3: "M.5,.5 L.5,0" }, "┝": { 1: "M.5,0 L.5,1", 3: "M.5,.5 L1,.5" }, "┞": { 1: "M0.5,1 L.5,.5 L1,.5", 3: "M.5,.5 L.5,0" }, "┟": { 1: "M.5,0 L.5,.5 L1,.5", 3: "M.5,.5 L.5,1" }, "┠": { 1: "M.5,.5 L1,.5", 3: "M.5,0 L.5,1" }, "┡": { 1: "M.5,.5 L.5,1", 3: "M.5,0 L.5,.5 L1,.5" }, "┢": { 1: "M.5,.5 L.5,0", 3: "M0.5,1 L.5,.5 L1,.5" }, "┥": { 1: "M.5,0 L.5,1", 3: "M.5,.5 L0,.5" }, "┦": { 1: "M0,.5 L.5,.5 L.5,1", 3: "M.5,.5 L.5,0" }, "┧": { 1: "M.5,0 L.5,.5 L0,.5", 3: "M.5,.5 L.5,1" }, "┨": { 1: "M.5,.5 L0,.5", 3: "M.5,0 L.5,1" }, "┩": { 1: "M.5,.5 L.5,1", 3: "M.5,0 L.5,.5 L0,.5" }, "┪": { 1: "M.5,.5 L.5,0", 3: "M0,.5 L.5,.5 L.5,1" }, "┭": { 1: "M0.5,1 L.5,.5 L1,.5", 3: "M.5,.5 L0,.5" }, "┮": { 1: "M0,.5 L.5,.5 L.5,1", 3: "M.5,.5 L1,.5" }, "┯": { 1: "M.5,.5 L.5,1", 3: "M0,.5 L1,.5" }, "┰": { 1: "M0,.5 L1,.5", 3: "M.5,.5 L.5,1" }, "┱": { 1: "M.5,.5 L1,.5", 3: "M0,.5 L.5,.5 L.5,1" }, "┲": { 1: "M.5,.5 L0,.5", 3: "M0.5,1 L.5,.5 L1,.5" }, "┵": { 1: "M.5,0 L.5,.5 L1,.5", 3: "M.5,.5 L0,.5" }, "┶": { 1: "M.5,0 L.5,.5 L0,.5", 3: "M.5,.5 L1,.5" }, "┷": { 1: "M.5,.5 L.5,0", 3: "M0,.5 L1,.5" }, "┸": { 1: "M0,.5 L1,.5", 3: "M.5,.5 L.5,0" }, "┹": { 1: "M.5,.5 L1,.5", 3: "M.5,0 L.5,.5 L0,.5" }, "┺": { 1: "M.5,.5 L0,.5", 3: "M.5,0 L.5,.5 L1,.5" }, "┽": { 1: "M.5,0 L.5,1 M.5,.5 L1,.5", 3: "M.5,.5 L0,.5" }, "┾": { 1: "M.5,0 L.5,1 M.5,.5 L0,.5", 3: "M.5,.5 L1,.5" }, "┿": { 1: "M.5,0 L.5,1", 3: "M0,.5 L1,.5" }, "╀": { 1: "M0,.5 L1,.5 M.5,.5 L.5,1", 3: "M.5,.5 L.5,0" }, "╁": { 1: "M.5,.5 L.5,0 M0,.5 L1,.5", 3: "M.5,.5 L.5,1" }, "╂": { 1: "M0,.5 L1,.5", 3: "M.5,0 L.5,1" }, "╃": { 1: "M0.5,1 L.5,.5 L1,.5", 3: "M.5,0 L.5,.5 L0,.5" }, "╄": { 1: "M0,.5 L.5,.5 L.5,1", 3: "M.5,0 L.5,.5 L1,.5" }, "╅": { 1: "M.5,0 L.5,.5 L1,.5", 3: "M0,.5 L.5,.5 L.5,1" }, "╆": { 1: "M.5,0 L.5,.5 L0,.5", 3: "M0.5,1 L.5,.5 L1,.5" }, "╇": { 1: "M.5,.5 L.5,1", 3: "M.5,.5 L.5,0 M0,.5 L1,.5" }, "╈": { 1: "M.5,.5 L.5,0", 3: "M0,.5 L1,.5 M.5,.5 L.5,1" }, "╉": { 1: "M.5,.5 L1,.5", 3: "M.5,0 L.5,1 M.5,.5 L0,.5" }, "╊": { 1: "M.5,.5 L0,.5", 3: "M.5,0 L.5,1 M.5,.5 L1,.5" }, "╌": { 1: "M.1,.5 L.4,.5 M.6,.5 L.9,.5" }, "╍": { 3: "M.1,.5 L.4,.5 M.6,.5 L.9,.5" }, "┄": { 1: "M.0667,.5 L.2667,.5 M.4,.5 L.6,.5 M.7333,.5 L.9333,.5" }, "┅": { 3: "M.0667,.5 L.2667,.5 M.4,.5 L.6,.5 M.7333,.5 L.9333,.5" }, "┈": { 1: "M.05,.5 L.2,.5 M.3,.5 L.45,.5 M.55,.5 L.7,.5 M.8,.5 L.95,.5" }, "┉": { 3: "M.05,.5 L.2,.5 M.3,.5 L.45,.5 M.55,.5 L.7,.5 M.8,.5 L.95,.5" }, "╎": { 1: "M.5,.1 L.5,.4 M.5,.6 L.5,.9" }, "╏": { 3: "M.5,.1 L.5,.4 M.5,.6 L.5,.9" }, "┆": { 1: "M.5,.0667 L.5,.2667 M.5,.4 L.5,.6 M.5,.7333 L.5,.9333" }, "┇": { 3: "M.5,.0667 L.5,.2667 M.5,.4 L.5,.6 M.5,.7333 L.5,.9333" }, "┊": { 1: "M.5,.05 L.5,.2 M.5,.3 L.5,.45 L.5,.55 M.5,.7 L.5,.95" }, "┋": { 3: "M.5,.05 L.5,.2 M.5,.3 L.5,.45 L.5,.55 M.5,.7 L.5,.95" }, "╭": { 1: (n, e) => `M.5,1 L.5,${0.5 + e / 0.15 * 0.5} C.5,${0.5 + e / 0.15 * 0.5},.5,.5,1,.5` }, "╮": { 1: (n, e) => `M.5,1 L.5,${0.5 + e / 0.15 * 0.5} C.5,${0.5 + e / 0.15 * 0.5},.5,.5,0,.5` }, "╯": { 1: (n, e) => `M.5,0 L.5,${0.5 - e / 0.15 * 0.5} C.5,${0.5 - e / 0.15 * 0.5},.5,.5,0,.5` }, "╰": { 1: (n, e) => `M.5,0 L.5,${0.5 - e / 0.15 * 0.5} C.5,${0.5 - e / 0.15 * 0.5},.5,.5,1,.5` } }, t.powerlineDefinitions = { "": { d: "M0,0 L1,.5 L0,1", type: 0, rightPadding: 2 }, "": { d: "M-1,-.5 L1,.5 L-1,1.5", type: 1, leftPadding: 1, rightPadding: 1 }, "": { d: "M1,0 L0,.5 L1,1", type: 0, leftPadding: 2 }, "": { d: "M2,-.5 L0,.5 L2,1.5", type: 1, leftPadding: 1, rightPadding: 1 }, "": { d: "M0,0 L0,1 C0.552,1,1,0.776,1,.5 C1,0.224,0.552,0,0,0", type: 0, rightPadding: 1 }, "": { d: "M.2,1 C.422,1,.8,.826,.78,.5 C.8,.174,0.422,0,.2,0", type: 1, rightPadding: 1 }, "": { d: "M1,0 L1,1 C0.448,1,0,0.776,0,.5 C0,0.224,0.448,0,1,0", type: 0, leftPadding: 1 }, "": { d: "M.8,1 C0.578,1,0.2,.826,.22,.5 C0.2,0.174,0.578,0,0.8,0", type: 1, leftPadding: 1 }, "": { d: "M-.5,-.5 L1.5,1.5 L-.5,1.5", type: 0 }, "": { d: "M-.5,-.5 L1.5,1.5", type: 1, leftPadding: 1, rightPadding: 1 }, "": { d: "M1.5,-.5 L-.5,1.5 L1.5,1.5", type: 0 }, "": { d: "M1.5,-.5 L-.5,1.5 L-.5,-.5", type: 0 }, "": { d: "M1.5,-.5 L-.5,1.5", type: 1, leftPadding: 1, rightPadding: 1 }, "": { d: "M-.5,-.5 L1.5,1.5 L1.5,-.5", type: 0 } }, t.powerlineDefinitions[""] = t.powerlineDefinitions[""], t.powerlineDefinitions[""] = t.powerlineDefinitions[""], t.tryDrawCustomChar = function(n, e, o, s, i, u, p, l) {
          const m = t.blockElementDefinitions[e];
          if (m) return (function(w, S, b, x, A, P) {
            for (let k = 0; k < S.length; k++) {
              const M = S[k], y = A / 8, L = P / 8;
              w.fillRect(b + M.x * y, x + M.y * L, M.w * y, M.h * L);
            }
          })(n, m, o, s, i, u), !0;
          const _ = h[e];
          if (_) return (function(w, S, b, x, A, P) {
            let k = r.get(S);
            k || (k = /* @__PURE__ */ new Map(), r.set(S, k));
            const M = w.fillStyle;
            if (typeof M != "string") throw new Error(`Unexpected fillStyle type "${M}"`);
            let y = k.get(M);
            if (!y) {
              const L = S[0].length, R = S.length, D = w.canvas.ownerDocument.createElement("canvas");
              D.width = L, D.height = R;
              const F = (0, c.throwIfFalsy)(D.getContext("2d")), U = new ImageData(L, R);
              let K, q, O, E;
              if (M.startsWith("#")) K = parseInt(M.slice(1, 3), 16), q = parseInt(M.slice(3, 5), 16), O = parseInt(M.slice(5, 7), 16), E = M.length > 7 && parseInt(M.slice(7, 9), 16) || 1;
              else {
                if (!M.startsWith("rgba")) throw new Error(`Unexpected fillStyle color format "${M}" when drawing pattern glyph`);
                [K, q, O, E] = M.substring(5, M.length - 1).split(",").map(((H) => parseFloat(H)));
              }
              for (let H = 0; H < R; H++) for (let N = 0; N < L; N++) U.data[4 * (H * L + N)] = K, U.data[4 * (H * L + N) + 1] = q, U.data[4 * (H * L + N) + 2] = O, U.data[4 * (H * L + N) + 3] = S[H][N] * (255 * E);
              F.putImageData(U, 0, 0), y = (0, c.throwIfFalsy)(w.createPattern(D, null)), k.set(M, y);
            }
            w.fillStyle = y, w.fillRect(b, x, A, P);
          })(n, _, o, s, i, u), !0;
          const v = t.boxDrawingDefinitions[e];
          if (v) return (function(w, S, b, x, A, P, k) {
            w.strokeStyle = w.fillStyle;
            for (const [M, y] of Object.entries(S)) {
              let L;
              w.beginPath(), w.lineWidth = k * Number.parseInt(M), L = typeof y == "function" ? y(0.15, 0.15 / P * A) : y;
              for (const R of L.split(" ")) {
                const D = R[0], F = f[D];
                if (!F) {
                  console.error(`Could not find drawing instructions for "${D}"`);
                  continue;
                }
                const U = R.substring(1).split(",");
                U[0] && U[1] && F(w, g(U, A, P, b, x, !0, k));
              }
              w.stroke(), w.closePath();
            }
          })(n, v, o, s, i, u, l), !0;
          const C = t.powerlineDefinitions[e];
          return !!C && ((function(w, S, b, x, A, P, k, M) {
            var R, D;
            const y = new Path2D();
            y.rect(b, x, A, P), w.clip(y), w.beginPath();
            const L = k / 12;
            w.lineWidth = M * L;
            for (const F of S.d.split(" ")) {
              const U = F[0], K = f[U];
              if (!K) {
                console.error(`Could not find drawing instructions for "${U}"`);
                continue;
              }
              const q = F.substring(1).split(",");
              q[0] && q[1] && K(w, g(q, A, P, b, x, !1, M, ((R = S.leftPadding) != null ? R : 0) * (L / 2), ((D = S.rightPadding) != null ? D : 0) * (L / 2)));
            }
            S.type === 1 ? (w.strokeStyle = w.fillStyle, w.stroke()) : w.fill(), w.closePath();
          })(n, C, o, s, i, u, p, l), !0);
        };
        const r = /* @__PURE__ */ new Map();
        function d(n, e, o = 0) {
          return Math.max(Math.min(n, e), o);
        }
        const f = { C: (n, e) => n.bezierCurveTo(e[0], e[1], e[2], e[3], e[4], e[5]), L: (n, e) => n.lineTo(e[0], e[1]), M: (n, e) => n.moveTo(e[0], e[1]) };
        function g(n, e, o, s, i, u, p, l = 0, m = 0) {
          const _ = n.map(((v) => parseFloat(v) || parseInt(v)));
          if (_.length < 2) throw new Error("Too few arguments for instruction");
          for (let v = 0; v < _.length; v += 2) _[v] *= e - l * p - m * p, u && _[v] !== 0 && (_[v] = d(Math.round(_[v] + 0.5) - 0.5, e, 0)), _[v] += s + l * p;
          for (let v = 1; v < _.length; v += 2) _[v] *= o, u && _[v] !== 0 && (_[v] = d(Math.round(_[v] + 0.5) - 0.5, o, 0)), _[v] += i;
          return _;
        }
      }, 56: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.observeDevicePixelDimensions = void 0;
        const c = a(859);
        t.observeDevicePixelDimensions = function(h, r, d) {
          let f = new r.ResizeObserver(((g) => {
            const n = g.find(((s) => s.target === h));
            if (!n) return;
            if (!("devicePixelContentBoxSize" in n)) return f == null || f.disconnect(), void (f = void 0);
            const e = n.devicePixelContentBoxSize[0].inlineSize, o = n.devicePixelContentBoxSize[0].blockSize;
            e > 0 && o > 0 && d(e, o);
          }));
          try {
            f.observe(h, { box: ["device-pixel-content-box"] });
          } catch (g) {
            f.disconnect(), f = void 0;
          }
          return (0, c.toDisposable)((() => f == null ? void 0 : f.disconnect()));
        };
      }, 374: (T, t) => {
        function a(h) {
          return 57508 <= h && h <= 57558;
        }
        function c(h) {
          return h >= 128512 && h <= 128591 || h >= 127744 && h <= 128511 || h >= 128640 && h <= 128767 || h >= 9728 && h <= 9983 || h >= 9984 && h <= 10175 || h >= 65024 && h <= 65039 || h >= 129280 && h <= 129535 || h >= 127462 && h <= 127487;
        }
        Object.defineProperty(t, "__esModule", { value: !0 }), t.computeNextVariantOffset = t.createRenderDimensions = t.treatGlyphAsBackgroundColor = t.allowRescaling = t.isEmoji = t.isRestrictedPowerlineGlyph = t.isPowerlineGlyph = t.throwIfFalsy = void 0, t.throwIfFalsy = function(h) {
          if (!h) throw new Error("value must not be falsy");
          return h;
        }, t.isPowerlineGlyph = a, t.isRestrictedPowerlineGlyph = function(h) {
          return 57520 <= h && h <= 57527;
        }, t.isEmoji = c, t.allowRescaling = function(h, r, d, f) {
          return r === 1 && d > Math.ceil(1.5 * f) && h !== void 0 && h > 255 && !c(h) && !a(h) && !(function(g) {
            return 57344 <= g && g <= 63743;
          })(h);
        }, t.treatGlyphAsBackgroundColor = function(h) {
          return a(h) || (function(r) {
            return 9472 <= r && r <= 9631;
          })(h);
        }, t.createRenderDimensions = function() {
          return { css: { canvas: { width: 0, height: 0 }, cell: { width: 0, height: 0 } }, device: { canvas: { width: 0, height: 0 }, cell: { width: 0, height: 0 }, char: { width: 0, height: 0, left: 0, top: 0 } } };
        }, t.computeNextVariantOffset = function(h, r, d = 0) {
          return (h - (2 * Math.round(r) - d)) % (2 * Math.round(r));
        };
      }, 296: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.createSelectionRenderModel = void 0;
        class a {
          constructor() {
            this.clear();
          }
          clear() {
            this.hasSelection = !1, this.columnSelectMode = !1, this.viewportStartRow = 0, this.viewportEndRow = 0, this.viewportCappedStartRow = 0, this.viewportCappedEndRow = 0, this.startCol = 0, this.endCol = 0, this.selectionStart = void 0, this.selectionEnd = void 0;
          }
          update(h, r, d, f = !1) {
            if (this.selectionStart = r, this.selectionEnd = d, !r || !d || r[0] === d[0] && r[1] === d[1]) return void this.clear();
            const g = h.buffers.active.ydisp, n = r[1] - g, e = d[1] - g, o = Math.max(n, 0), s = Math.min(e, h.rows - 1);
            o >= h.rows || s < 0 ? this.clear() : (this.hasSelection = !0, this.columnSelectMode = f, this.viewportStartRow = n, this.viewportEndRow = e, this.viewportCappedStartRow = o, this.viewportCappedEndRow = s, this.startCol = r[0], this.endCol = d[0]);
          }
          isCellSelected(h, r, d) {
            return !!this.hasSelection && (d -= h.buffer.active.viewportY, this.columnSelectMode ? this.startCol <= this.endCol ? r >= this.startCol && d >= this.viewportCappedStartRow && r < this.endCol && d <= this.viewportCappedEndRow : r < this.startCol && d >= this.viewportCappedStartRow && r >= this.endCol && d <= this.viewportCappedEndRow : d > this.viewportStartRow && d < this.viewportEndRow || this.viewportStartRow === this.viewportEndRow && d === this.viewportStartRow && r >= this.startCol && r < this.endCol || this.viewportStartRow < this.viewportEndRow && d === this.viewportEndRow && r < this.endCol || this.viewportStartRow < this.viewportEndRow && d === this.viewportStartRow && r >= this.startCol);
          }
        }
        t.createSelectionRenderModel = function() {
          return new a();
        };
      }, 509: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.TextureAtlas = void 0;
        const c = a(237), h = a(860), r = a(374), d = a(160), f = a(345), g = a(485), n = a(385), e = a(147), o = a(855), s = { texturePage: 0, texturePosition: { x: 0, y: 0 }, texturePositionClipSpace: { x: 0, y: 0 }, offset: { x: 0, y: 0 }, size: { x: 0, y: 0 }, sizeClipSpace: { x: 0, y: 0 } };
        let i;
        class u {
          get pages() {
            return this._pages;
          }
          constructor(v, C, w) {
            this._document = v, this._config = C, this._unicodeService = w, this._didWarmUp = !1, this._cacheMap = new g.FourKeyMap(), this._cacheMapCombined = new g.FourKeyMap(), this._pages = [], this._activePages = [], this._workBoundingBox = { top: 0, left: 0, bottom: 0, right: 0 }, this._workAttributeData = new e.AttributeData(), this._textureSize = 512, this._onAddTextureAtlasCanvas = new f.EventEmitter(), this.onAddTextureAtlasCanvas = this._onAddTextureAtlasCanvas.event, this._onRemoveTextureAtlasCanvas = new f.EventEmitter(), this.onRemoveTextureAtlasCanvas = this._onRemoveTextureAtlasCanvas.event, this._requestClearModel = !1, this._createNewPage(), this._tmpCanvas = m(v, 4 * this._config.deviceCellWidth + 4, this._config.deviceCellHeight + 4), this._tmpCtx = (0, r.throwIfFalsy)(this._tmpCanvas.getContext("2d", { alpha: this._config.allowTransparency, willReadFrequently: !0 }));
          }
          dispose() {
            for (const v of this.pages) v.canvas.remove();
            this._onAddTextureAtlasCanvas.dispose();
          }
          warmUp() {
            this._didWarmUp || (this._doWarmUp(), this._didWarmUp = !0);
          }
          _doWarmUp() {
            const v = new n.IdleTaskQueue();
            for (let C = 33; C < 126; C++) v.enqueue((() => {
              if (!this._cacheMap.get(C, o.DEFAULT_COLOR, o.DEFAULT_COLOR, o.DEFAULT_EXT)) {
                const w = this._drawToCache(C, o.DEFAULT_COLOR, o.DEFAULT_COLOR, o.DEFAULT_EXT);
                this._cacheMap.set(C, o.DEFAULT_COLOR, o.DEFAULT_COLOR, o.DEFAULT_EXT, w);
              }
            }));
          }
          beginFrame() {
            return this._requestClearModel;
          }
          clearTexture() {
            if (this._pages[0].currentRow.x !== 0 || this._pages[0].currentRow.y !== 0) {
              for (const v of this._pages) v.clear();
              this._cacheMap.clear(), this._cacheMapCombined.clear(), this._didWarmUp = !1;
            }
          }
          _createNewPage() {
            if (u.maxAtlasPages && this._pages.length >= Math.max(4, u.maxAtlasPages)) {
              const C = this._pages.filter(((k) => 2 * k.canvas.width <= (u.maxTextureSize || 4096))).sort(((k, M) => M.canvas.width !== k.canvas.width ? M.canvas.width - k.canvas.width : M.percentageUsed - k.percentageUsed));
              let w = -1, S = 0;
              for (let k = 0; k < C.length; k++) if (C[k].canvas.width !== S) w = k, S = C[k].canvas.width;
              else if (k - w == 3) break;
              const b = C.slice(w, w + 4), x = b.map(((k) => k.glyphs[0].texturePage)).sort(((k, M) => k > M ? 1 : -1)), A = this.pages.length - b.length, P = this._mergePages(b, A);
              P.version++;
              for (let k = x.length - 1; k >= 0; k--) this._deletePage(x[k]);
              this.pages.push(P), this._requestClearModel = !0, this._onAddTextureAtlasCanvas.fire(P.canvas);
            }
            const v = new p(this._document, this._textureSize);
            return this._pages.push(v), this._activePages.push(v), this._onAddTextureAtlasCanvas.fire(v.canvas), v;
          }
          _mergePages(v, C) {
            const w = 2 * v[0].canvas.width, S = new p(this._document, w, v);
            for (const [b, x] of v.entries()) {
              const A = b * x.canvas.width % w, P = Math.floor(b / 2) * x.canvas.height;
              S.ctx.drawImage(x.canvas, A, P);
              for (const M of x.glyphs) M.texturePage = C, M.sizeClipSpace.x = M.size.x / w, M.sizeClipSpace.y = M.size.y / w, M.texturePosition.x += A, M.texturePosition.y += P, M.texturePositionClipSpace.x = M.texturePosition.x / w, M.texturePositionClipSpace.y = M.texturePosition.y / w;
              this._onRemoveTextureAtlasCanvas.fire(x.canvas);
              const k = this._activePages.indexOf(x);
              k !== -1 && this._activePages.splice(k, 1);
            }
            return S;
          }
          _deletePage(v) {
            this._pages.splice(v, 1);
            for (let C = v; C < this._pages.length; C++) {
              const w = this._pages[C];
              for (const S of w.glyphs) S.texturePage--;
              w.version++;
            }
          }
          getRasterizedGlyphCombinedChar(v, C, w, S, b) {
            return this._getFromCacheMap(this._cacheMapCombined, v, C, w, S, b);
          }
          getRasterizedGlyph(v, C, w, S, b) {
            return this._getFromCacheMap(this._cacheMap, v, C, w, S, b);
          }
          _getFromCacheMap(v, C, w, S, b, x = !1) {
            return i = v.get(C, w, S, b), i || (i = this._drawToCache(C, w, S, b, x), v.set(C, w, S, b, i)), i;
          }
          _getColorFromAnsiIndex(v) {
            if (v >= this._config.colors.ansi.length) throw new Error("No color found for idx " + v);
            return this._config.colors.ansi[v];
          }
          _getBackgroundColor(v, C, w, S) {
            if (this._config.allowTransparency) return d.NULL_COLOR;
            let b;
            switch (v) {
              case 16777216:
              case 33554432:
                b = this._getColorFromAnsiIndex(C);
                break;
              case 50331648:
                const x = e.AttributeData.toColorRGB(C);
                b = d.channels.toColor(x[0], x[1], x[2]);
                break;
              default:
                b = w ? d.color.opaque(this._config.colors.foreground) : this._config.colors.background;
            }
            return b;
          }
          _getForegroundColor(v, C, w, S, b, x, A, P, k, M) {
            const y = this._getMinimumContrastColor(v, C, w, S, b, x, A, k, P, M);
            if (y) return y;
            let L;
            switch (b) {
              case 16777216:
              case 33554432:
                this._config.drawBoldTextInBrightColors && k && x < 8 && (x += 8), L = this._getColorFromAnsiIndex(x);
                break;
              case 50331648:
                const R = e.AttributeData.toColorRGB(x);
                L = d.channels.toColor(R[0], R[1], R[2]);
                break;
              default:
                L = A ? this._config.colors.background : this._config.colors.foreground;
            }
            return this._config.allowTransparency && (L = d.color.opaque(L)), P && (L = d.color.multiplyOpacity(L, c.DIM_OPACITY)), L;
          }
          _resolveBackgroundRgba(v, C, w) {
            switch (v) {
              case 16777216:
              case 33554432:
                return this._getColorFromAnsiIndex(C).rgba;
              case 50331648:
                return C << 8;
              default:
                return w ? this._config.colors.foreground.rgba : this._config.colors.background.rgba;
            }
          }
          _resolveForegroundRgba(v, C, w, S) {
            switch (v) {
              case 16777216:
              case 33554432:
                return this._config.drawBoldTextInBrightColors && S && C < 8 && (C += 8), this._getColorFromAnsiIndex(C).rgba;
              case 50331648:
                return C << 8;
              default:
                return w ? this._config.colors.background.rgba : this._config.colors.foreground.rgba;
            }
          }
          _getMinimumContrastColor(v, C, w, S, b, x, A, P, k, M) {
            if (this._config.minimumContrastRatio === 1 || M) return;
            const y = this._getContrastCache(k), L = y.getColor(v, S);
            if (L !== void 0) return L || void 0;
            const R = this._resolveBackgroundRgba(C, w, A), D = this._resolveForegroundRgba(b, x, A, P), F = d.rgba.ensureContrastRatio(R, D, this._config.minimumContrastRatio / (k ? 2 : 1));
            if (!F) return void y.setColor(v, S, null);
            const U = d.channels.toColor(F >> 24 & 255, F >> 16 & 255, F >> 8 & 255);
            return y.setColor(v, S, U), U;
          }
          _getContrastCache(v) {
            return v ? this._config.colors.halfContrastCache : this._config.colors.contrastCache;
          }
          _drawToCache(v, C, w, S, b = !1) {
            const x = typeof v == "number" ? String.fromCharCode(v) : v, A = Math.min(this._config.deviceCellWidth * Math.max(x.length, 2) + 4, this._textureSize);
            this._tmpCanvas.width < A && (this._tmpCanvas.width = A);
            const P = Math.min(this._config.deviceCellHeight + 8, this._textureSize);
            if (this._tmpCanvas.height < P && (this._tmpCanvas.height = P), this._tmpCtx.save(), this._workAttributeData.fg = w, this._workAttributeData.bg = C, this._workAttributeData.extended.ext = S, this._workAttributeData.isInvisible()) return s;
            const k = !!this._workAttributeData.isBold(), M = !!this._workAttributeData.isInverse(), y = !!this._workAttributeData.isDim(), L = !!this._workAttributeData.isItalic(), R = !!this._workAttributeData.isUnderline(), D = !!this._workAttributeData.isStrikethrough(), F = !!this._workAttributeData.isOverline();
            let U = this._workAttributeData.getFgColor(), K = this._workAttributeData.getFgColorMode(), q = this._workAttributeData.getBgColor(), O = this._workAttributeData.getBgColorMode();
            if (M) {
              const z = U;
              U = q, q = z;
              const Q = K;
              K = O, O = Q;
            }
            const E = this._getBackgroundColor(O, q, M, y);
            this._tmpCtx.globalCompositeOperation = "copy", this._tmpCtx.fillStyle = E.css, this._tmpCtx.fillRect(0, 0, this._tmpCanvas.width, this._tmpCanvas.height), this._tmpCtx.globalCompositeOperation = "source-over";
            const H = k ? this._config.fontWeightBold : this._config.fontWeight, N = L ? "italic" : "";
            this._tmpCtx.font = `${N} ${H} ${this._config.fontSize * this._config.devicePixelRatio}px ${this._config.fontFamily}`, this._tmpCtx.textBaseline = c.TEXT_BASELINE;
            const G = x.length === 1 && (0, r.isPowerlineGlyph)(x.charCodeAt(0)), j = x.length === 1 && (0, r.isRestrictedPowerlineGlyph)(x.charCodeAt(0)), ie = this._getForegroundColor(C, O, q, w, K, U, M, y, k, (0, r.treatGlyphAsBackgroundColor)(x.charCodeAt(0)));
            this._tmpCtx.fillStyle = ie.css;
            const V = j ? 0 : 4;
            let ae = !1;
            this._config.customGlyphs !== !1 && (ae = (0, h.tryDrawCustomChar)(this._tmpCtx, x, V, V, this._config.deviceCellWidth, this._config.deviceCellHeight, this._config.fontSize, this._config.devicePixelRatio));
            let ce, ee = !G;
            if (ce = typeof v == "number" ? this._unicodeService.wcwidth(v) : this._unicodeService.getStringCellWidth(v), R) {
              this._tmpCtx.save();
              const z = Math.max(1, Math.floor(this._config.fontSize * this._config.devicePixelRatio / 15)), Q = z % 2 == 1 ? 0.5 : 0;
              if (this._tmpCtx.lineWidth = z, this._workAttributeData.isUnderlineColorDefault()) this._tmpCtx.strokeStyle = this._tmpCtx.fillStyle;
              else if (this._workAttributeData.isUnderlineColorRGB()) ee = !1, this._tmpCtx.strokeStyle = `rgb(${e.AttributeData.toColorRGB(this._workAttributeData.getUnderlineColor()).join(",")})`;
              else {
                ee = !1;
                let le = this._workAttributeData.getUnderlineColor();
                this._config.drawBoldTextInBrightColors && this._workAttributeData.isBold() && le < 8 && (le += 8), this._tmpCtx.strokeStyle = this._getColorFromAnsiIndex(le).css;
              }
              this._tmpCtx.beginPath();
              const he = V, re = Math.ceil(V + this._config.deviceCharHeight) - Q - (b ? 2 * z : 0), fe = re + z, de = re + 2 * z;
              let ue = this._workAttributeData.getUnderlineVariantOffset();
              for (let le = 0; le < ce; le++) {
                this._tmpCtx.save();
                const se = he + le * this._config.deviceCellWidth, te = he + (le + 1) * this._config.deviceCellWidth, ve = se + this._config.deviceCellWidth / 2;
                switch (this._workAttributeData.extended.underlineStyle) {
                  case 2:
                    this._tmpCtx.moveTo(se, re), this._tmpCtx.lineTo(te, re), this._tmpCtx.moveTo(se, de), this._tmpCtx.lineTo(te, de);
                    break;
                  case 3:
                    const pe = z <= 1 ? de : Math.ceil(V + this._config.deviceCharHeight - z / 2) - Q, me = z <= 1 ? re : Math.ceil(V + this._config.deviceCharHeight + z / 2) - Q, we = new Path2D();
                    we.rect(se, re, this._config.deviceCellWidth, de - re), this._tmpCtx.clip(we), this._tmpCtx.moveTo(se - this._config.deviceCellWidth / 2, fe), this._tmpCtx.bezierCurveTo(se - this._config.deviceCellWidth / 2, me, se, me, se, fe), this._tmpCtx.bezierCurveTo(se, pe, ve, pe, ve, fe), this._tmpCtx.bezierCurveTo(ve, me, te, me, te, fe), this._tmpCtx.bezierCurveTo(te, pe, te + this._config.deviceCellWidth / 2, pe, te + this._config.deviceCellWidth / 2, fe);
                    break;
                  case 4:
                    const Ce = ue === 0 ? 0 : ue >= z ? 2 * z - ue : z - ue;
                    ue >= z || Ce === 0 ? (this._tmpCtx.setLineDash([Math.round(z), Math.round(z)]), this._tmpCtx.moveTo(se + Ce, re), this._tmpCtx.lineTo(te, re)) : (this._tmpCtx.setLineDash([Math.round(z), Math.round(z)]), this._tmpCtx.moveTo(se, re), this._tmpCtx.lineTo(se + Ce, re), this._tmpCtx.moveTo(se + Ce + z, re), this._tmpCtx.lineTo(te, re)), ue = (0, r.computeNextVariantOffset)(te - se, z, ue);
                    break;
                  case 5:
                    const Ee = 0.6, Re = 0.3, Se = te - se, be = Math.floor(Ee * Se), ye = Math.floor(Re * Se), Me = Se - be - ye;
                    this._tmpCtx.setLineDash([be, ye, Me]), this._tmpCtx.moveTo(se, re), this._tmpCtx.lineTo(te, re);
                    break;
                  default:
                    this._tmpCtx.moveTo(se, re), this._tmpCtx.lineTo(te, re);
                }
                this._tmpCtx.stroke(), this._tmpCtx.restore();
              }
              if (this._tmpCtx.restore(), !ae && this._config.fontSize >= 12 && !this._config.allowTransparency && x !== " ") {
                this._tmpCtx.save(), this._tmpCtx.textBaseline = "alphabetic";
                const le = this._tmpCtx.measureText(x);
                if (this._tmpCtx.restore(), "actualBoundingBoxDescent" in le && le.actualBoundingBoxDescent > 0) {
                  this._tmpCtx.save();
                  const se = new Path2D();
                  se.rect(he, re - Math.ceil(z / 2), this._config.deviceCellWidth * ce, de - re + Math.ceil(z / 2)), this._tmpCtx.clip(se), this._tmpCtx.lineWidth = 3 * this._config.devicePixelRatio, this._tmpCtx.strokeStyle = E.css, this._tmpCtx.strokeText(x, V, V + this._config.deviceCharHeight), this._tmpCtx.restore();
                }
              }
            }
            if (F) {
              const z = Math.max(1, Math.floor(this._config.fontSize * this._config.devicePixelRatio / 15)), Q = z % 2 == 1 ? 0.5 : 0;
              this._tmpCtx.lineWidth = z, this._tmpCtx.strokeStyle = this._tmpCtx.fillStyle, this._tmpCtx.beginPath(), this._tmpCtx.moveTo(V, V + Q), this._tmpCtx.lineTo(V + this._config.deviceCharWidth * ce, V + Q), this._tmpCtx.stroke();
            }
            if (ae || this._tmpCtx.fillText(x, V, V + this._config.deviceCharHeight), x === "_" && !this._config.allowTransparency) {
              let z = l(this._tmpCtx.getImageData(V, V, this._config.deviceCellWidth, this._config.deviceCellHeight), E, ie, ee);
              if (z) for (let Q = 1; Q <= 5 && (this._tmpCtx.save(), this._tmpCtx.fillStyle = E.css, this._tmpCtx.fillRect(0, 0, this._tmpCanvas.width, this._tmpCanvas.height), this._tmpCtx.restore(), this._tmpCtx.fillText(x, V, V + this._config.deviceCharHeight - Q), z = l(this._tmpCtx.getImageData(V, V, this._config.deviceCellWidth, this._config.deviceCellHeight), E, ie, ee), z); Q++) ;
            }
            if (D) {
              const z = Math.max(1, Math.floor(this._config.fontSize * this._config.devicePixelRatio / 10)), Q = this._tmpCtx.lineWidth % 2 == 1 ? 0.5 : 0;
              this._tmpCtx.lineWidth = z, this._tmpCtx.strokeStyle = this._tmpCtx.fillStyle, this._tmpCtx.beginPath(), this._tmpCtx.moveTo(V, V + Math.floor(this._config.deviceCharHeight / 2) - Q), this._tmpCtx.lineTo(V + this._config.deviceCharWidth * ce, V + Math.floor(this._config.deviceCharHeight / 2) - Q), this._tmpCtx.stroke();
            }
            this._tmpCtx.restore();
            const _e = this._tmpCtx.getImageData(0, 0, this._tmpCanvas.width, this._tmpCanvas.height);
            let ge;
            if (ge = this._config.allowTransparency ? (function(z) {
              for (let Q = 0; Q < z.data.length; Q += 4) if (z.data[Q + 3] > 0) return !1;
              return !0;
            })(_e) : l(_e, E, ie, ee), ge) return s;
            const Z = this._findGlyphBoundingBox(_e, this._workBoundingBox, A, j, ae, V);
            let X, J;
            for (; ; ) {
              if (this._activePages.length === 0) {
                const z = this._createNewPage();
                X = z, J = z.currentRow, J.height = Z.size.y;
                break;
              }
              X = this._activePages[this._activePages.length - 1], J = X.currentRow;
              for (const z of this._activePages) Z.size.y <= z.currentRow.height && (X = z, J = z.currentRow);
              for (let z = this._activePages.length - 1; z >= 0; z--) for (const Q of this._activePages[z].fixedRows) Q.height <= J.height && Z.size.y <= Q.height && (X = this._activePages[z], J = Q);
              if (J.y + Z.size.y >= X.canvas.height || J.height > Z.size.y + 2) {
                let z = !1;
                if (X.currentRow.y + X.currentRow.height + Z.size.y >= X.canvas.height) {
                  let Q;
                  for (const he of this._activePages) if (he.currentRow.y + he.currentRow.height + Z.size.y < he.canvas.height) {
                    Q = he;
                    break;
                  }
                  if (Q) X = Q;
                  else if (u.maxAtlasPages && this._pages.length >= u.maxAtlasPages && J.y + Z.size.y <= X.canvas.height && J.height >= Z.size.y && J.x + Z.size.x <= X.canvas.width) z = !0;
                  else {
                    const he = this._createNewPage();
                    X = he, J = he.currentRow, J.height = Z.size.y, z = !0;
                  }
                }
                z || (X.currentRow.height > 0 && X.fixedRows.push(X.currentRow), J = { x: 0, y: X.currentRow.y + X.currentRow.height, height: Z.size.y }, X.fixedRows.push(J), X.currentRow = { x: 0, y: J.y + J.height, height: 0 });
              }
              if (J.x + Z.size.x <= X.canvas.width) break;
              J === X.currentRow ? (J.x = 0, J.y += J.height, J.height = 0) : X.fixedRows.splice(X.fixedRows.indexOf(J), 1);
            }
            return Z.texturePage = this._pages.indexOf(X), Z.texturePosition.x = J.x, Z.texturePosition.y = J.y, Z.texturePositionClipSpace.x = J.x / X.canvas.width, Z.texturePositionClipSpace.y = J.y / X.canvas.height, Z.sizeClipSpace.x /= X.canvas.width, Z.sizeClipSpace.y /= X.canvas.height, J.height = Math.max(J.height, Z.size.y), J.x += Z.size.x, X.ctx.putImageData(_e, Z.texturePosition.x - this._workBoundingBox.left, Z.texturePosition.y - this._workBoundingBox.top, this._workBoundingBox.left, this._workBoundingBox.top, Z.size.x, Z.size.y), X.addGlyph(Z), X.version++, Z;
          }
          _findGlyphBoundingBox(v, C, w, S, b, x) {
            C.top = 0;
            const A = S ? this._config.deviceCellHeight : this._tmpCanvas.height, P = S ? this._config.deviceCellWidth : w;
            let k = !1;
            for (let M = 0; M < A; M++) {
              for (let y = 0; y < P; y++) {
                const L = M * this._tmpCanvas.width * 4 + 4 * y + 3;
                if (v.data[L] !== 0) {
                  C.top = M, k = !0;
                  break;
                }
              }
              if (k) break;
            }
            C.left = 0, k = !1;
            for (let M = 0; M < x + P; M++) {
              for (let y = 0; y < A; y++) {
                const L = y * this._tmpCanvas.width * 4 + 4 * M + 3;
                if (v.data[L] !== 0) {
                  C.left = M, k = !0;
                  break;
                }
              }
              if (k) break;
            }
            C.right = P, k = !1;
            for (let M = x + P - 1; M >= x; M--) {
              for (let y = 0; y < A; y++) {
                const L = y * this._tmpCanvas.width * 4 + 4 * M + 3;
                if (v.data[L] !== 0) {
                  C.right = M, k = !0;
                  break;
                }
              }
              if (k) break;
            }
            C.bottom = A, k = !1;
            for (let M = A - 1; M >= 0; M--) {
              for (let y = 0; y < P; y++) {
                const L = M * this._tmpCanvas.width * 4 + 4 * y + 3;
                if (v.data[L] !== 0) {
                  C.bottom = M, k = !0;
                  break;
                }
              }
              if (k) break;
            }
            return { texturePage: 0, texturePosition: { x: 0, y: 0 }, texturePositionClipSpace: { x: 0, y: 0 }, size: { x: C.right - C.left + 1, y: C.bottom - C.top + 1 }, sizeClipSpace: { x: C.right - C.left + 1, y: C.bottom - C.top + 1 }, offset: { x: -C.left + x + (S || b ? Math.floor((this._config.deviceCellWidth - this._config.deviceCharWidth) / 2) : 0), y: -C.top + x + (S || b ? this._config.lineHeight === 1 ? 0 : Math.round((this._config.deviceCellHeight - this._config.deviceCharHeight) / 2) : 0) } };
          }
        }
        t.TextureAtlas = u;
        class p {
          get percentageUsed() {
            return this._usedPixels / (this.canvas.width * this.canvas.height);
          }
          get glyphs() {
            return this._glyphs;
          }
          addGlyph(v) {
            this._glyphs.push(v), this._usedPixels += v.size.x * v.size.y;
          }
          constructor(v, C, w) {
            if (this._usedPixels = 0, this._glyphs = [], this.version = 0, this.currentRow = { x: 0, y: 0, height: 0 }, this.fixedRows = [], w) for (const S of w) this._glyphs.push(...S.glyphs), this._usedPixels += S._usedPixels;
            this.canvas = m(v, C, C), this.ctx = (0, r.throwIfFalsy)(this.canvas.getContext("2d", { alpha: !0 }));
          }
          clear() {
            this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height), this.currentRow.x = 0, this.currentRow.y = 0, this.currentRow.height = 0, this.fixedRows.length = 0, this.version++;
          }
        }
        function l(_, v, C, w) {
          const S = v.rgba >>> 24, b = v.rgba >>> 16 & 255, x = v.rgba >>> 8 & 255, A = C.rgba >>> 24, P = C.rgba >>> 16 & 255, k = C.rgba >>> 8 & 255, M = Math.floor((Math.abs(S - A) + Math.abs(b - P) + Math.abs(x - k)) / 12);
          let y = !0;
          for (let L = 0; L < _.data.length; L += 4) _.data[L] === S && _.data[L + 1] === b && _.data[L + 2] === x || w && Math.abs(_.data[L] - S) + Math.abs(_.data[L + 1] - b) + Math.abs(_.data[L + 2] - x) < M ? _.data[L + 3] = 0 : y = !1;
          return y;
        }
        function m(_, v, C) {
          const w = _.createElement("canvas");
          return w.width = v, w.height = C, w;
        }
      }, 160: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.contrastRatio = t.toPaddedHex = t.rgba = t.rgb = t.css = t.color = t.channels = t.NULL_COLOR = void 0;
        let a = 0, c = 0, h = 0, r = 0;
        var d, f, g, n, e;
        function o(i) {
          const u = i.toString(16);
          return u.length < 2 ? "0" + u : u;
        }
        function s(i, u) {
          return i < u ? (u + 0.05) / (i + 0.05) : (i + 0.05) / (u + 0.05);
        }
        t.NULL_COLOR = { css: "#00000000", rgba: 0 }, (function(i) {
          i.toCss = function(u, p, l, m) {
            return m !== void 0 ? `#${o(u)}${o(p)}${o(l)}${o(m)}` : `#${o(u)}${o(p)}${o(l)}`;
          }, i.toRgba = function(u, p, l, m = 255) {
            return (u << 24 | p << 16 | l << 8 | m) >>> 0;
          }, i.toColor = function(u, p, l, m) {
            return { css: i.toCss(u, p, l, m), rgba: i.toRgba(u, p, l, m) };
          };
        })(d || (t.channels = d = {})), (function(i) {
          function u(p, l) {
            return r = Math.round(255 * l), [a, c, h] = e.toChannels(p.rgba), { css: d.toCss(a, c, h, r), rgba: d.toRgba(a, c, h, r) };
          }
          i.blend = function(p, l) {
            if (r = (255 & l.rgba) / 255, r === 1) return { css: l.css, rgba: l.rgba };
            const m = l.rgba >> 24 & 255, _ = l.rgba >> 16 & 255, v = l.rgba >> 8 & 255, C = p.rgba >> 24 & 255, w = p.rgba >> 16 & 255, S = p.rgba >> 8 & 255;
            return a = C + Math.round((m - C) * r), c = w + Math.round((_ - w) * r), h = S + Math.round((v - S) * r), { css: d.toCss(a, c, h), rgba: d.toRgba(a, c, h) };
          }, i.isOpaque = function(p) {
            return (255 & p.rgba) == 255;
          }, i.ensureContrastRatio = function(p, l, m) {
            const _ = e.ensureContrastRatio(p.rgba, l.rgba, m);
            if (_) return d.toColor(_ >> 24 & 255, _ >> 16 & 255, _ >> 8 & 255);
          }, i.opaque = function(p) {
            const l = (255 | p.rgba) >>> 0;
            return [a, c, h] = e.toChannels(l), { css: d.toCss(a, c, h), rgba: l };
          }, i.opacity = u, i.multiplyOpacity = function(p, l) {
            return r = 255 & p.rgba, u(p, r * l / 255);
          }, i.toColorRGB = function(p) {
            return [p.rgba >> 24 & 255, p.rgba >> 16 & 255, p.rgba >> 8 & 255];
          };
        })(f || (t.color = f = {})), (function(i) {
          let u, p;
          try {
            const l = document.createElement("canvas");
            l.width = 1, l.height = 1;
            const m = l.getContext("2d", { willReadFrequently: !0 });
            m && (u = m, u.globalCompositeOperation = "copy", p = u.createLinearGradient(0, 0, 1, 1));
          } catch (l) {
          }
          i.toColor = function(l) {
            if (l.match(/#[\da-f]{3,8}/i)) switch (l.length) {
              case 4:
                return a = parseInt(l.slice(1, 2).repeat(2), 16), c = parseInt(l.slice(2, 3).repeat(2), 16), h = parseInt(l.slice(3, 4).repeat(2), 16), d.toColor(a, c, h);
              case 5:
                return a = parseInt(l.slice(1, 2).repeat(2), 16), c = parseInt(l.slice(2, 3).repeat(2), 16), h = parseInt(l.slice(3, 4).repeat(2), 16), r = parseInt(l.slice(4, 5).repeat(2), 16), d.toColor(a, c, h, r);
              case 7:
                return { css: l, rgba: (parseInt(l.slice(1), 16) << 8 | 255) >>> 0 };
              case 9:
                return { css: l, rgba: parseInt(l.slice(1), 16) >>> 0 };
            }
            const m = l.match(/rgba?\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(,\s*(0|1|\d?\.(\d+))\s*)?\)/);
            if (m) return a = parseInt(m[1]), c = parseInt(m[2]), h = parseInt(m[3]), r = Math.round(255 * (m[5] === void 0 ? 1 : parseFloat(m[5]))), d.toColor(a, c, h, r);
            if (!u || !p) throw new Error("css.toColor: Unsupported css format");
            if (u.fillStyle = p, u.fillStyle = l, typeof u.fillStyle != "string") throw new Error("css.toColor: Unsupported css format");
            if (u.fillRect(0, 0, 1, 1), [a, c, h, r] = u.getImageData(0, 0, 1, 1).data, r !== 255) throw new Error("css.toColor: Unsupported css format");
            return { rgba: d.toRgba(a, c, h, r), css: l };
          };
        })(g || (t.css = g = {})), (function(i) {
          function u(p, l, m) {
            const _ = p / 255, v = l / 255, C = m / 255;
            return 0.2126 * (_ <= 0.03928 ? _ / 12.92 : Math.pow((_ + 0.055) / 1.055, 2.4)) + 0.7152 * (v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)) + 0.0722 * (C <= 0.03928 ? C / 12.92 : Math.pow((C + 0.055) / 1.055, 2.4));
          }
          i.relativeLuminance = function(p) {
            return u(p >> 16 & 255, p >> 8 & 255, 255 & p);
          }, i.relativeLuminance2 = u;
        })(n || (t.rgb = n = {})), (function(i) {
          function u(l, m, _) {
            const v = l >> 24 & 255, C = l >> 16 & 255, w = l >> 8 & 255;
            let S = m >> 24 & 255, b = m >> 16 & 255, x = m >> 8 & 255, A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            for (; A < _ && (S > 0 || b > 0 || x > 0); ) S -= Math.max(0, Math.ceil(0.1 * S)), b -= Math.max(0, Math.ceil(0.1 * b)), x -= Math.max(0, Math.ceil(0.1 * x)), A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            return (S << 24 | b << 16 | x << 8 | 255) >>> 0;
          }
          function p(l, m, _) {
            const v = l >> 24 & 255, C = l >> 16 & 255, w = l >> 8 & 255;
            let S = m >> 24 & 255, b = m >> 16 & 255, x = m >> 8 & 255, A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            for (; A < _ && (S < 255 || b < 255 || x < 255); ) S = Math.min(255, S + Math.ceil(0.1 * (255 - S))), b = Math.min(255, b + Math.ceil(0.1 * (255 - b))), x = Math.min(255, x + Math.ceil(0.1 * (255 - x))), A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            return (S << 24 | b << 16 | x << 8 | 255) >>> 0;
          }
          i.blend = function(l, m) {
            if (r = (255 & m) / 255, r === 1) return m;
            const _ = m >> 24 & 255, v = m >> 16 & 255, C = m >> 8 & 255, w = l >> 24 & 255, S = l >> 16 & 255, b = l >> 8 & 255;
            return a = w + Math.round((_ - w) * r), c = S + Math.round((v - S) * r), h = b + Math.round((C - b) * r), d.toRgba(a, c, h);
          }, i.ensureContrastRatio = function(l, m, _) {
            const v = n.relativeLuminance(l >> 8), C = n.relativeLuminance(m >> 8);
            if (s(v, C) < _) {
              if (C < v) {
                const b = u(l, m, _), x = s(v, n.relativeLuminance(b >> 8));
                if (x < _) {
                  const A = p(l, m, _);
                  return x > s(v, n.relativeLuminance(A >> 8)) ? b : A;
                }
                return b;
              }
              const w = p(l, m, _), S = s(v, n.relativeLuminance(w >> 8));
              if (S < _) {
                const b = u(l, m, _);
                return S > s(v, n.relativeLuminance(b >> 8)) ? w : b;
              }
              return w;
            }
          }, i.reduceLuminance = u, i.increaseLuminance = p, i.toChannels = function(l) {
            return [l >> 24 & 255, l >> 16 & 255, l >> 8 & 255, 255 & l];
          };
        })(e || (t.rgba = e = {})), t.toPaddedHex = o, t.contrastRatio = s;
      }, 345: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.runAndSubscribe = t.forwardEvent = t.EventEmitter = void 0, t.EventEmitter = class {
          constructor() {
            this._listeners = [], this._disposed = !1;
          }
          get event() {
            return this._event || (this._event = (a) => (this._listeners.push(a), { dispose: () => {
              if (!this._disposed) {
                for (let c = 0; c < this._listeners.length; c++) if (this._listeners[c] === a) return void this._listeners.splice(c, 1);
              }
            } })), this._event;
          }
          fire(a, c) {
            const h = [];
            for (let r = 0; r < this._listeners.length; r++) h.push(this._listeners[r]);
            for (let r = 0; r < h.length; r++) h[r].call(void 0, a, c);
          }
          dispose() {
            this.clearListeners(), this._disposed = !0;
          }
          clearListeners() {
            this._listeners && (this._listeners.length = 0);
          }
        }, t.forwardEvent = function(a, c) {
          return a(((h) => c.fire(h)));
        }, t.runAndSubscribe = function(a, c) {
          return c(void 0), a(((h) => c(h)));
        };
      }, 859: (T, t) => {
        function a(c) {
          for (const h of c) h.dispose();
          c.length = 0;
        }
        Object.defineProperty(t, "__esModule", { value: !0 }), t.getDisposeArrayDisposable = t.disposeArray = t.toDisposable = t.MutableDisposable = t.Disposable = void 0, t.Disposable = class {
          constructor() {
            this._disposables = [], this._isDisposed = !1;
          }
          dispose() {
            this._isDisposed = !0;
            for (const c of this._disposables) c.dispose();
            this._disposables.length = 0;
          }
          register(c) {
            return this._disposables.push(c), c;
          }
          unregister(c) {
            const h = this._disposables.indexOf(c);
            h !== -1 && this._disposables.splice(h, 1);
          }
        }, t.MutableDisposable = class {
          constructor() {
            this._isDisposed = !1;
          }
          get value() {
            return this._isDisposed ? void 0 : this._value;
          }
          set value(c) {
            var h;
            this._isDisposed || c === this._value || ((h = this._value) == null || h.dispose(), this._value = c);
          }
          clear() {
            this.value = void 0;
          }
          dispose() {
            var c;
            this._isDisposed = !0, (c = this._value) == null || c.dispose(), this._value = void 0;
          }
        }, t.toDisposable = function(c) {
          return { dispose: c };
        }, t.disposeArray = a, t.getDisposeArrayDisposable = function(c) {
          return { dispose: () => a(c) };
        };
      }, 485: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.FourKeyMap = t.TwoKeyMap = void 0;
        class a {
          constructor() {
            this._data = {};
          }
          set(h, r, d) {
            this._data[h] || (this._data[h] = {}), this._data[h][r] = d;
          }
          get(h, r) {
            return this._data[h] ? this._data[h][r] : void 0;
          }
          clear() {
            this._data = {};
          }
        }
        t.TwoKeyMap = a, t.FourKeyMap = class {
          constructor() {
            this._data = new a();
          }
          set(c, h, r, d, f) {
            this._data.get(c, h) || this._data.set(c, h, new a()), this._data.get(c, h).set(r, d, f);
          }
          get(c, h, r, d) {
            var f;
            return (f = this._data.get(c, h)) == null ? void 0 : f.get(r, d);
          }
          clear() {
            this._data.clear();
          }
        };
      }, 399: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.isChromeOS = t.isLinux = t.isWindows = t.isIphone = t.isIpad = t.isMac = t.getSafariVersion = t.isSafari = t.isLegacyEdge = t.isFirefox = t.isNode = void 0, t.isNode = typeof process != "undefined" && "title" in process;
        const a = t.isNode ? "node" : navigator.userAgent, c = t.isNode ? "node" : navigator.platform;
        t.isFirefox = a.includes("Firefox"), t.isLegacyEdge = a.includes("Edge"), t.isSafari = /^((?!chrome|android).)*safari/i.test(a), t.getSafariVersion = function() {
          if (!t.isSafari) return 0;
          const h = a.match(/Version\/(\d+)/);
          return h === null || h.length < 2 ? 0 : parseInt(h[1]);
        }, t.isMac = ["Macintosh", "MacIntel", "MacPPC", "Mac68K"].includes(c), t.isIpad = c === "iPad", t.isIphone = c === "iPhone", t.isWindows = ["Windows", "Win16", "Win32", "WinCE"].includes(c), t.isLinux = c.indexOf("Linux") >= 0, t.isChromeOS = /\bCrOS\b/.test(a);
      }, 385: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.DebouncedIdleTask = t.IdleTaskQueue = t.PriorityTaskQueue = void 0;
        const c = a(399);
        class h {
          constructor() {
            this._tasks = [], this._i = 0;
          }
          enqueue(f) {
            this._tasks.push(f), this._start();
          }
          flush() {
            for (; this._i < this._tasks.length; ) this._tasks[this._i]() || this._i++;
            this.clear();
          }
          clear() {
            this._idleCallback && (this._cancelCallback(this._idleCallback), this._idleCallback = void 0), this._i = 0, this._tasks.length = 0;
          }
          _start() {
            this._idleCallback || (this._idleCallback = this._requestCallback(this._process.bind(this)));
          }
          _process(f) {
            this._idleCallback = void 0;
            let g = 0, n = 0, e = f.timeRemaining(), o = 0;
            for (; this._i < this._tasks.length; ) {
              if (g = Date.now(), this._tasks[this._i]() || this._i++, g = Math.max(1, Date.now() - g), n = Math.max(g, n), o = f.timeRemaining(), 1.5 * n > o) return e - g < -20 && console.warn(`task queue exceeded allotted deadline by ${Math.abs(Math.round(e - g))}ms`), void this._start();
              e = o;
            }
            this.clear();
          }
        }
        class r extends h {
          _requestCallback(f) {
            return setTimeout((() => f(this._createDeadline(16))));
          }
          _cancelCallback(f) {
            clearTimeout(f);
          }
          _createDeadline(f) {
            const g = Date.now() + f;
            return { timeRemaining: () => Math.max(0, g - Date.now()) };
          }
        }
        t.PriorityTaskQueue = r, t.IdleTaskQueue = !c.isNode && "requestIdleCallback" in window ? class extends h {
          _requestCallback(d) {
            return requestIdleCallback(d);
          }
          _cancelCallback(d) {
            cancelIdleCallback(d);
          }
        } : r, t.DebouncedIdleTask = class {
          constructor() {
            this._queue = new t.IdleTaskQueue();
          }
          set(d) {
            this._queue.clear(), this._queue.enqueue(d);
          }
          flush() {
            this._queue.flush();
          }
        };
      }, 147: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.ExtendedAttrs = t.AttributeData = void 0;
        class a {
          constructor() {
            this.fg = 0, this.bg = 0, this.extended = new c();
          }
          static toColorRGB(r) {
            return [r >>> 16 & 255, r >>> 8 & 255, 255 & r];
          }
          static fromColorRGB(r) {
            return (255 & r[0]) << 16 | (255 & r[1]) << 8 | 255 & r[2];
          }
          clone() {
            const r = new a();
            return r.fg = this.fg, r.bg = this.bg, r.extended = this.extended.clone(), r;
          }
          isInverse() {
            return 67108864 & this.fg;
          }
          isBold() {
            return 134217728 & this.fg;
          }
          isUnderline() {
            return this.hasExtendedAttrs() && this.extended.underlineStyle !== 0 ? 1 : 268435456 & this.fg;
          }
          isBlink() {
            return 536870912 & this.fg;
          }
          isInvisible() {
            return 1073741824 & this.fg;
          }
          isItalic() {
            return 67108864 & this.bg;
          }
          isDim() {
            return 134217728 & this.bg;
          }
          isStrikethrough() {
            return 2147483648 & this.fg;
          }
          isProtected() {
            return 536870912 & this.bg;
          }
          isOverline() {
            return 1073741824 & this.bg;
          }
          getFgColorMode() {
            return 50331648 & this.fg;
          }
          getBgColorMode() {
            return 50331648 & this.bg;
          }
          isFgRGB() {
            return (50331648 & this.fg) == 50331648;
          }
          isBgRGB() {
            return (50331648 & this.bg) == 50331648;
          }
          isFgPalette() {
            return (50331648 & this.fg) == 16777216 || (50331648 & this.fg) == 33554432;
          }
          isBgPalette() {
            return (50331648 & this.bg) == 16777216 || (50331648 & this.bg) == 33554432;
          }
          isFgDefault() {
            return (50331648 & this.fg) == 0;
          }
          isBgDefault() {
            return (50331648 & this.bg) == 0;
          }
          isAttributeDefault() {
            return this.fg === 0 && this.bg === 0;
          }
          getFgColor() {
            switch (50331648 & this.fg) {
              case 16777216:
              case 33554432:
                return 255 & this.fg;
              case 50331648:
                return 16777215 & this.fg;
              default:
                return -1;
            }
          }
          getBgColor() {
            switch (50331648 & this.bg) {
              case 16777216:
              case 33554432:
                return 255 & this.bg;
              case 50331648:
                return 16777215 & this.bg;
              default:
                return -1;
            }
          }
          hasExtendedAttrs() {
            return 268435456 & this.bg;
          }
          updateExtended() {
            this.extended.isEmpty() ? this.bg &= -268435457 : this.bg |= 268435456;
          }
          getUnderlineColor() {
            if (268435456 & this.bg && ~this.extended.underlineColor) switch (50331648 & this.extended.underlineColor) {
              case 16777216:
              case 33554432:
                return 255 & this.extended.underlineColor;
              case 50331648:
                return 16777215 & this.extended.underlineColor;
              default:
                return this.getFgColor();
            }
            return this.getFgColor();
          }
          getUnderlineColorMode() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? 50331648 & this.extended.underlineColor : this.getFgColorMode();
          }
          isUnderlineColorRGB() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? (50331648 & this.extended.underlineColor) == 50331648 : this.isFgRGB();
          }
          isUnderlineColorPalette() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? (50331648 & this.extended.underlineColor) == 16777216 || (50331648 & this.extended.underlineColor) == 33554432 : this.isFgPalette();
          }
          isUnderlineColorDefault() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? (50331648 & this.extended.underlineColor) == 0 : this.isFgDefault();
          }
          getUnderlineStyle() {
            return 268435456 & this.fg ? 268435456 & this.bg ? this.extended.underlineStyle : 1 : 0;
          }
          getUnderlineVariantOffset() {
            return this.extended.underlineVariantOffset;
          }
        }
        t.AttributeData = a;
        class c {
          get ext() {
            return this._urlId ? -469762049 & this._ext | this.underlineStyle << 26 : this._ext;
          }
          set ext(r) {
            this._ext = r;
          }
          get underlineStyle() {
            return this._urlId ? 5 : (469762048 & this._ext) >> 26;
          }
          set underlineStyle(r) {
            this._ext &= -469762049, this._ext |= r << 26 & 469762048;
          }
          get underlineColor() {
            return 67108863 & this._ext;
          }
          set underlineColor(r) {
            this._ext &= -67108864, this._ext |= 67108863 & r;
          }
          get urlId() {
            return this._urlId;
          }
          set urlId(r) {
            this._urlId = r;
          }
          get underlineVariantOffset() {
            const r = (3758096384 & this._ext) >> 29;
            return r < 0 ? 4294967288 ^ r : r;
          }
          set underlineVariantOffset(r) {
            this._ext &= 536870911, this._ext |= r << 29 & 3758096384;
          }
          constructor(r = 0, d = 0) {
            this._ext = 0, this._urlId = 0, this._ext = r, this._urlId = d;
          }
          clone() {
            return new c(this._ext, this._urlId);
          }
          isEmpty() {
            return this.underlineStyle === 0 && this._urlId === 0;
          }
        }
        t.ExtendedAttrs = c;
      }, 782: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CellData = void 0;
        const c = a(133), h = a(855), r = a(147);
        class d extends r.AttributeData {
          constructor() {
            super(...arguments), this.content = 0, this.fg = 0, this.bg = 0, this.extended = new r.ExtendedAttrs(), this.combinedData = "";
          }
          static fromCharData(g) {
            const n = new d();
            return n.setFromCharData(g), n;
          }
          isCombined() {
            return 2097152 & this.content;
          }
          getWidth() {
            return this.content >> 22;
          }
          getChars() {
            return 2097152 & this.content ? this.combinedData : 2097151 & this.content ? (0, c.stringFromCodePoint)(2097151 & this.content) : "";
          }
          getCode() {
            return this.isCombined() ? this.combinedData.charCodeAt(this.combinedData.length - 1) : 2097151 & this.content;
          }
          setFromCharData(g) {
            this.fg = g[h.CHAR_DATA_ATTR_INDEX], this.bg = 0;
            let n = !1;
            if (g[h.CHAR_DATA_CHAR_INDEX].length > 2) n = !0;
            else if (g[h.CHAR_DATA_CHAR_INDEX].length === 2) {
              const e = g[h.CHAR_DATA_CHAR_INDEX].charCodeAt(0);
              if (55296 <= e && e <= 56319) {
                const o = g[h.CHAR_DATA_CHAR_INDEX].charCodeAt(1);
                56320 <= o && o <= 57343 ? this.content = 1024 * (e - 55296) + o - 56320 + 65536 | g[h.CHAR_DATA_WIDTH_INDEX] << 22 : n = !0;
              } else n = !0;
            } else this.content = g[h.CHAR_DATA_CHAR_INDEX].charCodeAt(0) | g[h.CHAR_DATA_WIDTH_INDEX] << 22;
            n && (this.combinedData = g[h.CHAR_DATA_CHAR_INDEX], this.content = 2097152 | g[h.CHAR_DATA_WIDTH_INDEX] << 22);
          }
          getAsCharData() {
            return [this.fg, this.getChars(), this.getWidth(), this.getCode()];
          }
        }
        t.CellData = d;
      }, 855: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.WHITESPACE_CELL_CODE = t.WHITESPACE_CELL_WIDTH = t.WHITESPACE_CELL_CHAR = t.NULL_CELL_CODE = t.NULL_CELL_WIDTH = t.NULL_CELL_CHAR = t.CHAR_DATA_CODE_INDEX = t.CHAR_DATA_WIDTH_INDEX = t.CHAR_DATA_CHAR_INDEX = t.CHAR_DATA_ATTR_INDEX = t.DEFAULT_EXT = t.DEFAULT_ATTR = t.DEFAULT_COLOR = void 0, t.DEFAULT_COLOR = 0, t.DEFAULT_ATTR = 256 | t.DEFAULT_COLOR << 9, t.DEFAULT_EXT = 0, t.CHAR_DATA_ATTR_INDEX = 0, t.CHAR_DATA_CHAR_INDEX = 1, t.CHAR_DATA_WIDTH_INDEX = 2, t.CHAR_DATA_CODE_INDEX = 3, t.NULL_CELL_CHAR = "", t.NULL_CELL_WIDTH = 1, t.NULL_CELL_CODE = 0, t.WHITESPACE_CELL_CHAR = " ", t.WHITESPACE_CELL_WIDTH = 1, t.WHITESPACE_CELL_CODE = 32;
      }, 133: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.Utf8ToUtf32 = t.StringToUtf32 = t.utf32ToString = t.stringFromCodePoint = void 0, t.stringFromCodePoint = function(a) {
          return a > 65535 ? (a -= 65536, String.fromCharCode(55296 + (a >> 10)) + String.fromCharCode(a % 1024 + 56320)) : String.fromCharCode(a);
        }, t.utf32ToString = function(a, c = 0, h = a.length) {
          let r = "";
          for (let d = c; d < h; ++d) {
            let f = a[d];
            f > 65535 ? (f -= 65536, r += String.fromCharCode(55296 + (f >> 10)) + String.fromCharCode(f % 1024 + 56320)) : r += String.fromCharCode(f);
          }
          return r;
        }, t.StringToUtf32 = class {
          constructor() {
            this._interim = 0;
          }
          clear() {
            this._interim = 0;
          }
          decode(a, c) {
            const h = a.length;
            if (!h) return 0;
            let r = 0, d = 0;
            if (this._interim) {
              const f = a.charCodeAt(d++);
              56320 <= f && f <= 57343 ? c[r++] = 1024 * (this._interim - 55296) + f - 56320 + 65536 : (c[r++] = this._interim, c[r++] = f), this._interim = 0;
            }
            for (let f = d; f < h; ++f) {
              const g = a.charCodeAt(f);
              if (55296 <= g && g <= 56319) {
                if (++f >= h) return this._interim = g, r;
                const n = a.charCodeAt(f);
                56320 <= n && n <= 57343 ? c[r++] = 1024 * (g - 55296) + n - 56320 + 65536 : (c[r++] = g, c[r++] = n);
              } else g !== 65279 && (c[r++] = g);
            }
            return r;
          }
        }, t.Utf8ToUtf32 = class {
          constructor() {
            this.interim = new Uint8Array(3);
          }
          clear() {
            this.interim.fill(0);
          }
          decode(a, c) {
            const h = a.length;
            if (!h) return 0;
            let r, d, f, g, n = 0, e = 0, o = 0;
            if (this.interim[0]) {
              let u = !1, p = this.interim[0];
              p &= (224 & p) == 192 ? 31 : (240 & p) == 224 ? 15 : 7;
              let l, m = 0;
              for (; (l = 63 & this.interim[++m]) && m < 4; ) p <<= 6, p |= l;
              const _ = (224 & this.interim[0]) == 192 ? 2 : (240 & this.interim[0]) == 224 ? 3 : 4, v = _ - m;
              for (; o < v; ) {
                if (o >= h) return 0;
                if (l = a[o++], (192 & l) != 128) {
                  o--, u = !0;
                  break;
                }
                this.interim[m++] = l, p <<= 6, p |= 63 & l;
              }
              u || (_ === 2 ? p < 128 ? o-- : c[n++] = p : _ === 3 ? p < 2048 || p >= 55296 && p <= 57343 || p === 65279 || (c[n++] = p) : p < 65536 || p > 1114111 || (c[n++] = p)), this.interim.fill(0);
            }
            const s = h - 4;
            let i = o;
            for (; i < h; ) {
              for (; !(!(i < s) || 128 & (r = a[i]) || 128 & (d = a[i + 1]) || 128 & (f = a[i + 2]) || 128 & (g = a[i + 3])); ) c[n++] = r, c[n++] = d, c[n++] = f, c[n++] = g, i += 4;
              if (r = a[i++], r < 128) c[n++] = r;
              else if ((224 & r) == 192) {
                if (i >= h) return this.interim[0] = r, n;
                if (d = a[i++], (192 & d) != 128) {
                  i--;
                  continue;
                }
                if (e = (31 & r) << 6 | 63 & d, e < 128) {
                  i--;
                  continue;
                }
                c[n++] = e;
              } else if ((240 & r) == 224) {
                if (i >= h) return this.interim[0] = r, n;
                if (d = a[i++], (192 & d) != 128) {
                  i--;
                  continue;
                }
                if (i >= h) return this.interim[0] = r, this.interim[1] = d, n;
                if (f = a[i++], (192 & f) != 128) {
                  i--;
                  continue;
                }
                if (e = (15 & r) << 12 | (63 & d) << 6 | 63 & f, e < 2048 || e >= 55296 && e <= 57343 || e === 65279) continue;
                c[n++] = e;
              } else if ((248 & r) == 240) {
                if (i >= h) return this.interim[0] = r, n;
                if (d = a[i++], (192 & d) != 128) {
                  i--;
                  continue;
                }
                if (i >= h) return this.interim[0] = r, this.interim[1] = d, n;
                if (f = a[i++], (192 & f) != 128) {
                  i--;
                  continue;
                }
                if (i >= h) return this.interim[0] = r, this.interim[1] = d, this.interim[2] = f, n;
                if (g = a[i++], (192 & g) != 128) {
                  i--;
                  continue;
                }
                if (e = (7 & r) << 18 | (63 & d) << 12 | (63 & f) << 6 | 63 & g, e < 65536 || e > 1114111) continue;
                c[n++] = e;
              }
            }
            return n;
          }
        };
      }, 776: function(T, t, a) {
        var c = this && this.__decorate || function(e, o, s, i) {
          var u, p = arguments.length, l = p < 3 ? o : i === null ? i = Object.getOwnPropertyDescriptor(o, s) : i;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") l = Reflect.decorate(e, o, s, i);
          else for (var m = e.length - 1; m >= 0; m--) (u = e[m]) && (l = (p < 3 ? u(l) : p > 3 ? u(o, s, l) : u(o, s)) || l);
          return p > 3 && l && Object.defineProperty(o, s, l), l;
        }, h = this && this.__param || function(e, o) {
          return function(s, i) {
            o(s, i, e);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.traceCall = t.setTraceLogger = t.LogService = void 0;
        const r = a(859), d = a(97), f = { trace: d.LogLevelEnum.TRACE, debug: d.LogLevelEnum.DEBUG, info: d.LogLevelEnum.INFO, warn: d.LogLevelEnum.WARN, error: d.LogLevelEnum.ERROR, off: d.LogLevelEnum.OFF };
        let g, n = t.LogService = class extends r.Disposable {
          get logLevel() {
            return this._logLevel;
          }
          constructor(e) {
            super(), this._optionsService = e, this._logLevel = d.LogLevelEnum.OFF, this._updateLogLevel(), this.register(this._optionsService.onSpecificOptionChange("logLevel", (() => this._updateLogLevel()))), g = this;
          }
          _updateLogLevel() {
            this._logLevel = f[this._optionsService.rawOptions.logLevel];
          }
          _evalLazyOptionalParams(e) {
            for (let o = 0; o < e.length; o++) typeof e[o] == "function" && (e[o] = e[o]());
          }
          _log(e, o, s) {
            this._evalLazyOptionalParams(s), e.call(console, (this._optionsService.options.logger ? "" : "xterm.js: ") + o, ...s);
          }
          trace(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.TRACE && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.trace.bind(this._optionsService.options.logger)) != null ? i : console.log, e, o);
          }
          debug(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.DEBUG && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.debug.bind(this._optionsService.options.logger)) != null ? i : console.log, e, o);
          }
          info(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.INFO && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.info.bind(this._optionsService.options.logger)) != null ? i : console.info, e, o);
          }
          warn(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.WARN && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.warn.bind(this._optionsService.options.logger)) != null ? i : console.warn, e, o);
          }
          error(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.ERROR && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.error.bind(this._optionsService.options.logger)) != null ? i : console.error, e, o);
          }
        };
        t.LogService = n = c([h(0, d.IOptionsService)], n), t.setTraceLogger = function(e) {
          g = e;
        }, t.traceCall = function(e, o, s) {
          if (typeof s.value != "function") throw new Error("not supported");
          const i = s.value;
          s.value = function(...u) {
            if (g.logLevel !== d.LogLevelEnum.TRACE) return i.apply(this, u);
            g.trace(`GlyphRenderer#${i.name}(${u.map(((l) => JSON.stringify(l))).join(", ")})`);
            const p = i.apply(this, u);
            return g.trace(`GlyphRenderer#${i.name} return`, p), p;
          };
        };
      }, 726: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.createDecorator = t.getServiceDependencies = t.serviceRegistry = void 0;
        const a = "di$target", c = "di$dependencies";
        t.serviceRegistry = /* @__PURE__ */ new Map(), t.getServiceDependencies = function(h) {
          return h[c] || [];
        }, t.createDecorator = function(h) {
          if (t.serviceRegistry.has(h)) return t.serviceRegistry.get(h);
          const r = function(d, f, g) {
            if (arguments.length !== 3) throw new Error("@IServiceName-decorator can only be used to decorate a parameter");
            (function(n, e, o) {
              e[a] === e ? e[c].push({ id: n, index: o }) : (e[c] = [{ id: n, index: o }], e[a] = e);
            })(r, d, g);
          };
          return r.toString = () => h, t.serviceRegistry.set(h, r), r;
        };
      }, 97: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.IDecorationService = t.IUnicodeService = t.IOscLinkService = t.IOptionsService = t.ILogService = t.LogLevelEnum = t.IInstantiationService = t.ICharsetService = t.ICoreService = t.ICoreMouseService = t.IBufferService = void 0;
        const c = a(726);
        var h;
        t.IBufferService = (0, c.createDecorator)("BufferService"), t.ICoreMouseService = (0, c.createDecorator)("CoreMouseService"), t.ICoreService = (0, c.createDecorator)("CoreService"), t.ICharsetService = (0, c.createDecorator)("CharsetService"), t.IInstantiationService = (0, c.createDecorator)("InstantiationService"), (function(r) {
          r[r.TRACE = 0] = "TRACE", r[r.DEBUG = 1] = "DEBUG", r[r.INFO = 2] = "INFO", r[r.WARN = 3] = "WARN", r[r.ERROR = 4] = "ERROR", r[r.OFF = 5] = "OFF";
        })(h || (t.LogLevelEnum = h = {})), t.ILogService = (0, c.createDecorator)("LogService"), t.IOptionsService = (0, c.createDecorator)("OptionsService"), t.IOscLinkService = (0, c.createDecorator)("OscLinkService"), t.IUnicodeService = (0, c.createDecorator)("UnicodeService"), t.IDecorationService = (0, c.createDecorator)("DecorationService");
      } }, $ = {};
      function W(T) {
        var t = $[T];
        if (t !== void 0) return t.exports;
        var a = $[T] = { exports: {} };
        return I[T].call(a.exports, a, a.exports, W), a.exports;
      }
      var Y = {};
      return (() => {
        var T = Y;
        Object.defineProperty(T, "__esModule", { value: !0 }), T.WebglAddon = void 0;
        const t = W(345), a = W(859), c = W(399), h = W(666), r = W(776);
        class d extends a.Disposable {
          constructor(g) {
            if (c.isSafari && (0, c.getSafariVersion)() < 16) {
              const n = { antialias: !1, depth: !1, preserveDrawingBuffer: !0 };
              if (!document.createElement("canvas").getContext("webgl2", n)) throw new Error("Webgl2 is only supported on Safari 16 and above");
            }
            super(), this._preserveDrawingBuffer = g, this._onChangeTextureAtlas = this.register(new t.EventEmitter()), this.onChangeTextureAtlas = this._onChangeTextureAtlas.event, this._onAddTextureAtlasCanvas = this.register(new t.EventEmitter()), this.onAddTextureAtlasCanvas = this._onAddTextureAtlasCanvas.event, this._onRemoveTextureAtlasCanvas = this.register(new t.EventEmitter()), this.onRemoveTextureAtlasCanvas = this._onRemoveTextureAtlasCanvas.event, this._onContextLoss = this.register(new t.EventEmitter()), this.onContextLoss = this._onContextLoss.event;
          }
          activate(g) {
            const n = g._core;
            if (!g.element) return void this.register(n.onWillOpen((() => this.activate(g))));
            this._terminal = g;
            const e = n.coreService, o = n.optionsService, s = n, i = s._renderService, u = s._characterJoinerService, p = s._charSizeService, l = s._coreBrowserService, m = s._decorationService, _ = s._logService, v = s._themeService;
            (0, r.setTraceLogger)(_), this._renderer = this.register(new h.WebglRenderer(g, u, p, l, e, m, o, v, this._preserveDrawingBuffer)), this.register((0, t.forwardEvent)(this._renderer.onContextLoss, this._onContextLoss)), this.register((0, t.forwardEvent)(this._renderer.onChangeTextureAtlas, this._onChangeTextureAtlas)), this.register((0, t.forwardEvent)(this._renderer.onAddTextureAtlasCanvas, this._onAddTextureAtlasCanvas)), this.register((0, t.forwardEvent)(this._renderer.onRemoveTextureAtlasCanvas, this._onRemoveTextureAtlasCanvas)), i.setRenderer(this._renderer), this.register((0, a.toDisposable)((() => {
              const C = this._terminal._core._renderService;
              C.setRenderer(this._terminal._core._createRenderer()), C.handleResize(g.cols, g.rows);
            })));
          }
          get textureAtlas() {
            var g;
            return (g = this._renderer) == null ? void 0 : g.textureAtlas;
          }
          clearTextureAtlas() {
            var g;
            (g = this._renderer) == null || g.clearTextureAtlas();
          }
        }
        T.WebglAddon = d;
      })(), Y;
    })()));
  })(Te)), Te.exports;
}
var ot = nt(), Be = { exports: {} }, je;
function at() {
  return je || (je = 1, (function(ne, B) {
    (function(I, $) {
      ne.exports = $();
    })(self, (() => (() => {
      var I = { 903: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.BaseRenderLayer = void 0;
        const c = a(274), h = a(627), r = a(237), d = a(860), f = a(374), g = a(296), n = a(345), e = a(859), o = a(399), s = a(855);
        class i extends e.Disposable {
          get canvas() {
            return this._canvas;
          }
          get cacheCanvas() {
            var l;
            return (l = this._charAtlas) == null ? void 0 : l.pages[0].canvas;
          }
          constructor(l, m, _, v, C, w, S, b, x, A) {
            super(), this._terminal = l, this._container = m, this._alpha = C, this._themeService = w, this._bufferService = S, this._optionsService = b, this._decorationService = x, this._coreBrowserService = A, this._deviceCharWidth = 0, this._deviceCharHeight = 0, this._deviceCellWidth = 0, this._deviceCellHeight = 0, this._deviceCharLeft = 0, this._deviceCharTop = 0, this._selectionModel = (0, g.createSelectionRenderModel)(), this._bitmapGenerator = [], this._charAtlasDisposable = this.register(new e.MutableDisposable()), this._onAddTextureAtlasCanvas = this.register(new n.EventEmitter()), this.onAddTextureAtlasCanvas = this._onAddTextureAtlasCanvas.event, this._cellColorResolver = new c.CellColorResolver(this._terminal, this._optionsService, this._selectionModel, this._decorationService, this._coreBrowserService, this._themeService), this._canvas = this._coreBrowserService.mainDocument.createElement("canvas"), this._canvas.classList.add(`xterm-${_}-layer`), this._canvas.style.zIndex = v.toString(), this._initCanvas(), this._container.appendChild(this._canvas), this._refreshCharAtlas(this._themeService.colors), this.register(this._themeService.onChangeColors(((P) => {
              this._refreshCharAtlas(P), this.reset(), this.handleSelectionChanged(this._selectionModel.selectionStart, this._selectionModel.selectionEnd, this._selectionModel.columnSelectMode);
            }))), this.register((0, e.toDisposable)((() => {
              this._canvas.remove();
            })));
          }
          _initCanvas() {
            this._ctx = (0, f.throwIfFalsy)(this._canvas.getContext("2d", { alpha: this._alpha })), this._alpha || this._clearAll();
          }
          handleBlur() {
          }
          handleFocus() {
          }
          handleCursorMove() {
          }
          handleGridChanged(l, m) {
          }
          handleSelectionChanged(l, m, _ = !1) {
            this._selectionModel.update(this._terminal._core, l, m, _);
          }
          _setTransparency(l) {
            if (l === this._alpha) return;
            const m = this._canvas;
            this._alpha = l, this._canvas = this._canvas.cloneNode(), this._initCanvas(), this._container.replaceChild(this._canvas, m), this._refreshCharAtlas(this._themeService.colors), this.handleGridChanged(0, this._bufferService.rows - 1);
          }
          _refreshCharAtlas(l) {
            if (!(this._deviceCharWidth <= 0 && this._deviceCharHeight <= 0)) {
              this._charAtlas = (0, h.acquireTextureAtlas)(this._terminal, this._optionsService.rawOptions, l, this._deviceCellWidth, this._deviceCellHeight, this._deviceCharWidth, this._deviceCharHeight, this._coreBrowserService.dpr), this._charAtlasDisposable.value = (0, n.forwardEvent)(this._charAtlas.onAddTextureAtlasCanvas, this._onAddTextureAtlasCanvas), this._charAtlas.warmUp();
              for (let m = 0; m < this._charAtlas.pages.length; m++) this._bitmapGenerator[m] = new u(this._charAtlas.pages[m].canvas);
            }
          }
          resize(l) {
            this._deviceCellWidth = l.device.cell.width, this._deviceCellHeight = l.device.cell.height, this._deviceCharWidth = l.device.char.width, this._deviceCharHeight = l.device.char.height, this._deviceCharLeft = l.device.char.left, this._deviceCharTop = l.device.char.top, this._canvas.width = l.device.canvas.width, this._canvas.height = l.device.canvas.height, this._canvas.style.width = `${l.css.canvas.width}px`, this._canvas.style.height = `${l.css.canvas.height}px`, this._alpha || this._clearAll(), this._refreshCharAtlas(this._themeService.colors);
          }
          clearTextureAtlas() {
            var l;
            (l = this._charAtlas) == null || l.clearTexture();
          }
          _fillCells(l, m, _, v) {
            this._ctx.fillRect(l * this._deviceCellWidth, m * this._deviceCellHeight, _ * this._deviceCellWidth, v * this._deviceCellHeight);
          }
          _fillMiddleLineAtCells(l, m, _ = 1) {
            const v = Math.ceil(0.5 * this._deviceCellHeight);
            this._ctx.fillRect(l * this._deviceCellWidth, (m + 1) * this._deviceCellHeight - v - this._coreBrowserService.dpr, _ * this._deviceCellWidth, this._coreBrowserService.dpr);
          }
          _fillBottomLineAtCells(l, m, _ = 1, v = 0) {
            this._ctx.fillRect(l * this._deviceCellWidth, (m + 1) * this._deviceCellHeight + v - this._coreBrowserService.dpr - 1, _ * this._deviceCellWidth, this._coreBrowserService.dpr);
          }
          _curlyUnderlineAtCell(l, m, _ = 1) {
            this._ctx.save(), this._ctx.beginPath(), this._ctx.strokeStyle = this._ctx.fillStyle;
            const v = this._coreBrowserService.dpr;
            this._ctx.lineWidth = v;
            for (let C = 0; C < _; C++) {
              const w = (l + C) * this._deviceCellWidth, S = (l + C + 0.5) * this._deviceCellWidth, b = (l + C + 1) * this._deviceCellWidth, x = (m + 1) * this._deviceCellHeight - v - 1, A = x - v, P = x + v;
              this._ctx.moveTo(w, x), this._ctx.bezierCurveTo(w, A, S, A, S, x), this._ctx.bezierCurveTo(S, P, b, P, b, x);
            }
            this._ctx.stroke(), this._ctx.restore();
          }
          _dottedUnderlineAtCell(l, m, _ = 1) {
            this._ctx.save(), this._ctx.beginPath(), this._ctx.strokeStyle = this._ctx.fillStyle;
            const v = this._coreBrowserService.dpr;
            this._ctx.lineWidth = v, this._ctx.setLineDash([2 * v, v]);
            const C = l * this._deviceCellWidth, w = (m + 1) * this._deviceCellHeight - v - 1;
            this._ctx.moveTo(C, w);
            for (let S = 0; S < _; S++) {
              const b = (l + _ + S) * this._deviceCellWidth;
              this._ctx.lineTo(b, w);
            }
            this._ctx.stroke(), this._ctx.closePath(), this._ctx.restore();
          }
          _dashedUnderlineAtCell(l, m, _ = 1) {
            this._ctx.save(), this._ctx.beginPath(), this._ctx.strokeStyle = this._ctx.fillStyle;
            const v = this._coreBrowserService.dpr;
            this._ctx.lineWidth = v, this._ctx.setLineDash([4 * v, 3 * v]);
            const C = l * this._deviceCellWidth, w = (l + _) * this._deviceCellWidth, S = (m + 1) * this._deviceCellHeight - v - 1;
            this._ctx.moveTo(C, S), this._ctx.lineTo(w, S), this._ctx.stroke(), this._ctx.closePath(), this._ctx.restore();
          }
          _fillLeftLineAtCell(l, m, _) {
            this._ctx.fillRect(l * this._deviceCellWidth, m * this._deviceCellHeight, this._coreBrowserService.dpr * _, this._deviceCellHeight);
          }
          _strokeRectAtCell(l, m, _, v) {
            const C = this._coreBrowserService.dpr;
            this._ctx.lineWidth = C, this._ctx.strokeRect(l * this._deviceCellWidth + C / 2, m * this._deviceCellHeight + C / 2, _ * this._deviceCellWidth - C, v * this._deviceCellHeight - C);
          }
          _clearAll() {
            this._alpha ? this._ctx.clearRect(0, 0, this._canvas.width, this._canvas.height) : (this._ctx.fillStyle = this._themeService.colors.background.css, this._ctx.fillRect(0, 0, this._canvas.width, this._canvas.height));
          }
          _clearCells(l, m, _, v) {
            this._alpha ? this._ctx.clearRect(l * this._deviceCellWidth, m * this._deviceCellHeight, _ * this._deviceCellWidth, v * this._deviceCellHeight) : (this._ctx.fillStyle = this._themeService.colors.background.css, this._ctx.fillRect(l * this._deviceCellWidth, m * this._deviceCellHeight, _ * this._deviceCellWidth, v * this._deviceCellHeight));
          }
          _fillCharTrueColor(l, m, _) {
            this._ctx.font = this._getFont(!1, !1), this._ctx.textBaseline = r.TEXT_BASELINE, this._clipRow(_);
            let v = !1;
            this._optionsService.rawOptions.customGlyphs !== !1 && (v = (0, d.tryDrawCustomChar)(this._ctx, l.getChars(), m * this._deviceCellWidth, _ * this._deviceCellHeight, this._deviceCellWidth, this._deviceCellHeight, this._optionsService.rawOptions.fontSize, this._coreBrowserService.dpr)), v || this._ctx.fillText(l.getChars(), m * this._deviceCellWidth + this._deviceCharLeft, _ * this._deviceCellHeight + this._deviceCharTop + this._deviceCharHeight);
          }
          _drawChars(l, m, _) {
            var x, A, P, k;
            const v = l.getChars(), C = l.getCode(), w = l.getWidth();
            if (this._cellColorResolver.resolve(l, m, this._bufferService.buffer.ydisp + _, this._deviceCellWidth), !this._charAtlas) return;
            let S;
            if (S = v && v.length > 1 ? this._charAtlas.getRasterizedGlyphCombinedChar(v, this._cellColorResolver.result.bg, this._cellColorResolver.result.fg, this._cellColorResolver.result.ext, !0) : this._charAtlas.getRasterizedGlyph(l.getCode() || s.WHITESPACE_CELL_CODE, this._cellColorResolver.result.bg, this._cellColorResolver.result.fg, this._cellColorResolver.result.ext, !0), !S.size.x || !S.size.y) return;
            this._ctx.save(), this._clipRow(_), this._bitmapGenerator[S.texturePage] && this._charAtlas.pages[S.texturePage].canvas !== this._bitmapGenerator[S.texturePage].canvas && ((A = (x = this._bitmapGenerator[S.texturePage]) == null ? void 0 : x.bitmap) == null || A.close(), delete this._bitmapGenerator[S.texturePage]), this._charAtlas.pages[S.texturePage].version !== ((P = this._bitmapGenerator[S.texturePage]) == null ? void 0 : P.version) && (this._bitmapGenerator[S.texturePage] || (this._bitmapGenerator[S.texturePage] = new u(this._charAtlas.pages[S.texturePage].canvas)), this._bitmapGenerator[S.texturePage].refresh(), this._bitmapGenerator[S.texturePage].version = this._charAtlas.pages[S.texturePage].version);
            let b = S.size.x;
            this._optionsService.rawOptions.rescaleOverlappingGlyphs && (0, f.allowRescaling)(C, w, S.size.x, this._deviceCellWidth) && (b = this._deviceCellWidth - 1), this._ctx.drawImage(((k = this._bitmapGenerator[S.texturePage]) == null ? void 0 : k.bitmap) || this._charAtlas.pages[S.texturePage].canvas, S.texturePosition.x, S.texturePosition.y, S.size.x, S.size.y, m * this._deviceCellWidth + this._deviceCharLeft - S.offset.x, _ * this._deviceCellHeight + this._deviceCharTop - S.offset.y, b, S.size.y), this._ctx.restore();
          }
          _clipRow(l) {
            this._ctx.beginPath(), this._ctx.rect(0, l * this._deviceCellHeight, this._bufferService.cols * this._deviceCellWidth, this._deviceCellHeight), this._ctx.clip();
          }
          _getFont(l, m) {
            return `${m ? "italic" : ""} ${l ? this._optionsService.rawOptions.fontWeightBold : this._optionsService.rawOptions.fontWeight} ${this._optionsService.rawOptions.fontSize * this._coreBrowserService.dpr}px ${this._optionsService.rawOptions.fontFamily}`;
          }
        }
        t.BaseRenderLayer = i;
        class u {
          get bitmap() {
            return this._bitmap;
          }
          constructor(l) {
            this.canvas = l, this._state = 0, this._commitTimeout = void 0, this._bitmap = void 0, this.version = -1;
          }
          refresh() {
            var l;
            (l = this._bitmap) == null || l.close(), this._bitmap = void 0, o.isSafari || (this._commitTimeout === void 0 && (this._commitTimeout = window.setTimeout((() => this._generate()), 100)), this._state === 1 && (this._state = 2));
          }
          _generate() {
            var l;
            this._state === 0 && ((l = this._bitmap) == null || l.close(), this._bitmap = void 0, this._state = 1, window.createImageBitmap(this.canvas).then(((m) => {
              this._state === 2 ? this.refresh() : this._bitmap = m, this._state = 0;
            })), this._commitTimeout && (this._commitTimeout = void 0));
          }
        }
      }, 949: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CanvasRenderer = void 0;
        const c = a(627), h = a(56), r = a(374), d = a(345), f = a(859), g = a(873), n = a(43), e = a(630), o = a(744);
        class s extends f.Disposable {
          constructor(u, p, l, m, _, v, C, w, S, b, x) {
            super(), this._terminal = u, this._screenElement = p, this._bufferService = m, this._charSizeService = _, this._optionsService = v, this._coreBrowserService = S, this._themeService = x, this._observerDisposable = this.register(new f.MutableDisposable()), this._onRequestRedraw = this.register(new d.EventEmitter()), this.onRequestRedraw = this._onRequestRedraw.event, this._onChangeTextureAtlas = this.register(new d.EventEmitter()), this.onChangeTextureAtlas = this._onChangeTextureAtlas.event, this._onAddTextureAtlasCanvas = this.register(new d.EventEmitter()), this.onAddTextureAtlasCanvas = this._onAddTextureAtlasCanvas.event;
            const A = this._optionsService.rawOptions.allowTransparency;
            this._renderLayers = [new o.TextRenderLayer(this._terminal, this._screenElement, 0, A, this._bufferService, this._optionsService, C, b, this._coreBrowserService, x), new e.SelectionRenderLayer(this._terminal, this._screenElement, 1, this._bufferService, this._coreBrowserService, b, this._optionsService, x), new n.LinkRenderLayer(this._terminal, this._screenElement, 2, l, this._bufferService, this._optionsService, b, this._coreBrowserService, x), new g.CursorRenderLayer(this._terminal, this._screenElement, 3, this._onRequestRedraw, this._bufferService, this._optionsService, w, this._coreBrowserService, b, x)];
            for (const P of this._renderLayers) (0, d.forwardEvent)(P.onAddTextureAtlasCanvas, this._onAddTextureAtlasCanvas);
            this.dimensions = (0, r.createRenderDimensions)(), this._devicePixelRatio = this._coreBrowserService.dpr, this._updateDimensions(), this._observerDisposable.value = (0, h.observeDevicePixelDimensions)(this._renderLayers[0].canvas, this._coreBrowserService.window, ((P, k) => this._setCanvasDevicePixelDimensions(P, k))), this.register(this._coreBrowserService.onWindowChange(((P) => {
              this._observerDisposable.value = (0, h.observeDevicePixelDimensions)(this._renderLayers[0].canvas, P, ((k, M) => this._setCanvasDevicePixelDimensions(k, M)));
            }))), this.register((0, f.toDisposable)((() => {
              for (const P of this._renderLayers) P.dispose();
              (0, c.removeTerminalFromCache)(this._terminal);
            })));
          }
          get textureAtlas() {
            return this._renderLayers[0].cacheCanvas;
          }
          handleDevicePixelRatioChange() {
            this._devicePixelRatio !== this._coreBrowserService.dpr && (this._devicePixelRatio = this._coreBrowserService.dpr, this.handleResize(this._bufferService.cols, this._bufferService.rows));
          }
          handleResize(u, p) {
            this._updateDimensions();
            for (const l of this._renderLayers) l.resize(this.dimensions);
            this._screenElement.style.width = `${this.dimensions.css.canvas.width}px`, this._screenElement.style.height = `${this.dimensions.css.canvas.height}px`;
          }
          handleCharSizeChanged() {
            this.handleResize(this._bufferService.cols, this._bufferService.rows);
          }
          handleBlur() {
            this._runOperation(((u) => u.handleBlur()));
          }
          handleFocus() {
            this._runOperation(((u) => u.handleFocus()));
          }
          handleSelectionChanged(u, p, l = !1) {
            this._runOperation(((m) => m.handleSelectionChanged(u, p, l))), this._themeService.colors.selectionForeground && this._onRequestRedraw.fire({ start: 0, end: this._bufferService.rows - 1 });
          }
          handleCursorMove() {
            this._runOperation(((u) => u.handleCursorMove()));
          }
          clear() {
            this._runOperation(((u) => u.reset()));
          }
          _runOperation(u) {
            for (const p of this._renderLayers) u(p);
          }
          renderRows(u, p) {
            for (const l of this._renderLayers) l.handleGridChanged(u, p);
          }
          clearTextureAtlas() {
            for (const u of this._renderLayers) u.clearTextureAtlas();
          }
          _updateDimensions() {
            if (!this._charSizeService.hasValidSize) return;
            const u = this._coreBrowserService.dpr;
            this.dimensions.device.char.width = Math.floor(this._charSizeService.width * u), this.dimensions.device.char.height = Math.ceil(this._charSizeService.height * u), this.dimensions.device.cell.height = Math.floor(this.dimensions.device.char.height * this._optionsService.rawOptions.lineHeight), this.dimensions.device.char.top = this._optionsService.rawOptions.lineHeight === 1 ? 0 : Math.round((this.dimensions.device.cell.height - this.dimensions.device.char.height) / 2), this.dimensions.device.cell.width = this.dimensions.device.char.width + Math.round(this._optionsService.rawOptions.letterSpacing), this.dimensions.device.char.left = Math.floor(this._optionsService.rawOptions.letterSpacing / 2), this.dimensions.device.canvas.height = this._bufferService.rows * this.dimensions.device.cell.height, this.dimensions.device.canvas.width = this._bufferService.cols * this.dimensions.device.cell.width, this.dimensions.css.canvas.height = Math.round(this.dimensions.device.canvas.height / u), this.dimensions.css.canvas.width = Math.round(this.dimensions.device.canvas.width / u), this.dimensions.css.cell.height = this.dimensions.css.canvas.height / this._bufferService.rows, this.dimensions.css.cell.width = this.dimensions.css.canvas.width / this._bufferService.cols;
          }
          _setCanvasDevicePixelDimensions(u, p) {
            this.dimensions.device.canvas.height = p, this.dimensions.device.canvas.width = u;
            for (const l of this._renderLayers) l.resize(this.dimensions);
            this._requestRedrawViewport();
          }
          _requestRedrawViewport() {
            this._onRequestRedraw.fire({ start: 0, end: this._bufferService.rows - 1 });
          }
        }
        t.CanvasRenderer = s;
      }, 873: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CursorRenderLayer = void 0;
        const c = a(457), h = a(859), r = a(399), d = a(782), f = a(903);
        class g extends f.BaseRenderLayer {
          constructor(e, o, s, i, u, p, l, m, _, v) {
            super(e, o, "cursor", s, !0, v, u, p, _, m), this._onRequestRedraw = i, this._coreService = l, this._cursorBlinkStateManager = this.register(new h.MutableDisposable()), this._cell = new d.CellData(), this._state = { x: 0, y: 0, isFocused: !1, style: "", width: 0 }, this._cursorRenderers = { bar: this._renderBarCursor.bind(this), block: this._renderBlockCursor.bind(this), underline: this._renderUnderlineCursor.bind(this), outline: this._renderOutlineCursor.bind(this) }, this.register(p.onOptionChange((() => this._handleOptionsChanged()))), this._handleOptionsChanged();
          }
          resize(e) {
            super.resize(e), this._state = { x: 0, y: 0, isFocused: !1, style: "", width: 0 };
          }
          reset() {
            var e;
            this._clearCursor(), (e = this._cursorBlinkStateManager.value) == null || e.restartBlinkAnimation(), this._handleOptionsChanged();
          }
          handleBlur() {
            var e;
            (e = this._cursorBlinkStateManager.value) == null || e.pause(), this._onRequestRedraw.fire({ start: this._bufferService.buffer.y, end: this._bufferService.buffer.y });
          }
          handleFocus() {
            var e;
            (e = this._cursorBlinkStateManager.value) == null || e.resume(), this._onRequestRedraw.fire({ start: this._bufferService.buffer.y, end: this._bufferService.buffer.y });
          }
          _handleOptionsChanged() {
            this._optionsService.rawOptions.cursorBlink ? this._cursorBlinkStateManager.value || (this._cursorBlinkStateManager.value = new c.CursorBlinkStateManager((() => this._render(!0)), this._coreBrowserService)) : this._cursorBlinkStateManager.clear(), this._onRequestRedraw.fire({ start: this._bufferService.buffer.y, end: this._bufferService.buffer.y });
          }
          handleCursorMove() {
            var e;
            (e = this._cursorBlinkStateManager.value) == null || e.restartBlinkAnimation();
          }
          handleGridChanged(e, o) {
            !this._cursorBlinkStateManager.value || this._cursorBlinkStateManager.value.isPaused ? this._render(!1) : this._cursorBlinkStateManager.value.restartBlinkAnimation();
          }
          _render(e) {
            if (!this._coreService.isCursorInitialized || this._coreService.isCursorHidden) return void this._clearCursor();
            const o = this._bufferService.buffer.ybase + this._bufferService.buffer.y, s = o - this._bufferService.buffer.ydisp;
            if (s < 0 || s >= this._bufferService.rows) return void this._clearCursor();
            const i = Math.min(this._bufferService.buffer.x, this._bufferService.cols - 1);
            if (this._bufferService.buffer.lines.get(o).loadCell(i, this._cell), this._cell.content !== void 0) {
              if (!this._coreBrowserService.isFocused) {
                this._clearCursor(), this._ctx.save(), this._ctx.fillStyle = this._themeService.colors.cursor.css;
                const u = this._optionsService.rawOptions.cursorStyle, p = this._optionsService.rawOptions.cursorInactiveStyle;
                return p && p !== "none" && this._cursorRenderers[p](i, s, this._cell), this._ctx.restore(), this._state.x = i, this._state.y = s, this._state.isFocused = !1, this._state.style = u, void (this._state.width = this._cell.getWidth());
              }
              if (!this._cursorBlinkStateManager.value || this._cursorBlinkStateManager.value.isCursorVisible) {
                if (this._state) {
                  if (this._state.x === i && this._state.y === s && this._state.isFocused === this._coreBrowserService.isFocused && this._state.style === this._optionsService.rawOptions.cursorStyle && this._state.width === this._cell.getWidth()) return;
                  this._clearCursor();
                }
                this._ctx.save(), this._cursorRenderers[this._optionsService.rawOptions.cursorStyle || "block"](i, s, this._cell), this._ctx.restore(), this._state.x = i, this._state.y = s, this._state.isFocused = !1, this._state.style = this._optionsService.rawOptions.cursorStyle, this._state.width = this._cell.getWidth();
              } else this._clearCursor();
            }
          }
          _clearCursor() {
            this._state && (r.isFirefox || this._coreBrowserService.dpr < 1 ? this._clearAll() : this._clearCells(this._state.x, this._state.y, this._state.width, 1), this._state = { x: 0, y: 0, isFocused: !1, style: "", width: 0 });
          }
          _renderBarCursor(e, o, s) {
            this._ctx.save(), this._ctx.fillStyle = this._themeService.colors.cursor.css, this._fillLeftLineAtCell(e, o, this._optionsService.rawOptions.cursorWidth), this._ctx.restore();
          }
          _renderBlockCursor(e, o, s) {
            this._ctx.save(), this._ctx.fillStyle = this._themeService.colors.cursor.css, this._fillCells(e, o, s.getWidth(), 1), this._ctx.fillStyle = this._themeService.colors.cursorAccent.css, this._fillCharTrueColor(s, e, o), this._ctx.restore();
          }
          _renderUnderlineCursor(e, o, s) {
            this._ctx.save(), this._ctx.fillStyle = this._themeService.colors.cursor.css, this._fillBottomLineAtCells(e, o), this._ctx.restore();
          }
          _renderOutlineCursor(e, o, s) {
            this._ctx.save(), this._ctx.strokeStyle = this._themeService.colors.cursor.css, this._strokeRectAtCell(e, o, s.getWidth(), 1), this._ctx.restore();
          }
        }
        t.CursorRenderLayer = g;
      }, 574: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.GridCache = void 0, t.GridCache = class {
          constructor() {
            this.cache = [];
          }
          resize(a, c) {
            for (let h = 0; h < a; h++) {
              this.cache.length <= h && this.cache.push([]);
              for (let r = this.cache[h].length; r < c; r++) this.cache[h].push(void 0);
              this.cache[h].length = c;
            }
            this.cache.length = a;
          }
          clear() {
            for (let a = 0; a < this.cache.length; a++) for (let c = 0; c < this.cache[a].length; c++) this.cache[a][c] = void 0;
          }
        };
      }, 43: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.LinkRenderLayer = void 0;
        const c = a(197), h = a(237), r = a(903);
        class d extends r.BaseRenderLayer {
          constructor(g, n, e, o, s, i, u, p, l) {
            super(g, n, "link", e, !0, l, s, i, u, p), this.register(o.onShowLinkUnderline(((m) => this._handleShowLinkUnderline(m)))), this.register(o.onHideLinkUnderline(((m) => this._handleHideLinkUnderline(m))));
          }
          resize(g) {
            super.resize(g), this._state = void 0;
          }
          reset() {
            this._clearCurrentLink();
          }
          _clearCurrentLink() {
            if (this._state) {
              this._clearCells(this._state.x1, this._state.y1, this._state.cols - this._state.x1, 1);
              const g = this._state.y2 - this._state.y1 - 1;
              g > 0 && this._clearCells(0, this._state.y1 + 1, this._state.cols, g), this._clearCells(0, this._state.y2, this._state.x2, 1), this._state = void 0;
            }
          }
          _handleShowLinkUnderline(g) {
            if (g.fg === h.INVERTED_DEFAULT_COLOR ? this._ctx.fillStyle = this._themeService.colors.background.css : g.fg && (0, c.is256Color)(g.fg) ? this._ctx.fillStyle = this._themeService.colors.ansi[g.fg].css : this._ctx.fillStyle = this._themeService.colors.foreground.css, g.y1 === g.y2) this._fillBottomLineAtCells(g.x1, g.y1, g.x2 - g.x1);
            else {
              this._fillBottomLineAtCells(g.x1, g.y1, g.cols - g.x1);
              for (let n = g.y1 + 1; n < g.y2; n++) this._fillBottomLineAtCells(0, n, g.cols);
              this._fillBottomLineAtCells(0, g.y2, g.x2);
            }
            this._state = g;
          }
          _handleHideLinkUnderline(g) {
            this._clearCurrentLink();
          }
        }
        t.LinkRenderLayer = d;
      }, 630: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.SelectionRenderLayer = void 0;
        const c = a(903);
        class h extends c.BaseRenderLayer {
          constructor(d, f, g, n, e, o, s, i) {
            super(d, f, "selection", g, !0, i, n, s, o, e), this._clearState();
          }
          _clearState() {
            this._state = { start: void 0, end: void 0, columnSelectMode: void 0, ydisp: void 0 };
          }
          resize(d) {
            super.resize(d), this._selectionModel.selectionStart && this._selectionModel.selectionEnd && (this._clearState(), this._redrawSelection(this._selectionModel.selectionStart, this._selectionModel.selectionEnd, this._selectionModel.columnSelectMode));
          }
          reset() {
            this._state.start && this._state.end && (this._clearState(), this._clearAll());
          }
          handleBlur() {
            this.reset(), this._redrawSelection(this._selectionModel.selectionStart, this._selectionModel.selectionEnd, this._selectionModel.columnSelectMode);
          }
          handleFocus() {
            this.reset(), this._redrawSelection(this._selectionModel.selectionStart, this._selectionModel.selectionEnd, this._selectionModel.columnSelectMode);
          }
          handleSelectionChanged(d, f, g) {
            super.handleSelectionChanged(d, f, g), this._redrawSelection(d, f, g);
          }
          _redrawSelection(d, f, g) {
            if (!this._didStateChange(d, f, g, this._bufferService.buffer.ydisp)) return;
            if (this._clearAll(), !d || !f) return void this._clearState();
            const n = d[1] - this._bufferService.buffer.ydisp, e = f[1] - this._bufferService.buffer.ydisp, o = Math.max(n, 0), s = Math.min(e, this._bufferService.rows - 1);
            if (o >= this._bufferService.rows || s < 0) this._state.ydisp = this._bufferService.buffer.ydisp;
            else {
              if (this._ctx.fillStyle = (this._coreBrowserService.isFocused ? this._themeService.colors.selectionBackgroundTransparent : this._themeService.colors.selectionInactiveBackgroundTransparent).css, g) {
                const i = d[0], u = f[0] - i, p = s - o + 1;
                this._fillCells(i, o, u, p);
              } else {
                const i = n === o ? d[0] : 0, u = o === e ? f[0] : this._bufferService.cols;
                this._fillCells(i, o, u - i, 1);
                const p = Math.max(s - o - 1, 0);
                if (this._fillCells(0, o + 1, this._bufferService.cols, p), o !== s) {
                  const l = e === s ? f[0] : this._bufferService.cols;
                  this._fillCells(0, s, l, 1);
                }
              }
              this._state.start = [d[0], d[1]], this._state.end = [f[0], f[1]], this._state.columnSelectMode = g, this._state.ydisp = this._bufferService.buffer.ydisp;
            }
          }
          _didStateChange(d, f, g, n) {
            return !this._areCoordinatesEqual(d, this._state.start) || !this._areCoordinatesEqual(f, this._state.end) || g !== this._state.columnSelectMode || n !== this._state.ydisp;
          }
          _areCoordinatesEqual(d, f) {
            return !(!d || !f) && d[0] === f[0] && d[1] === f[1];
          }
        }
        t.SelectionRenderLayer = h;
      }, 744: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.TextRenderLayer = void 0;
        const c = a(577), h = a(147), r = a(782), d = a(855), f = a(903), g = a(574);
        class n extends f.BaseRenderLayer {
          constructor(o, s, i, u, p, l, m, _, v, C) {
            super(o, s, "text", i, u, C, p, l, _, v), this._characterJoinerService = m, this._characterWidth = 0, this._characterFont = "", this._characterOverlapCache = {}, this._workCell = new r.CellData(), this._state = new g.GridCache(), this.register(l.onSpecificOptionChange("allowTransparency", ((w) => this._setTransparency(w))));
          }
          resize(o) {
            super.resize(o);
            const s = this._getFont(!1, !1);
            this._characterWidth === o.device.char.width && this._characterFont === s || (this._characterWidth = o.device.char.width, this._characterFont = s, this._characterOverlapCache = {}), this._state.clear(), this._state.resize(this._bufferService.cols, this._bufferService.rows);
          }
          reset() {
            this._state.clear(), this._clearAll();
          }
          _forEachCell(o, s, i) {
            for (let u = o; u <= s; u++) {
              const p = u + this._bufferService.buffer.ydisp, l = this._bufferService.buffer.lines.get(p), m = this._characterJoinerService.getJoinedCharacters(p);
              for (let _ = 0; _ < this._bufferService.cols; _++) {
                l.loadCell(_, this._workCell);
                let v = this._workCell, C = !1, w = _;
                if (v.getWidth() !== 0) {
                  if (m.length > 0 && _ === m[0][0]) {
                    C = !0;
                    const S = m.shift();
                    v = new c.JoinedCellData(this._workCell, l.translateToString(!0, S[0], S[1]), S[1] - S[0]), w = S[1] - 1;
                  }
                  !C && this._isOverlapping(v) && w < l.length - 1 && l.getCodePoint(w + 1) === d.NULL_CELL_CODE && (v.content &= -12582913, v.content |= 8388608), i(v, _, u), _ = w;
                }
              }
            }
          }
          _drawBackground(o, s) {
            const i = this._ctx, u = this._bufferService.cols;
            let p = 0, l = 0, m = null;
            i.save(), this._forEachCell(o, s, ((_, v, C) => {
              let w = null;
              _.isInverse() ? w = _.isFgDefault() ? this._themeService.colors.foreground.css : _.isFgRGB() ? `rgb(${h.AttributeData.toColorRGB(_.getFgColor()).join(",")})` : this._themeService.colors.ansi[_.getFgColor()].css : _.isBgRGB() ? w = `rgb(${h.AttributeData.toColorRGB(_.getBgColor()).join(",")})` : _.isBgPalette() && (w = this._themeService.colors.ansi[_.getBgColor()].css);
              let S = !1;
              this._decorationService.forEachDecorationAtCell(v, this._bufferService.buffer.ydisp + C, void 0, ((b) => {
                b.options.layer !== "top" && S || (b.backgroundColorRGB && (w = b.backgroundColorRGB.css), S = b.options.layer === "top");
              })), m === null && (p = v, l = C), C !== l ? (i.fillStyle = m || "", this._fillCells(p, l, u - p, 1), p = v, l = C) : m !== w && (i.fillStyle = m || "", this._fillCells(p, l, v - p, 1), p = v, l = C), m = w;
            })), m !== null && (i.fillStyle = m, this._fillCells(p, l, u - p, 1)), i.restore();
          }
          _drawForeground(o, s) {
            this._forEachCell(o, s, ((i, u, p) => this._drawChars(i, u, p)));
          }
          handleGridChanged(o, s) {
            this._state.cache.length !== 0 && (this._charAtlas && this._charAtlas.beginFrame(), this._clearCells(0, o, this._bufferService.cols, s - o + 1), this._drawBackground(o, s), this._drawForeground(o, s));
          }
          _isOverlapping(o) {
            if (o.getWidth() !== 1 || o.getCode() < 256) return !1;
            const s = o.getChars();
            if (this._characterOverlapCache.hasOwnProperty(s)) return this._characterOverlapCache[s];
            this._ctx.save(), this._ctx.font = this._characterFont;
            const i = Math.floor(this._ctx.measureText(s).width) > this._characterWidth;
            return this._ctx.restore(), this._characterOverlapCache[s] = i, i;
          }
        }
        t.TextRenderLayer = n;
      }, 274: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CellColorResolver = void 0;
        const c = a(855), h = a(160), r = a(374);
        let d, f = 0, g = 0, n = !1, e = !1, o = !1, s = 0;
        t.CellColorResolver = class {
          constructor(i, u, p, l, m, _) {
            this._terminal = i, this._optionService = u, this._selectionRenderModel = p, this._decorationService = l, this._coreBrowserService = m, this._themeService = _, this.result = { fg: 0, bg: 0, ext: 0 };
          }
          resolve(i, u, p, l) {
            if (this.result.bg = i.bg, this.result.fg = i.fg, this.result.ext = 268435456 & i.bg ? i.extended.ext : 0, g = 0, f = 0, e = !1, n = !1, o = !1, d = this._themeService.colors, s = 0, i.getCode() !== c.NULL_CELL_CODE && i.extended.underlineStyle === 4) {
              const m = Math.max(1, Math.floor(this._optionService.rawOptions.fontSize * this._coreBrowserService.dpr / 15));
              s = u * l % (2 * Math.round(m));
            }
            if (this._decorationService.forEachDecorationAtCell(u, p, "bottom", ((m) => {
              m.backgroundColorRGB && (g = m.backgroundColorRGB.rgba >> 8 & 16777215, e = !0), m.foregroundColorRGB && (f = m.foregroundColorRGB.rgba >> 8 & 16777215, n = !0);
            })), o = this._selectionRenderModel.isCellSelected(this._terminal, u, p), o) {
              if (67108864 & this.result.fg || (50331648 & this.result.bg) != 0) {
                if (67108864 & this.result.fg) switch (50331648 & this.result.fg) {
                  case 16777216:
                  case 33554432:
                    g = this._themeService.colors.ansi[255 & this.result.fg].rgba;
                    break;
                  case 50331648:
                    g = (16777215 & this.result.fg) << 8 | 255;
                    break;
                  default:
                    g = this._themeService.colors.foreground.rgba;
                }
                else switch (50331648 & this.result.bg) {
                  case 16777216:
                  case 33554432:
                    g = this._themeService.colors.ansi[255 & this.result.bg].rgba;
                    break;
                  case 50331648:
                    g = (16777215 & this.result.bg) << 8 | 255;
                }
                g = h.rgba.blend(g, 4294967040 & (this._coreBrowserService.isFocused ? d.selectionBackgroundOpaque : d.selectionInactiveBackgroundOpaque).rgba | 128) >> 8 & 16777215;
              } else g = (this._coreBrowserService.isFocused ? d.selectionBackgroundOpaque : d.selectionInactiveBackgroundOpaque).rgba >> 8 & 16777215;
              if (e = !0, d.selectionForeground && (f = d.selectionForeground.rgba >> 8 & 16777215, n = !0), (0, r.treatGlyphAsBackgroundColor)(i.getCode())) {
                if (67108864 & this.result.fg && (50331648 & this.result.bg) == 0) f = (this._coreBrowserService.isFocused ? d.selectionBackgroundOpaque : d.selectionInactiveBackgroundOpaque).rgba >> 8 & 16777215;
                else {
                  if (67108864 & this.result.fg) switch (50331648 & this.result.bg) {
                    case 16777216:
                    case 33554432:
                      f = this._themeService.colors.ansi[255 & this.result.bg].rgba;
                      break;
                    case 50331648:
                      f = (16777215 & this.result.bg) << 8 | 255;
                  }
                  else switch (50331648 & this.result.fg) {
                    case 16777216:
                    case 33554432:
                      f = this._themeService.colors.ansi[255 & this.result.fg].rgba;
                      break;
                    case 50331648:
                      f = (16777215 & this.result.fg) << 8 | 255;
                      break;
                    default:
                      f = this._themeService.colors.foreground.rgba;
                  }
                  f = h.rgba.blend(f, 4294967040 & (this._coreBrowserService.isFocused ? d.selectionBackgroundOpaque : d.selectionInactiveBackgroundOpaque).rgba | 128) >> 8 & 16777215;
                }
                n = !0;
              }
            }
            this._decorationService.forEachDecorationAtCell(u, p, "top", ((m) => {
              m.backgroundColorRGB && (g = m.backgroundColorRGB.rgba >> 8 & 16777215, e = !0), m.foregroundColorRGB && (f = m.foregroundColorRGB.rgba >> 8 & 16777215, n = !0);
            })), e && (g = o ? -16777216 & i.bg & -134217729 | g | 50331648 : -16777216 & i.bg | g | 50331648), n && (f = -16777216 & i.fg & -67108865 | f | 50331648), 67108864 & this.result.fg && (e && !n && (f = (50331648 & this.result.bg) == 0 ? -134217728 & this.result.fg | 16777215 & d.background.rgba >> 8 | 50331648 : -134217728 & this.result.fg | 67108863 & this.result.bg, n = !0), !e && n && (g = (50331648 & this.result.fg) == 0 ? -67108864 & this.result.bg | 16777215 & d.foreground.rgba >> 8 | 50331648 : -67108864 & this.result.bg | 67108863 & this.result.fg, e = !0)), d = void 0, this.result.bg = e ? g : this.result.bg, this.result.fg = n ? f : this.result.fg, this.result.ext &= 536870911, this.result.ext |= s << 29 & 3758096384;
          }
        };
      }, 627: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.removeTerminalFromCache = t.acquireTextureAtlas = void 0;
        const c = a(509), h = a(197), r = [];
        t.acquireTextureAtlas = function(d, f, g, n, e, o, s, i) {
          const u = (0, h.generateConfig)(n, e, o, s, f, g, i);
          for (let m = 0; m < r.length; m++) {
            const _ = r[m], v = _.ownedBy.indexOf(d);
            if (v >= 0) {
              if ((0, h.configEquals)(_.config, u)) return _.atlas;
              _.ownedBy.length === 1 ? (_.atlas.dispose(), r.splice(m, 1)) : _.ownedBy.splice(v, 1);
              break;
            }
          }
          for (let m = 0; m < r.length; m++) {
            const _ = r[m];
            if ((0, h.configEquals)(_.config, u)) return _.ownedBy.push(d), _.atlas;
          }
          const p = d._core, l = { atlas: new c.TextureAtlas(document, u, p.unicodeService), config: u, ownedBy: [d] };
          return r.push(l), l.atlas;
        }, t.removeTerminalFromCache = function(d) {
          for (let f = 0; f < r.length; f++) {
            const g = r[f].ownedBy.indexOf(d);
            if (g !== -1) {
              r[f].ownedBy.length === 1 ? (r[f].atlas.dispose(), r.splice(f, 1)) : r[f].ownedBy.splice(g, 1);
              break;
            }
          }
        };
      }, 197: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.is256Color = t.configEquals = t.generateConfig = void 0;
        const c = a(160);
        t.generateConfig = function(h, r, d, f, g, n, e) {
          const o = { foreground: n.foreground, background: n.background, cursor: c.NULL_COLOR, cursorAccent: c.NULL_COLOR, selectionForeground: c.NULL_COLOR, selectionBackgroundTransparent: c.NULL_COLOR, selectionBackgroundOpaque: c.NULL_COLOR, selectionInactiveBackgroundTransparent: c.NULL_COLOR, selectionInactiveBackgroundOpaque: c.NULL_COLOR, ansi: n.ansi.slice(), contrastCache: n.contrastCache, halfContrastCache: n.halfContrastCache };
          return { customGlyphs: g.customGlyphs, devicePixelRatio: e, letterSpacing: g.letterSpacing, lineHeight: g.lineHeight, deviceCellWidth: h, deviceCellHeight: r, deviceCharWidth: d, deviceCharHeight: f, fontFamily: g.fontFamily, fontSize: g.fontSize, fontWeight: g.fontWeight, fontWeightBold: g.fontWeightBold, allowTransparency: g.allowTransparency, drawBoldTextInBrightColors: g.drawBoldTextInBrightColors, minimumContrastRatio: g.minimumContrastRatio, colors: o };
        }, t.configEquals = function(h, r) {
          for (let d = 0; d < h.colors.ansi.length; d++) if (h.colors.ansi[d].rgba !== r.colors.ansi[d].rgba) return !1;
          return h.devicePixelRatio === r.devicePixelRatio && h.customGlyphs === r.customGlyphs && h.lineHeight === r.lineHeight && h.letterSpacing === r.letterSpacing && h.fontFamily === r.fontFamily && h.fontSize === r.fontSize && h.fontWeight === r.fontWeight && h.fontWeightBold === r.fontWeightBold && h.allowTransparency === r.allowTransparency && h.deviceCharWidth === r.deviceCharWidth && h.deviceCharHeight === r.deviceCharHeight && h.drawBoldTextInBrightColors === r.drawBoldTextInBrightColors && h.minimumContrastRatio === r.minimumContrastRatio && h.colors.foreground.rgba === r.colors.foreground.rgba && h.colors.background.rgba === r.colors.background.rgba;
        }, t.is256Color = function(h) {
          return (50331648 & h) == 16777216 || (50331648 & h) == 33554432;
        };
      }, 237: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.TEXT_BASELINE = t.DIM_OPACITY = t.INVERTED_DEFAULT_COLOR = void 0;
        const c = a(399);
        t.INVERTED_DEFAULT_COLOR = 257, t.DIM_OPACITY = 0.5, t.TEXT_BASELINE = c.isFirefox || c.isLegacyEdge ? "bottom" : "ideographic";
      }, 457: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CursorBlinkStateManager = void 0, t.CursorBlinkStateManager = class {
          constructor(a, c) {
            this._renderCallback = a, this._coreBrowserService = c, this.isCursorVisible = !0, this._coreBrowserService.isFocused && this._restartInterval();
          }
          get isPaused() {
            return !(this._blinkStartTimeout || this._blinkInterval);
          }
          dispose() {
            this._blinkInterval && (this._coreBrowserService.window.clearInterval(this._blinkInterval), this._blinkInterval = void 0), this._blinkStartTimeout && (this._coreBrowserService.window.clearTimeout(this._blinkStartTimeout), this._blinkStartTimeout = void 0), this._animationFrame && (this._coreBrowserService.window.cancelAnimationFrame(this._animationFrame), this._animationFrame = void 0);
          }
          restartBlinkAnimation() {
            this.isPaused || (this._animationTimeRestarted = Date.now(), this.isCursorVisible = !0, this._animationFrame || (this._animationFrame = this._coreBrowserService.window.requestAnimationFrame((() => {
              this._renderCallback(), this._animationFrame = void 0;
            }))));
          }
          _restartInterval(a = 600) {
            this._blinkInterval && (this._coreBrowserService.window.clearInterval(this._blinkInterval), this._blinkInterval = void 0), this._blinkStartTimeout = this._coreBrowserService.window.setTimeout((() => {
              if (this._animationTimeRestarted) {
                const c = 600 - (Date.now() - this._animationTimeRestarted);
                if (this._animationTimeRestarted = void 0, c > 0) return void this._restartInterval(c);
              }
              this.isCursorVisible = !1, this._animationFrame = this._coreBrowserService.window.requestAnimationFrame((() => {
                this._renderCallback(), this._animationFrame = void 0;
              })), this._blinkInterval = this._coreBrowserService.window.setInterval((() => {
                if (this._animationTimeRestarted) {
                  const c = 600 - (Date.now() - this._animationTimeRestarted);
                  return this._animationTimeRestarted = void 0, void this._restartInterval(c);
                }
                this.isCursorVisible = !this.isCursorVisible, this._animationFrame = this._coreBrowserService.window.requestAnimationFrame((() => {
                  this._renderCallback(), this._animationFrame = void 0;
                }));
              }), 600);
            }), a);
          }
          pause() {
            this.isCursorVisible = !0, this._blinkInterval && (this._coreBrowserService.window.clearInterval(this._blinkInterval), this._blinkInterval = void 0), this._blinkStartTimeout && (this._coreBrowserService.window.clearTimeout(this._blinkStartTimeout), this._blinkStartTimeout = void 0), this._animationFrame && (this._coreBrowserService.window.cancelAnimationFrame(this._animationFrame), this._animationFrame = void 0);
          }
          resume() {
            this.pause(), this._animationTimeRestarted = void 0, this._restartInterval(), this.restartBlinkAnimation();
          }
        };
      }, 860: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.tryDrawCustomChar = t.powerlineDefinitions = t.boxDrawingDefinitions = t.blockElementDefinitions = void 0;
        const c = a(374);
        t.blockElementDefinitions = { "▀": [{ x: 0, y: 0, w: 8, h: 4 }], "▁": [{ x: 0, y: 7, w: 8, h: 1 }], "▂": [{ x: 0, y: 6, w: 8, h: 2 }], "▃": [{ x: 0, y: 5, w: 8, h: 3 }], "▄": [{ x: 0, y: 4, w: 8, h: 4 }], "▅": [{ x: 0, y: 3, w: 8, h: 5 }], "▆": [{ x: 0, y: 2, w: 8, h: 6 }], "▇": [{ x: 0, y: 1, w: 8, h: 7 }], "█": [{ x: 0, y: 0, w: 8, h: 8 }], "▉": [{ x: 0, y: 0, w: 7, h: 8 }], "▊": [{ x: 0, y: 0, w: 6, h: 8 }], "▋": [{ x: 0, y: 0, w: 5, h: 8 }], "▌": [{ x: 0, y: 0, w: 4, h: 8 }], "▍": [{ x: 0, y: 0, w: 3, h: 8 }], "▎": [{ x: 0, y: 0, w: 2, h: 8 }], "▏": [{ x: 0, y: 0, w: 1, h: 8 }], "▐": [{ x: 4, y: 0, w: 4, h: 8 }], "▔": [{ x: 0, y: 0, w: 8, h: 1 }], "▕": [{ x: 7, y: 0, w: 1, h: 8 }], "▖": [{ x: 0, y: 4, w: 4, h: 4 }], "▗": [{ x: 4, y: 4, w: 4, h: 4 }], "▘": [{ x: 0, y: 0, w: 4, h: 4 }], "▙": [{ x: 0, y: 0, w: 4, h: 8 }, { x: 0, y: 4, w: 8, h: 4 }], "▚": [{ x: 0, y: 0, w: 4, h: 4 }, { x: 4, y: 4, w: 4, h: 4 }], "▛": [{ x: 0, y: 0, w: 4, h: 8 }, { x: 4, y: 0, w: 4, h: 4 }], "▜": [{ x: 0, y: 0, w: 8, h: 4 }, { x: 4, y: 0, w: 4, h: 8 }], "▝": [{ x: 4, y: 0, w: 4, h: 4 }], "▞": [{ x: 4, y: 0, w: 4, h: 4 }, { x: 0, y: 4, w: 4, h: 4 }], "▟": [{ x: 4, y: 0, w: 4, h: 8 }, { x: 0, y: 4, w: 8, h: 4 }], "🭰": [{ x: 1, y: 0, w: 1, h: 8 }], "🭱": [{ x: 2, y: 0, w: 1, h: 8 }], "🭲": [{ x: 3, y: 0, w: 1, h: 8 }], "🭳": [{ x: 4, y: 0, w: 1, h: 8 }], "🭴": [{ x: 5, y: 0, w: 1, h: 8 }], "🭵": [{ x: 6, y: 0, w: 1, h: 8 }], "🭶": [{ x: 0, y: 1, w: 8, h: 1 }], "🭷": [{ x: 0, y: 2, w: 8, h: 1 }], "🭸": [{ x: 0, y: 3, w: 8, h: 1 }], "🭹": [{ x: 0, y: 4, w: 8, h: 1 }], "🭺": [{ x: 0, y: 5, w: 8, h: 1 }], "🭻": [{ x: 0, y: 6, w: 8, h: 1 }], "🭼": [{ x: 0, y: 0, w: 1, h: 8 }, { x: 0, y: 7, w: 8, h: 1 }], "🭽": [{ x: 0, y: 0, w: 1, h: 8 }, { x: 0, y: 0, w: 8, h: 1 }], "🭾": [{ x: 7, y: 0, w: 1, h: 8 }, { x: 0, y: 0, w: 8, h: 1 }], "🭿": [{ x: 7, y: 0, w: 1, h: 8 }, { x: 0, y: 7, w: 8, h: 1 }], "🮀": [{ x: 0, y: 0, w: 8, h: 1 }, { x: 0, y: 7, w: 8, h: 1 }], "🮁": [{ x: 0, y: 0, w: 8, h: 1 }, { x: 0, y: 2, w: 8, h: 1 }, { x: 0, y: 4, w: 8, h: 1 }, { x: 0, y: 7, w: 8, h: 1 }], "🮂": [{ x: 0, y: 0, w: 8, h: 2 }], "🮃": [{ x: 0, y: 0, w: 8, h: 3 }], "🮄": [{ x: 0, y: 0, w: 8, h: 5 }], "🮅": [{ x: 0, y: 0, w: 8, h: 6 }], "🮆": [{ x: 0, y: 0, w: 8, h: 7 }], "🮇": [{ x: 6, y: 0, w: 2, h: 8 }], "🮈": [{ x: 5, y: 0, w: 3, h: 8 }], "🮉": [{ x: 3, y: 0, w: 5, h: 8 }], "🮊": [{ x: 2, y: 0, w: 6, h: 8 }], "🮋": [{ x: 1, y: 0, w: 7, h: 8 }], "🮕": [{ x: 0, y: 0, w: 2, h: 2 }, { x: 4, y: 0, w: 2, h: 2 }, { x: 2, y: 2, w: 2, h: 2 }, { x: 6, y: 2, w: 2, h: 2 }, { x: 0, y: 4, w: 2, h: 2 }, { x: 4, y: 4, w: 2, h: 2 }, { x: 2, y: 6, w: 2, h: 2 }, { x: 6, y: 6, w: 2, h: 2 }], "🮖": [{ x: 2, y: 0, w: 2, h: 2 }, { x: 6, y: 0, w: 2, h: 2 }, { x: 0, y: 2, w: 2, h: 2 }, { x: 4, y: 2, w: 2, h: 2 }, { x: 2, y: 4, w: 2, h: 2 }, { x: 6, y: 4, w: 2, h: 2 }, { x: 0, y: 6, w: 2, h: 2 }, { x: 4, y: 6, w: 2, h: 2 }], "🮗": [{ x: 0, y: 2, w: 8, h: 2 }, { x: 0, y: 6, w: 8, h: 2 }] };
        const h = { "░": [[1, 0, 0, 0], [0, 0, 0, 0], [0, 0, 1, 0], [0, 0, 0, 0]], "▒": [[1, 0], [0, 0], [0, 1], [0, 0]], "▓": [[0, 1], [1, 1], [1, 0], [1, 1]] };
        t.boxDrawingDefinitions = { "─": { 1: "M0,.5 L1,.5" }, "━": { 3: "M0,.5 L1,.5" }, "│": { 1: "M.5,0 L.5,1" }, "┃": { 3: "M.5,0 L.5,1" }, "┌": { 1: "M0.5,1 L.5,.5 L1,.5" }, "┏": { 3: "M0.5,1 L.5,.5 L1,.5" }, "┐": { 1: "M0,.5 L.5,.5 L.5,1" }, "┓": { 3: "M0,.5 L.5,.5 L.5,1" }, "└": { 1: "M.5,0 L.5,.5 L1,.5" }, "┗": { 3: "M.5,0 L.5,.5 L1,.5" }, "┘": { 1: "M.5,0 L.5,.5 L0,.5" }, "┛": { 3: "M.5,0 L.5,.5 L0,.5" }, "├": { 1: "M.5,0 L.5,1 M.5,.5 L1,.5" }, "┣": { 3: "M.5,0 L.5,1 M.5,.5 L1,.5" }, "┤": { 1: "M.5,0 L.5,1 M.5,.5 L0,.5" }, "┫": { 3: "M.5,0 L.5,1 M.5,.5 L0,.5" }, "┬": { 1: "M0,.5 L1,.5 M.5,.5 L.5,1" }, "┳": { 3: "M0,.5 L1,.5 M.5,.5 L.5,1" }, "┴": { 1: "M0,.5 L1,.5 M.5,.5 L.5,0" }, "┻": { 3: "M0,.5 L1,.5 M.5,.5 L.5,0" }, "┼": { 1: "M0,.5 L1,.5 M.5,0 L.5,1" }, "╋": { 3: "M0,.5 L1,.5 M.5,0 L.5,1" }, "╴": { 1: "M.5,.5 L0,.5" }, "╸": { 3: "M.5,.5 L0,.5" }, "╵": { 1: "M.5,.5 L.5,0" }, "╹": { 3: "M.5,.5 L.5,0" }, "╶": { 1: "M.5,.5 L1,.5" }, "╺": { 3: "M.5,.5 L1,.5" }, "╷": { 1: "M.5,.5 L.5,1" }, "╻": { 3: "M.5,.5 L.5,1" }, "═": { 1: (n, e) => `M0,${0.5 - e} L1,${0.5 - e} M0,${0.5 + e} L1,${0.5 + e}` }, "║": { 1: (n, e) => `M${0.5 - n},0 L${0.5 - n},1 M${0.5 + n},0 L${0.5 + n},1` }, "╒": { 1: (n, e) => `M.5,1 L.5,${0.5 - e} L1,${0.5 - e} M.5,${0.5 + e} L1,${0.5 + e}` }, "╓": { 1: (n, e) => `M${0.5 - n},1 L${0.5 - n},.5 L1,.5 M${0.5 + n},.5 L${0.5 + n},1` }, "╔": { 1: (n, e) => `M1,${0.5 - e} L${0.5 - n},${0.5 - e} L${0.5 - n},1 M1,${0.5 + e} L${0.5 + n},${0.5 + e} L${0.5 + n},1` }, "╕": { 1: (n, e) => `M0,${0.5 - e} L.5,${0.5 - e} L.5,1 M0,${0.5 + e} L.5,${0.5 + e}` }, "╖": { 1: (n, e) => `M${0.5 + n},1 L${0.5 + n},.5 L0,.5 M${0.5 - n},.5 L${0.5 - n},1` }, "╗": { 1: (n, e) => `M0,${0.5 + e} L${0.5 - n},${0.5 + e} L${0.5 - n},1 M0,${0.5 - e} L${0.5 + n},${0.5 - e} L${0.5 + n},1` }, "╘": { 1: (n, e) => `M.5,0 L.5,${0.5 + e} L1,${0.5 + e} M.5,${0.5 - e} L1,${0.5 - e}` }, "╙": { 1: (n, e) => `M1,.5 L${0.5 - n},.5 L${0.5 - n},0 M${0.5 + n},.5 L${0.5 + n},0` }, "╚": { 1: (n, e) => `M1,${0.5 - e} L${0.5 + n},${0.5 - e} L${0.5 + n},0 M1,${0.5 + e} L${0.5 - n},${0.5 + e} L${0.5 - n},0` }, "╛": { 1: (n, e) => `M0,${0.5 + e} L.5,${0.5 + e} L.5,0 M0,${0.5 - e} L.5,${0.5 - e}` }, "╜": { 1: (n, e) => `M0,.5 L${0.5 + n},.5 L${0.5 + n},0 M${0.5 - n},.5 L${0.5 - n},0` }, "╝": { 1: (n, e) => `M0,${0.5 - e} L${0.5 - n},${0.5 - e} L${0.5 - n},0 M0,${0.5 + e} L${0.5 + n},${0.5 + e} L${0.5 + n},0` }, "╞": { 1: (n, e) => `M.5,0 L.5,1 M.5,${0.5 - e} L1,${0.5 - e} M.5,${0.5 + e} L1,${0.5 + e}` }, "╟": { 1: (n, e) => `M${0.5 - n},0 L${0.5 - n},1 M${0.5 + n},0 L${0.5 + n},1 M${0.5 + n},.5 L1,.5` }, "╠": { 1: (n, e) => `M${0.5 - n},0 L${0.5 - n},1 M1,${0.5 + e} L${0.5 + n},${0.5 + e} L${0.5 + n},1 M1,${0.5 - e} L${0.5 + n},${0.5 - e} L${0.5 + n},0` }, "╡": { 1: (n, e) => `M.5,0 L.5,1 M0,${0.5 - e} L.5,${0.5 - e} M0,${0.5 + e} L.5,${0.5 + e}` }, "╢": { 1: (n, e) => `M0,.5 L${0.5 - n},.5 M${0.5 - n},0 L${0.5 - n},1 M${0.5 + n},0 L${0.5 + n},1` }, "╣": { 1: (n, e) => `M${0.5 + n},0 L${0.5 + n},1 M0,${0.5 + e} L${0.5 - n},${0.5 + e} L${0.5 - n},1 M0,${0.5 - e} L${0.5 - n},${0.5 - e} L${0.5 - n},0` }, "╤": { 1: (n, e) => `M0,${0.5 - e} L1,${0.5 - e} M0,${0.5 + e} L1,${0.5 + e} M.5,${0.5 + e} L.5,1` }, "╥": { 1: (n, e) => `M0,.5 L1,.5 M${0.5 - n},.5 L${0.5 - n},1 M${0.5 + n},.5 L${0.5 + n},1` }, "╦": { 1: (n, e) => `M0,${0.5 - e} L1,${0.5 - e} M0,${0.5 + e} L${0.5 - n},${0.5 + e} L${0.5 - n},1 M1,${0.5 + e} L${0.5 + n},${0.5 + e} L${0.5 + n},1` }, "╧": { 1: (n, e) => `M.5,0 L.5,${0.5 - e} M0,${0.5 - e} L1,${0.5 - e} M0,${0.5 + e} L1,${0.5 + e}` }, "╨": { 1: (n, e) => `M0,.5 L1,.5 M${0.5 - n},.5 L${0.5 - n},0 M${0.5 + n},.5 L${0.5 + n},0` }, "╩": { 1: (n, e) => `M0,${0.5 + e} L1,${0.5 + e} M0,${0.5 - e} L${0.5 - n},${0.5 - e} L${0.5 - n},0 M1,${0.5 - e} L${0.5 + n},${0.5 - e} L${0.5 + n},0` }, "╪": { 1: (n, e) => `M.5,0 L.5,1 M0,${0.5 - e} L1,${0.5 - e} M0,${0.5 + e} L1,${0.5 + e}` }, "╫": { 1: (n, e) => `M0,.5 L1,.5 M${0.5 - n},0 L${0.5 - n},1 M${0.5 + n},0 L${0.5 + n},1` }, "╬": { 1: (n, e) => `M0,${0.5 + e} L${0.5 - n},${0.5 + e} L${0.5 - n},1 M1,${0.5 + e} L${0.5 + n},${0.5 + e} L${0.5 + n},1 M0,${0.5 - e} L${0.5 - n},${0.5 - e} L${0.5 - n},0 M1,${0.5 - e} L${0.5 + n},${0.5 - e} L${0.5 + n},0` }, "╱": { 1: "M1,0 L0,1" }, "╲": { 1: "M0,0 L1,1" }, "╳": { 1: "M1,0 L0,1 M0,0 L1,1" }, "╼": { 1: "M.5,.5 L0,.5", 3: "M.5,.5 L1,.5" }, "╽": { 1: "M.5,.5 L.5,0", 3: "M.5,.5 L.5,1" }, "╾": { 1: "M.5,.5 L1,.5", 3: "M.5,.5 L0,.5" }, "╿": { 1: "M.5,.5 L.5,1", 3: "M.5,.5 L.5,0" }, "┍": { 1: "M.5,.5 L.5,1", 3: "M.5,.5 L1,.5" }, "┎": { 1: "M.5,.5 L1,.5", 3: "M.5,.5 L.5,1" }, "┑": { 1: "M.5,.5 L.5,1", 3: "M.5,.5 L0,.5" }, "┒": { 1: "M.5,.5 L0,.5", 3: "M.5,.5 L.5,1" }, "┕": { 1: "M.5,.5 L.5,0", 3: "M.5,.5 L1,.5" }, "┖": { 1: "M.5,.5 L1,.5", 3: "M.5,.5 L.5,0" }, "┙": { 1: "M.5,.5 L.5,0", 3: "M.5,.5 L0,.5" }, "┚": { 1: "M.5,.5 L0,.5", 3: "M.5,.5 L.5,0" }, "┝": { 1: "M.5,0 L.5,1", 3: "M.5,.5 L1,.5" }, "┞": { 1: "M0.5,1 L.5,.5 L1,.5", 3: "M.5,.5 L.5,0" }, "┟": { 1: "M.5,0 L.5,.5 L1,.5", 3: "M.5,.5 L.5,1" }, "┠": { 1: "M.5,.5 L1,.5", 3: "M.5,0 L.5,1" }, "┡": { 1: "M.5,.5 L.5,1", 3: "M.5,0 L.5,.5 L1,.5" }, "┢": { 1: "M.5,.5 L.5,0", 3: "M0.5,1 L.5,.5 L1,.5" }, "┥": { 1: "M.5,0 L.5,1", 3: "M.5,.5 L0,.5" }, "┦": { 1: "M0,.5 L.5,.5 L.5,1", 3: "M.5,.5 L.5,0" }, "┧": { 1: "M.5,0 L.5,.5 L0,.5", 3: "M.5,.5 L.5,1" }, "┨": { 1: "M.5,.5 L0,.5", 3: "M.5,0 L.5,1" }, "┩": { 1: "M.5,.5 L.5,1", 3: "M.5,0 L.5,.5 L0,.5" }, "┪": { 1: "M.5,.5 L.5,0", 3: "M0,.5 L.5,.5 L.5,1" }, "┭": { 1: "M0.5,1 L.5,.5 L1,.5", 3: "M.5,.5 L0,.5" }, "┮": { 1: "M0,.5 L.5,.5 L.5,1", 3: "M.5,.5 L1,.5" }, "┯": { 1: "M.5,.5 L.5,1", 3: "M0,.5 L1,.5" }, "┰": { 1: "M0,.5 L1,.5", 3: "M.5,.5 L.5,1" }, "┱": { 1: "M.5,.5 L1,.5", 3: "M0,.5 L.5,.5 L.5,1" }, "┲": { 1: "M.5,.5 L0,.5", 3: "M0.5,1 L.5,.5 L1,.5" }, "┵": { 1: "M.5,0 L.5,.5 L1,.5", 3: "M.5,.5 L0,.5" }, "┶": { 1: "M.5,0 L.5,.5 L0,.5", 3: "M.5,.5 L1,.5" }, "┷": { 1: "M.5,.5 L.5,0", 3: "M0,.5 L1,.5" }, "┸": { 1: "M0,.5 L1,.5", 3: "M.5,.5 L.5,0" }, "┹": { 1: "M.5,.5 L1,.5", 3: "M.5,0 L.5,.5 L0,.5" }, "┺": { 1: "M.5,.5 L0,.5", 3: "M.5,0 L.5,.5 L1,.5" }, "┽": { 1: "M.5,0 L.5,1 M.5,.5 L1,.5", 3: "M.5,.5 L0,.5" }, "┾": { 1: "M.5,0 L.5,1 M.5,.5 L0,.5", 3: "M.5,.5 L1,.5" }, "┿": { 1: "M.5,0 L.5,1", 3: "M0,.5 L1,.5" }, "╀": { 1: "M0,.5 L1,.5 M.5,.5 L.5,1", 3: "M.5,.5 L.5,0" }, "╁": { 1: "M.5,.5 L.5,0 M0,.5 L1,.5", 3: "M.5,.5 L.5,1" }, "╂": { 1: "M0,.5 L1,.5", 3: "M.5,0 L.5,1" }, "╃": { 1: "M0.5,1 L.5,.5 L1,.5", 3: "M.5,0 L.5,.5 L0,.5" }, "╄": { 1: "M0,.5 L.5,.5 L.5,1", 3: "M.5,0 L.5,.5 L1,.5" }, "╅": { 1: "M.5,0 L.5,.5 L1,.5", 3: "M0,.5 L.5,.5 L.5,1" }, "╆": { 1: "M.5,0 L.5,.5 L0,.5", 3: "M0.5,1 L.5,.5 L1,.5" }, "╇": { 1: "M.5,.5 L.5,1", 3: "M.5,.5 L.5,0 M0,.5 L1,.5" }, "╈": { 1: "M.5,.5 L.5,0", 3: "M0,.5 L1,.5 M.5,.5 L.5,1" }, "╉": { 1: "M.5,.5 L1,.5", 3: "M.5,0 L.5,1 M.5,.5 L0,.5" }, "╊": { 1: "M.5,.5 L0,.5", 3: "M.5,0 L.5,1 M.5,.5 L1,.5" }, "╌": { 1: "M.1,.5 L.4,.5 M.6,.5 L.9,.5" }, "╍": { 3: "M.1,.5 L.4,.5 M.6,.5 L.9,.5" }, "┄": { 1: "M.0667,.5 L.2667,.5 M.4,.5 L.6,.5 M.7333,.5 L.9333,.5" }, "┅": { 3: "M.0667,.5 L.2667,.5 M.4,.5 L.6,.5 M.7333,.5 L.9333,.5" }, "┈": { 1: "M.05,.5 L.2,.5 M.3,.5 L.45,.5 M.55,.5 L.7,.5 M.8,.5 L.95,.5" }, "┉": { 3: "M.05,.5 L.2,.5 M.3,.5 L.45,.5 M.55,.5 L.7,.5 M.8,.5 L.95,.5" }, "╎": { 1: "M.5,.1 L.5,.4 M.5,.6 L.5,.9" }, "╏": { 3: "M.5,.1 L.5,.4 M.5,.6 L.5,.9" }, "┆": { 1: "M.5,.0667 L.5,.2667 M.5,.4 L.5,.6 M.5,.7333 L.5,.9333" }, "┇": { 3: "M.5,.0667 L.5,.2667 M.5,.4 L.5,.6 M.5,.7333 L.5,.9333" }, "┊": { 1: "M.5,.05 L.5,.2 M.5,.3 L.5,.45 L.5,.55 M.5,.7 L.5,.95" }, "┋": { 3: "M.5,.05 L.5,.2 M.5,.3 L.5,.45 L.5,.55 M.5,.7 L.5,.95" }, "╭": { 1: (n, e) => `M.5,1 L.5,${0.5 + e / 0.15 * 0.5} C.5,${0.5 + e / 0.15 * 0.5},.5,.5,1,.5` }, "╮": { 1: (n, e) => `M.5,1 L.5,${0.5 + e / 0.15 * 0.5} C.5,${0.5 + e / 0.15 * 0.5},.5,.5,0,.5` }, "╯": { 1: (n, e) => `M.5,0 L.5,${0.5 - e / 0.15 * 0.5} C.5,${0.5 - e / 0.15 * 0.5},.5,.5,0,.5` }, "╰": { 1: (n, e) => `M.5,0 L.5,${0.5 - e / 0.15 * 0.5} C.5,${0.5 - e / 0.15 * 0.5},.5,.5,1,.5` } }, t.powerlineDefinitions = { "": { d: "M0,0 L1,.5 L0,1", type: 0, rightPadding: 2 }, "": { d: "M-1,-.5 L1,.5 L-1,1.5", type: 1, leftPadding: 1, rightPadding: 1 }, "": { d: "M1,0 L0,.5 L1,1", type: 0, leftPadding: 2 }, "": { d: "M2,-.5 L0,.5 L2,1.5", type: 1, leftPadding: 1, rightPadding: 1 }, "": { d: "M0,0 L0,1 C0.552,1,1,0.776,1,.5 C1,0.224,0.552,0,0,0", type: 0, rightPadding: 1 }, "": { d: "M.2,1 C.422,1,.8,.826,.78,.5 C.8,.174,0.422,0,.2,0", type: 1, rightPadding: 1 }, "": { d: "M1,0 L1,1 C0.448,1,0,0.776,0,.5 C0,0.224,0.448,0,1,0", type: 0, leftPadding: 1 }, "": { d: "M.8,1 C0.578,1,0.2,.826,.22,.5 C0.2,0.174,0.578,0,0.8,0", type: 1, leftPadding: 1 }, "": { d: "M-.5,-.5 L1.5,1.5 L-.5,1.5", type: 0 }, "": { d: "M-.5,-.5 L1.5,1.5", type: 1, leftPadding: 1, rightPadding: 1 }, "": { d: "M1.5,-.5 L-.5,1.5 L1.5,1.5", type: 0 }, "": { d: "M1.5,-.5 L-.5,1.5 L-.5,-.5", type: 0 }, "": { d: "M1.5,-.5 L-.5,1.5", type: 1, leftPadding: 1, rightPadding: 1 }, "": { d: "M-.5,-.5 L1.5,1.5 L1.5,-.5", type: 0 } }, t.powerlineDefinitions[""] = t.powerlineDefinitions[""], t.powerlineDefinitions[""] = t.powerlineDefinitions[""], t.tryDrawCustomChar = function(n, e, o, s, i, u, p, l) {
          const m = t.blockElementDefinitions[e];
          if (m) return (function(w, S, b, x, A, P) {
            for (let k = 0; k < S.length; k++) {
              const M = S[k], y = A / 8, L = P / 8;
              w.fillRect(b + M.x * y, x + M.y * L, M.w * y, M.h * L);
            }
          })(n, m, o, s, i, u), !0;
          const _ = h[e];
          if (_) return (function(w, S, b, x, A, P) {
            let k = r.get(S);
            k || (k = /* @__PURE__ */ new Map(), r.set(S, k));
            const M = w.fillStyle;
            if (typeof M != "string") throw new Error(`Unexpected fillStyle type "${M}"`);
            let y = k.get(M);
            if (!y) {
              const L = S[0].length, R = S.length, D = w.canvas.ownerDocument.createElement("canvas");
              D.width = L, D.height = R;
              const F = (0, c.throwIfFalsy)(D.getContext("2d")), U = new ImageData(L, R);
              let K, q, O, E;
              if (M.startsWith("#")) K = parseInt(M.slice(1, 3), 16), q = parseInt(M.slice(3, 5), 16), O = parseInt(M.slice(5, 7), 16), E = M.length > 7 && parseInt(M.slice(7, 9), 16) || 1;
              else {
                if (!M.startsWith("rgba")) throw new Error(`Unexpected fillStyle color format "${M}" when drawing pattern glyph`);
                [K, q, O, E] = M.substring(5, M.length - 1).split(",").map(((H) => parseFloat(H)));
              }
              for (let H = 0; H < R; H++) for (let N = 0; N < L; N++) U.data[4 * (H * L + N)] = K, U.data[4 * (H * L + N) + 1] = q, U.data[4 * (H * L + N) + 2] = O, U.data[4 * (H * L + N) + 3] = S[H][N] * (255 * E);
              F.putImageData(U, 0, 0), y = (0, c.throwIfFalsy)(w.createPattern(D, null)), k.set(M, y);
            }
            w.fillStyle = y, w.fillRect(b, x, A, P);
          })(n, _, o, s, i, u), !0;
          const v = t.boxDrawingDefinitions[e];
          if (v) return (function(w, S, b, x, A, P, k) {
            w.strokeStyle = w.fillStyle;
            for (const [M, y] of Object.entries(S)) {
              let L;
              w.beginPath(), w.lineWidth = k * Number.parseInt(M), L = typeof y == "function" ? y(0.15, 0.15 / P * A) : y;
              for (const R of L.split(" ")) {
                const D = R[0], F = f[D];
                if (!F) {
                  console.error(`Could not find drawing instructions for "${D}"`);
                  continue;
                }
                const U = R.substring(1).split(",");
                U[0] && U[1] && F(w, g(U, A, P, b, x, !0, k));
              }
              w.stroke(), w.closePath();
            }
          })(n, v, o, s, i, u, l), !0;
          const C = t.powerlineDefinitions[e];
          return !!C && ((function(w, S, b, x, A, P, k, M) {
            var R, D;
            const y = new Path2D();
            y.rect(b, x, A, P), w.clip(y), w.beginPath();
            const L = k / 12;
            w.lineWidth = M * L;
            for (const F of S.d.split(" ")) {
              const U = F[0], K = f[U];
              if (!K) {
                console.error(`Could not find drawing instructions for "${U}"`);
                continue;
              }
              const q = F.substring(1).split(",");
              q[0] && q[1] && K(w, g(q, A, P, b, x, !1, M, ((R = S.leftPadding) != null ? R : 0) * (L / 2), ((D = S.rightPadding) != null ? D : 0) * (L / 2)));
            }
            S.type === 1 ? (w.strokeStyle = w.fillStyle, w.stroke()) : w.fill(), w.closePath();
          })(n, C, o, s, i, u, p, l), !0);
        };
        const r = /* @__PURE__ */ new Map();
        function d(n, e, o = 0) {
          return Math.max(Math.min(n, e), o);
        }
        const f = { C: (n, e) => n.bezierCurveTo(e[0], e[1], e[2], e[3], e[4], e[5]), L: (n, e) => n.lineTo(e[0], e[1]), M: (n, e) => n.moveTo(e[0], e[1]) };
        function g(n, e, o, s, i, u, p, l = 0, m = 0) {
          const _ = n.map(((v) => parseFloat(v) || parseInt(v)));
          if (_.length < 2) throw new Error("Too few arguments for instruction");
          for (let v = 0; v < _.length; v += 2) _[v] *= e - l * p - m * p, u && _[v] !== 0 && (_[v] = d(Math.round(_[v] + 0.5) - 0.5, e, 0)), _[v] += s + l * p;
          for (let v = 1; v < _.length; v += 2) _[v] *= o, u && _[v] !== 0 && (_[v] = d(Math.round(_[v] + 0.5) - 0.5, o, 0)), _[v] += i;
          return _;
        }
      }, 56: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.observeDevicePixelDimensions = void 0;
        const c = a(859);
        t.observeDevicePixelDimensions = function(h, r, d) {
          let f = new r.ResizeObserver(((g) => {
            const n = g.find(((s) => s.target === h));
            if (!n) return;
            if (!("devicePixelContentBoxSize" in n)) return f == null || f.disconnect(), void (f = void 0);
            const e = n.devicePixelContentBoxSize[0].inlineSize, o = n.devicePixelContentBoxSize[0].blockSize;
            e > 0 && o > 0 && d(e, o);
          }));
          try {
            f.observe(h, { box: ["device-pixel-content-box"] });
          } catch (g) {
            f.disconnect(), f = void 0;
          }
          return (0, c.toDisposable)((() => f == null ? void 0 : f.disconnect()));
        };
      }, 374: (T, t) => {
        function a(h) {
          return 57508 <= h && h <= 57558;
        }
        function c(h) {
          return h >= 128512 && h <= 128591 || h >= 127744 && h <= 128511 || h >= 128640 && h <= 128767 || h >= 9728 && h <= 9983 || h >= 9984 && h <= 10175 || h >= 65024 && h <= 65039 || h >= 129280 && h <= 129535 || h >= 127462 && h <= 127487;
        }
        Object.defineProperty(t, "__esModule", { value: !0 }), t.computeNextVariantOffset = t.createRenderDimensions = t.treatGlyphAsBackgroundColor = t.allowRescaling = t.isEmoji = t.isRestrictedPowerlineGlyph = t.isPowerlineGlyph = t.throwIfFalsy = void 0, t.throwIfFalsy = function(h) {
          if (!h) throw new Error("value must not be falsy");
          return h;
        }, t.isPowerlineGlyph = a, t.isRestrictedPowerlineGlyph = function(h) {
          return 57520 <= h && h <= 57527;
        }, t.isEmoji = c, t.allowRescaling = function(h, r, d, f) {
          return r === 1 && d > Math.ceil(1.5 * f) && h !== void 0 && h > 255 && !c(h) && !a(h) && !(function(g) {
            return 57344 <= g && g <= 63743;
          })(h);
        }, t.treatGlyphAsBackgroundColor = function(h) {
          return a(h) || (function(r) {
            return 9472 <= r && r <= 9631;
          })(h);
        }, t.createRenderDimensions = function() {
          return { css: { canvas: { width: 0, height: 0 }, cell: { width: 0, height: 0 } }, device: { canvas: { width: 0, height: 0 }, cell: { width: 0, height: 0 }, char: { width: 0, height: 0, left: 0, top: 0 } } };
        }, t.computeNextVariantOffset = function(h, r, d = 0) {
          return (h - (2 * Math.round(r) - d)) % (2 * Math.round(r));
        };
      }, 296: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.createSelectionRenderModel = void 0;
        class a {
          constructor() {
            this.clear();
          }
          clear() {
            this.hasSelection = !1, this.columnSelectMode = !1, this.viewportStartRow = 0, this.viewportEndRow = 0, this.viewportCappedStartRow = 0, this.viewportCappedEndRow = 0, this.startCol = 0, this.endCol = 0, this.selectionStart = void 0, this.selectionEnd = void 0;
          }
          update(h, r, d, f = !1) {
            if (this.selectionStart = r, this.selectionEnd = d, !r || !d || r[0] === d[0] && r[1] === d[1]) return void this.clear();
            const g = h.buffers.active.ydisp, n = r[1] - g, e = d[1] - g, o = Math.max(n, 0), s = Math.min(e, h.rows - 1);
            o >= h.rows || s < 0 ? this.clear() : (this.hasSelection = !0, this.columnSelectMode = f, this.viewportStartRow = n, this.viewportEndRow = e, this.viewportCappedStartRow = o, this.viewportCappedEndRow = s, this.startCol = r[0], this.endCol = d[0]);
          }
          isCellSelected(h, r, d) {
            return !!this.hasSelection && (d -= h.buffer.active.viewportY, this.columnSelectMode ? this.startCol <= this.endCol ? r >= this.startCol && d >= this.viewportCappedStartRow && r < this.endCol && d <= this.viewportCappedEndRow : r < this.startCol && d >= this.viewportCappedStartRow && r >= this.endCol && d <= this.viewportCappedEndRow : d > this.viewportStartRow && d < this.viewportEndRow || this.viewportStartRow === this.viewportEndRow && d === this.viewportStartRow && r >= this.startCol && r < this.endCol || this.viewportStartRow < this.viewportEndRow && d === this.viewportEndRow && r < this.endCol || this.viewportStartRow < this.viewportEndRow && d === this.viewportStartRow && r >= this.startCol);
          }
        }
        t.createSelectionRenderModel = function() {
          return new a();
        };
      }, 509: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.TextureAtlas = void 0;
        const c = a(237), h = a(860), r = a(374), d = a(160), f = a(345), g = a(485), n = a(385), e = a(147), o = a(855), s = { texturePage: 0, texturePosition: { x: 0, y: 0 }, texturePositionClipSpace: { x: 0, y: 0 }, offset: { x: 0, y: 0 }, size: { x: 0, y: 0 }, sizeClipSpace: { x: 0, y: 0 } };
        let i;
        class u {
          get pages() {
            return this._pages;
          }
          constructor(v, C, w) {
            this._document = v, this._config = C, this._unicodeService = w, this._didWarmUp = !1, this._cacheMap = new g.FourKeyMap(), this._cacheMapCombined = new g.FourKeyMap(), this._pages = [], this._activePages = [], this._workBoundingBox = { top: 0, left: 0, bottom: 0, right: 0 }, this._workAttributeData = new e.AttributeData(), this._textureSize = 512, this._onAddTextureAtlasCanvas = new f.EventEmitter(), this.onAddTextureAtlasCanvas = this._onAddTextureAtlasCanvas.event, this._onRemoveTextureAtlasCanvas = new f.EventEmitter(), this.onRemoveTextureAtlasCanvas = this._onRemoveTextureAtlasCanvas.event, this._requestClearModel = !1, this._createNewPage(), this._tmpCanvas = m(v, 4 * this._config.deviceCellWidth + 4, this._config.deviceCellHeight + 4), this._tmpCtx = (0, r.throwIfFalsy)(this._tmpCanvas.getContext("2d", { alpha: this._config.allowTransparency, willReadFrequently: !0 }));
          }
          dispose() {
            for (const v of this.pages) v.canvas.remove();
            this._onAddTextureAtlasCanvas.dispose();
          }
          warmUp() {
            this._didWarmUp || (this._doWarmUp(), this._didWarmUp = !0);
          }
          _doWarmUp() {
            const v = new n.IdleTaskQueue();
            for (let C = 33; C < 126; C++) v.enqueue((() => {
              if (!this._cacheMap.get(C, o.DEFAULT_COLOR, o.DEFAULT_COLOR, o.DEFAULT_EXT)) {
                const w = this._drawToCache(C, o.DEFAULT_COLOR, o.DEFAULT_COLOR, o.DEFAULT_EXT);
                this._cacheMap.set(C, o.DEFAULT_COLOR, o.DEFAULT_COLOR, o.DEFAULT_EXT, w);
              }
            }));
          }
          beginFrame() {
            return this._requestClearModel;
          }
          clearTexture() {
            if (this._pages[0].currentRow.x !== 0 || this._pages[0].currentRow.y !== 0) {
              for (const v of this._pages) v.clear();
              this._cacheMap.clear(), this._cacheMapCombined.clear(), this._didWarmUp = !1;
            }
          }
          _createNewPage() {
            if (u.maxAtlasPages && this._pages.length >= Math.max(4, u.maxAtlasPages)) {
              const C = this._pages.filter(((k) => 2 * k.canvas.width <= (u.maxTextureSize || 4096))).sort(((k, M) => M.canvas.width !== k.canvas.width ? M.canvas.width - k.canvas.width : M.percentageUsed - k.percentageUsed));
              let w = -1, S = 0;
              for (let k = 0; k < C.length; k++) if (C[k].canvas.width !== S) w = k, S = C[k].canvas.width;
              else if (k - w == 3) break;
              const b = C.slice(w, w + 4), x = b.map(((k) => k.glyphs[0].texturePage)).sort(((k, M) => k > M ? 1 : -1)), A = this.pages.length - b.length, P = this._mergePages(b, A);
              P.version++;
              for (let k = x.length - 1; k >= 0; k--) this._deletePage(x[k]);
              this.pages.push(P), this._requestClearModel = !0, this._onAddTextureAtlasCanvas.fire(P.canvas);
            }
            const v = new p(this._document, this._textureSize);
            return this._pages.push(v), this._activePages.push(v), this._onAddTextureAtlasCanvas.fire(v.canvas), v;
          }
          _mergePages(v, C) {
            const w = 2 * v[0].canvas.width, S = new p(this._document, w, v);
            for (const [b, x] of v.entries()) {
              const A = b * x.canvas.width % w, P = Math.floor(b / 2) * x.canvas.height;
              S.ctx.drawImage(x.canvas, A, P);
              for (const M of x.glyphs) M.texturePage = C, M.sizeClipSpace.x = M.size.x / w, M.sizeClipSpace.y = M.size.y / w, M.texturePosition.x += A, M.texturePosition.y += P, M.texturePositionClipSpace.x = M.texturePosition.x / w, M.texturePositionClipSpace.y = M.texturePosition.y / w;
              this._onRemoveTextureAtlasCanvas.fire(x.canvas);
              const k = this._activePages.indexOf(x);
              k !== -1 && this._activePages.splice(k, 1);
            }
            return S;
          }
          _deletePage(v) {
            this._pages.splice(v, 1);
            for (let C = v; C < this._pages.length; C++) {
              const w = this._pages[C];
              for (const S of w.glyphs) S.texturePage--;
              w.version++;
            }
          }
          getRasterizedGlyphCombinedChar(v, C, w, S, b) {
            return this._getFromCacheMap(this._cacheMapCombined, v, C, w, S, b);
          }
          getRasterizedGlyph(v, C, w, S, b) {
            return this._getFromCacheMap(this._cacheMap, v, C, w, S, b);
          }
          _getFromCacheMap(v, C, w, S, b, x = !1) {
            return i = v.get(C, w, S, b), i || (i = this._drawToCache(C, w, S, b, x), v.set(C, w, S, b, i)), i;
          }
          _getColorFromAnsiIndex(v) {
            if (v >= this._config.colors.ansi.length) throw new Error("No color found for idx " + v);
            return this._config.colors.ansi[v];
          }
          _getBackgroundColor(v, C, w, S) {
            if (this._config.allowTransparency) return d.NULL_COLOR;
            let b;
            switch (v) {
              case 16777216:
              case 33554432:
                b = this._getColorFromAnsiIndex(C);
                break;
              case 50331648:
                const x = e.AttributeData.toColorRGB(C);
                b = d.channels.toColor(x[0], x[1], x[2]);
                break;
              default:
                b = w ? d.color.opaque(this._config.colors.foreground) : this._config.colors.background;
            }
            return b;
          }
          _getForegroundColor(v, C, w, S, b, x, A, P, k, M) {
            const y = this._getMinimumContrastColor(v, C, w, S, b, x, A, k, P, M);
            if (y) return y;
            let L;
            switch (b) {
              case 16777216:
              case 33554432:
                this._config.drawBoldTextInBrightColors && k && x < 8 && (x += 8), L = this._getColorFromAnsiIndex(x);
                break;
              case 50331648:
                const R = e.AttributeData.toColorRGB(x);
                L = d.channels.toColor(R[0], R[1], R[2]);
                break;
              default:
                L = A ? this._config.colors.background : this._config.colors.foreground;
            }
            return this._config.allowTransparency && (L = d.color.opaque(L)), P && (L = d.color.multiplyOpacity(L, c.DIM_OPACITY)), L;
          }
          _resolveBackgroundRgba(v, C, w) {
            switch (v) {
              case 16777216:
              case 33554432:
                return this._getColorFromAnsiIndex(C).rgba;
              case 50331648:
                return C << 8;
              default:
                return w ? this._config.colors.foreground.rgba : this._config.colors.background.rgba;
            }
          }
          _resolveForegroundRgba(v, C, w, S) {
            switch (v) {
              case 16777216:
              case 33554432:
                return this._config.drawBoldTextInBrightColors && S && C < 8 && (C += 8), this._getColorFromAnsiIndex(C).rgba;
              case 50331648:
                return C << 8;
              default:
                return w ? this._config.colors.background.rgba : this._config.colors.foreground.rgba;
            }
          }
          _getMinimumContrastColor(v, C, w, S, b, x, A, P, k, M) {
            if (this._config.minimumContrastRatio === 1 || M) return;
            const y = this._getContrastCache(k), L = y.getColor(v, S);
            if (L !== void 0) return L || void 0;
            const R = this._resolveBackgroundRgba(C, w, A), D = this._resolveForegroundRgba(b, x, A, P), F = d.rgba.ensureContrastRatio(R, D, this._config.minimumContrastRatio / (k ? 2 : 1));
            if (!F) return void y.setColor(v, S, null);
            const U = d.channels.toColor(F >> 24 & 255, F >> 16 & 255, F >> 8 & 255);
            return y.setColor(v, S, U), U;
          }
          _getContrastCache(v) {
            return v ? this._config.colors.halfContrastCache : this._config.colors.contrastCache;
          }
          _drawToCache(v, C, w, S, b = !1) {
            const x = typeof v == "number" ? String.fromCharCode(v) : v, A = Math.min(this._config.deviceCellWidth * Math.max(x.length, 2) + 4, this._textureSize);
            this._tmpCanvas.width < A && (this._tmpCanvas.width = A);
            const P = Math.min(this._config.deviceCellHeight + 8, this._textureSize);
            if (this._tmpCanvas.height < P && (this._tmpCanvas.height = P), this._tmpCtx.save(), this._workAttributeData.fg = w, this._workAttributeData.bg = C, this._workAttributeData.extended.ext = S, this._workAttributeData.isInvisible()) return s;
            const k = !!this._workAttributeData.isBold(), M = !!this._workAttributeData.isInverse(), y = !!this._workAttributeData.isDim(), L = !!this._workAttributeData.isItalic(), R = !!this._workAttributeData.isUnderline(), D = !!this._workAttributeData.isStrikethrough(), F = !!this._workAttributeData.isOverline();
            let U = this._workAttributeData.getFgColor(), K = this._workAttributeData.getFgColorMode(), q = this._workAttributeData.getBgColor(), O = this._workAttributeData.getBgColorMode();
            if (M) {
              const z = U;
              U = q, q = z;
              const Q = K;
              K = O, O = Q;
            }
            const E = this._getBackgroundColor(O, q, M, y);
            this._tmpCtx.globalCompositeOperation = "copy", this._tmpCtx.fillStyle = E.css, this._tmpCtx.fillRect(0, 0, this._tmpCanvas.width, this._tmpCanvas.height), this._tmpCtx.globalCompositeOperation = "source-over";
            const H = k ? this._config.fontWeightBold : this._config.fontWeight, N = L ? "italic" : "";
            this._tmpCtx.font = `${N} ${H} ${this._config.fontSize * this._config.devicePixelRatio}px ${this._config.fontFamily}`, this._tmpCtx.textBaseline = c.TEXT_BASELINE;
            const G = x.length === 1 && (0, r.isPowerlineGlyph)(x.charCodeAt(0)), j = x.length === 1 && (0, r.isRestrictedPowerlineGlyph)(x.charCodeAt(0)), ie = this._getForegroundColor(C, O, q, w, K, U, M, y, k, (0, r.treatGlyphAsBackgroundColor)(x.charCodeAt(0)));
            this._tmpCtx.fillStyle = ie.css;
            const V = j ? 0 : 4;
            let ae = !1;
            this._config.customGlyphs !== !1 && (ae = (0, h.tryDrawCustomChar)(this._tmpCtx, x, V, V, this._config.deviceCellWidth, this._config.deviceCellHeight, this._config.fontSize, this._config.devicePixelRatio));
            let ce, ee = !G;
            if (ce = typeof v == "number" ? this._unicodeService.wcwidth(v) : this._unicodeService.getStringCellWidth(v), R) {
              this._tmpCtx.save();
              const z = Math.max(1, Math.floor(this._config.fontSize * this._config.devicePixelRatio / 15)), Q = z % 2 == 1 ? 0.5 : 0;
              if (this._tmpCtx.lineWidth = z, this._workAttributeData.isUnderlineColorDefault()) this._tmpCtx.strokeStyle = this._tmpCtx.fillStyle;
              else if (this._workAttributeData.isUnderlineColorRGB()) ee = !1, this._tmpCtx.strokeStyle = `rgb(${e.AttributeData.toColorRGB(this._workAttributeData.getUnderlineColor()).join(",")})`;
              else {
                ee = !1;
                let le = this._workAttributeData.getUnderlineColor();
                this._config.drawBoldTextInBrightColors && this._workAttributeData.isBold() && le < 8 && (le += 8), this._tmpCtx.strokeStyle = this._getColorFromAnsiIndex(le).css;
              }
              this._tmpCtx.beginPath();
              const he = V, re = Math.ceil(V + this._config.deviceCharHeight) - Q - (b ? 2 * z : 0), fe = re + z, de = re + 2 * z;
              let ue = this._workAttributeData.getUnderlineVariantOffset();
              for (let le = 0; le < ce; le++) {
                this._tmpCtx.save();
                const se = he + le * this._config.deviceCellWidth, te = he + (le + 1) * this._config.deviceCellWidth, ve = se + this._config.deviceCellWidth / 2;
                switch (this._workAttributeData.extended.underlineStyle) {
                  case 2:
                    this._tmpCtx.moveTo(se, re), this._tmpCtx.lineTo(te, re), this._tmpCtx.moveTo(se, de), this._tmpCtx.lineTo(te, de);
                    break;
                  case 3:
                    const pe = z <= 1 ? de : Math.ceil(V + this._config.deviceCharHeight - z / 2) - Q, me = z <= 1 ? re : Math.ceil(V + this._config.deviceCharHeight + z / 2) - Q, we = new Path2D();
                    we.rect(se, re, this._config.deviceCellWidth, de - re), this._tmpCtx.clip(we), this._tmpCtx.moveTo(se - this._config.deviceCellWidth / 2, fe), this._tmpCtx.bezierCurveTo(se - this._config.deviceCellWidth / 2, me, se, me, se, fe), this._tmpCtx.bezierCurveTo(se, pe, ve, pe, ve, fe), this._tmpCtx.bezierCurveTo(ve, me, te, me, te, fe), this._tmpCtx.bezierCurveTo(te, pe, te + this._config.deviceCellWidth / 2, pe, te + this._config.deviceCellWidth / 2, fe);
                    break;
                  case 4:
                    const Ce = ue === 0 ? 0 : ue >= z ? 2 * z - ue : z - ue;
                    ue >= z || Ce === 0 ? (this._tmpCtx.setLineDash([Math.round(z), Math.round(z)]), this._tmpCtx.moveTo(se + Ce, re), this._tmpCtx.lineTo(te, re)) : (this._tmpCtx.setLineDash([Math.round(z), Math.round(z)]), this._tmpCtx.moveTo(se, re), this._tmpCtx.lineTo(se + Ce, re), this._tmpCtx.moveTo(se + Ce + z, re), this._tmpCtx.lineTo(te, re)), ue = (0, r.computeNextVariantOffset)(te - se, z, ue);
                    break;
                  case 5:
                    const Ee = 0.6, Re = 0.3, Se = te - se, be = Math.floor(Ee * Se), ye = Math.floor(Re * Se), Me = Se - be - ye;
                    this._tmpCtx.setLineDash([be, ye, Me]), this._tmpCtx.moveTo(se, re), this._tmpCtx.lineTo(te, re);
                    break;
                  default:
                    this._tmpCtx.moveTo(se, re), this._tmpCtx.lineTo(te, re);
                }
                this._tmpCtx.stroke(), this._tmpCtx.restore();
              }
              if (this._tmpCtx.restore(), !ae && this._config.fontSize >= 12 && !this._config.allowTransparency && x !== " ") {
                this._tmpCtx.save(), this._tmpCtx.textBaseline = "alphabetic";
                const le = this._tmpCtx.measureText(x);
                if (this._tmpCtx.restore(), "actualBoundingBoxDescent" in le && le.actualBoundingBoxDescent > 0) {
                  this._tmpCtx.save();
                  const se = new Path2D();
                  se.rect(he, re - Math.ceil(z / 2), this._config.deviceCellWidth * ce, de - re + Math.ceil(z / 2)), this._tmpCtx.clip(se), this._tmpCtx.lineWidth = 3 * this._config.devicePixelRatio, this._tmpCtx.strokeStyle = E.css, this._tmpCtx.strokeText(x, V, V + this._config.deviceCharHeight), this._tmpCtx.restore();
                }
              }
            }
            if (F) {
              const z = Math.max(1, Math.floor(this._config.fontSize * this._config.devicePixelRatio / 15)), Q = z % 2 == 1 ? 0.5 : 0;
              this._tmpCtx.lineWidth = z, this._tmpCtx.strokeStyle = this._tmpCtx.fillStyle, this._tmpCtx.beginPath(), this._tmpCtx.moveTo(V, V + Q), this._tmpCtx.lineTo(V + this._config.deviceCharWidth * ce, V + Q), this._tmpCtx.stroke();
            }
            if (ae || this._tmpCtx.fillText(x, V, V + this._config.deviceCharHeight), x === "_" && !this._config.allowTransparency) {
              let z = l(this._tmpCtx.getImageData(V, V, this._config.deviceCellWidth, this._config.deviceCellHeight), E, ie, ee);
              if (z) for (let Q = 1; Q <= 5 && (this._tmpCtx.save(), this._tmpCtx.fillStyle = E.css, this._tmpCtx.fillRect(0, 0, this._tmpCanvas.width, this._tmpCanvas.height), this._tmpCtx.restore(), this._tmpCtx.fillText(x, V, V + this._config.deviceCharHeight - Q), z = l(this._tmpCtx.getImageData(V, V, this._config.deviceCellWidth, this._config.deviceCellHeight), E, ie, ee), z); Q++) ;
            }
            if (D) {
              const z = Math.max(1, Math.floor(this._config.fontSize * this._config.devicePixelRatio / 10)), Q = this._tmpCtx.lineWidth % 2 == 1 ? 0.5 : 0;
              this._tmpCtx.lineWidth = z, this._tmpCtx.strokeStyle = this._tmpCtx.fillStyle, this._tmpCtx.beginPath(), this._tmpCtx.moveTo(V, V + Math.floor(this._config.deviceCharHeight / 2) - Q), this._tmpCtx.lineTo(V + this._config.deviceCharWidth * ce, V + Math.floor(this._config.deviceCharHeight / 2) - Q), this._tmpCtx.stroke();
            }
            this._tmpCtx.restore();
            const _e = this._tmpCtx.getImageData(0, 0, this._tmpCanvas.width, this._tmpCanvas.height);
            let ge;
            if (ge = this._config.allowTransparency ? (function(z) {
              for (let Q = 0; Q < z.data.length; Q += 4) if (z.data[Q + 3] > 0) return !1;
              return !0;
            })(_e) : l(_e, E, ie, ee), ge) return s;
            const Z = this._findGlyphBoundingBox(_e, this._workBoundingBox, A, j, ae, V);
            let X, J;
            for (; ; ) {
              if (this._activePages.length === 0) {
                const z = this._createNewPage();
                X = z, J = z.currentRow, J.height = Z.size.y;
                break;
              }
              X = this._activePages[this._activePages.length - 1], J = X.currentRow;
              for (const z of this._activePages) Z.size.y <= z.currentRow.height && (X = z, J = z.currentRow);
              for (let z = this._activePages.length - 1; z >= 0; z--) for (const Q of this._activePages[z].fixedRows) Q.height <= J.height && Z.size.y <= Q.height && (X = this._activePages[z], J = Q);
              if (J.y + Z.size.y >= X.canvas.height || J.height > Z.size.y + 2) {
                let z = !1;
                if (X.currentRow.y + X.currentRow.height + Z.size.y >= X.canvas.height) {
                  let Q;
                  for (const he of this._activePages) if (he.currentRow.y + he.currentRow.height + Z.size.y < he.canvas.height) {
                    Q = he;
                    break;
                  }
                  if (Q) X = Q;
                  else if (u.maxAtlasPages && this._pages.length >= u.maxAtlasPages && J.y + Z.size.y <= X.canvas.height && J.height >= Z.size.y && J.x + Z.size.x <= X.canvas.width) z = !0;
                  else {
                    const he = this._createNewPage();
                    X = he, J = he.currentRow, J.height = Z.size.y, z = !0;
                  }
                }
                z || (X.currentRow.height > 0 && X.fixedRows.push(X.currentRow), J = { x: 0, y: X.currentRow.y + X.currentRow.height, height: Z.size.y }, X.fixedRows.push(J), X.currentRow = { x: 0, y: J.y + J.height, height: 0 });
              }
              if (J.x + Z.size.x <= X.canvas.width) break;
              J === X.currentRow ? (J.x = 0, J.y += J.height, J.height = 0) : X.fixedRows.splice(X.fixedRows.indexOf(J), 1);
            }
            return Z.texturePage = this._pages.indexOf(X), Z.texturePosition.x = J.x, Z.texturePosition.y = J.y, Z.texturePositionClipSpace.x = J.x / X.canvas.width, Z.texturePositionClipSpace.y = J.y / X.canvas.height, Z.sizeClipSpace.x /= X.canvas.width, Z.sizeClipSpace.y /= X.canvas.height, J.height = Math.max(J.height, Z.size.y), J.x += Z.size.x, X.ctx.putImageData(_e, Z.texturePosition.x - this._workBoundingBox.left, Z.texturePosition.y - this._workBoundingBox.top, this._workBoundingBox.left, this._workBoundingBox.top, Z.size.x, Z.size.y), X.addGlyph(Z), X.version++, Z;
          }
          _findGlyphBoundingBox(v, C, w, S, b, x) {
            C.top = 0;
            const A = S ? this._config.deviceCellHeight : this._tmpCanvas.height, P = S ? this._config.deviceCellWidth : w;
            let k = !1;
            for (let M = 0; M < A; M++) {
              for (let y = 0; y < P; y++) {
                const L = M * this._tmpCanvas.width * 4 + 4 * y + 3;
                if (v.data[L] !== 0) {
                  C.top = M, k = !0;
                  break;
                }
              }
              if (k) break;
            }
            C.left = 0, k = !1;
            for (let M = 0; M < x + P; M++) {
              for (let y = 0; y < A; y++) {
                const L = y * this._tmpCanvas.width * 4 + 4 * M + 3;
                if (v.data[L] !== 0) {
                  C.left = M, k = !0;
                  break;
                }
              }
              if (k) break;
            }
            C.right = P, k = !1;
            for (let M = x + P - 1; M >= x; M--) {
              for (let y = 0; y < A; y++) {
                const L = y * this._tmpCanvas.width * 4 + 4 * M + 3;
                if (v.data[L] !== 0) {
                  C.right = M, k = !0;
                  break;
                }
              }
              if (k) break;
            }
            C.bottom = A, k = !1;
            for (let M = A - 1; M >= 0; M--) {
              for (let y = 0; y < P; y++) {
                const L = M * this._tmpCanvas.width * 4 + 4 * y + 3;
                if (v.data[L] !== 0) {
                  C.bottom = M, k = !0;
                  break;
                }
              }
              if (k) break;
            }
            return { texturePage: 0, texturePosition: { x: 0, y: 0 }, texturePositionClipSpace: { x: 0, y: 0 }, size: { x: C.right - C.left + 1, y: C.bottom - C.top + 1 }, sizeClipSpace: { x: C.right - C.left + 1, y: C.bottom - C.top + 1 }, offset: { x: -C.left + x + (S || b ? Math.floor((this._config.deviceCellWidth - this._config.deviceCharWidth) / 2) : 0), y: -C.top + x + (S || b ? this._config.lineHeight === 1 ? 0 : Math.round((this._config.deviceCellHeight - this._config.deviceCharHeight) / 2) : 0) } };
          }
        }
        t.TextureAtlas = u;
        class p {
          get percentageUsed() {
            return this._usedPixels / (this.canvas.width * this.canvas.height);
          }
          get glyphs() {
            return this._glyphs;
          }
          addGlyph(v) {
            this._glyphs.push(v), this._usedPixels += v.size.x * v.size.y;
          }
          constructor(v, C, w) {
            if (this._usedPixels = 0, this._glyphs = [], this.version = 0, this.currentRow = { x: 0, y: 0, height: 0 }, this.fixedRows = [], w) for (const S of w) this._glyphs.push(...S.glyphs), this._usedPixels += S._usedPixels;
            this.canvas = m(v, C, C), this.ctx = (0, r.throwIfFalsy)(this.canvas.getContext("2d", { alpha: !0 }));
          }
          clear() {
            this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height), this.currentRow.x = 0, this.currentRow.y = 0, this.currentRow.height = 0, this.fixedRows.length = 0, this.version++;
          }
        }
        function l(_, v, C, w) {
          const S = v.rgba >>> 24, b = v.rgba >>> 16 & 255, x = v.rgba >>> 8 & 255, A = C.rgba >>> 24, P = C.rgba >>> 16 & 255, k = C.rgba >>> 8 & 255, M = Math.floor((Math.abs(S - A) + Math.abs(b - P) + Math.abs(x - k)) / 12);
          let y = !0;
          for (let L = 0; L < _.data.length; L += 4) _.data[L] === S && _.data[L + 1] === b && _.data[L + 2] === x || w && Math.abs(_.data[L] - S) + Math.abs(_.data[L + 1] - b) + Math.abs(_.data[L + 2] - x) < M ? _.data[L + 3] = 0 : y = !1;
          return y;
        }
        function m(_, v, C) {
          const w = _.createElement("canvas");
          return w.width = v, w.height = C, w;
        }
      }, 577: function(T, t, a) {
        var c = this && this.__decorate || function(o, s, i, u) {
          var p, l = arguments.length, m = l < 3 ? s : u === null ? u = Object.getOwnPropertyDescriptor(s, i) : u;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") m = Reflect.decorate(o, s, i, u);
          else for (var _ = o.length - 1; _ >= 0; _--) (p = o[_]) && (m = (l < 3 ? p(m) : l > 3 ? p(s, i, m) : p(s, i)) || m);
          return l > 3 && m && Object.defineProperty(s, i, m), m;
        }, h = this && this.__param || function(o, s) {
          return function(i, u) {
            s(i, u, o);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CharacterJoinerService = t.JoinedCellData = void 0;
        const r = a(147), d = a(855), f = a(782), g = a(97);
        class n extends r.AttributeData {
          constructor(s, i, u) {
            super(), this.content = 0, this.combinedData = "", this.fg = s.fg, this.bg = s.bg, this.combinedData = i, this._width = u;
          }
          isCombined() {
            return 2097152;
          }
          getWidth() {
            return this._width;
          }
          getChars() {
            return this.combinedData;
          }
          getCode() {
            return 2097151;
          }
          setFromCharData(s) {
            throw new Error("not implemented");
          }
          getAsCharData() {
            return [this.fg, this.getChars(), this.getWidth(), this.getCode()];
          }
        }
        t.JoinedCellData = n;
        let e = t.CharacterJoinerService = class Ge {
          constructor(s) {
            this._bufferService = s, this._characterJoiners = [], this._nextCharacterJoinerId = 0, this._workCell = new f.CellData();
          }
          register(s) {
            const i = { id: this._nextCharacterJoinerId++, handler: s };
            return this._characterJoiners.push(i), i.id;
          }
          deregister(s) {
            for (let i = 0; i < this._characterJoiners.length; i++) if (this._characterJoiners[i].id === s) return this._characterJoiners.splice(i, 1), !0;
            return !1;
          }
          getJoinedCharacters(s) {
            if (this._characterJoiners.length === 0) return [];
            const i = this._bufferService.buffer.lines.get(s);
            if (!i || i.length === 0) return [];
            const u = [], p = i.translateToString(!0);
            let l = 0, m = 0, _ = 0, v = i.getFg(0), C = i.getBg(0);
            for (let w = 0; w < i.getTrimmedLength(); w++) if (i.loadCell(w, this._workCell), this._workCell.getWidth() !== 0) {
              if (this._workCell.fg !== v || this._workCell.bg !== C) {
                if (w - l > 1) {
                  const S = this._getJoinedRanges(p, _, m, i, l);
                  for (let b = 0; b < S.length; b++) u.push(S[b]);
                }
                l = w, _ = m, v = this._workCell.fg, C = this._workCell.bg;
              }
              m += this._workCell.getChars().length || d.WHITESPACE_CELL_CHAR.length;
            }
            if (this._bufferService.cols - l > 1) {
              const w = this._getJoinedRanges(p, _, m, i, l);
              for (let S = 0; S < w.length; S++) u.push(w[S]);
            }
            return u;
          }
          _getJoinedRanges(s, i, u, p, l) {
            const m = s.substring(i, u);
            let _ = [];
            try {
              _ = this._characterJoiners[0].handler(m);
            } catch (v) {
              console.error(v);
            }
            for (let v = 1; v < this._characterJoiners.length; v++) try {
              const C = this._characterJoiners[v].handler(m);
              for (let w = 0; w < C.length; w++) Ge._mergeRanges(_, C[w]);
            } catch (C) {
              console.error(C);
            }
            return this._stringRangesToCellRanges(_, p, l), _;
          }
          _stringRangesToCellRanges(s, i, u) {
            let p = 0, l = !1, m = 0, _ = s[p];
            if (_) {
              for (let v = u; v < this._bufferService.cols; v++) {
                const C = i.getWidth(v), w = i.getString(v).length || d.WHITESPACE_CELL_CHAR.length;
                if (C !== 0) {
                  if (!l && _[0] <= m && (_[0] = v, l = !0), _[1] <= m) {
                    if (_[1] = v, _ = s[++p], !_) break;
                    _[0] <= m ? (_[0] = v, l = !0) : l = !1;
                  }
                  m += w;
                }
              }
              _ && (_[1] = this._bufferService.cols);
            }
          }
          static _mergeRanges(s, i) {
            let u = !1;
            for (let p = 0; p < s.length; p++) {
              const l = s[p];
              if (u) {
                if (i[1] <= l[0]) return s[p - 1][1] = i[1], s;
                if (i[1] <= l[1]) return s[p - 1][1] = Math.max(i[1], l[1]), s.splice(p, 1), s;
                s.splice(p, 1), p--;
              } else {
                if (i[1] <= l[0]) return s.splice(p, 0, i), s;
                if (i[1] <= l[1]) return l[0] = Math.min(i[0], l[0]), s;
                i[0] < l[1] && (l[0] = Math.min(i[0], l[0]), u = !0);
              }
            }
            return u ? s[s.length - 1][1] = i[1] : s.push(i), s;
          }
        };
        t.CharacterJoinerService = e = c([h(0, g.IBufferService)], e);
      }, 160: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.contrastRatio = t.toPaddedHex = t.rgba = t.rgb = t.css = t.color = t.channels = t.NULL_COLOR = void 0;
        let a = 0, c = 0, h = 0, r = 0;
        var d, f, g, n, e;
        function o(i) {
          const u = i.toString(16);
          return u.length < 2 ? "0" + u : u;
        }
        function s(i, u) {
          return i < u ? (u + 0.05) / (i + 0.05) : (i + 0.05) / (u + 0.05);
        }
        t.NULL_COLOR = { css: "#00000000", rgba: 0 }, (function(i) {
          i.toCss = function(u, p, l, m) {
            return m !== void 0 ? `#${o(u)}${o(p)}${o(l)}${o(m)}` : `#${o(u)}${o(p)}${o(l)}`;
          }, i.toRgba = function(u, p, l, m = 255) {
            return (u << 24 | p << 16 | l << 8 | m) >>> 0;
          }, i.toColor = function(u, p, l, m) {
            return { css: i.toCss(u, p, l, m), rgba: i.toRgba(u, p, l, m) };
          };
        })(d || (t.channels = d = {})), (function(i) {
          function u(p, l) {
            return r = Math.round(255 * l), [a, c, h] = e.toChannels(p.rgba), { css: d.toCss(a, c, h, r), rgba: d.toRgba(a, c, h, r) };
          }
          i.blend = function(p, l) {
            if (r = (255 & l.rgba) / 255, r === 1) return { css: l.css, rgba: l.rgba };
            const m = l.rgba >> 24 & 255, _ = l.rgba >> 16 & 255, v = l.rgba >> 8 & 255, C = p.rgba >> 24 & 255, w = p.rgba >> 16 & 255, S = p.rgba >> 8 & 255;
            return a = C + Math.round((m - C) * r), c = w + Math.round((_ - w) * r), h = S + Math.round((v - S) * r), { css: d.toCss(a, c, h), rgba: d.toRgba(a, c, h) };
          }, i.isOpaque = function(p) {
            return (255 & p.rgba) == 255;
          }, i.ensureContrastRatio = function(p, l, m) {
            const _ = e.ensureContrastRatio(p.rgba, l.rgba, m);
            if (_) return d.toColor(_ >> 24 & 255, _ >> 16 & 255, _ >> 8 & 255);
          }, i.opaque = function(p) {
            const l = (255 | p.rgba) >>> 0;
            return [a, c, h] = e.toChannels(l), { css: d.toCss(a, c, h), rgba: l };
          }, i.opacity = u, i.multiplyOpacity = function(p, l) {
            return r = 255 & p.rgba, u(p, r * l / 255);
          }, i.toColorRGB = function(p) {
            return [p.rgba >> 24 & 255, p.rgba >> 16 & 255, p.rgba >> 8 & 255];
          };
        })(f || (t.color = f = {})), (function(i) {
          let u, p;
          try {
            const l = document.createElement("canvas");
            l.width = 1, l.height = 1;
            const m = l.getContext("2d", { willReadFrequently: !0 });
            m && (u = m, u.globalCompositeOperation = "copy", p = u.createLinearGradient(0, 0, 1, 1));
          } catch (l) {
          }
          i.toColor = function(l) {
            if (l.match(/#[\da-f]{3,8}/i)) switch (l.length) {
              case 4:
                return a = parseInt(l.slice(1, 2).repeat(2), 16), c = parseInt(l.slice(2, 3).repeat(2), 16), h = parseInt(l.slice(3, 4).repeat(2), 16), d.toColor(a, c, h);
              case 5:
                return a = parseInt(l.slice(1, 2).repeat(2), 16), c = parseInt(l.slice(2, 3).repeat(2), 16), h = parseInt(l.slice(3, 4).repeat(2), 16), r = parseInt(l.slice(4, 5).repeat(2), 16), d.toColor(a, c, h, r);
              case 7:
                return { css: l, rgba: (parseInt(l.slice(1), 16) << 8 | 255) >>> 0 };
              case 9:
                return { css: l, rgba: parseInt(l.slice(1), 16) >>> 0 };
            }
            const m = l.match(/rgba?\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*(,\s*(0|1|\d?\.(\d+))\s*)?\)/);
            if (m) return a = parseInt(m[1]), c = parseInt(m[2]), h = parseInt(m[3]), r = Math.round(255 * (m[5] === void 0 ? 1 : parseFloat(m[5]))), d.toColor(a, c, h, r);
            if (!u || !p) throw new Error("css.toColor: Unsupported css format");
            if (u.fillStyle = p, u.fillStyle = l, typeof u.fillStyle != "string") throw new Error("css.toColor: Unsupported css format");
            if (u.fillRect(0, 0, 1, 1), [a, c, h, r] = u.getImageData(0, 0, 1, 1).data, r !== 255) throw new Error("css.toColor: Unsupported css format");
            return { rgba: d.toRgba(a, c, h, r), css: l };
          };
        })(g || (t.css = g = {})), (function(i) {
          function u(p, l, m) {
            const _ = p / 255, v = l / 255, C = m / 255;
            return 0.2126 * (_ <= 0.03928 ? _ / 12.92 : Math.pow((_ + 0.055) / 1.055, 2.4)) + 0.7152 * (v <= 0.03928 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)) + 0.0722 * (C <= 0.03928 ? C / 12.92 : Math.pow((C + 0.055) / 1.055, 2.4));
          }
          i.relativeLuminance = function(p) {
            return u(p >> 16 & 255, p >> 8 & 255, 255 & p);
          }, i.relativeLuminance2 = u;
        })(n || (t.rgb = n = {})), (function(i) {
          function u(l, m, _) {
            const v = l >> 24 & 255, C = l >> 16 & 255, w = l >> 8 & 255;
            let S = m >> 24 & 255, b = m >> 16 & 255, x = m >> 8 & 255, A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            for (; A < _ && (S > 0 || b > 0 || x > 0); ) S -= Math.max(0, Math.ceil(0.1 * S)), b -= Math.max(0, Math.ceil(0.1 * b)), x -= Math.max(0, Math.ceil(0.1 * x)), A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            return (S << 24 | b << 16 | x << 8 | 255) >>> 0;
          }
          function p(l, m, _) {
            const v = l >> 24 & 255, C = l >> 16 & 255, w = l >> 8 & 255;
            let S = m >> 24 & 255, b = m >> 16 & 255, x = m >> 8 & 255, A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            for (; A < _ && (S < 255 || b < 255 || x < 255); ) S = Math.min(255, S + Math.ceil(0.1 * (255 - S))), b = Math.min(255, b + Math.ceil(0.1 * (255 - b))), x = Math.min(255, x + Math.ceil(0.1 * (255 - x))), A = s(n.relativeLuminance2(S, b, x), n.relativeLuminance2(v, C, w));
            return (S << 24 | b << 16 | x << 8 | 255) >>> 0;
          }
          i.blend = function(l, m) {
            if (r = (255 & m) / 255, r === 1) return m;
            const _ = m >> 24 & 255, v = m >> 16 & 255, C = m >> 8 & 255, w = l >> 24 & 255, S = l >> 16 & 255, b = l >> 8 & 255;
            return a = w + Math.round((_ - w) * r), c = S + Math.round((v - S) * r), h = b + Math.round((C - b) * r), d.toRgba(a, c, h);
          }, i.ensureContrastRatio = function(l, m, _) {
            const v = n.relativeLuminance(l >> 8), C = n.relativeLuminance(m >> 8);
            if (s(v, C) < _) {
              if (C < v) {
                const b = u(l, m, _), x = s(v, n.relativeLuminance(b >> 8));
                if (x < _) {
                  const A = p(l, m, _);
                  return x > s(v, n.relativeLuminance(A >> 8)) ? b : A;
                }
                return b;
              }
              const w = p(l, m, _), S = s(v, n.relativeLuminance(w >> 8));
              if (S < _) {
                const b = u(l, m, _);
                return S > s(v, n.relativeLuminance(b >> 8)) ? w : b;
              }
              return w;
            }
          }, i.reduceLuminance = u, i.increaseLuminance = p, i.toChannels = function(l) {
            return [l >> 24 & 255, l >> 16 & 255, l >> 8 & 255, 255 & l];
          };
        })(e || (t.rgba = e = {})), t.toPaddedHex = o, t.contrastRatio = s;
      }, 345: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.runAndSubscribe = t.forwardEvent = t.EventEmitter = void 0, t.EventEmitter = class {
          constructor() {
            this._listeners = [], this._disposed = !1;
          }
          get event() {
            return this._event || (this._event = (a) => (this._listeners.push(a), { dispose: () => {
              if (!this._disposed) {
                for (let c = 0; c < this._listeners.length; c++) if (this._listeners[c] === a) return void this._listeners.splice(c, 1);
              }
            } })), this._event;
          }
          fire(a, c) {
            const h = [];
            for (let r = 0; r < this._listeners.length; r++) h.push(this._listeners[r]);
            for (let r = 0; r < h.length; r++) h[r].call(void 0, a, c);
          }
          dispose() {
            this.clearListeners(), this._disposed = !0;
          }
          clearListeners() {
            this._listeners && (this._listeners.length = 0);
          }
        }, t.forwardEvent = function(a, c) {
          return a(((h) => c.fire(h)));
        }, t.runAndSubscribe = function(a, c) {
          return c(void 0), a(((h) => c(h)));
        };
      }, 859: (T, t) => {
        function a(c) {
          for (const h of c) h.dispose();
          c.length = 0;
        }
        Object.defineProperty(t, "__esModule", { value: !0 }), t.getDisposeArrayDisposable = t.disposeArray = t.toDisposable = t.MutableDisposable = t.Disposable = void 0, t.Disposable = class {
          constructor() {
            this._disposables = [], this._isDisposed = !1;
          }
          dispose() {
            this._isDisposed = !0;
            for (const c of this._disposables) c.dispose();
            this._disposables.length = 0;
          }
          register(c) {
            return this._disposables.push(c), c;
          }
          unregister(c) {
            const h = this._disposables.indexOf(c);
            h !== -1 && this._disposables.splice(h, 1);
          }
        }, t.MutableDisposable = class {
          constructor() {
            this._isDisposed = !1;
          }
          get value() {
            return this._isDisposed ? void 0 : this._value;
          }
          set value(c) {
            var h;
            this._isDisposed || c === this._value || ((h = this._value) == null || h.dispose(), this._value = c);
          }
          clear() {
            this.value = void 0;
          }
          dispose() {
            var c;
            this._isDisposed = !0, (c = this._value) == null || c.dispose(), this._value = void 0;
          }
        }, t.toDisposable = function(c) {
          return { dispose: c };
        }, t.disposeArray = a, t.getDisposeArrayDisposable = function(c) {
          return { dispose: () => a(c) };
        };
      }, 485: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.FourKeyMap = t.TwoKeyMap = void 0;
        class a {
          constructor() {
            this._data = {};
          }
          set(h, r, d) {
            this._data[h] || (this._data[h] = {}), this._data[h][r] = d;
          }
          get(h, r) {
            return this._data[h] ? this._data[h][r] : void 0;
          }
          clear() {
            this._data = {};
          }
        }
        t.TwoKeyMap = a, t.FourKeyMap = class {
          constructor() {
            this._data = new a();
          }
          set(c, h, r, d, f) {
            this._data.get(c, h) || this._data.set(c, h, new a()), this._data.get(c, h).set(r, d, f);
          }
          get(c, h, r, d) {
            var f;
            return (f = this._data.get(c, h)) == null ? void 0 : f.get(r, d);
          }
          clear() {
            this._data.clear();
          }
        };
      }, 399: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.isChromeOS = t.isLinux = t.isWindows = t.isIphone = t.isIpad = t.isMac = t.getSafariVersion = t.isSafari = t.isLegacyEdge = t.isFirefox = t.isNode = void 0, t.isNode = typeof process != "undefined" && "title" in process;
        const a = t.isNode ? "node" : navigator.userAgent, c = t.isNode ? "node" : navigator.platform;
        t.isFirefox = a.includes("Firefox"), t.isLegacyEdge = a.includes("Edge"), t.isSafari = /^((?!chrome|android).)*safari/i.test(a), t.getSafariVersion = function() {
          if (!t.isSafari) return 0;
          const h = a.match(/Version\/(\d+)/);
          return h === null || h.length < 2 ? 0 : parseInt(h[1]);
        }, t.isMac = ["Macintosh", "MacIntel", "MacPPC", "Mac68K"].includes(c), t.isIpad = c === "iPad", t.isIphone = c === "iPhone", t.isWindows = ["Windows", "Win16", "Win32", "WinCE"].includes(c), t.isLinux = c.indexOf("Linux") >= 0, t.isChromeOS = /\bCrOS\b/.test(a);
      }, 385: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.DebouncedIdleTask = t.IdleTaskQueue = t.PriorityTaskQueue = void 0;
        const c = a(399);
        class h {
          constructor() {
            this._tasks = [], this._i = 0;
          }
          enqueue(f) {
            this._tasks.push(f), this._start();
          }
          flush() {
            for (; this._i < this._tasks.length; ) this._tasks[this._i]() || this._i++;
            this.clear();
          }
          clear() {
            this._idleCallback && (this._cancelCallback(this._idleCallback), this._idleCallback = void 0), this._i = 0, this._tasks.length = 0;
          }
          _start() {
            this._idleCallback || (this._idleCallback = this._requestCallback(this._process.bind(this)));
          }
          _process(f) {
            this._idleCallback = void 0;
            let g = 0, n = 0, e = f.timeRemaining(), o = 0;
            for (; this._i < this._tasks.length; ) {
              if (g = Date.now(), this._tasks[this._i]() || this._i++, g = Math.max(1, Date.now() - g), n = Math.max(g, n), o = f.timeRemaining(), 1.5 * n > o) return e - g < -20 && console.warn(`task queue exceeded allotted deadline by ${Math.abs(Math.round(e - g))}ms`), void this._start();
              e = o;
            }
            this.clear();
          }
        }
        class r extends h {
          _requestCallback(f) {
            return setTimeout((() => f(this._createDeadline(16))));
          }
          _cancelCallback(f) {
            clearTimeout(f);
          }
          _createDeadline(f) {
            const g = Date.now() + f;
            return { timeRemaining: () => Math.max(0, g - Date.now()) };
          }
        }
        t.PriorityTaskQueue = r, t.IdleTaskQueue = !c.isNode && "requestIdleCallback" in window ? class extends h {
          _requestCallback(d) {
            return requestIdleCallback(d);
          }
          _cancelCallback(d) {
            cancelIdleCallback(d);
          }
        } : r, t.DebouncedIdleTask = class {
          constructor() {
            this._queue = new t.IdleTaskQueue();
          }
          set(d) {
            this._queue.clear(), this._queue.enqueue(d);
          }
          flush() {
            this._queue.flush();
          }
        };
      }, 147: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.ExtendedAttrs = t.AttributeData = void 0;
        class a {
          constructor() {
            this.fg = 0, this.bg = 0, this.extended = new c();
          }
          static toColorRGB(r) {
            return [r >>> 16 & 255, r >>> 8 & 255, 255 & r];
          }
          static fromColorRGB(r) {
            return (255 & r[0]) << 16 | (255 & r[1]) << 8 | 255 & r[2];
          }
          clone() {
            const r = new a();
            return r.fg = this.fg, r.bg = this.bg, r.extended = this.extended.clone(), r;
          }
          isInverse() {
            return 67108864 & this.fg;
          }
          isBold() {
            return 134217728 & this.fg;
          }
          isUnderline() {
            return this.hasExtendedAttrs() && this.extended.underlineStyle !== 0 ? 1 : 268435456 & this.fg;
          }
          isBlink() {
            return 536870912 & this.fg;
          }
          isInvisible() {
            return 1073741824 & this.fg;
          }
          isItalic() {
            return 67108864 & this.bg;
          }
          isDim() {
            return 134217728 & this.bg;
          }
          isStrikethrough() {
            return 2147483648 & this.fg;
          }
          isProtected() {
            return 536870912 & this.bg;
          }
          isOverline() {
            return 1073741824 & this.bg;
          }
          getFgColorMode() {
            return 50331648 & this.fg;
          }
          getBgColorMode() {
            return 50331648 & this.bg;
          }
          isFgRGB() {
            return (50331648 & this.fg) == 50331648;
          }
          isBgRGB() {
            return (50331648 & this.bg) == 50331648;
          }
          isFgPalette() {
            return (50331648 & this.fg) == 16777216 || (50331648 & this.fg) == 33554432;
          }
          isBgPalette() {
            return (50331648 & this.bg) == 16777216 || (50331648 & this.bg) == 33554432;
          }
          isFgDefault() {
            return (50331648 & this.fg) == 0;
          }
          isBgDefault() {
            return (50331648 & this.bg) == 0;
          }
          isAttributeDefault() {
            return this.fg === 0 && this.bg === 0;
          }
          getFgColor() {
            switch (50331648 & this.fg) {
              case 16777216:
              case 33554432:
                return 255 & this.fg;
              case 50331648:
                return 16777215 & this.fg;
              default:
                return -1;
            }
          }
          getBgColor() {
            switch (50331648 & this.bg) {
              case 16777216:
              case 33554432:
                return 255 & this.bg;
              case 50331648:
                return 16777215 & this.bg;
              default:
                return -1;
            }
          }
          hasExtendedAttrs() {
            return 268435456 & this.bg;
          }
          updateExtended() {
            this.extended.isEmpty() ? this.bg &= -268435457 : this.bg |= 268435456;
          }
          getUnderlineColor() {
            if (268435456 & this.bg && ~this.extended.underlineColor) switch (50331648 & this.extended.underlineColor) {
              case 16777216:
              case 33554432:
                return 255 & this.extended.underlineColor;
              case 50331648:
                return 16777215 & this.extended.underlineColor;
              default:
                return this.getFgColor();
            }
            return this.getFgColor();
          }
          getUnderlineColorMode() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? 50331648 & this.extended.underlineColor : this.getFgColorMode();
          }
          isUnderlineColorRGB() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? (50331648 & this.extended.underlineColor) == 50331648 : this.isFgRGB();
          }
          isUnderlineColorPalette() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? (50331648 & this.extended.underlineColor) == 16777216 || (50331648 & this.extended.underlineColor) == 33554432 : this.isFgPalette();
          }
          isUnderlineColorDefault() {
            return 268435456 & this.bg && ~this.extended.underlineColor ? (50331648 & this.extended.underlineColor) == 0 : this.isFgDefault();
          }
          getUnderlineStyle() {
            return 268435456 & this.fg ? 268435456 & this.bg ? this.extended.underlineStyle : 1 : 0;
          }
          getUnderlineVariantOffset() {
            return this.extended.underlineVariantOffset;
          }
        }
        t.AttributeData = a;
        class c {
          get ext() {
            return this._urlId ? -469762049 & this._ext | this.underlineStyle << 26 : this._ext;
          }
          set ext(r) {
            this._ext = r;
          }
          get underlineStyle() {
            return this._urlId ? 5 : (469762048 & this._ext) >> 26;
          }
          set underlineStyle(r) {
            this._ext &= -469762049, this._ext |= r << 26 & 469762048;
          }
          get underlineColor() {
            return 67108863 & this._ext;
          }
          set underlineColor(r) {
            this._ext &= -67108864, this._ext |= 67108863 & r;
          }
          get urlId() {
            return this._urlId;
          }
          set urlId(r) {
            this._urlId = r;
          }
          get underlineVariantOffset() {
            const r = (3758096384 & this._ext) >> 29;
            return r < 0 ? 4294967288 ^ r : r;
          }
          set underlineVariantOffset(r) {
            this._ext &= 536870911, this._ext |= r << 29 & 3758096384;
          }
          constructor(r = 0, d = 0) {
            this._ext = 0, this._urlId = 0, this._ext = r, this._urlId = d;
          }
          clone() {
            return new c(this._ext, this._urlId);
          }
          isEmpty() {
            return this.underlineStyle === 0 && this._urlId === 0;
          }
        }
        t.ExtendedAttrs = c;
      }, 782: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.CellData = void 0;
        const c = a(133), h = a(855), r = a(147);
        class d extends r.AttributeData {
          constructor() {
            super(...arguments), this.content = 0, this.fg = 0, this.bg = 0, this.extended = new r.ExtendedAttrs(), this.combinedData = "";
          }
          static fromCharData(g) {
            const n = new d();
            return n.setFromCharData(g), n;
          }
          isCombined() {
            return 2097152 & this.content;
          }
          getWidth() {
            return this.content >> 22;
          }
          getChars() {
            return 2097152 & this.content ? this.combinedData : 2097151 & this.content ? (0, c.stringFromCodePoint)(2097151 & this.content) : "";
          }
          getCode() {
            return this.isCombined() ? this.combinedData.charCodeAt(this.combinedData.length - 1) : 2097151 & this.content;
          }
          setFromCharData(g) {
            this.fg = g[h.CHAR_DATA_ATTR_INDEX], this.bg = 0;
            let n = !1;
            if (g[h.CHAR_DATA_CHAR_INDEX].length > 2) n = !0;
            else if (g[h.CHAR_DATA_CHAR_INDEX].length === 2) {
              const e = g[h.CHAR_DATA_CHAR_INDEX].charCodeAt(0);
              if (55296 <= e && e <= 56319) {
                const o = g[h.CHAR_DATA_CHAR_INDEX].charCodeAt(1);
                56320 <= o && o <= 57343 ? this.content = 1024 * (e - 55296) + o - 56320 + 65536 | g[h.CHAR_DATA_WIDTH_INDEX] << 22 : n = !0;
              } else n = !0;
            } else this.content = g[h.CHAR_DATA_CHAR_INDEX].charCodeAt(0) | g[h.CHAR_DATA_WIDTH_INDEX] << 22;
            n && (this.combinedData = g[h.CHAR_DATA_CHAR_INDEX], this.content = 2097152 | g[h.CHAR_DATA_WIDTH_INDEX] << 22);
          }
          getAsCharData() {
            return [this.fg, this.getChars(), this.getWidth(), this.getCode()];
          }
        }
        t.CellData = d;
      }, 855: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.WHITESPACE_CELL_CODE = t.WHITESPACE_CELL_WIDTH = t.WHITESPACE_CELL_CHAR = t.NULL_CELL_CODE = t.NULL_CELL_WIDTH = t.NULL_CELL_CHAR = t.CHAR_DATA_CODE_INDEX = t.CHAR_DATA_WIDTH_INDEX = t.CHAR_DATA_CHAR_INDEX = t.CHAR_DATA_ATTR_INDEX = t.DEFAULT_EXT = t.DEFAULT_ATTR = t.DEFAULT_COLOR = void 0, t.DEFAULT_COLOR = 0, t.DEFAULT_ATTR = 256 | t.DEFAULT_COLOR << 9, t.DEFAULT_EXT = 0, t.CHAR_DATA_ATTR_INDEX = 0, t.CHAR_DATA_CHAR_INDEX = 1, t.CHAR_DATA_WIDTH_INDEX = 2, t.CHAR_DATA_CODE_INDEX = 3, t.NULL_CELL_CHAR = "", t.NULL_CELL_WIDTH = 1, t.NULL_CELL_CODE = 0, t.WHITESPACE_CELL_CHAR = " ", t.WHITESPACE_CELL_WIDTH = 1, t.WHITESPACE_CELL_CODE = 32;
      }, 133: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.Utf8ToUtf32 = t.StringToUtf32 = t.utf32ToString = t.stringFromCodePoint = void 0, t.stringFromCodePoint = function(a) {
          return a > 65535 ? (a -= 65536, String.fromCharCode(55296 + (a >> 10)) + String.fromCharCode(a % 1024 + 56320)) : String.fromCharCode(a);
        }, t.utf32ToString = function(a, c = 0, h = a.length) {
          let r = "";
          for (let d = c; d < h; ++d) {
            let f = a[d];
            f > 65535 ? (f -= 65536, r += String.fromCharCode(55296 + (f >> 10)) + String.fromCharCode(f % 1024 + 56320)) : r += String.fromCharCode(f);
          }
          return r;
        }, t.StringToUtf32 = class {
          constructor() {
            this._interim = 0;
          }
          clear() {
            this._interim = 0;
          }
          decode(a, c) {
            const h = a.length;
            if (!h) return 0;
            let r = 0, d = 0;
            if (this._interim) {
              const f = a.charCodeAt(d++);
              56320 <= f && f <= 57343 ? c[r++] = 1024 * (this._interim - 55296) + f - 56320 + 65536 : (c[r++] = this._interim, c[r++] = f), this._interim = 0;
            }
            for (let f = d; f < h; ++f) {
              const g = a.charCodeAt(f);
              if (55296 <= g && g <= 56319) {
                if (++f >= h) return this._interim = g, r;
                const n = a.charCodeAt(f);
                56320 <= n && n <= 57343 ? c[r++] = 1024 * (g - 55296) + n - 56320 + 65536 : (c[r++] = g, c[r++] = n);
              } else g !== 65279 && (c[r++] = g);
            }
            return r;
          }
        }, t.Utf8ToUtf32 = class {
          constructor() {
            this.interim = new Uint8Array(3);
          }
          clear() {
            this.interim.fill(0);
          }
          decode(a, c) {
            const h = a.length;
            if (!h) return 0;
            let r, d, f, g, n = 0, e = 0, o = 0;
            if (this.interim[0]) {
              let u = !1, p = this.interim[0];
              p &= (224 & p) == 192 ? 31 : (240 & p) == 224 ? 15 : 7;
              let l, m = 0;
              for (; (l = 63 & this.interim[++m]) && m < 4; ) p <<= 6, p |= l;
              const _ = (224 & this.interim[0]) == 192 ? 2 : (240 & this.interim[0]) == 224 ? 3 : 4, v = _ - m;
              for (; o < v; ) {
                if (o >= h) return 0;
                if (l = a[o++], (192 & l) != 128) {
                  o--, u = !0;
                  break;
                }
                this.interim[m++] = l, p <<= 6, p |= 63 & l;
              }
              u || (_ === 2 ? p < 128 ? o-- : c[n++] = p : _ === 3 ? p < 2048 || p >= 55296 && p <= 57343 || p === 65279 || (c[n++] = p) : p < 65536 || p > 1114111 || (c[n++] = p)), this.interim.fill(0);
            }
            const s = h - 4;
            let i = o;
            for (; i < h; ) {
              for (; !(!(i < s) || 128 & (r = a[i]) || 128 & (d = a[i + 1]) || 128 & (f = a[i + 2]) || 128 & (g = a[i + 3])); ) c[n++] = r, c[n++] = d, c[n++] = f, c[n++] = g, i += 4;
              if (r = a[i++], r < 128) c[n++] = r;
              else if ((224 & r) == 192) {
                if (i >= h) return this.interim[0] = r, n;
                if (d = a[i++], (192 & d) != 128) {
                  i--;
                  continue;
                }
                if (e = (31 & r) << 6 | 63 & d, e < 128) {
                  i--;
                  continue;
                }
                c[n++] = e;
              } else if ((240 & r) == 224) {
                if (i >= h) return this.interim[0] = r, n;
                if (d = a[i++], (192 & d) != 128) {
                  i--;
                  continue;
                }
                if (i >= h) return this.interim[0] = r, this.interim[1] = d, n;
                if (f = a[i++], (192 & f) != 128) {
                  i--;
                  continue;
                }
                if (e = (15 & r) << 12 | (63 & d) << 6 | 63 & f, e < 2048 || e >= 55296 && e <= 57343 || e === 65279) continue;
                c[n++] = e;
              } else if ((248 & r) == 240) {
                if (i >= h) return this.interim[0] = r, n;
                if (d = a[i++], (192 & d) != 128) {
                  i--;
                  continue;
                }
                if (i >= h) return this.interim[0] = r, this.interim[1] = d, n;
                if (f = a[i++], (192 & f) != 128) {
                  i--;
                  continue;
                }
                if (i >= h) return this.interim[0] = r, this.interim[1] = d, this.interim[2] = f, n;
                if (g = a[i++], (192 & g) != 128) {
                  i--;
                  continue;
                }
                if (e = (7 & r) << 18 | (63 & d) << 12 | (63 & f) << 6 | 63 & g, e < 65536 || e > 1114111) continue;
                c[n++] = e;
              }
            }
            return n;
          }
        };
      }, 776: function(T, t, a) {
        var c = this && this.__decorate || function(e, o, s, i) {
          var u, p = arguments.length, l = p < 3 ? o : i === null ? i = Object.getOwnPropertyDescriptor(o, s) : i;
          if (typeof Reflect == "object" && typeof Reflect.decorate == "function") l = Reflect.decorate(e, o, s, i);
          else for (var m = e.length - 1; m >= 0; m--) (u = e[m]) && (l = (p < 3 ? u(l) : p > 3 ? u(o, s, l) : u(o, s)) || l);
          return p > 3 && l && Object.defineProperty(o, s, l), l;
        }, h = this && this.__param || function(e, o) {
          return function(s, i) {
            o(s, i, e);
          };
        };
        Object.defineProperty(t, "__esModule", { value: !0 }), t.traceCall = t.setTraceLogger = t.LogService = void 0;
        const r = a(859), d = a(97), f = { trace: d.LogLevelEnum.TRACE, debug: d.LogLevelEnum.DEBUG, info: d.LogLevelEnum.INFO, warn: d.LogLevelEnum.WARN, error: d.LogLevelEnum.ERROR, off: d.LogLevelEnum.OFF };
        let g, n = t.LogService = class extends r.Disposable {
          get logLevel() {
            return this._logLevel;
          }
          constructor(e) {
            super(), this._optionsService = e, this._logLevel = d.LogLevelEnum.OFF, this._updateLogLevel(), this.register(this._optionsService.onSpecificOptionChange("logLevel", (() => this._updateLogLevel()))), g = this;
          }
          _updateLogLevel() {
            this._logLevel = f[this._optionsService.rawOptions.logLevel];
          }
          _evalLazyOptionalParams(e) {
            for (let o = 0; o < e.length; o++) typeof e[o] == "function" && (e[o] = e[o]());
          }
          _log(e, o, s) {
            this._evalLazyOptionalParams(s), e.call(console, (this._optionsService.options.logger ? "" : "xterm.js: ") + o, ...s);
          }
          trace(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.TRACE && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.trace.bind(this._optionsService.options.logger)) != null ? i : console.log, e, o);
          }
          debug(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.DEBUG && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.debug.bind(this._optionsService.options.logger)) != null ? i : console.log, e, o);
          }
          info(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.INFO && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.info.bind(this._optionsService.options.logger)) != null ? i : console.info, e, o);
          }
          warn(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.WARN && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.warn.bind(this._optionsService.options.logger)) != null ? i : console.warn, e, o);
          }
          error(e, ...o) {
            var s, i;
            this._logLevel <= d.LogLevelEnum.ERROR && this._log((i = (s = this._optionsService.options.logger) == null ? void 0 : s.error.bind(this._optionsService.options.logger)) != null ? i : console.error, e, o);
          }
        };
        t.LogService = n = c([h(0, d.IOptionsService)], n), t.setTraceLogger = function(e) {
          g = e;
        }, t.traceCall = function(e, o, s) {
          if (typeof s.value != "function") throw new Error("not supported");
          const i = s.value;
          s.value = function(...u) {
            if (g.logLevel !== d.LogLevelEnum.TRACE) return i.apply(this, u);
            g.trace(`GlyphRenderer#${i.name}(${u.map(((l) => JSON.stringify(l))).join(", ")})`);
            const p = i.apply(this, u);
            return g.trace(`GlyphRenderer#${i.name} return`, p), p;
          };
        };
      }, 726: (T, t) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.createDecorator = t.getServiceDependencies = t.serviceRegistry = void 0;
        const a = "di$target", c = "di$dependencies";
        t.serviceRegistry = /* @__PURE__ */ new Map(), t.getServiceDependencies = function(h) {
          return h[c] || [];
        }, t.createDecorator = function(h) {
          if (t.serviceRegistry.has(h)) return t.serviceRegistry.get(h);
          const r = function(d, f, g) {
            if (arguments.length !== 3) throw new Error("@IServiceName-decorator can only be used to decorate a parameter");
            (function(n, e, o) {
              e[a] === e ? e[c].push({ id: n, index: o }) : (e[c] = [{ id: n, index: o }], e[a] = e);
            })(r, d, g);
          };
          return r.toString = () => h, t.serviceRegistry.set(h, r), r;
        };
      }, 97: (T, t, a) => {
        Object.defineProperty(t, "__esModule", { value: !0 }), t.IDecorationService = t.IUnicodeService = t.IOscLinkService = t.IOptionsService = t.ILogService = t.LogLevelEnum = t.IInstantiationService = t.ICharsetService = t.ICoreService = t.ICoreMouseService = t.IBufferService = void 0;
        const c = a(726);
        var h;
        t.IBufferService = (0, c.createDecorator)("BufferService"), t.ICoreMouseService = (0, c.createDecorator)("CoreMouseService"), t.ICoreService = (0, c.createDecorator)("CoreService"), t.ICharsetService = (0, c.createDecorator)("CharsetService"), t.IInstantiationService = (0, c.createDecorator)("InstantiationService"), (function(r) {
          r[r.TRACE = 0] = "TRACE", r[r.DEBUG = 1] = "DEBUG", r[r.INFO = 2] = "INFO", r[r.WARN = 3] = "WARN", r[r.ERROR = 4] = "ERROR", r[r.OFF = 5] = "OFF";
        })(h || (t.LogLevelEnum = h = {})), t.ILogService = (0, c.createDecorator)("LogService"), t.IOptionsService = (0, c.createDecorator)("OptionsService"), t.IOscLinkService = (0, c.createDecorator)("OscLinkService"), t.IUnicodeService = (0, c.createDecorator)("UnicodeService"), t.IDecorationService = (0, c.createDecorator)("DecorationService");
      } }, $ = {};
      function W(T) {
        var t = $[T];
        if (t !== void 0) return t.exports;
        var a = $[T] = { exports: {} };
        return I[T].call(a.exports, a, a.exports, W), a.exports;
      }
      var Y = {};
      return (() => {
        var T = Y;
        Object.defineProperty(T, "__esModule", { value: !0 }), T.CanvasAddon = void 0;
        const t = W(345), a = W(859), c = W(776), h = W(949);
        class r extends a.Disposable {
          constructor() {
            super(...arguments), this._onChangeTextureAtlas = this.register(new t.EventEmitter()), this.onChangeTextureAtlas = this._onChangeTextureAtlas.event, this._onAddTextureAtlasCanvas = this.register(new t.EventEmitter()), this.onAddTextureAtlasCanvas = this._onAddTextureAtlasCanvas.event;
          }
          get textureAtlas() {
            var f;
            return (f = this._renderer) == null ? void 0 : f.textureAtlas;
          }
          activate(f) {
            const g = f._core;
            if (!f.element) return void this.register(g.onWillOpen((() => this.activate(f))));
            this._terminal = f;
            const n = g.coreService, e = g.optionsService, o = g.screenElement, s = g.linkifier, i = g, u = i._bufferService, p = i._renderService, l = i._characterJoinerService, m = i._charSizeService, _ = i._coreBrowserService, v = i._decorationService, C = i._logService, w = i._themeService;
            (0, c.setTraceLogger)(C), this._renderer = new h.CanvasRenderer(f, o, s, u, m, e, l, n, _, v, w), this.register((0, t.forwardEvent)(this._renderer.onChangeTextureAtlas, this._onChangeTextureAtlas)), this.register((0, t.forwardEvent)(this._renderer.onAddTextureAtlasCanvas, this._onAddTextureAtlasCanvas)), p.setRenderer(this._renderer), p.handleResize(u.cols, u.rows), this.register((0, a.toDisposable)((() => {
              var S;
              p.setRenderer(this._terminal._core._createRenderer()), p.handleResize(f.cols, f.rows), (S = this._renderer) == null || S.dispose(), this._renderer = void 0;
            })));
          }
          clearTextureAtlas() {
            var f;
            (f = this._renderer) == null || f.clearTextureAtlas();
          }
        }
        T.CanvasAddon = r;
      })(), Y;
    })()));
  })(Be)), Be.exports;
}
var ht = at();
const Pe = {
  background: "#0d1117",
  foreground: "#c9d1d9",
  cursor: "#58a6ff",
  cursorAccent: "#0d1117",
  selectionBackground: "rgba(56, 139, 253, 0.4)",
  selectionForeground: "#ffffff",
  selectionInactiveBackground: "rgba(56, 139, 253, 0.2)",
  black: "#484f58",
  red: "#ff7b72",
  green: "#3fb950",
  yellow: "#d29922",
  blue: "#58a6ff",
  magenta: "#bc8cff",
  cyan: "#39c5cf",
  white: "#b1bac4",
  brightBlack: "#6e7681",
  brightRed: "#ffa198",
  brightGreen: "#56d364",
  brightYellow: "#e3b341",
  brightBlue: "#79c0ff",
  brightMagenta: "#d2a8ff",
  brightCyan: "#56d4dd",
  brightWhite: "#f0f6fc"
}, Ie = {
  background: "#ffffff",
  foreground: "#24292f",
  cursor: "#0969da",
  cursorAccent: "#ffffff",
  selectionBackground: "rgba(9, 105, 218, 0.3)",
  selectionForeground: "#24292f",
  selectionInactiveBackground: "rgba(9, 105, 218, 0.15)",
  black: "#24292f",
  red: "#cf222e",
  green: "#116329",
  yellow: "#4d2d00",
  blue: "#0969da",
  magenta: "#8250df",
  cyan: "#1b7c83",
  white: "#6e7781",
  brightBlack: "#57606a",
  brightRed: "#a40e26",
  brightGreen: "#1a7f37",
  brightYellow: "#633c01",
  brightBlue: "#218bff",
  brightMagenta: "#a475f9",
  brightCyan: "#3192aa",
  brightWhite: "#8c959f"
};
function Oe(ne) {
  return typeof ne == "string" ? ne === "light" ? Ie : Pe : ne;
}
const lt = "rexec.v1", ct = "rexec.token.";
class Fe {
  constructor(B = "https://rexec.dev", I) {
    oe(this, "baseUrl");
    oe(this, "token");
    this.baseUrl = B.replace(/\/$/, ""), this.token = I || null;
  }
  /**
   * Set the authentication token
   */
  setToken(B) {
    this.token = B;
  }
  /**
   * Get default headers for API requests
   */
  getHeaders() {
    const B = new Headers({
      "Content-Type": "application/json"
    });
    return this.token && B.set("Authorization", `Bearer ${this.token}`), B;
  }
  /**
   * Make an API request
   */
  async request(B, I = {}) {
    const $ = `${this.baseUrl}${B}`;
    try {
      const W = await fetch($, {
        ...I,
        headers: this.getHeaders()
      }), Y = W.headers.get("content-type");
      let T, t;
      if (Y != null && Y.includes("application/json")) {
        const a = await W.json();
        W.ok ? T = a : t = a.error || a.message || `Request failed: ${W.status}`;
      } else W.ok || (t = `Request failed: ${W.status}`);
      return { data: T, error: t };
    } catch (W) {
      return {
        error: W instanceof Error ? W.message : "Network error"
      };
    }
  }
  /**
   * Create a new container with the specified image and optional role
   */
  async createContainer(B = "ubuntu", I) {
    const $ = { image: B };
    return I && ($.role = I), console.log("[Rexec SDK] createContainer called with:", { image: B, role: I }), console.log("[Rexec SDK] Request body:", JSON.stringify($)), this.request("/api/containers", {
      method: "POST",
      body: JSON.stringify($)
    });
  }
  /**
   * Get container information
   */
  async getContainer(B) {
    return this.request(
      `/api/containers/${encodeURIComponent(B)}`
    );
  }
  /**
   * Wait for a container to be ready (running status)
   * Polls the container status until it's running or an error occurs
   */
  async waitForContainer(B, I = {}) {
    var Y, T, t, a, c;
    const $ = (Y = I.maxAttempts) != null ? Y : 60, W = (T = I.intervalMs) != null ? T : 2e3;
    for (let h = 1; h <= $; h++) {
      const { data: r, error: d } = await this.getContainer(B);
      if (d) {
        if (d.includes("404") || d.includes("not found")) {
          (t = I.onProgress) == null || t.call(I, "creating", h), await this.sleep(W);
          continue;
        }
        return { error: d };
      }
      if (r) {
        const f = ((a = r.status) == null ? void 0 : a.toLowerCase()) || "";
        if ((c = I.onProgress) == null || c.call(I, f, h), f === "running")
          return { data: r };
        if (f === "error" || f === "failed")
          return { error: `Container failed to start: ${f}` };
        if (f === "creating" || f === "configuring" || f === "starting" || f === "pulling") {
          await this.sleep(W);
          continue;
        }
        if (h < $) {
          await this.sleep(W);
          continue;
        }
      }
      await this.sleep(W);
    }
    return { error: "Timeout waiting for container to be ready" };
  }
  /**
   * Sleep for a given number of milliseconds
   */
  sleep(B) {
    return new Promise((I) => setTimeout(I, B));
  }
  /**
   * Join a collaborative session via share code
   */
  async joinSession(B) {
    return this.request(
      `/api/collab/join/${encodeURIComponent(B)}`
    );
  }
  /**
   * Start a new collaborative session for a container
   */
  async startCollabSession(B, I = "view") {
    return this.request("/api/collab/start", {
      method: "POST",
      body: JSON.stringify({ container_id: B, mode: I })
    });
  }
  /**
   * Get WebSocket URL for terminal connection
   */
  getTerminalWsUrl(B, I) {
    const $ = this.baseUrl.startsWith("https") ? "wss:" : "ws:", W = this.baseUrl.replace(/^https?:\/\//, "");
    return `${$}//${W}/ws/terminal/${encodeURIComponent(B)}?id=${encodeURIComponent(I)}`;
  }
  /**
   * Get WebSocket URL for agent terminal connection
   */
  getAgentTerminalWsUrl(B, I) {
    const $ = this.baseUrl.startsWith("https") ? "wss:" : "ws:", W = this.baseUrl.replace(/^https?:\/\//, "");
    return `${$}//${W}/ws/agent/${encodeURIComponent(B)}/terminal?id=${encodeURIComponent(I)}`;
  }
  /**
   * Get WebSocket URL for collab session
   */
  getCollabWsUrl(B) {
    const I = this.baseUrl.startsWith("https") ? "wss:" : "ws:", $ = this.baseUrl.replace(/^https?:\/\//, "");
    return `${I}//${$}/ws/collab/${encodeURIComponent(B)}`;
  }
}
function dt(ne) {
  if (ne)
    return [lt, `${ct}${ne}`];
}
function ut(ne, B) {
  const I = dt(B);
  return I ? new WebSocket(ne, I) : new WebSocket(ne);
}
class _t {
  constructor(B, I, $ = {}) {
    oe(this, "ws", null);
    oe(this, "url");
    oe(this, "token");
    oe(this, "reconnectAttempts", 0);
    oe(this, "maxReconnectAttempts");
    oe(this, "reconnectTimer", null);
    oe(this, "pingInterval", null);
    oe(this, "autoReconnect");
    // Callbacks
    oe(this, "onOpen", null);
    oe(this, "onClose", null);
    oe(this, "onError", null);
    oe(this, "onMessage", null);
    oe(this, "onReconnecting", null);
    var W, Y;
    this.url = B, this.token = I, this.autoReconnect = (W = $.autoReconnect) != null ? W : !0, this.maxReconnectAttempts = (Y = $.maxReconnectAttempts) != null ? Y : 10;
  }
  /**
   * Connect to the WebSocket
   */
  connect() {
    this.ws && (this.ws.readyState === WebSocket.OPEN || this.ws.readyState === WebSocket.CONNECTING) || (this.clearTimers(), this.ws = ut(this.url, this.token), this.ws.onopen = () => {
      var B;
      this.reconnectAttempts = 0, this.startPingInterval(), (B = this.onOpen) == null || B.call(this);
    }, this.ws.onclose = (B) => {
      var W;
      this.clearTimers(), (W = this.onClose) == null || W.call(this, B.code, B.reason);
      const I = B.code === 1e3, $ = B.code === 4e3 || B.code === 4001;
      this.autoReconnect && !I && !$ && this.attemptReconnect();
    }, this.ws.onerror = (B) => {
      var I;
      (I = this.onError) == null || I.call(this, B);
    }, this.ws.onmessage = (B) => {
      var I, $, W, Y, T;
      console.log(
        "[Rexec WS] Raw message received:",
        (($ = (I = B.data) == null ? void 0 : I.substring) == null ? void 0 : $.call(I, 0, 200)) || B.data
      );
      try {
        const t = JSON.parse(B.data);
        console.log(
          "[Rexec WS] Parsed message type:",
          t.type,
          "data length:",
          ((W = t.data) == null ? void 0 : W.length) || 0
        ), (Y = this.onMessage) == null || Y.call(this, t);
      } catch (t) {
        console.log("[Rexec WS] Non-JSON message, treating as output"), (T = this.onMessage) == null || T.call(this, { type: "output", data: B.data });
      }
    });
  }
  /**
   * Send a message through the WebSocket
   */
  send(B) {
    this.ws && this.ws.readyState === WebSocket.OPEN && this.ws.send(JSON.stringify(B));
  }
  /**
   * Send raw data (for terminal input)
   */
  sendRaw(B) {
    this.send({ type: "input", data: B });
  }
  /**
   * Send resize message
   */
  sendResize(B, I) {
    this.send({ type: "resize", cols: B, rows: I });
  }
  /**
   * Send ping message
   */
  sendPing() {
    this.send({ type: "ping" });
  }
  /**
   * Close the WebSocket connection
   */
  close(B = 1e3, I = "User disconnected") {
    this.autoReconnect = !1, this.clearTimers(), this.ws && (this.ws.close(B, I), this.ws = null);
  }
  /**
   * Check if connected
   */
  isConnected() {
    return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
  }
  /**
   * Get current ready state
   */
  getReadyState() {
    var B, I;
    return (I = (B = this.ws) == null ? void 0 : B.readyState) != null ? I : WebSocket.CLOSED;
  }
  /**
   * Clear all timers
   */
  clearTimers() {
    this.reconnectTimer && (clearTimeout(this.reconnectTimer), this.reconnectTimer = null), this.pingInterval && (clearInterval(this.pingInterval), this.pingInterval = null);
  }
  /**
   * Start ping interval to keep connection alive
   */
  startPingInterval() {
    this.pingInterval = setInterval(() => {
      this.sendPing();
    }, 2e4);
  }
  /**
   * Attempt to reconnect
   */
  attemptReconnect() {
    var I;
    if (this.reconnectAttempts >= this.maxReconnectAttempts)
      return;
    this.reconnectAttempts++;
    const B = Math.min(100 * Math.pow(2, this.reconnectAttempts), 8e3);
    (I = this.onReconnecting) == null || I.call(this, this.reconnectAttempts), this.reconnectTimer = setTimeout(() => {
      this.connect();
    }, B);
  }
  /**
   * Reset reconnect attempts counter
   */
  resetReconnectAttempts() {
    this.reconnectAttempts = 0;
  }
  /**
   * Update the URL (useful for reconnecting to a different session)
   */
  updateUrl(B) {
    this.url = B;
  }
}
function He() {
  return `embed-${Date.now()}-${Math.random().toString(36).slice(2, 11)}`;
}
const ft = 14, gt = 'JetBrains Mono, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace', vt = 5e3;
class pt {
  constructor() {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    oe(this, "listeners", /* @__PURE__ */ new Map());
  }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  on(B, I) {
    return this.listeners.has(B) || this.listeners.set(B, /* @__PURE__ */ new Set()), this.listeners.get(B).add(I), () => this.off(B, I);
  }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  off(B, I) {
    var $;
    ($ = this.listeners.get(B)) == null || $.delete(I);
  }
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  emit(B, ...I) {
    var $;
    ($ = this.listeners.get(B)) == null || $.forEach((W) => {
      try {
        W(...I);
      } catch (Y) {
        console.error(`[Rexec] Error in event handler for ${B}:`, Y);
      }
    });
  }
  removeAllListeners() {
    this.listeners.clear();
  }
}
class Ke {
  constructor(B, I = {}) {
    // Configuration
    oe(this, "config");
    oe(this, "container");
    // xterm.js terminal
    oe(this, "terminal", null);
    oe(this, "fitAddon", null);
    oe(this, "webglAddon", null);
    oe(this, "canvasAddon", null);
    oe(this, "resizeObserver", null);
    oe(this, "intersectionObserver", null);
    oe(this, "isVisible", !1);
    oe(this, "pendingFit", !1);
    // Connection
    oe(this, "api");
    oe(this, "ws", null);
    oe(this, "sessionId");
    // State
    oe(this, "_state", "idle");
    oe(this, "_session", null);
    oe(this, "_stats", null);
    oe(this, "destroyed", !1);
    // Events
    oe(this, "events", new pt());
    // Output buffering for performance
    oe(this, "outputBuffer", "");
    oe(this, "flushTimeout", null);
    var $, W, Y, T, t, a, c, h, r, d, f, g, n, e, o, s, i, u, p, l, m, _, v, C, w, S, b, x;
    if (typeof B == "string") {
      const A = document.querySelector(B);
      if (!A)
        throw new Error(`[Rexec] Element not found: ${B}`);
      this.container = A;
    } else
      this.container = B;
    this.config = {
      token: ($ = I.token) != null ? $ : "",
      container: (W = I.container) != null ? W : "",
      shareCode: (Y = I.shareCode) != null ? Y : "",
      role: (T = I.role) != null ? T : "",
      image: (t = I.image) != null ? t : "ubuntu",
      baseUrl: (a = I.baseUrl) != null ? a : this.detectBaseUrl(),
      theme: (c = I.theme) != null ? c : "dark",
      fontSize: (h = I.fontSize) != null ? h : ft,
      fontFamily: (r = I.fontFamily) != null ? r : gt,
      cursorStyle: (d = I.cursorStyle) != null ? d : "block",
      cursorBlink: (f = I.cursorBlink) != null ? f : !0,
      scrollback: (g = I.scrollback) != null ? g : vt,
      webgl: (n = I.webgl) != null ? n : !1,
      showToolbar: (e = I.showToolbar) != null ? e : !0,
      showStatus: (o = I.showStatus) != null ? o : !0,
      allowCopy: (s = I.allowCopy) != null ? s : !0,
      allowPaste: (i = I.allowPaste) != null ? i : !0,
      onReady: (u = I.onReady) != null ? u : (() => {
      }),
      onStateChange: (p = I.onStateChange) != null ? p : (() => {
      }),
      onError: (l = I.onError) != null ? l : (() => {
      }),
      onData: (m = I.onData) != null ? m : (() => {
      }),
      onResize: (_ = I.onResize) != null ? _ : (() => {
      }),
      onDisconnect: (v = I.onDisconnect) != null ? v : (() => {
      }),
      autoReconnect: (C = I.autoReconnect) != null ? C : !0,
      maxReconnectAttempts: (w = I.maxReconnectAttempts) != null ? w : 10,
      initialCommand: (S = I.initialCommand) != null ? S : "",
      className: (b = I.className) != null ? b : "",
      fitToContainer: (x = I.fitToContainer) != null ? x : !0
    }, this.api = new Fe(
      this.config.baseUrl,
      this.config.token || void 0
    ), this.sessionId = He(), this.config.onReady && this.on("ready", this.config.onReady), this.config.onStateChange && this.on("stateChange", this.config.onStateChange), this.config.onError && this.on("error", this.config.onError), this.config.onData && this.on("data", this.config.onData), this.config.onResize && this.on("resize", this.config.onResize), this.config.onDisconnect && this.on("disconnect", this.config.onDisconnect), this.init();
  }
  // ========== Public Properties ==========
  get state() {
    return this._state;
  }
  get session() {
    return this._session;
  }
  get stats() {
    return this._stats;
  }
  // ========== Public Methods ==========
  write(B) {
    if (!this.ws || !this.ws.isConnected()) {
      console.warn("[Rexec] Cannot write: not connected");
      return;
    }
    this.ws.sendRaw(B);
  }
  writeln(B) {
    this.write(B + "\r");
  }
  clear() {
    var B;
    (B = this.terminal) == null || B.clear();
  }
  fit() {
    var B;
    if (this.fitAddon && this.terminal)
      try {
        const I = this.container.getBoundingClientRect();
        if (console.log(
          "[Rexec SDK] fit() called, container size:",
          I.width,
          "x",
          I.height,
          "visible:",
          this.isVisible
        ), I.width === 0 || I.height === 0) {
          console.warn(
            "[Rexec SDK] Container has zero dimensions, skipping fit"
          ), this.pendingFit = !0;
          return;
        }
        this.isVisible || (console.log(
          "[Rexec SDK] Container not visible, marking fit as pending"
        ), this.pendingFit = !0), this.fitAddon.fit();
        const $ = this.fitAddon.proposeDimensions();
        console.log("[Rexec SDK] fit() proposed dimensions:", $), $ && ((B = this.ws) == null || B.sendResize($.cols, $.rows), this.events.emit("resize", $.cols, $.rows));
      } catch (I) {
        console.error("[Rexec SDK] fit() error:", I);
      }
  }
  focus() {
    var B;
    (B = this.terminal) == null || B.focus();
  }
  blur() {
    var B;
    (B = this.terminal) == null || B.blur();
  }
  async reconnect() {
    this.disconnect(), await this.connect();
  }
  disconnect() {
    var B;
    (B = this.ws) == null || B.close(), this.ws = null, this.setState("disconnected");
  }
  destroy() {
    var B, I, $, W, Y, T;
    this.destroyed || (this.destroyed = !0, this.flushTimeout && (clearTimeout(this.flushTimeout), this.flushTimeout = null), this.disconnect(), (B = this.resizeObserver) == null || B.disconnect(), this.resizeObserver = null, (I = this.intersectionObserver) == null || I.disconnect(), this.intersectionObserver = null, ($ = this.webglAddon) == null || $.dispose(), this.webglAddon = null, (W = this.canvasAddon) == null || W.dispose(), this.canvasAddon = null, (Y = this.fitAddon) == null || Y.dispose(), this.fitAddon = null, (T = this.terminal) == null || T.dispose(), this.terminal = null, this.container.innerHTML = "", this.events.removeAllListeners());
  }
  getDimensions() {
    return this.terminal ? {
      cols: this.terminal.cols,
      rows: this.terminal.rows
    } : { cols: 80, rows: 24 };
  }
  async copySelection() {
    var I;
    if (!this.config.allowCopy) return !1;
    const B = (I = this.terminal) == null ? void 0 : I.getSelection();
    if (B)
      try {
        return await navigator.clipboard.writeText(B), !0;
      } catch ($) {
        return !1;
      }
    return !1;
  }
  async paste() {
    if (this.config.allowPaste)
      try {
        const B = await navigator.clipboard.readText();
        B && this.write(B);
      } catch (B) {
      }
  }
  selectAll() {
    var B;
    (B = this.terminal) == null || B.selectAll();
  }
  scrollToBottom() {
    var B;
    (B = this.terminal) == null || B.scrollToBottom();
  }
  setFontSize(B) {
    this.terminal && (this.terminal.options.fontSize = Math.max(8, Math.min(32, B)), this.fit());
  }
  setTheme(B) {
    this.terminal && (this.terminal.options.theme = Oe(B));
  }
  on(B, I) {
    return this.events.on(B, I);
  }
  off(B, I) {
    this.events.off(B, I);
  }
  // ========== Private Methods ==========
  /**
   * Detect base URL from script src or current page
   */
  detectBaseUrl() {
    const B = document.getElementsByTagName("script");
    for (const I of B) {
      const $ = I.src;
      if ($ && ($.includes("rexec") || $.includes("embed")))
        try {
          const W = new URL($);
          return `${W.protocol}//${W.host}`;
        } catch (W) {
        }
    }
    return typeof window != "undefined" && window.location.origin !== "null" ? window.location.origin : "https://rexec.dev";
  }
  /**
   * Initialize the terminal
   */
  async init() {
    this.setupContainer(), this.createTerminal(), await this.connect();
  }
  /**
   * Set up the container element
   */
  setupContainer() {
    if (this.container.classList.add("rexec-embed"), this.config.className && this.container.classList.add(this.config.className), window.getComputedStyle(this.container).position === "static" && (this.container.style.position = "relative"), !document.getElementById("rexec-embed-styles")) {
      const W = document.createElement("style");
      W.id = "rexec-embed-styles", W.textContent = `
        .rexec-embed {
          width: 100%;
          height: 100%;
          min-height: 300px;
          overflow: hidden;
          background: #0d1117;
          position: relative;
          display: flex;
          flex-direction: column;
        }
        .rexec-embed .terminal-wrapper {
          width: 100%;
          flex: 1;
          min-height: 0;
          position: relative;
          display: flex;
          flex-direction: column;
        }
        .rexec-embed .xterm {
          padding: 8px;
          padding-bottom: 28px;
          flex: 1;
          min-height: 0;
        }
        .rexec-embed .xterm-screen {
          width: 100% !important;
          height: 100% !important;
        }
        .rexec-embed .xterm-viewport {
          width: 100% !important;
        }
        /* xterm-helper-textarea styles are managed by xterm.js - don't override */
        .rexec-embed .xterm-screen {
          cursor: text;
        }
        .rexec-embed .terminal-wrapper:focus-within .xterm-cursor {
          animation: blink 1s step-end infinite;
        }
        @keyframes blink {
          50% { opacity: 0; }
        }
        .rexec-embed .xterm-viewport::-webkit-scrollbar {
          width: 8px;
        }
        .rexec-embed .xterm-viewport::-webkit-scrollbar-thumb {
          background: rgba(255, 255, 255, 0.2);
          border-radius: 4px;
        }
        .rexec-embed .xterm-viewport::-webkit-scrollbar-track {
          background: transparent;
        }
        .rexec-embed .status-overlay {
          position: absolute;
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          color: #58a6ff;
          font-family: system-ui, sans-serif;
          font-size: 14px;
          text-align: center;
          z-index: 10;
          pointer-events: none;
        }
        .rexec-embed .status-overlay .spinner {
          width: 24px;
          height: 24px;
          border: 2px solid rgba(88, 166, 255, 0.3);
          border-top-color: #58a6ff;
          border-radius: 50%;
          animation: rexec-spin 1s linear infinite;
          margin: 0 auto 8px;
        }
        @keyframes rexec-spin {
          to { transform: rotate(360deg); }
        }
        .rexec-embed .rexec-branding {
          position: absolute;
          bottom: 0;
          left: 0;
          right: 0;
          height: 24px;
          background: linear-gradient(to top, rgba(13, 17, 23, 0.95) 0%, rgba(13, 17, 23, 0.8) 70%, transparent 100%);
          display: flex;
          align-items: center;
          justify-content: flex-end;
          padding: 0 10px;
          z-index: 5;
          pointer-events: auto;
        }
        .rexec-embed .rexec-branding a {
          display: flex;
          align-items: center;
          gap: 6px;
          text-decoration: none;
          color: rgba(255, 255, 255, 0.5);
          font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
          font-size: 11px;
          font-weight: 500;
          transition: color 0.2s, transform 0.2s;
        }
        .rexec-embed .rexec-branding a:hover {
          color: #00ff41;
          transform: translateY(-1px);
        }
        .rexec-embed .rexec-branding .rexec-logo {
          width: 14px;
          height: 14px;
          fill: currentColor;
        }
        .rexec-embed .rexec-branding .powered-text {
          opacity: 0.7;
        }
        .rexec-embed .rexec-branding .rexec-name {
          color: #00ff41;
          font-weight: 600;
          letter-spacing: 0.5px;
        }
        .rexec-embed .rexec-branding a:hover .rexec-name {
          text-shadow: 0 0 8px rgba(0, 255, 65, 0.5);
        }
      `, document.head.appendChild(W);
    }
    const I = document.createElement("div");
    I.className = "terminal-wrapper", I.setAttribute("tabindex", "0"), this.container.appendChild(I);
    const $ = document.createElement("div");
    $.className = "rexec-branding", $.innerHTML = `
      <a href="https://rexec.sh" target="_blank" rel="noopener noreferrer" title="Powered by Rexec - Terminal as a Service">
        <span class="powered-text">Powered by</span>
        <svg class="rexec-logo" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path d="M4 4h16v2H4V4zm0 4h10v2H4V8zm0 4h16v2H4v-2zm0 4h10v2H4v-2zm12 0h4v4h-4v-4z"/>
        </svg>
        <span class="rexec-name">Rexec</span>
      </a>
    `, this.container.appendChild($), this.container.addEventListener("click", (W) => {
      var Y;
      W.target.closest(".rexec-branding") || (Y = this.terminal) == null || Y.focus();
    });
  }
  /**
   * Create the xterm.js terminal
   */
  createTerminal() {
    console.log("[Rexec SDK] createTerminal called");
    const B = this.container.querySelector(".terminal-wrapper");
    if (!B) {
      console.error("[Rexec SDK] No .terminal-wrapper found in container!");
      return;
    }
    console.log("[Rexec SDK] Found terminal wrapper:", B), this.terminal = new Ye.Terminal({
      cursorBlink: this.config.cursorBlink,
      cursorStyle: this.config.cursorStyle,
      fontSize: this.config.fontSize,
      fontFamily: this.config.fontFamily,
      theme: Oe(this.config.theme),
      scrollback: this.config.scrollback,
      allowProposedApi: !0,
      convertEol: !0,
      scrollOnUserInput: !0,
      altClickMovesCursor: !0,
      macOptionIsMeta: !0,
      macOptionClickForcesSelection: !0
    }), this.fitAddon = new Qe.FitAddon(), this.terminal.loadAddon(this.fitAddon);
    const I = new rt.Unicode11Addon();
    this.terminal.loadAddon(I), this.terminal.unicode.activeVersion = "11";
    const $ = new tt.WebLinksAddon();
    if (this.terminal.loadAddon($), console.log("[Rexec SDK] Opening terminal in wrapper"), this.terminal.open(B), console.log("[Rexec SDK] Terminal opened, element:", this.terminal.element), this.config.webgl)
      try {
        this.webglAddon = new ot.WebglAddon(), this.webglAddon.onContextLoss(() => {
          var W;
          console.warn(
            "[Rexec SDK] WebGL context lost, falling back to canvas"
          ), (W = this.webglAddon) == null || W.dispose(), this.webglAddon = null, this.loadCanvasAddon();
        }), this.terminal.loadAddon(this.webglAddon), console.log("[Rexec SDK] WebGL renderer loaded");
      } catch (W) {
        console.warn("[Rexec SDK] WebGL not available, using canvas renderer"), this.loadCanvasAddon();
      }
    else
      this.loadCanvasAddon();
    requestAnimationFrame(() => {
      console.log("[Rexec SDK] Running initial fit sequence"), this.waitForDimensionsAndFit();
    }), this.setupResizeObserver(), this.setupIntersectionObserver(), setTimeout(() => {
      console.log("[Rexec SDK] Forcing initial render"), this.terminal && this.terminal.refresh(0, this.terminal.rows - 1);
    }, 100), this.terminal.write(
      `\x1B[33m[Rexec SDK] Terminal initialized...\x1B[0m\r
`
    ), console.log("[Rexec SDK] Wrote test message to terminal"), this.terminal.onData((W) => {
      var Y;
      console.log("[Rexec SDK] Terminal input:", W.length, "chars"), (Y = this.ws) != null && Y.isConnected() ? this.ws.sendRaw(W) : console.warn("[Rexec SDK] WebSocket not connected, can't send input");
    }), this.terminal.onResize(({ cols: W, rows: Y }) => {
      var T;
      (T = this.ws) == null || T.sendResize(W, Y), this.events.emit("resize", W, Y);
    }), this.config.allowPaste && this.terminal.attachCustomKeyEventHandler((W) => {
      var Y;
      return W.type === "keydown" && W.key === "v" && (W.ctrlKey || W.metaKey) ? (this.paste(), !1) : W.type === "keydown" && W.key === "c" && (W.ctrlKey || W.metaKey) && ((Y = this.terminal) != null && Y.hasSelection()) ? (this.copySelection(), !1) : !0;
    });
  }
  /**
   * Load the canvas addon for reliable rendering
   */
  loadCanvasAddon() {
    if (!(!this.terminal || this.canvasAddon))
      try {
        this.canvasAddon = new ht.CanvasAddon(), this.terminal.loadAddon(this.canvasAddon), console.log("[Rexec SDK] Canvas renderer loaded");
      } catch (B) {
        console.warn("[Rexec SDK] Canvas addon failed to load:", B);
      }
  }
  /**
   * Set up resize observer for auto-fitting
   */
  setupResizeObserver() {
    if (!this.config.fitToContainer) return;
    let B = null, I = !1;
    this.resizeObserver = new ResizeObserver((W) => {
      const Y = W[0];
      if (!Y) return;
      const { width: T, height: t } = Y.contentRect;
      if (console.log("[Rexec SDK] ResizeObserver triggered:", T, "x", t), T > 0 && t > 0 && !I) {
        I = !0, console.log("[Rexec SDK] Container has dimensions, doing initial fit"), setTimeout(() => this.fit(), 0), setTimeout(() => this.fit(), 100);
        return;
      }
      B && clearTimeout(B), B = setTimeout(() => this.fit(), 50);
    }), this.resizeObserver.observe(this.container);
    const $ = this.container.querySelector(".terminal-wrapper");
    $ && this.resizeObserver.observe($);
  }
  /**
   * Set up intersection observer to detect when terminal becomes visible
   * This handles cases where terminal is in a modal, tab, or hidden container
   */
  setupIntersectionObserver() {
    console.log("[Rexec SDK] Setting up IntersectionObserver"), this.intersectionObserver = new IntersectionObserver(
      (B) => {
        const I = B[0];
        if (!I) return;
        const $ = this.isVisible;
        this.isVisible = I.isIntersecting && I.intersectionRatio > 0, console.log(
          "[Rexec SDK] IntersectionObserver:",
          "visible:",
          this.isVisible,
          "ratio:",
          I.intersectionRatio
        ), this.isVisible && !$ && (console.log("[Rexec SDK] Terminal became visible, triggering fit"), setTimeout(() => this.fit(), 0), setTimeout(() => this.fit(), 50), setTimeout(() => this.fit(), 150), setTimeout(() => this.fit(), 300), setTimeout(() => {
          var W;
          return (W = this.terminal) == null ? void 0 : W.focus();
        }, 100)), this.isVisible && this.pendingFit && (this.pendingFit = !1, this.fit());
      },
      {
        root: null,
        // viewport
        threshold: [0, 0.1, 0.5, 1]
        // trigger at multiple visibility levels
      }
    ), this.intersectionObserver.observe(this.container);
  }
  /**
   * Wait for container to have dimensions, then fit
   */
  waitForDimensionsAndFit() {
    const B = () => {
      const Y = this.container.getBoundingClientRect();
      return console.log(
        "[Rexec SDK] Checking dimensions:",
        Y.width,
        "x",
        Y.height
      ), Y.width > 0 && Y.height > 0 ? (console.log("[Rexec SDK] Container ready, fitting terminal"), this.fit(), !0) : !1;
    };
    if (B()) return;
    let I = 0;
    const $ = 20, W = setInterval(() => {
      I++, (B() || I >= $) && (clearInterval(W), I >= $ && console.warn(
        "[Rexec SDK] Container never got dimensions after",
        $,
        "attempts"
      ));
    }, 100);
  }
  /**
   * Connect to the terminal session
   */
  async connect() {
    if (!this.destroyed) {
      this.setState("connecting"), this.showStatus("Connecting...");
      try {
        let B, I;
        if (this.config.shareCode) {
          const { data: $, error: W } = await this.api.joinSession(
            this.config.shareCode
          );
          if (W || !$)
            throw this.createError(
              "JOIN_FAILED",
              W || "Failed to join session"
            );
          B = $.container_id, this._session = {
            id: $.session_id,
            containerId: $.container_id,
            containerName: $.container_name,
            mode: $.mode,
            expiresAt: $.expires_at
          }, I = this.api.getTerminalWsUrl(B, this.sessionId);
        } else if (this.config.container)
          B = this.config.container, this._session = {
            id: this.sessionId,
            containerId: B
          }, I = this.api.getTerminalWsUrl(B, this.sessionId);
        else if (this.config.role || this.config.image) {
          this.showStatus("Creating container...");
          const { data: $, error: W } = await this.api.createContainer(
            this.config.image || "ubuntu",
            this.config.role
          );
          if (W || !$)
            throw this.createError(
              "CREATE_FAILED",
              W || "Failed to create container"
            );
          const Y = $.id;
          this.showStatus("Waiting for container to start...");
          const { data: T, error: t } = await this.api.waitForContainer(Y, {
            maxAttempts: 90,
            // Up to 3 minutes for slow roles
            intervalMs: 2e3,
            onProgress: (a, c) => {
              const r = {
                creating: "Creating container...",
                pulling: "Pulling image...",
                configuring: "Configuring environment...",
                starting: "Starting container...",
                running: "Container ready!"
              }[a] || `Preparing container (${a})...`;
              this.showStatus(r);
            }
          });
          if (t || !T)
            throw this.createError(
              "CREATE_FAILED",
              t || "Container failed to start"
            );
          B = T.docker_id || T.id, this._session = {
            id: this.sessionId,
            containerId: B,
            containerName: T.name,
            role: T.role
          }, I = this.api.getTerminalWsUrl(B, this.sessionId);
        } else
          throw this.createError(
            "CONFIG_ERROR",
            "Must provide container, shareCode, role, or image"
          );
        this.connectWebSocket(I);
      } catch (B) {
        const I = B instanceof Error ? this.createError("CONNECT_ERROR", B.message) : B;
        this.handleError(I);
      }
    }
  }
  /**
   * Connect WebSocket to the terminal
   */
  connectWebSocket(B) {
    console.log("[Rexec SDK] connectWebSocket called with URL:", B), this.ws = new _t(B, this.config.token || null, {
      autoReconnect: this.config.autoReconnect,
      maxReconnectAttempts: this.config.maxReconnectAttempts
    }), this.ws.onOpen = () => {
      var $;
      console.log("[Rexec SDK] WebSocket opened!"), this.hideStatus(), this.setState("connected");
      const I = this.getDimensions();
      ($ = this.ws) == null || $.sendResize(I.cols, I.rows), setTimeout(() => {
        this.terminal && (console.log("[Rexec SDK] Forcing terminal refresh"), this.terminal.refresh(0, this.terminal.rows - 1), this.fit());
      }, 50), setTimeout(() => {
        this.terminal && (this.terminal.refresh(0, this.terminal.rows - 1), this.fit());
      }, 200), setTimeout(() => {
        var Y;
        (Y = this.terminal) == null || Y.focus();
        const W = this.container.querySelector(
          ".xterm-helper-textarea"
        );
        W && W.focus();
      }, 100), this.config.initialCommand && setTimeout(() => {
        this.writeln(this.config.initialCommand);
      }, 500), this.events.emit("ready", this);
    }, this.ws.onClose = (I, $) => {
      console.log("[Rexec SDK] WebSocket closed:", I, $), I !== 1e3 && this.events.emit("disconnect", $ || "Connection closed"), this._state !== "reconnecting" && this.setState("disconnected");
    }, this.ws.onError = () => {
    }, this.ws.onReconnecting = (I) => {
      this.setState("reconnecting"), this.showStatus(`Reconnecting... (${I})`);
    }, this.ws.onMessage = (I) => {
      var $, W;
      console.log(
        "[Rexec SDK] WS message received:",
        I.type,
        "data:",
        ((W = ($ = I.data) == null ? void 0 : $.substring) == null ? void 0 : W.call($, 0, 50)) || "(none)"
      ), this.handleMessage(I);
    }, console.log("[Rexec SDK] Calling ws.connect()"), this.ws.connect();
  }
  /**
   * Handle incoming WebSocket message
   */
  handleMessage(B) {
    var I;
    switch (console.log(
      "[Rexec Terminal] handleMessage:",
      B.type,
      "data length:",
      ((I = B.data) == null ? void 0 : I.length) || 0
    ), B.type) {
      case "output":
        B.data && (console.log(
          "[Rexec Terminal] Writing output to terminal:",
          B.data.substring(0, 100)
        ), this.writeToTerminal(B.data), this.events.emit("data", B.data), this.terminal && this.terminal.refresh(0, this.terminal.rows - 1));
        break;
      case "connected":
        this.hideStatus(), this.setState("connected");
        break;
      case "stats":
        if (B.data)
          try {
            const $ = typeof B.data == "string" ? JSON.parse(B.data) : B.data;
            this._stats = {
              cpu: $.cpu || 0,
              memory: $.memory || 0,
              memoryLimit: $.memory_limit || 0,
              diskRead: $.disk_read || 0,
              diskWrite: $.disk_write || 0,
              diskUsage: $.disk_usage,
              diskLimit: $.disk_limit || 0,
              netRx: $.net_rx || 0,
              netTx: $.net_tx || 0
            }, this.events.emit("stats", this._stats);
          } catch ($) {
          }
        break;
      case "error":
        this.handleError(
          this.createError("SERVER_ERROR", B.data || "Server error")
        );
        break;
      case "setup":
        this.showStatus(B.data || "Setting up...");
        break;
      default:
        B.data && typeof B.data == "string" && this.writeToTerminal(B.data);
    }
  }
  /**
   * Write data to terminal with buffering for performance
   */
  writeToTerminal(B) {
    if (console.log(
      "[Rexec Terminal] writeToTerminal called, terminal exists:",
      !!this.terminal,
      "data length:",
      B.length
    ), !this.terminal) {
      console.error("[Rexec Terminal] No terminal instance!");
      return;
    }
    if (B.length < 256) {
      console.log("[Rexec Terminal] Writing small output directly"), this.terminal.write(B);
      return;
    }
    if (this.outputBuffer += B, this.outputBuffer.length > 32 * 1024) {
      this.flushOutput();
      return;
    }
    this.flushTimeout || (this.flushTimeout = setTimeout(() => this.flushOutput(), 8));
  }
  /**
   * Flush output buffer to terminal
   */
  flushOutput() {
    this.flushTimeout && (clearTimeout(this.flushTimeout), this.flushTimeout = null), this.outputBuffer && this.terminal && (this.terminal.write(this.outputBuffer), this.outputBuffer = "");
  }
  /**
   * Update connection state
   */
  setState(B) {
    this._state !== B && (this._state = B, this.events.emit("stateChange", B));
  }
  /**
   * Show status overlay
   */
  showStatus(B) {
    if (!this.config.showStatus) return;
    let I = this.container.querySelector(".status-overlay");
    I || (I = document.createElement("div"), I.className = "status-overlay", this.container.appendChild(I)), I.innerHTML = `
      <div class="spinner"></div>
      <div>${B}</div>
      <div style="margin-top: 16px; display: flex; align-items: center; gap: 6px; opacity: 0.6;">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" xmlns="http://www.w3.org/2000/svg">
          <path d="M4 4h16v2H4V4zm0 4h10v2H4V8zm0 4h16v2H4v-2zm0 4h10v2H4v-2zm12 0h4v4h-4v-4z"/>
        </svg>
        <span style="font-size: 12px; color: #00ff41; font-weight: 600; letter-spacing: 0.5px;">Rexec</span>
      </div>
    `;
  }
  /**
   * Hide status overlay
   */
  hideStatus() {
    const B = this.container.querySelector(".status-overlay");
    B && B.remove();
  }
  /**
   * Create an error object
   */
  createError(B, I, $ = !1) {
    return { code: B, message: I, recoverable: $ };
  }
  /**
   * Handle an error
   */
  handleError(B) {
    this.setState("error"), this.showStatus(`Error: ${B.message}`), this.events.emit("error", B);
  }
}
const mt = "1.0.0", Le = /* @__PURE__ */ new Map();
function Ct() {
  return `rexec-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}
function St(ne, B = {}) {
  const I = new Ke(ne, B), $ = Ct();
  return Le.set($, I), I;
}
function wt() {
  return Array.from(Le.values());
}
function bt() {
  Le.forEach((ne) => ne.destroy()), Le.clear();
}
function yt(ne, B) {
  return new Fe(ne, B);
}
const xt = {
  dark: Pe,
  light: Ie,
  get: Oe
}, Lt = {
  generateSessionId: He
}, Et = {
  // Main API
  embed: St,
  createClient: yt,
  // Instance management
  getInstances: wt,
  destroyAll: bt,
  // Themes
  themes: xt,
  DARK_THEME: Pe,
  LIGHT_THEME: Ie,
  // Classes for advanced usage
  Terminal: Ke,
  ApiClient: Fe,
  // Utilities
  utils: Lt,
  generateSessionId: He,
  // Version
  VERSION: mt
};
typeof window != "undefined" && (window.Rexec = Et);
export {
  Pe as DARK_THEME,
  Ie as LIGHT_THEME,
  Fe as RexecApiClient,
  Ke as RexecTerminal,
  _t as TerminalWebSocket,
  mt as VERSION,
  yt as createClient,
  Et as default,
  bt as destroyAll,
  St as embed,
  He as generateSessionId,
  wt as getInstances,
  Oe as getTheme,
  xt as themes,
  Lt as utils
};
//# sourceMappingURL=rexec.esm.js.map
