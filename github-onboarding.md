# Onboarding a Product on GitHub Actions

This guide explains every parameter a product team must set and the GitHub
Actions workflow template they need to wire up CICD with the Landing Zone.

---

## 1 – Landing Zone configuration (done once by the platform team)

### 1a. `config/main.yaml` – WIF pool kind

The security block must declare `kind: github` and the GitHub organisation name:

```yaml
core:
  security:
    project_id: p-<org>-security-prj-001
    workload_identity_pool:
      kind: github               # ← must be "github" for GitHub Actions
      id: github-wip
      display_name: github-wip
      description: Workload Identity Pool for Products
      organization_id: <github-org>  # e.g. "xgourmandin"
```

> **Effect:** `101-core` creates a WIF pool + provider named `github-actions`
> (issuer `https://token.actions.githubusercontent.com`) and restricts accepted
> tokens to the declared GitHub organisation.

---

### 1b. `config/products/<product-id>.yaml` – product YAML

The product team (or platform team) creates/updates the product YAML file.
The only CICD-specific field is **`projects.cicd.repository`**.

```yaml
product_location: <subsidiary-id>   # e.g. "hes"

projects:

  cicd:
    id: <cicd-project-id>            # e.g. "d-hes-myapp-cicd-prj-001"
    display_name: <cicd-display>     # e.g. "d-hes-myapp-cicd-prj"
    repository: "<github-org>/<github-repo>"   # ← KEY FIELD
    #            Must match the GitHub repository that runs Actions.
    #            Example: "xgourmandin/my-product"
    #            The LZ builds the WIF IAM binding:
    #              principalSet://.../attribute.repository/<github-org>/<github-repo>

  labels:
    team: <team-name>

  environments:
    dev:
      projects:
        - id: d-<org>-<app>-prj-001
          display_name: d-<org>-<app>-prj
    prod:
      projects:
        - id: p-<org>-<app>-prj-001
          display_name: p-<org>-<app>-prj
```

> After the LZ applies this configuration (`make apply PRODUCT_ID=<product-id>`
> in `201-product-envs/`), the following GCP resources are created automatically:
>
> | Resource | Value |
> |---|---|
> | CICD project | `<cicd-project-id>` |
> | CICD service account | `sa-<product-id>-cicd@<cicd-project-id>.iam.gserviceaccount.com` |
> | WIF IAM binding | `principalSet://.../attribute.repository/<github-org>/<github-repo>` |
> | IaC state bucket | `<product-id>-iac-state-<hex>` (in the CICD project) |
> | Env project access | CICD SA has `roles/owner` on every environment project |

---

## 2 – Values the product team must retrieve from the LZ

After the platform team applies the LZ for the product, they communicate
(or the product team reads from Terraform outputs) the following three values:

| Variable | Where to find it | Example |
|---|---|---|
| `WIF_PROVIDER` | Security project → WIF pool → provider resource name | `projects/123456789/locations/global/workloadIdentityPools/github-wip/providers/github-actions` |
| `CICD_SA_EMAIL` | Terraform output / GCP console → CICD project → IAM | `sa-myapp-cicd@d-hes-myapp-cicd-prj-001.iam.gserviceaccount.com` |
| `IAC_STATE_BUCKET` | Terraform output `google_storage_bucket.iac_state.name` | `myapp-iac-state-4a2f` |

Store them as **GitHub repository variables** (or secrets) in the product repo:

```
Settings → Secrets and variables → Actions → Variables
  WIF_PROVIDER      = projects/.../providers/github-actions
  CICD_SA_EMAIL     = sa-myapp-cicd@...iam.gserviceaccount.com
  IAC_STATE_BUCKET  = myapp-iac-state-4a2f
```

> You can also retrieve the WIF provider name with:
> ```bash
> gcloud iam workload-identity-pools providers describe github-actions \
>   --workload-identity-pool=github-wip \
>   --project=<security-project-id> \
>   --location=global \
>   --format="value(name)"
> ```

---

## 3 – GitHub Actions workflow

Copy the template from
[`docs/github-actions-workflow-template.yml`](./github-actions-workflow-template.yml)
into your product repository at `.github/workflows/iac.yml`.

### Key points

```yaml
permissions:
  id-token: write   # ← mandatory – allows the runner to request an OIDC token
  contents: read
```

```yaml
- uses: google-github-actions/auth@v2
  with:
    workload_identity_provider: ${{ vars.WIF_PROVIDER }}
    service_account:            ${{ vars.CICD_SA_EMAIL }}
```

