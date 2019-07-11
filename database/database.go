package database

import (
	"encoding/json"
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql" // 导入数据库驱动
)

type User struct {
	Uid         int `orm:"column(uid);pk"`
	Username    string
	Pass_md5    string
	Pub_cert    string
	Friend_list string
	Token       string
	Ip          string
	Msg         string
}

func Init_database() {
	// 设置默认数据库
	orm.RegisterDriver("mysql", orm.DRMySQL)
	orm.RegisterDataBase("default", "mysql", "root:qq784400047@/GolangChatRoom?charset=utf8", 30)

	// 注册定义的 model

	//RegisterModel 也可以同时注册多个 model
	orm.RegisterModel(new(User))
	// 创建 table
	orm.RunSyncdb("default", false, true)
}

func GetCert(username string) (string, error) {
	o := orm.NewOrm()
	var tmp_user User
	qs := o.QueryTable(User{})
	err := qs.Filter("Username", username).One(&tmp_user)
	return tmp_user.Pub_cert, err
}

func GetPass(username string) (string, error) {
	o := orm.NewOrm()
	var tmp_user User
	qs := o.QueryTable(User{})
	err := qs.Filter("Username", username).One(&tmp_user)
	return tmp_user.Pass_md5, err
}

func Register(username string, pass_md5 string, cert string) string {
	o := orm.NewOrm()

	var try_username_exist User
	err := o.QueryTable(User{}).Filter("Username", username).One(&try_username_exist)
	if err != nil {

		_, err := o.Insert(&User{Username: username, Pass_md5: pass_md5, Pub_cert: cert})
		if err != nil {
			return err.Error()
		} else {
			return "ok"
		}
	}
	return "User exist"
}

func PushToken(username string, token string) error {
	o := orm.NewOrm()

	var tmp_user User
	err := o.QueryTable(User{}).Filter("Username", username).One(&tmp_user)
	if err == nil {
		tmp_user.Token = token

		_, err := o.Update(&tmp_user)
		return err
	}
	return err
}

func PushIP(username string, IP string) error {
	o := orm.NewOrm()

	var tmp_user User
	err := o.QueryTable(User{}).Filter("Username", username).One(&tmp_user)
	if err == nil {
		tmp_user.Ip = IP

		_, err := o.Update(&tmp_user)
		return err
	}
	return err
}

func GetToken(username string) (string, error) {
	o := orm.NewOrm()

	var tmp_user User
	err := o.QueryTable(User{}).Filter("Username", username).One(&tmp_user)
	if err == nil {
		return tmp_user.Token, err
	}
	return "", err
}

func GetFriend(username string) ([]string, error) {
	o := orm.NewOrm()

	var tmp_user User
	err := o.QueryTable(User{}).Filter("Username", username).One(&tmp_user)
	if err == nil {
		var friend_list []string
		json.Unmarshal([]byte(tmp_user.Friend_list), &friend_list)
		return friend_list, err
	}
	return nil, err
}

func GetIP(username string) (string, error) {
	o := orm.NewOrm()

	var tmp_user User
	err := o.QueryTable(User{}).Filter("Username", username).One(&tmp_user)
	if err == nil {
		return tmp_user.Ip, err
	}
	return "", err
}

func PushFriend(username string, friend string) error {
	o := orm.NewOrm()

	var tmp_user User
	err := o.QueryTable(User{}).Filter("Username", username).One(&tmp_user)
	if err == nil {
		var friend_list []string
		json.Unmarshal([]byte(tmp_user.Friend_list), &friend_list)
		friend_list = append(friend_list, friend)
		fin_friend_list, _ := json.Marshal(friend_list)
		tmp_user.Friend_list = string(fin_friend_list)
		_, err := o.Update(&tmp_user)
		return err
	}
	return err
}

func DelFriend(username string, friend string) error {
	o := orm.NewOrm()

	var tmp_user User
	err := o.QueryTable(User{}).Filter("Username", username).One(&tmp_user)
	if err == nil {
		var friend_list []string
		var new_friend_list []string
		json.Unmarshal([]byte(tmp_user.Friend_list), &friend_list)
		for _, n := range friend_list {
			if friend != n {
				new_friend_list = append(new_friend_list, n)
			}
		}
		fin_friend_list, _ := json.Marshal(new_friend_list)
		tmp_user.Friend_list = string(fin_friend_list)
		_, err := o.Update(&tmp_user)
		return err
	}
	return err
}

func PushOfflineMsg(fromUser string, toUser string, toMsg string) string {
	o := orm.NewOrm()

	var tmp_user User
	err := o.QueryTable(User{}).Filter("Username", toUser).One(&tmp_user)
	if err == nil {
		var MsgList []string
		json.Unmarshal([]byte(tmp_user.Msg), &MsgList)
		MsgList = append(MsgList, fromUser+"#"+toMsg)
		tmpMsgList, _ := json.Marshal(MsgList)
		tmp_user.Msg = string(tmpMsgList)
		_, err := o.Update(&tmp_user)
		if err == nil {
			return "ok"
		}
		return err.Error()
	}
	return err.Error()
}

func GetOfflineMsg(fromUser string) ([]string, string) {
	o := orm.NewOrm()

	var tmp_user User
	err := o.QueryTable(User{}).Filter("Username", fromUser).One(&tmp_user)
	if err == nil {
		var MsgList []string
		json.Unmarshal([]byte(tmp_user.Msg), &MsgList)

		tmp_user.Msg = ""
		_, err := o.Update(&tmp_user)
		if err == nil {
			return MsgList, "ok"
		}
		return nil, err.Error()
	}
	return nil, err.Error()
}

func DelOfflineMsg(fromUser string) string {
	o := orm.NewOrm()

	var tmp_user User
	err := o.QueryTable(User{}).Filter("Username", fromUser).One(&tmp_user)
	if err == nil {
		tmp_user.Msg = ""
		_, err := o.Update(&tmp_user)
		if err == nil {
			return "ok"
		}
		return err.Error()
	}
	return err.Error()

}
