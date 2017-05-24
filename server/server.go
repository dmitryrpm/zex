/*
Package zex.server defines gRPS DSL service
*/
package server

import (
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pborman/uuid"
	zex "zex/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"io"
	"sync"
	"strings"
)

// Zex options
type zexOptions func(*zexServer)

// Mock wrapper
func WithInvoker(invoker Invoker) zexOptions {
	return func(srv *zexServer) {
		srv.Invoke = invoker
	}
}

// New constructor for ZexServer
func New(opts ...zexOptions) zex.ZexServer {
	srv := &zexServer{
		lockForRegger:    &sync.RWMutex{},
		RegisterServices: make(map[string]*grpc.ClientConn),

		lockForPipper:    &sync.RWMutex{},
		PipelineInfo:     make(map[string][]*zex.Cmd),

		lockForPather:    &sync.RWMutex{},
		PathToServices:   make(map[string][]string),

		Invoke:           grpc.Invoke,
	}

	for _, opt := range opts {
		opt(srv)
	}
	return srv
}

// Invoker function
type Invoker func(ctx context.Context, method string, args, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error


// ZexServer structure
type zexServer struct {
	lockForRegger    *sync.RWMutex
	RegisterServices map[string]*grpc.ClientConn

	lockForPipper    *sync.RWMutex
	PipelineInfo map[string][]*zex.Cmd

	lockForPather    *sync.RWMutex
	PathToServices   map[string][]string

	Invoke Invoker
}

// Register service interface impl
func (s *zexServer) Register(ctx context.Context, service *zex.Service) (*zex.Empty, error) {
	grpclog.Printf("Start registraion service in Zex (%s, %s)", service.Name, service.Addr)
	// ---------------------
	// Do reflection request
	// ---------------------
	serviceKey := service.Addr + "/" + service.Name
	grpclog.Printf("start get all methods in registred \"%s\" service", serviceKey)

	// create connect
	conn, err := grpc.Dial(service.Addr, grpc.WithInsecure())
	if err != nil {
		grpclog.Printf("did not connect: %v", err)
		conn.Close()
		return nil, err
	}

	// get host, port
	ss := strings.Split(service.Addr, ":")
	host, _ := ss[0], ss[1]

	// get services info
	c := rpb.NewServerReflectionClient(conn)
	grpclog.Printf("do info request")
	informer, err := c.ServerReflectionInfo(ctx)
	if err != nil {
		grpclog.Printf("did not informer: %v", err)
		return nil, err
	}

	// send to informer request, and received answer
	grpclog.Printf("do reflation request for list services")
	err = informer.Send(&rpb.ServerReflectionRequest{
		Host:           host, //FIXME
		MessageRequest: &rpb.ServerReflectionRequest_ListServices{},
	})
	if err != nil {
		grpclog.Printf("can't send info req: %v", err)
		return nil, err

	}

	info, err := informer.Recv()
	if err != nil {
		grpclog.Printf("Recv err: %v", err)
		return nil, err
	}
	grpclog.Printf("get info: host: \"%s\"", info.String())

	// Load FileDescriptorProto with services
	for _, srv := range info.GetListServicesResponse().GetService() {
		if srv.Name != "grpc.reflection.v1alpha.ServerReflection" {
			err = informer.Send(&rpb.ServerReflectionRequest{
				Host: host,
				MessageRequest: &rpb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: srv.Name},
			})
			if err != nil {
				return nil, err
			}

			answer, err := informer.Recv()
			if err != nil {
				return nil, err
			}

			// unpack file get file description
			desc, err := extractFile(answer.GetFileDescriptorResponse().FileDescriptorProto[0])
			if err != nil {
				return nil, err
			}

			// read file, get services and register it
			for _, serviceObj := range desc.GetService() {
				for _, methodObj := range serviceObj.GetMethod() {
					method := fmt.Sprintf("/%s.%s/%s", desc.GetPackage(), serviceObj.GetName(), methodObj.GetName())
					grpclog.Printf("registry method %s => %s", serviceKey, method)
					s.lockForPather.Lock()
					s.PathToServices[method] = []string{serviceKey}
					s.lockForPather.Unlock()
				}
			}

		}
	}

	// Add to RegisterServices with locks
	s.lockForRegger.Lock()
	s.RegisterServices[serviceKey] = conn
	s.lockForRegger.Unlock()

	grpclog.Printf("Add service to map with key %s successed", serviceKey)
	return &zex.Empty{}, nil
}

