package store

import (
	"journal/model"
)

type IStore interface {
	Init() error
	GetServerConf() (model.ServerConf, error)
	SaveServerConf(config model.ServerConf) error
	GetUsers() ([]model.User, error)
	GetUserByName(username string) (model.User, error)
	GetUserByID(userid string) (model.User, error)
	SaveUser(user model.User) error
	DeleteUser(username string) error
	GetPath() string
}
