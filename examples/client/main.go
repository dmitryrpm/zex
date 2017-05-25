package main

import (
	"flag"
	"io"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	a "zex/examples/services/A/proto"
	"github.com/golang/protobuf/proto"
	"zex/proto"
)

var (
	zexServerAddr = flag.String("zex", "127.0.0.1:54321", "Zex server in the format of host:port")
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

	client := zex.NewZexClient(conn)

	grpclog.Printf("Start send pipeline task to in Zex")
	stream, err := client.Pipeline(context.Background())
	if err != nil {
		grpclog.Fatalf("%v.Pipeline(_) = _, %v", client, err)
	}

	// тут мы шлем реальное тело
	var (
		req1 = &a.Req{Name: "AloxaA"}
		req2 = &a.Req{Name: "AloxaB"}
		req3 = &a.Req{Name: "AloxaC"}
	)

	body1, _ := proto.Marshal(req1)
	body2, _ := proto.Marshal(req2)
	body3, _ := proto.Marshal(req3)

	// три раза!
	if err := stream.Send(&zex.Cmd{zex.CmdType_INVOKE, "/A.A/CallA", body1}); err != nil {
		grpclog.Fatalf("%v.Send(%v) = %v", stream, "/A.A/CallA", err)
	}
	if err := stream.Send(&zex.Cmd{zex.CmdType_INVOKE, "/A.A/CallB", body2}); err != nil {
		grpclog.Fatalf("%v.Send(%v) = %v", stream, "/A.A/CallB", err)
	}
	if err := stream.Send(&zex.Cmd{zex.CmdType_INVOKE, "/A.A/CallC", body3}); err != nil {
		grpclog.Fatalf("%v.Send(%v) = %v", stream, "/A.A/CallC", err)
	}

	pid, err := stream.CloseAndRecv()
	if err == io.EOF {
		grpclog.Printf("close stream")
	} else if err != nil {
		grpclog.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
		return
	}
	grpclog.Printf("Pipeline close: %v", pid)
}
