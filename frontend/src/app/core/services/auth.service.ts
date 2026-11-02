import { Injectable, computed, inject, signal } from '@angular/core';
import { ApiService } from './api.service';
import { UserContext } from '../models/project.model';

const PROFILE_STORAGE_PREFIX = 'jynko.userProfile.';

@Injectable({
  providedIn: 'root',
})
export class AuthService {
  private readonly api = inject(ApiService);

  readonly currentUser = signal<UserContext>({
    user_id: 'dev-user-1',
    email: 'dev@jynko.local',
    name: 'Local Developer',
    avatar_color: '#ea580c',
  });

  readonly userInitial = computed<string>(() => {
    const u = this.currentUser();
    const base = (u.name || u.email || 'U').trim();
    return base.charAt(0).toUpperCase();
  });

  readonly avatarColor = computed<string>(() => {
    return this.currentUser().avatar_color || '#ea580c';
  });

  loadCurrentUser(): void {
    this.api.get<UserContext>('/auth/me').subscribe({
      next: (user) => {
        const merged = this.mergeStoredProfile(user);
        this.currentUser.set(merged);
      },
      error: () => {
        const fallback = this.mergeStoredProfile(this.currentUser());
        this.currentUser.set(fallback);
      },
    });
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
        },
      });
  }

  signOut(): void {
    // Behind Google Cloud IAP, signing out redirects to the IAP clear-login endpoint;
    // in local development, reload or navigate to /home.
    if (typeof window !== 'undefined') {
      const host = window.location.hostname;
      if (host !== 'localhost' && host !== '127.0.0.1') {
        window.location.href = '/_gcp_iap/clear_login_cookie';
        return;
      }
    }
  }

  private mergeStoredProfile(user: UserContext): UserContext {
    if (typeof localStorage === 'undefined') {
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
        name: parsed.name || user.name,
        avatar_url: parsed.avatar_url ?? user.avatar_url,
        avatar_color: parsed.avatar_color || user.avatar_color || '#ea580c',
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
}
