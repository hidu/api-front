package proxy

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"github.com/hidu/goutils"
	"log"
)

type users []string

type User struct {
	ID       string
	Email    string
	NickName string
	Picture  string
	PswMd5   string
}

func init() {
	gob.Register(&User{})
}

func NewUsers() users {
	return users{}
}

func (u *User) pswEnc() string {
	return utils.StrMd5(fmt.Sprintf("%s201501116%s", u.ID, u.PswMd5))
}

func (u *User) String() string {
	bs, _ := json.MarshalIndent(u, "", "  ")
	return string(bs)
}

func (u *User)DisplayName()string{
	if(u.NickName!=""){
		return u.NickName
	}
	return u.ID
}

type usersConf struct {
	users map[string]*User
}

func (us users) hasUser(name string) bool {
	for _, n := range us {
		if n == name || n == ":any" {
			return true
		}
	}
	return false
}

func (uc *usersConf) checkUser(id string, psw string) *User {
	if u, has := uc.users[id]; has && u.PswMd5 == utils.StrMd5(psw) {
		return u
	}
	return nil
}

func (uc *usersConf) getUser(id string) *User {
	if u, has := uc.users[id]; has {
		return u
	}
	return nil
}

func loadUsers(confPath string) (uc *usersConf) {
	log.Println("loadUsers file:", confPath)
	uc = &usersConf{
		users: make(map[string]*User),
	}
	if !utils.File_exists(confPath) {
		log.Println("usersFile not exists")
		return
	}
	userInfoByte, err := utils.File_get_contents(confPath)
	if err != nil {
		log.Println("load user file failed:", confPath, err)
		return
	}
	log.Println(string(userInfoByte))
	lines := utils.LoadText2SliceMap(string(userInfoByte))
	for _, line := range lines {
		id, has := line["id"]
		if !has || id == "" {
			continue
		}
		
		if _, has := uc.users[id]; has {
			log.Println("dup id in users:", id, line)
			continue
		}

		user := new(User)
		user.ID = id
		
		if name, has := line["name"];has{
			user.NickName=name
		}
		
		if val, has := line["psw_md5"]; has {
			user.PswMd5 = val
		}
		uc.users[user.ID] = user
	}
	return
}
