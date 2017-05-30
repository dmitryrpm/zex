package main

import (
	"flag"
	"io"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"github.com/dmitryrpm/zex/examples/a"
	"github.com/golang/protobuf/proto"
	"github.com/dmitryrpm/zex/proto"
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

	body1, _ := proto.Marshal(&a.Req{Name: "AloxaA"})
	body2, _ := proto.Marshal(&a.Req{Name: "AloxaB"})
	body3, _ := proto.Marshal(&a.Req{Name: "AloxaC"})

	cmd1 := &zex.Cmd{zex.CmdType_INVOKE, "/a.A/CallA", body1}
	if err := stream.Send(cmd1); err != nil {
		grpclog.Fatalf("%v.Send(%v) = %v", stream, cmd1.Path, err)
	}

	cmd2 := &zex.Cmd{zex.CmdType_INVOKE, "/a.A/CallB", body2}
	if err := stream.Send(cmd2); err != nil {
		grpclog.Fatalf("%v.Send(%v) = %v", stream, cmd2.Path, err)
	}

	cmd3 := &zex.Cmd{zex.CmdType_INVOKE, "/a.A/CallC", body3}
	if err := stream.Send(cmd3); err != nil {
		grpclog.Fatalf("%v.Send(%v) = %v", stream, cmd3.Path, err)
	}

	stream.Send(&zex.Cmd{zex.CmdType_INVOKE, "/b.B/CallA", body1})
	stream.Send(&zex.Cmd{zex.CmdType_INVOKE, "/b.B/CallB", body2})
	stream.Send(&zex.Cmd{zex.CmdType_INVOKE, "/b.B/CallC", body3})

	stream.Send(&zex.Cmd{zex.CmdType_INVOKE, "/c.C/CallA", body1})
	stream.Send(&zex.Cmd{zex.CmdType_INVOKE, "/c.C/CallB", body2})
	stream.Send(&zex.Cmd{zex.CmdType_INVOKE, "/c.C/CallC", body3})

	//stream.Send(&zex.Cmd{zex.CmdType_INVOKE, "/d.D/CallA", body3})

	pid, err := stream.CloseAndRecv()
	if err == io.EOF {
		grpclog.Printf("close stream")
	} else if err != nil {
		grpclog.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
		return
	}
	grpclog.Printf("Pipeline close: %v", pid)

	_, strErr := client.Subscribe(context.Background(), pid)
	if strErr != nil {
		grpclog.Printf("Pipeline done with error: %s", strErr)
	} else {
		grpclog.Printf("Pipeline done correct %s", strErr)
	}
}
