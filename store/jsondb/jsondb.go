package jsondb

import (
	"encoding/json"
	"errors"
	"fmt"
	"journal/model"
	"journal/util"
	"os"
	"path"

	"github.com/sdomino/scribble"
)

type JsonDB struct {
	conn   *scribble.Driver
	dbPath string
}

// New returns a new pointer JsonDB
func New(dbPath string) (*JsonDB, error) {
	conn, err := scribble.New(dbPath, nil)
	if err != nil {
		return nil, err
	}
	ans := JsonDB{
		conn:   conn,
		dbPath: dbPath,
	}
	return &ans, nil

}

func (o *JsonDB) Init() error {
	var userPath string = path.Join(o.dbPath, "users")
	var serverPath string = path.Join(o.dbPath, "server")
	var configPath string = path.Join(serverPath, "config.json")
	// create directories if they do not exist
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		os.MkdirAll(serverPath, os.ModePerm)
	}
	if _, err := os.Stat(userPath); os.IsNotExist(err) {
		os.MkdirAll(userPath, os.ModePerm)
	}

	// server's configuration
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		srvConf := new(model.ServerConf)
		srvConf.JwtSecret = util.JwtSecret
		o.conn.Write("server", "config", srvConf)
		os.Chmod(configPath, 0600)
	}

	// default user info for admin
	results, err := o.conn.ReadAll("users")
	if err != nil || len(results) < 1 {
		user := new(model.User)
		user.Username = util.DefaultUsername
		user.Password = util.DefaultPassword
		user.Admin = util.DefaultIsAdmin
		user.PasswordHash, _ = util.HashPassword(util.DefaultPassword)
		user.Enable2FA = false
		o.conn.Write("users", user.Username, user)
		os.Chmod(path.Join(path.Join(o.dbPath, "users"), user.Username+".json"), 0600)
	}

	return nil
}

// GetUser func to query user info from the database
func (o *JsonDB) GetServerConf() (model.ServerConf, error) {
	config := model.ServerConf{}
	return config, o.conn.Read("server", "config", &config)
}

// Save server config func to save config in the database
func (o *JsonDB) SaveServerConf(config model.ServerConf) error {
	configPath := path.Join(path.Join(o.dbPath, "server"), "config.json")
	output := o.conn.Write("server", "config", config)
	os.Chmod(configPath, 0600)
	return output
}

// GetUsers func to get all users from the database
func (o *JsonDB) GetUsers() ([]model.User, error) {
	var users []model.User
	results, err := o.conn.ReadAll("users")
	if err != nil {
		return users, err
	}
	for _, i := range results {
		user := model.User{}

		if err := json.Unmarshal([]byte(i), &user); err != nil {
			return users, fmt.Errorf("cannot decode user json structure: %v", err)
		}
		users = append(users, user)

	}
	return users, err
}

// GetUserByName func to get single user from the database
func (o *JsonDB) GetUserByName(username string) (model.User, error) {
	user := model.User{}

	if err := o.conn.Read("users", username, &user); err != nil {
		return user, err
	}

	return user, nil
}

func (o *JsonDB) GetUserByID(userid string) (model.User, error) {
	users, err := o.GetUsers()
	if err != nil {
		return model.User{}, err
	}
	for _, user := range users {
		if userid == user.Userid {
			return user, nil
		}
	}
	return model.User{}, errors.New("invalid user id" + userid)
}

// SaveUser func to save user in the database
func (o *JsonDB) SaveUser(user model.User) error {
	userPath := path.Join(path.Join(o.dbPath, "users"), user.Username+".json")
	output := o.conn.Write("users", user.Username, user)
	os.Chmod(userPath, 0600)
	return output
}

// DeleteUser func to remove user from the database
func (o *JsonDB) DeleteUser(username string) error {
	return o.conn.Delete("users", username)
}

func (o *JsonDB) GetPath() string {
	return o.dbPath
}
