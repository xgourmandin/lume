# Project: Lume - GCP Landing Zone Observer

## 1. Overview
Lume is a specialized internal platform for GCP Platform Teams. It provides a "Single Pane of Glass" for Landing Zones managed by OpenTofu/Terraform.
The goal is to move away from "opaque" state files and provide a human-friendly, hierarchy-aware view of the cloud environment, detect manual "ClickOps" drift, and safely onboard new projects via GitOps.

## 2. Tech Stack

* **Backend**: Golang 1.22+
  * Framework: Mach (High-performance web framework)
    * Docs : https://macha.shabel.me/docs
    * Github : https://github.com/mrshabel/mach
  * Dependency Injection: Uber FX
  * Cloud SDK: Google Cloud Go SDK
* **Frontend**: Next.js (TypeScript) + Tailwind CSS
* **Infrastructure**:
  * Storage: Firestore (Metadata/Tree Cache), BigQuery (Drift Logs), GCS (State Source)
  * Compute: Cloud Run (API & Frontend), Cloud Run Jobs (Scanner)
* **IaC Engine**: OpenTofu (CLI-based execution within containers)

## 3. Core Features (MVP Scope)

### A. The Hierarchy Navigator (Visibility)
* Parse OpenTofu/Terraform `.tfstate` files (JSON) and reconstruct the GCP Resource Hierarchy (Org > Folders > Projects).
* Map Terraform resource addresses to actual GCP resource IDs.
* Provide a searchable, nested Tree View in the UI.

### B. Drift Detection (Observability)
* Scheduled background jobs that run `tofu plan -json`.
* Compare "Desired State" (Code) vs "Actual State" (Cloud).
* UI Status Indicators:
  * **CLEAN**: State matches cloud.
  * **DRIFTED**: Manual changes detected.
  * **ERROR**: Plan failed to execute.

### C. Project Vending Machine (Onboarding)
* A guided form to request a new GCP Project.
* Logic: The backend clones a specified GitHub/GitLab repository, creates a new `.tf` file from a template, and opens a Pull Request.
* No direct 'Apply': All changes must go through the existing CI/CD pipeline.

## 4. Architectural Patterns for the Agent

### Dependency Injection (Uber FX)
All modules (HTTP handlers, Services, Repositories) must be wired using `fx.Provide`. The application lifecycle (Start/Stop) should be managed via `fx.Invoke`.

### API Structure (Mach Framework)
* Use group-based routing (e.g., `/api/v1/hierarchy`, `/api/v1/projects`).
* Middleware for Google Identity (IAP) header validation.

### Data Model (Firestore)
* **Collection**: `workspaces`
  * `id`: string (Tofu workspace name)
  * `last_sync`: timestamp
  * `status`: enum (clean, drifted, error)
  * `hierarchy_cache`: map (nested JSON of the folder/project tree)
* **Collection**: `drift_logs` (Historical record of changes)

## 5. Development Tasks (Prompting Instructions)

> [!IMPORTANT]
> To the Coding Agent: Please follow these specific implementation steps:
> * **Phase 1 (Core)**: Initialize the Go project with `uber-go/fx` and `mach`. Setup the basic HTTP server structure.
> * **Phase 2 (Parser)**: Create a service that accepts a Tofu JSON state and outputs a nested Folder tree structure.
> * **Phase 3 (GCP Integration)**: Implement a client to read objects from GCS buckets.
> * **Phase 4 (Frontend)**: Build a recursive Tree component in Next.js to render the hierarchy.
> * **Phase 5 (GitOps)**: Build the PR generator service using the GitHub/GitLab API.

## 6. Security Requirements
* **Service Accounts**: The scanner requires `roles/viewer` at the Org level and `roles/storage.objectViewer` on the Tofu state bucket.
* **IAP**: The frontend and API must be protected by Google Identity-Aware Proxy. The backend must validate `X-Goog-IAP-JWT-Assertion`.