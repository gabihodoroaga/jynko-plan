import { ActivatedRouteSnapshot, CanActivateFn, Router } from '@angular/router';
import { inject } from '@angular/core';
import { ProjectService } from '../services/project.service';
import { AuthService } from '../services/auth.service';
import { ProjectRole } from '../models/project.model';

const ROLE_WEIGHT: Record<ProjectRole, number> = {
  VIEWER: 1,
  MEMBER: 2,
  ADMIN: 3,
};

export const rbacGuard = (minRole: ProjectRole = 'VIEWER'): CanActivateFn => {
  return (route: ActivatedRouteSnapshot) => {
    const authService = inject(AuthService);
    const projectService = inject(ProjectService);
    const router = inject(Router);

    if (authService.isSignedOut()) {
      return router.createUrlTree(['/signed-out']);
    }

    const routeProjectId = route.paramMap.get('projectId') || route.queryParamMap.get('project');
    if (routeProjectId) {
      projectService.activateProjectFromRoute(routeProjectId);
    }

    const currentRole = projectService.currentUserRole();
    if (ROLE_WEIGHT[currentRole] >= ROLE_WEIGHT[minRole]) {
      return true;
    }
    const activeId = projectService.activeProjectId() || 'proj-jynko-core';
    return router.createUrlTree(['/projects', activeId, 'work-items']);
  };
};

export const activeProjectRedirectGuard = (subPath?: string): CanActivateFn => {
  return () => {
    const authService = inject(AuthService);
    const projectService = inject(ProjectService);
    const router = inject(Router);

    if (authService.isSignedOut()) {
      return router.createUrlTree(['/signed-out']);
    }

    const activeId = projectService.activeProjectId() || 'proj-jynko-core';
    return subPath
      ? router.createUrlTree(['/projects', activeId, subPath])
      : router.createUrlTree(['/projects', activeId]);
  };
};
