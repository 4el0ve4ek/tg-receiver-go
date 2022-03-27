FROM golang:1.18

MAINTAINER Daniil Aksenov

WORKDIR /app

COPY . .
RUN go get -d -v
RUN go build -v -o /start-bot

CMD ["/start-bot"]
