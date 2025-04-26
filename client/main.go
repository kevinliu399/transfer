package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
	"sync"
	"time"

	ft "github.com/kevinliu399/transfer/proto"

	"github.com/schollz/progressbar/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	chunk_size  = 32 * 1024 // 32 KB
	concurrency = 4
)

func main() {

	creds, err := credentials.NewClientTLSFromFile("certs/server.crt", "")
	if err != nil {
		log.Fatalf("failed to create TLS credentials %v", err)
	}

	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := ft.NewFileTransferClient(conn)

	file_path := "test.txt"
	err = uploadFile(client, file_path)
	if err != nil {
		log.Fatalf("could not upload file: %v", err)
	}
}

func uploadFile(client ft.FileTransferClient, filePath string) error {
	stream, err := client.Upload(context.Background())
	if err != nil {
		return err
	}

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
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	bar := progressbar.NewOptions64(
		totalSize,
		progressbar.OptionSetDescription("Uploading"),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(40),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionSetPredictTime(true),
	)

	defer bar.Finish()
	buf := make([]byte, chunk_size)
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
	log.Printf("\n\nUpload result: success=%v, message=%q", resp.Success, resp.Message)
	return nil
}
