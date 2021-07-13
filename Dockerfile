FROM golang:1.16 AS builder

ARG VERSION=unknown

WORKDIR $GOPATH/src/gihub.com/rverst/rss-filter

ENV CGO_ENABLED 0
ENV GOOS=linux
ENV GOARCH=amd64

COPY . .

RUN go build -ldflags="-X 'main.version=${VERSION}'" -o /rss-filter

FROM alpine

RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

COPY --from=builder /rss-filter /usr/bin/rss-filter

EXPOSE 80/tcp
ENV LISTEN_ADDR=":80"
ENV API_KEY=""

ENTRYPOINT ["rss-filter"]