package main

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	superInt = 4294967295
)

//IPacket implements all packet types' write functions
type IPacket interface {
	Write(*Packet)
}

//Write is middleman for individual packet type Write functions
func Write(i IPacket, p *Packet) {
	i.Write(p)
}

//ReadPacket takes low-level Packet type and returns a high level type
//note that if needed it can become a Client receiver method
func ReadPacket(p *Packet) interface{} {
	p.ID = int(p.Data[4])
	switch p.ID {
	case FailureID:
		f := Failure{}
		f.FailureID = p.ReadInt32()
		f.FailureMessage = p.ReadString()
		return f
	case ReconnectID:
		r := Reconnect{}
		r.Name = p.ReadString()
		r.Host = p.ReadString()
		r.Stats = p.ReadString()
		r.Port = p.ReadInt32()
		r.GameID = p.ReadInt32()
		r.KeyTime = p.ReadInt32()
		r.IsFromArena = p.ReadBool()
		arrLen := int(p.ReadInt16())
		if arrLen > 0 {
			r.Key = make([]byte, arrLen)
			for i := 0; i < arrLen; i++ {
				r.Key[i] = p.ReadByte()
			}
		}
		return r
	case MapInfoID:
		m := MapInfo{}
		m.Width = p.ReadInt32()
		m.Height = p.ReadInt32()
		m.Name = p.ReadString()
		m.DisplayName = p.ReadString()
		m.RealmName = p.ReadString()
		m.Difficulty = p.ReadInt32()
		m.Fp = p.ReadUInt32()
		m.Background = p.ReadInt32()
		m.AllowPlayerTeleport = p.ReadBool()
		m.ShowDisplays = p.ReadBool()
		m.MaxPlayers = p.ReadInt16()
		m.ConnectionGUID = p.ReadString()
		m.GameOpenedTime = p.ReadUInt32()
		m.ServerVersion = p.ReadString()
		// counts := int(p.ReadInt16())
		// if counts > 0 {
		// 	// return m
		// 	for i := 0; i < counts; i++ {
		// 		m.ClientXML[i] = p.ReadUTFString()
		// 	}
		// }
		// counts = int(p.ReadInt16())
		// if counts > 0 {
		// 	for i := 0; i < counts; i++ {
		// 		m.ExtraXML[i] = p.ReadUTFString()
		// 	}
		// }
		return m
	case CreateSuccessID:
		c := CreateSuccess{}
		c.ObjectID = p.ReadInt32()
		c.CharID = p.ReadInt32()
		return c
	case UpdateID:
		u := Update{}
		i := 0
		var itemLen int
		// itemLen = int(p.ReadInt16())
		itemLen = int(p.ReadCompressedInt())
		// i = itemLen
		if itemLen > 0 {
			u.Tiles = make([]GroundTile, itemLen)
			for i = 0; i < itemLen; i++ {
				u.Tiles[i] = p.ReadGroundTile()
			}
		}
		// itemLen = int(p.ReadInt16())
		itemLen = int(p.ReadCompressedInt())
		if itemLen > 0 {
			u.NewObjs = make([]ObjectData, itemLen)
			for i = 0; i < itemLen; i++ {
				u.NewObjs[i] = p.ReadObjectData()
			}
		}
		// itemLen = int(p.ReadInt16())
		itemLen = int(p.ReadCompressedInt())
		if itemLen > 0 {
			u.Drops = make([]int32, itemLen)
			for i = 0; i < itemLen; i++ {
				// u.Drops[i] = p.ReadInt32()
				u.Drops[i] = p.ReadCompressedInt()
			}
		}
		return u
	case NewTickID:
		n := NewTick{}
		n.TickID = p.ReadInt32()
		n.TickTime = p.ReadInt32()
		n.ServerRealTimeMS = p.ReadUInt32()
		n.ServerLastRTTMS = p.ReadUInt16()
		arrLen := int(p.ReadInt16())
		if arrLen > 0 {
			n.Statuses = make([]ObjectStatusData, arrLen)
			for i := 0; i < arrLen; i++ {
				n.Statuses[i] = p.ReadObjectStatusData()
			}
		}
		return n
	case PingID:
		pi := Ping{}
		pi.Serial = p.ReadInt32()
		return pi
	case ClientStatID:
		cl := ClientStat{}
		cl.Name = p.ReadString()
		cl.Value = p.ReadInt32()
		return cl
	case TextID:
		t := Text{}
		t.Name = p.ReadString()
		t.ObjectID = p.ReadInt32()
		t.Stars = p.ReadInt32()
		t.BubbleTime = p.ReadByte()
		t.Recipient = p.ReadString()
		t.Message = p.ReadString()
		t.CleanMessage = p.ReadString()
		t.Supporter = p.ReadBool()
		t.StarBG = p.ReadInt32()
		return t
	case InvitedToGuildID:
		i := InvitedToGuild{}
		i.Name = p.ReadString()
		i.GuildName = p.ReadString()
		return i
	case ServerPlayerShootID:
		s := ServerPlayerShoot{}
		s.BulletID = p.ReadByte()
		s.OwnerID = p.ReadInt32()
		s.ContainerType = p.ReadInt32()
		s.StartingPos = p.ReadWorldPosData()
		s.Angle = p.ReadFloat()
		s.Damage = p.ReadInt16()
		return s
	case DamageID:
		d := Damage{}

		return d
	case EnemyShootID:
		e := EnemyShoot{}
		e.BulletID = p.ReadByte()
		e.OwnerID = p.ReadInt32()
		e.BulletType = p.ReadByte()
		e.Location = p.ReadWorldPosData()
		e.Angle = p.ReadFloat()
		e.Damage = p.ReadInt16()
		if len(p.Data[p.Index:]) > 0 {
			e.NumShots = p.ReadByte()
			e.AngleInc = p.ReadFloat()
		} else {
			e.NumShots = 1
			e.AngleInc = 0.0
		}
		return e
	case DeathID:
		d := Death{}
		d.AccountID = p.ReadString()
		d.CharID = p.ReadInt32()
		d.KilledBy = p.ReadString()
		d.ZombieType = p.ReadInt32()
		d.ZombieID = p.ReadInt32()
		d.Unknown = p.ReadInt32()
		return d
	case TradeChangedID:
		tc := TradeChanged{}
		arrLen := p.ReadInt16()
		if arrLen > 0 {
			tc.TheirOffers = make([]bool, arrLen)
			for x := 0; x < int(arrLen); x++ {
				tc.TheirOffers[x] = p.ReadBool()
			}
		}
		return tc
	case AoEID:
		a := AoE{}
		a.Position = p.ReadWorldPosData()
		a.Radius = p.ReadFloat()
		a.Damage = p.ReadUInt16()
		a.Effects = p.ReadConditionEffect()
		a.EffectDuration = p.ReadFloat()
		a.OriginType = p.ReadInt16()
		a.Color = p.ReadInt32()
		a.ArmorPierce = p.ReadBool()
		return a
	case NotificationID:
		n := Notification{}
		n.ObjectID = p.ReadInt32()
		n.Message = p.ReadString()
		n.Color = p.ReadInt32()
		return n
	case ShowEffectID:
		s := ShowEffect{}
		return s
	case GotoID:
		g := Goto{}
		g.ObjectID = p.ReadInt32()
		g.Location = p.ReadWorldPosData()
		return g
	case AccountListID:
		a := AccountList{}
		a.AccountListID = p.ReadInt32()
		arrLen := int(p.ReadInt16())
		a.AccountIDs = make([]string, arrLen)
		for i := 0; i < arrLen; i++ {
			a.AccountIDs[i] = p.ReadString()
		}
		a.LockAction = p.ReadInt32()
		return a
	case QuestObjIDID:
		q := QuestObjID{}
		q.ObjectID = p.ReadInt32()
		len := p.ReadCompressedInt()
		q.HealthBars = make([]int32, len)
		for i := 0; i < int(len); i++ {
			q.HealthBars[i] = p.ReadCompressedInt()
		}
		return q
	case InvResultID:
		i := InvResult{}
		i.Result = p.ReadInt32()
		return i
	case TradeAcceptedID:
		t := TradeAccepted{}
		myCount := int(p.ReadInt16())
		t.MyOffers = make([]bool, myCount)
		if myCount > 0 {
			for i := 0; i < myCount; i++ {
				t.MyOffers[i] = p.ReadBool()
			}
		}
		theirCount := int(p.ReadInt16())
		t.TheirOffers = make([]bool, theirCount)
		if theirCount > 0 {
			for i := 0; i < theirCount; i++ {
				t.TheirOffers[i] = p.ReadBool()
			}
		}
		return t
	case TradeStartID:
		t := TradeStart{}
		myCount := int(p.ReadInt16())
		t.MyItems = make([]TradeItem, myCount)
		if myCount > 0 {
			for i := 0; i < myCount; i++ {
				t.MyItems[i] = p.ReadTradeItem()
			}
		}
		t.TheirName = p.ReadString()
		theirCount := int(p.ReadInt16())
		t.TheirItems = make([]TradeItem, theirCount)
		if theirCount > 0 {
			for i := 0; i < theirCount; i++ {
				t.TheirItems[i] = p.ReadTradeItem()
			}
		}
		t.YourObjID = p.ReadInt32()
		return t
	case TradeRequestedID:
		t := TradeRequested{}
		t.Name = p.ReadString()
		return t
	case TradeDoneID:
		t := TradeDone{}
		t.ResultCode = p.ReadInt32()
		t.Message = p.ReadString()
		return t
	case CreateGuildResultID:
		cgr := CreateGuildResult{}
		cgr.Success = p.ReadBool()
		cgr.ErrorMessage = p.ReadString()
		return cgr
	case GlobalNotificationID:
		g := GlobalNotification{}
		g.TypeID = p.ReadInt32()
		g.Text = p.ReadString()
		return g
	case FileID:
		f := File{}
		f.Name = p.ReadString()
		length := int(p.ReadUInt32())
		arr := make([]byte, length)
		for i := 0; i < length; i++ {
			arr[i] = p.ReadByte()
		}
		return f //not really any point in returning but w/e
	case AllyShootID:
		a := AllyShoot{}
		return a
	case PlaySoundID:
		ps := PlaySound{}
		return ps
	case LoginRewardRecvID:
		lrr := LoginRewardRecv{}
		lrr.ItemID = p.ReadInt32()
		lrr.Quantity = p.ReadInt32()
		lrr.Gold = p.ReadInt32()
		return lrr
	case RealmHeroLeftID:
		rh := RealmHeroLeft{}
		rh.HeroesLeft = p.ReadInt32()
		return rh
	case UnlockInformationID:
		ui := UnlockInformation{}
		return ui
	case NewCharacterInformationID:
		nci := NewCharacterInformation{}
		return nci
	case UnknownID:
		u := Unknown{}
		return u
	default:
		fmt.Printf("Got %d for packet id in ReadPacket\n", p.ID)
		return nil
	}
}

