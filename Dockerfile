FROM node:24-alpine AS frontend
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
COPY THIRD_PARTY_LICENSES.txt ../THIRD_PARTY_LICENSES.txt
RUN npm run build

FROM golang:1.27.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/internal/web/dist ./internal/web/dist
COPY --from=frontend /src/internal/browser/recorder.js ./internal/browser/recorder.js
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" -o /kipuray ./cmd/kipuray

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && addgroup -S kipuray && adduser -S -G kipuray kipuray && mkdir /data && chown kipuray:kipuray /data
COPY --from=build /kipuray /usr/local/bin/kipuray
COPY LICENSE NOTICE THIRD_PARTY_LICENSES.txt /usr/share/doc/kipuray/
USER kipuray
VOLUME ["/data"]
EXPOSE 4317
ENTRYPOINT ["kipuray"]
CMD ["serve", "--listen", "0.0.0.0:4317", "--data-dir", "/data", "--no-open"]
