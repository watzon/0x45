FROM golang:1.23.3 AS build

WORKDIR /build/src

COPY . .

RUN mkdir -p /build/src/app

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o app/0x45 .

FROM scratch

WORKDIR /usr/app

COPY --from=build /build/src/app /usr/app

# Copy ca-certificates for TLS (specifically STARTTLS for SMTP)
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY ./views /usr/app/views

COPY ./public /usr/app/public

ENTRYPOINT ["./0x45"]
