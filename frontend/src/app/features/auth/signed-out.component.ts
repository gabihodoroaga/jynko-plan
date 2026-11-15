import { Component, ElementRef, OnInit, ViewChild, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService, DevAccountPreset } from '../../core/services/auth.service';

@Component({
  selector: 'app-signed-out',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <main class="min-h-dvh w-full bg-slate-50 flex flex-col items-center justify-center p-4 sm:p-6 text-slate-900">
      <div data-testid="signed-out-card" class="w-full max-w-md bg-white border border-slate-200 rounded-2xl shadow-[0_12px_32px_rgba(15,23,42,0.08)] overflow-hidden">
        <!-- Brand & Heading -->
        <div class="px-6 pt-6 pb-4 border-b border-slate-100 text-center">
          <div class="inline-flex items-center justify-center gap-2 mb-3">
            <span
              class="w-8 h-8 rounded-xl text-white flex items-center justify-center shadow-sm"
              style="background: linear-gradient(135deg, #1a73e8 0%, #6366f1 52%, #a855f7 100%)"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none">
                <path
                  d="M12 2.5 20.5 7.5v9L12 21.5 3.5 16.5v-9L12 2.5Z"
                  stroke="currentColor"
                  stroke-width="2.2"
                  stroke-linejoin="round"
                />
                <path
                  d="M12 7v10M7.5 9.5 12 12l4.5-2.5"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </span>
            <span class="text-base font-bold tracking-tight text-slate-900">Jynko</span>
          </div>

          <h1 class="text-lg font-semibold text-slate-900">
            {{ isSwitchMode() ? 'Choose an account' : 'You are signed out' }}
          </h1>
          @if (authService.lastSignedOutUser(); as prev) {
            <p class="mt-1 text-xs text-slate-500">
              Previous session: <span class="font-medium text-slate-700">{{ prev.name }}</span>
              @if (prev.email && prev.email !== prev.name) {
                <span> ({{ prev.email }})</span>
              }
            </p>
          } @else {
            <p class="mt-1 text-xs text-slate-500">
              Select how you would like to sign in to Jynko.
            </p>
          }
        </div>

        @if (!authService.isDevMode) {
          <!-- Production Google Cloud Identity-Aware Proxy (IAP) Flow -->
          <div class="p-6 space-y-4">
            <button
              type="button"
              (click)="continueWithIAP()"
              aria-label="Sign in with Google Cloud IAP"
              class="w-full min-h-11 px-4 py-2.5 rounded-xl bg-sky-600 hover:bg-sky-700 text-white text-sm font-medium transition-colors shadow-sm flex items-center justify-center gap-2 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-600"
            >
              <svg class="w-4 h-4 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M15 3h4a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-4M10 17l5-5-5-5M15 12H3" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
              <span>{{ isSwitchMode() ? 'Continue to Google Account Chooser' : 'Sign in with Google' }}</span>
            </button>

            <div class="rounded-xl bg-slate-50 border border-slate-200 p-3.5 space-y-2.5">
              <div class="text-xs font-semibold text-slate-800">
                Want to switch to a different Google account?
              </div>
              <p class="text-[11px] leading-relaxed text-slate-600">
                Your Jynko IAP session cookie has been cleared. If multiple Google accounts are active in this browser, clicking above opens the Google Account Chooser. If only one Google account is signed in, add another account or sign out of Google first so Google does not auto-select it:
              </p>
              <div class="flex flex-wrap items-center gap-2 pt-0.5">
                <a
                  href="https://accounts.google.com/AddSession"
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label="Add another Google account"
                  class="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-white border border-slate-300 hover:bg-slate-100 text-xs font-medium text-slate-700 transition-colors"
                >
                  <span>Add Google account</span>
                  <span aria-hidden="true">↗</span>
                </a>
                <a
                  href="https://accounts.google.com/Logout"
                  target="_blank"
                  rel="noopener noreferrer"
                  aria-label="Sign out of Google in browser"
                  class="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg bg-white border border-slate-300 hover:bg-slate-100 text-xs font-medium text-slate-700 transition-colors"
                >
                  <span>Sign out of Google</span>
                  <span aria-hidden="true">↗</span>
                </a>
              </div>
            </div>
          </div>
        } @else {
          <!-- Local Development (ENV=dev) Persona & Custom Header Identity Chooser -->
          <div class="p-5 space-y-5">
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <span class="text-[11px] font-semibold uppercase tracking-wider text-slate-500">
                  Choose an Account (Local Dev)
                </span>
                <span class="text-[10px] px-2 py-0.5 rounded-full bg-amber-50 border border-amber-200 text-amber-700 font-medium">
                  ENV=dev
                </span>
              </div>

              <div class="divide-y divide-slate-200 border border-slate-200 rounded-xl overflow-hidden bg-white">
                @for (acc of authService.availableDevAccounts(); track acc.email) {
                  <button
                    type="button"
                    (click)="selectDevAccount(acc)"
                    [attr.aria-label]="'Sign in as ' + acc.email"
                    class="w-full px-3.5 py-2.5 flex items-center gap-3 text-left hover:bg-slate-50 transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-sky-600"
                  >
                    <span
                      class="w-8 h-8 rounded-full text-white text-xs font-bold flex items-center justify-center shrink-0"
                      [style.backgroundColor]="acc.avatar_color || '#ea580c'"
                    >
                      {{ (acc.name || acc.email).charAt(0).toUpperCase() }}
                    </span>
                    <span class="min-w-0 flex-1">
                      <span class="block text-xs font-semibold text-slate-900 truncate">{{ acc.name }}</span>
                      <span class="block text-[11px] text-slate-500 truncate">{{ acc.email }}</span>
                    </span>
                    @if (acc.badge) {
                      <span class="text-[10px] px-2 py-0.5 rounded-full bg-slate-100 border border-slate-200 text-slate-600 font-medium shrink-0">
                        {{ acc.badge }}
                      </span>
                    }
                  </button>
                }
              </div>
            </div>

            <!-- Custom Dev Identity Form (sets X-Dev-User-* headers, no passwords) -->
            <form
              aria-labelledby="custom-dev-user-heading"
              (submit)="submitCustomIdentity($event)"
              novalidate
              class="pt-3 border-t border-slate-200 space-y-3"
            >
              <h2 id="custom-dev-user-heading" class="text-xs font-semibold text-slate-800">
                Use another development account
              </h2>

              <div class="flex flex-col gap-1">
                <label for="signed-out-dev-name" class="text-xs font-medium text-slate-700">
                  Display name (optional)
                </label>
                <input
                  id="signed-out-dev-name"
                  name="name"
                  type="text"
                  [(ngModel)]="customName"
                  autocomplete="name"
                  enterkeyhint="next"
                  maxlength="120"
                  class="w-full px-3 py-2 rounded-lg border border-slate-300 bg-white text-sm text-slate-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 focus-visible:border-sky-500"
                />
              </div>

              <div class="flex flex-col gap-1">
                <label for="signed-out-dev-email" class="text-xs font-medium text-slate-700">
                  Email address <span aria-hidden="true" class="text-rose-600">*</span>
                </label>
                <span id="signed-out-email-hint" class="text-[11px] text-slate-500">
                  Format: you&#64;company.com (forwarded via X-Dev-User-Email)
                </span>
                <input
                  #customEmailInput
                  id="signed-out-dev-email"
                  name="email"
                  type="email"
                  required
                  [(ngModel)]="customEmail"
                  (blur)="onEmailBlur()"
                  (input)="onEmailInput()"
                  autocomplete="email"
                  autocorrect="off"
                  autocapitalize="none"
                  spellcheck="false"
                  enterkeyhint="done"
                  [attr.aria-invalid]="emailError() ? 'true' : 'false'"
                  [attr.aria-describedby]="
                    emailError()
                      ? 'signed-out-email-hint signed-out-email-error'
                      : 'signed-out-email-hint'
                  "
                  class="w-full px-3 py-2 rounded-lg border bg-white text-sm text-slate-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500"
                  [ngClass]="
                    emailError()
                      ? 'border-rose-500 bg-rose-50/40'
                      : 'border-slate-300 focus-visible:border-sky-500'
                  "
                />
                @if (emailError()) {
                  <div id="signed-out-email-error" class="text-xs text-rose-600 flex items-center gap-1 mt-0.5">
                    <span role="img" aria-label="Error">⚠</span>
                    <span>{{ emailError() }}</span>
                  </div>
                }
              </div>

              <button
                type="submit"
                [disabled]="isSubmitting()"
                class="w-full min-h-10 px-4 py-2 rounded-lg bg-sky-600 hover:bg-sky-700 disabled:opacity-60 text-white text-xs font-medium transition-colors shadow-sm focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-600"
              >
                Continue with this account
              </button>
            </form>
          </div>
        }
      </div>
    </main>
  `,
})
export class SignedOutComponent implements OnInit {
  readonly authService = inject(AuthService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  @ViewChild('customEmailInput') customEmailInputRef?: ElementRef<HTMLInputElement>;

  readonly mode = signal<'signout' | 'switch'>('signout');
  readonly isSwitchMode = computed(() => this.mode() === 'switch');

  customName = '';
  customEmail = '';
  private emailEdited = false;
  readonly emailError = signal<string | null>(null);
  readonly isSubmitting = signal<boolean>(false);

  ngOnInit(): void {
    this.route.queryParamMap.subscribe((params) => {
      this.mode.set(params.get('mode') === 'switch' ? 'switch' : 'signout');
    });
  }

  continueWithIAP(): void {
    const redirected = this.authService.signInWithIAP(true);
    if (!redirected) {
      this.router.navigate(['/home']);
    }
  }

  selectDevAccount(account: DevAccountPreset): void {
    this.authService.switchDevUser(account);
    this.router.navigate(['/home']);
  }

  onEmailInput(): void {
    this.emailEdited = true;
    if (this.emailError() && this.isValidEmail(this.customEmail)) {
      this.emailError.set(null);
    }
  }

  onEmailBlur(): void {
    if (!this.emailEdited) return;
    if (!this.isValidEmail(this.customEmail)) {
      this.emailError.set('Enter a valid email address in the format you@company.com.');
    } else {
      this.emailError.set(null);
    }
  }

  submitCustomIdentity(event: Event): void {
    event.preventDefault();
    this.emailEdited = true;
    if (!this.isValidEmail(this.customEmail)) {
      this.emailError.set('Enter a valid email address in the format you@company.com.');
      this.customEmailInputRef?.nativeElement.focus();
      return;
    }
    this.emailError.set(null);
    this.isSubmitting.set(true);
    this.authService.switchDevUser({
      email: this.customEmail,
      name: this.customName,
    });
    this.router.navigate(['/home']);
  }

  private isValidEmail(val: string): boolean {
    const trimmed = val.trim();
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(trimmed);
  }
}
