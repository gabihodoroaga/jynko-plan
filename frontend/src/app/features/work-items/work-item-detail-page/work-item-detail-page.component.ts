import { Component, HostListener, OnInit, computed, effect, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { WorkItemService } from '../../../core/services/work-item.service';
import { ProjectService } from '../../../core/services/project.service';
import {
  ALL_PRIORITIES,
  ALL_STATUSES,
  Label,
  SubWorkItem,
  WorkItem,
  WorkItemPriority,
  WorkItemStatus,
} from '../../../core/models/project.model';
import { DropdownComponent, DropdownOption } from '../../../shared/components/dropdown.component';
import { MarkdownPipe } from '../../../shared/pipes/markdown.pipe';

const STATUS_DOT_COLORS: Record<WorkItemStatus, string> = {
  BACKLOG: '#94a3b8',
  TODO: '#64748b',
  IN_PROGRESS: '#0284c7',
  IN_REVIEW: '#d97706',
  DONE: '#059669',
  CANCELLED: '#e11d48',
};

const PRIORITY_DOT_COLORS: Record<WorkItemPriority, string> = {
  URGENT: '#e11d48',
  HIGH: '#f97316',
  MEDIUM: '#0284c7',
  LOW: '#64748b',
  NONE: '#cbd5e1',
};

@Component({
  selector: 'app-work-item-detail-page',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterLink,
    DropdownComponent,
    MarkdownPipe,
  ],
  template: `
    <div class="flex flex-col h-full bg-white" (click)="openPopover.set(null)">
      @if (currentItem(); as item) {
        <!-- Top Page Header / Breadcrumb Bar: Work Item Key + Status, Priority, Assignee, Cycle on the left; Delete on the right -->
        <div class="flex items-start justify-between gap-2 px-6 py-3 border-b border-slate-200 bg-slate-50 shrink-0">
          <div class="flex-1 min-w-0 flex flex-wrap items-center gap-1.5">
            <a
              [routerLink]="['/projects', item.project_id, 'work-items']"
              aria-label="Back to Work Items"
              class="inline-flex items-center gap-1 px-2 py-1 rounded-md border border-slate-200 bg-white hover:bg-slate-100 text-xs font-medium text-slate-600 transition-colors"
            >
              <span>←</span>
              <span>Work Items</span>
            </a>
            <span class="text-slate-300 mx-0.5">/</span>
            <span class="font-mono text-xs font-semibold text-slate-600 px-2 py-0.5 rounded bg-white border border-slate-200 mr-1">
              {{ item.key }}
            </span>

            <!-- Status -->
            <app-dropdown
              [options]="statusOptions"
              [value]="item.status"
              (valueChange)="onStatusChange(item.id, $event)"
              [disabled]="!projectService.canEdit()"
              ariaLabel="Work item status"
              size="sm"
            ></app-dropdown>

            <!-- Priority -->
            <app-dropdown
              [options]="priorityOptions"
              [value]="item.priority"
              emptyLabel="Priority"
              [emptyValues]="['', 'NONE']"
              (valueChange)="onPriorityChange(item.id, $event)"
              [disabled]="!projectService.canEdit()"
              ariaLabel="Work item priority"
              size="sm"
            ></app-dropdown>

            <!-- Assignee -->
            <app-dropdown
              [options]="assigneeOptions()"
              [value]="item.assignee_id || ''"
              emptyLabel="Assignee"
              (valueChange)="workItemService.updateItem(item.id, { assignee_id: $event })"
              [disabled]="!projectService.canEdit()"
              ariaLabel="Work item assignee"
              size="sm"
              maxWidth="max-w-[140px]"
            ></app-dropdown>

            <!-- Cycle -->
            <app-dropdown
              [options]="cycleOptions()"
              [value]="item.cycle_id || ''"
              emptyLabel="Cycle"
              (valueChange)="workItemService.updateItem(item.id, { cycle_id: $event })"
              [disabled]="!projectService.canEdit()"
              ariaLabel="Work item cycle"
              size="sm"
              maxWidth="max-w-[180px]"
            ></app-dropdown>
          </div>

          <div class="flex items-center gap-2 shrink-0 ml-auto">
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
        <div class="flex-1 overflow-y-auto w-full p-6 space-y-5">
          <!-- 1. Title -->
          <div>
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
          </div>

          <!-- 2. Below-Title Compact Properties Strip (Module, Epic, Start Date, Due Date, Estimate) + Separate Labels Line -->
          <div class="space-y-2.5 pb-3 border-b border-slate-200/80" data-testid="work-item-top-properties">
            <!-- Line 1: Module, Epic, Start Date, Due Date, Estimate -->
            <div class="flex flex-wrap items-center gap-1.5">
              <!-- Module -->
              <app-dropdown
                [options]="moduleOptions()"
                [value]="item.module_id || ''"
                emptyLabel="Module"
                (valueChange)="workItemService.updateItem(item.id, { module_id: $event })"
                [disabled]="!projectService.canEdit()"
                ariaLabel="Work item module"
                size="sm"
              ></app-dropdown>

              <!-- Epic -->
              <app-dropdown
                [options]="epicOptions()"
                [value]="item.epic_id || ''"
                emptyLabel="Epic"
                (valueChange)="workItemService.updateItem(item.id, { epic_id: $event })"
                [disabled]="!projectService.canEdit()"
                ariaLabel="Work item epic"
                size="sm"
              ></app-dropdown>

              <!-- Start Date Pill + Dropdown -->
              <div class="relative inline-block">
                <button
                  type="button"
                  [disabled]="!projectService.canEdit()"
                  (click)="togglePopover('start_date', $event)"
                  aria-label="Work item start date"
                  [attr.aria-expanded]="openPopover() === 'start_date'"
                  class="flex items-center justify-between gap-1.5 px-2 py-1 text-[11px] bg-white border border-slate-200 hover:bg-slate-50 hover:border-slate-300 rounded-md transition-colors focus:outline-none disabled:opacity-50"
                >
                  <span [ngClass]="item.start_date ? 'font-medium text-slate-700' : 'text-slate-400'">
                    {{ item.start_date ? (item.start_date | date: 'MMM d, y') : 'Start date' }}
                  </span>
                  <svg class="w-3.5 h-3.5 text-slate-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="m6 9 6 6 6-6" />
                  </svg>
                </button>

                @if (openPopover() === 'start_date') {
                  <div
                    role="listbox"
                    aria-label="Start date options"
                    (click)="$event.stopPropagation()"
                    class="absolute left-0 z-50 mt-1 w-52 rounded-lg bg-white border border-slate-200 shadow-[0_8px_24px_rgba(15,23,42,0.10)] py-1.5 text-xs space-y-1"
                  >
                    <button
                      type="button"
                      (click)="applyPresetDate(item.id, 'start_date', 0)"
                      class="w-full px-2.5 py-1.5 text-left text-slate-700 hover:bg-slate-100/80 flex items-center justify-between"
                    >
                      <span>Today</span>
                      <span class="font-mono text-[10px] text-slate-400">{{ getPresetDateLabel(0) }}</span>
                    </button>
                    <button
                      type="button"
                      (click)="applyPresetDate(item.id, 'start_date', 1)"
                      class="w-full px-2.5 py-1.5 text-left text-slate-700 hover:bg-slate-100/80 flex items-center justify-between"
                    >
                      <span>Tomorrow</span>
                      <span class="font-mono text-[10px] text-slate-400">{{ getPresetDateLabel(1) }}</span>
                    </button>
                    <button
                      type="button"
                      (click)="applyPresetDate(item.id, 'start_date', 7)"
                      class="w-full px-2.5 py-1.5 text-left text-slate-700 hover:bg-slate-100/80 flex items-center justify-between"
                    >
                      <span>In 1 week</span>
                      <span class="font-mono text-[10px] text-slate-400">{{ getPresetDateLabel(7) }}</span>
                    </button>
                    @if (item.start_date) {
                      <button
                        type="button"
                        (click)="onDateChange(item.id, 'start_date', ''); openPopover.set(null)"
                        class="w-full px-2.5 py-1.5 text-left text-rose-600 hover:bg-rose-50"
                      >
                        Clear start date
                      </button>
                    }
                    <div class="px-2.5 pt-1.5 border-t border-slate-100">
                      <label class="block text-[10px] font-semibold uppercase text-slate-400 mb-1">Custom date</label>
                      <input
                        type="date"
                        [ngModel]="formatDateInput(item.start_date)"
                        (ngModelChange)="onDateChange(item.id, 'start_date', $event); openPopover.set(null)"
                        aria-label="Custom start date input"
                        class="w-full h-7 px-2 rounded border border-slate-200 bg-slate-50 text-xs text-slate-800 focus:outline-none focus:border-slate-300"
                      />
                    </div>
                  </div>
                }
              </div>

              <!-- Due Date Pill + Dropdown -->
              <div class="relative inline-block">
                <button
                  type="button"
                  [disabled]="!projectService.canEdit()"
                  (click)="togglePopover('due_date', $event)"
                  aria-label="Work item due date"
                  [attr.aria-expanded]="openPopover() === 'due_date'"
                  class="flex items-center justify-between gap-1.5 px-2 py-1 text-[11px] bg-white border border-slate-200 hover:bg-slate-50 hover:border-slate-300 rounded-md transition-colors focus:outline-none disabled:opacity-50"
                >
                  <span [ngClass]="item.due_date ? 'font-medium text-slate-700' : 'text-slate-400'">
                    {{ item.due_date ? (item.due_date | date: 'MMM d, y') : 'Due date' }}
                  </span>
                  <svg class="w-3.5 h-3.5 text-slate-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="m6 9 6 6 6-6" />
                  </svg>
                </button>

                @if (openPopover() === 'due_date') {
                  <div
                    role="listbox"
                    aria-label="Due date options"
                    (click)="$event.stopPropagation()"
                    class="absolute left-0 z-50 mt-1 w-52 rounded-lg bg-white border border-slate-200 shadow-[0_8px_24px_rgba(15,23,42,0.10)] py-1.5 text-xs space-y-1"
                  >
                    <button
                      type="button"
                      (click)="applyPresetDate(item.id, 'due_date', 0)"
                      class="w-full px-2.5 py-1.5 text-left text-slate-700 hover:bg-slate-100/80 flex items-center justify-between"
                    >
                      <span>Today</span>
                      <span class="font-mono text-[10px] text-slate-400">{{ getPresetDateLabel(0) }}</span>
                    </button>
                    <button
                      type="button"
                      (click)="applyPresetDate(item.id, 'due_date', 1)"
                      class="w-full px-2.5 py-1.5 text-left text-slate-700 hover:bg-slate-100/80 flex items-center justify-between"
                    >
                      <span>Tomorrow</span>
                      <span class="font-mono text-[10px] text-slate-400">{{ getPresetDateLabel(1) }}</span>
                    </button>
                    <button
                      type="button"
                      (click)="applyPresetDate(item.id, 'due_date', 7)"
                      class="w-full px-2.5 py-1.5 text-left text-slate-700 hover:bg-slate-100/80 flex items-center justify-between"
                    >
                      <span>In 1 week</span>
                      <span class="font-mono text-[10px] text-slate-400">{{ getPresetDateLabel(7) }}</span>
                    </button>
                    <button
                      type="button"
                      (click)="applyPresetDate(item.id, 'due_date', 14)"
                      class="w-full px-2.5 py-1.5 text-left text-slate-700 hover:bg-slate-100/80 flex items-center justify-between"
                    >
                      <span>In 2 weeks</span>
                      <span class="font-mono text-[10px] text-slate-400">{{ getPresetDateLabel(14) }}</span>
                    </button>
                    @if (item.due_date) {
                      <button
                        type="button"
                        (click)="onDateChange(item.id, 'due_date', ''); openPopover.set(null)"
                        class="w-full px-2.5 py-1.5 text-left text-rose-600 hover:bg-rose-50"
                      >
                        Clear due date
                      </button>
                    }
                    <div class="px-2.5 pt-1.5 border-t border-slate-100">
                      <label class="block text-[10px] font-semibold uppercase text-slate-400 mb-1">Custom date</label>
                      <input
                        type="date"
                        [ngModel]="formatDateInput(item.due_date)"
                        (ngModelChange)="onDateChange(item.id, 'due_date', $event); openPopover.set(null)"
                        aria-label="Custom due date input"
                        class="w-full h-7 px-2 rounded border border-slate-200 bg-slate-50 text-xs text-slate-800 focus:outline-none focus:border-slate-300"
                      />
                    </div>
                  </div>
                }
              </div>

              <!-- Estimate -->
              <app-dropdown
                [options]="getEstimateOptions(item)"
                [value]="item.estimate !== undefined && item.estimate !== null ? '' + item.estimate : ''"
                emptyLabel="Estimate"
                (valueChange)="onEstimateChange(item.id, $event)"
                [disabled]="!projectService.canEdit()"
                ariaLabel="Work item estimate"
                size="sm"
              ></app-dropdown>
            </div>

            <!-- Line 2: Separate Labels Line with 'Labels' in front and all labels kept together -->
            <div class="flex flex-wrap items-center gap-1.5" data-testid="work-item-labels">
              <span class="text-[11px] font-medium text-slate-400 mr-0.5">Labels:</span>

              @for (lbl of getAssignedLabels(item); track lbl.id) {
                <span
                  (click)="togglePopover('labels', $event)"
                  class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full border border-slate-200 bg-slate-50 hover:bg-slate-100 text-[11px] font-medium text-slate-700 cursor-pointer transition-colors"
                >
                  <span class="w-2 h-2 rounded-full shrink-0" [style.backgroundColor]="lbl.color"></span>
                  <span>{{ lbl.name }}</span>
                  @if (projectService.canEdit()) {
                    <button
                      type="button"
                      (click)="removeLabelFromItem($event, item.id, lbl.id)"
                      [attr.aria-label]="'Remove work item label ' + lbl.name"
                      class="text-slate-400 hover:text-rose-600 ml-0.5 leading-none"
                      title="Remove label"
                    >
                      ×
                    </button>
                  }
                </span>
              }

              @if (projectService.canEdit()) {
                <div class="relative inline-block">
                  <button
                    type="button"
                    (click)="togglePopover('labels', $event)"
                    aria-label="Toggle work item label dropdown"
                    [attr.aria-expanded]="openPopover() === 'labels'"
                    class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full border border-dashed border-slate-300 bg-white hover:bg-slate-50 hover:border-slate-400 text-[11px] font-medium text-slate-500 transition-colors"
                  >
                    <span>+</span>
                    <span>{{ getAssignedLabels(item).length === 0 ? 'Add label' : 'Add' }}</span>
                  </button>

                  @if (openPopover() === 'labels') {
                    <div
                      role="listbox"
                      aria-label="Work item labels options"
                      (click)="$event.stopPropagation()"
                      class="absolute left-0 z-50 mt-1 w-52 max-h-60 overflow-y-auto rounded-lg bg-white border border-slate-200 shadow-[0_8px_24px_rgba(15,23,42,0.10)] py-1 text-xs"
                    >
                      @for (lbl of projectService.labels(); track lbl.id) {
                        <button
                          type="button"
                          role="option"
                          [attr.aria-label]="'Toggle work item label ' + lbl.name"
                          [attr.aria-selected]="hasLabel(item, lbl.id)"
                          (click)="workItemService.toggleItemLabel(item.id, lbl.id)"
                          class="w-full flex items-center justify-between gap-2 px-2.5 py-1.5 text-left transition-colors"
                          [ngClass]="
                            hasLabel(item, lbl.id)
                              ? 'bg-sky-50/80 text-sky-700 font-medium'
                              : 'text-slate-700 hover:bg-slate-100/80'
                          "
                        >
                          <span class="flex items-center gap-2 truncate">
                            <span class="w-2 h-2 rounded-full shrink-0" [style.backgroundColor]="lbl.color"></span>
                            <span class="truncate">{{ lbl.name }}</span>
                          </span>
                          @if (hasLabel(item, lbl.id)) {
                            <svg class="w-3.5 h-3.5 text-sky-600 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
                              <path d="M20 6 9 17l-5-5" />
                            </svg>
                          }
                        </button>
                      } @empty {
                        <div class="px-2.5 py-2 text-[11px] text-slate-400">No project labels defined.</div>
                      }
                    </div>
                  }
                </div>
              } @else if (getAssignedLabels(item).length === 0) {
                <span class="text-[11px] text-slate-400">None</span>
              }
            </div>
          </div>

          <!-- 3. Description (Markdown) -->
          <div class="space-y-2">
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

          <!-- 4. Sub-Work Items Section -->
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

          <!-- 5. Threaded Discussion Comments -->
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
  readonly openPopover = signal<'start_date' | 'due_date' | 'labels' | null>(null);

  readonly statusOptions: DropdownOption[] = ALL_STATUSES.map((s) => ({
    value: s.value,
    label: s.label,
    dotColor: STATUS_DOT_COLORS[s.value],
  }));
  readonly priorityOptions: DropdownOption[] = ALL_PRIORITIES.map((p) => ({
    value: p.value,
    label: p.value === 'NONE' ? 'No priority' : p.label,
    dotColor: PRIORITY_DOT_COLORS[p.value],
  }));
  readonly assigneeOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'Unassigned' },
    ...this.projectService.members().map((m) => ({ value: m.user_id, label: m.name })),
  ]);
  readonly moduleOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'No module' },
    ...this.projectService.modules().map((m) => ({ value: m.id, label: m.name })),
  ]);
  readonly epicOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'No epic' },
    ...this.projectService.epics().map((e) => ({ value: e.id, label: e.name })),
  ]);
  readonly cycleOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'No cycle' },
    ...this.projectService.cycles().map((c) => ({ value: c.id, label: c.name })),
  ]);

  private readonly baseEstimatePoints = [1, 2, 3, 5, 8, 13, 21];

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

  getEstimateOptions(item: WorkItem): DropdownOption[] {
    const pts = [...this.baseEstimatePoints];
    if (item.estimate !== undefined && item.estimate !== null && !pts.includes(item.estimate)) {
      pts.push(item.estimate);
      pts.sort((a, b) => a - b);
    }
    return [
      { value: '', label: 'No estimate' },
      ...pts.map((n) => ({
        value: String(n),
        label: n === 1 ? '1 pt' : `${n} pts`,
      })),
    ];
  }

  onEstimateChange(itemId: string, val: string): void {
    const parsed = val === '' ? undefined : Number(val);
    this.workItemService.updateItem(itemId, { estimate: parsed });
  }

  onStatusChange(itemId: string, status: string): void {
    this.workItemService.updateItem(itemId, { status: status as WorkItemStatus });
  }

  onPriorityChange(itemId: string, priority: string): void {
    this.workItemService.updateItem(itemId, { priority: priority as WorkItemPriority });
  }

  hasLabel(item: WorkItem, labelId: string): boolean {
    return (item.label_ids ?? []).includes(labelId);
  }

  getAssignedLabels(item: WorkItem): Label[] {
    const ids = item.label_ids ?? [];
    if (ids.length === 0) return [];
    return this.projectService.labels().filter((l) => ids.includes(l.id));
  }

  removeLabelFromItem(event: MouseEvent, itemId: string, labelId: string): void {
    event.stopPropagation();
    this.workItemService.toggleItemLabel(itemId, labelId);
  }

  togglePopover(which: 'start_date' | 'due_date' | 'labels', event: MouseEvent): void {
    event.stopPropagation();
    if (!this.projectService.canEdit()) return;
    this.openPopover.update((curr) => (curr === which ? null : which));
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    if (this.openPopover()) {
      this.openPopover.set(null);
    }
  }

  getPresetDateLabel(offsetDays: number): string {
    const d = new Date();
    d.setDate(d.getDate() + offsetDays);
    return d.toISOString().slice(5, 10);
  }

  applyPresetDate(itemId: string, field: 'start_date' | 'due_date', offsetDays: number): void {
    const d = new Date();
    d.setDate(d.getDate() + offsetDays);
    const yyyyMmDd = d.toISOString().slice(0, 10);
    this.onDateChange(itemId, field, yyyyMmDd);
    this.openPopover.set(null);
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
