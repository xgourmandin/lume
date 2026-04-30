import { NextRequest, NextResponse } from 'next/server';

const BACKEND_URL = process.env.API_URL?.replace(/\/$/, '') ?? 'http://localhost:3000';

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ layerId: string; tfWorkspaceId: string }> },
) {
  try {
    const { layerId, tfWorkspaceId } = await params;
    const res = await fetch(
      `${BACKEND_URL}/api/v1/drift/${encodeURIComponent(layerId)}/${encodeURIComponent(tfWorkspaceId)}`,
      { cache: 'no-store' },
    );
    const data = await res.json();
    return NextResponse.json(data, { status: res.status });
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    return NextResponse.json({ error: message }, { status: 502 });
  }
}

