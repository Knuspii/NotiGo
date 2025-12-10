package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/gen2brain/beeep"
	"github.com/shirou/gopsutil/v4/net"
)

var blocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

const (
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Cyan    = "\033[36m"
	Reset   = "\033[0m"
	Version = "NotiGo v0.1"
)

func speedToBlock(speed, max float64) rune {
	idx := int((speed / max) * float64(len(blocks)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(blocks) {
		idx = len(blocks) - 1
	}
	return blocks[idx]
}

func clearScreen() {
	fmt.Print("\033[2J")
	fmt.Print("\033[H")
}

func printLine(termWidth int) {
	fmt.Printf("+%s~%s\n", strings.Repeat("-", termWidth), Reset)
}

func triggerBeep() {
	beeep.Beep(500, 400)
	beeep.Alert("NotiGo", "Download finished", "")
	time.Sleep(300 * time.Millisecond)
	beeep.Beep(500, 400)
}

func renderUI(
	termWidth int,
	autoDetect bool,
	statusWord string,
	statusColor string,
	displaySpeed float64,
	graphStr string,
) {
	fmt.Print("\033]0;NotiGo\007")
	clearScreen()
	printLine(termWidth)
	fmt.Printf("| %s┳┓   •┏┓  %s\n", Cyan, Reset)
	fmt.Printf("| %s┃┃┏┓╋┓┃┓┏┓%s\n", Cyan, Reset)
	fmt.Printf("| %s┛┗┗┛┗┗┗┛┗┛%s %s%s%s\n", Cyan, Reset, Yellow, Version, Reset)
	printLine(termWidth)
	fmt.Printf("| AutoDetect: %v\n", autoDetect)
	fmt.Printf("| Download:   %s%s%s\n", statusColor, statusWord, Reset)
	fmt.Printf("| Speed:      %.0f KB/s\n", displaySpeed)
	printLine(termWidth)
	fmt.Printf("| Graph: [%s%s%s]\n", Yellow, graphStr, Reset)
	printLine(termWidth)
	fmt.Printf("| [Q] Quit | [S] Toggle AutoDetect\n")
	printLine(termWidth)
}

func main() {
	const termWidth = 35
	const maxGraphSpeed = 10000.0

	beepEnabled := flag.Bool("b", false, "manual beep on start")
	versionFlag := flag.Bool("v", false, "show version")
	help := flag.Bool("h", false, "help")
	refreshRate := flag.Int("r", 3, "refresh rate in seconds")
	thresholdFlag := flag.Int("t", 300000, "download threshold in bytes")
	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		return
	}

	if *help {
		fmt.Println("NotiGo flags")
		fmt.Println("-b  manual beep")
		fmt.Println("-v  show version")
		fmt.Println("-r  refresh rate in seconds")
		fmt.Println("-t  threshold in bytes")
		fmt.Println("-h  help")
		return
	}

	refreshInterval := time.Duration(*refreshRate) * time.Second
	threshold := *thresholdFlag

	if *beepEnabled {
		triggerBeep()
		fmt.Printf("NotiGo Beep, finished!\n")
		os.Exit(0)
	}

	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer keyboard.Close()

	keyEvents, _ := keyboard.GetKeys(10)

	prev, err := net.IOCounters(false)
	if err != nil || len(prev) == 0 {
		panic("Cannot get network stats")
	}

	autoDetect := true
	downloading := false
	finished := false
	graphData := make([]float64, 0, termWidth-6)

	renderTicker := time.NewTicker(refreshInterval)
	defer renderTicker.Stop()

	inputTicker := time.NewTicker(50 * time.Millisecond)
	defer inputTicker.Stop()

	renderUI(termWidth, autoDetect, "Loading...", Red, 0, "")

loop:
	for {
		select {

		case <-inputTicker.C:
			select {
			case ev := <-keyEvents:
				if ev.Err != nil {
					continue
				}
				if ev.Rune == 'q' || ev.Key == keyboard.KeyCtrlC {
					break loop
				}
				if ev.Rune == 's' {
					autoDetect = !autoDetect
				}
			default:
			}

		case <-renderTicker.C:

			cur, err := net.IOCounters(false)
			if err != nil || len(cur) == 0 {
				continue
			}

			delta := cur[0].BytesRecv - prev[0].BytesRecv
			prev = cur

			speed := (float64(delta) / 1024) / refreshInterval.Seconds()
			if speed < 0 {
				speed = 0
			}

			displaySpeed := speed
			if displaySpeed > 99999 {
				displaySpeed = 99999
			}

			graphData = append(graphData, speed)
			if len(graphData) > termWidth-10 {
				graphData = graphData[1:]
			}

			graphStr := ""
			for _, v := range graphData {
				graphStr += string(speedToBlock(v, maxGraphSpeed))
			}

			statusWord := "Idle"
			statusColor := Yellow

			if autoDetect {
				if speed > float64(threshold)/1024 {
					downloading = true
					finished = false
					statusWord = "ACTIVE"
					statusColor = Green
				} else if speed < float64(threshold)/1024 && downloading {
					downloading = false
					if !finished {
						finished = true
						statusWord = "FINISHED"
						statusColor = Red
						triggerBeep()
					}
				}
			}

			renderUI(
				termWidth,
				autoDetect,
				statusWord,
				statusColor,
				displaySpeed,
				graphStr,
			)
		}
	}

	fmt.Println("Exiting NotiGo.")
	os.Exit(0)
}
