package main

import (
	"flag"
	"fmt"
	"net"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"zex/proto"
	"zex/server"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	port = flag.Int("port", 10000, "The server port")
)

func main() {
	flag.Parse()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}

	levelDB, err := leveldb.OpenFile("zex.db", nil)

	if err != nil {
		panic("incorrect create level db")
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	zs := server.New(levelDB)

	zex.RegisterZexServer(grpcServer, zs)
	grpcServer.Serve(listener)
}
