package block

import (
	"blockchain/utils"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
	"sync"
	"encoding/hex"
	"net/http"

)
const (
	MINING_DIFFICULTY = 3
	MINING_SENDER = "THE BLOCKCHAIN"
	MINING_REWARD = 1.0
	MINING_TIMER_SEC = 20
	BLOCKCHAIN_PORT_RANGE_START = 5000
	BLOCKCHAIN_PORT_RANGE_END = 5003
	NEIGHBOR_IP_RANGE_START = 0
	NEIGHBOR_IP_RANGE_END = 20

)

type AmountResponse struct {
    Amount float32 `json:"amount"`
}
type Block struct {
	timestamp    int64
	nonce        int
	previousHash [32]byte
	transactions []*Transaction
}

func NewBlock(nonce int, previousHash [32]byte, transactions []*Transaction) *Block {
	b := new(Block)
	b.timestamp = time.Now().UnixNano()
	b.nonce = nonce
	b.previousHash = previousHash
	b.transactions = transactions
	return b
}

func (b *Block) PreviousHash()  [32]byte{
	return b.previousHash

}


func (b *Block) Nonce()  int {
	return b.nonce

}
func (b *Block) Transactions() []*Transaction {
	return b.transactions
}
func (b *Block) Print() {
	fmt.Printf("timestamp         %d\n", b.timestamp)
	fmt.Printf("nonce            %d\n", b.nonce)
	fmt.Printf("previous_hash    %x\n", b.previousHash)
	for _, t := range b.transactions {
		t.Print()
	}
}

func (b *Block) Hash() [32]byte {
	m, _ := json.Marshal(b)
	
	return sha256.Sum256(m)
}

func (b *Block) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Timestamp    int64          `json:"timestamp"`
		Nonce        int            `json:"nonce"`
		PreviousHash string         `json:"previous_hash"`
		Transactions []*Transaction `json:"transactions"`
	}{
		Timestamp:    b.timestamp,
		Nonce:        b.nonce,
		PreviousHash: fmt.Sprintf("%x", b.previousHash),
		Transactions: b.transactions,
	})
}
func (b *Block) UnmarshalJSON(data []byte) error {
	var previousHash string
	v := &struct {
		Timestamp *int64 `json:"timestamp"`
		Nonce *int `json:"nonce"`
		PreviousHash *string `json:"previous_hash"`
		Transactions *[]*Transaction `json:"transaction"`
	}{
      Timestamp: &b.timestamp,
	  Nonce: &b.nonce,
	  PreviousHash: &previousHash,
	  Transactions: &b.transactions,
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err 
	}
	
	ph, err := hex.DecodeString(*v.PreviousHash) 
	if err != nil {
		return err 
	}
	
	copy(b.previousHash[:], ph[:32])
	return nil 
	
}
type Blockchain struct {
	transactionPool    []*Transaction
	chain              []*Block
	blockchainAddress    string
	port                 uint16
	mux                 sync.Mutex

	neighbors   []string

	muxNeighbor     sync.Mutex
}

func NewBlockchain(blockchainAddress string, port uint16) *Blockchain {
    b := &Block{}
    bc := new(Blockchain)
    
    
    bc.blockchainAddress = blockchainAddress
    bc.port = port
    

    bc.CreateBlock(0, b.Hash()) 
    
    return bc
}



func (bc *Blockchain)  Chain() []*Block {
	return bc.chain
}
func (bc *Blockchain) Run() {
	bc.StartSyncNeighbors()
	bc.ResolveConflicts()
}

func (bc *Blockchain) SetNeighbors() {
	bc.neighbors = utils.FindNeighbors(
    utils.GetHost(), bc.port,
	 NEIGHBOR_IP_RANGE_START, NEIGHBOR_IP_RANGE_END,
	 BLOCKCHAIN_PORT_RANGE_START, BLOCKCHAIN_PORT_RANGE_END)
	
	log.Printf("%v", bc.neighbors)
	}

func (bc *Blockchain) SyncNeighbors() {
	bc.muxNeighbor.Lock()
	defer bc.muxNeighbor.Unlock()
	bc.SetNeighbors()
}


