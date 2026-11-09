import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { WorkItemService } from '../../core/services/work-item.service';
import { ProjectService } from '../../core/services/project.service';
import { AuthService } from '../../core/services/auth.service';
import {
  ALL_PRIORITIES,
  ALL_STATUSES,
  Project,
  WorkItem,
  WorkItemStatus,
} from '../../core/models/project.model';
import { StatusBadgeComponent } from '../../shared/components/status-badge.component';
import { PriorityBadgeComponent } from '../../shared/components/priority-badge.component';
import { WorkItemDetailDrawerComponent } from '../work-items/work-item-detail-drawer/work-item-detail-drawer.component';
import { DropdownComponent, DropdownOption } from '../../shared/components/dropdown.component';

interface ProjectWorkGroup {
  project: Project;
  items: WorkItem[];
}

interface StatusWorkGroup {
  status: WorkItemStatus;
  label: string;
  items: WorkItem[];
}

@Component({
  selector: 'app-my-work',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    StatusBadgeComponent,
    PriorityBadgeComponent,
    WorkItemDetailDrawerComponent,
    DropdownComponent,
  ],
  template: `
    <div class="w-full p-4 sm:p-6 space-y-6 bg-slate-50 min-h-full">
      <!-- Header & Grouping Controls -->
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 class="text-lg font-semibold text-slate-900">My Work</h1>
          <p class="text-xs text-slate-500 mt-0.5">
            Work items assigned to <span class="font-medium text-slate-700">{{ authService.currentUser().name }}</span> across all <span class="font-mono font-medium text-slate-700">{{ projectService.projects().length }}</span> projects
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2.5">
          <!-- Styled Status filter dropdown -->
          <app-dropdown
            [options]="statusFilterOptions"
            [value]="statusFilter()"
            (valueChange)="statusFilter.set($event)"
            ariaLabel="Filter My Work by Status"
          ></app-dropdown>

          <!-- Styled Priority filter dropdown -->
          <app-dropdown
            [options]="priorityFilterOptions"
            [value]="priorityFilter()"
            (valueChange)="priorityFilter.set($event)"
            ariaLabel="Filter My Work by Priority"
          ></app-dropdown>

          <!-- Group By Switcher: Project vs Status -->
          <div class="inline-flex items-center bg-slate-100 p-0.5 rounded-md border border-slate-200">
            <button
              type="button"
              (click)="groupBy.set('PROJECT')"
              class="px-2.5 py-1 rounded text-xs font-medium transition-colors"
              [ngClass]="
                groupBy() === 'PROJECT'
                  ? 'bg-white text-slate-900 shadow-card border border-slate-200/80'
                  : 'text-slate-600 hover:text-slate-900'
              "
            >
              Group by Project
            </button>
            <button
              type="button"
              (click)="groupBy.set('STATUS')"
              class="px-2.5 py-1 rounded text-xs font-medium transition-colors"
              [ngClass]="
                groupBy() === 'STATUS'
                  ? 'bg-white text-slate-900 shadow-card border border-slate-200/80'
                  : 'text-slate-600 hover:text-slate-900'
              "
            >
              Group by Status
            </button>
          </div>
        </div>
      </div>

      <!-- Personal Cross-Project KPI Counter Cards -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3.5">
        <div class="bg-white border border-slate-200 rounded-lg p-4 shadow-card">
          <div class="text-[11px] font-medium text-slate-500">Total Assigned to Me</div>
          <div class="text-2xl font-semibold text-slate-900 mt-1 font-mono">
            {{ allMyItems().length }}
          </div>
        </div>
        <div class="bg-white border border-slate-200 rounded-lg p-4 shadow-card">
          <div class="text-[11px] font-medium text-slate-500">In Progress</div>
          <div class="text-2xl font-semibold text-amber-600 mt-1 font-mono">
            {{ inProgressCount() }}
          </div>
        </div>
        <div class="bg-white border border-slate-200 rounded-lg p-4 shadow-card">
          <div class="text-[11px] font-medium text-slate-500">In Review / Todo</div>
          <div class="text-2xl font-semibold text-sky-600 mt-1 font-mono">
            {{ openAndReviewCount() }}
          </div>
        </div>
        <div class="bg-white border border-slate-200 rounded-lg p-4 shadow-card">
          <div class="text-[11px] font-medium text-slate-500">Completed</div>
          <div class="text-2xl font-semibold text-emerald-600 mt-1 font-mono">
            {{ doneCount() }}
          </div>
        </div>
      </div>

      <!-- Mode 1: Grouped by Project -->
      @if (groupBy() === 'PROJECT') {
        <div class="space-y-4">
          @for (group of projectGroups(); track group.project.id) {
            <div class="bg-white border border-slate-200 rounded-lg shadow-card overflow-hidden">
              <!-- Project Group Header -->
              <div class="px-4 py-3 bg-slate-50/80 border-b border-slate-200 flex items-center justify-between">
                <div class="flex items-center gap-2.5">
                  <span class="text-sm">👇</span>
                  <span class="text-xs font-semibold text-slate-900">{{ group.project.name }}</span>
                  <span class="px-1.5 py-0.5 rounded bg-white border border-slate-200 font-mono text-[10px] font-semibold text-slate-600">
                    {{ group.project.key_prefix }}
                  </span>
                  <span class="px-2 py-0.5 rounded-full bg-slate-200/70 text-slate-700 font-mono text-[10px] font-medium">
                    {{ group.items.length }} assigned
                  </span>
                </div>

                <button
                  type="button"
                  (click)="openProjectBoard(group.project.id)"
                  class="text-[11px] font-medium text-sky-600 hover:underline"
                >
                  Open {{ group.project.name }} Board →
                </button>
              </div>

              <!-- Assigned Work Items in this Project -->
              <div class="divide-y divide-slate-200">
                @for (item of group.items; track item.id) {
                  <div
                    (click)="openItem(item)"
                    class="px-4 py-2.5 flex flex-col sm:flex-row sm:items-center justify-between gap-2 sm:gap-3 hover:bg-slate-50 cursor-pointer transition-colors"
                  >
                    <div class="flex items-center gap-3 min-w-0">
                      <span class="font-mono text-[11px] text-slate-400 w-16 shrink-0">
                        {{ item.key }}
                      </span>
                      <span class="text-xs font-medium text-slate-900 truncate">
                        {{ item.title }}
                      </span>
                    </div>

                    <div class="flex flex-wrap items-center gap-2 sm:gap-2.5 shrink-0 self-start sm:self-auto" (click)="$event.stopPropagation()">
                      @if (item.due_date) {
                        <span class="text-[11px] font-mono text-slate-500 hidden sm:inline">
                          Due {{ item.due_date | date: 'MMM d' }}
                        </span>
                      }
                      @if (item.estimate) {
                        <span class="px-1.5 py-0.5 rounded bg-slate-100 border border-slate-200 font-mono text-[10px] text-slate-600">
                          {{ item.estimate }}pt
                        </span>
                      }
                      <app-priority-badge [priority]="item.priority"></app-priority-badge>
                      <app-status-badge [status]="item.status"></app-status-badge>
                    </div>
                  </div>
                } @empty {
                  <div class="px-4 py-6 text-center text-xs text-slate-400">
                    No matching work items assigned to you in {{ group.project.name }}.
                  </div>
                }
              </div>
            </div>
          }
        </div>
      } @else {
        <!-- Mode 2: Grouped by Status Across All Projects -->
        <div class="space-y-4">
          @for (group of statusGroups(); track group.status) {
            <div class="bg-white border border-slate-200 rounded-lg shadow-card overflow-hidden">
              <div class="px-4 py-2.5 bg-slate-50/80 border-b border-slate-200 flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <app-status-badge [status]="group.status"></app-status-badge>
                  <span class="font-mono text-[11px] text-slate-500">({{ group.items.length }})</span>
                </div>
              </div>

              <div class="divide-y divide-slate-200">
                @for (item of group.items; track item.id) {
                  <div
                    (click)="openItem(item)"
                    class="px-4 py-2.5 flex flex-col sm:flex-row sm:items-center justify-between gap-2 sm:gap-3 hover:bg-slate-50 cursor-pointer transition-colors"
                  >
                    <div class="flex items-center gap-2.5 min-w-0">
                      <span class="px-1.5 py-0.5 rounded bg-slate-100 border border-slate-200 font-mono text-[10px] font-medium text-slate-600 shrink-0">
                        {{ getProjectName(item.project_id) }}
                      </span>
                      <span class="font-mono text-[11px] text-slate-400 w-16 shrink-0">
                        {{ item.key }}
                      </span>
                      <span class="text-xs font-medium text-slate-900 truncate">
                        {{ item.title }}
                      </span>
                    </div>

                    <div class="flex flex-wrap items-center gap-2 sm:gap-2.5 shrink-0 self-start sm:self-auto" (click)="$event.stopPropagation()">
                      @if (item.due_date) {
                        <span class="text-[11px] font-mono text-slate-500 hidden sm:inline">
                          Due {{ item.due_date | date: 'MMM d' }}
                        </span>
                      }
                      <app-priority-badge [priority]="item.priority"></app-priority-badge>
                      <app-status-badge [status]="item.status"></app-status-badge>
                    </div>
                  </div>
                }
              </div>
            </div>
          } @empty {
            <div class="bg-white border border-slate-200 rounded-lg p-8 text-center text-xs text-slate-400">
              No work items assigned to you match the selected filters.
            </div>
          }
        </div>
      }

      <!-- Slide-over Work Item Detail Drawer -->
      <app-work-item-detail-drawer></app-work-item-detail-drawer>
    </div>
  `,
})
export class MyWorkComponent {
  readonly workItemService = inject(WorkItemService);
  readonly projectService = inject(ProjectService);
  readonly authService = inject(AuthService);
  private readonly router = inject(Router);

