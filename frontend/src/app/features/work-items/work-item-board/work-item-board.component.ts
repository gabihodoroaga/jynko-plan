import { Component, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import {
  CdkDragDrop,
  DragDropModule,
} from '@angular/cdk/drag-drop';
import { WorkItemService } from '../../../core/services/work-item.service';
import { ProjectService } from '../../../core/services/project.service';
import { SettingsService } from '../../../core/services/settings.service';
import {
  ALL_STATUSES,
  WorkItem,
  WorkItemStatus,
} from '../../../core/models/project.model';
import { StatusBadgeComponent } from '../../../shared/components/status-badge.component';
import { PriorityBadgeComponent } from '../../../shared/components/priority-badge.component';

@Component({
  selector: 'app-work-item-board',
  standalone: true,
  imports: [CommonModule, DragDropModule, StatusBadgeComponent, PriorityBadgeComponent],
  host: {
    class: 'block h-full',
  },
  template: `
    <div
      cdkDropListGroup
      class="flex gap-3.5 overflow-x-auto pb-2 items-stretch h-full min-h-[calc(100vh-10rem)]"
    >
      @for (col of columns(); track col.status) {
        <div
          [attr.data-board-column]="col.status"
          class="board-column w-72 shrink-0 h-full bg-slate-50 border border-slate-200 rounded-md flex flex-col"
        >
          <!-- Column Header -->
          <div class="flex items-center justify-between px-3 py-2.5 border-b border-slate-200 bg-white rounded-t-md shrink-0">
            <div class="flex items-center gap-2">
              <app-status-badge [status]="col.status"></app-status-badge>
              <span class="text-[11px] font-mono text-slate-400">{{ col.items.length }}</span>
            </div>
          </div>

          <!-- Cards Drop List -->
          <div
            cdkDropList
            [cdkDropListData]="col.status"
            (cdkDropListDropped)="onDrop($event, col.status)"
            class="p-2.5 space-y-2.5 overflow-y-auto flex-1 min-h-0"
          >
            @for (item of col.items; track item.id) {
              <div
                cdkDrag
                [cdkDragData]="item"
                [cdkDragDisabled]="!projectService.canEdit()"
                [attr.data-item-key]="item.key"
                (click)="workItemService.openItemDetail(item.id)"
                class="bg-white border border-slate-200 rounded-md p-3 shadow-card hover:border-slate-300 transition-colors cursor-pointer select-none"
              >
                <div class="flex items-center justify-between mb-1.5">
                  <span class="font-mono text-[11px] font-medium text-slate-400">
                    {{ item.key }}
                  </span>
                  @if (item.estimate !== undefined && item.estimate !== null) {
                    <span class="px-1.5 py-0.5 rounded bg-slate-100 border border-slate-200 text-[10px] font-mono text-slate-600">
                      {{ item.estimate }}p
                    </span>
                  }
                </div>

                <h4 class="text-xs font-medium text-slate-900 leading-snug mb-2.5">
                  {{ item.title }}
                </h4>

                <div class="flex flex-wrap items-center gap-1.5">
                  <app-priority-badge [priority]="item.priority"></app-priority-badge>

                  @if (item.due_date) {
                    <span
                      class="px-1.5 py-0.5 rounded-md bg-slate-50 border border-slate-200 text-[11px] text-slate-600"
                    >
                      {{ item.due_date | date: 'MMM d' }}
                    </span>
                  }

                  @if (item.sub_item_count > 0) {
                    <span
                      class="px-1.5 py-0.5 rounded-md bg-slate-50 border border-slate-200 text-[11px] font-mono text-slate-600"
                    >
                      ✓ {{ item.completed_sub_count }}/{{ item.sub_item_count }}
                    </span>
                  }

                  @for (lblId of item.label_ids; track lblId) {
                    @if (getLabel(lblId); as lbl) {
                      <span
                        class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-md border border-slate-200 bg-slate-50 text-[11px] text-slate-600"
                      >
                        <span
                          class="w-1.5 h-1.5 rounded-full"
                          [style.backgroundColor]="lbl.color"
                        ></span>
                        {{ lbl.name }}
                      </span>
                    }
                  }
                </div>
              </div>
            }
          </div>
        </div>
      }
    </div>
  `,
})
export class WorkItemBoardComponent {
  readonly workItemService = inject(WorkItemService);
  readonly projectService = inject(ProjectService);
  readonly settingsService = inject(SettingsService);

  readonly columns = computed(() => {
    const list = this.workItemService.filteredItems();
    const visibleSet = new Set(this.settingsService.visibleBoardStatuses());
    return ALL_STATUSES.filter((st) => visibleSet.has(st.value)).map((st) => ({
      status: st.value,
      label: st.label,
      items: list.filter((i) => i.status === st.value),
    }));
  });

  onDrop(event: CdkDragDrop<WorkItemStatus>, targetStatus: WorkItemStatus): void {
    const item = event.item.data as WorkItem | undefined;
    if (!item || item.status === targetStatus) return;
    this.workItemService.updateItem(item.id, { status: targetStatus });
  }

  getLabel(labelId: string) {
    return this.projectService.labels().find((l) => l.id === labelId);
  }
}
