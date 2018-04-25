package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Blockchain struct {
	Blocks              []*Block
	CurrentTransactions []*Transaction
}

type Transaction struct {
	Sender    string
	Recipient string
	Amount    int64
}

type Block struct {
	Index        int
	Timestamp    int64
	Transactions []*Transaction
	Proof        int64
	PreviousHash string
}

func (bc *Blockchain) NewBlock(proof int64, previousHash string) *Block {
	block := &Block{
		Index:        len(bc.Blocks) + 1,
		Timestamp:    time.Now().Unix(),
		Transactions: bc.CurrentTransactions,
		Proof:        proof,
		PreviousHash: previousHash,
	}

	bc.CurrentTransactions = nil
	bc.Blocks = append(bc.Blocks, block)

	return block
}

func (bc *Blockchain) NewTransaction(sender, recipient string, amount int64) int {
	transaction := &Transaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
	bc.CurrentTransactions = append(bc.CurrentTransactions, transaction)

	return bc.Blocks[len(bc.Blocks)-1].Index + 1
}

func (bc *Blockchain) ProofOfWork(last_block *Block) int64 {
	lastProof := last_block.Proof
	lastHash := bc.Hash(last_block)

	proof := 0
	for !bc.ValidProof(lastProof, proof, lastHash) {
		fmt.Printf("HASH: %v\nProof: %v\n", lastHash, proof)
		proof++
	}

	return int64(proof)

}

func (bc *Blockchain) Hash(block *Block) string {
	b, err := json.Marshal(block)
	if err != nil {
		panic(err)
	}

	s := sha256.New()
	s.Write(b)
	return fmt.Sprintf("%x", s.Sum(nil))
}

func (bc *Blockchain) ValidProof(last_proof int64, proof int, last_hash string) bool {
	lpb := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(lpb, last_proof)

	pb := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(pb, int64(proof))

	headers := bytes.Join([][]byte{lpb, pb, []byte(last_hash)}, []byte{})
	s := sha256.New()
	s.Write(headers)
	hash := fmt.Sprintf("%x", s.Sum(nil))
	return hash[:4] == "0000"
}

func (bc *Blockchain) newGenesisBlock() {
	bc.NewBlock(100, "1")
}

func main() {
	r := gin.Default()

	bc := &Blockchain{}
	bc.newGenesisBlock()

	r.GET("/mine", func(c *gin.Context) {
		lastBlock := bc.Blocks[len(bc.Blocks)-1]
		proof := bc.ProofOfWork(lastBlock)

		bc.NewTransaction("0", "asdfoiasodifjasoidfjaosjf", 1)

		previousHash := bc.Hash(lastBlock)
		block := bc.NewBlock(proof, previousHash)
		c.JSON(http.StatusOK, gin.H{
			"message":       "New Block Forged",
			"index":         block.Index,
			"transactions":  block.Transactions,
			"proof":         block.Proof,
			"previous_hash": block.PreviousHash,
		})
	})

	r.POST("/transactions/new", func(c *gin.Context) {
		var transaction Transaction
		err := c.BindJSON(transaction)
		if err != nil {
			panic(err)
		}

		index := bc.NewTransaction(transaction.Sender, transaction.Recipient, transaction.Amount)
		c.JSON(http.StatusOK, gin.H{
			"message": "Transaction wil be added to Block " + strconv.Itoa(index),
		})
	})

	r.GET("/chain", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"chain":  bc.Blocks,
			"length": len(bc.Blocks),
		})
	})

	r.Run(":5000")
}
