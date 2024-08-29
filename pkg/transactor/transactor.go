package transactor

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

func Transaction(addrMainOnChainStr string, addrTestOnChainStr string) {
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
	txValue.Set(big.NewInt(1000000000000000))

	//approvalABI, _ := abi.JSON(strings.NewReader(`[{"constant":false,"inputs":[{"name":"_spender","type":"address"},{"name":"_value","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"payable":false,"stateMutability":"nonpayable","type":"function"}]`))
	//data := approvalABI.Pack("approve", )

	// ETH client
	cl, err := ethclient.Dial("https://rpc.sepolia.org/")
	if err != nil {
		log.Fatalf("failed to make eth client %s", err.Error())
	}

	// Address
	addr := common.HexToAddress(addrMainOnChainStr)
	receiptAddr := common.HexToAddress(addrTestOnChainStr)

	//FETCH nonce
	nonce, err := cl.PendingNonceAt(context.Background(), addr)
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
		From:     addr,
		To:       &receiptAddr,
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
	balance, err := cl.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		log.Fatalf("failed to get balance on address %s %s", addr.Hex(), err.Error())
	}
	fmt.Println("balance: ", balance)
	if balance.Cmp(totalCost) < 0 {
		log.Fatalf("Insufficient funds: balance %s, total cost %s", balance, totalCost)
	}

	//Как получить приват кей

	//Compose transaction
	tx := types.NewTx(&types.DynamicFeeTx{
		Nonce:     nonce,
		GasTipCap: maxProirityFeePerGas,
		GasFeeCap: maxFeePerGas,
		Gas:       gasLimit.Uint64(),
		To:        &receiptAddr,
		Value:     txValue,
		Data:      nil,
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
