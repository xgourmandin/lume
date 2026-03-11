"use client";

import React, { useState, useMemo, useEffect, useRef } from 'react';
import { ChevronRight, ChevronDown, Folder, Briefcase, Building2, Cpu, ExternalLink, Search, X } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { List } from 'react-window';
import { Organization, Folder as GCPFolder, Project, Resource } from '../types';
import { SelectedNode } from './DetailPanel';
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

const RESOURCE_PAGE_SIZE = 10;
const VIRTUAL_THRESHOLD = 20;   // switch to react-window above this count
const VIRTUAL_ITEM_HEIGHT = 36; // px — matches py-1.5 + text-sm row
const VIRTUAL_MAX_HEIGHT = 360; // px — max height of the virtualised list

function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}

// ─── Layer filtering helpers ──────────────────────────────────────────────────

function filterResourcesByLayer(resources: Resource[], layers: Set<string>): Resource[] {
    return resources.filter(r => r.layer_id ? layers.has(r.layer_id) : true);
}

function filterProjectsByLayer(projects: Project[], layers: Set<string>): Project[] {
    return projects
        .map(p => {
            const filteredResources = filterResourcesByLayer(p.resources ?? [], layers);
            const projectMatches = p.layer_id ? layers.has(p.layer_id) : filteredResources.length > 0;
            if (!projectMatches && filteredResources.length === 0) return null;
            return { ...p, resources: filteredResources };
        })
        .filter(Boolean) as Project[];
}

function filterFoldersByLayer(folders: GCPFolder[], layers: Set<string>): GCPFolder[] {
    return folders
        .map(f => {
            const filteredSubFolders = filterFoldersByLayer(f.folders ?? [], layers);
            const filteredProjects = filterProjectsByLayer(f.projects ?? [], layers);
            const folderMatches = f.layer_id ? layers.has(f.layer_id) : false;
            if (!folderMatches && filteredSubFolders.length === 0 && filteredProjects.length === 0) return null;
            return { ...f, folders: filteredSubFolders, projects: filteredProjects };
        })
        .filter(Boolean) as GCPFolder[];
}

// ─── Filtering helpers ────────────────────────────────────────────────────────

function matchesQuery(text: string, query: string) {
    return text.toLowerCase().includes(query.toLowerCase());
}

function filterResources(resources: Resource[], query: string): Resource[] {
    return resources.filter(r =>
        matchesQuery(r.name, query) || matchesQuery(r.type, query) || matchesQuery(r.address, query)
    );
}

function filterProjects(projects: Project[], query: string): Project[] {
    return projects
        .map(p => {
            const directMatch = matchesQuery(p.display_name, query) || matchesQuery(p.project_id, query) || matchesQuery(p.id, query);
            const filteredResources = filterResources(p.resources ?? [], query);
            if (directMatch || filteredResources.length > 0) {
                return { ...p, resources: filteredResources };
            }
            return null;
        })
        .filter(Boolean) as Project[];
}

function filterFolders(folders: GCPFolder[], query: string): GCPFolder[] {
    return folders
        .map(f => {
            const directMatch = matchesQuery(f.display_name, query) || matchesQuery(f.id, query);
            const filteredSubFolders = filterFolders(f.folders ?? [], query);
            const filteredProjects = filterProjects(f.projects ?? [], query);
            if (directMatch || filteredSubFolders.length > 0 || filteredProjects.length > 0) {
                return { ...f, folders: filteredSubFolders, projects: filteredProjects };
            }
            return null;
        })
        .filter(Boolean) as GCPFolder[];
}

// ─── Sorting helpers ──────────────────────────────────────────────────────────

const byName = (a: { display_name?: string; name?: string }, b: { display_name?: string; name?: string }) =>
    (a.display_name ?? a.name ?? '').localeCompare(b.display_name ?? b.name ?? '');

function sortFolder(folder: GCPFolder): GCPFolder {
    return {
        ...folder,
        folders: [...(folder.folders ?? [])].sort(byName).map(sortFolder),
        projects: [...(folder.projects ?? [])].sort(byName).map(sortProject),
    };
}

function sortProject(project: Project): Project {
    return {
        ...project,
        resources: [...(project.resources ?? [])].sort((a, b) => a.name.localeCompare(b.name)),
    };
}

