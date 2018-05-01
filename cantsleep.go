package main

import (
	"fmt"
	"strconv"
	"syscall"

	"github.com/caseymrm/go-assertions"
	"github.com/caseymrm/go-statusbar/tray"
)

var sleepKeywords = map[string]bool{
	"PreventUserIdleDisplaySleep": true,
	//"PreventUserIdleSystemSleep":  true,
}
var canSleepTitle = "😴"
var cantSleepTitle = "😫"

func canSleep() bool {
	asserts := assertions.GetAssertions()
	for key, val := range asserts {
		if val == 1 && sleepKeywords[key] {
			return false
		}
	}
	return true
}

func menuItems() []tray.MenuItem {
	items := make([]tray.MenuItem, 0, 1)
	pidAsserts := assertions.GetPIDAssertions()
	for key := range sleepKeywords {
		pids := pidAsserts[key]
		for _, pid := range pids {
			items = append(items, tray.MenuItem{
				Text:     fmt.Sprintf("%s (pid %d)", pid.Name, pid.PID),
				Callback: fmt.Sprintf("%d", pid.PID),
			})
		}
	}
	preAmble := []tray.MenuItem{{Text: "Your laptop can sleep!"}}
	if len(items) == 1 {
		preAmble = []tray.MenuItem{{Text: "1 process is keeping your laptop awake:"}}
	} else if len(items) > 1 {
		preAmble = []tray.MenuItem{{Text: fmt.Sprintf("%d processes are keeping your laptop awake:", len(items))}}
	}
	if len(items) > 0 {
		preAmble = append(preAmble, tray.MenuItem{Text: "---"})
	}
	return append(preAmble, items...)
}

func menuState() *tray.MenuState {
	if canSleep() {
		return &tray.MenuState{
			Title: canSleepTitle,
		}
	}
	return &tray.MenuState{
		Title: cantSleepTitle,
	}
}

func monitorAssertionChanges(channel chan assertions.AssertionChange) {
	for change := range channel {
		if sleepKeywords[change.Type] {
			tray.App().SetMenuState(menuState())
		}
	}
}

func handleClicks(callback chan string) {
	for pidString := range callback {
		pid, _ := strconv.Atoi(pidString)
		go func() {
			switch tray.App().Alert("Kill process?", fmt.Sprintf("PID %d", pid), "Kill", "Kill -9", "Cancel") {
			case 0:
				fmt.Printf("Killing pid %d\n", pid)
				syscall.Kill(pid, syscall.SIGTERM)
			case 1:
				fmt.Printf("Killing -9 pid %d\n", pid)
				syscall.Kill(pid, syscall.SIGKILL)
			}
		}()
	}
}

func main() {
	assertionsChannel := make(chan assertions.AssertionChange)
	trayChannel := make(chan string)
	assertions.SubscribeAssertionChanges(assertionsChannel)
	go monitorAssertionChanges(assertionsChannel)
	app := tray.App()
	app.SetMenuState(menuState())
	app.Clicked = trayChannel
	app.MenuOpened = func() []tray.MenuItem {
		return menuItems()
	}
	go handleClicks(trayChannel)
	app.RunApplication()
}