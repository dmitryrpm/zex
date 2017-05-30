package server

import (
	"fmt"
	"github.com/dmitryrpm/zex/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
	"google.golang.org/grpc/grpclog"
	"errors"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/dmitryrpm/zex/storage/mock"
	//"io"
)


type mockInvoker struct {
	lock *sync.Mutex
	data []string

	letency   time.Duration
	callCount int
	errCount  int
	err       error
}

const timeLetency = 1 * time.Microsecond

func (m *mockInvoker) Invoke(ctx context.Context, method string, args, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
	var (
		err error
		str = method + "(" + args.(fmt.Stringer).String() + ")"
	)

	m.lock.Lock()
	m.callCount++
	if (m.errCount == 0 || m.callCount == m.errCount) && m.err != nil {
		err = m.err
	}
	m.lock.Unlock()

	// if for test need emulations of net latency
	if m.letency > 0 {
		grpclog.Printf("[test] set letency %s", m.letency)
		time.Sleep(m.letency)
		grpclog.Printf("wait %s", m.letency)
	}


	// other routes cancellation
	if err != nil {
		select {
		case <-ctx.Done():
			err = ctx.Err()
		default:
		}
	}

	str = fmt.Sprintf("%s->%s", str, err)
	m.lock.Lock()
	m.data = append(m.data, str)
	m.lock.Unlock()

	return err
}

func (m *mockInvoker) Dial (target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	var (
		err error
	)
	err = m.err
	return &grpc.ClientConn{}, err
}

type MockDbLevel struct {
	batch *leveldb.Batch
}


func (db *MockDbLevel) Write(batch *leveldb.Batch, wo *opt.WriteOptions) error {
	if batch == nil || batch.Len() == 0 {
		return errors.New("batch null")
	}
	return nil
}

type MockZexServer struct {
	desc string

	invokeLetency time.Duration // help for cancel when err cancel
	invokeErr     error
	invokeErrWhen int
	pid           string

	setPathToServices   map[string][]string
	setRegisterServices map[string]*grpc.ClientConn
	pipeline            []*zex.Cmd

	countRows int
	expCallerCmd []string

}


// Pipeline tests
func TestRunPipelineUnits(t *testing.T) {

	errStrLetency := fmt.Sprintf("waiting %s", timeLetency)

	zexMocks := []MockZexServer{
		{
			desc: "test Pipeline + runPipeline success simple",
			pid:  "pid-1",
			setPathToServices: map[string][]string{
				"/A.A/CallC": []string{"localhost:2345"},
				"/A.A/CallB": []string{"localhost:2345"},
			},
			setRegisterServices: map[string]*grpc.ClientConn{
				"localhost:2345": &grpc.ClientConn{},
			},
			pipeline: []*zex.Cmd{
				{
					Path: "/A.A/CallC",
					Body: []byte("aaaa"),
				},
				{
					Path: "/A.A/CallB",
					Body: []byte("aaaa"),
				},
			},
			expCallerCmd: []string{
				"/A.A/CallC(aaaa)->%!s(<nil>)",
				"/A.A/CallB(aaaa)->%!s(<nil>)",
			},
			countRows: 0,

		},
		{
			desc:          fmt.Sprintf("test Pipeline + runPipeoine fail with letency %s", errStrLetency),
			pid:           "pid-2",
			invokeLetency: timeLetency,
			invokeErr:     errors.New(errStrLetency),
			setPathToServices: map[string][]string{
				"/A.A/CallC": []string{"localhost:2345"},
			},
			setRegisterServices: map[string]*grpc.ClientConn{
				"localhost:2345": &grpc.ClientConn{},
			},
			pipeline: []*zex.Cmd{
				{
					Path: "/A.A/CallC",
					Body: []byte("aaaa"),
				},
			},
			countRows: 1,
			expCallerCmd: []string{
				fmt.Sprintf("/A.A/CallC(aaaa)->%s", errStrLetency),
			},
		},
	}

	for _, tc := range zexMocks {
		t.Run(tc.desc, func(tt *testing.T) {
			m := &mockInvoker{
				lock:     &sync.Mutex{},
				errCount: tc.invokeErrWhen,
				err:      tc.invokeErr,
				letency:  tc.invokeLetency,
			}

			// example for show how work with options
			storageMock, _ := mock.NewMock("test")
			impl := NewMock(m.Invoke, storageMock, m.Dial)
			impl.PathToServices = tc.setPathToServices
			impl.RegisterServices = tc.setRegisterServices

			tr := storageMock.NewTransaction()
			for _, cmd := range tc.pipeline {
				str := tc.pid + "_" + cmd.Path
				tr.Put([]byte(str), []byte(cmd.Body))
			}
			tr.Commit()

			impl.runPipeline(tc.pid)
			count := impl.DB.GetRowsCount()
			if count != tc.countRows {
				tt.Errorf("mock rows shoude be %s,"+
					" but we have rows \"%v\"", tc.countRows, count)
			}

			// sort expected
			sort.Strings(tc.expCallerCmd)
			// sort called
			sort.Strings(m.data)
			if strings.Join(tc.expCallerCmd, ",") != strings.Join(m.data, ",") {
				tt.Errorf("expected equals, but \"%v\" != \"%v\"", tc.expCallerCmd, m.data)
			}
		})
	}
}


type SubscribeTestCase struct {
	pipeline            []*zex.Cmd
	cancelTimeout time.Duration
	defaultTimeout time.Duration
	pid           string
	desc          string
	status        []byte
	error          error
}


