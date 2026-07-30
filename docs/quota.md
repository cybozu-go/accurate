# Design Doc: Hierarchical Resource Quota for Accurate

**Status:** Draft · **Author:** @mhkarimi1383 · **Last updated:** 2026-07-30

---

## 1. Background & Motivation

[Accurate](https://github.com/cybozu-go/accurate) is a Kubernetes controller for soft multi-tenancy that (1) propagates namespace-scoped resources between namespaces and (2) lets tenant users create/delete sub-namespaces. Namespace trees are encoded purely with labels — `accurate.cybozu.com/type=root` marks a root, `accurate.cybozu.com/parent=<name>` marks a sub-namespace — and the `SubNamespace` CRD (`accurate.cybozu.com/v2`) is the user-facing handle for creating children. Crucially, Accurate does **not** use the `SubNamespace` resource to discover parent/child relationships; it walks the `parent` label so trees can be restructured freely. 【turn1fetch0】【turn2fetch1】【turn5fetch0】

Resource propagation is opt-in: a resource annotated `accurate.cybozu.com/propagate=create|update` is copied from parent to sub-namespaces, and copies are stamped with `accurate.cybozu.com/from=<parent>`. The reconcilers are `NamespaceReconciler`, `SubNamespaceReconciler`, and one `PropagateController` per watched GVK configured in `config.yaml` (`watches`, `labelKeys`, `annotationKeys`, …). 【turn1fetch1】【turn3fetch0】

**The gap.** A native `ResourceQuota` can be propagated today, but propagation only *copies the same object* into each child. Each child then independently enforces the parent's `hard` values, so the sum of children can vastly exceed the parent's intent — the opposite of a budget. Native `ResourceQuota` is also strictly per-namespace: `kube-controller-manager`'s quota controller only accounts usage in the namespace where the quota lives, with no subtree roll-up. 【turn2search10】【turn2search11】

We need a quota primitive that bounds the **aggregate allocation of a namespace *and its entire sub-namespace tree***, with explicit support for "no limit" nodes and arbitrary tree depth.

## 2. Goals & Non-Goals

**Goals**

- Bound total resource allocation (CPU, memory, and optionally pods/PVCs/services/configmaps/etc.) of a namespace *plus all of its descendants*.
- Allow any node in the tree to declare "no limit" while still contributing its real usage to every ancestor's aggregate.
- Preserve Accurate's opt-in, label-driven tree model; work with `kubectl accurate sub create/move/cut/graft`.
- Surface live aggregate usage and remaining capacity in a `status` field per node.
- Handle tree mutations (reparenting, cascading delete, root conversion) consistently.

**Non-Goals**

- Replacing native `ResourceQuota` for non-hierarchical use cases.
- Hard multi-tenancy isolation (network, node-level, sandboxed runtimes).
- GPU/device quota accounting beyond what `ResourceQuota` already supports (we mirror its resource list semantics).
- Cross-cluster quota federation.

## 3. Use Cases

The canonical tree from the request:

```
Namespace a  (10 cores, 5 GiB)              ← root, capped
└── Namespace b  (5 cores, 2.5 GiB)         ← sub, capped (≤ a)
    └── … continue the tree

Namespace c  (no limit)                      ← root, uncapped
├── Namespace d  (2.5 cores, 1 GiB)          ← sub, capped
└── Namespace e  (no limit)                  ← sub, uncapped
```

Semantics this must satisfy:

| Node | Own HRQ hard | Subtree aggregate cap (effective) |
|------|--------------|-----------------------------------|
| `a`  | cpu=10, mem=5Gi | min(10/5Gi) over {a, b, …} |
| `b`  | cpu=5, mem=2.5Gi | min(5/2.5Gi, ancestor 10/5Gi) over {b, …} |
| `c`  | none | ∅ (unlimited) over {c, d, e} |
| `d`  | cpu=2.5, mem=1Gi | 2.5/1Gi over {d} |
| `e`  | none | ∅ (unlimited); only ancestor is `c` (also unlimited) |

A Pod scheduled into `d` is bounded by `d`'s 2.5/1Gi. A Pod into `e` is bounded by nothing (both `e` and `c` are unlimited). A Pod into `b` is bounded by `b`'s 5/2.5Gi **and** by `a`'s 10/5Gi — i.e., if `a`'s subtree is already at 8 cores, `b` can only admit 2 more cores even though its own HRQ still says 5.

## 4. Proposed Design

### 4.1 New CRD: `HierarchicalResourceQuota` (HRQ)

Introduce a new namespace-scoped CRD in the `accurate.cybozu.com` API group. An HRQ placed in namespace N declares the hard ceiling for **N's entire subtree** (N itself + all transitive descendants). Presence of an HRQ is opt-in; namespaces without one are "no limit" nodes.

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: hierarchicalresourcequotas.accurate.cybozu.com
spec:
  group: accurate.cybozu.com
  names:
    kind: HierarchicalResourceQuota
    listKind: HierarchicalResourceQuotaList
    singular: hierarchicalresourcequota
    plural: hierarchicalresourcequotas
    shortNames: [hrq]
  scope: Namespaced
  versions:
    - name: v2
      served: true
      storage: true
      subresources:
        status: {}
      schema:
        openAPIV3Schema:
          type: object
          x-kubernetes-preserve-unknown-fields: true   # mirror ResourceQuota corev1.ResourceList
          properties:
            spec:
              type: object
              x-kubernetes-preserve-unknown-fields: true
              properties:
                hard:
                  type: object
                  description: "Hard limits for the namespace subtree (corev1.ResourceList)."
                  x-kubernetes-preserve-unknown-fields: true
                scopes:
                  type: array
                  items: { type: string }
                  description: "Mirrors corev1.ResourceQuotaScope; optional."
                scopeSelector:
                  type: object
                  x-kubernetes-preserve-unknown-fields: true
            status:
              type: object
              x-kubernetes-preserve-unknown-fields: true
              properties:
                hard:
                  type: object
                  x-kubernetes-preserve-unknown-fields: true
                used:
                  type: object
                  x-kubernetes-preserve-unknown-fields: true
                subtreeUsed:
                  type: object
                  description: "Aggregate usage across this namespace and all descendants."
                  x-kubernetes-preserve-unknown-fields: true
                remaining:
                  type: object
                  description: "hard − subtreeUsed, clamped at 0."
                  x-kubernetes-preserve-unknown-fields: true
                children:
                  type: array
                  description: "Direct sub-namespaces accounted in subtreeUsed."
                  items: { type: string }
                observedDescendants:
                  type: array
                  items: { type: string }
                conditions:
                  type: array
                  items:
                    type: object
                    x-kubernetes-preserve-unknown-fields: true
```

We intentionally mirror `corev1.ResourceQuotaSpec` (`hard`, `scopes`, `scopeSelector`) so users fluent with native `ResourceQuota` transfer directly, and so we can reuse `k8s.io/api/core/v1.ResourceList` and the same resource-name grammar (`requests.cpu`, `limits.memory`, `pods`, `persistentvolumeclaims`, …). 【turn2search10】

### 4.2 Semantics

1. **Subtree aggregate.** `status.subtreeUsed[N] = used[N] + Σ subtreeUsed[child]` for every direct child `c` whose `accurate.cybozu.com/parent` label points at this namespace. A node with no HRQ still reports `status.subtreeUsed` (computed by the controller) so ancestors can roll it up, but exposes no `hard`.
2. **Effective ceiling.** For a namespace N, the effective ceiling is `min(N.spec.hard, min over ancestors A of A.spec.hard)` for each resource key. Ancestors without an HRQ (or without that key) impose no constraint for that key.
3. **"No limit".** A namespace with no HRQ object, or an HRQ with an empty `spec.hard`, is unlimited. It is still counted into every ancestor's `subtreeUsed`.
4. **Relationship to native `ResourceQuota`.** Native RQs and HRQs coexist. To avoid double accounting, an HRQ-managed resource key in a namespace **should not** also be enforced by a native `ResourceQuota` for the same key; the controller annotates any native RQ it owns with `accurate.cybozu.com/owned-hrq: <name>` and will not touch others. Users may keep native RQs for keys *not* covered by any HRQ.
5. **Propagation.** An HRQ is **not** a propagated resource. It is a policy object. Putting `accurate.cybozu.com/propagate` on it has no effect — this is enforced by the propagator ignoring the `HierarchicalResourceQuota` GVK (it is not listed in `config.yaml.watches`).

### 4.3 Controller architecture

Two new reconcilers join the existing three (`NamespaceReconciler`, `SubNamespaceReconciler`, `PropagateController`): 【turn1fetch0】

- **`HierarchicalResourceQuotaReconciler`** — owns the HRQ object: recomputes `status.used` (via shared quota evaluators), `status.subtreeUsed`, `status.remaining`, and the `children`/`observedDescendants` lists. Triggers re-enqueue of the parent HRQ (and ancestors) when subtree membership changes.
- **`HRQUsageReconciler`** (a `PropagateController`-style multi-GVK watcher) — watches the same GVKs the core `resourcequota` controller does (Pod, PVC, Service, ConfigMap, Secret, RC, RS, Deployment, StatefulSet, …, plus `ResourceQuota` itself) but only in namespaces that are part of an HRQ-managed tree. On any change it enqueues the HRQ in that namespace and walks up the `parent` chain.

A small helper, `pkg/hrq/tree.go`, provides:

- `Descendants(ctx, ns) ([]string, error)` — BFS over namespaces where `accurate.cybozu.com/parent` chains down from `ns`.
- `Ancestors(ctx, ns) ([]string, error)` — walk `parent` label up to a root.
- `EffectiveHard(ctx, ns) (corev1.ResourceList, error)` — `min` over `ns` and ancestors.

Tree discovery reuses Accurate's existing label convention (`accurate.cybozu.com/parent`, `accurate.cybozu.com/type=root`) so `sub move`/`cut`/`graft` automatically re-shape accounting without any new CLI. 【turn2fetch1】【turn5fetch0】

### 4.4 Enforcement: admission webhook

Accurate's design notes state *"No webhooks for propagated resources"* — the rationale being (a) any namespace-scoped resource can be propagated, so a Pod webhook risks chicken-and-egg bootstrapping, and (b) Accurate should not surprise users who didn't ask for limits. 【turn1fetch0】 HRQ enforcement is a different concern: it is an **opt-in quota policy**, structurally equivalent to native `ResourceQuota` (which is itself an admission plugin). The webhook therefore:

- Is registered with `namespaceSelector` matching `accurate.cybozu.com/hrq-managed=true` (a label the controller sets on every namespace that is, or descends from, an HRQ-bearing namespace). Clusters without HRQs pay zero webhook cost.
- Uses `failurePolicy=Fail` only for the quota-scoped create/update verbs; `failurePolicy` is configurable.
- Intercepts the same verbs/GVKs the core `resourcequota` admission plugin does (Pod/PVC/PV-claim/Service/ConfigMap/…).
- On admission, computes the projected `subtreeUsed` for each ancestor HRQ *as if the request were admitted*, and rejects with a `Forbidden` error citing the first HRQ that would be breached and the offending resource key.

This keeps the model precise: the webhook is the single source of truth for "would this create exceed the subtree cap?" The `status` fields are advisory/observability and may lag briefly under contention; the webhook is authoritative.

The alternative — dynamically rewriting native `ResourceQuota.spec.hard` to the "remaining" capacity per namespace — is discussed in §10 and rejected as the default because "remaining" depends on sibling usage and causes `hard` thrash plus permanent over-admission states for already-running workloads.

### 4.5 Reconciliation flow

```mermaid
flowchart TD
    A[Pod/PVC/etc. change in namespace N] --> B[HRQUsageReconciler enqueues HRQ in N]
    B --> C{HRQ exists in N?}
    C -- no --> D[Recompute status.subtreeUsed for N<br/>(no hard) and bubble to parent]
    C -- yes --> E[Recompute status.used for N via quota evaluators]
    E --> F[Fetch all descendants, sum their status.subtreeUsed]
    F --> G[status.subtreeUsed = used + Σ children.subtreeUsed]
    G --> H[status.remaining = max(0, hard − subtreeUsed)]
    H --> I[Enqueue parent HRQ along accurate.cybozu.com/parent]
    I --> J{Reached root?}
    J -- no --> E
    J -- yes --> K[Done; status committed]

    L[Admission: create Pod in N] --> M[Webhook: walk Ancestors(N) ∪ {N}]
    M --> N2[For each HRQ with hard: project subtreeUsed += request]
    N2 --> O{Any breach?}
    O -- yes --> P[Reject 403 with HRQ name + key]
    O -- no --> Q[Allow]
```

### 4.6 Reconciliation walk-through with the example tree

Assume usage at a point in time: `a` runs 3 cores/1.5Gi of its own pods; `b` runs 4 cores/2Gi; `d` runs 1 core/0.5Gi; `c` and `e` empty.

| Namespace | own `used` | `subtreeUsed` | `hard` | `remaining` |
|-----------|-----------|---------------|--------|-------------|
| `a`  | cpu=3, mem=1.5Gi | cpu=7 (3+4), mem=3.5Gi (1.5+2) | cpu=10, mem=5Gi | cpu=3, mem=1.5Gi |
| `b`  | cpu=4, mem=2Gi   | cpu=4, mem=2Gi                 | cpu=5, mem=2.5Gi | cpu=1, mem=0.5Gi |
| `c`  | cpu=0, mem=0     | cpu=1, mem=0.5Gi (from d)      | — (no limit)     | — |
| `d`  | cpu=1, mem=0.5Gi | cpu=1, mem=0.5Gi               | cpu=2.5, mem=1Gi | cpu=1.5, mem=0.5Gi |
| `e`  | cpu=0, mem=0     | cpu=0, mem=0                   | — (no limit)     | — |

A new Pod requesting `cpu=2` lands in `b`. Webhook projects `b.subtreeUsed → 6` (≤5? **no, breach**) → rejected even though `a` still has 3 cores free. This is the intended behavior: `b`'s own slice is the binding constraint.

A new Pod requesting `cpu=2` lands in `a` directly. Webhook projects `a.subtreeUsed → 9` (≤10 ✓) and there is no HRQ in `a`'s ancestors → admitted. `a.remaining` becomes cpu=1.

## 5. Data Model — user-facing YAML

**Root `a` with a subtree budget:**

```yaml
apiVersion: accurate.cybozu.com/v2
kind: HierarchicalResourceQuota
metadata:
  name: budget
  namespace: a
spec:
  hard:
    requests.cpu: "10"
    requests.memory: 5Gi
    limits.cpu: "20"
    limits.memory: 10Gi
    pods: "100"
    persistentvolumeclaims: "20"
```

**Sub-namespace `b` carved out of `a`:**

```yaml
apiVersion: accurate.cybozu.com/v2
kind: SubNamespace
metadata:
  name: b
  namespace: a        # parent
---
apiVersion: accurate.cybozu.com/v2
kind: HierarchicalResourceQuota
metadata:
  name: budget
  namespace: b
spec:
  hard:
    requests.cpu: "5"
    requests.memory: 2.5Gi
```

**Root `c` with no limit (simply omit the HRQ), and capped child `d`:**

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: c
  labels:
    accurate.cybozu.com/type: root
---
apiVersion: accurate.cybozu.com/v2
kind: SubNamespace
metadata:
  name: d
  namespace: c
---
apiVersion: accurate.cybozu.com/v2
kind: SubNamespace
metadata:
  name: e
  namespace: c
---
apiVersion: accurate.cybozu.com/v2
kind: HierarchicalResourceQuota
metadata:
  name: budget
  namespace: d
spec:
  hard:
    requests.cpu: "2.5"
    requests.memory: 1Gi
# No HRQ in c, no HRQ in e → both "no limit"
```

The `SubNamespace` CRD's `spec.labels`/`spec.annotations` continue to work for label propagation; no change to that API is required. 【turn2fetch1】

## 6. Reconciliation algorithm (pseudocode)

```
// HierarchicalResourceQuotaReconciler.Reconcile(req)
func Reconcile(req):
    hrq := get(req.NamespacedName)
    ns  := req.Namespace

    // 1. own usage (re-evaluate via shared quota evaluator, same logic as
    //    k8s.io/kubernetes/pkg/quota evaluator registry)
    ownUsed := evaluator.Usage(ns)

    // 2. descendants' aggregate
    desc := tree.Descendants(ns)         // BFS over accurate.cybozu.com/parent
    subtreeUsed := copy(ownUsed)
    for child in directChildren(ns):
        childHRQ := get(child + "/budget")
        if childHRQ != nil:
            add(subtreeUsed, childHRQ.status.subtreeUsed)
        else:
            // child has no HRQ object but controller still maintains a
            // shadow status via the NamespaceReconciler annotation:
            //   accurate.cybozu.com/hrq-subtree-used=<json>
            add(subtreeUsed, shadowSubtreeUsed(child))

    // 3. commit status
    hrq.status.used         = ownUsed
    hrq.status.subtreeUsed  = subtreeUsed
    hrq.status.hard         = hrq.spec.hard
    hrq.status.remaining    = clampZero(sub(hard, subtreeUsed))
    hrq.status.children     = directChildren(ns)
    hrq.status.observedDescendants = desc
    updateStatus(hrq)

    // 4. bubble up to parent so its subtreeUsed refreshes
    if parent := parentLabel(ns); parent != "":
        enqueue(parent + "/budget")   // or shadow enqueue if parent has no HRQ

    // 5. manage the hrq-managed label on this namespace
    setLabel(ns, "accurate.cybozu.com/hrq-managed", "true")
```

**Tree-mutation handling.** `NamespaceReconciler` already reconciles on `accurate.cybozu.com/parent` changes. We extend it to, on any parent/type change, enqueue the HRQ (or shadow) for both the old and new parent so `subtreeUsed` is recomputed top-to-bottom. Because aggregation bubbles upward one level per reconcile, a reparent at depth *d* triggers *d* chained requeues; we cap requeue depth and fall back to a periodic full resync (default 60 s) to guarantee eventual convergence. This mirrors how the existing propagator already re-converges after `sub move`. 【turn1fetch1】【turn2fetch1】

**Cascading delete.** Accurate blocks deletion of a namespace that has sub-namespaces unless cascading is enabled. 【turn1fetch0】 When a subtree is pruned, the parent HRQ reconcile re-sums the (now smaller) descendant set; `remaining` grows accordingly. No special delete logic is needed beyond enqueuing the parent.

## 7. Enforcement details

- **Webhook object:** `ValidatingWebhookConfiguration accurate-hrq-validator`, scoped via `namespaceSelector: accurate.cybozu.com/hrq-managed=true`. Rules mirror the core `resourcequota` admission plugin's resource list (Pod, PVC, Service, ConfigMap, Secret, RC, RS, Deployment, StatefulSet, DaemonSet, Job, CronJob, …) on `create`/`update` (not `delete` — quota releases happen asynchronously via the usage reconciler).
- **Projection logic:** for the admitted object, compute its `corev1.ResourceList` contribution using the same evaluator registry, add to the namespace's current `subtreeUsed`, then for each ancestor HRQ (and the namespace's own) check `subtreeUsed + delta ≤ hard` per key. Reject on the first breach, naming the HRQ, the namespace, and the key.
- **Race window:** between webhook accept and usage reconciler update, two near-simultaneous creates can both pass. This is identical to native `ResourceQuota`'s well-known best-effort race and is acceptable for soft multi-tenancy. `status` is advisory; the webhook is the gate.
- **PriorityClass / scopes:** `spec.scopes` and `spec.scopeSelector` are forwarded to the evaluator verbatim, so e.g. `Terminating` scope or priority-class-gated quotas behave as in native RQ. 【turn2search10】
- **Failure mode:** if the webhook is unavailable, `failurePolicy=Fail` blocks creates in HRQ-managed namespaces (fail-closed for quota safety). This is configurable to `Ignore` for environments that prefer availability over strictness, with the caveat that overshoot can occur until the webhook recovers.

