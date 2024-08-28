package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	addrMainStr = "0x6e3a03a637b75c68bd068cd1e5f381e52e99cc32"
	addrSecStr  = "0xa188d420908b71c1d98BAC8622b4AE07805cdEDf"
)

func main() {

	file, err := os.ReadFile("./bin/UTC--2024-08-27T18-30-35.677182573Z--6e3a03a637b75c68bd068cd1e5f381e52e99cc32")
	if err != nil {
		log.Fatalf(err.Error())
	}

	key, err := keystore.DecryptKey(file, "0000")
	if err != nil {
		log.Fatalf(err.Error())
	}

	pkData := crypto.FromECDSA(key.PrivateKey)
	pk := hexutil.Encode(pkData)
	fmt.Println("private key ", pk)

	cl, err := ethclient.Dial("http://127.0.0.1:8545")
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer cl.Close()

	addrM := common.HexToAddress(addrMainStr)
	addrS := common.HexToAddress(addrSecStr)

	// transaction data
	//var txData []byte
	var txValue *big.Int

	txValue = new(big.Int)
	txValue.Set(big.NewInt(100000000000000))

	//FETCH nonce
	nonce, err := cl.PendingNonceAt(context.Background(), addrM)
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
		From:     addrM,
		To:       &addrS,
		Gas:      0,
		GasPrice: baseGasPrice,
		Value:    txValue,
		Data:     nil,
	})
	if err != nil {
		log.Fatalf(err.Error())
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
	balance, err := cl.BalanceAt(context.Background(), addrM, nil)
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Println("balance main address ", balance)
	b, _ := cl.BalanceAt(context.Background(), addrS, nil)
	fmt.Println("balance receipt address ", b)

	if balance.Cmp(totalCost) < 0 {
		log.Fatalf("Insufficient funds: balance %s, total cost %s", balance, totalCost)
	}

	//Compose transaction
	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		GasTipCap: maxProirityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gasLimit.Uint64(),
		To:        &addrS,
		Value:     txValue,
		Data:      nil,
	})

	chainID, err := cl.ChainID(context.Background())
	if err != nil {
		log.Fatalf(err.Error())
	}
	signerTX := types.LatestSignerForChainID(chainID)
	fmt.Println("chain ID ", chainID)

	sigedTX, err := types.SignTx(tx, signerTX, key.PrivateKey)
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
