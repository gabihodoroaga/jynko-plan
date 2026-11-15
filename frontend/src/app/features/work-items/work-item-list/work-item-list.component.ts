import { Component, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { WorkItemService } from '../../../core/services/work-item.service';
import { ProjectService } from '../../../core/services/project.service';
import { SettingsService } from '../../../core/services/settings.service';
import {
  ALL_PRIORITIES,
  ALL_STATUSES,
  WorkItem,
  WorkItemPriority,
  WorkItemStatus,
} from '../../../core/models/project.model';
import { StatusBadgeComponent } from '../../../shared/components/status-badge.component';
import { PriorityBadgeComponent } from '../../../shared/components/priority-badge.component';
import { DropdownComponent, DropdownOption } from '../../../shared/components/dropdown.component';

@Component({
  selector: 'app-work-item-list',
  standalone: true,
  imports: [CommonModule, FormsModule, StatusBadgeComponent, PriorityBadgeComponent, DropdownComponent],
  template: `
    <div class="space-y-4">
      @for (group of groupedItems(); track group.status) {
        <div class="bg-white border border-slate-200 rounded-md shadow-card">
          <div class="flex items-center justify-between px-3.5 py-2 bg-slate-50 border-b border-slate-200 rounded-t-md">
            <div class="flex items-center gap-2">
              <app-status-badge [status]="group.status"></app-status-badge>
              <span class="text-[11px] font-mono text-slate-400">{{ group.items.length }}</span>
            </div>
          </div>

          @if (group.items.length === 0) {
            <div class="px-4 py-3 text-xs text-slate-400">No work items in {{ group.label }}.</div>
          } @else {
            <div class="divide-y divide-slate-200">
              @for (item of group.items; track item.id) {
                <div
                  [attr.data-item-key]="item.key"
                  class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 sm:gap-3 px-3.5 hover:bg-slate-100/70 transition-colors cursor-pointer"
                  [ngClass]="settingsService.generalSettings().compactTableRows ? 'py-1.5' : 'py-2.5'"
                  (click)="workItemService.openItemDetail(item.id)"
                >
                  <div class="flex items-center gap-2.5 sm:gap-3 min-w-0 flex-1">
                    <span class="font-mono text-[11px] font-medium text-slate-400 shrink-0 w-16">
                      {{ item.key }}
                    </span>
                    <span class="text-xs font-medium text-slate-900 truncate">
                      {{ item.title }}
                    </span>
                    @if (item.sub_item_count > 0) {
                      <span
                        class="px-1.5 py-0.5 rounded bg-slate-100 border border-slate-200 text-[10px] font-mono text-slate-600 shrink-0"
                      >
                        ✓ {{ item.completed_sub_count }}/{{ item.sub_item_count }}
                      </span>
                    }
                  </div>

                  <div class="flex flex-wrap items-center gap-1.5 sm:gap-2 shrink-0" (click)="$event.stopPropagation()">
                    @if (item.estimate !== undefined && item.estimate !== null) {
                      <span
                        class="px-1.5 py-0.5 rounded bg-slate-100 border border-slate-200 text-[11px] font-mono text-slate-600"
                        title="Story points estimate"
                      >
                        {{ item.estimate }} pts
                      </span>
                    }

                    @if (item.due_date) {
                      <span
                        class="px-2 py-0.5 rounded-md bg-slate-50 border border-slate-200 text-[11px] text-slate-600"
                      >
                        Due {{ item.due_date | date: 'MMM d' }}
                      </span>
                    }

                    <app-dropdown
                      [options]="priorityOptions"
                      [value]="item.priority"
                      (valueChange)="onPriorityChange(item, $event)"
                      [disabled]="!projectService.canEdit()"
                      size="sm"
                      align="right"
                      ariaLabel="Priority"
                    ></app-dropdown>

                    <app-dropdown
                      [options]="statusOptions"
                      [value]="item.status"
                      (valueChange)="onStatusChange(item, $event)"
                      [disabled]="!projectService.canEdit()"
                      size="sm"
                      align="right"
                      ariaLabel="Status"
                    ></app-dropdown>

                    <span
                      class="px-2 py-0.5 rounded-full bg-slate-100 border border-slate-200 text-[11px] text-slate-600"
                    >
                      {{ memberName(item.assignee_id) }}
                    </span>
                  </div>
                </div>
              }
            </div>
          }
        </div>
      }
    </div>
  `,
})
export class WorkItemListComponent {
  readonly workItemService = inject(WorkItemService);
  readonly projectService = inject(ProjectService);
  readonly settingsService = inject(SettingsService);

  readonly statuses = ALL_STATUSES;
  readonly priorities = ALL_PRIORITIES;

  readonly statusOptions: DropdownOption[] = ALL_STATUSES.map((s) => ({
    value: s.value,
    label: s.label,
  }));
  readonly priorityOptions: DropdownOption[] = ALL_PRIORITIES.map((p) => ({
    value: p.value,
    label: p.label,
  }));

  readonly groupedItems = computed(() => {
    const items = this.workItemService.filteredItems();
    return ALL_STATUSES.map((st) => ({
      status: st.value,
      label: st.label,
      items: items.filter((i) => i.status === st.value),
    }));
  });

  onStatusChange(item: WorkItem, status: string): void {
    this.workItemService.updateItem(item.id, { status: status as WorkItemStatus });
  }

  onPriorityChange(item: WorkItem, priority: string): void {
    this.workItemService.updateItem(item.id, { priority: priority as WorkItemPriority });
  }

  memberName(userId?: string): string {
    if (!userId) return 'Unassigned';
    const found = this.projectService.members().find((m) => m.user_id === userId);
    return found ? found.name : userId;
  }
}
