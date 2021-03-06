package builtin

import (
  "bytes"
  "flag"
  "fmt"
  "github.com/easonliao/flea/core"
  "os"
  "os/user"
  "sort"
)

func UsageCommit() {
  usage :=
  `Usage: flea commit [-a] [-m <msg>]

  -a: Tell the command to automatically stage files that have been modified and deleted,
      but new files you have not told Flea  about are not affected.
  -m: Use the given <msg> as the commit message.
  `
  fmt.Println(usage)
  os.Exit(1)
}

func CmdCommit() error {
  if len(os.Args) <= 2 {
    fmt.Println("Not enough arguments.")
    UsageCommit()
  }
  flags := flag.NewFlagSet("commit", 0)
  comment := flags.String("m", "No Comment", "comment")
  all := flags.Bool("a", false, "all")
  flags.Parse(os.Args[2:])

  branch, err := core.GetCurrentBranch()
  if err == core.ErrNotBranch {
    // We're in non-branch, can't commit anything.
    fmt.Println("Can't commit in a non-branch.")
    os.Exit(1)
  }

  indexTree := core.GetIndexTree()

  if *all {
    // -a option is specified, we need to add all the modified/deleted files in working
    // directory to index tree.
    fsTree := core.GetFsTree()
    modifiedMap := make(map[string][]byte)
    deletedPaths := make([]string, 0)
    fn := func(treePath string, node core.Node) error {
      peerNode, err := fsTree.Get(treePath)
      if err == core.ErrPathNotExist {
        // The file has been deleted.
        deletedPaths = append(deletedPaths, treePath)
      } else if !node.IsDir() {
        if bytes.Compare(node.GetHashValue(), peerNode.GetHashValue()) != 0 {
          // The file has been modified.
          modifiedMap[treePath] = peerNode.GetHashValue()
        }
      }
      return nil
    }

    indexTree.Traverse(fn, "/")
    // Sorts the path in descending order so we'll delete files/dirs in reverse order of
    // the namespace hierarchy.
    sort.Sort(sort.Reverse(sort.StringSlice(deletedPaths)))
    for treePath, hash := range(modifiedMap) {
      retHash, err := addFileToStore(treePath)
      if err != nil {
        panic(err.Error())
      }
      if bytes.Compare(retHash, hash) != 0 {
        panic("The hashs don't match")
      }
      if err := indexTree.MkFile(treePath, hash); err != nil {
        panic(err.Error())
      }
    }
    for _, treePath := range(deletedPaths) {
      if err := indexTree.Delete(treePath); err != nil {
        panic(err.Error())
      }
    }
  }

  commit, err := core.GetCurrentCommit()
  if err == nil {
    if bytes.Compare(commit.Tree, indexTree.GetHash()) == 0 {
      // Compares the hash of the commit tree in to the hash of the index tree, if they
      // match then there's nothing to be committed.
      fmt.Println("There's nothing to commit")
      os.Exit(0)
    }
  }

  // Creats a CATree from staging area.
  caTree, err := core.BuildCATreeFromIndexFile()
  if err != nil {
    fmt.Printf("Failed to build CATree from staging area: %s\n", err.Error())
    os.Exit(1)
  }

  // Hash of current commit, or nil if there's no commit in history of current branch.
  var commitHash []byte = nil
  if commit != nil {
    commitHash = commit.GetCommitHash()
  }

  var username string = "unknown"
  if user, err := user.Current(); err == nil {
    username = user.Username
  }
  // Creates a commit object.
  hash, err := core.CreateCommitObject(caTree.GetHash(), commitHash, username, *comment)

  if err != nil {
    fmt.Printf("Failed to create the commit object: %s", err.Error())
    os.Exit(1)
  }

  if _, err := core.GetCurrentBranch(); err == nil {
    // We are in a valid branch, just update the HEAD of the branch.
    core.UpdateBranchHead(branch, hash)
  } else if err == core.ErrNoHeadFile {
    // There's no history and branch. Creates a default master branch and updates its HEAD.
    core.WriteHeadFile([]byte("ref:master"))
    branch = "master"
    core.UpdateBranchHead(branch, hash)
  }
  return nil
}
