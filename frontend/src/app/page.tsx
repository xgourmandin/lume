"use client";

import React, { useState, useEffect } from 'react';
import { HierarchyTree } from '@/components/HierarchyTree';
import { Organization } from '@/types';
import { Shield, Server, RefreshCw, Layers } from 'lucide-react';

// Sample data for initial showcase
const mockOrg: Organization = {
  id: "9876543210",
  display_name: "Cloud Adopt Global",
  folders: [
    {
      id: "1234567890",
      display_name: "Engineering",
      parent: "organizations/9876543210",
      folders: [
        {
          id: "2345678901",
          display_name: "Platforms",
          parent: "folders/1234567890",
          projects: [
            {
              id: "lume-dev-project",
              project_id: "lume-dev-123",
              display_name: "Lume Development",
              parent: "folders/2345678901",
              resources: [
                { type: "google_compute_instance", name: "api-server", address: "module.api.instance", id: "123" },
                { type: "google_firestore_database", name: "main-db", address: "google_firestore_database.main", id: "456" }
              ]
            }
          ]
        },
        {
          id: "3456789012",
          display_name: "Security",
          parent: "folders/1234567890",
          projects: []
        }
      ]
    }
  ]
};

export default function Home() {
  const [org, setOrg] = useState<Organization>(mockOrg);
  const [isSyncing, setIsSyncing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchHierarchy = async () => {
    try {
      const res = await fetch('http://localhost:3000/api/v1/hierarchy/default');
      if (!res.ok) throw new Error('Failed to fetch hierarchy');
      const data = await res.json();
      setOrg(data);
      setError(null);
    } catch (err) {
      console.error('Using mock data as fallback:', err);
      // Fallback is implicit as we initialize with mockOrg
      setError('Live data unreachable. Showing fallback.');
    }
  };

  useEffect(() => {
    fetchHierarchy();
  }, []);

  const handleSync = async () => {
    setIsSyncing(true);
    try {
      const res = await fetch('http://localhost:3000/api/v1/hierarchy/sync', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          workspace_id: 'default',
          bucket: 'terraform-state-lume', // These could be config values
          object: 'terraform.tfstate'
        })
      });

      if (!res.ok) throw new Error('Sync failed');
      const updatedOrg = await res.json();
      setOrg(updatedOrg);
      setError(null);
    } catch (err) {
      console.error('Sync failed:', err);
      setError('Sync failed. Please check backend logs.');
    } finally {
      setIsSyncing(false);
    }
  };

  return (
    <main className="min-h-screen bg-[#020617] text-slate-200 selection:bg-blue-500/30">
      {/* Background Decor */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/10 blur-[120px] rounded-full" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-emerald-600/10 blur-[120px] rounded-full" />
      </div>

      <nav className="relative z-10 border-b border-white/5 bg-slate-950/50 backdrop-blur-md px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 bg-gradient-to-br from-blue-500 to-emerald-500 rounded-xl flex items-center justify-center shadow-lg shadow-blue-500/20">
            <Shield className="w-6 h-6 text-white" />
          </div>
          <div>
            <h1 className="text-xl font-bold tracking-tight text-white">Lume</h1>
            <p className="text-[10px] text-white/40 font-semibold uppercase tracking-widest">Observer Console</p>
          </div>
        </div>

        <div className="flex items-center gap-4">
          {error && (
            <span className="text-[10px] bg-amber-500/10 text-amber-500 px-3 py-1 rounded-full border border-amber-500/20 font-medium">
              {error}
            </span>
          )}
          <button
            onClick={handleSync}
            disabled={isSyncing}
            className={`flex items-center gap-2 px-4 py-2 bg-white/5 hover:bg-white/10 rounded-full text-sm font-medium transition-all active:scale-95 border border-white/10 ${isSyncing ? 'opacity-50 cursor-not-allowed' : ''}`}
          >
            <RefreshCw className={`w-4 h-4 text-emerald-400 ${isSyncing ? 'animate-spin' : ''}`} />
            <span>{isSyncing ? 'Syncing...' : 'Sync Hierarchy'}</span>
          </button>
          <div className="w-10 h-10 rounded-full bg-blue-500/20 border border-blue-500/30 flex items-center justify-center">
            <div className="w-2 h-2 rounded-full bg-blue-400 animate-pulse" />
          </div>
        </div>
      </nav>

      <div className="relative z-10 max-w-7xl mx-auto grid grid-cols-12 gap-8 p-8">
        {/* Left Stats Column */}
        <div className="col-span-12 lg:col-span-3 space-y-6">
          <div className="p-6 bg-white/5 border border-white/10 rounded-2xl">
            <div className="flex items-center gap-3 mb-4">
              <Server className="w-5 h-5 text-blue-400" />
              <h3 className="font-semibold text-white">Workspaces</h3>
            </div>
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <span className="text-sm text-white/50">Total Projects</span>
                <span className="text-lg font-bold text-white">
                  {org.folders?.reduce((acc, f) => acc + (f.projects?.length || 0), 0) || 0}
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-white/50">Healthy</span>
                <span className="text-sm font-semibold text-emerald-400 bg-emerald-400/10 px-2 py-0.5 rounded">38</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-white/50">Drifted</span>
                <span className="text-sm font-semibold text-amber-400 bg-amber-400/10 px-2 py-0.5 rounded">4</span>
              </div>
            </div>
          </div>

          <div className="p-6 bg-white/5 border border-white/10 rounded-2xl">
            <div className="flex items-center gap-3 mb-4">
              <Layers className="w-5 h-5 text-emerald-400" />
              <h3 className="font-semibold text-white">Quick Actions</h3>
            </div>
            <div className="grid grid-cols-1 gap-2">
              <button className="text-left w-full px-4 py-2.5 rounded-xl bg-white/5 hover:bg-white/10 border border-white/5 text-sm transition-colors">
                New Project Request
              </button>
              <button className="text-left w-full px-4 py-2.5 rounded-xl bg-white/5 hover:bg-white/10 border border-white/5 text-sm transition-colors text-white/60">
                View Drift Reports
              </button>
            </div>
          </div>
        </div>

        {/* Center Hierarchy Main Column */}
        <div className="col-span-12 lg:col-span-9">
          <HierarchyTree organization={org} />
        </div>
      </div>
    </main>
  );
}
