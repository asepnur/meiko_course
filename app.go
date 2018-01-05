package main

import (
	"flag"
	"log"
	"runtime"

	"github.com/asepnur/meiko_course/src/cron"
	"github.com/asepnur/meiko_course/src/email"
	"github.com/asepnur/meiko_course/src/util/alias"
	"github.com/asepnur/meiko_course/src/util/auth"
	"github.com/asepnur/meiko_course/src/util/conn"
	"github.com/asepnur/meiko_course/src/util/env"
	"github.com/asepnur/meiko_course/src/util/jsonconfig"
	"github.com/asepnur/meiko_course/src/webserver"
	"github.com/asepnur/meiko_course/src/webserver/handler/bot"
)

type configuration struct {
	Directory alias.DirectoryConfig `json:"directory"`
	Database  conn.DatabaseConfig   `json:"database"`
	Redis     conn.RedisConfig      `json:"redis"`
	Webserver webserver.Config      `json:"webserver"`
	Email     email.Config          `json:"email"`
	Auth      auth.Config           `json:"auth"`
}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	flag.Parse()

	// load configuration
	cfgenv := env.Get()
	config := &configuration{}
	isLoaded := jsonconfig.Load(&config, "/etc/meiko", cfgenv) || jsonconfig.Load(&config, "./files/etc/meiko", cfgenv)
	if !isLoaded {
		log.Fatal("Failed to load configuration")
	}

	// initiate instance
	alias.InitDirectory(config.Directory)
	conn.InitDB(config.Database)
	conn.InitRedis(config.Redis)
	bot.Init()
	cron.Init()
	auth.Init(config.Auth)
	email.Init(config.Email)
	webserver.Start(config.Webserver)
}
