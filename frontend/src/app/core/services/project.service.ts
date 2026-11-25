import { Injectable, WritableSignal, computed, inject, signal } from '@angular/core';
import { ApiService } from './api.service';
import { AuthService } from './auth.service';
import {
  Cycle,
  Epic,
  Label,
  Module,
  Project,
  ProjectMember,
  ProjectRole,
  ProjectSettingsModel,
  ScionAgent,
  ScionDefaults,
} from '../models/project.model';

export type ProjectCollectionKind = 'cycles' | 'epics' | 'modules';

@Injectable({
  providedIn: 'root',
})
export class ProjectService {
  private readonly api = inject(ApiService);
  private readonly auth = inject(AuthService);

  readonly projects = signal<Project[]>([]);
  readonly activeProjectId = signal<string>('proj-jynko-core');
  private pendingRouteProject: string | null = null;

  readonly activeProject = computed<Project | undefined>(() =>
    this.projects().find((p) => p.id === this.activeProjectId()) ?? this.projects()[0]
  );

  /** True once the accessible project list has been loaded (changes only on empty <-> non-empty). */
  readonly hasProjects = computed<boolean>(() => this.projects().length > 0);

  getProjectById(projectId: string): Project | undefined {
    return this.projects().find((p) => p.id === projectId);
  }

  resolveProject(idOrSlug: string, list: Project[] = this.projects()): Project | undefined {
    if (!idOrSlug) return undefined;
    const clean = idOrSlug.trim().toLowerCase();
    return (
      list.find((p) => p.id === idOrSlug) ??
      list.find((p) => p.id.toLowerCase() === clean) ??
      list.find((p) => p.name.toLowerCase() === clean) ??
      list.find((p) => p.key_prefix.toLowerCase() === clean)
    );
  }

  activateProjectFromRoute(projectIdOrSlug: string): string {
    if (!projectIdOrSlug) {
      return this.activeProjectId();
    }
    this.pendingRouteProject = projectIdOrSlug;
    const matched = this.resolveProject(projectIdOrSlug);
    if (matched) {
      if (this.activeProjectId() !== matched.id) {
        this.selectProject(matched.id);
      }
      return matched.id;
    }
    if (this.activeProjectId() !== projectIdOrSlug) {
      this.selectProject(projectIdOrSlug);
    }
    return projectIdOrSlug;
  }

  readonly members = signal<ProjectMember[]>([]);
  readonly modules = signal<Module[]>([]);
  readonly epics = signal<Epic[]>([]);
  readonly cycles = signal<Cycle[]>([]);
  readonly labels = signal<Label[]>([]);

  readonly currentUserRole = computed<ProjectRole>(() => {
    const user = this.auth.currentUser();
    const userEmail = (user.email || '').trim().toLowerCase();
    const list = this.members();
    const match = list.find(
      (m) =>
        m.user_id === user.user_id ||
        (!!userEmail && (m.email || '').trim().toLowerCase() === userEmail) ||
        (!!userEmail && m.user_id.trim().toLowerCase() === userEmail)
    );
    if (match) {
      return match.role;
    }
    const active = this.activeProject();
    if (active && active.created_by && active.created_by === user.user_id) {
      return 'ADMIN';
    }
    return 'VIEWER';
  });

  readonly canEdit = computed<boolean>(() => {
    const role = this.currentUserRole();
    return role === 'ADMIN' || role === 'MEMBER';
  });

  readonly isAdmin = computed<boolean>(() => this.currentUserRole() === 'ADMIN');

  /**
   * PLAN-12: tracks which project each lazily loaded collection (cycles, epics,
   * modules) currently holds, so pages fetch only their own data on navigation
   * and supporting pickers can lazily "ensure" a collection without refetching.
   */
  private readonly loadedCollections = new Map<ProjectCollectionKind, string>();
  private readonly inflightRequests = new Map<string, Promise<void>>();

  readonly isProjectModalOpen = signal<boolean>(false);
  readonly editingProject = signal<Project | null>(null);

  constructor() {
    this.auth.registerSessionLifecycle({
      onReset: () => this.resetState(),
      onReload: () => this.loadProjects(),
    });
  }

