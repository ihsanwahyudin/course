# Build from the repository root: docker build -f apps/api/Dockerfile .
# One image serves the API and the Expo web app (WEB_DIR), so the web app
# calls the API on its own origin.
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM node:24-alpine AS web
WORKDIR /web
COPY apps/mobile/package.json apps/mobile/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY apps/mobile/ ./
# No EXPO_PUBLIC_API_URL: the deployed web app calls the server it came from.
RUN rm -f .env && npx expo export -p web --output-dir /out/web

FROM alpine:3.22
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=web /out/web /app/web
COPY packages/courses /app/courses
# Lesson audio cache; mount a volume here so it survives new containers.
RUN mkdir -p /app/data/tts && chown -R app:app /app/data
USER app
ENV COURSES_DIR=/app/courses PORT=8080 SPEECH_CACHE_DIR=/app/data/tts WEB_DIR=/app/web
EXPOSE 8080
ENTRYPOINT ["/app/server"]
