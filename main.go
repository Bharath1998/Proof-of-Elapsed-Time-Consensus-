package main

import (
	"bufio"
	"crypto"
	"crypto/sha256"
	"crypto/rsa"
	"crypto/crand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
)

type Wallet struct {
	priv *rsa.PrivateKey
	pub *rsa.PublicKey
}

type User struct {
	name string
	balance int
	wallet Wallet	
}

type Transaction struct {
	from string
	to string
	amount int
	Timestamp string
}

// Block represents each 'item' in the blockchain
type Block struct {
	Index     int
	Timestamp string
	Transactions       [2]Transaction
	MerkleRoot	string
	Hash      string
	PrevHash  string
	Validator string
}
const TOTAL_BLOCKS = 200
const BLOCK_SLEEP_TIME = 10

// Blockchain is a series of validated Blocks
var Blockchain []Block
var tempBlocks []Block
var Users []User

var TxPool [TOTAL_BLOCKS]Transaction
var TxSigns [TOTAL_BLOCKS]string

// candidateBlocks handles incoming blocks for validation
var candidateBlocks = make(chan Block)

// announcements broadcasts winning validator to all nodes
var announcements = make(chan string)

var mutex = &sync.Mutex{}

// validators keeps track of open validators and balances
var validators = make(map[string]int)
var	miner_blockCount = make(map[string]int)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	// create genesis block
	t := time.Now()
	var dummy_trans [2]Transaction
	genesisBlock := Block{}
	genesisBlock = Block{0, t.String(), dummy_trans, "", calculateBlockHash(genesisBlock), "", ""}
	spew.Dump(genesisBlock)
	Blockchain = append(Blockchain, genesisBlock)
	
	//Users 
	Users = append(Users,generateUser("a",1000))
	Users = append(Users,generateUser("b",2000))

	from := "a"
	to := "b"
	var privkey *rsa.PrivateKey
	for i:=0;i<TOTAL_BLOCKS;i++ {
		TxPool[i] = generateTransaction(from,to,i)
		for _,user := range Users{
			if(user.name==from){
				privkey = user.wallet.priv
				break
			}
		}
		TxSigns[i] = signTransaction(TxPool[i], privkey)
		tmp := from
		from = to
		to = tmp
	}

	httpPort := os.Getenv("PORT")

	// start TCP and serve TCP server
	server, err := net.Listen("tcp", ":"+httpPort)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("HTTP Server Listening on port :", httpPort)
	defer server.Close()

	go func() {
		for candidate := range candidateBlocks {
			mutex.Lock()
			tempBlocks = append(tempBlocks, candidate)
			mutex.Unlock()
		}
	}()
	go func() {
		for {
			pickWinner()
		}
	}()

	for {
		conn, err := server.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleConn(conn)
	}
}

