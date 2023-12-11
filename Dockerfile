FROM alpine:3.19.0

RUN apk add --no-cache ca-certificates

ADD ./app-operator /app-operator

ENTRYPOINT ["/app-operator"]
