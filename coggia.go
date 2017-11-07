package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

//SimpleChaincode implementation
type SimpleChaincode struct {
}

// We build the only and most important object, the User
// We have an address related to the user, where the money will go
// We have a balance related to the user, the amount of money he/she/they hold

type user struct {
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

	if function == "initUser" {
		return t.initUser(stub, args)
	} else if function == "transferFunds" {
		return t.transferFunds(stub, args)
	} else if function == "readUser" {
		return t.readUser(stub, args)
	} else if function == "getUsersByRange" {
		return t.getUsersByRange(stub, args)
	} else if function == "getHistoyForUser" {
		return t.getHistoyForUser(stub, args)
	}
	//If the function needed didn't exist, then we return an error
	fmt.Println("Invoke didn't find function: " + function)
	return shim.Error("Received Unknown function invocation")
}

//InitUser assigns the address and initial balance of an user account
//1. Receives and sanitizes the input
//2. Assigns it to an user object
//3. Saves the user to the blockchain
//4. Adds the user to an index to find it faster later

func (t *SimpleChaincode) initUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var err error
	// 	  0			  1
	// Address	Initial Balance

	if len(args) != 2 {
		return shim.Error("Incorrect Number of arguments, expecting 2")
	}

	//Input Sanitation as this part is really important
	fmt.Printf(" - Initializing User - ")

	if len(args[0]) <= 0 || len(args[0]) <= 0 {
		return shim.Error("Arguments can't be non empty")
	}

	//Variable initialization
	address := args[0]
	balance, _ := strconv.Atoi(args[1])
	if err != nil {
		return shim.Error("2nd Argument must be a numeric string")
	}

	//Create the user object and convert it to bytes to save
	user := user{Address: address, Balance: balance}
	userJSONasBytes, err := json.Marshal(user)
	if err != nil {
		return shim.Error(err.Error())
	}

	//Save the user to the blockchain
	err = stub.PutState(address, userJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//Create an Index to look faster for Users
	indexName := "address~balance"
	addressBalanceIndexKey, err := stub.CreateCompositeKey(indexName, []string{user.Address, strconv.Itoa(user.Balance)})
	if err != nil {
		return shim.Error(err.Error())
	}

	//Save Index to State
	value := []byte{0x00}
	stub.PutState(addressBalanceIndexKey, value)

	//User saved and indexed, return success
	fmt.Println(" - END User Init - ")
	return shim.Success(nil)
}

//ReadUser searches for an user by address to look at it's information
//1. We take the data and sanitize it
//2. We search for this user on the Blockchain
//3. We return the user data as a JSON Document

func (t *SimpleChaincode) readUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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
		jsonResp = "{\"Error\":\"User does not exist: " + address + "\"}"
		return shim.Error(jsonResp)
	}

	return shim.Success(valAsBytes)
}

//TransferFunds transfers funds from an user to the other
//[NOTE] This part REALLY needs to be as minimal as possible
//1. We take the input and sanitize it
//2. We search for both users in the blockchain
//3. There's a check where an user can only spend as much as he has
//4. Funds are transfered
//5. User states are updated and pushed to the Blockchain

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

	//if user 'from' doesn't exist, then the transfer halts
	fromAsBytes, err := stub.GetState(from)
	if err != nil {
		return shim.Error("Failed to get User: " + err.Error())
	} else if fromAsBytes == nil {
		return shim.Error("User 1 doesn't exist")
	}

	//if user 'to' doesn't exist, then the transfer halts
	toAsBytes, err := stub.GetState(to)
	if err != nil {
		return shim.Error("Failed to get User: " + err.Error())
	} else if toAsBytes == nil {
		return shim.Error("User 1 doesn't exist")
	}

	//Make User 'from' usable for us
	userFrom := user{}
	err = json.Unmarshal(fromAsBytes, &userFrom)
	if err != nil {
		return shim.Error(err.Error())
	}

	//Make User 'To' usable for us
	userTo := user{}
	err = json.Unmarshal(toAsBytes, &userTo)
	if err != nil {
		return shim.Error(err.Error())
	}

	//This is the main balance transfer mechanism
	//As far as we know, this part is really simple
	//1. Checks if an user has enough funds to transfer to another user
	//2. Checks if the transfer amount is not negative (that'd be really weird)
	//3. Then, it simply 'transfers' it.

	userTo.Balance += transfer
	userFrom.Balance -= transfer

	//The state is updated to the blockchain for both
	//the 'to' user and the 'from' user

	userToAsBytes, _ := json.Marshal(userTo)
	err = stub.PutState(to, userToAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	userFromAsBytes, _ := json.Marshal(userFrom)
	err = stub.PutState(from, userFromAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	fmt.Println(" - END Transaction (success) - ")
	return shim.Success(nil)
}

func (t *SimpleChaincode) getUsersByRange(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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
	fmt.Printf("- get USER by RANGE queryResult:\n%s\n", buffer.String())
	return shim.Success(buffer.Bytes())
}

func (t *SimpleChaincode) getHistoyForUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	address := args[0]

	fmt.Printf("- start getHistoyForUser: %s\n", address)

	resultsIterator, err := stub.GetHistoryForKey(address)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing historic values for the marble
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(response.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Value\":")

		// if it was a delete operation on given key, then we need to set the
		//corresponding value null. Else, we will write the response.Value
		//as-is (as the Value itself a JSON user)
		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(", \"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getHistoryForMarble returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}
