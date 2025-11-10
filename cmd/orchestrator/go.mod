module github.com/InnoFusionTech/ExplainIQ/cmd/orchestrator

go 1.24.4

toolchain go1.24.10

require (
	github.com/InnoFusionTech/ExplainIQ/internal/adk v0.0.0-00010101000000-000000000000
	github.com/InnoFusionTech/ExplainIQ/internal/auth v0.0.0-00010101000000-000000000000
	github.com/InnoFusionTech/ExplainIQ/internal/brainprint v0.0.0
	github.com/InnoFusionTech/ExplainIQ/internal/cost_tracker v0.0.0-00010101000000-000000000000
	github.com/InnoFusionTech/ExplainIQ/internal/elastic v0.0.0-00010101000000-000000000000
	github.com/InnoFusionTech/ExplainIQ/internal/llm v0.0.0-00010101000000-000000000000
	github.com/InnoFusionTech/ExplainIQ/internal/quota v0.0.0-00010101000000-000000000000
	github.com/InnoFusionTech/ExplainIQ/internal/rate_limiter v0.0.0-00010101000000-000000000000
	github.com/InnoFusionTech/ExplainIQ/internal/storage v0.0.0-00010101000000-000000000000
	github.com/gin-gonic/gin v1.11.0
	github.com/go-chi/chi/v5 v5.0.10
	github.com/go-chi/cors v1.2.1
	github.com/google/uuid v1.6.0
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.11.1
)

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/ai v0.7.0 // indirect
	cloud.google.com/go/auth v0.17.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/firestore v1.19.0 // indirect
	cloud.google.com/go/longrunning v0.7.0 // indirect
	github.com/bytedance/sonic v1.14.0 // indirect
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/cncf/xds/go v0.0.0-20251014123835-2ee22ca58382 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.3.0 // indirect
	github.com/elastic/go-elasticsearch/v8 v8.11.1 // indirect
	github.com/envoyproxy/go-control-plane/envoy v1.35.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.0 // indirect
	github.com/google/generative-ai-go v0.15.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.54.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.63.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.63.0 // indirect
	go.opentelemetry.io/otel v1.38.0 // indirect
	go.opentelemetry.io/otel/metric v1.38.0 // indirect
	go.opentelemetry.io/otel/trace v1.38.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	golang.org/x/arch v0.20.0 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/mod v0.28.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/oauth2 v0.32.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	golang.org/x/tools v0.37.0 // indirect
	google.golang.org/api v0.252.0 // indirect
	google.golang.org/genproto v0.0.0-20251014184007-4626949a642f // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20251014184007-4626949a642f // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251014184007-4626949a642f // indirect
	google.golang.org/grpc v1.76.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/InnoFusionTech/ExplainIQ => ../../

replace github.com/InnoFusionTech/ExplainIQ/internal/adk => ../../internal/adk

replace github.com/InnoFusionTech/ExplainIQ/internal/auth => ../../internal/auth

replace github.com/InnoFusionTech/ExplainIQ/internal/brainprint => ../../internal/brainprint

replace github.com/InnoFusionTech/ExplainIQ/internal/cost_tracker => ../../internal/cost_tracker

replace github.com/InnoFusionTech/ExplainIQ/internal/elastic => ../../internal/elastic

replace github.com/InnoFusionTech/ExplainIQ/internal/llm => ../../internal/llm

replace github.com/InnoFusionTech/ExplainIQ/internal/quota => ../../internal/quota

replace github.com/InnoFusionTech/ExplainIQ/internal/rate_limiter => ../../internal/rate_limiter

replace github.com/InnoFusionTech/ExplainIQ/internal/storage => ../../internal/storage
