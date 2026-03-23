"use client";

import React, { useEffect, useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { AlertTriangle, XCircle, CheckCircle2, Clock, Plus, Minus, RefreshCw, ChevronDown, ChevronUp } from 'lucide-react';
import type { DriftResult } from '@/types';
import { fetchDriftResult } from '@/lib/api';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function formatDate(iso: string): string {
  if (!iso) return '—';
  try {
    return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(
      new Date(iso),
    );
  } catch {
    return iso;
  }
}

function timeAgo(iso: string): string {
  if (!iso) return '';
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.floor(hours / 24)}d ago`;
}

// ---------------------------------------------------------------------------
// Sub-components
// ---------------------------------------------------------------------------

function ChangeCounter({
  count,
  label,
  colorClass,
  bgClass,
  icon,
}: {
  count: number;
  label: string;
  colorClass: string;
  bgClass: string;
  icon: React.ReactNode;
}) {
  return (
    <div className={`flex flex-col items-center gap-1.5 rounded-xl px-4 py-3 border ${bgClass}`}>
      <div className={`flex items-center gap-1 ${colorClass}`}>
        {icon}
        <span className="text-2xl font-bold leading-none">{count}</span>
      </div>
      <span className="text-[10px] uppercase tracking-widest text-white/40 font-semibold">
        {label}
      </span>
    </div>
  );
}

// ---------------------------------------------------------------------------
// DriftReport
// ---------------------------------------------------------------------------

interface DriftReportProps {
  workspaceId: string;
  layerId: string;
  tfWorkspaceId: string;
}

export const DriftReport: React.FC<DriftReportProps> = ({
  workspaceId,
  layerId,
  tfWorkspaceId,
}) => {
  const [result, setResult] = useState<DriftResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [errorExpanded, setErrorExpanded] = useState(false);

  const load = React.useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchDriftResult(workspaceId, layerId, tfWorkspaceId);
      setResult(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [workspaceId, layerId, tfWorkspaceId]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <div className="mt-4">
      {/* Section label */}
      <p className="text-[10px] uppercase tracking-widest text-white/30 font-semibold mb-2">
        Drift Report
      </p>

      <AnimatePresence mode="wait">
        {/* Loading skeleton */}
        {loading && (
          <motion.div
            key="loading"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="rounded-xl border border-white/8 bg-white/3 p-4 space-y-3 animate-pulse"
          >
            <div className="h-8 w-2/3 bg-white/5 rounded-lg" />
            <div className="grid grid-cols-3 gap-2">
              <div className="h-16 bg-white/5 rounded-xl" />
              <div className="h-16 bg-white/5 rounded-xl" />
              <div className="h-16 bg-white/5 rounded-xl" />
            </div>
          </motion.div>
        )}

        {/* Fetch error (no result available) */}
        {!loading && error && (
          <motion.div
            key="fetch-error"
            initial={{ opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            className="flex items-center justify-between gap-3 rounded-xl border border-white/8 bg-white/3 px-4 py-3"
          >
            <p className="text-sm text-white/40">{error}</p>
            <button
              onClick={load}
              className="text-white/30 hover:text-white/60 transition-colors"
              title="Retry"
            >
              <RefreshCw className="w-3.5 h-3.5" />
            </button>
          </motion.div>
        )}

        {/* Result */}
        {!loading && result && (
          <motion.div
            key="result"
            initial={{ opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0 }}
            className="rounded-xl border border-white/8 bg-white/3 overflow-hidden"
          >
            {/* Status banner */}
            {result.status === 'clean' && (
              <div className="flex items-center gap-3 px-4 py-3 bg-emerald-500/10 border-b border-emerald-500/15">
                <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0" />
                <div>
                  <p className="text-sm font-semibold text-emerald-300">No drift detected</p>
                  <p className="text-[11px] text-emerald-400/60">
                    Infrastructure matches the desired state.
                  </p>
                </div>
              </div>
            )}

            {result.status === 'drifted' && (
              <div className="flex items-center gap-3 px-4 py-3 bg-amber-500/10 border-b border-amber-500/15">
                <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0" />
                <div>
                  <p className="text-sm font-semibold text-amber-300">Drift detected</p>
                  <p className="text-[11px] text-amber-400/60">
                    Manual changes found — code and cloud are out of sync.
                  </p>
                </div>
              </div>
            )}

            {result.status === 'error' && (
              <div className="flex items-center gap-3 px-4 py-3 bg-red-500/10 border-b border-red-500/15">
                <XCircle className="w-5 h-5 text-red-400 shrink-0" />
                <div>
                  <p className="text-sm font-semibold text-red-300">Plan execution failed</p>
                  <p className="text-[11px] text-red-400/60">
                    The scanner could not run successfully.
                  </p>
                </div>
              </div>
            )}

            <div className="p-4 space-y-4">
              {/* Change counters — shown for clean and drifted */}
              {result.status !== 'error' && (
                <div className="grid grid-cols-3 gap-2">
                  <ChangeCounter
                    count={result.add_count}
                    label="to add"
                    colorClass="text-emerald-400"
                    bgClass="bg-emerald-500/8 border-emerald-500/15"
                    icon={<Plus className="w-4 h-4" />}
                  />
                  <ChangeCounter
                    count={result.change_count}
                    label="to change"
                    colorClass="text-amber-400"
                    bgClass="bg-amber-500/8 border-amber-500/15"
                    icon={<RefreshCw className="w-4 h-4" />}
                  />
                  <ChangeCounter
                    count={result.destroy_count}
                    label="to destroy"
                    colorClass="text-red-400"
                    bgClass="bg-red-500/8 border-red-500/15"
                    icon={<Minus className="w-4 h-4" />}
                  />
                </div>
              )}

              {/* Error details — collapsible */}
              {result.status === 'error' && result.error_message && (
                <div className="rounded-lg border border-red-500/20 bg-red-500/5 overflow-hidden">
                  <button
                    onClick={() => setErrorExpanded(v => !v)}
                    className="w-full flex items-center justify-between px-3 py-2 hover:bg-red-500/5 transition-colors"
                  >
                    <span className="text-[11px] font-semibold text-red-400 uppercase tracking-wider">
                      Error details
                    </span>
                    {errorExpanded ? (
                      <ChevronUp className="w-3.5 h-3.5 text-red-400/60" />
                    ) : (
                      <ChevronDown className="w-3.5 h-3.5 text-red-400/60" />
                    )}
                  </button>
                  <AnimatePresence>
                    {errorExpanded && (
                      <motion.div
                        initial={{ height: 0, opacity: 0 }}
                        animate={{ height: 'auto', opacity: 1 }}
                        exit={{ height: 0, opacity: 0 }}
                        transition={{ duration: 0.2 }}
                        className="overflow-hidden"
                      >
                        <pre className="px-3 pb-3 text-[10px] font-mono text-red-300/70 whitespace-pre-wrap break-all leading-relaxed max-h-48 overflow-y-auto">
                          {result.error_message}
                        </pre>
                      </motion.div>
                    )}
                  </AnimatePresence>
                </div>
              )}

              {/* Scanned at */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5 text-[11px] text-white/30">
                  <Clock className="w-3 h-3" />
                  <span>Scanned {timeAgo(result.scanned_at)}</span>
                </div>
                <span className="text-[10px] font-mono text-white/20">
                  {formatDate(result.scanned_at)}
                </span>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

