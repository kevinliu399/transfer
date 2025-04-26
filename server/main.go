package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"

	ft "github.com/kevinliu399/transfer/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type server struct {
	ft.UnimplementedFileTransferServer
}

func (s *server) Upload(stream ft.FileTransfer_UploadServer) error {

	var (
		outFile *os.File
		err     error
	)

	for {
		chunk, errRcv := stream.Recv()
		if errRcv == io.EOF {
			break
		}
		if errRcv != nil {
			return stream.SendAndClose(&ft.UploadStatus{
				Success: false,
				Message: errRcv.Error(),
			})
		}

		if outFile == nil {
			fname := filepath.Base(chunk.Filename)
			outPath := filepath.Join("./shared", fname)
			outFile, err = os.OpenFile(outPath, os.O_CREATE|os.O_RDWR, 0644)
			if err != nil {
				return stream.SendAndClose(&ft.UploadStatus{Success: false, Message: err.Error()})
			}
			defer outFile.Close()
		}

		gotSum := sha256.Sum256(chunk.Data)
		if hex.EncodeToString(gotSum[:]) != chunk.Checksum {
			return stream.SendAndClose(&ft.UploadStatus{
				Success: false,
				Message: fmt.Sprintf("checksum mismatch on chunk %d", chunk.ChunkNumber),
			})
		}

		if _, err := outFile.Seek(chunk.Offset, io.SeekStart); err != nil {
			return stream.SendAndClose(&ft.UploadStatus{
				Success: false,
				Message: err.Error(),
			})
		}
		if _, err := outFile.Write(chunk.Data); err != nil {
			return stream.SendAndClose(&ft.UploadStatus{
				Success: false,
				Message: err.Error(),
			})
		}
	}

	return stream.SendAndClose(&ft.UploadStatus{
		Success: true,
		Message: "Upload successful",
	})
}

func main() {
	conn, err := net.Listen("tcp", ":50051")
	if err != nil {
		fmt.Printf("failed to listen: %v", err)
		return
	}

	creds, err := credentials.NewServerTLSFromFile(
		"certs/server.crt",
		"certs/server.key",
	)
	if err != nil {
		fmt.Printf("failed to create TLS credentials: %v", err)
		return
	}
	s := grpc.NewServer(grpc.Creds(creds))
	ft.RegisterFileTransferServer(s, &server{})

	fmt.Printf("Server is listening on port :50051\n")
	if err := s.Serve(conn); err != nil {
		fmt.Printf("failed to serve: %v", err)
		return
	}
}
