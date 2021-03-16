package platform

type Platform interface {
	Check() []Check
	Version() (string, error)
}
