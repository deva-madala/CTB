package main

import (
	"encoding/json"
	"fmt"
	"crypto/x509"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	sc "github.com/hyperledger/fabric/protos/peer"
	"encoding/pem"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"crypto"
	"bytes"
	"strconv"
	"time"
)

// Define the Smart Contract structure
type SmartContract struct {
}

// Define the certificate structure, with 2 properties.  Structure tags are used by encoding/json library
type Certificate struct {
	SubjectName string `json:"subjectName"`
	CertString  string `json:"certString"`
}

func (s *SmartContract) Init(APIstub shim.ChaincodeStubInterface) sc.Response {
	return shim.Success(nil)
}

func (s *SmartContract) Invoke(APIstub shim.ChaincodeStubInterface) sc.Response {
	function, args := APIstub.GetFunctionAndParameters()
	if function == "queryCertificate" {
		return s.queryCertificate(APIstub, args)
	} else if function == "queryCertificateHistory" {
		return s.queryCertificateHistory(APIstub, args)
	} else if function == "addCertificate" {
		return s.addCertificate(APIstub, args)
	}
	return shim.Error("Invalid Smart Contract function name.")
}

func (s *SmartContract) queryCertificate(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}
	certificateAsBytes, _ := APIstub.GetState(args[0])
	if certificateAsBytes == nil {
		return shim.Error("Entry not available")
	}
	return shim.Success(certificateAsBytes)
}

func (s *SmartContract) queryCertificateHistory(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) < 1 {
		return shim.Error("Incorrect number of arguments. Expecting 1")
	}

	subjectName := args[0]

	fmt.Printf("- start getHistoryForSubject: %s\n", subjectName)

	resultsIterator, err := APIstub.GetHistoryForKey(subjectName)
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
		//as-is (as the Value itself a JSON marble)
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

	fmt.Printf("- getHistoryForSubject returning:\n%s\n", buffer.String())

	return shim.Success(buffer.Bytes())
}

func verifySignature(sigString string, rsaPubKey *rsa.PublicKey) bool {
	message := []byte("This is a genuine request!")
	hashed := sha256.Sum256(message)
	signature, _ := hex.DecodeString(sigString)
	err := rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, hashed[:], signature)
	if err != nil {
		return false
	}
	return true
}

func (s *SmartContract) addCertificate(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	certString := args[0]
	intermediateCertString := args[1]
	sigString := args[2]

	certPEM := []byte(certString)
	intermediateCertPEM := []byte(intermediateCertString)

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return shim.Error("failed to parse certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return shim.Error("failed to parse certificate: " + err.Error())
	}
	subjectName := cert.Subject.CommonName
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(intermediateCertPEM)
	if !ok {
		return shim.Error("failed to parse root certificate")
	}
	opts := x509.VerifyOptions{
		DNSName: subjectName,
		Roots:   roots,
	}
	if _, err := cert.Verify(opts); err != nil {
		return shim.Error("failed to verify certificate: " + err.Error())
	}

	certAsBytes, err := APIstub.GetState(subjectName)

	if err != nil {

		return shim.Error("Failed to check ledger for certificate: " + err.Error())

	} else if certAsBytes != nil {

		oldCertificate := Certificate{}
		err = json.Unmarshal(certAsBytes, &oldCertificate)
		oldCertString := oldCertificate.CertString
		oldCertPEM := []byte(oldCertString)
		oldBlock, _ := pem.Decode(oldCertPEM)
		if oldBlock == nil {
			return shim.Error("failed to parse old certificate PEM")
		}
		oldCert, err := x509.ParseCertificate(oldBlock.Bytes)
		if err != nil {
			return shim.Error("failed to parse old certificate: " + err.Error())
		}

		oldPublicKey := oldCert.PublicKey.(*rsa.PublicKey)
		oldCertExpiry := oldCert.NotAfter
		currentTime := time.Now()

		oldCertGraceExpiry := oldCertExpiry
		oldCertGraceExpiry.Add(90)

		if currentTime.After(oldCertGraceExpiry) {

			// no signature needed
			fmt.Println(oldCertGraceExpiry)
			fmt.Println("Signature not needed")

			oldCertificate.CertString = certString
			oldCertificateAsBytes, _ := json.Marshal(oldCertificate)
			err = APIstub.PutState(subjectName, oldCertificateAsBytes)
			if err != nil {
				return shim.Error(err.Error())
			}
			return shim.Success(nil)

		} else {

			// signature needed

			if sigString == "" {
				return shim.Error("Verification failed: signature not provided.")
			}

			//update the certificate after checking signature with old public key

			fmt.Println(oldPublicKey)
			isValidSign := verifySignature(sigString, oldPublicKey)
			if isValidSign {
				oldCertificate.CertString = certString
				oldCertificateAsBytes, _ := json.Marshal(oldCertificate)
				err = APIstub.PutState(subjectName, oldCertificateAsBytes)
				if err != nil {
					return shim.Error(err.Error())
				}
				return shim.Success(nil)
			} else {
				return shim.Error("Signature verification using old public key failed!")
			}

		}

	}

	var certificate = Certificate{SubjectName: subjectName, CertString: certString}

	certificateAsBytes, _ := json.Marshal(certificate)
	err = APIstub.PutState(subjectName, certificateAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func main() {

	// Create a new Smart Contract
	err := shim.Start(new(SmartContract))
	if err != nil {
		fmt.Printf("Error creating new Smart Contract: %s", err)
	}
}