The `auth` action:
1. Requests a GitHub OIDC JWT (short-lived, signed by GitHub).
2. Exchanges it at `https://sts.googleapis.com` using the WIF pool.
3. The WIF provider checks `attribute.repository == "<github-org>/<github-repo>"`.
4. Returns a short-lived GCP access token impersonating `CICD_SA_EMAIL`.

No static service account keys are involved.

```yaml
- name: Terraform Init
  run: |
    terraform init \
      -backend-config="bucket=${{ vars.IAC_STATE_BUCKET }}" \
      -backend-config="prefix=tf/${{ github.repository }}"
```

The IaC state is stored in the LZ-managed bucket inside the product's CICD project.

---

## 4 – Managing multiple environments

### Overview

The three LZ values (`WIF_PROVIDER`, `CICD_SA_EMAIL`, `IAC_STATE_BUCKET`) are
**shared across all environments** — there is one CICD service account and one
state bucket for the whole product, not one per environment.

What changes between environments:

| What varies | How to handle it |
|---|---|
| GCP project IDs | Terraform variable or `.tfvars` file per environment |
| Feature flags, sizing, replica counts | Terraform variable or `.tfvars` file per environment |
| Manual approval before deploy | GitHub Environment protection rules |
| State isolation | State prefix per environment inside the same bucket |

---

### 4a – GitHub Environments (approval gates)

Create one **GitHub Environment** per LZ environment in your repo:
`Settings → Environments → New environment`

| GitHub Environment | Suggested protection rule |
|---|---|
| `dev` | None – auto-deploy |
| `staging` | Required reviewers: 1 |
| `prod` | Required reviewers: 2 + deployment branch rule: `main` only |

The LZ variables (`WIF_PROVIDER`, `CICD_SA_EMAIL`, `IAC_STATE_BUCKET`) can be
set at the **repository level** (shared) or overridden at the environment level
if you ever need per-environment values.

---

### 4b – Environment-specific Terraform variables

The recommended approach is a **`.tfvars` file per environment** committed to
the repository:

```
terraform/
  main.tf
  variables.tf
  envs/
    dev.tfvars
    staging.tfvars
    prod.tfvars
```

Example `envs/dev.tfvars`:
```hcl
gcp_project_id = "d-hes-myapp-prj-001"
region         = "europe-west1"
replica_count  = 1
```

Example `envs/prod.tfvars`:
```hcl
gcp_project_id = "p-hes-myapp-prj-001"
region         = "europe-west1"
replica_count  = 3
```

Then in the workflow, pass the file that matches the target environment:
```yaml
run: terraform apply -var-file="envs/${{ matrix.environment }}.tfvars" -auto-approve
```

Alternatively, use **GitHub Environment variables** (`Settings → Environments →
<env> → Environment variables`) for values you do not want in source control
(e.g. third-party API endpoints):

```yaml
env:
  GCP_PROJECT_ID: ${{ vars.GCP_PROJECT_ID }}   # set per GitHub Environment
```

---

### 4c – Multi-environment workflow

Replace `docs/github-actions-workflow-template.yml` with the multi-environment
version below, or use it as the file placed at `.github/workflows/iac.yml` in
your product repository.

