# Build the manager binary. This always executes in the native architecture of the building machine.
FROM --platform=$BUILDPLATFORM ghcr.io/cybozu/golang:1.26.10.2_noble@sha256:8d900a9ee78c02dbe4e2d2e23800c4d21714dfaf8c679935afa668ca0993a0e4 AS builder

ENV GOPROXY=https://golang.flatt.tech
ENV NETRC=/run/secrets/netrc

# Download modules only here, with the optional secret
COPY go.mod go.sum ./
RUN --mount=type=secret,id=netrc,target=/run/secrets/netrc,required=false \
    go mod download

COPY ./ .

# Build the binary, cross-compiling if necessary
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH GOPROXY=off \
	go build -ldflags="-w -s" -o accurate-controller ./cmd/accurate-controller

# the controller image, this is in the target architecture.
FROM scratch
LABEL org.opencontainers.image.source=https://github.com/cybozu-go/accurate

COPY --from=builder /work/accurate-controller ./
USER 10000:10000

ENTRYPOINT ["/accurate-controller"]
