FROM golang:1.27.3-alpine3.24 AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go build -o /out/goflow ./cmd/goflow

FROM alpine:3.24
RUN apk add --no-cache ca-certificates
COPY --from=build /out/goflow /usr/local/bin/goflow
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/goflow"]
