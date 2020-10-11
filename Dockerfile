FROM golang:1.13

WORKDIR /app

COPY . /app

RUN go build main.go

ENV PORT

ENTRYPOINT ["/app/main"]
