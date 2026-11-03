import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ProjectService } from '../../core/services/project.service';
import { WorkItemService } from '../../core/services/work-item.service';
import { Cycle } from '../../core/models/project.model';
import { ModalComponent } from '../../shared/components/modal.component';

@Component({
  selector: 'app-cycles',
  standalone: true,
  imports: [CommonModule, FormsModule, ModalComponent],
  template: `
    <div class="w-full p-6 space-y-5 bg-slate-50 min-h-full">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-base font-semibold text-slate-900">Sprint Cycles</h2>
          <p class="text-xs text-slate-500">
            Time-boxed iterations for scheduled execution in {{ projectService.activeProject()?.name }}
          </p>
        </div>
        @if (projectService.canEdit()) {
          <button
            type="button"
            aria-label="Create new cycle"
            (click)="openCreateModal()"
            class="bg-sky-600 hover:bg-sky-700 text-white rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
          >
            + New Cycle
          </button>
        }
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        @for (row of cycleRows(); track row.cycle.id) {
          <div
            [attr.data-cycle-id]="row.cycle.id"
            class="bg-white border border-slate-200 rounded-md p-4 shadow-card flex flex-col justify-between space-y-4"
          >
            <div class="space-y-2">
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2 min-w-0">
                  <h3 class="text-sm font-semibold text-slate-900 truncate">{{ row.cycle.name }}</h3>
                  <button
                    type="button"
                    [disabled]="!projectService.canEdit()"
                    (click)="toggleCycleActive(row.cycle)"
                    [attr.aria-label]="'Toggle active status for ' + row.cycle.name"
                    [title]="projectService.canEdit() ? 'Click to toggle active status' : ''"
                    class="px-2 py-0.5 rounded-full border text-[10px] font-medium shrink-0 transition-colors disabled:cursor-default"
                    [ngClass]="
                      row.cycle.is_active
                        ? 'bg-emerald-50 border-emerald-200 text-emerald-700 hover:bg-emerald-100/70'
                        : 'bg-slate-100 border-slate-200 text-slate-600 hover:bg-slate-200/70'
                    "
                  >
                    {{ row.cycle.is_active ? 'Active Cycle' : 'Completed / Planned' }}
                  </button>
                </div>

                @if (projectService.canEdit()) {
                  <div class="flex items-center gap-2 shrink-0">
                    @if (confirmDeleteId() === row.cycle.id) {
                      <span class="text-[11px] text-slate-500">Delete?</span>
                      <button
                        type="button"
                        (click)="confirmDeleteCycle(row.cycle.id)"
                        [attr.aria-label]="'Confirm delete cycle ' + row.cycle.name"
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
                    } @else {
                      <button
                        type="button"
                        (click)="openEditModal(row.cycle)"
                        [attr.aria-label]="'Edit cycle ' + row.cycle.name"
                        class="text-[11px] font-medium text-slate-500 hover:text-sky-600 transition-colors"
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        (click)="confirmDeleteId.set(row.cycle.id)"
                        [attr.aria-label]="'Delete cycle ' + row.cycle.name"
                        class="text-[11px] text-slate-400 hover:text-rose-600 transition-colors"
                      >
                        Delete
                      </button>
                    }
                  </div>
                }
              </div>

              <p class="text-xs text-slate-600">{{ row.cycle.description || 'No description provided.' }}</p>

              <div class="flex items-center gap-3 text-[11px] font-mono text-slate-500">
                <span>{{ row.cycle.start_date | date: 'MMM d, yyyy' }}</span>
                <span>→</span>
                <span>{{ row.cycle.end_date | date: 'MMM d, yyyy' }}</span>
              </div>
            </div>

            <!-- Progress & Actions -->
            <div class="space-y-2 pt-3 border-t border-slate-200">
              <div class="flex items-center justify-between text-[11px] text-slate-600">
                <span>Progress: {{ row.doneCount }}/{{ row.totalCount }} items completed</span>
                <span class="font-mono font-medium">{{ row.percent }}%</span>
              </div>
              <div class="w-full h-1.5 bg-slate-100 rounded-full overflow-hidden">
                <div
                  class="h-full bg-sky-600 transition-all"
                  [style.width.%]="row.percent"
                ></div>
              </div>
              <div class="flex justify-end pt-1">
                <button
                  type="button"
                  (click)="viewCycleItems(row.cycle.id)"
                  class="text-xs font-medium text-sky-600 hover:underline"
                >
                  View cycle items in Board →
                </button>
              </div>
            </div>
          </div>
        } @empty {
          <div class="col-span-full bg-white border border-slate-200 rounded-md p-8 text-center text-xs text-slate-400">
            No sprint cycles created yet for this project. Click <strong>+ New Cycle</strong> to schedule one.
          </div>
        }
      </div>

      <!-- Create Sprint Cycle Modal -->
      <app-modal
        [isOpen]="isCreateOpen()"
        title="Create Sprint Cycle"
        (closed)="isCreateOpen.set(false)"
      >
        <form (ngSubmit)="submitCreate()" class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Cycle Name</label>
            <input
              type="text"
              name="name"
              [(ngModel)]="newName"
              required
              aria-label="New cycle name"
              placeholder="e.g., Cycle 5 — Performance & Security"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Description</label>
            <textarea
              rows="2"
              name="description"
              [(ngModel)]="newDesc"
              aria-label="New cycle description"
              placeholder="Goals and deliverables for this cycle..."
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            ></textarea>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">Start Date</label>
              <input
                type="date"
                name="startDate"
                [(ngModel)]="newStart"
                aria-label="New cycle start date"
                class="w-full px-2.5 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">End Date</label>
              <input
                type="date"
                name="endDate"
                [(ngModel)]="newEnd"
                [min]="newStart"
                aria-label="New cycle end date"
                class="w-full px-2.5 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
              />
            </div>
          </div>
          @if (dateValidationError()) {
            <p class="text-[11px] text-rose-600">{{ dateValidationError() }}</p>
          }
          <label class="flex items-center gap-2 text-xs text-slate-700 cursor-pointer">
            <input type="checkbox" name="isActive" [(ngModel)]="newIsActive" />
            Mark as Active Sprint Cycle
          </label>
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
              aria-label="Submit create cycle"
              class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
            >
              Create Cycle
            </button>
          </div>
        </form>
      </app-modal>

      <!-- Edit Sprint Cycle Modal -->
      <app-modal
        [isOpen]="isEditOpen()"
        title="Edit Sprint Cycle"
        (closed)="closeEditModal()"
      >
        <form (ngSubmit)="submitEdit()" class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Cycle Name</label>
            <input
              type="text"
              name="editName"
              [(ngModel)]="editName"
              required
              aria-label="Edit cycle name"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Description</label>
            <textarea
              rows="2"
              name="editDescription"
              [(ngModel)]="editDesc"
              aria-label="Edit cycle description"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            ></textarea>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">Start Date</label>
              <input
                type="date"
                name="editStartDate"
                [(ngModel)]="editStart"
                aria-label="Edit cycle start date"
                class="w-full px-2.5 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
              />
            </div>
            <div>
              <label class="block text-xs font-medium text-slate-700 mb-1">End Date</label>
              <input
                type="date"
                name="editEndDate"
                [(ngModel)]="editEnd"
                [min]="editStart"
                aria-label="Edit cycle end date"
                class="w-full px-2.5 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
              />
            </div>
          </div>
          @if (dateValidationError()) {
            <p class="text-[11px] text-rose-600">{{ dateValidationError() }}</p>
          }
          <label class="flex items-center gap-2 text-xs text-slate-700 cursor-pointer">
            <input type="checkbox" name="editIsActive" [(ngModel)]="editIsActive" />
            Mark as Active Sprint Cycle
          </label>
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
              aria-label="Save cycle changes"
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
export class CyclesComponent {
  readonly projectService = inject(ProjectService);
  readonly workItemService = inject(WorkItemService);
  private readonly router = inject(Router);

  readonly isCreateOpen = signal(false);
  readonly isEditOpen = signal(false);
  readonly editingCycleId = signal<string | null>(null);
  readonly confirmDeleteId = signal<string | null>(null);
  readonly dateValidationError = signal<string>('');

  newName = '';
  newDesc = '';
  newStart = this.getOffsetDateString(0);
  newEnd = this.getOffsetDateString(14);
  newIsActive = true;

  editName = '';
  editDesc = '';
  editStart = '';
  editEnd = '';
  editIsActive = false;

  readonly cycleRows = computed(() => {
    const items = this.workItemService.items();
    return this.projectService.cycles().map((cycle) => {
      const linked = items.filter((i) => i.cycle_id === cycle.id);
      const done = linked.filter((i) => i.status === 'DONE').length;
      const percent = linked.length > 0 ? Math.round((done / linked.length) * 100) : 0;
      return {
        cycle,
        totalCount: linked.length,
        doneCount: done,
        percent,
      };
    });
  });

  private getOffsetDateString(daysOffset: number): string {
    const d = new Date();
    d.setDate(d.getDate() + daysOffset);
    return d.toISOString().slice(0, 10);
  }

  private formatDateInput(iso?: string): string {
    if (!iso) return this.getOffsetDateString(0);
    return iso.slice(0, 10);
  }

  openCreateModal(): void {
    this.newName = '';
    this.newDesc = '';
    this.newStart = this.getOffsetDateString(0);
    this.newEnd = this.getOffsetDateString(14);
    this.newIsActive = true;
    this.dateValidationError.set('');
    this.isCreateOpen.set(true);
  }

  openEditModal(cycle: Cycle): void {
    this.editingCycleId.set(cycle.id);
    this.editName = cycle.name;
    this.editDesc = cycle.description || '';
    this.editStart = this.formatDateInput(cycle.start_date);
    this.editEnd = this.formatDateInput(cycle.end_date);
    this.editIsActive = cycle.is_active;
    this.dateValidationError.set('');
    this.isEditOpen.set(true);
  }

  closeEditModal(): void {
    this.isEditOpen.set(false);
    this.editingCycleId.set(null);
    this.dateValidationError.set('');
  }

  toggleCycleActive(cycle: Cycle): void {
    if (!this.projectService.canEdit()) return;
    this.projectService.updateCycle(cycle.id, { is_active: !cycle.is_active });
  }

  confirmDeleteCycle(cycleId: string): void {
    this.projectService.deleteCycle(cycleId);
    this.workItemService.items.update((list) =>
      list.map((item) => (item.cycle_id === cycleId ? { ...item, cycle_id: '' } : item))
    );
    if (this.workItemService.filter().cycle_id === cycleId) {
      this.workItemService.updateFilter({ cycle_id: '' });
    }
    this.confirmDeleteId.set(null);
  }

  viewCycleItems(cycleId: string): void {
    this.workItemService.updateFilter({ cycle_id: cycleId });
    this.workItemService.activeView.set('BOARD');
    this.router.navigate(['/projects', this.projectService.activeProjectId(), 'work-items']);
  }

  submitCreate(): void {
    const trimmedName = this.newName.trim();
    if (!trimmedName) return;
    if (this.newStart && this.newEnd && this.newEnd < this.newStart) {
      this.dateValidationError.set('End date must be on or after the start date.');
      return;
    }
    this.dateValidationError.set('');
    this.projectService.createCycle(
      trimmedName,
      this.newDesc.trim(),
      this.newStart,
      this.newEnd,
      this.newIsActive
    );
    this.newName = '';
    this.newDesc = '';
    this.isCreateOpen.set(false);
  }

  submitEdit(): void {
    const cycleId = this.editingCycleId();
    const trimmedName = this.editName.trim();
    if (!cycleId || !trimmedName) return;
    if (this.editStart && this.editEnd && this.editEnd < this.editStart) {
      this.dateValidationError.set('End date must be on or after the start date.');
      return;
    }
    this.dateValidationError.set('');
    this.projectService.updateCycle(cycleId, {
      name: trimmedName,
      description: this.editDesc.trim(),
      start_date: this.editStart,
      end_date: this.editEnd,
      is_active: this.editIsActive,
    });
    this.closeEditModal();
  }
}
