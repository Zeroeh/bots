package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	rootURL = "https://realmofthemadgodhrd.appspot.com/"
)

//Client is the main struct for a bot
type Client struct {
	Base        *Account
	Connection  GameConnection
	Running     bool
	Debugging   bool
	ReadUpdates bool //if we want to parse update packets

	ObjectID   int32
	CurrentMap string
	InGameName string

	Recon        ReconBase
	Moves        MovementBase
	Times        TimeKeeper
	Master       MasterBase
	Stats        StatBase
	Combat       CombatBase
	Game         GameObjects
	Trade        TradeBase
	Misc         MiscBase
	Mod          ModuleBase
	Vault        VaultBase
	GeneralMutex sync.Mutex
}

//Start is called once the accounts have been filled into the struct from json format; this function should never return
func (c *Client) Start(gid int32, key []byte, keytime int32) {
	//Start initializes all client "static" variables and initates the clients core loop. This function shouldn't really return unless the bot gets completely shut down
	//Note that these variables can be adjusted at any point if needed. This is just to initialize them
	c.Times.NewClock()
	c.ClearMaps() //init maps
	//c.BlockRecon = true
	c.Running = true
	c.Recon.GameID = gid
	c.Recon.GameKey = key
	c.Recon.GameKeyTime = keytime
	c.Recon.ReconWaitMultiplier = uint64(settings.ReconDelay)
	c.Recon.ReconAllowedAttempts = 3
	c.Running = true
	c.Times.SwapSpeedMS = 760
	if c.Base.Module != "nil" {
		c.Moves.MoveMultiplier = 0.9 //dont want bots dcing on high priority tasks
	} else {
		c.Moves.MoveMultiplier = 1.0
	}
	c.GameLoop()
}

//GameLoop is the "main()" of our client. Keeps our bot connected and running.
func (c *Client) GameLoop() {
	for c.Running == true { //higher abstraction of client running
		//check here if we are connected and reconnect behavior
		if c.Connection.Connected == false {
			if c.Recon.ReconQueued == true {
				if c.Recon.ReconCheck() == true {
					fmt.Printf("%s is over the recon limit, killing client...\n", c.Base.Email)
					c.KillClient()
					return
				}
				time.Sleep(time.Second * time.Duration(c.Recon.ReconWaitMultiplier))
			}
			c.InitGameConnection()        //todo: return an error and use an else statement to send hello
			c.Connection.Connected = true //connected just means an open socket connection, NOT sitting in the actual game
			c.Recon.ReconQueued = false
			c.SendHello(c.Recon.GameKey, c.Recon.GameKeyTime) //-2 nexus : -5 vault : 0 realm
		}
		for c.Connection.Connected == true {
			if c.Recon.ReconQueued == true {
				//c.Connection.InLoop = false //so we start receive again
				c.Disconnect() //this SHOULD be the only place that the connection gets killed, unless a forceful kill is needed in a module
				continue
			}
			if c.Connection.InLoop == false && c.Connection.Connected == true { //will be false on first starts
				c.Connection.InLoop = true //immediately set true so we dont have multiple receive calls
				go c.Receive()
			} else if c.Connection.Killed == true && c.Connection.Connected == true { //this is run if we called KillSocket() or get forcefully disconnected
				if c.Recon.ReconnectOnError == true {

				} else {
					c.Connection.Connected = false
					c.Running = false
				}
			}
			time.Sleep(time.Millisecond * time.Duration(settings.ThreadDelay) * time.Duration(settings.FPS)) //new counter, higher delay = less cpu used but more response delay in bot (such as movement and projectiles)
			c.calculateMovement()
			c.calculateProjectiles()
			//if this were a normal client we would do our gui calls in this loop
		}
	}
}

//QueueRecon sets the client "ReconQueue" to true and
func (c *Client) QueueRecon(gid int32, key []byte, keytime int32) {
	if c.Recon.BlockingReconnects == false {
		c.Recon.ReconQueued = true
		c.Recon.GameID = gid
		c.Recon.GameKey = key
		c.Recon.GameKeyTime = keytime
		//now do some cleanups based on module
		switch c.Base.Module {
		case "dupe":
			c.Mod.DupeMoveIndex = 0
		case "vaultunlock":
			c.Mod.DupeMoveIndex = 0
			c.Mod.LowIndex = 0
			c.Mod.HighIndex = 0
		case "vaultbegin":
			c.Mod.Phase = -1
			c.Mod.DupeMoveIndex = 0
			c.Mod.LowIndex = 0
			c.Mod.HighIndex = 0
		}
		c.Recon.Increment(false)
		c.ClearMaps()
	}
}

