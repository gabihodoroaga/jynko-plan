import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';
import { WorkItemPriority } from '../../core/models/project.model';

@Component({
  selector: 'app-priority-badge',
  standalone: true,
  imports: [CommonModule],
  template: `
    <span
      class="inline-flex items-center px-2 py-0.5 rounded-md border text-[11px] font-medium leading-4"
      [ngClass]="badgeClasses"
    >
      <span>{{ label }}</span>
    </span>
  `,
})
export class PriorityBadgeComponent {
  @Input({ required: true }) priority: WorkItemPriority = 'MEDIUM';

  get label(): string {
    switch (this.priority) {
      case 'URGENT':
        return 'Urgent';
      case 'HIGH':
        return 'High';
      case 'MEDIUM':
        return 'Medium';
      case 'LOW':
        return 'Low';
      case 'NONE':
        return 'None';
      default:
        return this.priority;
    }
  }

  get badgeClasses(): string {
    switch (this.priority) {
      case 'URGENT':
        return 'bg-rose-50 text-rose-700 border-rose-200';
      case 'HIGH':
        return 'bg-amber-50 text-amber-700 border-amber-200';
      case 'MEDIUM':
        return 'bg-sky-50 text-sky-700 border-sky-200';
      case 'LOW':
        return 'bg-slate-100 text-slate-600 border-slate-200';
      case 'NONE':
      default:
        return 'bg-slate-50 text-slate-400 border-slate-200';
    }
  }
}
