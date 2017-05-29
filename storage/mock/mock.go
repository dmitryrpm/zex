package mock

import (
	"github.com/dmitryrpm/zex/proto"
	"github.com/dmitryrpm/zex/storage"
)


// ---------------------------
// LevelDB Mock constructor
// ---------------------------
func NewMock(_ string) (db *LevelDBMock, err error) {
	pipeline := make([]zex.Cmd, 0)
	return &LevelDBMock{
		pipeline: pipeline,
	}, nil
}

// ---------------------------
// LevelDB Mock structure
//---------------------------

type LevelDBMock struct {
	pipeline        []zex.Cmd
	mockIterator    storage.Iterator
}

func (st *LevelDBMock) GetIterator() storage.Iterator {
	return NewMockIterator(&st.pipeline)
}

func (st *LevelDBMock) GetRowsCount() int {
	return len(st.pipeline)
}

func (st *LevelDBMock) NewTransaction() storage.Transaction {
	pipeline := make([]zex.Cmd, len(st.pipeline))
	copy(pipeline, st.pipeline)
	return &levelDBTransactionMock{
		pipeline: pipeline,
		storage: st,
	}
}

//--------------------------------
// LevelDB Mock transaction structure
//-------------------------------
type levelDBTransactionMock struct {
	pipeline         []zex.Cmd
	storage          *LevelDBMock
}

func (t *levelDBTransactionMock) Put(k []byte, v []byte) {
	k_str := string(k)
	cmd := zex.Cmd{zex.CmdType_INVOKE, k_str, v}
	t.pipeline = append(
		t.pipeline, cmd)
}

func (t *levelDBTransactionMock) Delete(k []byte) {
	for i, cmd := range t.pipeline {
		if cmd.Path == string(k) {
			t.pipeline = t.pipeline[:i+copy(t.pipeline[i:], t.pipeline[i+1:])]
		}
	}
}

func (t *levelDBTransactionMock) Commit() error {
	t.storage.pipeline = t.pipeline
	return nil
}

// ---------------------------
// Iterator Mock
// ---------------------------

func NewMockIterator(cmdSlice *[]zex.Cmd) storage.Iterator {
	return &MockIterator{pipeline: *cmdSlice, current: -1}
}

type MockIterator struct {
	pipeline    []zex.Cmd
	current     int
}

func (i *MockIterator) Next() bool {

	if i.current == len(i.pipeline)-1 {
		return false
	} else {
		i.current++
		return true
	}

}
func (i *MockIterator) Release() {
	i.current = -1
}


func (i *MockIterator) Key() []byte {
	cmd := i.pipeline[i.current]
	return []byte(cmd.Path)

}

func (i *MockIterator) Value() []byte {
	cmd := i.pipeline[i.current]
	return []byte(cmd.Body)
}

//func (i *MockIterator) Prev() bool           { i.rErr(); return false }
//func (*MockIterator) Valid() bool            { return false }
//func (i *MockIterator) First() bool          { i.rErr(); return false }
//func (i *MockIterator) Last() bool           { i.rErr(); return false }
//func (i *MockIterator) Seek(key []byte) bool { i.rErr(); return false }
//func (i *MockIterator) Error() error         { return i.err }