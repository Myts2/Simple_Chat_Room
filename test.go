package main

import (
	"fmt"
	"github.com/Myts2/Simple_Chat_Room/database"
	"sync"
)

var mutex sync.Mutex
var lockbool = make(chan bool)

func main() {
	mutex.Lock()
	mutex.Lock()
	//"root:qq784400047@/GolangChatRoom?charset=utf8"
	database.Init_database("root", "qq784400047", "127.0.0.1", "3306", "GolangChatRoom")
	fmt.Println(database.PushOfflineMsg("test", "test01", "test"))
	fmt.Println(database.PushOfflineMsg("test", "test01", "test"))
	msg_list, err := database.GetOfflineMsg("test01")
	if err != "ok" {
		fmt.Println(err)
	}
	fmt.Printf("%v", msg_list)
	mutex.Unlock()
}
