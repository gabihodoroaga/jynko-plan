import { Component, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ProjectService } from '../../core/services/project.service';
import { PageService } from '../../core/services/page.service';
import { WorkItemService } from '../../core/services/work-item.service';
import { AuthService } from '../../core/services/auth.service';
import { Project, ProjectRole } from '../../core/models/project.model';
import { Page } from '../../core/models/page.model';
import { DropdownComponent, DropdownOption } from '../../shared/components/dropdown.component';

@Component({
  selector: 'app-projects',
  standalone: true,
  imports: [CommonModule, FormsModule, DropdownComponent],
  template: `
    <div class="w-full p-6 space-y-6 bg-slate-50 min-h-full">
      <!-- Active Project Card -->
      @if (projectService.activeProject(); as proj) {
        <div class="bg-white border border-slate-200 rounded-md p-4 shadow-card space-y-3">
          <div class="flex items-center justify-between gap-2">
            <div class="flex items-center gap-2.5">
              <span class="px-2 py-0.5 rounded bg-slate-100 border border-slate-200 font-mono text-xs font-semibold text-slate-700">
                {{ proj.key_prefix }}
              </span>
              <h3 class="text-sm font-semibold text-slate-900">{{ proj.name }}</h3>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <span class="px-2 py-0.5 rounded-full bg-sky-50 border border-sky-200 text-sky-700 text-[11px] font-medium">
                Your Role: {{ projectService.currentUserRole() }}
              </span>
              @if (projectService.isAdmin()) {
                <button
                  type="button"
                  (click)="projectService.openEditProjectModal(proj)"
                  aria-label="Edit project"
                  class="px-2.5 py-0.5 rounded border border-slate-200 bg-white hover:bg-slate-50 text-[11px] font-medium text-slate-600 transition-colors"
                >
                  Edit
                </button>
              }
            </div>
          </div>
          <p class="text-xs text-slate-600">{{ proj.description }}</p>
          <div class="flex flex-wrap items-center justify-between gap-2 pt-1">
            <div class="text-[11px] font-mono text-slate-400">
              Firestore Path: projects/{{ proj.id }} · Next Item Key: {{ proj.key_prefix }}-{{ proj.item_seq + 1 }}
            </div>
            <button
              type="button"
              (click)="openProjectCollectionInWiki(proj)"
              aria-label="Open project collection in Wiki"
              class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-slate-200 bg-slate-50 hover:bg-slate-100 text-xs font-medium text-sky-700 transition-colors"
            >
              <span>📖 Open "{{ proj.name }}" Collection in Wiki →</span>
            </button>
          </div>
        </div>

        <!-- Project Pages (Linked Wiki Collection) -->
        <div class="bg-white border border-slate-200 rounded-md shadow-card overflow-hidden">
          <div class="px-4 py-3 bg-slate-50 border-b border-slate-200 flex items-center justify-between">
            <div>
              <h3 class="text-xs font-semibold text-slate-900">
                Project Pages — Wiki Collection "{{ proj.name }}"
              </h3>
              <p class="text-[11px] text-slate-500">
                Every project automatically creates a corresponding collection in the Wiki section.
              </p>
            </div>
            <button
              type="button"
              (click)="openProjectCollectionInWiki(proj)"
              class="text-xs font-medium text-sky-600 hover:underline"
            >
              View in Wiki →
            </button>
          </div>
          <div class="divide-y divide-slate-200">
            @for (pg of activeProjectPages(); track pg.id) {
              <button
                type="button"
                (click)="openPageInProjectWiki(proj, pg)"
                class="w-full px-4 py-2.5 flex items-center justify-between text-left hover:bg-slate-50 transition-colors"
              >
                <div class="flex items-center gap-2 min-w-0">
                  <svg class="w-3.5 h-3.5 text-slate-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                    <path d="M6 3h9l5 5v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z" />
                    <path d="M8 11h8M8 15h5" />
                  </svg>
                  <span class="text-xs font-medium text-slate-900 truncate">{{ pg.title }}</span>
                </div>
                <span class="text-[11px] font-mono text-slate-400 shrink-0">
                  Collection: {{ proj.name }} →
                </span>
              </button>
            }
          </div>
        </div>
      }

      <!-- Per-Project RBAC Members Table -->
      <div class="bg-white border border-slate-200 rounded-md shadow-card overflow-hidden">
        <div class="px-4 py-3 bg-slate-50 border-b border-slate-200 flex items-center justify-between">
          <div>
            <h3 class="text-xs font-semibold text-slate-900">Project Members & RBAC Roles</h3>
            <p class="text-[11px] text-slate-500">
              Stored in <code class="font-mono">projects/{{ projectService.activeProjectId() }}/members/{{ '{userId}' }}</code>
            </p>
          </div>
        </div>

        <div class="divide-y divide-slate-200">
          @for (mem of projectService.members(); track mem.user_id) {
            <div class="px-4 py-2.5 flex items-center justify-between text-xs">
              <div>
                <div class="font-medium text-slate-900">{{ mem.name }}</div>
                <div class="text-[11px] font-mono text-slate-400">{{ mem.email }}</div>
              </div>
              <div class="flex items-center gap-2.5">
                <span
                  class="px-2 py-0.5 rounded-md border text-[11px] font-mono font-medium"
                  [ngClass]="
                    mem.role === 'ADMIN'
                      ? 'bg-sky-50 text-sky-700 border-sky-200'
                      : mem.role === 'MEMBER'
                        ? 'bg-emerald-50 text-emerald-700 border-emerald-200'
                        : 'bg-slate-100 text-slate-600 border-slate-200'
                  "
                >
                  {{ mem.role }}
                </span>

                @if (projectService.isAdmin() && mem.user_id !== authService.currentUser().user_id) {
                  <button
                    type="button"
                    (click)="projectService.removeMember(mem.user_id)"
                    class="text-[11px] text-slate-400 hover:text-rose-600"
                  >
                    Remove
                  </button>
                }
              </div>
            </div>
          }
        </div>

        @if (projectService.isAdmin()) {
          <form
            (ngSubmit)="addMember()"
            class="p-3.5 bg-slate-50 border-t border-slate-200 flex flex-wrap items-center gap-2"
          >
            <input
              type="email"
              name="memberEmail"
              [(ngModel)]="newMemberEmail"
              required
              placeholder="user@organization.com"
              class="flex-1 min-w-[180px] px-2.5 py-1.5 text-xs bg-white border border-slate-200 rounded-md"
            />
            <input
              type="text"
              name="memberName"
              [(ngModel)]="newMemberName"
              placeholder="Display Name"
              class="w-40 px-2.5 py-1.5 text-xs bg-white border border-slate-200 rounded-md"
            />
            <app-dropdown
              [options]="roleOptions"
              [value]="newMemberRole"
              (valueChange)="onRoleChange($event)"
              ariaLabel="Select Member Role"
            ></app-dropdown>
            <button
              type="submit"
              class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
            >
              Assign Role
            </button>
          </form>
        } @else {
          <div class="p-3 bg-slate-50 border-t border-slate-200 text-[11px] text-slate-500">
            Only project <strong>ADMIN</strong>s can modify member RBAC assignments.
          </div>
        }
      </div>

      <!-- Project Labels Section -->
      <div class="bg-white border border-slate-200 rounded-md p-4 shadow-card space-y-3">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="text-xs font-semibold text-slate-900">Project Labels</h3>
            <p class="text-[11px] text-slate-500">
              Click a label name or color swatch to edit, or click <code class="font-mono">×</code> to remove it.
            </p>
          </div>
        </div>
        <div class="flex flex-wrap gap-2">
          @for (lbl of projectService.labels(); track lbl.id) {
            @if (projectService.canEdit()) {
              <div
                [attr.data-label-id]="lbl.id"
                class="inline-flex items-center gap-1.5 pl-2 pr-1.5 py-1 rounded-full border border-slate-200 bg-slate-50 text-xs text-slate-700 focus-within:bg-white focus-within:border-sky-500 focus-within:ring-1 focus-within:ring-sky-500 transition-colors"
              >
                <label class="relative inline-flex items-center cursor-pointer" [title]="'Change color for ' + lbl.name">
                  <span class="w-2.5 h-2.5 rounded-full border border-black/10" [style.backgroundColor]="lbl.color"></span>
                  <input
                    type="color"
                    [ngModel]="lbl.color"
                    (ngModelChange)="onLabelColorChange(lbl.id, $event)"
                    [attr.aria-label]="'Edit label color ' + lbl.name"
                    class="sr-only"
                  />
                </label>
                <input
                  type="text"
                  [ngModel]="lbl.name"
                  [ngModelOptions]="{ updateOn: 'blur' }"
                  (ngModelChange)="onLabelNameChange(lbl.id, $event)"
                  (keydown.enter)="$any($event.target).blur()"
                  [attr.aria-label]="'Edit label name ' + lbl.name"
                  [style.width.ch]="Math.max(lbl.name.length + 1, 6)"
                  class="bg-transparent text-xs font-medium text-slate-700 focus:outline-none min-w-[4ch]"
                />
                <button
                  type="button"
                  (click)="removeLabel(lbl.id)"
                  [attr.aria-label]="'Remove label ' + lbl.name"
                  title="Remove label"
                  class="w-4 h-4 rounded-full inline-flex items-center justify-center text-slate-400 hover:text-rose-600 hover:bg-rose-50 transition-colors"
                >
                  ×
                </button>
              </div>
            } @else {
              <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full border border-slate-200 bg-slate-50 text-xs text-slate-700">
                <span class="w-2 h-2 rounded-full" [style.backgroundColor]="lbl.color"></span>
                {{ lbl.name }}
              </span>
            }
          }
        </div>
        @if (projectService.canEdit()) {
          <form (ngSubmit)="addLabel()" class="flex items-center gap-2 pt-2">
            <input
              type="text"
              name="labelName"
              [(ngModel)]="newLabelName"
              aria-label="New label name"
              placeholder="New label name..."
              class="px-2.5 py-1 text-xs border border-slate-200 rounded-md"
            />
            <input
              type="color"
              name="labelColor"
              [(ngModel)]="newLabelColor"
              aria-label="New label color"
              class="h-7 w-10 border border-slate-200 rounded cursor-pointer"
            />
            <button
              type="submit"
              aria-label="Add project label"
              class="px-2.5 py-1 rounded-md bg-slate-100 hover:bg-slate-200 border border-slate-200 text-xs font-medium text-slate-700"
            >
              + Add Label
            </button>
          </form>
        }
      </div>
    </div>
  `,
})
export class ProjectsComponent {
  readonly projectService = inject(ProjectService);
  readonly pageService = inject(PageService);
  readonly workItemService = inject(WorkItemService);
  readonly authService = inject(AuthService);
  private readonly router = inject(Router);
  protected readonly Math = Math;

