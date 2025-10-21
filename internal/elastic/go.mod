module github.com/explainiq/agent/internal/elastic

go 1.22

require (
	github.com/elastic/go-elasticsearch/v8 v8.11.1
	github.com/sirupsen/logrus v1.9.3
)

require (
	github.com/elastic/elastic-transport-go/v8 v8.3.0 // indirect
	github.com/stretchr/testify v1.8.3 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace github.com/explainiq/agent => ../../
