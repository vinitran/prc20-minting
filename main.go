package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/joho/godotenv"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"
)

const (
	envPath = ".env,.env.local"
)

func main() {
	if err := godotenv.Overload(strings.Split(envPath, ",")...); err != nil {
		fmt.Println("Load env error", err.Error())
	}

	amount, err := strconv.Atoi(os.Getenv("AMOUNT_CALL"))
	if err != nil {
		log.Fatal(err)
	}

	currentNonce := GetCurrentNonce()
	for i := 0; i < amount; i++ {
		log.Println(currentNonce)
		err := Mint(currentNonce)
		if err != nil {
			log.Println(err)
			i = i - 1
			continue
		}
		currentNonce = currentNonce + 1
	}
}

func GetCurrentNonce() uint64 {
	client, err := ethclient.Dial(os.Getenv("POLYGON_RPC"))
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := crypto.HexToECDSA(os.Getenv("PRIVATE_KEY"))
	if err != nil {
		log.Fatal(err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		log.Fatal(err)
	}
	return nonce
}

func Mint(nonce uint64) error {
	client, err := ethclient.Dial(os.Getenv("POLYGON_RPC"))
	if err != nil {
		return err
	}

	privateKey, err := crypto.HexToECDSA(os.Getenv("PRIVATE_KEY"))
	if err != nil {
		return err
	}

	value := big.NewInt(0)    // in wei (1 eth)
	gasLimit := uint64(30000) // in units
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}
	toAddress := common.HexToAddress(os.Getenv("TO_ADDRESS"))

	dataStr := fmt.Sprintf(`data:,{"p":"%s","op":"%s","tick":"%s","amt":"%s"}`,
		os.Getenv("PROTOCOL"),
		os.Getenv("OPERATION"),
		os.Getenv("SYMBOL"),
		os.Getenv("AMOUNT"),
	)

	data := []byte(dataStr)
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &toAddress,
		Value:    value,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})

	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		return err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}

	log.Println("Tx Hash", signedTx.Hash().String())
	return nil
}
