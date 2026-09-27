FROM golang:1.26.7-alpine3.23@sha256:b17af760035fc2f338eed92d448a6c67f2d45438844fc6c60678fa5f99e44b57 AS build

WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false -ldflags="-s -w -X main.version=action" -o /out/changefence .

FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

RUN apk add --no-cache ca-certificates git \
    && addgroup -S changefence \
    && adduser -S -D -u 10001 -G changefence changefence
COPY --from=build /out/changefence /usr/local/bin/changefence

ENV HOME=/home/changefence
USER changefence
ENTRYPOINT ["/usr/local/bin/changefence"]
