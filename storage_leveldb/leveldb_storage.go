package storage_leveldb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/dmitryrpm/zex/storage"
)

// ---------------------------
// LevelDB constructor
// ---------------------------
func New(path string) (db *LevelDB, err error) {
	levelDB, err := leveldb.OpenFile(path, nil)
	if err != nil{
		return
	}
	return &LevelDB{DB: levelDB}, err
}

// ---------------------------
// LevelDB structure
//---------------------------
type LevelDB struct {
	DB *leveldb.DB
}

func (st *LevelDB) GetIterator() storage.Iterator {
	return st.DB.NewIterator(nil, nil)
}

func (st *LevelDB) GetRowsCount() int {
	var levelDbLen int
	i := st.DB.NewIterator(nil, nil)
	for i.Next() {levelDbLen++}
	i.Release()
	return levelDbLen
}

func (st *LevelDB) NewTransaction() storage.Transaction {
	return &levelDBTransaction{
		storage: st,
		batch: new(leveldb.Batch),
	}
}

//--------------------------------
// LevelDB transaction structure
//-------------------------------
type levelDBTransaction struct {
	storage     *LevelDB
	batch  *leveldb.Batch
}

func (t *levelDBTransaction) Put(k []byte, v []byte) {
	t.batch.Put(k, v)
}

func (t *levelDBTransaction) Delete(k []byte) {
	t.batch.Delete(k)
}

func (t *levelDBTransaction) Commit() error {
	return t.storage.DB.Write(t.batch, nil)
}