## 8. Edge cases

| Scenario | Handling |
|----------|----------|
| Child HRQ `hard` > parent HRQ `hard` | Allowed at the API level (admin error). Effective cap is the parent's; `status.remaining` for the child is clamped by the parent. A `conditions` entry `OverProvisioned` is set on the child. |
| Reparenting (`sub move`) that moves a capped subtree under an unlimited root | Old parent's `subtreeUsed` drops; new parent (unlimited) absorbs. Both enqueued. |
| `sub cut` (sub → root) | The cut node becomes a new tree root; its `subtreeUsed` is recomputed from its own descendants only. |
| Cycles | Already prevented by Accurate's Namespace validating webhook. 【turn1fetch0】 HRQ controller additionally guards with a visited-set during BFS. |
| HRQ in a non-root, non-sub namespace | Rejected by a new validating rule on the HRQ CRD (the namespace must carry `type=root` or `parent`). |
| Two HRQs in one namespace | Rejected: one HRQ named `budget` per namespace (enforced by a validating webhook on HRQ create). Future: allow multiple with disjoint scopes. |
| Resource key in HRQ not understood by evaluators | `conditions: UnknownResource` set; that key is ignored for enforcement but still surfaced in `status.hard`. |
| Very deep trees (≥1000 nodes) | BFS is O(V+E) per reconcile; bubble-up chains are bounded by depth. Full resync is sharded by namespace hash across workers. Documented scalability target: 5 000 namespaces per controller instance. |
| Native `ResourceQuota` already present for same key | Controller annotates it `accurate.cybozu.com/conflicts-with-hrq` and emits an event; enforcement responsibility stays with native RQ until the user removes it. |

