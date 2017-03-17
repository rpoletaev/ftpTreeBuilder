package ftpTreeBuilder

import (
	"sync"
)

// Tree Структура описывающая структуру каталогов на FTP сервере
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

// FilesCount Возвращает общее количество Архивов
func (t *Tree) FilesCount() int {
	t.fm.RLock()
	defer t.fm.RUnlock()
	return t.filesCount
}

// DirsCount Возвращает общее количество каталогов
func (t *Tree) DirsCount() int {
	t.dm.RLock()
	defer t.dm.RUnlock()
	return t.dirsCount
}
