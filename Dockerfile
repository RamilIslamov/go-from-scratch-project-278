FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build/code

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /build/app .

FROM alpine:3.22

WORKDIR /app

COPY --from=builder /build/app /app/bin/app
COPY bin/run.sh /app/bin/run.sh

RUN chmod +x /app/bin/run.sh

EXPOSE 8080

CMD ["/app/bin/run.sh"]