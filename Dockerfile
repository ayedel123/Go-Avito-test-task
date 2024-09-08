
FROM golang:1.23

WORKDIR /home/go-server

COPY . .

WORKDIR /home/go-server/src

RUN go mod tidy

RUN go build -o main .

EXPOSE 8080

CMD ["./main"]
