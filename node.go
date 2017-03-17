package ftpTreeBuilder

import (
	"path/filepath"
)

const (
	NodeTypeFolder = 1
	NodeTypeFile   = 2
)

// FTPNode узел с содержимым
type FTPNode struct {
	tree      *Tree
	Path      string
	NodeType  uint
	ErrorText string
	Children  []*FTPNode
}

// Name returns node name
func (n FTPNode) Name() string {
	if n.Path == "/" {
		return n.Path
	}

	_, f := filepath.Split(n.Path)
	return f
}

// Walk Обходит все дерево и выполняет над каждым узлом wf
func (c *FTPNode) Walk(wf func(content *FTPNode) error) error {
	err := wf(c)
	if err != nil {
		return err
	}

	for _, child := range c.Children {
		err = child.Walk(wf)
		if err != nil {
			return err
		}
	}
	return nil
}
