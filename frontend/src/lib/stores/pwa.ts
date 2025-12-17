import { writable, derived, get } from 'svelte/store';

interface BeforeInstallPromptEvent extends Event {
  readonly platforms: string[];
  readonly userChoice: Promise<{
    outcome: 'accepted' | 'dismissed';
    platform: string;
  }>;
  prompt(): Promise<void>;
}

const INSTALL_DISMISSED_KEY = 'rexec_install_dismissed';

// Store for the deferred install prompt
const deferredPrompt = writable<BeforeInstallPromptEvent | null>(null);

// Store to track if app is installed
const isInstalled = writable(false);

// Store to track if user has dismissed the install prompt
const installDismissed = writable(false);

// Load dismissed state from localStorage
function loadDismissedState(): boolean {
  if (typeof window === 'undefined') return false;
  try {
    return localStorage.getItem(INSTALL_DISMISSED_KEY) === 'true';
  } catch {
    return false;
  }
}

// Save dismissed state to localStorage
function saveDismissedState() {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(INSTALL_DISMISSED_KEY, 'true');
  } catch {
    // Ignore localStorage errors
  }
}

// Derived store to check if install is available
// Only show if: prompt available, not installed, and user hasn't dismissed it before
export const canInstall = derived(
  [deferredPrompt, isInstalled, installDismissed],
  ([$deferredPrompt, $isInstalled, $installDismissed]) =>
    $deferredPrompt !== null && !$isInstalled && !$installDismissed
);

// Initialize install prompt handling
export function initInstallPrompt() {
  // Load dismissed state from localStorage
  installDismissed.set(loadDismissedState());

  // Check if already installed (standalone mode)
  if (window.matchMedia('(display-mode: standalone)').matches) {
    isInstalled.set(true);
    return;
  }

  // Listen for the beforeinstallprompt event
  window.addEventListener('beforeinstallprompt', (e) => {
    e.preventDefault();
    deferredPrompt.set(e as BeforeInstallPromptEvent);
    console.log('PWA: Install prompt captured');
  });

  // Listen for successful installation
  window.addEventListener('appinstalled', () => {
    deferredPrompt.set(null);
    isInstalled.set(true);
    console.log('Rexec app was installed');
  });
}

// Trigger the install prompt
export async function promptInstall(): Promise<boolean> {
  const prompt = get(deferredPrompt);

  if (!prompt) {
    return false;
  }

  try {
    await prompt.prompt();
    const { outcome } = await prompt.userChoice;
    
    if (outcome === 'accepted') {
      deferredPrompt.set(null);
      return true;
    } else {
      // User dismissed - never show again
      installDismissed.set(true);
      saveDismissedState();
      deferredPrompt.set(null);
      return false;
    }
  } catch (error) {
    console.error('Error showing install prompt:', error);
    return false;
  }
}

// Dismiss install prompt without showing native prompt (e.g., clicking X button)
export function dismissInstallPrompt() {
  installDismissed.set(true);
  saveDismissedState();
  deferredPrompt.set(null);
}

// Export stores for direct subscription
export { deferredPrompt, isInstalled, installDismissed };
