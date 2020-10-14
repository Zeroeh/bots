package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

func (c *Client) OnFailure(failure Failure) {
	//comment out this printf if you prefer less verbose failure messages, however, this also acts as a "catch-all" in case some errors are not added in the handler
	SwitchColor(Red)
	log.Printf("%s:%d-> Failure: %d | %s\n", c.Base.Email, c.Mod.Phase, failure.FailureID, failure.FailureMessage)
	SwitchColor(Normal)
	c.HandleFailure(&failure) //is a source of bugs, needs to be redone tbh
}

func (c *Client) OnMapInfo(mapinfo MapInfo) {
	c.CurrentMap = mapinfo.Name
	//fmt.Println(c.CurrentMap)
	c.Moves.MapHeight = mapinfo.Height
	c.Moves.MapWidth = mapinfo.Width
	// fmt.Printf("Map Size: %dx%d\n", c.Moves.MapWidth, c.Moves.MapHeight)
	c.Times.OtherAction = time.Now() //just so we can "reset" any actions
	c.Recon.ConnectionGUID = mapinfo.ConnectionGUID
	// fmt.Println("ConnectionGUID:", c.Recon.ConnectionGUID)
	if c.Base.Module == "dupe" {
		c.Trade.AllowedAccept = false
	}
	if c.Base.Module == "filter" {
		c.Mod.ReadBlackList()
		c.Mod.OpenLogFile(c.Base.ServerIP)
	}
	if c.Base.CharID != 0 && c.Base.FetchNewData == false {
		//we have a character
		if c.Base.CharID == -1 { //gonna be invalid
			c.Disconnect()
			return
		}
		load := Load{}
		load.CharID = int32(c.Base.CharID)
		load.IsFromArena = false
		c.Connection.Send(WritePacket(load))
	} else {
		if c.Base.FetchNewData == true && c.Base.CharID == 0 {
			//get an updated char id
			c.Base.CharID = c.Base.GetNewCharID()
		}
		//check if it returned 0 or a real id
		if c.Base.CharID == 0 {
			create := Create{}
			create.ClassType = 782 //default to wizard for now
			create.SkinType = 0
			c.Connection.Send(WritePacket(create))
		} else { //-1 char id goes thru here lol
			load := Load{}
			load.CharID = int32(c.Base.CharID)
			load.IsFromArena = false
			c.Connection.Send(WritePacket(load))
		}
	}
}

func (c *Client) OnCreateSuccess(createsuccess CreateSuccess) {
	c.ObjectID = createsuccess.ObjectID
	c.Recon.Reset()
	c.Base.CharID = int(createsuccess.CharID)
	//reset any variables on mapinfo, not here
	SwitchColor(Yellow)
	fmt.Printf("%s joined %s! ObjectID: %d\n", c.Base.Email, c.CurrentMap, c.ObjectID) //used to make sure we've actually connected to the game
	SwitchColor(Normal)
	c.ClearMaps()
}

func (c *Client) OnUpdate(update Update) {
	updateack := UpdateAck{}
	c.Connection.Send(WritePacket(updateack))
	c.ParseUpdateData(&update) //update our bots' position, only works on map load. Get updated values from newtick!
	if c.Base.Module == "darkeye" {
		go c.ModuleDarkeyeUpdateCallback(&update) //if any issues arise, try removing this goroutine
	}
	if c.Base.Module == "vip" {
		go c.ModuleVIPTrackCallback(&update)
	}
	if c.Base.Module == "supporttracker" {
		go c.ModuleSupportTrackCallback(&update)
	}
}

func (c *Client) OnPing(ping Ping) {
	//fmt.Println("Seconds connected:", ping.Serial+1) //each "ID" is 1 second; server sends ping every second
	pong := Pong{}
	pong.Serial = ping.Serial
	pong.Time = c.Times.GetTime()
	c.Connection.Send(WritePacket(pong))
}

