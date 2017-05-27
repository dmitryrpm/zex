package storage


// Abstract Interface for Databases (Facade-pattern)
type Database interface {
	// Get iterator [0..n] rows
	GetIterator() Iterator
	// Create new transaction
	NewTransaction() Transaction
	// Get count all rows into db
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