//ParseUpdateData parses all the data in the update packet
// it parses in this order: objects, drops, tiles
func (c *Client) ParseUpdateData(u *Update) {
	if len(u.Tiles) > 0 {
		for z := 0; len(u.Tiles) > z; z++ {
			c.Moves.Tiles[u.Tiles[z]] = u.Tiles[z].Type
		}
	}
	if len(u.NewObjs) > 0 {
		for x := 0; len(u.NewObjs) > x; x++ {
			if u.NewObjs[x].Status.ObjectID == c.ObjectID {
				c.Moves.CurrentPosition = u.NewObjs[x].Status.Pos
				if c.CurrentMap == "Nexus" && c.Base.Module != "filter" { //this will "teleport" you to a designated coordinate. works within 15 tiles of spawn position, ignoring speed and possible barriers
					// c.Moves.CurrentPosition = nexusVaultPortal
					// c.Moves.CurrentPosition = 
				}
				if c.CurrentMap == "Shatters" || c.CurrentMap == "The Shatters" {
					c.Moves.CurrentPosition = shattersNewSpawn
				}
				c.Moves.TargetPosition = c.Moves.CurrentPosition //if we dont set this the bot will move towards 0,0 when loading in
				//fmt.Println("Pos:", c.CurrentPosition)
				c.Moves.LastPosition = c.Moves.CurrentPosition
				//populate our stats
				for k := range u.NewObjs[x].Status.Stats {
					c.Stats.StatMap[k] = u.NewObjs[x].Status.Stats[k]
				}
				c.InGameName = c.Stats.StatMap[NAME].StrStatValue
			}
			if IsContainer(u.NewObjs[x].ObjectType) {
				c.Game.Containers = append(c.Game.Containers, u.NewObjs[x].Status.ObjectID)
			}
			if u.NewObjs[x].ObjectType == vaultChest { //open vault chest
				c.GetChestData(&u.NewObjs[x].Status)
			} else if u.NewObjs[x].ObjectType == vaultChestClosed {

			}
			// if u.NewObjs[x].ObjectType == vaultPortal
			if u.NewObjs[x].ObjectType == giftChest { //open gift chest
				c.GetGiftChestData(&u.NewObjs[x].Status)
			}
			if u.NewObjs[x].Status.FindStat(NAME).StrStatValue == c.Master.DoxName { //works as intended
				c.Master.DoxID = int(u.NewObjs[x].Status.ObjectID)
				// c.Master.DoxStats = u.NewObjs[x].Status.Stats
			}
			// fmt.Println(u.NewObjs[x])
			c.Game.EntityMap[u.NewObjs[x].Status.ObjectID] = u.NewObjs[x]
		}
	}
	if len(u.Drops) > 0 {
		for y := 0; len(u.Drops) > y; y++ {
			delete(c.Game.EntityMap, u.Drops[y])
		}
	}
}

//ParseNewTickData gets all the updated stats of players
func (c *Client) ParseNewTickData(n *NewTick) {
	for i := 0; len(n.Statuses) > i; i++ {
		if n.Statuses[i].ObjectID == c.ObjectID { //get our stats
			c.Moves.ServerPosition = n.Statuses[i].Pos
			if len(n.Statuses[i].Stats) > 0 { //sanity check: only update if we got new stats
				for k := range n.Statuses[i].Stats {
					c.Stats.StatMap[k] = n.Statuses[i].Stats[k]
				}
			}
		} else if n.Statuses[i].ObjectID == c.Master.MasterID && c.Master.MasterID != 0 && c.Master.FollowMaster == true {
			c.Master.MasterPos = n.Statuses[i].Pos
		} else if n.Statuses[i].ObjectID == int32(c.Master.DoxID) {
			// fmt.Println(n.Statuses[i].Stats)
		}
		chest, ok := c.Vault.MyVaults[n.Statuses[i].Pos]
		if ok {
			//might have to iterate for individual slots if they all dont get sent at once
			// fmt.Println("New vault stat:", n.Statuses[i])
			for k := range n.Statuses[i].Stats {
				chest.Stats[k] = n.Statuses[i].Stats[k]
			}
			c.Vault.MyVaults[n.Statuses[i].Pos] = chest
		}
		obj := c.Game.EntityMap[n.Statuses[i].ObjectID]
		obj.Status.Pos = n.Statuses[i].Pos
		//update or insert items
		for k := range n.Statuses[i].Stats {
			obj.Status.Stats[k] = n.Statuses[i].Stats[k]
		}
		//since obj isn't a pointer, reassign to the entitymap
		c.Game.EntityMap[n.Statuses[i].ObjectID] = obj
	}
}

func (c *Client) KillClient() {
	fmt.Printf("Killing %s...\n", c.Base.Email)
	c.Recon.ReconQueued = false
	c.Running = false
	c.Disconnect()
}

func (c *Client) Disconnect() {
	c.Connection.KillConnection()
}

