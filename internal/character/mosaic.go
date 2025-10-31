package character

import (
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/npc"
	"github.com/hectorgimenez/d2go/pkg/data/skill"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
	"github.com/hectorgimenez/d2go/pkg/data/state"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/game"
)

type MosaicSin struct {
	BaseCharacter
	lastChargeTime time.Time // Track when we last built charges
}

func (s MosaicSin) ShouldIgnoreMonster(m data.Monster) bool {
	return false
}

func (s MosaicSin) CheckKeyBindings() []skill.ID {
	ctx := context.Get()
	requireKeybindings := []skill.ID{skill.TigerStrike, skill.CobraStrike, skill.PhoenixStrike, skill.ClawsOfThunder, skill.BladesOfIce, skill.TomeOfTownPortal}
	
	// Add Dragon Flight if configured
	if ctx.CharacterCfg.Character.MosaicSin.UseDragonFlight {
		requireKeybindings = append(requireKeybindings, skill.DragonFlight)
	}
	
	// Add Dragon Talon if configured
	if ctx.CharacterCfg.Character.MosaicSin.UseDragonTalon {
		requireKeybindings = append(requireKeybindings, skill.DragonTalon)
	}
	
	// Add Mind Blast if configured
	if ctx.CharacterCfg.Character.MosaicSin.UseMindBlast {
		requireKeybindings = append(requireKeybindings, skill.MindBlast)
	}
	
	missingKeybindings := []skill.ID{}

	for _, cskill := range requireKeybindings {
		if _, found := s.Data.KeyBindings.KeyBindingForSkill(cskill); !found {
			missingKeybindings = append(missingKeybindings, cskill)
		}
	}

	if len(missingKeybindings) > 0 {
		s.Logger.Debug("There are missing required key bindings.", slog.Any("Bindings", missingKeybindings))
	}

	return missingKeybindings
}