  readonly activeProjectPages = computed<Page[]>(() => {
    const proj = this.projectService.activeProject();
    if (!proj) return [];
    return this.pageService.getPagesForProject(proj.id);
  });

  openProjectCollectionInWiki(proj: Project): void {
    this.pageService.navigateToProjectCollection(proj);
    this.router.navigate(['/wiki']);
  }

  openPageInProjectWiki(proj: Project, page: Page): void {
    this.pageService.navigateToProjectCollection(proj);
    this.pageService.selectPage(page.id);
    this.router.navigate(['/wiki']);
  }

  newMemberEmail = '';
  newMemberName = '';
  newMemberRole: ProjectRole = 'MEMBER';

  newLabelName = '';
  newLabelColor = '#0284c7';

  readonly roleOptions: DropdownOption[] = [
    { value: 'ADMIN', label: 'ADMIN' },
    { value: 'MEMBER', label: 'MEMBER' },
    { value: 'VIEWER', label: 'VIEWER' },
  ];

  onRoleChange(val: string): void {
    this.newMemberRole = val as ProjectRole;
  }

  addMember(): void {
    const email = this.newMemberEmail.trim();
    if (!email) return;
    this.projectService.upsertMember(email, this.newMemberName.trim(), this.newMemberRole);
    this.newMemberEmail = '';
    this.newMemberName = '';
  }

  addLabel(): void {
    const name = this.newLabelName.trim();
    if (!name) return;
    this.projectService.createLabel(name, this.newLabelColor);
    this.newLabelName = '';
  }

  onLabelNameChange(labelId: string, name: string): void {
    const trimmed = name.trim();
    if (!trimmed) return;
    this.projectService.updateLabel(labelId, { name: trimmed });
  }

  onLabelColorChange(labelId: string, color: string): void {
    if (!color) return;
    this.projectService.updateLabel(labelId, { color });
  }

  removeLabel(labelId: string): void {
    this.projectService.deleteLabel(labelId);
    this.workItemService.items.update((list) =>
      list.map((item) =>
        item.label_ids?.includes(labelId)
          ? { ...item, label_ids: item.label_ids.filter((id) => id !== labelId) }
          : item
      )
    );
  }
}
