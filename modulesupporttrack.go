package main

import (
	"log"
	"strings"
	"time"
)

func (c *Client) ModuleSupportTrackCheckAccountIDExists(i OnlinePlayer) (bool, bool) {
	rows, err := dbConn.Query("SELECT * FROM users")
	if err != nil {
		log.Println("Error querying:", err)
	}
	defer rows.Close()
	for rows.Next() {
		p := OnlinePlayer{}
		id := 0 //the key in the database
		err = rows.Scan(&id, &p.AccountID, &p.Name, &p.SupportPoints)
		if err != nil {
			log.Println("Error retreiving row data:", err)
		}
		if strings.ToLower(i.AccountID) == strings.ToLower(p.AccountID) { //the account already exists in our database...
			if i.SupportPoints > p.SupportPoints { //they earned more points in game from when we last logged them
				//fmt.Println(i.SupportPoints, p.SupportPoints)
				return true, true
			}
			return true, false
		}
	}
	return false, false
}

//Register an account to the tracker database
func (c *Client) ModuleSupportTrackRegisterAccount(i OnlinePlayer) error {
	_, err := dbConn.Exec("INSERT INTO users(accountid, name, supportpoints) VALUES(?,?,?)", i.AccountID, i.Name, i.SupportPoints)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) ModuleSupportTrackUpdatePoints(i OnlinePlayer) {
	stmt, err := dbConn.Prepare("UPDATE users SET supportpoints=? WHERE accountid=?")
	if err != nil {
		log.Println("Error preparing statement:", err)
		return
	}
	_, err = stmt.Exec(i.SupportPoints, i.AccountID)
	if err != nil {
		log.Println("Error executing statement:", err)
	}
	log.Printf("Updated '%s' points to %d\n", i.Name, i.SupportPoints)
}

func (c *Client) ModuleSupportTrackLogAccounts() {
	ticker := time.Tick(time.Minute) //log to the database every minute
sel:
	select {
	case <-ticker:
		cCopy := c.Mod.MyPlayers                         //make a copy of our list so that we dont get issues when adding more while in this function
		c.Mod.MyPlayers = make([]OnlinePlayer, 0, 10000) //delete our current list
		clen := len(cCopy)
		if clen == 0 {
			goto sel //could become the cause of bugs
		}
		globalMutex.Lock()
		//c.Mutex.Lock() //local copy needed?
		for _, i := range cCopy {
			isAlreadyThere := false
			pointsUpdate := false
			if i.AccountID == "" { //if we don't have the accountid, ignore the player
				goto nexto
			}
			//check if the player is already in the database (by accountid) and check if they need updated points
			isAlreadyThere, pointsUpdate = c.ModuleSupportTrackCheckAccountIDExists(i)
			if isAlreadyThere == true { //update their points (if applicable)
				if pointsUpdate == true {
					c.ModuleSupportTrackUpdatePoints(i)
				}
			} else { //add them to the database
				err := c.ModuleSupportTrackRegisterAccount(i)
				if err != nil {
					log.Println("Error registering account:", err)
				} else {
					log.Printf("Created '%s' with %d points\n", i.Name, i.SupportPoints)
				}
			}
		nexto:
		}
		//c.Mutex.Unlock()
		globalMutex.Unlock()
	}
	c.Mod.MutexBool = false //incase the function ever ends we can reset
}

func (c *Client) ModuleSupportTrackMain(n *NewTick) {
	c.Moves.TargetPosition = c.Moves.CurrentPosition
	for i := 0; i < len(n.Statuses); i++ {
		for x := 0; x < len(c.Mod.MyPlayers); x++ {
			if c.Mod.MyPlayers[x].ObjectID == n.Statuses[i].ObjectID {
				aid := n.Statuses[i].FindStat(ACCOUNTID).StrStatValue
				if aid == "" {
					goto nexti
				}
				c.Mod.MyPlayers[x].AccountID = aid
				//log.Printf("Matched %s to %s\n", aid, c.MyPlayers[x].Name)
			}
		nexti:
		}
	}
	if c.Mod.MutexBool == false {
		c.Mod.MutexBool = true
		go c.ModuleSupportTrackLogAccounts()
	}
}

func (c *Client) ModuleSupportTrackCallback(u *Update) {
	for i := 0; i < len(u.NewObjs); i++ {
		starcnt := u.NewObjs[i].Status.FindStat(STARS).StatValue
		if starcnt >= 2 { //anyone higher than 1 stars is logged
			newPlayer := OnlinePlayer{}
			newPlayer.ObjectID = u.NewObjs[i].Status.ObjectID
			newPlayer.Position = u.NewObjs[i].Status.Pos
			name := u.NewObjs[i].Status.FindStat(NAME).StrStatValue
			if name == "" {
				goto end //not a player idk why were parsing O_O
			}
			newPlayer.Name = name
			accountid := u.NewObjs[i].Status.FindStat(ACCOUNTID).StrStatValue
			if accountid == "" {
				accountid = ""
			}
			newPlayer.AccountID = accountid

			supportpoints := u.NewObjs[i].Status.FindStat(SUPPORTERPOINTS).StatValue
			newPlayer.SupportPoints = int(supportpoints)
			if supportpoints <= 0 {
				return //do not add player and clear stack
			}
			c.Mod.MyPlayers = append(c.Mod.MyPlayers, newPlayer)
		}
	end:
	}
}
