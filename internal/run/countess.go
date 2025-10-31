package run

import (
	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/area"
	"github.com/hectorgimenez/koolo/internal/action"
	"github.com/hectorgimenez/koolo/internal/config"
	"github.com/hectorgimenez/koolo/internal/context"
)

type Countess struct {
	ctx *context.Status
}

func NewCountess() *Countess {
	return &Countess{
		ctx: context.Get(),
	}
}

func (c Countess) Name() string {
	return string(config.CountessRun)
}

func (c Countess) Run() error {
	// Travel to boss level
	err := action.WayPoint(area.BlackMarsh)
	if err != nil {
		return err
	}

	// Set filter based on elite focus setting
	filter := data.MonsterAnyFilter()
	if c.ctx.CharacterCfg.Game.Countess.FocusOnElitePacks {
		filter = data.MonsterEliteFilter()
	}

	// Define areas with their corresponding clear settings
	areaConfigs := []struct {
		id         area.ID
		shouldClear bool
	}{
		{area.ForgottenTower, c.ctx.CharacterCfg.Game.Countess.ClearForgottenTower},
		{area.TowerCellarLevel1, c.ctx.CharacterCfg.Game.Countess.ClearTowerCellar1},
		{area.TowerCellarLevel2, c.ctx.CharacterCfg.Game.Countess.ClearTowerCellar2},
		{area.TowerCellarLevel3, c.ctx.CharacterCfg.Game.Countess.ClearTowerCellar3},
		{area.TowerCellarLevel4, c.ctx.CharacterCfg.Game.Countess.ClearTowerCellar4},
		{area.TowerCellarLevel5, c.ctx.CharacterCfg.Game.Countess.ClearTowerCellar5},
	}

	// Process each area based on configuration
	for i, areaConfig := range areaConfigs {
		// Always move to the area (needed for progression)
		err = action.MoveToArea(areaConfig.id)
		if err != nil {
			return err
		}

		// Clear area if configured
		if areaConfig.shouldClear || c.ctx.CharacterCfg.Game.Countess.ClearFloors {
			if c.ctx.CharacterCfg.Game.Countess.ClearOnlyPath {
				// Get the next area's destination for clearing path
				var destPos data.Position
				if i < len(areaConfigs)-1 {
					// Get next area entrance position from adjacent levels
					nextAreaID := areaConfigs[i+1].id
					for _, lvl := range c.ctx.Data.AdjacentLevels {
						if lvl.Area == nextAreaID {
							destPos = lvl.Position
							break
						}
					}
				} else {
					// Last level - use Countess position
					areaData := c.ctx.Data.Areas[area.TowerCellarLevel5]
					countessNPC, found := areaData.NPCs.FindOne(740)
					if found {
						destPos = countessNPC.Positions[0]
					}
				}
				// Clear only path to next area with proper destination
				if destPos.X != 0 || destPos.Y != 0 {
					action.ClearThroughPath(destPos, 15, filter)
				}
			} else {
				// Clear entire area
				action.ClearCurrentLevel(false, filter)
			}
		}
	}

	err = action.MoveTo(func() (data.Position, bool) {
		areaData := c.ctx.Data.Areas[area.TowerCellarLevel5]
		countessNPC, found := areaData.NPCs.FindOne(740)
		if !found {
			return data.Position{}, false
		}

		return countessNPC.Positions[0], true
	})
	if err != nil {
		return err
	}

	// Kill Countess
	err = c.ctx.Char.KillCountess()
	if err != nil {
		return err
	}

	// Display items with ALT if configured
	return action.DisplayItemsWithAlt()
}