// pickWinner creates a lottery pool of validators and chooses the validator who gets to forge a block to the blockchain
// by random selecting from the pool, weighted by amount of tokens staked
func pickWinner() {
	time.Sleep(time.Duration(BLOCK_SLEEP_TIME) * time.Second)
	mutex.Lock()
	temp := tempBlocks
	mutex.Unlock()

	lotteryPool := []string{}
	miner_priorities := make(map[string]int)
	lotteryWinner := "NA"

	if len(temp) > 0 {

		// slightly modified traditional proof of stake algorithm
		// from all validators who submitted a block, weight them by the number of staked tokens
		// in traditional proof of stake, validators can participate without submitting a block to be forged
	OUTER:
		for _, block := range temp {
			// if already in lottery pool, skip
			for _, node := range lotteryPool {
				if block.Validator == node {
					continue OUTER
				}
			}

			// lock list of validators to prevent data race
			mutex.Lock()
			setValidators := validators
			mutex.Unlock()
			// randomly pick winner from lottery pool
			s := rand.NewSource(time.Now().Unix())
			r := rand.New(s)
    		for k := range setValidators {
    			mutex.Lock()
        		miner_priorities[k] = r.Intn(10)
        		mutex.Unlock()
    		}
    	// find minimum first
		min := math.MaxInt64
		for _,v := range miner_priorities {
			if(v<min){
				min=v
			} 
		}

		// add all elements that have a value equal to min
		minKeyList := []string{}
		for k,v := range miner_priorities {
		    if(v == min) {
		        minKeyList=append(minKeyList,k)
		    }
		}
		min_blockCount := math.MaxInt64
    	for key := range minKeyList	{
    		k:=minKeyList[key]
    		// fmt.Println("k:")
    		// fmt.Println(k)
    		x:=miner_blockCount[k]
    		if(x<min_blockCount){
    			min_blockCount=x
    			lotteryWinner=k
    		}
    	}
    	// fmt.Println("LOTTERY WINNER:")
    	// fmt.Println(lotteryWinner)
			_, ok := setValidators[block.Validator]
			if ok {
			// 	for i := 0; i < k; i++ {
					lotteryPool = append(lotteryPool, block.Validator)
				}
			// }
			// fmt.Println("I have mined a block!")
		}

		// add block of winner to blockchain and let all the other nodes know
		for _, block := range temp {
			if block.Validator == lotteryWinner {
				mutex.Lock()
				fmt.Println("Priorities:")
    			fmt.Println(miner_priorities)
				processTransactions(block)
				fmt.Println("\nMINER REWARD STATUS:")
				fmt.Println(validators)
				fmt.Println("\n\n")
				miner_blockCount[lotteryWinner] = miner_blockCount[lotteryWinner] + 1
				Blockchain = append(Blockchain, block)
				fmt.Println("\n-----BLOCK COUNT------\n")
    			fmt.Println(miner_blockCount)
				mutex.Unlock()
				for _ = range validators {
					announcements <- "\n\nWinning Validator: " + lotteryWinner + "\n"
					// announcements <- string(miner_blockCount)
				}
				break
			}
		}
	}

	mutex.Lock()
	tempBlocks = []Block{}
	mutex.Unlock()
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	go func() {
		for {
			msg := <-announcements
			io.WriteString(conn, msg)
		}
	}()
	// validator address
	var address string

	// allow user to allocate number of tokens to stake
	// the greater the number of tokens, the greater chance to forging a new block
	io.WriteString(conn, "Enter some number to validate yourself and start mining:")
	valid_miner := bufio.NewScanner(conn)
	for valid_miner.Scan() {
		_, err := strconv.Atoi(valid_miner.Text())
		if err != nil {
			log.Printf("%v not a number: %v",valid_miner.Text(), err)
			return
		}
		t := time.Now()
		address = calculateHash(t.String())
		validators[address] = 0
		miner_blockCount[address] = 0
		// fmt.Println(validators)s
		break
	}

	// io.WriteString(conn, "\nEnter a new BPM:")

	// wscanBPM := bufio.NewScanner(conn)
	// io.WriteString(conn,"Press Enter to start mining...")
	// bufio.NewReader(conn).ReadBytes('\n')

	go func() {
		// for {
			// take in BPM from stdin and add it to blockchain after conducting necessary validation
			for i := 0;  i<TOTAL_BLOCKS; i++ {
				// bpm, err := strconv.Atoi(scanBPM.Text())
				// if malicious party tries to mutate the chain with a bad input, delete them as a validator and they lose their staked tokens
				// if err != nil {
				// 	log.Printf("%v not a number: %v", scanBPM.Text(), err)
				// 	delete(validators, address)
				// 	conn.Close()
				// }

				mutex.Lock()
				oldLastIndex := Blockchain[len(Blockchain)-1]
				mutex.Unlock()

				var transactions = [2]Transaction {
					Transaction {
						from : "bank",
						to : address,
						amount : 15,
						Timestamp : time.Now().String(),
					},
				}
				var publicKey *rsa.PublicKey
				for _,user := range Users{
					if(TxPool[i].from==user.name){
						publicKey = user.wallet.pub
						break
					}
				}
				if(verifyTransaction(TxPool[i],publicKey,[]byte(TxSigns[i]))){
					fmt.Println("Transaction verified!")
					transactions[1] = TxPool[i]
				}

				// create newBlock for consideration to be forged
				newBlock, err := generateBlock(oldLastIndex, transactions, address)
				if err != nil {
					log.Println(err)
					continue
				}
				if isBlockValid(newBlock, oldLastIndex) {
					candidateBlocks <- newBlock
				}
				time.Sleep(time.Duration(BLOCK_SLEEP_TIME) * time.Second)
				// io.WriteString(conn,"Press Enter to start mining...")
				// bufio.NewReader(conn).ReadBytes('\n')
				// io.WriteString(conn, "\nEnter a new BPM:")
			}
			fmt.Println("Done Mining...")
		// }
	}()

	// simulate receiving broadcast
	for {
		time.Sleep(time.Minute)
		mutex.Lock()
		output, err := json.Marshal(Blockchain)
		mutex.Unlock()
		if err != nil {
			log.Fatal(err)
		}
		io.WriteString(conn, "\nCURRENT BLOCKCHAIN:\n\n"+string(output)+"\n")
	}

}

