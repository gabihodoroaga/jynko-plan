import { HttpErrorResponse, HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { catchError, throwError } from 'rxjs';
import { AuthService } from '../services/auth.service';
import { environment } from '../../../environments/environment';

let isRedirectingToLogin = false;
const IAP_RELOAD_TS_KEY = 'jynko.iap.reload.ts';
const IAP_RELOAD_COOLDOWN_MS = 15_000;

function canTriggerIapReload(): boolean {
  try {
    const prev = Number(sessionStorage.getItem(IAP_RELOAD_TS_KEY) || '0');
    if (prev > 0 && Date.now() - prev < IAP_RELOAD_COOLDOWN_MS) {
      return false;
    }
    sessionStorage.setItem(IAP_RELOAD_TS_KEY, String(Date.now()));
  } catch {
    // Allow single attempt if sessionStorage is unavailable
  }
  return true;
}

/**
 * In production behind Google Cloud IAP, authentication headers are injected at the edge proxy.
 *
 * This interceptor guarantees two critical behaviors:
 * 1. Attaches 'X-Requested-With': 'XMLHttpRequest' to all API calls. Per Google Cloud IAP
 *    architecture, this signals programmatic/AJAX requests so that when an IAP session expires,
 *    IAP responds with HTTP 401 Unauthorized instead of an HTTP 302 redirect to accounts.google.com
 *    (which gets blocked by browser CORS policy).
 * 2. Catches HTTP 401 Unauthorized responses and seamlessly reloads the window with '?ngsw-bypass=true'.
 *    This bypasses the Angular Service Worker cache for document navigation, directly contacting IAP
 *    to trigger Google OAuth re-authentication in the main browser window.
 * 3. In local development (environment.devMode), this interceptor attaches X-Dev-User-* headers.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);

  const headers: Record<string, string> = {
    'X-Requested-With': 'XMLHttpRequest',
  };

  if (environment.devMode) {
    const user = authService.currentUser();
    headers['X-Dev-User-Id'] = user.user_id;
    headers['X-Dev-User-Email'] = user.email;
    headers['X-Dev-User-Name'] = user.name;
  }

  const cloned = req.clone({
    setHeaders: headers,
  });

  return next(cloned).pipe(
    catchError((error: unknown) => {
      if (error instanceof HttpErrorResponse && error.status === 401) {
        if (
          !environment.devMode &&
          !authService.isSignedOut() &&
          typeof window !== 'undefined' &&
          !isRedirectingToLogin &&
          canTriggerIapReload()
        ) {
          isRedirectingToLogin = true;
          try {
            const target = new URL(window.location.href);
            target.searchParams.set('ngsw-bypass', 'true');
            window.location.href = target.toString();
          } catch {
            const current = window.location.href;
            const sep = current.includes('?') ? '&' : '?';
            window.location.href = current + sep + 'ngsw-bypass=true';
          }
        }
      }
      return throwError(() => error);
    })
  );
};
