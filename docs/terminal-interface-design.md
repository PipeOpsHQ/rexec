# Rexec Terminal Interface Design Guide

This document describes the Rexec terminal interface design system, including colors, typography, components, and styling patterns. Use this as a reference to create consistent interfaces in your console or terminal-based applications.

---

## Color Palette

### Dark Theme (Default)

| Variable | Value | Usage |
|----------|-------|-------|
| `--bg` | `#050505` | Main background |
| `--bg-card` | `#0a0a0a` | Card/panel backgrounds |
| `--bg-secondary` | `#0a0a0a` | Secondary backgrounds |
| `--bg-tertiary` | `#111` | Tertiary/elevated backgrounds |
| `--terminal-bg` | `#0a0a0a` | Terminal area background |
| `--border` | `#333` | Default borders |
| `--border-muted` | `#222` | Subtle borders |
| `--border-active` | `#00ff41` | Active/focused borders |
| `--text` | `#e0e0e0` | Primary text |
| `--text-secondary` | `#a0a0a0` | Secondary text |
| `--text-muted` | `#a0a0a0` | Muted/disabled text |
| `--accent` | `#00ff41` | Primary accent (neon green) |
| `--accent-dim` | `rgba(0, 255, 65, 0.1)` | Subtle accent background |
| `--green` | `#00ff41` | Success/connected |
| `--red` | `#ff003c` | Error/danger |
| `--yellow` | `#fcee0a` | Warning/pending |

### Terminal Color Scheme (xterm.js)

```javascript
const DARK_THEME = {
  background: '#0d1117',
  foreground: '#c9d1d9',
  cursor: '#58a6ff',
  cursorAccent: '#0d1117',
  selectionBackground: 'rgba(56, 139, 253, 0.4)',
  selectionForeground: '#ffffff',
  
  // ANSI Colors
  black: '#484f58',
  red: '#ff7b72',
  green: '#3fb950',
  yellow: '#d29922',
  blue: '#58a6ff',
  magenta: '#bc8cff',
  cyan: '#39c5cf',
  white: '#b1bac4',
  
  // Bright ANSI Colors
  brightBlack: '#6e7681',
  brightRed: '#ffa198',
  brightGreen: '#56d364',
  brightYellow: '#e3b341',
  brightBlue: '#79c0ff',
  brightMagenta: '#d2a8ff',
  brightCyan: '#56d4dd',
  brightWhite: '#f0f6fc',
};
```

### Light Theme

| Variable | Value | Usage |
|----------|-------|-------|
| `--bg` | `#f5f5f5` | Main background |
| `--bg-card` | `#ffffff` | Card/panel backgrounds |
| `--border` | `#d0d0d0` | Default borders |
| `--text` | `#1a1a1a` | Primary text |
| `--accent` | `#00a830` | Primary accent (darker green) |

---

## Typography

### Font Families

```css
--font-mono: "JetBrains Mono", monospace;
--font-sans: "Inter", sans-serif;
```

- **Terminal/Code**: Always use `--font-mono`
- **UI Labels**: Can use either, prefer mono for consistency

### Font Sizes

| Element | Size |
|---------|------|
| Terminal text | `14px` (default) |
| Toolbar buttons | `11-12px` |
| Tab names | `13px` |
| Status labels | `10-11px` |
| Headers | `14px` uppercase |

---

## Terminal Container Structure

```
┌─────────────────────────────────────────────────────────────┐
│ ┌─────────────────────────────────────────────────────────┐ │
│ │  TABS BAR                                     ACTIONS   │ │
│ │  [Tab1] [Tab2] [+]                    [📋][⬇][▢][✕]    │ │
│ └─────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │  TOOLBAR                                                │ │
│ │  name | ● status | role | CPU: 5% | MEM: 128MB | ...   │ │
│ └─────────────────────────────────────────────────────────┘ │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │                                                         │ │
│ │                    TERMINAL AREA                        │ │
│ │                                                         │ │
│ │  user@container:~$ _                                    │ │
│ │                                                         │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Styles

### Tabs

```css
.tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  background: #0a0a0a;
  border: 1px solid #333;
  border-radius: 4px 4px 0 0;
  color: #a0a0a0;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s ease;
  border-bottom: none;
  margin-bottom: -1px;
}

.tab:hover {
  background: #0a0a0a;
  color: #e0e0e0;
}

.tab.active {
  background: #050505;
  border-color: #00ff41;
  border-bottom-color: #050505;
  color: #e0e0e0;
}

/* Agent tabs have special styling */
.tab.agent-tab {
  border-color: rgba(0, 255, 65, 0.5);
  background: rgba(0, 255, 65, 0.08);
  color: #00ff41;
}