  resetState(): void {
    this.projects.set([]);
    this.members.set([]);
    this.modules.set([]);
    this.epics.set([]);
    this.cycles.set([]);
    this.labels.set([]);
    this.loadedCollections.clear();
    this.inflightRequests.clear();
    this.pendingRouteProject = null;
    this.closeProjectModal();
  }

  openCreateProjectModal(): void {
    this.editingProject.set(null);
    this.isProjectModalOpen.set(true);
  }

  openEditProjectModal(project: Project): void {
    this.editingProject.set(project);
    this.isProjectModalOpen.set(true);
  }

  closeProjectModal(): void {
    this.isProjectModalOpen.set(false);
    this.editingProject.set(null);
  }

  loadProjects(): void {
    if (this.auth.isSignedOut()) {
      return;
    }
    this.api.get<Project[]>('/projects').subscribe({
      next: (list) => {
        const projects = list ?? [];
        this.projects.set(projects);
        if (projects.length > 0) {
          const targetKey = this.pendingRouteProject || this.activeProjectId();
          const matched = this.resolveProject(targetKey, projects) ?? projects[0];
          this.selectProject(matched.id);
        } else {
          this.members.set([]);
          this.modules.set([]);
          this.epics.set([]);
          this.cycles.set([]);
          this.labels.set([]);
        }
      },
    });
  }

  selectProject(projectId: string): void {
    this.pendingRouteProject = projectId;
    const changed = this.activeProjectId() !== projectId;
    this.activeProjectId.set(projectId);
    if (changed) {
      this.members.set([]);
      this.modules.set([]);
      this.epics.set([]);
      this.cycles.set([]);
      this.labels.set([]);
      this.loadedCollections.clear();
    }
    this.loadProjectMetadata(projectId);
  }

  /**
   * Loads the lightweight project metadata every project page depends on
   * (members for RBAC, labels for badges). Page-specific collections
   * (cycles, epics, modules) are fetched by the page that shows them.
   */
  loadProjectMetadata(projectId: string): Promise<void> {
    if (!projectId || this.auth.isSignedOut()) return Promise.resolve();
    return Promise.all([
      this.dedupe(`members:${projectId}`, () =>
        this.fetchInto<ProjectMember>(projectId, 'members', this.members)
      ),
      this.dedupe(`labels:${projectId}`, () =>
        this.fetchInto<Label>(projectId, 'labels', this.labels)
      ),
    ]).then(() => undefined);
  }

  /** Re-fetches the project record itself plus its members and labels (Overview page). */
  refreshProjectOverview(projectId: string = this.activeProjectId()): Promise<void> {
    if (!projectId || this.auth.isSignedOut()) return Promise.resolve();
    const projectReq = new Promise<void>((resolve) => {
      this.api.get<Project>(`/projects/${projectId}`).subscribe({
        next: (saved) => {
          if (saved) {
            this.projects.update((prev) => prev.map((p) => (p.id === saved.id ? saved : p)));
          }
          resolve();
        },
        error: () => resolve(),
      });
    });
    return Promise.all([projectReq, this.loadProjectMetadata(projectId)]).then(() => undefined);
  }

  /** Always fetches the latest cycles of the project from the server. */
  loadCycles(projectId: string = this.activeProjectId()): Promise<void> {
    return this.loadCollection<Cycle>('cycles', projectId, this.cycles);
  }

  /** Always fetches the latest epics of the project from the server. */
  loadEpics(projectId: string = this.activeProjectId()): Promise<void> {
    return this.loadCollection<Epic>('epics', projectId, this.epics);
  }

  /** Always fetches the latest modules of the project from the server. */
  loadModules(projectId: string = this.activeProjectId()): Promise<void> {
    return this.loadCollection<Module>('modules', projectId, this.modules);
  }

  /** True when the given collection already holds data for the project. */
  isCollectionLoaded(kind: ProjectCollectionKind, projectId: string = this.activeProjectId()): boolean {
    return !!projectId && this.loadedCollections.get(kind) === projectId;
  }

  /**
   * Fetches cycles / epics / modules only if they have not been loaded yet for
   * the project. Used by pickers (filters, drawer, quick-create) that need the
   * options but are not the page that owns those collections.
   */
  ensurePlanningCollections(projectId: string = this.activeProjectId()): Promise<void> {
    if (!projectId) return Promise.resolve();
    const pending: Promise<void>[] = [];
    if (!this.isCollectionLoaded('cycles', projectId)) pending.push(this.loadCycles(projectId));
    if (!this.isCollectionLoaded('epics', projectId)) pending.push(this.loadEpics(projectId));
    if (!this.isCollectionLoaded('modules', projectId)) pending.push(this.loadModules(projectId));
    return Promise.all(pending).then(() => undefined);
  }

