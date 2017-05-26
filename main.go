package main

import (
	"flag"
	"fmt"
	"net"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"github.com/dmitryrpm/zex/proto"
	"github.com/dmitryrpm/zex/server"
	"github.com/dmitryrpm/zex/storage_leveldb"
)

var (
	port = flag.Int("port", 54321, "The server port")
)

func main() {
	flag.Parse()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		grpclog.Fatalf("failed to listen: %v", err)
	}

	stLevelDB, err := storage_leveldb.New("/tmp/zex.db")

	if err != nil {
		panic("incorrect create level db")
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)


	zs := server.New(stLevelDB)

	zex.RegisterZexServer(grpcServer, zs)
	grpcServer.Serve(listener)
}
