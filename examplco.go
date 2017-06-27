package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/net/context"

	"github.com/coreos/etcd/client"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app          = kingpin.New("examplco", "An etcd demonstration")
	peerlist     = app.Flag("peers", "etcd peers").Default("http://127.0.0.1:4001,http://127.0.0.1:2379").OverrideDefaultFromEnvar("EX_PEERS").String()
	username     = app.Flag("user", "etcd User").OverrideDefaultFromEnvar("EX_USER").String()
	password     = app.Flag("pass", "etcd Password").OverrideDefaultFromEnvar("EX_PASS").String()
	config       = app.Command("config", "Change config data")
	configserver = config.Arg("server", "Server name").Required().String()
	configvar    = config.Arg("var", "Config variable").Required().String()
	configval    = config.Arg("val", "Config value").Required().String()
	server       = app.Command("server", "Go into server mode and listen for changes")
	servername   = server.Arg("server", "Server name").Required().String()
)

var configbase = "/config/"

func main() {
	kingpin.Version("0.0.1")
	command := kingpin.MustParse(app.Parse(os.Args[1:]))

	peers := strings.Split(*peerlist, ",")

	cfg := client.Config{
		Endpoints:               peers,
		HeaderTimeoutPerRequest: time.Minute,
		Username:                *username,
		Password:                *password,
	}

	etcdclient, err := client.New(cfg)

	if err != nil {
		log.Fatal(err)
	}

	kapi := client.NewKeysAPI(etcdclient)

	switch command {
	case config.FullCommand():
		doConfig(kapi)
	case server.FullCommand():
		doServer(kapi)
	}
}

func doConfig(kapi client.KeysAPI) {
	var key = configbase + *configserver + "/" + *configvar

	resp, err := kapi.Set(context.TODO(), key, *configval, nil)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Action + " " + resp.Node.Key + " to " + resp.Node.Value)
}

func doServer(kapi client.KeysAPI) {
	var key = configbase + *servername

	var settings map[string]string
	settings = make(map[string]string)

	resp, err := kapi.Get(context.TODO(), key, &client.GetOptions{Recursive: true})
	if err != nil {
		log.Fatal(err)
	}

	for _, node := range resp.Node.Nodes {
		_, setting := path.Split(node.Key)
		settings[setting] = node.Value
	}

	fmt.Println(settings)

	watcher := kapi.Watcher(key, &client.WatcherOptions{Recursive: true})

	for true {
		resp, err := watcher.Next(context.TODO())

		if err != nil {
			if _, ok := err.(*client.ClusterError); ok {
				continue
			}
			log.Fatal(err)
		}

		switch resp.Action {
		case "set":
			_, setting := path.Split(resp.Node.Key)
			settings[setting] = resp.Node.Value
		case "delete", "expire":
			_, setting := path.Split(resp.Node.Key)
			delete(settings, setting)
		}

		fmt.Println(settings)
	}

}
