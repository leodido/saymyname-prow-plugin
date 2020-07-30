module github.com/leodido/saymyname-prow-plugin

go 1.14

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible
	k8s.io/client-go => k8s.io/client-go v0.17.3
)

require (
	github.com/sirupsen/logrus v1.6.0
	k8s.io/test-infra v0.0.0-20200725055416-2388ca759f42
)
