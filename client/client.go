package main

import (
	"crypto/rc4"
	"encoding/hex"
	"fmt"
	"github.com/Myts2/Simple_Chat_Room/RSA"
	"github.com/Myts2/Simple_Chat_Room/cui"
	"github.com/google/uuid"
	tp "github.com/henrylee2cn/teleport"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"
)

//go:generate go build $GOFILE

type Arg struct {
	username  string
	pass_md5  string
	from_name string
	to_name   string
	content   []byte
	cert      string
}

var myPubkey string
var myPrikey string

var KeySAUser = make(chan (string))
var KeySAUserPub = make(chan (string))

var UserIPName = make(chan (string))
var UserIPPort = make(chan (string))

var Session = make(map[string]tp.Session)
var SessionKey = make(map[string]*rc4.Cipher)

var addConfirm = make(chan (string))
var reqConfirm = make(chan (bool))

var Token string
var CurUser string
var ServerSession tp.Session

func GenArg(args ...string) string {
	return strings.Join(args, ",")
}

func P2pServer() {
	defer tp.FlushLogger()
	// graceful
	go tp.GraceSignal()
	tp.SetLoggerLevel("OFF")
	// server peer
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:   true,
		ListenPort:  7080,
		PrintDetail: false,
	})
	// srv.SetTLSConfig(tp.GenerateTLSConfigForServer())

	// router
	srv.RouteCall(new(p2pcall))
	srv.RoutePush(new(p2push))
	srv.ListenAndServe()
	select {}
}

func main() {
	//fmt.Println("OK")
	go P2pServer()
	defer tp.SetLoggerLevel("OFF")()
	cli := tp.NewPeer(
		tp.PeerConfig{},
	)
	//fmt.Println("OK")
	defer cli.Close()
	group := cli.SubRoute("/cli")
	group.RoutePush(new(push))
	group.RouteCall(new(call))
	//fmt.Println("OK")
	var ServerIP string
	fmt.Print("Server IP:")
	fmt.Scanf("%s", &ServerIP)
	sess, err := cli.Dial(ServerIP + ":9090") //external IP
	if err != nil {
		panic(err)
	}
	//fmt.Println("OK")
	ServerSession = sess
	go func() {
		for {
			ToUser := <-KeySAUser
			var ToUserPub string
			sess.Call(
				"/srv/front/push_pub",
				GenArg(CurUser, Token, ToUser),
				&ToUserPub,
			)
			KeySAUserPub <- ToUserPub
		}
	}()
	go func() {
		for {
			ToUser := <-UserIPName
			var ToUserIP string
			sess.Call(
				"/srv/front/push_ip",
				GenArg(CurUser, Token, ToUser),
				&ToUserIP,
			)
			UserIPPort <- ToUserIP
		}
	}()
	go func() {
		for {
			<-cui.GetFriend
			var FriendList_raw string
			sess.Call(
				"/srv/front/push_friend",
				GenArg(CurUser, Token),
				&FriendList_raw,
			)
			FriendList := strings.Split(FriendList_raw, ",")
			FriendList = append(FriendList, "Global")
			cui.CurFriendList <- FriendList
		}
	}()
	go func() {
		for {
			Loginuser := <-cui.LoginUser
			Loginpass := <-cui.LoginPass
			cui.LoginStatus <- Login(Loginuser, Loginpass)
		}
	}()
	go func() {
		for {
			Reguser := <-cui.RegUser
			Regpass := <-cui.RegPass
			cui.RegStatus <- Register(Reguser, Regpass)
		}
	}()
	go func() {
		for {
			ChatUser := <-cui.ChatUser
			Chatmsg := <-cui.Chatmsg
			if ChatUser == "Global" {
				command := strings.Split(Chatmsg[:len(Chatmsg)-1], " ")
				var result string
				select {
				case _ = <-reqConfirm:
					switch command[0] {
					case "y":
						addConfirm <- "y"
					case "n":
						addConfirm <- "n"
					}
				default:
				}
				if len(command) == 2 {
					switch command[0] {
					case "add":
						result = AddFriend(command[1])
					case "del":
						result = DelFriend(command[1])
					}
				}

				cui.RecvUser <- "Global"
				cui.RecvMsg <- result
			} else {
				Chat(ChatUser, Chatmsg)
			}
		}
	}()
	cui.InitCui()
}

