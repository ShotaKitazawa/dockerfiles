FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY main.go main_test.go ./
RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build -trimpath -o /dcr-proxy .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /dcr-proxy /dcr-proxy
ENTRYPOINT ["/dcr-proxy"]
