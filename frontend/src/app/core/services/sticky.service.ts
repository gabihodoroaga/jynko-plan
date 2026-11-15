import { Injectable, computed, effect, inject, signal } from '@angular/core';
import { ApiService } from './api.service';
import { ProjectService } from './project.service';
import { AuthService } from './auth.service';
import {
  STICKY_PALETTES,
  Sticky,
  StickyColor,
  StickyPaletteMeta,
} from '../models/project.model';

const RANDOM_STICKY_THOUGHTS: string[] = [
  '🚀 Benchmark Firestore atomic sequence transactions under concurrent load',
  '🎨 Keep all surfaces strictly in light-mode slate-50 palette',
  '🧠 Test Gemini 2.5 Flash context ranking with 50+ work items and wiki pages',
  '🔐 Double-check X-Goog-Authenticated-User-Email fallback in local dev mode',
  '📌 Convert sprint retro action items directly into HODO work items',
  '✨ Auto-create project Wiki collection whenever a new project is initialized',
];

@Injectable({
  providedIn: 'root',
})
export class StickyService {
  private readonly api = inject(ApiService);
  private readonly projectService = inject(ProjectService);
  private readonly authService = inject(AuthService);

  readonly palettes = STICKY_PALETTES;

  readonly stickies = signal<Sticky[]>([]);
  readonly searchQuery = signal<string>('');
  readonly selectedColor = signal<StickyColor | 'ALL'>('ALL');
  private readonly loadedProjects = new Set<string>();
  private readonly projectIdsKey = computed(() =>
    this.projectService
      .projects()
      .map((p) => p.id)
      .join(',')
  );

  constructor() {
    this.authService.registerSessionLifecycle({
      onReset: () => this.resetState(),
    });

    effect(
      () => {
        const key = this.projectIdsKey();
        const ids = key ? key.split(',').filter(Boolean) : [];
        const currentSet = new Set(ids);

        let pruned = false;
        for (const existing of Array.from(this.loadedProjects)) {
          if (!currentSet.has(existing)) {
            this.loadedProjects.delete(existing);
            pruned = true;
          }
        }
        if (pruned) {
          this.stickies.update((prev) => prev.filter((s) => currentSet.has(s.project_id)));
        }
        for (const id of ids) {
          if (!this.loadedProjects.has(id)) {
            this.loadedProjects.add(id);
            this.loadStickies(id);
          }
        }
      },
      { allowSignalWrites: true }
    );
  }

  resetState(): void {
    this.loadedProjects.clear();
    this.stickies.set([]);
    this.searchQuery.set('');
    this.selectedColor.set('ALL');
  }

  readonly filteredStickies = computed<Sticky[]>(() => {
    const q = this.searchQuery().trim().toLowerCase();
    const color = this.selectedColor();

    return [...this.stickies()]
      .filter((s) => {
        if (color !== 'ALL' && s.color !== color) return false;
        if (q && !s.content.toLowerCase().includes(q)) return false;
        return true;
      })
      .sort((a, b) => {
        if (a.is_pinned !== b.is_pinned) return a.is_pinned ? -1 : 1;
        return b.updated_at.localeCompare(a.updated_at);
      });
  });

  getPalette(color: StickyColor): StickyPaletteMeta {
    return this.palettes.find((p) => p.color === color) ?? this.palettes[0];
  }

  pickRandomColor(): StickyColor {
    const idx = Math.floor(Math.random() * this.palettes.length);
    return this.palettes[idx].color;
  }

  loadStickies(projectId: string): void {
    if (!projectId) return;
    this.api.get<Sticky[]>(`/projects/${projectId}/stickies`).subscribe({
      next: (list) => {
        const incoming = list ?? [];
        this.stickies.update((prev) => [
          ...prev.filter((s) => s.project_id !== projectId),
          ...incoming,
        ]);
      },
    });
  }

  createSticky(content: string, color?: StickyColor, isPinned = false): void {
    const projectId = this.projectService.activeProjectId();
    if (!projectId) return;
    const chosenColor = color ?? this.pickRandomColor();
    const cleanContent = content.trim() || 'New sticky note — click to edit...';

    this.api
      .post<Sticky>(`/projects/${projectId}/stickies`, {
        content: cleanContent,
        color: chosenColor,
        is_pinned: isPinned,
      })
      .subscribe({
        next: (saved) => {
          this.stickies.update((prev) => [saved, ...prev]);
        },
      });
  }

  addRandomSticky(): void {
    const thought =
      RANDOM_STICKY_THOUGHTS[Math.floor(Math.random() * RANDOM_STICKY_THOUGHTS.length)];
    this.createSticky(thought, this.pickRandomColor(), false);
  }

  updateSticky(
    stickyId: string,
    patch: Partial<Pick<Sticky, 'content' | 'color' | 'is_pinned'>>
  ): void {
    const current = this.stickies().find((s) => s.id === stickyId);
    if (!current) return;

    const now = new Date().toISOString();
    const updated: Sticky = {
      ...current,
      ...patch,
      updated_at: now,
    };

    this.stickies.update((prev) => prev.map((s) => (s.id === stickyId ? updated : s)));

    this.api
      .patch<Sticky>(`/projects/${current.project_id}/stickies/${stickyId}`, patch)
      .subscribe({
        next: (saved) => {
          this.stickies.update((prev) => prev.map((s) => (s.id === stickyId ? saved : s)));
        },
        error: () => {
          this.stickies.update((prev) => prev.map((s) => (s.id === stickyId ? current : s)));
        },
      });
  }

  togglePin(stickyId: string): void {
    const current = this.stickies().find((s) => s.id === stickyId);
    if (!current) return;
    this.updateSticky(stickyId, { is_pinned: !current.is_pinned });
  }

  deleteSticky(stickyId: string): void {
    const current = this.stickies().find((s) => s.id === stickyId);
    if (!current) return;
    this.api.delete(`/projects/${current.project_id}/stickies/${stickyId}`).subscribe({
      next: () => {
        this.stickies.update((prev) => prev.filter((s) => s.id !== stickyId));
      },
    });
  }
}