func (c *Client) OnNewTick(newtick NewTick) {
	c.Times.LastTickTime = c.Times.CurrentTickTime
	c.Times.CurrentTickTime = c.Times.GetTime()
	c.Times.LastServerRealTimeMS = newtick.ServerRealTimeMS
	c.Moves.TickCount++

	if c.Combat.ShootingEnabled == true && c.Master.MasterID != 0 {
		pos := c.Game.GetObjByID(c.Master.MasterID).Status.Pos
		c.Shoot(c.Moves.CurrentPosition.angleTo(&pos))
	}

	c.moveTo(c.Moves.TargetPosition)
	move := Move{}
	move.Time = c.Times.GetTime()
	move.TickID = newtick.TickID
	move.ServerRealTimeMSOfLastNewTick = c.Times.LastServerRealTimeMS
	move.NewPosition = c.Moves.CurrentPosition
	//move records stuff
	lastClear := c.Moves.LastClearTime
	if lastClear >= 0 && move.Time-lastClear > 125 {
		// llen :=
		// 	const len = Math.min(10, this.moveRecords.records.length);
		//   for (let i = 0; i < len; i++) {
		//     if (this.moveRecords.records[i].time >= movePacket.time - 25) {
		//       break;
		//     }
		//     movePacket.records.push(this.moveRecords.records[i].clone());
		//   }
	}
	// c.Moves.Records.clear()
	c.Connection.Send(WritePacket(move))
	c.ParseNewTickData(&newtick)
	c.Moves.LastTickID = newtick.TickID
	switch c.Base.Module {
	case "dupe": //dupes up items
		c.ModuleDupeMain()
	case "receive": //receives items duped from the dupe module
		// c.ModuleReceiveMain()
	case "rankdupe": //secret dupe
		// c.ModuleRankMain()
	case "spam": //self explanatory
		c.ModuleSpamMain()
	case "darkeye": //darkeye player logger
		c.ModuleDarkeyeMain(&newtick)
	case "dailylogin": //logs bots in and claims the desired item(s)
		c.ModuleDailyLoginMain()
	case "vip": //player tracker
		c.ModuleVIPTrackMain()
	case "supporttracker": //tracks players' supporter points for the unity campaign. now deprecated
		c.ModuleSupportTrackMain(&newtick)
	case "vaultunlock": //unlocks the vault by duping vault chest unlockers
		// c.ModuleVaultUnlockMain()
	case "vaultbegin": //sets up the bots for the vaultunlock module
		// c.ModuleVaultBeginMain()
	case "nil": //default module, bot spawns in the server and does nothing
		c.ModuleNilMain()
	case "complete": //special module that signifies that this bot is finished with its task
		c.Moves.TargetPosition = c.Moves.CurrentPosition
		// c.DelayAction(10000, func() { //works, but it needs some sort of lock
		// 	c.QueueRecon(-2, []byte{}, GetUnixTime())
		// })
		c.KillClient()
	default:
		c.ModuleNilMain()
	}
	if c.Combat.ShootingEnabled == true && c.Stats.StatMap[INVENTORY0].StatValue != 0 && c.Combat.Target != 0 {
		c.ShootTarget()
	}
}

func (c *Client) OnReconnect(reconnect Reconnect) {
	if c.Recon.BlockingReconnects == true {
		return
	}
	if reconnect.Host != "" {
		c.Recon.PreviousServer = c.Recon.CurrentServer
		c.Recon.CurrentServer = reconnect.Host
	}
	fmt.Println("Reconnecting to", reconnect.Name)
	c.QueueRecon(reconnect.GameID, reconnect.Key, reconnect.KeyTime)
}

func (c *Client) OnQuestObjID(questobjid QuestObjID) {
}

