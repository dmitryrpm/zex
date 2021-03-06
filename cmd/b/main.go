package main

import (
	"flag"
	"net"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/reflection"
	"sync"
	"github.com/dmitryrpm/zex/proto"
	"github.com/dmitryrpm/zex/examples/b"
	"github.com/dmitryrpm/zex/cmd"
)

var (
	zexServerAddr = flag.String("zex", "127.0.0.1:54321", "Zex server in the format of host:port")
	serverAddr    = flag.String("addr", "127.0.0.1:54323", "The local server address in the format of host:port")
)

func main() {
	flag.Parse()

	grpclog.Printf("starting local service: %s", *serverAddr)
	listener, err := net.Listen("tcp", *serverAddr)
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	s := new(b.BService)
	b.RegisterBServer(grpcServer, s)
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

	service := &zex.Service{Name: "A", Addr: *serverAddr}
	cmd.RegisterZex(service, zexServerAddr)
	wg.Wait()

}
