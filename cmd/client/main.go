package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	pb "github.com/johannaojeling/go-grpc-web-scraper/pkg/api/v1"
)

var (
	host         = flag.String("host", "", "Host")
	port         = flag.String("port", "443", "Port")
	withInsecure = flag.Bool(
		"withInsecure",
		false,
		"Whether to disable transport security in connection",
	)
	token    = flag.String("token", "", "GCP access token")
	maxDepth = flag.Int("maxDepth", 1, "Maximum depth of URLs to scrape")
	timeout  = flag.Int("timeout", 30, "Timeout in seconds for request")
	output   = flag.String("output", "output.json", "File for storing json output")
)

func main() {
	flag.Parse()

	address := net.JoinHostPort(*host, *port)
	log.Printf("connecting to address %v", address)

	conn, err := newConnection(address)
	if err != nil {
		log.Fatalf("error creating connection: %v", err)
	}
	defer conn.Close()

	ctx, cancel := newContext(*token, *timeout)
	defer cancel()

	client := pb.NewScraperServiceClient(conn)
	request := &pb.ScrapeRequest{
		Url:            "https://en.wikipedia.org/wiki/Main_Page",
		AllowedDomains: []string{"en.wikipedia.org"},
		MaxDepth:       int32(*maxDepth),
	}

	stream, err := client.ScrapeUrl(ctx, request)
	if err != nil {
		log.Fatalf("error calling ScrapeUrl: %v", err)
	}

	err = receiveToFile(stream, *output)
	if err != nil {
		log.Fatalf("error receiving: %v", err)
	}
}

func newConnection(address string) (*grpc.ClientConn, error) {
	creds := newCredentials(*withInsecure)
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("error connecting: %v", err)
	}
	return conn, nil
}

func newContext(token string, timeout int) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	if token != "" {
		ctx = metadata.AppendToOutgoingContext(
			ctx,
			"authorization",
			fmt.Sprintf("Bearer %s", token),
		)
	}
	return ctx, cancel
}

func newCredentials(withInsecure bool) credentials.TransportCredentials {
	if withInsecure {
		return insecure.NewCredentials()
	}
	return credentials.NewTLS(&tls.Config{
		InsecureSkipVerify: false,
	})
}

func receiveToFile(stream pb.ScraperService_ScrapeUrlClient, output string) error {
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	for {
		response, err := stream.Recv()
		if err == io.EOF {
			log.Println("received all messages")
			writer.Flush()
			return nil
		}

		if err != nil {
			errStatus, ok := status.FromError(err)
			if ok && errStatus.Code() == codes.DeadlineExceeded {
				log.Println("deadline exceeded")
				writer.Flush()
				return nil
			}
			writer.Flush()
			return fmt.Errorf("error receiving: %v", err)
		}

		page := response.GetPage()
		log.Printf("received message for: %s", page.Url)

		err = writeJsonLine(writer, page)
		if err != nil {
			return fmt.Errorf("error writing message to file: %v", err)
		}
	}
}

func writeJsonLine(writer *bufio.Writer, message proto.Message) error {
	data, err := protojson.Marshal(message)
	if err != nil {
		return err
	}

	data = append(data, byte('\n'))
	_, err = writer.Write(data)
	return err
}
