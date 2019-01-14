package service

import (
	"errors"
	"log"
	"sync"
)

// IpcSubscriber interface for subscribers
type IpcSubscriber interface {
	IpcUpdated(ipc string)
}

// IpcManager has the global IPC, it takes care of informing all fetcher when IPC is changed
type IpcManager struct {
	ipcList     []string
	curIPC      string
	subscribers []*IpcSubscriber
}

var ipcManager *IpcManager
var once sync.Once

// Subscribe method for a subscriber to call
func (im *IpcManager) Subscribe(sub *IpcSubscriber) {
	im.subscribers = append(im.subscribers, sub)
}

// GetIPC getter for ipc
func (im *IpcManager) GetIPC() string {
	return im.curIPC
}

// SetIPC gets called from main
func (im *IpcManager) SetIPC(ipcs []string) error {
	if ipcs == nil || len(ipcs) == 0 {
		return errors.New("No IPC specified")
	}
	tmp := map[string]string{}
	for _, ipc := range ipcs {
		if ipc == "" {
			return errors.New("Blank ipc")
		}
		if tmp[ipc] != "" {
			return errors.New("Duplicate IPC: " + ipc)
		}
		tmp[ipc] = ipc
	}
	im.ipcList = ipcs
	im.curIPC = im.ipcList[0]
	log.Printf("Initial ipc: %v \n", im.curIPC)
	return nil
}

// ChangeIPC change IPC and inform all subscribers about the change
func (im *IpcManager) ChangeIPC() {
	log.Println("Attempt to change IPC...")
	if len(im.ipcList) <= 1 {
		log.Printf("Cannot change ipc because there is only %v ipc provided \n", im.ipcList)
		return
	}
	im.curIPC = NextIPC(im.curIPC, im.ipcList)
	log.Printf("New IPC: %v \n", im.curIPC)
	for _, sub := range im.subscribers {
		(*sub).IpcUpdated(im.curIPC)
	}
}

// NextIPC get next ipc from a list
func NextIPC(curIPC string, ipcList []string) string {
	index := 0
	for i, ipc := range ipcList {
		if ipc == curIPC {
			index = i
			break
		}
	}
	newIndex := (index + 1) % len(ipcList)
	return ipcList[newIndex]
}

// GetIpcManager singleton impl
func GetIpcManager() *IpcManager {
	once.Do(func() {
		ipcManager = &IpcManager{subscribers: []*IpcSubscriber{}}
	})
	return ipcManager
}