func (c *Client) OnInvResult(result InvResult) {
	//code -1 means the item did not swap
	switch c.Base.Module {
	case "dupe":
		if result.Result == 0 {
			if c.Mod.Phase == 1 || c.Mod.Phase == 2 || c.Mod.Phase == 3 { //just incase because then any invswap would increase it
				c.Mod.DupeMoveIndex++
			}
		}
	case "receive":
		if result.Result == 0 {
			if c.Mod.Phase == 2 || c.Mod.Phase == 1 {
				c.Mod.GoodStatus = true
				//c.Connection.Kill(c, true)
			}
		}
	// case "vaultunlock":
	// 	if result.Result == 0 {
	// 		if c.Mod.Phase == 1 || c.Mod.Phase == 2 || c.Mod.Phase == 3 || c.Mod.Phase == 4 { //just incase because then any invswap would increase it
	// 			c.Mod.DupeMoveIndex++
	// 		}
	// 		if c.Mod.Phase == 4 { //ignore this code block, invresults are not received on useitem
	// 			c.Times.AllowedSwap = true
	// 			c.Mod.ChestsUnlocked++
	// 			if c.Mod.ChestsUnlocked == maxVaultCount {
	// 				c.Base.Module = "complete"
	// 			}
	// 		}
	// 	}
	case "vaultbegin":
		if result.Result == 0 {
			switch c.Mod.Phase {
			case 0:
				c.Mod.Phase = 1
				// fmt.Println("Invswap success, moved to phase 1")
			case 1:
				// c.Phase = 2
				// fmt.Println("Dropped into chest, moving to phase 3")
				c.QueueRecon(-5, []byte{}, GetUnixTime())
			case 3:
				c.Mod.LowIndex++
				c.Mod.HighIndex++
			case 4:
				c.Mod.DupeMoveIndex++
			case 6:
				c.Mod.LowIndex++
				c.Mod.HighIndex++
			}

		}
	}
}

func (c *Client) OnTradeRequested(traderequested TradeRequested) {
	fmt.Printf("%s wants to trade with %s\n", traderequested.Name, c.Base.Email)
	if c.Base.Module != "receive" { //so idiots dont hold up my duping :)
		reqtrade := RequestTrade{}
		reqtrade.PlayerName = traderequested.Name
		c.Connection.Send(WritePacket(reqtrade))
	}
}

func (c *Client) OnTradeAccepted(tradeaccepted TradeAccepted) {

}

func (c *Client) OnTradeStart(tradestart TradeStart) {
	//calculate anything we may want
	c.Trade.InTrade = true
	c.Trade.TradersName = tradestart.TheirName
	c.Trade.TheirOffers = c.Trade.SelectNone()
	//packet flow will stop if this hangs so
	//start a ticker so that someone cant keep our bot in a trade indefinitely
	cancel := time.After(10 * time.Second)
	tick := time.Tick(1 * time.Second)
	go func() {
	check:
		select {
		case <-cancel:
			//this will be handy if our dupe bots screw up
			fmt.Println("Stopping trade...")
			c.Trade.InTrade = false
			c.Trade.TradeSuccess = false
			c.Trade.AcceptedTrade = false
			c.Trade.SentOffer = false
			c.Trade.AllowedAccept = false
			ctrade := CancelTrade{}
			c.Connection.Send(WritePacket(ctrade))
			return
		case <-tick:
			if c.Trade.InTrade == false {
				break //break out of select
			}
			goto check
		}
	}()
}

func (c *Client) OnTradeChanged(tradechanged TradeChanged) {
	c.Trade.AllowedAccept = false
	c.Trade.AcceptedTrade = false
	c.Trade.TheirOffers = tradechanged.TheirOffers
}

func (c *Client) OnTradeDone(tradedone TradeDone) {
	c.Trade.InTrade = false
	c.Trade.AllowedAccept = false
	if tradedone.ResultCode == 0 {
		c.Trade.TradeSuccess = true
	} else {
		c.Trade.TradeSuccess = false
	}
	c.Trade.AcceptedTrade = false
	c.Trade.SentOffer = false
	if tradedone.ResultCode == 0 {
		log.Printf("%s: Trade Success!\n", c.InGameName)
	} else {
		fmt.Printf("%s finished trade. Code: %d | Message: %s\n", c.InGameName, tradedone.ResultCode, tradedone.Message)
	}

}

