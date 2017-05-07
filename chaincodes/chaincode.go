/*
Copyright IBM Corp 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// Keys for storing data in the ledger
var openTradesStr = "_opentrades" //Description for the key/value that will store all open trades
var institutionKey = "_institutions"
var patientKey = "_patients"
var historyKey = "_history"

type Institution struct {
	ID       int       `json: "id"`
	Name     string    `json: "name"`
	Patients []Patient `json: "patients"`
}

type Patient struct {
	ID      int       `json: "id"`
	Name    string    `json: "name"`
	Cpf     int       `json: "cpf"`
	History []History `json: "history"`
}

type History struct {
	ID          int    `json: "id"`
	Title       string `json: "title"`
	Description string `json: "description"`
}

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

func main() {
	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// Init resets all the things
func (t *SimpleChaincode) Init(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	var Aval int
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	// Initialize the chaincode
	Aval, err = strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Expecting integer value for init")
	}

	// Write the state to the ledger
	err = stub.PutState("abc", []byte(strconv.Itoa(Aval))) //making a test var "abc", I find it handy to read/write to it right away to test the network
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ============================================================================================================================
// InitDemo - Initializes all that is needed to run the code
//
// 0 = Institution 1 ID
// 1 = Institution 1 name
// 2 = Institution 2 ID
// 3 = Institution 2 name
// 4 = Institution 3 ID
// 5 = Institution 3 name
// ============================================================================================================================
func (t *SimpleChaincode) InitDemo(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var err error

	if len(args) != 6 {
		return nil, errors.New("Incorrect number of arguments. Expecting 6")
	}

	// Declaring variables
	var initInstitutions = Institution{}
	var institutionsList []Institution

	// Creates all institutions with no patients
	initInstitutions = generateInstitution(stub, strconv.Itoa(1), "Hospital 1")
	institutionsList = append(institutionsList, initInstitutions)

	initInstitutions = generateInstitution(stub, strconv.Itoa(2), "Hospital 2")
	institutionsList = append(institutionsList, initInstitutions)

	initInstitutions = generateInstitution(stub, strconv.Itoa(3), "Clinic 1")
	institutionsList = append(institutionsList, initInstitutions)

	// Adds institutions to the ledger via putState
	institutionsBytesToWrite, err := json.Marshal(&institutionsList)
	if err != nil {
		fmt.Println("Error marshalling keys")
		return nil, errors.New("Error marshalling the institutionsBytesToWrite")
	}

	err = stub.PutState(institutionKey, institutionsBytesToWrite)
	if err != nil {
		fmt.Println("Error writting keys institutionsBytesToWrite")
		return nil, errors.New("Error writing the keys institutions	BytesToWrite")
	}

	return nil, nil
}

// Invoke isur entry point to invoke a chaincode function
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {
		return t.Init(stub, "init", args)
	} else if function == "write" {
		return t.write(stub, args)
	} else if function == "initdemo" {
		return t.InitDemo(stub, args)
	}
	fmt.Println("invoke did not find func: " + function)

	return nil, errors.New("Received unknown function invocation")
}

// Query is our entry point for queries
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" { //read a variable
		return t.read(stub, args)
	}
	fmt.Println("query did not find func: " + function)

	return nil, errors.New("Received unknown function query")
}

// write - invoke function to write key/value pair
func (t *SimpleChaincode) write(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, value string
	var err error
	fmt.Println("running write()")

	if len(args) != 2 {
		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the key and value to set")
	}

	key = args[0] //rename for funsies
	value = args[1]
	err = stub.PutState(key, []byte(value)) //write the variable into the chaincode state
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// read - query function to read key/value pair
func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var key, jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the key to query")
	}

	key = args[0]
	valAsbytes, err := stub.GetState(key)
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + key + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil
}

// Generates a new institution
func generateInstitution(stub shim.ChaincodeStubInterface, ID string, name string) Institution {
	var err error

	var initInstitutions = Institution{}

	//Assigns the parameters to the temporary Institution struct
	initInstitutions.ID, err = strconv.Atoi(ID)
	if err != nil {
		msg := "initInstitutions.IdInstitution error: " + ID
		fmt.Println(msg)
		os.Exit(1)
	}

	initInstitutions.Name = name

	fmt.Println("Institution " + name + " successfully created")

	return initInstitutions
}

// Generates a new patient
func generatePatient(stub shim.ChaincodeStubInterface, ID string, name string, cpf string) Patient {
	var err error

	var initPatients = Patient{}

	//Assigns the parameters to the temporary Patient struct
	initPatients.ID, err = strconv.Atoi(ID)
	if err != nil {
		msg := "initPatients.ID error: " + ID
		fmt.Println(msg)
		os.Exit(1)
	}

	initPatients.Cpf, err = strconv.Atoi(cpf)
	if err != nil {
		msg := "initPatients.cpf error: " + cpf
		fmt.Println(msg)
		os.Exit(1)
	}

	initPatients.Name = name

	fmt.Println("Patient " + name + " successfully created")

	return initPatients
}
