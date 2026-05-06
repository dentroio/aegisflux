# Build from repository root: docker build -f backend/detection-pipeline/Dockerfile .
FROM golang:1.23-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /src
COPY backend/detection-pipeline/go.mod backend/detection-pipeline/go.sum ./backend/detection-pipeline/
WORKDIR /src/backend/detection-pipeline
RUN go mod download
COPY backend/detection-pipeline/ ./
COPY schemas/detection/ /schemas/detection/
ENV DETECTION_PACK_SCHEMA_PATH=/schemas/detection/detection-pack.v1.schema.json
RUN CGO_ENABLED=0 GOOS=linux go build -o /detection-pipeline ./cmd/detection-pipeline

FROM gcr.io/distroless/static-debian12
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /detection-pipeline /detection-pipeline
COPY --from=builder /schemas/detection /schemas/detection
ENV DETECTION_PACK_SCHEMA_PATH=/schemas/detection/detection-pack.v1.schema.json
EXPOSE 8089
ENTRYPOINT ["/detection-pipeline"]