func (s MosaicSin) KillMonsterSequence(
	monsterSelector func(d game.Data) (data.UnitID, bool),
	skipOnImmunities []stat.Resist,
) error {
	ctx := context.Get()
	
	// Disable item pickup during combat to prevent charge loss and combat interruption
	ctx.DisableItemPickup()
	defer ctx.EnableItemPickup()
	
	lastRefresh := time.Now()

	for {
		context.Get().PauseIfNotPriority()

		// Limit refresh rate to 10 times per second to avoid excessive CPU usage
		if time.Since(lastRefresh) > time.Millisecond*100 {
			ctx.RefreshGameData()
			lastRefresh = time.Now()
		}

		// Get the charges for each skill we're using
		tigerCharges, foundTiger := ctx.Data.PlayerUnit.Stats.FindStat(stat.ProgressiveDamage, 0)
		cobraCharges, foundCobra := ctx.Data.PlayerUnit.Stats.FindStat(stat.ProgressiveSteal, 0)
		phoenixCharges, foundPhoenix := ctx.Data.PlayerUnit.Stats.FindStat(stat.ProgressiveOther, 0)
		clawsCharges, foundClaws := ctx.Data.PlayerUnit.Stats.FindStat(stat.ProgressiveLightning, 0)
		bladesCharges, foundBlades := ctx.Data.PlayerUnit.Stats.FindStat(stat.ProgressiveCold, 0)
		firstCharges, foundFirst := ctx.Data.PlayerUnit.Stats.FindStat(stat.ProgressiveFire, 0)

		id, found := monsterSelector(*s.Data)
		if !found {
			return nil
		}

		monster, found := s.Data.Monsters.FindByID(id)
		if !found {
			s.Logger.Info("Monster not found", slog.String("monster", fmt.Sprintf("%v", monster)))
			return nil
		}

		if !s.preBattleChecks(id, skipOnImmunities) {
			return nil
		}

		dist := ctx.PathFinder.DistanceFromMe(monster.Position)
		
		// Use Dragon Flight to teleport to monsters out of melee range if configured
		// Extended range to 30 for boss fights (Baal can spawn far away)
		if ctx.CharacterCfg.Character.MosaicSin.UseDragonFlight && dist > 5 && dist <= 30 {
			s.Logger.Debug("Using Dragon Flight to teleport to monster", slog.Int("distance", dist))
			// Dragon Flight is a finisher that teleports, so use it to get in range
			if err := step.SecondaryAttack(skill.DragonFlight, id, 1, step.Distance(1, 30)); err == nil {
				// Successfully teleported, reset charges will rebuild
				continue
			}
		}

		// Initial move to monster if we're too far
		// Mosaic Assassin is melee, so we need to be close (within 5 units for reliable attacks)
		if dist > 5 {
			// Try to move to monster position
			if err := step.MoveTo(monster.Position); err != nil {
				s.Logger.Debug("Failed to move to monster position", slog.String("error", err.Error()))
				
				if dist <= 30 {
					// Monster is within Dragon Flight range, enable force attack to bypass LoS
					s.Logger.Debug("Path blocked but monster in range, forcing attack", slog.Int("distance", dist))
					ctx.ForceAttack = true // Enable force attack to bypass LoS check
					defer func() { ctx.ForceAttack = false }() // Reset after this iteration
					// Don't return nil, let Dragon Flight handle it in finisher selection
				} else {
					// Monster is truly too far (>30), skip it
					s.Logger.Debug("Monster too far and path blocked, skipping", slog.Int("distance", dist))
					return nil
				}
			}
		}

		// Enable aggressive mode: force attacks even when merc/shadow blocks
		if ctx.CharacterCfg.Character.MosaicSin.AggressiveMode {
			ctx.ForceAttack = true
			defer func() { ctx.ForceAttack = false }()
		}

		if !s.MobAlive(id, *s.Data) {
			return nil
		}

		// Tiger Strike - 3 charges
		if ctx.CharacterCfg.Character.MosaicSin.UseTigerStrike {
			if !s.Data.PlayerUnit.States.HasState(state.Tigerstrike) || (foundTiger && tigerCharges.Value < 3) {
				step.SecondaryAttack(skill.TigerStrike, id, 1)
				continue
			}
		}

		if !s.MobAlive(id, *s.Data) {
			return nil
		}

		// Cobra Strike - 3 charges
		if ctx.CharacterCfg.Character.MosaicSin.UseCobraStrike {
			if !s.Data.PlayerUnit.States.HasState(state.Cobrastrike) || (foundCobra && cobraCharges.Value < 3) {
				step.SecondaryAttack(skill.CobraStrike, id, 1)
				continue
			}
		}

		if !s.MobAlive(id, *s.Data) {
			return nil
		}

		// Phoenix Strike - configurable charges (1=Ice Bolt, 2=Lightning, 3=Meteor)
		phoenixTargetCharges := ctx.CharacterCfg.Character.MosaicSin.PhoenixChargeCount
		if phoenixTargetCharges == 0 {
			phoenixTargetCharges = 2 // Default to 2 charges (Lightning)
		}
		if !s.Data.PlayerUnit.States.HasState(state.Phoenixstrike) || (foundPhoenix && phoenixCharges.Value < phoenixTargetCharges) {
			step.SecondaryAttack(skill.PhoenixStrike, id, 1)
			continue
		}

		if !s.MobAlive(id, *s.Data) {
			return nil
		}

		// Claws of Thunder - 3 charges
		if ctx.CharacterCfg.Character.MosaicSin.UseClawsOfThunder {
			if !s.Data.PlayerUnit.States.HasState(state.Clawsofthunder) || (foundClaws && clawsCharges.Value < 3) {
				step.SecondaryAttack(skill.ClawsOfThunder, id, 1)
				continue
			}
		}

		if !s.MobAlive(id, *s.Data) {
			return nil
		}

		// Blades of Ice - 3 charges
		if ctx.CharacterCfg.Character.MosaicSin.UseBladesOfIce {
			if !s.Data.PlayerUnit.States.HasState(state.Bladesofice) || (foundBlades && bladesCharges.Value < 3) {
				step.SecondaryAttack(skill.BladesOfIce, id, 1)
				continue
			}
		}

		// First of Fire - 3 charges
		if ctx.CharacterCfg.Character.MosaicSin.UseFistsOfFire {
			if !s.Data.PlayerUnit.States.HasState(state.Fistsoffire) || (foundFirst && firstCharges.Value < 3) {
				step.SecondaryAttack(skill.FistsOfFire, id, 1)
				continue
			}
		}

		if !s.MobAlive(id, *s.Data) {
			return nil
		}

		// Check if we need to refresh charges using Mind Blast
		chargeRefreshThreshold := ctx.CharacterCfg.Character.MosaicSin.ChargeRefreshTime
		if chargeRefreshThreshold == 0 {
			chargeRefreshThreshold = 12 // Default to 12 seconds
		}
		
		// If we have charges and they're about to expire, use Mind Blast to refresh
		hasCharges := (foundTiger && tigerCharges.Value > 0) || 
			(foundCobra && cobraCharges.Value > 0) || 
			(foundPhoenix && phoenixCharges.Value > 0) ||
			(foundClaws && clawsCharges.Value > 0) ||
			(foundBlades && bladesCharges.Value > 0) ||
			(foundFirst && firstCharges.Value > 0)
		
		if ctx.CharacterCfg.Character.MosaicSin.UseMindBlast && hasCharges && !s.lastChargeTime.IsZero() {
			timeSinceCharge := time.Since(s.lastChargeTime)
			if timeSinceCharge.Seconds() >= float64(chargeRefreshThreshold) {
				s.Logger.Debug("Refreshing charges with Mind Blast", slog.Float64("timeSinceCharge", timeSinceCharge.Seconds()))
				// Use Mind Blast on the target to stun and refresh charge timer
				step.SecondaryAttack(skill.MindBlast, id, 1, step.Distance(1, 15))
				s.lastChargeTime = time.Now() // Reset the timer
				continue
			}
		}
		
		// Update last charge time when we have full charges
		if hasCharges && s.lastChargeTime.IsZero() {
			s.lastChargeTime = time.Now()
		}

		// Choose finisher based on distance and configuration
		dist = ctx.PathFinder.DistanceFromMe(monster.Position)
		
		// Use Dragon Talon when in melee range (more efficient, no cooldown)
		if ctx.CharacterCfg.Character.MosaicSin.UseDragonTalon && dist <= 5 {
			s.Logger.Debug("Using Dragon Talon finisher (melee range)", slog.Int("distance", dist))
			step.SecondaryAttack(skill.DragonTalon, id, 1, step.Distance(1, 5))
			s.lastChargeTime = time.Time{} // Reset charge timer after using finisher
		} else if ctx.CharacterCfg.Character.MosaicSin.UseDragonFlight && dist > 5 && dist <= 30 {
			// Use Dragon Flight for teleporting when target is far (extended range for bosses)
			s.Logger.Debug("Using Dragon Flight finisher (teleport)", slog.Int("distance", dist))
			step.SecondaryAttack(skill.DragonFlight, id, 1, step.Distance(1, 30))
			s.lastChargeTime = time.Time{} // Reset charge timer after using finisher
		} else {
			// Fallback to primary attack if no finisher configured or out of range
			opts := step.Distance(1, 5)
			step.PrimaryAttack(id, 1, false, opts)
			s.lastChargeTime = time.Time{} // Reset charge timer
		}
	}
}