function sortOrg(org: Organization): Organization {
    return {
        ...org,
        folders: [...(org.folders ?? [])].sort(byName).map(sortFolder),
        projects: [...(org.projects ?? [])].sort(byName).map(sortProject),
    };
}

// ─── Highlight ────────────────────────────────────────────────────────────────

function Highlight({ text, query }: { text: string; query: string }) {
    if (!query) return <>{text}</>;
    const idx = text.toLowerCase().indexOf(query.toLowerCase());
    if (idx === -1) return <>{text}</>;
    return (
        <>
            {text.slice(0, idx)}
            <mark className="bg-amber-400/30 text-amber-200 rounded px-0.5">{text.slice(idx, idx + query.length)}</mark>
            {text.slice(idx + query.length)}
        </>
    );
}

// ─── HierarchyNode ────────────────────────────────────────────────────────────

interface HierarchyNodeProps {
    label: string;
    type: 'org' | 'folder' | 'project' | 'resource';
    children?: React.ReactNode;
    id?: string;
    layerId?: string;
    badge?: string;
    defaultExpanded?: boolean;
    forceExpanded?: boolean;
    isSelected?: boolean;
    onSelect?: () => void;
    query?: string;
    onExpandChange?: (expanded: boolean) => void;
}

const HierarchyNode: React.FC<HierarchyNodeProps> = ({
    label,
    type,
    children,
    id,
    layerId,
    badge,
    defaultExpanded = false,
    forceExpanded = false,
    isSelected = false,
    onSelect,
    query = '',
    onExpandChange,
}) => {
    const [isExpandedState, setIsExpandedState] = useState(defaultExpanded);
    const isExpanded = forceExpanded || isExpandedState;
    const hasChildren = React.Children.count(children) > 0;

    const getIcon = () => {
        switch (type) {
            case 'org': return <Building2 className="w-4 h-4 text-blue-400" />;
            case 'folder': return <Folder className="w-4 h-4 text-amber-400" />;
            case 'project': return <Briefcase className="w-4 h-4 text-emerald-400" />;
            case 'resource': return <Cpu className="w-4 h-4 text-slate-400" />;
        }
    };

    const handleClick = () => {
        const next = !isExpandedState;
        setIsExpandedState(next);
        onExpandChange?.(forceExpanded || next);
        onSelect?.();
    };

    return (
        <div className="select-none">
            <div
                className={cn(
                    "flex items-center py-1.5 px-2 rounded-lg hover:bg-white/5 cursor-pointer transition-colors group",
                    (isExpanded || isSelected) && "bg-white/5",
                    isSelected && "ring-1 ring-inset ring-blue-500/40"
                )}
                onClick={handleClick}
            >
                <span className="w-5 flex items-center justify-center mr-1">
                    {hasChildren && (
                        isExpanded
                            ? <ChevronDown className="w-3.5 h-3.5 text-white/50" />
                            : <ChevronRight className="w-3.5 h-3.5 text-white/50" />
                    )}
                </span>

                <span className="mr-2">{getIcon()}</span>

                <span className="text-sm font-medium text-white/90 group-hover:text-white transition-colors">
                    <Highlight text={label} query={query} />
                </span>

                {badge && (
                    <span className="ml-2 text-[10px] font-mono text-white/30 bg-white/5 border border-white/10 px-1.5 py-0.5 rounded-full shrink-0">
                        {badge}
                    </span>
                )}

                {id && (
                    <span className="ml-2 text-[10px] font-mono text-white/30 hidden group-hover:inline-block">
                        {id}
                    </span>
                )}

                {layerId && (
                    <span className="ml-auto pl-2 text-[9px] font-semibold uppercase tracking-wider text-blue-300/60 bg-blue-500/10 border border-blue-500/15 px-1.5 py-0.5 rounded-full hidden group-hover:inline-flex items-center shrink-0">
                        {layerId}
                    </span>
                )}

                {type === 'project' && (
                    <div className="ml-auto opacity-0 group-hover:opacity-100 transition-opacity">
                        <ExternalLink className="w-3 h-3 text-white/40 hover:text-white/70" />
                    </div>
                )}
            </div>

            <AnimatePresence>
                {isExpanded && hasChildren && (
                    <motion.div
                        initial={{ height: 0, opacity: 0 }}
                        animate={{ height: 'auto', opacity: 1 }}
                        exit={{ height: 0, opacity: 0 }}
                        transition={{ duration: 0.2, ease: "easeInOut" }}
                        className="ml-6 border-l border-white/10 overflow-hidden"
                    >
                        <div className="pt-1 pb-2">
                            {children}
                        </div>
                    </motion.div>
                )}
            </AnimatePresence>
        </div>
    );
};

