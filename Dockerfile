FROM golang:1.23-alpine AS build

WORKDIR /src

# Download dependencies first so source changes don't invalidate this layer.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /app /app
USER nonroot:nonroot
ENTRYPOINT ["/app"]
