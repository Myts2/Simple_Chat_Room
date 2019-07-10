package main

import (
	"crypto/md5"
	"crypto/rc4"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/Myts2/Simple_Chat_Room/RSA"
	"github.com/google/uuid"
	"github.com/henrylee2cn/teleport"
	"io/ioutil"
	"log"
	"crypto/rand"
	"math"
	"math/big"
	"strconv"
	"strings"
)

//go:generate go build $GOFILE

type Arg struct {
	username string
	pass_md5 string
	from_name string
	to_name string
	content []byte
	cert string

}

var myPubkey string
var myPrikey string

var KeySAUser = make(chan(string))
var KeySAUserPub = make(chan(string))

var UserIPName = make(chan(string))
var UserIPPort = make(chan(string))

var Session = make(map[string]tp.Session)
var SessionKey = make(map[string]*rc4.Cipher)

var Token string
var CurUser string
var ServerSession tp.Session

func GenArg(args ...string) (string){
	return strings.Join(args, ",")
}

func P2pServer(){
	defer tp.FlushLogger()
	// graceful
	go tp.GraceSignal()

	// server peer
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:   true,
		ListenPort:  7080,
		PrintDetail: true,
	})
	// srv.SetTLSConfig(tp.GenerateTLSConfigForServer())

	// router
	srv.RouteCall(new(p2pcall))
	srv.RoutePush(new(p2push))
	srv.ListenAndServe()
	select {}
}

func main() {
	username := flag.String("u","","Username")
	password := flag.String("p","","password")
	sendmsg := flag.Bool("send",false,"sendmsg")
	to_user := flag.String("t","","to_user")
	msg := flag.String("m","","msg")
	reg := flag.Bool("reg",false,"register")
	reg_name := flag.String("rn","","reg_name")
	reg_pass := flag.String("rp","","reg_pass")
	add := flag.Bool("add",false,"register")
	addUser := flag.String("au","","addUser")
	del := flag.Bool("del",false,"del")
	delUser := flag.String("du","","deluser")
	//reg_cert := flag.String("rc","","reg_cert")
	flag.Parse()



	if *reg{
		Register(*reg_name,*reg_pass)
		return
	}
	Login(*username,*password)
	if *sendmsg{
		Chat(*to_user,*msg)
	}
	if *add{
		AddFriend(*addUser)
	}
	if *del{
		DelFriend(*delUser)
	}

	//user2_token := result


	select {

	}
}

func ClientInit(){
	go P2pServer()

	defer tp.SetLoggerLevel("ERROR")()

	cli := tp.NewPeer(
		tp.PeerConfig{},
	)
	defer cli.Close()
	group := cli.SubRoute("/cli")
	group.RoutePush(new(push))
	group.RouteCall(new(call))


	sess, err := cli.Dial("172.17.0.1:9090") //external IP
	if err != nil {
		tp.Fatalf("%v", err)
	}
	ServerSession = sess
	go func(){
		for{
			ToUser := <-KeySAUser
			var ToUserPub string
			sess.Call(
				"/srv/front/push_pub",
				GenArg(CurUser,Token,ToUser),
				&ToUserPub,
			)
			KeySAUserPub<-ToUserPub
		}
	}()
	go func(){
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

}

func md5V2(str string) string {
	data := []byte(str)
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has)
	return md5str
}

func RandNumGen() (int){
	n,err := rand.Int(rand.Reader,big.NewInt(math.MaxInt16))
	if err != nil{
		n,err = rand.Int(rand.Reader,big.NewInt(math.MaxInt16))
	}
	return int(n.Int64())
}

func Login(username string,password string){
	var result string
	ServerSession.Call("/srv/front/login",
		GenArg(username,password),
		&result,
		tp.WithSetMeta("push_status", "yes"),
	)
	//tp.Printf("result: %s", result)
	user1_token := result
	_,err_UUID := uuid.Parse(user1_token)
	if err_UUID != nil{
		tp.Printf("result: %s", result)
		return
	}
	Token = user1_token
	CurUser = username
	fmt.Println("Get Token OK")
	privateFilename := "/tmp/PEMs/"+CurUser+"_private.pem"
	myPrikey_raw,err := ioutil.ReadFile(privateFilename)
	if err != nil {
		fmt.Print(err)
	}
	myPrikey = string(myPrikey_raw)
}

func Register(username string,password string){
	var result string
	Privkey, PublicKey := RSA.GenerateKeyPair(2048)
	PrikeyBytes := (RSA.PrivateKeyToBytes(Privkey))
	PubkeyStr := string(RSA.PublicKeyToBytes(PublicKey))
	err := ioutil.WriteFile("/tmp/PEMs/"+username+"_private.pem",PrikeyBytes,0644)
	if err!= nil{
		panic(err)
	}
	ServerSession.Call("/srv/front/register",
		GenArg(username,password,PubkeyStr),
		&result,
		tp.WithSetMeta("push_status", "yes"),
	)
	fmt.Println(result)
}

