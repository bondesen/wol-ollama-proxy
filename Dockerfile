ARG BUILD_FROM

# --- build the Go binary ---
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -o /wolproxy .

# --- runtime (plain base; binaeren laeser /data/options.json selv) ---
FROM ${BUILD_FROM}
COPY --from=build /wolproxy /usr/bin/wolproxy
CMD [ "/usr/bin/wolproxy" ]
