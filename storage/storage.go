package storage

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Abstract storage Interface
type Storage interface {
	Write(batch *leveldb.Batch, wo *opt.WriteOptions) error
	NewIterator(slice *util.Range, ro *opt.ReadOptions) IteratorInterface
}

// Level db structure
type DbLevelStorage struct {

	DB *leveldb.DB
}

func (storage *DbLevelStorage) Write(batch *leveldb.Batch, wo *opt.WriteOptions) error {
	return storage.DB.Write(batch, wo)
}


func (storage *DbLevelStorage) NewIterator(slice *util.Range, ro *opt.ReadOptions) IteratorInterface {
	return storage.DB.NewIterator(slice, ro)
}

// Level db MOCK structure
type DbLevelStorageMock struct {
}


// Level db structure
type Iterator struct {}

func Next(*Iterator) bool           { return false }
func Key(*Iterator) []byte            { return nil }
func Value(*Iterator) []byte          { return nil }
func Release(*Iterator)               {  }


func (storage *DbLevelStorageMock) Write(batch *leveldb.Batch, wo *opt.WriteOptions) error {
	return nil
}


func (storage *DbLevelStorageMock) NewIterator(slice *util.Range, ro *opt.ReadOptions) IteratorInterface {
	i := Iterator{}
	return i
}


type IteratorInterface interface {
	Next() bool
	Key() []byte
	Value() []byte
	Release()
}




