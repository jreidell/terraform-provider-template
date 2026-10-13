default: fmt vet test

build:
	go build -v ./...

install:
	go install -v ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

# Unit tests only; acceptance tests are skipped unless TF_ACC is set.
test:
	go test -v -cover -timeout=120s ./...

# Acceptance tests run real Terraform against an in-process provider.
testacc:
	TF_ACC=1 go test -v -cover -timeout=10m ./...

# Regenerates docs/ from the schema descriptions and examples/.
docs:
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.25.0 generate --provider-name template

.PHONY: default build install fmt vet test testacc docs
