# syntax=docker/dockerfile:1
# Build any service: docker build --build-arg SERVICE=gateway .
FROM golang:1.26 AS build
ARG SERVICE
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${SERVICE}

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
COPY --from=build /out/app /app
COPY --from=build /src/config /config
ENTRYPOINT ["/app"]

# The crawler shells out to poppler's pdftotext for PDF text extraction.
FROM debian:12-slim AS runtime-crawler
RUN apt-get update && apt-get install -y --no-install-recommends poppler-utils ca-certificates \
    && rm -rf /var/lib/apt/lists/*
COPY --from=build /out/app /app
COPY --from=build /src/config /config
USER 65532:65532
ENTRYPOINT ["/app"]
