"use client";

import React, { useState, useEffect, useCallback } from 'react';
import { HierarchyTree } from '@/components/HierarchyTree';
import { DetailPanel } from '@/components/DetailPanel';
import { WorkspaceLayers } from '@/components/WorkspaceLayers';
import type { SelectedNode } from '@/components/DetailPanel';
import type { Organization, Workspace, TerraformWorkspace } from '@/types';
import { fetchHierarchy, fetchWorkspace, syncWorkspace } from '@/lib/api';
import { Shield, Server, RefreshCw, Layers, Building2 } from 'lucide-react';

// ---------------------------------------------------------------------------
// Fallback mock for when the backend is not reachable
// ---------------------------------------------------------------------------
const MOCK_ORG: Organization = {
  id: "987654321098",
  display_name: "Cloud Adopt Global",
  folders: [
    // ── Engineering ──────────────────────────────────────────────────────────
    {
      id: "100000000001",
      display_name: "Engineering",
      parent: "organizations/987654321098",
      layer_id: "org",
      folders: [
        {
          id: "200000000001",
          display_name: "Platform",
          parent: "folders/100000000001",
          layer_id: "org",
          projects: [
            {
              id: "proj-platform-infra",
              project_id: "cag-platform-infra",
              display_name: "Platform Infrastructure",
              parent: "folders/200000000001",
              layer_id: "network",
              workspace_id: "default",
              resources: [
                { type: "google_compute_network", name: "vpc-main", address: "module.network.google_compute_network.main", id: "projects/cag-platform-infra/global/networks/vpc-main", layer_id: "network" },
                { type: "google_compute_subnetwork", name: "subnet-europe-west1", address: "module.network.google_compute_subnetwork.europe_west1", id: "projects/cag-platform-infra/regions/europe-west1/subnetworks/subnet-europe-west1", layer_id: "network" },
                { type: "google_compute_subnetwork", name: "subnet-us-central1", address: "module.network.google_compute_subnetwork.us_central1", id: "projects/cag-platform-infra/regions/us-central1/subnetworks/subnet-us-central1", layer_id: "network" },
                { type: "google_dns_managed_zone", name: "internal-dns", address: "google_dns_managed_zone.internal", id: "projects/cag-platform-infra/managedZones/internal-dns", layer_id: "network" },
                { type: "google_artifact_registry_repository", name: "docker-registry", address: "google_artifact_registry_repository.docker", id: "projects/cag-platform-infra/locations/europe-west1/repositories/docker-registry", layer_id: "network" },
              ],
            },
            {
              id: "proj-platform-ci",
              project_id: "cag-platform-cicd",
              display_name: "CI/CD Pipeline",
              parent: "folders/200000000001",
              layer_id: "projects",
              workspace_id: "default",
              resources: [
                { type: "google_cloudbuild_trigger", name: "main-build", address: "google_cloudbuild_trigger.main", id: "projects/cag-platform-cicd/locations/global/triggers/main-build", layer_id: "projects" },
                { type: "google_storage_bucket", name: "build-artifacts", address: "google_storage_bucket.artifacts", id: "cag-platform-cicd-artifacts", layer_id: "projects" },
                { type: "google_service_account", name: "cloudbuild-sa", address: "google_service_account.cloudbuild", id: "projects/cag-platform-cicd/serviceAccounts/cloudbuild@cag-platform-cicd.iam.gserviceaccount.com", layer_id: "projects" },
              ],
            },
          ],
        },
        {
          id: "200000000002",
          display_name: "Applications",
          parent: "folders/100000000001",
          layer_id: "org",
          folders: [
            {
              id: "300000000001",
              display_name: "Production",
              parent: "folders/200000000002",
              layer_id: "org",
              projects: [
                {
                  id: "proj-app-prod",
                  project_id: "cag-app-prod",
                  display_name: "App — Production",
                  parent: "folders/300000000001",
                  layer_id: "apps",
                  workspace_id: "prod",
                  resources: [
                    { type: "google_cloud_run_v2_service", name: "api-service", address: "module.app.google_cloud_run_v2_service.api", id: "projects/cag-app-prod/locations/europe-west1/services/api-service", layer_id: "apps" },
                    { type: "google_cloud_run_v2_service", name: "frontend-service", address: "module.app.google_cloud_run_v2_service.frontend", id: "projects/cag-app-prod/locations/europe-west1/services/frontend-service", layer_id: "apps" },
                    { type: "google_sql_database_instance", name: "postgres-main", address: "module.db.google_sql_database_instance.main", id: "projects/cag-app-prod/instances/postgres-main", layer_id: "apps" },
                    { type: "google_redis_instance", name: "cache-main", address: "module.cache.google_redis_instance.main", id: "projects/cag-app-prod/locations/europe-west1/instances/cache-main", layer_id: "apps" },
                    { type: "google_storage_bucket", name: "media-assets", address: "google_storage_bucket.media", id: "cag-app-prod-media", layer_id: "apps" },
                    { type: "google_pubsub_topic", name: "events-topic", address: "google_pubsub_topic.events", id: "projects/cag-app-prod/topics/events-topic", layer_id: "apps" },
                  ],
                },
                {
                  id: "proj-app-prod-data",
                  project_id: "cag-app-prod-data",
                  display_name: "App — Data Warehouse",
                  parent: "folders/300000000001",
                  layer_id: "apps",
                  workspace_id: "prod",
                  resources: [
                    { type: "google_bigquery_dataset", name: "analytics", address: "google_bigquery_dataset.analytics", id: "projects/cag-app-prod-data/datasets/analytics", layer_id: "apps" },
                    { type: "google_bigquery_table", name: "events_raw", address: "google_bigquery_table.events_raw", id: "projects/cag-app-prod-data/datasets/analytics/tables/events_raw", layer_id: "apps" },
                    { type: "google_dataflow_job", name: "events-pipeline", address: "google_dataflow_job.events", id: "2024-events-pipeline-job", layer_id: "apps" },
                  ],
                },
                // ── Large project for pagination / virtualisation testing ──────
                {
                  id: "proj-platform-mesh",
                  project_id: "cag-platform-mesh",
                  display_name: "Platform — Service Mesh (25 resources)",
                  parent: "folders/300000000001",
                  layer_id: "apps",
                  workspace_id: "prod",
                  resources: [
                    { type: "google_compute_network", name: "mesh-vpc", address: "module.mesh.google_compute_network.vpc", id: "projects/cag-platform-mesh/global/networks/mesh-vpc", layer_id: "apps" },
                    { type: "google_compute_subnetwork", name: "mesh-subnet-ew1", address: "module.mesh.google_compute_subnetwork.ew1", id: "projects/cag-platform-mesh/regions/europe-west1/subnetworks/mesh-subnet-ew1", layer_id: "apps" },
                    { type: "google_compute_subnetwork", name: "mesh-subnet-ew4", address: "module.mesh.google_compute_subnetwork.ew4", id: "projects/cag-platform-mesh/regions/europe-west4/subnetworks/mesh-subnet-ew4", layer_id: "apps" },
                    { type: "google_container_cluster", name: "gke-prod-ew1", address: "module.gke.google_container_cluster.prod_ew1", id: "projects/cag-platform-mesh/locations/europe-west1/clusters/gke-prod-ew1", layer_id: "apps" },
                    { type: "google_container_cluster", name: "gke-prod-ew4", address: "module.gke.google_container_cluster.prod_ew4", id: "projects/cag-platform-mesh/locations/europe-west4/clusters/gke-prod-ew4", layer_id: "apps" },
                    { type: "google_container_node_pool", name: "default-pool-ew1", address: "module.gke.google_container_node_pool.default_ew1", id: "projects/cag-platform-mesh/locations/europe-west1/clusters/gke-prod-ew1/nodePools/default-pool", layer_id: "apps" },
                    { type: "google_container_node_pool", name: "default-pool-ew4", address: "module.gke.google_container_node_pool.default_ew4", id: "projects/cag-platform-mesh/locations/europe-west4/clusters/gke-prod-ew4/nodePools/default-pool", layer_id: "apps" },
                    { type: "google_service_account", name: "gke-workload-sa", address: "google_service_account.gke_workload", id: "projects/cag-platform-mesh/serviceAccounts/gke-workload@cag-platform-mesh.iam.gserviceaccount.com", layer_id: "apps" },
                    { type: "google_service_account", name: "istio-pilot-sa", address: "google_service_account.istio_pilot", id: "projects/cag-platform-mesh/serviceAccounts/istio-pilot@cag-platform-mesh.iam.gserviceaccount.com", layer_id: "apps" },
                    { type: "google_compute_global_address", name: "mesh-ingress-ip", address: "google_compute_global_address.ingress", id: "projects/cag-platform-mesh/global/addresses/mesh-ingress-ip", layer_id: "apps" },
                    { type: "google_compute_ssl_certificate", name: "mesh-tls-cert", address: "google_compute_ssl_certificate.tls", id: "projects/cag-platform-mesh/global/sslCertificates/mesh-tls-cert", layer_id: "apps" },
                    { type: "google_compute_backend_service", name: "api-backend", address: "google_compute_backend_service.api", id: "projects/cag-platform-mesh/global/backendServices/api-backend", layer_id: "apps" },
                    { type: "google_compute_url_map", name: "mesh-url-map", address: "google_compute_url_map.mesh", id: "projects/cag-platform-mesh/global/urlMaps/mesh-url-map", layer_id: "apps" },
                    { type: "google_dns_managed_zone", name: "mesh-internal-dns", address: "google_dns_managed_zone.internal", id: "projects/cag-platform-mesh/managedZones/mesh-internal-dns", layer_id: "apps" },
                    { type: "google_dns_record_set", name: "api-dns-record", address: "google_dns_record_set.api", id: "projects/cag-platform-mesh/managedZones/mesh-internal-dns/rrsets/api.internal./A", layer_id: "apps" },
                    { type: "google_secret_manager_secret", name: "db-password", address: "google_secret_manager_secret.db_password", id: "projects/cag-platform-mesh/secrets/db-password", layer_id: "apps" },
                    { type: "google_secret_manager_secret", name: "api-key", address: "google_secret_manager_secret.api_key", id: "projects/cag-platform-mesh/secrets/api-key", layer_id: "apps" },
                    { type: "google_kms_key_ring", name: "mesh-keyring", address: "google_kms_key_ring.mesh", id: "projects/cag-platform-mesh/locations/europe-west1/keyRings/mesh-keyring", layer_id: "apps" },
                    { type: "google_kms_crypto_key", name: "mesh-enc-key", address: "google_kms_crypto_key.mesh_enc", id: "projects/cag-platform-mesh/locations/europe-west1/keyRings/mesh-keyring/cryptoKeys/mesh-enc-key", layer_id: "apps" },
                    { type: "google_pubsub_topic", name: "mesh-events", address: "google_pubsub_topic.mesh_events", id: "projects/cag-platform-mesh/topics/mesh-events", layer_id: "apps" },
                    { type: "google_pubsub_subscription", name: "mesh-events-sub", address: "google_pubsub_subscription.mesh_events", id: "projects/cag-platform-mesh/subscriptions/mesh-events-sub", layer_id: "apps" },
                    { type: "google_storage_bucket", name: "mesh-artifacts", address: "google_storage_bucket.artifacts", id: "cag-platform-mesh-artifacts", layer_id: "apps" },
                    { type: "google_monitoring_dashboard", name: "mesh-dashboard", address: "google_monitoring_dashboard.mesh", id: "projects/cag-platform-mesh/dashboards/mesh-dashboard", layer_id: "apps" },
                    { type: "google_monitoring_alert_policy", name: "mesh-latency-alert", address: "google_monitoring_alert_policy.latency", id: "projects/cag-platform-mesh/alertPolicies/11111111111", layer_id: "apps" },
                    { type: "google_logging_project_sink", name: "mesh-audit-sink", address: "google_logging_project_sink.audit", id: "projects/cag-platform-mesh/sinks/mesh-audit-sink", layer_id: "apps" },
                  ],
                },
              ],
            },
            {
              id: "300000000002",
              display_name: "Staging",
              parent: "folders/200000000002",
              layer_id: "org",
              projects: [
                {
                  id: "proj-app-staging",
                  project_id: "cag-app-staging",
                  display_name: "App — Staging",
                  parent: "folders/300000000002",
                  layer_id: "apps",
                  workspace_id: "staging",
                  resources: [
                    { type: "google_cloud_run_v2_service", name: "api-service", address: "module.app.google_cloud_run_v2_service.api", id: "projects/cag-app-staging/locations/europe-west1/services/api-service", layer_id: "apps" },
                    { type: "google_sql_database_instance", name: "postgres-staging", address: "module.db.google_sql_database_instance.staging", id: "projects/cag-app-staging/instances/postgres-staging", layer_id: "apps" },
                  ],
                },
              ],
            },
            {
              id: "300000000003",
              display_name: "Development",
              parent: "folders/200000000002",
              layer_id: "org",
              projects: [
                {
                  id: "proj-app-dev",
                  project_id: "cag-app-dev",
                  display_name: "App — Development",
                  parent: "folders/300000000003",
                  layer_id: "apps",
                  workspace_id: "dev",
                  resources: [
                    { type: "google_cloud_run_v2_service", name: "api-service-dev", address: "module.app.google_cloud_run_v2_service.api", id: "projects/cag-app-dev/locations/europe-west1/services/api-service-dev", layer_id: "apps" },
                    { type: "google_firestore_database", name: "dev-db", address: "google_firestore_database.dev", id: "(default)", layer_id: "apps" },
                  ],
                },
              ],
            },
          ],
        },
        {
          id: "200000000003",
          display_name: "Security",
          parent: "folders/100000000001",
          layer_id: "org",
          projects: [
            {
              id: "proj-security-ops",
              project_id: "cag-security-ops",
              display_name: "Security Operations",
              parent: "folders/200000000003",
              layer_id: "security",
              workspace_id: "default",
              resources: [
                { type: "google_scc_notification_config", name: "scc-alerts", address: "google_scc_notification_config.alerts", id: "organizations/987654321098/notificationConfigs/scc-alerts", layer_id: "security" },
                { type: "google_kms_key_ring", name: "primary-keyring", address: "google_kms_key_ring.primary", id: "projects/cag-security-ops/locations/europe-west1/keyRings/primary-keyring", layer_id: "security" },
                { type: "google_kms_crypto_key", name: "data-encryption-key", address: "google_kms_crypto_key.data_enc", id: "projects/cag-security-ops/locations/europe-west1/keyRings/primary-keyring/cryptoKeys/data-encryption-key", layer_id: "security" },
                { type: "google_logging_project_sink", name: "audit-sink", address: "google_logging_project_sink.audit", id: "projects/cag-security-ops/sinks/audit-sink", layer_id: "security" },
              ],
            },
          ],
        },
      ],
    },
    // ── Operations ───────────────────────────────────────────────────────────
    {
      id: "100000000002",
      display_name: "Operations",
      parent: "organizations/987654321098",
      layer_id: "org",
      folders: [
        {
          id: "200000000004",
          display_name: "Observability",
          parent: "folders/100000000002",
          layer_id: "org",
          projects: [
            {
              id: "proj-monitoring",
              project_id: "cag-monitoring",
              display_name: "Central Monitoring",
              parent: "folders/200000000004",
              layer_id: "projects",
              workspace_id: "default",
              resources: [
                { type: "google_monitoring_dashboard", name: "infra-overview", address: "google_monitoring_dashboard.infra", id: "projects/cag-monitoring/dashboards/infra-overview", layer_id: "projects" },
                { type: "google_monitoring_alert_policy", name: "high-cpu-alert", address: "google_monitoring_alert_policy.high_cpu", id: "projects/cag-monitoring/alertPolicies/12345678901", layer_id: "projects" },
                { type: "google_monitoring_alert_policy", name: "error-rate-alert", address: "google_monitoring_alert_policy.error_rate", id: "projects/cag-monitoring/alertPolicies/98765432100", layer_id: "projects" },
                { type: "google_logging_metric", name: "error-log-metric", address: "google_logging_metric.errors", id: "projects/cag-monitoring/metrics/error-log-metric", layer_id: "projects" },
              ],
            },
          ],
        },
      ],
      // Direct project under Operations
      projects: [
        {
          id: "proj-billing",
          project_id: "cag-billing-mgmt",
          display_name: "Billing Management",
          parent: "folders/100000000002",
          layer_id: "org",
          workspace_id: "default",
          resources: [
            { type: "google_billing_budget", name: "org-monthly-budget", address: "google_billing_budget.monthly", id: "billingAccounts/01ABCD-EF1234-567890/budgets/org-monthly-budget", layer_id: "org" },
          ],
        },
      ],
    },
  ],
  // Direct projects under the org (no folder)
  projects: [
    {
      id: "proj-org-bootstrap",
      project_id: "cag-org-bootstrap",
      display_name: "Org Bootstrap",
      parent: "organizations/987654321098",
      layer_id: "org",
      workspace_id: "default",
      resources: [
        { type: "google_organization_policy", name: "restrict-domains", address: "google_organization_policy.restrict_domains", id: "organizations/987654321098/policies/iam.allowedPolicyMemberDomains", layer_id: "org" },
        { type: "google_project", name: "seed-project", address: "google_project.seed", id: "projects/cag-org-bootstrap", layer_id: "org" },
      ],
    },
  ],
};

