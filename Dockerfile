FROM golang:1.13.0-buster AS builder
ENV GO111MODULE on
WORKDIR /go/src/github.com/form3tech-oss/openfaas-sqs-connector
COPY go.mod go.sum ./
RUN go mod vendor
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN go build -o /openfaas-sqs-connector -v ./cmd/main.go

FROM gcr.io/distroless/base
COPY --from=builder /openfaas-sqs-connector /openfaas-sqs-connector
ENTRYPOINT ["/openfaas-sqs-connector"]
CMD ["--help"]
