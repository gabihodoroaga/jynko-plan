import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ProjectService } from '../../core/services/project.service';
import { WorkItemService } from '../../core/services/work-item.service';
import { Epic, EpicStatus } from '../../core/models/project.model';
import { ModalComponent } from '../../shared/components/modal.component';
import { DropdownComponent, DropdownOption } from '../../shared/components/dropdown.component';

@Component({
  selector: 'app-epics',
  standalone: true,
  imports: [CommonModule, FormsModule, ModalComponent, DropdownComponent],
  template: `
    <div class="w-full p-6 space-y-5 bg-slate-50 min-h-full">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-base font-semibold text-slate-900">Epics & Strategic Milestones</h2>
          <p class="text-xs text-slate-500">
            Time-bound strategic objectives grouping related work items in {{ projectService.activeProject()?.name }}
          </p>
        </div>
        @if (projectService.canEdit()) {
          <button
            type="button"
            aria-label="Create new epic"
            (click)="openCreateModal()"
            class="bg-sky-600 hover:bg-sky-700 text-white rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
          >
            + New Epic
          </button>
        }
      </div>

      <div class="space-y-3">
        @for (row of epicRows(); track row.epic.id) {
          <div
            [attr.data-epic-id]="row.epic.id"
            class="bg-white border border-slate-200 rounded-md p-4 shadow-card space-y-3"
          >
            <div class="flex flex-wrap items-center justify-between gap-2">
              <div class="flex items-center gap-2.5 min-w-0">
                <h3 class="text-sm font-semibold text-slate-900 truncate">{{ row.epic.name }}</h3>
                <app-dropdown
                  [options]="epicStatusOptions"
                  [value]="row.epic.status"
                  (valueChange)="onInlineStatusChange(row.epic.id, $event)"
                  [disabled]="!projectService.canEdit()"
                  [ariaLabel]="'Epic status for ' + row.epic.name"
                  size="sm"
                ></app-dropdown>
              </div>

              <div class="flex flex-wrap items-center gap-3">
                @if (row.epic.start_date || row.epic.target_date) {
                  <span class="text-[11px] font-mono text-slate-500">
                    @if (row.epic.start_date) {
                      <span>{{ row.epic.start_date | date: 'MMM d, yyyy' }}</span>
                    }
                    @if (row.epic.start_date && row.epic.target_date) {
                      <span class="mx-1">→</span>
                    }
                    @if (row.epic.target_date) {
                      <span>Target: {{ row.epic.target_date | date: 'MMM d, yyyy' }}</span>
                    }
                  </span>
                }
                <button
                  type="button"
                  (click)="viewEpicTimeline(row.epic.id)"
                  class="text-xs font-medium text-sky-600 hover:underline"
                >
                  Open in Timeline →
                </button>
                @if (projectService.canEdit()) {
                  @if (confirmDeleteId() === row.epic.id) {
                    <div class="inline-flex items-center gap-1.5">
                      <span class="text-[11px] text-slate-500">Delete?</span>
                      <button
                        type="button"
                        (click)="confirmDeleteEpic(row.epic.id)"
                        [attr.aria-label]="'Confirm delete epic ' + row.epic.name"
                        class="px-2 py-0.5 rounded bg-rose-600 hover:bg-rose-700 text-white text-[11px] font-medium"
                      >
                        Yes
                      </button>
                      <button
                        type="button"
                        (click)="confirmDeleteId.set(null)"
                        class="px-2 py-0.5 rounded border border-slate-200 bg-white hover:bg-slate-50 text-slate-600 text-[11px]"
                      >
                        Cancel
                      </button>
                    </div>
                  } @else {
                    <button
                      type="button"
                      (click)="openEditModal(row.epic)"
                      [attr.aria-label]="'Edit epic ' + row.epic.name"
                      class="text-[11px] font-medium text-slate-500 hover:text-sky-600 transition-colors"
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      (click)="confirmDeleteId.set(row.epic.id)"
                      [attr.aria-label]="'Delete epic ' + row.epic.name"
                      class="text-[11px] text-slate-400 hover:text-rose-600 transition-colors"
                    >
                      Delete
                    </button>
                  }
                }
              </div>
            </div>

            <p class="text-xs text-slate-600">{{ row.epic.description || 'No description provided.' }}</p>

            <div class="space-y-1.5 pt-2 border-t border-slate-200">
              <div class="flex items-center justify-between text-[11px] text-slate-600">
                <span>Linked Work Items: {{ row.doneCount }}/{{ row.totalCount }} Done</span>
                <span class="font-mono font-medium">{{ row.percent }}%</span>
              </div>
              <div class="w-full h-1.5 bg-slate-100 rounded-full overflow-hidden">
                <div
                  class="h-full bg-emerald-600 transition-all"
                  [style.width.%]="row.percent"
                ></div>
              </div>
            </div>
          </div>
        } @empty {
          <div class="bg-white border border-slate-200 rounded-md p-8 text-center text-xs text-slate-400">
            No strategic epics defined yet for this project. Click <strong>+ New Epic</strong> to create one.
          </div>
        }
      </div>

      <!-- Create Strategic Epic Modal -->
      <app-modal
        [isOpen]="isCreateOpen()"
        title="Create Strategic Epic"
        (closed)="isCreateOpen.set(false)"
      >
        <form (ngSubmit)="submitCreate()" class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Epic Name</label>
            <input
              type="text"
              name="name"
              [(ngModel)]="newName"
              required
              aria-label="New epic name"
              placeholder="e.g., Enterprise SSO & Audit Logging"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Description</label>
            <textarea
              rows="2"
              name="description"
              [(ngModel)]="newDesc"
              aria-label="New epic description"
              placeholder="High-level scope and measurable outcomes..."
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            ></textarea>
          </div>
          <div class="grid grid-cols-3 gap-2.5">
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">Status</label>
              <app-dropdown
                [options]="epicStatusOptions"
                [value]="newStatus"
                (valueChange)="onEpicStatusChange($event)"
                [fullWidth]="true"
              ></app-dropdown>
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">
                Start Date <span class="text-slate-400 font-normal">(optional)</span>
              </label>
              <input
                type="date"
                name="startDate"
                [(ngModel)]="newStart"
                aria-label="New epic start date"
                class="w-full px-2 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">
                Target Date <span class="text-slate-400 font-normal">(optional)</span>
              </label>
              <input
                type="date"
                name="targetDate"
                [(ngModel)]="newTarget"
                [min]="newStart"
                aria-label="New epic target date"
                class="w-full px-2 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
              />
            </div>
          </div>
          @if (dateValidationError()) {
            <p class="text-[11px] text-rose-600">{{ dateValidationError() }}</p>
          }
          <div class="flex justify-end gap-2 pt-2 border-t border-slate-200">
            <button
              type="button"
              (click)="isCreateOpen.set(false)"
              class="px-3 py-1.5 rounded-md border border-slate-200 text-xs text-slate-600 hover:bg-slate-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              aria-label="Submit create epic"
              class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
            >
              Create Epic
            </button>
          </div>
        </form>
      </app-modal>

      <!-- Edit Strategic Epic Modal -->
      <app-modal
        [isOpen]="isEditOpen()"
        title="Edit Strategic Epic"
        (closed)="closeEditModal()"
      >
        <form (ngSubmit)="submitEdit()" class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Epic Name</label>
            <input
              type="text"
              name="editName"
              [(ngModel)]="editName"
              required
              aria-label="Edit epic name"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Description</label>
            <textarea
              rows="2"
              name="editDescription"
              [(ngModel)]="editDesc"
              aria-label="Edit epic description"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            ></textarea>
          </div>
          <div class="grid grid-cols-3 gap-2.5">
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">Status</label>
              <app-dropdown
                [options]="epicStatusOptions"
                [value]="editStatus"
                (valueChange)="onEditStatusChange($event)"
                [fullWidth]="true"
              ></app-dropdown>
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">
                Start Date <span class="text-slate-400 font-normal">(optional)</span>
              </label>
              <input
                type="date"
                name="editStartDate"
                [(ngModel)]="editStart"
                aria-label="Edit epic start date"
                class="w-full px-2 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">
                Target Date <span class="text-slate-400 font-normal">(optional)</span>
              </label>
              <input
                type="date"
                name="editTargetDate"
                [(ngModel)]="editTarget"
                [min]="editStart"
                aria-label="Edit epic target date"
                class="w-full px-2 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
              />
            </div>
          </div>
          @if (dateValidationError()) {
            <p class="text-[11px] text-rose-600">{{ dateValidationError() }}</p>
          }
          <div class="flex justify-end gap-2 pt-2 border-t border-slate-200">
            <button
              type="button"
              (click)="closeEditModal()"
              class="px-3 py-1.5 rounded-md border border-slate-200 text-xs text-slate-600 hover:bg-slate-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              aria-label="Save epic changes"
              class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
            >
              Save Changes
            </button>
          </div>
        </form>
      </app-modal>
    </div>
  `,
})
export class EpicsComponent {
  readonly projectService = inject(ProjectService);
  readonly workItemService = inject(WorkItemService);
  private readonly router = inject(Router);

