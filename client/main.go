package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	ft "github.com/kevinliu399/transfer/proto"
	"github.com/schollz/progressbar/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	chunkSize    = 32 * 1024 // 32 KB
	uploadWorker = 4
)

func main() {
	creds, err := credentials.NewClientTLSFromFile("certs/server.crt", "")
	if err != nil {
		log.Fatalf("failed to create TLS credentials: %v", err)
	}

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := ft.NewFileTransferClient(conn)

	uploadDir := "client/to_upload"
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		log.Fatalf("failed to read upload dir %q: %v", uploadDir, err)
	}

	var totalBytes int64
	var files []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(uploadDir, entry.Name())
		files = append(files, path)

		fi, err := os.Stat(path)
		if err != nil {
			log.Fatalf("failed to stat file %q: %v", path, err)
		}
		totalBytes += fi.Size()
	}

	bar := progressbar.NewOptions64(
		totalBytes,
		progressbar.OptionSetDescription("Uploading all files"),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
	)
	defer bar.Finish()

	var wg sync.WaitGroup
	sem := make(chan struct{}, uploadWorker)

	for _, path := range files {
		wg.Add(1)
		sem <- struct{}{}

		go func(path string) {
			defer wg.Done()
			defer func() { <-sem }()

			fmt.Printf("Uploading %s …\n", path)
			if err := uploadFile(client, path, bar); err != nil {
				log.Printf("error uploading %s: %v", path, err)
			} else {
				fmt.Printf("%s uploaded successfully!\n", path)
			}
		}(path)
	}

	wg.Wait()

	fmt.Println("\n✅ All files uploaded successfully!")
}

func uploadFile(client ft.FileTransferClient, filePath string, bar *progressbar.ProgressBar) error {
	stream, err := client.Upload(context.Background())
	if err != nil {
		return err
	}
	defer func() {
		_ = stream.CloseSend()
	}()

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}
	totalSize := fi.Size()

	workers := calculateWorkers(totalSize)

	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup

	buf := make([]byte, chunkSize)
	var offset int64
	var chunkNum int64

	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		data := make([]byte, n)
		copy(data, buf[:n])

		sum := sha256.Sum256(data)
		cs := hex.EncodeToString(sum[:])

		wg.Add(1)
		sem <- struct{}{}

		go func(data []byte, off int64, num int64, checksum string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := stream.Send(&ft.FileChunk{
				Filename:    filePath,
				Data:        data,
				ChunkNumber: num,
				Offset:      off,
				Checksum:    checksum,
			}); err != nil {
				log.Printf("send chunk %d failed: %v", num, err)
			}
			bar.Add(len(data))
		}(data, offset, chunkNum, cs)

		chunkNum++
		offset += int64(n)
	}

	wg.Wait()
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}
	if !resp.Success {
		return fmt.Errorf("upload failed: %s", resp.Message)
	}
	return nil
}

func calculateWorkers(totalSize int64) int {
	switch {
	case totalSize < 10*1024*1024:
		return 2
	case totalSize < 500*1024*1024:
		return 4
	default:
		return 8
	}
}
