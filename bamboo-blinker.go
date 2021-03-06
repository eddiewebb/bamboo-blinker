package main

import (
	"fmt"
	"github.com/davidehringer/bamboo-blinker/bunny"
	"github.com/davidehringer/goblync"
	"os"
	"os/signal"
	"syscall"
	"time"
	"strconv"
)

func main() {



	numArgs := len(os.Args)
	if numArgs < 2 {
			fmt.Println("Usage: bamboo-blinker URL [INTERVAL_SECONDS] [BACKOFF_MS] [BLYNC_ID]")
			fmt.Printf("\t This will start monitoring the provided URL using a specific BlyncLight, default to ID 1\n")
			fmt.Println("OR : bamboo-blinker TellMeHowItWillBe")
			fmt.Printf("\t This will flash all connect blync lights in logical ID order based on USB port\n")
			os.Exit(1)
	}

	light := blync.NewBlyncLight()

	if os.Args[1] == "FlashMob"{
			light.FlashOrder()
			os.Exit(1)
	}
	url := os.Args[1]
	buildBunny := bunny.NewBunny(url)

	defaultInterval := 10
	if numArgs == 3 {
		value, err := strconv.ParseInt(os.Args[2], 10, 32)
		if err != nil {
			fmt.Println("INTERVAL_SECONDS must be an integer")
			os.Exit(1)
		}
		defaultInterval = int(value)
	}
	activeInterval := defaultInterval

	backoffLimit := 100
	if numArgs == 4 {
		value, err := strconv.ParseInt(os.Args[3], 10, 32)
		if err != nil {
			fmt.Println("BACKOFF_MS must be an integer")
			os.Exit(1)
		}
		backoffLimit = int(value)
	}

	blyncId := 1
	if numArgs == 5 {
		value, err := strconv.ParseInt(os.Args[4], 10, 32)
		if err != nil {
			fmt.Println("BLYNC_ID must be an integer")
			os.Exit(1)
		}
		blyncId = int(value)
	}


	light.SetColor(blync.Green, blyncId)

	// clean shutdown of light
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		light.Close(blyncId)
		os.Exit(1)
	}()

	monitor := NewMonitor(light, blyncId)

	for {
		bunnyStatus := buildBunny.Update()

		//backoff if more than backoffLimit ms to calculate this on server side.
		if bunnyStatus.ProcessTime > backoffLimit{
			activeInterval = activeInterval * 2;	
			if activeInterval > 600 {
				activeInterval = 600
			}		
			fmt.Printf("Interval increased to %d seconds since TimeToEvaluate was %d ms\n" , activeInterval,bunnyStatus.ProcessTime)
		}else{

			if(defaultInterval != activeInterval){
				activeInterval = defaultInterval;
				fmt.Printf("Interval reset to %d seconds since TimeToEvaluate was %d ms\n" , activeInterval,bunnyStatus.ProcessTime)
			}			
		}

		if bunnyStatus.Status == "OK" {
			monitor.SetHealthy();
		}else{
			monitor.SetUnhealthy();			
		}
		time.Sleep(time.Second * time.Duration(activeInterval))
	}
}

type monitor struct {
	healthy bool
	light blync.BlyncLight
	blyncId int
}

func NewMonitor(light blync.BlyncLight, blyncId int) (m monitor) {
	m.healthy = true;
	m.light = light
	m.blyncId = blyncId
	return
}

func (m *monitor) SetHealthy() {
	if !m.healthy {
		m.healthy = true
		m.light.StopPlay(m.blyncId)
		m.light.SetBlinkRate(blync.BlinkOff, m.blyncId)
		for i := 0; i < 256; i++ {
			m.light.SetColor([3]byte{255 - byte(i), byte(i), 0x00}, m.blyncId)
			time.Sleep(13 * time.Millisecond)
		}
		m.light.Play(28, m.blyncId)
	}
}

func (m *monitor) SetUnhealthy() {
	if m.healthy {
		m.healthy = false
		for i := 0; i < 256; i++ {
			m.light.SetColor([3]byte{byte(i), 255 - byte(i), 0x00}, m.blyncId)
			time.Sleep(13 * time.Millisecond)
		}
		m.light.SetBlinkRate(blync.BlinkMedium, m.blyncId)
		m.light.Play(52, m.blyncId)
		// We using a never ending sound because it was one of the only ones that 
		// had some sound of urgency to it.  But we don't want it to keep playing
		time.Sleep(time.Second * 15)
		m.light.StopPlay(m.blyncId)
	}
}
