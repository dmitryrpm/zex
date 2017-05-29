package c

import (
	"google.golang.org/grpc/grpclog"
	"golang.org/x/net/context"
)

type CService struct{}

func (s *CService) CallA(ctx context.Context, empty *Req) (*Empty, error) {
	defer grpclog.Printf("Call services C.%s with req", empty)
	//time.Sleep(100 * time.Second)
	return &Empty{}, nil
}

func (s *CService) CallB(ctx context.Context, empty *Req) (*Empty, error) {
	grpclog.Printf("Call services C.%s with req", empty)
	return &Empty{}, nil
}

func (s *CService) CallC(ctx context.Context, empty *Req) (*Empty, error) {
	grpclog.Printf("Call services C.%s with req", empty)
	return &Empty{}, nil
}