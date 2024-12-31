ARG BINARY_NAME=bubo-tool-gen

FROM golang:latest AS builder

WORKDIR /workspace

ENV CGO_ENABLED=1
RUN go env -w GOCACHE=/go-cache

ARG BINARY_NAME

RUN --mount=type=cache,target=/go/pkg/mod \
  --mount=type=cache,target=/go-cache \
  --mount=type=bind,target=/workspace \
  go install ./cmd/${BINARY_NAME}

FROM gcr.io/distroless/base

ARG BINARY_NAME

COPY --from=builder /go/bin/${BINARY_NAME} /usr/bin/${BINARY_NAME}

ENTRYPOINT ["/usr/bin/${BINARY_NAME}"]
CMD ["--help"]
