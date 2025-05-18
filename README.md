# sporelet

> **Production-ready micro-VMs for AI powered workflows that start in <50ms.**

Sporelet is a Kubernetes-native runtime that snapshots fully-formed _microVMs_ (powered by Firecracker or Cloud Hypervisor) so they can spring to life on demand—perfect for AI agents, bursty serverless tasks, or any workload that should feel _instant_ yet remain strongly isolated.

- **Snapshot-centric workflow** - Build once, _freeze_ the VM at just-the-right moment, push layers to an OCI registry.
- **Layered images** - Layer 0 (kernel+init), Layer 1 (containerd + Docker Compose pre-warmed), optional Layer 2 (model mmap-hot). Publish deltas, not gigabytes.
- **Turborepo monorepo** - Single repo houses the CLI (`sporectl`), the snapshot builder, and the Kubernetes operator.
- **CI in minutes** - GitHub Actions produces golden snapshots on every push and tags them by digest.

---

## Why Sporelet?

| Pain today                                            | Sporelet's answer                             |
| ----------------------------------------------------- | --------------------------------------------- |
| 400-900ms to cold-start even the lightest Kata pod    | ≤50ms resume from snapshot                    |
| Double network & storage setup (host + nested Docker) | Compose is pre-warmed _inside_ the snapshot   |
| Huge model weights slow AI pods                       | mmap-hot models baked into Layer 2            |
| VM snapshots are clunky to ship                       | OCI artifact layers - works with any registry |

---

## Architecture at a glance

```text
┌──────────────── Kubernetes ────────────────┐
│  apiVersion: sporelet.ai/v1alpha1         │
│  kind: Sporelet                           │
│  …                                        │
└────────────────────────────────────────────┘
          │ (custom controller)
          ▼
┌──────── Worker node ──────────────────────────────────────┐
│ containerd 1.8 + spore-snapshotter                       │
│  ├─ spore-shim (Firecracker) → Layer 0/1/2 snapshot      │
│  └─ spore-shim (Cloud-Hypervisor) → GPU snapshot         │
└───────────────────────────────────────────────────────────┘
```

Snapshots live in a **node-local reflink cache**; KSM deduplicates identical pages across microVMs so you can pack hundreds of dormant agents per node.

---

## Directory layout

```text
.
├── apps/
│   ├── sporectl/              # CLI for humans & CI
│   └── snapshot-builder/      # Builds Layer 0/1/2 images
├── packages/
│   ├── spore-fc-tools/        # Go lib wrapping Firecracker API + ORAS push
│   └── compose-preheater/     # Warms Docker Compose before snapshot
├── infra/
│   └── dev-vm.Dockerfile      # Reproducible build env
├── .github/workflows/
│   └── sporelet-snapshot.yml  # Golden snapshot CI
├── turbo.json                 # Turborepo pipeline
├── package.json               # pnpm workspaces root
└── README.md                  # ← you are here
```

---

## 🚀 Quickstart

> **Prereqs:** Linux host with KVM, Node ≥ 20, Go ≥ 1.22, pnpm ≥ 9, and `firecracker` in `$PATH`.

```bash
# 1. clone & bootstrap
$ git clone https://github.com/quinnovator/sporelet.git && cd sporelet
$ pnpm install

# 2. build a local golden snapshot (Layer 1)
$ docker build -f apps/snapshot-builder/Dockerfile -t sporelet-builder .
$ docker run --rm --privileged -v $PWD/dist:/snapshot sporelet-builder

# 3. push to GitHub Container Registry (or any OCI registry)
$ export OCI_REF=ghcr.io/quinnovator/sporelet/layer1:dev
$ oras push $OCI_REF \
    --artifact-type application/vnd.firecracker.layer.v1 \
    dist/layer1.mem dist/layer1.vmstate dist/layer1.config

# 4. deploy to a dev cluster (K3s + KVM recommended)
$ kubectl apply -f k8s/sporelet-operator.yaml   # coming soon
$ kubectl apply -f examples/hello-sporelet.yaml  # points at your $OCI_REF
```

A _ready_ event should appear in under 50 ms in the operator logs. 🔥

### Docker-based build

A reproducible toolchain is provided via `infra/dev-vm.Dockerfile`.

```bash
$ docker build -f infra/dev-vm.Dockerfile -t sporelet-dev .
$ docker run --rm -it --privileged -v $PWD:/workspace sporelet-dev
```

Run the snapshot commands from inside the container.


---

## 🔧 Turborepo tasks

| Task              | Purpose                              | Outputs          |
| ----------------- | ------------------------------------ | ---------------- |
| `build`           | compile Go packages                  | _none_           |
| `snapshot:layer0` | build virt kernel + rootfs           | `dist/layer0/**` |
| `snapshot:layer1` | run Compose pre-heater & snapshot VM | `dist/layer1/**` |
| `snapshot:push`   | push snapshots to registry           | _none_           |
| `snapshot:ci`     | layer1 → push (used in CI)           | _none_           |

---

## 🤖 GitHub Actions

`.github/workflows/sporelet-snapshot.yml`

- triggers on every push to `main`
- runs `pnpm turbo run snapshot:ci`
- uploads artifacts and pushes to `ghcr.io/quinnovator/sporelet/layer1:<sha>`

OIDC-based auth means **no Docker passwords** in secrets.

---

## 🧑‍💻 Development

```bash
# Rebuild CLI & snapshot tools on change
$ pnpm turbo run build --watch

# Run integration test (boots a microVM in CI mode)
$ pnpm test
```

### Extending

- **GPU snapshots** - switch shim binary to Cloud-Hypervisor, add `vfio-pci` devices.
- **Layer 2 pipeline** - after your model server mmap-loads weights, call `sporectl snapshot diff` to capture a third layer.
- **Operator scheduling hints** - co-locate Sporelets sharing the same Layer 2 to maximise KSM gains.

---

## 🛤 Roadmap

TBD.

## 🤝 Contributing

PRs welcome! Check out `CONTRIBUTING.md` for branch naming, DCO sign-offs, and our lightweight RFC process.

### Maintainers

| GitHub       | Role              |
| ------------ | ----------------- |
| @quinnovator | Lead architect    |
| @quinnovator | Snapshot pipeline |
| @quinnovator | Operator & CRD    |

---

## 📜 License

Sporelet is released under the **Apache License 2.0** - see `LICENSE` for details.