func (c *Client) IsConnected() bool {
	return c.Connection.Connected
}

//ClearMaps clears all the maps held by the client
func (c *Client) ClearMaps() {
	c.Game.EntityMap = make(map[int32]ObjectData)
	c.Moves.Tiles = make(map[GroundTile]uint16)
	c.Moves.Targets = make(map[uint]WorldPosData) //typically we shouldnt clear this
	c.Stats.StatMap = make(map[byte]StatData)
	c.Master.DoxStats.StatMap = make(map[byte]StatData)
	// fmt.Println("Before:", c.Vault.MyVaults)
	c.Vault.MyVaults = make(map[WorldPosData]*ObjectStatusData)
	// fmt.Println("After:", c.Vault.MyVaults)
	c.Vault.MyGiftVaults = make(map[WorldPosData]*ObjectStatusData)
	c.Game.Containers = make([]int32, 0)
}

func (c *Client) HandleFailure(f *Failure) {
	//note: return is required after any queuerecon so that the behaviors dont fall through to the else statement
	//overall, this function needs to be redone in a much nicer way
	if c.Base.Module == "filter" {
		c.Mod.LogFile.Close()
	}
	if f != nil {
		if strings.Contains(f.FailureMessage, "Character not found") == true {
			//c.Connection.Kill(c)
			//c.Base.FetchNewData = true
			//c.Start(gid, []byte{}, GetUnixTime())
			if c.Base.Module == "vaultbegin" { //probably dont have unlockers
				c.KillClient()
			}
			return
		}
		if strings.Contains(f.FailureMessage, "Character is dead") == true {
			c.Base.CharID = 0 //so we run create instead of load
			c.QueueRecon(c.Recon.GameID, []byte{}, GetUnixTime())
			return
		}
		if strings.Contains(f.FailureMessage, "Account is under maintenance") == true {
			c.KillClient()
			return
		}
		if strings.Contains(f.FailureMessage, "The transaction could not be committed. Please try again") == true {
			return
		}
		if strings.Contains(f.FailureMessage, "Account already has ") == true {
			//c.Base.CharID = 0
			//c.Base.FetchNewData = true
			c.Base.CharID++
			c.QueueRecon(c.Recon.GameID, []byte{}, GetUnixTime())
			return
		}
		if strings.Contains(f.FailureMessage, "Sorry, the player limit has been reached") == true {
			time.Sleep(2000 * time.Millisecond)
			c.QueueRecon(c.Recon.GameID, []byte{}, GetUnixTime())
			return
		}
		if strings.Contains(f.FailureMessage, "Protocol error") == true {
			if c.Base.Module != "dupe" {
				c.QueueRecon(c.Recon.GameID, []byte{}, GetUnixTime())
				return
			}
		}
		if strings.Contains(f.FailureMessage, "Invalid connection guid, you want to get banned?") == true {
			fmt.Println("Bad uuid")
			return
		}
		if strings.Contains(f.FailureMessage, "Lack of pongs") == true {
			fmt.Println("Lack of pongs??? wtf???")
			return
		}
		if strings.Contains(f.FailureMessage, "Not allowed in the game") == true {
			//apparently this is some sort of 3 hour temp ban if you play continuously for 8 hours?? idk tho
			log.Println("Got 8 hour ban message? Attempting to recon", c.Base.Email)
			time.Sleep(2000 * time.Millisecond)
			c.QueueRecon(-2, []byte{}, GetUnixTime()) //attempt to connect to nexus
			return
		}
		if strings.Contains(f.FailureMessage, "Server restarting") == true {
			time.Sleep(5000 * time.Millisecond)
			c.QueueRecon(-2, []byte{}, GetUnixTime()) //just restart in the nexus
			return
		}
		if strings.Contains(f.FailureMessage, "Lost connection to server") == true {
			if c.Base.Module != "dupe" {
				if c.Base.Module == "receive" { //we always rerun the app to make sure the bots are filled anyways so this doesnt matter as much
					return
				}
				c.QueueRecon(c.Recon.GameID, []byte{}, GetUnixTime())
				return
			}
		}
		if strings.Contains(f.FailureMessage, "Bad message received") == true { //bad packet id / bad hello packet??

		}
		if strings.Contains(f.FailureMessage, "Too many concurrent connections. Max allowed: 5.") == true {
			time.Sleep(3000 * time.Millisecond)
			c.QueueRecon(c.Recon.GameID, c.Recon.GameKey, c.Recon.GameKeyTime)
			return
		}
		if strings.Contains(f.FailureMessage, "Account credentials not valid") == true {
			return
		}
		if strings.Contains(f.FailureMessage, "Account in use") == true {
			//deca removed the seconds in the account in use string so we cant use that anymore
			// index1 := strings.Index(f.FailureMessage, "use ")
			// index2 := strings.Index(f.FailureMessage, " second")
			// timeLeftString := f.FailureMessage[index1+5 : index2]
			// timeLeft, err := strconv.Atoi(timeLeftString)
			// if err != nil {
			// 	timeLeft = -1
			// }
			timeLeft := 0
			//fmt.Println("Seconds left:", timeLeft)
			if timeLeft != -1 {
				if c.Base.Module != "dupe" && c.Base.Module != "vaultbegin" && c.Base.Module != "vaultunlock" {
					fmt.Printf("%s: Reconnecting in %d seconds...\n", c.Base.Email, timeLeft)
					time.Sleep(time.Duration(timeLeft) * time.Second)                  //sleep BEFORE queueing
					c.QueueRecon(c.Recon.GameID, c.Recon.GameKey, c.Recon.GameKeyTime) //set to use whatever we had previously incase were entering a dungeon
					//if accounts in use runs for too long the dungeon key will expire so we should try going back to nexus after some time
					return
				} else {
					time.Sleep(3000 * time.Millisecond) //wait an additional few seconds as we know it's going to be a few tries
					c.Recon.ReconAttempts--
					c.QueueRecon(c.Recon.GameID, []byte{}, GetUnixTime())
					return
				}
			}
		}
		//if no other error statements are caught then we move to the general catch all else statement
	} else {
		if c.Base.Module != "receive" { //causes the bots to reconnect after filling up
			c.QueueRecon(c.Recon.GameID, []byte{}, GetUnixTime())
			return
		}
	}
}

