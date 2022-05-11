package main

import (
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	pb "github.com/johannaojeling/go-grpc-web-scraper/pkg/api/v1"
	"github.com/johannaojeling/go-grpc-web-scraper/pkg/server"
)

func main() {
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
		log.Printf("defaulting to port %v", port)
	}

	log.Printf("listening on port %v", port)
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("error listening: %v", port)
	}

	grpcServer := grpc.NewServer()
	scraperServer := server.NewServer()
	pb.RegisterScraperServiceServer(grpcServer, scraperServer)

	log.Println("serving...")
	if err = grpcServer.Serve(listener); err != nil {
		log.Fatalf("error serving: %v", err)
	}
}
