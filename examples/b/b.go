package b

import (
	"google.golang.org/grpc/grpclog"
	"golang.org/x/net/context"
)

type BService struct{}

func (s *BService) CallA(ctx context.Context, empty *Req) (*Empty, error) {
	defer grpclog.Printf("Call services B.%s with req", empty)
	//time.Sleep(100 * time.Second)
	return &Empty{}, nil
}

func (s *BService) CallB(ctx context.Context, empty *Req) (*Empty, error) {
	grpclog.Printf("Call services B.%s with req", empty)
	return &Empty{}, nil
}

func (s *BService) CallC(ctx context.Context, empty *Req) (*Empty, error) {
	grpclog.Printf("Call services B.%s with req", empty)
	return &Empty{}, nil
}