func(bc *Blockchain) StartSyncNeighbors() {
   bc.SyncNeighbors()
   _= time.AfterFunc(time.Second * BLOCKCHAIN_NEIGHBOR_SYNC_TIME_SEC, bc.StartSyncNeighbors)
}
func (bc *Blockchain) TransactionPool() []*Transaction {
	return bc.transactionPool
}
func (bc *Blockchain) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Blocks []*Block `json: "chain"`
	}{
		Blocks: bc.chain,
	})
}
func (bc *Blockchain)  UnmarshalJSON(data []byte)  error {
	v := &struct {
		Blocks *[]*Block `json:"chain"`
	}{
		Blocks: &bc.chain,
	}
	if err := json.Unmarshal(data, &v) ; err !=nil {
       return err
	}
	return nil
}

func (bc *Blockchain) CreateBlock(nonce int, previousHash [32]byte) *Block {
	b := NewBlock(nonce, previousHash, bc.transactionPool)
	bc.chain = append(bc.chain, b)
	bc.transactionPool = []*Transaction{} 
	return b
}

func (bc *Blockchain) LastBlock() *Block {
	if len(bc.chain) == 0 {
		return nil
	}
	return bc.chain[len(bc.chain)-1]
}

func (bc *Blockchain) Print() {
	for i, block := range bc.chain {
		fmt.Printf("%s chain %d  %s\n", strings.Repeat("=", 25), i, strings.Repeat("=", 25))
		block.Print()
	}
	fmt.Printf("%s\n", strings.Repeat("=", 25))
}
func (bc *Blockchain) CreateTransaction(sender string, recipient string, value float32,
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature, t *Transaction) bool { 

	// Pass `t` correctly
	isTransacted := bc.AddTransaction(sender, recipient, value, senderPublicKey, s, t)

	return isTransacted
}


func (bc *Blockchain) AddTransaction(sender string, recipient string, value float32,
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature, t *Transaction) bool {

	t = NewTransaction(sender, recipient, value) // Existing t use kar rahe hain, naye t declare nahi kar rahe
   if sender == MINING_SENDER {
	bc.transactionPool = append(bc.transactionPool, t)
	return true
   }
	if bc.VerifyTransactionSignature(senderPublicKey, s, t) {
		if bc.CalculateTotalAmount(sender) < value{
			log.Println("Error : not enough balance in a wallet")
		}
		bc.transactionPool = append(bc.transactionPool, t) // Transaction add kar rahe hain
		return true
	} else {
		log.Println("ERROR: verify transaction")
		return false // Yahan return kar diya, taaki function exit ho jaye
	}


}


 
func (bc *Blockchain) VerifyTransactionSignature(
	senderPublicKey *ecdsa.PublicKey, s *utils.Signature, t *Transaction) bool {
	m, _ := json.Marshal(t)             // Serialize transaction
	h := sha256.Sum256([]byte(m))               // Correct usage of sha256
	return ecdsa.Verify(senderPublicKey, h[:], s.R, s.S) // Verify signature
}

func (bc *Blockchain) CopyTransactionPool() []*Transaction {
	transactions := make([]*Transaction, 0)
	for _ , t := range bc.transactionPool {
		transactions = append(transactions,
		NewTransaction(
			t.senderBlockchainAddress,
			t.recipientBlockchainAddress,
			 t.value))
	}
	return transactions
}
func (bc *Blockchain) ValidProof(nonce int , previousHash [32]byte, transactions []*Transaction, difficulty int) bool {
	zeros := strings.Repeat("0", difficulty)
	guessBlock := Block{0, nonce, previousHash, transactions}
	guessHashStr := fmt.Sprintf("%x", guessBlock.Hash())
	return guessHashStr[:difficulty] == zeros
}
func (bc *Blockchain) ProofOfWork() int {
	transactions := bc.CopyTransactionPool()
	previousHash := bc.LastBlock().Hash()
	nonce := 0
	for !bc.ValidProof(nonce, previousHash, transactions, MINING_DIFFICULTY){
        nonce +=1
	}
   return nonce
}
func (bc *Blockchain) Mining() bool {

	bc.mux.Lock()
	defer bc.mux.Unlock()

	if len(bc.transactionPool)  ==  0 {
		return false
	}
	bc.AddTransaction(MINING_SENDER, bc.blockchainAddress, MINING_REWARD, nil, nil, nil)

	nonce := bc.ProofOfWork()
	previousHash := bc.LastBlock().Hash()
	bc.CreateBlock(nonce, previousHash)
	log.Println("action=mining, status=success")
	for _, n := range bc.neighbors {
		endpoint := fmt.Sprintf("http://%s/consensus", n)
		client := &http.Client{}
	
		req, err := http.NewRequest("PUT", endpoint, nil)
		if err != nil {
			log.Printf("ERROR: Failed to create request: %v", err)
			continue
		}
	
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("ERROR: Request failed: %v", err)
			continue
		}
		defer resp.Body.Close() // Ensure response body is closed to avoid memory leaks
	
		log.Printf("Response: %v", resp)
	}
	
	return true
	
}
func (bc *Blockchain) StartMining() {
	bc.Mining()
	_ = time.AfterFunc(time.Second * MINING_TIMER_SEC, bc.StartMining)
}

