package main

import (
	"fmt"
	"github.com/Myts2/Simple_Chat_Room/database"
)

func main() {
	database.Init_database()
	fmt.Println(database.PushOfflineMsg("test", "fin1", "test"))
	msg_list, err := database.GetOfflineMsg("fin1")
	if err != "ok" {
		fmt.Println(err)
	}
	fmt.Printf("%v", msg_list)

}
