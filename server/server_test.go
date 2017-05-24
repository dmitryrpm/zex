package server

import (
	"fmt"
	zex "zex/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
	"google.golang.org/grpc/grpclog"
	"errors"
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
	// other routes cancellation
	if err != nil {
		// if for test need emulations of net latency
		if m.letency > 0 {
			grpclog.Printf("[test] set letency %s", m.letency)
			time.Sleep(m.letency)
			grpclog.Printf("wait %s", m.letency)
		}

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


type MockZexServer struct {
	desc string

	invokeLetency time.Duration // help for cancel when err cancel
	invokeErr     error
	invokeErrWhen int
	pid           string

	setPathToServices   map[string][]string
	setRegisterServices map[string]*grpc.ClientConn
	pipeline            []*zex.Cmd

	expPipeAfter bool
	expCallerCmd []string
}

//
//func (s *MockZexServer) prepare() error {
//	return nil
//}

//func WithPrepare(invoker func) error {
//	return func(srv *MockZexServer) {
//		srv.prepare = invoker
//	}
//}

func TestRunEngine(t *testing.T) {

	errStrLetency := fmt.Sprintf("waiting %s", timeLetency)

	zexMocks := []MockZexServer{
		{
			desc: "success simple",
			pid: "pid-1",
			setPathToServices: map[string][]string{
				"a.A/callA": []string{"localhost:2345"},
			},
			setRegisterServices: map[string]*grpc.ClientConn{
				"localhost:2345": &grpc.ClientConn{},
			},
			pipeline: []*zex.Cmd{
				{
					Path: "a.A/callA",
					Body: []byte("aaaa"),
				},
				{
					Path: "a.A/callA",
					Body: []byte("bbbb"),
				},
			},
			expPipeAfter: false, //was deleted
			expCallerCmd: []string{
				"a.A/callA(aaaa)->%!s(<nil>)",
				"a.A/callA(bbbb)->%!s(<nil>)",
			},

		},
		{
			desc: fmt.Sprintf("success with %s", errStrLetency),
			pid: "pid-2",
			invokeLetency: timeLetency,
			invokeErr: errors.New(errStrLetency),
			setPathToServices: map[string][]string{
				"a.A/callA": []string{"localhost:2345"},
			},
			setRegisterServices: map[string]*grpc.ClientConn{
				"localhost:2345": &grpc.ClientConn{},
			},
			pipeline: []*zex.Cmd{
				{
					Path: "a.A/callA",
					Body: []byte("aaaa"),
				},
				{
					Path: "a.A/callA",
					Body: []byte("bbbb"),
				},
			},
			expPipeAfter: true, //was deleted
			expCallerCmd: []string{
				fmt.Sprintf("a.A/callA(bbbb)->%s", errStrLetency),
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
			s := New(WithInvoker(m.Invoke))

			impl := s.(*zexServer)
			impl.PathToServices = tc.setPathToServices
			impl.RegisterServices = tc.setRegisterServices
			impl.PipelineInfo[tc.pid] = tc.pipeline

			impl.runPipeline(tc.pid)

			_, ok := impl.PipelineInfo[tc.pid]

			if tc.expPipeAfter != ok {
				tt.Errorf("expected equals, but \"%v\" != \"%v\"", tc.expPipeAfter, ok)
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
