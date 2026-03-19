"use client";

import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Building2, Folder, Briefcase, Cpu, X, Hash, Link2, Layers, Tag, CheckCircle2, AlertTriangle, XCircle, Clock, GitBranch } from 'lucide-react';
import { Organization, Folder as GCPFolder, Project, Resource, Layer, TerraformWorkspace, SyncStatus } from '../types';

export type SelectedNode =
  | { type: 'org'; data: Organization }
  | { type: 'folder'; data: GCPFolder }
  | { type: 'project'; data: Project }
  | { type: 'resource'; data: Resource }
  | { type: 'layer'; data: Layer }
  | { type: 'tf_workspace'; data: TerraformWorkspace };

interface DetailPanelProps {
  node: SelectedNode | null;
  onClose: () => void;
}

const Badge: React.FC<{ label: string; value: string; icon?: React.ReactNode; mono?: boolean }> = ({
  label,
  value,
  icon,
  mono,
}) => (
  <div className="flex flex-col gap-1">
    <span className="text-[10px] uppercase tracking-widest text-white/30 font-semibold">{label}</span>
    <div className="flex items-center gap-2 bg-white/5 rounded-lg px-3 py-2 border border-white/5">
      {icon && <span className="text-white/40 shrink-0">{icon}</span>}
      <span className={`text-sm text-white/80 break-all ${mono ? 'font-mono' : 'font-medium'}`}>{value}</span>
    </div>
  </div>
);

const SectionTitle: React.FC<{ children: React.ReactNode }> = ({ children }) => (
  <p className="text-[10px] uppercase tracking-widest text-white/30 font-semibold mt-4 mb-2">{children}</p>
);

function OrgDetail({ data }: { data: Organization }) {
  const totalFolders = (folders: GCPFolder[] | undefined): number =>
    (folders ?? []).reduce((acc, f) => acc + 1 + totalFolders(f.folders), 0);

  const totalProjects = (node: Organization | GCPFolder): number => {
    const direct = (node.projects ?? []).length;
    const nested = ((node as Organization | GCPFolder).folders ?? []).reduce(
      (acc: number, f: GCPFolder) => acc + totalProjects(f),
      0
    );
    return direct + nested;
  };

  return (
    <div className="space-y-3">
      <Badge label="Organization ID" value={data.id} icon={<Hash className="w-3.5 h-3.5" />} mono />
      <Badge label="Display Name" value={data.display_name} icon={<Tag className="w-3.5 h-3.5" />} />
      <div className="grid grid-cols-2 gap-3 mt-2">
        <div className="bg-blue-500/10 border border-blue-500/20 rounded-xl p-4 text-center">
          <p className="text-2xl font-bold text-blue-300">{totalFolders(data.folders)}</p>
          <p className="text-xs text-white/40 mt-1">Folders</p>
        </div>
        <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-xl p-4 text-center">
          <p className="text-2xl font-bold text-emerald-300">{totalProjects(data)}</p>
          <p className="text-xs text-white/40 mt-1">Projects</p>
        </div>
      </div>
    </div>
  );
}

function countFolders(folder: GCPFolder): number {
  return (folder.folders ?? []).reduce((acc, f) => acc + 1 + countFolders(f), 0);
}

function countProjects(folder: GCPFolder): number {
  const direct = (folder.projects ?? []).length;
  const nested = (folder.folders ?? []).reduce((acc, f) => acc + countProjects(f), 0);
  return direct + nested;
}

function FolderDetail({ data }: { data: GCPFolder }) {
  return (
    <div className="space-y-3">
      <Badge label="Folder ID" value={data.id} icon={<Hash className="w-3.5 h-3.5" />} mono />
      <Badge label="Display Name" value={data.display_name} icon={<Tag className="w-3.5 h-3.5" />} />
      <Badge label="Parent" value={data.parent} icon={<Link2 className="w-3.5 h-3.5" />} mono />
      <div className="grid grid-cols-2 gap-3 mt-2">
        <div className="bg-amber-500/10 border border-amber-500/20 rounded-xl p-4 text-center">
          <p className="text-2xl font-bold text-amber-300">{countFolders(data)}</p>
          <p className="text-xs text-white/40 mt-1">Sub-Folders</p>
        </div>
        <div className="bg-emerald-500/10 border border-emerald-500/20 rounded-xl p-4 text-center">
          <p className="text-2xl font-bold text-emerald-300">{countProjects(data)}</p>
          <p className="text-xs text-white/40 mt-1">Projects</p>
        </div>
      </div>
    </div>
  );
}