func (s MosaicSin) MobAlive(mob data.UnitID, d game.Data) bool {
	monster, found := s.Data.Monsters.FindByID(mob)
	return found && monster.Stats[stat.Life] > 0
}

func (s MosaicSin) BuffSkills() []skill.ID {
	skillsList := make([]skill.ID, 0)

	// Prefer Fade over Burst of Speed for resistance and damage reduction
	if _, found := s.Data.KeyBindings.KeyBindingForSkill(skill.Fade); found {
		skillsList = append(skillsList, skill.Fade)
	} else if _, found := s.Data.KeyBindings.KeyBindingForSkill(skill.BurstOfSpeed); found {
		// If we don't have Fade bound, use Burst of Speed
		skillsList = append(skillsList, skill.BurstOfSpeed)
	}

	return skillsList
}

func (s MosaicSin) PreCTABuffSkills() []skill.ID {
	if _, found := s.Data.KeyBindings.KeyBindingForSkill(skill.ShadowMaster); found {
		return []skill.ID{skill.ShadowMaster}
	} else if _, found := s.Data.KeyBindings.KeyBindingForSkill(skill.ShadowWarrior); found {
		return []skill.ID{skill.ShadowWarrior}
	}
	return []skill.ID{}
}

func (s MosaicSin) killMonster(npc npc.ID, t data.MonsterType) error {
	return s.KillMonsterSequence(func(d game.Data) (data.UnitID, bool) {
		m, found := d.Monsters.FindOne(npc, t)
		if !found {
			return 0, false
		}
		return m.UnitID, true
	}, nil)
}

func (s MosaicSin) KillCountess() error {
	return s.killMonster(npc.DarkStalker, data.MonsterTypeSuperUnique)
}

func (s MosaicSin) KillAndariel() error {
	return s.killMonster(npc.Andariel, data.MonsterTypeUnique)
}