// Subscribe tests
func TestSubscribeUnits(t *testing.T) {
	subMocks := []SubscribeTestCase{
		{
			desc: "test correct subscribe empty",
			pid:  "pid-3",
			pipeline: []*zex.Cmd{},
			status: make([]byte, 0),
			error: nil,
			cancelTimeout: time.Second,
			defaultTimeout: time.Second,
		},
		{
			desc: "test correct subscribe with timeout",
			pid:  "pid-6",
			pipeline: []*zex.Cmd{
				{
					Path: "/A.A/CallC",
					Body: []byte("aaaa"),
				},
			},
			status: make([]byte, 0),
			error: errors.New("timeout"),
			cancelTimeout: 50*time.Millisecond,
			defaultTimeout: 10*time.Millisecond,
		},
		{
			desc: "test correct subscribe with cancel context",
			pid:  "pid-4",
			pipeline: []*zex.Cmd{
				{
					Path: "/A.A/CallC",
					Body: []byte("aaaa"),
				},
			},
			status: make([]byte, 0),
			error: errors.New("context cancel"),
			cancelTimeout: 10*time.Millisecond,
			defaultTimeout: 50*time.Millisecond,
		},{
			desc: "test with errors subscribe",
			pid:  "pid-5",
			pipeline: []*zex.Cmd{
				{
					Path: "/A.A/CallC",
					Body: []byte("aaaa"),
				},
				{
					Path: "/A.A/CallB",
					Body: []byte("bbbb"),
				},
			},
			status: []byte("incorrect request"),
			error: errors.New("incorrect request"),
			cancelTimeout: time.Second,
			defaultTimeout: time.Second,
		},
	}

	for _, tc := range subMocks {
		t.Run(tc.desc, func(tt *testing.T) {
			m := &mockInvoker{}
			storageMock, _ := mock.NewMock("test")
			tr := storageMock.NewTransaction()
			for _, cmd := range tc.pipeline {
				str := tc.pid + "_" + cmd.Path
				tr.Put([]byte(str), []byte(cmd.Body))
			}
			if len(tc.pipeline) > 0 {
				tr.Put([]byte(tc.pid + "_status"), tc.status)
			}
			tr.Commit()
			impl := NewMock(m.Invoke, storageMock, m.Dial)
			impl.defaultLoopTimeout = 20 * time.Microsecond
			impl.defaultTimeout = tc.defaultTimeout

			ctx, _ := context.WithTimeout(context.Background(), tc.cancelTimeout)

			_, errS := impl.Subscribe(ctx, &zex.Pid{ID: tc.pid})
			if errS == nil && errS != tc.error ||
				errS != nil && errS.Error() != tc.error.Error() {
				tt.Errorf("error in subsribe call no correct '%v' need '%s'", errS, tc.error)
			}

		})
	}


}


type RegisterTestCase struct {
	service       zex.Service
	pid           string
	desc          string
	error          error
}


func TestRegisterUnits(t *testing.T) {
	RegMocks := []RegisterTestCase{
		{
			desc:          "test incorrect service format",
			pid:           "reg-1",
			service:       zex.Service{"xxx", "xxx.com"},
			error: 		errors.New("incorrect format host:port"),
		},
		//{
		//	desc:          "test correct service",
		//	pid:           "reg-1",
		//	service:       zex.Service{"google.com", "google.com:80"},
		//	error: 		errors.New("incorrect format host:port"),
		//},
	}
	for _, tc := range RegMocks {
		t.Run("test registry services", func(tt *testing.T) {
			m := &mockInvoker{}
			storageMock, _ := mock.NewMock("test")
			impl := NewMock(m.Invoke, storageMock, m.Dial)
			_, errS := impl.Register(context.Background(), &tc.service)
			if errS == nil && errS != tc.error ||
				errS != nil && errS.Error() != tc.error.Error() {
				tt.Errorf("error in subsribe call no correct '%v' need '%s'", errS, tc.error)
			}
		})
	}

}


//Test pipeline
//type PipelineTestCase struct {
//	pipeline            []*zex.Cmd
//	pid           string
//	desc          string
//	status        []byte
//	error          error
//}
//
//type PipelineStreamMock struct {
//	r int
//	pipeline            []*zex.Cmd
//	grpc.ServerStream
//}
//
//func (p *PipelineStreamMock) SendAndClose(pid *zex.Pid) error {
//	return nil
//}
//
//func (p *PipelineStreamMock) Recv() (*zex.Cmd, error) {
//	p.r++
//	err := io.EOF
//	if p.r == len(p.pipeline){
//		err = nil
//	}
//	return p.pipeline[p.r], err
//}
//
//
//func TestPipelineUnits(t *testing.T) {
//
//	RegMocks := []PipelineTestCase{
//		{
//			desc:          "test correct pipeline service",
//			pid:           "pipe-1",
//			pipeline: 	[]*zex.Cmd{
//				{
//					Path: "/A.A/CallC",
//					Body: []byte("aaaa"),
//				},
//				{
//					Path: "/A.A/CallB",
//					Body: []byte("aaaa"),
//				},
//			},
//		},
//	}
//	for _, tc := range RegMocks {
//		t.Run(tc.desc, func(tt *testing.T) {
//			m := &mockInvoker{}
//			storageMock, _ := mock.NewMock("test")
//			impl := NewMock(m.Invoke, storageMock, m.Dial)
//			stream := &PipelineStreamMock{}
//			stream.pipeline = tc.pipeline
//			err := impl.Pipeline(stream)
//
//
//		})
//	}
//
//}
