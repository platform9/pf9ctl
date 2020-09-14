module github.com/platform9/pf9ctl

go 1.14

require (
	github.com/erwinvaneyk/cobras v0.0.0-20200914200705-1d2dfabe2493
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/spf13/cobra v1.0.0
	github.com/spf13/viper v1.6.3
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20200220183623-bac4c82f6975
	k8s.io/apimachinery v0.0.0-20200713125709-8e7d6bb9bd6d
	k8s.io/client-go v1.5.1
)

// Keep Kubernetes dependencies in sync between all components (e.g., Sunpike).
replace (
	k8s.io/api => k8s.io/api v0.0.0-20200713130235-be360156aa6a
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20200713125709-8e7d6bb9bd6d
	k8s.io/client-go => k8s.io/client-go v0.0.0-20200713130841-505a1f443178
)