//WritePacket takes a packet type and returns a Packet pointer
// with Packet.Data filled in from v's type
//note that it can become a Client receiver method
func WritePacket(v interface{}) *Packet {
	p := new(Packet)
	p.Index = 5
	//need to alloc space BEFORE writing...
	//todo: make it dynamically make size
	p.Data = make([]byte, packetSize+packetSize)
	p.Data[4] = byte(InferIDFromPacket(v))
	switch v.(type) {
	case Move:
		Write(v.(Move), p)
	case Pong:
		Write(v.(Pong), p)
	case UpdateAck:
		Write(v.(UpdateAck), p)
	case AoEAck:
		Write(v.(AoEAck), p)
	case ShootAck:
		Write(v.(ShootAck), p)
	case RequestTrade:
		Write(v.(RequestTrade), p)
	case CancelTrade:
		Write(v.(CancelTrade), p)
	case InvSwap:
		Write(v.(InvSwap), p)
	case InvDrop:
		Write(v.(InvDrop), p)
	case JoinGuild:
		Write(v.(JoinGuild), p)
	case EditAccountList:
		Write(v.(EditAccountList), p)
	case PlayerShoot:
		Write(v.(PlayerShoot), p)
	case PlayerText:
		Write(v.(PlayerText), p)
	case UseItem:
		Write(v.(UseItem), p)
	case Escape:
		Write(v.(Escape), p)
	case GotoAck:
		Write(v.(GotoAck), p)
	case UsePortal:
		Write(v.(UsePortal), p)
	case Teleport:
		Write(v.(Teleport), p)
	case PlayerHit:
		Write(v.(PlayerHit), p)
	case GroundDamage:
		Write(v.(GroundDamage), p)
	case AcceptTrade:
		Write(v.(AcceptTrade), p)
	case ChangeTrade:
		Write(v.(ChangeTrade), p)
	case ChangeGuildRank:
		Write(v.(ChangeGuildRank), p)
	case GuildInvite:
		Write(v.(GuildInvite), p)
	case GuildRemove:
		Write(v.(GuildRemove), p)
	case Buy:
		Write(v.(Buy), p)
	case PetChangeSkin:
		Write(v.(PetChangeSkin), p)
	case KeyInfoRequest:
		Write(v.(KeyInfoRequest), p)
	case SetCondition:
		Write(v.(SetCondition), p)
	case LoginRewardSend:
		Write(v.(LoginRewardSend), p)
	case PetUpgradeRequest:
		Write(v.(PetUpgradeRequest), p)
	case ActivePetUpdateSend:
		Write(v.(ActivePetUpdateSend), p)
	case Load:
		Write(v.(Load), p)
	case Create:
		Write(v.(Create), p)
	case Hello:
		Write(v.(Hello), p)
	default:
		fmt.Println("Unknown Packet in WritePacket:", v)
	}
	//do some finalization
	p.Length = uint(p.Index)
	binary.BigEndian.PutUint32(p.Data[0:4], uint32(len(p.Data[:p.Length])))
	p.ID = int(p.Data[4])
	p.Data = p.Data[:p.Length] //now shrink our buffer to the appropriate size
	return p
}