const MOCK_WORKSPACE: Workspace = {
  id: "landing-zone",
  last_sync: new Date(Date.now() - 8 * 60 * 1000).toISOString(), // 8 min ago
  status: "drifted",
  layers: [
    {
      id: "org",
      name: "Organization",
      last_sync: new Date(Date.now() - 8 * 60 * 1000).toISOString(),
      status: "clean",
      workspaces: [
        { id: "default", layer_id: "org", status: "clean", last_sync: new Date(Date.now() - 8 * 60 * 1000).toISOString() },
      ],
    },
    {
      id: "network",
      name: "Network",
      last_sync: new Date(Date.now() - 8 * 60 * 1000).toISOString(),
      status: "drifted",
      workspaces: [
        { id: "default", layer_id: "network", status: "drifted", last_sync: new Date(Date.now() - 8 * 60 * 1000).toISOString() },
      ],
    },
    {
      id: "security",
      name: "Security",
      last_sync: new Date(Date.now() - 25 * 60 * 1000).toISOString(),
      status: "clean",
      workspaces: [
        { id: "default", layer_id: "security", status: "clean", last_sync: new Date(Date.now() - 25 * 60 * 1000).toISOString() },
      ],
    },
    {
      id: "projects",
      name: "Projects",
      last_sync: new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString(),
      status: "error",
      workspaces: [
        { id: "default", layer_id: "projects", status: "error", last_sync: new Date(Date.now() - 3 * 60 * 60 * 1000).toISOString() },
      ],
    },
    {
      id: "apps",
      name: "Applications",
      last_sync: new Date(Date.now() - 12 * 60 * 60 * 1000).toISOString(),
      status: "clean",
      workspaces: [
        { id: "default", layer_id: "apps", status: "clean", last_sync: new Date(Date.now() - 12 * 60 * 60 * 1000).toISOString() },
        { id: "prod",    layer_id: "apps", status: "drifted", last_sync: new Date(Date.now() - 13 * 60 * 60 * 1000).toISOString() },
        { id: "staging", layer_id: "apps", status: "clean",   last_sync: new Date(Date.now() - 15 * 60 * 60 * 1000).toISOString() },
        { id: "dev",     layer_id: "apps", status: "clean",   last_sync: new Date(Date.now() - 20 * 60 * 60 * 1000).toISOString() },
      ],
    },
  ],
};

