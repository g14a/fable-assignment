FROM alpine:latest

RUN apk add --no-cache bash curl go

WORKDIR /app

# Copy the project files
COPY . .

# Build the Go application
RUN go build -o app

# Set the entry point for the container
CMD ["./app"]
