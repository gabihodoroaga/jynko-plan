import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { WorkItemService } from '../../../core/services/work-item.service';
import { ProjectService } from '../../../core/services/project.service';
import { WorkItem } from '../../../core/models/project.model';
import { StatusBadgeComponent } from '../../../shared/components/status-badge.component';
import { PriorityBadgeComponent } from '../../../shared/components/priority-badge.component';

interface TimelineBarRow {
  item: WorkItem;
  epicName: string;
  leftPercent: number;
  widthPercent: number;
  startLabel: string;
  dueLabel: string;
}

@Component({
  selector: 'app-work-item-timeline',
  standalone: true,
  imports: [CommonModule, StatusBadgeComponent, PriorityBadgeComponent],
  template: `
    <div class="bg-white border border-slate-200 rounded-md shadow-card overflow-hidden">
      <!-- Timeline Header -->
      <div class="flex flex-wrap items-center justify-between gap-2 px-3.5 sm:px-4 py-2.5 bg-slate-50 border-b border-slate-200">
        <div>
          <h3 class="text-sm font-semibold text-slate-900">Timeline / Gantt Roadmap</h3>
          <p class="text-[11px] text-slate-500">
            Visualizing work item duration bars from <code class="font-mono">start_date</code> to <code class="font-mono">due_date</code> ({{ windowRangeLabel() }})
          </p>
        </div>
        <div class="flex flex-wrap items-center gap-3 text-[11px] text-slate-600">
          <div class="inline-flex items-center gap-1">
            <button
              type="button"
              (click)="shiftWindow(-12)"
              aria-label="Timeline previous window"
              class="px-2 py-1 text-xs bg-white border border-slate-200 rounded hover:bg-slate-100 text-slate-700"
            >
              ← Prev
            </button>
            <button
              type="button"
              (click)="jumpToToday()"
              aria-label="Timeline jump to today"
              class="px-2 py-1 text-xs bg-white border border-slate-200 rounded hover:bg-slate-100 text-slate-700"
            >
              Today
            </button>
            <button
              type="button"
              (click)="shiftWindow(12)"
              aria-label="Timeline next window"
              class="px-2 py-1 text-xs bg-white border border-slate-200 rounded hover:bg-slate-100 text-slate-700"
            >
              Next →
            </button>
          </div>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-sm bg-emerald-500"></span> Done
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-sm bg-amber-500"></span> In Progress
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-sm bg-sky-600"></span> In Review / Todo
          </span>
        </div>
      </div>

      <!-- Scrollable Gantt Grid Wrapper for Mobile -->
      <div class="overflow-x-auto">
        <div class="min-w-[680px]">
          <!-- Axis Labels -->
          <div class="grid grid-cols-12 border-b border-slate-200 bg-slate-50 text-[11px] font-semibold text-slate-600">
            <div class="col-span-4 px-3.5 py-2 border-r border-slate-200">Work Item & Epic</div>
            <div class="col-span-8 grid grid-cols-6 divide-x divide-slate-200">
              @for (tick of dateTicks(); track tick) {
                <div class="px-2 py-2 text-center font-mono text-[10px] text-slate-500">{{ tick }}</div>
              }
            </div>
          </div>

          <!-- Gantt Rows -->
          <div class="divide-y divide-slate-200">
            @for (row of timelineRows(); track row.item.id) {
              <div
                class="grid grid-cols-12 items-center hover:bg-slate-50 transition-colors cursor-pointer"
                (click)="workItemService.openItemDetail(row.item.id)"
              >
                <!-- Left Metadata Column -->
                <div class="col-span-4 px-3.5 py-3 border-r border-slate-200 flex items-center justify-between gap-2 min-w-0">
                  <div class="min-w-0">
                    <div class="flex items-center gap-2">
                      <span class="font-mono text-[11px] text-slate-400">{{ row.item.key }}</span>
                      <span class="text-xs font-medium text-slate-900 truncate">{{ row.item.title }}</span>
                    </div>
                    <div class="text-[11px] text-slate-400 truncate mt-0.5">
                      Epic: {{ row.epicName }}
                    </div>
                  </div>
                  <app-priority-badge [priority]="row.item.priority"></app-priority-badge>
                </div>

                <!-- Right Horizontal Bar Track -->
                <div class="col-span-8 relative h-12 px-2 flex items-center bg-slate-50/40">
                  <!-- Subtle vertical grid lines -->
                  <div class="absolute inset-0 grid grid-cols-6 divide-x divide-slate-200/70 pointer-events-none">
                    <div></div><div></div><div></div><div></div><div></div><div></div>
                  </div>

                  <!-- Duration Bar -->
                  <div
                    class="relative h-6 rounded-md border px-2 flex items-center justify-between text-[11px] font-medium shadow-sm transition-all"
                    [ngClass]="barClasses(row.item)"
                    [style.marginLeft.%]="row.leftPercent"
                    [style.width.%]="row.widthPercent"
                  >
                    <span class="truncate">{{ row.item.key }} · {{ row.startLabel }} → {{ row.dueLabel }}</span>
                  </div>
                </div>
              </div>
            }
          </div>
        </div>
      </div>
    </div>
  `,
})
export class WorkItemTimelineComponent {
  readonly workItemService = inject(WorkItemService);
  readonly projectService = inject(ProjectService);

