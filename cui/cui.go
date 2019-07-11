// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cui

import (
	"fmt"
	"github.com/jroimartin/gocui"
	"log"
	"strings"
	"time"
)

var CurUser string

var LoginUser = make(chan string)
var LoginPass = make(chan string)
var LoginStatus = make(chan string)

var RegUser = make(chan string)
var RegPass = make(chan string)
var RegStatus = make(chan string)

var GetFriend = make(chan bool)
var CurFriendList = make(chan []string)
var Curfriend string

var ChatUser = make(chan string)
var Chatmsg = make(chan string)

var RecvUser = make(chan string)
var RecvMsg = make(chan string)

var OnlineUser = make(chan string)

func nextViewLogin(g *gocui.Gui, v *gocui.View) error {
	if v == nil || v.Name() == "Username" {
		_, err := g.SetCurrentView("Password")
		return err
	}
	_, err := g.SetCurrentView("Username")
	return err
}

func nextViewReg(g *gocui.Gui, v *gocui.View) error {
	switch v.Name() {
	case "UsernameR":
		_, err := g.SetCurrentView("PasswordR")
		return err
	case "PasswordR":
		_, err := g.SetCurrentView("ConfirmR")
		return err
	case "ConfirmR":
		_, err := g.SetCurrentView("UsernameR")
		return err
	}
	return nil
}

func nextViewMsg(g *gocui.Gui, v *gocui.View) error {
	switch v.Name() {
	case "textMsg":
		_, err := g.SetCurrentView("FriendList")
		return err
	case "FriendList":
		_, err := g.SetCurrentView("textMsg")
		return err
	}
	return nil
}

func LoginEnter(g *gocui.Gui, v *gocui.View) error {
	vUsername, err := g.View("Username")

	if err != nil {
		panic(err)
	}
	username := strings.TrimSpace(vUsername.Buffer())
	//fmt.Println("username: "+ username)
	vPassword, err := g.View("Password")
	if err != nil {
		panic(err)
	}
	password := strings.TrimSpace(vPassword.Buffer())
	LoginUser <- username
	LoginPass <- password
	//fmt.Println("password: "+password)
	loginStatus := <-LoginStatus
	if loginStatus == "ok" {
		CurUser = username
		LoginOK(g)
	} else {
		fmt.Fprintln(v, loginStatus)
	}
	return err
}

func updateMsg(recvuser string, g *gocui.Gui) {
	recvmsg := <-RecvMsg
	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("msgBox")
		if err != nil {
			panic(err)
		}
		Msgtime := time.Now().String()[:19]
		fmt.Fprintf(v, "%s %s:\n   %s\n", recvuser, Msgtime, recvmsg)
		for _, _ = range v.BufferLines() {
			cursorDown(g, v)
		}
		return nil
	})
}

func updateOnline(onlineuser string, g *gocui.Gui) {
	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("FriendList")
		if err != nil {
			panic(err)
		}
		friendlist := strings.Split(v.Buffer(), "\n")
		newFriendlist := []string{"\u001b[32m" + onlineuser + "\u001b[0m"}
		for _, n := range friendlist {
			if n != onlineuser {
				newFriendlist = append(newFriendlist, n)
			}
		}
		return nil
	})
}

func updateFriend(friendList []string, g *gocui.Gui) {
	g.Update(func(g *gocui.Gui) error {
		v, err := g.View("FriendList")
		if err != nil {
			panic(err)
		}
		v.Clear()
		for _, singleName := range friendList {
			fmt.Fprintln(v, singleName)
		}
		return nil
	})
}

func msgUpdate(g *gocui.Gui) {
	for {
		select {
		case recvuser := <-RecvUser:
			go updateMsg(recvuser, g)
		case onlineuser := <-OnlineUser:
			go updateOnline(onlineuser, g)
		case friendList := <-CurFriendList:
			go updateFriend(friendList, g)
		}
	}
}

func LoginOK(g *gocui.Gui) error {

	maxX, maxY := g.Size()
	if v, err := g.SetView("FriendList", -1, -1, 18, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = true
		v.Editable = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack

		GetFriend <- true
		friendList := <-CurFriendList
		for _, singleName := range friendList {
			fmt.Fprintln(v, singleName)
		}
	}
	if v, err := g.SetView("msgBox", 19, -1, maxX, maxY-10); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = false
		v.SelBgColor = gocui.ColorWhite
		v.SelFgColor = gocui.ColorBlack

	}
	if v, err := g.SetView("textMsg", 19, maxY-9, maxX, maxY); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
	}
	if _, err := g.SetCurrentView("FriendList"); err != nil {
		return err
	}
	go msgUpdate(g)
	return nil
}

