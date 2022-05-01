FROM golang:latest as builder

RUN mkdir /app
WORKDIR /app

ENV GO111MODULE=on

COPY . .

RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -o app ./main.go


FROM alpine:latest

RUN apk add --no-cache libc6-compat
RUN apk --no-cache add ca-certificates

RUN mkdir /app
WORKDIR /app
COPY --from=builder /app .

EXPOSE 8080

CMD ["./app"]
