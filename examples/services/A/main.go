package main

import (
	"flag"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/credentials"
	pb "github.com/zex/zex/proto"
	"google.golang.org/grpc/grpclog"
	"io"
)

var (
	serverAddr = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")
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

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()
	// Init service client
	client := pb.NewZexClient(conn)

	registerZex(client, &pb.Service{Name: "A", Addr: *serverAddr})

	grpclog.Printf("Start send pipeline task to in Zex")
	stream, err := client.Pipeline(context.Background())
	if err != nil {
		grpclog.Fatalf("%v.Pipeline(_) = _, %v", client, err)
	}

	cmd := pb.Cmd{pb.CmdType(1),"1",[]byte("Body")}
	if err := stream.Send(&cmd); err != nil {
		grpclog.Fatalf("%v.Send(%v) = %v", stream, cmd, err)
	}

	ok, err := stream.CloseAndRecv()
	if err == io.EOF {
		grpclog.Printf("close stream")
	} else if err != nil {
		grpclog.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
	}
	grpclog.Printf("Pipeline close: %v", ok)



}