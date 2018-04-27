package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Blockchain ブロックチェーンの構造体
type Blockchain struct {
	Blocks              []*Block       `json:"blocks"`
	CurrentTransactions []*Transaction `json:"current_transactions"`
	Nodes               []*Node        `json:"nodes"`
}

// Transaction トランザクションの構造体
type Transaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
}

// Block ブロックの構造体
type Block struct {
	Index        int            `json:"index"`
	Timestamp    int64          `json:"timestamp"`
	Transactions []*Transaction `json:"transactions"`
	Proof        int64          `json:"proof"`
	PreviousHash string         `json:"previous_hash"`
	Hash         string         `json:"hash"`
}

// Node ノード
type Node struct {
	Address string `json:"address"`
}

// NewBlock ブロックを追加する
func (bc *Blockchain) NewBlock(proof int64, previousHash string) *Block {
	block := &Block{
		Index:        len(bc.Blocks) + 1,
		Timestamp:    time.Now().Unix(),
		Transactions: bc.CurrentTransactions,
		Proof:        proof,
		PreviousHash: previousHash,
	}
	block.SetHash()

	bc.CurrentTransactions = nil
	bc.Blocks = append(bc.Blocks, block)

	return block
}

func (b *Block) SetHash() {
	bytes, err := json.Marshal(b)
	if err != nil {
		panic(err)
	}

	s := sha256.New()
	s.Write(bytes)
	b.Hash = fmt.Sprintf("%x", s.Sum(nil))
}

// NewTransaction トランザクションを追加する
func (bc *Blockchain) NewTransaction(sender, recipient string, amount int64) int {
	transaction := &Transaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
	bc.CurrentTransactions = append(bc.CurrentTransactions, transaction)

	return bc.Blocks[len(bc.Blocks)-1].Index + 1
}

// ProofOfWork PoWをする
func (bc *Blockchain) ProofOfWork(lastBlock *Block) int64 {
	lastProof := lastBlock.Proof
	lastHash := lastBlock.Hash

	proof := int64(0)
	for !bc.ValidProof(lastProof, proof, lastHash) {
		proof++
	}

	return int64(proof)

}

// ValidProof Proofが正しいか
func (bc *Blockchain) ValidProof(lastProof int64, proof int64, lastHash string) bool {
	headers := []byte(strconv.FormatInt(lastProof, 10) + strconv.FormatInt(proof, 10) + lastHash)
	s := sha256.New()
	s.Write(headers)
	hash := fmt.Sprintf("%x", s.Sum(nil))
	fmt.Println(hash)
	return hash[:4] == "0000"
}

func (bc *Blockchain) newGenesisBlock() {
	bc.NewBlock(100, "1")
}

func (bc *Blockchain) registerNode(addr string) {
	parsedURL, err := url.Parse(addr)
	if err != nil {
		panic(err)
	}

	node := &Node{
		Address: parsedURL.Host,
	}
	bc.Nodes = append(bc.Nodes, node)
}

func (bc *Blockchain) validChain(chain []*Block) bool {
	lastBlock := chain[0]
	currentIndex := 1
	for currentIndex < len(chain) {
		block := chain[currentIndex]
		if block.PreviousHash != lastBlock.Hash {
			return false
		}
		fmt.Println(lastBlock.Proof)
		fmt.Println(block.Proof)
		fmt.Println(lastBlock.PreviousHash)
		if !bc.ValidProof(lastBlock.Proof, block.Proof, lastBlock.PreviousHash) {
			fmt.Println("proofが正常じゃないよ")
			return false
		}

		lastBlock = block
		currentIndex++
	}

	return true
}

func (bc *Blockchain) resolveConflicts() bool {
	neighbours := bc.Nodes
	var newChain []*Block
	maxLength := len(bc.Blocks)

	for _, node := range neighbours {
		response, err := http.Get("http://" + node.Address + "/chain")
		if err != nil {
			panic(err)
		}

		defer response.Body.Close()

		if response.StatusCode == 200 {
			var res struct {
				Length int      `json:"length"`
				Chain  []*Block `json:"chain"`
			}
			err := json.NewDecoder(response.Body).Decode(&res)
			if err != nil {
				panic(err)
			}

			length := res.Length
			chain := res.Chain
			if length > maxLength && bc.validChain(chain) {
				maxLength = length
				newChain = chain
			}
		}
	}

	fmt.Println("new chain length")
	fmt.Println(len(newChain))
	if len(newChain) >= 1 {
		bc.Blocks = newChain
		return true
	}

	return false

}

func main() {
	r := gin.Default()

	bc := &Blockchain{}
	bc.newGenesisBlock()

	r.GET("/mine", func(c *gin.Context) {
		lastBlock := bc.Blocks[len(bc.Blocks)-1]
		proof := bc.ProofOfWork(lastBlock)

		block := bc.NewBlock(proof, lastBlock.Hash)
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
		err := c.BindJSON(&transaction)
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

	r.POST("/nodes/register", func(c *gin.Context) {
		var nodes struct {
			Address []string `json:"nodes"`
		}
		c.BindJSON(&nodes)
		if len(nodes.Address) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "Error: Please supply a valid list of nodes",
			})
			return
		}

		for _, addr := range nodes.Address {
			bc.registerNode(addr)
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":     "New nodes have been added",
			"total_nodes": bc.Nodes,
		})
	})

	r.GET("/nodes/resolve", func(c *gin.Context) {
		replaced := bc.resolveConflicts()
		if replaced {
			c.JSON(http.StatusOK, gin.H{
				"message":   "Our chain was replaced",
				"new_chain": bc.Blocks,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "Our chain is authoritative",
				"chain":   bc.Blocks,
			})
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	r.Run(":" + port)
}
