FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go test ./... && go build -o /scp2317 ./cmd/api
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /scp2317 /scp2317
USER nonroot:nonroot
ENTRYPOINT ["/scp2317"]
