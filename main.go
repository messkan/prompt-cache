package main

import (
	"log"
	"net/http"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/gin-gonic/gin"
)

func main() {
	opts := badger.DefaultOptions("./data")
	opts.WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	r.POST("/cache/set", func(c *gin.Context) {
		var body struct {
			Key        string `json:"key" binding:"required"`
			Value      string `json:"value" binding:"required"`
			TTLSeconds int    `json:"ttl_seconds" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := db.Update(func(txn *badger.Txn) error {
			e := badger.NewEntry([]byte(body.Key), []byte(body.Value))
			if body.TTLSeconds > 0 {
				e.WithTTL(time.Duration(body.TTLSeconds) * time.Second)
			}
			return txn.SetEntry(e)
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.GET("/cache/get/:key", func(c *gin.Context) {
		key := c.Param("key")
		var val []byte
		err := db.View(func(txn *badger.Txn) error {
			item, err := txn.Get([]byte(key))
			if err != nil {
				return err
			}
			return item.Value(func(v []byte) error {
				val = append([]byte{}, v...)
				return nil
			})
		})
		if err != nil {
			if err == badger.ErrKeyNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"key": key, "value": string(val)})
	})
}
