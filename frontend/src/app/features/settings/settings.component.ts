import { Component, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { SettingsService } from '../../core/services/settings.service';
import { ProjectService } from '../../core/services/project.service';
import { PageService } from '../../core/services/page.service';
import { AuthService } from '../../core/services/auth.service';
import { DropdownComponent, DropdownOption } from '../../shared/components/dropdown.component';
import { StatusBadgeComponent } from '../../shared/components/status-badge.component';
import { ALL_STATUSES, Project, ProjectRole } from '../../core/models/project.model';

@Component({
  selector: 'app-settings',
  standalone: true,
  imports: [CommonModule, FormsModule, DropdownComponent, StatusBadgeComponent],
  template: `
    <div class="w-full p-6 space-y-6 bg-slate-50 min-h-full">
      <!-- Top Settings Header -->
      <div class="border-b border-slate-200 pb-4">
        <h1 class="text-lg font-semibold text-slate-900">Settings</h1>
      </div>

      @if (settingsService.activeScope() === 'GENERAL') {
        <!-- ==================== 1. GENERAL APP SETTINGS ==================== -->
        @let gen = settingsService.generalSettings();
        <div class="space-y-5">
          <!-- Card 1: Application & Interface Preferences -->
          <div class="bg-white border border-slate-200 rounded-lg p-5 shadow-card space-y-4">
            <div class="border-b border-slate-100 pb-3">
              <h2 class="text-sm font-semibold text-slate-900">Application & Interface Preferences</h2>
              <p class="text-xs text-slate-500">
                Global workspace-free shell defaults and display preferences across Work and Wiki sections.
              </p>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label class="block text-xs font-medium text-slate-700 mb-1">Instance Display Name</label>
                <input
                  type="text"
                  [ngModel]="gen.instanceName"
                  (ngModelChange)="settingsService.updateGeneral('instanceName', $event)"
                  aria-label="Instance display name"
                  class="w-full px-3 py-1.5 text-xs bg-white border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
                />
              </div>

              <div>
                <label class="block text-xs font-medium text-slate-700 mb-1">Default Landing Section</label>
                <app-dropdown
                  [options]="landingRouteOptions"
                  [value]="gen.defaultLandingRoute"
                  (valueChange)="settingsService.updateGeneral('defaultLandingRoute', $event)"
                  [fullWidth]="true"
                  ariaLabel="Default landing section"
                ></app-dropdown>
              </div>

              <div>
                <label class="block text-xs font-medium text-slate-700 mb-1">Default Work Items View</label>
                <app-dropdown
                  [options]="workItemViewOptions"
                  [value]="gen.defaultWorkItemView"
                  (valueChange)="settingsService.updateGeneral('defaultWorkItemView', $event)"
                  [fullWidth]="true"
                  ariaLabel="Default work items view"
                ></app-dropdown>
              </div>
            </div>

            <div class="divide-y divide-slate-100 pt-2">
              <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                <div>
                  <div class="text-xs font-medium text-slate-800">Compact Table & List Rows</div>
                  <div class="text-[11px] text-slate-500">
                    Use dense 36px rows in Work Items List, My Work, and Spreadsheet views.
                  </div>
                </div>
                <input
                  type="checkbox"
                  [attr.switch]="''"
                  role="switch"
                  class="jynko-switch"
                  [checked]="gen.compactTableRows"
                  (change)="settingsService.updateGeneral('compactTableRows', $any($event.target).checked)"
                />
              </label>

              <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                <div>
                  <div class="text-xs font-medium text-slate-800">Show Sub-Item Progress on Kanban Cards</div>
                  <div class="text-[11px] text-slate-500">
                    Display completed sub-item counters (e.g., 2/3) on Board and Timeline cards.
                  </div>
                </div>
                <input
                  type="checkbox"
                  [attr.switch]="''"
                  role="switch"
                  class="jynko-switch"
                  [checked]="gen.showSubItemProgressOnCards"
                  (change)="settingsService.updateGeneral('showSubItemProgressOnCards', $any($event.target).checked)"
                />
              </label>

              <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                <div>
                  <div class="text-xs font-medium text-slate-800">Enable Global Keyboard Shortcuts</div>
                  <div class="text-[11px] text-slate-500">
                    Allow single-key shortcuts for quick work item creation and navigation.
                  </div>
                </div>
                <input
                  type="checkbox"
                  [attr.switch]="''"
                  role="switch"
                  class="jynko-switch"
                  [checked]="gen.enableKeyboardShortcuts"
                  (change)="settingsService.updateGeneral('enableKeyboardShortcuts', $any($event.target).checked)"
                />
              </label>
            </div>
          </div>

          <!-- Card 2: Notifications & Digest (Mock) -->
          <div class="bg-white border border-slate-200 rounded-lg p-5 shadow-card space-y-3">
            <div class="border-b border-slate-100 pb-3">
              <h2 class="text-sm font-semibold text-slate-900">Notifications & Activity Digest</h2>
              <p class="text-xs text-slate-500">
                Choose which cross-project updates trigger alerts for {{ authService.currentUser().email }}.
              </p>
            </div>

            <div class="divide-y divide-slate-100">
              <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                <div>
                  <div class="text-xs font-medium text-slate-800">Daily "My Work" Morning Digest</div>
                  <div class="text-[11px] text-slate-500">
                    Send a summary of assigned items due this week across all projects at 08:30.
                  </div>
                </div>
                <input
                  type="checkbox"
                  [attr.switch]="''"
                  role="switch"
                  class="jynko-switch"
                  [checked]="gen.dailyMorningDigest"
                  (change)="settingsService.updateGeneral('dailyMorningDigest', $any($event.target).checked)"
                />
              </label>

              <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                <div>
                  <div class="text-xs font-medium text-slate-800">Work Item Assignment & Comment Mentions</div>
                  <div class="text-[11px] text-slate-500">
                    Notify immediately when an item is assigned to you or someone mentions you in a comment.
                  </div>
                </div>
                <input
                  type="checkbox"
                  [attr.switch]="''"
                  role="switch"
                  class="jynko-switch"
                  [checked]="gen.notifyOnAssignment"
                  (change)="settingsService.updateGeneral('notifyOnAssignment', $any($event.target).checked)"
                />
              </label>

              <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                <div>
                  <div class="text-xs font-medium text-slate-800">Wiki Page Lock / Unlock Events</div>
                  <div class="text-[11px] text-slate-500">
                    Notify when a locked specification or runbook page you authored is unlocked.
                  </div>
                </div>
                <input
                  type="checkbox"
                  [attr.switch]="''"
                  role="switch"
                  class="jynko-switch"
                  [checked]="gen.notifyOnWikiUnlock"
                  (change)="settingsService.updateGeneral('notifyOnWikiUnlock', $any($event.target).checked)"
                />
              </label>
            </div>
          </div>

          <!-- Card 4: Google Cloud IAP & Identity Context -->
          <div class="bg-white border border-slate-200 rounded-lg p-5 shadow-card space-y-3">
            <div class="flex items-center justify-between">
              <div>
                <h2 class="text-sm font-semibold text-slate-900">Google Cloud IAP & Session Identity</h2>
                <p class="text-xs text-slate-500">
                  Zero-trust identity extracted from Google Cloud Identity-Aware Proxy headers.
                </p>
              </div>
              <span class="px-2 py-0.5 rounded-full bg-emerald-50 border border-emerald-200 text-emerald-700 text-[11px] font-medium">
                IAP Authenticated
              </span>
            </div>

            <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-1 text-xs">
              <div class="p-3 rounded-md bg-slate-50 border border-slate-200/80">
                <div class="text-[10px] font-mono uppercase text-slate-400">User ID</div>
                <div class="font-mono font-medium text-slate-800 mt-0.5">{{ authService.currentUser().user_id }}</div>
              </div>
              <div class="p-3 rounded-md bg-slate-50 border border-slate-200/80">
                <div class="text-[10px] font-mono uppercase text-slate-400">Authenticated Email</div>
                <div class="font-mono font-medium text-slate-800 mt-0.5">{{ authService.currentUser().email }}</div>
              </div>
              <div class="p-3 rounded-md bg-slate-50 border border-slate-200/80">
                <div class="text-[10px] font-mono uppercase text-slate-400">Display Name</div>
                <div class="font-medium text-slate-800 mt-0.5">{{ authService.currentUser().name }}</div>
              </div>
            </div>
          </div>
        </div>
      } @else {
        <!-- ==================== 2. PER-PROJECT SETTINGS ==================== -->
        @if (projectService.activeProject(); as proj) {
          @let pset = settingsService.activeProjectSettings();
          <div class="space-y-5">
            <!-- Active Project Bar -->
            <div class="bg-white border border-slate-200 rounded-lg p-4 shadow-card flex flex-wrap items-center justify-between gap-3">
              <div class="flex items-center gap-2.5">
                <span class="text-xs font-semibold text-slate-600">Configuring Project:</span>
                <div class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-slate-200 bg-slate-50 text-xs font-medium text-slate-800">
                  <span>{{ proj.name }}</span>
                  <span class="font-mono text-[10px] px-1 rounded bg-slate-200/70 text-slate-600">
                    {{ proj.key_prefix }}
                  </span>
                </div>
              </div>

              <span class="px-2.5 py-0.5 rounded-full bg-sky-50 border border-sky-200 text-sky-700 text-[11px] font-medium">
                Your Role in {{ proj.name }}: {{ projectService.currentUserRole() }}
              </span>
            </div>

            <!-- Card 1: Project Identity & Firestore Metadata -->
            <div class="bg-white border border-slate-200 rounded-lg p-5 shadow-card space-y-4">
              <div class="flex flex-wrap items-center justify-between gap-2 border-b border-slate-100 pb-3">
                <div>
                  <h2 class="text-sm font-semibold text-slate-900">
                    Project Identity & Metadata — {{ proj.name }}
                  </h2>
                  <p class="text-xs text-slate-500">
                    Firestore root document <code class="font-mono">projects/{{ proj.id }}</code> · Next work item key: <code class="font-mono">{{ proj.key_prefix }}-{{ proj.item_seq + 1 }}</code>
                  </p>
                </div>
                <button
                  type="button"
                  (click)="openProjectWikiCollection(proj)"
                  aria-label="Open project Wiki collection from settings"
                  class="px-2.5 py-1 rounded-md border border-slate-200 bg-slate-50 hover:bg-slate-100 text-xs font-medium text-sky-700"
                >
                  📖 Open "{{ proj.name }}" Wiki Collection →
                </button>
              </div>

              <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div>
                  <label class="block text-xs font-medium text-slate-700 mb-1">Project Name</label>
                  <input
                    type="text"
                    [ngModel]="proj.name"
                    (ngModelChange)="updateProjectField(proj.id, 'name', $event)"
                    aria-label="Project setting name"
                    class="w-full px-3 py-1.5 text-xs bg-white border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
                  />
                </div>
                <div>
                  <label class="block text-xs font-medium text-slate-700 mb-1">Key Prefix</label>
                  <input
                    type="text"
                    [ngModel]="proj.key_prefix"
                    (ngModelChange)="updateProjectField(proj.id, 'key_prefix', $event)"
                    aria-label="Project setting key prefix"
                    class="w-full px-3 py-1.5 text-xs font-mono uppercase bg-white border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
                  />
                </div>
                <div>
                  <label class="block text-xs font-medium text-slate-700 mb-1">Description</label>
                  <input
                    type="text"
                    [ngModel]="proj.description"
                    (ngModelChange)="updateProjectField(proj.id, 'description', $event)"
                    aria-label="Project setting description"
                    class="w-full px-3 py-1.5 text-xs bg-white border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
                  />
                </div>
              </div>
            </div>

            <!-- Card 2: Work Item & Cycle Workflow Defaults (Per-Project Mock Settings) -->
            <div class="bg-white border border-slate-200 rounded-lg p-5 shadow-card space-y-4">
              <div class="border-b border-slate-100 pb-3">
                <h2 class="text-sm font-semibold text-slate-900">Work Item & Cycle Defaults ({{ proj.name }})</h2>
                <p class="text-xs text-slate-500">
                  Configure default status, priority, estimation scale, and sprint cycle automation for this project.
                </p>
              </div>

              <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                <div>
                  <label class="block text-xs font-medium text-slate-700 mb-1">Default New Item Status</label>
                  <app-dropdown
                    [options]="defaultStatusOptions"
                    [value]="pset.defaultNewItemStatus"
                    (valueChange)="settingsService.updateActiveProjectSetting('defaultNewItemStatus', $event)"
                    [fullWidth]="true"
                    ariaLabel="Default new item status"
                  ></app-dropdown>
                </div>

                <div>
                  <label class="block text-xs font-medium text-slate-700 mb-1">Default New Item Priority</label>
                  <app-dropdown
                    [options]="defaultPriorityOptions"
                    [value]="pset.defaultNewItemPriority"
                    (valueChange)="settingsService.updateActiveProjectSetting('defaultNewItemPriority', $event)"
                    [fullWidth]="true"
                    ariaLabel="Default new item priority"
                  ></app-dropdown>
                </div>

                <div>
                  <label class="block text-xs font-medium text-slate-700 mb-1">Story Point Scale</label>
                  <app-dropdown
                    [options]="estimationScaleOptions"
                    [value]="pset.estimationScale"
                    (valueChange)="settingsService.updateActiveProjectSetting('estimationScale', $event)"
                    [fullWidth]="true"
                    ariaLabel="Story point estimation scale"
                  ></app-dropdown>
                </div>

                <div>
                  <label class="block text-xs font-medium text-slate-700 mb-1">Cycle Length</label>
                  <app-dropdown
                    [options]="cycleDurationOptions"
                    [value]="pset.cycleDurationWeeks"
                    (valueChange)="settingsService.updateActiveProjectSetting('cycleDurationWeeks', $event)"
                    [fullWidth]="true"
                    ariaLabel="Default cycle duration"
                  ></app-dropdown>
                </div>
              </div>

              <div class="divide-y divide-slate-100 pt-1">
                <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                  <div>
                    <div class="text-xs font-medium text-slate-800">Auto-Assign New Work Items to Module Lead</div>
                    <div class="text-[11px] text-slate-500">
                      Automatically assign newly created items to the lead of the selected Module.
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    [attr.switch]="''"
                    role="switch"
                    aria-label="Toggle auto-assign new work items to module lead"
                    class="jynko-switch"
                    [checked]="pset.autoAssignToLead"
                    (change)="settingsService.updateActiveProjectSetting('autoAssignToLead', $any($event.target).checked)"
                  />
                </label>

                <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                  <div>
                    <div class="text-xs font-medium text-slate-800">Auto-Rollover Unfinished Cycle Work Items</div>
                    <div class="text-[11px] text-slate-500">
                      Move incomplete items from a closing Cycle into the next planned Cycle automatically.
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    [attr.switch]="''"
                    role="switch"
                    aria-label="Toggle auto-rollover unfinished cycle work items"
                    class="jynko-switch"
                    [checked]="pset.autoRolloverUnfinishedCycleItems"
                    (change)="settingsService.updateActiveProjectSetting('autoRolloverUnfinishedCycleItems', $any($event.target).checked)"
                  />
                </label>

                <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                  <div>
                    <div class="text-xs font-medium text-slate-800">Require Point Estimate Before "In Progress"</div>
                    <div class="text-[11px] text-slate-500">
                      Prompt team members to set an effort estimate before transitioning an item to IN_PROGRESS.
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    [attr.switch]="''"
                    role="switch"
                    aria-label="Toggle require point estimate before in progress"
                    class="jynko-switch"
                    [checked]="pset.requireEstimateBeforeInProgress"
                    (change)="settingsService.updateActiveProjectSetting('requireEstimateBeforeInProgress', $any($event.target).checked)"
                  />
                </label>
              </div>

              <!-- Kanban Board Visible Status Columns Configuration -->
              <div class="pt-3 border-t border-slate-100 space-y-2.5">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <div>
                    <div class="text-xs font-medium text-slate-800">
                      Kanban Board Visible Columns ({{ settingsService.visibleBoardStatuses().length }}/{{ allStatuses.length }})
                    </div>
                    <div class="text-[11px] text-slate-500">
                      Choose which workflow statuses are displayed as columns on the Kanban board (e.g., hide Cancelled).
                    </div>
                  </div>
                  <div class="flex items-center gap-2 text-[11px]">
                    <button
                      type="button"
                      (click)="settingsService.resetBoardColumnsToDefault()"
                      class="px-2 py-1 rounded border border-slate-200 bg-slate-50 hover:bg-slate-100 text-slate-700 font-medium"
                    >
                      Default (Hide Cancelled)
                    </button>
                    <button
                      type="button"
                      (click)="settingsService.showAllBoardColumns()"
                      class="px-2 py-1 rounded border border-slate-200 bg-slate-50 hover:bg-slate-100 text-sky-700 font-medium"
                    >
                      Show All
                    </button>
                  </div>
                </div>

                <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-6 gap-2">
                  @for (st of allStatuses; track st.value) {
                    @let isColVisible = settingsService.isBoardColumnVisible(st.value);
                    <label
                      class="flex items-center justify-between gap-2 px-3 py-2 rounded-md border cursor-pointer transition-colors"
                      [ngClass]="
                        isColVisible
                          ? 'bg-sky-50/40 border-sky-200'
                          : 'bg-slate-50 border-slate-200 opacity-70'
                      "
                    >
                      <app-status-badge [status]="st.value"></app-status-badge>
                      <input
                        type="checkbox"
                        [attr.switch]="''"
                        [checked]="isColVisible"
                        [disabled]="isColVisible && settingsService.visibleBoardStatuses().length <= 1"
                        (change)="settingsService.toggleBoardColumn(st.value)"
                        class="jynko-switch"
                      />
                    </label>
                  }
                </div>
              </div>
            </div>

            <!-- Card 3: Project Wiki & Documentation Settings -->
            <div class="bg-white border border-slate-200 rounded-lg p-5 shadow-card space-y-3">
              <div class="border-b border-slate-100 pb-3">
                <h2 class="text-sm font-semibold text-slate-900">Wiki Collection & Documentation Policy ({{ proj.name }})</h2>
                <p class="text-xs text-slate-500">
                  Controls for the automatic <code class="font-mono">{{ proj.name }}</code> collection in the Wiki section.
                </p>
              </div>

              <div class="divide-y divide-slate-100">
                <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                  <div>
                    <div class="text-xs font-medium text-slate-800">Auto-Lock Published Runbook Pages</div>
                    <div class="text-[11px] text-slate-500">
                      Lock pages tagged as runbooks after publishing to prevent accidental edits.
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    [attr.switch]="''"
                    role="switch"
                    aria-label="Toggle auto-lock published runbooks"
                    class="jynko-switch"
                    [checked]="pset.autoLockPublishedRunbooks"
                    (change)="settingsService.updateActiveProjectSetting('autoLockPublishedRunbooks', $any($event.target).checked)"
                  />
                </label>

                <label class="py-2.5 flex items-center justify-between gap-4 cursor-pointer">
                  <div>
                    <div class="text-xs font-medium text-slate-800">Record Immutable Version Snapshot on Every Save</div>
                    <div class="text-[11px] text-slate-500">
                      Store a full Markdown revision under <code class="font-mono">projects/{{ proj.id }}/pages/:pageId/versions</code>.
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    [attr.switch]="''"
                    role="switch"
                    aria-label="Toggle record immutable version snapshot on every save"
                    class="jynko-switch"
                    [checked]="pset.createVersionOnEverySave"
                    (change)="settingsService.updateActiveProjectSetting('createVersionOnEverySave', $any($event.target).checked)"
                  />
                </label>
              </div>
            </div>

            <!-- Card 4: Per-Project RBAC Members & Roles -->
            <div class="bg-white border border-slate-200 rounded-lg shadow-card overflow-hidden">
              <div class="px-5 py-3.5 bg-slate-50 border-b border-slate-200 flex items-center justify-between">
                <div>
                  <h2 class="text-xs font-semibold text-slate-900">
                    Per-Project RBAC Members ({{ proj.name }})
                  </h2>
                  <p class="text-[11px] text-slate-500">
                    Stored in <code class="font-mono">projects/{{ proj.id }}/members/{{ '{userId}' }}</code>
                  </p>
                </div>
              </div>

              <div class="divide-y divide-slate-200">
                @for (mem of projectService.members(); track mem.user_id) {
                  <div class="px-5 py-2.5 flex items-center justify-between text-xs">
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
                  class="p-4 bg-slate-50 border-t border-slate-200 flex flex-wrap items-center gap-2"
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
                    ariaLabel="Select member role"
                  ></app-dropdown>
                  <button
                    type="submit"
                    class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
                  >
                    Assign Role
                  </button>
                </form>
              }
            </div>
          </div>
        }
      }
    </div>
  `,
})
export class SettingsComponent {
  readonly settingsService = inject(SettingsService);
  readonly projectService = inject(ProjectService);
  readonly pageService = inject(PageService);
  readonly authService = inject(AuthService);
  private readonly router = inject(Router);

  readonly allStatuses = ALL_STATUSES;

  newMemberEmail = '';
  newMemberName = '';
  newMemberRole: ProjectRole = 'MEMBER';

  readonly landingRouteOptions: DropdownOption[] = [
    { value: '/home', label: 'Home — All Projects Overview' },
    { value: '/my-work', label: 'My Work — Assigned Items' },
    { value: '/work-items', label: 'Work Items — Active Project' },
    { value: '/wiki', label: 'Wiki — Knowledge Base' },
  ];

  readonly workItemViewOptions: DropdownOption[] = [
    { value: 'LIST', label: 'List View' },
    { value: 'BOARD', label: 'Kanban Board' },
    { value: 'CALENDAR', label: 'Calendar View' },
    { value: 'TIMELINE', label: 'Timeline (Gantt)' },
    { value: 'SPREADSHEET', label: 'Spreadsheet Table' },
  ];

  readonly defaultStatusOptions: DropdownOption[] = [
    { value: 'BACKLOG', label: 'Backlog' },
    { value: 'TODO', label: 'Todo' },
    { value: 'IN_PROGRESS', label: 'In Progress' },
  ];

  readonly defaultPriorityOptions: DropdownOption[] = [
    { value: 'NONE', label: 'None' },
    { value: 'LOW', label: 'Low' },
    { value: 'MEDIUM', label: 'Medium' },
    { value: 'HIGH', label: 'High' },
    { value: 'URGENT', label: 'Urgent' },
  ];

  readonly estimationScaleOptions: DropdownOption[] = [
    { value: 'FIBONACCI', label: 'Fibonacci (1, 2, 3, 5, 8, 13)' },
    { value: 'LINEAR', label: 'Linear (1 - 10 pts)' },
    { value: 'POWERS_OF_TWO', label: 'Powers of 2 (1, 2, 4, 8, 16)' },
  ];

  readonly cycleDurationOptions: DropdownOption[] = [
    { value: '1', label: '1 Week' },
    { value: '2', label: '2 Weeks' },
    { value: '3', label: '3 Weeks' },
    { value: '4', label: '4 Weeks' },
  ];

  readonly roleOptions: DropdownOption[] = [
    { value: 'ADMIN', label: 'ADMIN' },
    { value: 'MEMBER', label: 'MEMBER' },
    { value: 'VIEWER', label: 'VIEWER' },
  ];

  updateProjectField(projectId: string, field: 'name' | 'key_prefix' | 'description', value: string): void {
    this.projectService.updateProject(projectId, { [field]: value });
  }

  openProjectWikiCollection(proj: Project): void {
    this.pageService.navigateToProjectCollection(proj);
    this.router.navigate(['/wiki']);
  }

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
}
