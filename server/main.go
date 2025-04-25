package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	ft "github.com/kevinliu399/transfer/proto"
	"google.golang.org/grpc"
)

type server struct {
	ft.UnimplementedFileTransferServer
}

func (s *server) Upload(stream ft.FileTransfer_UploadServer) error {

	chunks := make(chan *ft.FileChunk)
	err_chan := make(chan error, 1)

	go func() {
		uploadDir := "./shared"
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			err_chan <- fmt.Errorf("mkdir: %w", err)
			return
		}

		var outFile *os.File
		var err error
		defer func() {
			if outFile != nil {
				outFile.Close()
			}
			close(err_chan)
		}()

		for chunk := range chunks {
			if outFile == nil {
				safeName := filepath.Base(chunk.Filename)
				dest := filepath.Join(uploadDir, safeName)
				fmt.Printf("Writing to %s\n", dest)
				outFile, err = os.Create(dest)
				if err != nil {
					err_chan <- fmt.Errorf("create file %s: %w", dest, err)
					return
				}
			}

			if _, err = outFile.Write(chunk.Data); err != nil {
				err_chan <- fmt.Errorf("write chunk %d: %w", chunk.ChunkNumber, err)
				return
			}

			err_chan <- nil
		}
	}()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("stream.Recv: %w", err)
		}
		chunks <- chunk
	}

	close(chunks)

	if writeErr := <-err_chan; writeErr != nil {
		return stream.SendAndClose(&ft.UploadStatus{
			Success: false,
			Message: writeErr.Error(),
		})
	}

	return stream.SendAndClose(&ft.UploadStatus{
		Success: true,
		Message: "File uploaded successfully",
	})
}

func main() {
	conn, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
		return
	}

	s := grpc.NewServer()
	ft.RegisterFileTransferServer(s, &server{})
	fmt.Printf("Server is listening on port :50051\n")
	if err := s.Serve(conn); err != nil {
		fmt.Printf("failed to serve: %v", err)
		return
	}
}
