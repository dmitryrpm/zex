package main

import (
	"flag"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/credentials"
	pb "github.com/zex/zex/proto"
	lpb "github.com/zex/examples/services/A/proto"
	"google.golang.org/grpc/grpclog"
	"net"
	"google.golang.org/grpc/reflection"
)

var (
	zexServerAddr = flag.String("zex_server_addr", "127.0.0.1:10000", "Zex server in the format of host:port")
	serverAddr = flag.String("server_addr", "127.0.0.1:9999", "The local server address in the format of host:port")
)

// Registry service to Zex
func registerZex(client pb.ZexClient, service *pb.Service) {
	grpclog.Printf("Start registry service in Zex (%s, %s)", service.Name, service.Addr)
	zex, err := client.Register(context.Background(), service)
	if err != nil {
		grpclog.Fatalf("%v.Registry(_) = _, %v: ", client, err)
	}
	grpclog.Println(zex)
}

type AServer struct {}

func (s *AServer) CallA (ctx context.Context, empty *lpb.Empty) (*lpb.Empty, error) {
	grpclog.Printf("Call service A.%s with empty: ", empty)
	return &lpb.Empty{}, nil
}

func (s *AServer) CallB (ctx context.Context, empty *lpb.Empty) (*lpb.Empty, error) {
	grpclog.Printf("Call service A.%s with empty: ", empty)
	return &lpb.Empty{}, nil
}

func (s *AServer) CallC (ctx context.Context, empty *lpb.Empty) (*lpb.Empty, error) {
	grpclog.Printf("Call service A.%s with empty: ", empty)
	return &lpb.Empty{}, nil
}

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	grpclog.Println("registering to Zex")
	conn, err := grpc.Dial(*zexServerAddr, opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
	}
	client := pb.NewZexClient(conn)
	registerZex(client, &pb.Service{Name: "A", Addr: *serverAddr})
	grpclog.Println("registed... close connection")
	conn.Close()

	grpclog.Println("starting local service in ", serverAddr)
	tcp_connect, err := net.Listen("tcp", *serverAddr)
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}
	var server_opts []grpc.ServerOption
	grpcServer := grpc.NewServer(server_opts...)

	my_service := new(AServer)
	lpb.RegisterAServer(grpcServer, my_service)
	reflection.Register(grpcServer)
	grpcServer.Serve(tcp_connect)


	//grpcServer := grpc.NewServer()




	//pb.RegisterYourOwnServer(grpcServer, &my_service)
	// Register reflection service on gRPC server.
	//reflection.Register(grpcServer)
	//grpcServer.Serve(tcp_connect)
}