"use client";

import React, { useState } from 'react';
import { ChevronRight, ChevronDown, Folder, Briefcase, Building2, Cpu, ExternalLink } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { Organization, Folder as GCPFolder, Project, Resource } from '../types';
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

function cn(...inputs: ClassValue[]) {
    return twMerge(clsx(inputs));
}

interface HierarchyNodeProps {
    label: string;
    type: 'org' | 'folder' | 'project' | 'resource';
    children?: React.ReactNode;
    id?: string;
    defaultExpanded?: boolean;
}

const HierarchyNode: React.FC<HierarchyNodeProps> = ({
    label,
    type,
    children,
    id,
    defaultExpanded = false
}) => {
    const [isExpanded, setIsExpanded] = useState(defaultExpanded);
    const hasChildren = React.Children.count(children) > 0;

    const getIcon = () => {
        switch (type) {
            case 'org': return <Building2 className="w-4 h-4 text-blue-400" />;
            case 'folder': return <Folder className="w-4 h-4 text-amber-400" />;
            case 'project': return <Briefcase className="w-4 h-4 text-emerald-400" />;
            case 'resource': return <Cpu className="w-4 h-4 text-slate-400" />;
        }
    };

    return (
        <div className="select-none">
            <div
                className={cn(
                    "flex items-center py-1.5 px-2 rounded-lg hover:bg-white/5 cursor-pointer transition-colors group",
                    isExpanded && "bg-white/5"
                )}
                onClick={() => setIsExpanded(!isExpanded)}
            >
                <span className="w-5 flex items-center justify-center mr-1">
                    {hasChildren && (
                        isExpanded ? <ChevronDown className="w-3.5 h-3.5 text-white/50" /> : <ChevronRight className="w-3.5 h-3.5 text-white/50" />
                    )}
                </span>

                <span className="mr-2">
                    {getIcon()}
                </span>

                <span className="text-sm font-medium text-white/90 group-hover:text-white transition-colors">
                    {label}
                </span>

                {id && (
                    <span className="ml-2 text-[10px] font-mono text-white/30 hidden group-hover:inline-block">
                        {id}
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

export const HierarchyTree: React.FC<{ organization: Organization }> = ({ organization }) => {
    return (
        <div className="p-4 bg-slate-900/50 backdrop-blur-xl border border-white/10 rounded-2xl shadow-2xl overflow-hidden">
            <div className="flex items-center gap-2 mb-6 px-2">
                <div className="p-2 bg-blue-500/10 rounded-lg">
                    <Building2 className="w-5 h-5 text-blue-400" />
                </div>
                <div>
                    <h2 className="text-lg font-semibold text-white">Infrastructure Hierarchy</h2>
                    <p className="text-xs text-white/40 font-mono uppercase tracking-wider">Landing Zone Observer</p>
                </div>
            </div>

            <div className="space-y-1">
                <HierarchyNode
                    label={organization.display_name}
                    type="org"
                    id={organization.id}
                    defaultExpanded={true}
                >
                    {organization.folders?.map(folder => (
                        <FolderNode key={folder.id} folder={folder} />
                    ))}
                    {organization.projects?.map(project => (
                        <ProjectNode key={project.id} project={project} />
                    ))}
                </HierarchyNode>
            </div>
        </div>
    );
};

const FolderNode: React.FC<{ folder: GCPFolder }> = ({ folder }) => (
    <HierarchyNode label={folder.display_name} type="folder" id={folder.id}>
        {folder.folders?.map(f => (
            <FolderNode key={f.id} folder={f} />
        ))}
        {folder.projects?.map(p => (
            <ProjectNode key={p.id} project={p} />
        ))}
    </HierarchyNode>
);

const ProjectNode: React.FC<{ project: Project }> = ({ project }) => (
    <HierarchyNode label={project.display_name} type="project" id={project.project_id}>
        {project.resources?.map(res => (
            <HierarchyNode
                key={res.address}
                label={res.name}
                type="resource"
                id={res.type}
            />
        ))}
    </HierarchyNode>
);
