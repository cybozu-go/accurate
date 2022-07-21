# Build the manager binary
FROM quay.io/cybozu/golang:1.18-focal as builder

COPY ./ .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o accurate-controller ./cmd/accurate-controller

# the controller image
FROM scratch
LABEL org.opencontainers.image.source https://github.com/cybozu-go/accurate

COPY --from=builder /work/accurate-controller ./
USER 10000:10000

ENTRYPOINT ["/accurate-controller"]
