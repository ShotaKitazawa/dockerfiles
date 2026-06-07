FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 go build -trimpath -o /hydra-login-consent .

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /hydra-login-consent /hydra-login-consent
ENTRYPOINT ["/hydra-login-consent"]