//Read takes a bytes packet and returns a defined and filled packet struct
func Read() interface{} {
	return nil
}

/* Failure protocol error code list
	0 - failed to respond to ping
	2 - Wrong position sent
	7 - Responding to the wrong packet (responding to playershoot when you didnt shoot)
*/

//Failure server packet
// is sent to client when something is not correct.
// note that the server can dc you without sending this!
type Failure struct {
	FailureID      int32
	FailureMessage string
}

func (h *Failure) Write(p *Packet) {
	p.WriteInt32(h.FailureID)
	p.WriteString(h.FailureMessage)
}

//CreateSuccess server packet
// sent after sending load or create
// note: an update packet is appended to this one
// when it's first sent to us
type CreateSuccess struct {
	ObjectID int32
	CharID   int32
}

//MapInfo server packet
// the first packet to get received
type MapInfo struct {
	Width               int32
	Height              int32
	Name                string
	DisplayName         string
	RealmName           string
	Difficulty          int32
	Fp                  uint32
	Background          int32
	AllowPlayerTeleport bool
	ShowDisplays        bool
	MaxPlayers          int16
	ConnectionGUID      string
	ClientXML           []string //utf
	ExtraXML            []string //utf
	GameOpenedTime      uint32
	ServerVersion       string
}

//Hello client packet
// the first packet to get sent
type Hello struct {
	BuildVersion           string
	GameID                 int32
	GUID                   string
	Random1                int32
	Password               string
	Random2                int32
	Secret                 string
	KeyTime                uint32
	Key                    []byte
	MapJSON                string
	EntryTag               string
	GameNet                string
	GameNetUserID          string
	PlayPlatform           string
	PlatformToken          string
	UserToken              string
	ClientToken            string
	PreviousConnectionGUID string
}

func (h Hello) Write(p *Packet) {
	p.WriteString(h.BuildVersion)
	p.WriteInt32(h.GameID)
	p.WriteString(EncryptString(h.GUID))
	// fmt.Println("Email:", EncryptString(h.GUID))
	p.WriteInt32(h.Random1)
	if h.Password != "" {
		p.WriteString(EncryptString(h.Password))
	} else {
		p.WriteString(h.Password)
	}
	// fmt.Println("Password:", EncryptString(h.Password))
	p.WriteInt32(h.Random2)
	if h.Secret != "" { //so we dont write a bunch of encrypted nothings
		p.WriteString(EncryptString(h.Secret))
	} else {
		p.WriteString(h.Secret)
	}
	p.WriteUInt32(h.KeyTime)
	p.WriteInt16(int16(len(h.Key)))
	for i := 0; i < len(h.Key); i++ {
		p.WriteByte(h.Key[i])
	}
	p.WriteUTFString(h.MapJSON)
	p.WriteString(h.EntryTag)
	p.WriteString(h.GameNet)
	p.WriteString(h.GameNetUserID)
	p.WriteString(h.PlayPlatform)
	p.WriteString(h.PlatformToken)
	p.WriteString(h.UserToken)
	p.WriteString(h.ClientToken)
	p.WriteString(h.PreviousConnectionGUID)
}

//Load client packet
type Load struct {
	CharID       int32
	IsFromArena  bool
	IsChallenger bool
}

func (l Load) Write(p *Packet) {
	p.WriteInt32(l.CharID)
	p.WriteBool(l.IsFromArena)
	p.WriteBool(l.IsChallenger)
}

//Create client packet
type Create struct {
	ClassType    uint16
	SkinType     uint16
	IsChallenger bool
}

func (c Create) Write(p *Packet) {
	p.WriteUInt16(c.ClassType)
	p.WriteUInt16(c.SkinType)
	p.WriteBool(c.IsChallenger)
}

//PlayerShoot client packet
type PlayerShoot struct {
	Time          int32
	BulletID      byte
	ContainerType int16
	Position      WorldPosData
	Angle         float32
	SpeedMult     int16
	lifeMult      int16
}

func (ps PlayerShoot) Write(p *Packet) {
	p.WriteInt32(ps.Time)
	p.WriteByte(ps.BulletID)
	p.WriteInt16(ps.ContainerType)
	p.WriteWorldPosData(ps.Position)
	p.WriteFloat(ps.Angle)
	p.WriteInt16(ps.SpeedMult)
	p.WriteInt16(ps.lifeMult)
}

//Move client packet
type Move struct {
	TickID                        int32
	Time                          int32
	ServerRealTimeMSOfLastNewTick uint32
	NewPosition                   WorldPosData
	Records                       []PositionRecords
}

func (m Move) Write(p *Packet) {
	p.WriteInt32(m.TickID)
	p.WriteInt32(m.Time)
	p.WriteUInt32(m.ServerRealTimeMSOfLastNewTick)
	p.WriteWorldPosData(m.NewPosition)
	p.WriteInt16(int16(len(m.Records)))
	for i := 0; i < int(len(m.Records)); i++ {
		p.WritePositionRecord(m.Records[i])
	}
}

//PlayerText client packet
type PlayerText struct {
	Message string
}

func (pt PlayerText) Write(p *Packet) {
	p.WriteString(pt.Message)
}

//Text server packet
type Text struct {
	Name         string
	ObjectID     int32
	Stars        int32
	BubbleTime   byte
	Recipient    string
	Message      string
	CleanMessage string
	Supporter    bool
	StarBG       int32
}

