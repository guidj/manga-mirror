package utils

import (
	"net/url"
	"strconv"
	"time"
	
	"github.com/boltdb/bolt"
)

func AddPage(db *bolt.DB, u *url.URL) error {

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Crawl.Pages"))
		err := b.Put([]byte(u.String()), []byte(strconv.FormatInt(time.Now().Unix(), 10)))
		return err
	})

	return err
}

func AddImage(db *bolt.DB, u *url.URL) error {

	err := db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Crawl.Images"))
		err := b.Put([]byte(u.String()), []byte(strconv.FormatInt(time.Now().Unix(), 10)))
		return err
	})

	return err
}

func IsPageCrawled(db *bolt.DB, u *url.URL) (bool, error) {

	var exists bool

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Crawl.Pages"))
		v := b.Get([]byte(u.String()))

		exists = (v != nil)

		return nil
	})

	if err != nil {
		return false, err
	}

	return exists, nil
}

func IsImageCrawled(db *bolt.DB, u *url.URL) (bool, error) {

	var exists bool

	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("Crawl.Images"))
		v := b.Get([]byte(u.String()))

		exists = (v != nil)

		return nil
	})

	if err != nil {
		return false, err
	}

	return exists, nil
}