## 9. Rollout & migration

1. **CRD install** — `config/crd/bases/hierarchicalresourcequotas.accurate.cybozu.com.yaml`, shipped via the existing Helm chart under a new `hrq.enabled` value (default `false` in v1.x). 【turn2search7】
2. **Controller feature gate** — `--feature-gates=HierarchicalResourceQuota=true`. Off by default; existing Accurate behavior is unchanged when off.
3. **Webhook** — deployed only when the feature gate is on; cert-manager provisions the TLS cert (Accurate already requires cert-manager for its existing webhooks). 【turn2search7】
4. **Migration path** — for tenants currently using propagated native `ResourceQuota`:
   - Remove `accurate.cybozu.com/propagate` from the parent RQ.
   - Create an HRQ in the root with the same `hard`.
   - Create HRQs in each sub-namespace with the desired slice.
   - Delete the now-redundant propagated RQ copies (or leave them; the controller annotates conflicts).
5. **Versioning** — `accurate.cybozu.com/v2` (matches `SubNamespace`). A `v1beta1` preview may be offered first; promotion to `v2` GA after two minor releases of field soak.

## 10. Alternatives considered

| Alternative | How it works | Pros | Cons | Verdict |
|-------------|--------------|------|------|---------|
| **A. Propagate native `ResourceQuota` with sliced `hard`** | Controller writes a per-namespace RQ whose `hard` = that namespace's slice; sum-of-children ≤ parent enforced by a validating webhook on slice changes. | Reuses native RQ admission; no Pod webhook. | Cannot express "parent's own pods share the budget with children"; no borrowing; a child with spare can't use parent headroom; "no limit" nodes need a sentinel RQ. | Rejected — fails the user's "include sub-namespaces" aggregate requirement. |
| **B. Dynamic native RQ `hard` = remaining** | Controller rewrites each namespace's RQ `hard` to `min(own hard, ancestors' remaining)` every reconcile. | No Pod webhook; reuses native enforcement. | `remaining` depends on sibling usage → `hard` thrash; native RQ allows `hard < used` (over-admitted running pods, new creates blocked indefinitely); complex min-over-ancestors per key; confusing UX ("why did my quota drop?"). | Rejected as default; offered as `enforcementMode: native-rewrite` for clusters that forbid Pod webhooks. |
| **C. HNC `HierarchicalResourceQuota` as-is** | Adopt the kubernetes-sigs/hnc HRQ CRD verbatim. | Battle-tested. | HNC's HRQ aggregates only along HNC's own hierarchy (HNC `HierarchicalConfiguration`), which Accurate deliberately does not use (Accurate is opt-in, HNC is opt-out; Accurate propagates labels/annotations, HNC doesn't). 【turn1fetch0】 Pulling in HNC's hierarchy model would contradict Accurate's design. Coupling two controllers' tree opinions invites divergence. | Rejected. |
| **D. Per-namespace native RQ only (no hierarchy)** | One RQ per namespace, no subtree aggregation. | Trivial. | Does not satisfy the requirement. | Rejected. |
| **E. HRQ + admission webhook (this proposal)** | New CRD + usage reconciler + scoped validating webhook. | True aggregate; precise; "no limit" natural; fits Accurate's label tree. | Adds a Pod-scoped webhook (opt-in, namespace-labeled). | **Selected.** |

## 11. Open questions

1. **Borrowing.** Should a child be allowed to temporarily exceed its `hard` if the parent has spare, with a `conditions: Borrowing` flag and a payback window? (Out of scope for v1; tracked as future work.)
2. **Quota for `object` counts** (ConfigMaps, Secrets). Native RQ supports these; HRQ will inherit support, but the webhook projection cost grows. Need a benchmark.
3. **Cross-root aggregation.** If two roots should share a cluster-wide budget, do we need a cluster-scoped `ClusterHierarchicalResourceQuota`? Defer.
4. **Interaction with `LimitRange`.** `LimitRange` defaults inflate Pod requests at admission; the webhook must evaluate *post-default* requests, which a `ValidatingWebhook` (post-mutating) sees — confirmed, but worth a test matrix.
5. **Status freshness vs. webhook authority.** Document explicitly that `status` is best-effort and the webhook is the source of truth, to set operator expectations.

## 12. Future work

- `kubectl accurate hrq` subcommands: `set`, `show` (pretty-print tree with per-node `used`/`hard`/`remaining`), `verify` (dry-run a Pod spec against the subtree).
- Borrowing with explicit `spec.maxBorrow` and reclaim.
- Priority-class-aware subtree caps.
- Prometheus metrics: `accurate_hrq_subtree_used`, `accurate_hrq_remaining`, `accurate_hrq_admission_rejections_total`.
- Optional `enforcementMode: native-rewrite` (Alternative B) for clusters that cannot run Pod webhooks.
- Cluster-scoped `ClusterHierarchicalResourceQuota` for cross-root budgets.

---

### References to the Accurate codebase used in this design

- Design notes — opt-in propagation, no webhooks for propagated resources, tree encoded by labels. 【turn1fetch0】
- Reconciliation rules — `SubNamespace`, root, sub-namespace, and watched-resource handling. 【turn1fetch1】
- Labels (`type`, `template`, `parent`) and annotations (`from`, `propagate`) used as the hierarchy substrate. 【turn5fetch0】【turn5fetch1】
- Controller package: `NamespaceReconciler`, `SubNamespaceReconciler`, `PropagateController` (the extension point for new reconcilers). 【turn1fetch0】
- `config.yaml` schema (`watches`, `labelKeys`, …) — HRQ is deliberately *not* added to `watches`. 【turn3fetch0】
- Native `ResourceQuota` API (`k8s.io/api/core/v1`) whose `ResourceList`/scopes semantics HRQ mirrors. 【turn2search10】
