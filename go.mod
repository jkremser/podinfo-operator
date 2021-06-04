module github.com/jkremser/podinfo-operator

go 1.16

require (
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)

//replace github.com/jkremser/podinfo-operator/api/v1alpha1 => ./podinfo-operator/api/v1alpha1
