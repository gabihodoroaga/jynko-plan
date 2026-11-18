import { Component, Input, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterLink } from '@angular/router';
import { ProjectService } from '../../core/services/project.service';

@Component({
  selector: 'app-project-breadcrumb',
  standalone: true,
  imports: [CommonModule, RouterLink],
  styles: [
    `
      .jynko-breadcrumb > ol {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 0.375rem;
        list-style: none;
        margin: 0;
        padding: 0;
      }

      .jynko-breadcrumb > ol > li {
        display: inline-flex;
        align-items: center;
        gap: 0.375rem;
      }

      /* Decorative separator kept out of the accessibility tree via empty alt text */
      .jynko-breadcrumb > ol > li + li::before {
        content: '/' / '';
        color: #94a3b8;
        font-weight: 400;
      }
    `,
  ],
  template: `
    @if (projectService.activeProject(); as proj) {
      <nav aria-label="Breadcrumbs" class="jynko-breadcrumb">
        <ol class="text-xs">
          <li>
            <a
              [routerLink]="['/projects', proj.id]"
              [attr.aria-label]="'Open ' + proj.name + ' overview'"
              data-testid="page-project-link"
              class="inline-flex items-center gap-1.5 font-medium text-slate-600 hover:text-sky-600 hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-600 rounded transition-colors"
            >
              <span class="px-1.5 py-0.5 rounded bg-slate-100 border border-slate-200 font-mono text-[10px] font-semibold text-slate-600">
                {{ proj.key_prefix }}
              </span>
              <span>{{ proj.name }}</span>
            </a>
          </li>
          <li>
            <a
              [routerLink]="sectionRoute"
              aria-current="page"
              class="font-semibold text-slate-900 hover:underline focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-sky-600 rounded"
            >
              {{ sectionLabel }}
            </a>
          </li>
        </ol>
      </nav>
    }
  `,
})
export class ProjectBreadcrumbComponent {
  readonly projectService = inject(ProjectService);

  @Input() sectionLabel = '';
  @Input() sectionRoute: string[] = [];
}
