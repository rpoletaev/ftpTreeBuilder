package ftpTreeBuilder

import (
	"github.com/dutchcoders/goftp"
	"path"
	"sync"
	"time"
)

var (
	availableConnections chan *goftp.FTP
	wg                   sync.WaitGroup
)

type Tree struct {
	Root       *FTPNode
	fm         sync.RWMutex
	filesCount int
	dm         sync.RWMutex
	dirsCount  int
}

func (t *Tree) incrFiles() {
	t.fm.Lock()
	defer t.fm.Unlock()
	t.filesCount++
}

func (t *Tree) incrDirs() {
	t.dm.Lock()
	defer t.dm.Unlock()
	t.dirsCount++
}

type FTPBuilder struct {
	*FTPBuilderConfig
}

// BuildTree Строит дерево каталогов
func (b *FTPBuilder) BuildTree() *Tree {
	availableConnections = make(chan *goftp.FTP, b.MaxFTPCons)
	for len(availableConnections) < b.MaxFTPCons {
		availableConnections <- b.ftpConnect()
	}

	stop := make(chan struct{})
	//Периодически выводим количество свободных соединений
	go func(done <-chan struct{}) {
		ticker := time.NewTicker(20 * time.Second)
		for {
			select {
			case <-ticker.C:
				b.Println(len(availableConnections))
			case <-done:
				ticker.Stop()
				closeAvailableConnections()
				return
			}
		}
	}(stop)

	tree := &Tree{}
	root := &FTPNode{
		tree: tree,
		Path: b.RootNodeDirectory,
	}

	tree.Root = root

	ts := time.Now()
	b.processDir(tree.Root)
	wg.Wait()
	stop <- struct{}{}
	close(stop)

	b.Printf("Время выполнения: %v\n", time.Since(ts))
	return tree
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
		}

		if IsDir(child.Name()) {
			b.processDir(child)
			content.Children[i] = child
		} else if IsZip(child.Name()) {
			processFile(child)
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
	wg.Add(1)
	go b.CreateTree(content)
}

//Обработка листа файла
func processFile(content *FTPNode) {
	content.NodeType = NodeTypeFile
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
