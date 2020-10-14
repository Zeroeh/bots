package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strconv"
)

// contains all the game object ids, such as vault chests, items, monsters, projectiles, ect

// https://www.binaryhexconverter.com/hex-to-decimal-converter

//maps we use to access object stats/properties
var (
	items       = make(map[int]int) //change value type from int to item struct
	jsontiles   *JSONTiles
	jsonobjects *JSONObject
	tiles       = make(map[int]*Tile)
	objects     = make(map[int]*SimpleObject)
	enemies     = make(map[int]*Object)
	equips      = make(map[int32]*Equip)
	portals     = make(map[string]int)
	containers  = make(map[int]*Container)
	projectiles = make(map[int][]*Projectile) //map[objectid]:[bulletID]projectile
)

// (useful) containers https://static.drips.pw/rotmg/production/current/xml/Containers.xml
const (
	vaultUnlocker    = 811
	vaultChest       = 1284
	vaultChestClosed = 1285
	giftChest        = 1860
	giftChestClosed  = 1859
	// vaultPortal = 
)

// equips https://static.drips.pw/rotmg/production/current/xml/Equip.xml
const ()

func convertToRealTiles() {
	for i := 0; i < len(jsontiles.Tiles); i++ {
		tileint, _ := strconv.ParseUint(jsontiles.Tiles[i].Type, 0, 0)
		t := int(tileint)
		tiles[t] = new(Tile) //create the pointer
		tiles[t].ID = jsontiles.Tiles[i].ID
		max, _ := strconv.Atoi(jsontiles.Tiles[i].MaxDamage)
		min, _ := strconv.Atoi(jsontiles.Tiles[i].MinDamage)
		nowalk := false
		if jsontiles.Tiles[i].NoWalk == "1" {
			nowalk = true
		}
		sink := false
		if jsontiles.Tiles[i].Sink == "1" {
			sink = true
		}
		speed, _ := strconv.ParseFloat(jsontiles.Tiles[i].Speed, 32)
		if speed == 0 {
			speed = 1
		}
		tiles[t].MaxDamage = int(max)
		tiles[t].MinDamage = int(min)
		tiles[t].NoWalk = nowalk
		tiles[t].Sink = sink
		tiles[t].Speed = float32(speed)
		tiles[t].Type = t
	}
}

func extractCondition(r reflect.Value, c map[string]float32) {
	iter := r.MapRange()
	var dura float32 = 0.0
	for iter.Next() {
		k := iter.Key().Interface().(string)
		switch k {
		case "_":
			c[iter.Value().Interface().(string)] = dura
		case "duration":
			dura = atof(iter.Value().Interface().(string))
		}
	}
	for k := range c {
		c[k] = dura
	}
}

func projWrapper(i interface{}) *Projectile {
	return extractProjectile(reflect.ValueOf(i))
}

func extractPortal(js JSONGeneric) {
	name := js.ID
	id := htoi(js.Type)
	portals[name] = id
}

func extractProjectile(r reflect.Value) *Projectile {
	ip := new(Projectile)
	iter := r.MapRange()
	conds := make(map[string]float32)
	for iter.Next() { //get individual properties
		k := iter.Key().Interface().(string)
		v := iter.Value().Elem()
		switch v.Kind() { //determine the type of the map value
		case reflect.Slice: //slice of conditions
			for z := 0; z < v.Len(); z++ {
				extractCondition(v.Index(z).Elem(), conds)
			}
		case reflect.Map: //singular condition
			extractCondition(v, conds)
		case reflect.String: //normal fields
			vs := v.Interface().(string)
			switch k {
			case "ObjectId":
				ip.ObjectID = vs
			case "Amplitude":
				ip.Amplitude = atof(vs)
			case "Frequency":
				ip.Frequency = atof(vs)
			case "Magnitude":
				ip.Magnitude = atof(vs)
			case "Damage":
				ip.Damage = atoi(vs)
			case "Speed":
				ip.Speed = atoi(vs)
			case "MinDamage":
				ip.MinDamage = atoi(vs)
			case "MaxDamage":
				ip.MaxDamage = atoi(vs)
			case "ArmorPiercing":
				ip.ArmorPierce = atob(vs)
			case "Parametric":
				ip.Parametric = atob(vs)
			case "Boomerang":
				ip.Boomerang = atob(vs)
			case "Wavy":
				ip.Wavy = atob(vs)
			case "MultiHit":
				ip.MultiHit = atob(vs)
			case "LifetimeMS":
				ip.LifetimeMS = atoi(vs)
			case "id":
				ip.ID = atoi(vs)
			}
		}
		if len(conds) > 0 {
			ip.Condition = conds
		}
	}
	return ip
}

