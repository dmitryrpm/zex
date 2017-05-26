package storage


// Abstract Interface for Databases (Facade-pattern)
type Database interface {
	GetIterator() Iterator
	NewTransaction() Transaction
	GetRowsCount() int
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






