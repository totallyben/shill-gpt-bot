FROM golang:1.21-bullseye

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["shill-gpt-bot", "run"]