func extractProjectiles(proj interface{}) []*Projectile {
	p := make([]*Projectile, 0)
	projval := reflect.ValueOf(proj)
	switch projval.Kind() {
	case reflect.Slice: //slice of projectiles
		for z := 0; z < projval.Len(); z++ {
			e := projval.Index(z).Elem()
			switch e.Kind() {
			case reflect.Map:
				p = append(p, extractProjectile(e))
			}
		}
	case reflect.Map: //singular projectile
		p = append(p, extractProjectile(projval))
	}
	return p
}

func extractEquip(js JSONGeneric) *Equip {
	e := new(Equip)
	e.Name = js.ID
	e.ID = htoi(js.Type)
	e.BagType = atoi(js.BagType)
	e.Consumable = atob(js.Consumable)
	e.Description = js.Description
	e.FameBonus = atoi(js.FameBonus)
	e.MpCost = atoi(js.MpCost)
	e.NumProjectiles = atoi(js.NumProjectiles)
	if js.Projectile != nil {
		e.Projectile = *projWrapper(js.Projectile)
	}
	e.Quantity = atoi(js.Quantity)
	e.SlotType = byte(atoi(js.SlotType))
	e.Soulbound = atob(js.Soulbound)
	e.Usable = atob(js.Usable)
	e.RateOfFire = atof(js.RateOfFire)
	e.ArcGap = atof(js.ArcGap)
	return e
}

func extractContainer(js JSONGeneric) *Container {
	c := new(Container)
	return c
}

func convertToRealObjects() {
	for i := 0; i < len(jsonobjects.Obj); i++ {
		typ := htoi(jsonobjects.Obj[i].Type)
		name := jsonobjects.Obj[i].ID
		switch jsonobjects.Obj[i].Class {
		case "Equipment":
			equips[int32(typ)] = extractEquip(jsonobjects.Obj[i])
		case "Character":
			proj := jsonobjects.Obj[i].Projectile
			if proj != nil {
				projectiles[typ] = extractProjectiles(proj)
			}
		case "Projectile":
			// fmt.Println(jsonobjects.Obj[i])
		case "GameObject":
			obj := new(SimpleObject)
			obj.ID = int(typ)
			obj.Name = name
			objects[obj.ID] = obj
		case "Portal":
			extractPortal(jsonobjects.Obj[i])
		case "Container":
			extractContainer(jsonobjects.Obj[i])
		case "OneWayContainer":
			extractContainer(jsonobjects.Obj[i])
		case "Pet":
		case "": //ignore empty
			// default: fmt.Println("Not implemented:", jsonobjects.Obj[i].Class)
		}
	}
}

