  
FROM golang:1.11

WORKDIR /go/src/app
COPY ./slack-bot .

RUN go get -d -v ./...
RUN go install -v ./...

CMD ["app"]