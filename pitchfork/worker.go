package main

import (
	log "github.com/golang/glog"
	"net/http"
	"os"
	"time"
)


func Work(p *Pitchfork) {
	var (
		stores             StoreList
		pitchforks         PitchforkList
		storeChanges       <-chan zk.Event
		pitchforkChanges   <-chan zk.Event
		allStores          map[string]StoreList
		stopper            chan struct{}
		store              *Store
		err                error
	)
	for {
		stores, storeChanges, err = p.WatchGetStores()
		if err != nil {
			log.Errorf("WatchGetStores() called error(%v)", err)
			return
		}

		pitchforks, pitchforkChanges, err = p.WatchGetPitchfork()
		if err != nil {
			log.Errorf("WatchGetPitchfork() called error(%v)", err)
			return
		}

		allStores, err = divideStoreBetweenPitchfork(pitchforks, stores)

		stopper = make(chan struct{})

		for _, store = range allStores[p.ID] {
			go func(stopper chan struct{}) {
				for {
					if err = p.probeStore(store); err != nil {
						log.Errorf("probeStore() called error(%v)", err)
					}
					select {
						case <- stopper:
							break
						case <- time.After(p.config.ProbeInterval * time.Second):
					}
				}
			}(stopper)
		}


		select {
		case <-p.stopper:
			close(stopper)
			return

		case <-storeChanges:
			log.Infof("Triggering rebalance due to store list change")
			close(stopper)

		case <-pitchforkChanges:
			log.Infof("Triggering rebalance due to pitchfork list change")
			close(stopper)
		}
	}
}


