# Build from the repository root: docker build -f apps/api/Dockerfile .
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/server /app/server
COPY packages/courses /app/courses
USER app
ENV COURSES_DIR=/app/courses PORT=8080
EXPOSE 8080
ENTRYPOINT ["/app/server"]
