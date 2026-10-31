FROM node:24-bookworm AS toy-build
WORKDIR /src/web-toy
COPY web-toy/package*.json ./
RUN npm ci --no-audit --no-fund
COPY web-toy/ ./
COPY shared/ /src/shared/
COPY frontend/src/ /src/frontend/src/
COPY frontend/public/ /src/frontend/public/
ARG VITE_API_BASE_URL
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
RUN test -n "$VITE_API_BASE_URL" && npm run test:timing && npm run build

FROM scratch AS toy-dist
COPY --from=toy-build /src/web-toy/dist/ /

FROM golang:1.27.3-bookworm AS server-build
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/pp-piano-api .

FROM scratch AS api-binary
COPY --from=server-build /out/pp-piano-api /pp-piano-api

FROM debian:bookworm-slim AS server
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/* \
 && groupadd -g 10001 piano && useradd -u 10001 -g piano -M piano && mkdir /data && chown 10001:10001 /data
WORKDIR /app
COPY --from=server-build /out/pp-piano-api /app/pp-piano-api
USER 10001:10001
ENV ADDR=0.0.0.0:8080 DATA_DIR=/data
VOLUME ["/data"]
EXPOSE 8080
CMD ["/app/pp-piano-api"]