//see ParseBool documentation on how this works
func atob(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func htoi(s string) int {
	e, _ := strconv.ParseInt(s, 0, 0)
	return int(e)
}

func htoi32(s string) uint16 {
	e, _ := strconv.ParseInt(s, 0, 0)
	return uint16(e)
}

func atof(s string) float32 {
	f, _ := strconv.ParseFloat(s, 32)
	return float32(f)
}

func atoi(s string) int {
	x, _ := strconv.Atoi(s)
	return x
}

//SimpleObject is an object where we just want the types id and the name of the object
type SimpleObject struct {
	ID   int
	Name string
}

type Object struct {
	Type      int
	ID        string
	DisplayID string
	Enemy     bool
	Class     string
}

type Tile struct {
	Type      int
	ID        string
	Sink      bool
	Speed     float32
	NoWalk    bool
	MinDamage int
	MaxDamage int
}

type Projectile struct {
	ID          int
	ObjectID    string  //name of the projectile
	Damage      int     `json:"Damage"`
	MinDamage   int     `json:"MinDamage"` //for player items
	MaxDamage   int     `json:"MaxDamage"` //for player items
	Speed       int     `json:"Speed"`
	MultiHit    bool    `json:"MultiHit"`
	ArmorPierce bool    `json:"ArmorPierce"`
	LifetimeMS  int     `json:"LifetimeMS"`
	Amplitude   float32 `json:"Amplitude"`
	Frequency   float32 `json:"Frequency"`
	Magnitude   float32
	Boomerang   bool `json:"Boomerang"`
	Parametric  bool `json:"Parametric"`
	Wavy        bool `json:"Wavy"`
	Condition   map[string]float32
	// Condition   []*Condition `json:"ConditionEffect"`
}

type Condition struct {
	Cond     string  `json:"_"`
	Duration float32 `json:"duration"`
}

type Equip struct {
	ID             int
	Name           string
	SlotType       byte
	Description    string
	Quantity       int
	Activate       string
	Usable         bool //abilities
	FameBonus      int
	MpCost         int
	Soulbound      bool
	Consumable     bool
	BagType        int
	NumProjectiles int
	RateOfFire     float32
	ArcGap         float32
	Projectile     Projectile
}

type Container struct {
	CanPutNormal    bool //not sure what these items are???
	CanPutSoulbound bool
}

//probably the same as JSONCondition, but too assed to double check
type Effect struct {
	Cond     string `json:"_"`
	Effect   string `json:"effect"`
	Duration string `json:"duration"`
}

/* All yee who wander past here shall suffer an un-curable illness known as  "WTF IS THIS JSON FILE PARSING???" */

func readTiles() {
	f, err := os.Open(directoryAppend + "resources/GroundTypes.json")
	if err != nil {
		fmt.Printf("error opening GroundTypes.json: %s\n", err)
		os.Exit(1)
	}
	if err := json.NewDecoder(f).Decode(&jsontiles); err != nil {
		fmt.Printf("error decoding GroundTypes.json: %s\n", err)
		os.Exit(1)
	}
	f.Close()
	convertToRealTiles()
	jsontiles = nil //free up memory
}

func readObjects() {
	f, err := os.Open(directoryAppend + "resources/Objects.json")
	if err != nil {
		fmt.Printf("error opening Objects.json: %s\n", err)
		os.Exit(1)
	}
	dec := json.NewDecoder(f)
	err = dec.Decode(&jsonobjects)
	if err != nil {
		fmt.Printf("error decoding Objects.json: %s\n", err)
		os.Exit(1)
	}
	f.Close()
	convertToRealObjects()
	jsonobjects = nil //free up memory
}

//JSONObject is for storing our json reads
type JSONObject struct {
	Obj []JSONGeneric `json:"Object"`
}

/* Classes available (not an exhaustive list):
GameObject
Character
Player
Equipment
Projectile
Skin
Wall
Dye
*/
type JSONGeneric struct {
	Type       string      `json:"type"`      //integer id of object
	ID         string      `json:"id"`        //string name of object
	DisplayID  string      `json:"DisplayId"` //common string name of the object
	Enemy      string      `json:"Enemy"`     //enemy bool
	Item       string      `json:"Item"`      //equip bool
	Class      string      `json:"Class"`
	Projectile interface{} `json:"Projectile"` //projectile array
	// Projectile []struct {
	// 	ID          string `json:"id"`
	// 	ObjectID    string `json:"ObjectId"`
	// 	Damage      string `json:"Damage"`
	// 	MinDamage   string `json:"MinDamage"` //for player items
	// 	MaxDamage   string `json:"MaxDamage"` //for player items
	// 	Speed       string `json:"Speed"`
	// 	MultiHit    string `json:"MultiHit"`
	// 	ArmorPierce string `json:"ArmorPierce"`
	// 	LifetimeMS  string `json:"LifetimeMS"`
	// 	Amplitude   string `json:"Amplitude"`
	// 	Frequency   string `json:"Frequency"`
	// 	Boomerang   string `json:"Boomerang"`
	// 	Parametric  string `json:"Parametric"`
	// 	Cond        []struct {
	// 	} `json:"ConditionEffect"`
	// } `json:"Projectile"`
	Defense        interface{} `json:"Defense"`
	HP             interface{} `json:"MaxHitPoints"`
	SlotType       string      `json:"SlotType"`  //for equipment
	Usable         string      `json:"Usable"`    //for equipment
	Soulbound      string      `json:"Soulbound"` //for equipment
	FameBonus      string      `json:"FameBonus"`
	Consumable     string      `json:"Consumable"`
	MpCost         string      `json:"MpCost"`
	BagType        string      `json:"BagType"`
	Quantity       string      `json:"Quantity"`
	Description    string      `json:"Description"`
	NumProjectiles string      `json:"NumProjectiles"`
	ArcGap         string      `json:"ArcGap"`
	RateOfFire     string      `json:"RateOfFire"` //for equipment
	Cooldown       string      `json:"Cooldown"`   //for equipment:abilities
	Activate       interface{} `json:"Activate"`
}

type JSONTiles struct {
	Tiles []JSONTile `json:"Ground"`
}

type JSONTile struct {
	Type      string `json:"type"` //the integer type of the object
	ID        string `json:"id"`   //the string name of the object
	Sink      string `json:"Sink"`
	Speed     string `json:"Speed"` //speed mult for the tile
	NoWalk    string `json:"NoWalk"`
	MinDamage string `json:"MinDamage"`
	MaxDamage string `json:"MaxDamage"`
}
