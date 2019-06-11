package service

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
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
			log.WithField("subscription", (*sub).Name()).Info("IpcManager: Already has the subscription no need to subscribe again")
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
	log.WithField("ipc", im.curIPC).Info("IpcManager: Initial ipc")
	return nil
}

// ForceChangeIPC force change ipc, otherwise panic
func (im *IpcManager) ForceChangeIPC() {
	if len(im.ipcList) <= 1 {
		panic(errors.New("IpcManager: Cannot switch IPC, number of IPC = " + strconv.Itoa(len(im.ipcList))))
	}
	im.ChangeIPC()
}

// EnableSwitchIPC enable ForceChangeIPC
func (im *IpcManager) EnableSwitchIPC() {
	log.Info("IpcManager: EnableSwitchIPC")
	atomic.StoreInt32(&im.switchIPCCounter, 0)
}

// ChangeIPC change IPC and inform all subscribers about the change
func (im *IpcManager) ChangeIPC() {
	atomic.AddInt32(&im.switchIPCCounter, 1)
	counter := atomic.LoadInt32(&im.switchIPCCounter)
	if counter > 1 {
		log.WithField("counter", counter).Info("IpcManager: Switching is in progress")
		return
	}
	log.Info("IpcManager: Attempt to change IPC...")
	if len(im.ipcList) <= 1 {
		log.WithField("ipcList", im.ipcList).Info("IpcManager: Cannot change ipc")
		return
	}
	im.curIPC = NextIPC(im.curIPC, im.ipcList)
	log.WithFields(log.Fields{
		"curIPC":         im.curIPC,
		"numSubscribers": len(im.subscribers),
	}).Info("IPCManager: ChangeIPC")
	for _, sub := range im.subscribers {
		log.WithField("ipc", (*sub).Name()).Info("IpcManager: Updating new ipc")
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
