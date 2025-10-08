FROM gsoci.azurecr.io/giantswarm/alpine:3.22.2

RUN apk add --no-cache ca-certificates

ADD ./app-operator /app-operator

ENTRYPOINT ["/app-operator"]
