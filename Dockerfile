# Stage 1: Build frontend
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.26-alpine AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist/ ./internal/api/frontend_dist/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /compliance-operator-dashboard .

# Stage 3: Runtime
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=go-builder /compliance-operator-dashboard /compliance-operator-dashboard
EXPOSE 8080
ENTRYPOINT ["/compliance-operator-dashboard"]
CMD ["serve"]
