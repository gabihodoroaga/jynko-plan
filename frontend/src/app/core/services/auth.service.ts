import { HttpErrorResponse } from '@angular/common/http';
import { Injectable, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { ApiService } from './api.service';
import { UserContext } from '../models/project.model';
import { environment } from '../../../environments/environment';

const PROFILE_STORAGE_PREFIX = 'jynko.userProfile.';
const SIGNED_OUT_SESSION_KEY = 'jynko.auth.signedOut';
const LAST_USER_SESSION_KEY = 'jynko.auth.lastUser';
const DEV_ACTIVE_USER_SESSION_KEY = 'jynko.dev.activeUser';
const DEV_CUSTOM_USERS_STORAGE_KEY = 'jynko.dev.customUsers';

export interface DevAccountPreset extends UserContext {
  badge?: string;
}

export interface AppRuntimeMeta {
  env: 'dev' | 'stg' | 'prd';
  build_sha?: string;
  build_branch?: string;
  allow_dev_auth?: boolean;
}

export const DEFAULT_DEV_ACCOUNTS: DevAccountPreset[] = [
  {
    user_id: 'dev-user-1',
    email: 'dev@jynko.local',
    name: 'Local Developer',
    avatar_color: '#ea580c',
    badge: 'Default Admin',
  },
  {
    user_id: 'user-alice',
    email: 'alice@jynko.local',
    name: 'Alice Chen',
    avatar_color: '#0284c7',
    badge: 'Member',
  },
  {
    user_id: 'user-bob',
    email: 'bob@jynko.local',
    name: 'Bob Miller',
    avatar_color: '#7c3aed',
    badge: 'Member',
  },
  {
    user_id: 'user-viewer',
    email: 'viewer@jynko.local',
    name: 'Sam Taylor',
    avatar_color: '#64748b',
    badge: 'Viewer',
  },
];

const AVATAR_PALETTE = ['#ea580c', '#0284c7', '#7c3aed', '#059669', '#db2777', '#4f46e5'];

const UNAUTHENTICATED_USER: UserContext = {
  user_id: '',
  email: '',
  name: '',
  avatar_color: '#64748b',
};

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly api = inject(ApiService);
  private readonly router = inject(Router);
  private readonly resetCallbacks: Array<() => void> = [];
  private readonly reloadCallbacks: Array<() => void> = [];

  readonly isDevMode = environment.devMode;
  readonly appMeta = signal<AppRuntimeMeta>({
    env: environment.devMode ? 'dev' : 'prd',
    allow_dev_auth: environment.devMode,
  });
  readonly isSignedOut = signal<boolean>(this.readInitialSignedOut());
  readonly lastSignedOutUser = signal<{ name: string; email: string } | null>(
    this.readLastSignedOutUser()
  );

  readonly currentUser = signal<UserContext>(this.readInitialUser());
  readonly customDevUsers = signal<DevAccountPreset[]>(this.readCustomDevUsers());

  readonly availableDevAccounts = computed<DevAccountPreset[]>(() => {
    const merged: DevAccountPreset[] = [...DEFAULT_DEV_ACCOUNTS];
    const seen = new Set(
      merged.flatMap((u) => [u.user_id.toLowerCase(), u.email.toLowerCase()])
    );
    for (const custom of this.customDevUsers()) {
      const uid = custom.user_id.toLowerCase();
      const email = custom.email.toLowerCase();
      if (!seen.has(uid) && !seen.has(email)) {
        seen.add(uid);
        seen.add(email);
        merged.push(custom);
      }
    }
    return merged;
  });

  readonly userInitial = computed<string>(() => {
    const u = this.currentUser();
    const base = (u.name || u.email || 'U').trim();
    return base.charAt(0).toUpperCase();
  });

  readonly avatarColor = computed<string>(() => {
    return this.currentUser().avatar_color || '#ea580c';
  });

  registerSessionLifecycle(hooks: { onReset?: () => void; onReload?: () => void }): void {
    if (hooks.onReset) {
      this.resetCallbacks.push(hooks.onReset);
    }
    if (hooks.onReload) {
      this.reloadCallbacks.push(hooks.onReload);
    }
  }

  loadCurrentUser(): void {
    if (this.isSignedOut()) {
      return;
    }
    this.loadAppMeta();
    this.api.get<UserContext>('/auth/me').subscribe({
      next: (user) => {
        const merged = this.mergeStoredProfile(user);
        this.currentUser.set(merged);
      },
      error: (err: unknown) => {
        if (environment.devMode) {
          const fallback = this.mergeStoredProfile(this.currentUser());
          this.currentUser.set(fallback);
          return;
        }
        this.currentUser.set({ ...UNAUTHENTICATED_USER });
        if (
          err instanceof HttpErrorResponse &&
          (err.status === 401 || err.status === 403) &&
          !this.isSignedOut()
        ) {
          this.isSignedOut.set(true);
          void this.router.navigate(['/signed-out']);
        }
      },
    });
  }

  loadAppMeta(): void {
    this.applyEnvironmentDocumentTitle(this.appMeta().env);
    this.api.get<AppRuntimeMeta>('/meta').subscribe({
      next: (meta) => {
        if (meta && (meta.env === 'dev' || meta.env === 'stg' || meta.env === 'prd')) {
          this.appMeta.set(meta);
          this.applyEnvironmentDocumentTitle(meta.env);
        }
      },
      error: () => {
        this.applyEnvironmentDocumentTitle(this.appMeta().env);
      },
    });
  }

  private applyEnvironmentDocumentTitle(env: 'dev' | 'stg' | 'prd'): void {
    if (typeof document === 'undefined') {
      return;
    }
    if (env === 'dev') {
      document.title = '[DEV] Jynko';
    } else if (env === 'stg') {
      document.title = '[STG] Jynko';
    } else {
      document.title = 'Jynko';
    }
  }

  updateProfile(patch: {
    name?: string;
    avatar_url?: string;
    avatar_color?: string;
  }): void {
    const curr = this.currentUser();
    const next: UserContext = {
      ...curr,
      name: patch.name !== undefined ? patch.name.trim() || curr.name : curr.name,
      avatar_url: patch.avatar_url !== undefined ? patch.avatar_url.trim() : curr.avatar_url,
      avatar_color: patch.avatar_color !== undefined ? patch.avatar_color : curr.avatar_color,
    };
    this.currentUser.set(next);
    this.saveStoredProfile(next);
    if (this.isDevMode) {
      this.saveActiveDevUser(next);
    }

    this.api
      .patch<UserContext>('/auth/me', {
        name: next.name,
        avatar_url: next.avatar_url ?? '',
        avatar_color: next.avatar_color ?? '#ea580c',
      })
      .subscribe({
        next: (updated) => {
          const synced: UserContext = {
            ...next,
            ...updated,
            avatar_color: updated.avatar_color || next.avatar_color || '#ea580c',
          };
          this.currentUser.set(synced);
          this.saveStoredProfile(synced);
          if (this.isDevMode) {
            this.saveActiveDevUser(synced);
          }
        },
      });
  }

  /**
   * Signs out of the active Jynko session, clears in-memory stores, and (behind IAP)
   * expires the GCP IAP session cookie in the background without triggering silent SSO auto-relogin.
   */
  signOut(): void {
    const curr = this.currentUser();
    if (curr.email || curr.name) {
      const summary = { name: curr.name || curr.email, email: curr.email };
      this.lastSignedOutUser.set(summary);
      if (typeof sessionStorage !== 'undefined') {
        try {
          sessionStorage.setItem(LAST_USER_SESSION_KEY, JSON.stringify(summary));
        } catch {
          // Ignore storage errors
        }
      }
    }

    this.isSignedOut.set(true);
    if (typeof sessionStorage !== 'undefined') {
      try {
        sessionStorage.setItem(SIGNED_OUT_SESSION_KEY, '1');
      } catch {
        // Ignore storage errors
      }
    }

    for (const reset of this.resetCallbacks) {
      reset();
    }

    if (!this.isDevMode && typeof window !== 'undefined' && typeof fetch === 'function') {
      const host = window.location.hostname;
      if (host !== 'localhost' && host !== '127.0.0.1') {
        fetch('/_gcp_iap/clear_login_cookie', {
          method: 'GET',
          credentials: 'same-origin',
          cache: 'no-store',
          redirect: 'manual',
        }).catch(() => {});
        fetch('/?gcp-iap-mode=CLEAR_LOGIN_COOKIE', {
          method: 'GET',
          credentials: 'same-origin',
          cache: 'no-store',
          redirect: 'manual',
        }).catch(() => {});
      }
    }
  }

  /**
   * Initiates re-authentication or account selection via Google Cloud IAP in production,
   * or restores the active dev persona in local development.
   * Returns true if a full-page browser navigation was initiated.
   */
  signInWithIAP(clearCookieFirst = true): boolean {
    this.clearSignedOutFlag();

    if (!this.isDevMode && typeof window !== 'undefined') {
      const host = window.location.hostname;
      if (host !== 'localhost' && host !== '127.0.0.1') {
        const target = clearCookieFirst
          ? '/?gcp-iap-mode=CLEAR_LOGIN_COOKIE&ngsw-bypass=true'
          : '/?ngsw-bypass=true';
        window.location.assign(target);
        return true;
      }
    }

    this.loadCurrentUser();
    for (const reload of this.reloadCallbacks) {
      reload();
    }
    return false;
  }

  /**
   * Switches the active local development identity (X-Dev-User-* headers) and reloads workspace data.
   * Strictly no-op outside environment.devMode.
   */
  switchDevUser(input: {
    user_id?: string;
    email: string;
    name?: string;
    avatar_color?: string;
  }): void {
    if (!this.isDevMode) {
      return;
    }
    const cleanEmail = input.email.trim().toLowerCase();
    if (!cleanEmail) {
      return;
    }
    const cleanName =
      (input.name || '').trim() ||
      cleanEmail
        .split('@')[0]
        .replace(/[._-]+/g, ' ')
        .replace(/\b\w/g, (c) => c.toUpperCase());

    const preset = DEFAULT_DEV_ACCOUNTS.find(
      (p) =>
        p.email.toLowerCase() === cleanEmail ||
        (!!input.user_id && p.user_id === input.user_id.trim())
    );

    const userId = (input.user_id || preset?.user_id || cleanEmail).trim();
    const color =
      input.avatar_color ||
      preset?.avatar_color ||
      AVATAR_PALETTE[Math.abs(this.hashString(cleanEmail)) % AVATAR_PALETTE.length];

    const nextUser = this.mergeStoredProfile({
      user_id: userId,
      email: cleanEmail,
      name: cleanName,
      avatar_color: color,
    });

    this.clearSignedOutFlag();
    this.saveActiveDevUser(nextUser);

    if (!preset) {
      this.rememberCustomDevUser({
        ...nextUser,
        badge: 'Custom Dev',
      });
    }

    for (const reset of this.resetCallbacks) {
      reset();
    }

    this.currentUser.set(nextUser);
    this.loadCurrentUser();
    for (const reload of this.reloadCallbacks) {
      reload();
    }
  }

  clearSignedOutFlag(): void {
    this.isSignedOut.set(false);
    if (typeof sessionStorage !== 'undefined') {
      try {
        sessionStorage.removeItem(SIGNED_OUT_SESSION_KEY);
      } catch {
        // Ignore storage errors
      }
    }
  }

  private readInitialSignedOut(): boolean {
    if (typeof window === 'undefined' || typeof sessionStorage === 'undefined') {
      return false;
    }
    try {
      const flagged = sessionStorage.getItem(SIGNED_OUT_SESSION_KEY) === '1';
      if (!flagged) {
        return false;
      }
      if (!window.location.pathname.startsWith('/signed-out')) {
        sessionStorage.removeItem(SIGNED_OUT_SESSION_KEY);
        return false;
      }
      return true;
    } catch {
      return false;
    }
  }

  private readLastSignedOutUser(): { name: string; email: string } | null {
    if (typeof sessionStorage === 'undefined') {
      return null;
    }
    try {
      const raw = sessionStorage.getItem(LAST_USER_SESSION_KEY);
      if (!raw) return null;
      const parsed = JSON.parse(raw) as { name?: string; email?: string };
      if (parsed && (parsed.email || parsed.name)) {
        return { name: parsed.name || parsed.email || '', email: parsed.email || '' };
      }
    } catch {
      // Ignore storage errors
    }
    return null;
  }

  private readInitialUser(): UserContext {
    if (!environment.devMode) {
      return { ...UNAUTHENTICATED_USER };
    }
    if (typeof sessionStorage !== 'undefined') {
      try {
        const raw = sessionStorage.getItem(DEV_ACTIVE_USER_SESSION_KEY);
        if (raw) {
          const parsed = JSON.parse(raw) as UserContext;
          if (parsed && parsed.user_id && parsed.email) {
            return {
              user_id: parsed.user_id,
              email: parsed.email,
              name: parsed.name || parsed.email,
              avatar_url: parsed.avatar_url,
              avatar_color: parsed.avatar_color || '#ea580c',
            };
          }
        }
      } catch {
        // Ignore storage errors
      }
    }
    return {
      user_id: 'dev-user-1',
      email: 'dev@jynko.local',
      name: 'Local Developer',
      avatar_color: '#ea580c',
    };
  }

  private saveActiveDevUser(user: UserContext): void {
    if (typeof sessionStorage === 'undefined') return;
    try {
      sessionStorage.setItem(DEV_ACTIVE_USER_SESSION_KEY, JSON.stringify(user));
    } catch {
      // Ignore storage errors
    }
  }

  private readCustomDevUsers(): DevAccountPreset[] {
    if (!environment.devMode || typeof localStorage === 'undefined') {
      return [];
    }
    try {
      const raw = localStorage.getItem(DEV_CUSTOM_USERS_STORAGE_KEY);
      if (!raw) return [];
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        return parsed.filter((u) => u && typeof u.email === 'string' && typeof u.user_id === 'string');
      }
    } catch {
      // Ignore storage errors
    }
    return [];
  }

  private rememberCustomDevUser(account: DevAccountPreset): void {
    const updated = [
      account,
      ...this.customDevUsers().filter(
        (u) => u.email.toLowerCase() !== account.email.toLowerCase()
      ),
    ].slice(0, 8);
    this.customDevUsers.set(updated);
    if (typeof localStorage !== 'undefined') {
      try {
        localStorage.setItem(DEV_CUSTOM_USERS_STORAGE_KEY, JSON.stringify(updated));
      } catch {
        // Ignore storage errors
      }
    }
  }

  private mergeStoredProfile(user: UserContext): UserContext {
    if (!user.user_id || typeof localStorage === 'undefined') {
      return {
        ...user,
        avatar_color: user.avatar_color || '#ea580c',
      };
    }
    try {
      const raw = localStorage.getItem(PROFILE_STORAGE_PREFIX + user.user_id);
      if (!raw) {
        return {
          ...user,
          avatar_color: user.avatar_color || '#ea580c',
        };
      }
      const parsed = JSON.parse(raw) as Partial<UserContext>;
      return {
        ...user,
        name: user.name || parsed.name || 'Authenticated User',
        avatar_url: user.avatar_url || parsed.avatar_url,
        avatar_color: user.avatar_color || parsed.avatar_color || '#ea580c',
      };
    } catch {
      return {
        ...user,
        avatar_color: user.avatar_color || '#ea580c',
      };
    }
  }

  private saveStoredProfile(user: UserContext): void {
    if (typeof localStorage === 'undefined') return;
    try {
      localStorage.setItem(
        PROFILE_STORAGE_PREFIX + user.user_id,
        JSON.stringify({
          name: user.name,
          avatar_url: user.avatar_url ?? '',
          avatar_color: user.avatar_color ?? '#ea580c',
        })
      );
    } catch {
      // Ignore storage errors
    }
  }

  private hashString(str: string): number {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      hash = (hash << 5) - hash + str.charCodeAt(i);
      hash |= 0;
    }
    return hash;
  }
}
