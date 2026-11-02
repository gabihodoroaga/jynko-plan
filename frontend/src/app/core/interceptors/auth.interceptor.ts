import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { AuthService } from '../services/auth.service';
import { environment } from '../../../environments/environment';

/**
 * In production behind Google Cloud IAP, authentication headers are injected at the edge proxy.
 * In local development (environment.devMode), this interceptor attaches X-Dev-User-* headers.
 */
export const authInterceptor: HttpInterceptorFn = (req, next) => {
  const authService = inject(AuthService);
  if (environment.devMode) {
    const user = authService.currentUser();
    const cloned = req.clone({
      setHeaders: {
        'X-Dev-User-Id': user.user_id,
        'X-Dev-User-Email': user.email,
        'X-Dev-User-Name': user.name,
      },
    });
    return next(cloned);
  }
  return next(req);
};
