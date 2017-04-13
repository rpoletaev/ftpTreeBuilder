package main

import (
	//"bufio"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	tb "github.com/rpoletaev/ftpTreeBuilder"
	"log"
	"net/http"
	"os"
)

func main() {
	config := &tb.FTPBuilderConfig{
		FTPAddr:           "ftp.zakupki.gov.ru:21",
		FTPLogin:          "free",
		FTPPass:           "free",
		RootNodeDirectory: "/fcs_regions",
		DBConString:       "root:test@tcp(mysql:3306)/cache?parseTime=true",
		RedisConString:    "redis:6379",
		ServicePort:       ":6767",
		Logger:            log.New(os.Stdout, "ftptree ", 1),
	}

	var stop chan struct{}
	b := tb.GetFTPBuilder(config)

	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		stop = make(chan struct{})
		go b.BuildTree(stop)
		w.Write([]byte("Сервис запущен"))
	})

	// http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
	// 	err = svc.Stop()
	// 	if err != nil {
	// 		w.Write([]byte(err.Error()))
	// 		return
	// 	}

	// 	w.Write([]byte("Сервис остановлен"))
	// })

	config.Fatal(http.ListenAndServe(config.ServicePort, nil))
}
