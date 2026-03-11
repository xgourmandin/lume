"use client";

import React from 'react';
import { motion } from 'framer-motion';
import { Layers, CheckCircle2, AlertTriangle, XCircle, Clock } from 'lucide-react';
import type { Workspace, Layer, SyncStatus } from '@/types';

// ---------------------------------------------------------------------------
// Status helpers
// ---------------------------------------------------------------------------

const statusConfig: Record<SyncStatus, { label: string; icon: React.ReactNode; pill: string; dot: string }> = {
  clean: {
    label: 'Clean',
    icon: <CheckCircle2 className="w-3.5 h-3.5" />,
    pill: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20',
    dot: 'bg-emerald-400',
  },
  drifted: {
    label: 'Drifted',
    icon: <AlertTriangle className="w-3.5 h-3.5" />,
    pill: 'bg-amber-500/10 text-amber-400 border-amber-500/20',
    dot: 'bg-amber-400',
  },
  error: {
    label: 'Error',
    icon: <XCircle className="w-3.5 h-3.5" />,
    pill: 'bg-red-500/10 text-red-400 border-red-500/20',
    dot: 'bg-red-400',
  },
};

function StatusBadge({ status }: { status: SyncStatus }) {
  const cfg = statusConfig[status] ?? statusConfig.error;
  return (
    <span className={`flex items-center gap-1 text-[10px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded-full border ${cfg.pill}`}>
      {cfg.icon}
      {cfg.label}
    </span>
  );
}

function formatDate(iso: string): string {
  if (!iso) return '—';
  try {
    return new Intl.DateTimeFormat(undefined, {
      dateStyle: 'medium',
      timeStyle: 'short',
    }).format(new Date(iso));
  } catch {
    return iso;
  }
}

// ---------------------------------------------------------------------------
// LayerRow
// ---------------------------------------------------------------------------

interface LayerRowProps {
  layer: Layer;
  isSelected: boolean;
  onToggle: () => void;
}

const LayerRow: React.FC<LayerRowProps> = ({ layer, isSelected, onToggle }) => (
  <button
    onClick={onToggle}
    className={`w-full text-left flex items-center gap-3 px-3 py-2.5 rounded-xl border transition-all
      ${isSelected
        ? 'bg-blue-500/10 border-blue-500/30 ring-1 ring-inset ring-blue-500/20'
        : 'bg-white/5 border-white/5 hover:bg-white/10 hover:border-white/10'}`}
  >
    {/* Checkbox indicator */}
    <span className={`w-4 h-4 rounded border flex items-center justify-center shrink-0 transition-all
      ${isSelected ? 'bg-blue-500 border-blue-500' : 'border-white/20 bg-transparent'}`}>
      {isSelected && (
        <svg className="w-2.5 h-2.5 text-white" viewBox="0 0 10 10" fill="none">
          <path d="M1.5 5L4 7.5L8.5 2.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
        </svg>
      )}
    </span>
    <span className={`w-2 h-2 rounded-full shrink-0 ${statusConfig[layer.status]?.dot ?? 'bg-slate-400'}`} />
    <div className="min-w-0 flex-1">
      <p className="text-sm font-medium text-white/90 truncate">{layer.name || layer.id}</p>
      <p className="text-[10px] text-white/30 flex items-center gap-1 mt-0.5">
        <Clock className="w-2.5 h-2.5" />
        {formatDate(layer.last_sync)}
      </p>
    </div>
    <StatusBadge status={layer.status} />
  </button>
);

// ---------------------------------------------------------------------------
// WorkspaceLayers
// ---------------------------------------------------------------------------

interface WorkspaceLayersProps {
  workspace: Workspace;
  /** Set of selected layer IDs; empty set or null means "All layers". */
  selectedLayerIds: Set<string>;
  onToggleLayer: (layerId: string) => void;
  onClearSelection: () => void;
}

export const WorkspaceLayers: React.FC<WorkspaceLayersProps> = ({
  workspace,
  selectedLayerIds,
  onToggleLayer,
  onClearSelection,
}) => {
  const layers = workspace.layers ?? [];
  const hasSelection = selectedLayerIds.size > 0;

  const counts = layers.reduce(
    (acc, l) => {
      acc[l.status] = (acc[l.status] ?? 0) + 1;
      return acc;
    },
    {} as Record<SyncStatus, number>,
  );

  return (
    <motion.div
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      className="p-5 bg-white/5 border border-white/10 rounded-2xl space-y-4"
    >
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Layers className="w-4 h-4 text-blue-400" />
          <h3 className="text-sm font-semibold text-white">Layers</h3>
          <span className="text-[10px] font-mono text-white/30">{workspace.id}</span>
        </div>
        <StatusBadge status={workspace.status} />
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-3 gap-2">
        {(['clean', 'drifted', 'error'] as SyncStatus[]).map(s => (
          <div key={s} className="text-center bg-white/5 rounded-lg py-2">
            <p className={`text-lg font-bold ${s === 'clean' ? 'text-emerald-300' : s === 'drifted' ? 'text-amber-300' : 'text-red-300'}`}>
              {counts[s] ?? 0}
            </p>
            <p className="text-[10px] text-white/40 capitalize">{s}</p>
          </div>
        ))}
      </div>

      {/* Active filter badge */}
      {hasSelection && (
        <div className="flex items-center gap-2 px-3 py-2 bg-blue-500/10 border border-blue-500/20 rounded-xl">
          <span className="text-[11px] text-blue-300 flex-1">
            Filtering: {selectedLayerIds.size} layer{selectedLayerIds.size > 1 ? 's' : ''}
          </span>
          <button
            onClick={onClearSelection}
            className="text-[10px] text-blue-400 hover:text-white transition-colors underline"
          >
            Clear
          </button>
        </div>
      )}

      {/* Layer list */}
      {layers.length === 0 ? (
        <p className="text-xs text-white/30 text-center py-3">No layers synced yet.</p>
      ) : (
        <div className="space-y-1.5">
          {/* "All layers" option */}
          <button
            onClick={onClearSelection}
            className={`w-full text-left flex items-center gap-2 px-3 py-2 rounded-xl border text-sm transition-all
              ${!hasSelection
                ? 'bg-blue-500/10 border-blue-500/30 text-white ring-1 ring-inset ring-blue-500/20'
                : 'bg-white/5 border-white/5 hover:bg-white/10 text-white/60 hover:text-white'}`}
          >
            <Layers className="w-3.5 h-3.5 shrink-0" />
            <span className="font-medium">All layers (merged)</span>
          </button>
          {layers.map(layer => (
            <LayerRow
              key={layer.id}
              layer={layer}
              isSelected={selectedLayerIds.has(layer.id)}
              onToggle={() => onToggleLayer(layer.id)}
            />
          ))}
        </div>
      )}

      {/* Workspace last sync */}
      <p className="text-[10px] text-white/25 flex items-center gap-1 pt-1">
        <Clock className="w-2.5 h-2.5" />
        Workspace last synced: {formatDate(workspace.last_sync)}
      </p>
    </motion.div>
  );
};

