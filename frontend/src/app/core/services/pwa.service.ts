import { Injectable, NgZone, inject, signal } from '@angular/core';
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
  private readonly ngZone = inject(NgZone);

  private deferredInstallPrompt: BeforeInstallPromptEvent | null = null;
  private updateIntervalId: ReturnType<typeof setInterval> | null = null;

  readonly canInstall = signal<boolean>(false);
  readonly isStandalone = signal<boolean>(this.detectStandalone());
  readonly isOnline = signal<boolean>(typeof navigator !== 'undefined' ? navigator.onLine : true);
  readonly swEnabled = signal<boolean>(Boolean(this.swUpdate?.isEnabled));
  readonly updateAvailable = signal<boolean>(false);
  readonly checkingForUpdate = signal<boolean>(false);
  readonly refreshing = signal<boolean>(false);
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

      this.swUpdate.unrecoverable.subscribe(() => {
        this.updateAvailable.set(true);
        this.statusMessage.set(
          'Application cache is out of date — refresh to load the latest version.'
        );
      });

      // Run outside Angular zone so the 30-minute timer does not block Angular stability
      // and delay registerWhenStable:30000 Service Worker registration.
      this.ngZone.runOutsideAngular(() => {
        this.updateIntervalId = setInterval(() => {
          if (this.settingsService.generalSettings().pwaAutoUpdate && this.isOnline()) {
            this.ngZone.run(() => {
              void this.checkForUpdates(true);
            });
          }
        }, 30 * 60 * 1000);
      });
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

  /**
   * Checks for and activates any available Service Worker update, bypasses stale cached HTML
   * when a Service Worker controls the page online, and reloads the application.
   */
  async refreshApp(): Promise<void> {
    if (this.refreshing()) {
      return;
    }
    this.refreshing.set(true);
    const startedAt = Date.now();
    let shouldBypassSw = false;

    if (this.isOnline()) {
      if (this.swUpdate?.isEnabled) {
        try {
          const res = await this.withTimeout(
            this.swUpdate.checkForUpdate(),
            4000,
            'timeout' as const
          );
          if (res === true || this.updateAvailable()) {
            await this.withTimeout(this.swUpdate.activateUpdate(), 2500, false);
          } else if (res === 'timeout') {
            shouldBypassSw = true;
          }
        } catch {
          shouldBypassSw = true;
        }
      }

      if (typeof navigator !== 'undefined' && 'serviceWorker' in navigator) {
        await this.withTimeout(
          navigator.serviceWorker.getRegistration().then((r) => r?.update()),
          1500,
          undefined
        ).catch(() => {});
        if (navigator.serviceWorker.controller) {
          shouldBypassSw = true;
        }
      }
    }

    const elapsed = Date.now() - startedAt;
    if (elapsed < 180) {
      await new Promise((resolve) => setTimeout(resolve, 180 - elapsed));
    }

    this.performPageReload(shouldBypassSw && this.isOnline());
  }

  protected performPageReload(bypassServiceWorker: boolean): void {
    if (typeof window === 'undefined') {
      return;
    }
    if (bypassServiceWorker) {
      try {
        const target = new URL(window.location.href);
        target.searchParams.set('ngsw-bypass', 'true');
        window.location.replace(target.toString());
        return;
      } catch {}
    }
    window.location.reload();
  }

  private async withTimeout<T, F>(promise: Promise<T>, ms: number, fallback: F): Promise<T | F> {
    let timerId: ReturnType<typeof setTimeout> | undefined;
    try {
      return await Promise.race([
        promise,
        new Promise<F>((resolve) => {
          timerId = setTimeout(() => resolve(fallback), ms);
        }),
      ]);
    } finally {
      if (timerId !== undefined) {
        clearTimeout(timerId);
      }
    }
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
