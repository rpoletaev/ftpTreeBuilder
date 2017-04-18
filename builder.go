package ftpTreeBuilder

import (
	"fmt"
	"github.com/dutchcoders/goftp"
	"github.com/garyburd/redigo/redis"
	"github.com/jinzhu/gorm"
	"path"
	"sync"
	"time"
)

var (
	availableConnections chan *goftp.FTP
	wg                   sync.WaitGroup
)

type FTPBuilder struct {
	*FTPBuilderConfig
	tree               *Tree
	db                 *gorm.DB
	redisPool          *redis.Pool
	mySQLReconnectDone chan struct{}
	Done               bool
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
	dbc.DB().SetMaxOpenConns(10)
	b := &FTPBuilder{
		FTPBuilderConfig:   c,
		db:                 dbc,
		mySQLReconnectDone: make(chan struct{}, 1),
		redisPool:          newRedisPool(c.RedisConString),
	}
	go b.MySQLReconnect()
	return b
}

// BuildTree Строит дерево каталогов
func (b *FTPBuilder) BuildTree(done <-chan struct{}) {
	availableConnections = make(chan *goftp.FTP, b.MaxFTPCons)
	for len(availableConnections) < b.MaxFTPCons {
		availableConnections <- b.ftpConnect()
	}

	stop := make(chan struct{})
	//Периодически выводим количество свободных соединений
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		for {
			select {
			case <-ticker.C:
				b.Println(len(availableConnections))
			case <-stop:
				ticker.Stop()
				closeAvailableConnections()
				return
			}
		}
	}()

	b.tree = &Tree{}
	root := &FTPNode{
		tree: b.tree,
		Path: b.RootNodeDirectory,
	}

	b.tree.Root = root

	ts := time.Now()
	b.processDir(b.tree.Root)
	wg.Wait()
	stop <- struct{}{}
	close(stop)

	b.Printf("Время построения дерева: %v\n", time.Since(ts))
	b.Printf("Количество файлов: %d", b.tree.FilesCount())
	b.Printf("Количество папок: %d", b.tree.DirsCount())
	b.writeResultMessage()
}

// CreateTree достраивает дерево от переданного узла
func (b *FTPBuilder) CreateTree(content *FTPNode) {
	defer wg.Done()
	list, err := b.getList(content.Path)
	if err != nil {
		b.Printf("Error on process %s\n", content.Path)
		b.Println("Continueing...")
		content.ErrorText = err.Error()
		return
	}

	if len(list) > 0 {
		content.Children = make([]*FTPNode, len(list), len(list))
	}

	for i, item := range list {
		name, err := NameFromFileSting(item)
		if err != nil {
			b.Printf("Не удалось получить имя из строки списка: %s\nError: %v", item, err)
			continue
		}

		child := &FTPNode{
			Path: path.Join(content.Path, name),
			tree: content.tree,
		}

		if IsDir(child.Name()) {
			b.processDir(child)
			content.Children[i] = child
		} else if IsZip(child.Name()) {
			finded := []FTPNode{}
			b.db.Where("path = ?", child.Path).First(&finded)
			processFile(child)
			if len(finded) == 0 {
				if err = b.db.Create(child).Error; err != nil {
					b.Printf("%v\n", err)
				}

				if err = b.fileToQueue(child.Path); err != nil {
					b.Printf("%v\n", err)
				}
			}
			content.Children[i] = child
		}
	}
}

// Прочитаем с FTP список дочерних узлов
func (b *FTPBuilder) getList(path string) ([]string, error) {
	ftp := getConnection()
	defer func() {
		availableConnections <- ftp
	}()

	start := time.Now()
	iteration := 1
	var list []string
	var err error
	for list, err = ftp.List(path); err != nil; list, err = ftp.List(path) {
		ftp.Close()
		b.Println(err)
		duration := time.Since(start)
		b.Printf("Error on iteration %d process path: %s", iteration, path)
		b.Printf("Timeout v duration: %v", duration)
		b.Println("Process slipping")
		if iteration == 5 {
			return list, err
		}
		time.Sleep(20 * time.Second)
		ftp = b.ftpConnect()
		start = time.Now()
		iteration++
	}

	return list, nil
}

//Обработка листа каталога
func (b *FTPBuilder) processDir(content *FTPNode) {
	content.NodeType = NodeTypeFolder
	content.tree.incrDirs()
	fmt.Println(content.Path)
	wg.Add(1)
	go b.CreateTree(content)
}

//Обработка листа файла
func processFile(content *FTPNode) {
	content.NodeType = NodeTypeArchive
	content.tree.incrFiles()
}

//Пытаемся установить соединение 5 раз с интервалом в 10 секунд,
//если пооытка удачная, то возвращаем установленное соединение
func (b *FTPBuilderConfig) ftpConnect() *goftp.FTP {
	c, err := connect(b.FTPAddr, b.FTPLogin, b.FTPPass)
	if err == nil {
		return c
	}

	for i := 0; i <= 4 && err != nil; i++ {
		time.Sleep(10 * time.Second)
		c, err = connect(b.FTPAddr, b.FTPLogin, b.FTPPass)
		if err == nil {
			return c
		}
	}

	b.Fatalln("Не удается установить ftp соединение: ", err)
	return c
}

// попытка установки соединения и авторизация с ftp
func connect(address, login, pass string) (*goftp.FTP, error) {
	ftp, err := goftp.Connect(address)
	if err != nil {
		return ftp, err
	}
	if login == "" && pass == "" {
		return ftp, err
	}

	if err = ftp.Login(login, pass); err != nil {
		panic("не удалось авторизоваться")
	}
	return ftp, err
}

// Получаем соединение из пула
func getConnection() *goftp.FTP {
	for {
		select {
		case c := <-availableConnections:
			fmt.Println("return connection")
			return c
		}
	}
}

func closeAvailableConnections() {
	for c := range availableConnections {
		c.Close()
	}
	close(availableConnections)
}

func (b *FTPBuilder) fileToQueue(fname string) error {
	conn := b.redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("LPUSH", "DownloadQueue", fname)
	return err
}

func newRedisPool(addr string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", addr, redis.DialDatabase(0))
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
