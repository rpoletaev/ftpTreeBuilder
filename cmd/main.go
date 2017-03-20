package main

import (
	//"bufio"
	tb "github.com/rpoletaev/ftpTreeBuilder"
	"log"
	"os"
)

func main() {
	config := &tb.FTPBuilderConfig{
		FTPAddr:           "ftp.zakupki.gov.ru:21",
		FTPLogin:          "free",
		FTPPass:           "free",
		RootNodeDirectory: "/fcs_regions",
		DBConString:       "root:test@tcp(mysql:3306)/cache?parseTime=true",
		Logger:            log.New(os.Stdout, "ftptree ", 1),
	}

	stop := make(chan struct{})
	b := tb.GetFTPBuilder(config)

	tree := b.BuildTree(stop)
	println("DirsCount: ", tree.DirsCount())
	println("FilesCount: ", tree.FilesCount())
}
