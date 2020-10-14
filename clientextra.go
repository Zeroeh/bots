package main

import (
	"fmt"
	"time"
)

//clientextra.go is an extension of client.go and game.go that aims to reduce clutter in client.go
// and provide higher level abstractions in here rather than in low-level client.go functions

//GetChestData parses the object and adds it to a vault slice in the client
func (c *Client) GetChestData(o *ObjectStatusData) {
	if c.Vault.MyVaults == nil {
		return
	}
	c.Vault.MyVaults[o.Pos] = o
}

func (c *Client) GetGiftChestData(o *ObjectStatusData) {
	if c.Vault.MyGiftVaults == nil {
		return
	}
	c.Vault.MyGiftVaults[o.Pos] = o
}

//SwapItem takes 3 ints for parameters.
// the first 2 parameters dictate what slot is moving to what slot
// the code parameter dictates what type of container is to be used
/*
	container = bag, chest, any object that can hold items
	code = 1 -> inventory <-> inventory
	code = 2 -> inventory <-> backpack
	code = 3 -> inventory -> container
	code = 4 -> container -> inventory
	code = 5 -> backpack -> container
	code = 6 -> container -> backpack
	code = 7 -> backpack <-> backpack
	code = 8 -> container to container ???? lol ( like moving stuff around in bag)
	code = 9 -> ?????????
*/
func (c *Client) SwapItem(oldSlot byte, newSlot byte, code int, object *ObjectStatusData) {
	//object is a pointer so that we can pass nil if we aren't using any special objects to swap with
	if SinceLast(c.Times.SwapAction) <= c.Times.SwapSpeedMS { //used to be <= 500 but it's better to force it this way
		return
	}
	// if newSlot > 8 || oldSlot > 8 { //check so that we aren't attempting swaps that would crash us. MAY NOT WORK WITH BACKPACKS! ADD NEW SANITY CHECK!
	// 	return
	// }
	switch code { //sanity checks
	case 3:
		if newSlot > 7 {
			return
		}
	case 4:
		if newSlot > 11 {
			return
		}
	}
	if object == nil {
		if code == 3 || code == 4 || code == 5 || code == 6 {
			return
		}
		//otherwise it's fine if nil
	}
	e := InvSwap{}
	e.Time = c.Times.GetTime()
	e.Position = c.Moves.CurrentPosition //I don't think this would be anything else? using other object positions seems to be asking for disconnects
	old := SlotObjectData{}
	new := SlotObjectData{}
	switch code {
	case 1:
		old.ObjectID = c.ObjectID
		new.ObjectID = c.ObjectID
		old.SlotID = byte(oldSlot)
		new.SlotID = byte(newSlot)
		old.ObjectType = c.Stats.StatMap[oldSlot+INVENTORY0].StatValue
		new.ObjectType = c.Stats.StatMap[newSlot+INVENTORY0].StatValue
	case 2: //backpacks use slot id 12-19
		old.ObjectID = c.ObjectID
		new.ObjectID = c.ObjectID
		old.SlotID = byte(oldSlot)
		new.SlotID = byte(newSlot)
		if oldSlot > newSlot { //going from backpack to inventory
			old.ObjectType = c.Stats.StatMap[oldSlot+BACKPACK0-12].StatValue
			new.ObjectType = c.Stats.StatMap[newSlot+INVENTORY0].StatValue //-12 cuz backpack is +12 spaces ahead
		} else { //inventory to backpack
			old.ObjectType = c.Stats.StatMap[oldSlot+INVENTORY0].StatValue
			new.ObjectType = c.Stats.StatMap[newSlot+BACKPACK0-12].StatValue //-12 cuz backpack is +12 spaces ahead
		}
	case 3:
		old.ObjectID = c.ObjectID
		new.ObjectID = object.ObjectID
		old.SlotID = byte(oldSlot)
		new.SlotID = byte(newSlot)
		old.ObjectType = c.Stats.StatMap[oldSlot+INVENTORY0].StatValue
		new.ObjectType = object.Stats[newSlot+INVENTORY0].StatValue
	case 4:
		old.ObjectID = object.ObjectID
		new.ObjectID = c.ObjectID
		old.SlotID = byte(oldSlot)
		new.SlotID = byte(newSlot)
		old.ObjectType = object.Stats[oldSlot+INVENTORY0].StatValue
		new.ObjectType = c.Stats.StatMap[newSlot+INVENTORY0].StatValue
	case 5:
		old.ObjectID = c.ObjectID
		new.ObjectID = object.ObjectID
		old.SlotID = byte(oldSlot)
		new.SlotID = byte(newSlot)
		old.ObjectType = c.Stats.StatMap[oldSlot+BACKPACK0-12].StatValue
		new.ObjectType = object.Stats[newSlot+INVENTORY0].StatValue
	case 6:
		old.ObjectID = object.ObjectID
		new.ObjectID = c.ObjectID
		old.SlotID = byte(oldSlot)
		new.SlotID = byte(newSlot)
		old.ObjectType = object.Stats[oldSlot+INVENTORY0].StatValue
		new.ObjectType = c.Stats.StatMap[newSlot+BACKPACK0-12].StatValue
	case 7:
		old.ObjectID = c.ObjectID
		new.ObjectID = c.ObjectID
		old.SlotID = byte(oldSlot)
		new.SlotID = byte(newSlot)
		old.ObjectType = c.Stats.StatMap[oldSlot+BACKPACK0-12].StatValue
		new.ObjectType = c.Stats.StatMap[newSlot+BACKPACK0-12].StatValue
	default:
		return
	}
	e.OldSlot = old
	e.NewSlot = new
	// fmt.Println("Swap:", e) //uncomment for debugging
	c.Times.SwapAction = time.Now()
	c.Connection.Send(WritePacket(e))
}

