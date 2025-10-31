package action

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/npc"
	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/utils"
	"github.com/lxn/win"
)

func OpenTPIfLeader() error {
	ctx := context.Get()
	ctx.SetLastAction("OpenTPIfLeader")

	isLeader := ctx.CharacterCfg.Companion.Leader

	if isLeader {
		return step.OpenPortal()
	}

	return nil
}

func IsMonsterSealElite(monster data.Monster) bool {
	return monster.Type == data.MonsterTypeSuperUnique && (monster.Name == npc.OblivionKnight || monster.Name == npc.VenomLord || monster.Name == npc.StormCaster)
}

func PostRun(isLastRun bool) error {
	ctx := context.Get()
	ctx.SetLastAction("PostRun")

	// Allow some time for items drop to the ground, otherwise we might miss some
	utils.Sleep(200)
	ClearAreaAroundPlayer(5, data.MonsterAnyFilter())
	ItemPickup(-1)

	// Don't return town on last run
	if !isLastRun {
		return ReturnTown()
	}

	return nil
}
func AreaCorrection() error {
	ctx := context.Get()
	currentArea := ctx.Data.PlayerUnit.Area
	expectedArea := ctx.CurrentGame.AreaCorrection.ExpectedArea

	// Skip correction if in town, if we're in the expected area, or if expected area is not set
	if currentArea.IsTown() || currentArea == expectedArea || expectedArea == 0 {
		return nil
	}

	if ctx.CurrentGame.AreaCorrection.Enabled && ctx.CurrentGame.AreaCorrection.ExpectedArea != ctx.Data.AreaData.Area {
		ctx.Logger.Info("Accidentally went to adjacent area, returning to expected area",
			"current", ctx.Data.AreaData.Area.Area().Name,
			"expected", ctx.CurrentGame.AreaCorrection.ExpectedArea.Area().Name)
		return MoveToArea(ctx.CurrentGame.AreaCorrection.ExpectedArea)
	}

	return nil
}
func HidePortraits() error {
	ctx := context.Get()
	ctx.SetLastAction("HidePortraits")

	// Hide portraits if configured
	if ctx.CharacterCfg.HidePortraits && ctx.Data.OpenMenus.PortraitsShown {
		ctx.HID.PressKey(ctx.Data.KeyBindings.ShowPortraits.Key1[0])
	}
	return nil
}
func ClearMessages() error {
	ctx := context.Get()
	ctx.SetLastAction("ClearMessages")
	ctx.HID.PressKey(ctx.Data.KeyBindings.ClearMessages.Key1[0])
	return nil
}

func PressKeyForDuration(key string, durationSeconds int) error {
	ctx := context.Get()
	ctx.SetLastAction("PressKeyForDuration")

	// Get the virtual key code from keyboard.go mappings
	vkCode := ctx.HID.GetASCIICode(key)
	if vkCode == 0 {
		// Try to get from common key mappings
		switch key {
		case "alt":
			vkCode = byte(win.VK_MENU)
		case "ctrl":
			vkCode = byte(win.VK_CONTROL)
		case "shift":
			vkCode = byte(win.VK_LSHIFT)
		default:
			return fmt.Errorf("unsupported key: %s", key)
		}
	}

	// Use the HID methods directly
	hwnd := ctx.HID.GetHWND()
	lParamDown := ctx.HID.CalculateLParam(vkCode, true)
	lParamUp := ctx.HID.CalculateLParam(vkCode, false)
	
	// Press the key down
	win.PostMessage(hwnd, win.WM_KEYDOWN, uintptr(vkCode), lParamDown)
	
	// Wait for the specified duration
	time.Sleep(time.Duration(durationSeconds) * time.Second)
	
	// Release the key
	win.PostMessage(hwnd, win.WM_KEYUP, uintptr(vkCode), lParamUp)
	
	ctx.Logger.Debug("Pressed key for duration", slog.String("key", key), slog.Int("duration", durationSeconds))
	
	return nil
}

// DisplayItemsWithAlt displays items on the ground using ALT key for configured duration.
// This function checks the AltDisplayTime configuration and only displays if > 0.
func DisplayItemsWithAlt() error {
	ctx := context.Get()
	
	if ctx.CharacterCfg.Game.AltDisplayTime > 0 {
		ctx.Logger.Info("🎯 RUN COMPLETED - Displaying items with ALT (you can pause bot now if needed)", slog.Int("duration", ctx.CharacterCfg.Game.AltDisplayTime))
		return PressKeyForDuration("alt", ctx.CharacterCfg.Game.AltDisplayTime)
	}
	
	ctx.Logger.Info("🎯 RUN COMPLETED - No ALT display configured")
	return nil
}