//moveTo uses an algorithm to adjust the clients move speed accordingly
func (c *Client) moveTo(target WorldPosData) {
	/*if target.outOfBounds(math.Float32frombits(uint32(c.MapWidth))) {
		return c.CurrentPosition
	}*/
	if c.hasEffect(PAUSED) == true {
		//weird that the game server does it like this...
		c.Moves.LastPosition = c.Moves.CurrentPosition
		c.Moves.CurrentPosition = WorldPosData{ //actual client uses -1 but this is really funny! any value below 0 will work when paused
			X: -99999999999999999999999999999999999999,
			Y: -99999999999999999999999999999999999999,
		}
		if c.Moves.ResetPos == false {
			c.Moves.ResetPos = true //set to reset once we unpause
		}
		return //dont calculate the rest as we cant move when paused
	} else if c.hasEffect(PAUSED) == false && c.Moves.ResetPos == true {
		c.Moves.CurrentPosition = c.Moves.ServerPosition //set our position back to what the server has us at
		c.Moves.ResetPos = false
		return
	}
	floatCurrentTime := math.Float32frombits(uint32(c.Times.CurrentTickTime))
	floatLastTime := math.Float32frombits(uint32(c.Times.LastTickTime))
	tmp := WorldPosData{}
	var elapsed float32
	if floatCurrentTime-floatLastTime > 200.0 {
		elapsed = floatCurrentTime - floatLastTime
	} else {
		elapsed = 200.0
	}
	var step = c.getMoveSpeed() * elapsed

	if c.Moves.CurrentPosition.sqDistanceTo(&target) > step*step {
		var angle = c.Moves.CurrentPosition.angleTo(&target)
		tmp.X = c.Moves.CurrentPosition.X + f32cos(angle)*step
		tmp.Y = c.Moves.CurrentPosition.Y + f32sin(angle)*step
	} else {
		tmp = target
	}
	c.Moves.LastPosition = c.Moves.CurrentPosition
	c.Moves.CurrentPosition = tmp
}

//Check if a coordinate is out of bounds
func (p *WorldPosData) outOfBounds(width float32) bool {
	return p.X < 0 || p.Y < 0 || p.X > width || p.Y > width
}

func (c *Client) getMoveSpeed() float32 {
	if c.hasEffect(SLOWED) == true {
		return MinMoveSpeed
	}
	// moveSpd := MinMoveSpeed + math.Float32frombits(uint32(c.Stats.StatMap[SPEED].StatValue))/75.0*(MaxMoveSpeed-MinMoveSpeed)
	moveSpd := MinMoveSpeed + float32(c.Stats.StatMap[SPEED].StatValue)/75*(MaxMoveSpeed-MinMoveSpeed)
	if c.hasEffect(SPEEDY) == true || c.hasEffect(NINJASPEEDY) == true {
		moveSpd *= 1.5
	}
	return moveSpd * c.GetPosTileMoveMult() * c.Moves.MoveMultiplier
}

func (c *Client) GetPosTileMoveMult() float32 {
	for i := range c.Moves.Tiles {
		if c.IsOnTile(i) {
			if tiles[int(i.Type)].Speed != 0.0 { //make sure we dont set our speed to 0
				return tiles[int(i.Type)].Speed
			}
			return 1.0
		}
	}
	return 1.0
}

