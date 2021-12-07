module github.com/gardener/machine-controller-manager-provider-openstack

go 1.16

require (
	github.com/ahmetb/gen-crd-api-reference-docs v0.2.0
	github.com/gardener/gardener v1.35.1
	github.com/gardener/machine-controller-manager v0.42.0
	github.com/golang/mock v1.6.0
	github.com/gophercloud/gophercloud v0.19.0
	github.com/gophercloud/utils v0.0.0-20210720165645-8a3ad2ad9e70
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/code-generator v0.22.2
	k8s.io/component-base v0.22.2
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20210819203725-bdf08cb9a70a
)

replace (
	github.com/gardener/gardener-resource-manager/api => github.com/gardener/gardener-resource-manager/api v0.25.0
	k8s.io/api => k8s.io/api v0.20.6
	k8s.io/apimachinery => k8s.io/apimachinery v0.20.6
	k8s.io/client-go => k8s.io/client-go v0.20.6
	k8s.io/code-generator => k8s.io/code-generator v0.20.6
	k8s.io/component-base => k8s.io/component-base v0.20.6
)
