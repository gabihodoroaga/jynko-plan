import {
  AfterViewInit,
  Component,
  ElementRef,
  EventEmitter,
  Input,
  OnChanges,
  Output,
  SimpleChanges,
  ViewChild,
} from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-modal',
  standalone: true,
  imports: [CommonModule],
  template: `
    <dialog
      #dialogEl
      [attr.closedby]="'any'"
      [attr.aria-labelledby]="titleId"
      (close)="onNativeClose()"
      class="w-full rounded-lg bg-white border border-slate-200 shadow-lg p-0 text-slate-900"
      [ngClass]="maxWidth"
    >
      <div class="flex items-center justify-between px-4 py-3 border-b border-slate-200">
        <h2 [id]="titleId" class="text-sm font-semibold text-slate-900">{{ title }}</h2>
        <button
          type="button"
          (click)="close()"
          class="text-slate-400 hover:text-slate-700 text-xs px-1.5 py-0.5 rounded hover:bg-slate-100"
          aria-label="Close modal"
        >
          ✕
        </button>
      </div>
      <div class="p-4">
        <ng-content></ng-content>
      </div>
    </dialog>
  `,
})
export class ModalComponent implements AfterViewInit, OnChanges {
  @Input() isOpen = false;
  @Input() title = 'Dialog';
  @Input() maxWidth = 'max-w-lg';
  @Output() closed = new EventEmitter<void>();

  @ViewChild('dialogEl') dialogRef?: ElementRef<HTMLDialogElement>;

  readonly titleId = `dialog-title-${Math.random().toString(36).slice(2, 9)}`;

  ngAfterViewInit(): void {
    const dialog = this.dialogRef?.nativeElement;
    if (!dialog) return;

    // Fallback for browsers without native <dialog closedby="any"> support
    if (typeof HTMLDialogElement !== 'undefined' && !('closedBy' in HTMLDialogElement.prototype)) {
      dialog.addEventListener('click', (event: MouseEvent) => {
        if (event.target !== dialog) return;
        const rect = dialog.getBoundingClientRect();
        const isDialogContent =
          rect.top <= event.clientY &&
          event.clientY <= rect.top + rect.height &&
          rect.left <= event.clientX &&
          event.clientX <= rect.left + rect.width;
        if (isDialogContent) return;
        dialog.close();
      });
    }

    this.syncOpenState();
  }

  ngOnChanges(_changes: SimpleChanges): void {
    this.syncOpenState();
  }

  close(): void {
    const dialog = this.dialogRef?.nativeElement;
    if (dialog && dialog.open) {
      dialog.close();
    } else {
      this.closed.emit();
    }
  }

  onNativeClose(): void {
    this.closed.emit();
  }

  private syncOpenState(): void {
    const dialog = this.dialogRef?.nativeElement;
    if (!dialog) return;
    if (this.isOpen && !dialog.open) {
      dialog.showModal();
    } else if (!this.isOpen && dialog.open) {
      dialog.close();
    }
  }
}
