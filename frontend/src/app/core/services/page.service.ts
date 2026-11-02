import { Injectable, computed, effect, inject, signal } from '@angular/core';
import { ApiService } from './api.service';
import { ProjectService } from './project.service';
import { AuthService } from './auth.service';
import { Page, PageCollection, PageTreeNode, PageVersion } from '../models/page.model';
import { Project } from '../models/project.model';

@Injectable({
  providedIn: 'root',
})
export class PageService {
  private readonly api = inject(ApiService);
  private readonly projectService = inject(ProjectService);
  private readonly authService = inject(AuthService);

  readonly collections = signal<PageCollection[]>([]);
  readonly pages = signal<Page[]>([]);
  readonly activeCollectionId = signal<string>('col-general');
  readonly activePageId = signal<string>('page-welcome');
  readonly versionsByPage = signal<Record<string, PageVersion[]>>({});
  private readonly updateSeqByPage = new Map<string, number>();

  constructor() {
    effect(() => {
      const projects = this.projectService.projects();
      for (const proj of projects) {
        this.loadWiki(proj.id);
      }
    });
  }

  readonly projectPages = computed<Page[]>(() => {
    const projectId = this.projectService.activeProjectId();
    const colId = this.activeCollectionId();
    if (colId) {
      return this.pages().filter((p) => (p.collection_id || 'col-general') === colId);
    }
    return this.pages().filter((p) => p.project_id === projectId);
  });

  readonly activePage = computed<Page | null>(() => {
    const id = this.activePageId();
    const colPages = this.projectPages();
    return (
      colPages.find((p) => p.id === id) ??
      colPages[0] ??
      this.pages().find((p) => p.id === id) ??
      this.pages()[0] ??
      null
    );
  });

  readonly pageTree = computed<PageTreeNode[]>(() => {
    const list = this.projectPages();
    return this.buildTree(list);
  });

  getPagesTreeForCollection(collectionId: string): PageTreeNode[] {
    const list = this.pages().filter((p) => (p.collection_id || 'col-general') === collectionId);
    return this.buildTree(list);
  }

  getPagesForProject(projectId: string): Page[] {
    const col = this.getProjectCollection(projectId);
    if (col) {
      return this.pages().filter((p) => p.collection_id === col.id);
    }
    return this.pages().filter((p) => p.project_id === projectId);
  }

  getProjectCollection(projectId: string, projectName?: string): PageCollection | undefined {
    const expectedId = `col-${projectId}`;
    return (
      this.collections().find((c) => c.id === expectedId) ??
      this.collections().find(
        (c) =>
          (c.project_id === projectId && c.id !== 'col-general') ||
          (projectName && c.name.toLowerCase() === projectName.toLowerCase())
      )
    );
  }

  /**
   * Ensures a dedicated Wiki PageCollection and starter Page exist for the given project.
   */
  ensureProjectCollection(
    project: Pick<Project, 'id' | 'name' | 'key_prefix' | 'description'>
  ): PageCollection | undefined {
    const existing = this.getProjectCollection(project.id, project.name);
    if (existing) {
      return existing;
    }
    this.loadWiki(project.id);
    return undefined;
  }

  /**
   * Selects a project's corresponding Wiki collection and activates its first page.
   */
  navigateToProjectCollection(
    project: Pick<Project, 'id' | 'name' | 'key_prefix' | 'description'>
  ): PageCollection | undefined {
    const col = this.getProjectCollection(project.id, project.name);
    if (col) {
      this.selectCollection(col.id);
      return col;
    }
    const fallbackColId = `col-${project.id}`;
    this.selectCollection(fallbackColId);
    this.loadWiki(project.id);
    return undefined;
  }

  selectCollection(collectionId: string): void {
    this.activeCollectionId.set(collectionId);
    const colPages = this.pages().filter(
      (p) => (p.collection_id || 'col-general') === collectionId
    );
    if (colPages.length > 0 && !colPages.some((p) => p.id === this.activePageId())) {
      this.activePageId.set(colPages[0].id);
    }
  }

  getCollectionName(collectionId?: string): string {
    if (!collectionId) return 'General';
    return this.collections().find((c) => c.id === collectionId)?.name ?? 'General';
  }

  private buildTree(list: Page[]): PageTreeNode[] {
    const map = new Map<string, PageTreeNode>();
    for (const p of list) {
      map.set(p.id, { page: p, children: [], expanded: true });
    }
    const roots: PageTreeNode[] = [];
    for (const p of list) {
      const node = map.get(p.id)!;
      if (p.parent_page_id && map.has(p.parent_page_id)) {
        map.get(p.parent_page_id)!.children.push(node);
      } else {
        roots.push(node);
      }
    }
    return roots;
  }

  loadWiki(projectId: string): void {
    if (!projectId) return;
    this.api.get<PageCollection[]>(`/projects/${projectId}/collections`).subscribe({
      next: (cols) => {
        const incoming = cols ?? [];
        this.collections.update((existing) => [
          ...existing.filter((c) => c.project_id !== projectId),
          ...incoming,
        ]);
      },
    });
    this.api.get<Page[]>(`/projects/${projectId}/pages`).subscribe({
      next: (pgs) => {
        const incoming = pgs ?? [];
        this.pages.update((existing) => [
          ...existing.filter((p) => p.project_id !== projectId),
          ...incoming,
        ]);
        const activeColPages = this.projectPages();
        const currentPageId = this.activePageId();
        if (activeColPages.length > 0 && !activeColPages.some((p) => p.id === currentPageId)) {
          const preferred =
            activeColPages.find((p) => p.id === 'page-welcome') ?? activeColPages[0];
          this.activePageId.set(preferred.id);
        }
      },
    });
  }

