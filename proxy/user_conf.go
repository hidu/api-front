package proxy

import (
	"github.com/hidu/goutils"
	"log"
)

type users []string

type user struct {
	Name   string
	PswMd5 string
}

type usersConf struct {
	users map[string]*user
}

func (us users) hasUser(name string) bool {
	for _, n := range us {
		if n == name {
			return true
		}
	}
	return false
}

func (uc *usersConf) checkUser(name string, psw string) *user {
	if u, has := uc.users[name]; has && u.PswMd5 == utils.StrMd5(psw) {
		return u
	}
	return nil
}

func (uc *usersConf) getUser(name string) *user {
	if u, has := uc.users[name]; has {
		return u
	}
	return nil
}

func loadUsers(confPath string) (uc *usersConf) {
	log.Println("loadUsers file:", confPath)
	uc = &usersConf{
		users: make(map[string]*user),
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
		name, has := line["name"]
		if !has || name == "" {
			continue
		}
		if _, has := uc.users[name]; has {
			log.Println("dup name in users:", name, line)
			continue
		}

		user := new(user)
		user.Name = name
		if val, has := line["psw_md5"]; has {
			user.PswMd5 = val
		}
		uc.users[user.Name] = user
	}
	return
}
