import { Injectable, inject } from '@angular/core';
import { HttpClient, HttpEvent, HttpRequest } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import { Attachment } from '../models/project.model';
import { ApiService } from './api.service';

@Injectable({
  providedIn: 'root',
})
export class AttachmentService {
  private readonly http = inject(HttpClient);
  private readonly api = inject(ApiService);
  private readonly baseUrl = environment.apiBaseUrl;

  upload(projectId: string, file: File, itemId?: string): Observable<HttpEvent<Attachment>> {
    const formData = new FormData();
    formData.append('file', file, file.name);
    if (itemId) {
      formData.append('itemId', itemId);
    }

    const req = new HttpRequest(
      'POST',
      `${this.baseUrl}/projects/${projectId}/attachments`,
      formData,
      {
        reportProgress: true,
      }
    );
    return this.http.request<Attachment>(req);
  }

  list(projectId: string, itemId?: string): Observable<Attachment[]> {
    return this.api.get<Attachment[]>(`/projects/${projectId}/attachments`, {
      itemId,
    });
  }

  delete(projectId: string, attachmentId: string): Observable<{ status: string }> {
    return this.api.delete<{ status: string }>(
      `/projects/${projectId}/attachments/${attachmentId}`
    );
  }
}
