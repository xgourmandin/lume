import { NextResponse } from 'next/server';
import { BACKEND_URL, backendAuthHeaders } from '@/lib/backend';

export async function GET() {
  try {
    const res = await fetch(`${BACKEND_URL}/api/v1/hierarchy`, {
      headers: { 'Content-Type': 'application/json', ...(await backendAuthHeaders()) },
      cache: 'no-store',
    });
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    return NextResponse.json({ error: message }, { status: 502 });
  }
}