func (bc *Blockchain) CalculateTotalAmount(blockchainAddress string) float32 {
	var totalAmount float32 = 0.0
	for _, b := range bc.chain {
		for _, t := range b.transactions {
			value := t.value
			if blockchainAddress == t.recipientBlockchainAddress{
				totalAmount += value
			}
			if blockchainAddress == t.senderBlockchainAddress {
				totalAmount -= value
			}
		}
	}
	return totalAmount
}
func (bc *Blockchain) ValidChain(chain []*Block) bool {
	preBlock := chain[0]
	currentIndex := 1
	for   currentIndex < len(chain) {
		b := chain[currentIndex]
		if b.previousHash != preBlock.Hash() {
			return false
		}

		if !bc.ValidProof(b.Nonce(), b.PreviousHash(), b.Transactions() , MINING_DIFFICULTY) {
			return false
		}
		preBlock = b
		currentIndex += 1
	}
	return true
}
   

func (bc *Blockchain) ResolveConflicts() bool {
	var longestChain []*Block = nil
	maxLength := len(bc.chain) // Ensure Chain is a slice

	for _, n := range bc.neighbors {
		endpoint := fmt.Sprintf("http://%s/chain", n)
		resp, err := http.Get(endpoint)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			var bcResp struct {
				chain []*Block `json:"chain"`
			}

			decoder := json.NewDecoder(resp.Body)
			err = decoder.Decode(&bcResp)
			if err != nil {
				continue
			}

			// Ensure `bcResp.Chain` is a slice, not a function
			if len(bcResp.chain) > maxLength {
				maxLength = len(bcResp.chain)
				longestChain = bcResp.chain
			}
		}
	}

	if longestChain != nil {
		bc.chain = longestChain
		return true
	}
	return false
}


type Transaction struct {
	senderBlockchainAddress    string
	recipientBlockchainAddress string
	value                      float32
}

func NewTransaction(sender string, recipient string, value float32) *Transaction {
	return &Transaction{sender, recipient, value}
}

func (t *Transaction) Print() {
	fmt.Printf("%s\n", strings.Repeat("-", 40))
	fmt.Printf("sender_blockchain_address      %s\n", t.senderBlockchainAddress)
	fmt.Printf("recipient_blockchain_address   %s\n", t.recipientBlockchainAddress)
	fmt.Printf("value                          %.1f\n", t.value)
}
func (t *Transaction) MarshalJSON() ([]byte, error) {
    return json.Marshal(struct {
        Sender    string  `json:"sender_blockchain_address"`
        Recipient string  `json:"recipient_blockchain_address"`
        Value     float32 `json:"value"`
    }{
        Sender:    t.senderBlockchainAddress,
        Recipient: t.recipientBlockchainAddress,
        Value:     t.value,
    })
}
func (t *Transaction) UnmarshalJSON() ([]byte, error) {
    v := &struct {
		Sender    *string  `json:"sender_blockchain_address"`
        Recipient *string  `json:"recipient_blockchain_address"`
        Value     *float32 `json:"value"`
    }{
        Sender:    &t.senderBlockchainAddress,
        Recipient: &t.recipientBlockchainAddress,
        Value:     &t.value,
    }
	if err := json.Unmarshal(data, v); err != nil {
		return err
	}
	return nil

}




type TransactionRequest struct {
	SenderBlockchainAddress *string `json:"Sender_blockchain_address"`
	RecipientBlockchainAddress *string `json:"recipient_blockchain_address"`
	SenderPublicKey *string `json:"sender_public_key"`
	Value *float32 `json:"value"`
	Signature *string `json:"signature"`

}


func (tr *TransactionRequest) Validate() bool {
	if tr.SenderBlockchainAddress == nil ||
	tr.RecipientBlockchainAddress == nil ||
	tr.SenderPublicKey == nil ||
	tr.Value == nil ||
	tr.Signature == nil {
   return false 
}
   return true
}