func (c *Client) OnInvitedToGuild(invite InvitedToGuild) {
	jguild := JoinGuild{}
	jguild.GuildName = invite.GuildName
	c.Connection.Send(WritePacket(jguild))
}

func (c *Client) OnServerPlayerShoot(serverplayershoot ServerPlayerShoot) {
	if serverplayershoot.OwnerID == c.ObjectID {
		shootack := ShootAck{}
		shootack.Time = c.Times.GetTime()
		c.Connection.Send(WritePacket(shootack))
	}
}

func (c *Client) OnClientStat(clientstat ClientStat) {

}

func (c *Client) OnCreateGuildResult(cgr CreateGuildResult) {
	if cgr.Success == true {
		return
	}
	fmt.Println("GuildResult Error:", cgr.ErrorMessage)
}

func (c *Client) OnDamage(damage Damage) {

}

func (c *Client) OnDeath(death Death) {
	c.Base.CharID = 0 //so we send create instead of load, or we can choose another char
	c.QueueRecon(c.Recon.GameID, []byte{}, GetUnixTime())
}

func (c *Client) OnEnemyShoot(enemyshoot EnemyShoot) {
	shootack := ShootAck{}
	shootack.Time = c.Times.GetTime()
	// owner := c.Combat.Enemies[enemyshoot.OwnerID]
	// if owner == nil || owner.dead {
	// 	shootack.Time = -1
	// }
	c.Connection.Send(WritePacket(shootack))
	// if owner == nil || owner.dead {
	// 	return
	// }
	// for (let i = 0; i < enemyShootPacket.numShots; i++) {
	// 	this.projectiles.push(
	// 	  new Projectile(
	// 		owner.properties.type,
	// 		enemyShootPacket.bulletType,
	// 		enemyShootPacket.ownerId,
	// 		(enemyShootPacket.bulletId + i) % 256,
	// 		enemyShootPacket.angle + i * enemyShootPacket.angleInc,
	// 		this.getTime(),
	// 		enemyShootPacket.startingPos.toPrecisePoint()
	// 	  )
	// 	);
	// 	this.projectiles[this.projectiles.length - 1].setDamage(enemyShootPacket.damage);
	//   }
	//   this.checkProjectiles();
	//c.DamageMe(enemyshoot.Damage, enemyshoot.BulletID, c.ObjectID)
}

func (c *Client) OnAoE(aoe AoE) {
	aoeack := AoEAck{}
	aoeack.Time = c.Times.GetTime()
	aoeack.Position = c.Moves.CurrentPosition
	if c.Moves.CurrentPosition.distanceTo(&aoe.Position) < aoe.Radius {
		//send damage packet
		//might actually be calculated server side
		//c.DamageMe(int16(aoe.Damage), 0, 0)
	}
	//needs to see if distanceTo(pos) < aoe.Radius, if true, send damage packet
	//or do we even always send an aoe response? and only respond when it is in our radius?
	c.Connection.Send(WritePacket(aoeack))
}

func (c *Client) OnAccountList(accountlist AccountList) {
}

func (c *Client) OnText(text Text) {
	c.HandleText(&text)
}

func (c *Client) OnGoto(got Goto) {
	gack := GotoAck{}
	gack.Time = c.Times.GetTime()
	c.Connection.Send(WritePacket(gack))
	if got.ObjectID == c.ObjectID {
		c.Moves.CurrentPosition = got.Location
	}
}

func (c *Client) OnShowEffect(showeffect ShowEffect) {

}

func (c *Client) OnAllyShoot(allyshoot AllyShoot) {

}