  readonly isCreateOpen = signal(false);
  readonly isEditOpen = signal(false);
  readonly editingEpicId = signal<string | null>(null);
  readonly confirmDeleteId = signal<string | null>(null);
  readonly dateValidationError = signal<string>('');

  newName = '';
  newDesc = '';
  newStatus: EpicStatus = 'PLANNED';
  newStart = '';
  newTarget = '';

  editName = '';
  editDesc = '';
  editStatus: EpicStatus = 'PLANNED';
  editStart = '';
  editTarget = '';

  readonly epicStatusOptions: DropdownOption[] = [
    { value: 'PLANNED', label: 'Planned', dotColor: '#64748b' },
    { value: 'IN_PROGRESS', label: 'In Progress', dotColor: '#d97706' },
    { value: 'COMPLETED', label: 'Completed', dotColor: '#059669' },
  ];

  private formatDateInput(iso?: string): string {
    if (!iso) return '';
    return iso.slice(0, 10);
  }

  onEpicStatusChange(val: string): void {
    this.newStatus = val as EpicStatus;
  }

  onEditStatusChange(val: string): void {
    this.editStatus = val as EpicStatus;
  }

  onInlineStatusChange(epicId: string, val: string): void {
    this.projectService.updateEpic(epicId, { status: val as EpicStatus });
  }

