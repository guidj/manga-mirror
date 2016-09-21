package storage

import (
	"github.com/boltdb/bolt"
)

const DB_BUCKET = "mngrdr"

func Init(db *bolt.DB) {
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(DB_BUCKET))
		if err != nil {
			panic(err)
		}

		return nil
	})
}

//TODO: accept value
//Save stores a key in the data store, with the current Unix timestamp
func Save(db *bolt.DB, key string, value string) (err error) {

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(DB_BUCKET))
		err := b.Put([]byte(key), []byte(value))
		return err
	})

	return
}

//Get reads an item from the data store
func Get(db *bolt.DB, key string) (value string, err error) {

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(DB_BUCKET))
		value = string(b.Get([]byte(key)))

		return nil
	})

	return
}

//Exists check if a key is present in the data store
func Exists(db *bolt.DB, key string) (exists bool, err error) {

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(DB_BUCKET))
		v := b.Get([]byte(key))

		exists = (v != nil)

		return nil
	})

	return
}