func (c *Client) OnFile(f File) {
	//attempt to write to file
	file, err := os.OpenFile(f.Name, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	defer file.Close()
	if err != nil {
		log.Fatal(err)
	}
	if _, err := file.Write(f.Bytes); err != nil {
		log.Fatal(err)
	}

}

func (c *Client) OnNotification(notification Notification) {
	//fmt.Println("Notification:", notification.Message)
}

func (c *Client) OnGlobalNotification(globalnotification GlobalNotification) {
	//fmt.Printf("Server: %s\n", globalnotification.Text)
}

func (c *Client) OnPlaySound(playsound PlaySound) {

}

func (c *Client) OnLoginRewardRecv(lr LoginRewardRecv) {
	if c.Base.Module == "dailylogin" {
		if lr.ItemID != desiredItem {
			fmt.Println("Didn't get desired item, got:", lr)
			//do some sort of debug routine perhaps?
			c.Mod.Phase = 2 //for testing
		} else {
			fmt.Printf("%s | Claimed the desired item!\n", c.Base.Email)
			c.Mod.Phase = 2
		}
	} else {
		fmt.Println("Claimed item!")
	}
}

func (c *Client) OnHeroLeft(r RealmHeroLeft) {
	fmt.Printf("Got %d heroes leaving\n", r.HeroesLeft)
}

//EvaluatePacket takes a packet type and performs a switch
// and runs the function of the specific packet type
func (c *Client) EvaluatePacket(v interface{}) {
	switch v.(type) {
	case Failure:
		c.OnFailure(v.(Failure))
	case MapInfo:
		c.OnMapInfo(v.(MapInfo))
	case CreateSuccess:
		c.OnCreateSuccess(v.(CreateSuccess))
	case Update:
		c.OnUpdate(v.(Update))
	case Ping:
		c.OnPing(v.(Ping))
	case NewTick:
		c.OnNewTick(v.(NewTick))
	case Reconnect:
		c.OnReconnect(v.(Reconnect))
	case AoE:
		c.OnAoE(v.(AoE))
	case Text:
		c.OnText(v.(Text))
	case GlobalNotification:
		c.OnGlobalNotification(v.(GlobalNotification))
	case AccountList:
		c.OnAccountList(v.(AccountList))
	case AllyShoot:
		c.OnAllyShoot(v.(AllyShoot))
	case ShowEffect:
		c.OnShowEffect(v.(ShowEffect))
	case InvResult:
		c.OnInvResult(v.(InvResult))
	case Notification:
		c.OnNotification(v.(Notification))
	case Goto:
		c.OnGoto(v.(Goto))
	case QuestObjID:
		c.OnQuestObjID(v.(QuestObjID))
	case TradeRequested:
		c.OnTradeRequested(v.(TradeRequested))
	case TradeStart:
		c.OnTradeStart(v.(TradeStart))
	case TradeAccepted:
		c.OnTradeAccepted(v.(TradeAccepted))
	case TradeChanged:
		c.OnTradeChanged(v.(TradeChanged))
	case TradeDone:
		c.OnTradeDone(v.(TradeDone))
	case InvitedToGuild:
		c.OnInvitedToGuild(v.(InvitedToGuild))
	case ServerPlayerShoot:
		c.OnServerPlayerShoot(v.(ServerPlayerShoot))
	case CreateGuildResult:
		c.OnCreateGuildResult(v.(CreateGuildResult))
	case Damage:
		c.OnDamage(v.(Damage))
	case Death:
		c.OnDeath(v.(Death))
	case EnemyShoot:
		c.OnEnemyShoot(v.(EnemyShoot))
	case ClientStat:
		c.OnClientStat(v.(ClientStat))
	case File:
		c.OnFile(v.(File))
	case PlaySound:
		c.OnPlaySound(v.(PlaySound))
	case LoginRewardRecv:
		c.OnLoginRewardRecv(v.(LoginRewardRecv))
	case RealmHeroLeft:
		c.OnHeroLeft(v.(RealmHeroLeft))
	case Unknown:
	case NewCharacterInformation:
	case QueueInformation:
	case UnlockInformation:
	case nil:
		log.Println("Receive() sent 255 / nil.")
		c.Disconnect()
	default:
		log.Println("Receive() sent:", v)
		//for these errors I may just have it quit instead
		//note: not a good idea unless we have ALL packet cases added
		c.Disconnect()
	}
}
