FROM alpine:3.16.1

RUN apk add --no-cache ca-certificates

ADD ./app-operator /app-operator

ENTRYPOINT ["/app-operator"]
