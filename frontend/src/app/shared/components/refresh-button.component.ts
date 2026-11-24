import { Component, EventEmitter, Input, Output } from '@angular/core';

/**
 * Icon-only refresh button placed on the left of each project page's
 * "+ Add ..." primary action (PLAN-12). It re-fetches the current page's
 * items from the server without navigating away.
 */
@Component({
  selector: 'app-refresh-button',
  standalone: true,
  template: `
    <button
      type="button"
      data-testid="page-refresh-btn"
      [attr.aria-label]="label"
      [attr.aria-busy]="refreshing"
      [title]="label"
      [disabled]="refreshing"
      (click)="refresh.emit()"
      class="w-7 h-7 rounded-md flex items-center justify-center text-slate-600 hover:text-slate-900 hover:bg-slate-100 border border-slate-200 bg-white focus:outline-none focus-visible:ring-2 focus-visible:ring-sky-500 transition-colors disabled:opacity-60 shrink-0"
    >
      <svg
        class="w-3.5 h-3.5"
        [class.animate-spin]="refreshing"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <path d="M21 12a9 9 0 1 1-2.64-6.36" />
        <path d="M21 3v6h-6" />
      </svg>
    </button>
  `,
})
export class RefreshButtonComponent {
  @Input() label = 'Refresh';
  @Input() refreshing = false;
  @Output() readonly refresh = new EventEmitter<void>();
}
