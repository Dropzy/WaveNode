FROM golang:1.25.11-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/wavenode-updater .

FROM docker:28.5.2-cli

ARG VERSION=dev
LABEL org.opencontainers.image.title="WaveNode Updater" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.source="https://github.com/Dropzy/WaveNode"

COPY --from=build /out/wavenode-updater /usr/local/bin/wavenode-updater

EXPOSE 8090
ENTRYPOINT ["wavenode-updater"]
