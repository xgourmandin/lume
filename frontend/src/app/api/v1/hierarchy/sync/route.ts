import { NextRequest } from 'next/server';
import { proxyBackend } from '@/lib/backend';

export async function POST(request: NextRequest) {
  const body = await request.json();
  return proxyBackend('/api/v1/hierarchy/sync', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}
