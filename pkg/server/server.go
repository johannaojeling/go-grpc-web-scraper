package server

import (
	"log"
	"time"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/johannaojeling/go-grpc-web-scraper/pkg/api/v1"
)

type server struct {
	pb.UnimplementedScraperServiceServer
}

func NewServer() *server {
	return &server{}
}

func (*server) ScrapeUrl(in *pb.ScrapeRequest, stream pb.ScraperService_ScrapeUrlServer) error {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("interrupting scraping: %v", rec)
		}
	}()

	allowedDomains := in.GetAllowedDomains()
	maxDepth := int(in.GetMaxDepth())
	collector := newCollector(allowedDomains, maxDepth, stream)

	log.Println("starting scraping")
	url := in.GetUrl()
	err := collector.Visit(url)
	if err != nil {
		return status.Errorf(codes.Internal, "error scraping: %v", err)
	}

	log.Println("finished scraping")
	return nil
}

func newCollector(
	allowedDomains []string,
	maxDepth int,
	stream pb.ScraperService_ScrapeUrlServer,
) *colly.Collector {
	collector := colly.NewCollector(
		colly.AllowedDomains(allowedDomains...),
		colly.MaxDepth(maxDepth),
		colly.ParseHTTPErrorResponse(),
	)
	extensions.RandomUserAgent(collector)
	extensions.Referer(collector)

	collector.OnResponse(func(response *colly.Response) {
		if err := stream.Context().Err(); err != nil {
			panic(err)
		}

		url := response.Request.URL.String()
		log.Printf("scraping %s", url)

		page := &pb.Page{
			Url:       url,
			Status:    int32(response.StatusCode),
			Text:      string(response.Body),
			Timestamp: timestamppb.New(time.Now().UTC()),
		}

		scrapeResponse := &pb.ScrapeResponse{Page: page}
		err := stream.Send(scrapeResponse)
		if err != nil {
			panic(err)
		}
	})

	collector.OnHTML("a[href]", func(element *colly.HTMLElement) {
		link := element.Attr("href")
		_ = element.Request.Visit(element.Request.AbsoluteURL(link))
	})

	return collector
}