.tab.agent-tab.active {
  border-color: #00ff41;
  box-shadow: 0 0 8px rgba(0, 255, 65, 0.35);
}
```

### Status Indicators

```css
.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.status-connected {
  background: #00ff41;
}

.status-connecting {
  background: #fcee0a;
  animation: pulse 1s infinite;
}

.status-disconnected {
  background: #ff003c;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
```

### Toolbar

```css
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 6px 12px;
  background: #0a0a0a;
  border-bottom: 1px solid #333;
  font-size: 12px;
  overflow-x: auto;
}

.toolbar-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 4px 8px;
  background: transparent;
  border: 1px solid #333;
  color: #a0a0a0;
  font-family: "JetBrains Mono", monospace;
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s;
}

.toolbar-btn:hover {
  border-color: #00ff41;
  color: #00ff41;
  background: rgba(0, 255, 65, 0.1);
}

.toolbar-divider {
  width: 1px;
  height: 16px;
  background: #333;
}
```

### Stats Display

```css
.stats {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 11px;
  color: #a0a0a0;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.stat-label {
  text-transform: uppercase;
  letter-spacing: 0.5px;
  font-size: 10px;
}

.stat-value {
  font-weight: 500;
  color: #e0e0e0;
}

/* Color coding for resource usage */
.stat-value.low { color: #3fb950; }    /* < 50% */
.stat-value.medium { color: #d29922; } /* 50-80% */
.stat-value.high { color: #ff7b72; }   /* > 80% */
```

### Buttons

```css
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 8px 16px;
  border: 1px solid #333;
  background: #0a0a0a;
  color: #e0e0e0;
  font-family: "JetBrains Mono", monospace;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  cursor: pointer;
  transition: all 0.2s;
  border-radius: 0; /* Sharp corners */
}

.btn:hover {
  border-color: #00ff41;
  color: #00ff41;
  box-shadow: 0 0 10px rgba(0, 255, 65, 0.5);
}

.btn-primary {
  background: rgba(0, 255, 65, 0.1);
  border-color: #00ff41;
  color: #00ff41;
}

.btn-primary:hover {
  background: #00ff41;
  color: #050505;
}

.btn-danger {
  border-color: #ff003c;
  color: #ff003c;
}

.btn-danger:hover {
  background: #ff003c;
  color: #050505;
  box-shadow: 0 0 10px rgba(255, 0, 60, 0.5);
}
```

### Modals/Overlays

```css
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.85);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10001;
  backdrop-filter: blur(2px);
}

.modal {
  background: #0a0a0a;
  border: 1px solid #333;
  border-radius: 0;
  box-shadow: 0 0 20px rgba(0, 0, 0, 0.8);
  max-width: 400px;
  width: 90%;
  animation: modalSlideIn 0.2s ease-out;
}