func signTransaction(t Transaction, rsaPrivateKey *rsa.PrivateKey) string{
	rng := crand.Reader
	record := []byte(t.from + t.to + t.Timestamp  + strconv.Itoa(t.amount))
	hashed := sha256.Sum256(record)
	// fmt.Println(string(hashed))
	signature, err := rsa.SignPKCS1v15(rng, rsaPrivateKey, crypto.SHA256, hashed[:])
	if err != nil {
    	fmt.Fprintf(os.Stderr, "Error from signing: %s\n", err)
    	return ""
	}
	return string(signature)
}

func verifyTransaction(t Transaction, rsaPublicKey *rsa.PublicKey, signature []byte) bool{
	record := []byte(t.from + t.to + t.Timestamp + strconv.Itoa(t.amount))
	hashed := sha256.Sum256(record)
	err := rsa.VerifyPKCS1v15(rsaPublicKey, crypto.SHA256, hashed[:], signature)
	if err != nil {
    	fmt.Println("Error from verification of transaction")
    	fmt.Println(t)
    	return false
	}
	val := false
	for _,user := range Users{
		if(t.from==user.name){
			val = true
			if(t.amount>user.balance){
				fmt.Println("Spending more than intended!")
				fmt.Println(t)
				return false
			}
			break
		}
	}
	return val
}

// isBlockValid makes sure block is valid by checking index
// and comparing the hash of the previous block
func isBlockValid(newBlock, oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}
	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateBlockHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

// SHA256 hasing
// calculateHash is a simple SHA256 hashing function
func calculateHash(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

//calculateBlockHash returns the hash of all block information
func calculateBlockHash(block Block) string {
	record := string(block.Index) + block.Timestamp + string(block.Transactions[0].amount) + block.PrevHash
	return calculateHash(record)
}

func calculateTransactionHash(t Transaction) string {
	var record []byte
	record = []byte(t.from + t.to + t.Timestamp + string(t.amount))
	hash := sha256.Sum256(record)
	s:=""
	copy(hash[:],s)
	return s
}

// generateBlock creates a new block using previous block's hash
func generateBlock(oldBlock Block, Transactions [2]Transaction, address string) (Block, error) {

	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Transactions = Transactions
	newBlock.MerkleRoot = calculateHash(calculateTransactionHash(Transactions[0])+calculateTransactionHash(Transactions[1]))
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateBlockHash(newBlock)
	newBlock.Validator = address

	return newBlock, nil
}

func generateTransaction(from string, to string, amount int) Transaction{
	transaction := Transaction{}
	transaction = Transaction{from,to,amount,time.Now().String()}
	return transaction
}

func generateUser(name string, balance int) User{
	user := User{}
	user = User{name,balance,generateWallet(2048)}
	return user
}

func generateWallet(bits int) Wallet{
	privkey, err := rsa.GenerateKey(crand.Reader, bits)
	if err != nil {
		log.Fatal(err)
	}
	wallet := Wallet{}
	wallet = Wallet{privkey, &privkey.PublicKey}
	return wallet
}
func processTransactions(block Block){
	for _,t := range block.Transactions{
		if(t.from=="bank"){
			validators[t.to]+=t.amount
			return
		}
		for _,user := range Users {
			if(user.name==t.from){
				user.balance-=t.amount
			}
			if(user.name==t.to){
				user.balance+=t.amount
			}
		}
	}

}