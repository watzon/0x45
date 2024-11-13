FROM golang:1.23.3 AS build

WORKDIR /build/src

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o app ./cmd/server/main.go

FROM scratch

COPY --from=build /build/src/app /usr/bin/app

ENTRYPOINT ["/usr/bin/app"]
