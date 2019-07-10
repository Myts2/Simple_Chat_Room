#CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o p2p_test_docker/client ./client.go
GOOS=linux go build -o p2p_test_docker/client ./client/client.go

chmod +x p2p_test_docker/client
sudo docker run -it -v ~/go/src/github.com/Myts2/Simple_Chat_Room/p2p_test_docker/client:/tmp/client -v ~/go/src/github.com/Myts2/Simple_Chat_Room/PEMs:/tmp/PEMs --rm ubuntu  /tmp/client $*
