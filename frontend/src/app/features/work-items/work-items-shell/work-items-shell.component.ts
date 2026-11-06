import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { WorkItemService } from '../../../core/services/work-item.service';
import { ProjectService } from '../../../core/services/project.service';
import { SettingsService } from '../../../core/services/settings.service';
import {
  ALL_PRIORITIES,
  ALL_STATUSES,
  WorkItemPriority,
  WorkItemStatus,
  WorkItemViewMode,
} from '../../../core/models/project.model';
import { WorkItemListComponent } from '../work-item-list/work-item-list.component';
import { WorkItemBoardComponent } from '../work-item-board/work-item-board.component';
import { WorkItemCalendarComponent } from '../work-item-calendar/work-item-calendar.component';
import { WorkItemTimelineComponent } from '../work-item-timeline/work-item-timeline.component';
import { WorkItemSpreadsheetComponent } from '../work-item-spreadsheet/work-item-spreadsheet.component';
import { WorkItemDetailDrawerComponent } from '../work-item-detail-drawer/work-item-detail-drawer.component';
import { DropdownComponent, DropdownOption } from '../../../shared/components/dropdown.component';
import { StatusBadgeComponent } from '../../../shared/components/status-badge.component';

@Component({
  selector: 'app-work-items-shell',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    WorkItemListComponent,
    WorkItemBoardComponent,
    WorkItemCalendarComponent,
    WorkItemTimelineComponent,
    WorkItemSpreadsheetComponent,
    WorkItemDetailDrawerComponent,
    DropdownComponent,
    StatusBadgeComponent,
  ],
  template: `
    <div class="flex flex-col w-full h-full">
      <!-- View Toolbar & Filter Bar (h-11 / 44px) -->
      <div
        class="min-h-[44px] px-4 py-1.5 bg-white border-b border-slate-200 flex flex-wrap items-center justify-between gap-2"
      >
        <!-- 5 Segmented View Switcher Buttons -->
        <div class="inline-flex items-center bg-slate-100 p-0.5 rounded-md border border-slate-200">
          @for (v of views; track v.mode) {
            <button
              type="button"
              (click)="workItemService.activeView.set(v.mode)"
              [attr.aria-label]="'Switch to ' + v.label + ' view'"
              class="px-2.5 py-1 rounded text-xs font-medium transition-colors"
              [ngClass]="
                workItemService.activeView() === v.mode
                  ? 'bg-white text-slate-900 shadow-card border border-slate-200/80'
                  : 'text-slate-600 hover:text-slate-900'
              "
            >
              {{ v.label }}
            </button>
          }
        </div>

        <!-- Multi-Attribute Styled Dropdown Filters, Board Columns Configurator & Primary Action Button -->
        <div class="flex flex-wrap items-center gap-2">
          <!-- Configurable Kanban Board Columns Popover Trigger (shown in BOARD mode) -->
          @if (workItemService.activeView() === 'BOARD') {
            <div class="relative">
              <button
                type="button"
                (click)="isColumnsPopoverOpen.set(!isColumnsPopoverOpen())"
                aria-label="Configure Kanban board columns"
                class="h-7 inline-flex items-center gap-1.5 px-2.5 rounded-md border border-slate-200 bg-white hover:bg-slate-50 text-xs font-medium text-slate-700 transition-colors"
              >
                <svg class="w-3.5 h-3.5 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                  <rect x="3" y="4" width="5" height="16" rx="1" />
                  <rect x="10" y="4" width="5" height="16" rx="1" />
                  <rect x="17" y="4" width="4" height="16" rx="1" />
                </svg>
                <span>Columns ({{ settingsService.visibleBoardStatuses().length }}/{{ allStatuses.length }})</span>
              </button>

              @if (isColumnsPopoverOpen()) {
                <div
                  role="dialog"
                  aria-label="Kanban Board Columns Configuration"
                  class="absolute left-0 mt-1.5 z-40 w-72 rounded-lg bg-white border border-slate-200 shadow-lg p-3 space-y-2.5"
                >
                  <div class="flex items-center justify-between border-b border-slate-100 pb-2">
                    <div>
                      <div class="text-xs font-semibold text-slate-900">Board Status Columns</div>
                      <div class="text-[11px] text-slate-500">
                        Hide statuses you don't want on the Kanban board
                      </div>
                    </div>
                    <button
                      type="button"
                      (click)="isColumnsPopoverOpen.set(false)"
                      class="text-xs text-slate-400 hover:text-slate-700 px-1"
                    >
                      ✕
                    </button>
                  </div>

                  <div class="space-y-1">
                    @for (st of allStatuses; track st.value) {
                      @let isVisible = settingsService.isBoardColumnVisible(st.value);
                      <div
                        class="flex items-center justify-between px-2 py-1.5 rounded-md hover:bg-slate-50"
                      >
                        <app-status-badge [status]="st.value"></app-status-badge>
                        <input
                          type="checkbox"
                          [attr.switch]="''"
                          [checked]="isVisible"
                          [disabled]="isVisible && settingsService.visibleBoardStatuses().length <= 1"
                          (change)="settingsService.toggleBoardColumn(st.value)"
                          [attr.aria-label]="'Toggle board column ' + st.label"
                          class="jynko-switch"
                        />
                      </div>
                    }
                  </div>

                  <div class="pt-2 border-t border-slate-100 flex items-center justify-between text-[11px]">
                    <button
                      type="button"
                      (click)="settingsService.resetBoardColumnsToDefault()"
                      aria-label="Reset board columns to default"
                      class="text-slate-600 hover:text-slate-900 font-medium"
                    >
                      Default (No Cancelled)
                    </button>
                    <button
                      type="button"
                      (click)="settingsService.showAllBoardColumns()"
                      aria-label="Show all board columns"
                      class="text-sky-600 hover:text-sky-700 font-medium"
                    >
                      Show All ({{ allStatuses.length }})
                    </button>
                  </div>
                </div>
              }
            </div>
          }

          <app-dropdown
            [options]="statusFilterOptions"
            [value]="workItemService.filter().status"
            (valueChange)="onStatusFilterChange($event)"
            size="sm"
            ariaLabel="Filter by Status"
          ></app-dropdown>

          <app-dropdown
            [options]="priorityFilterOptions"
            [value]="workItemService.filter().priority"
            (valueChange)="onPriorityFilterChange($event)"
            size="sm"
            ariaLabel="Filter by Priority"
          ></app-dropdown>

          <app-dropdown
            [options]="cycleFilterOptions()"
            [value]="workItemService.filter().cycle_id"
            (valueChange)="workItemService.updateFilter({ cycle_id: $event })"
            size="sm"
            ariaLabel="Filter by Cycle"
          ></app-dropdown>

          <app-dropdown
            [options]="moduleFilterOptions()"
            [value]="workItemService.filter().module_id"
            (valueChange)="workItemService.updateFilter({ module_id: $event })"
            size="sm"
            ariaLabel="Filter by Module"
          ></app-dropdown>

          <app-dropdown
            [options]="epicFilterOptions()"
            [value]="workItemService.filter().epic_id"
            (valueChange)="workItemService.updateFilter({ epic_id: $event })"
            size="sm"
            ariaLabel="Filter by Epic"
          ></app-dropdown>

          @if (hasActiveFilters()) {
            <button
              type="button"
              (click)="workItemService.resetFilters()"
              class="text-[11px] text-slate-500 hover:text-slate-800 px-2 py-1"
            >
              Clear
            </button>
          }

          @if (projectService.canEdit()) {
            <button
              type="button"
              (click)="workItemService.isQuickCreateOpen.set(true)"
              class="bg-sky-600 hover:bg-sky-700 text-white rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
            >
              + Add work item
            </button>
          }
        </div>
      </div>

      <!-- Active View Canvas (bg-slate-50) -->
      <div class="flex-1 min-h-0 p-4 bg-slate-50 overflow-auto flex flex-col">
        @switch (workItemService.activeView()) {
          @case ('LIST') {
            <app-work-item-list></app-work-item-list>
          }
          @case ('BOARD') {
            <app-work-item-board class="flex-1 min-h-0"></app-work-item-board>
          }
          @case ('CALENDAR') {
            <app-work-item-calendar></app-work-item-calendar>
          }
          @case ('TIMELINE') {
            <app-work-item-timeline></app-work-item-timeline>
          }
          @case ('SPREADSHEET') {
            <app-work-item-spreadsheet></app-work-item-spreadsheet>
          }
        }
      </div>

      <!-- Slide-over Work Item Detail Drawer -->
      <app-work-item-detail-drawer></app-work-item-detail-drawer>
    </div>
  `,
})
export class WorkItemsShellComponent {
  readonly workItemService = inject(WorkItemService);
  readonly projectService = inject(ProjectService);
  readonly settingsService = inject(SettingsService);

