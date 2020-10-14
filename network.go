package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

//All network and high-level packet based functions based here

const packetSize = 420 //only needs to be large enough to handle packets we send
// const gameToken = "XTeP7hERdchV5jrBZEYNebAqDPU6tKU6" //flash
const gameToken = "8bV53M5ysJdVjU4M97fh2g7BnPXhefnc" //exalt

//GameConnection is the base socket configuration for a client
type GameConnection struct {
	InLoop      bool
	Killed      bool
	Connected   bool
	SocketDebug bool
	ConnHandle  net.Conn
	GameSocket  *bufio.ReadWriter
	KeyPair     *Ciphers
}

//Deprecated
func (p *Packet) Init() {
	p.Length = 0
	p.ID = 0
	p.Index = 5 //start at the first byte of data
	p.Data = make([]byte, packetSize)
}

//Deprecated
func NewPacket(v interface{}) *Packet {
	p := new(Packet)
	p.Init()
	switch v.(type) { //this is awesome
	case byte:
		p.ID = int(v.(uint8))
	default:
		p.ID = InferIDFromPacket(v)
	}
	return p
}

//NewReceivePacket creates a packet pointer based on the bytes received
func NewReceivePacket(b []byte) *Packet {
	if b == nil {
		return nil
	}
	p := new(Packet)
	if len(b) != 5 {
		fmt.Println("(NewReceivePacket)Did not get 5 bytes, got:", b)
		return nil
	}
	p.Index = 5
	p.ID = int(b[4])
	p.Length = uint(binary.BigEndian.Uint32(b[0:4]))
	if p.Length < 100000 {
		p.Data = make([]byte, p.Length+5)
	} else {
		fmt.Println(b)
		//todo: resize?
		return nil
	}
	p.Data[0] = b[0]
	p.Data[1] = b[1]
	p.Data[2] = b[2]
	p.Data[3] = b[3]
	p.Data[4] = b[4]
	return p
}

//CipherData Rc4 encrypts the given packet data
func (p *Packet) CipherData(c *Cipher) {
	c.XorKeyStreamGeneric(p.Data[5:p.Length], p.Data[5:p.Length])
}

//ResizeBuffer resizes our packet to include the packet data portion (this vs alloc large buffer and shrink)
func (p *Packet) ResizeBuffer(size int) {
	oldBuffer := p.Data
	p.Data = make([]byte, size)
	//keep the size in the new buffer
	p.Data[0] = oldBuffer[0]
	p.Data[1] = oldBuffer[1]
	p.Data[2] = oldBuffer[2]
	p.Data[3] = oldBuffer[3]
	p.Data[4] = oldBuffer[4] //dont forget our packet id
}

//Send writes packets data to GameConnection as well as sends it out
func (g *GameConnection) Send(p *Packet) {
	if g.SocketDebug == true {
		fmt.Println("Send:", p.Data)
	}
	if g.GameSocket == nil { //dont write anything if we dont have a socket
		return
	}
	p.CipherData(g.KeyPair.SendCipher)
	bytes, err := g.GameSocket.Writer.Write(p.Data)
	if err != nil {
		if strings.Contains(err.Error(), "broken pipe") { //if it's just the broken pipe message then i dont want to see it
			g.KillConnection()
			return
		}
		if strings.Contains(err.Error(), "reset") {
			g.KillConnection()
			return
		}
		fmt.Println("Write error:", err)
		return
	}
	if uint(bytes) != p.Length {
		fmt.Printf("Bytes Written: %d | Packet length: %d\n", bytes, p.Length)
	}
	err = g.GameSocket.Flush()
	if err != nil {
		if strings.Contains(err.Error(), "broken pipe") {
			g.KillConnection()
			return
		}
		if strings.Contains(err.Error(), "reset") {
			g.KillConnection()
			return
		}
		fmt.Println("Flush error:", err)
		return
	}
}

