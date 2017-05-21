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
	zexServerAddr = flag.String("zex_server_addr", "127.0.0.1:10000", "Zex server in the format of host:port")
)

func main() {
	flag.Parse()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial(*zexServerAddr, opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewZexClient(conn)

	grpclog.Printf("Start send pipeline task to in Zex")
	stream, err := client.Pipeline(context.Background())
	if err != nil {
		grpclog.Fatalf("%v.Pipeline(_) = _, %v", client, err)
	}

	cmd := pb.Cmd{pb.CmdType(1),"A.callA",[]byte("Body")}
	if err := stream.Send(&cmd); err != nil {
		grpclog.Fatalf("%v.Send(%v) = %v", stream, cmd, err)
	}

	uuid, err := stream.CloseAndRecv()
	if err == io.EOF {
		grpclog.Printf("close stream")
	} else if err != nil {
		grpclog.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
		return
	}
	grpclog.Printf("Pipeline close: %v", uuid)
	_, err_run := client.RunPipeline(context.Background(), uuid)
	if err_run != nil {
		grpclog.Fatalf("%v.Pipeline(_) = _, %v", client, err)
	}

}