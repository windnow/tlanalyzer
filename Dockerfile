# BACK
FROM golang:1.19.4-alpine3.17 as gobuilder
WORKDIR /build
# ENV SSL_CERT_DIR=/etc/ssl/certs
RUN apk add --update --no-cache ca-certificates
COPY go.mod .
RUN go mod download
COPY . .
RUN go build -v -o ./tlserver ./cmd/tlserver

# TARGET IMAGE
FROM alpine:3.17.1
WORKDIR /app
RUN apk update && apk add --no-cache ca-certificates tzdata curl
COPY --from=gobuilder /build/tlserver .
EXPOSE 8000
ENTRYPOINT ["./tlserver"]
