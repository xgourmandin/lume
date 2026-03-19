"use client";

import React, { useState, useRef, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Layers, CheckCircle2, AlertTriangle, XCircle, Clock, ChevronDown, GitBranch, Search } from 'lucide-react';
import type { Workspace, Layer, TerraformWorkspace, SyncStatus } from '@/types';

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
    return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(iso));
  } catch {
    return iso;
  }
}

// ---------------------------------------------------------------------------
// WorkspaceComboBox — compact multi-select combo box for a layer's workspaces
// ---------------------------------------------------------------------------

interface WorkspaceComboBoxProps {
  workspaces: TerraformWorkspace[];
  selectedIds: Set<string>;
  onToggle: (ws: TerraformWorkspace) => void;
}

const WorkspaceComboBox: React.FC<WorkspaceComboBoxProps> = ({ workspaces, selectedIds, onToggle }) => {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const containerRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);

  // Close on outside click
  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
        setQuery('');
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  // Close on Escape
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') { setOpen(false); setQuery(''); }
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, [open]);

  // Auto-focus search input when dropdown opens (only when search is shown)
  useEffect(() => {
    if (open && workspaces.length > 5) {
      setTimeout(() => searchRef.current?.focus(), 50);
    }
  }, [open, workspaces.length]);

  const filtered = query.trim()
    ? workspaces.filter(ws => ws.id.toLowerCase().includes(query.toLowerCase()))
    : workspaces;

  const selectedInLayer = workspaces.filter(ws => selectedIds.has(ws.id));
  const selectedCount = selectedInLayer.length;
  const allSelected = selectedCount === workspaces.length;
  const isFiltered = selectedCount > 0 && !allSelected;

  // Trigger label
  let triggerLabel: string;
  if (selectedCount === 0 || allSelected) {
    triggerLabel = 'All workspaces';
  } else if (selectedCount === 1) {
    triggerLabel = selectedInLayer[0].id;
  } else {
    triggerLabel = `${selectedCount} of ${workspaces.length} selected`;
  }

  return (
    <div ref={containerRef} className="relative">
      {/* Trigger button */}
      <button
        onClick={() => setOpen(o => !o)}
        className={`w-full flex items-center gap-1.5 px-2 py-1.5 rounded-lg border text-[11px] transition-all
          ${open
            ? 'bg-white/10 border-violet-500/40 ring-1 ring-inset ring-violet-500/20'
            : isFiltered
            ? 'bg-violet-500/10 border-violet-500/30 hover:bg-violet-500/15'
            : 'bg-white/5 border-white/10 hover:bg-white/8 hover:border-white/20'}`}
      >
        <GitBranch className={`w-3 h-3 shrink-0 ${isFiltered ? 'text-violet-400' : 'text-white/30'}`} />
        <span className={`flex-1 text-left font-mono truncate
          ${isFiltered ? 'text-violet-200' : 'text-white/45'}`}>
          {triggerLabel}
        </span>
        {isFiltered && (
          <span className="text-[9px] font-bold bg-violet-500/25 text-violet-300 border border-violet-500/30 px-1.5 py-0.5 rounded-full shrink-0">
            {selectedCount}
          </span>
        )}
        <ChevronDown className={`w-3 h-3 text-white/30 shrink-0 transition-transform duration-150 ${open ? 'rotate-180' : ''}`} />
      </button>

      {/* Dropdown */}
      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, y: -6, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -6, scale: 0.97 }}
            transition={{ duration: 0.13, ease: 'easeOut' }}
            className="absolute z-50 top-full mt-1.5 left-0 right-0 bg-slate-900 border border-white/12 rounded-xl shadow-2xl shadow-black/60 overflow-hidden"
          >
            {/* Search — shown when there are more than 5 workspaces */}
            {workspaces.length > 5 && (
              <div className="px-2 pt-2 pb-1.5 border-b border-white/8">
                <div className="flex items-center gap-1.5 bg-white/5 border border-white/10 rounded-lg px-2 py-1 focus-within:border-violet-500/40 transition-colors">
                  <Search className="w-3 h-3 text-white/30 shrink-0" />
                  <input
                    ref={searchRef}
                    type="text"
                    placeholder="Filter workspaces…"
                    value={query}
                    onChange={e => setQuery(e.target.value)}
                    className="flex-1 bg-transparent text-[11px] text-white placeholder:text-white/30 outline-none"
                  />
                </div>
              </div>
            )}

            {/* Quick actions */}
            <div className="flex items-center gap-1 px-2 pt-1.5 pb-1">
              <button
                onClick={() => workspaces.forEach(ws => { if (!selectedIds.has(ws.id)) onToggle(ws); })}
                className="text-[10px] text-violet-400 hover:text-violet-300 transition-colors px-1.5 py-0.5 rounded hover:bg-violet-500/10"
              >
                Select all
              </button>
              <span className="text-white/15 text-[10px]">·</span>
              <button
                onClick={() => workspaces.forEach(ws => { if (selectedIds.has(ws.id)) onToggle(ws); })}
                className="text-[10px] text-white/35 hover:text-white/60 transition-colors px-1.5 py-0.5 rounded hover:bg-white/5"
              >
                Clear
              </button>
            </div>

            {/* Options list */}
            <div className="max-h-52 overflow-y-auto py-1 px-1.5 space-y-0.5">
              {filtered.length === 0 ? (
                <p className="text-[11px] text-white/30 text-center py-3">No workspaces match</p>
              ) : (
                filtered.map(ws => {
                  const cfg = statusConfig[ws.status] ?? statusConfig.clean;
                  const sel = selectedIds.has(ws.id);
                  return (
                    <button
                      key={ws.id}
                      onClick={() => onToggle(ws)}
                      className={`w-full flex items-center gap-2 px-2 py-1.5 rounded-lg border transition-all text-left
                        ${sel
                          ? 'bg-violet-500/10 border-violet-500/30'
                          : 'bg-transparent border-transparent hover:bg-white/5 hover:border-white/8'}`}
                    >
                      {/* Checkbox */}
                      <span className={`w-3.5 h-3.5 rounded border flex items-center justify-center shrink-0 transition-all
                        ${sel ? 'bg-violet-500 border-violet-500' : 'border-white/20 bg-transparent'}`}>
                        {sel && (
                          <svg className="w-2 h-2 text-white" viewBox="0 0 10 10" fill="none">
                            <path d="M1.5 5L4 7.5L8.5 2.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
                          </svg>
                        )}
                      </span>

                      {/* Status dot */}
                      <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${cfg.dot}`} />

                      {/* Name */}
                      <span className={`text-[11px] font-mono flex-1 truncate
                        ${sel ? 'text-violet-200' : 'text-white/60'}`}>
                        {ws.id}
                      </span>

                      {/* Non-clean badge */}
                      {ws.status !== 'clean' && (
                        <span className={`text-[9px] font-semibold uppercase px-1.5 py-0.5 rounded-full border ${cfg.pill} shrink-0`}>
                          {ws.status}
                        </span>
                      )}
                    </button>
                  );
                })
              )}
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

// ---------------------------------------------------------------------------
// LayerRow — one layer with inline WorkspaceComboBox
// ---------------------------------------------------------------------------

interface LayerRowProps {
  layer: Layer;
  isLayerSelected: boolean;
  selectedWorkspaceIds: Set<string>;
  onToggleLayer: () => void;
  onToggleWorkspace: (ws: TerraformWorkspace) => void;
}

const LayerRow: React.FC<LayerRowProps> = ({
  layer,
  isLayerSelected,
  selectedWorkspaceIds,
  onToggleLayer,
  onToggleWorkspace,
}) => {
  const workspaces: TerraformWorkspace[] = layer.workspaces ?? [
    { id: 'default', layer_id: layer.id, status: layer.status, last_sync: layer.last_sync },
  ];

  const activeWsCount = workspaces.filter(w => selectedWorkspaceIds.has(w.id)).length;
  const isFiltered = activeWsCount > 0 && activeWsCount < workspaces.length;

  return (
    <div className={`rounded-xl border transition-all
      ${isLayerSelected ? 'bg-blue-500/10 border-blue-500/30' : 'bg-white/5 border-white/5'}`}
    >
      {/* Layer header row */}
      <div className="flex items-center gap-2 px-3 py-2.5">
        {/* Layer checkbox */}
        <button
          onClick={onToggleLayer}
          className={`w-4 h-4 rounded border flex items-center justify-center shrink-0 transition-all
            ${isLayerSelected ? 'bg-blue-500 border-blue-500' : 'border-white/20 bg-transparent hover:border-white/40'}`}
        >
          {isLayerSelected && (
            <svg className="w-2.5 h-2.5 text-white" viewBox="0 0 10 10" fill="none">
              <path d="M1.5 5L4 7.5L8.5 2.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          )}
        </button>

        <span className={`w-2 h-2 rounded-full shrink-0 ${statusConfig[layer.status]?.dot ?? 'bg-slate-400'}`} />

        <button onClick={onToggleLayer} className="min-w-0 flex-1 text-left">
          <p className={`text-sm font-medium truncate ${isLayerSelected ? 'text-white' : 'text-white/80'}`}>
            {layer.name || layer.id}
          </p>
          <p className="text-[10px] text-white/30 flex items-center gap-1 mt-0.5">
            <Clock className="w-2.5 h-2.5" />
            {formatDate(layer.last_sync)}
          </p>
        </button>

        {isFiltered && (
          <span className="text-[9px] font-bold bg-violet-500/20 text-violet-300 border border-violet-500/30 px-1.5 py-0.5 rounded-full shrink-0">
            {activeWsCount}w
          </span>
        )}

        <StatusBadge status={layer.status} />
      </div>

      {/* Workspace combo box */}
      <div className="px-3 pb-2.5">
        <WorkspaceComboBox
          workspaces={workspaces}
          selectedIds={selectedWorkspaceIds}
          onToggle={onToggleWorkspace}
        />
      </div>
    </div>
  );
};

// ---------------------------------------------------------------------------
// WorkspaceLayers
// ---------------------------------------------------------------------------

interface WorkspaceLayersProps {
  workspace: Workspace;
  /** Set of selected layer IDs; empty set = "All layers". */
  selectedLayerIds: Set<string>;
  /** Set of selected Terraform workspace names (e.g. "prod", "staging"). */
  selectedWorkspaceIds: Set<string>;
  onToggleLayer: (layerId: string) => void;
  onToggleWorkspace: (ws: TerraformWorkspace) => void;
  onClearSelection: () => void;
}

export const WorkspaceLayers: React.FC<WorkspaceLayersProps> = ({
  workspace,
  selectedLayerIds,
  selectedWorkspaceIds,
  onToggleLayer,
  onToggleWorkspace,
  onClearSelection,
}) => {
  const layers = workspace.layers ?? [];
  const hasLayerSelection = selectedLayerIds.size > 0;
  const hasWorkspaceSelection = selectedWorkspaceIds.size > 0;
  const hasSelection = hasLayerSelection || hasWorkspaceSelection;

  const counts = layers.reduce(
    (acc, l) => { acc[l.status] = (acc[l.status] ?? 0) + 1; return acc; },
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
          <h3 className="text-sm font-semibold text-white">Layers & Workspaces</h3>
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

      {/* Active filter summary */}
      {hasSelection && (
        <div className="flex items-center gap-2 px-3 py-2 bg-slate-800/60 border border-white/10 rounded-xl">
          <div className="flex-1 min-w-0 space-y-0.5">
            {hasLayerSelection && (
              <p className="text-[11px] text-blue-300 truncate">
                <span className="text-white/40">Layers: </span>
                {[...selectedLayerIds].join(', ')}
              </p>
            )}
            {hasWorkspaceSelection && (
              <p className="text-[11px] text-violet-300 truncate">
                <span className="text-white/40">Workspaces: </span>
                {[...selectedWorkspaceIds].join(', ')}
              </p>
            )}
          </div>
          <button
            onClick={onClearSelection}
            className="text-[10px] text-white/40 hover:text-white transition-colors underline shrink-0"
          >
            Clear
          </button>
        </div>
      )}

      {/* Layer list */}
      {layers.length === 0 ? (
        <p className="text-xs text-white/30 text-center py-3">No layers synced yet.</p>
      ) : (
        <div className="space-y-2">
          {/* "All" option */}
          <button
            onClick={onClearSelection}
            className={`w-full text-left flex items-center gap-2 px-3 py-2 rounded-xl border text-sm transition-all
              ${!hasSelection
                ? 'bg-blue-500/10 border-blue-500/30 text-white ring-1 ring-inset ring-blue-500/20'
                : 'bg-white/5 border-white/5 hover:bg-white/10 text-white/60 hover:text-white'}`}
          >
            <Layers className="w-3.5 h-3.5 shrink-0" />
            <span className="font-medium">All layers · all workspaces</span>
          </button>

          {layers.map(layer => (
            <LayerRow
              key={layer.id}
              layer={layer}
              isLayerSelected={selectedLayerIds.has(layer.id)}
              selectedWorkspaceIds={selectedWorkspaceIds}
              onToggleLayer={() => onToggleLayer(layer.id)}
              onToggleWorkspace={onToggleWorkspace}
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