  openCreateModal(): void {
    this.newName = '';
    this.newDesc = '';
    this.newStatus = 'PLANNED';
    this.newStart = '';
    this.newTarget = '';
    this.dateValidationError.set('');
    this.isCreateOpen.set(true);
  }

  openEditModal(epic: Epic): void {
    this.editingEpicId.set(epic.id);
    this.editName = epic.name;
    this.editDesc = epic.description || '';
    this.editStatus = epic.status;
    this.editStart = this.formatDateInput(epic.start_date);
    this.editTarget = this.formatDateInput(epic.target_date);
    this.dateValidationError.set('');
    this.isEditOpen.set(true);
  }

  closeEditModal(): void {
    this.isEditOpen.set(false);
    this.editingEpicId.set(null);
    this.dateValidationError.set('');
  }

  confirmDeleteEpic(epicId: string): void {
    this.projectService.deleteEpic(epicId);
    this.workItemService.items.update((list) =>
      list.map((item) => (item.epic_id === epicId ? { ...item, epic_id: '' } : item))
    );
    if (this.workItemService.filter().epic_id === epicId) {
      this.workItemService.updateFilter({ epic_id: '' });
    }
    this.confirmDeleteId.set(null);
  }

  readonly epicRows = computed(() => {
    const items = this.workItemService.items();
    return this.projectService.epics().map((epic) => {
      const linked = items.filter((i) => i.epic_id === epic.id);
      const done = linked.filter((i) => i.status === 'DONE').length;
      const percent = linked.length > 0 ? Math.round((done / linked.length) * 100) : 0;
      return {
        epic,
        totalCount: linked.length,
        doneCount: done,
        percent,
      };
    });
  });

  viewEpicTimeline(epicId: string): void {
    this.workItemService.updateFilter({ epic_id: epicId });
    this.workItemService.activeView.set('TIMELINE');
    this.router.navigate(['/projects', this.projectService.activeProjectId(), 'work-items']);
  }

  submitCreate(): void {
    const trimmedName = this.newName.trim();
    if (!trimmedName) return;
    if (this.newStart && this.newTarget && this.newTarget < this.newStart) {
      this.dateValidationError.set('Target date must be on or after the start date.');
      return;
    }
    this.dateValidationError.set('');
    this.projectService.createEpic(
      trimmedName,
      this.newDesc.trim(),
      this.newStatus,
      this.newStart || undefined,
      this.newTarget || undefined
    );
    this.newName = '';
    this.newDesc = '';
    this.isCreateOpen.set(false);
  }

  submitEdit(): void {
    const epicId = this.editingEpicId();
    const trimmedName = this.editName.trim();
    if (!epicId || !trimmedName) return;
    if (this.editStart && this.editTarget && this.editTarget < this.editStart) {
      this.dateValidationError.set('Target date must be on or after the start date.');
      return;
    }
    this.dateValidationError.set('');
    this.projectService.updateEpic(epicId, {
      name: trimmedName,
      description: this.editDesc.trim(),
      status: this.editStatus,
      start_date: this.editStart,
      target_date: this.editTarget,
    });
    this.closeEditModal();
  }
}