func (c *Client) IsOnTile(t GroundTile) bool {
	pos := WorldPosData{
		X: float32(t.X),
		Y: float32(t.Y),
	}
	if c.Moves.CurrentPosition.distanceTo(&pos) < 1 {
		return true
	}
	return false
}

func (c *Client) rotateClockwise(w WorldPosData) {
	c.Moves.MoveArc += .1
	c.Moves.TargetPosition.X = w.X + f32cos(c.Moves.MoveArc)*c.Moves.Radius
	c.Moves.TargetPosition.Y = w.Y + f32sin(c.Moves.MoveArc)*c.Moves.Radius
}

func (c *Client) rotateCounterClockwise(w WorldPosData) {
	c.Moves.MoveArc += .1
	c.Moves.TargetPosition.X = w.X + f32sin(c.Moves.MoveArc)*c.Moves.Radius
	c.Moves.TargetPosition.Y = w.Y + f32cos(c.Moves.MoveArc)*c.Moves.Radius
}

func (c *Client) getAttackMultiplier() float32 {
	if c.hasEffect(WEAK) == true {
		return MinAttackMult
	}
	// atkMult := MinAttackMult + math.Float32frombits(uint32(c.Stats.StatMap[ATTACK].StatValue))/75*(MaxAttackMult-MinAttackMult)
	atkMult := MinAttackMult + float32(c.Stats.StatMap[ATTACK].StatValue)/75*(MaxAttackMult-MinAttackMult)
	if c.hasEffect(DAMAGING) == true {
		atkMult *= 1.5
	}
	return atkMult
}

func (c *Client) getAttackFrequency() float32 {
	if c.hasEffect(DAZED) == true {
		return MinAttackFreq
	}
	// atkFreq := MinAttackFreq + math.Float32frombits(uint32(c.Stats.StatMap[DEXTERITY].StatValue))/75*(MaxAttackFreq-MinAttackFreq)
	atkFreq := MinAttackFreq + float32(c.Stats.StatMap[DEXTERITY].StatValue)/75*(MaxAttackFreq-MinAttackFreq)
	if c.hasEffect(BERSERK) == true {
		atkFreq *= 1.5
	}
	return atkFreq
}

//Shoot sends playershoot at the desired angle
func (c *Client) Shoot(angle float32) {
	if c.hasEffect(STUNNED) == true || c.hasEffect(PAUSED) == true {
		return
	}
	time := c.Times.GetTime()
	item := equips[c.Stats.StatMap[INVENTORY0].StatValue]
	attackPeriod := 1 / c.getAttackFrequency() * (1 / item.RateOfFire)
	numProjectiles := 1
	if item.NumProjectiles > 0 {
		numProjectiles = item.NumProjectiles
	}
	if time < c.Combat.LastAttackTime+int32(attackPeriod) {
		return
	}
	c.Combat.LastAttackTime = time
	arcRads := float32(item.ArcGap / 180 * math.Pi)
	totalArc := arcRads * float32((numProjectiles - 1))
	if arcRads <= 0 {
		totalArc = 0
	}
	angle -= totalArc / 2
	for i := 0; i < numProjectiles; i++ { //replace 1 with item.numprojectiles
		ps := PlayerShoot{}
		ps.Time = time
		ps.BulletID = c.getBulletID()
		ps.Angle = angle
		ps.ContainerType = int16(item.ID) //item.type aka the item hex id
		ps.Position = c.Moves.CurrentPosition
		ps.Position.X += f32cos(angle) * 0.3
		ps.Position.Y += f32sin(angle) * 0.3
		c.Connection.Send(WritePacket(ps))
		pj := GameProjectile{}
		pj.ContainerType = int32(ps.ContainerType)
		pj.BulletType = 0
		pj.OwnerObjectID = c.ObjectID
		pj.BulletID = ps.BulletID
		pj.StartAngle = angle
		pj.StartTime = time
		pj.StartPosition.X = ps.Position.X
		pj.StartPosition.Y = ps.Position.Y
		c.Combat.PushProjectile(pj)
		if arcRads > 0 {
			angle += arcRads
		}
		projectile := item.Projectile
		damage := GetIntInRange(projectile.MinDamage, projectile.MaxDamage)
		if time > c.Moves.LastClearTime+600 { //not sure why this is needed
			damage = 0
		}
		c.Combat.Projectiles[len(c.Combat.Projectiles)-1].setDamage(damage * int(c.getAttackMultiplier()))
	}
	// c.CheckProjectile()
}

//getBulletID gets the next available bullet id
func (c *Client) getBulletID() byte {
	bID := c.Combat.CurrentBulletID
	c.Combat.CurrentBulletID = (c.Combat.CurrentBulletID + 1) % 128
	return bID
}

