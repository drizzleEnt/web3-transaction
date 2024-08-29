package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/crypto/sha3"
)

const (
	linkContract = "0x779877A7B0D9E8603169DdbD7836e478b4624789"

	addrMainPrivateChainStr = "0x6e3a03a637b75c68bd068cd1e5f381e52e99cc32"
	addrSecPrivateChainStr  = "0xa188d420908b71c1d98BAC8622b4AE07805cdEDf"

	addrMainOnChainStr = "0x5295AFCE96E05C716d3C415236572DBAB9b5dA92"
	addrTestOnChainStr = "0x140133C4cd251ef34DD884248f25C964dC75f0A6"
)

func main() {

	file, err := os.ReadFile("./bin/privateKey")
	if err != nil {
		log.Fatalf(err.Error())
	}

	privateKey, err := crypto.HexToECDSA(string(file))
	if err != nil {
		log.Fatalf(err.Error())
	}

	// transaction data
	//var txData []byte
	var txValue *big.Int

	txValue = new(big.Int)
	txValue.Set(big.NewInt(1000000000))

	// ETH client
	cl, err := ethclient.Dial("https://rpc.sepolia.org/")
	if err != nil {
		log.Fatalf("failed to make eth client %s", err.Error())
	}
	defer cl.Close()

	// Address
	fromAddr := common.HexToAddress(addrMainOnChainStr)
	toAddr := common.HexToAddress(addrTestOnChainStr)
	tokenAddr := common.HexToAddress(linkContract)

	//
	transferFnSignature := []byte("transfer(address,uint256)")
	hash := sha3.NewLegacyKeccak256()
	hash.Write(transferFnSignature)
	methodID := hash.Sum(nil)[:4]
	fmt.Println("methodID: ", hexutil.Encode(methodID))

	paddedAddress := common.LeftPadBytes(toAddr.Bytes(), 32)
	fmt.Println("paddedAddress: ", hexutil.Encode(paddedAddress))

	amount := new(big.Int)
	amount.SetString("1000000000000000000", 10)

	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)
	fmt.Println("paddedAmount: ", hexutil.Encode(paddedAmount))

	var data []byte
	data = append(data, methodID...)
	data = append(data, paddedAddress...)
	data = append(data, paddedAmount...)

	//

	//FETCH nonce
	nonce, err := cl.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Println("Fetch nonce ", nonce)

	//Fetch Gas price
	baseGasPrice, err := cl.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatalf(err.Error())
	}
	baseGas := new(big.Int).Set(baseGasPrice)
	maxProirityFeePerGas := new(big.Int).SetUint64(1 * 1e9)
	fmt.Println("gas price ", baseGasPrice)

	//estimate gas limit
	gasEstimate, err := cl.EstimateGas(context.Background(), ethereum.CallMsg{
		From: fromAddr,
		To:   &tokenAddr,
		Data: data,
	})
	if err != nil {
		log.Fatalf("failed estimate gas %s", err.Error())
	}
	gasLimit := new(big.Int).SetUint64(uint64(gasEstimate + gasEstimate/10))
	maxFeePerGas := new(big.Int).Set(baseGas).Add(baseGas, new(big.Int).SetUint64(1e9))

	fmt.Println("gas estimate ", gasEstimate)
	fmt.Println("gas limit ", gasLimit)
	fmt.Println("gas max fee gas ", maxFeePerGas)

	// gas cost
	gasCost := new(big.Int).Mul(big.NewInt(int64(gasEstimate)), maxFeePerGas)
	totalCost := new(big.Int).Add(txValue, gasCost)
	fmt.Println("gas cost ", gasCost)
	fmt.Println("total cost ", totalCost)

	// balance
	balance, err := cl.BalanceAt(context.Background(), fromAddr, nil)
	if err != nil {
		log.Fatalf("failed to get balance on address %s %s", fromAddr.Hex(), err.Error())
	}
	fmt.Println("balance: ", balance)
	if balance.Cmp(totalCost) < 0 {
		log.Fatalf("Insufficient funds: balance %s, total cost %s", balance, totalCost)
	}

	//Compose transaction
	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		GasTipCap: maxProirityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gasLimit.Uint64(),
		To:        &tokenAddr,
		Value:     new(big.Int).SetInt64(0),
		Data:      data,
	})

	chainID, err := cl.ChainID(context.Background())
	if err != nil {
		log.Fatalf(err.Error())
	}
	signerTX := types.LatestSignerForChainID(chainID)
	fmt.Println("chain ID ", chainID)

	sigedTX, err := types.SignTx(tx, signerTX, privateKey)
	if err != nil {
		log.Fatalf(err.Error())
	}

	err = cl.SendTransaction(context.Background(), sigedTX)
	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Println("transaction seccseed ", sigedTX.Hash().Hex())

	var receipt *types.Receipt

	for {
		receipt, err = cl.TransactionReceipt(context.Background(), sigedTX.Hash())
		if err == nil && receipt != nil {
			break
		}

		time.Sleep(3 * time.Second)
	}

	fmt.Println("Tr status ", receipt.Status)
}
