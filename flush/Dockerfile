FROM golang:1.23-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod tidy

COPY . .

RUN go build -o flush_redis .

FROM alpine:latest  

RUN apk --no-cache add redis

WORKDIR /root/

COPY --from=build /app/flush_redis .

CMD ["./flush_redis"]