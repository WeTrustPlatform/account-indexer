package service

import "sync"

// IpcSubscriber interface for subscribers
type IpcSubscriber interface {
	IpcUpdated(ipc string)
}

// IpcManager has the global IPC, it takes care of informing all fetcher when IPC is changed
type IpcManager struct {
	ipc         string
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
	return im.ipc
}

// ChangeIPC change IPC and inform all subscribers about the change
func (im *IpcManager) ChangeIPC(newIpc string) {
	if im.ipc == newIpc {
		return
	}
	im.ipc = newIpc
	for _, sub := range im.subscribers {
		(*sub).IpcUpdated(im.ipc)
	}
}

// GetIpcManager singleton impl
func GetIpcManager() *IpcManager {
	once.Do(func() {
		ipcManager = &IpcManager{subscribers: []*IpcSubscriber{}}
	})
	return ipcManager
}
