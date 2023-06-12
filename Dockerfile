# Build stage
FROM golang:1.17-alpine AS build

WORKDIR /app

# Copy all files from the project directory
COPY . .

# Build the Go application
RUN go build -o app

# Final stage
FROM alpine:latest AS final

RUN apk add --no-cache bash curl

WORKDIR /app

# Copy the built executable from the build stage
COPY --from=build /app/app .

COPY --from=build /app/migrations migrations

COPY test.sh .

RUN chmod +x test.sh

# Set the entry point for the container
CMD ["./app"]