//Receive is the main receiving function for the socket
func (c *Client) Receive() {
	var bytesRead = 5
	var goalBytes = 0
	var err error
	var read int
	for c.Connection.Killed != true && c.Recon.ReconQueued == false { //loop until dc'ed
		p := NewReceivePacket(c.GetPacketHeader())
		if p == nil {
			SwitchColor(Red)
			fmt.Printf("%s received nil packet\n", c.Base.Email)
			SwitchColor(Normal)
			c.QueueRecon(c.Recon.GameID, c.Recon.GameKey, c.Recon.GameKeyTime)
			return
		}
		bytesRead = 5
		goalBytes = len(p.Data) - bytesRead
		for bytesRead != goalBytes {
			if c.Connection.GameSocket == nil {
				break //return or break???
			}
			read, err = c.Connection.GameSocket.Read(p.Data[bytesRead:goalBytes])
			if err != nil {
				break
			}
			bytesRead += read
		}
		p.CipherData(c.Connection.KeyPair.RecvCipher)
		c.EvaluatePacket(ReadPacket(p))
	}
}

func (c *Client) GetPacketHeader() []byte {
	bytesRead := 0
	buffer := make([]byte, 5)
	var err error
	var read int
	for {
		if c.Connection.GameSocket == nil {
			return nil
		}
		read, err = c.Connection.GameSocket.Read(buffer[bytesRead:])
		if err != nil {
			return nil
		}
		bytesRead += read
		if buffer[0] == 255 {
			return nil
			//queue recon?
		}
		if bytesRead == 5 {
			return buffer
		}
	}
}

//InitGameConnection starts up the socket for the client as well as sets up the rc4 key pair
func (c *Client) InitGameConnection() {
	c.Connection = GameConnection{}
	theServer := ""
	if isIPAddress(c.Recon.CurrentServer) == false {
		theServer = serverNameToIP(c.Recon.CurrentServer) + ":2050"
	} else {
		theServer = c.Recon.CurrentServer + ":2050"
	}

	if c.Base.UseSocks == true {
		proxyDial, err := proxy.SOCKS5("tcp", c.Base.SockProxy, nil, proxy.Direct)
		if err != nil {
			log.Println("Error resolving proxy:", err)
			// c.blackListIP() //this should only run if the connection timed out, not if it was refused
			return
		}
		conn, err := proxyDial.Dial("tcp", theServer)
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") == true {
				log.Printf("%s | The server refused connection from the proxy: %s->%s\n", c.Base.Email, c.Base.SockProxy, c.Base.ServerIP)
				// c.KillClient() //temp fix
				time.Sleep(time.Millisecond * 5000)
				c.Recon.ReconAttempts++
				return
			} else {
				log.Println("Error dialing server via proxy:", err)
			}
			// c.blackListIP() //this should only run if the connection timed out, not if it was refused
			return
		}
		tcpConn, ok := conn.(*net.TCPConn)
		if ok {
			err = tcpConn.SetNoDelay(true)
			if err != nil {
				log.Println("Error setting no delay:", err)
				return
			}
			c.Connection.WrapSocket(tcpConn)
		} else { //fallback if it doesnt work
			c.Connection.WrapSocket(conn)
		}
	} else {
		hostIP, err := net.ResolveTCPAddr("tcp", theServer)
		if err != nil {
			log.Println("Error resolving server:", err)
			return
		}
		conn, err := net.DialTCP("tcp", nil, hostIP)
		if err != nil {
			log.Println("Error dialing server:", err)
			return
		}
		err = conn.SetNoDelay(true)
		if err != nil {
			log.Println("Error setting no delay:", err)
			return
		}
		c.Connection.WrapSocket(conn)
	}
	c.Connection.InitCiphers()
	if c.Debugging == true {
		c.Connection.SocketDebug = true
	}
	c.Connection.Connected = true
	c.Recon.ReconnectOnError = true
	c.Running = true
}

