ARG BUILD_FROM

# --- build the Go binary ---
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -o /wolproxy .

# --- runtime (HA base image, has s6 + bashio) ---
FROM ${BUILD_FROM}
COPY --from=build /wolproxy /usr/bin/wolproxy
COPY run.sh /
RUN chmod a+x /run.sh
CMD [ "/run.sh" ]
