package boltbrowserweb

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/boltdb/bolt"
	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

var Db *bolt.DB

func Index(c *gin.Context) {

	c.Redirect(301, "/web/html/layout.html")

}

func CreateBucket(c *gin.Context) {

	if c.PostForm("bucket") == "" {
		c.String(200, "no bucket name | n")
	}

	Db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(c.PostForm("bucket")))
		b = b
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	c.String(200, "ok")

}

func DeleteBucket(c *gin.Context) {

	if c.PostForm("bucket") == "" {
		c.String(200, "no bucket name | n")
	}

	Db.Update(func(tx *bolt.Tx) error {
		err := tx.DeleteBucket([]byte(c.PostForm("bucket")))

		if err != nil {

			c.String(200, "error no such bucket | n")
			return fmt.Errorf("bucket: %s", err)
		}

		return nil
	})

	c.String(200, "ok")

}

func DeleteKey(c *gin.Context) {

	if c.PostForm("bucket") == "" || c.PostForm("key") == "" {
		c.String(200, "no bucket name or key | n")
	}

	Db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(c.PostForm("bucket")))
		b = b
		if err != nil {

			c.String(200, "error no such bucket | n")
			return fmt.Errorf("bucket: %s", err)
		}
		key := parseKey(c.PostForm("key"))

		err = b.Delete([]byte(key))

		if err != nil {

			c.String(200, "error Deleting KV | n")
			return fmt.Errorf("delete kv: %s", err)
		}

		return nil
	})

	c.String(200, "ok")

}

func Put(c *gin.Context) {

	if c.PostForm("bucket") == "" || c.PostForm("key") == "" {
		c.String(200, "no bucket name or key | n")
	}

	Db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(c.PostForm("bucket")))
		b = b
		if err != nil {

			c.String(200, "error  creating bucket | n")
			return fmt.Errorf("create bucket: %s", err)
		}
		key := parseKey(c.PostForm("key"))

		err = b.Put([]byte(key), []byte(c.PostForm("value")))

		if err != nil {

			c.String(200, "error writing KV | n")
			return fmt.Errorf("create kv: %s", err)
		}

		return nil
	})

	c.String(200, "ok")

}

func Get(c *gin.Context) {

	res := []string{"nok", ""}

	if c.PostForm("bucket") == "" || c.PostForm("key") == "" {

		res[1] = "no bucket name or key | n"
		c.JSON(200, res)
	}

	Db.View(func(tx *bolt.Tx) error {
		key := parseKey(c.PostForm("key"))

		b := tx.Bucket([]byte(c.PostForm("bucket")))

		if b != nil {

			v := b.Get([]byte(key))

			res[0] = "ok"
			res[1] = string(v)

			fmt.Printf("Key: %s\n", v)

		} else {

			res[1] = "error opening bucket| does it exist? | n"

		}
		return nil

	})

	c.JSON(200, res)

}

type Result struct {
	Result string
	M      map[string]string
}

func PrefixScan(c *gin.Context) {

	res := Result{Result: "nok"}

	res.M = make(map[string]string)

	if c.PostForm("bucket") == "" {

		res.Result = "no bucket name | n"
		c.JSON(200, res)
	}

	count := 0

	searchText := c.PostForm("text")

	if c.PostForm("key") == "" {

		Db.View(func(tx *bolt.Tx) error {
			// Assume bucket exists and has keys
			b := tx.Bucket([]byte(c.PostForm("bucket")))

			if b != nil {
				res.M = collectValuesFromBucket(b, searchText)
				res.Result = "ok"
			} else {

				res.Result = "no such bucket available | n"

			}

			return nil
		})

	} else {

		Db.View(func(tx *bolt.Tx) error {
			// Assume bucket exists and has keys
			b := tx.Bucket([]byte(c.PostForm("bucket"))).Cursor()

			if b != nil {

				prefix := parseKey(c.PostForm("key"))

				for k, v := b.Seek(prefix); bytes.HasPrefix(k, prefix); k, v = b.Next() {
					key := parseDBKey(k)

					value := string(v)
					if searchText != "" && !strings.Contains(value, searchText) {
						continue
					}

					res.M[key] = string(value)
					if count > 2000 {
						break
					}
					count++
				}

				res.Result = "ok"

			} else {

				res.Result = "no such bucket available | n"

			}

			return nil
		})

	}

	c.JSON(200, res)

}

func collectValuesFromBucket(b *bolt.Bucket, searchText string) map[string]string {
	c := b.Cursor()
	res := map[string]string{}
	var count int
	for k, v := c.First(); k != nil; k, v = c.Next() {
		key := parseDBKey(k)
		value := string(v)
		b2 := b.Bucket(k)
		if b2 != nil {
			subRes := collectValuesFromBucket(b2, searchText)
			if len(subRes) == 0 && value == "" { // last bucket is used as value by us
				value = key
			}
			for subKey, subVal := range subRes {
				res[fmt.Sprintf("%v:\n  %v", key, subKey)] = subVal
			}
			continue
		}

		if searchText != "" && !strings.Contains(value, searchText) {
			continue
		}

		res[key] = value

		if count > 2000 {
			break
		}
		count++
	}
	return res
}

func parseDBKey(k []byte) string {
	var key string
	id, err := uuid.FromBytes(k)
	if err != nil {
		key = string(k)
	} else {
		key = id.String()
	}
	return key
}

func parseKey(key string) []byte {
	var prefix []byte
	id, err := uuid.FromString(key)
	if err != nil {
		prefix = []byte(key)
	} else {
		prefix = id.Bytes()
	}
	return prefix
}

func Buckets(c *gin.Context) {

	res := []string{}

	Db.View(func(tx *bolt.Tx) error {

		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {

			b := []string{string(name)}
			res = append(res, b...)
			return nil
		})

	})

	c.JSON(200, res)

}