//DamageMe damages the object id
func (c *Client) DamageMe(damage int16, bulletID byte, objectID int32) {
	ph := PlayerHit{}
	ph.BulletID = bulletID
	ph.ObjectID = objectID
	c.Connection.Send(WritePacket(ph))
}

//no longer works
func (c *Client) suicide(target WorldPosData) {
	gd := GroundDamage{}
	gd.Position = target
	for i := 0; i < 200; i++ {
		gd.Time = c.Times.GetTime()
		c.Connection.Send(WritePacket(gd))
	}
}

//hasEffect checks if the player has the specified condition effect
func (c *Client) hasEffect(status uint) bool {
	//check if our status is greater than 31 if so, use EFFECTS2 from statdata
	var condition int
	var effectBit int
	if status > 31 {
		condition = int(c.Stats.StatMap[EFFECTS2].StatValue)
		effectBit = 1 << (status - 32)
		ok := (condition & effectBit) == effectBit
		return ok
	} else {
		condition = int(c.Stats.StatMap[EFFECTS].StatValue)
		effectBit = 1 << (status - 1)
		ok := condition&effectBit == effectBit
		return ok
	}
}

//consumeItem consumes a consumeable. CONSUME!
func (c *Client) consumeItem(slotobj SlotObjectData) {
	ui := UseItem{}
	ui.Time = c.Times.GetTime()
	ui.Position = c.Moves.CurrentPosition //current position as this is for consuming items, not using ability
	ui.UseType = 0
	/* usetype enum:
	default = 0
	start use = 1
	end use = 2
	1 and 2 = shit like ninja ability
	*/
	ui.SlotObject = slotobj
	c.Connection.Send(WritePacket(ui))
}

func (c *Client) ChangeModule(mod string, gameid int) {
	SwitchColor(Blue)
	fmt.Printf("%s change mod: %s -> %s\n", c.Base.Email, c.Base.Module, mod)
	SwitchColor(Normal)
	c.Base.Module = mod
	if mod == "vaultunlock" {
		c.Mod.Phase = -1
	} else {
		c.Mod.Phase = 0
	}
	c.QueueRecon(int32(gameid), []byte{}, GetUnixTime())
}

//DelayAction takes a sleep time in ms and the function to delay execution
func (c *Client) DelayAction(ms int, f func()) {
	go func() {
		time.Sleep(time.Millisecond * time.Duration(ms))
		f()
	}()
}

//this is only meant to be used on the VPS where everything is structured properly
func (c *Client) blackListIP() {
	f, err := os.OpenFile("../../goscripts/blacklist.txt", os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Println("Error opening blacklist:", err)
		return
	}
	defer f.Close()
	proxy := c.Base.SockProxy
	_, err = f.Write([]byte(proxy[:strings.Index(proxy, ":")]))
	if err != nil {
		fmt.Println("Error writing to blacklist:", err)
	}
}

func (c *Client) getURL(rx string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, rx, nil)
	if err != nil {
		log.Println("getURL:", err)
	}
	req.Header.Set("User-Agent", getRandomHeader())
	if c.Base.UseHTTP == true {
		var err error
		proxyURL, err := url.Parse(c.Base.HTTPProxy)
		if err != nil {
			return "", err
		}
		myClient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
		myClient.Timeout = time.Second * 5
		ex, err := myClient.Do(req)
		if err != nil {
			return "", err
		}
		body, err := ioutil.ReadAll(ex.Body) //don't think we'll error if we got to here
		if err != nil {
			return "", err
		}
		ex.Body.Close()
		return string(body), nil
	} else {
		//no proxy, use actual ip
		myClient := &http.Client{}
		myClient.Timeout = time.Second * 5
		ex, err := myClient.Do(req)
		if err != nil {
			return "", err
		}
		body, err := ioutil.ReadAll(ex.Body)
		if err != nil {
			return "", err
		}
		ex.Body.Close()
		return string(body), nil
	}
}

func (c *Client) WriteText(message string) {
	ptext := PlayerText{}
	ptext.Message = message
	c.Connection.Send(WritePacket(ptext))
}

