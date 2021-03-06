package core

import (
  "path/filepath"
)

var indexTree *IndexTree = nil

// IndexTree represents the tree structure stored in index file. All the files
// in staging area are in index file.
type IndexTree struct {
  indexFile string
  memTree *MemTree
}

// Gets the singleton IndexTree instance.
func GetIndexTree() *IndexTree {
  if indexTree == nil {
    var err error
    indexTree, err = newIndexTree(filepath.Join(GetFleaDirectory(),"index"))
    if err != nil {
      panic("Can create index tree.")
    }
  }
  return indexTree
}

func newIndexTree(filePath string) (*IndexTree, error) {
  var err error
  if exists(filePath) {
    // The index file already exists.
    data, _ := read(filePath)
    // Restores the data to MemTree.
    memTree, err := Deserialize(data)
    if err != nil {
      return nil, err
    }
    return &IndexTree{filePath, memTree}, nil
  } else {
    // The index file doesn't exist.
    memTree := NewMemTree()
    tree := &IndexTree{filePath, memTree}
    err = tree.flush()
    return tree, err
  }
}

// Gets the node for a given path.
func (tree *IndexTree) Get(treePath string) (Node, error) {
  return tree.memTree.Get(treePath)
}

// Traverse the tree structure. MemTree traverses the tree in DFS way.
func (tree *IndexTree) Traverse(fn VisitFn, root string) error {
  return tree.memTree.Traverse(fn, root)
}

// See Tree interface.
func (tree *IndexTree) GetHash() []byte {
  return tree.memTree.root.GetHashValue()
}

// Creates a directory in tree.
func (tree *IndexTree) MkDir(treePath string) (err error) {
  err = tree.memTree.MkDir(treePath)
  if err == nil {
    err = tree.flush()
  }
  return
}

// MkDirAll creates a directory named path, along with any necessary parents.
func (tree *IndexTree) MkDirAll(treePath string) (err error) {
  err = tree.memTree.MkDirAll(treePath)
  if err == nil {
    err = tree.flush()
  }
  return
}

// Creates a file with given hash value in tree. If the file exists then update the file.
func (tree *IndexTree) MkFile(treePath string, hash []byte) (err error) {
  err = tree.memTree.MkFile(treePath, hash)
  if err == nil {
    err = tree.flush()
  }
  return
}

// MkFileAll creates a file with given path and hash value, along with any necessary parents.
func (tree *IndexTree) MkFileAll(treePath string, hash []byte) (err error) {
  err = tree.memTree.MkFileAll(treePath, hash)
  if err == nil {
    err = tree.flush()
  }
  return
}

// Deletes a node from the tree. If the node is a directory the whole directory will be
// deleted.
func (tree *IndexTree) Delete(treePath string) (err error) {
  err = tree.memTree.Delete(treePath)
  if err == nil {
    err = tree.flush()
  }
  return
}

// See MemTree.
func (tree *IndexTree) Clear() {
  tree.memTree.Clear()
  tree.flush()
}

// Fluses the in-memory data of index tree to index file.
func (tree *IndexTree) flush() error {
  data, err := tree.memTree.Serialize()
  if err == nil {
    err = write(tree.indexFile, data)
  }
  return err
}