func (g *GameConnection) WrapSocket(c net.Conn) {
	g.Killed = false
	g.ConnHandle = c
	g.GameSocket = bufio.NewReadWriter(bufio.NewReader(c), bufio.NewWriter(c))
}

func (g *GameConnection) InitCiphers() {
	g.KeyPair = new(Ciphers)
	g.KeyPair.SendCipher, _ = NewCipher([]byte(RC4OutgoingKey))
	g.KeyPair.RecvCipher, _ = NewCipher([]byte(RC4IncomingKey))
}

//SendHello creates a hello packet and sends it to the currently connected server
func (c *Client) SendHello(key []byte, keyTime int32) {
	hello := Hello{}
	hello.BuildVersion = settings.GameVersion
	hello.GameID = c.Recon.GameID //implicitly used
	if isEmail(c.Base.Email) {
		hello.GUID = c.Base.Email
		hello.Password = c.Base.Password
		hello.Secret = ""
	} else { //steam / kong
		hello.GUID = c.Base.Email
		hello.Password = ""
		hello.Secret = c.Base.Password // since we don't have an @, the secret is the email
	}

	hello.Random1 = rand.Int31()
	// hello.Password = c.Base.Password
	hello.Random2 = rand.Int31()
	// hello.Secret = "" //this gets rsa encrypted
	hello.KeyTime = uint32(keyTime)
	hello.Key = key
	hello.MapJSON = ""
	hello.EntryTag = ""
	hello.GameNet = "rotmg" //default
	hello.GameNetUserID = ""
	hello.PlayPlatform = "rotmg" //default
	hello.PlatformToken = ""
	hello.UserToken = ""
	hello.ClientToken = gameToken
	hello.PreviousConnectionGUID = c.Recon.ConnectionGUID
	// uuid version 4 variant 1 => 7794C8EA-A7C3-457B-A57C-6992EBE339B2
	// hello.PreviousConnectionGUID = "AAAAAAAA-AAAA-1AAA-AAAA-------------"
	// fmt.Println("PreviousConnectionGUID:", c.Recon.ConnectionGUID)
	c.Connection.Connected = true
	// fmt.Println("Sent hello!") //debug
	// fmt.Println(hello)
	c.Connection.Send(WritePacket(hello)) //outgoing data should be 442 bytes as of Oct 10 2019
}

//Kill will zero out the ciphers and socket so the GC can reclaim memory. Not sure if socket needs to explicitly be closed, but adding the explicit close anyways
func (g *GameConnection) KillConnection() {
	g.Connected = false
	g.Killed = true
	g.InLoop = false
	if g.KeyPair != nil {
		g.KeyPair.Kill()
		g.KeyPair = nil
	}
	if g.ConnHandle != nil {
		g.ConnHandle.Close()
	}
	g.GameSocket = nil
	//runtime.GC() //shouldnt need any more since the reconnecting behavior was redone
	//time.Sleep(20000 * time.Millisecond)
}

