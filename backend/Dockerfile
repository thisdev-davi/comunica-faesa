FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO desligado: binário estático, roda no alpine sem as libs de C do Go
RUN CGO_ENABLED=0 go build -o /api ./cmd/api \
 && CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.28.0

# Roda uma vez antes da API (serviço migrate do compose.prod.yaml); o goose só aplica o que falta.
# Banco e pasta vêm de GOOSE_DRIVER, GOOSE_DBSTRING e GOOSE_MIGRATION_DIR.
FROM alpine:3.22 AS migrate
COPY --from=build /go/bin/goose /usr/local/bin/goose
COPY db/migrations /migrations
ENV GOOSE_DRIVER=postgres GOOSE_MIGRATION_DIR=/migrations
USER nobody
CMD ["goose", "up"]

# Último estágio = o que `docker build` gera sem --target. O alpine já traz os certificados para falar com o Discord.
FROM alpine:3.22
COPY --from=build /api /api
USER nobody
EXPOSE 8090
CMD ["/api"]
