package cmd

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"github.com/dmitryrpm/zex/proto"
)

// Registry services to Zex
func RegisterZex(service *zex.Service, addr *string) {

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	zexConn, err := grpc.Dial(*addr, opts...)
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
		return
	}

	zexClient := zex.NewZexClient(zexConn)
	_, errReg := zexClient.Register(context.Background(), service)
	if errReg != nil {
		grpclog.Fatalf("%v.Registry(_) = _, %v: ", zexClient, err)
	}

	zexConn.Close()
}
