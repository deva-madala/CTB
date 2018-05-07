node enrollAdmin.js org1
node enrollAdmin.js org2
node enrollAdmin.js org3
node enrollAdmin.js org4

node registerUser.js org1 userCA1
node registerUser.js org2 userCA2
node registerUser.js org3 userCA3
node registerUser.js org4 userChrome

node invoke.js userCA1 certificates/CA1/ashoka/ashoka.pem certificates/CA1/CA1.pem