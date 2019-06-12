FROM alpine:3.9

RUN apk add --no-cache ca-certificates

ADD ./app-operator /app-operator

ENTRYPOINT ["/app-operator"]
