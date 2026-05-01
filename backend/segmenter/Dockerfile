FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/segmenter ./cmd/segmenter

FROM alpine:3.20
COPY --from=build /out/segmenter /usr/local/bin/segmenter
EXPOSE 8086
ENTRYPOINT ["/usr/local/bin/segmenter"]
