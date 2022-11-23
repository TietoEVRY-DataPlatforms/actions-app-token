FROM golang:1.19-alpine as builder

WORKDIR /src

COPY . /src
RUN apk update && apk upgrade && apk add --no-cache ca-certificates
RUN update-ca-certificates

RUN go get . && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w' -o gh-token-fetcher .

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /src/gh-token-fetcher /bin/gh-token-fetcher

CMD ["gh-token-fetcher"]