  readonly isColumnsPopoverOpen = signal<boolean>(false);
  readonly allStatuses = ALL_STATUSES;

  readonly views: { mode: WorkItemViewMode; label: string }[] = [
    { mode: 'LIST', label: 'List' },
    { mode: 'BOARD', label: 'Board' },
    { mode: 'CALENDAR', label: 'Calendar' },
    { mode: 'TIMELINE', label: 'Timeline' },
    { mode: 'SPREADSHEET', label: 'Spreadsheet' },
  ];

  readonly statusFilterOptions: DropdownOption[] = [
    { value: '', label: 'All Statuses' },
    ...ALL_STATUSES.map((s) => ({ value: s.value, label: s.label })),
  ];

  readonly priorityFilterOptions: DropdownOption[] = [
    { value: '', label: 'All Priorities' },
    ...ALL_PRIORITIES.map((p) => ({ value: p.value, label: p.label })),
  ];

  readonly cycleFilterOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'All Cycles' },
    ...this.projectService.cycles().map((c) => ({ value: c.id, label: c.name })),
  ]);

  readonly moduleFilterOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'All Modules' },
    ...this.projectService.modules().map((m) => ({ value: m.id, label: m.name })),
  ]);

  readonly epicFilterOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'All Epics' },
    ...this.projectService.epics().map((e) => ({ value: e.id, label: e.name })),
  ]);

  onStatusFilterChange(val: string): void {
    this.workItemService.updateFilter({ status: val as WorkItemStatus | '' });
  }

  onPriorityFilterChange(val: string): void {
    this.workItemService.updateFilter({ priority: val as WorkItemPriority | '' });
  }

  hasActiveFilters(): boolean {
    const f = this.workItemService.filter();
    return Boolean(
      f.status ||
        f.priority ||
        f.module_id ||
        f.epic_id ||
        f.cycle_id ||
        f.assignee_id ||
        f.label_id ||
        f.search
    );
  }
}
