package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/gin-gonic/gin"
)

const DB_NAME string = "7b.db"
const ADMIN_SECRET string = "foo"
const AUTH_BUCKET string = "auth"       // stores password hashes
const BALANCE_BUCKET string = "balance" // stores user balances
const TXN_BUCKET string = "txn"         // stores list of transactions

var db bolt.DB

type Txn struct {
	SrcUserId int64 `json:"srcUserId"`
	DstUserId int64 `json:"dstUserId"`
	Amount    int64 `json:"amount"`
}

func int64ToByteArray(i int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(i))
	return b
}

func byteArrayToInt64(b []byte) int64 {
	return int64(binary.LittleEndian.Uint64(b))
}

func handleGetBalance(c *gin.Context) {
	userId, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad userId."})
		return
	}
	var balance int64
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(BALANCE_BUCKET))
		if bucket == nil {
			return errors.New("balance bucket not found")
		}
		balanceBytes := bucket.Get([]byte(int64ToByteArray(userId)))
		if balanceBytes == nil {
			return fmt.Errorf("failed to get balance: user %v not exist", userId)
		}
		balance = byteArrayToInt64(balanceBytes)
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User does not exist."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": balance})
}

func handleGetTxn(c *gin.Context) {
	txnId, err := strconv.ParseInt(c.Param("txnId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bad txnId."})
		return
	}
	var txn Txn
	err = db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(TXN_BUCKET))
		if bucket == nil {
			return errors.New("transaction bucket not found")
		}
		txnBytes := bucket.Get([]byte(int64ToByteArray(txnId)))
		if txnBytes == nil {
			return fmt.Errorf("failed to get balance: txn %v not exist", txnId)
		}
		err = json.Unmarshal(txnBytes, txn)
		if err != nil {
			return errors.New("failed to deserialize transaction")
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User does not exist."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": txn})
}

func handleCreateTxn(c *gin.Context) {
	// Request must contain sender password
	// Send money from src to dst; src balance cannot go below zero.
}

func handleCreateUser(c *gin.Context) {
	// Request must contain admin secret key
	// Every new user is given 1M tokens
}

func main() {
	api := gin.Default()
	db, err := bolt.Open(DB_NAME, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	admin := api.Group("/admin")

	// Create new user -- only the admin can do this!
	// When a user is created, a password is generated;
	// admin must securely transfer this password to the user.
	admin.POST("/user", handleCreateUser)

	// Get user balance by id
	api.GET("/balance/:userId", handleGetBalance)

	// Get the details of a past transaction by id
	api.GET("/txn/:txnId", handleGetTxn)

	// Make a new transaction
	api.POST("/txn", handleCreateTxn)

	api.Run() // listen and serve on 0.0.0.0:8080
}
