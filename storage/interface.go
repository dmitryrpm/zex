package storage

import "github.com/syndtr/goleveldb/leveldb/opt"

// Abstract Interface for Databases (Facade-pattern)
type Database interface {
	// Get iterator [0..n] rows
	GetIterator(start string, stop string) Iterator
	// Create new transaction
	NewTransaction() Transaction
	// Get count all rows into db
	GetRowsCount() int
        Get(key []byte, ro *opt.ReadOptions) (value []byte, err error)
        Put(key, value []byte, wo *opt.WriteOptions) error
	IsErrNotFound (error) bool
}

// Iterator interface
type Iterator interface {
	Key() []byte
	Value() []byte
	Release()
	Next() bool
}

// Transaction interface
type Transaction interface {
	Put(k []byte, v []byte)
	Delete(k []byte)
	Commit() error
}
