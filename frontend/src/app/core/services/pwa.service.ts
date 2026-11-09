import { Injectable, inject, signal } from '@angular/core';
import { SwUpdate, VersionEvent } from '@angular/service-worker';
import { SettingsService } from './settings.service';

interface BeforeInstallPromptEvent extends Event {
  readonly platforms: string[];
  readonly userChoice: Promise<{ outcome: 'accepted' | 'dismissed'; platform: string }>;
  prompt(): Promise<void>;
}

@Injectable({
  providedIn: 'root',
})
export class PwaService {
  private readonly swUpdate = inject(SwUpdate, { optional: true });
  private readonly settingsService = inject(SettingsService);

  private deferredInstallPrompt: BeforeInstallPromptEvent | null = null;
  private updateIntervalId: ReturnType<typeof setInterval> | null = null;

  readonly canInstall = signal<boolean>(false);
  readonly isStandalone = signal<boolean>(this.detectStandalone());
  readonly isOnline = signal<boolean>(typeof navigator !== 'undefined' ? navigator.onLine : true);
  readonly swEnabled = signal<boolean>(Boolean(this.swUpdate?.isEnabled));
  readonly updateAvailable = signal<boolean>(false);
  readonly checkingForUpdate = signal<boolean>(false);
  readonly lastCheckedAt = signal<string | null>(null);
  readonly statusMessage = signal<string | null>(null);
  readonly showInstallGuide = signal<boolean>(false);

  constructor() {
    if (typeof window === 'undefined') {
      return;
    }

    window.addEventListener('beforeinstallprompt', (event: Event) => {
      event.preventDefault();
      this.deferredInstallPrompt = event as BeforeInstallPromptEvent;
      this.canInstall.set(true);
    });

    window.addEventListener('appinstalled', () => {
      this.deferredInstallPrompt = null;
      this.canInstall.set(false);
      this.isStandalone.set(true);
      this.showInstallGuide.set(false);
      this.statusMessage.set('Jynko is now installed as a standalone app.');
    });

    window.addEventListener('online', () => this.isOnline.set(true));
    window.addEventListener('offline', () => this.isOnline.set(false));

    if (typeof window.matchMedia === 'function') {
      const mql = window.matchMedia('(display-mode: standalone)');
      const onChange = () => this.isStandalone.set(this.detectStandalone());
      if (typeof mql.addEventListener === 'function') {
        mql.addEventListener('change', onChange);
      }
    }

    if (this.swUpdate?.isEnabled) {
      this.swUpdate.versionUpdates.subscribe((evt: VersionEvent) => {
        if (evt.type === 'VERSION_READY') {
          this.updateAvailable.set(true);
        }
      });

      // Periodically poll for updates in background if pwaAutoUpdate is enabled (every 30 minutes)
      this.updateIntervalId = setInterval(() => {
        if (this.settingsService.generalSettings().pwaAutoUpdate && this.isOnline()) {
          void this.checkForUpdates(true);
        }
      }, 30 * 60 * 1000);
    }
  }

  private detectStandalone(): boolean {
    if (typeof window === 'undefined') {
      return false;
    }
    const mediaStandalone =
      typeof window.matchMedia === 'function' &&
      (window.matchMedia('(display-mode: standalone)').matches ||
        window.matchMedia('(display-mode: window-controls-overlay)').matches);
    const iosStandalone = Boolean((navigator as unknown as { standalone?: boolean }).standalone);
    return mediaStandalone || iosStandalone;
  }

  getInstallInstructions(): string {
    if (typeof navigator === 'undefined') {
      return 'Use your browser menu to install this app to your device.';
    }
    const ua = navigator.userAgent || '';
    if (/iPad|iPhone|iPod/i.test(ua)) {
      return 'On iOS/iPadOS Safari: tap the Share button in the browser bar, then choose "Add to Home Screen".';
    }
    if (/Safari/i.test(ua) && !/Chrome|Chromium|Edg/i.test(ua)) {
      return 'On macOS Safari: open File → "Add to Dock..." from the top menu bar.';
    }
    return 'Open your browser menu (⋮ or address bar install icon) and select "Install Jynko" or "Add to Home screen".';
  }

  async promptInstall(): Promise<'accepted' | 'dismissed' | 'guide'> {
    if (!this.deferredInstallPrompt) {
      this.showInstallGuide.set(true);
      return 'guide';
    }
    await this.deferredInstallPrompt.prompt();
    const choice = await this.deferredInstallPrompt.userChoice;
    if (choice.outcome === 'accepted') {
      this.deferredInstallPrompt = null;
      this.canInstall.set(false);
    }
    return choice.outcome;
  }

  async checkForUpdates(silent = false): Promise<boolean> {
    if (!this.swUpdate?.isEnabled) {
      if (!silent) {
        this.lastCheckedAt.set(new Date().toLocaleTimeString());
        this.statusMessage.set(
          'Service worker is disabled in local development builds; production builds check ngsw.json automatically.'
        );
      }
      return false;
    }

    this.checkingForUpdate.set(true);
    try {
      const found = await this.swUpdate.checkForUpdate();
      this.lastCheckedAt.set(new Date().toLocaleTimeString());
      if (found) {
        this.updateAvailable.set(true);
        if (!silent) {
          this.statusMessage.set('A new version of Jynko is ready to activate.');
        }
      } else if (!silent) {
        this.statusMessage.set('Jynko is up to date.');
      }
      return found;
    } catch {
      if (!silent) {
        this.statusMessage.set('Unable to check for updates right now.');
      }
      return false;
    } finally {
      this.checkingForUpdate.set(false);
    }
  }

  async applyUpdateAndReload(): Promise<void> {
    if (this.swUpdate?.isEnabled) {
      try {
        await this.swUpdate.activateUpdate();
      } catch {}
    }
    if (typeof window !== 'undefined') {
      window.location.reload();
    }
  }

  dismissUpdateBanner(): void {
    this.updateAvailable.set(false);
  }

  async clearCacheAndReload(): Promise<void> {
    if (typeof window === 'undefined') {
      return;
    }
    try {
      if ('caches' in window) {
        const keys = await window.caches.keys();
        await Promise.all(keys.map((k) => window.caches.delete(k)));
      }
      if ('serviceWorker' in navigator) {
        const regs = await navigator.serviceWorker.getRegistrations();
        await Promise.all(regs.map((r) => r.unregister()));
      }
    } catch {}
    window.location.reload();
  }
}
