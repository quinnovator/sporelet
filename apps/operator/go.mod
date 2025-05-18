module github.com/quinnovator/sporelet/apps/operator

go 1.21

require (
    github.com/quinnovator/sporelet/packages/fc-snapshot-tools v0.0.0
    sigs.k8s.io/controller-runtime v0.16.3
)

replace github.com/quinnovator/sporelet/packages/fc-snapshot-tools => ../../packages/fc-snapshot-tools
