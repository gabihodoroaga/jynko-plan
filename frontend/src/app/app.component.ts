import { Component, ElementRef, OnInit, ViewChild, computed, effect, inject, signal, untracked } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { NavigationEnd, Router, RouterLink, RouterLinkActive, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs';
import { AuthService, DevAccountPreset } from './core/services/auth.service';
import { ProjectService } from './core/services/project.service';
import { WorkItemService } from './core/services/work-item.service';
import { PageService } from './core/services/page.service';
import { SettingsService } from './core/services/settings.service';
import { PwaService } from './core/services/pwa.service';
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
    @if (authService.isSignedOut() || isSignedOutRoute()) {
      <router-outlet />
    } @else {
    <div class="flex flex-col h-dvh w-full overflow-hidden bg-[#f0f4f9] text-slate-900">
      <!-- Global Top Application Bar (Jynko App Icon on left, Search in center, User on right) -->
      <header class="h-11 px-3 sm:px-3.5 bg-[#f0f4f9] flex items-center justify-between gap-2 sm:gap-4 shrink-0 select-none">
        <!-- Left: Mobile Hamburger + Simple Application Brand Icon (no dropdown caret) -->
        <div class="flex items-center gap-1.5 md:min-w-[180px] shrink-0">
          <button
            type="button"
            (click)="toggleMobileNavigation()"
            aria-label="Toggle mobile navigation"
            [attr.aria-expanded]="settingsService.isMobileMenuOpen()"
            class="md:hidden w-8 h-8 rounded-lg flex items-center justify-center text-slate-700 hover:bg-[#e1e9f4] transition-colors"
            title="Toggle navigation menu"
          >
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round">
              @if (settingsService.isMobileMenuOpen()) {
                <path d="M18 6 6 18M6 6l12 12" />
              } @else {
                <path d="M4 7h16M4 12h16M4 17h16" />
              }
            </svg>
          </button>

          <a
            routerLink="/home"
            (click)="activeRailSection.set('WORK'); settingsService.closeMobileMenu()"
            class="inline-flex items-center gap-2 px-1.5 py-1 rounded-lg hover:bg-[#e1e9f4] transition-colors"
            title="Jynko Home"
          >
            <!-- Custom Jynko Geometric Prism / Gemini Spark Application Icon -->
            <span
              class="w-6 h-6 rounded-lg text-white flex items-center justify-center shadow-sm"
              style="background: linear-gradient(135deg, #1a73e8 0%, #6366f1 52%, #a855f7 100%)"
            >
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

          @if (authService.appMeta().env === 'dev' || authService.appMeta().env === 'stg') {
            <span
              data-testid="env-badge"
              class="inline-flex items-center gap-1 px-1.5 py-0.5 rounded-full border text-[10px] font-semibold uppercase tracking-wide shrink-0"
              [ngClass]="
                authService.appMeta().env === 'dev'
                  ? 'bg-amber-50 border-amber-200 text-amber-800'
                  : 'bg-sky-50 border-sky-200 text-sky-800'
              "
            >
              <span>{{ authService.appMeta().env }}</span>
              @if (authService.appMeta().build_sha) {
                <span class="hidden lg:inline font-mono font-normal lowercase opacity-80">
                  · {{ authService.appMeta().build_sha | slice:0:7 }}
                </span>
              }
            </span>
          }
        </div>

        <!-- Center: Global Search Input -->
        <div class="flex-1 max-w-md">
          <div class="relative flex items-center">
            <svg
              class="w-3.5 h-3.5 text-slate-500 absolute left-3 pointer-events-none"
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
              class="w-full h-7 pl-8 pr-3 rounded-full bg-[#e3e9f2] hover:bg-[#dce3ee] focus:bg-white border border-transparent focus:border-[#a8c7fa] text-xs text-slate-800 placeholder-slate-500 focus:outline-none transition-colors"
            />
          </div>
        </div>

        <!-- Right: Offline Indicator + User Avatar & Account Popover Card -->
        <div class="relative flex items-center gap-2">
          @if (!pwaService.isOnline() && settingsService.generalSettings().pwaOfflineBanner) {
            <span
              data-testid="pwa-offline-badge"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-amber-50 border border-amber-200 text-amber-700 text-[11px] font-medium shrink-0"
              title="Network connection lost — reading from cached data where available"
            >
              <span class="w-1.5 h-1.5 rounded-full bg-amber-500"></span>
              <span class="hidden sm:inline">Offline</span>
            </span>
          }

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
              (click)="isUserMenuOpen.set(false); isSwitchAccountOpen.set(false)"
            ></div>

            <!-- Google-Style Account Card Dropdown -->
            <div
              data-testid="user-dropdown-card"
              (click)="$event.stopPropagation()"
              class="absolute right-0 top-9 z-50 w-[calc(100vw-1.5rem)] max-w-80 rounded-2xl bg-white border border-slate-200 shadow-[0_12px_32px_rgba(15,23,42,0.14)] overflow-hidden"
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

                  <div class="mt-3 flex flex-wrap items-center gap-2">
                    <button
                      type="button"
                      (click)="openProfilePage()"
                      aria-label="Open user profile"
                      class="inline-flex items-center px-3.5 py-1.5 rounded-full bg-[#1a73e8] hover:bg-[#1557b0] text-white text-xs font-medium transition-colors shadow-sm"
                    >
                      Profile
                    </button>
                    <button
                      type="button"
                      (click)="onSwitchAccountClick()"
                      aria-label="Switch user account"
                      [attr.aria-expanded]="authService.isDevMode ? isSwitchAccountOpen() : null"
                      [attr.aria-controls]="authService.isDevMode ? 'user-dropdown-account-switcher' : null"
                      class="inline-flex items-center gap-1 px-3 py-1.5 rounded-full border border-slate-300 bg-white hover:bg-slate-50 text-xs font-medium text-slate-700 transition-colors"
                    >
                      <svg class="w-3.5 h-3.5 text-slate-500 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M16 3l4 4-4 4M20 7H4M8 21l-4-4 4-4M4 17h16" stroke-linecap="round" stroke-linejoin="round" />
                      </svg>
                      <span>Switch account</span>
                    </button>
                  </div>
                </div>
              </div>

              @if (authService.isDevMode && isSwitchAccountOpen()) {
                <div
                  id="user-dropdown-account-switcher"
                  data-testid="dev-account-switcher-list"
                  class="border-t border-slate-200 bg-slate-50/70 px-3 py-2.5 space-y-1.5 max-h-60 overflow-y-auto"
                >
                  <div class="px-2 flex items-center justify-between">
                    <span class="text-[10px] font-semibold uppercase tracking-wider text-slate-500">
                      Other accounts (ENV=dev)
                    </span>
                  </div>
                  @for (acc of otherSwitchableAccounts(); track acc.email) {
                    <button
                      type="button"
                      (click)="switchToDevAccount(acc)"
                      [attr.aria-label]="'Switch to account ' + acc.email"
                      class="w-full px-2.5 py-2 rounded-xl flex items-center gap-2.5 text-left hover:bg-white border border-transparent hover:border-slate-200 transition-colors"
                    >
                      <span
                        class="w-7 h-7 rounded-full text-white text-xs font-bold flex items-center justify-center shrink-0"
                        [style.backgroundColor]="acc.avatar_color || '#0284c7'"
                      >
                        {{ (acc.name || acc.email).charAt(0).toUpperCase() }}
                      </span>
                      <span class="min-w-0 flex-1">
                        <span class="block text-xs font-medium text-slate-900 truncate">{{ acc.name }}</span>
                        <span class="block text-[11px] text-slate-500 truncate">{{ acc.email }}</span>
                      </span>
                      @if (acc.badge) {
                        <span class="text-[10px] px-1.5 py-0.5 rounded bg-slate-200/70 text-slate-600 font-medium shrink-0">
                          {{ acc.badge }}
                        </span>
                      }
                    </button>
                  }
                  <button
                    type="button"
                    (click)="openSignedOutChooser()"
                    aria-label="Use another account"
                    class="w-full px-2.5 py-2 rounded-xl flex items-center gap-2.5 text-left text-xs font-medium text-sky-700 hover:bg-white border border-transparent hover:border-slate-200 transition-colors"
                  >
                    <span class="w-7 h-7 rounded-full bg-sky-100 text-sky-700 flex items-center justify-center text-sm font-semibold shrink-0">
                      +
                    </span>
                    <span>Use another account…</span>
                  </button>
                </div>
              }

              <!-- Bottom Action Bar: Install App (when in browser tab) + Sign out -->
              <div class="px-5 py-3 border-t border-slate-200 bg-slate-50/50 flex items-center justify-between gap-2">
                @if (!pwaService.isStandalone() && settingsService.generalSettings().pwaShowInstallAction) {
                  <button
                    type="button"
                    (click)="installPwaFromMenu()"
                    aria-label="Install Jynko PWA"
                    class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-sky-200 bg-sky-50 hover:bg-sky-100 text-xs font-medium text-sky-700 transition-colors"
                  >
                    <svg class="w-3.5 h-3.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M12 3v12m0 0-4-4m4 4 4-4M4 17v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" stroke-linecap="round" stroke-linejoin="round" />
                    </svg>
                    <span>Install app</span>
                  </button>
                } @else {
                  <span></span>
                }
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
      <div class="relative flex flex-1 min-h-0 overflow-hidden">
        <!-- Mobile Backdrop Overlay when off-canvas navigation drawer is open -->
        @if (settingsService.isMobileMenuOpen()) {
          <div
            data-testid="mobile-nav-backdrop"
            class="fixed inset-0 z-40 bg-slate-900/35 backdrop-blur-[1px] md:hidden"
            (click)="settingsService.closeMobileMenu()"
          ></div>
        }

        <!-- Single-DOM Off-Canvas Navigation Wrapper on < md; Static Flex Row on >= md -->
        <div
          data-testid="app-navigation-drawer"
          class="fixed inset-y-0 left-0 z-50 flex transition-transform duration-200 ease-out md:static md:z-auto md:translate-x-0 md:shadow-none shrink-0"
          [ngClass]="
            settingsService.isMobileMenuOpen()
              ? 'translate-x-0 shadow-2xl'
              : '-translate-x-full md:translate-x-0'
          "
        >
        <!-- 1. Left Icon Navigation Rail (w-[68px]) with ONLY 3 buttons: Work, Wiki, Settings -->
        <nav
          aria-label="Primary rail"
          class="w-[68px] bg-[#f0f4f9] flex flex-col items-center pt-2 pb-4 shrink-0 select-none"
        >
          <div class="flex flex-col items-center gap-3.5 w-full">
            <!-- Button 1: Work -->
            <a
              [routerLink]="['/projects', projectService.activeProjectId(), 'work-items']"
              (click)="activeRailSection.set('WORK'); sidebarCollapsed.set(false)"
              class="group flex flex-col items-center gap-1 text-center cursor-pointer"
            >
              <div
                class="w-10 h-10 rounded-full flex items-center justify-center transition-colors"
                [ngClass]="
                  activeRailSection() === 'WORK'
                    ? 'bg-[#d3e3fd] text-[#041e49]'
                    : 'text-slate-700 group-hover:bg-[#e1e9f4]'
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
                    [attr.fill]="activeRailSection() === 'WORK' ? '#d3e3fd' : '#f0f4f9'"
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
                class="w-10 h-10 rounded-full flex items-center justify-center transition-colors"
                [ngClass]="
                  activeRailSection() === 'WIKI'
                    ? 'bg-[#d3e3fd] text-[#041e49]'
                    : 'text-slate-600 group-hover:bg-[#e1e9f4]'
                "
              >
                <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none">
                  <path
                    d="M10.586 3.586a2 2 0 0 1 2.828 0l7 7a2 2 0 0 1 0 2.828l-7 7a2 2 0 0 1-2.828 0l-7-7a2 2 0 0 1 0-2.828l7-7Z"
                    fill="currentColor"
                  />
                  <circle cx="12" cy="12" r="3.2" [attr.fill]="activeRailSection() === 'WIKI' ? '#d3e3fd' : '#f0f4f9'" />
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
                class="w-10 h-10 rounded-full flex items-center justify-center transition-colors"
                [ngClass]="
                  activeRailSection() === 'SETTINGS'
                    ? 'bg-[#d3e3fd] text-[#041e49]'
                    : 'text-slate-600 group-hover:bg-[#e1e9f4]'
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
            class="w-64 bg-[#f8fafd] border-r border-t border-slate-200/90 rounded-tl-2xl flex flex-col justify-between shrink-0 select-none overflow-hidden"
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
                    (click)="selectWikiHome()"
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
                                      ? 'bg-[#d3e3fd] text-[#041e49] font-medium'
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
                                            ? 'bg-[#d3e3fd] text-[#041e49] font-medium'
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
                        ? 'bg-[#d3e3fd] text-[#041e49] font-medium'
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
                            ? 'bg-[#d3e3fd] text-[#041e49] font-medium'
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
                @if (projectService.canEdit()) {
                  <div>
                    <button
                      type="button"
                      data-testid="open-create-work-item-btn"
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
                }

                <!-- Top Personal Navigation Links: Home, My Work, Stickies (No Drafts) -->
                <div class="space-y-0.5">
                  <a
                    routerLink="/home"
                    routerLinkActive="bg-[#d3e3fd] text-[#041e49] font-medium"
                    class="flex items-center gap-2.5 px-2.5 py-1.5 rounded-lg text-[13px] text-slate-700 hover:bg-slate-100 transition-colors"
                  >
                    <svg class="w-4 h-4 text-slate-600 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8">
                      <path d="M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6h-6v6H4a1 1 0 0 1-1-1v-9.5Z" />
                    </svg>
                    <span>Home</span>
                  </a>

                  <a
                    routerLink="/my-work"
                    routerLinkActive="bg-[#d3e3fd] text-[#041e49] font-medium"
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
                    routerLinkActive="bg-[#d3e3fd] text-[#041e49] font-medium"
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
                      (click)="projectService.openCreateProjectModal()"
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
                                routerLinkActive="bg-[#d3e3fd] text-[#041e49] font-medium"
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
                                routerLinkActive="bg-[#d3e3fd] text-[#041e49] font-medium"
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
                                routerLinkActive="bg-[#d3e3fd] text-[#041e49] font-medium"
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
                                routerLinkActive="bg-[#d3e3fd] text-[#041e49] font-medium"
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
                                routerLinkActive="bg-[#d3e3fd] text-[#041e49] font-medium"
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
            class="w-11 bg-[#f8fafd] border-r border-t border-slate-200/90 rounded-tl-2xl flex flex-col items-center pt-3.5 pb-4 shrink-0 select-none"
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
        </div>

        <!-- 3. Right Main Workspace Column -->
        <div class="flex-1 flex flex-col min-w-0 overflow-hidden bg-white border-t border-slate-200/90">
          <div class="flex-1 flex overflow-hidden">
            <main
              class="flex-1 w-full overflow-auto"
              [style.--project-bg]="settingsService.activeProjectBackground().canvas"
              [attr.data-project-bg]="settingsService.activeProjectBackground().key"
            >
              <router-outlet></router-outlet>
            </main>
          </div>
        </div>
      </div>

      <!-- Global Quick Create Work Item Modal -->
      <app-modal
        [isOpen]="workItemService.isQuickCreateOpen()"
        title="Create work item"
        maxWidth="max-w-2xl"
        (closed)="workItemService.isQuickCreateOpen.set(false)"
      >
        <form (ngSubmit)="submitQuickCreate()" class="space-y-3.5">
          <div>
            <label for="quick-create-title" class="block text-xs font-medium text-slate-700 mb-1">Title</label>
            <input
              id="quick-create-title"
              type="text"
              name="title"
              [(ngModel)]="newTitle"
              required
              placeholder="e.g., Implement Firestore atomic sequence counter"
              class="w-full px-3 py-1.5 text-xs border border-slate-200 rounded-md focus:outline-none focus:border-slate-300"
            />
          </div>

          <div>
            <label for="quick-create-description" class="block text-xs font-medium text-slate-700 mb-1">Description (Markdown)</label>
            <div
              class="growable-textarea text-xs font-mono leading-relaxed"
              [attr.data-replicated-value]="newDescription || ''"
            >
              <textarea
                id="quick-create-description"
                name="description"
                [(ngModel)]="newDescription"
                (input)="syncGrowableTextarea($event)"
                placeholder="Describe acceptance criteria and technical details..."
                class="auto-grow-description w-full min-h-[160px] max-h-[200vh] px-3 py-2 text-xs font-mono leading-relaxed border border-slate-200 rounded-md focus:outline-none focus:border-slate-300 resize-y"
              ></textarea>
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
            <div>
              <span class="block text-[11px] font-medium text-slate-600 mb-1">Status</span>
              <app-dropdown
                [options]="statusOptions"
                [value]="newStatus"
                (valueChange)="onNewStatusChange($event)"
                ariaLabel="New work item status"
                [fullWidth]="true"
              ></app-dropdown>
            </div>

            <div>
              <span class="block text-[11px] font-medium text-slate-600 mb-1">Priority</span>
              <app-dropdown
                [options]="priorityOptions"
                [value]="newPriority"
                (valueChange)="onNewPriorityChange($event)"
                ariaLabel="New work item priority"
                [fullWidth]="true"
              ></app-dropdown>
            </div>

            <div>
              <span class="block text-[11px] font-medium text-slate-600 mb-1">Cycle</span>
              <app-dropdown
                [options]="cycleOptions()"
                [value]="newCycleId"
                emptyLabel="No cycle"
                (valueChange)="onNewCycleChange($event)"
                ariaLabel="New work item cycle"
                [fullWidth]="true"
              ></app-dropdown>
            </div>

            <div>
              <label for="quick-create-estimate" class="block text-[11px] font-medium text-slate-600 mb-1">Estimate (pts)</label>
              <input
                id="quick-create-estimate"
                type="number"
                name="estimate"
                [(ngModel)]="newEstimate"
                min="0"
                class="w-full text-xs border border-slate-200 rounded-md px-2.5 py-1.5 focus:outline-none focus:border-slate-300"
              />
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
            <div>
              <div class="flex items-center justify-between mb-1">
                <label for="quick-create-start-date" class="block text-[11px] font-medium text-slate-600">
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
                id="quick-create-start-date"
                type="date"
                name="startDate"
                [(ngModel)]="newStartDate"
                class="w-full text-xs border border-slate-200 rounded-md px-2.5 py-1.5 focus:outline-none focus:border-slate-300"
              />
            </div>
            <div>
              <div class="flex items-center justify-between mb-1">
                <label for="quick-create-due-date" class="block text-[11px] font-medium text-slate-600">
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
                id="quick-create-due-date"
                type="date"
                name="dueDate"
                [(ngModel)]="newDueDate"
                class="w-full text-xs border border-slate-200 rounded-md px-2.5 py-1.5 focus:outline-none focus:border-slate-300"
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
              Create work item
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

      <!-- Global Create / Edit Project Modal (triggered from "+" button at root of Projects list in left panel or "Edit" button in Project Overview) -->
      <app-modal
        [isOpen]="projectService.isProjectModalOpen()"
        [title]="projectService.editingProject() ? 'Edit Project' : 'Create New Project'"
        (closed)="projectService.closeProjectModal()"
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
              (click)="projectService.closeProjectModal()"
              class="px-3 py-1.5 rounded-md border border-slate-200 text-xs text-slate-600 hover:bg-slate-100"
            >
              Cancel
            </button>
            <button
              type="submit"
              aria-label="Submit create project"
              class="px-3 py-1.5 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-xs font-medium"
            >
              {{ projectService.editingProject() ? 'Save Changes' : 'Create Project' }}
            </button>
          </div>
        </form>
      </app-modal>

      <!-- Non-Intrusive Top-Layer PWA Update Toast (popover="manual" with explicit Close & Reload) -->
      <div
        #pwaToastEl
        id="pwa-update-toast"
        [attr.popover]="'manual'"
        role="status"
        aria-live="polite"
        class="pwa-toast rounded-lg bg-white border border-slate-200 shadow-[0_10px_28px_rgba(15,23,42,0.14)] p-3.5 text-xs text-slate-800"
        [class.pwa-toast-fallback-open]="pwaService.updateAvailable()"
      >
        <div class="flex items-center justify-between gap-3">
          <div class="min-w-0">
            <div class="font-semibold text-slate-900">Update available</div>
            <div class="text-[11px] text-slate-500 mt-0.5">A new version of Jynko is ready.</div>
          </div>
          <div class="flex items-center gap-1.5 shrink-0">
            <button
              type="button"
              (click)="pwaService.applyUpdateAndReload()"
              class="px-2.5 py-1 rounded-md bg-sky-600 hover:bg-sky-700 text-white text-[11px] font-medium transition-colors"
            >
              Reload
            </button>
            <button
              type="button"
              [attr.popovertarget]="'pwa-update-toast'"
              [attr.popovertargetaction]="'hide'"
              (click)="pwaService.dismissUpdateBanner()"
              aria-label="Close update notification"
              class="px-2 py-1 rounded-md border border-slate-200 bg-white hover:bg-slate-100 text-[11px] text-slate-600 transition-colors"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
    }
  `,
})
export class AppComponent implements OnInit {
  readonly authService = inject(AuthService);
  readonly projectService = inject(ProjectService);
  readonly workItemService = inject(WorkItemService);
  readonly pageService = inject(PageService);
  readonly settingsService = inject(SettingsService);
  readonly pwaService = inject(PwaService);
  private readonly router = inject(Router);

  @ViewChild('pwaToastEl') pwaToastRef?: ElementRef<HTMLElement>;

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

  // Create / Edit Project modal state (triggered from "+" button at root of Projects list or "Edit" in Project Overview)
  newProjectName = '';
  newProjectPrefix = '';
  newProjectDesc = '';

  constructor() {
    effect(() => {
      if (this.projectService.isProjectModalOpen()) {
        const editing = this.projectService.editingProject();
        if (editing) {
          this.newProjectName = editing.name;
          this.newProjectPrefix = editing.key_prefix;
          this.newProjectDesc = editing.description || '';
        } else {
          this.newProjectName = '';
          this.newProjectPrefix = '';
          this.newProjectDesc = '';
        }
      }
    });

    effect(() => {
      if (this.workItemService.isQuickCreateOpen()) {
        untracked(() => {
          const pset = this.settingsService.activeProjectSettings();
          if (ALL_STATUSES.some((s) => s.value === pset.defaultNewItemStatus)) {
            this.newStatus = pset.defaultNewItemStatus as WorkItemStatus;
          }
          if (ALL_PRIORITIES.some((p) => p.value === pset.defaultNewItemPriority)) {
            this.newPriority = pset.defaultNewItemPriority as WorkItemPriority;
          }
          const activeFilterCycle = this.workItemService.filter().cycle_id;
          const cycles = this.projectService.cycles();
          this.newCycleId =
            activeFilterCycle && cycles.some((c) => c.id === activeFilterCycle)
              ? activeFilterCycle
              : '';
          this.newEstimate = 1;
        });
      }
    });

    effect(() => {
      const showToast = this.pwaService.updateAvailable();
      const el = this.pwaToastRef?.nativeElement as (HTMLElement & { showPopover?: () => void; hidePopover?: () => void }) | undefined;
      if (!el || typeof HTMLElement === 'undefined' || !('popover' in HTMLElement.prototype)) {
        return;
      }
      try {
        if (showToast) {
          el.showPopover?.();
        } else {
          el.hidePopover?.();
        }
      } catch {}
    });
  }

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
  readonly cycleOptions = computed<DropdownOption[]>(() => [
    { value: '', label: 'No cycle' },
    ...this.projectService.cycles().map((c) => ({
      value: c.id,
      label: c.name,
      badge: c.is_active ? 'Active' : undefined,
    })),
  ]);
  readonly collectionOptions = computed<DropdownOption[]>(() =>
    this.pageService.collections().map((c) => ({ value: c.id, label: c.name }))
  );

  newTitle = '';
  newDescription = '';
  newStatus: WorkItemStatus = 'TODO';
  newPriority: WorkItemPriority = 'MEDIUM';
  newCycleId = '';
  newEstimate = 1;
  newStartDate = '';
  newDueDate = '';

  readonly isUserMenuOpen = signal<boolean>(false);
  readonly isSwitchAccountOpen = signal<boolean>(false);
  readonly isSignedOutRoute = signal<boolean>(false);

  readonly otherSwitchableAccounts = computed<DevAccountPreset[]>(() => {
    const curr = this.authService.currentUser();
    const currEmail = (curr.email || '').toLowerCase();
    const currId = (curr.user_id || '').toLowerCase();
    const base = [...this.authService.availableDevAccounts()];
    const seen = new Set(base.map((a) => a.email.toLowerCase()));

    for (const m of this.projectService.members()) {
      const mEmail = (m.email || '').trim().toLowerCase();
      if (mEmail && !seen.has(mEmail)) {
        seen.add(mEmail);
        base.push({
          user_id: m.user_id,
          email: m.email,
          name: m.name || m.email,
          avatar_color: '#0284c7',
          badge: m.role,
        });
      }
    }

    return base.filter(
      (a) =>
        a.email.toLowerCase() !== currEmail &&
        a.user_id.toLowerCase() !== currId
    );
  });

  ngOnInit(): void {
    if (
      typeof window !== 'undefined' &&
      (window.location.search.includes('ngsw-bypass') ||
        window.location.search.includes('gcp-iap-mode'))
    ) {
      try {
        const u = new URL(window.location.href);
        u.searchParams.delete('ngsw-bypass');
        u.searchParams.delete('gcp-iap-mode');
        window.history.replaceState({}, document.title, u.pathname + (u.search ? u.search : '') + u.hash);
      } catch {}
    }

    this.isSignedOutRoute.set(this.router.url.startsWith('/signed-out'));
    if (!this.authService.isSignedOut() && !this.isSignedOutRoute()) {
      this.authService.loadCurrentUser();
      this.projectService.loadProjects();
    }

    // Sync activeRailSection with the current route URL
    if (this.router.url.startsWith('/wiki')) {
      this.activeRailSection.set('WIKI');
    } else if (this.router.url.startsWith('/settings')) {
      this.activeRailSection.set('SETTINGS');
    }
    this.router.events
      .pipe(filter((e): e is NavigationEnd => e instanceof NavigationEnd))
      .subscribe((e) => {
        this.settingsService.closeMobileMenu();
        this.isSignedOutRoute.set(e.urlAfterRedirects.startsWith('/signed-out'));
        if (e.urlAfterRedirects.startsWith('/wiki')) {
          this.activeRailSection.set('WIKI');
        } else if (e.urlAfterRedirects.startsWith('/settings')) {
          this.activeRailSection.set('SETTINGS');
        } else if (this.activeRailSection() !== 'WORK') {
          this.activeRailSection.set('WORK');
        }
      });
  }

  toggleMobileNavigation(): void {
    const next = !this.settingsService.isMobileMenuOpen();
    if (next) {
      this.sidebarCollapsed.set(false);
    }
    this.settingsService.isMobileMenuOpen.set(next);
  }

  async installPwaFromMenu(): Promise<void> {
    this.isUserMenuOpen.set(false);
    const result = await this.pwaService.promptInstall();
    if (result === 'guide') {
      this.activeRailSection.set('SETTINGS');
      this.settingsService.openGeneralSettings();
      await this.router.navigate(['/settings']);
    }
  }

  openGeneralSettingsFromSidebar(): void {
    this.settingsService.openGeneralSettings();
    this.settingsService.closeMobileMenu();
    this.router.navigate(['/settings']);
  }

  openProjectSettingsFromSidebar(projectId: string): void {
    this.selectProjectOnly(projectId);
    this.settingsService.openProjectSettings(projectId);
    this.settingsService.closeMobileMenu();
    this.router.navigate(['/settings'], { queryParams: { project: projectId } });
  }

  isProjectExpanded(projectId: string): boolean {
    return this.expandedProjects()[projectId] ?? true;
  }

  selectProjectOnly(projectId: string): void {
    if (this.projectService.activeProjectId() !== projectId) {
      this.projectService.selectProject(projectId);
      this.workItemService.clearProjectScopedFilters();
      this.workItemService.loadItems(projectId);
      this.pageService.loadWiki(projectId);
    }
  }

  selectAndToggleProject(projectId: string): void {
    if (this.projectService.activeProjectId() !== projectId) {
      this.projectService.selectProject(projectId);
      this.workItemService.clearProjectScopedFilters();
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
    this.settingsService.closeMobileMenu();
    this.router.navigate(['/wiki']);
  }

  selectWikiHome(): void {
    const welcome =
      this.pageService.pages().find((p) => p.id === 'page-welcome') ??
      this.pageService.pages()[0];
    if (welcome) {
      this.selectWikiPage(welcome.id);
    } else if (!this.router.url.startsWith('/wiki')) {
      this.router.navigate(['/wiki']);
    }
  }

  selectWikiPage(pageId: string): void {
    this.pageService.selectPage(pageId);
    this.settingsService.closeMobileMenu();
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
    const editing = this.projectService.editingProject();
    if (editing) {
      this.projectService.updateProject(editing.id, {
        name,
        key_prefix: prefix,
        description: desc,
      });
    } else {
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
    }
    this.newProjectName = '';
    this.newProjectPrefix = '';
    this.newProjectDesc = '';
    this.projectService.closeProjectModal();
  }

  onNewStatusChange(val: string): void {
    this.newStatus = val as WorkItemStatus;
  }

  onNewPriorityChange(val: string): void {
    this.newPriority = val as WorkItemPriority;
  }

  onNewCycleChange(val: string): void {
    this.newCycleId = val;
  }

  submitQuickCreate(): void {
    const title = this.newTitle.trim();
    if (!title) return;
    this.workItemService.createItem({
      title,
      description: this.newDescription.trim(),
      status: this.newStatus,
      priority: this.newPriority,
      cycle_id: this.newCycleId || undefined,
      estimate: this.newEstimate,
      start_date: this.newStartDate || undefined,
      due_date: this.newDueDate || undefined,
    });
    this.newTitle = '';
    this.newDescription = '';
    this.newCycleId = '';
    this.newEstimate = 1;
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
    const next = !this.isUserMenuOpen();
    this.isUserMenuOpen.set(next);
    if (!next) {
      this.isSwitchAccountOpen.set(false);
    }
  }

  openProfilePage(): void {
    this.isUserMenuOpen.set(false);
    this.isSwitchAccountOpen.set(false);
    this.router.navigate(['/profile']);
  }

  onSwitchAccountClick(): void {
    if (this.authService.isDevMode) {
      this.isSwitchAccountOpen.set(!this.isSwitchAccountOpen());
      return;
    }
    this.openSignedOutChooser();
  }

  switchToDevAccount(acc: DevAccountPreset): void {
    this.isUserMenuOpen.set(false);
    this.isSwitchAccountOpen.set(false);
    this.authService.switchDevUser(acc);
  }

  openSignedOutChooser(): void {
    this.isUserMenuOpen.set(false);
    this.isSwitchAccountOpen.set(false);
    this.authService.signOut();
    this.router.navigate(['/signed-out'], { queryParams: { mode: 'switch' } });
  }

  signOut(): void {
    this.isUserMenuOpen.set(false);
    this.isSwitchAccountOpen.set(false);
    this.authService.signOut();
    this.router.navigate(['/signed-out']);
  }
}