// Pipeline service interface impl
func (s *zexServer) Pipeline(stream zex.Zex_PipelineServer) error {
	grpclog.Printf("listen stream pipeline")

	pid := uuid.New()

	pipeline := make([]*zex.Cmd, 0)

	// open stream, after close run pipeline
	for {
		cmd, err := stream.Recv()
		if err == io.EOF {
			grpclog.Printf("close stream, %s", s.PipelineInfo)

			err = stream.SendAndClose(&zex.Pid{ID: pid})
			if err != nil {
				return err
			}
			s.lockForPipper.Lock()
			s.PipelineInfo[pid] = pipeline
			s.lockForPipper.Unlock()

			// like send to scheduler =))))
			go s.runPipeline(pid)
			return nil
		}

		if err != nil {
			grpclog.Printf("%v.Pipeline(_) = _, %v", cmd, err)
			return err
		} else {
			grpclog.Println("Add cmd to pipeline", cmd)
			pipeline = append(pipeline, cmd)
		}
	}

	return nil
}

// Subscribe service interface impl
func (s *zexServer) Subscribe(ctx context.Context, pid *zex.Pid) (*zex.Empty, error) {
	grpclog.Printf("Start Subscribe uuid %s", pid.ID)

	return &zex.Empty{}, nil
}

// Run pipeline
func (s *zexServer) runPipeline(pid string) {
	grpclog.Printf("Start RunPipeline uuid %s", pid)
	grpclog.Println("connect to host localhost")

	// Get context for cancel all goroutine calls
	var (
		ctx, cancel = context.WithCancel(context.Background())
		pipeline    []*zex.Cmd
		ok          bool
	)

	s.lockForPipper.RLock()
	pipeline, ok = s.PipelineInfo[pid]
	s.lockForPipper.RUnlock()

	// Check pipeline exists
	if !ok {
		grpclog.Printf("not found pipeline by id %s", pid)
		return
	}

	// Create chan errC and wait doing all goroutine
	lengthPipiline := len(pipeline)
	errC := make(chan error, lengthPipiline)
	for _, cmd := range pipeline {
		go s.callCmd(ctx, cmd, errC)
	}

	grpclog.Printf("Pipeline wait errors... %s", pid)
	for err := range errC {
		lengthPipiline--

		// If one of this fail, all failed
		if err != nil {
			cancel()
			grpclog.Printf("Pipeline was failed by id %s: %s", pid, err)
			return
		}

		// Close context if ok
		if lengthPipiline == 0 {
			close(errC)
		}
	}

	// delete pipeline from map
	s.lockForPipper.Lock()
	delete(s.PipelineInfo, pid)
	s.lockForPipper.Unlock()
	grpclog.Printf("Pipeline %s was done ", pid)
}

// Call command service
func (s *zexServer) callCmd(ctx context.Context, cmd *zex.Cmd, errC chan error) {
	var (
		out        = &any.Any{}
		cc         *grpc.ClientConn
		serviceKey []string
		ok         bool
	)
	s.lockForPather.RLock()
	serviceKey, ok = s.PathToServices[cmd.Path]
	s.lockForPather.RUnlock()

	// if not found
	if !ok || len(serviceKey) == 0 {
		errC <- errors.New(`not found path in PathToServices`)
		return
	}

	s.lockForRegger.RLock()
	cc, ok = s.RegisterServices[serviceKey[0]]
	s.lockForRegger.RUnlock()

	// if incorrect path
	if !ok {
		errC <- errors.New(`not found path in RegisterServices`)
		return
	}

	grpclog.Println("start ", serviceKey, cmd.Path)
	in := byteProto(cmd.Body)
	// The grpc.Invoke or Mock in test returns an error, or nil to chan
	errC <- s.Invoke(ctx, cmd.Path, &in, out, cc)
}


func extractFile(gz []byte) (*descriptor.FileDescriptorProto, error) {
	fd := new(descriptor.FileDescriptorProto)
	if err := proto.Unmarshal(gz, fd); err != nil {
		return nil, fmt.Errorf("malformed FileDescriptorProto: %v", err)
	}

	return fd, nil
}

type byteProto []byte

func (b byteProto) Marshal() ([]byte, error) {
	return b, nil
}

func (b byteProto) Reset()         {}
func (b byteProto) String() string { return string(b) }
func (b byteProto) ProtoMessage()  {}