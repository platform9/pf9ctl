package platform

type Check struct {
	Name      string
	Mandatory bool
	Result    bool
	Err       error
	UserErr   string
}