// ─── HierarchyTree ────────────────────────────────────────────────────────────

interface HierarchyTreeProps {
    organization: Organization;
    selectedNode?: SelectedNode | null;
    onSelect?: (node: SelectedNode) => void;
    /** Set of layer IDs to filter by. Empty set = show all. */
    selectedLayerIds?: Set<string>;
}

export const HierarchyTree: React.FC<HierarchyTreeProps> = ({ organization, selectedNode, onSelect, selectedLayerIds }) => {
    const [searchQuery, setSearchQuery] = useState('');
    const isSearching = searchQuery.trim().length > 0;
    const isLayerFiltering = (selectedLayerIds?.size ?? 0) > 0;

    // Apply layer filter first, then text search on top
    const layerFilteredOrg = useMemo<Organization>(() => {
        const base = !isLayerFiltering || !selectedLayerIds ? organization : {
            ...organization,
            folders: filterFoldersByLayer(organization.folders ?? [], selectedLayerIds),
            projects: filterProjectsByLayer(organization.projects ?? [], selectedLayerIds),
        };
        return sortOrg(base);
    }, [organization, selectedLayerIds, isLayerFiltering]);

    const filteredOrg = useMemo<Organization>(() => {
        if (!isSearching) return layerFilteredOrg;
        return sortOrg({
            ...layerFilteredOrg,
            folders: filterFolders(layerFilteredOrg.folders ?? [], searchQuery),
            projects: filterProjects(layerFilteredOrg.projects ?? [], searchQuery),
        });
    }, [layerFilteredOrg, searchQuery, isSearching]);

    return (
        <div className="p-4 bg-slate-900/50 backdrop-blur-xl border border-white/10 rounded-2xl shadow-2xl overflow-hidden">
            {/* Header */}
            <div className="flex items-center gap-2 mb-4 px-2">
                <div className="p-2 bg-blue-500/10 rounded-lg">
                    <Building2 className="w-5 h-5 text-blue-400" />
                </div>
                <div>
                    <h2 className="text-lg font-semibold text-white">Infrastructure Hierarchy</h2>
                    <p className="text-xs text-white/40 font-mono uppercase tracking-wider">Landing Zone Observer</p>
                </div>
                {isLayerFiltering && (
                    <span className="ml-auto text-[10px] bg-blue-500/15 text-blue-300 border border-blue-500/20 px-2 py-0.5 rounded-full font-semibold">
                        {selectedLayerIds!.size} layer{selectedLayerIds!.size > 1 ? 's' : ''} active
                    </span>
                )}
            </div>

            {/* Search Bar */}
            <div className="relative mb-4 px-1">
                <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-white/30 pointer-events-none" />
                <input
                    type="text"
                    value={searchQuery}
                    onChange={e => setSearchQuery(e.target.value)}
                    placeholder="Search nodes…"
                    className="w-full bg-white/5 border border-white/10 rounded-lg pl-8 pr-8 py-2 text-sm text-white/80 placeholder:text-white/25 focus:outline-none focus:ring-1 focus:ring-blue-500/50 focus:border-blue-500/40 transition-all"
                />
                {isSearching && (
                    <button
                        onClick={() => setSearchQuery('')}
                        className="absolute right-3 top-1/2 -translate-y-1/2 text-white/30 hover:text-white/70 transition-colors"
                    >
                        <X className="w-3.5 h-3.5" />
                    </button>
                )}
            </div>

            {/* Tree */}
            <div className="space-y-1">
                <HierarchyNode
                    label={filteredOrg.display_name}
                    type="org"
                    id={filteredOrg.id}
                    defaultExpanded={true}
                    forceExpanded={isSearching}
                    isSelected={selectedNode?.type === 'org' && selectedNode.data.id === organization.id}
                    onSelect={() => onSelect?.({ type: 'org', data: organization })}
                    query={searchQuery}
                >
                    {filteredOrg.folders?.map(folder => (
                        <FolderNode
                            key={folder.id}
                            folder={folder}
                            selectedNode={selectedNode}
                            onSelect={onSelect}
                            query={searchQuery}
                            forceExpanded={isSearching}
                        />
                    ))}
                    {filteredOrg.projects?.map(project => (
                        <ProjectNode
                            key={project.id}
                            project={project}
                            selectedNode={selectedNode}
                            onSelect={onSelect}
                            query={searchQuery}
                        />
                    ))}
                </HierarchyNode>

                {isSearching && filteredOrg.folders?.length === 0 && filteredOrg.projects?.length === 0 && (
                    <p className="text-xs text-white/30 text-center py-4">
                        No nodes match &ldquo;{searchQuery}&rdquo;{isLayerFiltering ? ' in the selected layers' : ''}
                    </p>
                )}
                {!isSearching && isLayerFiltering && filteredOrg.folders?.length === 0 && filteredOrg.projects?.length === 0 && (
                    <p className="text-xs text-white/30 text-center py-4">
                        No resources found in the selected layer{selectedLayerIds!.size > 1 ? 's' : ''}.
                    </p>
                )}
            </div>
        </div>
    );
};

