'use strict';
const shim = require('fabric-shim');
const util = require('util');
const x509 = require('x509');
const exec = require('child_process').exec;


let Chaincode = class {

    async Init(stub) {
        console.info('=========== Instantiated ca-blockchain chaincode ===========');
        return shim.success();
    }

    // The Invoke method is called as a result of an application request to run the Smart Contract
    // The calling application program has also specified the particular smart contract function to be called, with arguments
    async Invoke(stub) {
        let ret = stub.getFunctionAndParameters();
        console.info(ret);

        let method = this[ret.fcn];
        if (!method) {
            console.error('no function of name:' + ret.fcn + ' found');
            throw new Error('Received unknown function ' + ret.fcn + ' invocation');
        }
        try {
            let payload = await method(stub, ret.params);
            return shim.success(payload);
        } catch (err) {
            console.log(err);
            return shim.error(err);
        }
    }


    async addCertificate(stub, args) {
        console.info('============= START : ADD CERTIFICATE ===========');
        if (args.length !== 2) {
            throw new Error('Incorrect number of arguments. Expecting 2: Certificate, Intermediate-certificate');
        }

        var certPath = args[0];
        var intermediateCertPath = args[1];

        var command = "openssl verify -untrusted " + intermediateCertPath + " " + certPath;

        exec(command, function (err, stdout, stderr) {
            if (err) {
                console.info(stderr);
                console.info(stdout);
                console.info(err);
                console.info("Invalid Certificate");
                throw new Error("Certificate Verification failed");
            } else {
                console.info("Valid Certificate");
            }
        });

        var certDetails = x509.parseCert(certPath);
        var subjectName = certDetails.subject.commonName;

        var certificate = {
            docType: 'certificate',
            subjectName: subjectName,
            certDetails: certDetails,
        };

        await stub.putState(subjectName, Buffer.from(JSON.stringify(certificate)));
        console.info('============= END : ADD CERTIFICATE ===========');
    }

    async queryCertificate(stub, args) {
        if (args.length !== 1) {
            throw new Error('Incorrect number of arguments. Expecting Certificate subject name ex: www.example.com');
        }
        let subjectName = args[0];

        let certificateAsBytes = await stub.getState(subjectName); //get the certificate from chaincode state
        if (!certificateAsBytes || certificateAsBytes.toString().length <= 0) {
            throw new Error(subjectName + ' does not exist: ');
        }
        console.log(certificateAsBytes.toString());
        return certificateAsBytes;
    }
};

shim.start(new Chaincode());
