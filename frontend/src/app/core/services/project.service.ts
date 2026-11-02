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

  deleteModule(moduleId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.api.delete(`/projects/${projectId}/modules/${moduleId}`).subscribe({
      next: () => this.modules.update((prev) => prev.filter((m) => m.id !== moduleId)),
    });
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

  deleteEpic(epicId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.api.delete(`/projects/${projectId}/epics/${epicId}`).subscribe({
      next: () => this.epics.update((prev) => prev.filter((e) => e.id !== epicId)),
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

  deleteCycle(cycleId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.api.delete(`/projects/${projectId}/cycles/${cycleId}`).subscribe({
      next: () => this.cycles.update((prev) => prev.filter((c) => c.id !== cycleId)),
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
      });
  }

  deleteLabel(labelId: string): void {
    const projectId = this.activeProjectId();
    if (!projectId) return;
    this.labels.update((prev) => prev.filter((l) => l.id !== labelId));
    this.api.delete(`/projects/${projectId}/labels/${labelId}`).subscribe();
  }
}
