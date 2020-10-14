package main

import (
	"bufio"
	"os"
	"time"
)

type MovementBase struct {
	CurrentPosition WorldPosData
	TargetPosition  WorldPosData
	LastPosition    WorldPosData
	ServerPosition  WorldPosData
	MoveMultiplier  float32
	TickCount       uint64
	LastTickID      int32
	ResetPos        bool
	MapWidth        int32
	MapHeight       int32
	MoveArc         float32
	Radius          float32
	TargetCode      uint
	Paused          bool
	Records         MoveRecord
	LastClearTime   int32
	Targets         map[uint]WorldPosData
	Tiles           map[GroundTile]uint16
}

func (m *MovementBase) asd() {

}

type ReconBase struct {
	CurrentServer        string
	PreviousServer       string
	BlockingReconnects   bool
	ReconQueued          bool
	ReconnectOnError     bool
	ReconAttempts        uint32
	ReconWaitMultiplier  uint64 //time to sleep in seconds before reconnecting
	ReconAllowedAttempts uint32
	GameID               int32
	GameKey              []byte
	GameKeyTime          int32
	ConnectionGUID       string
	PerformedAction      bool //allows for performing a certain amount of actions and resetting on reconnect
}

func (r *ReconBase) Reset() {
	r.BlockingReconnects = false
	r.ReconAttempts = 0
	r.ReconWaitMultiplier = 1
	r.PerformedAction = false
}

func (r *ReconBase) ReconCheck() bool {
	return r.ReconAttempts > r.ReconAllowedAttempts
}

func (r *ReconBase) Increment(w bool) {
	if r.ReconAttempts > r.ReconAllowedAttempts {
		r.ReconQueued = false
	}
	r.ReconAttempts++
	if w == true {
		r.ReconWaitMultiplier++
	}
}

type TimeKeeper struct {
	StartTime            time.Time
	LastTickID           int32
	LastServerRealTimeMS uint32
	CurrentTickTime      int32
	LastTickTime         int32
	SwapSpeedMS          int
	PreviousTime         int32
	SwapAction           time.Time
	OtherAction          time.Time
	ThreadDelayMS        int32
	AllowedSwap          bool
}

//GetTime returns the time in ms since starting the connection
func (t *TimeKeeper) GetTime() int32 {
	return int32(time.Duration(time.Since(t.StartTime) / time.Millisecond))
}

//GetOtherTime can be used as a more "network" based approach to timing. Each newtick adds 200ms
func (c *Client) GetOtherTime() int32 {
	if c.Moves.LastTickID != 0 {
		return int32(200 * c.Moves.LastTickID)
	}
	return 0
}

//SinceLast returns the number of elapsed Ms since the last designated time
func SinceLast(t time.Time) int {
	return int(time.Duration(time.Since(t) / time.Millisecond))
}

//NewClock returns a time.Time variable to be used for the duration of the clients connection
func (t *TimeKeeper) NewClock() {
	t.StartTime = time.Now()
}

func GetUnixTime() int32 {
	return int32(time.Now().Unix())
}

type MasterBase struct {
	MasterID     int32
	FollowMaster bool
	MasterPos    WorldPosData
	MasterName   string
	DoxName      string
	DoxID        int
	DoxStats     StatBase
}

type StatBase struct {
	StatMap map[byte]StatData
}

type CombatBase struct {
	CurrentBulletID byte
	Target          int32 //objid of target
	ShootingEnabled bool
	LastAttackTime  int32
	Projectiles     []GameProjectile
}

func (c *Client) CheckProjectiles() {
	if len(c.Combat.Projectiles) > 0 {

	}
}

func (c *CombatBase) PushProjectile(p GameProjectile) {
	c.Projectiles = append(c.Projectiles, p)
}

func (c *CombatBase) PopProjectile() GameProjectile {
	var projectile GameProjectile
	projectile, c.Projectiles = c.Projectiles[len(c.Projectiles)-1], c.Projectiles[:len(c.Projectiles)-1]
	return projectile
}