  private loadCollection<T>(
    kind: ProjectCollectionKind,
    projectId: string,
    target: WritableSignal<T[]>
  ): Promise<void> {
    if (!projectId || this.auth.isSignedOut()) return Promise.resolve();
    return this.dedupe(`${kind}:${projectId}`, () =>
      this.fetchInto<T>(projectId, kind, target, () => this.loadedCollections.set(kind, projectId))
    );
  }

  /** Shares a single in-flight request between concurrent callers of the same key. */
  private dedupe(key: string, start: () => Promise<void>): Promise<void> {
    const inflight = this.inflightRequests.get(key);
    if (inflight) return inflight;
    const req = start().finally(() => this.inflightRequests.delete(key));
    this.inflightRequests.set(key, req);
    return req;
  }

  private fetchInto<T>(
    projectId: string,
    path: string,
    target: WritableSignal<T[]>,
    onLoaded?: () => void
  ): Promise<void> {
    return new Promise<void>((resolve) => {
      this.api.get<T[]>(`/projects/${projectId}/${path}`).subscribe({
        next: (res) => {
          if (this.activeProjectId() === projectId) {
            target.set(res ?? []);
            onLoaded?.();
          }
          resolve();
        },
        error: () => resolve(),
      });
    });
  }

  createProject(
    name: string,
    keyPrefix: string,
    description: string,
    onCreated?: (project: Project) => void
  ): void {
    const cleanName = name.trim();
    const cleanPrefix = keyPrefix.trim().toUpperCase() || 'JYNKO';
    const cleanDesc = description.trim();
    const payload = { name: cleanName, key_prefix: cleanPrefix, description: cleanDesc };

    this.api.post<Project>('/projects', payload).subscribe({
      next: (created) => {
        this.projects.update((prev) => [...prev, created]);
        this.selectProject(created.id);
        if (onCreated) {
          onCreated(created);
        }
      },
    });
  }

  updateProject(
    projectId: string,
    patch: { name?: string; key_prefix?: string; description?: string; settings?: ProjectSettingsModel },
    onUpdated?: (project: Project) => void
  ): void {
    if (!projectId) return;
    const current = this.projects().find((p) => p.id === projectId);
    if (!current) return;

    const payload: Record<string, unknown> = {};
    const optimistic: Partial<Project> = {};

    if (patch.name !== undefined) {
      const trimmed = patch.name.trim();
      if (!trimmed) return;
      payload['name'] = trimmed;
      optimistic.name = trimmed;
    }
    if (patch.key_prefix !== undefined) {
      const cleanPrefix = patch.key_prefix.trim().toUpperCase();
      if (cleanPrefix) {
        payload['key_prefix'] = cleanPrefix;
        optimistic.key_prefix = cleanPrefix;
      }
    }
    if (patch.description !== undefined) {
      const trimmed = patch.description.trim();
      payload['description'] = trimmed;
      optimistic.description = trimmed;
    }
    if (patch.settings !== undefined) {
      payload['settings'] = patch.settings;
      optimistic.settings = patch.settings;
    }

    this.projects.update((prev) =>
      prev.map((p) => (p.id === projectId ? { ...p, ...optimistic } : p))
    );

    this.api.patch<Project>(`/projects/${projectId}`, payload).subscribe({
      next: (saved) => {
        this.projects.update((prev) => prev.map((p) => (p.id === projectId ? saved : p)));
        if (onUpdated) {
          onUpdated(saved);
        }
      },
      error: () => {
        this.projects.update((prev) => prev.map((p) => (p.id === projectId ? current : p)));
      },
    });
  }

