import { Injectable, computed, effect, inject, signal } from '@angular/core';
import { ApiService } from './api.service';
import { ProjectService } from './project.service';
import { AuthService } from './auth.service';
import {
  Comment,
  SubWorkItem,
  WorkItem,
  WorkItemFilterState,
  WorkItemPriority,
  WorkItemStatus,
  WorkItemViewMode,
} from '../models/project.model';

@Injectable({
  providedIn: 'root',
})
export class WorkItemService {
  private readonly api = inject(ApiService);
  private readonly projectService = inject(ProjectService);
  private readonly authService = inject(AuthService);

  readonly activeView = signal<WorkItemViewMode>('BOARD');
  readonly isQuickCreateOpen = signal<boolean>(false);

  readonly filter = signal<WorkItemFilterState>({
    status: '',
    priority: '',
    module_id: '',
    epic_id: '',
    cycle_id: '',
    assignee_id: '',
    label_id: '',
    search: '',
  });

  readonly selectedItemId = signal<string | null>(null);

  readonly items = signal<WorkItem[]>([]);
  readonly subItemsByItem = signal<Record<string, SubWorkItem[]>>({});
  readonly commentsByItem = signal<Record<string, Comment[]>>({});
  private readonly updateSeqByItem = new Map<string, number>();

  constructor() {
    effect(() => {
      const projects = this.projectService.projects();
      for (const proj of projects) {
        this.loadItems(proj.id);
      }
    });
  }