const DEFAULT_WORKSPACE_ID = "default";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function countProjectsDeep(node: Organization | import('@/types').Folder): number {
  const direct = (node.projects ?? []).length;
  const nested = (node.folders ?? []).reduce((acc, f) => acc + countProjectsDeep(f), 0);
  return direct + nested;
}

// ---------------------------------------------------------------------------
// Page
// ---------------------------------------------------------------------------

export default function Home() {
  const [org, setOrg] = useState<Organization>(MOCK_ORG);
  const [workspace, setWorkspace] = useState<Workspace>(MOCK_WORKSPACE);
  const [selectedLayerIds, setSelectedLayerIds] = useState<Set<string>>(new Set());
  const [selectedWorkspaceIds, setSelectedWorkspaceIds] = useState<Set<string>>(new Set());
  const [isSyncing, setIsSyncing] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedNode, setSelectedNode] = useState<SelectedNode | null>(null);

  // Load hierarchy + workspace metadata on mount
  const loadData = useCallback(async () => {
    setIsLoading(true);
    try {
      const [hier, ws] = await Promise.all([
        fetchHierarchy(DEFAULT_WORKSPACE_ID),
        fetchWorkspace(DEFAULT_WORKSPACE_ID),
      ]);
      setOrg(hier);
      setWorkspace(ws);
      setError(null);
    } catch (err) {
      console.error('Using mock data as fallback:', err);
      setError('Live data unreachable — showing fallback data.');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleSync = async () => {
    setIsSyncing(true);
    setError(null);
    try {
      const updatedOrg = await syncWorkspace({
        workspace_id: DEFAULT_WORKSPACE_ID,
        layer_id: selectedLayerIds.size === 1 ? [...selectedLayerIds][0] : 'default',
        bucket: 'terraform-state-lume',
        object: 'terraform.tfstate',
      });
      setOrg(updatedOrg);
      // Refresh workspace metadata to get updated layer statuses
      const ws = await fetchWorkspace(DEFAULT_WORKSPACE_ID);
      setWorkspace(ws);
    } catch (err) {
      console.error('Sync failed:', err);
      setError('Sync failed. Please check backend logs.');
    } finally {
      setIsSyncing(false);
    }
  };

  const totalProjects = countProjectsDeep(org);
  const cleanLayers = workspace.layers.filter(l => l.status === 'clean').length;
  const driftedLayers = workspace.layers.filter(l => l.status === 'drifted').length;

  return (
    <main className="min-h-screen bg-[#020617] text-slate-200 selection:bg-blue-500/30">
      {/* Background Decor */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-blue-600/10 blur-[120px] rounded-full" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-emerald-600/10 blur-[120px] rounded-full" />
      </div>

      {/* Nav */}
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
          {isLoading && (
            <span className="text-[10px] bg-blue-500/10 text-blue-400 px-3 py-1 rounded-full border border-blue-500/20 font-medium animate-pulse">
              Loading…
            </span>
          )}
          {error && (
            <span className="text-[10px] bg-amber-500/10 text-amber-500 px-3 py-1 rounded-full border border-amber-500/20 font-medium">
              {error}
            </span>
          )}
          <button
            onClick={handleSync}
            disabled={isSyncing || isLoading}
            className={`flex items-center gap-2 px-4 py-2 bg-white/5 hover:bg-white/10 rounded-full text-sm font-medium transition-all active:scale-95 border border-white/10 ${(isSyncing || isLoading) ? 'opacity-50 cursor-not-allowed' : ''}`}
          >
            <RefreshCw className={`w-4 h-4 text-emerald-400 ${isSyncing ? 'animate-spin' : ''}`} />
            <span>{isSyncing ? 'Syncing…' : 'Sync'}</span>
          </button>
          <div className="w-10 h-10 rounded-full bg-blue-500/20 border border-blue-500/30 flex items-center justify-center">
            <div className="w-2 h-2 rounded-full bg-blue-400 animate-pulse" />
          </div>
        </div>
      </nav>

      <div className="relative z-10 max-w-7xl mx-auto grid grid-cols-12 gap-8 p-8">
        {/* Left Column — Workspace stats + layer selector */}
        <div className="col-span-12 lg:col-span-3 space-y-6">
          {/* Stats card */}
          <div className="p-6 bg-white/5 border border-white/10 rounded-2xl">
            <div className="flex items-center gap-3 mb-4">
              <Server className="w-5 h-5 text-blue-400" />
              <h3 className="font-semibold text-white">Overview</h3>
            </div>
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <span className="text-sm text-white/50">Total Projects</span>
                <span className="text-lg font-bold text-white">{totalProjects}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-white/50">Layers</span>
                <span className="text-lg font-bold text-white">{workspace.layers.length}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-white/50">Clean</span>
                <span className="text-sm font-semibold text-emerald-400 bg-emerald-400/10 px-2 py-0.5 rounded">{cleanLayers}</span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm text-white/50">Drifted</span>
                <span className="text-sm font-semibold text-amber-400 bg-amber-400/10 px-2 py-0.5 rounded">{driftedLayers}</span>
              </div>
            </div>
          </div>

          {/* Workspace layers panel */}
          <WorkspaceLayers
            workspace={workspace}
            selectedLayerIds={selectedLayerIds}
            selectedWorkspaceIds={selectedWorkspaceIds}
            onToggleLayer={(id) => {
              setSelectedLayerIds(prev => {
                const next = new Set(prev);
                if (next.has(id)) {
                  next.delete(id);
                } else {
                  next.add(id);
                }
                return next;
              });
              setSelectedWorkspaceIds(new Set());
              const layer = workspace.layers.find(l => l.id === id);
              if (layer) setSelectedNode({ type: 'layer', data: layer });
            }}
            onToggleWorkspace={(ws: TerraformWorkspace) => {
              setSelectedWorkspaceIds(prev => {
                const next = new Set(prev);
                if (next.has(ws.id)) {
                  next.delete(ws.id);
                } else {
                  next.add(ws.id);
                }
                return next;
              });
              // Auto-select the parent layer if not already selected
              setSelectedLayerIds(prev => {
                if (!prev.has(ws.layer_id)) return new Set([...prev, ws.layer_id]);
                return prev;
              });
              setSelectedNode({ type: 'tf_workspace', data: ws });
            }}
            onClearSelection={() => {
              setSelectedLayerIds(new Set());
              setSelectedWorkspaceIds(new Set());
              setSelectedNode(null);
            }}
          />
        </div>

        {/* Center — Hierarchy + Detail */}
        <div className="col-span-12 lg:col-span-9 space-y-6">
          {/* Org banner */}
          <div className="flex items-center gap-3 px-5 py-3 bg-white/5 border border-white/10 rounded-2xl">
            <div className="p-2 bg-blue-500/10 rounded-lg border border-blue-500/20">
              <Building2 className="w-5 h-5 text-blue-400" />
            </div>
            <div>
              <p className="text-[10px] uppercase tracking-widest text-white/30 font-semibold">Organization</p>
              <p className="text-base font-semibold text-white">{org.display_name}</p>
            </div>
            <span className="ml-auto text-[10px] font-mono text-white/25">{org.id}</span>
          </div>

          <HierarchyTree
            organization={org}
            selectedNode={selectedNode}
            onSelect={setSelectedNode}
            selectedLayerIds={selectedLayerIds}
            selectedWorkspaceIds={selectedWorkspaceIds}
          />
          <DetailPanel
            node={selectedNode}
            onClose={() => setSelectedNode(null)}
            workspaceId={workspace.id}
            onSelectTfWorkspace={(ws) => setSelectedNode({ type: 'tf_workspace', data: ws })}
          />
        </div>
      </div>
    </main>
  );
}
