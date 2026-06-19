import { proxyBackend } from '@/lib/backend';

export async function POST() {
  return proxyBackend('/api/v1/hierarchy/sync-all', { method: 'POST' });
}