//ServerPlayerShoot server packet
type ServerPlayerShoot struct {
	BulletID      byte
	OwnerID       int32
	ContainerType int32
	StartingPos   WorldPosData
	Angle         float32
	Damage        int16
}

//Damage server packet
type Damage struct {
	TargetID      int32
	Effects       ConditionEffect
	Damage        uint16
	Killed        bool
	ArmorPiercing bool
	BulletID      byte
	ObjectID      int32
}

//Update server packet
type Update struct {
	Tiles   []GroundTile
	NewObjs []ObjectData
	Drops   []int32
}

//UpdateAck client packet
// this is sent for every update packet received
type UpdateAck struct {
}

func (u UpdateAck) Write(p *Packet) {
}

//Notification server packet
type Notification struct {
	ObjectID int32
	Message  string
	Color    int32
}

//NewTick server packet
// received roughly every 200ms
type NewTick struct {
	TickID           int32
	TickTime         int32
	ServerRealTimeMS uint32
	ServerLastRTTMS  uint16
	Statuses         []ObjectStatusData
}

//InvSwap client packet
type InvSwap struct {
	Time     int32
	Position WorldPosData
	OldSlot  SlotObjectData
	NewSlot  SlotObjectData
}

func (i InvSwap) Write(p *Packet) {
	p.WriteInt32(i.Time)
	p.WriteWorldPosData(i.Position)
	p.WriteSlotObjectData(i.OldSlot)
	p.WriteSlotObjectData(i.NewSlot)
}

//UseItem client packet
type UseItem struct {
	Time       int32
	SlotObject SlotObjectData
	Position   WorldPosData
	UseType    byte
}

func (u UseItem) Write(p *Packet) {
	p.WriteInt32(u.Time)
	p.WriteSlotObjectData(u.SlotObject)
	p.WriteWorldPosData(u.Position)
	p.WriteByte(u.UseType)
}

//ShowEffect server packet
type ShowEffect struct {
	Effect   EffectType
	TargetID int32
	PosA     WorldPosData
	PosB     WorldPosData
	Color    ARGB
	Duration float32
}

//Goto server packet
// sent whenever something teleports afaik
type Goto struct {
	ObjectID int32
	Location WorldPosData
}

//InvDrop client packet
type InvDrop struct {
	SlotObject SlotObjectData
}

func (i InvDrop) Write(p *Packet) {
	p.WriteSlotObjectData(i.SlotObject)
}

//InvResult server packet
type InvResult struct {
	Result int32
}

//Reconnect server packet
type Reconnect struct {
	Name        string
	Host        string
	Stats       string
	Port        int32
	GameID      int32
	KeyTime     int32
	IsFromArena bool
	Key         []byte
}

//Ping server packet
// sent about every second
type Ping struct {
	Serial int32
}

//Pong client packet
type Pong struct {
	Serial int32
	Time   int32
}

func (po Pong) Write(p *Packet) {
	p.WriteInt32(po.Serial)
	p.WriteInt32(po.Time)
}

//Pic server packet
// don't think this gets used anymore
type Pic struct {
	PicData BitMapData
}

//SetCondition client packet
type SetCondition struct {
	ConditionEffect   byte
	ConditionDuration float32
}

func (c SetCondition) Write(p *Packet) {
	p.WriteByte(c.ConditionEffect)
	p.WriteFloat(c.ConditionDuration)
}

//Teleport client packet
type Teleport struct {
	ObjectID int32
}

func (c Teleport) Write(p *Packet) {
	p.WriteInt32(c.ObjectID)
}

//UsePortal client packet
type UsePortal struct {
	ObjectID int32
}

func (c UsePortal) Write(p *Packet) {
	p.WriteInt32(c.ObjectID)
}

//Death server packet
type Death struct {
	AccountID  string
	CharID     int32
	KilledBy   string
	ZombieType int32
	ZombieID   int32
	Unknown    int32
}

//Buy client packet
type Buy struct {
	ObjectID int32
	Quantity int32
}

func (c Buy) Write(p *Packet) {
	p.WriteInt32(c.ObjectID)
	p.WriteInt32(c.Quantity)
}

//BuyResult server packet
type BuyResult struct {
	/*	Unknown -1
		Success 0
		InvalidCharacter 1
		ItemNotFound 2
		NotEnoughGold 3
		InventoryFull 4
		TooLowRank 5
		NotEnoughFame 6
		PetFeedSuccess 7 */
	ResultCode int32
	Message    string
}

//AoE server packet
//sent when enemy or something throws a grenade
type AoE struct {
	Position       WorldPosData
	Radius         float32
	Damage         uint16
	Effects        ConditionEffect
	EffectDuration float32
	OriginType     int16
	Color          int32
	ArmorPierce    bool
}

//AoEAck client packet
type AoEAck struct {
	Time     int32
	Position WorldPosData
}

func (a AoEAck) Write(p *Packet) {
	p.WriteInt32(a.Time)
	p.WriteWorldPosData(a.Position)
}

//GroundDamage client
type GroundDamage struct {
	Time     int32
	Position WorldPosData
}

func (c GroundDamage) Write(p *Packet) {
	p.WriteInt32(c.Time)
	p.WriteWorldPosData(c.Position)
}

//PlayerHit client
type PlayerHit struct {
	BulletID byte
	ObjectID int32
}

func (c PlayerHit) Write(p *Packet) {
	p.WriteByte(c.BulletID)
	p.WriteInt32(c.ObjectID)
}

//EnemyHit client
type EnemyHit struct {
	Time     int32
	BulletID byte
	TargetID int32
	Killed   bool
}

func (c EnemyHit) Write(p *Packet) {
	p.WriteInt32(c.Time)
	p.WriteByte(c.BulletID)
	p.WriteInt32(c.TargetID)
	p.WriteBool(c.Killed)
}

//ShootAck client
type ShootAck struct {
	Time int32
}

func (s ShootAck) Write(p *Packet) {
	p.WriteInt32(s.Time)
}

//OtherHit client
type OtherHit struct {
	Time     int32
	BulletID byte
	ObjectID int32
	TargetID int32
}

func (c OtherHit) Write(p *Packet) {
	p.WriteInt32(c.Time)
	p.WriteByte(c.BulletID)
	p.WriteInt32(c.ObjectID)
	p.WriteInt32(c.TargetID)
}

//SquareHit client
type SquareHit struct {
	Time     int32
	BulletID byte
	ObjectID int32
}

func (c SquareHit) Write(p *Packet) {
	p.WriteInt32(c.Time)
	p.WriteByte(c.BulletID)
	p.WriteInt32(c.ObjectID)
}

