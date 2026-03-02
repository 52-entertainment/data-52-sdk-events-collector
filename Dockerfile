# Build
FROM golang:1.25 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /collector ./cmd/collector

# Run (distroless)
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /collector /collector
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/collector"]
