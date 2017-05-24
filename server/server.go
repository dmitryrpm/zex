package server

import (
	"errors"
	"fmt"
	zex "zex/proto"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pborman/uuid"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"io"
	"sync"
)
type ZexServer struct {
	lockForRegger    *sync.RWMutex
	lockForPipper    *sync.RWMutex
	lockForPather    *sync.RWMutex
	RegisterServicesConn map[string]*grpc.ClientConn
	PathToServices   map[string][]string
	PipelineInfo map[string][]*zex.Cmd
}

func New() ZexServer {
	return ZexServer{
		lockForRegger:    &sync.RWMutex{},
		lockForPipper:    &sync.RWMutex{},
		lockForPather:    &sync.RWMutex{},
		RegisterServicesConn: make(map[string]*grpc.ClientConn),
		PipelineInfo:     make(map[string][]*zex.Cmd),
		PathToServices:   make(map[string][]string),
	}
}

// Register service interface impl
func (s *ZexServer) Register(ctx context.Context, service *zex.Service) (*zex.Empty, error) {
	grpclog.Printf("Start registraion service in Zex (%s, %s)", service.Name, service.Addr)
	// ---------------------
	// Do reflection request
	// ---------------------
	serviceKey := service.Addr + `/` + service.Name
	grpclog.Printf("start get all methods in registred \"%s\" service", serviceKey)

	// create connect
	conn, err := grpc.Dial(service.Addr, grpc.WithInsecure())
	if err != nil {
		grpclog.Printf("did not connect: %v", err)
		conn.Close()
		return nil, err
	}

	// get services info
	c := rpb.NewServerReflectionClient(conn)
	grpclog.Printf("do info request")
	informer, err := c.ServerReflectionInfo(ctx)
	if err != nil {
		grpclog.Printf("did not informer: %v", err)
		return nil, err
	}
	grpclog.Printf("do reflation request for list services")
	err = informer.Send(&rpb.ServerReflectionRequest{
		Host:           "localhost", //FIXME
		MessageRequest: &rpb.ServerReflectionRequest_ListServices{},
	})
	if err != nil {
		grpclog.Printf("can't send info req: %v", err)
		return nil, err

	}
	answer, err := informer.Recv()
	grpclog.Println("get answer")
	if err != nil {
		grpclog.Printf("Recv err: %v", err)
		return nil, err
	}
	grpclog.Printf("answer: \"%s\"", answer.String())

	for _, srv := range answer.GetListServicesResponse().GetService() {
		if srv.Name != `grpc.reflection.v1alpha.ServerReflection` {
			err = informer.Send(&rpb.ServerReflectionRequest{
				Host:           "localhost",
				MessageRequest: &rpb.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: srv.Name},
			})
			if err != nil {
				return nil, err
			}
			answerBySrv, err := informer.Recv()
			if err != nil {
				return nil, err
			}

			desc, err := extractFile(answerBySrv.GetFileDescriptorResponse().FileDescriptorProto[0])
			if err != nil {
				return nil, err
			}

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

	// Add to RegisterServicesConn
	s.lockForRegger.Lock()
	s.RegisterServicesConn[serviceKey] = conn
	s.lockForRegger.Unlock()

	grpclog.Printf("Add service to map with key %s successed", serviceKey)
	return &zex.Empty{}, nil
}

// Pipeline service interface impl
func (s *ZexServer) Pipeline(stream zex.Zex_PipelineServer) error {
	grpclog.Printf("listen stream pipeline")

	pid := uuid.New()

	pipeline := make([]*zex.Cmd, 0)

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

			go s.RunPipeline(pid)
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
func (s *ZexServer) Subscribe(ctx context.Context, pid *zex.Pid) (*zex.Empty, error) {
	grpclog.Printf("Start Subscribe uuid %s", pid.ID)

	return &zex.Empty{}, nil
}

// Run pipeline
func (s *ZexServer) RunPipeline(pid string) {
	grpclog.Printf("Start RunPipeline uuid %s", pid)
	grpclog.Println("connect to host localhost")

	var (
		ctx, cancel = context.WithCancel(context.Background())
		pipeline    []*zex.Cmd
		ok          bool
	)
	println(pid)
	s.lockForPipper.RLock()
	pipeline, ok = s.PipelineInfo[pid]
	grpclog.Println("PipelineInfo=", s.PipelineInfo)
	s.lockForPipper.RUnlock()

	if !ok {
		grpclog.Printf("not found pipeline by id %s", pid)
		return
	}

	lengthPipiline := len(pipeline)
	errC := make(chan error, lengthPipiline)
	for _, cmd := range pipeline {
		grpclog.Printf("start cmd... %s", cmd)
		go s.runCmd(ctx, cmd, errC)
	}

	grpclog.Printf("Pipeline wait errors... %s", pid)
	for err := range errC {
		grpclog.Println("asd")
		lengthPipiline--
		if err != nil {
			cancel()
			grpclog.Printf("Pipeline was failed by id %s: %s", pid, err)
			return
		}
		if lengthPipiline == 0 {
			close(errC)
		}
	}

	s.lockForPipper.Lock()
	delete(s.PipelineInfo, pid)
	s.lockForPipper.Unlock()
	grpclog.Printf("Pipeline %s was done ", pid)
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

func (s *ZexServer) runCmd(ctx context.Context, cmd *zex.Cmd, errC chan error) {

	var (
		out        = &any.Any{}
		cc         *grpc.ClientConn
		serviceKey []string
		ok         bool
	)
	s.lockForPather.RLock()
	serviceKey, ok = s.PathToServices[cmd.Path]
	s.lockForPather.RUnlock()

	if !ok || len(serviceKey) == 0 {
		errC <- errors.New(`not found path in services of registry`)
		return
	}

	s.lockForRegger.RLock()
	cc, ok = s.RegisterServicesConn[serviceKey[0]]
	s.lockForRegger.RUnlock()

	if !ok {
		errC <- errors.New(`not found path in services of registry`)
		return
	}

	grpclog.Println("start ", serviceKey, cmd.Path)
	in := byteProto(cmd.Body)
	errC <- grpc.Invoke(ctx, cmd.Path, &in, out, cc)
}
