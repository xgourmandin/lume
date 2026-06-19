import { NextRequest } from 'next/server';
import { proxyBackend } from '@/lib/backend';

export async function GET(
  _request: NextRequest,
  { params }: { params: Promise<{ layerId: string; tfWorkspaceId: string }> },
) {
  const { layerId, tfWorkspaceId } = await params;
  return proxyBackend(
    `/api/v1/drift/${encodeURIComponent(layerId)}/${encodeURIComponent(tfWorkspaceId)}`,
  );
}
