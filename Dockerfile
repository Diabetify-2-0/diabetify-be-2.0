FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /out/diabetify-be ./cmd

FROM alpine:3.20

ENV TZ=Asia/Jakarta

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/diabetify-be /app/diabetify-be

EXPOSE 8080

CMD ["/app/diabetify-be"]
