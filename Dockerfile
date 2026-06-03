# Build the manager binary. This always executes in the native architecture of the building machine.
FROM --platform=$BUILDPLATFORM ghcr.io/cybozu/golang:1.26.3.1_noble@sha256:0da22bb6f9a876d774654892d411131272ae3dd14c530b6b4ad9598b0a74d1da AS builder

COPY ./ .

# Build the binary, cross-compiling if necessary
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH \
	go build -ldflags="-w -s" -o accurate-controller ./cmd/accurate-controller

# the controller image, this is in the target architecture.
FROM scratch
LABEL org.opencontainers.image.source https://github.com/cybozu-go/accurate

COPY --from=builder /work/accurate-controller ./
USER 10000:10000

ENTRYPOINT ["/accurate-controller"]