func Login(username string, password string) string {
	var result string
	ServerSession.Call("/srv/front/login",
		GenArg(username, password),
		&result,
		tp.WithSetMeta("push_status", "yes"),
	)
	//tp.Printf("result: %s", result)
	user1_token := result
	_, err_UUID := uuid.Parse(user1_token)
	if err_UUID != nil {
		return result
	}
	Token = user1_token
	CurUser = username

	privateFilename := CurUser + "_private.pem"
	myPrikey_raw, err := ioutil.ReadFile(privateFilename)
	if err != nil {
		fmt.Println(err)
	}
	myPrikey = string(myPrikey_raw)
	return "ok"
}

func Register(username string, password string) string {
	var result string
	Privkey, PublicKey := RSA.GenerateKeyPair(2048)
	PrikeyBytes := (RSA.PrivateKeyToBytes(Privkey))
	PubkeyStr := string(RSA.PublicKeyToBytes(PublicKey))
	err := ioutil.WriteFile(username+"_private.pem", PrikeyBytes, 0644)
	if err != nil {
		panic(err)
	}
	ServerSession.Call("/srv/front/register",
		GenArg(username, password, PubkeyStr),
		&result,
		tp.WithSetMeta("push_status", "yes"),
	)
	return result
}

func KeySAEnc(username string, Key int) string {
	KeySAUser <- username
	ToUserPub := <-KeySAUserPub
	publicKey := RSA.BytesToPublicKey([]byte(ToUserPub))
	KeyEnc_raw := RSA.EncryptWithPublicKey([]byte("key"+strconv.Itoa(int(Key))), publicKey)
	KeyEnc := make([]byte, hex.EncodedLen(len(KeyEnc_raw)))
	hex.Encode(KeyEnc, KeyEnc_raw)
	return string(KeyEnc)
}

func KeySADec(KeyEnc string) int {
	KeyEnc_raw := make([]byte, hex.DecodedLen(len(KeyEnc)))
	_, err := hex.Decode(KeyEnc_raw, []byte(KeyEnc))
	if err != nil {
		log.Fatal(err)
	}
	privateKey := RSA.BytesToPrivateKey([]byte(myPrikey))

	Key_raw := string(RSA.DecryptWithPrivateKey(KeyEnc_raw, privateKey))
	if Key_raw[:3] == "key" {
		key, err := strconv.Atoi(string(Key_raw)[3:])
		if err != nil {
			log.Fatal(err)
		}
		return key
	} else {
		return 0
	}
}

func Chat(to_name string, msg string) {

	cli := tp.NewPeer(
		tp.PeerConfig{},
	)
	//fmt.Println(to_name)
	UserIPName <- to_name
	toUserIPPort := <-UserIPPort
	if toUserIPPort == "offline" {
		var result string
		ServerSession.Call(
			"/srv/front/recv_offline_msg",
			GenArg(CurUser, Token, to_name, msg),
			&result,
		)
		cui.RecvUser <- to_name
		cui.RecvMsg <- "(Server Recv Offline msg)"

		SessionKey[to_name] = nil
		return
	}
	TmpSession, Perr := cli.Dial(fmt.Sprintf("%s:7080", strings.Split(toUserIPPort, ":")[0]))
	if Perr != nil {
		panic(Perr)
	}
	Session[to_name] = TmpSession
	//fmt.Println("Session INIT OK")

	//fmt.Printf("Session to %s OK\n",to_name)
	if SessionKey[to_name] == nil {
		KeyA := RSA.RandNumGen()
		KeyAEnc := KeySAEnc(to_name, KeyA)
		var result string
		Session[to_name].Call(
			"/p2pcall/key_sa",
			GenArg(CurUser, KeyAEnc),
			&result,
		)
		KeyB := KeySADec(result)
		FinalRc4Key := strconv.Itoa(int(KeyA) * int(KeyB))
		//fmt.Printf("KeyA:%d,KeyB:%d,FinalKey:%s\n",KeyA,int(KeyB),RSA.Md5V2(FinalRc4Key))
		SessionKey[to_name], _ = rc4.NewCipher([]byte(RSA.Md5V2(FinalRc4Key)))
	}
	//fmt.Printf("SessionKey to %s OK\n",to_name)
	msgEncode := make([]byte, len(msg))
	SessionKey[to_name].XORKeyStream(msgEncode, []byte(msg))
	msgEncode_raw := make([]byte, hex.EncodedLen(len(msgEncode)))
	hex.Encode(msgEncode_raw, msgEncode)
	Session[to_name].Push(
		"/p2push/chat",
		GenArg(CurUser, string(msgEncode_raw)))
	return
}

