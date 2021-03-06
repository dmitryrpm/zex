// +build integration
package server

import (
	"fmt"
	"github.com/dmitryrpm/zex/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"testing"
	"google.golang.org/grpc/grpclog"
	"github.com/dmitryrpm/zex/storage/leveldb"
	"net"
	"google.golang.org/grpc/reflection"
	"github.com/dmitryrpm/zex/examples/a"
	"os/exec"
)

const DBPath = "/tmp/zex.db.test"
const zexPort = 4981
const serviceAPort = 4898

func TestItegrate(tt *testing.T) {

	stLevelDB, err := leveldb.New(DBPath)
	cmd := exec.Command("rm", "-rf", DBPath)
	defer cmd.Run()

	// Start zex server
	zexConnect, err := net.Listen("tcp", fmt.Sprintf(":%d", zexPort))
	if err != nil {
		grpclog.Fatalf(err.Error())
		tt.Fail()
	}
	s := grpc.NewServer()
	zs := New(stLevelDB)
	zex.RegisterZexServer(s, zs)
	go s.Serve(zexConnect)

	beforeLenServices := len(zs.RegisterServices)
	if beforeLenServices != 0 {
		tt.Errorf("we registered %d services, but it has been empty", beforeLenServices)
	}

	beforeLenPath := len(zs.PathToServices)
	grpclog.Println(zs.PathToServices)
	if beforeLenPath != 0 {
		tt.Errorf("we registered %d services path, but we have %d rows", 0, beforeLenPath)
	}

	// Register services A
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", serviceAPort))
	if err != nil {
		grpclog.Fatalf(err.Error())
		tt.Fail()
	}

	sa := new(a.AService)
	a.RegisterAServer(s, sa)
	reflection.Register(s)
	go s.Serve(listener)

	grpclog.Println("registering to Zex")
	zexConn, err := grpc.Dial(fmt.Sprintf(":%d", serviceAPort), grpc.WithInsecure())
	if err != nil {
		grpclog.Fatalf("fail to dial: %v", err)
	}
	zexClient := zex.NewZexClient(zexConn)

	_, errc := zexClient.Register(context.Background(),
		&zex.Service{Name: "A", Addr: fmt.Sprintf(":%d", serviceAPort)})
	if errc != nil {
		grpclog.Fatalf("%v.Registry(_) = _, %v: ", zexClient, errc)
	}
	grpclog.Println("registed... close connection")
	zexConn.Close()

	afterLenServices := len(zs.RegisterServices)
	if afterLenServices != 1 {
		tt.Errorf("we registered %d services, but we have %d rows", 1, afterLenServices)
	}

	afterLenPath := len(zs.PathToServices)
	if afterLenPath != 3 {
		tt.Errorf("we registered %d services path, but we have %d rows", 6, afterLenPath)
	}
}
