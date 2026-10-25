FROM golang:1.27.3-alpine3.23@sha256:754d5200fc3edae76d3b6598a9a3fac7e74981ee6a5413ab955025f185753ef2 AS build

WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -X main.version=action" -o /out/changefence .

FROM alpine:3.23.5@sha256:fd791d74b68913cbb027c6546007b3f0d3bc45125f797758156952bc2d6daf40

RUN apk add --no-cache ca-certificates git \
    && addgroup -S changefence \
    && adduser -S -D -u 10001 -G changefence changefence
COPY --from=build /out/changefence /usr/local/bin/changefence

ENV HOME=/home/changefence
USER changefence
ENTRYPOINT ["/usr/local/bin/changefence"]
