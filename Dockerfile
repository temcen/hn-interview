FROM golang:1.24-alpine AS build

WORKDIR /src

# Stdlib only: no go.sum, nothing to download. Kept as its own layer so adding
# dependencies later doesn't invalidate the source layer.
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app ./cmd/hncapture

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /app /app
USER nonroot:nonroot
# Archives go to ./output; mount it to keep them:
#   docker run --rm -v "$PWD/output:/home/nonroot/output" hncapture
WORKDIR /home/nonroot
ENTRYPOINT ["/app"]
