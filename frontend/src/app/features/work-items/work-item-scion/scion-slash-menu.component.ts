import { Component, computed, effect, input, output, signal } from '@angular/core';
import { CommonModule } from '@angular/common';

export interface SlashCommandOption {
  command: string;
  usage: string;
  description: string;
}

export const SCION_SLASH_COMMANDS: SlashCommandOption[] = [
  {
    command: 'implement',
    usage: '/implement [instructions]',
    description: 'Create or wake the implementation agent on feat/<key>',
  },
  {
    command: 'review',
    usage: '/review [instructions]',
    description: 'Create or wake the review agent to review the branch',
  },
  {
    command: 'fix',
    usage: '/fix [instructions]',
    description: 'Ask the agent to fix review feedback or test failures',
  },
  {
    command: 'plan',
    usage: '/plan [instructions]',
    description: 'Ask the agent to analyze and propose an implementation plan',
  },
  {
    command: 'ask',
    usage: '/ask <question>',
    description: 'Ask an agent a question without editing code',
  },
  {
    command: 'stop',
    usage: '/stop [--all]',
    description: 'Stop the bound agent (--all stops both impl & review)',
  },
];

@Component({
  selector: 'app-scion-slash-menu',
  standalone: true,
  imports: [CommonModule],
  template: `
    @if (isVisible()) {
      <div
        role="menu"
        aria-label="Scion slash commands"
        data-testid="scion-slash-menu"
        class="mb-1.5 rounded-md bg-white border border-slate-200 shadow-lg overflow-hidden divide-y divide-slate-100"
      >
        <div class="px-2.5 py-1 bg-slate-50 text-[10px] font-semibold uppercase tracking-wider text-slate-500">
          Scion Slash Commands (↑/↓ to navigate, Enter/Tab to select)
        </div>
        <div class="max-h-48 overflow-y-auto py-1">
          @for (opt of filteredOptions(); track opt.command; let idx = $index) {
            <button
              type="button"
              role="menuitem"
              (click)="select.emit('/' + opt.command + ' ')"
              (mouseenter)="activeIndex.set(idx)"
              class="w-full px-3 py-1.5 text-left flex items-center justify-between gap-3 text-xs transition-colors"
              [ngClass]="
                idx === activeIndex() ? 'bg-sky-50 text-sky-900' : 'text-slate-700 hover:bg-slate-50'
              "
            >
              <span class="font-mono font-semibold text-sky-700">{{ opt.usage }}</span>
              <span class="text-[11px] text-slate-500 truncate">{{ opt.description }}</span>
            </button>
          }
        </div>
      </div>
    }
  `,
})
export class ScionSlashMenuComponent {
  readonly enabled = input<boolean>(false);
  readonly queryText = input<string>('');
  readonly dismissed = input<boolean>(false);
  readonly select = output<string>();
  readonly dismiss = output<void>();

  readonly activeIndex = signal<number>(0);

  readonly filteredOptions = computed<SlashCommandOption[]>(() => {
    const text = this.queryText().trimStart();
    if (!text.startsWith('/')) return [];
    // Only show autocomplete while typing the first token (before first whitespace).
    if (/\s/.test(text)) return [];
    const token = text.slice(1).toLowerCase();
    return SCION_SLASH_COMMANDS.filter((o) => o.command.startsWith(token));
  });

  readonly isVisible = computed(
    () => this.enabled() && !this.dismissed() && this.filteredOptions().length > 0
  );

  constructor() {
    effect(() => {
      const len = this.filteredOptions().length;
      if (this.activeIndex() >= len) {
        this.activeIndex.set(0);
      }
    });
  }

  handleKey(event: KeyboardEvent): boolean {
    if (!this.isVisible()) return false;
    const opts = this.filteredOptions();
    if (opts.length === 0) return false;

    if (event.key === 'ArrowDown') {
      event.preventDefault();
      this.activeIndex.update((i) => (i + 1) % opts.length);
      return true;
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault();
      this.activeIndex.update((i) => (i - 1 + opts.length) % opts.length);
      return true;
    }
    if (event.key === 'Enter' && !event.metaKey && !event.ctrlKey) {
      event.preventDefault();
      const chosen = opts[this.activeIndex()] ?? opts[0];
      if (chosen) {
        this.select.emit('/' + chosen.command + ' ');
      }
      return true;
    }
    if (event.key === 'Tab') {
      event.preventDefault();
      const chosen = opts[this.activeIndex()] ?? opts[0];
      if (chosen) {
        this.select.emit('/' + chosen.command + ' ');
      }
      return true;
    }
    if (event.key === 'Escape') {
      event.preventDefault();
      this.dismiss.emit();
      return true;
    }
    return false;
  }
}