func (c *Client) IsContainerFull(o *ObjectStatusData) bool {
	var guardIndex = 0
	if o.Stats[INVENTORY0].StatValue != -1 {
		guardIndex++
	}
	if o.Stats[INVENTORY1].StatValue != -1 {
		guardIndex++
	}
	if o.Stats[INVENTORY2].StatValue != -1 {
		guardIndex++
	}
	if o.Stats[INVENTORY3].StatValue != -1 {
		guardIndex++
	}
	if o.Stats[INVENTORY4].StatValue != -1 {
		guardIndex++
	}
	if o.Stats[INVENTORY5].StatValue != -1 {
		guardIndex++
	}
	if o.Stats[INVENTORY6].StatValue != -1 {
		guardIndex++
	}
	if o.Stats[INVENTORY7].StatValue != -1 {
		guardIndex++
	}
	if guardIndex == 8 {
		return true
	}
	return false
}

//IsVaultChestFull checks if a vault chest at the targetted position is full
//could also be used to work with bag drops and other containers if c.MyVaults is changed to something more agnostic
func (c *Client) IsVaultChestFull(targetVaultPos WorldPosData) bool {
	//in our case it'd be better to use a position as they are absolute rather than objectids which can be relative
	if len(c.Vault.MyVaults) == 0 {
		return false
	}
	c.Vault.SelectedVaultID = c.Vault.MyVaults[targetVaultPos].ObjectID
	return c.IsContainerFull(c.Vault.MyVaults[targetVaultPos])
}

func (c *Client) IsInventoryFull() bool {
	var guardIndex = 0
	if c.Stats.StatMap[INVENTORY4].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[INVENTORY5].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[INVENTORY6].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[INVENTORY7].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[INVENTORY8].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[INVENTORY9].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[INVENTORY10].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[INVENTORY11].StatValue != -1 {
		guardIndex++
	}
	if guardIndex == 8 {
		return true
	}
	return false
}

func (c *Client) IsBackpackFull() bool {
	if c.Stats.StatMap[HASBACKPACK].StatValue == 0 { //I don't think the server sends us backpack info if this isnt true
		return false
	}
	var guardIndex = 0
	if c.Stats.StatMap[BACKPACK0].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[BACKPACK1].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[BACKPACK2].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[BACKPACK3].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[BACKPACK4].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[BACKPACK5].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[BACKPACK6].StatValue != -1 {
		guardIndex++
	}
	if c.Stats.StatMap[BACKPACK7].StatValue != -1 {
		guardIndex++
	}
	if guardIndex == 8 {
		return true
	}
	return false
}

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func GetRandString(n int) string {
	b := make([]byte, n)
	for i, cache, remain := n-1, appsettings.RandSrc.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = appsettings.RandSrc.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}

