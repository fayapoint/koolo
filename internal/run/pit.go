package run

import (
	"time"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/area"
	"github.com/hectorgimenez/koolo/internal/action"
	"github.com/hectorgimenez/koolo/internal/character"
	"github.com/hectorgimenez/koolo/internal/config"
	"github.com/hectorgimenez/koolo/internal/context"
)

type Pit struct {
	ctx *context.Status
}

func NewPit() *Pit {
	return &Pit{
		ctx: context.Get(),
	}
}

func (p Pit) Name() string {
	return string(config.PitRun)
}

func (p Pit) Run() error {
	// Define a default filter
	monsterFilter := data.MonsterAnyFilter()

	// Update filter if we selected to clear only elites
	if p.ctx.CharacterCfg.Game.Pit.FocusOnElitePacks {
		monsterFilter = data.MonsterEliteFilter()
	}

	if !p.ctx.CharacterCfg.Game.Pit.MoveThroughBlackMarsh {
		err := action.WayPoint(area.OuterCloister)
		if err != nil {
			return err
		}

		if err = action.MoveToArea(area.MonasteryGate); err != nil {
			return err
		}

		if err = action.MoveToArea(area.TamoeHighland); err != nil {
			return err
		}
	} else {
		err := action.WayPoint(area.BlackMarsh)
		if err != nil {
			return err
		}

		if err = action.MoveToArea(area.TamoeHighland); err != nil {
			return err
		}
	}
	if err := action.MoveToArea(area.PitLevel1); err != nil {
		return err
	}

	// Open a TP If we're the leader
	action.OpenTPIfLeader()
	// Clear the area if we don't have only clear lvl2 selected
	if !p.ctx.CharacterCfg.Game.Pit.OnlyClearLevel2 {
		if err := action.ClearCurrentLevel(p.ctx.CharacterCfg.Game.Pit.OpenChests, monsterFilter); err != nil {
			return err
		}
	}

	// Move to PitLvl2
	if err := action.MoveToArea(area.PitLevel2); err != nil {
		return err
	}

	// Clear it
	if err := action.ClearCurrentLevel(p.ctx.CharacterCfg.Game.Pit.OpenChests, monsterFilter); err != nil {
		return err
	}

	// Charge Battery strategy for Mosaic Assassin in low-density areas
	if mosaicChar, ok := p.ctx.Char.(character.MosaicSin); ok {
		if p.ctx.CharacterCfg.Character.MosaicSin.UseChargeBattery {
			duration := p.ctx.CharacterCfg.Character.MosaicSin.ChargeBatteryDuration
			if duration == 0 {
				duration = 30 // Default 30 seconds
			}
			
			p.ctx.Logger.Info("Pit cleared - attempting charge battery strategy for low-density area")
			if err := mosaicChar.MaintainChargesWithBattery(time.Duration(duration) * time.Second); err != nil {
				p.ctx.Logger.Warn("Charge battery strategy failed, continuing normally", "error", err.Error())
			}
		}
	}

	return action.DisplayItemsWithAlt()
}
