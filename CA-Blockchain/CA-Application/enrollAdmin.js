'use strict';

var Fabric_Client = require('fabric-client');
var Fabric_CA_Client = require('fabric-ca-client');
var path = require('path');
var util = require('util');
var os = require('os');

//
var fabric_client = new Fabric_Client();
var fabric_ca_client = null;
var admin_user = null;
var member_user = null;
var store_path = path.join(__dirname, 'hfc-key-store');
console.log(' Store path:' + store_path);

var args = process.argv.slice(2);
var org = args[0];
var admin = null;
var addr = null;
var caOrg = null;
var orgMSP = null;

if (org === 'org1') {
    admin = 'adminOrg1';
    addr = 'http://localhost:7054';
    caOrg = 'ca1.example.com';
    orgMSP = 'Org1MSP';
}

if (org === 'org2') {
    admin = 'adminOrg2';
    addr = 'http://localhost:8054';
    caOrg = 'ca2.example.com';
    orgMSP = 'Org2MSP';
}

if (org === 'org3') {
    admin = 'adminOrg3';
    addr = 'http://localhost:9054';
    caOrg = 'ca3.example.com';
    orgMSP = 'Org3MSP';
}

if (org === 'org4') {
    admin = 'adminOrg4';
    addr = 'http://localhost:10054';
    caOrg = 'ca4.example.com';
    orgMSP = 'Org4MSP';
}

console.log(org + ' ' + admin + ' ' + addr + ' ' + caOrg + ' ' + orgMSP);

Fabric_Client.newDefaultKeyValueStore({

    path: store_path

}).then((state_store) => {

    fabric_client.setStateStore(state_store);
    var crypto_suite = Fabric_Client.newCryptoSuite();
    var crypto_store = Fabric_Client.newCryptoKeyStore({path: store_path});
    crypto_suite.setCryptoKeyStore(crypto_store);
    fabric_client.setCryptoSuite(crypto_suite);
    var tlsOptions = {
        trustedRoots: [],
        verify: false
    };
    fabric_ca_client = new Fabric_CA_Client(addr, tlsOptions, caOrg, crypto_suite);

    return fabric_client.getUserContext(admin, true); // first check to see if the admin is already enrolled

}).then((user_from_store) => {
    if (user_from_store && user_from_store.isEnrolled()) {
        console.log('Successfully loaded admin from persistence');
        admin_user = user_from_store;
        return null;
    } else {
        // need to enroll it with CA server
        console.log("Enrolling");
        return fabric_ca_client.enroll({
            enrollmentID: 'admin',
            enrollmentSecret: 'adminpw'

        }).then((enrollment) => {

            console.log('Successfully enrolled admin user ' + admin);

            return fabric_client.createUser(
                {
                    username: admin,
                    mspid: orgMSP,
                    cryptoContent: {privateKeyPEM: enrollment.key.toBytes(), signedCertPEM: enrollment.certificate}
                });

        }).then((user) => {

            admin_user = user;

            return fabric_client.setUserContext(admin_user);

        }).catch((err) => {

            console.error('Failed to enroll and persist admin. Error: ' + err.stack ? err.stack : err);
            throw new Error('Failed to enroll admin');

        });
    }

}).then(() => {

    console.log('Assigned the admin user to the fabric client ::' + admin_user.toString());

}).catch((err) => {

    console.error('Failed to enroll admin: ' + err);
});