function ProjectDetail({ data }: { data: Project }) {
  return (
    <div className="space-y-3">
      <Badge label="Project ID" value={data.project_id} icon={<Hash className="w-3.5 h-3.5" />} mono />
      <Badge label="Display Name" value={data.display_name} icon={<Tag className="w-3.5 h-3.5" />} />
      <Badge label="Internal ID" value={data.id} icon={<Hash className="w-3.5 h-3.5" />} mono />
      <Badge label="Parent" value={data.parent} icon={<Link2 className="w-3.5 h-3.5" />} mono />
      {data.layer_id && <Badge label="Layer" value={data.layer_id} icon={<Layers className="w-3.5 h-3.5" />} mono />}
      {data.workspace_id && <Badge label="TF Workspace" value={data.workspace_id} icon={<GitBranch className="w-3.5 h-3.5" />} mono />}

      {data.resources && data.resources.length > 0 && (
        <>
          <SectionTitle>Resources ({data.resources.length})</SectionTitle>
          <div className="space-y-2">
            {data.resources.map((res) => (
              <div
                key={res.address}
                className="flex items-start gap-3 bg-white/5 border border-white/5 rounded-lg px-3 py-2.5"
              >
                <Cpu className="w-4 h-4 text-slate-400 mt-0.5 shrink-0" />
                <div className="min-w-0">
                  <p className="text-sm font-medium text-white/80">{res.name}</p>
                  <p className="text-xs font-mono text-white/30 truncate">{res.type}</p>
                  <p className="text-xs font-mono text-white/20 truncate">{res.address}</p>
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}

function ResourceDetail({ data }: { data: Resource }) {
  return (
    <div className="space-y-3">
      <Badge label="Name" value={data.name} icon={<Tag className="w-3.5 h-3.5" />} />
      <Badge label="Resource Type" value={data.type} icon={<Layers className="w-3.5 h-3.5" />} mono />
      <Badge label="Address" value={data.address} icon={<Link2 className="w-3.5 h-3.5" />} mono />
      <Badge label="ID" value={data.id} icon={<Hash className="w-3.5 h-3.5" />} mono />
      {data.layer_id && <Badge label="Layer" value={data.layer_id} icon={<Layers className="w-3.5 h-3.5" />} mono />}
      {data.workspace_id && <Badge label="TF Workspace" value={data.workspace_id} icon={<GitBranch className="w-3.5 h-3.5" />} mono />}
    </div>
  );
}

function TfWorkspaceDetail({ data }: { data: TerraformWorkspace }) {
  const colorClass = statusColors[data.status] ?? statusColors.error;
  return (
    <div className="space-y-3">
      <Badge label="Workspace Name" value={data.id} icon={<GitBranch className="w-3.5 h-3.5" />} mono />
      <Badge label="Layer" value={data.layer_id} icon={<Layers className="w-3.5 h-3.5" />} mono />
      <div className="flex flex-col gap-1">
        <span className="text-[10px] uppercase tracking-widest text-white/30 font-semibold">Status</span>
        <div className={`flex items-center gap-2 rounded-lg px-3 py-2 border ${colorClass}`}>
          {statusIcons[data.status]}
          <span className="text-sm font-semibold capitalize">{data.status}</span>
        </div>
      </div>
      <div className="flex flex-col gap-1">
        <span className="text-[10px] uppercase tracking-widest text-white/30 font-semibold">Last Sync</span>
        <div className="flex items-center gap-2 bg-white/5 rounded-lg px-3 py-2 border border-white/5">
          <Clock className="w-3.5 h-3.5 text-white/40 shrink-0" />
          <span className="text-sm text-white/80 font-mono">{formatDate(data.last_sync)}</span>
        </div>
      </div>
      <div className="mt-2 p-3 bg-violet-500/5 border border-violet-500/15 rounded-xl">
        <p className="text-[11px] text-violet-300/70 leading-relaxed">
          This is a <span className="font-semibold text-violet-300">Terraform workspace</span> — a named environment
          within the <span className="font-mono">{data.layer_id}</span> layer. All resources parsed from its state
          file carry <span className="font-mono">workspace_id: &quot;{data.id}&quot;</span>.
        </p>
      </div>
    </div>
  );
}

const statusColors: Record<SyncStatus, string> = {
  clean:   'text-emerald-400 bg-emerald-400/10 border-emerald-400/20',
  drifted: 'text-amber-400 bg-amber-400/10 border-amber-400/20',
  error:   'text-red-400 bg-red-400/10 border-red-400/20',
};

const statusIcons: Record<SyncStatus, React.ReactNode> = {
  clean:   <CheckCircle2 className="w-4 h-4" />,
  drifted: <AlertTriangle className="w-4 h-4" />,
  error:   <XCircle className="w-4 h-4" />,
};

function formatDate(iso: string): string {
  if (!iso) return '—';
  try {
    return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(iso));
  } catch {
    return iso;
  }
}

function LayerDetail({ data }: { data: Layer }) {
  const colorClass = statusColors[data.status] ?? statusColors.error;
  return (
    <div className="space-y-3">
      <Badge label="Layer ID" value={data.id} icon={<Hash className="w-3.5 h-3.5" />} mono />
      <Badge label="Name" value={data.name || data.id} icon={<Tag className="w-3.5 h-3.5" />} />
      <div className="flex flex-col gap-1">
        <span className="text-[10px] uppercase tracking-widest text-white/30 font-semibold">Status</span>
        <div className={`flex items-center gap-2 rounded-lg px-3 py-2 border ${colorClass}`}>
          {statusIcons[data.status]}
          <span className="text-sm font-semibold capitalize">{data.status}</span>
        </div>
      </div>
      <div className="flex flex-col gap-1">
        <span className="text-[10px] uppercase tracking-widest text-white/30 font-semibold">Last Sync</span>
        <div className="flex items-center gap-2 bg-white/5 rounded-lg px-3 py-2 border border-white/5">
          <Clock className="w-3.5 h-3.5 text-white/40 shrink-0" />
          <span className="text-sm text-white/80 font-mono">{formatDate(data.last_sync)}</span>
        </div>
      </div>
    </div>
  );
}

const typeConfig = {
  org: {
    icon: <Building2 className="w-5 h-5 text-blue-400" />,
    bg: 'bg-blue-500/10',
    border: 'border-blue-500/30',
    label: 'Organization',
    color: 'text-blue-400',
  },
  folder: {
    icon: <Folder className="w-5 h-5 text-amber-400" />,
    bg: 'bg-amber-500/10',
    border: 'border-amber-500/30',
    label: 'Folder',
    color: 'text-amber-400',
  },
  project: {
    icon: <Briefcase className="w-5 h-5 text-emerald-400" />,
    bg: 'bg-emerald-500/10',
    border: 'border-emerald-500/30',
    label: 'Project',
    color: 'text-emerald-400',
  },
  resource: {
    icon: <Cpu className="w-5 h-5 text-slate-400" />,
    bg: 'bg-slate-500/10',
    border: 'border-slate-500/30',
    label: 'Resource',
    color: 'text-slate-400',
  },
  layer: {
    icon: <Layers className="w-5 h-5 text-purple-400" />,
    bg: 'bg-purple-500/10',
    border: 'border-purple-500/30',
    label: 'Layer',
    color: 'text-purple-400',
  },
  tf_workspace: {
    icon: <GitBranch className="w-5 h-5 text-violet-400" />,
    bg: 'bg-violet-500/10',
    border: 'border-violet-500/30',
    label: 'TF Workspace',
    color: 'text-violet-400',
  },
};

export const DetailPanel: React.FC<DetailPanelProps> = ({ node, onClose }) => {
  return (
    <AnimatePresence>
      {node && (
        <motion.div
          key={`${node.type}-${
            node.type === 'resource' ? node.data.address
            : node.type === 'tf_workspace' ? `${node.data.layer_id}-${node.data.id}`
            : node.data.id
          }`}
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          exit={{ opacity: 0, y: 12 }}
          transition={{ duration: 0.2, ease: 'easeOut' }}
          className="p-5 bg-slate-900/60 backdrop-blur-xl border border-white/10 rounded-2xl shadow-2xl"
        >
          {/* Header */}
          <div className="flex items-center justify-between mb-5">
            <div className="flex items-center gap-3">
              <div className={`p-2 rounded-lg ${typeConfig[node.type].bg} border ${typeConfig[node.type].border}`}>
                {typeConfig[node.type].icon}
              </div>
              <div>
                <p className={`text-[10px] uppercase tracking-widest font-semibold ${typeConfig[node.type].color}`}>
                  {typeConfig[node.type].label}
                </p>
                <h3 className="text-base font-semibold text-white leading-tight">
                  {node.type === 'layer'
                    ? (node.data.name || node.data.id)
                    : node.type === 'tf_workspace'
                    ? `${node.data.layer_id} / ${node.data.id}`
                    : node.type === 'resource'
                    ? node.data.name
                    : node.data.display_name}
                </h3>
              </div>
            </div>
            <button
              onClick={onClose}
              className="p-1.5 rounded-lg hover:bg-white/10 transition-colors text-white/40 hover:text-white/80"
            >
              <X className="w-4 h-4" />
            </button>
          </div>

          {/* Content */}
          {node.type === 'org' && <OrgDetail data={node.data as Organization} />}
          {node.type === 'folder' && <FolderDetail data={node.data as GCPFolder} />}
          {node.type === 'project' && <ProjectDetail data={node.data as Project} />}
          {node.type === 'resource' && <ResourceDetail data={node.data as Resource} />}
          {node.type === 'layer' && <LayerDetail data={node.data as Layer} />}
          {node.type === 'tf_workspace' && <TfWorkspaceDetail data={node.data as TerraformWorkspace} />}
        </motion.div>
      )}
    </AnimatePresence>
  );
};

