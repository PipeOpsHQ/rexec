import { mount } from 'svelte';
import App from './App.svelte';
import './styles/global.css';
import { registerSW } from 'virtual:pwa-register';
import { initInstallPrompt } from './lib/stores/pwa';

const app = mount(App, {
  target: document.getElementById('app')!,
});

// Initialize PWA install prompt handling
initInstallPrompt();

// Register service worker with auto-update
const updateSW = registerSW({
  onNeedRefresh() {
    // Show a prompt to update when new content is available
    if (confirm('New content available. Reload to update?')) {
      updateSW(true);
    }
  },
  onOfflineReady() {
    console.log('App ready to work offline');
  },
  onRegistered(r) {
    // Check for updates every hour
    if (r) {
      setInterval(() => {
        r.update();
      }, 60 * 60 * 1000);
    }
  },
  onRegisterError(error) {
    console.error('SW registration error:', error);
  },
});

export default app;
