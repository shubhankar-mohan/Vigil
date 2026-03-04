# Stage 1: Build React UI
FROM node:20-alpine AS ui-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.24-alpine AS go-builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=ui-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=1 go build -o /vigil ./cmd/vigil

# Stage 3: Final image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go-builder /vigil /app/vigil
COPY --from=ui-builder /app/web/dist /app/web/dist

EXPOSE 8080
VOLUME /data

ENV DB_PATH=/data/vigil.db
ENV LISTEN_ADDR=:8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/vigil"]