//GotoAck client
type GotoAck struct {
	Time int32
}

func (g GotoAck) Write(p *Packet) {
	p.WriteInt32(g.Time)
}

//EditAccountList client
type EditAccountList struct {
	AccountListID int32
	Add           bool
	ObjectID      int32
}

func (e EditAccountList) Write(p *Packet) {
	p.WriteInt32(e.AccountListID)
	p.WriteBool(e.Add)
	p.WriteInt32(e.ObjectID)
}

//AccountList server
type AccountList struct {
	AccountListID int32
	AccountIDs    []string
	LockAction    int32
}

//QuestObjID server
type QuestObjID struct {
	ObjectID   int32
	HealthBars []int32
}

//ChooseName client
type ChooseName struct {
	MyName string
}

func (c ChooseName) Write(p *Packet) {
	p.WriteString(c.MyName)
}

//NameResult server
type NameResult struct {
	Success      bool
	ErrorMessage string
}

//CreateGuild client
type CreateGuild struct {
	GuildName string
}

func (c CreateGuild) Write(p *Packet) {
	p.WriteString(c.GuildName)
}

//CreateGuildResult server
type CreateGuildResult struct {
	Success      bool
	ErrorMessage string
}

//GuildRemove client
type GuildRemove struct {
	PlayerName string
}

func (c GuildRemove) Write(p *Packet) {
	p.WriteString(c.PlayerName)
}

//GuildInvite client
type GuildInvite struct {
	PlayerName string
}

func (c GuildInvite) Write(p *Packet) {
	p.WriteString(c.PlayerName)
}

//AllyShoot server
type AllyShoot struct {
	BulletID      byte
	OwnerID       int32
	ContainerType int16
	Angle         float32
	Bard          bool
}

//EnemyShoot server
type EnemyShoot struct {
	BulletID   byte
	OwnerID    int32
	BulletType byte
	Location   WorldPosData
	Angle      float32
	Damage     int16
	NumShots   byte
	AngleInc   float32
}

//RequestTrade client
type RequestTrade struct {
	PlayerName string
}

func (r RequestTrade) Write(p *Packet) {
	p.WriteString(r.PlayerName)
}

//TradeRequested server
type TradeRequested struct {
	Name string
}

//TradeStart server
type TradeStart struct {
	MyItems    []TradeItem
	TheirName  string
	TheirItems []TradeItem
	YourObjID  int32
}

//ChangeTrade client
type ChangeTrade struct {
	MyOffers []bool
}

func (c ChangeTrade) Write(p *Packet) {
	arrLen := len(c.MyOffers)
	p.WriteInt16(int16(arrLen))
	if arrLen > 0 {
		for x := 0; x < arrLen; x++ {
			p.WriteBool(c.MyOffers[x])
		}
	}

}

//TradeChanged server
type TradeChanged struct {
	TheirOffers []bool
}

//AcceptTrade client
type AcceptTrade struct {
	MyOffers    []bool
	TheirOffers []bool
}

func (a AcceptTrade) Write(p *Packet) {
	arrLen := len(a.MyOffers)
	p.WriteInt16(int16(arrLen))
	if arrLen > 0 {
		for x := 0; x < arrLen; x++ {
			p.WriteBool(a.MyOffers[x])
		}
	}
	arrLen = len(a.TheirOffers)
	p.WriteInt16(int16(arrLen))
	if arrLen > 0 {
		for x := 0; x < arrLen; x++ {
			p.WriteBool(a.TheirOffers[x])
		}
	}
}

//CancelTrade client
type CancelTrade struct {
}

func (c CancelTrade) Write(p *Packet) {
}

//TradeDone server
type TradeDone struct {
	/*  TradeSuccessful 0
	PlayerCancelled 1
	TradeError ? */
	ResultCode int32
	Message    string
}

//TradeAccepted server
type TradeAccepted struct {
	MyOffers    []bool
	TheirOffers []bool
}

//ClientStat server
type ClientStat struct {
	Name  string
	Value int32
}

type CheckCredits struct { //Client

}

func (c CheckCredits) Write(p *Packet) {
}

type Escape struct { //Client

}

func (c Escape) Write(p *Packet) {
}

//File server packet
// don't think this has ever been used in rotmg history
// probably could be used to drop some sort of RAT on users xD
type File struct {
	Name  string
	Bytes []byte
}

//InvitedToGuild server
type InvitedToGuild struct {
	Name      string
	GuildName string
}

//JoinGuild client
type JoinGuild struct {
	GuildName string
}

func (j JoinGuild) Write(p *Packet) {
	p.WriteString(j.GuildName)
}

//ChangeGuildRank client packet
type ChangeGuildRank struct {
	Name      string
	GuildRank int32
}

func (c ChangeGuildRank) Write(p *Packet) {
	p.WriteString(c.Name)
	p.WriteInt32(c.GuildRank)
}

//PlaySound server packet
type PlaySound struct {
	OwnerID int32
	SoundID byte
}

//GlobalNotification server
type GlobalNotification struct {
	TypeID int32
	Text   string
}

//ReSkin client packet
type ReSkin struct {
	SkinID int32
}

func (c ReSkin) Write(p *Packet) {
	p.WriteInt32(c.SkinID)
}

//PetUpgradeRequest client
type PetUpgradeRequest struct {
	PetTransType     byte
	PIDOne           int32
	PIDTwo           int32
	ObjectID         int32
	SlotObject       SlotObjectData
	PaymentTransType byte
}

func (c PetUpgradeRequest) Write(p *Packet) {
	p.WriteByte(c.PetTransType)
	p.WriteInt32(c.PIDOne)
	p.WriteInt32(c.PIDTwo)
	p.WriteInt32(c.ObjectID)
	p.WriteSlotObjectData(c.SlotObject)
	p.WriteByte(c.PaymentTransType)
}

type ActivePetUpdateSend struct { //Client
	CommandType byte
	InstanceID  int32
}

func (c ActivePetUpdateSend) Write(p *Packet) {
	p.WriteByte(c.CommandType)
	p.WriteInt32(c.InstanceID)
}

type ActivePetUpdateRecv struct { //Server
	InstanceID int32
}

type NewAbility struct { //Server
	Type int32
}

type PetYardUpdate struct { //Server
	Type int32
}

type EvolvePet struct { //Server
	PetID       int32
	InitialSkin int32
	FinalSkin   int32
}

type DeletePet struct { //Server
	PetID int32
}

type HatchPet struct { //Server
	PetName  string
	PetSkin  int32
	ItemType int32
}

