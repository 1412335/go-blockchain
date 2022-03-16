FROM golang:1.16-alpine AS builder

ENV PATH=$PATH:/usr/local/go/bin

RUN  apk update \
    && apk add build-base

ENV TZ=Asia/Ho_Chi_Minh

RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

RUN mkdir -p /$GOPATH/src/app

WORKDIR $GOPATH/src/app

COPY go.mod go.sum ./

ENV GO111MODULE=on

RUN go mod download \
    && go mod verify

COPY . .

RUN go test -v ./...

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o /go/bin/app ./cmd


FROM alpine:latest AS deploy

RUN apk --no-cache add ca-certificates

WORKDIR /srv/
COPY --from=builder /go/bin/app /srv/
COPY --from=builder /go/src/app/database /srv/database/

RUN chmod +x /srv/app

EXPOSE 8080

# ENTRYPOINT ["/srv/app"]
# remain running container
CMD exec /bin/sh -c "trap : TERM INT; sleep infinity & wait"