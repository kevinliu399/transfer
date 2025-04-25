package main

import (
	"context"
	"io"
	"log"
	"os"

	ft "github.com/kevinliu399/transfer/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const chunk_size = 32 * 1024 // 32 KB

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := ft.NewFileTransferClient(conn)

	file_path := "path/to/your/file.txt"
	err = uploadFile(client, file_path)
	if err != nil {
		log.Fatalf("could not upload file: %v", err)
	}
}

func uploadFile(client ft.FileTransferClient, file_path string) error {

	stream, err := client.Upload(context.Background())
	if err != nil {
		return err
	}

	file, err := os.Open(file_path)
	if err != nil {
		return err
	}
	defer file.Close()

	buf := make([]byte, chunk_size)
	var ChunkNumber int32
	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		if err := stream.Send(&ft.FileChunk{
			Filename:    file_path,
			Data:        buf[:n],
			ChunkNumber: ChunkNumber,
		}); err != nil {
			return err
		}
		ChunkNumber++
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}

	log.Printf("Upload result: success=%v, message=%s", resp.Success, resp.Message)
	return nil
}