  readonly groupBy = signal<'PROJECT' | 'STATUS'>('PROJECT');
  readonly statusFilter = signal<string>('');
  readonly priorityFilter = signal<string>('');

  readonly statuses = ALL_STATUSES;
  readonly priorities = ALL_PRIORITIES;

  readonly statusFilterOptions: DropdownOption[] = [
    { value: '', label: 'All Statuses' },
    ...ALL_STATUSES.map((s) => ({ value: s.value, label: s.label })),
  ];
  readonly priorityFilterOptions: DropdownOption[] = [
    { value: '', label: 'All Priorities' },
    ...ALL_PRIORITIES.map((p) => ({ value: p.value, label: p.label })),
  ];

  readonly allMyItems = computed(() => this.workItemService.myAssignedItems());

  readonly filteredMyItems = computed(() => {
    const sf = this.statusFilter();
    const pf = this.priorityFilter();
    return this.allMyItems().filter((item) => {
      if (sf && item.status !== sf) return false;
      if (pf && item.priority !== pf) return false;
      return true;
    });
  });

  readonly inProgressCount = computed(
    () => this.allMyItems().filter((i) => i.status === 'IN_PROGRESS').length
  );
  readonly openAndReviewCount = computed(
    () =>
      this.allMyItems().filter(
        (i) => i.status === 'IN_REVIEW' || i.status === 'TODO' || i.status === 'BACKLOG'
      ).length
  );
  readonly doneCount = computed(
    () => this.allMyItems().filter((i) => i.status === 'DONE').length
  );

  readonly projectGroups = computed<ProjectWorkGroup[]>(() => {
    const items = this.filteredMyItems();
    return this.projectService.projects().map((project) => ({
      project,
      items: items.filter((i) => i.project_id === project.id),
    }));
  });

  readonly statusGroups = computed<StatusWorkGroup[]>(() => {
    const items = this.filteredMyItems();
    return this.statuses
      .map((s) => ({
        status: s.value,
        label: s.label,
        items: items.filter((i) => i.status === s.value),
      }))
      .filter((g) => g.items.length > 0);
  });

  getProjectName(projectId: string): string {
    return this.projectService.getProjectById(projectId)?.name ?? projectId;
  }

  openItem(item: WorkItem): void {
    this.projectService.selectProject(item.project_id);
    this.workItemService.openItemDetail(item.id);
  }

  openProjectBoard(projectId: string): void {
    this.projectService.selectProject(projectId);
    this.workItemService.resetFilters();
    this.router.navigate(['/projects', projectId, 'work-items']);
  }
}
