package a

import (
	"google.golang.org/grpc/grpclog"
	"golang.org/x/net/context"
)

type AService struct{}

func (s *AService) CallA(ctx context.Context, empty *Req) (*Empty, error) {
	defer grpclog.Printf("Call services A.%s with req", empty)
	//time.Sleep(100 * time.Second)
	return &Empty{}, nil
}

func (s *AService) CallB(ctx context.Context, empty *Req) (*Empty, error) {
	grpclog.Printf("Call services A.%s with req", empty)
	return &Empty{}, nil
}

func (s *AService) CallC(ctx context.Context, empty *Req) (*Empty, error) {
	grpclog.Printf("Call services A.%s with req", empty)
	return &Empty{}, nil
}