type EnterArena struct { //Client
	Currency int32
}

func (c EnterArena) Write(p *Packet) {
	p.WriteInt32(c.Currency)
}

type ImminentArenaWave struct { //Server
	CurrentRuntime int32
}

type ArenaDeath struct { //Server
	Cost int32
}

type AcceptArenaDeath struct { //Client
}

func (c AcceptArenaDeath) Write(p *Packet) {
}

type VerifyEmail struct { //Server
}

type ReSkinUnlock struct { //Server
	SkinID    int32
	IsPetSkin int32
}

type PasswordPrompt struct { //Server
	CleanPasswordStatus int32
}

type QuestFetchAsk struct { //Client
}

func (c QuestFetchAsk) Write(p *Packet) {
}

type QuestRedeem struct { //Client
	SlotObject SlotObjectData
}

func (c QuestRedeem) Write(p *Packet) {
	p.WriteSlotObjectData(c.SlotObject)
}

type QuestFetchResponse struct { //Server
	Tier        int32
	Goal        string
	Description string
	Image       string
}

type QuestRedeemResponse struct { //Server
	Success bool
	Message string
}

type PetChangeFormMessage struct { //
	//fuck is this?
}

type KeyInfoRequest struct { //Client
	ItemType int32
}

func (c KeyInfoRequest) Write(p *Packet) {
	p.WriteInt32(c.ItemType)
}

type KeyInfoResponse struct { //Server
	Name        string
	Description string
	Creator     string
}

type LoginRewardSend struct { //Client
	ClaimKey string //a b64 encoded string, obtained from https://realmofthemadgodhrd.appspot.com/dailyLogin/fetchCalendar
	// replace "_" with "/" and "-" with "+" to successfully decode the string. Not sure if this needs to be done before sending the packet
	Type string // is "consecutive" or "nonconsecutive"
}

func (c LoginRewardSend) Write(p *Packet) {
	p.WriteString(c.ClaimKey)
	p.WriteString(c.Type)
}

type LoginRewardRecv struct { //Server
	ItemID   int32
	Quantity int32
	Gold     int32
}

type QuestRoomMessage struct { //Client

}

type PetChangeSkin struct { //Client
	PetID    int32
	SkinType int32
	Currency int32
}

func (c PetChangeSkin) Write(p *Packet) {
	p.WriteInt32(c.PetID)
	p.WriteInt32(c.SkinType)
	p.WriteInt32(c.Currency)
}

/*   Pseudo Packets
These aren't really "packets" with ids, just incoming message structs
*/

type ActivePet struct { //Server
	InstanceID int32
}

type PetYard struct { //Server
	Type int32
}

type ReskinPet struct { //Client client says it gets assigned id of ENTER_ARENA
	PetInstanceID    int32
	PickedNewPetType int32
	Item             SlotObjectData
}

func (r ReskinPet) Write(p *Packet) {
	p.WriteInt32(r.PetInstanceID)
	p.WriteInt32(r.PickedNewPetType)
	p.WriteSlotObjectData(r.Item)
}

type ChatHello struct { //client
	AccountID string //rsa encrypted
	Token     string
}

type ChatToken struct { //Server
	Token string
	Host  string
	Port  int
}

type ChatLogout struct { //client blank packet?

}

type RealmHeroLeft struct { //server
	HeroesLeft int32
}

//Unknown some random?
type Unknown struct { //server

}

type NewCharacterInformation struct { //server
	CharXML string
}

type QueueInformation struct { //server
	CurrentPosition uint16
	MaximumPosition uint16
}

type UnlockInformation struct { //server
	UnlockType int32
}

/*  Data Structures  */

type GroundTile struct {
	X    int16
	Y    int16
	Type uint16
}

func (p *Packet) ReadGroundTile() GroundTile {
	tmp := GroundTile{}
	tmp.X = p.ReadInt16()
	tmp.Y = p.ReadInt16()
	tmp.Type = p.ReadUInt16()
	return tmp
}

func (p *Packet) WriteGroundTile(x GroundTile) {
	p.WriteInt16(x.X)
	p.WriteInt16(x.Y)
	p.WriteUInt16(x.Type)
}

type MoveRecord struct {
	Time int32
	X    float32
	Y    float32
}

type ObjectData struct {
	ObjectType uint16
	Status     ObjectStatusData
}

func (p *Packet) ReadObjectData() ObjectData {
	tmp := ObjectData{}
	tmp.ObjectType = p.ReadUInt16()
	tmp.Status = p.ReadObjectStatusData()
	return tmp
}

type ObjectStatusData struct {
	ObjectID int32
	Pos      WorldPosData
	// Stats    []StatData
	Stats map[byte]StatData
}

//Find gets the true stat value based on search criteria (stype) casting is meant for int instead of int32
// func (o *ObjectStatusData) Find(stype int, cast bool) interface{} {
// 	ssize := len(o.Stats)
// 	btype := byte(stype)
// 	for i := 0; ssize > i; i++ {
// 		if o.Stats[i].StatType == btype {
// 			if isStringStat(btype) {
// 				return o.Stats[i].StrStatValue
// 			}
// 			if cast == true {
// 				return int(o.Stats[i].StatValue)
// 			}
// 			return o.Stats[i].StatValue
// 		}
// 	}
// 	return nil
// }

func (o *ObjectStatusData) FindStat(stat byte) StatData {
	_, ok := o.Stats[stat]
	if ok {
		return o.Stats[stat]
	}
	return StatData{}
}

//FindTrue finds the index in an array of statdata and returns the index based on input stattype
// func (o *ObjectStatusData) FindTrue(stype byte) byte {
// 	ssize := len(o.Stats)
// 	if ssize > 0 {
// 		for i := 0; ssize > i; i++ {
// 			if o.Stats[i].StatType == stype {
// 				return byte(i)
// 			}
// 		}
// 	} else {
// 		fmt.Println("Error in FindTrue:", ssize)
// 	}
// 	return 255
// }

func (p *Packet) ReadObjectStatusData() ObjectStatusData {
	tmp := ObjectStatusData{}
	// tmp.ObjectID = p.ReadInt32()
	tmp.ObjectID = p.ReadCompressedInt()
	tmp.Pos = p.ReadWorldPosData()
	// arrLen := byte(p.ReadUInt16()) //shouldnt be more than total number of stats
	arrLen := byte(p.ReadCompressedInt())
	if arrLen > 0 {
		//fmt.Println("Stats(arrlen|pdata):", arrLen, p.Data[p.Index:p.Index+arrLen])
		tmp.Stats = make(map[byte]StatData)
		var i byte
		// fmt.Println("Stat arrlen:", arrLen)
		for i = 0; i < arrLen; i++ {
			stat := p.ReadStatData()
			tmp.Stats[stat.StatType] = stat
		}
	}
	return tmp
}

