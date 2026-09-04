// useConnectionState — convenience hook that returns a derived label +
// color for the topbar badge. Combines health check + SSE state.

import { useHealth } from '@/app/api/health';
import { useConnectionStore } from '@/app/store/connection';

export interface ConnectionState {
  label: string;
  color: string; // CSS var or hex
  claudeAvailable: boolean;
}

export function useConnectionState(): ConnectionState {
  const health = useHealth();
  const sseStatus = useConnectionStore((s) => s.status);

  const runtime = health.data?.agentRuntime?.runtime;
  const claudeAvailable = runtime?.available ?? health.data?.claude?.available ?? false;

  if (health.isError) {
    return { label: '后端离线', color: 'var(--pink)', claudeAvailable: false };
  }
  if (!claudeAvailable) {
    return { label: `${runtime?.name ?? 'Agent CLI'} 未就绪`, color: 'var(--orange)', claudeAvailable: false };
  }
  if (sseStatus === 'reconnecting' || sseStatus === 'error') {
    return { label: '重连中', color: 'var(--orange)', claudeAvailable };
  }
  if (sseStatus === 'connecting' || sseStatus === 'idle') {
    return { label: '连接中', color: 'var(--muted)', claudeAvailable };
  }
  return { label: `${runtime?.name ?? 'Agent CLI'} 在线`, color: 'var(--accent)', claudeAvailable };
}
