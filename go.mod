module github.com/talos-systems/metal-controller-manager

go 1.13

replace github.com/talos-systems/go-smbios => github.com/andrewrynhard/go-smbios v0.0.0-20200203055812-9bd4b6dd8cd2

require (
	github.com/go-logr/logr v0.1.0
	github.com/hashicorp/go-multierror v1.0.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/pin/tftp v2.1.1-0.20200117065540-2f79be2dba4e+incompatible
	github.com/talos-systems/go-smbios v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.0.0-20200202094626-16171245cfb2 // indirect
	golang.org/x/sys v0.0.0-20200212091648-12a6c2dcc1e4
	k8s.io/apimachinery v0.0.0-20190913080033-27d36303b655
	k8s.io/client-go v0.0.0-20190918160344-1fbdaa4c8d90
	sigs.k8s.io/controller-runtime v0.4.0
)
