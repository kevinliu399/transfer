# Transfer

A high-performance file transfer utility using TLS encryption and gRPC for communication.

## Features

- Fast single file uploads (869.64 MB/s)
- Concurrent upload support
- TLS encryption for secure transfers
- gRPC for efficient communication
- Checksum verification to ensure data integrity

## Benchmark Results

### Single Upload

Average:   0.02s → 869.64 MB/s

### Concurrent Uploads

| Clients | Average Time |
|---------|-------------|
| 10      | 0.21s       |
| 100     | 2.25s       |
| 500     | 13.46s      |