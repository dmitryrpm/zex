package storage_leveldb

import "github.com/syndtr/goleveldb/leveldb"
import "zex/storage"

// New constructor LevelDB
func New(path string) (db *DBLevelStorage, err error) {
	levelDB, err := leveldb.OpenFile(path, nil)
	if err != nil{
		return
	}
	return &DBLevelStorage{DB: levelDB}, err
}

// Storage LevelDB structure
type DBLevelStorage struct {
	DB *leveldb.DB
}

func (st *DBLevelStorage) GetIterator() storage.Iterator {
	return st.DB.NewIterator(nil, nil)
}

func (st *DBLevelStorage) GetRowsCount() int {
	var levelDbLen int
	i := st.DB.NewIterator(nil, nil)
	for i.Next() {levelDbLen++}
	i.Release()
	return levelDbLen
}


func (st *DBLevelStorage) NewTransaction() storage.Transaction {
	return &LevelDBTransaction{
		storage: st,
		batch: new(leveldb.Batch),
	}
}

// Transaction LevelDB structure
type LevelDBTransaction struct {
	storage     *DBLevelStorage
	batch  *leveldb.Batch
}

func (t *LevelDBTransaction) Put(k []byte, v []byte) {
	t.batch.Put(k, v)
}

func (t *LevelDBTransaction) Delete(k []byte) {
	t.batch.Delete(k)
}

func (t *LevelDBTransaction) Commit() error {
	return t.storage.DB.Write(t.batch, nil)
}
