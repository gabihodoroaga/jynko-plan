import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { WorkItemService } from '../../../core/services/work-item.service';
import { ProjectService } from '../../../core/services/project.service';
import {
  ALL_PRIORITIES,
  ALL_STATUSES,
  WorkItem,
  WorkItemPriority,
  WorkItemStatus,
} from '../../../core/models/project.model';

@Component({
  selector: 'app-work-item-spreadsheet',
  standalone: true,
  imports: [CommonModule, FormsModule],
  template: `
    <div class="bg-white border border-slate-200 rounded-md shadow-card overflow-hidden">
      <div
        tabindex="0"
        role="region"
        aria-labelledby="spreadsheet-table-caption"
        class="table-wrapper"
      >
        <table class="w-full text-left text-xs">
          <caption
            id="spreadsheet-table-caption"
            class="px-4 py-2 text-left text-[11px] font-medium text-slate-500 bg-slate-50 border-b border-slate-200"
          >
            Work Items Spreadsheet Grid — Inline edit titles, statuses, priorities, assignees, estimates, cycles, modules, epics, and dates.
          </caption>
          <thead class="bg-slate-50 text-[11px] font-semibold text-slate-600">
            <tr>
              <th class="px-3 py-2 border-b border-r border-slate-200 bg-slate-50 w-24">Key</th>
              <th class="px-3 py-2 border-b border-r border-slate-200 bg-slate-50 min-w-[260px]">Title</th>
              <th class="px-3 py-2 border-b border-r border-slate-200 bg-slate-50 w-32">Status</th>
              <th class="px-3 py-2 border-b border-r border-slate-200 bg-slate-50 w-28">Priority</th>
              <th class="px-3 py-2 border-b border-r border-slate-200 bg-slate-50 w-36">Assignee</th>
              <th class="px-3 py-2 border-b border-r border-slate-200 bg-slate-50 w-20">Est (pts)</th>
              <th class="px-3 py-2 border-b border-r border-slate-200 bg-slate-50 w-36">Module</th>
              <th class="px-3 py-2 border-b border-r border-slate-200 bg-slate-50 w-36">Epic</th>
              <th class="px-3 py-2 border-b border-r border-slate-200 bg-slate-50 w-36">Cycle</th>
              <th class="px-3 py-2 border-b border-r border-slate-200 bg-slate-50 w-32">Start Date</th>
              <th class="px-3 py-2 border-b border-slate-200 bg-slate-50 w-32">Due Date</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-200">
            @for (item of workItemService.filteredItems(); track item.id) {
              <tr class="hover:bg-slate-100/60 transition-colors">
                <!-- Sticky Row Header (Key) -->
                <th
                  scope="row"
                  (click)="workItemService.openItemDetail(item.id)"
                  class="px-3 py-2 border-r border-slate-200 font-mono text-[11px] font-medium text-sky-700 hover:underline cursor-pointer"
                >
                  {{ item.key }}
                </th>

                <!-- Inline Editable Title -->
                <td class="px-2 py-1 border-r border-slate-200">
                  <input
                    type="text"
                    [ngModel]="item.title"
                    [ngModelOptions]="{ updateOn: 'blur' }"
                    (ngModelChange)="updateField(item, { title: $event })"
                    (keydown.enter)="$any($event.target).blur()"
                    [disabled]="!projectService.canEdit()"
                    class="w-full px-1.5 py-1 text-xs text-slate-900 bg-transparent rounded focus:bg-white focus:outline-none focus:ring-1 focus:ring-slate-300"
                  />
                </td>

                <!-- Status -->
                <td class="px-2 py-1 border-r border-slate-200">
                  <select
                    [ngModel]="item.status"
                    (ngModelChange)="updateStatus(item, $event)"
                    [disabled]="!projectService.canEdit()"
                    class="w-full text-[11px] bg-transparent px-1 py-1 rounded focus:bg-white focus:outline-none focus:ring-1 focus:ring-slate-300"
                  >
                    @for (s of statuses; track s.value) {
                      <option [value]="s.value">{{ s.label }}</option>
                    }
                  </select>
                </td>

                <!-- Priority -->
                <td class="px-2 py-1 border-r border-slate-200">
                  <select
                    [ngModel]="item.priority"
                    (ngModelChange)="updatePriority(item, $event)"
                    [disabled]="!projectService.canEdit()"
                    class="w-full text-[11px] bg-transparent px-1 py-1 rounded focus:bg-white focus:outline-none focus:ring-1 focus:ring-slate-300"
                  >
                    @for (p of priorities; track p.value) {
                      <option [value]="p.value">{{ p.label }}</option>
                    }
                  </select>
                </td>

                <!-- Assignee -->
                <td class="px-2 py-1 border-r border-slate-200">
                  <select
                    [ngModel]="item.assignee_id || ''"
                    (ngModelChange)="updateField(item, { assignee_id: $event })"
                    [disabled]="!projectService.canEdit()"
                    class="w-full text-[11px] bg-transparent px-1 py-1 rounded focus:bg-white focus:outline-none focus:ring-1 focus:ring-slate-300"
                  >
                    <option value="">Unassigned</option>
                    @for (m of projectService.members(); track m.user_id) {
                      <option [value]="m.user_id">{{ m.name }}</option>
                    }
                  </select>
                </td>

                <!-- Estimate -->
                <td class="px-2 py-1 border-r border-slate-200">
                  <input
                    type="number"
                    min="0"
                    max="100"
                    [ngModel]="item.estimate ?? null"
                    [ngModelOptions]="{ updateOn: 'blur' }"
                    (ngModelChange)="updateEstimate(item, $event)"
                    (keydown.enter)="$any($event.target).blur()"
                    [disabled]="!projectService.canEdit()"
                    class="w-full px-1.5 py-1 font-mono text-[11px] text-slate-700 bg-transparent rounded focus:bg-white focus:outline-none focus:ring-1 focus:ring-slate-300"
                  />
                </td>

                <!-- Module -->
                <td class="px-2 py-1 border-r border-slate-200">
                  <select
                    [ngModel]="item.module_id || ''"
                    (ngModelChange)="updateField(item, { module_id: $event })"
                    [disabled]="!projectService.canEdit()"
                    class="w-full text-[11px] bg-transparent px-1 py-1 rounded focus:bg-white focus:outline-none focus:ring-1 focus:ring-slate-300"
                  >
                    <option value="">None</option>
                    @for (mod of projectService.modules(); track mod.id) {
                      <option [value]="mod.id">{{ mod.name }}</option>
                    }
                  </select>
                </td>

                <!-- Epic -->
                <td class="px-2 py-1 border-r border-slate-200">
                  <select
                    [ngModel]="item.epic_id || ''"
                    (ngModelChange)="updateField(item, { epic_id: $event })"
                    [disabled]="!projectService.canEdit()"
                    class="w-full text-[11px] bg-transparent px-1 py-1 rounded focus:bg-white focus:outline-none focus:ring-1 focus:ring-slate-300"
                  >
                    <option value="">None</option>
                    @for (ep of projectService.epics(); track ep.id) {
                      <option [value]="ep.id">{{ ep.name }}</option>
                    }
                  </select>
                </td>

                <!-- Cycle -->
                <td class="px-2 py-1 border-r border-slate-200">
                  <select
                    [ngModel]="item.cycle_id || ''"
                    (ngModelChange)="updateField(item, { cycle_id: $event })"
                    [disabled]="!projectService.canEdit()"
                    class="w-full text-[11px] bg-transparent px-1 py-1 rounded focus:bg-white focus:outline-none focus:ring-1 focus:ring-slate-300"
                  >
                    <option value="">None</option>
                    @for (cy of projectService.cycles(); track cy.id) {
                      <option [value]="cy.id">{{ cy.name }}</option>
                    }
                  </select>
                </td>

                <!-- Start Date -->
                <td class="px-2 py-1 border-r border-slate-200">
                  <input
                    type="date"
                    [ngModel]="toDateInput(item.start_date)"
                    (ngModelChange)="updateDate(item, 'start_date', $event)"
                    [disabled]="!projectService.canEdit()"
                    class="w-full text-[11px] font-mono text-slate-600 bg-transparent px-1 py-1 rounded focus:bg-white focus:outline-none"
                  />
                </td>

                <!-- Due Date -->
                <td class="px-2 py-1">
                  <input
                    type="date"
                    [ngModel]="toDateInput(item.due_date)"
                    (ngModelChange)="updateDate(item, 'due_date', $event)"
                    [disabled]="!projectService.canEdit()"
                    class="w-full text-[11px] font-mono text-slate-600 bg-transparent px-1 py-1 rounded focus:bg-white focus:outline-none"
                  />
                </td>
              </tr>
            }
          </tbody>
        </table>
      </div>
    </div>
  `,
})
export class WorkItemSpreadsheetComponent {
  readonly workItemService = inject(WorkItemService);
  readonly projectService = inject(ProjectService);

  readonly statuses = ALL_STATUSES;
  readonly priorities = ALL_PRIORITIES;

  updateField(item: WorkItem, patch: Partial<WorkItem>): void {
    this.workItemService.updateItem(item.id, patch);
  }

  updateStatus(item: WorkItem, status: WorkItemStatus): void {
    this.workItemService.updateItem(item.id, { status });
  }

  updatePriority(item: WorkItem, priority: WorkItemPriority): void {
    this.workItemService.updateItem(item.id, { priority });
  }

  updateEstimate(item: WorkItem, val: number | null): void {
    this.workItemService.updateItem(item.id, {
      estimate: val !== null && !isNaN(Number(val)) ? Number(val) : undefined,
    });
  }

  updateDate(item: WorkItem, field: 'start_date' | 'due_date', value: string): void {
    const iso = value ? new Date(value + 'T00:00:00Z').toISOString() : undefined;
    this.workItemService.updateItem(item.id, { [field]: iso });
  }

  toDateInput(iso?: string): string {
    if (!iso) return '';
    return iso.slice(0, 10);
  }
}
