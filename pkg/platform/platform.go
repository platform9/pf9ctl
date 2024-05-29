package platform

var SkipOSChecks bool

type Platform interface {
	Check() []Check
	Version() (string, error)
	CheckExistingInstallation() (bool, error)
	CheckKubernetesCluster() (bool, error)
}
