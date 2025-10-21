module github.com/explainiq/agent/cmd/orchestrator

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.10
	github.com/go-chi/cors v1.2.1
	github.com/google/uuid v1.6.0
	github.com/sirupsen/logrus v1.9.3
)

require (
	github.com/stretchr/testify v1.8.3 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/explainiq/agent => ../../
