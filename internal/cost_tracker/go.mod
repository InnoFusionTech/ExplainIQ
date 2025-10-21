module github.com/explainiq/agent/internal/cost_tracker

go 1.22

require (
	github.com/explainiq/agent/internal/storage v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.9.3
)

require (
	github.com/stretchr/testify v1.8.3 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/explainiq/agent => ../../
