package main

import "math"

type GameProjectile struct {
	ContainerType int32 //the id of the producing entity (hex identifier)
	BulletType    int
	OwnerObjectID int32 //the object id of the producing entity
	BulletID      byte
	StartAngle    float32
	StartTime     int32
	StartPosition WorldPosData
	// ContainerProperties
	ProjectileProperties Projectile
	DamagesPlayers       bool
	DamagesEnemies       bool
	Damage               int //the damage to be applied on a hit
	CurrentPosition      WorldPosData
}

func (p *GameProjectile)setDamage(d int) {
	p.Damage = d
}

func (p *GameProjectile)update(t int) bool {
	elapsed := GetTime() - p.StartTime
	if int(elapsed) > p.ProjectileProperties.LifetimeMS {
		return false
	}
	p.CurrentPosition = p.getPositionAt(elapsed)
	return true
}

func (p *GameProjectile) getPositionAt(t int32) WorldPosData {
	ftime := float32(t)
	loc := WorldPosData{
		X: p.StartPosition.X,
		Y: p.StartPosition.Y,
	}
	distanceTravelled := ftime * float32((p.ProjectileProperties.Speed / 10000))
	var phase float32 = 0.0
	if p.BulletID%2 != 0 {
		phase = math.Pi
	}
	if p.ProjectileProperties.Wavy == true {
		newAngle := float32(p.StartAngle) + (math.Pi/64)*f32sin(phase+(6.0*math.Pi)*ftime/1000)
		loc.X += distanceTravelled * f32cos(newAngle)
		loc.Y += distanceTravelled * f32sin(newAngle)
	} else if p.ProjectileProperties.Parametric == true {
		offset1 := ftime / float32(p.ProjectileProperties.LifetimeMS) * 2.0 * math.Pi
		var adjustment float32
		//these adjustments might be buggy due to conversion loss
		if p.BulletID%2 == 1 {
			adjustment = 1
		} else {
			adjustment = -1
		}
		offset2 := f32sin(offset1) * adjustment
		if p.BulletID%4 < 2 {
			adjustment = 1
		} else {
			adjustment = -1
		}
		offset3 := f32sin(2*offset1) * adjustment
		angleX := f32cos(float32(p.StartAngle))
		angleY := f32sin(float32(p.StartAngle))
		loc.X += (offset2*angleY - offset3*angleX) * p.ProjectileProperties.Magnitude
		loc.Y += (offset2*angleX - offset3*angleY) * p.ProjectileProperties.Magnitude
	} else {
		if p.ProjectileProperties.Boomerang == true {
			halfwaySpot := float32(p.ProjectileProperties.LifetimeMS) * float32((p.ProjectileProperties.Speed/10000)/2)
			if distanceTravelled > halfwaySpot {
				distanceTravelled = halfwaySpot - (distanceTravelled - halfwaySpot)
			}
		}
		loc.X += float32(distanceTravelled) * f32cos(float32(p.StartAngle))
		loc.Y += float32(distanceTravelled) * f32sin(float32(p.StartAngle))
		if p.ProjectileProperties.Amplitude != 0 {
			deflection := p.ProjectileProperties.Amplitude * f32sin(phase+ftime/float32(p.ProjectileProperties.LifetimeMS)*p.ProjectileProperties.Frequency*2.0*math.Pi)
			loc.X += deflection * f32cos(float32(p.StartAngle)+math.Pi/2.0)
			loc.Y += deflection * f32sin(float32(p.StartAngle)+math.Pi/2.0)
		}
	}
	return loc
}

func f32sin(f float32) float32 {
	return float32(math.Sin(float64(f)))
}

func f32cos(f float32) float32 {
	return float32(math.Cos(float64(f)))
}
