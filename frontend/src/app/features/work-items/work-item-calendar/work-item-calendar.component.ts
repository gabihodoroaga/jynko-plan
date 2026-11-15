import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { WorkItemService } from '../../../core/services/work-item.service';
import { WorkItem } from '../../../core/models/project.model';
import { StatusBadgeComponent } from '../../../shared/components/status-badge.component';

interface CalendarDayCell {
  date: Date;
  isoDate: string;
  dayNumber: number;
  isCurrentMonth: boolean;
  isToday: boolean;
  items: WorkItem[];
}

@Component({
  selector: 'app-work-item-calendar',
  standalone: true,
  imports: [CommonModule, StatusBadgeComponent],
  template: `
    <div class="bg-white border border-slate-200 rounded-md shadow-card overflow-hidden">
      <!-- Calendar Month Toolbar -->
      <div class="flex flex-wrap items-center justify-between gap-2 px-3 sm:px-4 py-2.5 border-b border-slate-200 bg-slate-50">
        <div class="flex flex-wrap items-center gap-2">
          <h3 class="text-sm font-semibold text-slate-900">
            {{ currentMonthDate() | date: 'MMMM yyyy' }}
          </h3>
          <span class="text-[11px] text-slate-500">
            Showing items by due date (or start date)
          </span>
        </div>
        <div class="flex items-center gap-1.5">
          <button
            type="button"
            (click)="changeMonth(-1)"
            class="px-2.5 py-1 text-xs bg-white border border-slate-200 rounded hover:bg-slate-100 text-slate-700"
          >
            ← Prev
          </button>
          <button
            type="button"
            (click)="resetMonth()"
            aria-label="Calendar jump to current month"
            class="px-2.5 py-1 text-xs bg-white border border-slate-200 rounded hover:bg-slate-100 text-slate-700"
          >
            Today
          </button>
          <button
            type="button"
            (click)="changeMonth(1)"
            class="px-2.5 py-1 text-xs bg-white border border-slate-200 rounded hover:bg-slate-100 text-slate-700"
          >
            Next →
          </button>
        </div>
      </div>

      <!-- Scrollable Calendar Grid Wrapper for Mobile -->
      <div class="overflow-x-auto">
        <div class="min-w-[640px]">
          <!-- Days of Week Header -->
          <div class="grid grid-cols-7 border-b border-slate-200 bg-slate-50 text-[11px] font-semibold text-slate-600">
            @for (dayName of weekDays; track dayName) {
              <div class="px-2.5 py-1.5 border-r last:border-r-0 border-slate-200">
                {{ dayName }}
              </div>
            }
          </div>

          <!-- 42-Cell Calendar Grid -->
          <div class="grid grid-cols-7 divide-x divide-y divide-slate-200">
            @for (cell of calendarCells(); track cell.isoDate) {
              <div
                class="min-h-[112px] p-2 transition-colors"
                [ngClass]="cell.isCurrentMonth ? 'bg-white' : 'bg-slate-50/70 text-slate-400'"
              >
                <div class="flex items-center justify-between mb-1.5">
                  <span
                    class="text-[11px] font-mono px-1.5 py-0.5 rounded"
                    [ngClass]="
                      cell.isToday
                        ? 'bg-sky-600 text-white font-semibold'
                        : 'text-slate-600'
                    "
                  >
                    {{ cell.dayNumber }}
                  </span>
                  @if (cell.items.length > 0) {
                    <span class="text-[10px] font-mono text-slate-400">{{ cell.items.length }}</span>
                  }
                </div>

                <div class="space-y-1">
                  @for (item of cell.items; track item.id) {
                    <div
                      (click)="workItemService.openItemDetail(item.id)"
                      class="px-2 py-1 rounded border text-[11px] cursor-pointer truncate transition-colors"
                      [ngClass]="itemPillClasses(item)"
                      [title]="item.key + ': ' + item.title"
                    >
                      <span class="font-mono font-semibold mr-1">{{ item.key }}</span>
                      <span>{{ item.title }}</span>
                    </div>
                  }
                </div>
              </div>
            }
          </div>
        </div>
      </div>
    </div>
  `,
})
export class WorkItemCalendarComponent {
  readonly workItemService = inject(WorkItemService);

  readonly weekDays = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
  readonly currentMonthDate = signal<Date>(this.startOfCurrentMonth());

  readonly calendarCells = computed<CalendarDayCell[]>(() => {
    const ref = this.currentMonthDate();
    const year = ref.getFullYear();
    const month = ref.getMonth();
    const todayIso = this.toYMD(new Date());

    const firstDayOfMonth = new Date(year, month, 1);
    const startOffset = firstDayOfMonth.getDay();
    const gridStart = new Date(year, month, 1 - startOffset);

    const items = this.workItemService.filteredItems();
    const cells: CalendarDayCell[] = [];

    for (let i = 0; i < 42; i++) {
      const cellDate = new Date(
        gridStart.getFullYear(),
        gridStart.getMonth(),
        gridStart.getDate() + i
      );
      const isoPrefix = this.toYMD(cellDate);

      const dayItems = items.filter((it) => {
        const target = it.due_date || it.start_date;
        if (!target) return false;
        return target.slice(0, 10) === isoPrefix;
      });

      cells.push({
        date: cellDate,
        isoDate: isoPrefix,
        dayNumber: cellDate.getDate(),
        isCurrentMonth: cellDate.getMonth() === month,
        isToday: isoPrefix === todayIso,
        items: dayItems,
      });
    }
    return cells;
  });

  private startOfCurrentMonth(): Date {
    const now = new Date();
    return new Date(now.getFullYear(), now.getMonth(), 1);
  }

  changeMonth(delta: number): void {
    const curr = this.currentMonthDate();
    this.currentMonthDate.set(new Date(curr.getFullYear(), curr.getMonth() + delta, 1));
  }

  resetMonth(): void {
    this.currentMonthDate.set(this.startOfCurrentMonth());
  }

  itemPillClasses(item: WorkItem): string {
    switch (item.status) {
      case 'DONE':
        return 'bg-emerald-50 text-emerald-700 border-emerald-200 hover:bg-emerald-100';
      case 'IN_PROGRESS':
        return 'bg-amber-50 text-amber-700 border-amber-200 hover:bg-amber-100';
      case 'IN_REVIEW':
        return 'bg-sky-50 text-sky-700 border-sky-200 hover:bg-sky-100';
      case 'CANCELLED':
        return 'bg-rose-50 text-rose-700 border-rose-200 line-through';
      default:
        return 'bg-slate-100 text-slate-700 border-slate-200 hover:bg-slate-200/70';
    }
  }

  private toYMD(d: Date): string {
    const yyyy = d.getFullYear();
    const mm = String(d.getMonth() + 1).padStart(2, '0');
    const dd = String(d.getDate()).padStart(2, '0');
    return `${yyyy}-${mm}-${dd}`;
  }
}
