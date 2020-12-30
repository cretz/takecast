package chrome

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type patchableFile struct {
	path            string
	fileBytes       []byte
	fileMode        os.FileMode
	certIndex       int
	existingCertLen int
}

type PatchableLib interface {
	Path() string
	Patch(replacementRootCADERBytes []byte) error
}

type UnpatchableLib interface {
	Path() string
	OrigPath() string
	Unpatch() error
}

func FindPatchableLib(startDir string, existingRootCADERBytes []byte) (PatchableLib, error) {
	return findPatchableFile(startDir, existingRootCADERBytes, false)
}

func FindUnpatchableLib(startDir string, existingRootCADERBytes []byte) (UnpatchableLib, error) {
	return findPatchableFile(startDir, existingRootCADERBytes, true)
}

func findPatchableFile(startDir string, existingRootCADERBytes []byte, backup bool) (file *patchableFile, err error) {
	var patchFileMatch func(string) bool
	switch runtime.GOOS {
	case "windows":
		patchFileMatch = func(s string) bool { return strings.HasSuffix(s, "chrome.dll") }
	// TODO: need to test others
	default:
		return nil, fmt.Errorf("OS not supported yet: %v", runtime.GOOS)
	}
	if backup {
		origFileMatch := patchFileMatch
		patchFileMatch = func(s string) bool {
			return strings.HasSuffix(s, ".patched") && origFileMatch(strings.TrimSuffix(s, ".patched"))
		}
	}
	err = filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		} else if file != nil {
			return filepath.SkipDir
		} else if info.IsDir() || !patchFileMatch(path) {
			return nil
		} else if fileBytes, err := ioutil.ReadFile(path); err != nil {
			return err
		} else if certIndex := bytes.Index(fileBytes, existingRootCADERBytes); certIndex != -1 {
			file = &patchableFile{
				path:            path,
				fileBytes:       fileBytes,
				fileMode:        info.Mode(),
				certIndex:       certIndex,
				existingCertLen: len(existingRootCADERBytes),
			}
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil && err != filepath.SkipDir {
		return nil, err
	} else if file == nil {
		return nil, fmt.Errorf("failed finding file with cert to patch")
	}
	return
}

func (p *patchableFile) Path() string     { return p.path }
func (p *patchableFile) OrigPath() string { return strings.TrimSuffix(p.path, ".bak") }

func (p *patchableFile) Patch(replacementRootCADERBytes []byte) error {
	if p.fileBytes == nil {
		return fmt.Errorf("patch cannot be run a second time")
	}
	if p.existingCertLen != len(replacementRootCADERBytes) {
		return fmt.Errorf("replacement byte size %v != existing byte size %v",
			len(replacementRootCADERBytes), p.existingCertLen)
	}
	// First make backup
	if err := ioutil.WriteFile(p.path+".patched", p.fileBytes, p.fileMode); err != nil {
		return fmt.Errorf("failed making backup: %w", err)
	}
	// Now replace the bytes and write
	for i, b := range replacementRootCADERBytes {
		p.fileBytes[p.certIndex+i] = b
	}
	err := ioutil.WriteFile(p.path, p.fileBytes, p.fileMode)
	p.fileBytes = nil
	if err != nil {
		return fmt.Errorf("failed patching file %v", p.path)
	}
	return nil
}

func (p *patchableFile) Unpatch() error {
	// Put bytes back
	if err := ioutil.WriteFile(p.OrigPath(), p.fileBytes, p.fileMode); err != nil {
		return fmt.Errorf("failed overwriting existing file from backup: %w", err)
	}
	// Delete backup
	if err := os.Remove(p.path); err != nil {
		return fmt.Errorf("unpatched, but unable to remove backup: %w", err)
	}
	return nil
}