func RegEnter(g *gocui.Gui, v *gocui.View) error {
	vUsername, err := g.View("UsernameR")
	vPassword, err := g.View("PasswordR")
	vConfirm, err := g.View("ConfirmR")
	if err != nil {
		panic(err)
	}
	username := strings.TrimSpace(vUsername.Buffer())
	password := strings.TrimSpace(vPassword.Buffer())
	confirm := strings.TrimSpace(vConfirm.Buffer())
	if password == confirm {
		RegUser <- username
		RegPass <- password
	}
	if <-RegStatus == "ok" {
		err = ReturnMain(g, v)
	}

	return err
}

func ReturnMain(g *gocui.Gui, v *gocui.View) error {
	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	return nil
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		if l, err := v.Line(cy + 1); err == nil {
			if l == "" {
				return err
			}
		} else {
			return err
		}
		if err := v.SetCursor(cx, cy+1); err != nil {
			ox, oy := v.Origin()
			if err := v.SetOrigin(ox, oy+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func getLine(g *gocui.Gui, v *gocui.View) error {
	var l string
	var err error

	_, cy := v.Cursor()
	if l, err = v.Line(cy); err != nil {
		l = ""
	}
	_, err = g.SetCurrentView("textMsg")
	if err != nil {
		panic(err)
	}
	Curfriend = l

	return nil
}

func sendMsg(g *gocui.Gui, v *gocui.View) error {

	msg := v.Buffer()
	if msg == "" {
		return nil
	}
	ChatUser <- Curfriend
	Chatmsg <- msg
	vMsgBox, err := g.View("msgBox")
	Msgtime := time.Now().String()[:19]
	fmt.Fprintf(vMsgBox, "%s %s:\n   %s\n", CurUser, Msgtime, msg)
	vMsgBox.MoveCursor(0, len(v.BufferLines()), false)
	v.Clear()
	v.SetCursor(0, 0)
	v.SetOrigin(0, 0)
	return err
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func keybindings(g *gocui.Gui) error {
	if err := g.SetKeybinding("main", gocui.KeyCtrlL, gocui.ModNone, LoginView); err != nil {
		return err
	}
	if err := g.SetKeybinding("main", gocui.KeyCtrlR, gocui.ModNone, RegView); err != nil {
		return err
	}
	if err := g.SetKeybinding("UsernameR", gocui.KeyTab, gocui.ModNone, nextViewReg); err != nil {
		return err
	}
	if err := g.SetKeybinding("PasswordR", gocui.KeyTab, gocui.ModNone, nextViewReg); err != nil {
		return err
	}
	if err := g.SetKeybinding("ConfirmR", gocui.KeyTab, gocui.ModNone, nextViewReg); err != nil {
		return err
	}
	if err := g.SetKeybinding("Username", gocui.KeyTab, gocui.ModNone, nextViewLogin); err != nil {
		return err
	}
	if err := g.SetKeybinding("Password", gocui.KeyTab, gocui.ModNone, nextViewLogin); err != nil {
		return err
	}
	if err := g.SetKeybinding("Username", gocui.KeyEnter, gocui.ModNone, LoginEnter); err != nil {
		return err
	}
	if err := g.SetKeybinding("Password", gocui.KeyEnter, gocui.ModNone, LoginEnter); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlSpace, gocui.ModNone, ReturnMain); err != nil {
		return err
	}

	if err := g.SetKeybinding("UsernameR", gocui.KeyEnter, gocui.ModNone, RegEnter); err != nil {
		return err
	}
	if err := g.SetKeybinding("PasswordR", gocui.KeyEnter, gocui.ModNone, RegEnter); err != nil {
		return err
	}
	if err := g.SetKeybinding("ConfirmR", gocui.KeyEnter, gocui.ModNone, RegEnter); err != nil {
		return err
	}

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("FriendList", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("FriendList", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
	if err := g.SetKeybinding("FriendList", gocui.KeyTab, gocui.ModNone, getLine); err != nil {
		return err
	}
	if err := g.SetKeybinding("FriendList", gocui.KeySpace, gocui.ModNone, getLine); err != nil {
		return err
	}
	if err := g.SetKeybinding("textMsg", gocui.KeyTab, gocui.ModNone, nextViewMsg); err != nil {
		return err
	}
	if err := g.SetKeybinding("textMsg", gocui.KeyEnter, gocui.ModNone, sendMsg); err != nil {
		return err
	}
	return nil
}

func layout(g *gocui.Gui) error {
	//_, maxY := g.Size()
	if v, err := g.SetView("Banner", 5, 2, 115, 8); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = false
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		fmt.Fprintln(v, "                    __  __     ____         _       __           __    __   ________          __ ")
		fmt.Fprintln(v, "                   / / / /__  / / /___     | |     / /___  _____/ /___/ /  / ____/ /_  ____ _/ /_")
		fmt.Fprintln(v, "                  / /_/ / _ \\/ / / __ \\    | | /| / / __ \\/ ___/ / __  /  / /   / __ \\/ __ `/ __/")
		fmt.Fprintln(v, "                 / __  /  __/ / / /_/ /    | |/ |/ / /_/ / /  / / /_/ /  / /___/ / / / /_/ / /_  ")
		fmt.Fprintln(v, "                /_/ /_/\\___/_/_/\\____/     |__/|__/\\____/_/  /_/\\__,_/   \\____/_/ /_/\\__,_/\\__/  ")
	}
	if v, err := g.SetView("main", 20, 12, 100, 30); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(v, "\n\n")
		fmt.Fprintln(v, "                                1.Login")
		fmt.Fprintln(v, "\n\n\n")
		fmt.Fprintln(v, "                                2.Register")
		fmt.Fprintln(v, "\n\n\n")
		fmt.Fprintln(v, "                                3.Exit")
		if _, err := g.SetCurrentView("main"); err != nil {
			return err
		}
	}
	return nil
}

func setCurrentViewOnTop(g *gocui.Gui, name string) (*gocui.View, error) {
	if _, err := g.SetCurrentView(name); err != nil {
		return nil, err
	}
	return g.SetViewOnTop(name)
}

func LoginView(g *gocui.Gui, v *gocui.View) error {
	//maxX, maxY := g.Size()

	if v, err := g.SetView("Username", 30, 15, 70, 17); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Username"
		v.Editable = true
		if _, err = g.SetCurrentView("Username"); err != nil {
			return err
		}
	}
	if v, err := g.SetView("Password", 30, 19, 70, 21); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Password"
		v.Editable = true
		v.Mask ^= '*'
		//if _, err = setCurrentViewOnTop(g, "Password"); err != nil {
		//	return err
		//}
	}
	if v, err := g.SetView("OK", 33, 24, 53, 26); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(v, "OK(ENTER)")

		//if _, err = setCurrentViewOnTop(g, "Password"); err != nil {
		//	return err
		//}
	}
	if v, err := g.SetView("Cancel", 55, 24, 75, 26); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		fmt.Fprintln(v, "Cancel(ESC)")

		//if _, err = setCurrentViewOnTop(g, "Password"); err != nil {
		//	return err
		//}
	}

	return nil
}

