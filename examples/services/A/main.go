package main

import (
	"flag"
	"net"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
	"sync"
	"github.com/dmitryrpm/zex/proto"
	"github.com/dmitryrpm/zex/examples/services/A/proto"
)

var (
	zexServerAddr = flag.String("zex", "127.0.0.1:54321", "Zex server in the format of host:port")
	serverAddr    = flag.String("addr", "127.0.0.1:54322", "The local server address in the format of host:port")
)

// Registry service to Zex
func registerZex(client zex.ZexClient, service *zex.Service) {
	grpclog.Printf("Start registry service in Zex (%s, %s)", service.Name, service.Addr)
	z, err := client.Register(context.Background(), service)
	if err != nil {
		grpclog.Fatalf("%v.Registry(_) = _, %v: ", client, err)
	}
	grpclog.Println(z)
}

type AServer struct{}

func (s *AServer) CallA(ctx context.Context, empty *A.Req) (*A.Empty, error) {
	defer grpclog.Printf("Call service A.%s with req", empty)
	return &A.Empty{}, nil

}

func (s *AServer) CallB(ctx context.Context, empty *A.Req) (*A.Empty, error) {
	grpclog.Printf("Call service A.%s with req", empty)
	return &A.Empty{}, nil
}

func (s *AServer) CallC(ctx context.Context, empty *A.Req) (*A.Empty, error) {
	grpclog.Printf("Call service A.%s with req", empty)
	return &A.Empty{}, nil
}

func main() {
	flag.Parse()

	grpclog.Println("starting local service in ", serverAddr)
	listener, err := net.Listen("tcp", *serverAddr)
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()

	s := new(AServer)
	A.RegisterAServer(grpcServer, s)
	reflection.Register(grpcServer)

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		wg.Done()
		err = grpcServer.Serve(listener)
		if err != nil {
			grpclog.Fatalf("fail serve: %v", err)
		}
		wg.Done()
	}()

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	grpclog.Println("registering to Zex")
	zexConn, err := grpc.Dial(*zexServerAddr, opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
	}
	zexClient := zex.NewZexClient(zexConn)
	registerZex(zexClient, &zex.Service{Name: "serviceA1", Addr: *serverAddr})
	grpclog.Println("registed... close connection")
	zexConn.Close()

	wg.Wait()

}
