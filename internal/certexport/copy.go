package certexport

import (
	"os"
	"path/filepath"
)

func copyPEMPair(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return err
	}
	return publishPEMPair(srcDir,dstDir,filepath.Join,pairOps{
		stat:os.Lstat,rename:os.Rename,remove:os.Remove,
		write:writeLocalStage,
	})
}

func removePEMPair(dir string) error {
	for _, name := range []string{certFileName, keyFileName} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) { return err }
	}
	return nil
}

func copyFileAtomic(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(dst), ".fonu-*")
	if err != nil { return err }
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode); err == nil { _, err = f.Write(data) }
	if err == nil { err = f.Sync() }
	closeErr := f.Close()
	if err != nil { return err }
	if closeErr != nil { return closeErr }
	return os.Rename(tmp, dst)
}

func writeLocalStage(path string,data []byte)error{
	f,err:=os.OpenFile(path,os.O_WRONLY|os.O_CREATE|os.O_EXCL,0600)
	if err!=nil{return err}
	_,err=f.Write(data)
	if err==nil{err=f.Sync()}
	closeErr:=f.Close()
	if err==nil{err=closeErr}
	if err!=nil{_ = os.Remove(path)}
	return err
}
