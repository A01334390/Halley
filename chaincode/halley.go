package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"crypto/md5"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	sc "github.com/hyperledger/fabric/protos/peer"
)

// ______  __      ___________             
// ___  / / /_____ ___  /__  /_________  __
// __  /_/ /_  __ `/_  /__  /_  _ \_  / / /
// _  __  / / /_/ /_  / _  / /  __/  /_/ / 
// /_/ /_/  \__,_/ /_/  /_/  \___/_\__, /  
//                                /____/   


// Define the smart contract Structure
type SmartContract struct {
}

/* Define the Wallet Structure with 3 properties
/ [ID] <-- Wallet Identifier made up of an md5 hash
/ [Balance] <-- Balance that indicates the amount of money a wallet holds
/ [Owner] <-- Owner that is the holder of a wallet
*/
type Wallet struct {
	id string `json:"id"`
	balance int `json:"balance"`
	owner string `json:"owner"`
}

/*
*The Init method is called when the Smart Contract 'Halley' is instantiated by the blockchain network
* Best practice is to have any Ledger initialization as a separate function
*/

func (s *SmartContract) Init(APIstub shim.ChaincodeStubInterface) sc.Response {
	return shim.Success(nil);
}

/*
* The Invoke method is called as a result of an application request to the Smart Contract 'Halley'
* The calling application program has also specified the particular smart contract function to be called, with arguments
*/

func (s *SmartContract) Invoke(APIstub shim.ChaincodeStubInterface) sc.Response {
	// Retrieve the requested Smart Contract function and arguments
	function, args := APIstub.GetFunctionAndParameters();
	//Route to the appropiate handler function to interact with the ledger appropiately
	if function == "transferFunds" {
		return s.transferFunds(APIstub, args)
	} else if function == "getWalletsByRange" {
		return s.getWalletsByRange(APIstub, args)
	} else if function == "createWallet" {
		return s.createWallet(APIstub, args)
	} else if function == "getWalletHistory" {
		return s.getWalletHistory(APIstub, args)
	} else if function == "queryWallet" {
		return s.queryWallet(APIstub, args)
	} else if function == "initLedger" {
		return s.initLedger(APIstub, args)
	} else if function == "delete" {
		return s.delete(APIstub, args)
	} else if function == "queryWallet" {
		return s.queryWallet(APIstub,args)
	}

	// If nothing was invoked, launch an error
	return shim.Error("Received Unknown function invocation")
}

/*
* The main method is only relevant in unit test mode. 
* Included here for completeness
*/
func main(){
	//Create a new Smart Contract
	err :=shim.Start(new(SmartContract))
	if(err!=nil){
		fmt.Printf("Error creating a new Smart Contract: %s",err)
	}
}

/*
* transferFunds
* This method is the main driver for the application, it allows the transfer of balance between wallets
* [from]	= This is the id for a wallet that's sending money
* [to]		= This is the id for a wallet that's receiving money
* [balance]	= This is the amount of money that it's being transfered
*/

func (s *SmartContract) transferFunds(APIstub shim.ChaincodeStubInterface, args[]string) sc.Response {
	if len(args) < 3 {
		return shim.Error("Incorrect Number of arguments. Expecting 3")
	}
	
	/** We need to get the from wallet */
	fromAsBytes, err := APIstub.GetState(args[0])
	if err != nil {
		return shim.Error("Failed to retrieve wallet: "+err.Error())
	} else if fromAsBytes == nil {
		return shim.Error("Wallet doesn't exist")
	}
	/** We make it usable */
	from := Wallet {}
	json.Unmarshal(fromAsBytes, &from)

	/** Then we get the to wallet */
	toAsBytes, err:= APIstub.GetState(args[1])
	if err != nil {
		return shim.Error("Failed to retrieve wallet: "+err.Error())
	} else if toAsBytes == nil {
		return shim.Error("Wallet doesn't exist")
	}
	/** We make it usable */
	to := Wallet {}
	json.Unmarshal(toAsBytes, &to)

	/** Make the transfer */
	funds := strconv.Atoi(args[2])
	from.balance -= funds
	to.balance += funds

	/** Persist into a new state */
	fromJSONasBytes, _ := json.Marshal(from)
	err = APIstub.PutState(args[0],fromJSONasBytes)
	if err!=nil {
		return shim.Error(err.Error)
	}

	toJSONasBytes, _ := json.Marshal(to)
	err = APIstub.PutState(args[1],toJSONasBytes)
	if err!=nil {
		return shim.Error(err.Error)
	}

	/* If everything went right */
	return shim.Success(nil)
}
/*
* getWalletsByRange
* This method returns all wallets within a range that reside in the ledger
* [Start]	= Starting key to query the ledger
* [End]		= Ending key to query the ledger
* (JSON)	= JSON with all wallets on the ledger
*/

