package run

import (
	"errors"
	"time"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/area"
	"github.com/hectorgimenez/d2go/pkg/data/npc"
	"github.com/hectorgimenez/d2go/pkg/data/object"
	"github.com/hectorgimenez/koolo/internal/action"
	"github.com/hectorgimenez/koolo/internal/config"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/utils"
)

var baalThronePosition = data.Position{
	X: 15095,
	Y: 5042,
}

type Baal struct {
	ctx                *context.Status
	clearMonsterFilter data.MonsterFilter // Used to clear area (basically TZ)
	preAtkLast         time.Time
	decoyLast          time.Time
}

func NewBaal(clearMonsterFilter data.MonsterFilter) *Baal {
	return &Baal{
		ctx:                context.Get(),
		clearMonsterFilter: clearMonsterFilter,
	}
}

func (s Baal) Name() string {
	return string(config.BaalRun)
}

func (s *Baal) Run() error {
	// Set filter
	filter := data.MonsterAnyFilter()
	if s.ctx.CharacterCfg.Game.Baal.OnlyElites {
		filter = data.MonsterEliteFilter()
	}
	if s.clearMonsterFilter != nil {
		filter = s.clearMonsterFilter
	}

	err := action.WayPoint(area.TheWorldStoneKeepLevel2)
	if err != nil {
		return err
	}

	if s.ctx.CharacterCfg.Game.Baal.ClearFloors || s.clearMonsterFilter != nil {
		action.ClearCurrentLevel(false, filter)
	}

	err = action.MoveToArea(area.TheWorldStoneKeepLevel3)
	if err != nil {
		return err
	}

	if s.ctx.CharacterCfg.Game.Baal.ClearFloors || s.clearMonsterFilter != nil {
		action.ClearCurrentLevel(false, filter)
	}

	err = action.MoveToArea(area.ThroneOfDestruction)
	if err != nil {
		return err
	}
	err = action.MoveToCoords(baalThronePosition)
	if err != nil {
		return err
	}
	if s.checkForSoulsOrDolls() {
		return errors.New("souls or dolls detected, skipping")
	}

	// Let's move to a safe area and open the portal in companion mode
	if s.ctx.CharacterCfg.Companion.Leader {
		action.MoveToCoords(data.Position{
			X: 15116,
			Y: 5071,
		})
		action.OpenTPIfLeader()
	}

	err = action.ClearAreaAroundPlayer(50, data.MonsterAnyFilter())
	if err != nil {
		return err
	}

	// Force rebuff before waves
	action.Buff()

	// Come back to previous position
	err = action.MoveToCoords(baalThronePosition)
	if err != nil {
		return err
	}

	lastWave := false
	for !lastWave {
		if _, found := s.ctx.Data.Monsters.FindOne(npc.BaalsMinion, data.MonsterTypeMinion); found {
			lastWave = true
		}

		if baalPortal, foundPortal := s.ctx.Data.Objects.FindOne(object.BaalsPortal); foundPortal {
			if baalPortal.Selectable {
				lastWave = true
			}
		}

		// Return to throne position between waves
		err = action.ClearAreaAroundPosition(baalThronePosition, 50, data.MonsterAnyFilter())
		if err != nil {
			return err
		}

		action.MoveToCoords(baalThronePosition)

		// Preattack between waves (inspired by kolbot baal.js)
		//s.preAttackBaalWaves() //tempoarily disabled until fixed
	}

	// Let's be sure everything is dead
	err = action.ClearAreaAroundPosition(baalThronePosition, 50, data.MonsterAnyFilter())

	_, isLevelingChar := s.ctx.Char.(context.LevelingCharacter)
	if s.ctx.CharacterCfg.Game.Baal.KillBaal || isLevelingChar {
		utils.Sleep(15000)
		
		// Check if Mosaic - they should already have charges, skip pre-portal buff
		isMosaic := s.ctx.CharacterCfg.Character.Class == "mosaic"
		
		if !isMosaic {
			// Non-Mosaic characters: buff before entering portal
			action.Buff()
			utils.Sleep(500) // Small delay to let buff animations complete
		} else {
			s.ctx.Logger.Info("Mosaic detected: Skipping pre-portal buff to preserve charges and prevent portal interaction issues")
		}
		
		// Move to a position closer to the portal to ensure we can interact with it
		if err = action.MoveToCoords(data.Position{X: 15095, Y: 5029}); err != nil {
			s.ctx.Logger.Warn("Failed to move closer to portal, attempting interaction anyway", "error", err)
		}
		
		// Exception: Baal portal has no destination in memory
		baalPortal, found := s.ctx.Data.Objects.FindOne(object.BaalsPortal)
		if !found {
			return errors.New("baal portal not found")
		}
		
		s.ctx.Logger.Info("Entering Worldstone Chamber through Baal's portal")
		err = action.InteractObject(baalPortal, func() bool {
			return s.ctx.Data.PlayerUnit.Area == area.TheWorldstoneChamber
		})
		if err != nil {
			return err
		}

		s.ctx.Logger.Info("Entered Worldstone Chamber, moving to Baal's position")
		_ = action.MoveToCoords(data.Position{X: 15136, Y: 5943})
		
		// Buff AFTER entering chamber - now it's safe and won't interrupt anything
		action.Buff()
		s.ctx.Logger.Info("Buffed inside Worldstone Chamber, ready for Baal fight")

		if err := s.ctx.Char.KillBaal(); err != nil {
			return err
		}

		// Display items with ALT if configured
		return action.DisplayItemsWithAlt()
	}

	// Display items with ALT if configured
	return action.DisplayItemsWithAlt()
}

func (s Baal) checkForSoulsOrDolls() bool {
	var npcIds []npc.ID

	if s.ctx.CharacterCfg.Game.Baal.DollQuit {
		npcIds = append(npcIds, npc.UndeadStygianDoll2, npc.UndeadSoulKiller2)
	}
	if s.ctx.CharacterCfg.Game.Baal.SoulQuit {
		npcIds = append(npcIds, npc.BlackSoul2, npc.BurningSoul2)
	}

	for _, id := range npcIds {
		if _, found := s.ctx.Data.Monsters.FindOne(id, data.MonsterTypeNone); found {
			return true
		}
	}

	return false
}
