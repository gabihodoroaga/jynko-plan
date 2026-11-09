import { Component, computed, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink } from '@angular/router';
import { ProjectService } from '../../core/services/project.service';
import { WorkItemService } from '../../core/services/work-item.service';
import { PageService } from '../../core/services/page.service';
import { AuthService } from '../../core/services/auth.service';
import { StatusBadgeComponent } from '../../shared/components/status-badge.component';
import { PriorityBadgeComponent } from '../../shared/components/priority-badge.component';
import { Project } from '../../core/models/project.model';

interface ProjectOverviewCard {
  project: Project;
  total: number;
  todoCount: number;
  inProgressCount: number;
  inReviewCount: number;
  doneCount: number;
  progressPct: number;
}

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [CommonModule, RouterLink, StatusBadgeComponent, PriorityBadgeComponent],
  template: `
    <div class="w-full p-4 sm:p-6 space-y-6 bg-slate-50 min-h-full">
      <!-- Header (no "+ New work item" button) -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-lg font-semibold text-slate-900">
            Good morning, {{ authService.currentUser().name }}
          </h1>
          <p class="text-xs text-slate-500 mt-0.5">
            Overview across all <span class="font-mono font-medium text-slate-700">{{ projectService.projects().length }}</span> projects you are a member of
          </p>
        </div>
      </div>

      <!-- Aggregate Cross-Project Stat Cards -->
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3.5">
        <div class="bg-white border border-slate-200 rounded-lg p-4 shadow-card">
          <div class="text-[11px] font-medium text-slate-500">Active Projects</div>
          <div class="text-2xl font-semibold text-slate-900 mt-1 font-mono">
            {{ projectService.projects().length }}
          </div>
        </div>
        <div class="bg-white border border-slate-200 rounded-lg p-4 shadow-card">
          <div class="text-[11px] font-medium text-slate-500">Total Work Items</div>
          <div class="text-2xl font-semibold text-slate-900 mt-1 font-mono">
            {{ workItemService.items().length }}
          </div>
        </div>
        <div class="bg-white border border-slate-200 rounded-lg p-4 shadow-card">
          <div class="text-[11px] font-medium text-slate-500">In Progress (All Projects)</div>
          <div class="text-2xl font-semibold text-amber-600 mt-1 font-mono">
            {{ totalInProgressCount() }}
          </div>
        </div>
        <div class="bg-white border border-slate-200 rounded-lg p-4 shadow-card">
          <div class="text-[11px] font-medium text-slate-500">Assigned to Me</div>
          <div class="text-2xl font-semibold text-sky-600 mt-1 font-mono">
            {{ workItemService.myAssignedItems().length }}
          </div>
        </div>
      </div>

      <!-- All User Projects Overview Grid -->
      <div class="space-y-3">
        <div class="flex items-center justify-between">
          <h2 class="text-xs font-semibold uppercase tracking-wider text-slate-500">
            Your Projects ({{ projectCards().length }})
          </h2>
          <a [routerLink]="['/projects', projectService.activeProjectId()]" class="text-xs font-medium text-sky-600 hover:underline">
            Manage Projects →
          </a>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          @for (card of projectCards(); track card.project.id) {
            <div class="bg-white border border-slate-200 rounded-lg p-4 shadow-card flex flex-col justify-between gap-4 hover:border-slate-300 transition-colors">
              <div class="space-y-2.5">
                <div class="flex items-start justify-between gap-3">
                  <div class="flex items-center gap-2.5">
                    <span class="w-8 h-8 rounded-lg bg-slate-100 border border-slate-200 flex items-center justify-center text-sm">
                      👇
                    </span>
                    <div>
                      <div class="flex items-center gap-2">
                        <span class="text-sm font-semibold text-slate-900">{{ card.project.name }}</span>
                        <span class="px-1.5 py-0.5 rounded bg-slate-100 border border-slate-200 font-mono text-[10px] font-semibold text-slate-600">
                          {{ card.project.key_prefix }}
                        </span>
                      </div>
                      <div class="text-[11px] text-slate-400 font-mono">
                        {{ card.project.member_ids.length }} members · {{ card.total }} work items
                      </div>
                    </div>
                  </div>

                  <span class="px-2 py-0.5 rounded-full bg-emerald-50 border border-emerald-200 text-emerald-700 font-mono text-[11px] font-medium">
                    {{ card.progressPct }}% Done
                  </span>
                </div>

                <p class="text-xs text-slate-600 line-clamp-2">
                  {{ card.project.description }}
                </p>

                <!-- Progress Bar -->
                <div class="w-full h-1.5 bg-slate-100 rounded-full overflow-hidden">
                  <div
                    class="h-full bg-emerald-500 rounded-full transition-all"
                    [style.width.%]="card.progressPct"
                  ></div>
                </div>

                <!-- Status Breakdown Pills -->
                <div class="flex flex-wrap items-center gap-2 pt-1 text-[11px]">
                  <span class="px-2 py-0.5 rounded bg-slate-100 text-slate-700 font-mono">
                    Open: {{ card.todoCount }}
                  </span>
                  <span class="px-2 py-0.5 rounded bg-amber-50 text-amber-700 border border-amber-200/70 font-mono">
                    In Progress: {{ card.inProgressCount }}
                  </span>
                  <span class="px-2 py-0.5 rounded bg-sky-50 text-sky-700 border border-sky-200/70 font-mono">
                    In Review: {{ card.inReviewCount }}
                  </span>
                  <span class="px-2 py-0.5 rounded bg-emerald-50 text-emerald-700 border border-emerald-200/70 font-mono">
                    Done: {{ card.doneCount }}
                  </span>
                </div>
              </div>

              <!-- Quick Navigation Footer -->
              <div class="pt-3 border-t border-slate-100 flex items-center justify-between text-xs">
                <button
                  type="button"
                  (click)="openProjectWorkItems(card.project.id)"
                  class="font-medium text-sky-600 hover:text-sky-700 hover:underline"
                >
                  Open Work Items →
                </button>
                <div class="flex items-center gap-3">
                  <button
                    type="button"
                    (click)="openProjectPages(card.project)"
                    class="text-slate-500 hover:text-sky-600"
                  >
                    Project Pages
                  </button>
                  <button
                    type="button"
                    (click)="openProjectOverview(card.project.id)"
                    class="text-slate-500 hover:text-slate-800"
                  >
                    Project Overview
                  </button>
                </div>
              </div>
            </div>
          }
        </div>
      </div>

      <!-- Recent Activity Across All Projects -->
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-5">
        <div class="lg:col-span-2 bg-white border border-slate-200 rounded-lg shadow-card overflow-hidden">
          <div class="px-4 py-3 border-b border-slate-200 flex items-center justify-between bg-slate-50/60">
            <h2 class="text-xs font-semibold text-slate-900">Recent Work Items Across All Projects</h2>
            <a routerLink="/my-work" class="text-[11px] font-medium text-sky-600 hover:underline">
              My Work →
            </a>
          </div>
          <div class="divide-y divide-slate-200">
            @for (item of workItemService.items().slice(0, 6); track item.id) {
              <div class="px-4 py-2.5 flex flex-wrap sm:flex-nowrap items-center justify-between gap-2 sm:gap-3 hover:bg-slate-50">
                <div class="flex items-center gap-2 min-w-0">
                  <span class="px-1.5 py-0.5 rounded bg-slate-100 border border-slate-200 font-mono text-[10px] font-medium text-slate-600 shrink-0">
                    {{ getProjectName(item.project_id) }}
                  </span>
                  <span class="font-mono text-[11px] text-slate-400 shrink-0">{{ item.key }}</span>
                  <span class="text-xs font-medium text-slate-900 truncate">{{ item.title }}</span>
                </div>
                <div class="flex items-center gap-2 shrink-0">
                  <app-priority-badge [priority]="item.priority"></app-priority-badge>
                  <app-status-badge [status]="item.status"></app-status-badge>
                </div>
              </div>
            }
          </div>
        </div>

        <div class="bg-white border border-slate-200 rounded-lg shadow-card overflow-hidden">
          <div class="px-4 py-3 border-b border-slate-200 flex items-center justify-between bg-slate-50/60">
            <h2 class="text-xs font-semibold text-slate-900">Knowledge Base Pages</h2>
            <a routerLink="/wiki" class="text-[11px] font-medium text-sky-600 hover:underline">
              Open Wiki →
            </a>
          </div>
          <div class="divide-y divide-slate-200">
            @for (page of pageService.pages(); track page.id) {
              <a
                routerLink="/wiki"
                (click)="pageService.selectPage(page.id)"
                class="block px-4 py-3 hover:bg-slate-50 transition-colors"
              >
                <div class="text-xs font-medium text-slate-900">{{ page.title }}</div>
                <div class="text-[11px] text-slate-400 font-mono mt-0.5">
                  {{ pageService.getCollectionName(page.collection_id) }} · Updated {{ page.updated_at | date: 'MMM d' }}
                </div>
              </a>
            }
          </div>
        </div>
      </div>
    </div>
  `,
})
export class HomeComponent {
  readonly projectService = inject(ProjectService);
  readonly workItemService = inject(WorkItemService);
  readonly pageService = inject(PageService);
  readonly authService = inject(AuthService);
  private readonly router = inject(Router);

