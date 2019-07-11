package main

import (
	"flag"
	"fmt"
	"github.com/Myts2/Simple_Chat_Room/database"
	"github.com/google/uuid"
	"strings"
	"time"

	//micro "github.com/xiaoenai/tp-micro"
	"github.com/henrylee2cn/teleport"
)

//go:generate go build $GOFILE

type User_chat struct {
	username string
	token    string
}

var login_msg = make(chan User_chat)

var addFriendName = make(chan string)
var addFriendFrom = make(chan string)
var addFriendStatus = make(chan string)

var IFonlineUser = make(chan string)
var IFonlineStatus = make(chan string)

func main() {
	mysqlUser := flag.String("u", "", "mysql username")
	mysqlPass := flag.String("p", "", "mysql password")
	mysqlHost := flag.String("h", "127.0.0.1", "mysql host(default 127.0.0.1)")
	mysqlPort := flag.String("mp", "3306", "mysql port(default 3306)")
	mysqlTable := flag.String("t", "", "mysql table")
	flag.Parse()
	if *mysqlUser == "" {
		flag.Usage()
		return
	}
	database.Init_database(*mysqlUser, *mysqlPass, *mysqlHost, *mysqlPort, *mysqlTable)
	defer tp.FlushLogger()
	srv := tp.NewPeer(tp.PeerConfig{
		CountTime:   true,
		ListenPort:  9090,
		PrintDetail: true,
	})
	group := srv.SubRoute("/srv")
	group.RouteCall(new(Front))
	go func() {
		for {
			userchat := <-login_msg
			friendList, _ := database.GetFriend(userchat.username)
			for _, single_user_ID := range friendList {
				sess, ok := srv.GetSession(single_user_ID)
				if ok {
					sess.Push(
						"/cli/push/online",
						userchat.username,
					)
					go func() {
						time.Sleep(time.Second / 2)
						sess_self, ok := srv.GetSession(userchat.username)
						if ok {
							sess_self.Push(
								"/cli/push/online",
								single_user_ID,
							)
						} else {
							tp.Printf("Not OK ONline")
						}
					}()

				}

			}
			offlineMsgList, err := database.GetOfflineMsg(userchat.username)
			if err == "ok" {
				database.DelOfflineMsg(userchat.username)
				sess_self, ok := srv.GetSession(userchat.username)
				if ok {
					for _, msg_raw := range offlineMsgList {
						msgBlock := strings.Split(msg_raw, "#")
						fromUser := msgBlock[0]
						msgRecv := msgBlock[1]
						sess_self.Push(
							"/cli/push/push_offline",
							strings.Join([]string{fromUser, msgRecv}, ","),
						)
					}

				}
			}

		}
	}()
	go func() {
		for {
			add_who := <-addFriendName
			add_from := <-addFriendFrom
			sess, ok := srv.GetSession(add_who)
			if ok {
				var result string
				sess.Call(
					"/cli/call/add_confirm",
					add_from,
					&result)
				if result == "ok" {
					addFriendStatus <- "ok"
				} else {
					addFriendStatus <- "deny"
				}
			} else {
				addFriendStatus <- "offline"
			}
		}
	}()
	go func() {
		for {
			onlineUser := <-IFonlineUser
			_, ok := srv.GetSession(onlineUser)
			if ok {
				IFonlineStatus <- "ok"
			} else {
				IFonlineStatus <- "offline"
			}
		}

	}()
	srv.ListenAndServe()
}

type Front struct {
	tp.CallCtx
}

func gen_token(pass string) string {
	uuid_new, _ := uuid.NewRandom()
	return uuid_new.String()
}

func (m *Front) Login(arg_raw *string) (string, *tp.Rerror) {
	arg := strings.Split(*arg_raw, ",")

	//m.Session().Push(
	//	"/cli/push/server_status",
	//	fmt.Sprintf("Username:%s,Password_md5:%s", arg[0],arg[1]),
	//)

	username := arg[0]
	pass_md5 := arg[1]
	token := ""
	real_md5, err := database.GetPass(username)
	m.Session().Push(
		"/cli/push/server_status",
		fmt.Sprintf(*arg_raw),
	)
	if err != nil {
		return "Username is invalid", nil
	}
	if pass_md5 == real_md5 {
		token = gen_token(pass_md5)
	} else {
		return "Password is incorrect", nil
	}

	userchat := new(User_chat)
	userchat.username = username
	userchat.token = token
	login_msg <- *userchat
	database.PushToken(username, token)
	m.Session().SetID(username)
	database.PushIP(username, m.Session().RemoteAddr().String())
	return token, nil
}

