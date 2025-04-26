# Transfer

A high-performance file transfer utility using TLS encryption, gRPC, and Go channels for efficient and secure data transmission.

## Features

- **High Performance**: Achieve transfer speeds up to 880+ MB/s
- **Concurrent Transfers**: Support for multiple parallel file uploads
- **Security**: TLS encryption for secure data transmission
- **Reliability**: Checksum verification ensures data integrity
- **Efficient Communication**: Powered by gRPC and Go channels
- **Simple Interface**: Easy-to-use client and server components

## Benchmark Results

### 21 MB File

#### Single Upload
- **Average Speed**: 869.64 MB/s (0.02s)

#### Concurrent Uploads

| Concurrent Clients | Average Time |
|-------------------|--------------|
| 10                | 0.21s        |
| 100               | 2.25s        |
| 500               | 13.46s       |

### 171 MB File

#### Single Upload
- **Average Speed**: 881.22 MB/s (0.19s)

#### Concurrent Uploads

| Concurrent Clients | Average Time |
|-------------------|--------------|
| 10                | 1.52s        |
| 50                | 8.33s        |
| 100               | 16.35s       |