func (s *SmartContract) getWalletsByRange(APIstub shim.ChaincodeStubInterface, args[]string) sc.Response {
	if len(args) < 2 {
		return shim.Error("Incorrect Number of arguments. Expecting 2")
	}
	resultsIterator, err := APIstub.GetStateByRange(args[0],args[1])
	if err!= nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()
	//buffer is a JSON array containing QueryResults
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return shim.Success(buffer.Bytes())
}

/*
* createWallet
* This method creates a wallet and initializes it into the system
* [id]		= This is an md5 hash that identifies the wallet
* [balance]	= This is the numerical balance of the account
*/

func (s *SmartContract) createWallet(APIstub shim.ChaincodeStubInterface, args[]string) sc.Response {
	if len(args) !=3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	/** We need to sanitize the input */
	if len(args[0] <=0){
		return shim.Error("Arguments must be non-empty strings")
	}
	if len(args[1] <=0){
		return shim.Error("Arguments must be non-empty strings")
	}
	if len(args[2] <=0){
		return shim.Error("Arguments must be non-empty strings")
	}

	/** We create the wallet */
	var wallet = Wallet{
		id: md5.Sum(args[0]),
		balance: strconv.Atoi(args[1]),
		owner:args[2]
	}

	/** We save the wallet */

	walletAsBytes, _ := json.Marshal(wallet)
	APIstub.PutState(walletAsBytes)
	return shim.Success(nil)
}

/*
* getWalletHistory
* This method returns the history of the wallet asset through the network
* [id]		= This is an md5 hash that identifies the wallet
* (JSON)	= JSON Document with the complete history
*/

func (s *SmartContract) getWalletHistory(APIstub shim.ChaincodeStubInterface, args[]string) sc.Response {
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	resultsIterator, err := APIstub.GetHistoryForKey(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	/* buffer is a JSON array containing the historic values for the marble */
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return shim.Success(buffer.Bytes())
}

/*
* queryWallet
* This method returns the current state of a wallet on the ledger
* [id]		= This is an md5 hash that identifies the wallet 
* (JSON)	= JSON Document with the current state of the wallet
*/

func (s *SmartContract) queryWallet(APIstub shim.ChaincodeStubInterface, args[]string) sc.Response {
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	walletAsBytes, _:= APIstub.GetState(args[0])
	return shim.Success(walletAsBytes)
}

/*
* initLedger
* This method intitializes the ledger fast and easily
* [seed]	= Amount of accounts to be created
*/

func (s *SmartContract) initLedger(APIstub shim.ChaincodeStubInterface, args[]string) sc.Response {
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	i := 0
	for i < len(args[0]) {
		wallet := Wallet{
				id: md5.Sum(args[0]),
				balance: strconv.Atoi(args[1]),
				owner:args[2]
		}
		walletAsBytes, _ := json.Marshal(wallet)
		APIstub.PutState(wallet.id,walletAsBytes)
		i = i + 1
	}
} 

/*
* delete
* This method 'deletes' the wallet from the ledger
* [id]		= This is the md5 hash of the wallet to be deleted
*/

func (s *SmartContract) delete(APIstub shim.ChaincodeStubInterface, args[]string) sc.Response {
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	err = APIstub.DelState(args[0])
	if err !=nil {
		return shim.Error("Failed to delete state: "+err.Error())
	}
}

/*
* queryWallet
* This method performs a rich query against the ledger 
* [query]	= This is the query to be performed
*/

func (s *SmartContract) queryWallet(APIstub shim.ChaincodeStubInterface, args[]string) sc.Response {
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	resultsIterator, err := APIstub.GetQueryResult(args[0])
	if err !=nil {
		return shim.Error("Failed to execute query: "+err.Error())
	}
	defer resultsIterator.Close()

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return shim.Success(buffer.Bytes())
}