type SlotObjectData struct {
	ObjectID   int32
	SlotID     byte
	ObjectType int32
}

func (p *Packet) ReadSlotObjectData() SlotObjectData {
	tmp := SlotObjectData{}
	tmp.ObjectID = p.ReadInt32()
	tmp.SlotID = p.ReadByte()
	tmp.ObjectType = p.ReadInt32()
	return tmp
}

func (p *Packet) WriteSlotObjectData(x SlotObjectData) {
	p.WriteInt32(x.ObjectID)
	p.WriteByte(x.SlotID)
	p.WriteInt32(x.ObjectType)
}

type StatData struct {
	StatType     byte
	StatValue    int32
	StrStatValue string
}

func (p *Packet) ReadStatData() StatData {
	tmp := StatData{}
	tmp.StatType = p.ReadByte()
	//We read the byte which is the specific stat and check if its a string
	if isStringStat(tmp.StatType) == true {
		tmp.StrStatValue = p.ReadString()
	} else {
		tmp.StatValue = int32(p.ReadCompressedInt())
		// tmp.StatValue = p.ReadInt32()
	}
	return tmp
}

func isStringStat(stat byte) bool {
	switch stat {
	case NAME:
		return true
	case ACCOUNTID:
		return true
	case OWNERACCOUNTID:
		return true
	case GUILDNAME:
		return true
	case PETNAME:
		return true
	default:
		return false
	}
}

type TradeItem struct {
	Item      int32
	SlotType  int32
	Tradeable bool
	Included  bool
}

func (p *Packet) ReadTradeItem() TradeItem {
	tmp := TradeItem{}
	tmp.Item = p.ReadInt32()
	tmp.SlotType = p.ReadInt32()
	tmp.Tradeable = p.ReadBool()
	tmp.Included = p.ReadBool()
	return tmp
}

type WorldPosData struct {
	X float32
	Y float32
}

func (p *Packet) ReadWorldPosData() WorldPosData {
	tmp := WorldPosData{}
	tmp.X = p.ReadFloat()
	tmp.Y = p.ReadFloat()
	return tmp
}

func (p *Packet) WriteWorldPosData(w WorldPosData) {
	p.WriteFloat(w.X)
	p.WriteFloat(w.Y)
}

func (w *WorldPosData) sqDistanceTo(p *WorldPosData) float32 {
	var x = p.X - w.X
	var y = p.Y - w.Y
	return x*x + y*y
}

func (w *WorldPosData) distanceTo(p *WorldPosData) float32 {
	return float32(math.Sqrt(float64(w.sqDistanceTo(p))))
}

func (w *WorldPosData) angleTo(p *WorldPosData) float32 {
	return float32(math.Atan2(float64(p.Y-w.Y), float64(p.X-w.X)))
}

type PositionRecords struct {
	Time int32
}

func (p *Packet) ReadPositionRecord() PositionRecords {
	tmp := PositionRecords{}
	tmp.Time = p.ReadInt32()
	return tmp
}

func (p *Packet) WritePositionRecord(x PositionRecords) {
	p.WriteInt32(x.Time)
}

type ConditionEffect struct {
	Condition byte
}

func (p *Packet) ReadConditionEffect() ConditionEffect {
	tmp := ConditionEffect{}
	tmp.Condition = p.ReadByte()
	return tmp
}

type EffectType struct {
}

type BitMapData struct {
	Width  int32
	Height int32
	Data   []byte
}

type ARGB struct {
	A byte
	R byte
	G byte
	B byte
}

//This statdata group is for hard-coded ints. This assigns them to StatData{} properly
const (
	MAXIMUMHP                   = 0
	HP                          = 1
	SIZE                        = 2
	MAXIMUMMP                   = 3
	MP                          = 4
	NEXTLEVELEXPERIENCE         = 5
	EXPERIENCE                  = 6
	LEVEL                       = 7
	INVENTORY0                  = 8
	INVENTORY1                  = 9
	INVENTORY2                  = 10
	INVENTORY3                  = 11
	INVENTORY4                  = 12
	INVENTORY5                  = 13
	INVENTORY6                  = 14
	INVENTORY7                  = 15
	INVENTORY8                  = 16
	INVENTORY9                  = 17
	INVENTORY10                 = 18
	INVENTORY11                 = 19
	ATTACK                      = 20
	DEFENSE                     = 21
	SPEED                       = 22
	PLACEHOLDER1                = 23
	PLACEHOLDER2                = 24 //placeholders... hopefully fixes statdata issues
	PLACEHOLDER3                = 25
	VITALITY                    = 26
	WISDOM                      = 27
	DEXTERITY                   = 28
	EFFECTS                     = 29 //
	STARS                       = 30
	NAME                        = 31 //string
	TEXTURE1                    = 32
	TEXTURE2                    = 33
	MERCHANDISETYPE             = 34
	CREDITS                     = 35
	MERCHANDISEPRICE            = 36
	PORTALUSABLE                = 37
	ACCOUNTID                   = 38 //string
	ACCOUNTFAME                 = 39
	MERCHANDISECURRENCY         = 40
	OBJECTCONNECTION            = 41
	MERCHANDISEREMAININGCOUNT   = 42
	MERCHANDISEREMAININGMINUTES = 43
	MERCHANDISEDISCOUNT         = 44
	MERCHANDISERANKREQUIREMENT  = 45
	HEALTHBONUS                 = 46
	MANABONUS                   = 47
	ATTACKBONUS                 = 48
	DEFENSEBONUS                = 49
	SPEEDBONUS                  = 50
	VITALITYBONUS               = 51
	WISDOMBONUS                 = 52
	DEXTERITYBONUS              = 53
	OWNERACCOUNTID              = 54 //string
	RANKREQUIRED                = 55
	NAMECHOSEN                  = 56
	CHARACTERFAME               = 57
	CHARACTERFAMEGOAL           = 58
	GLOWING                     = 59
	SINKLEVEL                   = 60
	ALTTEXTUREINDEX             = 61
	GUILDNAME                   = 62 //string
	GUILDRANK                   = 63
	OXYGENBAR                   = 64
	XPBOOSTERACTIVE             = 65
	XPBOOSTTIME                 = 66
	LOOTDROPBOOSTTIME           = 67
	LOOTTIERBOOSTTIME           = 68
	HEALTHPOTIONCOUNT           = 69
	MAGICPOTIONCOUNT            = 70
	BACKPACK0                   = 71
	BACKPACK1                   = 72
	BACKPACK2                   = 73
	BACKPACK3                   = 74
	BACKPACK4                   = 75
	BACKPACK5                   = 76
	BACKPACK6                   = 77
	BACKPACK7                   = 78
	HASBACKPACK                 = 79
	SKIN                        = 80
	PETINSTANCEID               = 81
	PETNAME                     = 82 //string
	PETTYPE                     = 83
	PETRARITY                   = 84
	PETMAXIMUMLEVEL             = 85
	PETFAMILY                   = 86
	PETPOINTS0                  = 87
	PETPOINTS1                  = 88
	PETPOINTS2                  = 89
	PETLEVEL0                   = 90
	PETLEVEL1                   = 91
	PETLEVEL2                   = 92
	PETABILITYTYPE0             = 93
	PETABILITYTYPE1             = 94
	PETABILITYTYPE2             = 95
	EFFECTS2                    = 96 //CURSE, PETRIFY
	FORTUNETOKENS               = 97
	SUPPORTERPOINTS             = 98
	SUPPORTER                   = 99
	CHALLENGERSTARBGSTAT        = 100
	PLACEHOLDER4                = 101
	PROJECTILESPEEDMULT         = 102
	PROJECTILELIFEMULT          = 103
)

