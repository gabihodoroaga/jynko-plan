import { Component, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ProjectService } from '../../core/services/project.service';
import { WorkItemService } from '../../core/services/work-item.service';
import { Module } from '../../core/models/project.model';
import { ModalComponent } from '../../shared/components/modal.component';
import { DropdownComponent, DropdownOption } from '../../shared/components/dropdown.component';

@Component({
  selector: 'app-modules',
  standalone: true,
  imports: [CommonModule, FormsModule, ModalComponent, DropdownComponent],
  template: `
    <div class="w-full p-6 space-y-5 bg-slate-50 min-h-full">
      <div class="flex items-center justify-between">
        <div>
          <h2 class="text-base font-semibold text-slate-900">Feature Modules</h2>
          <p class="text-xs text-slate-500">
            Persistent functional domains across the software lifecycle in {{ projectService.activeProject()?.name }}
          </p>
        </div>
        @if (projectService.canEdit()) {
          <button
            type="button"
            aria-label="Create new module"
            (click)="openCreateModal()"
            class="bg-sky-600 hover:bg-sky-700 text-white rounded-md px-3 py-1.5 text-xs font-medium transition-colors"
          >
            + New Module
          </button>
        }
      </div>

      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        @for (row of moduleRows(); track row.module.id) {
          <div
            [attr.data-module-id]="row.module.id"
            class="bg-white border border-slate-200 rounded-md p-4 shadow-card flex flex-col justify-between space-y-4"
          >
            <div class="space-y-2.5">
              <div class="flex items-center justify-between gap-2">
                <h3 class="text-sm font-semibold text-slate-900 truncate">{{ row.module.name }}</h3>
                @if (projectService.canEdit()) {
                  <div class="flex items-center gap-2 shrink-0">
                    @if (confirmDeleteId() === row.module.id) {
                      <span class="text-[11px] text-slate-500">Delete?</span>
                      <button
                        type="button"
                        (click)="confirmDeleteModule(row.module.id)"
                        [attr.aria-label]="'Confirm delete module ' + row.module.name"
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
                        (click)="openEditModal(row.module)"
                        [attr.aria-label]="'Edit module ' + row.module.name"
                        class="text-[11px] font-medium text-slate-500 hover:text-sky-600 transition-colors"
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        (click)="confirmDeleteId.set(row.module.id)"
                        [attr.aria-label]="'Delete module ' + row.module.name"
                        class="text-[11px] text-slate-400 hover:text-rose-600 transition-colors"
                      >
                        Delete
                      </button>
                    }
                  </div>
                }
              </div>

              <p class="text-xs text-slate-600">{{ row.module.description || 'No description provided.' }}</p>

              <div class="flex items-center gap-2 text-[11px] text-slate-500 pt-0.5">
                <span>Lead:</span>
                <app-dropdown
                  [options]="leadDropdownOptions()"
                  [value]="row.module.lead_user_id || ''"
                  emptyLabel="Unassigned"
                  (valueChange)="onInlineLeadChange(row.module.id, $event)"
                  [disabled]="!projectService.canEdit()"
                  [ariaLabel]="'Module lead for ' + row.module.name"
                  size="sm"
                ></app-dropdown>
              </div>
            </div>

            <div class="pt-3 border-t border-slate-200 flex items-center justify-between">
              <span class="text-[11px] font-mono text-slate-500">
                {{ row.itemCount }} work items
              </span>
              <button
                type="button"
                (click)="filterByModule(row.module.id)"
                class="text-xs font-medium text-sky-600 hover:underline"
              >
                View items →
              </button>
            </div>
          </div>
        } @empty {
          <div class="col-span-full bg-white border border-slate-200 rounded-md p-8 text-center text-xs text-slate-400">
            No feature modules defined yet for this project. Click <strong>+ New Module</strong> to create one.
          </div>
        }
      </div>

      <!-- Create Feature Module Modal -->
      <app-modal
        [isOpen]="isCreateOpen()"
        title="Create Feature Module"
        (closed)="isCreateOpen.set(false)"
      >
        <form (ngSubmit)="submitCreate()" class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Module Name</label>
            <input
              type="text"
              name="name"
              [(ngModel)]="newName"
              required
              aria-label="New module name"
              placeholder="e.g., Firestore Sync & Webhooks"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Description</label>
            <textarea
              rows="2"
              name="description"
              [(ngModel)]="newDesc"
              aria-label="New module description"
              placeholder="Functional responsibilities of this module..."
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            ></textarea>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Module Lead</label>
            <app-dropdown
              [options]="memberOptions()"
              [value]="newLeadId"
              (valueChange)="newLeadId = $event"
              [fullWidth]="true"
            ></app-dropdown>
          </div>
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
              aria-label="Submit create module"
              class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
            >
              Create Module
            </button>
          </div>
        </form>
      </app-modal>

      <!-- Edit Feature Module Modal -->
      <app-modal
        [isOpen]="isEditOpen()"
        title="Edit Feature Module"
        (closed)="closeEditModal()"
      >
        <form (ngSubmit)="submitEdit()" class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Module Name</label>
            <input
              type="text"
              name="editName"
              [(ngModel)]="editName"
              required
              aria-label="Edit module name"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Description</label>
            <textarea
              rows="2"
              name="editDescription"
              [(ngModel)]="editDesc"
              aria-label="Edit module description"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            ></textarea>
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Module Lead</label>
            <app-dropdown
              [options]="memberOptions()"
              [value]="editLeadId"
              (valueChange)="editLeadId = $event"
              [fullWidth]="true"
            ></app-dropdown>
          </div>
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
              aria-label="Save module changes"
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
export class ModulesComponent {
  readonly projectService = inject(ProjectService);
  readonly workItemService = inject(WorkItemService);
  private readonly router = inject(Router);

  readonly isCreateOpen = signal(false);
  readonly isEditOpen = signal(false);
  readonly editingModuleId = signal<string | null>(null);
  readonly confirmDeleteId = signal<string | null>(null);

  newName = '';
  newDesc = '';
  newLeadId = 'dev-user-1';

  editName = '';
  editDesc = '';
  editLeadId = '';

  readonly memberOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'Unassigned' },
    ...this.projectService.members().map((m) => ({
      value: m.user_id,
      label: `${m.name} (${m.email})`,
    })),
  ]);

  readonly leadDropdownOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'Unassigned' },
    ...this.projectService.members().map((m) => ({
      value: m.user_id,
      label: m.name,
    })),
  ]);

  readonly moduleRows = computed(() => {
    const items = this.workItemService.items();
    const members = this.projectService.members();
    return this.projectService.modules().map((module) => {
      const lead = members.find((m) => m.user_id === module.lead_user_id);
      const count = items.filter((i) => i.module_id === module.id).length;
      return {
        module,
        leadName: lead ? lead.name : module.lead_user_id || 'Unassigned',
        itemCount: count,
      };
    });
  });

  openCreateModal(): void {
    this.newName = '';
    this.newDesc = '';
    const firstMember = this.projectService.members()[0]?.user_id || '';
    this.newLeadId = firstMember;
    this.isCreateOpen.set(true);
  }

  openEditModal(module: Module): void {
    this.editingModuleId.set(module.id);
    this.editName = module.name;
    this.editDesc = module.description || '';
    this.editLeadId = module.lead_user_id || '';
    this.isEditOpen.set(true);
  }

  closeEditModal(): void {
    this.isEditOpen.set(false);
    this.editingModuleId.set(null);
  }

  onInlineLeadChange(moduleId: string, leadUserId: string): void {
    this.projectService.updateModule(moduleId, { lead_user_id: leadUserId });
  }

  confirmDeleteModule(moduleId: string): void {
    this.projectService.deleteModule(moduleId);
    this.workItemService.items.update((list) =>
      list.map((item) => (item.module_id === moduleId ? { ...item, module_id: '' } : item))
    );
    if (this.workItemService.filter().module_id === moduleId) {
      this.workItemService.updateFilter({ module_id: '' });
    }
    this.confirmDeleteId.set(null);
  }

  filterByModule(moduleId: string): void {
    this.workItemService.updateFilter({ module_id: moduleId });
    this.router.navigate(['/projects', this.projectService.activeProjectId(), 'work-items']);
  }

  submitCreate(): void {
    const trimmedName = this.newName.trim();
    if (!trimmedName) return;
    this.projectService.createModule(trimmedName, this.newDesc.trim(), this.newLeadId);
    this.newName = '';
    this.newDesc = '';
    this.isCreateOpen.set(false);
  }

  submitEdit(): void {
    const moduleId = this.editingModuleId();
    const trimmedName = this.editName.trim();
    if (!moduleId || !trimmedName) return;
    this.projectService.updateModule(moduleId, {
      name: trimmedName,
      description: this.editDesc.trim(),
      lead_user_id: this.editLeadId,
    });
    this.closeEditModal();
  }
}
