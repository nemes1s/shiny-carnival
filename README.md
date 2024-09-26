# README.md

## Overview

This repository contains two Go applications designed to work together:

1. **CLI Application**: A command-line tool that reads records from a CSV file, sanitizes the data, and sends the records to a microservice for processing.

2. **Microservice Application**: An HTTP server that processes incoming records, enriches them using an external Enrichment Service, batches the enriched records, and sends them to an Analytics Service while respecting rate limits.

---

## Table of Contents

- [CLI Application](#cli-application)
  - [Features](#features)
  - [Installation](#installation)
  - [Usage](#usage)
  - [Possible Future Improvements](#possible-future-improvements)
  - [Deployment Tips](#deployment-tips)
  - [Sample Dockerfile](#sample-dockerfile)
- [Microservice Application](#microservice-application)
  - [Features](#features-1)
  - [Installation](#installation-1)
  - [Usage](#usage-1)
  - [Possible Future Improvements](#possible-future-improvements-1)
  - [Deployment Tips](#deployment-tips-1)
  - [Sample Dockerfile](#sample-dockerfile-1)
- [General Deployment Scenarios](#general-deployment-scenarios)
  - [Docker Compose](#docker-compose)
  - [Kubernetes Deployment](#kubernetes-deployment)
- [Conclusion](#conclusion)
- [Future work](#future-work)

---

## CLI Application

### Features

- **CSV Reading**: Reads records from a CSV file with customizable delimiters.
- **Data Sanitization**: Sanitizes the "category" field to ensure consistency before processing.
- **Concurrent Processing**: Utilizes goroutines for concurrent sending of records to the microservice.
- **Error Handling**: Skips invalid or malformed records and logs errors appropriately.
- **Filtering**: Supports filtering records by category through command-line arguments.

### Installation

#### Prerequisites

- **Go**: Version 1.16 or higher.
- **Git**: For cloning the repository.

#### Steps

1. **Clone the Repository**

   ```bash
   git clone https://github.com/yourusername/yourrepository.git
   cd yourrepository/cli
   ```

2. **Build the CLI Application**

   ```bash
   go build -o cli-app
   ```

### Usage

#### Command-Line Arguments

- `-file`: Path to the CSV file (default: `data.csv`).
- `-category`: Filter records by category (optional).

#### Running the Application

```bash
./cli-app -file=path/to/your/data.csv -category=phishing
```

#### Example

```bash
./cli-app -file=data.csv
```

This command reads records from `data.csv`, sanitizes them, and sends them to the microservice at `http://localhost:8081/process`.

### Possible Future Improvements

- **Enhanced Error Handling**: Collect and aggregate errors from goroutines for better reporting.
- **Configurable Concurrency**: Make the number of concurrent goroutines adjustable via command-line flags or configuration files.
- **Progress Indicators**: Implement progress bars or indicators for processing large datasets.
- **Advanced Logging**: Integrate a logging library to support different log levels and output formats.
- **Unit Testing**: Add comprehensive unit tests to ensure code reliability.

### Deployment Tips

- **Environment Variables**: Use environment variables for configuration to make the application more flexible.
- **Cross-Compilation**: Build binaries for different platforms using Go's cross-compilation capabilities.
- **CI/CD Integration**: Set up continuous integration and delivery pipelines for automated testing and deployment.

### Sample Dockerfile

```dockerfile
# Use the official Golang image as the base image
FROM golang:1.16-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o cli-app

# Set the entrypoint
ENTRYPOINT ["./cli-app"]

# Default command (can be overridden)
CMD ["-file=data.csv"]
```

---

## Microservice Application

### Features

- **HTTP Server**: Listens for incoming HTTP POST requests containing records to process.
- **Data Enrichment**: Enriches records by calling an external Enrichment Service.
- **Batch Processing**: Batches enriched records and sends them to the Analytics Service.
- **Rate Limiting**: Respects rate limits by sending up to 20 records every 10 seconds.
- **Retry Mechanism**: Implements retry logic for failed enrichment attempts.
- **Comprehensive Logging**: Logs all incoming requests, external service calls, and internal processing steps.

### Installation

#### Prerequisites

- **Go**: Version 1.16 or higher.
- **Git**: For cloning the repository.

#### Steps

1. **Clone the Repository**

   ```bash
   git clone https://github.com/yourusername/yourrepository.git
   cd yourrepository/microservice
   ```

2. **Build the Microservice Application**

   ```bash
   go build -o microservice-app
   ```

### Usage

#### Running the Microservice

```bash
./microservice-app
```

The microservice will start and listen on `http://localhost:8081`.

#### Endpoints

- `POST /process`: Accepts records for processing.

#### Example Request

```bash
curl -X POST http://localhost:8081/process \
  -H "Content-Type: application/json" \
  -d '{"id":"123", "asset_name":"Asset1", "ip":"192.168.1.1", "created_utc":"2021-01-01T00:00:00Z", "source":"source1", "category":"phishing"}'
```

### Possible Future Improvements

- **Graceful Shutdown**: Implement context cancellation and cleanup for graceful shutdowns.
- **Configuration Management**: Use environment variables or configuration files for settings like external service URLs and rate limits.
- **Enhanced Error Handling**: Improve retry mechanisms with exponential backoff and circuit breaker patterns.
- **Monitoring and Metrics**: Integrate with monitoring tools to track performance and errors.
- **Security Enhancements**: Add authentication and authorization mechanisms.
- **Unit and Integration Testing**: Develop tests to cover all critical paths and external interactions.

### Deployment Tips

- **Containerization**: Use Docker to containerize the microservice for consistent deployment across environments.
- **Orchestration**: Deploy using orchestration tools like Kubernetes or Docker Compose for scalability and management.
- **Scaling**: Ensure the microservice is stateless to allow horizontal scaling.
- **Logging**: Centralize logging using tools like ELK Stack or cloud-based solutions.

### Sample Dockerfile

```dockerfile
# Use the official Golang image as the base image
FROM golang:1.16-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN go build -o microservice-app

# Expose the port the microservice listens on
EXPOSE 8081

# Set the entrypoint
ENTRYPOINT ["./microservice-app"]
```

---

## General Deployment Scenarios

### Docker Compose

You can use Docker Compose to run both the CLI and microservice applications together.

**docker-compose.yml**

```yaml
version: '3'
services:
  microservice:
    build:
      context: ./microservice
      dockerfile: Dockerfile
    ports:
      - "8081:8081"
    environment:
      - ENRICHMENT_SERVICE_URL=https://api.example.com/enrichment
      - ANALYTICS_SERVICE_URL=https://api.example.com/analytics

  cli:
    build:
      context: ./cli
      dockerfile: Dockerfile
    depends_on:
      - microservice
    volumes:
      - ./data.csv:/app/data.csv
    command: ["-file=data.csv"]
```

### Kubernetes Deployment

For production environments, consider deploying the microservice using Kubernetes.

**microservice-deployment.yaml**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: microservice-deployment
spec:
  replicas: 3
  selector:
    matchLabels:
      app: microservice
  template:
    metadata:
      labels:
        app: microservice
    spec:
      containers:
      - name: microservice-container
        image: yourusername/microservice-app:latest
        ports:
        - containerPort: 8081
        env:
        - name: ENRICHMENT_SERVICE_URL
          value: "https://api.example.com/enrichment"
        - name: ANALYTICS_SERVICE_URL
          value: "https://api.example.com/analytics"
```

**microservice-service.yaml**

```yaml
apiVersion: v1
kind: Service
metadata:
  name: microservice-service
spec:
  selector:
    app: microservice
  ports:
    - protocol: TCP
      port: 8081
      targetPort: 8081
  type: ClusterIP
```

---

## Conclusion

These two Go applications are designed to work seamlessly together to process and enrich data efficiently. The CLI application handles data ingestion and preprocessing, while the microservice manages data enrichment, batching, and forwarding to the Analytics Service.

By following the installation and usage instructions, and considering the possible future improvements and deployment tips, you can effectively utilize and deploy these applications in your environment.

For any questions, issues, or contributions, feel free to open an issue or submit a pull request.

---

**Note**: Replace `yourusername` and `yourrepository` with your actual GitHub username and repository name. Adjust the Docker images, environment variables, and configurations according to your specific setup.


## Future work

- Improve configuration by using environment variables or configuration files for settings like external service URLs and rate limits.

- Store secrets securely using tools like HashiCorp Vault or AWS Secrets Manager.

- Implement context cancellation and cleanup for graceful shutdowns.