  /**
   * Deletes a project (PLAN-13). The backend soft-deletes it (it can be restored via the API until the
   * cleanup job purges it); locally it disappears from the project list right away. If it was the active
   * project, the next remaining project becomes active. Rejects with the server error message on failure.
   */
  deleteProject(projectId: string): Promise<void> {
    if (!projectId) return Promise.reject(new Error('Missing project ID'));
    return new Promise<void>((resolve, reject) => {
      this.api.delete<unknown>(`/projects/${encodeURIComponent(projectId)}`).subscribe({
        next: () => {
          const wasActive = this.activeProjectId() === projectId;
          const remaining = this.projects().filter((p) => p.id !== projectId);
          this.projects.set(remaining);
          if (this.editingProject()?.id === projectId) {
            this.closeProjectModal();
          }
          if (wasActive) {
            if (remaining.length > 0) {
              this.selectProject(remaining[0].id);
            } else {
              this.pendingRouteProject = null;
              this.activeProjectId.set('');
              this.members.set([]);
              this.modules.set([]);
              this.epics.set([]);
              this.cycles.set([]);
              this.labels.set([]);
              this.loadedCollections.clear();
            }
          }
          resolve();
        },
        error: (err) => {
          reject(new Error(err?.error?.error || err?.message || 'Failed to delete project'));
        },
      });
    });
  }

