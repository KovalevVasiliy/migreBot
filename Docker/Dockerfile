FROM golang:1.14

WORKDIR /go/src/app
COPY go.mod .
COPY go.sum .
COPY /src .

RUN go mod download
RUN go get -d -v ./...
RUN go install -v ./...

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /go/bin/app

CMD ["/go/bin/app"]