func KeySAEnc(username string,Key int)(string){
	KeySAUser<-username
	ToUserPub := <-KeySAUserPub
	publicKey := RSA.BytesToPublicKey([]byte(ToUserPub))
	KeyEnc_raw := RSA.EncryptWithPublicKey([]byte("key"+strconv.Itoa(int(Key))),publicKey)
	KeyEnc := make([]byte, hex.EncodedLen(len(KeyEnc_raw)))
	hex.Encode(KeyEnc, KeyEnc_raw)
	return string(KeyEnc)
}

func KeySADec(KeyEnc string)(int){
	KeyEnc_raw := make([]byte, hex.DecodedLen(len(KeyEnc)))
	_, err := hex.Decode(KeyEnc_raw, []byte(KeyEnc))
	if err != nil {
		log.Fatal(err)
	}
	privateKey := RSA.BytesToPrivateKey([]byte(myPrikey))

	Key_raw := string(RSA.DecryptWithPrivateKey(KeyEnc_raw,privateKey))
	if Key_raw[:3] == "key"{
		key,err := strconv.Atoi(string(Key_raw)[3:])
		if err != nil {
			log.Fatal(err)
		}
		return key
	}else{
		return 0
	}
}

func Chat(to_name string,msg string){
	if Session[to_name] == nil{
		cli := tp.NewPeer(
			tp.PeerConfig{},
		)
		fmt.Println(to_name)
		UserIPName <- to_name
		toUserIPPort := <- UserIPPort

		TmpSession , Perr := cli.Dial(fmt.Sprintf("%s:7080",strings.Split(toUserIPPort,":")[0]))
		if Perr != nil{
			panic(Perr)
		}
		Session[to_name] = TmpSession
		fmt.Println("Session INIT OK")
	}
	fmt.Printf("Session to %s OK\n",to_name)
	if SessionKey[to_name] == nil{
		KeyA := RandNumGen()
		KeyAEnc:=KeySAEnc(to_name,KeyA)
		var result string
		Session[to_name].Call(
			"/p2pcall/key_sa",
			GenArg(CurUser,KeyAEnc),
			&result,
		)
		KeyB := KeySADec(result)
		FinalRc4Key := strconv.Itoa(int(KeyA)*int(KeyB))
		fmt.Printf("KeyA:%d,KeyB:%d,FinalKey:%s\n",KeyA,int(KeyB),md5V2(FinalRc4Key))
		SessionKey[to_name],_ = rc4.NewCipher([]byte(md5V2(FinalRc4Key)))
	}
	fmt.Printf("SessionKey to %s OK\n",to_name)
	msgEncode := make([]byte, len(msg))
	SessionKey[to_name].XORKeyStream(msgEncode, []byte(msg))
	msgEncode_raw := make([]byte, hex.EncodedLen(len(msgEncode)))
	hex.Encode(msgEncode_raw, msgEncode)
	Session[to_name].Push(
		"/p2push/chat",
		GenArg(CurUser,string(msgEncode_raw)))
	return
}

func AddFriend(adduser string)(string){
	var result string
	ServerSession.Call("/srv/front/add_friend",
		GenArg(CurUser,Token,adduser),
		&result,
		tp.WithSetMeta("push_status", "yes"),
	)
	fmt.Println(result)
	return result
}

func DelFriend(deluser string)(string){
	var result string
	ServerSession.Call("/srv/front/del_friend",
		GenArg(CurUser,Token,deluser),
		&result,
		tp.WithSetMeta("push_status", "yes"),
	)
	fmt.Println(result)
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
	fmt.Printf("%s Online!\n",onlineUser)
	return nil
}

func (c *call) AddConfirm(arg_raw *string) (string,*tp.Rerror){
	addWho := *arg_raw
	fmt.Printf("%s add you,confirm?(y/n)",addWho)
	var result string
	fmt.Scanf("%s",&result)
	if result == "y"{
		return "ok",nil
	}else{
		return "deny",nil
	}
}


func (c *p2pcall) KeySA(arg_raw *string) (string,*tp.Rerror) {
	fmt.Print("Handle KeySA")
	arg := strings.Split(*arg_raw, ",")
	username := arg[0]
	KeyEnc := arg[1]
	KeyA := KeySADec(KeyEnc)

	KeyB := RandNumGen()
	KeyBenc := KeySAEnc(username,KeyB)
	FinalRc4Key := strconv.Itoa(KeyA*int(KeyB))
	fmt.Printf("KeyA:%d,KeyB:%d,FinalKey:%s\n",KeyA,int(KeyB),md5V2(FinalRc4Key))
	SessionKey[username],_ = rc4.NewCipher([]byte(md5V2(FinalRc4Key)))
	return KeyBenc,nil
}

func (p *p2push) Chat(arg_raw *string) (*tp.Rerror){
	arg := strings.Split(*arg_raw,",")
	username := arg[0]
	msg := arg[1]
	msg_raw := make([]byte, hex.DecodedLen(len(msg)))
	_, err := hex.Decode(msg_raw, []byte(msg))
	if err!=nil{
		panic(err)
	}
	if SessionKey[username] != nil{
		msgDecode := make([]byte, len(msg_raw))
		SessionKey[username].XORKeyStream(msgDecode, []byte(msg_raw))
		fmt.Printf("%s: %s\n",username,msgDecode)
	}else{
		fmt.Println("Err in SessionKey")
	}
	return nil
}