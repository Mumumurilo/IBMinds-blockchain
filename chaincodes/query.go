//------------------------------------------------------------------------------
/* Copyright 2016 IBM Corp. All Rights Reserved.
 * IBM Blockchain POC for Bradesco
 * Made by IBM Brasil Innovation Team
 */
//------------------------------------------------------------------------------

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

var openTradesStr = "_opentrades" //Description for the key/value that will store all open trades
var merchantKey = "_merchants"
var paymentsKey = "_payments"
var tokensKey = "_tokens"
var transacoesKey = "_transacoes"
var quantTokens int
var merchantLista []Merchant
var paymentsLista []Payments
var transacoesLista []Transacao
var transacao Transacao
var tokensLista []Token
var err error

// /var logger = shim.NewLogger("BradescoChaincode")
var found = false

type Payments struct {
	ID          int        `json:"id_payment"`
	Descricao   string     `json:"desc_payment"`
	Tokens      []Token    `json:"Token"`
	MyMerchants []Merchant `json:"clientes"`
}

type Merchant struct {
	ID          int     `json:"id_merchant"`
	RazaoSocial string  `json:"razao_social"`
	Cnpj        string  `json:"cnpj"`
	Tokens      []Token `json:"Token"`
}

type Token struct {
	Value string `json:"token_value"`
	Owner int    `json:"owner"`
}

type Transacao struct {
	IdTransac   int     `json:"id"`
	IdComprador int     `json:"id_comprador"`
	IdVendedor  int     `json:"id_fornecedor"`
	QtdTokens   int     `json:"qtd_tokens"`
	Cotacao     float64 `json:"valor_cotacao"`
	Timestamp   string  `json:"horario_transacao"`
	SaldoRsComp float64 `json:"saldo_atual_reais_comprador"`
	SaldoRsVend float64 `json:"saldo_atual_reais_vendedor"`
	SaldoTsComp int     `json:"saldo_atual_tokens_comprador"`
	SaldoTsVend int     `json:"saldo_atual_tokens_vendedor"`
}

// SimpleChaincode example simple Chaincode implementation
type SimpleChaincode struct {
}

//Depois precisamos ajustar o uso da msToTime nas transacoes
const (
	millisPerSecond     = int64(time.Second / time.Millisecond)
	nanosPerMillisecond = int64(time.Millisecond / time.Nanosecond)
)

func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(msInt/millisPerSecond,
		(msInt%millisPerSecond)*nanosPerMillisecond), nil
}