func InferIDFromPacket(v interface{}) int {
	switch v.(type) {
	case Failure:
		return FailureID
	case CreateSuccess:
		return CreateSuccessID
	case Create:
		return CreateID
	case PlayerShoot:
		return PlayerShootID
	case Move:
		return MoveID
	case PlayerText:
		return PlayerTextID
	case Text:
		return TextID
	case ServerPlayerShoot:
		return ServerPlayerShootID
	case Damage:
		return DamageID
	case Update:
		return UpdateID
	case UpdateAck:
		return UpdateAckID
	case Notification:
		return NotificationID
	case NewTick:
		return NewTickID
	case InvSwap:
		return InvSwapID
	case UseItem:
		return UseItemID
	case ShowEffect:
		return ShowEffectID
	case Hello:
		return HelloID
	case Goto:
		return GotoID
	case InvDrop:
		return InvDropID
	case InvResult:
		return InvResultID
	case Reconnect:
		return ReconnectID
	case Ping:
		return PingID
	case Pong:
		return PongID
	case MapInfo:
		return MapInfoID
	case Load:
		return LoadID
	case Pic:
		return PicID
	case SetCondition:
		return SetConditionID
	case Teleport:
		return TeleportID
	case UsePortal:
		return UsePortalID
	case Death:
		return DeathID
	case Buy:
		return BuyID
	case BuyResult:
		return BuyResultID
	case AoE:
		return AoEID
	case GroundDamage:
		return GroundDamageID
	case PlayerHit:
		return PlayerHitID
	case EnemyHit:
		return EnemyHitID
	case AoEAck:
		return AoEAckID
	case ShootAck:
		return ShootAckID
	case OtherHit:
		return OtherHitID
	case SquareHit:
		return SquareHitID
	case GotoAck:
		return GotoAckID
	case EditAccountList:
		return EditAccountListID
	case AccountList:
		return AccountListID
	case QuestObjID:
		return QuestObjIDID
	case ChooseName:
		return ChooseNameID
	case NameResult:
		return NameResultID
	case CreateGuild:
		return CreateGuildID
	case CreateGuildResult:
		return CreateGuildResultID
	case GuildRemove:
		return GuildRemoveID
	case GuildInvite:
		return GuildInviteID
	case AllyShoot:
		return AllyShootID
	case EnemyShoot:
		return EnemyShootID
	case RequestTrade:
		return RequestTradeID
	case TradeRequested:
		return TradeRequestedID
	case TradeStart:
		return TradeStartID
	case ChangeTrade:
		return ChangeTradeID
	case TradeChanged:
		return TradeChangedID
	case AcceptTrade:
		return AcceptTradeID
	case CancelTrade:
		return CancelTradeID
	case TradeDone:
		return TradeDoneID
	case TradeAccepted:
		return TradeAcceptedID
	case ClientStat:
		return ClientStatID
	case CheckCredits:
		return CheckCreditsID
	case Escape:
		return EscapeID
	case File:
		return FileID
	case InvitedToGuild:
		return InvitedToGuildID
	case JoinGuild:
		return JoinGuildID
	case ChangeGuildRank:
		return ChangeGuildRankID
	case PlaySound:
		return PlaySoundID
	case GlobalNotification:
		return GlobalNotificationID
	case ReSkin:
		return ReSkinID
	case PetUpgradeRequest:
		return PetUpgradeRequestID
	case ActivePetUpdateSend:
		return ActivePetUpdateSendID
	case ActivePetUpdateRecv:
		return ActivePetUpdateRecvID
	case NewAbility:
		return NewAbilityID
	case PetYardUpdate:
		return PetYardUpdateID
	case EvolvePet:
		return EvolvePetID
	case DeletePet:
		return DeletePetID
	case HatchPet:
		return HatchPetID
	case EnterArena:
		return EnterArenaID
	case ImminentArenaWave:
		return ImminentArenaWaveID
	case ArenaDeath:
		return ArenaDeathID
	case AcceptArenaDeath:
		return AcceptArenaDeathID
	case VerifyEmail:
		return VerifyEmailID
	case ReSkinUnlock:
		return ReSkinUnlockID
	case PasswordPrompt:
		return PasswordPromptID
	case QuestFetchAsk:
		return QuestFetchAskID
	case QuestRedeem:
		return QuestRedeemID
	case QuestFetchResponse:
		return QuestFetchResponseID
	case QuestRedeemResponse:
		return QuestRedeemResponseID
	case PetChangeFormMessage:
		return PetChangeFormMessageID
	case KeyInfoRequest:
		return KeyInfoRequestID
	case KeyInfoResponse:
		return KeyInfoResponseID
	case LoginRewardSend:
		return LoginRewardSendID
	case LoginRewardRecv:
		return LoginRewardRecvID
	case QuestRoomMessage:
		return QuestRoomMessageID
	case ChatHello:
		return ChatHelloID
	case ChatToken:
		return ChatTokenID
	case ChatLogout:
		return ChatLogoutID
	case RealmHeroLeft:
		return RealmHeroLeftID
	case QueueInformation:
		return QueueInformationID
	case NewCharacterInformation:
		return NewCharacterInformationID
	case UnlockInformation:
		return UnlockInformationID
	case Unknown:
		return UnknownID
	default:
		return 255
	}
}