  selectPage(pageId: string): void {
    this.activePageId.set(pageId);
    const found = this.pages().find((p) => p.id === pageId);
    if (found?.collection_id) {
      this.activeCollectionId.set(found.collection_id);
    }
    const projectId = found?.project_id ?? this.projectService.activeProjectId();
    if (!projectId) return;
    this.api.get<PageVersion[]>(`/projects/${projectId}/pages/${pageId}/versions`).subscribe({
      next: (vers) => {
        this.versionsByPage.update((m) => ({ ...m, [pageId]: vers ?? [] }));
      },
    });
  }

  createPage(title: string, parentPageId?: string, collectionId?: string, content?: string): void {
    const targetColId = collectionId || this.activeCollectionId() || 'col-general';
    const col = this.collections().find((c) => c.id === targetColId);
    const projectId = col?.project_id || this.projectService.activeProjectId();
    if (!projectId) return;
    const initialContent =
      content ?? `# ${title}\n\nStart writing or type \`/\` for block commands...`;

    this.api
      .post<Page>(`/projects/${projectId}/pages`, {
        title,
        content: initialContent,
        parent_page_id: parentPageId,
        collection_id: targetColId,
        order: this.projectPages().length + 1,
      })
      .subscribe({
        next: (created) => {
          this.pages.update((prev) => [...prev, created]);
          this.activeCollectionId.set(created.collection_id || targetColId);
          this.activePageId.set(created.id);
        },
      });
  }

  updatePage(
    pageId: string,
    patch: { title?: string; content?: string; parent_page_id?: string }
  ): void {
    const user = this.authService.currentUser();
    const current = this.pages().find((p) => p.id === pageId);
    if (!current || current.is_locked) return;

    const noTitleChange = patch.title === undefined || patch.title === current.title;
    const noContentChange = patch.content === undefined || patch.content === current.content;
    const noParentChange =
      patch.parent_page_id === undefined || patch.parent_page_id === current.parent_page_id;
    if (noTitleChange && noContentChange && noParentChange) {
      return;
    }

    const seq = (this.updateSeqByPage.get(pageId) ?? 0) + 1;
    this.updateSeqByPage.set(pageId, seq);

    const now = new Date().toISOString();
    const updatedLocal: Page = {
      ...current,
      ...patch,
      updated_by: user.user_id,
      updated_at: now,
    };
    this.pages.update((list) => list.map((p) => (p.id === pageId ? updatedLocal : p)));

    this.api.put<Page>(`/projects/${current.project_id}/pages/${pageId}`, patch).subscribe({
      next: (saved) => {
        if (this.updateSeqByPage.get(pageId) !== seq) {
          return;
        }
        this.pages.update((list) => list.map((p) => (p.id === pageId ? saved : p)));
        this.api
          .get<PageVersion[]>(`/projects/${current.project_id}/pages/${pageId}/versions`)
          .subscribe({
            next: (vers) => {
              if (this.updateSeqByPage.get(pageId) !== seq) {
                return;
              }
              this.versionsByPage.update((m) => ({ ...m, [pageId]: vers ?? [] }));
            },
          });
      },
    });
  }

  toggleLock(pageId: string): void {
    const current = this.pages().find((p) => p.id === pageId);
    if (!current) return;
    const nextLocked = !current.is_locked;
    this.api
      .patch<Page>(`/projects/${current.project_id}/pages/${pageId}/lock`, {
        is_locked: nextLocked,
      })
      .subscribe({
        next: (saved) => {
          this.pages.update((list) => list.map((p) => (p.id === pageId ? saved : p)));
        },
      });
  }

  deletePage(pageId: string): void {
    const current = this.pages().find((p) => p.id === pageId);
    if (!current) return;
    this.api.delete(`/projects/${current.project_id}/pages/${pageId}`).subscribe({
      next: () => {
        this.pages.update((list) => list.filter((p) => p.id !== pageId));
        if (this.activePageId() === pageId) {
          const remaining = this.projectPages();
          this.activePageId.set(remaining[0]?.id ?? '');
        }
      },
    });
  }

  restoreVersion(pageId: string, version: PageVersion): void {
    const current = this.pages().find((p) => p.id === pageId);
    if (!current || current.is_locked) return;

    this.api
      .post<Page>(
        `/projects/${current.project_id}/pages/${pageId}/versions/${version.id}/restore`,
        {}
      )
      .subscribe({
        next: (restored) => {
          this.pages.update((list) => list.map((p) => (p.id === pageId ? restored : p)));
          this.selectPage(pageId);
        },
      });
  }

  createCollection(name: string, description: string): void {
    const projectId = this.projectService.activeProjectId();
    if (!projectId) return;
    this.api
      .post<PageCollection>(`/projects/${projectId}/collections`, { name, description })
      .subscribe({
        next: (col) => this.collections.update((prev) => [...prev, col]),
      });
  }
}