  readonly windowOffsetDays = signal<number>(0);
  readonly anchoredToToday = signal<boolean>(false);

  private readonly windowBounds = computed(() => {
    const dayMs = 86_400_000;
    const now = new Date();
    const todayUtc = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());

    let anchorMs = todayUtc - 4 * dayMs;
    if (!this.anchoredToToday()) {
      const dated = this.workItemService
        .filteredItems()
        .map((it) => (it.start_date ? Date.parse(it.start_date) : it.due_date ? Date.parse(it.due_date) - 4 * dayMs : NaN))
        .filter((ms) => Number.isFinite(ms));
      if (dated.length > 0) {
        anchorMs = Math.min(...dated) - 2 * dayMs;
      }
    }

    const startMs = anchorMs + this.windowOffsetDays() * dayMs;
    const endMs = startMs + 24 * dayMs;
    return { startMs, endMs };
  });

  readonly dateTicks = computed<string[]>(() => {
    const { startMs } = this.windowBounds();
    const dayMs = 86_400_000;
    return [0, 4, 8, 12, 16, 20].map((offset) =>
      new Date(startMs + offset * dayMs).toLocaleDateString('en-US', {
        month: 'short',
        day: '2-digit',
      })
    );
  });

  readonly windowRangeLabel = computed<string>(() => {
    const { startMs, endMs } = this.windowBounds();
    const fmt = (ms: number) =>
      new Date(ms).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      });
    return `${fmt(startMs)} – ${fmt(endMs)}`;
  });

  shiftWindow(days: number): void {
    this.windowOffsetDays.update((d) => d + days);
  }

  jumpToToday(): void {
    this.anchoredToToday.set(true);
    this.windowOffsetDays.set(0);
  }

  readonly timelineRows = computed<TimelineBarRow[]>(() => {
    const items = this.workItemService.filteredItems();
    const epics = this.projectService.epics();
    const { startMs: windowStartMs, endMs: windowEndMs } = this.windowBounds();
    const span = windowEndMs - windowStartMs;
    const defaultStartMs = windowStartMs + 4 * 86400000;

    return items.map((item) => {
      const startMs = item.start_date
        ? new Date(item.start_date).getTime()
        : item.due_date
          ? new Date(item.due_date).getTime() - 4 * 86400000
          : defaultStartMs;
      const endMs = item.due_date
        ? new Date(item.due_date).getTime()
        : startMs + 4 * 86400000;

      const clampedStart = Math.max(windowStartMs, Math.min(startMs, windowEndMs));
      const clampedEnd = Math.max(clampedStart + 86400000, Math.min(endMs, windowEndMs));

      const leftPercent = Math.max(0, Math.min(82, ((clampedStart - windowStartMs) / span) * 100));
      const widthPercent = Math.max(14, Math.min(100 - leftPercent, ((clampedEnd - clampedStart) / span) * 100));

      const epic = epics.find((e) => e.id === item.epic_id);

      return {
        item,
        epicName: epic ? epic.name : 'Unassigned Epic',
        leftPercent,
        widthPercent,
        startLabel: new Date(startMs).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
        dueLabel: new Date(endMs).toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
      };
    });
  });

  barClasses(item: WorkItem): string {
    switch (item.status) {
      case 'DONE':
        return 'bg-emerald-50 text-emerald-800 border-emerald-300';
      case 'IN_PROGRESS':
        return 'bg-amber-50 text-amber-800 border-amber-300';
      case 'IN_REVIEW':
        return 'bg-sky-50 text-sky-800 border-sky-300';
      case 'CANCELLED':
        return 'bg-rose-50 text-rose-800 border-rose-300 line-through';
      default:
        return 'bg-white text-slate-700 border-slate-300';
    }
  }
}
