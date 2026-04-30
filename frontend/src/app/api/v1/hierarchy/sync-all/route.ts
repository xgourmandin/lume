import { NextResponse } from 'next/server';

const BACKEND_URL = process.env.API_URL?.replace(/\/$/, '') ?? 'http://localhost:3000';

export async function POST() {
  try {
    const res = await fetch(`${BACKEND_URL}/api/v1/hierarchy/sync-all`, {
      method: 'POST',
    });
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    return NextResponse.json({ error: message }, { status: 502 });
  }
}

