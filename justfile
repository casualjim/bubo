tool_names := `ls cmd`

default: test

# Run all linters
lint:
  @pre-commit run --all-files

# Run all tests
test:
  @gotestsum --format testdox -- -race -count=1 -v ./...

# Build all tools
build: build-all

# Generate build targets for each tool
build-all:
  @for tool in {{tool_names}}; do \
    echo "Building $tool"; \
    just build-single-tool $tool; \
  done

# Build individual tool - this is used by the build-all recipe
build-single-tool tool:
  @docker buildx build \
    --platform linux/amd64,linux/arm64 \
    --build-arg BINARY_NAME={{tool}} \
    --tag casualjim/{{tool}}:latest \
    .

run-example example:
  @go run ./examples/{{example}}