func RegView(g *gocui.Gui, v *gocui.View) error {
	//maxX, maxY := g.Size()
	if v, err := g.SetView("UsernameR", 30, 15, 70, 17); err != nil {
		if err != gocui.ErrUnknownView {
			panic(err)
			return err
		}
		v.Title = "Username"
		v.Editable = true
		if _, err = g.SetCurrentView("UsernameR"); err != nil {
			panic(err)
			return err
		}
	}
	if v, err := g.SetView("PasswordR", 30, 19, 70, 21); err != nil {
		if err != gocui.ErrUnknownView {
			panic(err)
			return err
		}
		v.Title = "Password"
		v.Editable = true
		v.Mask ^= '*'
		//if _, err = setCurrentViewOnTop(g, "Password"); err != nil {
		//	return err
		//}
	}
	if v, err := g.SetView("ConfirmR", 30, 23, 70, 25); err != nil {
		if err != gocui.ErrUnknownView {
			panic(err)
			return err
		}
		v.Title = "Confirm"
		v.Editable = true
		v.Mask ^= '*'
		//if _, err = setCurrentViewOnTop(g, "Password"); err != nil {
		//	return err
		//}
	}
	if v, err := g.SetView("OKR", 40, 27, 50, 29); err != nil {
		if err != gocui.ErrUnknownView {
			panic(err)
			return err
		}
		fmt.Fprintln(v, "OK(ENTER)")

		//if _, err = setCurrentViewOnTop(g, "Password"); err != nil {
		//	return err
		//}
	}
	if v, err := g.SetView("CancelR", 53, 27, 65, 29); err != nil {
		if err != gocui.ErrUnknownView {
			panic(err)
			return err
		}
		fmt.Fprintln(v, "Cancel(ESC)")

		//if _, err = setCurrentViewOnTop(g, "Password"); err != nil {
		//	return err
		//}
	}
	//fmt.Println("test OK")
	return nil
}

func InitCui() {

	g, err := gocui.NewGui(gocui.OutputNormal)
	//fmt.Println("OK")
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.Cursor = true

	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