const (
	FailureID                 = 0
	CreateSuccessID           = 101
	CreateID                  = 61
	PlayerShootID             = 30
	MoveID                    = 42
	PlayerTextID              = 10
	TextID                    = 44
	ServerPlayerShootID       = 12
	DamageID                  = 75
	UpdateID                  = 62
	UpdateAckID               = 81
	NotificationID            = 67
	NewTickID                 = 9
	InvSwapID                 = 19
	UseItemID                 = 11
	ShowEffectID              = 13
	HelloID                   = 1
	GotoID                    = 18
	InvDropID                 = 55
	InvResultID               = 95
	ReconnectID               = 45
	PingID                    = 8
	PongID                    = 31
	MapInfoID                 = 92
	LoadID                    = 57
	PicID                     = 83
	SetConditionID            = 60
	TeleportID                = 74
	UsePortalID               = 47
	DeathID                   = 46
	BuyID                     = 85
	BuyResultID               = 22
	AoEID                     = 64
	GroundDamageID            = 103
	PlayerHitID               = 90
	EnemyHitID                = 25
	AoEAckID                  = 89
	ShootAckID                = 100
	OtherHitID                = 20
	SquareHitID               = 40
	GotoAckID                 = 65
	EditAccountListID         = 27
	AccountListID             = 99
	QuestObjIDID              = 82
	ChooseNameID              = 97
	NameResultID              = 21
	CreateGuildID             = 59
	CreateGuildResultID       = 26
	GuildRemoveID             = 15
	GuildInviteID             = 104
	AllyShootID               = 49
	EnemyShootID              = 35
	RequestTradeID            = 5
	TradeRequestedID          = 88
	TradeStartID              = 86
	ChangeTradeID             = 56
	TradeChangedID            = 28
	AcceptTradeID             = 36
	CancelTradeID             = 91
	TradeDoneID               = 34
	TradeAcceptedID           = 14
	ClientStatID              = 69
	CheckCreditsID            = 102
	EscapeID                  = 105
	FileID                    = 106
	InvitedToGuildID          = 77
	JoinGuildID               = 7
	ChangeGuildRankID         = 37
	PlaySoundID               = 38
	GlobalNotificationID      = 66
	ReSkinID                  = 51
	PetUpgradeRequestID       = 16
	ActivePetUpdateSendID     = 24
	ActivePetUpdateRecvID     = 76
	NewAbilityID              = 41
	PetYardUpdateID           = 78
	EvolvePetID               = 87
	DeletePetID               = 4
	HatchPetID                = 23
	EnterArenaID              = 17
	ImminentArenaWaveID       = 50
	ArenaDeathID              = 68
	AcceptArenaDeathID        = 80
	VerifyEmailID             = 39
	ReSkinUnlockID            = 107
	PasswordPromptID          = 79
	QuestFetchAskID           = 98
	QuestRedeemID             = 58
	QuestFetchResponseID      = 6
	QuestRedeemResponseID     = 96
	PetChangeFormMessageID    = 53
	KeyInfoRequestID          = 94
	KeyInfoResponseID         = 63
	LoginRewardSendID         = 3
	LoginRewardRecvID         = 93
	QuestRoomMessageID        = 48
	PetChangeSkinID           = 33
	ChatHelloID               = 206
	ChatTokenID               = 207
	ChatLogoutID              = 208
	RealmHeroLeftID           = 84
	ResetDailyQuestsID        = 52
	QueueInformationID        = 112
	UnlockInformationID       = 109
	NewCharacterInformationID = 108
	UnknownID                 = 114
)
