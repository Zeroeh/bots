package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	/*
		char slot unlocker = 810
		vault unlocker = 811
	*/
	consecutive          = "consecutive"
	nonconsecutive       = "nonconsecutive"
	desiredItem    int32 = 811 //these will be global for all accounts so no need to privatize
	desiredDay           = 31
)

type LoginCalendar struct {
	XMLName xml.Name           `xml:"NonConsecutive"`
	Text    string             `xml:",chardata"`
	Days    string             `xml:"days,attr"`
	Login   []LoginCalendarDay `xml:"Login"`
}

type LoginCalendarDay struct {
	Text   string `xml:",chardata"`
	Days   string `xml:"Days"`
	ItemId struct {
		Text     string `xml:",chardata"`
		Quantity string `xml:"quantity,attr"`
	} `xml:"ItemId"`
	Gold string `xml:"Gold"`
	Key  string `xml:"key"`
}

func (c *Client) ModuleDailyLoginMain() {
	c.Moves.TargetPosition = c.Moves.CurrentPosition
	if c.CurrentMap == "Daily Quest Room" {
		switch c.Mod.Phase {
		case 0: //grabbing the calendar
			if c.Mod.GrabbedCalendar == false {
				c.Mod.GrabbedCalendar = true
				go c.NonBlockGrabCalendar()
			}
		case 1: //claim item
			//todo: when i add multiple item support, append each item id to a slice and iterate the slice
			// use a client variable to keep track of the last item id attempted to be claimed
			if SinceLast(c.Times.SwapAction) >= c.Times.SwapSpeedMS {
				c.Times.SwapAction = time.Now()
				dl := LoginRewardSend{}
				dl.ClaimKey = c.GrabDayKey(desiredDay) //desiredDay
				dl.Type = nonconsecutive
				if dl.ClaimKey == "" {
					// fmt.Println("Couldnt get the key")
					c.KillClient()
					return
					//c.Phase = 2 //kill the client and log this account to a file
				}
				c.Connection.Send(WritePacket(dl))
				fmt.Println("Sent claim packet...")
			}
		case 2:
			// fmt.Println("Claimed item, killing client")
			c.KillClient()
		default:
			fmt.Println("Reached unknown state in dailylogin")
		}
	} else {
		qr := QuestRoomMessage{}
		c.Connection.Send(WritePacket(qr))
	}
}

//Uses a mutex to not block newtick packets
func (c *Client)NonBlockGrabCalendar() {
	c.GeneralMutex.Lock()
	err := c.GrabCalendar() //this blocks
	if err != nil {
		fmt.Println("Unable to grab calendar:", err) 
		//log this to a file?? switch proxy?
	} else {
		// fmt.Println("Got calendar!")
		c.Mod.GrabbedCalendar = true //redundant, but keeping for verbosity
		c.Mod.Phase = 1
	}
	c.GeneralMutex.Unlock()
}

//Grabs the accounts calendar. Returns error if unable to get calendar
func (c *Client) GrabCalendar() error {
	resp, err := c.getURL(rootURL + "dailyLogin/fetchCalendar?guid=" + c.Base.Email + "&password=" + c.Base.Password)
	if err != nil { //just pass this one up the stack
		return err
	}
	idx1 := strings.Index(resp, "<NonConsecutive")
	if idx1 == -1 {
		return errors.New("First login tag not found")
	}
	idx2 := strings.Index(resp, "</NonConsecutive>")
	if idx2 == -1 {
		return errors.New("Second tag not found")
	}
	idx2 += len("</NonConsecutive>")
	resp2 := resp[idx1:idx2]
	rdr := bytes.NewBufferString(resp2)
	err = xml.NewDecoder(rdr).Decode(&c.Mod.CalendarDays)
	if err != nil {
		return err
	}
	return nil
}

//grabs the specified days key. Returns empty string if no key is available
func (c *Client) GrabDayKey(day int) string {
	for i := range c.Mod.CalendarDays.Login {
		if c.Mod.CalendarDays.Login[i].Days == strconv.Itoa(day) {
			if c.Mod.CalendarDays.Login[i].Key != "" {
				return c.Mod.CalendarDays.Login[i].Key
			}
			return ""
		}
	}
	return ""
}
