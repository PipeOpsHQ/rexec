import { writable, derived, get } from "svelte/store";

type Theme = "dark" | "light";

export interface AccentColor {
  name: string;
  value: string;
  light?: string; // Optional lighter version for light theme
}

// Preset accent colors
export const accentPresets: AccentColor[] = [
  { name: "Matrix Green", value: "#00ff41", light: "#00a830" },
  { name: "Cyan", value: "#00d4ff", light: "#0099cc" },
  { name: "Purple", value: "#a855f7", light: "#7c3aed" },
  { name: "Pink", value: "#ec4899", light: "#db2777" },
  { name: "Orange", value: "#f97316", light: "#ea580c" },
  { name: "Yellow", value: "#facc15", light: "#ca8a04" },
  { name: "Red", value: "#ef4444", light: "#dc2626" },
  { name: "Blue", value: "#3b82f6", light: "#2563eb" },
  { name: "Teal", value: "#14b8a6", light: "#0d9488" },
  { name: "Lime", value: "#84cc16", light: "#65a30d" },
];

// Check if we're in a browser environment
const browser = typeof window !== "undefined";

// Storage keys
const THEME_KEY = "rexec-theme";
const ACCENT_KEY = "rexec-accent";

function createThemeStore() {
  // Get initial theme from localStorage or system preference
  const getInitialTheme = (): Theme => {
    if (!browser) return "dark";

    const stored = localStorage.getItem(THEME_KEY);
    if (stored === "light" || stored === "dark") {
      return stored;
    }

    // Check system preference
    if (
      window.matchMedia &&
      window.matchMedia("(prefers-color-scheme: light)").matches
    ) {
      return "light";
    }

    return "dark";
  };

  // Get initial accent color from localStorage
  const getInitialAccent = (): string => {
    if (!browser) return accentPresets[0].value;

    const stored = localStorage.getItem(ACCENT_KEY);
    if (stored) {
      return stored;
    }

    return accentPresets[0].value; // Default: Matrix Green
  };

  const { subscribe, set, update } = writable<Theme>(getInitialTheme());
  const accentStore = writable<string>(getInitialAccent());

  // Helper to convert hex to RGB components
  const hexToRgb = (
    hex: string,
  ): { r: number; g: number; b: number } | null => {
    const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
    return result
      ? {
          r: parseInt(result[1], 16),
          g: parseInt(result[2], 16),
          b: parseInt(result[3], 16),
        }
      : null;
  };

  // Helper to darken a color for light theme
  const darkenColor = (hex: string, percent: number = 20): string => {
    const rgb = hexToRgb(hex);
    if (!rgb) return hex;

    const factor = (100 - percent) / 100;
    const r = Math.round(rgb.r * factor);
    const g = Math.round(rgb.g * factor);
    const b = Math.round(rgb.b * factor);

    return `#${r.toString(16).padStart(2, "0")}${g.toString(16).padStart(2, "0")}${b.toString(16).padStart(2, "0")}`;
  };

  // Helper to lighten a color for hover states
  const lightenColor = (hex: string, percent: number = 15): string => {
    const rgb = hexToRgb(hex);
    if (!rgb) return hex;

    const r = Math.min(
      255,
      Math.round(rgb.r + (255 - rgb.r) * (percent / 100)),
    );
    const g = Math.min(
      255,
      Math.round(rgb.g + (255 - rgb.g) * (percent / 100)),
    );
    const b = Math.min(
      255,
      Math.round(rgb.b + (255 - rgb.b) * (percent / 100)),
    );

    return `#${r.toString(16).padStart(2, "0")}${g.toString(16).padStart(2, "0")}${b.toString(16).padStart(2, "0")}`;
  };

  // Apply theme and accent to document
  const applyTheme = (theme: Theme, accent?: string) => {
    if (!browser) return;

    document.documentElement.setAttribute("data-theme", theme);
    localStorage.setItem(THEME_KEY, theme);

    if (accent) {
      applyAccent(accent, theme);
    }
  };

  // Apply accent color to CSS variables
  const applyAccent = (accent: string, theme?: Theme) => {
    if (!browser) return;

    const currentTheme = theme || get({ subscribe });
    const rgb = hexToRgb(accent);

    if (!rgb) return;

    // Find preset or use custom
    const preset = accentPresets.find(
      (p) => p.value.toLowerCase() === accent.toLowerCase(),
    );
    const effectiveAccent =
      currentTheme === "light"
        ? preset?.light || darkenColor(accent, 25)
        : accent;

    const effectiveRgb = hexToRgb(effectiveAccent) || rgb;

    // Calculate hover color (slightly brighter)
    const hoverAccent = lightenColor(effectiveAccent, 15);

    // Set CSS custom properties
    document.documentElement.style.setProperty("--accent", effectiveAccent);
    document.documentElement.style.setProperty("--accent-hover", hoverAccent);
    document.documentElement.style.setProperty(
      "--accent-rgb",
      `${effectiveRgb.r}, ${effectiveRgb.g}, ${effectiveRgb.b}`,
    );
    document.documentElement.style.setProperty(
      "--accent-dim",
      `rgba(${effectiveRgb.r}, ${effectiveRgb.g}, ${effectiveRgb.b}, 0.1)`,
    );
    document.documentElement.style.setProperty(
      "--accent-glow",
      `0 0 10px rgba(${effectiveRgb.r}, ${effectiveRgb.g}, ${effectiveRgb.b}, 0.5)`,
    );
    document.documentElement.style.setProperty("--green", effectiveAccent);

    // Update cursor colors (for custom cursors)
    updateCursors(effectiveAccent);

    localStorage.setItem(ACCENT_KEY, accent);
    accentStore.set(accent);
  };

  // Update custom cursor SVGs with new accent color
  const updateCursors = (accent: string) => {
    const encodedColor = encodeURIComponent(accent);

    const cursorDefault = `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='24' height='24' viewBox='0 0 24 24'%3E%3Cpath d='M5 3L5 21L10 16L14 24L18 22L14 14L21 14L5 3Z' fill='${encodedColor}' stroke='%23000' stroke-width='1.5'/%3E%3C/svg%3E")`;
    const cursorPointer = `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='24' height='24' viewBox='0 0 24 24'%3E%3Cpath d='M7 2C6 2 5.5 2.5 5.5 3.5L5.5 12L4 12C3 12 2.5 12.5 2.5 13.5L2.5 14C2.5 17 5 20 9 20L13 20C16 20 18 17.5 18 14.5L18 9.5C18 8.5 17.5 8 16.5 8C16 8 15.5 8.2 15.2 8.5C15 8 14.5 7.5 13.5 7.5C13 7.5 12.5 7.7 12.2 8C12 7.5 11.5 7 10.5 7C10 7 9.5 7.2 9.2 7.5L9.2 3.5C9.2 2.5 8.5 2 7.5 2L7 2Z' fill='${encodedColor}' stroke='%23000' stroke-width='1'/%3E%3C/svg%3E")`;
    const cursorText = `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='24' height='24' viewBox='0 0 24 24'%3E%3Cpath d='M9 4L15 4M12 4L12 20M9 20L15 20' fill='none' stroke='${encodedColor}' stroke-width='2.5' stroke-linecap='round'/%3E%3Cpath d='M9 4L15 4M12 4L12 20M9 20L15 20' fill='none' stroke='%23000' stroke-width='4' stroke-linecap='round' style='opacity:0.3'/%3E%3Cpath d='M9 4L15 4M12 4L12 20M9 20L15 20' fill='none' stroke='${encodedColor}' stroke-width='2' stroke-linecap='round'/%3E%3C/svg%3E")`;

    document.documentElement.style.setProperty(
      "--cursor-default",
      cursorDefault,
    );
    document.documentElement.style.setProperty(
      "--cursor-pointer",
      cursorPointer,
    );
    document.documentElement.style.setProperty("--cursor-text", cursorText);
  };

  // Initialize theme and accent on load
  if (browser) {
    const initialTheme = getInitialTheme();
    const initialAccent = getInitialAccent();
    applyTheme(initialTheme, initialAccent);

    // Listen for system theme changes
    window
      .matchMedia("(prefers-color-scheme: light)")
      .addEventListener("change", (e) => {
        const stored = localStorage.getItem(THEME_KEY);
        if (!stored) {
          const newTheme = e.matches ? "light" : "dark";
          set(newTheme);
          applyTheme(newTheme, get(accentStore));
        }
      });
  }

  return {
    subscribe,
    accent: { subscribe: accentStore.subscribe },
    toggle: () => {
      update((current) => {
        const newTheme = current === "dark" ? "light" : "dark";
        applyTheme(newTheme, get(accentStore));
        return newTheme;
      });
    },
    setTheme: (theme: Theme) => {
      set(theme);
      applyTheme(theme, get(accentStore));
    },
    setAccent: (accent: string) => {
      const currentTheme = get({ subscribe });
      applyAccent(accent, currentTheme);
    },
    getAccent: (): string => {
      return get(accentStore);
    },
    resetAccent: () => {
      const currentTheme = get({ subscribe });
      const defaultAccent = accentPresets[0].value;
      applyAccent(defaultAccent, currentTheme);
    },
    isDark: () => {
      let current: Theme = "dark";
      subscribe((v) => (current = v))();
      return current === "dark";
    },
    // Get the current effective accent (with light theme adjustment)
    getEffectiveAccent: (): string => {
      const currentTheme = get({ subscribe });
      const accent = get(accentStore);

      if (currentTheme === "light") {
        const preset = accentPresets.find(
          (p) => p.value.toLowerCase() === accent.toLowerCase(),
        );
        return preset?.light || darkenColor(accent, 25);
      }

      return accent;
    },
  };
}

export const theme = createThemeStore();

// Derived store for reactive accent color access
export const accentColor = derived(
  [{ subscribe: theme.subscribe }, theme.accent],
  ([$theme, $accent]) => ({
    theme: $theme,
    accent: $accent,
    effectiveAccent: theme.getEffectiveAccent(),
  }),
);