func (s MosaicSin) KillSummoner() error {
	return s.killMonster(npc.Summoner, data.MonsterTypeUnique)
}

func (s MosaicSin) KillDuriel() error {
	return s.killMonster(npc.Duriel, data.MonsterTypeUnique)
}

func (s MosaicSin) KillCouncil() error {
	return s.KillMonsterSequence(func(d game.Data) (data.UnitID, bool) {
		var councilMembers []data.Monster
		for _, m := range d.Monsters {
			if m.Name == npc.CouncilMember || m.Name == npc.CouncilMember2 || m.Name == npc.CouncilMember3 {
				councilMembers = append(councilMembers, m)
			}
		}

		sort.Slice(councilMembers, func(i, j int) bool {
			distanceI := s.PathFinder.DistanceFromMe(councilMembers[i].Position)
			distanceJ := s.PathFinder.DistanceFromMe(councilMembers[j].Position)
			return distanceI < distanceJ
		})

		if len(councilMembers) > 0 {
			return councilMembers[0].UnitID, true
		}

		return 0, false
	}, nil)
}

func (s MosaicSin) KillMephisto() error {
	return s.killMonster(npc.Mephisto, data.MonsterTypeUnique)
}

func (s MosaicSin) KillIzual() error {
	return s.killMonster(npc.Izual, data.MonsterTypeUnique)
}

func (s MosaicSin) KillDiablo() error {
	timeout := time.Second * 20
	startTime := time.Now()
	diabloFound := false

	for {
		if time.Since(startTime) > timeout && !diabloFound {
			s.Logger.Error("Diablo was not found, timeout reached")
			return nil
		}

		diablo, found := s.Data.Monsters.FindOne(npc.Diablo, data.MonsterTypeUnique)
		if !found || diablo.Stats[stat.Life] <= 0 {
			if diabloFound {
				return nil
			}
			time.Sleep(200 * time.Millisecond)
			continue
		}

		diabloFound = true
		s.Logger.Info("Diablo detected, attacking")
		return s.killMonster(npc.Diablo, data.MonsterTypeUnique)
	}
}

func (s MosaicSin) KillPindle() error {
	return s.killMonster(npc.DefiledWarrior, data.MonsterTypeSuperUnique)
}

func (s MosaicSin) KillNihlathak() error {
	return s.killMonster(npc.Nihlathak, data.MonsterTypeSuperUnique)
}

