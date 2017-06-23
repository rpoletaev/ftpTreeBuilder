package ftpTreeBuilder

import (
	"fmt"
	"path"
	"sync"
	"time"

	"bytes"
	fp "path/filepath"
	"strconv"

	"strings"

	"github.com/garyburd/redigo/redis"
	"github.com/jinzhu/gorm"
	"github.com/rpoletaev/goftp"
)

var (
	availableConnections chan *goftp.FTP
	wg                   sync.WaitGroup
)

type dirInfo struct {
	Path string
	List []string
}

type fileInfo struct {
	Name   string
	Size   int
	Folder string
}

func (f fileInfo) Path() string {
	return fp.Join(f.Folder, f.Name)
}

type FTPBuilder struct {
	*FTPBuilderConfig
	tree               *Tree
	db                 *gorm.DB
	redisPool          *redis.Pool
	mySQLReconnectDone chan struct{}
	batchChan          chan FTPNode
	// filesChan          chan fileInfo
	Done bool
}

func GetFTPBuilder(c *FTPBuilderConfig) *FTPBuilder {
	defer c.Prepare()
	fmt.Println(*c)
	dbc, err := gorm.Open("mysql", c.DBConString)
	if err != nil {
		panic(err)
	}

	dbc.AutoMigrate(&FTPNode{})
	dbc.DB().SetConnMaxLifetime(time.Minute)
	dbc.DB().SetMaxOpenConns(5)
	b := &FTPBuilder{
		FTPBuilderConfig:   c,
		db:                 dbc,
		mySQLReconnectDone: make(chan struct{}, 1),
		redisPool:          newRedisPool(c.RedisConString, "t=95ZZZ%"),
	}

	go b.MySQLReconnect()
	return b
}

// BuildTree Строит дерево каталогов
func (b *FTPBuilder) BuildTree(done <-chan struct{}) {
	b.batchChan = make(chan FTPNode, 500)
	go b.Batch()
	ftp, err := b.getConnection()
	if err != nil {
		println(err.Error())
		return
	}

	start := time.Now()
	list, err := ftp.List(b.RootNodeDirectory)
	ftp.Close()
	if err != nil {
		println(err.Error())
		return
	}

	regions := make(chan string, 10)
	go func() {
		for _, str := range list {
			name, err := NameFromFileSting(str)
			if err != nil {
				println(err.Error())
				continue
			}

			if IsDir(name) {
				regions <- name
			}
		}
		close(regions)
	}()

	files := make(chan fileInfo)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for region := range regions {
				// b.processRegion(region, files)
				b.processDir(path.Join(b.RootNodeDirectory, region), files)
			}
		}()
	}

	go func() {
		wg.Wait()
		println("close files channel")
		close(files)
	}()

	var counter int64
	for f := range files {
		b.proccessFile(f)
		counter++
	}

	close(b.batchChan)
	println(counter)
	println(time.Since(start).String())
	b.writeResultMessage()
}

func (b *FTPBuilder) processRegion(name string, files chan fileInfo) {
	println("processRegion", name)
	regionPath := path.Join(b.RootNodeDirectory, name)
	ftp, err := b.getConnection()
	if err != nil {
		println(err.Error())
		return
	}

	docs, err := ftp.List(regionPath)
	if err != nil {
		println(err.Error())
		return
	}

	for _, docItem := range docs {
		doc, err := NameFromFileSting(docItem)
		if err != nil {
			println(err.Error())
			continue
		}
		docPath := path.Join(regionPath, doc)
		b.processDir(docPath, files)
	}
}

func (b *FTPBuilder) processDir(dirPath string, files chan fileInfo) error {
	ftp, err := b.getConnection()
	if err != nil {
		return err
	}

	dirList, err := ftp.List(dirPath)
	println("process dir ", dirPath)
	if err != nil {
		return err
	}

	ftp.Close()

	for _, item := range dirList {
		name, err := NameFromFileSting(item)
		if err != nil {
			return err
		}

		if IsDir(name) {
			dir := path.Join(dirPath, name)
			b.processDir(dir, files)
			continue
		}

		if IsZip(name) {
			size, err := SizeFromFileString(item)
			if err != nil {
				fmt.Println("Ошибка при получении размера файла", err.Error())
				continue
			}

			files <- fileInfo{
				Name: path.Join(dirPath, name),
				Size: size,
			}
		}
	}

	return nil
}

func (b *FTPBuilder) proccessFile(f fileInfo) {
	var downloaded uint8
	if f.Size <= 22 {
		downloaded = 1
	} else {
		downloaded = 0
	}

	model := FTPNode{
		Path:       f.Name,
		Downloaded: downloaded,
	}

	count := 0
	b.db.Model(&model).Where("path = ?", f.Name).Count(&count)
	if count == 0 {
		b.batchChan <- model
		// if err := b.fileToQueue(p); err != nil {
		// 	b.Printf("%v\n", err)
		// }
	}
}

func (b *FTPBuilder) pathToErrors(path string) error {
	conn := b.redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SADD", "UnprocessPath", path)
	return err
}
func newRedisPool(addr, pwd string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr, redis.DialDatabase(0), redis.DialPassword(pwd))
		},
	}
}

func (b *FTPBuilder) MySQLReconnect() {
	ticker := time.NewTicker(30 * time.Second)
	for _ = range ticker.C {
		select {
		case <-b.mySQLReconnectDone:
			ticker.Stop()
			return
		default:
			if err := b.db.DB().Ping(); err != nil {
				if b.db, err = gorm.Open("mysql", b.DBConString); err != nil {
					b.Logger.Fatalf("Ошибка соединения с mysql: %v", err)
				}
			}
		}
	}
}

func (b *FTPBuilder) writeResultMessage() {
	c := b.redisPool.Get()
	if _, err := c.Do("PUBLISH", "FTPBuilderResult", time.Now().Unix()); err != nil {
		b.Logger.Printf("Error on sending result: %v\n", err)
	}
}

func (b *FTPBuilder) Batch() {
	var buf *bytes.Buffer
	counter := 0

	for s := range b.batchChan {
		if counter == 0 {
			buf = bytes.NewBufferString("INSERT INTO ftp_nodes (path, downloaded, sort) VALUES ")
		}

		downloaded := strconv.FormatUint(uint64(s.Downloaded), 10)
		buf.WriteString("(\"")
		buf.WriteString(s.Path)
		buf.WriteString(`", ` + downloaded)
		buf.WriteString(`, ` + strconv.Itoa(orderFromPath(s.Path)) + `)`)

		if counter < 500 {
			counter++
			buf.WriteString(",")
		} else {
			counter = 0
			q := buf.String()
			if err := b.db.Exec(q).Error; err != nil {
				println(q)
				fmt.Printf("%v", err)
			}

		}
	}

	println("Запишем оставшиеся записи")
	bts := buf.Bytes()
	q := string(bts[:len(bts)-1])
	if err := b.db.Exec(q).Error; err != nil {
		fmt.Printf("%v", err)
		println(q)
	}
}

func orderFromPath(path string) int {
	return 0
	if strings.Contains(path, "currMonth") {
		return 0
	}

	if strings.Contains(path, "prevMonth") {
		return 1
	}

	return 2
}
