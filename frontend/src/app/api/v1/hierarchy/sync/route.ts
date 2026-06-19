import { NextRequest } from 'next/server';
import { proxyBackend } from '@/lib/backend';

export async function POST(request: NextRequest) {
  // Forward the raw body untouched — parsing/re-serializing here would throw an
  // uncaught SyntaxError on a malformed payload (a 500 instead of the backend's
  // own 400) and gains nothing, since the backend validates the JSON anyway.
  const body = await request.text();
  return proxyBackend('/api/v1/hierarchy/sync', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body,
  });
}
