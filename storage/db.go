package storage

import (
	"log"

	bolt "go.etcd.io/bbolt"
)

const dbBucket = "mngrdr"

// KeyStore is an wrapper for a Key-Value data store.
type KeyStore struct {
	path string
	db   *bolt.DB
}

// NewKeyStore creates an instance of KeyStore for a given path.
func NewKeyStore(path string) *KeyStore {
	ks := new(KeyStore)
	ks.path = path
	return ks
}

// Init creates the physical storage files and bucket in the data store.
func (ks *KeyStore) Init() {
	var err error
	ks.db, err = bolt.Open(ks.path, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	ks.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(dbBucket))
		if err != nil {
			panic(err)
		}
		return nil
	})
}

// Close closes data store.
func (ks *KeyStore) Close() {
	ks.db.Close()
}

// Save stores a key value pair in the data store.
func (ks *KeyStore) Save(key string, value string) (err error) {
	err = ks.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dbBucket))
		err := b.Put([]byte(key), []byte(value))
		return err
	})
	return
}

// Get reads a key from the data store.
func (ks *KeyStore) Get(key string) (value string, err error) {
	err = ks.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dbBucket))
		value = string(b.Get([]byte(key)))
		return nil
	})
	return
}

// Exists check if a key is present in the data store.
func (ks *KeyStore) Exists(key string) (exists bool, err error) {
	err = ks.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(dbBucket))
		v := b.Get([]byte(key))
		exists = (v != nil)
		return nil
	})
	return
}
