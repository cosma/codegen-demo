FROM golang:1.23

WORKDIR /app

# Copy the entire project
COPY . .

RUN go mod download
RUN ls -all

# Initialize Go module and run the application
CMD ["sh", "-c", "go mod download && go run cmd/server/main.go"]