func (m *Front) Register(arg_raw *string) (string, *tp.Rerror) {

	arg := strings.Split(*arg_raw, ",")
	username := arg[0]
	pass_md5 := arg[1]
	cert := arg[2]

	err := database.Register(username, pass_md5, cert)
	return err, nil
}

func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func (m *Front) AddFriend(arg_raw *string) (string, *tp.Rerror) {
	arg := strings.Split(*arg_raw, ",")
	username := arg[0]
	token := arg[1]
	real_token, _ := database.GetToken(username)
	if real_token == token {
		add_who := arg[2]
		friendList, _ := database.GetFriend(username)
		if Contains(friendList, add_who) {
			return "Already add this user!", nil
		}
		addFriendName <- add_who
		addFriendFrom <- username
		status := <-addFriendStatus
		if status == "ok" {
			database.PushFriend(username, add_who)
			database.PushFriend(add_who, username)
			return "ok", nil
		} else {
			return status, nil
		}
	}
	return "Wrong Token", nil
}

func (m *Front) DelFriend(arg_raw *string) (string, *tp.Rerror) {
	arg := strings.Split(*arg_raw, ",")
	username := arg[0]
	token := arg[1]
	real_token, _ := database.GetToken(username)
	if real_token == token {
		del_who := arg[2]
		friendList, _ := database.GetFriend(username)
		if !Contains(friendList, del_who) {
			return "No this user!", nil
		}

		database.DelFriend(username, del_who)
		database.DelFriend(del_who, username)
		return "del ok", nil
	}
	return "Wrong Token", nil
}

func (m *Front) PushPub(arg_raw *string) (string, *tp.Rerror) {
	arg := strings.Split(*arg_raw, ",")
	username := arg[0]
	token := arg[1]
	real_token, _ := database.GetToken(username)
	if real_token == token {
		toUser := arg[2]
		cert, _ := database.GetCert(toUser)
		return cert, nil
	}
	return "Wrong Token", nil
}

func (m *Front) PushIP(arg_raw *string) (string, *tp.Rerror) {
	arg := strings.Split(*arg_raw, ",")
	username := arg[0]
	token := arg[1]
	toUser := arg[2]
	IFonlineUser <- toUser
	onlineStatus := <-IFonlineStatus
	real_token, _ := database.GetToken(username)
	if real_token == token {
		if onlineStatus == "offline" {
			return "offline", nil
		}
		IP, _ := database.GetIP(toUser)
		return IP, nil
	}
	return "Wrong Token", nil
}

func (m *Front) PushFriend(arg_raw *string) (string, *tp.Rerror) {
	arg := strings.Split(*arg_raw, ",")
	username := arg[0]
	token := arg[1]
	real_token, _ := database.GetToken(username)
	if real_token == token {
		friendList, _ := database.GetFriend(username)
		friendList_raw := strings.Join(friendList, ",")
		return friendList_raw, nil
	}
	return "Wrong Token", nil
}

func (m *Front) RecvOfflineMsg(arg_raw *string) (string, *tp.Rerror) {
	arg := strings.Split(*arg_raw, ",")
	username := arg[0]
	token := arg[1]
	real_token, _ := database.GetToken(username)
	if real_token == token {
		toUser := arg[2]
		toMsg := arg[3]
		err := database.PushOfflineMsg(username, toUser, toMsg)
		return err, nil
	}
	return "Wrong Token", nil
}

func (m *Front) Test(arg_raw *string) (string, rerror *tp.Rerror) {
	arg := strings.Split(*arg_raw, ",")
	m.Session().Push(
		"/cli/push/server_status",
		fmt.Sprintf("Arg:%s,Username:%s,Password_md5:%s", *arg_raw, arg[0], arg[1]),
	)
	return nil, nil
}
