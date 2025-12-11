import { writable } from 'svelte/store';

type Theme = 'dark' | 'light';

// Check if we're in a browser environment
const browser = typeof window !== 'undefined';

function createThemeStore() {
    // Get initial theme from localStorage or system preference
    const getInitialTheme = (): Theme => {
        if (!browser) return 'dark';
        
        const stored = localStorage.getItem('rexec-theme');
        if (stored === 'light' || stored === 'dark') {
            return stored;
        }
        
        // Check system preference
        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
            return 'light';
        }
        
        return 'dark';
    };

    const { subscribe, set, update } = writable<Theme>(getInitialTheme());

    // Apply theme to document
    const applyTheme = (theme: Theme) => {
        if (!browser) return;
        
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('rexec-theme', theme);
    };

    // Initialize theme on load
    if (browser) {
        const initial = getInitialTheme();
        applyTheme(initial);
        
        // Listen for system theme changes
        window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', (e) => {
            const stored = localStorage.getItem('rexec-theme');
            if (!stored) {
                const newTheme = e.matches ? 'light' : 'dark';
                set(newTheme);
                applyTheme(newTheme);
            }
        });
    }

    return {
        subscribe,
        toggle: () => {
            update(current => {
                const newTheme = current === 'dark' ? 'light' : 'dark';
                applyTheme(newTheme);
                return newTheme;
            });
        },
        setTheme: (theme: Theme) => {
            set(theme);
            applyTheme(theme);
        },
        isDark: () => {
            let current: Theme = 'dark';
            subscribe(v => current = v)();
            return current === 'dark';
        }
    };
}

export const theme = createThemeStore();
