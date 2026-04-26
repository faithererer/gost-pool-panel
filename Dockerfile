FROM golang:1.22-alpine AS build

ARG GOST_VERSION=3.2.6
ARG TARGETARCH

WORKDIR /src
COPY . .
RUN apk add --no-cache ca-certificates tar wget \
    && go build -o /out/gost-pool-panel ./cmd/panel \
    && mkdir -p /out/dist \
    && GOOS=linux GOARCH=amd64 go build -o /out/dist/gost-pool-agent-linux-amd64 ./cmd/agent \
    && GOOS=linux GOARCH=arm64 go build -o /out/dist/gost-pool-agent-linux-arm64 ./cmd/agent \
    && GOST_ARCH="${TARGETARCH:-amd64}" \
    && wget -qO /tmp/gost.tgz "https://github.com/go-gost/gost/releases/download/v${GOST_VERSION}/gost_${GOST_VERSION}_linux_${GOST_ARCH}.tar.gz" \
    && mkdir -p /tmp/gost \
    && tar -xzf /tmp/gost.tgz -C /tmp/gost \
    && find /tmp/gost -type f -name gost -exec cp {} /out/gost \; \
    && chmod +x /out/gost

FROM node:20-alpine AS frontend

WORKDIR /src/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend ./
RUN npm run build

FROM alpine:3.20

WORKDIR /app
ENV PANEL_LISTEN=:3000
ENV PANEL_PORT=3000
ENV PANEL_DATA_PATH=/data/state.json
COPY --from=build /out/gost-pool-panel /app/gost-pool-panel
COPY --from=build /out/dist /app/dist
COPY --from=build /out/gost /usr/local/bin/gost
COPY --from=frontend /src/frontend/dist /app/frontend/dist
VOLUME ["/data"]
EXPOSE 3000
CMD ["/app/gost-pool-panel"]
