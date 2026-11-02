import { Component, OnInit, computed, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { WorkItemService } from '../../../core/services/work-item.service';
import { ProjectService } from '../../../core/services/project.service';
import {
  ALL_PRIORITIES,
  ALL_STATUSES,
  SubWorkItem,
  WorkItem,
  WorkItemPriority,
  WorkItemStatus,
} from '../../../core/models/project.model';
import { StatusBadgeComponent } from '../../../shared/components/status-badge.component';
import { PriorityBadgeComponent } from '../../../shared/components/priority-badge.component';
import { DropdownComponent, DropdownOption } from '../../../shared/components/dropdown.component';
import { MarkdownPipe } from '../../../shared/pipes/markdown.pipe';

@Component({
  selector: 'app-work-item-detail-page',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    StatusBadgeComponent,
    PriorityBadgeComponent,
    DropdownComponent,
    MarkdownPipe,
  ],
  template: `
    <div class="flex flex-col h-full bg-white">
      @if (currentItem(); as item) {
        <!-- Top Page Header / Breadcrumb Bar -->
        <div class="flex items-center justify-between px-6 py-3 border-b border-slate-200 bg-slate-50 shrink-0">
          <div class="flex items-center gap-2.5">
            <a
              [routerLink]="['/projects', item.project_id, 'work-items']"
              aria-label="Back to Work Items"
              class="inline-flex items-center gap-1 px-2 py-1 rounded-md border border-slate-200 bg-white hover:bg-slate-100 text-xs font-medium text-slate-600 transition-colors"
            >
              <span>←</span>
              <span>Work Items</span>
            </a>
            <span class="text-slate-300">/</span>
            <span class="font-mono text-xs font-semibold text-slate-600 px-2 py-0.5 rounded bg-white border border-slate-200">
              {{ item.key }}
            </span>
            <app-status-badge [status]="item.status"></app-status-badge>
            <app-priority-badge [priority]="item.priority"></app-priority-badge>
          </div>

          <div class="flex items-center gap-2">
            @if (projectService.canEdit()) {
              <button
                type="button"
                (click)="deleteAndReturn(item)"
                class="px-2.5 py-1 rounded-md border border-rose-200 bg-rose-50 hover:bg-rose-100 text-[11px] font-medium text-rose-700"
              >
                Delete
              </button>
            }
          </div>
        </div>

        <!-- Full-Width Single-Column Vertical Stack Body -->
        <div class="flex-1 overflow-y-auto w-full p-6 space-y-6">
          <!-- 1. Title & Description -->
          <div class="space-y-3">
            <input
              type="text"
              [ngModel]="item.title"
              [ngModelOptions]="{ updateOn: 'blur' }"
              (ngModelChange)="workItemService.updateItem(item.id, { title: $event })"
              (keydown.enter)="$any($event.target).blur()"
              [disabled]="!projectService.canEdit()"
              aria-label="Work item title"
              class="w-full text-xl font-semibold text-slate-900 bg-transparent border-b border-transparent focus:border-slate-300 focus:outline-none pb-1"
            />

            <div class="flex items-center justify-between">
              <span class="text-[11px] font-semibold uppercase tracking-wider text-slate-400">
                Description (Markdown)
              </span>
              <button
                type="button"
                (click)="previewDesc.set(!previewDesc())"
                class="text-[11px] text-sky-600 hover:underline"
              >
                {{ previewDesc() ? 'Edit Markdown' : 'Preview Rendered' }}
              </button>
            </div>

            @if (previewDesc()) {
              <div
                class="p-4 rounded-md bg-slate-50 border border-slate-200 min-h-[180px] max-h-[200vh] overflow-y-auto"
                [innerHTML]="item.description | markdown"
              ></div>
            } @else {
              <div
                class="growable-textarea text-xs font-mono leading-relaxed"
                [attr.data-replicated-value]="item.description || ''"
              >
                <textarea
                  [ngModel]="item.description"
                  [ngModelOptions]="{ updateOn: 'blur' }"
                  (ngModelChange)="workItemService.updateItem(item.id, { description: $event })"
                  (input)="syncGrowableTextarea($event)"
                  [disabled]="!projectService.canEdit()"
                  aria-label="Work item description"
                  placeholder="Add a detailed Markdown description..."
                  class="auto-grow-description w-full min-h-[180px] max-h-[200vh] p-3 text-xs font-mono leading-relaxed text-slate-800 bg-slate-50 border border-slate-200 rounded-md focus:bg-white focus:outline-none focus:border-slate-300 resize-y"
                ></textarea>
              </div>
            }
          </div>

          <!-- 2. Metadata Attributes Grid (Styled Dropdowns & Dates) -->
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-3.5 p-4 rounded-md bg-slate-50 border border-slate-200">
            <div>
              <label class="block text-[10px] font-semibold uppercase text-slate-400 mb-1">Status</label>
              <app-dropdown
                [options]="statusOptions"
                [value]="item.status"
                (valueChange)="onStatusChange(item.id, $event)"
                [disabled]="!projectService.canEdit()"
                [fullWidth]="true"
                size="sm"
              ></app-dropdown>
            </div>

            <div>
              <label class="block text-[10px] font-semibold uppercase text-slate-400 mb-1">Priority</label>
              <app-dropdown
                [options]="priorityOptions"
                [value]="item.priority"
                (valueChange)="onPriorityChange(item.id, $event)"
                [disabled]="!projectService.canEdit()"
                [fullWidth]="true"
                size="sm"
              ></app-dropdown>
            </div>

            <div>
              <label class="block text-[10px] font-semibold uppercase text-slate-400 mb-1">Assignee</label>
              <app-dropdown
                [options]="assigneeOptions()"
                [value]="item.assignee_id || ''"
                (valueChange)="workItemService.updateItem(item.id, { assignee_id: $event })"
                [disabled]="!projectService.canEdit()"
                [fullWidth]="true"
                size="sm"
              ></app-dropdown>
            </div>

            <div>
              <label class="block text-[10px] font-semibold uppercase text-slate-400 mb-1">Module</label>
              <app-dropdown
                [options]="moduleOptions()"
                [value]="item.module_id || ''"
                (valueChange)="workItemService.updateItem(item.id, { module_id: $event })"
                [disabled]="!projectService.canEdit()"
                [fullWidth]="true"
                size="sm"
              ></app-dropdown>
            </div>

            <div>
              <label class="block text-[10px] font-semibold uppercase text-slate-400 mb-1">Epic</label>
              <app-dropdown
                [options]="epicOptions()"
                [value]="item.epic_id || ''"
                (valueChange)="workItemService.updateItem(item.id, { epic_id: $event })"
                [disabled]="!projectService.canEdit()"
                [fullWidth]="true"
                size="sm"
              ></app-dropdown>
            </div>

            <div>
              <label class="block text-[10px] font-semibold uppercase text-slate-400 mb-1">Cycle</label>
              <app-dropdown
                [options]="cycleOptions()"
                [value]="item.cycle_id || ''"
                (valueChange)="workItemService.updateItem(item.id, { cycle_id: $event })"
                [disabled]="!projectService.canEdit()"
                [fullWidth]="true"
                size="sm"
              ></app-dropdown>
            </div>

            <div>
              <div class="flex items-center justify-between mb-1">
                <label class="block text-[10px] font-semibold uppercase text-slate-400">Start Date</label>
                @if (item.start_date && projectService.canEdit()) {
                  <button
                    type="button"
                    (click)="onDateChange(item.id, 'start_date', '')"
                    class="text-[10px] text-slate-400 hover:text-slate-600"
                  >
                    Clear
                  </button>
                }
              </div>
              <input
                type="date"
                [ngModel]="formatDateInput(item.start_date)"
                (ngModelChange)="onDateChange(item.id, 'start_date', $event)"
                [disabled]="!projectService.canEdit()"
                aria-label="Work item start date"
                class="w-full h-7 px-2 rounded-md border border-slate-200 bg-white text-xs text-slate-800 focus:outline-none focus:border-slate-300 disabled:opacity-60"
              />
            </div>

            <div>
              <div class="flex items-center justify-between mb-1">
                <label class="block text-[10px] font-semibold uppercase text-slate-400">Due Date</label>
                @if (item.due_date && projectService.canEdit()) {
                  <button
                    type="button"
                    (click)="onDateChange(item.id, 'due_date', '')"
                    class="text-[10px] text-slate-400 hover:text-slate-600"
                  >
                    Clear
                  </button>
                }
              </div>
              <input
                type="date"
                [ngModel]="formatDateInput(item.due_date)"
                (ngModelChange)="onDateChange(item.id, 'due_date', $event)"
                [disabled]="!projectService.canEdit()"
                aria-label="Work item due date"
                class="w-full h-7 px-2 rounded-md border border-slate-200 bg-white text-xs text-slate-800 focus:outline-none focus:border-slate-300 disabled:opacity-60"
              />
            </div>

            <div>
              <label class="block text-[10px] font-semibold uppercase text-slate-400 mb-1">Estimate (pts)</label>
              <input
                type="number"
                min="0"
                [ngModel]="item.estimate ?? null"
                [ngModelOptions]="{ updateOn: 'blur' }"
                (ngModelChange)="workItemService.updateItem(item.id, { estimate: $event !== null ? +$event : undefined })"
                (keydown.enter)="$any($event.target).blur()"
                [disabled]="!projectService.canEdit()"
                aria-label="Work item estimate"
                placeholder="—"
                class="w-full h-7 px-2 rounded-md border border-slate-200 bg-white text-xs text-slate-800 focus:outline-none focus:border-slate-300 disabled:opacity-60"
              />
            </div>
          </div>

          <!-- 3. Sub-Work Items Section -->
          <div class="space-y-2.5">
            <div class="flex items-center justify-between">
              <h4 class="text-xs font-semibold text-slate-900">
                Sub-Work Items ({{ item.completed_sub_count }}/{{ item.sub_item_count }})
              </h4>
            </div>

            <div class="divide-y divide-slate-200 border border-slate-200 rounded-md bg-white">
              @for (sub of currentSubItems(); track sub.id) {
                <div class="flex items-center justify-between px-3 py-2 text-xs">
                  <label class="flex items-center gap-2.5 cursor-pointer flex-1">
                    <input
                      type="checkbox"
                      [checked]="sub.is_completed"
                      (change)="toggleSub(item.id, sub)"
                      [disabled]="!projectService.canEdit()"
                      class="rounded border-slate-300 text-sky-600 focus:ring-sky-500"
                    />
                    <span [ngClass]="sub.is_completed ? 'line-through text-slate-400' : 'text-slate-800'">
                      {{ sub.title }}
                    </span>
                  </label>
                  @if (projectService.canEdit()) {
                    <button
                      type="button"
                      (click)="workItemService.deleteSubItem(item.id, sub.id)"
                      class="text-slate-400 hover:text-rose-600 text-[11px]"
                    >
                      Remove
                    </button>
                  }
                </div>
              } @empty {
                <div class="px-3 py-2.5 text-xs text-slate-400">No sub-items yet.</div>
              }
            </div>

            @if (projectService.canEdit()) {
              <div class="flex gap-2">
                <input
                  type="text"
                  [(ngModel)]="newSubTitle"
                  (keyup.enter)="addSubItem(item.id)"
                  placeholder="Add granular sub-task..."
                  class="flex-1 text-xs px-2.5 py-1.5 rounded-md border border-slate-200 focus:outline-none focus:border-slate-300"
                />
                <button
                  type="button"
                  (click)="addSubItem(item.id)"
                  class="px-3 py-1.5 rounded-md bg-slate-100 hover:bg-slate-200 border border-slate-200 text-xs font-medium text-slate-700"
                >
                  + Add Sub-item
                </button>
              </div>
            }
          </div>

          <!-- 4. Threaded Discussion Comments -->
          <div class="space-y-3">
            <h4 class="text-xs font-semibold text-slate-900">
              Discussion Comments ({{ currentComments().length }})
            </h4>

            <div class="space-y-2.5">
              @for (cmt of currentComments(); track cmt.id) {
                <div class="p-3 rounded-md bg-slate-50 border border-slate-200">
                  <div class="flex items-center justify-between mb-1">
                    <span class="text-xs font-semibold text-slate-800">{{ cmt.author_name }}</span>
                    <span class="text-[10px] font-mono text-slate-400">
                      {{ cmt.created_at | date: 'MMM d, HH:mm' }}
                    </span>
                  </div>
                  <div class="text-xs text-slate-700" [innerHTML]="cmt.body | markdown"></div>
                </div>
              } @empty {
                <div class="text-xs text-slate-400">No comments yet. Start the technical discussion below.</div>
              }
            </div>

            @if (projectService.canEdit()) {
              <div class="space-y-2">
                <textarea
                  rows="3"
                  [(ngModel)]="newCommentBody"
                  placeholder="Write a comment (Markdown supported)..."
                  class="w-full p-2.5 text-xs bg-white border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
                ></textarea>
                <div class="flex justify-end">
                  <button
                    type="button"
                    (click)="postComment(item.id)"
                    class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
                  >
                    Post Comment
                  </button>
                </div>
              </div>
            }
          </div>
        </div>
      } @else {
        <div class="w-full p-6 text-xs text-slate-400">
          Loading work item details...
        </div>
      }
    </div>
  `,
})
export class WorkItemDetailPageComponent implements OnInit {
  readonly workItemService = inject(WorkItemService);
  readonly projectService = inject(ProjectService);
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);

  readonly routeItemId = signal<string>('');
  readonly previewDesc = signal<boolean>(false);

  readonly statusOptions: DropdownOption[] = ALL_STATUSES.map((s) => ({
    value: s.value,
    label: s.label,
  }));
  readonly priorityOptions: DropdownOption[] = ALL_PRIORITIES.map((p) => ({
    value: p.value,
    label: p.label,
  }));
  readonly assigneeOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'Unassigned' },
    ...this.projectService.members().map((m) => ({ value: m.user_id, label: m.name })),
  ]);
  readonly moduleOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'None' },
    ...this.projectService.modules().map((m) => ({ value: m.id, label: m.name })),
  ]);
  readonly epicOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'None' },
    ...this.projectService.epics().map((e) => ({ value: e.id, label: e.name })),
  ]);
  readonly cycleOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'None' },
    ...this.projectService.cycles().map((c) => ({ value: c.id, label: c.name })),
  ]);

  newSubTitle = '';
  newCommentBody = '';

  readonly currentItem = computed<WorkItem | null>(() => {
    const id = this.routeItemId();
    if (!id) return null;
    return this.workItemService.items().find((i) => i.id === id || i.key === id) ?? null;
  });

  readonly currentSubItems = computed<SubWorkItem[]>(() => {
    const item = this.currentItem();
    if (!item) return [];
    return this.workItemService.subItemsByItem()[item.id] ?? [];
  });

  readonly currentComments = computed(() => {
    const item = this.currentItem();
    if (!item) return [];
    return this.workItemService.commentsByItem()[item.id] ?? [];
  });

  private loadedRelationsKey = '';

  constructor() {
    effect(() => {
      const item = this.currentItem();
      if (item) {
        const key = `${item.project_id}:${item.id}`;
        if (this.loadedRelationsKey !== key) {
          this.loadedRelationsKey = key;
          this.workItemService.loadItemRelations(item.project_id, item.id);
        }
      }
    });
  }

  ngOnInit(): void {
    this.workItemService.closeItemDetail();
    this.route.paramMap.subscribe((params) => {
      const projectId = params.get('projectId') || '';
      const itemId = params.get('itemId') || '';
      this.routeItemId.set(itemId);
      if (projectId && projectId !== this.projectService.activeProjectId()) {
        this.projectService.selectProject(projectId);
        this.workItemService.loadItems(projectId);
      }
    });
  }

  onStatusChange(itemId: string, status: string): void {
    this.workItemService.updateItem(itemId, { status: status as WorkItemStatus });
  }

  onPriorityChange(itemId: string, priority: string): void {
    this.workItemService.updateItem(itemId, { priority: priority as WorkItemPriority });
  }

  formatDateInput(iso?: string): string {
    if (!iso) return '';
    return iso.slice(0, 10);
  }

  onDateChange(itemId: string, field: 'start_date' | 'due_date', value: string): void {
    this.workItemService.updateItem(itemId, { [field]: value || '' });
  }

  deleteAndReturn(item: WorkItem): void {
    const projId = item.project_id;
    this.workItemService.deleteItem(item.id);
    this.router.navigate(['/projects', projId, 'work-items']);
  }

  syncGrowableTextarea(event: Event): void {
    if (typeof CSS !== 'undefined' && !CSS.supports('field-sizing', 'content')) {
      const el = event.target as HTMLTextAreaElement;
      if (el?.parentElement) {
        el.parentElement.dataset['replicatedValue'] = el.value;
      }
    }
  }

  addSubItem(itemId: string): void {
    const t = this.newSubTitle.trim();
    if (!t) return;
    this.workItemService.addSubItem(itemId, t);
    this.newSubTitle = '';
  }

  toggleSub(itemId: string, sub: SubWorkItem): void {
    this.workItemService.toggleSubItem(itemId, sub);
  }

  postComment(itemId: string): void {
    const b = this.newCommentBody.trim();
    if (!b) return;
    this.workItemService.addComment(itemId, b);
    this.newCommentBody = '';
  }
}
