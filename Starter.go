package utility

import (
	"time"
)

type Starter struct {
	Interval time.Duration
	Work     func()
	running  bool
	stopChan chan bool
}

func (starter *Starter) Start() {
	starter.running = true
	go starter.run()
}

func (starter *Starter) run() {
	intervalSeconds := int(starter.Interval / time.Second)
	if intervalSeconds < 1 {
		intervalSeconds = 1
	}

	for {
		starter.Work()

		for i := 0; i < intervalSeconds; i++ {
			time.Sleep(time.Second)
			if !starter.running {
				break
			}
		}
		if !starter.running {
			break
		}
	}

	if starter.stopChan != nil {
		starter.stopChan <- true
		starter.stopChan = nil
	}
}

func (starter *Starter) Stop() {
	starter.stopChan = make(chan bool)
	starter.running = false
}

func (starter *Starter) Wait() {
	<-starter.stopChan
	starter.stopChan = nil
}
