FROM golang:1.13

WORKDIR /app

COPY . /app

RUN go build main.go

ENTRYPOINT ["/app/main"]
