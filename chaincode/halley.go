package main

import (
	"encoding/json"
	"fmt"
	"strconv"

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
	id      string `json:"id"`
	balance int    `json:"balance"`
	owner   string `json:"owner"`
}

/*
*The Init method is called when the Smart Contract 'Halley' is instantiated by the blockchain network
* Best practice is to have any Ledger initialization as a separate function
 */

func (s *SmartContract) Init(APIstub shim.ChaincodeStubInterface) sc.Response {
	return shim.Success(nil)
}

/*
* The Invoke method is called as a result of an application request to the Smart Contract 'Halley'
* The calling application program has also specified the particular smart contract function to be called, with arguments
 */

func (s *SmartContract) Invoke(APIstub shim.ChaincodeStubInterface) sc.Response {
	// Retrieve the requested Smart Contract function and arguments
	function, args := APIstub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)
	//Route to the appropiate handler function to interact with the ledger appropiately
	if function == "transferFunds" {
		return s.transferFunds(APIstub, args)
	} else if function == "createWallet" {
		return s.createWallet(APIstub, args)
	} else if function == "queryWallet" {
		return s.queryWallet(APIstub, args)
	} else if function == "deleteWallet" {
		return s.deleteWallet(APIstub, args)
	}

	// If nothing was invoked, launch an error
	fmt.Println("Invoke did not find func: " + function)
	return shim.Error("Received Unknown function invocation")
}

/*
* The main method is only relevant in unit test mode.
* Included here for completeness
 */
func main() {
	//Create a new Smart Contract
	err := shim.Start(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating a new Smart Contract: %s", err)
	}
}

/*
* transferFunds
* This method is the main driver for the application, it allows the transfer of balance between wallets
* [from]	= This is the id for a wallet that's sending money
* [to]		= This is the id for a wallet that's receiving money
* [balance]	= This is the amount of money that it's being transfered
 */

func (s *SmartContract) transferFunds(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) < 3 {
		return shim.Error("Incorrect Number of arguments. Expecting 3")
	}
	/* Get the [FROM] wallet state */
	fromAsBytes, err := APIstub.GetState(args[0])
	if err != nil {
		return shim.Error("Failed to get [FROM] Wallet")
	} else if fromAsBytes == nil {
		return shim.Error("Wallet [FROM] does not exist")
	}

	/*Get the [TO] wallet state */
	toAsBytes, err := APIstub.GetState(args[1])
	if err != nil {
		return shim.Error("Failed to get [TO] Wallet")
	} else if fromAsBytes == nil {
		return shim.Error("Wallet [TO] does not exist")
	}

	/* Unmarshal [FROM] wallet */
	from := Wallet{}
	err = json.Unmarshal(fromAsBytes, &from)
	if err != nil {
		return shim.Error("Failed to unmarshal wallet")
	}

	/*Unmarshal [TO] wallet */
	to := Wallet{}
	err = json.Unmarshal(toAsBytes, &to)
	if err != nil {
		return shim.Error("Failed to unmarshal wallet")
	}

	/* Make the transaction */
	funds, err := strconv.Atoi(args[2])
	if err != nil {
		return shim.Error("Failed to parse into Integer")
	}

	from.balance = from.balance - funds
	to.balance = to.balance + funds

	/* Prepare to store into ledger again */
	fromJSONasBytes, _ := json.Marshal(from)
	err = APIstub.PutState(from.id, fromJSONasBytes)
	if err != nil {
		return shim.Error("Error saving the state of wallet [F]" + err)
	}

	toJSONasBytes, _ := json.Marshal(to)
	err = APIstub.PutState(to.id, toJSONasBytes)
	if err != nil {
		return shim.Error("Error saving the state of the wallet [T]" + err)
	}

	/* Success! */
	return shim.Success(nil)
}

/*
* createWallet
* This method creates a wallet and initializes it into the system
* [id]		= This is a number that identifies the wallet
* [balance]	= This is the numerical balance of the account
 */

func (s *SmartContract) createWallet(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	/** We create the wallet */
	id := args[0]
	balance, err := strconv.Atoi(args[1])
	if err != nil {
		return shim.Error("2nd argument can't be parsed into an Integer")
	}
	owner := args[2]

	/** Check if the wallet already exists */
	walletAsBytes, err := APIstub.GetState(id)
	if err != nil {
		return shim.Error("Failed to get marble")
	} else if walletAsBytes != nil {
		fmt.Println("This wallet already exists: " + id)
		return shim.Error("This wallet already exists: " + id)
	}

	/** Create the wallet object and marshal it to JSON */
	wallet := &Wallet{id, balance, owner}
	walletJSONasBytes, err := json.Marshal(wallet)
	if err != nil {
		return shim.Error("Failed to marshal to JSON")
	}

	/** We save the wallet */
	err = APIstub.PutState(id, walletJSONasBytes)
	if err != nil {
		return shim.Error("Failed to save the wallet")
	}
	/** Success! */
	return shim.Success(nil)
}

/*
* queryWallet
* This method returns the current state of a wallet on the ledger
* [id]		= This is an md5 hash that identifies the wallet
* (JSON)	= JSON Document with the current state of the wallet
 */

func (s *SmartContract) queryWallet(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	var id, jsonResp string
	var err error

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	fmt.Println(" ===== START QUERYING WALLET =====")

	id = args[0]
	valAsBytes, err := APIstub.GetState(id) // Get the wallet from the chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + id + "\"}"
		return shim.Error(jsonResp)
	} else if valAsBytes == nil {
		jsonResp = "{\"Error\":\"Wallet does not exist: " + id + "\"}"
		return shim.Error(jsonResp)
	}

	jsonResp = "{\"ID\":\"" + id + "\",\"RESULT\":\"" + string(valAsBytes) + "\"}"
	fmt.Println(" ===== QUERYING WALLET COMPLETE =====")
	return shim.Success(jsonResp)
}

func (s *SmartContract) deleteWallet(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	id := args[0]
	//Delete the key from the state on the ledger
	err := APIstub.DelState(id)
	if err != nil {
		return shim.Error("Failed to delete state")
	}

	return shim.Success(nil)
}
