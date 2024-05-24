FROM gsoci.azurecr.io/giantswarm/alpine:3.20.0

RUN apk add --no-cache ca-certificates

ADD ./app-operator /app-operator

ENTRYPOINT ["/app-operator"]
