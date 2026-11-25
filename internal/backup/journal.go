package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/fonu/fonu/internal/secret"
)

const restoreRollbackName = ".restore-rollback"
const restoreJournalName = ".restore-journal.json"

type restoreMove struct { From, To string }
type restoreJournal struct {
	Moves []restoreMove
	Committed bool
}

func syncDir(path string) error {
	if runtime.GOOS == "windows" { return nil }
	f,err := os.Open(path); if err != nil { return err }
	defer f.Close()
	return f.Sync()
}

func saveRestoreJournal(dir string, j restoreJournal) error {
	raw,err := json.Marshal(j); if err != nil { return err }
	f,err := os.CreateTemp(dir,".restore-journal-*"); if err != nil { return err }
	defer os.Remove(f.Name())
	_,err = f.Write(raw)
	if err == nil { err = f.Sync() }
	closeErr := f.Close(); if err == nil { err = closeErr }; if err != nil { return err }
	if err := os.Rename(f.Name(),filepath.Join(dir,restoreJournalName)); err != nil { return err }
	return syncDir(dir)
}

func moveExists(path string) (bool,error) {
	_,err := os.Lstat(path)
	if os.IsNotExist(err) { return false,nil }
	return err == nil,err
}

func syncMove(m restoreMove) error {
	return errors.Join(syncDir(filepath.Dir(m.From)),syncDir(filepath.Dir(m.To)))
}

func validRestoreMove(dir string,m restoreMove) bool {
	for _,name := range []string{"fonu.db-wal","fonu.db-shm","fonu.db","certs","nginx",secret.KeyFileName} {
		live := filepath.Join(dir,name)
		if m.From == live && m.To == filepath.Join(dir,restoreRollbackName,name) { return true }
		if m.To == live && m.From == filepath.Join(dir,pendingDirName,name) { return true }
	}
	return false
}

// Undo the last recorded intent first. A journal entry is durable before its
// rename; either the source or destination must exist when that intent is undone.
func recoverRestore(dir string) error {
	path := filepath.Join(dir,restoreJournalName)
	raw,err := os.ReadFile(path)
	if os.IsNotExist(err) { return nil }; if err != nil { return err }
	var j restoreJournal
	if err := json.Unmarshal(raw,&j); err != nil { return err }
	for _,m := range j.Moves { if !validRestoreMove(dir,m) { return fmt.Errorf("invalid restore journal move") } }
	if !j.Committed {
		for len(j.Moves)>0 {
			m := j.Moves[len(j.Moves)-1]
			from,err := moveExists(m.From); if err != nil { return err }
			to,err := moveExists(m.To); if err != nil { return err }
			switch {
			case !from && to:
				if err := os.Rename(m.To,m.From); err != nil { return err }
				if err := syncMove(m); err != nil { return err }
			case from && !to:
				if err := syncMove(m); err != nil { return err }
			default: return fmt.Errorf("ambiguous interrupted restore; preserve recovery files")
			}
			j.Moves=j.Moves[:len(j.Moves)-1]
			if err := saveRestoreJournal(dir,j); err != nil { return err }
		}
	} else {
		if err := os.RemoveAll(filepath.Join(dir,pendingDirName)); err != nil { return err }
	}
	if err := os.RemoveAll(filepath.Join(dir,restoreRollbackName)); err != nil { return err }
	if err := syncDir(dir); err != nil { return err }
	if err := os.Remove(path); err != nil { return err }
	return syncDir(dir)
}

func swapRestoreFiles(dir,pending string,rename func(string,string) error) (err error) {
	rollback := filepath.Join(dir,restoreRollbackName)
	if err := os.Mkdir(rollback,0700); err != nil { return err }
	j := restoreJournal{}
	if err := saveRestoreJournal(dir,j); err != nil { return err }
	defer func() {
		if err != nil { err = errors.Join(err,recoverRestore(dir)) }
	}()
	move := func(from,to string) error {
		m := restoreMove{From:from,To:to}
		j.Moves=append(j.Moves,m)
		if err := saveRestoreJournal(dir,j); err != nil { return err }
		if err := rename(from,to); err != nil { return err }
		return syncMove(m)
	}
	for _,name := range []string{"fonu.db-wal","fonu.db-shm","fonu.db","certs","nginx",secret.KeyFileName} {
		src,live := filepath.Join(pending,name),filepath.Join(dir,name)
		staged,err := moveExists(src); if err != nil { return err }
		sidecar := name=="fonu.db-wal" || name=="fonu.db-shm"
		if !staged && !sidecar { continue }
		exists,err := moveExists(live); if err != nil { return err }
		if exists { if err := move(live,filepath.Join(rollback,name)); err != nil { return err } }
		if staged && !sidecar { if err := move(src,live); err != nil { return err } }
	}
	j.Committed=true
	if err := saveRestoreJournal(dir,j); err != nil { return err }
	return recoverRestore(dir)
}