const (
	NOTHING              = 0
	DEAD                 = 1
	QUIET                = 2
	WEAK                 = 3
	SLOWED               = 4
	SICK                 = 5
	DAZED                = 6
	STUNNED              = 7
	BLIND                = 8
	HALLUCINATING        = 9
	DRUNK                = 10
	CONFUSED             = 11
	STUNIMMUNE           = 12
	INVISIBLE            = 13
	PARALYZED            = 14
	SPEEDY               = 15
	BLEEDING             = 16
	ARMORBROKENIMMUNE    = 17
	HEALING              = 18
	DAMAGING             = 19
	BERSERK              = 20
	PAUSED               = 21
	STASIS               = 22
	STASISIMMUNE         = 23
	INVINCIBLE           = 24
	INVULNERABLE         = 25
	ARMORED              = 26
	ARMORBROKEN          = 27
	HEXED                = 28
	NINJASPEEDY          = 29
	UNSTABLE             = 30
	DARKNESS             = 31
	SLOWIMMUNE           = 32
	DAZEIMMUNE           = 33
	PARALYZEIMMUNE       = 34
	PETRIFIED            = 35
	PETRIFIEDIMMUNE      = 36
	PETSTASIS            = 37
	CURSE                = 38
	CURSEIMMUNE          = 39
	HPBOOST              = 40
	MPBOOST              = 41
	ATKBOOST             = 42
	DEFBOOST             = 43
	SPDBOOST             = 44
	VITBOOST             = 45
	WISBOOST             = 46
	DEXBOOST             = 47
	SILENCED             = 48
	EXPOSED              = 49
	ENERGIZED            = 50
	GROUNDDAMAGE         = 99
	DEADBIT              = 1<<DEAD - 1
	QUIETBIT             = 1<<QUIET - 1
	WEAKBIT              = 1<<WEAK - 1
	SLOWEDBIT            = 1<<SLOWED - 1
	SICKBIT              = 1<<SICK - 1
	DAZEDBIT             = 1<<DAZED - 1
	STUNNEDBIT           = 1<<STUNNED - 1
	BLINDBIT             = 1<<BLIND - 1
	HALLUCINATINGBIT     = 1<<HALLUCINATING - 1
	DRUNKBIT             = 1<<DRUNK - 1
	CONFUSEDBIT          = 1<<CONFUSED - 1
	STUNIMMUNEBIT        = 1<<STUNIMMUNE - 1
	INVISIBLEBIT         = 1<<INVISIBLE - 1
	PARALYZEDBIT         = 1<<PARALYZED - 1
	SPEEDYBIT            = 1<<SPEEDY - 1
	BLEEDINGBIT          = 1<<BLEEDING - 1
	ARMORBROKENIMMUNEBIT = 1<<ARMORBROKENIMMUNE - 1
	HEALINGBIT           = 1<<HEALING - 1
	DAMAGINGBIT          = 1<<DAMAGING - 1
	BERSERKBIT           = 1<<BERSERK - 1
	PAUSEDBIT            = 1<<PAUSED - 1
	STASISBIT            = 1<<STASIS - 1
	STASISIMMUNEBIT      = 1<<STASISIMMUNE - 1
	INVINCIBLEBIT        = 1<<INVINCIBLE - 1
	INVULNERABLEBIT      = 1<<INVULNERABLE - 1
	ARMOREDBIT           = 1<<ARMORED - 1
	ARMORBROKENBIT       = 1<<ARMORBROKEN - 1
	HEXEDBIT             = 1<<HEXED - 1
	NINJASPEEDYBIT       = 1<<NINJASPEEDY - 1
	UNSTABLEBIT          = 1<<UNSTABLE - 1
	DARKNESSBIT          = 1<<DARKNESS - 1
	SLOWEDIMMUNEBIT      = 1<<SLOWIMMUNE - 32
	DAZEDIMMUNEBIT       = 1<<DAZEIMMUNE - 32
	PARALYZEDIMMUNEBIT   = 1<<PARALYZEIMMUNE - 32
	PETRIFIEDBIT         = 1<<PETRIFIED - 32
	PETRIFIEDIMMUNEBIT   = 1<<PETRIFIEDIMMUNE - 32
	PETSTASISBIT         = 1<<PETSTASIS - 32
	CURSEBIT             = 1<<CURSE - 32
	CURSEIMMUNEBIT       = 1<<CURSEIMMUNE - 32
	HPBOOSTBIT           = 1<<HPBOOST - 32
	MPBOOSTBIT           = 1<<MPBOOST - 32
	ATKBOOSTBIT          = 1<<ATKBOOST - 32
	DEFBOOSTBIT          = 1<<DEFBOOST - 32
	SPDBOOSTBIT          = 1<<SPDBOOST - 32
	VITBOOSTBIT          = 1<<VITBOOST - 32
	WISBOOSTBIT          = 1<<WISBOOST - 32
	DEXBOOSTBIT          = 1<<DEXBOOST - 32
	SILENCEDBIT          = 1<<SILENCED - 32
	EXPOSEDBIT           = 1<<EXPOSED - 32
	ENERGIZEDBIT         = 1<<ENERGIZED - 32
)
