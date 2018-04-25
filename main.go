package main

import (
	"time"
)

type Blockchain struct {
	Blocks              []*Block
	CurrentTransactions []*Transaction
}

type Transaction struct {
	Sender    []byte
	Recipient []byte
	Amount    int64
}

type Block struct {
	Index        int
	Timestamp    int64
	Transactions []*Transaction
	Proof        int64
	PreviousHash []byte
}

func (bc *Blockchain) NewBlock(proof int64, previousHash []byte) *Block {
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

func (bc *Blockchain) NewTransaction(sender, recipient []byte, amount int64) int {
	transaction := &Transaction{
		Sender:    sender,
		Recipient: recipient,
		Amount:    amount,
	}
	bc.CurrentTransactions = append(bc.CurrentTransactions, transaction)

	return bc.Blocks[len(bc.Blocks)-1].Index + 1
}

func main() {

}
