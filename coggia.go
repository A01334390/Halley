package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

/** This is the Smart Contract Structure */
type SimpleChaincode struct {
}

/** We have the Wallet */

type Wallet struct {
	Address string `json:"address"`
	Balance int    `json:"balance"`
}

// Main
// Deploys the contract and starts the execution
func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Transaction Chaincode implementation: %s", err)
	}
}

// Starts the chaincode execution
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

// Invoke
// This will help us handle any request for functions
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("Invoke is running: " + function)
	//Simple if statement for function handling

	if function == "initWallet" {
		return t.initWallet(stub, args)
	} else if function == "transferFunds" {
		return t.transferFunds(stub, args)
	} else if function == "readWallet" {
		return t.readWallet(stub, args)
	} else if function == "getWalletsByRange" {
		return t.getWalletsByRange(stub, args)
	}

	//If the function needed didn't exist, then we return an error
	fmt.Println("Invoke didn't find function: " + function)
	return shim.Error("Received Unknown function invocation")
}

//InitWallet assigns the address and initial balance of an Wallet account
//1. Receives and sanitizes the input
//2. Assigns it to an Wallet object
//3. Saves the Wallet to the blockchain
//4. Adds the Wallet to an index to find it faster later

func (t *SimpleChaincode) initWallet(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	// 	  0			  1
	// Address	Initial Balance

	if len(args) != 2 {
		return shim.Error("Incorrect Number of arguments, expecting 2")
	}

	//Input Sanitation as this part is really important
	fmt.Printf(" - Initializing Wallet - ")

	if len(args[0]) <= 0 || len(args[0]) <= 0 {
		return shim.Error("Arguments can't be non empty")
	}

	//Variable initialization
	address := args[0]
	balance, _ := strconv.Atoi(args[1])
	if err != nil {
		return shim.Error("2nd Argument must be a numeric string")
	}

	//Create the Wallet object and convert it to bytes to save
	Wallet := Wallet{Address: address, Balance: balance}
	WalletJSONasBytes, err := json.Marshal(Wallet)
	if err != nil {
		return shim.Error(err.Error())
	}

	//Save the Wallet to the blockchain
	err = stub.PutState(address, WalletJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//Create an Index to look faster for Wallets
	indexName := "address~balance"
	addressBalanceIndexKey, err := stub.CreateCompositeKey(indexName, []string{Wallet.Address, strconv.Itoa(Wallet.Balance)})
	if err != nil {
		return shim.Error(err.Error())
	}

	//Save Index to State
	value := []byte{0x00}
	stub.PutState(addressBalanceIndexKey, value)

	//Wallet saved and indexed, return success
	fmt.Println(" - END Wallet Init - ")
	return shim.Success(nil)
}

//ReadWallet searches for an Wallet by address to look at it's information
//1. We take the data and sanitize it
//2. We search for this Wallet on the Blockchain
//3. We return the Wallet data as a JSON Document

func (t *SimpleChaincode) readWallet(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var address, jsonResp string
	var err error

	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments, expecting the address to query")
	}

	address = args[0]
	valAsBytes, err := stub.GetState(address)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + address + "\"}"
		return shim.Error(jsonResp)
	} else if valAsBytes == nil {
		jsonResp = "{\"Error\":\"Wallet does not exist: " + address + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsBytes)
}

//TransferFunds transfers funds from an Wallet to the other
//[NOTE] This part REALLY needs to be as minimal as possible
//1. We take the input and sanitize it
//2. We search for both Wallets in the blockchain
//3. There's a check where an Wallet can only spend as much as he has
//4. Funds are transfered
//5. Wallet states are updated and pushed to the Blockchain

func (t *SimpleChaincode) transferFunds(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//		 0			1		   2
	//		from		to		balance

	if len(args) < 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	//Variable setting from - to - ammount to be transfered
	from := args[0]
	to := args[1]
	transfer, _ := strconv.Atoi(args[2])

	//if Wallet 'from' doesn't exist, then the transfer halts
	fromAsBytes, err := stub.GetState(from)
	if err != nil {
		return shim.Error("Failed to get Wallet: " + err.Error())
	} else if fromAsBytes == nil {
		return shim.Error("Wallet 1 doesn't exist")
	}

	//if Wallet 'to' doesn't exist, then the transfer halts
	toAsBytes, err := stub.GetState(to)
	if err != nil {
		return shim.Error("Failed to get Wallet: " + err.Error())
	} else if toAsBytes == nil {
		return shim.Error("Wallet 1 doesn't exist")
	}

	//Make Wallet 'from' usable for us
	WalletFrom := Wallet{}
	err = json.Unmarshal(fromAsBytes, &WalletFrom)
	if err != nil {
		return shim.Error(err.Error())
	}

	//Make Wallet 'To' usable for us
	WalletTo := Wallet{}
	err = json.Unmarshal(toAsBytes, &WalletTo)
	if err != nil {
		return shim.Error(err.Error())
	}

	//This is the main balance transfer mechanism
	//As far as we know, this part is really simple
	//1. Checks if an Wallet has enough funds to transfer to another Wallet
	//2. Checks if the transfer amount is not negative (that'd be really weird)
	//3. Then, it simply 'transfers' it.

	WalletTo.Balance += transfer
	WalletFrom.Balance -= transfer

	//The state is updated to the blockchain for both
	//the 'to' Wallet and the 'from' Wallet

	WalletToAsBytes, _ := json.Marshal(WalletTo)
	err = stub.PutState(to, WalletToAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	WalletFromAsBytes, _ := json.Marshal(WalletFrom)
	err = stub.PutState(from, WalletFromAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println(" - END Transaction (success) - ")
	return shim.Success(nil)
}

func (t *SimpleChaincode) getWalletsByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2")
	}

	startKey := args[0]
	endKey := args[1]

	resultsIterator, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	//Buffer is a JSON Array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		//Add a comma before array members, supress ir for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		//Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")
	fmt.Printf("- get Wallet by RANGE queryResult:\n%s\n", buffer.String())
	return shim.Success(buffer.Bytes())
}
