"use client";

import React, { useState, useEffect } from 'react';
import { X, RefreshCw, CheckCircle2, AlertTriangle, ChevronDown, ChevronUp } from 'lucide-react';
import type { SyncAllResult } from '@/types';

interface Props {
  open: boolean;
  onClose: () => void;
  onSync: () => Promise<SyncAllResult>;
  /** Called with the fresh hierarchy after a successful sync */
  onSuccess: (result: SyncAllResult) => void;
}

export function SyncAllModal({ open, onClose, onSync, onSuccess }: Props) {
  const [syncing, setSyncing] = useState(false);
  const [result, setResult] = useState<SyncAllResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [errorsExpanded, setErrorsExpanded] = useState(false);

  // Reset state every time the modal opens
  useEffect(() => {
    if (open) {
      setResult(null);
      setError(null);
      setErrorsExpanded(false);
      setSyncing(false);
    }
  }, [open]);

  if (!open) return null;

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSyncing(true);
    setError(null);
    setResult(null);
    try {
      const res = await onSync();
      setResult(res);
      onSuccess(res);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Sync failed');
    } finally {
      setSyncing(false);
    }
  }

  const hasErrors = (result?.errors?.length ?? 0) > 0;

  return (
    /* Backdrop */
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm"
      onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}
    >
      <div className="relative w-full max-w-md mx-4 bg-slate-900 border border-white/10 rounded-2xl shadow-2xl shadow-black/60 overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between px-6 pt-5 pb-4 border-b border-white/8">
          <div className="flex items-center gap-2">
            <RefreshCw className="w-4 h-4 text-emerald-400" />
            <h2 className="text-base font-semibold text-white">Sync All State Files</h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg text-white/40 hover:text-white hover:bg-white/8 transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <form onSubmit={handleSubmit} className="px-6 py-5 space-y-4">
          <p className="text-sm text-white/50">
            Every <code className="text-emerald-400 text-xs bg-emerald-400/10 px-1 rounded">.tfstate</code> file
            in the configured GCS bucket will be downloaded and merged into the hierarchy.
          </p>

          {/* Error banner */}
          {error && (
            <div className="flex items-start gap-2 px-3 py-2.5 bg-red-500/10 border border-red-500/20 rounded-lg">
              <AlertTriangle className="w-4 h-4 text-red-400 flex-shrink-0 mt-0.5" />
              <p className="text-sm text-red-300">{error}</p>
            </div>
          )}

          {/* Result summary */}
          {result && (
            <div className="space-y-2">
              <div className="flex items-center gap-2 px-3 py-2.5 bg-emerald-500/10 border border-emerald-500/20 rounded-lg">
                <CheckCircle2 className="w-4 h-4 text-emerald-400 flex-shrink-0" />
                <p className="text-sm text-emerald-300">
                  <span className="font-semibold">{result.synced}</span> synced
                  {result.failed > 0 && (
                    <>, <span className="font-semibold text-amber-400">{result.failed}</span> failed</>
                  )}
                </p>
              </div>

              {hasErrors && (
                <div className="border border-amber-500/20 rounded-lg overflow-hidden">
                  <button
                    type="button"
                    onClick={() => setErrorsExpanded(v => !v)}
                    className="w-full flex items-center justify-between px-3 py-2 bg-amber-500/10 text-amber-400 text-xs font-medium hover:bg-amber-500/15 transition-colors"
                  >
                    <span>{result.errors!.length} object error{result.errors!.length !== 1 ? 's' : ''}</span>
                    {errorsExpanded ? <ChevronUp className="w-3 h-3" /> : <ChevronDown className="w-3 h-3" />}
                  </button>
                  {errorsExpanded && (
                    <ul className="divide-y divide-white/5 max-h-36 overflow-y-auto">
                      {result.errors!.map((e, i) => (
                        <li key={i} className="px-3 py-2 space-y-0.5">
                          <p className="text-[11px] font-mono text-white/70 truncate">{e.object}</p>
                          <p className="text-[11px] text-red-400">{e.error}</p>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Actions */}
          <div className="flex items-center justify-end gap-3 pt-1">
            <button
              type="button"
              onClick={onClose}
              disabled={syncing}
              className="px-4 py-2 rounded-lg text-sm text-white/60 hover:text-white hover:bg-white/8 transition-colors disabled:opacity-50"
            >
              {result ? 'Close' : 'Cancel'}
            </button>
            {!result && (
              <button
                type="submit"
                disabled={syncing}
                className="flex items-center gap-2 px-4 py-2 bg-emerald-600 hover:bg-emerald-500 disabled:opacity-50 disabled:cursor-not-allowed rounded-lg text-sm font-medium text-white transition-colors active:scale-95"
              >
                <RefreshCw className={`w-4 h-4 ${syncing ? 'animate-spin' : ''}`} />
                {syncing ? 'Syncing…' : 'Sync All'}
              </button>
            )}
          </div>
        </form>
      </div>
    </div>
  );
}
