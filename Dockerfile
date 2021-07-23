FROM golang:1.16.5-alpine

WORKDIR /src
COPY ./ ./

RUN go install

CMD "go-balancer"

EXPOSE 80
EXPOSE 443
EXPOSE 4501
