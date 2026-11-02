FROM gsoci.azurecr.io/giantswarm/alpine:3.24.3

RUN apk add --no-cache ca-certificates

ARG TARGETARCH
COPY app-operator-linux-${TARGETARCH} /app-operator

ENTRYPOINT ["/app-operator"]