// MaintainChargesWithBattery converts a minion with Mind Blast and maintains charges by attacking it
// This is used before boss fights to ensure full charges when boss spawns
func (s MosaicSin) MaintainChargesWithBattery(duration time.Duration) error {
	ctx := context.Get()
	
	if !ctx.CharacterCfg.Character.MosaicSin.UseChargeBattery || !ctx.CharacterCfg.Character.MosaicSin.UseMindBlast {
		s.Logger.Debug("Charge battery strategy disabled, skipping")
		return nil
	}
	
	s.Logger.Info("🔋 Starting charge battery strategy - converting minion to maintain charges", slog.Int("duration_seconds", int(duration.Seconds())))
	
	// Find a weak minion to convert (avoid elites/champions)
	var targetMinion data.Monster
	found := false
	
	for _, m := range ctx.Data.Monsters.Enemies(ctx.Data.MonsterFilterAnyReachable()) {
		// Skip bosses, champions, elites - we want a normal minion
		if m.IsElite() {
			continue
		}
		// Prefer weak minions that are close
		if ctx.PathFinder.DistanceFromMe(m.Position) <= 20 {
			targetMinion = m
			found = true
			break
		}
	}
	
	if !found {
		s.Logger.Warn("No suitable minion found for charge battery strategy")
		return nil
	}
	
	s.Logger.Info("Converting minion with Mind Blast", slog.Int("monsterID", int(targetMinion.Name)))
	
	// Convert the minion with Mind Blast
	if err := step.SecondaryAttack(skill.MindBlast, targetMinion.UnitID, 1, step.Distance(1, 20)); err != nil {
		s.Logger.Warn("Failed to convert minion with Mind Blast", slog.String("error", err.Error()))
		return nil
	}
	
	time.Sleep(time.Millisecond * 500)
	
	// Maintain charges using the converted minion
	startTime := time.Now()
	lastChargeRefresh := time.Now()
	
	for time.Since(startTime) < duration {
		ctx.PauseIfNotPriority()
		ctx.RefreshGameData()
		
		// Check if we need to refresh charges (every 12 seconds)
		refreshTime := ctx.CharacterCfg.Character.MosaicSin.ChargeRefreshTime
		if refreshTime == 0 {
			refreshTime = 12
		}
		
		if time.Since(lastChargeRefresh) >= time.Duration(refreshTime)*time.Second {
			s.Logger.Debug("Refreshing charges using converted minion")
			
			// Build charges on the converted minion
			convertedMinion, stillExists := ctx.Data.Monsters.FindByID(targetMinion.UnitID)
			if !stillExists {
				s.Logger.Warn("Converted minion died or disappeared, ending charge battery")
				break
			}
			
			dist := ctx.PathFinder.DistanceFromMe(convertedMinion.Position)
			
			// Build one charge of each type configured
			if ctx.CharacterCfg.Character.MosaicSin.UseTigerStrike {
				step.SecondaryAttack(skill.TigerStrike, convertedMinion.UnitID, 1, step.Distance(1, 20))
			}
			if ctx.CharacterCfg.Character.MosaicSin.UseCobraStrike {
				step.SecondaryAttack(skill.CobraStrike, convertedMinion.UnitID, 1, step.Distance(1, 20))
			}
			step.SecondaryAttack(skill.PhoenixStrike, convertedMinion.UnitID, 1, step.Distance(1, 20))
			
			// Use finisher to refresh charges (Dragon Talon or Dragon Flight)
			if dist <= 5 && ctx.CharacterCfg.Character.MosaicSin.UseDragonTalon {
				s.Logger.Debug("Using Dragon Talon on converted minion to refresh charges")
				step.SecondaryAttack(skill.DragonTalon, convertedMinion.UnitID, 1, step.Distance(1, 5))
			} else if ctx.CharacterCfg.Character.MosaicSin.UseDragonFlight {
				s.Logger.Debug("Using Dragon Flight on converted minion to refresh charges")
				step.SecondaryAttack(skill.DragonFlight, convertedMinion.UnitID, 1, step.Distance(1, 20))
			}
			
			lastChargeRefresh = time.Now()
			s.lastChargeTime = time.Now()
		}
		
		time.Sleep(time.Millisecond * 500)
	}
	
	s.Logger.Info("🔋 Charge battery strategy complete - charges refreshed and ready for boss!")
	return nil
}

func (s MosaicSin) KillBaal() error {
	s.Logger.Info("Starting persistent Baal kill sequence...")
	ctx := context.Get()
	timeout := time.Second * 600
	startTime := time.Now()

	for {
		ctx.PauseIfNotPriority()
		baal, found := s.Data.Monsters.FindOne(npc.BaalCrab, data.MonsterTypeUnique)

		if !found {
			if time.Since(startTime) > timeout {
				s.Logger.Error("Baal was not found, timeout reached after 10 minutes.")
				return errors.New("baal not found within the time limit")
			}
			s.Logger.Debug("Baal not visible, waiting...")
			time.Sleep(time.Millisecond * 500)
			ctx.RefreshGameData()
			continue
		}

		if baal.Stats[stat.Life] <= 0 {
			s.Logger.Info("Baal is dead!")
			return nil
		}

		dist := ctx.PathFinder.DistanceFromMe(baal.Position)
		s.Logger.Debug("Baal found, engaging", slog.Int("distance", dist), slog.Int("life", baal.Stats[stat.Life]))

		// If Baal is too far (>30), walk towards him first
		if dist > 30 {
			s.Logger.Info("Baal teleported far away, pursuing...", slog.Int("distance", dist))
			if err := step.MoveTo(baal.Position, step.WithDistanceToFinish(25)); err != nil {
				s.Logger.Debug("Failed to move closer to Baal, will retry", slog.String("error", err.Error()))
			}
			time.Sleep(time.Millisecond * 300)
			continue
		}

		// Attack Baal using normal KillMonsterSequence (builds charges + uses finishers)
		err := s.KillMonsterSequence(func(d game.Data) (data.UnitID, bool) {
			m, found := d.Monsters.FindOne(npc.BaalCrab, data.MonsterTypeUnique)
			if !found {
				return 0, false
			}
			if m.Stats[stat.Life] <= 0 {
				return 0, false
			}
			return m.UnitID, true
		}, nil)

		if err != nil {
			s.Logger.Debug("KillMonsterSequence returned, checking Baal status", slog.String("error", err.Error()))
		}

		time.Sleep(time.Millisecond * 250)
	}
}
