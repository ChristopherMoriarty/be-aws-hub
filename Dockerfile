FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api \
    && CGO_ENABLED=0 go build -o /out/migrate ./cmd/migrate

FROM alpine:3.21 AS migrate
RUN apk add --no-cache ca-certificates
COPY --from=build /out/migrate /migrate
ENTRYPOINT ["/migrate"]

FROM alpine:3.21 AS api
RUN apk add --no-cache ca-certificates wget
COPY --from=build /out/api /api
COPY --from=build /out/migrate /migrate
EXPOSE 8080
ENTRYPOINT ["/api"]
