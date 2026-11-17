# --- Build stage ---
FROM golang:1.27-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/buecherei .

# --- Runtime stage ---
FROM alpine:3.22
RUN apk add --no-cache ca-certificates
COPY --from=build /out/buecherei /usr/local/bin/buecherei

ENV GIN_MODE=release
EXPOSE 8080
CMD ["buecherei"]
