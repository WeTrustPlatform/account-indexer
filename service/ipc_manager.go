package service

import (
	"errors"
	"log"
	"strconv"
	"sync"
	"sync/atomic"
)

// IpcSubscriber interface for subscribers
type IpcSubscriber interface {
	IpcUpdated(ipc string)
	Name() string
}

// IpcManager has the global IPC, it takes care of informing all fetcher when IPC is changed
type IpcManager struct {
	ipcList          []string
	curIPC           string
	subscribers      []*IpcSubscriber
	switchIPCCounter int32
}

var ipcManager *IpcManager
var once sync.Once

// Subscribe method for a subscriber to call
func (im *IpcManager) Subscribe(sub *IpcSubscriber) {
	for _, oldSub := range im.subscribers {
		if oldSub == sub {
			log.Printf("IpcManager: Already has the subscription for %v, no need to subscribe again \n", (*sub).Name())
			break
		}
	}
	im.subscribers = append(im.subscribers, sub)
}

// GetIPC getter for ipc
func (im *IpcManager) GetIPC() string {
	return im.curIPC
}

// SetIPC gets called from main
func (im *IpcManager) SetIPC(ipcs []string) error {
	if len(ipcs) == 0 {
		return errors.New("no ipc specified")
	}
	tmp := map[string]string{}
	for _, ipc := range ipcs {
		if ipc == "" {
			return errors.New("blank ipc")
		}
		if tmp[ipc] != "" {
			return errors.New("duplicate IPC: " + ipc)
		}
		tmp[ipc] = ipc
	}
	im.ipcList = ipcs
	im.curIPC = im.ipcList[0]
	log.Printf("IpcManager: Initial ipc: %v \n", im.curIPC)
	return nil
}

// ForceChangeIPC force change ipc, otherwise log.Fatal
func (im *IpcManager) ForceChangeIPC() {
	if len(im.ipcList) <= 1 {
		log.Fatal("IpcManager: Cannot switch IPC, number of IPC = " + strconv.Itoa(len(im.ipcList)))
		return
	}
	im.ChangeIPC()
}

// EnableSwitchIPC enable ForceChangeIPC
func (im *IpcManager) EnableSwitchIPC() {
	log.Printf("IpcManager: EnableSwitchIPC")
	atomic.StoreInt32(&im.switchIPCCounter, 0)
}

// ChangeIPC change IPC and inform all subscribers about the change
func (im *IpcManager) ChangeIPC() {
	atomic.AddInt32(&im.switchIPCCounter, 1)
	counter := atomic.LoadInt32(&im.switchIPCCounter)
	if counter > 1 {
		log.Printf("IpcManager: Switching is in progress, counter=%v ... \n", counter)
		return
	}
	log.Println("IpcManager: Attempt to change IPC...")
	if len(im.ipcList) <= 1 {
		log.Printf("IpcManager: Cannot change ipc because there is only %v ipc provided \n", im.ipcList)
		return
	}
	im.curIPC = NextIPC(im.curIPC, im.ipcList)
	log.Printf("IpcManager: New IPC: %v, number of subscriber to update: %v \n", im.curIPC, len(im.subscribers))
	for _, sub := range im.subscribers {
		log.Printf("IpcManager: Updating new ipc to %v \n", (*sub).Name())
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