// ---------------------------------------------------------------------------------------------------------------------------- //
// ---------------------------------------------------------------------------------------------------------------------------- //
// ---------------------------------------------------------------------------------------------------------------------------- //
// ------------------------------------------------------ Funções Default ----------------------------------------------------- //
// ---------------------------------------------------------------------------------------------------------------------------- //
// ---------------------------------------------------------------------------------------------------------------------------- //
// ---------------------------------------------------------------------------------------------------------------------------- //

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {

	// logger.SetLevel(shim.LogDebug)

	// logLevel, _ := shim.LogLevel(os.Getenv("SHIM_LOGGING_LEVEL"))
	// shim.SetLoggingLevel(logLevel)

	err := shim.Start(new(SimpleChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}

// ============================================================================================================================
// Invoke - Our entry point for invocations
// ============================================================================================================================
func (t *SimpleChaincode) Invoke(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	if function == "init" { //initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	}
	if function == "initdemo" { //deletes an entity from its state
		return t.initDemo(stub, args)
	}
	if function == "cashin" { //compra de tokens
		return t.cashIn(stub, args)
	}
	if function == "transferTokens" { //Transferência de tokens entre merchants
		return t.transferTokens(stub, args)
	}
	if function == "cashOut" {
		return t.cashOut(stub, args)
	}
	if function == "deleteState" {
		return t.deleteState(stub, args)
	}
	if function == "resetAll" {
		return t.resetAll(stub)
	}

	return nil, errors.New("Pedido de invoke não reconhecido pela função. Reveja o nome da função enviado como parâmetro")
}

// ============================================================================================================================
// Query - Our entry point for Queries
// ============================================================================================================================
func (t *SimpleChaincode) Query(stub shim.ChaincodeStubInterface, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "read" { //read a variable
		return t.read(stub, args)
	}
	if function == "getTokens" { //retorna os valores de tokens de um peer
		return t.getTokens(stub, args)
	}
	if function == "getTransactions" { //Retorna todas as transações realizadas
		return t.getTransactions(stub, args)
	}

	fmt.Println("query did not find func: " + function) //error

	return nil, errors.New("Pedido de query não reconhecido pela função. Reveja o nome da função enviado como parâmetro")
}

// ============================================================================================================================
// Read - read a variable from chaincode state
// P.S.: Essa função será utilizada para receber os valores de tokens dos merchants (payments e FI)
//=============================================================================================================================
func (t *SimpleChaincode) read(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var arguments, jsonResp string
	var err error

	if len(args) < 1 {
		return nil, errors.New("Essa função precisa de argumentos! Apenas o primeiro será lido, mas ela recebe quantos você colocar.")
	}

	arguments = args[0]
	fmt.Println("Buscando pelo id: " + arguments + " no ledger...")
	valAsbytes, err := stub.GetState(arguments) //get the var from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + arguments + "\"}"
		return nil, errors.New(jsonResp)
	}
	if valAsbytes == nil {
		jsonResp = "O argumento enviado para ser lido não retornou nenhum dado do ledger. Por favor, reveja seus argumentos."
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil //send it onward
}

// ---------------------------------------------------------------------------------------------------------------------------- //
// ---------------------------------------------------------------------------------------------------------------------------- //
// ---------------------------------------------------------------------------------------------------------------------------- //
// -------------------------------------------------- Funções Personalizadas -------------------------------------------------- //
// ---------------------------------------------------------------------------------------------------------------------------- //
// ---------------------------------------------------------------------------------------------------------------------------- //
// ---------------------------------------------------------------------------------------------------------------------------- //

// ============================================================================================================================
// Init - Reset em todas as informações do peer
// ============================================================================================================================
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
// initDemo - Inicializa tudo o que é necessário para realizar testes
//
// 0 = ID do Payments
// 1 = Nome do Payments
// 2 = CNPJ do Payments
// 3 = Tokens do Payments
// 4 = ID do FI
// 5 = Nome do FI
// 6 = CNPJ do FI
// 7 = Tokens do FI
// 8 = ID do Merchant 1
// 9 = Nome do Merchant 1
// 10 = CNPJ do Merchant 1
// 11 = Tokens do Merchant 1
// 12 = ID do Merchant 2
// 13 = Nome do Merchant 2
// 14 = CNPJ do Merchant 2
// 15 = Tokens do Merchant 2
// 16 = ID do Merchant 3
// 17 = Nome do Merchant 3
// 18 = CNPJ do Merchant 3
// 19 = Tokens do Merchant 3
// ============================================================================================================================
func (t *SimpleChaincode) initDemo(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 20 {
		return nil, errors.New("Número incorreto de argumentos. Esperando 20")
	}

	var err error
	var volume1, volume2 int

	//Obtêm dos parâmetros a quantidade de tokens a ser criada para o Payments e FI
	volume1, err = strconv.Atoi(args[3])
	if err != nil {
		msg := "volume1 " + args[3]
		fmt.Println(msg)
		return nil, errors.New(msg)
	}
	volume2, err = strconv.Atoi(args[7])
	if err != nil {
		msg := "volume2 " + args[7]
		fmt.Println(msg)
		return nil, errors.New(msg)
	}

	//Declaração de outras variáveis
	var initTransacoes = Transacao{}
	var transacoesList []Transacao
	var initMerchants = Merchant{}
	var merchantsList []Merchant
	var initPayments = Payments{}
	var paymentsList []Payments
	var initFI = Payments{}
	var initTokens = Token{}
	var tokensList []Token

	initTransacoes.IdTransac = -1 // atribui um Id à transação
	initTransacoes.Timestamp = "Genesis Transaction"
	initTransacoes.IdComprador = 0
	initTransacoes.IdVendedor = 0
	initTransacoes.Cotacao = 0.00
	initTransacoes.QtdTokens = 0
	initTransacoes.SaldoRsComp = 0
	initTransacoes.SaldoRsVend = 0
	initTransacoes.SaldoTsComp = 0
	initTransacoes.SaldoTsVend = 0

	//Cria o Payments
	initPayments.ID, err = strconv.Atoi(args[0])
	fmt.Println("ID do payments assignado à lista")
	if err != nil {
		msg := "initPayments.Id_payment " + args[0]
		fmt.Println(msg)
		return nil, errors.New(msg)
	}

	initPayments.Descricao = args[1]
	fmt.Println("Descrição do payments assignado à lista")

	//Cria tokens para o Payments
	for i := 0; i < volume1; i++ {
		initTokens = generateToken(initPayments.ID)
		initPayments.Tokens = append(initPayments.Tokens, initTokens)
		tokensList = append(tokensList, initTokens)
		fmt.Println("Token de número " + strconv.Itoa(i) + " criado e assignado ao Payments")
	}

	//Cria o FI
	initFI.ID, err = strconv.Atoi(args[4])
	fmt.Println("ID do FI assignado à lista")
	if err != nil {
		msg := "(FI) initFI.Id_payment " + args[4]
		fmt.Println(msg)
		return nil, errors.New(msg)
	}

	initFI.Descricao = args[5]
	fmt.Println("Descrição do FI assignado à lista")

	//Cria tokens para o FI
	for i := 0; i < volume2; i++ {
		initTokens = generateToken(initFI.ID)
		initFI.Tokens = append(initFI.Tokens, initTokens)
		tokensList = append(tokensList, initTokens)
		fmt.Println("Token de número " + strconv.Itoa(i) + " criado e assignado ao FI")
	}

	//Cria o merchant 1 com o Payments como owner dos tokens
	initMerchants = generateMerchant(stub, args[8], args[9], args[10], args[11], args[0])
	initPayments.MyMerchants = append(initPayments.MyMerchants, initMerchants)
	merchantsList = append(merchantsList, initMerchants)

	//Cria o merchant 2 com o FI como owner dos tokens
	initMerchants = generateMerchant(stub, args[12], args[13], args[14], args[15], args[4])
	initFI.MyMerchants = append(initFI.MyMerchants, initMerchants)
	merchantsList = append(merchantsList, initMerchants)

	//Cria o merchant 3 com Payments como owner dos tokens
	initMerchants = generateMerchant(stub, args[16], args[17], args[18], args[19], args[0])
	initPayments.MyMerchants = append(initPayments.MyMerchants, initMerchants)
	merchantsList = append(merchantsList, initMerchants)

	//Acrescenta o payments e o FI com todos os dados dos merchants à lista de payments a ser armazenada no ledger
	paymentsList = append(paymentsList, initPayments)
	paymentsList = append(paymentsList, initFI)

	//Payments => putState()
	paymentsBytesToWrite, err := json.Marshal(&paymentsList)
	if err != nil {
		fmt.Println("Error marshalling paymentsBytesToWrite")
		return nil, errors.New("Error marshalling the keys")
	}

	err = stub.PutState(paymentsKey, paymentsBytesToWrite)
	if err != nil {
		fmt.Println("Error writting keys paymentsBytesToWrite")
		return nil, errors.New("Error writing the keys paymentsBytesToWrite")
	}

	//Merchant => putState()
	merchantsBytesToWrite, err := json.Marshal(&merchantsList)
	if err != nil {
		fmt.Println("Error marshalling keys")
		return nil, errors.New("Error marshalling the merchantsBytesToWrite")
	}

	err = stub.PutState(merchantKey, merchantsBytesToWrite)
	if err != nil {
		fmt.Println("Error writting keys paymentsBytesToWrite")
		return nil, errors.New("Error writing the keys paymentsBytesToWrite")
	}

	//Tokens => putState()
	tokensBytesToWrite, err := json.Marshal(&tokensList)
	if err != nil {
		fmt.Println("Error marshalling keys")
		return nil, errors.New("Error marshalling the tokensBytesToWrite")
	}

	err = stub.PutState(tokensKey, tokensBytesToWrite)
	if err != nil {
		fmt.Println("Error writting keys paymentsBytesToWrite")
		return nil, errors.New("Error writing the keys paymentsBytesToWrite")
	}

	//Inicializando as ocorrências de transacoes (genesis)
	transacoesList = append(transacoesList, initTransacoes)

	transacoesBytesToWrite, err := json.Marshal(&transacoesList)
	if err != nil {
		fmt.Println("Error marshalling transacoesBytesToWrite")
		return nil, errors.New("Error marshalling the keys")
	}

	err = stub.PutState(transacoesKey, transacoesBytesToWrite)
	if err != nil {
		fmt.Println("Error writting keys transacoesBytesToWrite")
		return nil, errors.New("Error writing the keys paymentsBytesToWrite")
	}

	return nil, nil

}

// ============================================================================================================================
// getTokens - retorna o valor dos tokens da conta de um peer
//
// 0 = ID do tipo do peer
// 1 = ID do peer
// ============================================================================================================================
func (t *SimpleChaincode) getTokens(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {

	var readDataM []Merchant
	var readDataP []Payments
	var quantTokens, IDPeer int

	if len(args) != 2 {
		return nil, errors.New("Número incorreto de argumentos. Esperando 2 (ID do tipo do peer e ID do peer)")
	}

	//Checa o tipo do peer para instanciar a variável
	if args[0] == merchantKey {
		fmt.Println("Procurando tokens de um merchant")
	} else if args[0] == paymentsKey {
		fmt.Println("Procurando tokens de um payments")
	} else {
		return nil, errors.New("Parâmetros inválidos. A função aceita somente Merchants e Payments como tipo.")
	}

	//Procura o peer no ledger
	jsonResp, err := t.read(stub, args)
	if err != nil {
		return nil, errors.New("Não foi possível ler os dados do ledger para o peer")
	}
	fmt.Println("Peer encontrado no ledger")

	//Converte de byte para o tipo correspondente
	if args[0] == merchantKey {
		json.Unmarshal(jsonResp, &readDataM)
	} else if args[0] == paymentsKey {
		json.Unmarshal(jsonResp, &readDataP)
	}
	fmt.Println("Peer convertido de byte para o tipo correto")

	//Checa se o ID está correto e depois conta a quantidade de tokens
	if args[0] == merchantKey {
		for i := range readDataM {
			ID := strconv.Itoa(readDataM[i].ID)
			if args[1] == ID {
				IDPeer, err = strconv.Atoi(ID)
				if err != nil {
					return nil, errors.New("Conversão Atoi inválida")
				}
				quantTokens = len(readDataM[i].Tokens)
			}
		}
	} else if args[0] == paymentsKey {
		for i := range readDataP {
			ID := strconv.Itoa(readDataP[i].ID)
			if args[1] == ID {
				IDPeer, err = strconv.Atoi(ID)
				if err != nil {
					return nil, errors.New("Conversão Atoi inválida")
				}
				quantTokens = len(readDataP[i].Tokens)
			}
		}
	}
	fmt.Println("ID do peer: " + strconv.Itoa(IDPeer))
	fmt.Println("Quantidade de tokens do peer: " + strconv.Itoa(quantTokens))

	//Converte para byte
	valAsBytes, err := json.Marshal(quantTokens)
	if err != nil {
		return nil, errors.New("Conversão de string falhou")
	}
	fmt.Println("Chegou no fim da função! :)")

	return valAsBytes, nil //send it onward
}

// ============================================================================================================================
// getTransactions - retorna todas as transações realizadas. Possui um flag para retornar apenas as transações feitas com o peer especificado na função.
//
// 0 = isExclusive
// 1 = ID do peer (utilizada caso isExclusive seja True)
// ============================================================================================================================
func (t *SimpleChaincode) getTransactions(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	var readData, arrayRetorno []Transacao
	var isExclusive bool
	var valAsBytes []byte

	if len(args) != 2 {
		return nil, errors.New("Número incorreto de argumentos. Esperando 2")
	}

	argument := []string{transacoesKey}
	fmt.Printf("Array argument: " + argument[0])

	isExclusive, err := strconv.ParseBool(args[0])
	fmt.Println("isExclusive = " + strconv.FormatBool(isExclusive))

	//Procura a lista de transações no ledger
	jsonResp, err := t.read(stub, argument)
	if err != nil {
		return nil, errors.New("Não foi possível ler os dados do ledger")
	}
	fmt.Println("Lista de transações encontrada no ledger")

	json.Unmarshal(jsonResp, &readData)
	fmt.Println("Unmarshal passado")

	//Caso seja exclusivo, lê todas as transações e salva no retorno apenas as que tenham o ID do peer desejado
	if isExclusive {
		idAtual, err := strconv.Atoi(args[1])
		for i := range readData {
			if err != nil {
				return nil, errors.New("Erro na conversão de string")
			}
			if readData[i].IdComprador == idAtual || readData[i].IdVendedor == idAtual {
				fmt.Println("Selecionando transação de número " + strconv.Itoa(i) + " do usuário " + strconv.Itoa(idAtual) + " para retorno")
				arrayRetorno = append(arrayRetorno, readData[i])
			}
		}
		valAsBytes, err = json.Marshal(arrayRetorno)
	} else {
		//Caso não seja exclusivo, apenas iguala o array ao retorno
		valAsBytes, err = json.Marshal(readData)
	}

	fmt.Println("Chegou no fim da função :)")

	return valAsBytes, nil //Retorno
}

// ============================================================================================================================
// cashIn - Payments => Merchant && Payments na compra de Tokens
//
// 0 = ID do comprador
// 1 = Quantidade de tokens a ser comprada
// 2 = ID do vendedor
// 3 = Cotação atual
// 4 = Timestamp/ID da transação
// 5 = Valor atual em reais do comprador
// 6 = Valor atual em reais do vendedor
// 7 = ID da Transação
// ============================================================================================================================
func (t *SimpleChaincode) cashIn(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 8 {
		return nil, errors.New("Incorrect number of arguments. Expecting 8")
	}

	if args[0] == args[2] {
		return nil, errors.New("Foi passado o mesmo ID para o comprador e o vendedor. Reveja os parâmetros passados na função.")
	}

	var isPayment = true //Condição que checa se o Payments está criando tokens ou se é apenas um cash in simples

	/*Recuperando os args*/
	comprador, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Failed to get comprador")
	}
	quantTokens, err = strconv.Atoi(args[1])
	if err != nil {
		return nil, errors.New("Failed to get quantTokens")
	}

	vendedor, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, errors.New("Failed to get vendedor")
	}

	cotacao, err := strconv.ParseFloat(args[3], 64)
	if err != nil {
		return nil, errors.New("Failed to get cotacao")
	}

	transacao.SaldoRsComp, err = strconv.ParseFloat(args[5], 64)
	if err != nil {
		return nil, errors.New("Failed to get saldoRsComp")
	}

	transacao.SaldoRsVend, err = strconv.ParseFloat(args[6], 64)
	if err != nil {
		return nil, errors.New("Failed to get saldoRsVend")
	}

	idTransaction, err := strconv.Atoi(args[7])
	if err != nil {
		return nil, errors.New("Failed to get idTransaction")
	}

	/*Recuperando os states de merchants,payments e transacoes - serao atualizados adiante*/
	merchantsAsBytes, err := stub.GetState(merchantKey)
	if err != nil {
		return nil, errors.New("Failed to get merchant list")
	}

	paymentsAsBytes, err := stub.GetState(paymentsKey)
	if err != nil {
		return nil, errors.New("Failed to get merchant list")
	}

	transacoesAsBytes, err := stub.GetState(transacoesKey)
	if err != nil {
		return nil, errors.New("Failed to get transacoes list")
	}

	json.Unmarshal(paymentsAsBytes, &paymentsLista)
	json.Unmarshal(merchantsAsBytes, &merchantLista)
	json.Unmarshal(transacoesAsBytes, &transacoesLista)

	//Lista todos os merchants (debug)
	fmt.Println("Merchants no mercado:")
	for m := range merchantLista {
		fmt.Println("Índice no array: " + strconv.Itoa(m) + " | Razão social: " + merchantLista[m].RazaoSocial + " | ID: " + strconv.Itoa(merchantLista[m].ID))
	}
	fmt.Println("Comprador: " + strconv.Itoa(comprador))

	//executando a transação de cashIn em si,primeiro busca o merchant com o args[0]-comprador
	for i := range merchantLista {
		if merchantLista[i].ID == comprador {
			isPayment = false
			fmt.Println("Cash-in de Payments para Merchant iniciado")
			//o próximo passo é identificar o payment que atenderá a solicitação segundo args[2] -> vendedor
			for x := range paymentsLista {
				if paymentsLista[x].ID == vendedor {

					//Realiza as checagens de quantidade de tokens de cada um deles
					if len(paymentsLista[x].Tokens) < quantTokens && len(paymentsLista[x].Tokens) != 0 {
						return nil, errors.New("Você está tentando realizar um cash in com mais tokens do que o Payments possui. Reveja seus argumentos.")
					}
					if len(paymentsLista[x].Tokens) == 0 {
						return nil, errors.New("Payments não possui tokens!")
					}

					//realiza a transferência de tokens do payment pro merchant
					for y := 0; y < quantTokens; y++ {
						fmt.Println("INÍCIO DO LOOP " + strconv.Itoa(y) + "-------------------------------------------------")

						//recuperando o ultimo token de payments para transferencia => merchant
						lastIndex := len(paymentsLista[x].Tokens) - 1

						fmt.Println("Recuperando último token na posição " + strconv.Itoa(lastIndex) + " para o payment " + strconv.Itoa(x) + " (" + strconv.Itoa(vendedor) + ")")
						merchantToken := len(merchantLista[i].Tokens)
						fmt.Println("Tokens do Merchant antes de adicionar: " + strconv.Itoa(merchantToken))

						lastToken := paymentsLista[x].Tokens[lastIndex]
						merchantLista[i].Tokens = append(merchantLista[i].Tokens, lastToken)
						merchantToken = len(merchantLista[i].Tokens)
						fmt.Println("Tokens do Merchant depois de adicionar: " + strconv.Itoa(merchantToken))

						paymentToken := len(paymentsLista[x].Tokens)
						fmt.Println("Tokens do Payments antes de deletar: " + strconv.Itoa(paymentToken))

						//delete tokens from last from payments(length-1)
						paymentsLista[x].Tokens = append(paymentsLista[x].Tokens[:lastIndex], paymentsLista[x].Tokens[lastIndex+1:]...)
						paymentToken = len(paymentsLista[x].Tokens)
						fmt.Println("Tokens do Payments depois de deletar: " + strconv.Itoa(paymentToken))
					}

					//Armazenando saldo atual de tokens do comprador e vendedor para poder salvar no struct de transações
					transacao.SaldoTsComp = len(merchantLista[i].Tokens)
					transacao.SaldoTsVend = len(paymentsLista[x].Tokens)
					fmt.Println("Saldo atual do comprador: ", transacao.SaldoTsComp)
					fmt.Println("Saldo atual do vendedor: ", transacao.SaldoTsVend)

					merchantAsBytes, _ := json.Marshal(merchantLista[i])
					paymentsAsBytes, _ := json.Marshal(paymentsLista[x])

					//salvando um novo state com o id do merchant como key
					fmt.Println("salvando um novo state com o id do merchant como key: " + merchantLista[i].RazaoSocial)
					err = stub.PutState(strconv.Itoa(merchantLista[i].ID), merchantAsBytes)
					if err != nil {
						return nil, err
					}

					//salvando um novo state com o id do payment como key
					fmt.Println("salvando um novo state com o id do payment como key: " + paymentsLista[x].Descricao)
					err = stub.PutState(strconv.Itoa(paymentsLista[x].ID), paymentsAsBytes)
					if err != nil {
						return nil, err
					}

					paymentsListaAsBytes, _ := json.Marshal(paymentsLista)
					//salvando a nova versao da lista de Payments
					fmt.Println("salvando a nova versao da lista de Payments " + paymentsKey)
					err = stub.PutState(paymentsKey, paymentsListaAsBytes)
					if err != nil {
						return nil, err
					}

					merchantListaAsBytes, _ := json.Marshal(merchantLista)
					//salvando a nova versao da lista de Merchants
					fmt.Println("salvando a nova versao da lista de Merchants " + merchantKey)
					err = stub.PutState(merchantKey, merchantListaAsBytes)
					if err != nil {
						return nil, err
					}
					break //Interrompe o loop
				}
			}
		}
	}
	//fecha o loop de merchantLista

	if isPayment {
		//Neste ponto também precisamos subtrair do saldo de payments o equivalente aos tokens comprados
		var paymentsFound = false
		for x := range paymentsLista {
			if paymentsLista[x].ID == comprador {
				paymentsFound = true

				fmt.Println("Cash in de Payment (disponibilizando Tokens no mercado) para " + paymentsLista[x].Descricao)

				//recupera o list de tokens no state antes inicializado(initdemo)
				tokensAsBytes, err := stub.GetState(tokensKey)
				if err != nil {
					return nil, errors.New("Failed to get tokensAsBytes list")
				}

				json.Unmarshal(tokensAsBytes, &tokensLista)

				//criando tokens pela quantidade na requisicao de payments
				for q := 0; q < quantTokens; q++ {
					fmt.Println("Criando token de número " + strconv.Itoa(q+1) + " e adicionando-o ao payments de nome " + paymentsLista[x].Descricao)
					initTokens := generateToken(paymentsLista[x].ID)
					paymentsLista[x].Tokens = append(paymentsLista[x].Tokens, initTokens)
				}

				//Armazenando saldo atual de tokens do comprador e vendedor para poder salvar no struct de transações
				transacao.SaldoTsComp = len(paymentsLista[x].Tokens)
				transacao.SaldoTsVend = len(paymentsLista[x].Tokens)
				fmt.Println("Saldo atual do comprador (tokens criados): ", transacao.SaldoTsComp)
				fmt.Println("Saldo atual do vendedor (tokens criados, é o mesmo que o de cima): ", transacao.SaldoTsVend)

				//atualiza o estado do payment(nova quantidade de tokens adquirida)
				paymentsAsBytes, _ := json.Marshal(paymentsLista[x])
				err = stub.PutState(strconv.Itoa(paymentsLista[x].ID), paymentsAsBytes)
				if err != nil {
					return nil, err
				}

			}
		}
		if !paymentsFound {
			return nil, errors.New("Erro ao buscar Payments na lista do ledger. O Payments não foi encontrado.")
		}

		//Atualizando state da lista de payments
		paymentsListaAsBytes, _ := json.Marshal(paymentsLista)

		//salvando um novo state com o id do payment como key
		fmt.Println("salvando um novo state da lista de payments no ledger")
		err = stub.PutState(paymentsKey, paymentsListaAsBytes)
		if err != nil {
			return nil, err
		}

		tokensASBytes, _ := json.Marshal(tokensLista)
		err = stub.PutState(tokensKey, tokensASBytes)
		if err != nil {
			return nil, err
		}

	}

	fmt.Println("Salvando transações")
	//salvando a lista de Transacoes
	transacao.IdTransac = idTransaction // atribui um Id à transação
	transacao.Cotacao = cotacao
	transacao.IdComprador = comprador
	transacao.IdVendedor = vendedor
	transacao.QtdTokens = quantTokens
	//precisamos utilizar o retorno da msToTime como nossos id's de transacao (args[4] - timestamp vindo da app,vira parametro pra msToTime)
	transacao.Timestamp = args[4]

	transacoesLista = append(transacoesLista, transacao)
	transacoesListaAsBytes, _ := json.Marshal(transacoesLista)
	//salvando o state de transação
	fmt.Println("salvando o state de transação: " + transacoesKey)
	err = stub.PutState(transacoesKey, transacoesListaAsBytes)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// ============================================================================================================================
// cashOut - devolução de tokens
// 0 = ID do merchant
// 1 = Quantidade de tokens a ser transferida
// 2 = ID do payment
// 3 = Cotacao
// 4 = Timestamp/ID da transação
// 5 = Valor atual em reais do comprador
// 6 = Valor atual em reais do vendedor
// 7 = ID da Transação
// ============================================================================================================================
func (t *SimpleChaincode) cashOut(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 8 {
		return nil, errors.New("Incorrect number of arguments. Expecting 8")
	}

	var found = false
	var merchant = Merchant{}
	var payment = Payments{}
	merchant.ID, err = strconv.Atoi(args[0])
	payment.ID, err = strconv.Atoi(args[2])

	quantTokens, err = strconv.Atoi(args[1])
	if err != nil {
		return nil, errors.New("Failed to convert args[1](int)")
	}

	cotacao, err := strconv.ParseFloat(args[3], 64)
	if err != nil {
		return nil, errors.New("Failed to get cotacao")
	}

	/*Recuperando os states de merchants,payments e transacoes - serao atualizados adiante*/
	merchantsAsBytes, err := stub.GetState(merchantKey)
	if err != nil {
		return nil, errors.New("Failed to get merchant list")
	}

	paymentsAsBytes, err := stub.GetState(paymentsKey)
	if err != nil {
		return nil, errors.New("Failed to get merchant list")
	}

	transacoesAsBytes, err := stub.GetState(transacoesKey)
	if err != nil {
		return nil, errors.New("Failed to get transacoes list")
	}

	transacao.SaldoRsComp, err = strconv.ParseFloat(args[5], 64)
	if err != nil {
		return nil, errors.New("Failed to get saldoRsComp")
	}

	transacao.SaldoRsVend, err = strconv.ParseFloat(args[6], 64)
	if err != nil {
		return nil, errors.New("Failed to get saldoRsVend")
	}

	idTransaction, err := strconv.Atoi(args[7])
	if err != nil {
		return nil, errors.New("Failed to get idTransaction")
	}

	json.Unmarshal(paymentsAsBytes, &paymentsLista)
	json.Unmarshal(merchantsAsBytes, &merchantLista)
	json.Unmarshal(transacoesAsBytes, &transacoesLista)

	//executando a transação de cashOut em si,primeiro busca o merchant com o args[0]-comprador
	for i := range merchantLista {
		if merchantLista[i].ID == merchant.ID {
			found = true
			fmt.Println("Cash out de Merchant: " + strconv.Itoa(merchant.ID))

			//o próximo passo é identificar o payment que atenderá a solicitação segundo args[2] -> vendedor
			for x := range paymentsLista {
				if paymentsLista[x].ID == payment.ID {

					//Realiza as checagens de quantidade de tokens de cada um deles NECESSÁRIO REVER
					if len(merchantLista[i].Tokens) < quantTokens && len(merchantLista[i].Tokens) != 0 {
						return nil, errors.New("Você está tentando realizar um cash out com mais tokens do que o Merchant possui. Reveja seus argumentos.")
					}
					if len(merchantLista[i].Tokens) == 0 {
						return nil, errors.New("Merchant não possui tokens!")
					}

					//realiza a transferência de tokens do merchant pro payments
					for y := 0; y < quantTokens; y++ {

						//recuperando o ultimo token de merchant para transferencia => payments
						arrayAtual := len(merchantLista[i].Tokens) - 1
						fmt.Println("Recuperando último token de valor " + strconv.Itoa(arrayAtual) + " para o merchant " + strconv.Itoa(i))

						lastToken := merchantLista[i].Tokens[len(merchantLista[i].Tokens)-1]
						lastIndex := len(merchantLista[i].Tokens) - 1
						paymentsLista[x].Tokens = append(paymentsLista[x].Tokens, lastToken)

						//delete tokens from last from merchant(length-1)
						merchantLista[i].Tokens = append(merchantLista[i].Tokens[:lastIndex], merchantLista[i].Tokens[lastIndex+1:]...)
					}

					//Armazenando saldo atual de tokens do comprador e vendedor para poder salvar no struct de transações
					transacao.SaldoTsComp = len(paymentsLista[x].Tokens)
					transacao.SaldoTsVend = len(merchantLista[i].Tokens)
					fmt.Println("Saldo atual do comprador: ", transacao.SaldoTsComp)
					fmt.Println("Saldo atual do vendedor: ", transacao.SaldoTsVend)

					merchantAsBytes, _ := json.Marshal(merchantLista[i])
					paymentsAsBytes, _ := json.Marshal(paymentsLista[x])

					//salvando um novo state com o id do merchant como key
					fmt.Println("salvando um novo state com o id do merchant como key: " + merchantLista[i].RazaoSocial)
					err = stub.PutState(strconv.Itoa(merchantLista[i].ID), merchantAsBytes)
					if err != nil {
						return nil, err
					}

					//salvando um novo state com o id do payment como key
					fmt.Println("salvando um novo state com o id do payment como key: " + paymentsLista[x].Descricao)
					err = stub.PutState(strconv.Itoa(paymentsLista[x].ID), paymentsAsBytes)
					if err != nil {
						return nil, err
					}

					paymentsListaAsBytes, _ := json.Marshal(paymentsLista)
					//salvando a nova versao da lista de Payments
					fmt.Println("salvando a nova versao da lista de Payments " + paymentsKey)
					err = stub.PutState(paymentsKey, paymentsListaAsBytes)
					if err != nil {
						return nil, err
					}

					merchantListaAsBytes, _ := json.Marshal(merchantLista)
					//salvando a nova versao da lista de Merchants
					fmt.Println("salvando a nova versao da lista de Merchants " + merchantKey)
					err = stub.PutState(merchantKey, merchantListaAsBytes)
					if err != nil {
						return nil, err
					}

					//salvando a lista de Transacoes
					transacao.IdTransac = idTransaction // atribui um Id à transação
					transacao.Cotacao = cotacao
					transacao.IdComprador = payment.ID
					transacao.IdVendedor = merchant.ID
					transacao.QtdTokens = quantTokens
					//precisamos utilizar o retorno da msToTime como nossos id's de transacao (args[5] - timestamp vindo da app,vira parametro pra msToTime)
					transacao.Timestamp = args[4]

					transacoesLista = append(transacoesLista, transacao)
					transacoesListaAsBytes, _ := json.Marshal(transacoesLista)
					//salvando o state de transação
					fmt.Println("salvando o state de transação(checkout): " + transacoesKey)
					err = stub.PutState(transacoesKey, transacoesListaAsBytes)
					if err != nil {
						return nil, err
					}

					break
				}
			}
		}
	}

	if found == false {
		return nil, errors.New("Não foi possível encontrar o Merchant: " + strconv.Itoa(merchant.ID))
	}

	return nil, nil
}

// ============================================================================================================================
// transferTokens - Transferência de tokens entre merchants. Construída em cima da função de cashIn
// 0 = ID do receptor
// 1 = Quantidade de tokens a ser transferida
// 2 = ID do fornecedor
// 3 = Timestamp/ID da transação
// 4 = Valor atual em reais do comprador
// 5 = Valor atual em reais do vendedor
// 6 = ID da Transação
// ============================================================================================================================
func (t *SimpleChaincode) transferTokens(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	if len(args) != 7 {
		return nil, errors.New("Incorrect number of arguments. Expecting 7")
	}
	var quantTokens int
	var merchantLista []Merchant
	var transacoesLista []Transacao
	var transacao Transacao
	var err error

	//Converte todos os parâmetros (string) em inteiros
	receptor, err := strconv.Atoi(args[0])
	if err != nil {
		return nil, errors.New("Não foi possível obter o receptor dos tokens")
	}
	quantTokens, err = strconv.Atoi(args[1])
	if err != nil {
		return nil, errors.New("Não foi possível obter a quantidade de tokens a ser transferida")
	}
	fornecedor, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, errors.New("Não foi possível obter o fornecedor dos tokens")
	}
	transacao.SaldoRsComp, err = strconv.ParseFloat(args[4], 64)
	if err != nil {
		return nil, errors.New("Failed to get saldoRsComp")
	}
	transacao.SaldoRsVend, err = strconv.ParseFloat(args[5], 64)
	if err != nil {
		return nil, errors.New("Failed to get saldoRsVend")
	}

	//Busca as listas de merchants e de transações no ledger
	merchantsAsBytes, err := stub.GetState(merchantKey)
	if err != nil {
		return nil, errors.New("Não foi possível obter a lista de merchants do ledger")
	}
	transacoesAsBytes, err := stub.GetState(transacoesKey)
	if err != nil {
		return nil, errors.New("Não foi possível obter a lista de transações do ledger")
	}

	idTransaction, err := strconv.Atoi(args[6])
	if err != nil {
		return nil, errors.New("Failed to get idTransaction")
	}

	//Converte todos os dados de byte para um formato que possa ser lido pelo Go
	json.Unmarshal(merchantsAsBytes, &merchantLista)
	json.Unmarshal(transacoesAsBytes, &transacoesLista)

	//Itera entre todos os merchants, buscando o ID do receptor
	for i := range merchantLista {
		if merchantLista[i].ID == receptor {
			//Itera entre todos os merchants, buscando o ID do fornecedor
			for x := range merchantLista {
				if merchantLista[x].ID == fornecedor {

					//Realiza as checagens de quantidade de tokens de cada um deles NECESSÁRIO REVER
					if len(merchantLista[x].Tokens) < quantTokens && len(merchantLista[x].Tokens) != 0 {
						return nil, errors.New("Você está tentando realizar uma transferência com mais tokens do que o outro Merchant possui. Reveja seus argumentos.")
					}
					if len(merchantLista[x].Tokens) == 0 {
						return nil, errors.New("O outro merchant não possui tokens!")
					}

					fmt.Println("Realizando a transferência de tokens de " + strconv.Itoa(fornecedor) + " para " + strconv.Itoa(receptor) + "...")
					//Realiza a transferência de tokens
					for y := 0; y < quantTokens; y++ {

						//Recuperando o último token da conta do fornecedor para transferir para o receptor
						lastToken := merchantLista[x].Tokens[len(merchantLista[x].Tokens)-1]
						lastIndex := len(merchantLista[x].Tokens) - 1
						merchantLista[i].Tokens = append(merchantLista[i].Tokens, lastToken)

						//Deleta os tokens do fornecedor a partir do último da lista
						merchantLista[x].Tokens = append(merchantLista[x].Tokens[:lastIndex], merchantLista[x].Tokens[lastIndex+1:]...)
					}

					//Armazenando saldo atual de tokens do comprador e vendedor para poder salvar no struct de transações
					transacao.SaldoTsComp = len(merchantLista[i].Tokens)
					transacao.SaldoTsVend = len(merchantLista[x].Tokens)
					fmt.Println("Saldo atual do comprador: ", transacao.SaldoTsComp)
					fmt.Println("Saldo atual do vendedor: ", transacao.SaldoTsVend)

					receptorAsBytes, _ := json.Marshal(merchantLista[i])
					fornecedorAsBytes, _ := json.Marshal(merchantLista[x])

					//Salva um novo state no ledger com os dados novos do receptor
					fmt.Println("Salvando um novo state no ledger para " + merchantLista[i].RazaoSocial + "...")
					err = stub.PutState(strconv.Itoa(merchantLista[i].ID), receptorAsBytes)
					if err != nil {
						return nil, err
					}

					//Salva um novo state no ledger com os dados novos do fornecedor
					fmt.Println("Salvando um novo state no ledger para " + merchantLista[x].RazaoSocial + "...")
					err = stub.PutState(strconv.Itoa(merchantLista[x].ID), fornecedorAsBytes)
					if err != nil {
						return nil, err
					}

					merchantListaAsBytes, _ := json.Marshal(merchantLista)
					//Salva a nova versão da lista de Merchants no ledger
					fmt.Println("Salvando a nova versão da lista de merchants: " + merchantKey + "...")
					err = stub.PutState(merchantKey, merchantListaAsBytes)
					if err != nil {
						return nil, err
					}

					break
				}
			}
		}
	}

	//Adiciona as transações realizadas ao ledger
	transacao.IdTransac = idTransaction // atribui um Id à transação
	transacao.IdComprador = receptor
	transacao.IdVendedor = fornecedor
	transacao.QtdTokens = quantTokens
	transacao.Timestamp = args[3] //Timestamp

	transacoesLista = append(transacoesLista, transacao)
	transacoesListaAsBytes, _ := json.Marshal(transacoesLista)
	//salvando o state de transação
	fmt.Println("Salvando o state de transação " + transacoesKey + "...")
	err = stub.PutState(transacoesKey, transacoesListaAsBytes)
	if err != nil {
		return nil, err
	}

	fmt.Println("Transferência concluída.")

	return nil, nil
}

// ============================================================================================================================
// generateToken - Cria tokens
//
// owner = Payments provedor dos token
//
// PS.: Essa função retorna apenas um struct de Token! Para adicionar ao ledger, use um append para um array.
// ============================================================================================================================
func generateToken(owner int) Token {
	var token = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	var initTokens = Token{}

	//Cria um hash para o token
	b := make([]rune, 10)
	for i := range b {
		b[i] = token[rand.Intn(len(token))]
	}

	//Assigna os parâmetros da função
	initTokens.Owner = owner
	initTokens.Value = string(b)

	fmt.Println("Novo token criado pelo payments de ID " + strconv.Itoa(owner))

	return initTokens
}

// ============================================================================================================================
// generateMerchant - Cria merchants (para a initDemo)
//
// stub = shim
// ID = ID do merchant
// razaoSocial = nome da empresa
// CNPJ = CNPJ (esse é straightforward vai)
// numTokens = Número de tokens a serem criados
// tokenOwner = ID do payments dono dos tokens
//
// PS.: Essa função retorna apenas um struct de Merchant! Para adicionar ao ledger, use um append para um array.
// ============================================================================================================================
func generateMerchant(stub shim.ChaincodeStubInterface, ID string, razaoSocial string, CNPJ string, numTokens string, tokenOwner string) Merchant {
	//Declaração de variáveis
	var initMerchants = Merchant{}
	var err error
	var intTokens int
	var ownerToken int

	//Assigna os parâmetros da função ao struct temporário initMerchants
	initMerchants.ID, err = strconv.Atoi(ID)
	if err != nil {
		msg := "initMerchants.IdMerchant error: " + ID
		fmt.Println(msg)
		os.Exit(1)
	}
	initMerchants.RazaoSocial = razaoSocial
	initMerchants.Cnpj = CNPJ

	//Converte o parâmetro numTokens
	intTokens, err = strconv.Atoi(numTokens)
	if err != nil {
		msg := "intTokens error: " + numTokens
		fmt.Println(msg)
		os.Exit(1)
	}

	//Converte o parâmetro tokenOwner
	ownerToken, err = strconv.Atoi(tokenOwner)
	if err != nil {
		msg := "ownerToken error: " + tokenOwner
		fmt.Println(msg)
		os.Exit(1)
	}

	//Cria o número de tokens definido nos parâmetros da função
	for i := 0; i < intTokens; i++ {
		structTokens := generateToken(ownerToken)
		initMerchants.Tokens = append(initMerchants.Tokens, structTokens)
	}

	//Adiciona os tokens na chave global de tokens no ledger
	tokensBytesToWrite, err := json.Marshal(&initMerchants.Tokens)
	if err != nil {
		fmt.Println("Error marshalling keys")
		os.Exit(1)
	}
	err = stub.PutState(tokensKey, tokensBytesToWrite)
	if err != nil {
		fmt.Println("Error writting keys paymentsBytesToWrite")
		os.Exit(1)
	}

	fmt.Println("Merchant de nome " + razaoSocial + " criado com sucesso!")

	return initMerchants
}

// ============================================================================================================================
// deleteState - Deleta algum peer ou hash do ledger [UTILIZADO PARA RESET DO LEDGER - MUITO CUIDADO AO FAZER USO!!!!!]
//
// 0 = key ou ID a ser deletado
// ===========================================================================================================================
func (t *SimpleChaincode) deleteState(stub shim.ChaincodeStubInterface, args []string) ([]byte, error) {
	err := stub.DelState(args[0])
	if err != nil {
		fmt.Println("Não foi possível deletar o peer desejado do ledger")
		return nil, errors.New("Não foi possível deletar o peer desejado do ledger")
	}

	fmt.Println("Peer deletado com sucesso")

	return nil, nil
}

// ============================================================================================================================
// resetAll - Reinicia o ledger, limpando-o [UTILIZADO PARA RESET DO LEDGER - MUITO CUIDADO AO FAZER USO!!!!!]
// ===========================================================================================================================
func (t *SimpleChaincode) resetAll(stub shim.ChaincodeStubInterface) ([]byte, error) {
	var err error

	//Deleta merchants
	err = stub.DelState(merchantKey)
	if err != nil {
		return nil, errors.New("Não foi possível deletar os merchants do ledger")
	}
	fmt.Println("Lista de merchants deletada")
	//Deleta payments
	err = stub.DelState(paymentsKey)
	if err != nil {
		return nil, errors.New("Não foi possível deletar os payments do ledger")
	}
	fmt.Println("Lista de payments deletada")
	//Deleta tokens
	err = stub.DelState(tokensKey)
	if err != nil {
		return nil, errors.New("Não foi possível deletar os tokens do ledger")
	}
	fmt.Println("Lista de tokens deletada")
	//Deleta transações
	err = stub.DelState(transacoesKey)
	if err != nil {
		return nil, errors.New("Não foi possível deletar as transações do ledger")
	}
	fmt.Println("Lista de transações deletada")
	//Deleta opentrades (?)
	err = stub.DelState(openTradesStr)
	if err != nil {
		return nil, errors.New("Não foi possível deletar as opentrades do ledger")
	}

	//Deleta tudo o que estiver armazenado do 1 ao 1000 para garantir que tudo foi deletado MESMO
	for i := 0; i < 1000; i++ {
		err = stub.DelState(strconv.Itoa(i))
		if err != nil {
			return nil, errors.New("Não foi possível deletar o state do peer " + strconv.Itoa(i+1))
		}
		fmt.Println("Peer de número " + strconv.Itoa(i+1) + " deletado")
	}

	fmt.Println("Ambiente limpo com sucesso!")

	return nil, nil
}
