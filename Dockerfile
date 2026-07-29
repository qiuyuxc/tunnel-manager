# syntax=docker/dockerfile:1
# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm npm ci --loglevel verbose
COPY frontend/ .
RUN npm run build

# Stage 2: Build backend
FROM golang:1.26-alpine AS backend
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download -x
COPY backend/ .
RUN CGO_ENABLED=0 go build -v -o tunnel-manager .

# Stage 3: Final image
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=backend /app/tunnel-manager .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN mkdir -p data

EXPOSE 8080
ENV PORT=8080
ENV STATIC_DIR=frontend/dist
ENV STORE_PATH=data/config.json

CMD ["./tunnel-manager"]
