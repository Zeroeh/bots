package main

import (
	"fmt"
)

//note: c.MyVaults[0].Pos can be turned into something like c.MyVaults[c.GetDupeChest()] when we have more than 1 vault unlocked
// we can use a for loop to iterate over all open chests and match the id with the position

var (
	vaultPortalGID = 711
	dupeSpamCount = 8
	chestLoc      = WorldPosData{
		X: 44.5,
		Y: 70.5,
	}
	tutorialChestLoc = WorldPosData{
		X: 102.5,
		Y: 138.5,
	}
	recoveryThreshold = 2
)

//ModuleDupeMain is the main core loop for duping on the bots
func (c *Client) ModuleDupeMain() {
	//set our position without being in phase
	if c.CurrentMap == "Vault" || c.CurrentMap == "Vault Explanation" {
		c.Moves.TargetPosition = tutorialFirstChestLoc
	} else {
		c.Moves.TargetPosition = nexusVaultPortal // used to be c.Moves.CurrentPosition
	}
	switch c.Mod.Phase {
	case 0: //we loaded into the vault now we check for dupe success
		if c.CurrentMap == "Vault" || c.CurrentMap == "Vault Explanation" {
			good := c.IsDupeSuccess()
			if good == true {
				fmt.Printf("%s: Dupe Success!\n", c.Base.Email)
				// quit() //COMMENT FOR MASS DUPE BOTS, UNCOMMENT FOR PERSONAL DUPING / SERVICES
				c.Mod.InventoryRecoveryCount = 0
				c.Mod.Phase = 6
			} else { //one of them is empty, lets find out which one
				invok := c.IsInventoryFull()
				chestok := c.IsVaultChestFull(tutorialFirstChestLoc)
				if invok == true && chestok == false { //inv is full, chest is not
					c.Mod.Phase = 2
				}
				if invok == false && chestok == true { //chest is full, inv is not
					c.Mod.Phase = 1
				}
				if invok == false && chestok == false { //shit got fucked up
					fmt.Printf("%s: Attempting recovery from bad inventory\n", c.Base.Email)
					c.GetBadIndex() //change our dupemoveindex
					c.Mod.InventoryRecoveryCount++
					if c.Mod.InventoryRecoveryCount > recoveryThreshold { //no point in having them try over and over again
						c.KillClient()
					}
					c.Mod.Phase = 3
				}
			}
		} else {
			c.QueueRecon(-5, []byte{}, GetUnixTime())
		}
	case 1: //our inventory is empty and chest has items, pick up the items
		if c.Moves.CurrentPosition.distanceTo(&tutorialFirstChestLoc) <= .5 {
			if SinceLast(c.Times.SwapAction) >= c.Times.SwapSpeedMS {
				if c.Mod.DupeMoveIndex >= 8 {
					c.Mod.Phase = 0
					c.Dupe()
					return
				}
				if c.Vault.MyVaults == nil {
					return
				}
				c.SwapItem(c.Mod.DupeMoveIndex, c.Mod.DupeMoveIndex+4, 4, c.Vault.MyVaults[tutorialFirstChestLoc])
				// fmt.Println("Take swap") //debug
			}
		}
		//no need to do anything as target is set to first chest
	case 2: //chest is empty and inventory is full, so put items back
		if c.Moves.CurrentPosition.distanceTo(&tutorialFirstChestLoc) <= .5 {
			if SinceLast(c.Times.SwapAction) >= c.Times.SwapSpeedMS {
				if c.Mod.DupeMoveIndex >= 8 {
					c.Mod.Phase = 0
					c.QueueRecon(-8, []byte{}, GetUnixTime())
					return
				}
				if c.Vault.MyVaults == nil {
					return
				}
				c.SwapItem(c.Mod.DupeMoveIndex+4, c.Mod.DupeMoveIndex, 3, c.Vault.MyVaults[tutorialFirstChestLoc])
				// fmt.Println("Put swap") //debug
			}
		}
	case 3: //recover from any messed up duping inventories, put any items back into chest and reconnect
		if c.Moves.CurrentPosition.distanceTo(&tutorialFirstChestLoc) <= .5 {
			if SinceLast(c.Times.SwapAction) >= c.Times.SwapSpeedMS {
				if c.Mod.DupeMoveIndex >= 8 {
					c.Mod.Phase = 0
					c.QueueRecon(-8, []byte{}, GetUnixTime())
					return
				}
				if c.Vault.MyVaults == nil {
					return
				}
				c.SwapItem(c.Mod.DupeMoveIndex+4, c.Mod.DupeMoveIndex, 3, c.Vault.MyVaults[tutorialFirstChestLoc])
				// fmt.Println("Recover Swapped") //debug
			}
		}
	case 4: //head over to vault portal and useportal
		
	case 5:
	case 6: //this is the phase where we nexus and wait for receive bot to trade
		if c.CurrentMap == "Nexus" {
			good := c.IsInventoryFull() //perform a quick check just to make sure
			if good == true {
				if c.Trade.InTrade == true && c.Trade.AcceptedTrade == false && c.Trade.SentOffer == false { //server acks that we are in a trade at this point
					ct := ChangeTrade{}
					ct.MyOffers = c.Trade.SelectAll()
					c.Connection.Send(WritePacket(ct))
					c.Trade.SentOffer = true
				}
				if c.Trade.InTrade == true && c.Trade.AllowedAccept == true && c.Trade.SentOffer == true && c.Trade.AcceptedTrade == false {
					c.AcceptMeAllThemNone()
					c.AcceptTrade()
				}
			} else {
				c.Mod.Phase = 0
				c.QueueRecon(-8, []byte{}, GetUnixTime())
			}
			// commented the below now that I will have fixed the mass dupe bots
			// c.Mod.Phase = 0
			// c.QueueRecon(-8, []byte{}, GetUnixTime())
		} else {
			c.QueueRecon(-2, []byte{}, GetUnixTime())
		}
	default:
		c.Mod.Phase = 0
	}
}

