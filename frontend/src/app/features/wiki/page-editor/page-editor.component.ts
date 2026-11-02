import { Component, EventEmitter, Output, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { PageService } from '../../../core/services/page.service';
import { ProjectService } from '../../../core/services/project.service';
import { MarkdownPipe } from '../../../shared/pipes/markdown.pipe';

interface SlashBlockCommand {
  cmd: string;
  label: string;
  description: string;
  snippet: string;
}

@Component({
  selector: 'app-page-editor',
  standalone: true,
  imports: [CommonModule, FormsModule, MarkdownPipe],
  template: `
    @if (pageService.activePage(); as page) {
      <div class="flex flex-col h-full bg-white">
        <!-- 1. Top Page Breadcrumb & Action Header (matches Wiki screenshot) -->
        <div class="h-11 px-5 border-b border-slate-200 flex items-center justify-between gap-4 shrink-0 bg-white">
          <!-- Left: Collection > Page Title Breadcrumb -->
          <div class="flex items-center gap-2 text-xs text-slate-600 min-w-0">
            <span class="inline-flex items-center gap-1.5 text-slate-600 font-medium shrink-0">
              <svg class="w-3.5 h-3.5 text-slate-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                <path d="M2 6a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v14a2 2 0 0 0-2-2H4a2 2 0 0 1-2-2V6Z" />
                <path d="M22 6a2 2 0 0 0-2-2h-6a2 2 0 0 0-2 2v14a2 2 0 0 1 2-2h6a2 2 0 0 0 2-2V6Z" />
              </svg>
              <span>{{ pageService.getCollectionName(page.collection_id) }}</span>
            </span>
            <span class="text-slate-300">›</span>
            <span class="inline-flex items-center gap-1.5 text-slate-800 font-medium truncate">
              <svg class="w-3.5 h-3.5 text-slate-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                <path d="M6 3h9l5 5v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z" />
                <path d="M8 11h8M8 15h5" />
              </svg>
              <span class="truncate">{{ page.title }}</span>
            </span>
            @if (page.is_locked) {
              <span class="ml-1 px-1.5 py-0.5 rounded bg-amber-50 border border-amber-200 text-amber-700 text-[10px] font-medium">
                Locked
              </span>
            }
          </div>

          <!-- Right: Author Avatar, Save, Mode Toggle, Lock, History, Delete -->
          <div class="flex items-center gap-2 shrink-0">
            <span
              class="w-5 h-5 rounded-full bg-orange-600 text-white text-[10px] font-bold flex items-center justify-center"
              title="Author"
            >
              G
            </span>

            @if (projectService.canEdit()) {
              <button
                type="button"
                aria-label="Save wiki page"
                (click)="savePage(page.id)"
                [disabled]="page.is_locked"
                class="px-2.5 py-1 rounded-md border border-slate-200 bg-white hover:bg-slate-50 text-xs font-medium text-slate-700 disabled:opacity-40"
              >
                {{ justSaved() ? 'Saved ✓' : 'Save' }}
              </button>
            }

            <!-- Reader / Markdown Editor Switcher -->
            <div class="inline-flex bg-slate-100 p-0.5 rounded-md border border-slate-200">
              <button
                type="button"
                (click)="editorMode.set('PREVIEW')"
                class="px-2 py-0.5 rounded text-[11px] font-medium transition-colors"
                [ngClass]="editorMode() === 'PREVIEW' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-600'"
              >
                Document
              </button>
              <button
                type="button"
                (click)="editorMode.set('EDIT')"
                class="px-2 py-0.5 rounded text-[11px] font-medium transition-colors"
                [ngClass]="editorMode() === 'EDIT' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-600'"
              >
                Edit Blocks
              </button>
              <button
                type="button"
                (click)="editorMode.set('SPLIT')"
                class="px-2 py-0.5 rounded text-[11px] font-medium transition-colors"
                [ngClass]="editorMode() === 'SPLIT' ? 'bg-white text-slate-900 shadow-sm' : 'text-slate-600'"
              >
                Split
              </button>
            </div>

            @if (projectService.canEdit()) {
              <!-- Lock / Unlock Button -->
              <button
                type="button"
                (click)="pageService.toggleLock(page.id)"
                class="p-1.5 rounded-md hover:bg-slate-100 text-slate-600"
                [ngClass]="page.is_locked ? 'text-amber-600 bg-amber-50' : ''"
                [title]="page.is_locked ? 'Unlock page' : 'Lock page'"
              >
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                  <rect x="5" y="11" width="14" height="10" rx="2" />
                  <path d="M8 11V7a4 4 0 0 1 8 0v4" />
                </svg>
              </button>
            }

            <!-- Version History Button -->
            <button
              type="button"
              (click)="toggleHistory.emit()"
              class="p-1.5 rounded-md hover:bg-slate-100 text-slate-600"
              title="Page Version History"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                <circle cx="12" cy="12" r="9" />
                <path d="M12 7v5l3 2" />
              </svg>
            </button>

            @if (projectService.canEdit()) {
              <!-- Delete Page Button -->
              <button
                type="button"
                (click)="pageService.deletePage(page.id)"
                class="p-1.5 rounded-md hover:bg-rose-50 text-slate-500 hover:text-rose-600"
                title="Delete page"
              >
                <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                  <path d="M4 7h16M10 11v6M14 11v6M6 7l1 13a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-13M9 7V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v3" />
                </svg>
              </button>
            }
          </div>
        </div>

        <!-- 2. Rich Formatting & Block Toolbar Row + "AI summary" button on right -->
        <div class="h-10 px-5 border-b border-slate-200 flex items-center justify-between gap-3 shrink-0 bg-white select-none">
          <!-- Formatting Controls -->
          <div class="flex items-center gap-1 text-xs text-slate-700 overflow-x-auto">
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, '\n## Section Heading\n')"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="inline-flex items-center gap-1 px-2 py-1 rounded hover:bg-slate-100 font-medium text-slate-700 disabled:opacity-40"
              title="Insert Heading"
            >
              <span>Text</span>
              <span class="text-[10px] text-slate-400">⌄</span>
            </button>

            <span class="h-4 w-px bg-slate-200 mx-1"></span>

            <button
              type="button"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="inline-flex items-center gap-1 px-2 py-1 rounded hover:bg-slate-100 font-medium text-slate-700 disabled:opacity-40"
              title="Text Color"
            >
              <span>Color</span>
              <span class="text-[10px] text-slate-400">⌄</span>
            </button>

            <span class="h-4 w-px bg-slate-200 mx-1"></span>

            <!-- B I U S -->
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, ' **bold text** ')"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="w-7 h-7 rounded hover:bg-slate-100 font-bold text-slate-700 disabled:opacity-40"
              title="Bold"
            >
              B
            </button>
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, ' *italic text* ')"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="w-7 h-7 rounded hover:bg-slate-100 italic font-serif text-slate-700 disabled:opacity-40"
              title="Italic"
            >
              I
            </button>
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, ' <u>underlined</u> ')"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="w-7 h-7 rounded hover:bg-slate-100 underline text-slate-700 disabled:opacity-40"
              title="Underline"
            >
              U
            </button>
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, ' ~~strikethrough~~ ')"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="w-7 h-7 rounded hover:bg-slate-100 line-through text-slate-700 disabled:opacity-40"
              title="Strikethrough"
            >
              S
            </button>

            <span class="h-4 w-px bg-slate-200 mx-1"></span>

            <!-- Lists & Checklist -->
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, '\n- Bullet item\n')"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="px-2 py-1 rounded hover:bg-slate-100 text-slate-600 disabled:opacity-40"
              title="Bullet List (/bullet)"
            >
              •≡
            </button>
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, '\n1. Numbered item\n')"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="px-2 py-1 rounded hover:bg-slate-100 text-slate-600 disabled:opacity-40"
              title="Numbered List"
            >
              1≡
            </button>
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, '\n- [ ] Action item\n')"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="px-2 py-1 rounded hover:bg-slate-100 text-slate-600 disabled:opacity-40"
              title="Checklist (/todo)"
            >
              ☑≡
            </button>

            <span class="h-4 w-px bg-slate-200 mx-1"></span>

            <!-- Quote, Code, Divider -->
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, '\n> Callout quote\n')"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="px-2 py-1 rounded hover:bg-slate-100 font-serif text-slate-600 disabled:opacity-40"
              title="Quote block (/quote)"
            >
              ”
            </button>
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, codeBlockSnippet)"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="px-2 py-1 rounded hover:bg-slate-100 font-mono text-[11px] text-slate-600 disabled:opacity-40"
              title="Code block (/code)"
            >
              &lt;/&gt;
            </button>
            <button
              type="button"
              (click)="insertBlockSnippet(page.id, page.content, '\n---\n')"
              [disabled]="page.is_locked || !projectService.canEdit()"
              class="px-2 py-1 rounded hover:bg-slate-100 text-slate-600 disabled:opacity-40"
              title="Horizontal divider (/divider)"
            >
              ―
            </button>
          </div>
        </div>

        <!-- 3. Main Document Canvas (100% Full-Width Document Reader / Markdown Block Editor) -->
        <div
          class="flex-1 w-full overflow-y-auto bg-white"
          [ngClass]="editorMode() === 'SPLIT' ? 'grid grid-cols-1 lg:grid-cols-2 divide-x divide-slate-200' : ''"
        >
          <!-- Markdown Source / Block Editor Column (shown in EDIT or SPLIT mode) -->
          @if (editorMode() === 'EDIT' || editorMode() === 'SPLIT') {
            <div class="w-full px-8 py-8 flex flex-col min-h-full">
              <input
                type="text"
                aria-label="Wiki page title"
                [ngModel]="page.title"
                [ngModelOptions]="{ updateOn: 'blur' }"
                (input)="onTitleInput($any($event.target).value)"
                (ngModelChange)="onTitleChange(page.id, $event)"
                (keydown.enter)="$any($event.target).blur()"
                [disabled]="page.is_locked || !projectService.canEdit()"
                placeholder="Page title..."
                class="w-full text-3xl font-bold text-slate-900 bg-transparent focus:outline-none mb-4 disabled:opacity-80"
              />

              <!-- Quick Slash Commands Bar -->
              @if (!page.is_locked && projectService.canEdit()) {
                <div class="flex flex-wrap items-center gap-1.5 pb-3 mb-4 border-b border-slate-100 text-[11px]">
                  <span class="font-mono text-slate-400 mr-1">Slash Blocks (/):</span>
                  @for (cmd of slashCommands; track cmd.cmd) {
                    <button
                      type="button"
                      (click)="insertBlockSnippet(page.id, page.content, cmd.snippet)"
                      class="px-2 py-0.5 rounded bg-slate-100 hover:bg-slate-200/80 border border-slate-200 font-mono text-slate-700"
                      [title]="cmd.description"
                    >
                      {{ cmd.cmd }}
                    </button>
                  }
                </div>
              }

              <div class="relative flex-1 flex flex-col">
                <textarea
                  aria-label="Wiki page markdown content"
                  [ngModel]="page.content"
                  [ngModelOptions]="{ updateOn: 'blur' }"
                  (ngModelChange)="onContentChange(page.id, $event)"
                  (input)="onContentInput($any($event.target).value)"
                  [disabled]="page.is_locked || !projectService.canEdit()"
                  rows="22"
                  placeholder="Write Markdown blocks here or type /h1, /h2, /todo, /code, /quote..."
                  class="auto-grow-description flex-1 w-full p-4 rounded-lg border border-slate-200 font-mono text-xs leading-relaxed text-slate-800 bg-slate-50/40 focus:bg-white focus:outline-none focus:border-slate-300 resize-y"
                ></textarea>

                @if (showSlashMenu() && !page.is_locked && projectService.canEdit()) {
                  <div class="absolute bottom-4 left-4 z-20 w-72 bg-white border border-slate-200 rounded-md shadow-lg p-1.5 space-y-1">
                    <div class="px-2 py-1 text-[10px] font-semibold uppercase text-slate-400 border-b border-slate-100">
                      Insert Markdown Block
                    </div>
                    @for (cmd of slashCommands; track cmd.cmd) {
                      <button
                        type="button"
                        (click)="applySlashCommand(page.id, page.content, cmd)"
                        class="w-full flex items-center justify-between px-2.5 py-1.5 rounded text-left text-xs hover:bg-slate-100"
                      >
                        <span class="font-medium text-slate-800">{{ cmd.label }}</span>
                        <span class="font-mono text-[10px] text-slate-400">{{ cmd.cmd }}</span>
                      </button>
                    }
                  </div>
                }
              </div>
            </div>
          }

          <!-- Full-Width Document View -->
          @if (editorMode() === 'PREVIEW' || editorMode() === 'SPLIT') {
            <div class="w-full px-8 py-8">
              <!-- Editable Page Title -->
              <input
                type="text"
                aria-label="Wiki page title"
                [ngModel]="page.title"
                [ngModelOptions]="{ updateOn: 'blur' }"
                (input)="onTitleInput($any($event.target).value)"
                (ngModelChange)="onTitleChange(page.id, $event)"
                (keydown.enter)="$any($event.target).blur()"
                [disabled]="page.is_locked || !projectService.canEdit()"
                placeholder="Page title..."
                class="w-full text-[30px] leading-tight font-bold text-slate-900 bg-transparent focus:outline-none mb-5 disabled:opacity-90"
              />

              <!-- Rendered Markdown Document Blocks (click to switch to Edit Blocks if unlocked) -->
              <div
                class="text-sm leading-relaxed text-slate-700 space-y-3"
                [innerHTML]="page.content | markdown"
                (dblclick)="!page.is_locked && projectService.canEdit() && editorMode.set('EDIT')"
                title="Double-click to edit Markdown blocks"
              ></div>
            </div>
          }
        </div>
      </div>
    }
  `,
})
export class PageEditorComponent {
  @Output() toggleHistory = new EventEmitter<void>();

  readonly pageService = inject(PageService);
  readonly projectService = inject(ProjectService);

  readonly editorMode = signal<'SPLIT' | 'EDIT' | 'PREVIEW'>('PREVIEW');
  readonly showSlashMenu = signal<boolean>(false);
  readonly justSaved = signal<boolean>(false);
  readonly codeBlockSnippet = '\n```\n// code block\n```\n';
  private draftTitle: string | null = null;
  private draftContent: string | null = null;

  readonly slashCommands: SlashBlockCommand[] = [
    { cmd: '/h1', label: 'Heading 1', description: 'Large section heading', snippet: '\n# Heading 1\n' },
    { cmd: '/h2', label: 'Heading 2', description: 'Medium section heading', snippet: '\n## Section Heading\n' },
    { cmd: '/h3', label: 'Heading 3', description: 'Sub-section heading', snippet: '\n### Sub-section\n' },
    { cmd: '/todo', label: 'Checklist Item', description: 'Actionable task checkbox', snippet: '\n- [ ] Action item\n' },
    { cmd: '/bullet', label: 'Bullet List', description: 'Unordered list item', snippet: '\n- List point\n' },
    { cmd: '/code', label: 'Code Block', description: 'Monospace fenced code snippet', snippet: '\n```\n// code block\n```\n' },
    { cmd: '/quote', label: 'Callout Quote', description: 'Highlighted quote block', snippet: '\n> Key architectural note\n' },
    { cmd: '/divider', label: 'Divider Line', description: 'Horizontal section rule', snippet: '\n---\n' },
  ];

  onTitleInput(title: string): void {
    this.draftTitle = title;
  }

  onTitleChange(pageId: string, title: string): void {
    this.draftTitle = null;
    this.pageService.updatePage(pageId, { title });
  }

  onContentInput(content: string): void {
    this.draftContent = content;
    const trimmedEnd = content.trimEnd();
    this.showSlashMenu.set(trimmedEnd.endsWith('\n/') || trimmedEnd === '/');
  }

  onContentChange(pageId: string, content: string): void {
    this.draftContent = null;
    this.showSlashMenu.set(false);
    this.pageService.updatePage(pageId, { content });
  }

  savePage(pageId: string): void {
    const patch: { title?: string; content?: string } = {};
    if (this.draftTitle !== null) {
      patch.title = this.draftTitle;
      this.draftTitle = null;
    }
    if (this.draftContent !== null) {
      patch.content = this.draftContent;
      this.draftContent = null;
    }
    if (patch.title !== undefined || patch.content !== undefined) {
      this.pageService.updatePage(pageId, patch);
    }
    this.justSaved.set(true);
    setTimeout(() => this.justSaved.set(false), 1500);
  }

  insertBlockSnippet(pageId: string, currentContent: string, snippet: string): void {
    const next = (currentContent || '') + snippet;
    this.showSlashMenu.set(false);
    this.pageService.updatePage(pageId, { content: next });
  }

  applySlashCommand(pageId: string, currentContent: string, cmd: SlashBlockCommand): void {
    const cleaned = currentContent.replace(/\/(\s*)$/, '');
    this.showSlashMenu.set(false);
    this.pageService.updatePage(pageId, { content: cleaned + cmd.snippet.trimStart() });
  }
}