//OnlinePlayer respresents another player
type OnlinePlayer struct {
	ObjectID  int32
	Position  WorldPosData
	Name      string `name`
	AccountID string `accountid`
	//below we change the data struct to our liking
	SupportPoints int `supportpoints`
}

//ShootUntilDead targets the objectid and continues to fire at it until the target is dead or a cancel request is sent
// func (c *Client) ShootUntilDead(obj Object) {
// 	//this should be in a for loop ?
// 	for obj.HP > 0 {
// 		if c.Moves.CurrentPosition.distanceTo(&obj.Data.Status.Pos) < 10 {
// 			fmt.Println("Target is within distance")
// 			angle := c.Moves.CurrentPosition.angleTo(&obj.Data.Status.Pos)
// 			atkSpeed := c.getAttackFrequency()
// 			ps := PlayerShoot{}
// 			ps.Time = c.Times.GetTime()
// 			ps.Angle = angle
// 			ps.BulletID = c.getBulletID()
// 			ps.Position = c.Moves.CurrentPosition
// 			ps.ContainerType = int16(c.Stats.StatMap[INVENTORY0].StatValue)
// 			c.Connection.Send(WritePacket(ps))
// 			time.Sleep(time.Duration(atkSpeed) * time.Millisecond)
// 		} else { //if our target is out of range then just break and let them come back into range later
// 			return
// 		}
// 	}
// }

//boosted bags redacted
func IsContainer(i uint16) bool {
	switch i {
	case htoi32("0x0501"): //treasure chest
		return true
	case htoi32("0x0504"): //vault chest
		return true
	case htoi32("0x0744"): //gift chest
		return true
	case htoi32("0x141"): //arena reward chest
		return true
	case htoi32("0x0503"): //purple bag
		return true
	case htoi32("0x0500"): //brown bag
		return true
	case htoi32("0x0506"): //pink bag?
		return true
	case htoi32("0x0507"): //cyan bag
		return true
	case htoi32("0x0508"): //blue bag
		return true
	case htoi32("0x050e"): //white bag
		return true
	case htoi32("0x0509"): //?
		return true
	case htoi32("0x050B"): //?
		return true
	case htoi32("0x50f"): //?
		return true
	case htoi32("0x6ac"): //?
		return true
	case htoi32("0x050c"): //?
		return true
	default:
		return false
	}
}

//will crash if o does not have INVENTORYn stats in teh statmap
func PrintLoot(o *ObjectStatusData) {
	if o == nil {
		return
	}
	for i := 0; i < 8; i++ {
		item := o.Stats[INVENTORY0+byte(i)].StatValue
		if item == -1 {
			continue
		}
		fmt.Println("Item:", equips[o.Stats[INVENTORY0+byte(i)].StatValue].Name)
	}
}

//ScanContainer takes the containers objectid and the desired item to search for and returns -1 if not found or the slot number if found
func (c *Client) ScanContainer(objectid int32, item int) int {
	return 0
}

func (c *Client) calculateProjectiles() {

}

func (c *Client) calculateMovement() {

}

func (c *Client) ShootTarget() {
	target := c.Game.EntityMap[c.Combat.Target].Status
	myPrimary := equips[c.Stats.StatMap[INVENTORY0].StatValue]
	if c.Moves.CurrentPosition.distanceTo(&target.Pos) > float32(myPrimary.Projectile.LifetimeMS) * (float32(myPrimary.Projectile.Speed) / 10000.0) {
		// fmt.Println("Target out of range!")
		return //cant hit if we are out of range
	}
	angleToTarget := c.Moves.CurrentPosition.angleTo(&target.Pos)
	c.Shoot(angleToTarget)
	//need to make sure we are right on top

	// const keys = Object.keys(this.enemies);
	//   const projectile = ResourceManager.items[this.playerData.inventory[0]].projectile;
	//   const distance = projectile.lifetimeMS * (projectile.speed / 10000);
	//   for (const key of keys) {
	//     const enemy = this.enemies[+key];
	//     if (enemy.squareDistanceTo(this.worldPos) < distance ** 2) {
	//       const angle = Math.atan2(enemy.objectData.worldPos.y - this.worldPos.y, enemy.objectData.worldPos.x - this.worldPos.x);
	//       this.shoot(angle);
	//     }
	//   }
	// }
}
