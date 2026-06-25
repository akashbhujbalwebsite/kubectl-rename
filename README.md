# kubectl-rename

A kubectl plugin that safely renames Kubernetes ConfigMaps and Secrets — with a pre-flight permission check, dependency scanner, `--dry-run` mode, and automatic partial-failure recovery.

## Installation

```bash
kubectl krew install rename
```

Or manually:
```bash
curl -L <release-url>/kubectl-rename_linux_amd64.tar.gz | tar xz
chmod +x kubectl-rename && mv kubectl-rename /usr/local/bin/
```

## Usage

```bash
# Rename a ConfigMap
kubectl rename configmap app-config app-config-v2 -n staging

# Rename a Secret
kubectl rename secret db-creds db-creds-v2 -n production

# Preview what will happen — no changes made
kubectl rename configmap app-config app-config-v2 -n staging --dry-run

# Skip confirmation prompt (for scripts/CI)
kubectl rename configmap app-config app-config-v2 -n staging -y
```

## Example Output

```
 ConfigMap Rename Plan
──────────────────────────────────────────────────
 Namespace : staging
 Old name  : app-config
 New name  : app-config-v2
──────────────────────────────────────────────────
 Steps:
   1. GET    configmap/app-config
   2. CREATE configmap/app-config-v2 (same data)
   3. DELETE configmap/app-config
──────────────────────────────────────────────────
 ⚠️  References found (manual update required after rename):
   Deployment/my-app (envFrom in container app)
   Pod/my-app-7d9f8b-xkr9p (volume: cfg-vol)
──────────────────────────────────────────────────
Rename ConfigMap "app-config" to "app-config-v2" in namespace "staging"? (yes/no): yes
✓ Created ConfigMap "app-config-v2"
✓ Deleted ConfigMap "app-config"

Done. ConfigMap renamed: "app-config" → "app-config-v2"

⚠️  Update the following resources to reference the new name:
   Deployment/my-app (envFrom in container app)
   Pod/my-app-7d9f8b-xkr9p (volume: cfg-vol)
```

### When a permission is missing

```
✗ Cannot rename: missing 'delete' permission on configmaps in namespace "staging"
  (you have 'get'+'create' but not 'delete' — partial rename would leave duplicate resources)
```

The tool **never starts the rename** until all three permissions are confirmed. No orphaned resources.

## How it works

```
1. Pre-flight check   → verify get + create + delete before touching anything
2. GET old resource   → read all data, labels, annotations
3. Recovery check     → if new name already exists with identical data, complete the delete and exit
4. Dependency scan    → find Pods/Deployments referencing this resource (read-only)
5. Show rename plan   → print exactly what will happen
6. --dry-run stop     → exit here if dry-run, nothing written
7. Confirm prompt     → ask yes/no (skip with -y)
8. CREATE new         → new resource with same data under new name
9. DELETE old         → remove old resource
10. Warn about refs   → remind user which resources need manual updates
```

## Partial-failure recovery

If the process is interrupted (crash, Ctrl+C, network drop) after CREATE but before DELETE, both names will exist. On the next re-run, the tool detects this automatically:

```
⚠️  Partial rename detected: "app-config-v2" already exists with identical data.
   This looks like a previous rename that was interrupted after CREATE but before DELETE.
   Completing rename by deleting "app-config"...
✓ Deleted ConfigMap "app-config"

Done. ConfigMap renamed: "app-config" → "app-config-v2"
```

**What counts as "identical":** only `data`/`binaryData` is compared — labels and annotations are intentionally excluded. A controller adding `app.kubernetes.io/managed-by: helm` to the new resource after CREATE does not block recovery.

If CREATE succeeds but DELETE fails (e.g., RBAC revoked mid-run, finalizer added):

```
⚠️  Partial rename: "app-config-v2" was created but "app-config" could not be deleted: <reason>
   Re-run this command to finish, or manually delete "app-config"
```

Re-running the same command detects the identical data and completes the delete.

`--dry-run` works on the recovery path too:

```
⚠️  Partial rename detected: "app-config-v2" already exists with identical data.
   [dry-run] Would delete "app-config" to complete the rename. No changes made.
```

## What is scanned for references

| Reference type | ConfigMap | Secret |
|---------------|-----------|--------|
| Pod volume mount | ✓ | ✓ |
| Pod `envFrom` | ✓ | ✓ |
| Pod `env.valueFrom` | ✓ | ✓ |
| Pod `imagePullSecrets` | — | ✓ |
| Pod init containers | ✓ | ✓ |
| Deployment volume mount | ✓ | ✓ |
| Deployment `envFrom` | ✓ | ✓ |
| Deployment `imagePullSecrets` | — | ✓ |
| Deployment init containers | ✓ | ✓ |

References are **read-only** — the scanner never modifies them. You are responsible for updating them after the rename.

## Required permissions (RBAC footprint)

All calls are scoped to the specified namespace. No cluster-wide access required.

| Permission | Why |
|-----------|-----|
| `get` configmaps/secrets | Read existing resource data |
| `create` configmaps/secrets | Create resource under new name |
| `delete` configmaps/secrets | Remove old resource |
| `list` pods | Dependency scan (read-only) |
| `list` deployments | Dependency scan (read-only) |
| `create` selfsubjectaccessreviews | Pre-flight permission check |

**No new RBAC types needed.** If you can already `get`, `create`, and `delete` a ConfigMap manually — this tool adds nothing new. It automates what you could already do by hand, safely.

## Why not just use `kubectl apply`?

```bash
# The manual way — error-prone:
kubectl get configmap app-config -n staging -o yaml > /tmp/cm.yaml
# Edit the name field...
kubectl apply -f /tmp/cm.yaml
kubectl delete configmap app-config -n staging
```

Problems with the manual approach:
- Easy to forget the delete step → leaves orphaned resource
- No dependency check → references silently break
- No confirmation → immediate destructive action
- No dry-run → can't preview safely
- No recovery → if interrupted halfway, no guidance on what state you're in

## Caveats

- References are scanned in the specified namespace only. Cross-namespace references are not detected.
- The rename is not atomic — brief window between CREATE and DELETE. Re-run auto-recovers (see above).
- Workloads referencing the renamed resource continue using old data until their next rollout. The tool warns but does not trigger a rollout.
- StatefulSets, DaemonSets, CronJobs, and CRDs are not scanned for references in v0.1.
- init containers inside Pods and Deployments are scanned.

## Releasing a new version

```bash
git tag v0.2.0 && git push origin v0.2.0
```

GitHub Actions builds binaries for all platforms and opens a krew-index PR automatically.

## License

Apache 2.0
