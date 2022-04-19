FROM golang:latest as builder

RUN mkdir -p /github.com/safe-area/range-info
WORKDIR /github.com/safe-area/range-info

ENV GO111MODULE=on

COPY . .

RUN go mod download
RUN CGO_ENABLED=1 GOOS=linux go build -o app ./main.go


FROM alpine:latest

RUN apk add --no-cache libc6-compat
RUN apk --no-cache add ca-certificates

RUN mkdir /app
WORKDIR /app
COPY --from=builder /github.com/safe-area/range-info/app .

EXPOSE 8080

CMD ["./app"]