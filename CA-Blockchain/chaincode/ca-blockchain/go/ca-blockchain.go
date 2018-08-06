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
	SubjectName  string `json:"subjectName"`
	CertString   string `json:"certString"`
	RevokeStatus string `json:"revokeStatus"`
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
	} else if function == "revokeCertificate" {
		return s.revokeCertificate(APIstub, args)
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

	// buffer is a JSON array containing historic values for the subject
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
		//as-is (as the Value itself a JSON)
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

func verifySignatureOnMessage(sigString string, message string, rsaPubKey *rsa.PublicKey) bool {
	messageBytes := []byte(message)
	hashed := sha256.Sum256(messageBytes)
	signature, _ := hex.DecodeString(sigString)
	err := rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, hashed[:], signature)
	if err != nil {
		return false
	}
	return true
}

func (s *SmartContract) revokeCertificate(APIstub shim.ChaincodeStubInterface, args []string) sc.Response {

	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3")
	}

	certString := args[0]
	caCertString := args[1]
	caSigOnCert := args[2]

	certPEM := []byte(certString)
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return shim.Error("failed to parse certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return shim.Error("failed to parse certificate: " + err.Error())
	}

	subjectName := cert.Subject.CommonName

	certificateAsBytes, _ := APIstub.GetState(subjectName)
	if certificateAsBytes == nil {
		return shim.Error("Certificate is not present in the ledger!")
	}

	certEntryInLedger := Certificate{}
	err = json.Unmarshal(certificateAsBytes, &certEntryInLedger)
	certInLedgerString := certEntryInLedger.CertString

	if certInLedgerString != certString {
		return shim.Error("Certificate mismatch: certificate for the domain present in the ledger is different")
	}

	certInLedgerPEM := []byte(certInLedgerString)
	certInLedgerBlock, _ := pem.Decode(certInLedgerPEM)
	if certInLedgerBlock == nil {
		return shim.Error("failed to parse certificate in Ledger PEM")
	}
	certInLedger, err := x509.ParseCertificate(certInLedgerBlock.Bytes)
	if err != nil {
		return shim.Error("failed to parse certificate: " + err.Error())
	}

	caCertPEM := []byte(caCertString)
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(caCertPEM)
	if !ok {
		return shim.Error("failed to parse CA certificate")
	}
	opts := x509.VerifyOptions{
		DNSName: subjectName,
		Roots:   roots,
	}
	if _, err := certInLedger.Verify(opts); err != nil {
		return shim.Error("failed to verify certificate using the given CA certificate: " + err.Error())
	}

	caBlock, _ := pem.Decode(caCertPEM)
	if caBlock == nil {
		return shim.Error("failed to parse CA certificate PEM")
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return shim.Error("failed to parse CA certificate: " + err.Error())
	}
	caPubKey := caCert.PublicKey.(*rsa.PublicKey)
	if !verifySignatureOnMessage(caSigOnCert, caCertString, caPubKey) {
		return shim.Error("failed to verify the signature!")
	}

	if certEntryInLedger.RevokeStatus == "revoked" {
		return shim.Error("Certificate is already revoked")
	}

	//all the conditions passed and certificate can be revoked.
	certEntryInLedger.RevokeStatus = "revoked"
	certAsBytes, _ := json.Marshal(certEntryInLedger)
	err = APIstub.PutState(subjectName, certAsBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
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

		// certificate is present in the ledger

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

		// get the old public key
		oldPublicKey := oldCert.PublicKey.(*rsa.PublicKey)
		oldCertExpiry := oldCert.NotAfter
		currentTime := time.Now()

		oldCertGraceExpiry := oldCertExpiry
		oldCertGraceExpiry = oldCertGraceExpiry.Add(time.Duration(90*24) * time.Hour)

		revokeStatus := oldCertificate.RevokeStatus

		// check if cert is active and signature is needed
		if currentTime.After(oldCertGraceExpiry) {

			// cert is not active - hence, no signature needed
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

			// cert is active

			if revokeStatus == "notRevoked" {

				// signature needed

				if sigString == "" {
					return shim.Error("Verification failed: signature not provided.")
				}

				//update the certificate after checking signature with old public key

				fmt.Println(oldPublicKey)
				isValidSign := verifySignatureOnMessage(sigString, certString, oldPublicKey)
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

			} else if revokeStatus == "revoked" {

				// since the last cert is revoked, check if idle period has passed

				resultsIterator, err := APIstub.GetHistoryForKey(subjectName)
				if err != nil {
					return shim.Error(err.Error())
				}
				defer resultsIterator.Close()

				finalTimestamp := ""
				for resultsIterator.HasNext() {
					response, err := resultsIterator.Next()
					if err != nil {
						return shim.Error(err.Error())
					}
					finalTimestamp = time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String()
				}
				layout := "2006-01-02 15:04:05 +0000 UTC"
				timeStampWithIdlePeriod, err := time.Parse(layout, finalTimestamp)
				if err != nil {
					return shim.Error("Error parsing timestamp " + err.Error())
				}
				timeStampWithIdlePeriod = timeStampWithIdlePeriod.Add(time.Duration(48) * time.Hour)

				if currentTime.After(timeStampWithIdlePeriod) {

					// idle period has passed. New certificate can be issued without checking signature

					oldCertificate.CertString = certString
					oldCertificate.RevokeStatus = "notRevoked"
					oldCertificateAsBytes, _ := json.Marshal(oldCertificate)
					err = APIstub.PutState(subjectName, oldCertificateAsBytes)
					if err != nil {
						return shim.Error(err.Error())
					}
					return shim.Success(nil)

				} else {

					// not valid - idle period has not passed - cannot issue a new certificate
					return shim.Error("Idle period for the last revoked certificate has not passed!")

				}

			}

		}

	}

	// executes only if certificate is not present in the ledger - new entry
	var certificate = Certificate{SubjectName: subjectName, CertString: certString, RevokeStatus: "notRevoked"}

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
