FROM golang:1.27-alpine AS build

WORKDIR /src

# Dependency disalin lebih dulu supaya layer-nya bisa dipakai ulang selama
# go.mod dan go.sum tidak berubah.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/article-service ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 10001 app

COPY --from=build /out/article-service /usr/local/bin/article-service
COPY --from=build /out/migrate /usr/local/bin/migrate

USER app
EXPOSE 8080

ENTRYPOINT ["article-service"]