  readonly totalInProgressCount = computed(
    () => this.workItemService.items().filter((i) => i.status === 'IN_PROGRESS').length
  );

  readonly projectCards = computed<ProjectOverviewCard[]>(() => {
    const allItems = this.workItemService.items();
    return this.projectService.projects().map((project) => {
      const items = allItems.filter((i) => i.project_id === project.id);
      const total = items.length;
      const doneCount = items.filter((i) => i.status === 'DONE').length;
      const inProgressCount = items.filter((i) => i.status === 'IN_PROGRESS').length;
      const inReviewCount = items.filter((i) => i.status === 'IN_REVIEW').length;
      const todoCount = items.filter((i) => i.status === 'TODO' || i.status === 'BACKLOG').length;
      const progressPct = total > 0 ? Math.round((doneCount / total) * 100) : 0;

      return {
        project,
        total,
        todoCount,
        inProgressCount,
        inReviewCount,
        doneCount,
        progressPct,
      };
    });
  });

  getProjectName(projectId: string): string {
    return this.projectService.getProjectById(projectId)?.name ?? projectId;
  }

  openProjectWorkItems(projectId: string): void {
    this.projectService.selectProject(projectId);
    this.workItemService.resetFilters();
    this.router.navigate(['/projects', projectId, 'work-items']);
  }

  openProjectOverview(projectId: string): void {
    this.projectService.selectProject(projectId);
    this.router.navigate(['/projects', projectId]);
  }

  openProjectPages(project: Project): void {
    this.projectService.selectProject(project.id);
    this.pageService.navigateToProjectCollection(project);
    this.router.navigate(['/wiki']);
  }
}