  readonly filteredItems = computed<WorkItem[]>(() => {
    const f = this.filter();
    const projectId = this.projectService.activeProjectId();
    const q = f.search.trim().toLowerCase();

    return this.items().filter((item) => {
      if (item.project_id !== projectId) return false;
      if (f.status && item.status !== f.status) return false;
      if (f.priority && item.priority !== f.priority) return false;
      if (f.module_id && item.module_id !== f.module_id) return false;
      if (f.epic_id && item.epic_id !== f.epic_id) return false;
      if (f.cycle_id && item.cycle_id !== f.cycle_id) return false;
      if (f.assignee_id && item.assignee_id !== f.assignee_id) return false;
      if (f.label_id && !(item.label_ids || []).includes(f.label_id)) return false;
      if (q) {
        const hay = `${item.key} ${item.title} ${item.description}`.toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });
  });

  readonly selectedItem = computed<WorkItem | null>(() => {
    const id = this.selectedItemId();
    if (!id) return null;
    return this.items().find((i) => i.id === id) ?? null;
  });

  readonly myAssignedItems = computed<WorkItem[]>(() => {
    const userId = this.authService.currentUser().user_id;
    const q = this.filter().search.trim().toLowerCase();
    return this.items().filter((item) => {
      if (item.assignee_id !== userId) return false;
      if (q) {
        const hay = `${item.key} ${item.title} ${item.description}`.toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });
  });

  getItemsForProject(projectId: string): WorkItem[] {
    return this.items().filter((item) => item.project_id === projectId);
  }

  loadItems(projectId: string): void {
    if (!projectId) return;
    this.api.get<WorkItem[]>(`/projects/${projectId}/items`).subscribe({
      next: (res) => {
        const incoming = res ?? [];
        this.items.update((prev) => [
          ...prev.filter((i) => i.project_id !== projectId),
          ...incoming,
        ]);
      },
    });
  }

  updateFilter(patch: Partial<WorkItemFilterState>): void {
    this.filter.update((curr) => ({ ...curr, ...patch }));
  }

  resetFilters(): void {
    this.filter.set({
      status: '',
      priority: '',
      module_id: '',
      epic_id: '',
      cycle_id: '',
      assignee_id: '',
      label_id: '',
      search: '',
    });
  }

  private resolveProjectIdForItem(itemId: string): string {
    const found = this.items().find((i) => i.id === itemId);
    return found?.project_id || this.projectService.activeProjectId();
  }

  openItemDetail(itemId: string): void {
    this.selectedItemId.set(itemId);
    const projectId = this.resolveProjectIdForItem(itemId);
    if (!projectId) return;
    this.loadItemRelations(projectId, itemId);
  }

  loadItemRelations(projectId: string, itemId: string): void {
    if (!projectId || !itemId) return;
    this.api.get<SubWorkItem[]>(`/projects/${projectId}/items/${itemId}/sub-items`).subscribe({
      next: (subs) => {
        this.subItemsByItem.update((m) => ({ ...m, [itemId]: subs ?? [] }));
      },
    });
    this.api.get<Comment[]>(`/projects/${projectId}/items/${itemId}/comments`).subscribe({
      next: (cmts) => {
        this.commentsByItem.update((m) => ({ ...m, [itemId]: cmts ?? [] }));
      },
    });
  }

  closeItemDetail(): void {
    this.selectedItemId.set(null);
  }

  createItem(input: {
    project_id?: string;
    title: string;
    description: string;
    status: WorkItemStatus;
    priority: WorkItemPriority;
    module_id?: string;
    epic_id?: string;
    cycle_id?: string;
    assignee_id?: string;
    label_ids?: string[];
    estimate?: number;
    start_date?: string;
    due_date?: string;
  }): void {
    const projectId = input.project_id || this.projectService.activeProjectId();
    if (!projectId) return;
    const payload = {
      ...input,
      start_date: input.start_date ? new Date(input.start_date).toISOString() : undefined,
      due_date: input.due_date ? new Date(input.due_date).toISOString() : undefined,
    };

    this.api.post<WorkItem>(`/projects/${projectId}/items`, payload).subscribe({
      next: (created) => {
        this.items.update((prev) => [created, ...prev]);
        this.projectService.projects.update((projs) =>
          projs.map((p) =>
            p.id === projectId
              ? { ...p, item_seq: Math.max(p.item_seq, created.sequence_id) }
              : p
          )
        );
      },
    });
  }

  updateItem(itemId: string, patch: Partial<WorkItem>): void {
    const projectId = this.resolveProjectIdForItem(itemId);
    if (!projectId) return;

    const current = this.items().find((i) => i.id === itemId);
    const apiPatch: Record<string, unknown> = { ...patch };
    const optimisticPatch: Partial<WorkItem> = { ...patch };

    if ('start_date' in patch) {
      if (!patch.start_date) {
        apiPatch['start_date'] = '';
        optimisticPatch.start_date = undefined;
      } else {
        const iso = patch.start_date.includes('T')
          ? patch.start_date
          : new Date(patch.start_date).toISOString();
        apiPatch['start_date'] = iso;
        optimisticPatch.start_date = iso;
      }
    }

    if ('due_date' in patch) {
      if (!patch.due_date) {
        apiPatch['due_date'] = '';
        optimisticPatch.due_date = undefined;
      } else {
        const iso = patch.due_date.includes('T')
          ? patch.due_date
          : new Date(patch.due_date).toISOString();
        apiPatch['due_date'] = iso;
        optimisticPatch.due_date = iso;
      }
    }

    if (current) {
      const hasChanges = (Object.keys(optimisticPatch) as (keyof WorkItem)[]).some(
        (key) => current[key] !== optimisticPatch[key]
      );
      if (!hasChanges) return;
    }

    const seq = (this.updateSeqByItem.get(itemId) ?? 0) + 1;
    this.updateSeqByItem.set(itemId, seq);

    // Optimistic update for responsive 60fps UI, confirmed by backend PATCH response
    this.items.update((list) =>
      list.map((it) =>
        it.id === itemId
          ? { ...it, ...optimisticPatch, updated_at: new Date().toISOString() }
          : it
      )
    );

    this.api.patch<WorkItem>(`/projects/${projectId}/items/${itemId}`, apiPatch).subscribe({
      next: (updated) => {
        if (this.updateSeqByItem.get(itemId) === seq) {
          this.items.update((list) => list.map((it) => (it.id === itemId ? updated : it)));
        }
      },
    });
  }

  toggleItemLabel(itemId: string, labelId: string): void {
    const current = this.items().find((i) => i.id === itemId);
    if (!current) return;
    const existing = current.label_ids ?? [];
    const nextIds = existing.includes(labelId)
      ? existing.filter((id) => id !== labelId)
      : [...existing, labelId];
    this.updateItem(itemId, { label_ids: nextIds });
  }

  deleteItem(itemId: string): void {
    const projectId = this.resolveProjectIdForItem(itemId);
    if (!projectId) return;
    this.api.delete(`/projects/${projectId}/items/${itemId}`).subscribe({
      next: () => {
        this.items.update((list) => list.filter((i) => i.id !== itemId));
        if (this.selectedItemId() === itemId) {
          this.selectedItemId.set(null);
        }
      },
    });
  }

  addSubItem(itemId: string, title: string): void {
    const projectId = this.resolveProjectIdForItem(itemId);
    if (!projectId) return;
    this.api
      .post<SubWorkItem>(`/projects/${projectId}/items/${itemId}/sub-items`, {
        title,
        is_completed: false,
      })
      .subscribe({
        next: (sub) => {
          this.subItemsByItem.update((m) => ({
            ...m,
            [itemId]: [...(m[itemId] ?? []), sub],
          }));
          this.recalcSubCounts(itemId);
        },
      });
  }

  toggleSubItem(itemId: string, subItem: SubWorkItem): void {
    const projectId = this.resolveProjectIdForItem(itemId);
    if (!projectId) return;
    const nextCompleted = !subItem.is_completed;
    this.api
      .patch<SubWorkItem>(`/projects/${projectId}/items/${itemId}/sub-items/${subItem.id}`, {
        is_completed: nextCompleted,
      })
      .subscribe({
        next: (updated) => {
          this.subItemsByItem.update((m) => ({
            ...m,
            [itemId]: (m[itemId] ?? []).map((s) => (s.id === subItem.id ? updated : s)),
          }));
          this.recalcSubCounts(itemId);
        },
      });
  }

  deleteSubItem(itemId: string, subItemId: string): void {
    const projectId = this.resolveProjectIdForItem(itemId);
    if (!projectId) return;
    this.api
      .delete(`/projects/${projectId}/items/${itemId}/sub-items/${subItemId}`)
      .subscribe({
        next: () => {
          this.subItemsByItem.update((m) => ({
            ...m,
            [itemId]: (m[itemId] ?? []).filter((s) => s.id !== subItemId),
          }));
          this.recalcSubCounts(itemId);
        },
      });
  }

  addComment(itemId: string, body: string): void {
    const projectId = this.resolveProjectIdForItem(itemId);
    if (!projectId) return;
    this.api.post<Comment>(`/projects/${projectId}/items/${itemId}/comments`, { body }).subscribe({
      next: (cmt) => {
        this.commentsByItem.update((m) => ({
          ...m,
          [itemId]: [...(m[itemId] ?? []), cmt],
        }));
      },
    });
  }

  private recalcSubCounts(itemId: string): void {
    const subs = this.subItemsByItem()[itemId] ?? [];
    const total = subs.length;
    const done = subs.filter((s) => s.is_completed).length;
    this.items.update((list) =>
      list.map((i) =>
        i.id === itemId ? { ...i, sub_item_count: total, completed_sub_count: done } : i
      )
    );
  }
}