func AddFriend(adduser string) string {
	var result string
	ServerSession.Call("/srv/front/add_friend",
		GenArg(CurUser, Token, adduser),
		&result,
		tp.WithSetMeta("push_status", "yes"),
	)
	if result == "ok" {
		cui.GetFriend <- true
	}
	//fmt.Println(result)
	return result
}

func DelFriend(deluser string) string {
	var result string
	ServerSession.Call("/srv/front/del_friend",
		GenArg(CurUser, Token, deluser),
		&result,
		tp.WithSetMeta("push_status", "yes"),
	)
	if result == "del ok" {
		cui.GetFriend <- true
	}
	//fmt.Println(result)
	return result
}

type push struct {
	tp.PushCtx
}

type call struct {
	tp.CallCtx
}

type p2pcall struct {
	tp.CallCtx
}

type p2push struct {
	tp.PushCtx
}

func (p *push) ServerStatus(arg *string) *tp.Rerror {
	tp.Printf("%v", *arg)
	return nil
}

func (p *push) Online(arg_raw *string) *tp.Rerror {
	onlineUser := *arg_raw
	cui.OnlineUser <- onlineUser
	return nil
}

func (p *push) PushOffline(arg_raw *string) *tp.Rerror {
	arg := strings.Split(*arg_raw, ",")
	fromUser := arg[0]
	msg := arg[1]
	cui.RecvUser <- fromUser
	cui.RecvMsg <- msg
	return nil
}

func (c *call) AddConfirm(arg_raw *string) (string, *tp.Rerror) {
	addWho := *arg_raw
	cui.RecvUser <- "Global"
	cui.RecvMsg <- fmt.Sprintf("%s add you,confirm?(y/n)", addWho)
	reqConfirm <- true
	var result string
	result = <-addConfirm
	if result == "y" {
		go func() {
			time.Sleep(time.Second * 3)
			cui.GetFriend <- true
		}()
		return "ok", nil
	} else {
		return "deny", nil
	}
}

func (c *call) DelConfirm(arg_raw *string) (string, *tp.Rerror) {
	delWho := *arg_raw
	cui.RecvUser <- "Global"
	cui.RecvMsg <- fmt.Sprintf("%s del you", delWho)

	go func() {
		time.Sleep(time.Second * 1)
		cui.GetFriend <- true
	}()
	return "ok", nil
}

func (c *p2pcall) KeySA(arg_raw *string) (string, *tp.Rerror) {
	//fmt.Print("Handle KeySA")
	arg := strings.Split(*arg_raw, ",")
	username := arg[0]
	KeyEnc := arg[1]
	KeyA := KeySADec(KeyEnc)

	KeyB := RSA.RandNumGen()
	KeyBenc := KeySAEnc(username, KeyB)
	FinalRc4Key := strconv.Itoa(KeyA * int(KeyB))
	//fmt.Printf("KeyA:%d,KeyB:%d,FinalKey:%s\n",KeyA,int(KeyB),RSA.Md5V2(FinalRc4Key))
	SessionKey[username], _ = rc4.NewCipher([]byte(RSA.Md5V2(FinalRc4Key)))
	return KeyBenc, nil
}

func (p *p2push) Chat(arg_raw *string) *tp.Rerror {
	arg := strings.Split(*arg_raw, ",")
	username := arg[0]
	msg := arg[1]
	msg_raw := make([]byte, hex.DecodedLen(len(msg)))
	_, err := hex.Decode(msg_raw, []byte(msg))
	if err != nil {
		panic(err)
	}
	if SessionKey[username] != nil {
		msgDecode := make([]byte, len(msg_raw))
		SessionKey[username].XORKeyStream(msgDecode, []byte(msg_raw))
		//fmt.Printf("%s: %s\n",username,msgDecode)
		cui.RecvUser <- username
		cui.RecvMsg <- string(msgDecode)
	} else {
		//fmt.Println("Err in SessionKey")
	}
	return nil
}