@keyframes modalSlideIn {
  from {
    opacity: 0;
    transform: scale(0.95) translateY(-10px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #333;
}

.modal-title {
  font-size: 14px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 1px;
  color: #00ff41;
}
```

### Inputs

```css
input, textarea, select {
  background: #0a0a0a;
  border: 1px solid #333;
  color: #e0e0e0;
  font-family: "JetBrains Mono", monospace;
  font-size: 13px;
  padding: 10px 14px;
  outline: none;
  transition: border-color 0.2s;
  border-radius: 0;
}

input:focus, textarea:focus, select:focus {
  border-color: #00ff41;
  box-shadow: 0 0 0 2px rgba(0, 255, 65, 0.15);
}

input::placeholder {
  color: #a0a0a0;
}
```

---

## Visual Effects

### CRT Scanline Effect (Optional)

```css
body::after {
  content: "";
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  background: linear-gradient(
    rgba(18, 16, 16, 0) 50%,
    rgba(0, 0, 0, 0.25) 50%
  );
  background-size: 100% 4px;
  pointer-events: none;
  z-index: 9999;
  opacity: 0.15;
}
```

### Grid Background

```css
body {
  background-color: #050505;
  background-image: radial-gradient(circle, #1a1a1a 1px, transparent 1px);
  background-size: 20px 20px;
}
```

### Glow Effects

```css
/* Accent glow for hover states */
.element:hover {
  box-shadow: 0 0 10px rgba(0, 255, 65, 0.5);
}

/* Agent connection glow */
.agent-element {
  box-shadow: 0 0 8px rgba(0, 255, 65, 0.35);
}
```

---

## Scrollbar Styling

```css
/* Firefox */
* {
  scrollbar-width: thin;
  scrollbar-color: #333 #050505;
}

/* WebKit (Chrome, Safari, Edge) */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: #050505;
}

::-webkit-scrollbar-thumb {
  background: #333;
  border: 1px solid #050505;
}

::-webkit-scrollbar-thumb:hover {
  background: #00ff41;
}
```

---

## Terminal View Modes

### 1. Docked Mode (Bottom Panel)

```css
.docked-terminal {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  height: 40vh; /* Adjustable */
  min-height: 150px;
  max-height: 90vh;
  background: #050505;
  border-top: 1px solid #333;
  z-index: 1000;
  display: flex;
  flex-direction: column;
}
```

### 2. Floating Mode (Draggable Window)

```css
.floating-terminal {
  position: fixed;
  width: 800px;
  height: 500px;
  background: #050505;
  border: 1px solid #333;
  box-shadow: 0 0 20px rgba(0, 0, 0, 0.8);
  z-index: 1002;
  display: flex;
  flex-direction: column;
  /* Draggable - position set via JS */
}

.floating-terminal.focused {
  border-color: #00ff41;
  box-shadow: 0 0 30px rgba(0, 255, 65, 0.2);
}
```

### 3. Fullscreen Mode

```css
.fullscreen-terminal {
  position: fixed;
  inset: 0;
  background: #050505;
  z-index: 10000;
  display: flex;
  flex-direction: column;
}
```

---

## Animations

```css
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slideUp {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
```

---

## Loading Spinner

```css
.spinner {
  width: 16px;
  height: 16px;
  border: 2px solid #333;
  border-top-color: #00ff41;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
```

---

## Badges

```css
.badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  border-radius: 0;
}

.badge-agent {
  background: rgba(0, 255, 65, 0.15);
  color: #00ff41;
  border: 1px solid rgba(0, 255, 65, 0.3);
}

.badge-role {
  background: rgba(0, 255, 65, 0.08);
  border: 1px solid rgba(0, 255, 65, 0.2);
  color: #00ff41;
}
```

---

## Key Design Principles

1. **Sharp Corners**: No border-radius (or minimal `0-4px`) for a terminal/hacker aesthetic
2. **Monospace Everything**: Use `JetBrains Mono` for all text
3. **Neon Green Accent**: `#00ff41` is the primary accent color
4. **Dark by Default**: Near-black backgrounds (`#050505`, `#0a0a0a`)
5. **Subtle Borders**: Use `#333` for borders, `#222` for muted
6. **Glow on Interaction**: Add box-shadow glow effects on hover/focus
7. **Uppercase Labels**: Small labels in uppercase with letter-spacing
8. **Minimal Animations**: Quick transitions (0.15s-0.2s), subtle effects

---

## xterm.js Configuration

```javascript
const terminal = new Terminal({
  cursorBlink: true,
  cursorStyle: 'block', // 'block' | 'underline' | 'bar'
  fontSize: 14,
  fontFamily: 'JetBrains Mono, Menlo, Monaco, Consolas, monospace',
  theme: DARK_THEME,
  scrollback: 5000,
  allowProposedApi: true,
  convertEol: true,
  scrollOnUserInput: true,
  altClickMovesCursor: true,
  macOptionIsMeta: true,
  macOptionClickForcesSelection: true,
});
```

---

## Example: Complete Terminal Panel

```html
<div class="terminal-panel">
  <!-- Toolbar -->
  <div class="toolbar">
    <span class="terminal-name">ubuntu-dev</span>
    <span class="status">
      <span class="status-dot status-connected"></span>
      connected
    </span>
    <span class="badge badge-role">ubuntu</span>
    
    <div class="stats">
      <span class="stat-item">
        <span class="stat-label">CPU</span>
        <span class="stat-value">5%</span>
      </span>
      <span class="stat-divider"></span>
      <span class="stat-item">
        <span class="stat-label">MEM</span>
        <span class="stat-value">128MB</span>
      </span>
    </div>
    
    <div class="toolbar-actions">
      <button class="toolbar-btn">Copy</button>
      <button class="toolbar-btn">Paste</button>
      <span class="toolbar-divider"></span>
      <button class="toolbar-btn btn-icon">⚙</button>
    </div>
  </div>
  
  <!-- Terminal Area -->
  <div class="terminal-container" id="terminal"></div>
</div>
```

```css
.terminal-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #050505;
  border: 1px solid #333;
}

.terminal-container {
  flex: 1;
  overflow: hidden;
  background: #0a0a0a;
}

.terminal-container :global(.xterm) {
  padding: 8px;
  height: 100%;
}
```