```yaml
name: "IaC – Multi-environment Plan & Apply"

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

permissions:
  id-token: write
  contents: read

env:
  # ── Shared LZ values (repository-level variables) ──────────────────────────
  WIF_PROVIDER:     ${{ vars.WIF_PROVIDER }}
  CICD_SA_EMAIL:    ${{ vars.CICD_SA_EMAIL }}
  IAC_STATE_BUCKET: ${{ vars.IAC_STATE_BUCKET }}
  TF_VERSION: "1.13.0"

# ─────────────────────────────────────────────────────────────────────────────
# PLAN – runs on every pull request across all environments (read-only)
# ─────────────────────────────────────────────────────────────────────────────
jobs:
  plan:
    name: "Plan (${{ matrix.environment }})"
    runs-on: ubuntu-latest
    if: github.event_name == 'pull_request'

    strategy:
      fail-fast: false
      matrix:
        environment: [dev, staging, prod]

    steps:
      - uses: actions/checkout@v4

      - uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ env.WIF_PROVIDER }}
          service_account:            ${{ env.CICD_SA_EMAIL }}

      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: ${{ env.TF_VERSION }}

      - name: Terraform Init
        working-directory: terraform/
        run: |
          terraform init \
            -backend-config="bucket=${{ env.IAC_STATE_BUCKET }}" \
            -backend-config="prefix=tf/${{ github.repository }}/${{ matrix.environment }}"

      - name: Terraform Plan
        working-directory: terraform/
        run: |
          terraform plan \
            -var-file="envs/${{ matrix.environment }}.tfvars"

# ─────────────────────────────────────────────────────────────────────────────
# DEPLOY – sequential promotion: dev → staging → prod
# Each job targets its own GitHub Environment for approval gates.
# ─────────────────────────────────────────────────────────────────────────────
  deploy-dev:
    name: "Deploy → dev"
    runs-on: ubuntu-latest
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    environment: dev     # no protection rule → auto-deploy

    steps:
      - uses: actions/checkout@v4
      - uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ env.WIF_PROVIDER }}
          service_account:            ${{ env.CICD_SA_EMAIL }}
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: ${{ env.TF_VERSION }}
      - name: Terraform Init
        working-directory: terraform/
        run: |
          terraform init \
            -backend-config="bucket=${{ env.IAC_STATE_BUCKET }}" \
            -backend-config="prefix=tf/${{ github.repository }}/dev"
      - name: Terraform Apply
        working-directory: terraform/
        run: terraform apply -var-file="envs/dev.tfvars" -auto-approve

  deploy-staging:
    name: "Deploy → staging"
    runs-on: ubuntu-latest
    needs: deploy-dev
    environment: staging   # requires 1 reviewer

    steps:
      - uses: actions/checkout@v4
      - uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ env.WIF_PROVIDER }}
          service_account:            ${{ env.CICD_SA_EMAIL }}
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: ${{ env.TF_VERSION }}
      - name: Terraform Init
        working-directory: terraform/
        run: |
          terraform init \
            -backend-config="bucket=${{ env.IAC_STATE_BUCKET }}" \
            -backend-config="prefix=tf/${{ github.repository }}/staging"
      - name: Terraform Apply
        working-directory: terraform/
        run: terraform apply -var-file="envs/staging.tfvars" -auto-approve

  deploy-prod:
    name: "Deploy → prod"
    runs-on: ubuntu-latest
    needs: deploy-staging
    environment: prod      # requires 2 reviewers, main branch only

    steps:
      - uses: actions/checkout@v4
      - uses: google-github-actions/auth@v2
        with:
          workload_identity_provider: ${{ env.WIF_PROVIDER }}
          service_account:            ${{ env.CICD_SA_EMAIL }}
      - uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: ${{ env.TF_VERSION }}
      - name: Terraform Init
        working-directory: terraform/
        run: |
          terraform init \
            -backend-config="bucket=${{ env.IAC_STATE_BUCKET }}" \
            -backend-config="prefix=tf/${{ github.repository }}/prod"
      - name: Terraform Apply
        working-directory: terraform/
        run: terraform apply -var-file="envs/prod.tfvars" -auto-approve
```

> **State isolation:** each environment gets its own prefix inside the shared
> bucket, e.g.:
> ```
> gs://<product-id>-iac-state-<hex>/tf/<github-org>/<repo>/dev/terraform.tfstate
> gs://<product-id>-iac-state-<hex>/tf/<github-org>/<repo>/staging/terraform.tfstate
> gs://<product-id>-iac-state-<hex>/tf/<github-org>/<repo>/prod/terraform.tfstate
> ```

---

## 5 – Summary checklist

### Platform team (per product onboarding)
- [ ] `config/main.yaml` has `security.workload_identity_pool.kind: github` and the correct `organization_id`
- [ ] `config/products/<product-id>.yaml` has `projects.cicd.repository: "<github-org>/<github-repo>"`
- [ ] Ran `make apply PRODUCT_ID=<product-id>` in `201-product-envs/`
- [ ] Communicated `WIF_PROVIDER`, `CICD_SA_EMAIL`, `IAC_STATE_BUCKET` to the product team

### Product team
- [ ] Added `WIF_PROVIDER`, `CICD_SA_EMAIL`, `IAC_STATE_BUCKET` as **repository-level** GitHub variables
- [ ] Created **GitHub Environments** (`dev`, `staging`, `prod`) with appropriate protection rules
- [ ] Added environment-specific variables (e.g. `GCP_PROJECT_ID`) to each GitHub Environment if not using `.tfvars` files
- [ ] Created `terraform/envs/<env>.tfvars` for each environment with environment-specific values
- [ ] Copied the workflow template to `.github/workflows/iac.yml`
- [ ] Set `permissions.id-token: write` in the workflow
- [ ] Adjusted `working-directory` in the workflow to point to their Terraform root
- [ ] Verified state prefixes are unique per environment in the bucket