// ─── FolderNode ───────────────────────────────────────────────────────────────

interface FolderNodeProps {
    folder: GCPFolder;
    selectedNode?: SelectedNode | null;
    onSelect?: (node: SelectedNode) => void;
    query?: string;
    forceExpanded?: boolean;
}

const FolderNode: React.FC<FolderNodeProps> = ({ folder, selectedNode, onSelect, query = '', forceExpanded = false }) => (
    <HierarchyNode
        label={folder.display_name}
        type="folder"
        id={folder.id}
        layerId={folder.layer_id}
        isSelected={selectedNode?.type === 'folder' && selectedNode.data.id === folder.id}
        onSelect={() => onSelect?.({ type: 'folder', data: folder })}
        query={query}
        forceExpanded={forceExpanded}
    >
        {folder.folders?.map(f => (
            <FolderNode key={f.id} folder={f} selectedNode={selectedNode} onSelect={onSelect} query={query} forceExpanded={forceExpanded} />
        ))}
        {folder.projects?.map(p => (
            <ProjectNode key={p.id} project={p} selectedNode={selectedNode} onSelect={onSelect} query={query} />
        ))}
    </HierarchyNode>
);

// ─── VirtualResourceList ──────────────────────────────────────────────────────

// RowProps for react-window v2 — must NOT include ariaAttributes, index, or style
type ResourceRowProps = {
    resources: Resource[];
    selectedNode?: SelectedNode | null;
    onSelect?: (node: SelectedNode) => void;
    query?: string;
};

// Row renderer: react-window v2 injects ariaAttributes + index + style on top of rowProps
function ResourceRow(props: {
    ariaAttributes: { 'aria-posinset': number; 'aria-setsize': number; role: 'listitem' };
    index: number;
    style: React.CSSProperties;
} & ResourceRowProps) {
    const { index, style, resources, selectedNode, onSelect, query = '' } = props;
    const res = resources[index];
    return (
        <div style={style}>
            <HierarchyNode
                label={res.name}
                type="resource"
                id={res.type}
                layerId={res.layer_id}
                isSelected={selectedNode?.type === 'resource' && selectedNode.data.address === res.address}
                onSelect={() => onSelect?.({ type: 'resource', data: res })}
                query={query}
            />
        </div>
    );
}

interface VirtualResourceListProps {
    resources: Resource[];
    selectedNode?: SelectedNode | null;
    onSelect?: (node: SelectedNode) => void;
    query?: string;
}

