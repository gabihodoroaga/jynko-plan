import { Component, OnInit, effect, inject, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpEvent, HttpEventType } from '@angular/common/http';
import { Attachment } from '../../../core/models/project.model';
import { AttachmentService } from '../../../core/services/attachment.service';

@Component({
  selector: 'app-work-item-attachments',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div
      class="space-y-2.5"
      (dragover)="onDragOver($event)"
      (dragleave)="onDragLeave($event)"
      (drop)="onDrop($event)"
    >
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <h4 class="text-xs font-semibold text-slate-900">
            Attachments ({{ attachments().length }})
          </h4>
          @if (isUploading()) {
            <span class="inline-flex items-center gap-1.5 text-[11px] text-sky-600 font-medium">
              <svg class="animate-spin h-3 w-3 text-sky-600" viewBox="0 0 24 24" fill="none">
                <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"></path>
              </svg>
              <span>Uploading...</span>
            </span>
          }
        </div>

        @if (canEdit()) {
          <div>
            <input
              #fileInput
              type="file"
              multiple
              class="hidden"
              (change)="onFileInputChange($event)"
            />
            <button
              type="button"
              (click)="fileInput.click()"
              aria-label="Upload attachment"
              class="inline-flex items-center gap-1 px-2 py-1 rounded border border-slate-200 bg-white hover:bg-slate-50 text-slate-600 text-xs font-medium transition-colors"
            >
              <svg class="w-3.5 h-3.5 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="m21.44 11.05-9.19 9.19a6 6 0 0 1-8.49-8.49l8.57-8.57A4 4 0 1 1 18 8.84l-8.59 8.57a2 2 0 0 1-2.83-2.83l8.49-8.48" />
              </svg>
              <span>Attach file</span>
            </button>
          </div>
        }
      </div>

      <!-- Upload Progress Bar (Styled native progress per modern web guidance) -->
      @if (uploadProgress() !== null) {
        <div class="space-y-1 p-2 rounded-lg bg-sky-50/60 border border-sky-100">
          <div class="flex items-center justify-between text-[11px] text-sky-800 font-medium">
            <span>Uploading {{ currentUploadFileName() }}...</span>
            <span>{{ uploadProgress() }}%</span>
          </div>
          <progress
            [value]="uploadProgress()"
            max="100"
            aria-label="Upload progress"
            class="w-full h-1.5 rounded-full overflow-hidden bg-slate-200 appearance-none [&::-webkit-progress-bar]:bg-slate-200 [&::-webkit-progress-value]:bg-sky-600 [&::-moz-progress-bar]:bg-sky-600"
          ></progress>
        </div>
      }

      @if (errorMessage()) {
        <div class="p-2 text-xs text-rose-600 bg-rose-50 border border-rose-200 rounded-md">
          {{ errorMessage() }}
        </div>
      }

      <!-- Attachments Grid -->
      @if (attachments().length > 0) {
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-2">
          @for (att of attachments(); track att.id) {
            <div
              class="flex items-center justify-between p-2 rounded-lg border border-slate-200 bg-white hover:border-slate-300 transition-colors group"
            >
              <a
                [href]="att.url"
                target="_blank"
                rel="noopener noreferrer"
                [title]="att.file_name"
                class="flex items-center gap-2 min-w-0 flex-1 mr-2"
              >
                @if (isImage(att)) {
                  <div class="w-8 h-8 rounded border border-slate-200 bg-slate-50 overflow-hidden shrink-0 flex items-center justify-center">
                    <img [src]="att.url" [alt]="att.file_name" class="w-full h-full object-cover" loading="lazy" />
                  </div>
                } @else {
                  <div class="w-8 h-8 rounded border border-slate-200 bg-slate-50 text-slate-500 shrink-0 flex items-center justify-center font-mono text-[9px] font-bold uppercase">
                    {{ getExtension(att.file_name) }}
                  </div>
                }

                <!-- Truncated name preserving extension per single-line-overflow guidance -->
                <div class="min-w-0 flex-1 text-left">
                  <div class="flex items-baseline text-xs font-medium text-slate-800 leading-tight">
                    <span class="truncate" [title]="att.file_name">{{ splitFileName(att.file_name).base }}</span>
                    <span class="shrink-0 text-slate-500">{{ splitFileName(att.file_name).ext }}</span>
                  </div>
                  <p class="text-[10px] text-slate-400 font-mono mt-0.5">
                    {{ formatSize(att.size_bytes) }} • {{ att.created_at | date: 'MMM d' }}
                  </p>
                </div>
              </a>

              <div class="flex items-center gap-1 shrink-0">
                <button
                  type="button"
                  (click)="copyMarkdown(att)"
                  [attr.aria-label]="'Copy markdown for ' + att.file_name"
                  [title]="copiedId() === att.id ? 'Copied!' : 'Copy Markdown'"
                  class="p-1.5 rounded text-slate-400 hover:text-slate-700 hover:bg-slate-100 transition-colors"
                >
                  @if (copiedId() === att.id) {
                    <span class="text-[10px] font-bold text-emerald-600">✓</span>
                  } @else {
                    <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect width="14" height="14" x="8" y="8" rx="2" ry="2"/>
                      <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"/>
                    </svg>
                  }
                </button>

                @if (canEdit()) {
                  <button
                    type="button"
                    (click)="deleteAttachment(att)"
                    [attr.aria-label]="'Delete attachment ' + att.file_name"
                    title="Delete attachment"
                    class="p-1.5 rounded text-slate-400 hover:text-rose-600 hover:bg-rose-50 transition-colors"
                  >
                    <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M3 6h18m-2 0v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6m3 0V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/>
                    </svg>
                  </button>
                }
              </div>
            </div>
          }
        </div>
      } @else {
        <div
          [class.border-sky-400]="isDragOver()"
          [class.bg-sky-50]="isDragOver()"
          class="p-4 rounded-lg border border-dashed border-slate-200 bg-slate-50/50 text-center transition-colors"
        >
          <p class="text-xs text-slate-500">
            No attachments yet. Drop files here or paste images in descriptions and comments.
          </p>
        </div>
      }
    </div>
  `,
})
export class WorkItemAttachmentsComponent implements OnInit {
  private readonly attachmentService = inject(AttachmentService);

  readonly projectId = input.required<string>();
  readonly itemId = input.required<string>();
  readonly canEdit = input<boolean>(true);

  readonly attachmentUploaded = output<Attachment>();
  readonly copySnippet = output<string>();

  readonly attachments = signal<Attachment[]>([]);
  readonly isUploading = signal<boolean>(false);
  readonly uploadProgress = signal<number | null>(null);
  readonly currentUploadFileName = signal<string>('');
  readonly errorMessage = signal<string | null>(null);
  readonly copiedId = signal<string | null>(null);
  readonly isDragOver = signal<boolean>(false);

  constructor() {
    effect(() => {
      const pid = this.projectId();
      const iid = this.itemId();
      if (pid && iid) {
        this.loadAttachments();
      }
    });
  }

  ngOnInit(): void {
    this.loadAttachments();
  }

  loadAttachments(): void {
    const pid = this.projectId();
    const iid = this.itemId();
    if (!pid) return;

    this.attachmentService.list(pid, iid).subscribe({
      next: (data) => {
        this.attachments.set(data || []);
      },
      error: () => {
        // Silently tolerate list error if project not ready
      },
    });
  }

  onFileInputChange(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files && input.files.length > 0) {
      this.uploadFiles(Array.from(input.files));
      input.value = '';
    }
  }

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    if (this.canEdit()) {
      this.isDragOver.set(true);
    }
  }

  onDragLeave(event: DragEvent): void {
    event.preventDefault();
    this.isDragOver.set(false);
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    this.isDragOver.set(false);
    if (!this.canEdit()) return;

    if (event.dataTransfer?.files && event.dataTransfer.files.length > 0) {
      this.uploadFiles(Array.from(event.dataTransfer.files));
    }
  }

  uploadFiles(files: File[]): void {
    const pid = this.projectId();
    const iid = this.itemId();
    if (!pid || files.length === 0) return;

    this.errorMessage.set(null);
    let index = 0;

    const uploadNext = () => {
      if (index >= files.length) {
        this.isUploading.set(false);
        this.uploadProgress.set(null);
        this.currentUploadFileName.set('');
        this.loadAttachments();
        return;
      }

      const file = files[index];
      this.isUploading.set(true);
      this.currentUploadFileName.set(file.name);
      this.uploadProgress.set(0);

      this.attachmentService.upload(pid, file, iid).subscribe({
        next: (event: HttpEvent<Attachment>) => {
          if (event.type === HttpEventType.UploadProgress && event.total) {
            this.uploadProgress.set(Math.round((100 * event.loaded) / event.total));
          } else if (event.type === HttpEventType.Response && event.body) {
            const att = event.body;
            this.attachments.update((list) => [att, ...list]);
            this.attachmentUploaded.emit(att);
            index++;
            uploadNext();
          }
        },
        error: (err) => {
          this.errorMessage.set(
            err.error?.error || `Failed to upload ${file.name}. Please ensure size is under 25MB.`
          );
          this.isUploading.set(false);
          this.uploadProgress.set(null);
        },
      });
    };

    uploadNext();
  }

  deleteAttachment(att: Attachment): void {
    const pid = this.projectId();
    if (!pid) return;

    this.attachmentService.delete(pid, att.id).subscribe({
      next: () => {
        this.attachments.update((list) => list.filter((a) => a.id !== att.id));
      },
      error: (err) => {
        this.errorMessage.set(err.error?.error || 'Failed to delete attachment.');
      },
    });
  }

  copyMarkdown(att: Attachment): void {
    if (navigator.clipboard) {
      navigator.clipboard.writeText(att.markdown).then(() => {
        this.copiedId.set(att.id);
        this.copySnippet.emit(att.markdown);
        setTimeout(() => {
          if (this.copiedId() === att.id) {
            this.copiedId.set(null);
          }
        }, 2000);
      });
    }
  }

  isImage(att: Attachment): boolean {
    return (
      att.content_type.startsWith('image/') ||
      /\.(png|jpe?g|gif|webp|svg)$/i.test(att.file_name)
    );
  }

  getExtension(name: string): string {
    const ext = name.split('.').pop() || 'file';
    return ext.slice(0, 4);
  }

  splitFileName(name: string): { base: string; ext: string } {
    const lastDot = name.lastIndexOf('.');
    if (lastDot > 0) {
      return {
        base: name.slice(0, lastDot),
        ext: name.slice(lastDot),
      };
    }
    return { base: name, ext: '' };
  }

  formatSize(bytes: number): string {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }
}
