# Use an official Golang image as the base
FROM golang:1.23-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy the Go source code and generated proto files into the container
COPY . .

# Build the Go gRPC scaler
RUN go build -o grpc-scaler .

# Create a directory to hold the file
RUN mkdir -p /data

# Expose port 8080
EXPOSE 8080

# Command to run the app
CMD ["./grpc-scaler"]
