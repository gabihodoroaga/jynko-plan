import { Component, OnInit, computed, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { AuthService } from '../../core/services/auth.service';

export interface AvatarColorOption {
  hex: string;
  label: string;
}

@Component({
  selector: 'app-profile',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="w-full p-6 space-y-6 bg-slate-50 min-h-full">
      <!-- Page Header -->
      <div class="flex items-center justify-between border-b border-slate-200 pb-4">
        <div>
          <h1 class="text-base font-semibold text-slate-900">User Profile</h1>
          <p class="text-xs text-slate-500">
            Manage your personal display name and avatar. Email is managed by Google Cloud IAP.
          </p>
        </div>

        @if (savedNotice()) {
          <span
            class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-emerald-50 border border-emerald-200 text-emerald-700 text-xs font-medium"
          >
            <span>✓</span>
            <span>Profile saved</span>
          </span>
        }
      </div>

      <!-- Profile Card -->
      <form
        (ngSubmit)="saveProfile()"
        class="bg-white border border-slate-200 rounded-lg p-6 shadow-card space-y-6"
      >
        <!-- 1. Photo / Avatar Section -->
        <div class="flex flex-col sm:flex-row sm:items-center gap-5 pb-6 border-b border-slate-100">
          <!-- Avatar Preview Circle -->
          <div class="shrink-0">
            @if (avatarUrl().trim()) {
              <img
                [src]="avatarUrl().trim()"
                [alt]="displayName() || 'User avatar'"
                class="w-20 h-20 rounded-full object-cover border border-slate-200 shadow-sm"
              />
            } @else {
              <div
                class="w-20 h-20 rounded-full text-white text-3xl font-medium flex items-center justify-center shadow-sm select-none"
                [style.backgroundColor]="avatarColor()"
              >
                {{ previewInitial() }}
              </div>
            }
          </div>

          <!-- Avatar Controls -->
          <div class="flex-1 space-y-3">
            <div>
              <h2 class="text-xs font-semibold text-slate-900">Photo or Avatar</h2>
              <p class="text-[11px] text-slate-500">
                Upload a profile picture, provide an image URL, or choose an initial badge color.
              </p>
            </div>

            <div class="flex flex-wrap items-center gap-2">
              <label
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md border border-slate-200 bg-white hover:bg-slate-50 text-xs font-medium text-slate-700 cursor-pointer transition-colors"
              >
                <span>Upload photo</span>
                <input
                  type="file"
                  accept="image/*"
                  (change)="onFileSelected($event)"
                  aria-label="Upload profile photo"
                  class="hidden"
                />
              </label>

              @if (avatarUrl().trim()) {
                <button
                  type="button"
                  (click)="clearPhoto()"
                  class="px-2.5 py-1.5 rounded-md border border-rose-200 bg-rose-50 hover:bg-rose-100 text-xs font-medium text-rose-700 transition-colors"
                >
                  Remove photo
                </button>
              }
            </div>

            <div class="max-w-md">
              <label class="block text-[11px] font-medium text-slate-600 mb-1">
                Photo / Avatar Image URL <span class="font-normal text-slate-400">(optional)</span>
              </label>
              <input
                type="url"
                name="avatarUrl"
                [ngModel]="avatarUrl()"
                (ngModelChange)="avatarUrl.set($event)"
                aria-label="Profile avatar URL"
                placeholder="https://example.com/avatar.png"
                class="w-full px-3 py-1.5 text-xs bg-white border border-slate-200 rounded-md text-slate-800 focus:outline-none focus:border-slate-300"
              />
            </div>

            <!-- Initial Avatar Color Swatches -->
            <div class="flex items-center gap-2 pt-1">
              <span class="text-[11px] font-medium text-slate-500 mr-1">Avatar color:</span>
              @for (sw of colorOptions; track sw.hex) {
                <button
                  type="button"
                  (click)="avatarColor.set(sw.hex)"
                  [attr.aria-label]="'Select avatar color ' + sw.label"
                  [title]="sw.label"
                  class="w-5 h-5 rounded-full border transition-transform"
                  [style.backgroundColor]="sw.hex"
                  [ngClass]="
                    avatarColor() === sw.hex
                      ? 'ring-2 ring-offset-2 ring-sky-600 scale-110 border-white'
                      : 'border-slate-200 hover:scale-105'
                  "
                ></button>
              }
            </div>
          </div>
        </div>

        <!-- 2. Name & Email Fields -->
        <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
          <!-- Name -->
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1.5">Name</label>
            <input
              type="text"
              name="displayName"
              [ngModel]="displayName()"
              (ngModelChange)="displayName.set($event)"
              required
              aria-label="Profile display name"
              placeholder="Your full name"
              class="w-full px-3 py-2 text-xs bg-white border border-slate-200 rounded-md text-slate-900 focus:outline-none focus:border-slate-300"
            />
            <p class="mt-1 text-[11px] text-slate-400">
              Displayed on work items, comments, and wiki pages across your projects.
            </p>
          </div>

          <!-- Email (Read-only from Google Cloud IAP) -->
          <div>
            <div class="flex items-center justify-between mb-1.5">
              <label class="block text-xs font-medium text-slate-700">Email</label>
              <span
                class="px-2 py-0.5 rounded bg-slate-100 border border-slate-200 text-[10px] font-medium text-slate-600"
              >
                Read-only · Google Cloud IAP
              </span>
            </div>
            <input
              type="email"
              name="email"
              [value]="authService.currentUser().email"
              readonly
              disabled
              aria-label="Profile email"
              class="w-full px-3 py-2 text-xs bg-slate-100 border border-slate-200 rounded-md text-slate-500 font-mono cursor-not-allowed select-all"
            />
            <p class="mt-1 text-[11px] text-slate-400">
              Authenticated via <code class="font-mono">X-Goog-Authenticated-User-Email</code> header.
            </p>
          </div>
        </div>

        <!-- Save Action Footer -->
        <div class="flex items-center justify-end gap-2.5 pt-4 border-t border-slate-100">
          <button
            type="submit"
            aria-label="Save user profile"
            class="px-4 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium transition-colors"
          >
            Save Profile
          </button>
        </div>
      </form>
    </div>
  `,
})
export class ProfileComponent implements OnInit {
  readonly authService = inject(AuthService);

  readonly displayName = signal<string>('');
  readonly avatarUrl = signal<string>('');
  readonly avatarColor = signal<string>('#ea580c');
  readonly savedNotice = signal<boolean>(false);

  readonly colorOptions: AvatarColorOption[] = [
    { hex: '#ea580c', label: 'Orange' },
    { hex: '#0284c7', label: 'Sky Blue' },
    { hex: '#059669', label: 'Emerald' },
    { hex: '#4f46e5', label: 'Indigo' },
    { hex: '#e11d48', label: 'Rose' },
    { hex: '#334155', label: 'Slate' },
  ];

  readonly previewInitial = computed<string>(() => {
    const base = (this.displayName() || this.authService.currentUser().email || 'U').trim();
    return base.charAt(0).toUpperCase();
  });

  private initializedUserId = '';

  constructor() {
    effect(() => {
      const user = this.authService.currentUser();
      if (user.user_id && this.initializedUserId !== user.user_id) {
        this.initializedUserId = user.user_id;
        this.displayName.set(user.name);
        this.avatarUrl.set(user.avatar_url ?? '');
        this.avatarColor.set(user.avatar_color || '#ea580c');
      }
    });
  }

  ngOnInit(): void {
    const user = this.authService.currentUser();
    this.displayName.set(user.name);
    this.avatarUrl.set(user.avatar_url ?? '');
    this.avatarColor.set(user.avatar_color || '#ea580c');
  }

  onFileSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input?.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === 'string') {
        this.avatarUrl.set(reader.result);
      }
    };
    reader.readAsDataURL(file);
  }

  clearPhoto(): void {
    this.avatarUrl.set('');
  }

  saveProfile(): void {
    const trimmedName = this.displayName().trim();
    if (!trimmedName) return;
    this.authService.updateProfile({
      name: trimmedName,
      avatar_url: this.avatarUrl().trim(),
      avatar_color: this.avatarColor(),
    });
    this.savedNotice.set(true);
    setTimeout(() => this.savedNotice.set(false), 2500);
  }
}
