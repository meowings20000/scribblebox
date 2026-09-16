# Multi-stage build: the runtime image carries only the compiled Go binary and wget.
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app .

FROM alpine:3.20
# wget backs the Compose healthcheck against /api/health.
RUN apk add --no-cache wget
COPY --from=build /out/app /app
EXPOSE 8080
ENV PORT=8080
CMD ["/app"]