func (c *Client) HandleText(t *Text) {
	if c.Base.Module == "filter" {
		if t.Stars == -1 || t.ObjectID == -1 {
			return
		}
		if t.Recipient == c.InGameName {
			return
		}
		c.ModuleFilterMain(t)
	}
	if c.Base.Module == "rankdupe" {
		if t.Message == "startit" {
			c.Master.DoxName = t.Name
			c.Mod.Phase = 1
		}
		// if t.Message == "testit" {
		// 	fmt.Println("Received gchat lmao")
		// }
	}
	splitMessage := strings.Split(t.Message, " ") //get args
	argCount := len(splitMessage) - 1
	if t.Recipient == c.InGameName {
		switch splitMessage[0] {
		case "update":
			fmt.Println(c.Stats.StatMap)
		case "alive": //debug check to see if the receiver is still handling packets
			c.WriteText("/t Artosh Alive!")
		case "bomb": //spellbombs the receivers position
			ui := UseItem{}
			ui.Time = c.Times.GetTime()
			so := SlotObjectData{}
			so.ObjectID = c.ObjectID
			so.SlotID = 1
			so.ObjectType = c.Stats.StatMap[INVENTORY1].StatValue
			ui.SlotObject = so
			ui.Position = c.Moves.CurrentPosition
			ui.UseType = equips[c.Stats.StatMap[INVENTORY1].StatValue].SlotType //i assume this is the type of ability
			c.Connection.Send(WritePacket(ui))
		case "shingeki": //crashes the receivers' current server
			user := t.Name
			m := "/tell " + user + " Unfortunately the Heaven's Strike is not implemented yet."
			fmt.Printf("%s attempted to use Heaven's Strike\n", t.Name)
			c.WriteText(m)
		case "nexus": //nexus the receiver. use "Recon" instead for fast nexus
			n := Escape{}
			c.Connection.Send(WritePacket(n))
		case "dc": //permanently disconnects the bot
			c.KillClient()
		case "useitem":
			if len(splitMessage) == 2 {
				itembyte := byte(atoi(splitMessage[1]))
				objt := c.Stats.StatMap[INVENTORY0+itembyte].StatValue
				if objt == -1 {
					return
				}
				ui := UseItem{}
				ui.Time = c.Times.GetTime()
				ui.Position = c.Moves.CurrentPosition

				ui.UseType = 0
				so := SlotObjectData{}
				so.ObjectID = c.ObjectID
				so.SlotID = itembyte
				so.ObjectType = c.Stats.StatMap[INVENTORY0+itembyte].StatValue
				ui.SlotObject = so
				c.Connection.Send(WritePacket(ui))
				fmt.Println("Consumed", equips[c.Stats.StatMap[INVENTORY0+itembyte].StatValue].Name)
			} else {
				fmt.Println("Not enough args to this cmd")
			}
		case "containers":
			if len(c.Game.Containers) == 0 {
				return
			}
			for i := 0; i < len(c.Game.Containers); i++ {
				if c.Game.EntityMap[c.Game.Containers[i]].ObjectType == 0 {
					continue
				}
				if c.Game.EntityMap[c.Game.Containers[i]].ObjectType != 1280 {
					e := c.Game.EntityMap[c.Game.Containers[i]].Status
					PrintLoot(&e)
				}
			}
		case "dox":
			if argCount != 1 {
				return
			}
			objid, err := strconv.Atoi(splitMessage[1])
			if err != nil {
				return
			}
			fmt.Println("Doxxing", objid)
			c.Master.DoxID = objid
		case "follow": //follows the sender
			if c.Master.FollowMaster == true {
				c.Master.FollowMaster = false
				return
			}
			c.Master.MasterID = t.ObjectID
			c.Master.FollowMaster = true
			c.Master.MasterPos = c.Game.EntityMap[t.ObjectID].Status.Pos
		case "master":
			c.Master.MasterID = t.ObjectID
			c.Master.MasterName = t.Name
			c.Master.MasterPos = c.Game.EntityMap[t.ObjectID].Status.Pos
		case "getdox": //prints targetted players stats to console
			fmt.Println("Dox stats:", c.Master.DoxStats)
		case "recon": //reconnects to nexus
			c.QueueRecon(-2, []byte{}, GetUnixTime())
		case "rank": //changes guild rank
			cr := ChangeGuildRank{}
			cr.GuildRank = 4
			cr.Name = t.Name
			c.Connection.Send(WritePacket(cr))
		case "pause": //pauses the receiver
			c.WriteText("/pause")
		case "killall": //kills the app
			os.Exit(0)
		case "teleport": //teleport to the sender
			tp := Teleport{}
			tp.ObjectID = t.ObjectID
			c.Connection.Send(WritePacket(tp))
		case "teleportf": //teleport to the sender and set them as master
			tp := Teleport{}
			tp.ObjectID = t.ObjectID
			c.Master.MasterID = t.ObjectID
			c.Connection.Send(WritePacket(tp))
		case "private": //privates the receivers profile on realmeye
			m := PlayerText{}
			m.Message = "/t mreyeball private profile"
			c.Connection.Send(WritePacket(m))
		case "gg": //basic packet spam
			c.Recon.BlockingReconnects = true
			n := UsePortal{}
			n.ObjectID = 14319 //vault portal
			for i := 0; i < 30000; i++ {
				c.Connection.Send(WritePacket(n))
			}
		case "gg2":
			ui := InvDrop{}
			so := SlotObjectData{}
			so.ObjectID = 888888
			so.SlotID = 255
			so.ObjectType = c.Stats.StatMap[INVENTORY1].StatValue
			ui.SlotObject = so

			for i := 0; i < 100; i++ {
				c.Connection.Send(WritePacket(ui))
			}
			// c.Connection.Send(WritePacket(ui))
			fmt.Println("Sent boom!")
		case "stats":
			fmt.Println(c.Game.EntityMap[c.ObjectID])
		case "checkvault":
			fmt.Println("Chest full:", c.IsVaultChestFull(tutorialFirstChestLoc))
		case "checkinv":
			fmt.Println("Inv full:", c.IsInventoryFull())
		case "checkpack":
			fmt.Println("Backpack full:", c.IsBackpackFull())
		case "get":
			if len(splitMessage) != 2 {
				return
			}
			objid := int32(atoi(splitMessage[1]))
			obj := c.Game.EntityMap[objid]
			fmt.Println(obj)
		case "dump":
			for i := range c.Game.EntityMap {
				if c.ObjectID == i {
					fmt.Println("<skipping self, use 'stats' for bots' stats>")
					continue
				}
				fmt.Printf("Obj <%d>: %v\n", i, c.Game.EntityMap[i])
			}
		case "drop":
			if len(splitMessage) != 2 {
				return
			}
			if splitMessage[1] == "" {
				return
			}
			slot := byte(atoi(splitMessage[1]))
			if strconv.Itoa(int(slot)) != splitMessage[1] {
				return
			}
			e := InvDrop{}
			e.SlotObject = SlotObjectData{
				ObjectID:   c.ObjectID,
				SlotID:     slot,
				ObjectType: c.Stats.StatMap[INVENTORY0+slot].StatValue,
			}
			c.Connection.Send(WritePacket(e))
			fmt.Println("Dropped", splitMessage[1])
		case "swap": //equip/dequip weapon
			if c.Stats.StatMap[INVENTORY5].StatValue == -1 {
				c.SwapItem(0, 4, 1, nil)
			} else {
				c.SwapItem(4, 0, 1, nil)
			}
		case "letsgo": //for dupe bots, allows them to accept the trade
			c.Trade.AllowedAccept = true
		case "shoot":
			if c.Combat.ShootingEnabled == true {
				c.Combat.ShootingEnabled = false
				fmt.Println("Shooting disabled")
			} else {
				c.Combat.ShootingEnabled = true
				fmt.Println("Shooting enabled")
			}

			// 	pos := c.Game.GetObjByID(t.ObjectID).Status.Pos
			// 	c.Shoot(c.Moves.CurrentPosition.angleTo(&pos))

			// c.Shoot(90)
		case "shootat":
			if len(splitMessage) != 2 {
				return
			}
			c.Combat.Target = int32(atoi(splitMessage[1]))
		case "exp": //duplicate of "gg"
			amt := 100000
			fmt.Printf("x is %d setcond\n", amt)
			e := SetCondition{}
			e.ConditionEffect = 255
			e.ConditionDuration = 1.0
			for x := 0; x < amt; x++ {
				c.Connection.Send(WritePacket(e))
			}
			fmt.Println("Sent batch")
		}
	}
}

