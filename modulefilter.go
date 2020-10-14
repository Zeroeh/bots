package main

import (
	"bufio"
	"strings"
)

var (
	blackList []string
)

const (
	previousLen int = 10
)

//triggers on every text packet
func (c *Client) ModuleFilterMain(chat *Text) {
	if c.Mod.ContainsFromBlacklist(chat.Message) == false {
		if c.Mod.HasSeenRecently(chat.Message) == false {
			// fmt.Printf("%s: %s\n", chat.Name, chat.Message)
			// logln("Got message:", chat.Message)
			if onlyContains(chat.Message, "ty") {
				return
			}
			if !c.Mod.LogMessage(chat.Name + ":" + chat.Message) {
				logln("Failed to log message:", chat.Message)
			}
		}
	}
}

func (m *ModuleBase) HasSeenRecently(s string) bool {
	for _, v := range m.PreviousMessages {
		if v == s {
			return true
		}
	}
	//pop the 0th string (oldest) if the slice len is greater than n
	if len(m.PreviousMessages) == previousLen {
		m.PreviousMessages = m.PreviousMessages[1:]
	}
	//push the new string onto the slice
	m.PreviousMessages = append(m.PreviousMessages, s)
	return false
}

func (m *ModuleBase) LogMessage(s string) bool {
	s1 := stripNewLine(s) + "\n"
	m.LogWriter.WriteString(s1)
	err := m.LogWriter.Flush()
	if err != nil {
		return false
	}
	return true
}

func (m *ModuleBase) ReadBlackList() {
	f, err := blanketOpenFile("blacklist.txt", false)
	if err != nil {
		logln("Error opening blacklist:", err)
	}
	m.BlackList = parseNewLines(f)
	f.Close()
}

func (m *ModuleBase) ContainsFromBlacklist(s string) bool {
	for x := 0; x < len(m.BlackList); x++ {
		if strings.Contains(toLower(s), toLower(m.BlackList[x])) == true {
			return true
		}
	}
	return false
}

func (m *ModuleBase) OpenLogFile(s string) {
	f, err := blanketOpenFile(s+"_dataset.txt", true)
	if err != nil {
		logln("Error opening logfile:", err)
	}
	m.LogFile = f
	m.LogWriter = bufio.NewWriter(m.LogFile)
}
