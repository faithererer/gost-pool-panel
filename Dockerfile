FROM golang:1.22-alpine AS build

WORKDIR /src
COPY . .
RUN go build -o /out/gost-pool-panel ./cmd/panel \
    && mkdir -p /out/dist \
    && GOOS=linux GOARCH=amd64 go build -o /out/dist/gost-pool-agent-linux-amd64 ./cmd/agent \
    && GOOS=linux GOARCH=arm64 go build -o /out/dist/gost-pool-agent-linux-arm64 ./cmd/agent

FROM alpine:3.20

WORKDIR /app
ENV PANEL_LISTEN=:3000
ENV PANEL_PORT=3000
ENV PANEL_DATA_PATH=/data/state.json
COPY --from=build /out/gost-pool-panel /app/gost-pool-panel
COPY --from=build /out/dist /app/dist
VOLUME ["/data"]
EXPOSE 3000
CMD ["/app/gost-pool-panel"]