var (
	vaultSpawnPoint = WorldPosData{
		X: 43.5,
		Y: 72.5,
	}
	nexusCenter = WorldPosData{ //summer beach nexus
		X: 134,
		Y: 140,
	}
	nexusVaultPortal = WorldPosData{
		X: 110.5,
		Y: 166.5,
	}
	nameChanger = WorldPosData{
		X: 103.5,
		Y: 161.5,
	}
	firstVaultChestLoc = WorldPosData{
		X: 44.5,
		Y: 70.5,
	}
	tutorialFirstChestLoc = WorldPosData{
		X: 102.5,
		Y: 138.5,
	}
	nexusSpawnpadCorner0 = WorldPosData{
		X: 111.5,
		Y: 167.5,
	}
	nexusSpawnpadCorner1 = WorldPosData{
		X: 102.5,
		Y: 167.5,
	}
	nexusSpawnpadCorner2 = WorldPosData{
		X: 102.5,
		Y: 160.5,
	}
	nexusSpawnpadCorner3 = WorldPosData{
		X: 111.5,
		Y: 160.5,
	}
	otherTarget = WorldPosData{
		X: 1126.5,
		Y: 1228.5,
	}
	shattersNewSpawn = WorldPosData{
		X: 53.5,
		Y: 363.5,
	}
)

const (
	MinMoveSpeed  float32 = 0.0041
	MaxMoveSpeed  float32 = 0.00961
	MinAttackFreq float32 = 0.0015
	MaxAttackFreq float32 = 0.008
	MinAttackMult float32 = 0.5
	MaxAttackMult float32 = 2.0
)
