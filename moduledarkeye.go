package main

import (
	"fmt"
)

func (c *Client) ModuleDarkeyeMain(n *NewTick) {
	c.Moves.TargetPosition = c.Moves.CurrentPosition
	for i := 0; i < len(n.Statuses); i++ {
		if n.Statuses[i].ObjectID == c.Mod.DarkEyeTrackerID {
			fmt.Println("Stat:", n.Statuses[i].Stats)
		}
	}
}

func (c *Client) ModuleDarkeyeUpdateCallback(u *Update) {
	for i := 0; i < len(u.NewObjs); i++ {
			starcnt := u.NewObjs[i].Status.FindStat(STARS).StatValue
			if starcnt > 0 { //weed out non-player objects
					name := u.NewObjs[i].Status.FindStat(NAME).StrStatValue
					if name == "" {
						goto end //not a player idk why were parsing O_O
					}
					if name != "Artosh" {
						goto end
					}
					c.Mod.DarkEyeTrackerID = u.NewObjs[i].Status.ObjectID
					fmt.Println("Stat:", u.NewObjs[i].Status.Stats)
					// stars := starcnt
					// accountid, ok := u.NewObjs[i].Status.Find(ACCOUNTID, false).(string)
					// if !ok {
					// 	accountid = ""
					// }
					// gold := u.NewObjs[i].Status.Find(CREDITS, true).(int)
					// fame := u.NewObjs[i].Status.Find(ACCOUNTFAME, true).(int)
					// owneraccountid, ok := u.NewObjs[i].Status.Find(OWNERACCOUNTID, false).(string)
					// if !ok {
					// 	owneraccountid = ""
					// }
					// //owneraccountid := ""
					// guildname, ok := u.NewObjs[i].Status.Find(GUILDNAME, false).(string)
					// if !ok {
					// 	guildname = ""
					// }
					// guildrank := u.NewObjs[i].Status.Find(GUILDRANK, true).(int)
					// _, err := dbConn.Exec("INSERT INTO users(name, stars, gold, accountid, fame, owneraccountid, guildname, guildrank) VALUES(?,?,?,?,?,?,?,?)", name, stars, gold, accountid, fame, owneraccountid, guildname, guildrank)
					// if err != nil {
					// 	fmt.Println("Error2:", err)
					// 	return
					// }
			}
			end:
			
	}
}