  upsertMember(email: string, name: string, role: ProjectRole): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const cleanEmail = email.trim();
    const payload = { user_id: cleanEmail, email: cleanEmail, name: name.trim(), role };
    this.api.post<ProjectMember>(`/projects/${projectId}/members`, payload).subscribe({
      next: (m) => {
        this.members.update((list) => {
          const filtered = list.filter(
            (x) =>
              x.user_id !== m.user_id &&
              (x.email || '').toLowerCase() !== (m.email || '').toLowerCase()
          );
          return [...filtered, m];
        });
      },
    });
  }

  updateMemberName(userId: string, name: string): void {
    const projectId = this.activeProjectId();
    if (!projectId || !userId) return;
    const current = this.members().find((m) => m.user_id === userId);
    const cleanName = (name ?? '').trim();
    if (!current || !cleanName || cleanName === current.name) return;
    this.members.update((list) =>
      list.map((m) => (m.user_id === userId ? { ...m, name: cleanName } : m))
    );
    this.api
      .patch<ProjectMember>(`/projects/${projectId}/members/${encodeURIComponent(userId)}`, {
        name: cleanName,
      })
      .subscribe({
        next: (saved) => {
          this.members.update((list) => list.map((m) => (m.user_id === userId ? saved : m)));
        },
        error: () => {
          // Roll back only this member so concurrent edits to other members are preserved.
          this.members.update((list) => list.map((m) => (m.user_id === userId ? current : m)));
        },
      });
  }

  removeMember(userId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId || !userId) return;
    const prevMembers = this.members();
    this.members.update((list) => list.filter((m) => m.user_id !== userId));
    this.api
      .delete(`/projects/${projectId}/members/${encodeURIComponent(userId)}`)
      .subscribe({
        error: () => {
          this.members.set(prevMembers);
        },
      });
  }

  createModule(
    name: string,
    description: string,
    leadUserId: string,
    gitRepoUrl?: string,
    baseBranch?: string
  ): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.api
      .post<Module>(`/projects/${projectId}/modules`, {
        name,
        description,
        lead_user_id: leadUserId,
        ...(gitRepoUrl?.trim() ? { git_repo_url: gitRepoUrl.trim() } : {}),
        ...(baseBranch?.trim() ? { base_branch: baseBranch.trim() } : {}),
      })
      .subscribe({
        next: (mod) => this.modules.update((prev) => [...prev, mod]),
      });
  }

  updateModule(
    moduleId: string,
    patch: {
      name?: string;
      description?: string;
      lead_user_id?: string;
      git_repo_url?: string;
      base_branch?: string;
    }
  ): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const current = this.modules().find((m) => m.id === moduleId);
    if (!current) return;

    const payload: Record<string, unknown> = {};
    const optimistic: Partial<Module> = {};

    if (patch.name !== undefined) {
      const trimmed = patch.name.trim();
      if (!trimmed) return;
      payload['name'] = trimmed;
      optimistic.name = trimmed;
    }
    if (patch.description !== undefined) {
      const trimmed = patch.description.trim();
      payload['description'] = trimmed;
      optimistic.description = trimmed;
    }
    if (patch.lead_user_id !== undefined) {
      const trimmed = patch.lead_user_id.trim();
      payload['lead_user_id'] = trimmed;
      optimistic.lead_user_id = trimmed;
    }
    if (patch.git_repo_url !== undefined) {
      const trimmed = patch.git_repo_url.trim();
      payload['git_repo_url'] = trimmed;
      optimistic.git_repo_url = trimmed;
    }
    if (patch.base_branch !== undefined) {
      const trimmed = patch.base_branch.trim();
      payload['base_branch'] = trimmed;
      optimistic.base_branch = trimmed;
    }

    this.modules.update((prev) =>
      prev.map((m) => (m.id === moduleId ? { ...m, ...optimistic } : m))
    );

    this.api.patch<Module>(`/projects/${projectId}/modules/${moduleId}`, payload).subscribe({
      next: (saved) => {
        this.modules.update((prev) => prev.map((m) => (m.id === moduleId ? saved : m)));
      },
      error: () => {
        this.modules.update((prev) => prev.map((m) => (m.id === moduleId ? current : m)));
      },
    });
  }

  deleteModule(moduleId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const snapshot = this.modules();
    this.modules.update((prev) => prev.filter((m) => m.id !== moduleId));
    this.api.delete(`/projects/${projectId}/modules/${moduleId}`).subscribe({
      error: () => this.modules.set(snapshot),
    });
  }

  getScionDefaults(projectId: string) {
    return this.api.get<ScionDefaults>(`/projects/${projectId}/scion/defaults`);
  }

  listScionAgents(projectId: string) {
    return this.api.get<{ configured: boolean; hub_url: string; agents: ScionAgent[]; error?: string }>(
      `/projects/${projectId}/scion/agents`
    );
  }

  createEpic(
    name: string,
    description: string,
    status: Epic['status'],
    startDate?: string,
    targetDate?: string
  ): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const payload = {
      name,
      description,
      status,
      start_date: startDate ? new Date(startDate).toISOString() : undefined,
      target_date: targetDate ? new Date(targetDate).toISOString() : undefined,
    };
    this.api.post<Epic>(`/projects/${projectId}/epics`, payload).subscribe({
      next: (epic) => this.epics.update((prev) => [...prev, epic]),
    });
  }

  updateEpic(
    epicId: string,
    patch: {
      name?: string;
      description?: string;
      status?: Epic['status'];
      start_date?: string;
      target_date?: string;
    }
  ): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const current = this.epics().find((e) => e.id === epicId);
    if (!current) return;

    const payload: Record<string, unknown> = {};
    const optimistic: Partial<Epic> = {};

    if (patch.name !== undefined) {
      const trimmed = patch.name.trim();
      if (!trimmed) return;
      payload['name'] = trimmed;
      optimistic.name = trimmed;
    }
    if (patch.description !== undefined) {
      const trimmed = patch.description.trim();
      payload['description'] = trimmed;
      optimistic.description = trimmed;
    }
    if (patch.status !== undefined) {
      payload['status'] = patch.status;
      optimistic.status = patch.status;
    }
    if (patch.start_date !== undefined) {
      if (!patch.start_date) {
        payload['start_date'] = '';
        optimistic.start_date = undefined;
      } else {
        const iso = patch.start_date.includes('T')
          ? patch.start_date
          : new Date(patch.start_date).toISOString();
        payload['start_date'] = iso;
        optimistic.start_date = iso;
      }
    }
    if (patch.target_date !== undefined) {
      if (!patch.target_date) {
        payload['target_date'] = '';
        optimistic.target_date = undefined;
      } else {
        const iso = patch.target_date.includes('T')
          ? patch.target_date
          : new Date(patch.target_date).toISOString();
        payload['target_date'] = iso;
        optimistic.target_date = iso;
      }
    }

    this.epics.update((prev) =>
      prev.map((e) => (e.id === epicId ? { ...e, ...optimistic } : e))
    );

    this.api.patch<Epic>(`/projects/${projectId}/epics/${epicId}`, payload).subscribe({
      next: (saved) => {
        this.epics.update((prev) => prev.map((e) => (e.id === epicId ? saved : e)));
      },
      error: () => {
        this.epics.update((prev) => prev.map((e) => (e.id === epicId ? current : e)));
      },
    });
  }

  deleteEpic(epicId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const snapshot = this.epics();
    this.epics.update((prev) => prev.filter((e) => e.id !== epicId));
    this.api.delete(`/projects/${projectId}/epics/${epicId}`).subscribe({
      error: () => this.epics.set(snapshot),
    });
  }

  createCycle(
    name: string,
    description: string,
    startDate: string,
    endDate: string,
    isActive: boolean
  ): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const payload = {
      name,
      description,
      start_date: new Date(startDate).toISOString(),
      end_date: new Date(endDate).toISOString(),
      is_active: isActive,
    };
    this.api.post<Cycle>(`/projects/${projectId}/cycles`, payload).subscribe({
      next: (cycle) => this.cycles.update((prev) => [...prev, cycle]),
    });
  }

  updateCycle(
    cycleId: string,
    patch: {
      name?: string;
      description?: string;
      start_date?: string;
      end_date?: string;
      is_active?: boolean;
    }
  ): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const current = this.cycles().find((c) => c.id === cycleId);
    if (!current) return;

    const payload: Record<string, unknown> = {};
    const optimistic: Partial<Cycle> = {};

    if (patch.name !== undefined) {
      const trimmed = patch.name.trim();
      if (!trimmed) return;
      payload['name'] = trimmed;
      optimistic.name = trimmed;
    }
    if (patch.description !== undefined) {
      const trimmed = patch.description.trim();
      payload['description'] = trimmed;
      optimistic.description = trimmed;
    }
    if (patch.start_date !== undefined && patch.start_date) {
      const iso = patch.start_date.includes('T')
        ? patch.start_date
        : new Date(patch.start_date).toISOString();
      payload['start_date'] = iso;
      optimistic.start_date = iso;
    }
    if (patch.end_date !== undefined && patch.end_date) {
      const iso = patch.end_date.includes('T')
        ? patch.end_date
        : new Date(patch.end_date).toISOString();
      payload['end_date'] = iso;
      optimistic.end_date = iso;
    }
    if (patch.is_active !== undefined) {
      payload['is_active'] = patch.is_active;
      optimistic.is_active = patch.is_active;
    }

    this.cycles.update((prev) =>
      prev.map((c) => (c.id === cycleId ? { ...c, ...optimistic } : c))
    );

    this.api.patch<Cycle>(`/projects/${projectId}/cycles/${cycleId}`, payload).subscribe({
      next: (saved) => {
        this.cycles.update((prev) => prev.map((c) => (c.id === cycleId ? saved : c)));
      },
      error: () => {
        this.cycles.update((prev) => prev.map((c) => (c.id === cycleId ? current : c)));
      },
    });
  }

  deleteCycle(cycleId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const snapshot = this.cycles();
    this.cycles.update((prev) => prev.filter((c) => c.id !== cycleId));
    this.api.delete(`/projects/${projectId}/cycles/${cycleId}`).subscribe({
      error: () => this.cycles.set(snapshot),
    });
  }

  createLabel(name: string, color: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.api.post<Label>(`/projects/${projectId}/labels`, { name, color }).subscribe({
      next: (lbl) => this.labels.update((prev) => [...prev, lbl]),
    });
  }

  updateLabel(labelId: string, patch: { name?: string; color?: string }): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const current = this.labels().find((l) => l.id === labelId);
    if (!current) return;

    const nextName = patch.name !== undefined ? patch.name.trim() : current.name;
    const nextColor = patch.color !== undefined ? patch.color.trim() : current.color;
    if (!nextName || (nextName === current.name && nextColor === current.color)) {
      return;
    }

    const updatedLocal: Label = { ...current, name: nextName, color: nextColor };
    this.labels.update((prev) => prev.map((l) => (l.id === labelId ? updatedLocal : l)));

    this.api
      .patch<Label>(`/projects/${projectId}/labels/${labelId}`, {
        name: nextName,
        color: nextColor,
      })
      .subscribe({
        next: (saved) => {
          this.labels.update((prev) => prev.map((l) => (l.id === labelId ? saved : l)));
        },
        error: () => {
          this.labels.update((prev) => prev.map((l) => (l.id === labelId ? current : l)));
        },
      });
  }

  deleteLabel(labelId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const snapshot = this.labels();
    this.labels.update((prev) => prev.filter((l) => l.id !== labelId));
    this.api.delete(`/projects/${projectId}/labels/${labelId}`).subscribe({
      error: () => this.labels.set(snapshot),
    });
  }
}
