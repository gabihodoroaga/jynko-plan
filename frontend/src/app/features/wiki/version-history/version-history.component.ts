import { Component, EventEmitter, Input, Output, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { PageService } from '../../../core/services/page.service';
import { ProjectService } from '../../../core/services/project.service';
import { PageVersion } from '../../../core/models/page.model';

@Component({
  selector: 'app-version-history',
  standalone: true,
  imports: [CommonModule],
  template: `
    @if (isOpen) {
      <div class="fixed inset-y-0 right-0 z-40 w-full max-w-xs shadow-xl md:static md:w-80 md:shadow-none border-l border-slate-200 bg-slate-50 h-full flex flex-col">
        <div class="flex items-center justify-between px-3.5 py-2.5 border-b border-slate-200 bg-white">
          <h4 class="text-xs font-semibold text-slate-900">Revision History</h4>
          <button
            type="button"
            (click)="closed.emit()"
            class="text-slate-400 hover:text-slate-700 text-xs px-1.5 py-0.5 rounded hover:bg-slate-100"
          >
            ✕
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-3 space-y-2.5">
          @for (ver of versions(); track ver.id) {
            <div class="bg-white border border-slate-200 rounded-md p-3 shadow-card space-y-1.5">
              <div class="flex items-center justify-between">
                <span class="text-xs font-medium text-slate-900 truncate">{{ ver.title }}</span>
              </div>
              <div class="flex items-center justify-between text-[10px] font-mono text-slate-400">
                <span>By {{ ver.created_by }}</span>
                <span>{{ ver.created_at | date: 'MMM d, HH:mm:ss' }}</span>
              </div>
              <pre class="text-[10px] font-mono text-slate-600 bg-slate-50 p-2 rounded border border-slate-200 max-h-24 overflow-hidden whitespace-pre-wrap">{{ ver.content }}</pre>
              @if (projectService.canEdit() && !pageService.activePage()?.is_locked) {
                <div class="flex justify-end pt-1">
                  <button
                    type="button"
                    (click)="restore(ver)"
                    class="text-[11px] font-medium text-sky-600 hover:underline"
                  >
                    Restore this snapshot
                  </button>
                </div>
              }
            </div>
          } @empty {
            <div class="text-xs text-slate-400">
              Edits to this page will automatically generate immutable PageVersion snapshots here.
            </div>
          }
        </div>
      </div>
    }
  `,
})
export class VersionHistoryComponent {
  @Input() isOpen = false;
  @Output() closed = new EventEmitter<void>();

  readonly pageService = inject(PageService);
  readonly projectService = inject(ProjectService);

  readonly versions = computed<PageVersion[]>(() => {
    const pageId = this.pageService.activePageId();
    if (!pageId) return [];
    return this.pageService.versionsByPage()[pageId] ?? [];
  });

  restore(ver: PageVersion): void {
    const pageId = this.pageService.activePageId();
    if (!pageId) return;
    this.pageService.restoreVersion(pageId, ver);
  }
}
