FROM golang:1.26.4-alpine AS build

ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY auth ./auth
COPY cmd ./cmd
COPY config ./config
COPY database ./database
COPY enrichment ./enrichment
COPY handlers ./handlers
COPY metadata ./metadata
COPY middleware ./middleware
COPY migrations ./migrations
COPY musicbrainz ./musicbrainz
COPY router ./router
COPY scanner ./scanner
COPY streaming ./streaming
COPY utils ./utils
COPY websocket ./websocket

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w -X music-server/router.WaveNodeVersion=${VERSION}" -o /out/wavenode ./cmd/server

FROM alpine:3.22
ARG VERSION=dev
LABEL org.opencontainers.image.title="WaveNode" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.source="https://github.com/wavenode/wavenode"

RUN apk add --no-cache ca-certificates ffmpeg tzdata \
    && addgroup -S -g 10001 wavenode \
    && adduser -S -D -H -u 10001 -G wavenode wavenode

WORKDIR /app
COPY --from=build /out/wavenode /app/wavenode
COPY --from=build /src/database/migrations /app/database/migrations
COPY --from=build /src/migrations /app/migrations

RUN mkdir -p /data/artwork /music \
    && chown -R wavenode:wavenode /data

USER wavenode
EXPOSE 8080
ENTRYPOINT ["/app/wavenode"]
