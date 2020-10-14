package main

import (
	
)

func (c *Client)ModuleSpamMain() {
	if c.CurrentMap == "Nexus" {
		switch c.Mod.Phase {
			case 0: c.Moves.TargetPosition = nexusSpawnpadCorner0
			case 1: c.Moves.TargetPosition = nexusSpawnpadCorner1
			case 2: c.Moves.TargetPosition = nexusSpawnpadCorner2
			case 3: c.Moves.TargetPosition = nexusSpawnpadCorner3
		}
		if c.Moves.CurrentPosition.distanceTo(&nexusSpawnpadCorner0) <= .1 {
			c.Mod.Phase = 1
			s1 := PlayerText{}
			//s1.Message = "https://realmsupply.xyz     Newest rotmg store, cheapest prices"
			s1.Message = "rea1msupply.club     Coming soon!,"
			c.Connection.Send(WritePacket(s1))
		} else if c.Moves.CurrentPosition.distanceTo(&nexusSpawnpadCorner1) <= .1 {
			c.Mod.Phase = 2
			s1 := PlayerText{}
			//s1.Message = "https://realmsupply.xyz     ST items are in stock! Check us out!"
			s1.Message = "rea1msupply.club     Coming soon!."
			c.Connection.Send(WritePacket(s1))
		} else if c.Moves.CurrentPosition.distanceTo(&nexusSpawnpadCorner2) <= .1 {
			c.Mod.Phase = 3
			s1 := PlayerText{}
			//s1.Message = "https://realmsupply.xyz     Divine accounts coming soon!"
			s1.Message = "rea1msupply.club     Coming soon!/"
			c.Connection.Send(WritePacket(s1))
		} else if c.Moves.CurrentPosition.distanceTo(&nexusSpawnpadCorner3) <= .1 {
			c.Mod.Phase = 0
			s1 := PlayerText{}
			//s1.Message = "https://realmsupply.xyz     Instant delivery available!"
			s1.Message = "rea1msupply.club     Coming soon!["
			c.Connection.Send(WritePacket(s1))
		}
	} else {
		c.QueueRecon(-2, []byte{}, -1)
	}
}