const VirtualResourceList: React.FC<VirtualResourceListProps> = ({ resources, selectedNode, onSelect, query = '' }) => {
    const height = Math.min(resources.length * VIRTUAL_ITEM_HEIGHT, VIRTUAL_MAX_HEIGHT);

    // Stable rowProps — List re-renders rows automatically when this changes
    const rowProps = useMemo<ResourceRowProps>(
        () => ({ resources, selectedNode, onSelect, query }),
        [resources, selectedNode, onSelect, query],
    );

    return (
        <List<ResourceRowProps>
            rowComponent={ResourceRow}
            rowCount={resources.length}
            rowHeight={VIRTUAL_ITEM_HEIGHT}
            rowProps={rowProps}
            style={{ height }}
            className="scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent"
        />
    );
};

// ─── ProjectNode ──────────────────────────────────────────────────────────────

interface ProjectNodeProps {
    project: Project;
    selectedNode?: SelectedNode | null;
    onSelect?: (node: SelectedNode) => void;
    query?: string;
}

const ProjectNode: React.FC<ProjectNodeProps> = ({ project, selectedNode, onSelect, query = '' }) => {
    const [visibleCount, setVisibleCount] = useState(RESOURCE_PAGE_SIZE);

    const allResources = project.resources ?? [];
    const total = allResources.length;

    // Bypass pagination when query is at least 3 characters
    const searchBypass = query.trim().length >= 3;

    // Reset visible count when collapsing
    const handleExpandChange = (expanded: boolean) => {
        if (!expanded) setVisibleCount(RESOURCE_PAGE_SIZE);
    };

    // Reset when search bypass transitions from true → false (user clears/shortens search)
    const prevSearchBypassRef = useRef(searchBypass);
    useEffect(() => {
        if (prevSearchBypassRef.current && !searchBypass) {
            const timer = setTimeout(() => setVisibleCount(RESOURCE_PAGE_SIZE), 0);
            prevSearchBypassRef.current = searchBypass;
            return () => clearTimeout(timer);
        }
        prevSearchBypassRef.current = searchBypass;
    }, [searchBypass]);

    const visibleResources = searchBypass ? allResources : allResources.slice(0, visibleCount);
    const remaining = total - visibleCount;
    const showMoreButton = !searchBypass && visibleCount < total;
    const useVirtual = visibleResources.length > VIRTUAL_THRESHOLD;

    const badge = total > 0 ? `${total} resource${total !== 1 ? 's' : ''}` : undefined;

    return (
        <HierarchyNode
            label={project.display_name}
            type="project"
            id={project.project_id}
            layerId={project.layer_id}
            badge={badge}
            isSelected={selectedNode?.type === 'project' && selectedNode.data.id === project.id}
            onSelect={() => onSelect?.({ type: 'project', data: project })}
            query={query}
            forceExpanded={query.length > 0}
            onExpandChange={handleExpandChange}
        >
            {useVirtual ? (
                <VirtualResourceList
                    resources={visibleResources}
                    selectedNode={selectedNode}
                    onSelect={onSelect}
                    query={query}
                />
            ) : (
                visibleResources.map(res => (
                    <HierarchyNode
                        key={res.address}
                        label={res.name}
                        type="resource"
                        id={res.type}
                        layerId={res.layer_id}
                        isSelected={selectedNode?.type === 'resource' && selectedNode.data.address === res.address}
                        onSelect={() => onSelect?.({ type: 'resource', data: res })}
                        query={query}
                    />
                ))
            )}

            {showMoreButton && (
                <div className="flex items-center gap-2 pl-2 pt-1">
                    <button
                        onClick={e => { e.stopPropagation(); setVisibleCount(c => c + RESOURCE_PAGE_SIZE); }}
                        className="text-xs text-blue-300/70 hover:text-blue-200 transition-colors flex items-center gap-1"
                    >
                        <ChevronDown className="w-3 h-3" />
                        Show {Math.min(remaining, RESOURCE_PAGE_SIZE)} more
                        <span className="text-white/25">({remaining} remaining)</span>
                    </button>
                    {remaining > RESOURCE_PAGE_SIZE && (
                        <button
                            onClick={e => { e.stopPropagation(); setVisibleCount(total); }}
                            className="text-xs text-white/30 hover:text-white/60 transition-colors"
                        >
                            Show all
                        </button>
                    )}
                </div>
            )}
        </HierarchyNode>
    );
};
