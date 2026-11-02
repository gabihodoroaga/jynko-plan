import { Routes } from '@angular/router';
import { activeProjectRedirectGuard, rbacGuard } from './core/guards/rbac.guard';
import { HomeComponent } from './features/home/home.component';
import { MyWorkComponent } from './features/my-work/my-work.component';
import { WorkItemsShellComponent } from './features/work-items/work-items-shell/work-items-shell.component';
import { WorkItemDetailPageComponent } from './features/work-items/work-item-detail-page/work-item-detail-page.component';
import { CyclesComponent } from './features/cycles/cycles.component';
import { EpicsComponent } from './features/epics/epics.component';
import { ModulesComponent } from './features/modules/modules.component';
import { StickiesComponent } from './features/stickies/stickies.component';
import { PageDetailComponent } from './features/wiki/page-detail/page-detail.component';
import { ProjectsComponent } from './features/projects/projects.component';
import { SettingsComponent } from './features/settings/settings.component';
import { ProfileComponent } from './features/profile/profile.component';

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'work-items' },
  {
    path: 'home',
    component: HomeComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  {
    path: 'my-work',
    component: MyWorkComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  {
    path: 'stickies',
    component: StickiesComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  {
    path: 'wiki',
    component: PageDetailComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  {
    path: 'settings',
    component: SettingsComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  {
    path: 'profile',
    component: ProfileComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  // Project-scoped routes reflecting the currently active project in the URL
  {
    path: 'projects/:projectId',
    component: ProjectsComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  {
    path: 'projects/:projectId/overview',
    redirectTo: 'projects/:projectId',
    pathMatch: 'full',
  },
  {
    path: 'projects/:projectId/work-items',
    component: WorkItemsShellComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  {
    path: 'projects/:projectId/work-items/:itemId',
    component: WorkItemDetailPageComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  {
    path: 'projects/:projectId/cycles',
    component: CyclesComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  {
    path: 'projects/:projectId/epics',
    component: EpicsComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  {
    path: 'projects/:projectId/modules',
    component: ModulesComponent,
    canActivate: [rbacGuard('VIEWER')],
  },
  // Redirect bare project-scoped routes to the currently active project URL
  {
    path: 'projects',
    pathMatch: 'full',
    children: [],
    canActivate: [activeProjectRedirectGuard()],
  },
  {
    path: 'work-items',
    pathMatch: 'full',
    children: [],
    canActivate: [activeProjectRedirectGuard('work-items')],
  },
  {
    path: 'cycles',
    pathMatch: 'full',
    children: [],
    canActivate: [activeProjectRedirectGuard('cycles')],
  },
  {
    path: 'epics',
    pathMatch: 'full',
    children: [],
    canActivate: [activeProjectRedirectGuard('epics')],
  },
  {
    path: 'modules',
    pathMatch: 'full',
    children: [],
    canActivate: [activeProjectRedirectGuard('modules')],
  },
  { path: '**', redirectTo: 'work-items' },
];
