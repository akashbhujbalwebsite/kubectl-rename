# kubectl-rename

A kubectl plugin that safely renames Kubernetes ConfigMaps and Secrets — with a pre-flight permission check, dependency scanner, and `--dry-run` mode.

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
3. Dependency scan    → find Pods/Deployments referencing this resource (read-only)
4. Show rename plan   → print exactly what will happen
5. --dry-run stop     → exit here if dry-run, nothing written
6. Confirm prompt     → ask yes/no (skip with -y)
7. CREATE new         → new resource with same data under new name
8. DELETE old         → remove old resource
9. Warn about refs    → remind user which resources need manual updates
```

## What is scanned for references

| Reference type | ConfigMap | Secret |
|---------------|-----------|--------|
| Pod volume mount | ✓ | ✓ |
| Pod `envFrom` | ✓ | ✓ |
| Pod `env.valueFrom` | ✓ | ✓ |
| Pod `imagePullSecrets` | — | ✓ |
| Deployment volume mount | ✓ | ✓ |
| Deployment `envFrom` | ✓ | ✓ |
| Deployment `imagePullSecrets` | — | ✓ |

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

**No new RBAC types needed.** If you can already `get`, `create`, and `delete` a ConfigMap manually — this tool adds nothing new. It automates what you could already do by hand, just more safely.

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

## Caveats

- References are scanned in the specified namespace only. Cross-namespace references (e.g., a Pod in another namespace using the same ConfigMap name) are not detected.
- The rename is not atomic — there is a brief window between CREATE and DELETE where both names exist. In practice this is milliseconds, but it is not a transaction.
- Deployment/StatefulSet/DaemonSet pods that mount the renamed ConfigMap/Secret will use the old data until their next rollout. The tool warns you but does not trigger a rollout.
- CRDs and other custom resources are not scanned for references in v0.1.

## Releasing a new version

```bash
git tag v0.2.0 && git push origin v0.2.0
```

GitHub Actions builds binaries for all platforms and opens a krew-index PR automatically.

## License

Apache 2.0
