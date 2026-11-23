import { Component, Input, OnDestroy, signal } from '@angular/core';

/**
 * Tiny icon-only button that copies the given raw text to the clipboard.
 * Shows a check icon briefly after a successful copy.
 */
@Component({
  selector: 'app-copy-button',
  standalone: true,
  template: `
    <button
      type="button"
      (click)="copy($event)"
      [attr.aria-label]="label"
      [attr.title]="copied() ? 'Copied!' : label"
      [attr.data-copied]="copied() ? 'true' : null"
      class="inline-flex items-center justify-center w-5 h-5 rounded bg-white/80 border border-slate-200 text-slate-400 hover:text-sky-600 hover:border-slate-300 transition-colors"
    >
      @if (copied()) {
        <svg class="w-3 h-3 text-emerald-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <path d="M20 6 9 17l-5-5" />
        </svg>
      } @else {
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
          <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
        </svg>
      }
    </button>
  `,
})
export class CopyButtonComponent implements OnDestroy {
  @Input() text: string | null | undefined = '';
  @Input() label = 'Copy to clipboard';

  readonly copied = signal(false);
  private resetTimer: ReturnType<typeof setTimeout> | null = null;

  async copy(event: Event): Promise<void> {
    event.stopPropagation();
    const value = this.text ?? '';
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(value);
      } else {
        this.fallbackCopy(value);
      }
    } catch {
      this.fallbackCopy(value);
    }
    this.copied.set(true);
    if (this.resetTimer) clearTimeout(this.resetTimer);
    this.resetTimer = setTimeout(() => this.copied.set(false), 1500);
  }

  ngOnDestroy(): void {
    if (this.resetTimer) clearTimeout(this.resetTimer);
  }

  private fallbackCopy(value: string): void {
    const ta = document.createElement('textarea');
    ta.value = value;
    ta.setAttribute('readonly', '');
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand('copy');
    } finally {
      document.body.removeChild(ta);
    }
  }
}
