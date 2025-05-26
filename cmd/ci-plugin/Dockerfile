FROM golang:1.24-alpine

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN make build-plugin

ENTRYPOINT ["/app/ci-plugin"]

CMD ["--help"]
