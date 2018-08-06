export FABRIC_CFG_PATH=$PWD
export CHANNEL_NAME=mychannel


echo "################################### Starting network ###################################"

CHANNEL_NAME=$CHANNEL_NAME TIMEOUT=3600 docker-compose -f docker-compose-cli.yaml -f docker-compose-couch.yaml up -d

sleep 10

echo "################################### Creating channel and Joining peers to channel ###################################"

docker exec cli peer channel create -o orderer.example.com:7050 -c $CHANNEL_NAME -f ./channel-artifacts/channel.tx
docker exec -e "CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp" -e "CORE_PEER_ADDRESS=peer0.org1.example.com:7051" -e "CORE_PEER_LOCALMSPID=Org1MSP" cli peer channel join -b mychannel.block
echo "============================= peer0.org1 joined channel ============================="
docker exec -e "CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp" -e "CORE_PEER_ADDRESS=peer0.org2.example.com:7051" -e "CORE_PEER_LOCALMSPID=Org2MSP" cli peer channel join -b mychannel.block
echo "============================= peer0.org2 joined channel ============================="
docker exec -e "CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org3.example.com/users/Admin@org3.example.com/msp" -e "CORE_PEER_ADDRESS=peer0.org3.example.com:7051" -e "CORE_PEER_LOCALMSPID=Org3MSP" cli peer channel join -b mychannel.block
echo "============================= peer0.org3 joined channel ============================="
docker exec -e "CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org4.example.com/users/Admin@org4.example.com/msp" -e "CORE_PEER_ADDRESS=peer0.org4.example.com:7051" -e "CORE_PEER_LOCALMSPID=Org4MSP" cli peer channel join -b mychannel.block
echo "============================= peer0.org4 joined channel ============================="


echo "################################### Installing and instantiating chaincode ###################################"

docker exec -e "CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp" -e "CORE_PEER_ADDRESS=peer0.org1.example.com:7051" -e "CORE_PEER_LOCALMSPID=Org1MSP" cli peer chaincode install -n ca-blockchain -v 1.0 -p github.com/chaincode/ca-blockchain/go
echo "============================= chaincode installed on peer0.org1 ============================="
docker exec -e "CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org2.example.com/users/Admin@org2.example.com/msp" -e "CORE_PEER_ADDRESS=peer0.org2.example.com:7051" -e "CORE_PEER_LOCALMSPID=Org2MSP" cli peer chaincode install -n ca-blockchain -v 1.0 -p github.com/chaincode/ca-blockchain/go
echo "============================= chaincode installed on peer0.org2 ============================="
docker exec -e "CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org3.example.com/users/Admin@org3.example.com/msp" -e "CORE_PEER_ADDRESS=peer0.org3.example.com:7051" -e "CORE_PEER_LOCALMSPID=Org3MSP" cli peer chaincode install -n ca-blockchain -v 1.0 -p github.com/chaincode/ca-blockchain/go
echo "============================= chaincode installed on peer0.org3 ============================="
docker exec -e "CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org4.example.com/users/Admin@org4.example.com/msp" -e "CORE_PEER_ADDRESS=peer0.org4.example.com:7051" -e "CORE_PEER_LOCALMSPID=Org4MSP" cli peer chaincode install -n ca-blockchain -v 1.0 -p github.com/chaincode/ca-blockchain/go
echo "============================= chaincode installed on peer0.org4 ============================="

docker exec -e "CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/org1.example.com/users/Admin@org1.example.com/msp" -e "CORE_PEER_ADDRESS=peer0.org1.example.com:7051" -e "CORE_PEER_LOCALMSPID=Org1MSP" cli peer chaincode instantiate -o orderer.example.com:7050 -C $CHANNEL_NAME -n ca-blockchain -v 1.0 -c '{"Args":[""]}' -P "AND ('Org1MSP.member','Org2MSP.member','Org3MSP.member')"
echo "============================= chaincode instantiated ============================="


echo "################################### Done ###################################"