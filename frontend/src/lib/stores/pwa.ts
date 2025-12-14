import { writable, derived } from 'svelte/store';

interface BeforeInstallPromptEvent extends Event {
  readonly platforms: string[];
  readonly userChoice: Promise<{
    outcome: 'accepted' | 'dismissed';
    platform: string;
  }>;
  prompt(): Promise<void>;
}

// Store for the deferred install prompt
const deferredPrompt = writable<BeforeInstallPromptEvent | null>(null);

// Store to track if app is installed
const isInstalled = writable(false);

// Derived store to check if install is available
export const canInstall = derived(
  [deferredPrompt, isInstalled],
  ([$deferredPrompt, $isInstalled]) => $deferredPrompt !== null && !$isInstalled
);

// Initialize install prompt handling
export function initInstallPrompt() {
  // Check if already installed (standalone mode)
  if (window.matchMedia('(display-mode: standalone)').matches) {
    isInstalled.set(true);
    return;
  }

  // Listen for the beforeinstallprompt event
  window.addEventListener('beforeinstallprompt', (e) => {
    e.preventDefault();
    deferredPrompt.set(e as BeforeInstallPromptEvent);
    console.log('PWA: Install prompt captured, Install App button should now be visible');
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
  let prompt: BeforeInstallPromptEvent | null = null;
  
  const unsubscribe = deferredPrompt.subscribe((value) => {
    prompt = value;
  });
  unsubscribe();

  if (!prompt) {
    return false;
  }

  try {
    await prompt.prompt();
    const { outcome } = await prompt.userChoice;
    
    if (outcome === 'accepted') {
      deferredPrompt.set(null);
      return true;
    }
    return false;
  } catch (error) {
    console.error('Error showing install prompt:', error);
    return false;
  }
}

// Export stores for direct subscription
export { deferredPrompt, isInstalled };
