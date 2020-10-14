package main

import (
	"time"
)

//The nil module simply idles the bot wherever it loads
func (c *Client)ModuleNilMain() {
	if c.CurrentMap == "Nexus" {
		if c.Master.MasterID != 0 && c.Master.FollowMaster == true {
			c.Moves.TargetPosition = c.Master.MasterPos
		} else {
			c.Moves.TargetPosition = c.Moves.CurrentPosition
			//c.TargetPosition = nameChanger
			//c.rotateClockwise(nexusCenter, .5)
			/*switch c.Phase {
			case 0: c.TargetPosition = nexusSpawnpadCorner0
			case 1: c.TargetPosition = nexusSpawnpadCorner1
			case 2: c.TargetPosition = nexusSpawnpadCorner2
			case 3: c.TargetPosition = nexusSpawnpadCorner3
			}
			if c.CurrentPosition.distanceTo(&nexusSpawnpadCorner0) <= .5 {
				c.Phase = 1
			} else if c.CurrentPosition.distanceTo(&nexusSpawnpadCorner1) <= .5 {
				c.Phase = 2
			} else if c.CurrentPosition.distanceTo(&nexusSpawnpadCorner2) <= .5 {
				c.Phase = 3
			} else if c.CurrentPosition.distanceTo(&nexusSpawnpadCorner3) <= .5 {
				c.Phase = 0
			}*/
			/*if SinceLast(c.OtherAction) >= 2000 {
				pt := PlayerText{}
				pt.Message = strings.ToUpper(GetRandString(127))
				//pt.Message = "/pause"
				c.Connection.Send(WritePacket(pt))
				c.OtherAction = time.Now()
			}*/
		}
	} else {
		if c.Master.MasterID != 0 && c.Master.FollowMaster == true {
			c.Moves.TargetPosition = c.Master.MasterPos
			if c.Moves.CurrentPosition.distanceTo(&c.Master.MasterPos) > 10.0 {
				tp := Teleport{}
				tp.ObjectID = c.Master.MasterID
				c.Connection.Send(WritePacket(tp))
			}
		} else {
			c.Moves.TargetPosition = c.Moves.CurrentPosition
			if SinceLast(c.Times.OtherAction) >= 5000 {
				c.Times.OtherAction = time.Now()
				//c.suicide(otherTarget)
			}
		}
	}
}