func (c *Client) GetBadIndex() {
	itemIndex := 0
	//if dupemoveindex is having issues then initialize itemIndex to -1
	//if the first few slots have items | works if picking stuff up
	//if the first few slots are empty | works if we were putting stuff away
	if c.Stats.StatMap[INVENTORY4].StatValue == -1 { //we were putting stuff into chest
		if c.Stats.StatMap[INVENTORY4].StatValue == -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY5].StatValue == -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY6].StatValue == -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY7].StatValue == -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY8].StatValue == -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY9].StatValue == -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY10].StatValue == -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY11].StatValue == -1 {
			itemIndex++
		}
	} else if c.Stats.StatMap[INVENTORY4].StatValue != -1 { //we were picking stuff up
		if c.Stats.StatMap[INVENTORY4].StatValue != -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY5].StatValue != -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY6].StatValue != -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY7].StatValue != -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY8].StatValue != -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY9].StatValue != -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY10].StatValue != -1 {
			itemIndex++
		}
		if c.Stats.StatMap[INVENTORY11].StatValue != -1 {
			itemIndex++
		}
	} else {
		fmt.Printf("%s: Error while recovering from bad dupe and needs manual fixing\n", c.Base.Email)
	}
	c.Mod.DupeMoveIndex = byte(itemIndex)
}

//IsDupeSuccess checks if the dupe was successful
func (c *Client) IsDupeSuccess() bool {
	invGood := c.IsInventoryFull()
	vaultGood := c.IsVaultChestFull(tutorialFirstChestLoc)
	if vaultGood == true && invGood == true {
		return true
	}
	return false
}

//Dupe IS TOP SECRET!!!!!!!!! COVER YOUR EYES!
func (c *Client) Dupe() {
	var isAdd = false
	e := EditAccountList{}
	e.AccountListID = 0
	e.ObjectID = c.ObjectID
	for i := 0; i < dupeSpamCount; i++ {
		if isAdd == true {
			e.Add = true
			isAdd = false
		} else {
			e.Add = false
			isAdd = true
		}
		c.Connection.Send(WritePacket(e))
	}
	if verbose == true {
		fmt.Println("Sent dupe!!")
	}
	// fmt.Println("DID THE DUPE!!!!") //debug
	// c.Disconnect()
	// c.DelayAction(500, func(){c.Base.getCharList()})
	// time.Sleep(1000 * time.Millisecond)
	// c.Base.getCharList()
	//todo: disconnect and then delay the queuerecon
	c.QueueRecon(-8, []byte{}, GetUnixTime())
}
