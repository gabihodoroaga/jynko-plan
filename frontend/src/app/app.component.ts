import { Component, OnInit, computed, inject, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs';
import { AuthService } from './core/services/auth.service';
import { ProjectService } from './core/services/project.service';
import { WorkItemService } from './core/services/work-item.service';
import { PageService } from './core/services/page.service';
import { SettingsService } from './core/services/settings.service';
import { ModalComponent } from './shared/components/modal.component';
import { DropdownComponent, DropdownOption } from './shared/components/dropdown.component';
import {
  ALL_PRIORITIES,
  ALL_STATUSES,
  Project,
  WorkItemPriority,
  WorkItemStatus,
} from './core/models/project.model';

@Component({
  selector: 'app-root',
  standalone: true,
  imports: [
    CommonModule,
    FormsModule,
    RouterOutlet,
    RouterLink,
    RouterLinkActive,
    ModalComponent,
    DropdownComponent,
  ],
  template: `
    <div class="flex flex-col h-screen w-screen overflow-hidden bg-[#f4f5f7] text-slate-900">
      <!-- Global Top Application Bar (Jynko App Icon on left, Search in center, User on right) -->
      <header class="h-11 px-3.5 bg-[#f4f5f7] flex items-center justify-between gap-4 shrink-0 select-none">
        <!-- Left: Simple Application Brand Icon (no dropdown caret) -->
        <div class="flex items-center gap-2 min-w-[180px]">
          <a
            routerLink="/home"
            (click)="activeRailSection.set('WORK')"
            class="inline-flex items-center gap-2 px-1.5 py-1 rounded-md hover:bg-slate-200/60 transition-colors"
            title="Jynko Home"
          >
            <!-- Custom Jynko Geometric Prism / Spark Application Icon -->
            <span class="w-6 h-6 rounded-lg bg-sky-600 text-white flex items-center justify-center shadow-sm">
              <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none">
                <path
                  d="M12 2.5 20.5 7.5v9L12 21.5 3.5 16.5v-9L12 2.5Z"
                  stroke="currentColor"
                  stroke-width="2.2"
                  stroke-linejoin="round"
                />
                <path
                  d="M12 7v10M7.5 9.5 12 12l4.5-2.5"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </span>
            <span class="text-[13px] font-bold tracking-tight text-slate-900">Jynko</span>
          </a>
        </div>

        <!-- Center: Global Search Input -->
        <div class="flex-1 max-w-md">
          <div class="relative flex items-center">
            <svg
              class="w-3.5 h-3.5 text-slate-400 absolute left-3 pointer-events-none"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
            >
              <circle cx="11" cy="11" r="7" />
              <path d="m20 20-3.5-3.5" />
            </svg>
            <input
              type="search"
              [ngModel]="workItemService.filter().search"
              (ngModelChange)="workItemService.updateFilter({ search: $event })"
              placeholder="Search"
              class="w-full h-7 pl-8 pr-3 rounded-md bg-white border border-slate-200/90 text-xs text-slate-800 placeholder-slate-400 focus:outline-none focus:border-slate-300 shadow-[0_1px_1px_rgba(0,0,0,0.02)]"
            />
          </div>
        </div>

        <!-- Right: User Avatar & Account Popover Card -->
        <div class="relative flex items-center gap-2.5">
          <button
            type="button"
            (click)="toggleUserMenu($event)"
            aria-label="Open user menu"
            [title]="authService.currentUser().name"
            class="w-7 h-7 rounded-full text-white text-xs font-bold flex items-center justify-center overflow-hidden ring-2 ring-transparent hover:ring-slate-300 focus:outline-none focus:ring-sky-500 transition-all shadow-sm"
            [style.backgroundColor]="authService.avatarColor()"
          >
            @if (authService.currentUser().avatar_url) {
              <img
                [src]="authService.currentUser().avatar_url"
                [alt]="authService.currentUser().name"
                class="w-full h-full object-cover"
              />
            } @else {
              <span>{{ authService.userInitial() }}</span>
            }
          </button>

          @if (isUserMenuOpen()) {
            <!-- Backdrop to dismiss on outside click -->
            <div
              class="fixed inset-0 z-40"
              (click)="isUserMenuOpen.set(false)"
            ></div>

            <!-- Google-Style Account Card Dropdown -->
            <div
              data-testid="user-dropdown-card"
              (click)="$event.stopPropagation()"
              class="absolute right-0 top-9 z-50 w-80 rounded-2xl bg-white border border-slate-200 shadow-[0_12px_32px_rgba(15,23,42,0.14)] overflow-hidden"
            >
              <!-- Upper Account Identity & Profile Action -->
              <div class="p-5 flex items-start gap-4">
                <div
                  class="w-16 h-16 rounded-full text-white text-2xl font-medium flex items-center justify-center shrink-0 overflow-hidden shadow-sm"
                  [style.backgroundColor]="authService.avatarColor()"
                >
                  @if (authService.currentUser().avatar_url) {
                    <img
                      [src]="authService.currentUser().avatar_url"
                      [alt]="authService.currentUser().name"
                      class="w-full h-full object-cover"
                    />
                  } @else {
                    <span>{{ authService.userInitial() }}</span>
                  }
                </div>

                <div class="min-w-0 flex-1">
                  <div class="text-sm font-semibold text-slate-900 truncate">
                    {{ authService.currentUser().name }}
                  </div>
                  <div class="text-xs text-slate-500 truncate mt-0.5">
                    {{ authService.currentUser().email }}
                  </div>

                  <button
                    type="button"
                    (click)="openProfilePage()"
                    aria-label="Open user profile"
                    class="mt-3 inline-flex items-center px-4 py-1.5 rounded-full bg-[#1a73e8] hover:bg-[#1557b0] text-white text-xs font-medium transition-colors shadow-sm"
                  >
                    Profile
                  </button>
                </div>
              </div>

              <!-- Bottom Action Bar: Sign out -->
              <div class="px-5 py-3 border-t border-slate-200 bg-slate-50/50 flex items-center justify-end">
                <button
                  type="button"
                  (click)="signOut()"
                  aria-label="Sign out"
                  class="px-4 py-1.5 rounded-lg border border-slate-300 bg-white hover:bg-slate-100 text-xs font-medium text-slate-700 transition-colors"
                >
                  Sign out
                </button>
              </div>
            </div>
          }
        </div>
      </header>

      <!-- Main 3-Column Body: [1. Left Rail] + [2. Contextual Secondary Panel (Work | Wiki)] + [3. Right Workspace] -->
      <div class="flex flex-1 min-h-0 overflow-hidden">
        <!-- 1. Left Icon Navigation Rail (w-[68px]) with ONLY 3 buttons: Work, Wiki, Settings -->
        <nav
          aria-label="Primary rail"
          class="w-[68px] bg-[#f4f5f7] flex flex-col items-center pt-2 pb-4 shrink-0 select-none"
        >
          <div class="flex flex-col items-center gap-3.5 w-full">
            <!-- Button 1: Work -->
            <a
              [routerLink]="['/projects', projectService.activeProjectId(), 'work-items']"
              (click)="activeRailSection.set('WORK'); sidebarCollapsed.set(false)"
              class="group flex flex-col items-center gap-1 text-center cursor-pointer"
            >
              <div
                class="w-10 h-10 rounded-xl flex items-center justify-center transition-colors"
                [ngClass]="
                  activeRailSection() === 'WORK'
                    ? 'bg-[#e2e4e9] text-slate-900'
                    : 'text-slate-700 group-hover:bg-slate-200/60'
                "
              >
                <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none">
                  <rect x="4" y="4" width="11" height="11" rx="2.5" fill="currentColor" />
                  <rect
                    x="9"
                    y="9"
                    width="11"
                    height="11"
                    rx="2.5"
                    fill="#f4f5f7"
                    stroke="currentColor"
                    stroke-width="2"
                  />
                </svg>
              </div>
              <span class="text-[11px] font-medium text-slate-800 leading-tight">Work</span>
            </a>

            <!-- Button 2: Wiki -->
            <a
              routerLink="/wiki"
              (click)="activeRailSection.set('WIKI'); sidebarCollapsed.set(false)"
              class="group flex flex-col items-center gap-1 text-center cursor-pointer"
            >
              <div
                class="w-10 h-10 rounded-xl flex items-center justify-center transition-colors"
                [ngClass]="
                  activeRailSection() === 'WIKI'
                    ? 'bg-[#e2e4e9] text-slate-900'
                    : 'text-slate-600 group-hover:bg-slate-200/60'
                "
              >
                <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none">
                  <path
                    d="M10.586 3.586a2 2 0 0 1 2.828 0l7 7a2 2 0 0 1 0 2.828l-7 7a2 2 0 0 1-2.828 0l-7-7a2 2 0 0 1 0-2.828l7-7Z"
                    fill="currentColor"
                  />
                  <circle cx="12" cy="12" r="3.2" fill="#f4f5f7" />
                </svg>
              </div>
              <span class="text-[11px] font-medium text-slate-700 leading-tight">Wiki</span>
            </a>

            <!-- Subtle Horizontal Divider -->
            <div class="w-7 border-b border-slate-300/80 my-0.5"></div>

            <!-- Button 3: Settings -->
            <a
              routerLink="/settings"
              (click)="activeRailSection.set('SETTINGS'); sidebarCollapsed.set(false); settingsService.openGeneralSettings()"
              aria-label="Open Settings"
              class="group flex flex-col items-center gap-1 text-center cursor-pointer"
            >
              <div
                class="w-10 h-10 rounded-xl flex items-center justify-center transition-colors"
                [ngClass]="
                  activeRailSection() === 'SETTINGS'
                    ? 'bg-[#e2e4e9] text-slate-900'
                    : 'text-slate-600 group-hover:bg-slate-200/60'
                "
              >
                <svg
                  class="w-5 h-5"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.8"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                >
                  <circle cx="12" cy="12" r="3" />
                  <path
                    d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z"
                  />
                </svg>
              </div>
              <span class="text-[11px] font-medium text-slate-600 leading-tight">Settings</span>
            </a>
          </div>
        </nav>

        <!-- 2. Contextual Secondary Panel (w-64 / 256px): Switches between WORK panel and WIKI panel -->
        @if (!sidebarCollapsed()) {
          <aside
            class="w-64 bg-white border-r border-t border-slate-200/90 rounded-tl-xl flex flex-col justify-between shrink-0 select-none overflow-hidden"
          >
            @if (activeRailSection() === 'WIKI') {
              <!-- ==================== SECTION 2: WIKI SECONDARY PANEL ==================== -->
              <div class="flex-1 overflow-y-auto px-3 pt-3.5 pb-4 space-y-3.5">
                <!-- Panel Header: "Wiki" + collapse icon -->
                <div class="flex items-center justify-between px-1">
                  <span class="text-[15px] font-semibold text-slate-900 tracking-tight">Wiki</span>
                  <button
                    type="button"
                    (click)="sidebarCollapsed.set(true)"
                    aria-label="Collapse secondary panel"
                    class="p-1 rounded hover:bg-slate-100 text-slate-500 hover:text-slate-800"
                    title="Collapse Wiki panel"
                  >
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <rect x="3" y="4" width="18" height="16" rx="2.5" />
                      <path d="M9 4v16" />
                    </svg>
                  </button>
                </div>

                <!-- "+ New page" Outlined Action Button -->
                @if (projectService.canEdit()) {
                  <div>
                    <button
                      type="button"
                      (click)="isCreatePageOpen.set(true)"
                      class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg border border-slate-200 bg-white hover:bg-slate-50 text-[13px] font-medium text-slate-800 shadow-[0_1px_2px_rgba(0,0,0,0.03)] transition-colors"
                    >
                      <svg class="w-4 h-4 text-slate-600 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                        <path d="M6 3h9l5 5v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z" />
                        <path d="M12 11v6M9 14h6" />
                      </svg>
                      <span>New page</span>
                    </button>
                  </div>
                }

                <!-- Wiki "Home" Link -->
                <div>
                  <button
                    type="button"
                    (click)="selectWikiPage('page-welcome')"
                    class="w-full flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors text-left"
                  >
                    <svg class="w-4 h-4 text-slate-600 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <path d="M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1v-9.5Z" />
                    </svg>
                    <span>Home</span>
                  </button>
                </div>

                <!-- "Collections" Section with Nested Pages Tree per Collection -->
                <div class="pt-1 space-y-1">
                  <div class="flex items-center justify-between px-2 py-1">
                    <button
                      type="button"
                      (click)="wikiCollectionsExpanded.set(!wikiCollectionsExpanded())"
                      class="flex-1 flex items-center justify-between text-xs font-semibold text-slate-500 hover:text-slate-800"
                    >
                      <span>Collections</span>
                      <svg
                        class="w-3.5 h-3.5 transition-transform"
                        [ngClass]="wikiCollectionsExpanded() ? '' : '-rotate-90'"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                      >
                        <path d="m6 9 6 6 6-6" />
                      </svg>
                    </button>
                  </div>

                  @if (wikiCollectionsExpanded()) {
                    <div class="space-y-1.5">
                      @for (col of pageService.collections(); track col.id) {
                        <div class="space-y-0.5">
                          <!-- Collection Row (e.g. 📖 General, 📖 hodo, 📖 cloud-infra) -->
                          <div
                            class="group flex items-center justify-between px-2 py-1.5 rounded-lg transition-colors"
                            [ngClass]="
                              pageService.activeCollectionId() === col.id
                                ? 'bg-sky-50/80 text-sky-900'
                                : 'hover:bg-slate-100'
                            "
                          >
                            <button
                              type="button"
                              (click)="selectAndToggleCollection(col.id)"
                              [attr.aria-label]="'Collection ' + col.name"
                              class="flex-1 flex items-center gap-2 text-[13px] font-medium text-slate-800 text-left min-w-0"
                            >
                              <svg class="w-4 h-4 text-slate-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                                <path d="M2 6a2 2 0 0 1 2-2h6a2 2 0 0 1 2 2v14a2 2 0 0 0-2-2H4a2 2 0 0 1-2-2V6Z" />
                                <path d="M22 6a2 2 0 0 0-2-2h-6a2 2 0 0 0-2 2v14a2 2 0 0 1 2-2h6a2 2 0 0 0 2-2V6Z" />
                              </svg>
                              <span class="truncate">{{ col.name }}</span>
                            </button>

                            @if (projectService.canEdit()) {
                              <button
                                type="button"
                                (click)="openNewPageInCollection(col.id)"
                                class="p-0.5 rounded hover:bg-slate-200/70 text-slate-400 hover:text-slate-700"
                                title="New page in {{ col.name }}"
                              >
                                <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                                  <path d="M6 3h9l5 5v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z" />
                                  <path d="M12 11v6M9 14h6" />
                                </svg>
                              </button>
                            }
                          </div>

                          <!-- Nested Pages Tree inside this Collection -->
                          @if (isCollectionExpanded(col.id)) {
                            <div class="ml-3.5 pl-2.5 border-l border-slate-200 space-y-0.5">
                              @for (node of pageService.getPagesTreeForCollection(col.id); track node.page.id) {
                                <!-- Root Page in Collection -->
                                <button
                                  type="button"
                                  (click)="selectWikiPage(node.page.id)"
                                  class="w-full flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-[13px] transition-colors text-left"
                                  [ngClass]="
                                    pageService.activePage()?.id === node.page.id
                                      ? 'bg-[#eef0f3] text-slate-900 font-medium'
                                      : 'text-slate-700 hover:bg-slate-100'
                                  "
                                >
                                  <svg class="w-3.5 h-3.5 text-slate-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                                    <path d="M6 3h9l5 5v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z" />
                                    <path d="M8 11h8M8 15h5" />
                                  </svg>
                                  <span class="truncate">{{ node.page.title }}</span>
                                </button>

                                <!-- Nested Child Sub-Pages -->
                                @if (node.children.length > 0) {
                                  <div class="ml-3.5 pl-2.5 border-l border-slate-200 space-y-0.5">
                                    @for (child of node.children; track child.page.id) {
                                      <button
                                        type="button"
                                        (click)="selectWikiPage(child.page.id)"
                                        class="w-full flex items-center gap-2 px-2 py-1 rounded-lg text-xs transition-colors text-left"
                                        [ngClass]="
                                          pageService.activePage()?.id === child.page.id
                                            ? 'bg-[#eef0f3] text-slate-900 font-medium'
                                            : 'text-slate-600 hover:bg-slate-100'
                                        "
                                      >
                                        <svg class="w-3.5 h-3.5 text-slate-400 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                                          <path d="M6 3h9l5 5v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z" />
                                        </svg>
                                        <span class="truncate">{{ child.page.title }}</span>
                                      </button>
                                    }
                                  </div>
                                }
                              }
                            </div>
                          }
                        </div>
                      }
                    </div>
                  }
                </div>

                <!-- Shared Section -->
                <div class="pt-1 space-y-1">
                  <button
                    type="button"
                    (click)="wikiSharedExpanded.set(!wikiSharedExpanded())"
                    class="w-full flex items-center justify-between px-2 py-1 text-xs font-semibold text-slate-500 hover:text-slate-800"
                  >
                    <span>Shared</span>
                    <svg
                      class="w-3.5 h-3.5 transition-transform"
                      [ngClass]="wikiSharedExpanded() ? '' : '-rotate-90'"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <path d="m6 9 6 6 6-6" />
                    </svg>
                  </button>
                  @if (wikiSharedExpanded()) {
                    <div class="py-2 text-center text-xs text-slate-400">No shared pages</div>
                  }
                </div>

                <!-- Private Section -->
                <div class="pt-1 space-y-1">
                  <button
                    type="button"
                    (click)="wikiPrivateExpanded.set(!wikiPrivateExpanded())"
                    class="w-full flex items-center justify-between px-2 py-1 text-xs font-semibold text-slate-500 hover:text-slate-800"
                  >
                    <span>Private</span>
                    <svg
                      class="w-3.5 h-3.5 transition-transform"
                      [ngClass]="wikiPrivateExpanded() ? '' : '-rotate-90'"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <path d="m6 9 6 6 6-6" />
                    </svg>
                  </button>
                  @if (wikiPrivateExpanded()) {
                    <div class="py-2 text-center text-xs text-slate-400">No private pages</div>
                  }
                </div>

                <!-- Archives Section -->
                <div class="pt-1">
                  <button
                    type="button"
                    class="w-full flex items-center justify-between px-2 py-1 text-xs font-semibold text-slate-500 hover:text-slate-800"
                  >
                    <span>Archives</span>
                    <svg class="w-3.5 h-3.5 -rotate-90" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="m6 9 6 6 6-6" />
                    </svg>
                  </button>
                </div>
              </div>
            } @else if (activeRailSection() === 'SETTINGS') {
              <!-- ==================== SECTION 3: SETTINGS SECONDARY PANEL ==================== -->
              <div class="flex-1 overflow-y-auto px-3 pt-3.5 pb-4 space-y-4">
                <!-- Panel Header: "Settings" + collapse icon -->
                <div class="flex items-center justify-between px-1">
                  <span class="text-[15px] font-semibold text-slate-900 tracking-tight">Settings</span>
                  <button
                    type="button"
                    (click)="sidebarCollapsed.set(true)"
                    aria-label="Collapse secondary panel"
                    class="p-1 rounded hover:bg-slate-100 text-slate-500 hover:text-slate-800"
                    title="Collapse Settings panel"
                  >
                    <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <rect x="3" y="4" width="18" height="16" rx="2.5" />
                      <path d="M9 4v16" />
                    </svg>
                  </button>
                </div>

                <!-- General Application Settings Section -->
                <div class="space-y-1">
                  <div class="px-2 py-1 text-xs font-semibold text-slate-500">Application</div>
                  <button
                    type="button"
                    (click)="openGeneralSettingsFromSidebar()"
                    aria-label="Open General App Settings"
                    class="w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-[13px] transition-colors text-left"
                    [ngClass]="
                      settingsService.activeScope() === 'GENERAL'
                        ? 'bg-[#eef0f3] text-slate-900 font-medium'
                        : 'text-slate-700 hover:bg-slate-100'
                    "
                  >
                    <svg class="w-4 h-4 text-slate-600 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <circle cx="12" cy="12" r="3" />
                      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1Z" />
                    </svg>
                    <span>General Settings</span>
                  </button>
                </div>

                <!-- Per-Project Settings List -->
                <div class="pt-1 space-y-1">
                  <div class="px-2 py-1 text-xs font-semibold text-slate-500">Per-Project Settings</div>
                  <div class="space-y-1">
                    @for (proj of projectService.projects(); track proj.id) {
                      <button
                        type="button"
                        (click)="openProjectSettingsFromSidebar(proj.id)"
                        [attr.aria-label]="'Open project settings for ' + proj.name"
                        class="w-full flex items-center justify-between px-2.5 py-2 rounded-lg text-[13px] transition-colors text-left"
                        [ngClass]="
                          settingsService.activeScope() === 'PROJECT' &&
                          projectService.activeProjectId() === proj.id
                            ? 'bg-[#eef0f3] text-slate-900 font-medium'
                            : 'text-slate-700 hover:bg-slate-100'
                        "
                      >
                        <span class="flex items-center gap-2 min-w-0">
                          <span class="text-sm leading-none">👇</span>
                          <span class="truncate">{{ proj.name }}</span>
                        </span>
                        <span class="px-1.5 py-0.5 rounded bg-slate-100 border border-slate-200 font-mono text-[10px] text-slate-500">
                          {{ proj.key_prefix }}
                        </span>
                      </button>
                    }
                  </div>
                </div>
              </div>
            } @else {
              <!-- ==================== SECTION 1: WORK SECONDARY PANEL ==================== -->
              <div class="flex-1 overflow-y-auto px-3 pt-3.5 pb-4 space-y-4">
                <!-- Panel Header: "Work" + filter/collapse icons -->
                <div class="flex items-center justify-between px-1">
                  <span class="text-[15px] font-semibold text-slate-900 tracking-tight">Work</span>
                  <div class="flex items-center gap-1.5 text-slate-500">
                    <button
                      type="button"
                      (click)="workItemService.resetFilters()"
                      class="p-1 rounded hover:bg-slate-100 hover:text-slate-800"
                      title="Reset view filters"
                    >
                      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                        <path d="M4 6h16M4 12h16M4 18h16" />
                        <circle cx="9" cy="6" r="2" fill="white" />
                        <circle cx="15" cy="12" r="2" fill="white" />
                        <circle cx="10" cy="18" r="2" fill="white" />
                      </svg>
                    </button>
                    <button
                      type="button"
                      (click)="sidebarCollapsed.set(true)"
                      aria-label="Collapse secondary panel"
                      class="p-1 rounded hover:bg-slate-100 hover:text-slate-800"
                      title="Collapse Work panel"
                    >
                      <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                        <rect x="3" y="4" width="18" height="16" rx="2.5" />
                        <path d="M9 4v16" />
                      </svg>
                    </button>
                  </div>
                </div>

                <!-- "+ New work item" Outlined Action Button -->
                <div>
                  <button
                    type="button"
                    (click)="workItemService.isQuickCreateOpen.set(true)"
                    class="w-full flex items-center gap-2.5 px-3 py-2 rounded-lg border border-slate-200 bg-white hover:bg-slate-50 text-[13px] font-medium text-slate-800 shadow-[0_1px_2px_rgba(0,0,0,0.03)] transition-colors"
                  >
                    <svg class="w-4 h-4 text-slate-600 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <rect x="4" y="4" width="12" height="14" rx="2" />
                      <path d="M8 2v2M16 14h6M19 11v6" />
                    </svg>
                    <span>New work item</span>
                  </button>
                </div>

                <!-- Top Personal Navigation Links: Home, My Work, Stickies (No Drafts) -->
                <div class="space-y-0.5">
                  <a
                    routerLink="/home"
                    routerLinkActive="bg-[#eef0f3] text-slate-900 font-medium"
                    class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors"
                  >
                    <svg class="w-4 h-4 text-slate-600 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <path d="M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1v-9.5Z" />
                    </svg>
                    <span>Home</span>
                  </a>

                  <a
                    routerLink="/my-work"
                    routerLinkActive="bg-[#eef0f3] text-slate-900 font-medium"
                    class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors"
                  >
                    <svg class="w-4 h-4 text-slate-600 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <circle cx="10" cy="8" r="4" />
                      <path d="M3 20c0-3.5 3.1-6 7-6M18 14v6M15 17h6" />
                    </svg>
                    <span>My Work</span>
                  </a>

                  <a
                    routerLink="/stickies"
                    routerLinkActive="bg-[#eef0f3] text-slate-900 font-medium"
                    class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors"
                  >
                    <svg class="w-4 h-4 text-slate-600 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <path d="M4 5a2 2 0 0 1 2-2h12a2 2 0 0 1 2 2v10l-5 5H6a2 2 0 0 1-2-2V5Z" />
                      <path d="M14 19v-4a1 1 0 0 1 1-1h4" />
                    </svg>
                    <span>Stickies</span>
                  </a>
                </div>

                <!-- Projects Tree Section with "+" Button at the Root of the Projects List -->
                <div class="pt-2 space-y-1">
                  <div class="flex items-center justify-between px-2 py-1">
                    <button
                      type="button"
                      (click)="projectsTreeExpanded.set(!projectsTreeExpanded())"
                      class="flex-1 flex items-center gap-1.5 text-xs font-semibold text-slate-500 hover:text-slate-800 text-left"
                    >
                      <span>Projects</span>
                      <svg
                        class="w-3.5 h-3.5 transition-transform"
                        [ngClass]="projectsTreeExpanded() ? '' : '-rotate-90'"
                        viewBox="0 0 24 24"
                        fill="none"
                        stroke="currentColor"
                        stroke-width="2"
                      >
                        <path d="m6 9 6 6 6-6" />
                      </svg>
                    </button>

                    <button
                      type="button"
                      (click)="isCreateProjectOpen.set(true)"
                      class="p-1 rounded hover:bg-slate-100 text-slate-500 hover:text-slate-800 transition-colors"
                      title="Create new project"
                      aria-label="Create new project"
                    >
                      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2">
                        <path d="M12 5v14M5 12h14" />
                      </svg>
                    </button>
                  </div>

                  @if (projectsTreeExpanded()) {
                    <div class="space-y-2">
                      @for (proj of projectService.projects(); track proj.id) {
                        <div class="space-y-1">
                          <!-- Project Node Header (e.g. 👇 hodo) -->
                          <button
                            type="button"
                            (click)="selectAndToggleProject(proj.id)"
                            class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg text-[13px] font-medium text-slate-800 hover:bg-slate-100 text-left"
                          >
                            <span class="text-sm leading-none">👇</span>
                            <span class="truncate">{{ proj.name }}</span>
                          </button>

                          <!-- Indented Project Child Links: Overview, Work items, Cycles, Epics, Modules, Pages -->
                          @if (isProjectExpanded(proj.id)) {
                            <div class="ml-4 pl-3 border-l border-slate-200 space-y-0.5">
                              <a
                                [routerLink]="['/projects', proj.id]"
                                routerLinkActive="bg-[#eef0f3] text-slate-900 font-medium"
                                [routerLinkActiveOptions]="{ exact: true }"
                                (click)="selectProjectOnly(proj.id)"
                                [attr.aria-label]="'Open ' + proj.name + ' overview'"
                                class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors"
                              >
                                <svg class="w-4 h-4 text-slate-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                                  <rect x="3" y="4" width="18" height="16" rx="2" />
                                  <path d="m7 14 3-3 3 2 4-4" />
                                </svg>
                                <span>Overview</span>
                              </a>

                              <a
                                [routerLink]="['/projects', proj.id, 'work-items']"
                                routerLinkActive="bg-[#eef0f3] text-slate-900 font-medium"
                                (click)="selectProjectOnly(proj.id); workItemService.resetFilters()"
                                [attr.aria-label]="'Open ' + proj.name + ' work items'"
                                class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors"
                              >
                                <svg class="w-4 h-4 text-slate-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                                  <rect x="4" y="5" width="11" height="13" rx="2" />
                                  <path d="M9 3h9a2 2 0 0 1 2 2v11" />
                                </svg>
                                <span>Work items</span>
                              </a>

                              <a
                                [routerLink]="['/projects', proj.id, 'cycles']"
                                routerLinkActive="bg-[#eef0f3] text-slate-900 font-medium"
                                (click)="selectProjectOnly(proj.id)"
                                class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors"
                              >
                                <svg class="w-4 h-4 text-slate-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                                  <circle cx="12" cy="12" r="8" />
                                  <path d="M12 4a8 8 0 0 1 0 16V4Z" fill="currentColor" />
                                </svg>
                                <span>Cycles</span>
                              </a>

                              <a
                                [routerLink]="['/projects', proj.id, 'epics']"
                                routerLinkActive="bg-[#eef0f3] text-slate-900 font-medium"
                                (click)="selectProjectOnly(proj.id)"
                                class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors"
                              >
                                <svg class="w-4 h-4 text-slate-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                                  <path d="M5 4v16M5 4h12l-2.5 4.5L17 13H5" />
                                </svg>
                                <span>Epics</span>
                              </a>

                              <a
                                [routerLink]="['/projects', proj.id, 'modules']"
                                routerLinkActive="bg-[#eef0f3] text-slate-900 font-medium"
                                (click)="selectProjectOnly(proj.id)"
                                class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors"
                              >
                                <svg class="w-4 h-4 text-slate-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                                  <rect x="4" y="4" width="16" height="16" rx="3" />
                                  <circle cx="9" cy="9" r="1.2" fill="currentColor" />
                                  <circle cx="15" cy="9" r="1.2" fill="currentColor" />
                                  <circle cx="9" cy="15" r="1.2" fill="currentColor" />
                                  <circle cx="15" cy="15" r="1.2" fill="currentColor" />
                                </svg>
                                <span>Modules</span>
                              </a>

                              <a
                                routerLink="/wiki"
                                (click)="openProjectPagesInWiki(proj)"
                                [attr.aria-label]="'Open ' + proj.name + ' pages in Wiki'"
                                class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors"
                              >
                                <svg class="w-4 h-4 text-slate-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                                  <path d="M6 3h9l5 5v12a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2Z" />
                                  <path d="M8 11h8M8 15h6" />
                                </svg>
                                <span>Pages</span>
                              </a>
                            </div>
                          }
                        </div>
                      }
                    </div>
                  }
                </div>
              </div>
            }
          </aside>
        } @else {
          <!-- Collapsed Secondary Panel Strip with Expand Button -->
          <aside
            class="w-11 bg-white border-r border-t border-slate-200/90 rounded-tl-xl flex flex-col items-center pt-3.5 pb-4 shrink-0 select-none"
          >
            <button
              type="button"
              (click)="sidebarCollapsed.set(false)"
              aria-label="Expand secondary panel"
              title="Expand panel"
              class="p-1.5 rounded-md hover:bg-slate-100 text-slate-500 hover:text-slate-900 transition-colors"
            >
              <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                <rect x="3" y="4" width="18" height="16" rx="2.5" />
                <path d="M9 4v16" />
                <path d="m13 9 3 3-3 3" stroke-linecap="round" stroke-linejoin="round" />
              </svg>
            </button>
          </aside>
        }

        <!-- 3. Right Main Workspace Column -->
        <div class="flex-1 flex flex-col min-w-0 overflow-hidden bg-white border-t border-slate-200/90">
          <div class="flex-1 flex overflow-hidden">
            <main class="flex-1 w-full overflow-auto">
              <router-outlet></router-outlet>
            </main>
          </div>
        </div>
      </div>

      <!-- Global Quick Create Work Item Modal -->
      <app-modal
        [isOpen]="workItemService.isQuickCreateOpen()"
        title="Create New Work Item"
        maxWidth="max-w-2xl"
        (closed)="workItemService.isQuickCreateOpen.set(false)"
      >
        <form (ngSubmit)="submitQuickCreate()" class="space-y-3.5">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Title</label>
            <input
              type="text"
              name="title"
              [(ngModel)]="newTitle"
              required
              placeholder="e.g., Implement Firestore atomic sequence counter"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Description (Markdown)</label>
            <div
              class="growable-textarea text-xs font-mono leading-relaxed"
              [attr.data-replicated-value]="newDescription || ''"
            >
              <textarea
                name="description"
                [(ngModel)]="newDescription"
                (input)="syncGrowableTextarea($event)"
                aria-label="New work item description"
                placeholder="Describe acceptance criteria and technical details..."
                class="auto-grow-description w-full min-h-[160px] max-h-[200vh] px-3 py-2 text-xs font-mono leading-relaxed border border-slate-200 rounded-md focus:outline-none focus:border-slate-300 resize-y"
              ></textarea>
            </div>
          </div>

          <div class="grid grid-cols-3 gap-2.5">
            <div>
              <label class="block text-[11px] font-medium text-slate-600 mb-1">Status</label>
              <app-dropdown
                [options]="statusOptions"
                [value]="newStatus"
                (valueChange)="onNewStatusChange($event)"
                [fullWidth]="true"
              ></app-dropdown>
            </div>

            <div>
              <label class="block text-[11px] font-medium text-slate-600 mb-1">Priority</label>
              <app-dropdown
                [options]="priorityOptions"
                [value]="newPriority"
                (valueChange)="onNewPriorityChange($event)"
                [fullWidth]="true"
              ></app-dropdown>
            </div>

            <div>
              <label class="block text-[11px] font-medium text-slate-600 mb-1">Estimate (pts)</label>
              <input
                type="number"
                name="estimate"
                [(ngModel)]="newEstimate"
                min="0"
                class="w-full text-xs border border-slate-200 rounded-md px-2 py-1.5"
              />
            </div>
          </div>

          <div class="grid grid-cols-2 gap-2.5">
            <div>
              <div class="flex items-center justify-between mb-1">
                <label class="block text-[11px] font-medium text-slate-600">
                  Start Date <span class="font-normal text-slate-400">(optional)</span>
                </label>
                @if (newStartDate) {
                  <button
                    type="button"
                    (click)="newStartDate = ''"
                    class="text-[10px] text-slate-400 hover:text-slate-600"
                  >
                    Clear
                  </button>
                }
              </div>
              <input
                type="date"
                name="startDate"
                [(ngModel)]="newStartDate"
                aria-label="New work item start date"
                class="w-full text-xs border border-slate-200 rounded-md px-2 py-1.5"
              />
            </div>
            <div>
              <div class="flex items-center justify-between mb-1">
                <label class="block text-[11px] font-medium text-slate-600">
                  Due Date <span class="font-normal text-slate-400">(optional)</span>
                </label>
                @if (newDueDate) {
                  <button
                    type="button"
                    (click)="newDueDate = ''"
                    class="text-[10px] text-slate-400 hover:text-slate-600"
                  >
                    Clear
                  </button>
                }
              </div>
              <input
                type="date"
                name="dueDate"
                [(ngModel)]="newDueDate"
                aria-label="New work item due date"
                class="w-full text-xs border border-slate-200 rounded-md px-2 py-1.5"
              />
            </div>
          </div>

          <div class="flex justify-end gap-2 pt-2 border-t border-slate-200">
            <button
              type="button"
              (click)="workItemService.isQuickCreateOpen.set(false)"
              class="px-3 py-1.5 rounded-md border border-slate-200 text-xs text-slate-600 hover:bg-slate-100"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
            >
              Create Work Item
            </button>
          </div>
        </form>
      </app-modal>

      <!-- Global Quick Create Wiki Page Modal (triggered from Wiki panel "+ New page") -->
      <app-modal
        [isOpen]="isCreatePageOpen()"
        title="Create New Wiki Page"
        (closed)="isCreatePageOpen.set(false)"
      >
        <form (ngSubmit)="submitCreateWikiPage()" class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Page Title</label>
            <input
              type="text"
              name="pageTitle"
              [(ngModel)]="newPageTitle"
              required
              placeholder="e.g., Q4 Architecture RFC"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Collection</label>
            <app-dropdown
              [options]="collectionOptions()"
              [value]="newPageCollectionId"
              (valueChange)="newPageCollectionId = $event"
              [fullWidth]="true"
            ></app-dropdown>
          </div>

          <div class="flex justify-end gap-2 pt-2 border-t border-slate-200">
            <button
              type="button"
              (click)="isCreatePageOpen.set(false)"
              class="px-3 py-1.5 rounded-md border border-slate-200 text-xs text-slate-600 hover:bg-slate-100"
            >
              Cancel
            </button>
            <button
              type="submit"
              class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
            >
              Create Page
            </button>
          </div>
        </form>
      </app-modal>

      <!-- Global Create New Project Modal (triggered from "+" button at root of Projects list in left panel) -->
      <app-modal
        [isOpen]="isCreateProjectOpen()"
        title="Create New Project"
        (closed)="isCreateProjectOpen.set(false)"
      >
        <form (ngSubmit)="submitCreateProject()" class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Project Name</label>
            <input
              type="text"
              name="projName"
              [(ngModel)]="newProjectName"
              required
              aria-label="New project name"
              placeholder="e.g., mobile-sdk"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">
              Key Prefix (2-10 uppercase chars)
            </label>
            <input
              type="text"
              name="projPrefix"
              [(ngModel)]="newProjectPrefix"
              required
              aria-label="New project key prefix"
              placeholder="e.g., MSDK"
              class="w-full px-3 py-1.5 text-xs font-mono uppercase border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-700 mb-1">Description</label>
            <textarea
              rows="2"
              name="projDesc"
              [(ngModel)]="newProjectDesc"
              aria-label="New project description"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            ></textarea>
          </div>
          <div class="flex justify-end gap-2 pt-2 border-t border-slate-200">
            <button
              type="button"
              (click)="isCreateProjectOpen.set(false)"
              class="px-3 py-1.5 rounded-md border border-slate-200 text-xs text-slate-600 hover:bg-slate-100"
            >
              Cancel
            </button>
            <button
              type="submit"
              aria-label="Submit create project"
              class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
            >
              Create Project
            </button>
          </div>
        </form>
      </app-modal>
    </div>
  `,
})
export class AppComponent implements OnInit {
  readonly authService = inject(AuthService);
  readonly projectService = inject(ProjectService);
  readonly workItemService = inject(WorkItemService);
  readonly pageService = inject(PageService);
  readonly settingsService = inject(SettingsService);
  private readonly router = inject(Router);

  readonly activeRailSection = signal<'WORK' | 'WIKI' | 'SETTINGS'>('WORK');
  readonly sidebarCollapsed = signal<boolean>(false);

  // Work panel state
  readonly projectsTreeExpanded = signal<boolean>(true);
  readonly expandedProjects = signal<Record<string, boolean>>({
    'proj-jynko-core': true,
    'proj-cloud-infra': true,
  });

  // Wiki panel state
  readonly wikiCollectionsExpanded = signal<boolean>(true);
  readonly wikiSharedExpanded = signal<boolean>(true);
  readonly wikiPrivateExpanded = signal<boolean>(true);
  readonly expandedCollections = signal<Record<string, boolean>>({
    'col-general': true,
    'col-proj-jynko-core': true,
    'col-proj-cloud-infra': true,
  });
  readonly isCreatePageOpen = signal<boolean>(false);
  newPageTitle = '';
  newPageCollectionId = 'col-general';

  // Create Project modal state (triggered from "+" button at root of Projects list)
  readonly isCreateProjectOpen = signal<boolean>(false);
  newProjectName = '';
  newProjectPrefix = '';
  newProjectDesc = '';

  readonly statuses = ALL_STATUSES;
  readonly priorities = ALL_PRIORITIES;

  readonly statusOptions: DropdownOption[] = ALL_STATUSES.map((s) => ({
    value: s.value,
    label: s.label,
  }));
  readonly priorityOptions: DropdownOption[] = ALL_PRIORITIES.map((p) => ({
    value: p.value,
    label: p.label,
  }));
  readonly collectionOptions = computed<DropdownOption[]>(() =>
    this.pageService.collections().map((c) => ({ value: c.id, label: c.name }))
  );

  newTitle = '';
  newDescription = '';
  newStatus: WorkItemStatus = 'TODO';
  newPriority: WorkItemPriority = 'MEDIUM';
  newEstimate = 3;
  newStartDate = '';
  newDueDate = '';

  readonly isUserMenuOpen = signal<boolean>(false);

  ngOnInit(): void {
    this.authService.loadCurrentUser();
    this.projectService.loadProjects();

    // Sync activeRailSection with the current route URL
    if (this.router.url.startsWith('/wiki')) {
      this.activeRailSection.set('WIKI');
    } else if (this.router.url.startsWith('/settings')) {
      this.activeRailSection.set('SETTINGS');
    }
    this.router.events
      .pipe(filter((e): e is NavigationEnd => e instanceof NavigationEnd))
      .subscribe((e) => {
        if (e.urlAfterRedirects.startsWith('/wiki')) {
          this.activeRailSection.set('WIKI');
        } else if (e.urlAfterRedirects.startsWith('/settings')) {
          this.activeRailSection.set('SETTINGS');
        } else if (this.activeRailSection() !== 'WORK') {
          this.activeRailSection.set('WORK');
        }
      });
  }

  openGeneralSettingsFromSidebar(): void {
    this.settingsService.openGeneralSettings();
    this.router.navigate(['/settings']);
  }

  openProjectSettingsFromSidebar(projectId: string): void {
    this.selectProjectOnly(projectId);
    this.settingsService.openProjectSettings(projectId);
    this.router.navigate(['/settings'], { queryParams: { project: projectId } });
  }

  isProjectExpanded(projectId: string): boolean {
    return this.expandedProjects()[projectId] ?? true;
  }

  selectProjectOnly(projectId: string): void {
    if (this.projectService.activeProjectId() !== projectId) {
      this.projectService.selectProject(projectId);
      this.workItemService.loadItems(projectId);
      this.pageService.loadWiki(projectId);
    }
  }

  selectAndToggleProject(projectId: string): void {
    if (this.projectService.activeProjectId() !== projectId) {
      this.projectService.selectProject(projectId);
      this.workItemService.loadItems(projectId);
      this.pageService.loadWiki(projectId);
      this.expandedProjects.update((m) => ({ ...m, [projectId]: true }));

      // Keep the browser URL in sync when switching projects while on a project-scoped route
      const urlPath = this.router.url.split('?')[0];
      const match = urlPath.match(/^\/projects\/[^/]+(?:\/(work-items|cycles|epics|modules)(?:\/.*)?)?$/);
      if (match) {
        const sub = match[1];
        if (sub) {
          this.router.navigate(['/projects', projectId, sub]);
        } else {
          this.router.navigate(['/projects', projectId]);
        }
      }
    } else {
      this.expandedProjects.update((m) => ({
        ...m,
        [projectId]: !(m[projectId] ?? true),
      }));
    }
  }

  isCollectionExpanded(colId: string): boolean {
    return this.expandedCollections()[colId] ?? true;
  }

  toggleCollectionExpanded(colId: string): void {
    this.expandedCollections.update((m) => ({
      ...m,
      [colId]: !(m[colId] ?? true),
    }));
  }

  selectAndToggleCollection(colId: string): void {
    if (this.pageService.activeCollectionId() !== colId) {
      this.pageService.selectCollection(colId);
      this.expandedCollections.update((m) => ({ ...m, [colId]: true }));
    } else {
      this.toggleCollectionExpanded(colId);
    }
  }

  openProjectPagesInWiki(proj: Project): void {
    this.selectProjectOnly(proj.id);
    const col = this.pageService.navigateToProjectCollection(proj);
    const colId = col?.id ?? `col-${proj.id}`;
    this.wikiCollectionsExpanded.set(true);
    this.expandedCollections.update((m) => ({ ...m, [colId]: true }));
    this.activeRailSection.set('WIKI');
    this.router.navigate(['/wiki']);
  }

  selectWikiPage(pageId: string): void {
    this.pageService.selectPage(pageId);
    if (!this.router.url.startsWith('/wiki')) {
      this.router.navigate(['/wiki']);
    }
  }

  openNewPageInCollection(colId: string): void {
    this.newPageCollectionId = colId;
    this.isCreatePageOpen.set(true);
  }

  submitCreateWikiPage(): void {
    const title = this.newPageTitle.trim();
    if (!title) return;
    this.pageService.createPage(title, undefined, this.newPageCollectionId);
    this.newPageTitle = '';
    this.isCreatePageOpen.set(false);
    if (!this.router.url.startsWith('/wiki')) {
      this.router.navigate(['/wiki']);
    }
  }

  submitCreateProject(): void {
    const name = this.newProjectName.trim();
    if (!name) return;
    const prefix = this.newProjectPrefix.trim().toUpperCase() || name.slice(0, 4).toUpperCase();
    const desc = this.newProjectDesc.trim();
    this.projectService.createProject(name, prefix, desc, (created) => {
      this.pageService.loadWiki(created.id);
      this.expandedProjects.update((m) => ({ ...m, [created.id]: true }));
      this.expandedCollections.update((m) => ({ ...m, [`col-${created.id}`]: true }));
      const urlPath = this.router.url.split('?')[0];
      const match = urlPath.match(/^\/projects\/[^/]+(?:\/(work-items|cycles|epics|modules))?$/);
      if (match) {
        const sub = match[1];
        this.router.navigate(sub ? ['/projects', created.id, sub] : ['/projects', created.id]);
      }
    });
    this.newProjectName = '';
    this.newProjectPrefix = '';
    this.newProjectDesc = '';
    this.isCreateProjectOpen.set(false);
  }

  onNewStatusChange(val: string): void {
    this.newStatus = val as WorkItemStatus;
  }

  onNewPriorityChange(val: string): void {
    this.newPriority = val as WorkItemPriority;
  }

  submitQuickCreate(): void {
    const title = this.newTitle.trim();
    if (!title) return;
    this.workItemService.createItem({
      title,
      description: this.newDescription.trim(),
      status: this.newStatus,
      priority: this.newPriority,
      estimate: this.newEstimate,
      start_date: this.newStartDate || undefined,
      due_date: this.newDueDate || undefined,
    });
    this.newTitle = '';
    this.newDescription = '';
    this.newStartDate = '';
    this.newDueDate = '';
    this.workItemService.isQuickCreateOpen.set(false);
  }

  syncGrowableTextarea(event: Event): void {
    if (typeof CSS !== 'undefined' && !CSS.supports('field-sizing', 'content')) {
      const el = event.target as HTMLTextAreaElement;
      if (el?.parentElement) {
        el.parentElement.dataset['replicatedValue'] = el.value;
      }
    }
  }

  toggleUserMenu(event: Event): void {
    event.stopPropagation();
    this.isUserMenuOpen.set(!this.isUserMenuOpen());
  }

  openProfilePage(): void {
    this.isUserMenuOpen.set(false);
    this.router.navigate(['/profile']);
  }

  signOut(): void {
    this.isUserMenuOpen.set(false);
    this.authService.signOut();
    this.router.navigate(['/home']);
  }
}
