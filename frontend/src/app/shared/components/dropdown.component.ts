import {
  Component,
  ElementRef,
  EventEmitter,
  HostListener,
  Input,
  Output,
  inject,
  signal,
} from '@angular/core';
import { CommonModule } from '@angular/common';

export interface DropdownOption {
  value: string;
  label: string;
  dotColor?: string;
  badge?: string;
}

@Component({
  selector: 'app-dropdown',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="relative inline-block text-left" [ngClass]="fullWidth ? 'w-full' : ''">
      <!-- Styled Trigger Button -->
      <button
        type="button"
        [disabled]="disabled"
        (click)="toggleOpen($event)"
        [attr.aria-label]="ariaLabel || placeholder"
        [attr.aria-expanded]="isOpen()"
        class="flex items-center justify-between gap-2 bg-white border border-slate-200 hover:bg-slate-50 hover:border-slate-300 rounded-md text-slate-700 transition-colors focus:outline-none focus:border-slate-300 disabled:opacity-50 disabled:pointer-events-none select-none"
        [ngClass]="[
          fullWidth ? 'w-full' : '',
          maxWidth ? maxWidth : '',
          size === 'sm' ? 'px-2 py-1 text-[11px]' : 'px-2.5 py-1.5 text-xs'
        ]"
      >
        <span class="flex items-center gap-1.5 truncate">
          @if (!isShowingEmptyLabel() && selectedOption()?.dotColor; as dot) {
            <span class="w-2 h-2 rounded-full shrink-0" [style.backgroundColor]="dot"></span>
          }
          <span
            class="truncate"
            [ngClass]="isShowingEmptyLabel() ? 'text-slate-400 font-normal' : 'font-medium text-slate-700'"
          >
            {{ triggerDisplayLabel() }}
          </span>
        </span>

        <!-- Chevron Icon -->
        <svg
          class="w-3.5 h-3.5 text-slate-400 shrink-0 transition-transform duration-150"
          [ngClass]="isOpen() ? 'rotate-180 text-slate-600' : ''"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
        >
          <path d="m6 9 6 6 6-6" />
        </svg>
      </button>

      <!-- Styled Floating Options Menu -->
      @if (isOpen()) {
        <div
          role="listbox"
          (click)="$event.stopPropagation()"
          class="absolute z-50 mt-1 min-w-[168px] max-h-60 overflow-y-auto rounded-lg bg-white border border-slate-200 shadow-[0_8px_24px_rgba(15,23,42,0.10)] py-1 text-xs"
          [ngClass]="[
            fullWidth ? 'w-full' : '',
            align === 'right' ? 'right-0' : 'left-0',
            dropUp ? 'bottom-full mb-1 mt-0' : ''
          ]"
        >
          @for (opt of options; track opt.value) {
            <button
              type="button"
              role="option"
              [attr.aria-selected]="opt.value === value"
              (click)="selectOption(opt.value, $event)"
              class="w-full flex items-center justify-between gap-2.5 px-2.5 py-1.5 text-left transition-colors"
              [ngClass]="
                opt.value === value
                  ? 'bg-sky-50/80 text-sky-700 font-medium'
                  : 'text-slate-700 hover:bg-slate-100/80'
              "
            >
              <span class="flex items-center gap-2 min-w-0 truncate">
                @if (opt.dotColor) {
                  <span class="w-2 h-2 rounded-full shrink-0" [style.backgroundColor]="opt.dotColor"></span>
                }
                <span class="truncate text-[12px]">{{ opt.label }}</span>
                @if (opt.badge) {
                  <span class="px-1.5 py-0.2 rounded bg-slate-100 border border-slate-200 font-mono text-[10px] text-slate-500">
                    {{ opt.badge }}
                  </span>
                }
              </span>

              @if (opt.value === value) {
                <svg
                  class="w-3.5 h-3.5 text-sky-600 shrink-0"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.2"
                >
                  <path d="M20 6 9 17l-5-5" />
                </svg>
              }
            </button>
          }
        </div>
      }
    </div>
  `,
})
export class DropdownComponent {
  private readonly hostEl = inject(ElementRef<HTMLElement>);

  @Input() options: DropdownOption[] = [];
  @Input() value = '';
  @Input() placeholder = 'Select...';
  @Input() emptyLabel = '';
  @Input() emptyValues: string[] = [''];
  @Input() ariaLabel = '';
  @Input() disabled = false;
  @Input() fullWidth = false;
  @Input() maxWidth = '';
  @Input() size: 'sm' | 'md' = 'md';
  @Input() align: 'left' | 'right' = 'left';
  @Input() dropUp = false;

  @Output() valueChange = new EventEmitter<string>();

  readonly isOpen = signal<boolean>(false);

  selectedOption(): DropdownOption | undefined {
    return this.options.find((o) => o.value === this.value);
  }

  isShowingEmptyLabel(): boolean {
    if (!this.emptyLabel) return false;
    return this.emptyValues.includes(this.value) || !this.selectedOption();
  }

  triggerDisplayLabel(): string {
    if (this.isShowingEmptyLabel()) {
      return this.emptyLabel;
    }
    return this.selectedOption()?.label ?? (this.emptyLabel || this.placeholder);
  }

  toggleOpen(event: MouseEvent): void {
    event.stopPropagation();
    if (this.disabled) return;
    this.isOpen.update((v) => !v);
  }

  selectOption(nextValue: string, event: MouseEvent): void {
    event.stopPropagation();
    this.value = nextValue;
    this.valueChange.emit(nextValue);
    this.isOpen.set(false);
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    if (!this.isOpen()) return;
    const target = event.target as Node | null;
    if (target && !this.hostEl.nativeElement.contains(target)) {
      this.isOpen.set(false);
    }
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.isOpen()) {
      this.isOpen.set(false);
    }
  }
}
