import { NextRequest, NextResponse } from 'next/server';
import { BACKEND_URL, backendAuthHeaders } from '@/lib/backend';

export async function POST(request: NextRequest) {
  try {
    const body = await request.json();
    const res = await fetch(`${BACKEND_URL}/api/v1/hierarchy/sync`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...(await backendAuthHeaders()) },
      body: JSON.stringify(body),
    });
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    return NextResponse.json({ error: message }, { status: 502 });
  }
}

