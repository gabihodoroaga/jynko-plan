import { Component, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { StickyService } from '../../core/services/sticky.service';
import { ProjectService } from '../../core/services/project.service';
import { WorkItemService } from '../../core/services/work-item.service';
import { Sticky, StickyColor } from '../../core/models/project.model';

@Component({
  selector: 'app-stickies',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="w-full p-4 sm:p-6 space-y-5 bg-slate-50 min-h-full">
      <!-- Top Header & Action Controls -->
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 class="text-base font-semibold text-slate-900">Stickies</h2>
          <p class="text-xs text-slate-500">
            Quick colorful sticky notes for brainstorming and scratchpad thoughts
          </p>
        </div>

        <div class="flex items-center gap-2">
          <!-- Search input -->
          <div class="relative">
            <input
              type="search"
              [ngModel]="stickyService.searchQuery()"
              (ngModelChange)="stickyService.searchQuery.set($event)"
              placeholder="Filter notes..."
              aria-label="Filter stickies by text"
              class="h-8 pl-3 pr-3 rounded-md bg-white border border-slate-200 text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:border-slate-300"
            />
          </div>

          @if (projectService.canEdit()) {
            <button
              type="button"
              (click)="stickyService.addRandomSticky()"
              aria-label="Surprise me random sticky"
              class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md border border-slate-200 bg-white hover:bg-slate-100 text-xs font-medium text-slate-700 shadow-sm transition-colors"
            >
              <span>🎲</span>
              <span>Surprise Me</span>
            </button>

            <button
              type="button"
              (click)="isComposerOpen.set(!isComposerOpen())"
              aria-label="Create new sticky"
              class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium shadow-sm transition-colors"
            >
              <span>+ New sticky</span>
            </button>
          }
        </div>
      </div>

      <!-- New Sticky Quick Composer Card -->
      @if (isComposerOpen() && projectService.canEdit()) {
        <form
          (ngSubmit)="submitNewSticky()"
          class="p-4 rounded-xl border border-slate-200 bg-white shadow-card space-y-3"
        >
          <div class="flex items-center justify-between">
            <span class="text-xs font-semibold text-slate-800">New Sticky Note</span>
            <div class="flex items-center gap-1.5">
              @for (pal of stickyService.palettes; track pal.color) {
                <button
                  type="button"
                  (click)="newStickyColor.set(pal.color)"
                  [attr.aria-label]="'Select new sticky color ' + pal.label"
                  [title]="pal.label"
                  class="w-5 h-5 rounded-full border transition-transform"
                  [style.backgroundColor]="pal.bgHex"
                  [style.borderColor]="pal.borderHex"
                  [ngClass]="newStickyColor() === pal.color ? 'scale-125 ring-2 ring-slate-800/30' : 'opacity-80 hover:opacity-100'"
                ></button>
              }
            </div>
          </div>

          <textarea
            name="newStickyContent"
            [(ngModel)]="newStickyContent"
            rows="3"
            placeholder="Jot down a quick idea, reminder, or retro note..."
            aria-label="New sticky note content"
            style="field-sizing: content; height: auto; width: 100%; min-block-size: 3lh; max-block-size: 10lh;"
            class="w-full p-3 rounded-lg border text-xs leading-relaxed focus:outline-none"
            [style.backgroundColor]="stickyService.getPalette(newStickyColor()).bgHex"
            [style.borderColor]="stickyService.getPalette(newStickyColor()).borderHex"
            [style.color]="stickyService.getPalette(newStickyColor()).textHex"
          ></textarea>

          <div class="flex items-center justify-between">
            <span class="text-[11px] font-mono text-slate-400">
              Palette: {{ stickyService.getPalette(newStickyColor()).label }}
            </span>
            <div class="flex items-center gap-2">
              <button
                type="button"
                (click)="isComposerOpen.set(false)"
                class="px-2.5 py-1 rounded-md border border-slate-200 text-xs text-slate-600 hover:bg-slate-100"
              >
                Cancel
              </button>
              <button
                type="submit"
                aria-label="Save new sticky"
                class="px-3 py-1 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
              >
                Add Sticky
              </button>
            </div>
          </div>
        </form>
      }

      <!-- Funny Color Palette Filter Pills -->
      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          (click)="stickyService.selectedColor.set('ALL')"
          class="px-2.5 py-1 rounded-full border text-xs font-medium transition-colors"
          [ngClass]="
            stickyService.selectedColor() === 'ALL'
              ? 'bg-slate-900 text-white border-slate-900'
              : 'bg-white text-slate-700 border-slate-200 hover:bg-slate-100'
          "
        >
          All Colors ({{ stickyService.stickies().length }})
        </button>

        @for (pal of stickyService.palettes; track pal.color) {
          <button
            type="button"
            (click)="stickyService.selectedColor.set(pal.color)"
            [attr.aria-label]="'Filter stickies by ' + pal.label"
            class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full border text-xs font-medium transition-colors"
            [ngClass]="
              stickyService.selectedColor() === pal.color
                ? 'bg-slate-900 text-white border-slate-900'
                : 'bg-white text-slate-700 border-slate-200 hover:bg-slate-100'
            "
          >
            <span
              class="w-2.5 h-2.5 rounded-full border"
              [style.backgroundColor]="pal.bgHex"
              [style.borderColor]="pal.borderHex"
            ></span>
            <span>{{ pal.label }}</span>
          </button>
        }
      </div>

      <!-- Colorful Stickies Grid -->
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        @for (sticky of stickyService.filteredStickies(); track sticky.id) {
          @let pal = stickyService.getPalette(sticky.color);
          <div
            class="rounded-xl border-2 p-4 shadow-sm flex flex-col justify-between gap-3 transition-transform hover:-translate-y-0.5"
            [style.backgroundColor]="pal.bgHex"
            [style.borderColor]="pal.borderHex"
            [style.color]="pal.textHex"
          >
            <!-- Top Row: Funny Color Label + Pin / Convert / Delete Actions -->
            <div class="flex items-center justify-between gap-2">
              <div class="flex items-center gap-1.5 min-w-0">
                <span
                  class="px-2 py-0.5 rounded-full text-[10px] font-mono font-semibold uppercase tracking-wide bg-white/60 border border-black/10"
                >
                  {{ pal.label }}
                </span>
                @if (sticky.is_pinned) {
                  <span
                    class="px-1.5 py-0.5 rounded bg-white/70 text-[10px] font-semibold"
                    title="Pinned note"
                  >
                    📌 Pinned
                  </span>
                }
              </div>

              @if (projectService.canEdit()) {
                <div class="flex items-center gap-1">
                  <button
                    type="button"
                    (click)="stickyService.togglePin(sticky.id)"
                    [attr.aria-label]="sticky.is_pinned ? 'Unpin sticky' : 'Pin sticky'"
                    class="p-1 rounded hover:bg-black/10 text-xs transition-colors"
                    [title]="sticky.is_pinned ? 'Unpin' : 'Pin to top'"
                  >
                    📌
                  </button>
                  <button
                    type="button"
                    (click)="convertToWorkItem(sticky)"
                    aria-label="Convert sticky to work item"
                    class="px-1.5 py-0.5 rounded hover:bg-black/10 text-[10px] font-medium transition-colors"
                    title="Convert note to Work Item"
                  >
                    ↗ Task
                  </button>
                  <button
                    type="button"
                    (click)="stickyService.deleteSticky(sticky.id)"
                    aria-label="Delete sticky"
                    class="p-1 rounded hover:bg-black/10 text-xs transition-colors"
                    title="Delete sticky"
                  >
                    ✕
                  </button>
                </div>
              }
            </div>

            <!-- Inline Auto-Sizing Content Textarea -->
            <textarea
              [ngModel]="sticky.content"
              (ngModelChange)="stickyService.updateSticky(sticky.id, { content: $event })"
              [disabled]="!projectService.canEdit()"
              rows="4"
              aria-label="Sticky note content"
              style="field-sizing: content; height: auto; width: 100%; min-block-size: 4lh; max-block-size: 12lh;"
              class="w-full bg-transparent resize-none border-0 focus:outline-none text-xs font-medium leading-relaxed placeholder-black/40"
              placeholder="Write your note..."
            ></textarea>

            <!-- Footer: Color Swatch Switcher & Project Tag -->
            <div class="pt-2 border-t border-black/10 flex items-center justify-between gap-2">
              <div class="flex items-center gap-1">
                @for (swatch of stickyService.palettes; track swatch.color) {
                  <button
                    type="button"
                    (click)="stickyService.updateSticky(sticky.id, { color: swatch.color })"
                    [disabled]="!projectService.canEdit()"
                    [attr.aria-label]="'Change sticky color to ' + swatch.label"
                    [title]="swatch.label"
                    class="w-3.5 h-3.5 rounded-full border border-black/25 transition-transform"
                    [style.backgroundColor]="swatch.dotHex"
                    [ngClass]="sticky.color === swatch.color ? 'scale-125 ring-1 ring-black/50' : 'opacity-75 hover:opacity-100'"
                  ></button>
                }
              </div>

              <span class="text-[10px] font-mono opacity-75">
                {{ getProjectName(sticky.project_id) }}
              </span>
            </div>
          </div>
        }
      </div>
    </div>
  `,
})
export class StickiesComponent {
  readonly stickyService = inject(StickyService);
  readonly projectService = inject(ProjectService);
  private readonly workItemService = inject(WorkItemService);

  readonly isComposerOpen = signal<boolean>(false);
  readonly newStickyColor = signal<StickyColor>('BANANA');
  newStickyContent = '';

  submitNewSticky(): void {
    const text = this.newStickyContent.trim();
    if (!text) return;
    this.stickyService.createSticky(text, this.newStickyColor(), false);
    this.newStickyContent = '';
    this.isComposerOpen.set(false);
  }

  convertToWorkItem(sticky: Sticky): void {
    const title = sticky.content.split('\n')[0].slice(0, 90) || 'Work item from sticky';
    this.workItemService.createItem({
      title,
      description: sticky.content,
      status: 'TODO',
      priority: 'MEDIUM',
    });
  }

  getProjectName(projectId: string): string {
    return this.projectService.getProjectById(projectId)?.name ?? 'hodo';
  }
}
