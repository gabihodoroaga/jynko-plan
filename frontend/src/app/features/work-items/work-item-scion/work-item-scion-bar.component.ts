import { Component, computed, effect, inject, input, output } from '@angular/core';
import { CommonModule } from '@angular/common';
import { WorkItemService } from '../../../core/services/work-item.service';
import { ScionAgent, WorkItem } from '../../../core/models/project.model';

@Component({
  selector: 'app-work-item-scion-bar',
  standalone: true,
  imports: [CommonModule],
  template: `
    @if (scionEnabled() || item().impl_agent_id || item().review_agent_id) {
      <div
        class="rounded-md bg-slate-50 border border-slate-200 p-2.5 space-y-2"
        data-testid="scion-agent-bar"
      >
        <div class="flex flex-wrap items-center justify-between gap-2">
          <div class="flex items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wider text-slate-500">
            <span class="w-1.5 h-1.5 rounded-full bg-sky-500"></span>
            Scion AI Agents
          </div>
          <button
            type="button"
            aria-label="Refresh Scion agent status"
            (click)="refresh()"
            class="px-2 py-0.5 rounded border border-slate-200 bg-white hover:bg-slate-100 text-[11px] text-slate-600 transition-colors"
          >
            Refresh
          </button>
        </div>

        <div class="flex flex-wrap items-center gap-2">
          <!-- Impl Agent Pill -->
          <div class="inline-flex items-center gap-1.5 px-2 py-1 rounded-md bg-white border border-slate-200 text-[11px]">
            <span class="font-medium text-slate-500">Impl:</span>
            @if (item().impl_agent_id) {
              <span
                class="w-1.5 h-1.5 rounded-full"
                [ngClass]="phaseDotClass(liveStatus()?.impl?.phase)"
              ></span>
              <span class="font-mono font-medium text-slate-800">
                {{ item().impl_agent_name || item().impl_agent_id }}
              </span>
              @if (item().impl_agent_harness) {
                <span class="px-1.5 py-0.25 rounded-full bg-slate-100 text-slate-600 text-[10px] font-mono">
                  {{ item().impl_agent_harness }}
                </span>
              }
              @if (liveStatus()?.impl?.phase; as phase) {
                <span class="text-[10px] font-mono text-slate-500">({{ phase }})</span>
              }
              @if (agentUrl(liveStatus()?.impl, item().impl_agent_id); as url) {
                <a
                  [href]="url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-sky-600 hover:underline font-medium"
                >
                  Open
                </a>
              }
            } @else {
              <span class="text-slate-400">None</span>
            }
          </div>

          <!-- Review Agent Pill -->
          <div class="inline-flex items-center gap-1.5 px-2 py-1 rounded-md bg-white border border-slate-200 text-[11px]">
            <span class="font-medium text-slate-500">Review:</span>
            @if (item().review_agent_id) {
              <span
                class="w-1.5 h-1.5 rounded-full"
                [ngClass]="phaseDotClass(liveStatus()?.review?.phase)"
              ></span>
              <span class="font-mono font-medium text-slate-800">
                {{ item().review_agent_name || item().review_agent_id }}
              </span>
              @if (item().review_agent_harness) {
                <span class="px-1.5 py-0.25 rounded-full bg-slate-100 text-slate-600 text-[10px] font-mono">
                  {{ item().review_agent_harness }}
                </span>
              }
              @if (liveStatus()?.review?.phase; as phase) {
                <span class="text-[10px] font-mono text-slate-500">({{ phase }})</span>
              }
              @if (agentUrl(liveStatus()?.review, item().review_agent_id); as url) {
                <a
                  [href]="url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-sky-600 hover:underline font-medium"
                >
                  Open
                </a>
              }
            } @else {
              <span class="text-slate-400">None</span>
            }
          </div>
        </div>

        @if (canEdit() && scionEnabled()) {
          <div class="flex flex-wrap items-center gap-1.5 pt-0.5">
            <span class="text-[11px] text-slate-400 mr-0.5">Quick commands:</span>
            @for (cmd of quickCommands; track cmd) {
              <button
                type="button"
                (click)="commandPrefill.emit('/' + cmd + ' ')"
                [attr.aria-label]="'Prefill /' + cmd + ' command'"
                class="px-2 py-0.5 rounded-full bg-white border border-slate-200 hover:border-sky-300 hover:text-sky-700 text-[11px] font-mono text-slate-600 transition-colors"
              >
                /{{ cmd }}
              </button>
            }
          </div>
        }
      </div>
    }
  `,
})
export class WorkItemScionBarComponent {
  private readonly workItemService = inject(WorkItemService);

  readonly item = input.required<WorkItem>();
  readonly canEdit = input<boolean>(true);
  readonly scionEnabled = input<boolean>(false);
  readonly commandPrefill = output<string>();

  readonly quickCommands = ['implement', 'review', 'fix', 'plan', 'ask', 'stop'];

  readonly liveStatus = computed(() => {
    return this.workItemService.itemAgentsByItem()[this.item().id] ?? null;
  });

  constructor() {
    effect(() => {
      const it = this.item();
      if (this.scionEnabled() || it.impl_agent_id || it.review_agent_id) {
        this.workItemService.loadItemAgents(it.id);
      }
    });
  }

  refresh(): void {
    this.workItemService.loadItemAgents(this.item().id);
  }

  phaseDotClass(phase?: string): string {
    const p = (phase || '').toLowerCase();
    if (p === 'running') return 'bg-emerald-500';
    if (p === 'error' || p === 'halted') return 'bg-rose-500';
    if (p === 'suspended' || p === 'stopped' || p === 'created') return 'bg-amber-500';
    return 'bg-slate-400';
  }

  agentUrl(agent: ScionAgent | null | undefined, agentId?: string): string {
    if (agent?.url) return agent.url;
    const hub = this.liveStatus()?.hub_url;
    if (hub && agentId) {
      return `${hub.replace(/\/+$/, '')}/agents/${agentId}`;
    }
    return '';
  }
}
