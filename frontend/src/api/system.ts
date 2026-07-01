import { http } from './client';

export interface ShutdownResponse {
  shuttingDown: boolean;
}

export function requestShutdown() {
  return http.post<ShutdownResponse>('/api/system/shutdown');
}
