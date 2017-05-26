package storage

import "github.com/syndtr/goleveldb/leveldb"


type Iterator interface {
	Key() []byte
	Value() []byte
	Release()
	Next() bool
}


// Storage Interface
type Storager interface {
	GetIterator() Iterator
	NewTransaction() Transactioner
	GetRowsCount() int
}


// New constructor LevelDB
func NewLevelDB(path string) (db *DbLevelStorage, err error) {
	levelDB, err := leveldb.OpenFile(path, nil)
	if err != nil{
		return
	}
	return &DbLevelStorage{DB: levelDB}, err
}


// Storage LevelDB structure
type DbLevelStorage struct {
	DB *leveldb.DB
}

func (storage *DbLevelStorage) GetIterator() Iterator {
	return storage.DB.NewIterator(nil, nil)
}

func (storage *DbLevelStorage) GetRowsCount() int {
	var levelDbLen int
	i := storage.DB.NewIterator(nil, nil)
	for i.Next() {levelDbLen++}
	i.Release()
	return levelDbLen
}


// Transaction interface
type Transactioner interface {
	Put(k []byte, v []byte)
	Delete(k []byte)
	Commit() error
}


func (st *DbLevelStorage) NewTransaction() Transactioner {
	return &LevelDBTransation{
		storage: st,
		batch: new(leveldb.Batch),
	}
}

// Transaction LevelDB structure
type LevelDBTransation struct {
	storage     *DbLevelStorage
	batch  *leveldb.Batch
}

func (t *LevelDBTransation) Put(k []byte, v []byte) {
	t.batch.Put(k, v)
}

func (t *LevelDBTransation) Delete(k []byte) {
	t.batch.Delete(k)
}

func (t *LevelDBTransation) Commit() error {
	return t.storage.DB.Write(t.batch, nil)
}




