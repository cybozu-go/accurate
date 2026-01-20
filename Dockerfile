# Build the manager binary. This always executes in the native architecture of the building machine.
FROM --platform=$BUILDPLATFORM ghcr.io/cybozu/golang:1.24-jammy@sha256:049bd8cc3c3bce0ca2981e679a7f28bb7db698353bbbed6a0413be89a6f18e6a AS builder

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
