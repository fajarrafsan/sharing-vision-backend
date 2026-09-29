FROM golang:1.27-alpine AS build

WORKDIR /src

# Dependency disalin lebih dulu supaya layer-nya bisa dipakai ulang selama
# go.mod dan go.sum tidak berubah.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/warta-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/warta-migrate ./cmd/migrate

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app \
    && mkdir -p /app/uploads && chown app:app /app/uploads

WORKDIR /app

COPY --from=build /out/warta-api /usr/local/bin/warta-api
COPY --from=build /out/warta-migrate /usr/local/bin/warta-migrate

USER app
# Gambar sampul yang diunggah. Pasang volume di sini supaya tidak hilang saat
# container diganti.
VOLUME /app/uploads
EXPOSE 8080

ENTRYPOINT ["warta-api"]
