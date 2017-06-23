package main

import (
	//"bufio"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	tb "github.com/rpoletaev/ftpTreeBuilder"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	cnfBts, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Ошбка при чтении конфига: %v", err)
	}

	cfg := &tb.FTPBuilderConfig{}
	err = yaml.Unmarshal(cnfBts, cfg)
	if err != nil {
		log.Fatalf("Ошбка конфигурации %v", err)
	}

	cfg.Logger = log.New(os.Stdout, "ftptree ", 1)

	var stop chan struct{}
	time.Sleep(30 * time.Second)
	b := tb.GetFTPBuilder(cfg)
	println("Сервис `готов")
	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		stop = make(chan struct{})
		go b.BuildTree(stop)
		w.Write([]byte("Сервис запущен"))
	})

	cfg.Fatal(http.ListenAndServe(cfg.ServicePort, nil))
}
