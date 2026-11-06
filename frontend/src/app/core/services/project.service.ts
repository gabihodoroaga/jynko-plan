import { Injectable, computed, inject, signal } from '@angular/core';
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
} from '../models/project.model';

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
    const list = this.members();
    const match = list.find(
      (m) => m.user_id === user.user_id || m.email.toLowerCase() === user.email.toLowerCase()
    );
    return match ? match.role : 'ADMIN';
  });

  readonly canEdit = computed<boolean>(() => {
    const role = this.currentUserRole();
    return role === 'ADMIN' || role === 'MEMBER';
  });

  readonly isAdmin = computed<boolean>(() => this.currentUserRole() === 'ADMIN');

  readonly isProjectModalOpen = signal<boolean>(false);
  readonly editingProject = signal<Project | null>(null);

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
    this.api.get<Project[]>('/projects').subscribe({
      next: (list) => {
        const projects = list ?? [];
        this.projects.set(projects);
        if (projects.length > 0) {
          const targetKey = this.pendingRouteProject || this.activeProjectId();
          const matched = this.resolveProject(targetKey, projects) ?? projects[0];
          this.selectProject(matched.id);
        }
      },
    });
  }

  selectProject(projectId: string): void {
    this.pendingRouteProject = projectId;
    this.activeProjectId.set(projectId);
    this.loadProjectMetadata(projectId);
  }

  loadProjectMetadata(projectId: string): void {
    if (!projectId) return;
    this.api.get<ProjectMember[]>(`/projects/${projectId}/members`).subscribe({
      next: (res) => this.members.set(res ?? []),
    });
    this.api.get<Module[]>(`/projects/${projectId}/modules`).subscribe({
      next: (res) => this.modules.set(res ?? []),
    });
    this.api.get<Epic[]>(`/projects/${projectId}/epics`).subscribe({
      next: (res) => this.epics.set(res ?? []),
    });
    this.api.get<Cycle[]>(`/projects/${projectId}/cycles`).subscribe({
      next: (res) => this.cycles.set(res ?? []),
    });
    this.api.get<Label[]>(`/projects/${projectId}/labels`).subscribe({
      next: (res) => this.labels.set(res ?? []),
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
    });
  }

  upsertMember(email: string, name: string, role: ProjectRole): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    const payload = { user_id: email, email, name, role };
    this.api.post<ProjectMember>(`/projects/${projectId}/members`, payload).subscribe({
      next: (m) => {
        this.members.update((list) => {
          const filtered = list.filter((x) => x.user_id !== m.user_id && x.email !== m.email);
          return [...filtered, m];
        });
      },
    });
  }

  removeMember(userId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.api.delete(`/projects/${projectId}/members/${userId}`).subscribe({
      next: () => {
        this.members.update((list) => list.filter((m) => m.user_id !== userId));
      },
    });
  }

  createModule(name: string, description: string, leadUserId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.api
      .post<Module>(`/projects/${projectId}/modules`, {
        name,
        description,
        lead_user_id: leadUserId,
      })
      .subscribe({
        next: (mod) => this.modules.update((prev) => [...prev, mod]),
      });
  }

  updateModule(
    moduleId: string,
    patch: { name?: string; description?: string; lead_user_id?: string }
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

    this.modules.update((prev) =>
      prev.map((m) => (m.id === moduleId ? { ...m, ...optimistic } : m))
    );

    this.api.patch<Module>(`/projects/${projectId}/modules/${moduleId}`, payload).subscribe({
      next: (saved) => {
        this.modules.update((prev) => prev.map((m) => (m.id === moduleId ? saved : m)));
      },
    });
  }

  deleteModule(moduleId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.modules.update((prev) => prev.filter((m) => m.id !== moduleId));
    this.api.delete(`/projects/${projectId}/modules/${moduleId}`).subscribe();
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
    });
  }

  deleteEpic(epicId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.epics.update((prev) => prev.filter((e) => e.id !== epicId));
    this.api.delete(`/projects/${projectId}/epics/${epicId}`).subscribe();
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
    });
  }

  deleteCycle(cycleId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.cycles.update((prev) => prev.filter((c) => c.id !== cycleId));
    this.api.delete(`/projects/${projectId}/cycles/${cycleId}`).subscribe();
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
      });
  }

  deleteLabel(labelId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.labels.update((prev) => prev.filter((l) => l.id !== labelId));
    this.api.delete(`/projects/${projectId}/labels/${labelId}`).subscribe();
  }
}