type TradeBase struct {
	MyOffers       []bool
	TheirOffers    []bool
	SentOffer      bool
	AllowedAccept  bool
	AcceptedTrade  bool
	TradeSuccess   bool
	InTrade        bool
	CanSendMessage bool
	TradersName    string
}

func (t *TradeBase) SelectAll() []bool {
	return []bool{
		false,
		false,
		false,
		false,
		true,
		true,
		true,
		true,
		true,
		true,
		true,
		true,
	}
}

func (t *TradeBase) SelectNone() []bool {
	return []bool{
		false,
		false,
		false,
		false,
		false,
		false,
		false,
		false,
		false,
		false,
		false,
		false,
	}
}

func (c *Client) AcceptMeNoneThemAll() {
	c.Trade.MyOffers = c.Trade.SelectNone()
	c.Trade.TheirOffers = c.Trade.SelectAll()
	c.AcceptTrade()
}

func (c *Client) AcceptMeAllThemNone() {
	c.Trade.MyOffers = c.Trade.SelectAll()
	c.Trade.TheirOffers = c.Trade.SelectNone()
	c.AcceptTrade()
}

func (c *Client) AcceptTrade() {
	at := AcceptTrade{}
	at.MyOffers = c.Trade.MyOffers
	at.TheirOffers = c.Trade.TheirOffers
	c.Connection.Send(WritePacket(at))
	c.Trade.AcceptedTrade = true
}

type GameObjects struct {
	EntityMap    map[int32]ObjectData //ALL game entities that are given to us
	TargetObject ObjectStatusData
	StoredObject ObjectStatusData
	Containers   []int32 //containers by their object id; this could become stale if update dictates a drop of the entity
}

func (g *GameObjects) GetObjByType(t uint16) ObjectData {
	for k := range g.EntityMap {
		if g.EntityMap[k].ObjectType == t {
			return g.EntityMap[k]
		}
	}
	return ObjectData{}
}

func (g *GameObjects) GetObjByID(id int32) ObjectData {
	v, ok := g.EntityMap[id]
	if !ok {
		return ObjectData{}
	}
	return v
}

func (g *GameObjects) GetEntitiesInRange(pos WorldPosData, ignore int32) []ObjectData {
	objs := make([]ObjectData, 0)
	for k := range g.EntityMap {
		addr := g.EntityMap[k].Status.Pos
		if pos.distanceTo(&addr) < 1.0 && g.EntityMap[k].Status.ObjectID != ignore {
			objs = append(objs, g.EntityMap[k])
		}
	}
	return objs
}

func (g *GameObjects) GetObjByName(name string) ObjectStatusData {
	for k := range g.EntityMap {
		if g.EntityMap[k].Status.Stats[NAME].StrStatValue == name {
			return g.EntityMap[k].Status
		}
	}
	return ObjectStatusData{}
}

type MiscBase struct {
	URLAttempts int
}

type ModuleBase struct {
	//general stuff
	// the general switch-case behavior to determine what step to perform in a module
	// if switching modules at runtime, set phase to 0 and force the bot to nexus to reset any settings that previous module used
	// you will also have to change the clients "module" setting
	Phase int
	//dupe module
	DupeMoveIndex          byte
	InventoryRecoveryCount int //threshold for disconnecting bugged accounts
	//receive module
	SentLetsGo  bool
	MyItemGroup int
	Attempts    int
	GoodStatus  bool
	//daily login
	DesiredClaimed  bool
	GrabbedCalendar bool
	CalendarDays    *LoginCalendar
	//tracker stuff
	MyPlayers        []OnlinePlayer
	MutexBool        bool //bool to run the db logger goroutine
	DarkEyeTrackerID int32
	//vault unlocker
	ChestsUnlocked int
	HighIndex      byte //index of inventory
	LowIndex       byte //index of inventory
	CheckStage     int
	DupeSuccesses  int
	//filter module, functions are defined in `modulefilter.go`
	BlackList        []string //blacklist of strings to filter
	LogFile          *os.File
	LogWriter        *bufio.Writer
	PreviousMessages []string
}

type VaultBase struct {
	SelectedVaultID int32
	MyVaults        map[WorldPosData]*ObjectStatusData
	MyGiftVaults    map[WorldPosData]*ObjectStatusData
}
