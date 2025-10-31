# Mosaic Assassin Improvements

This fork includes major enhancements for Mosaic Assassin gameplay, focusing on charge management, boss fights, and quality-of-life improvements.

## 🎯 Major Features

### 1. **Persistent Baal Kill System**
- Bot never gives up on Baal, even when he teleports
- Automatically pursues Baal at any distance
- Walks to get within Dragon Flight range (30 units)
- Handles multiple teleports in a single fight
- 10-minute timeout (600 seconds)

**Before**: Bot would skip Baal if he spawned >20 distance away
**After**: Bot pursues relentlessly until Baal is dead

### 2. **Mind Blast Charge Battery Strategy** 🔋
- Converts a minion with Mind Blast after clearing areas
- Maintains full charges by attacking converted ally
- Refreshes charges every 12 seconds (configurable)
- Perfect for low-density areas (The Pit, Ancient Tunnels, etc.)
- Configurable duration: 20-50 seconds (default 30s)

**How It Works**:
1. Clear area completely
2. Find a weak minion (avoids elites)
3. Mind Blast → converts to ally
4. Build charges on ally
5. Use finisher to refresh (ally survives!)
6. Repeat for configured duration
7. Return to town with FULL charges

**Enabled For**:
- ✅ The Pit
- 🔮 Can be added to other runs easily

### 3. **Two-Pass Diablo Mode** (FIXED)
- **First Pass**: Clears all seals/bosses with NO item pickup
- **Second Pass**: Backtracks to collect all loot
- Preserves charges throughout first pass
- No more accidental trips to town

**Bug Fixed**: `ClearAreaAroundPosition()` now respects pickup state

### 4. **Extended Dragon Flight Range**
- Increased from 20 to 30 units
- Handles distant boss spawns (Baal, Diablo)
- Better fallback for blocked paths
- More reliable teleport-finisher usage

### 5. **Charge Management System**
- Disables item pickup during combat
- Prevents charge loss from mid-fight pickups
- Automatic Mind Blast refresh at 12 seconds
- Smart finisher selection (Dragon Talon vs Dragon Flight)

### 6. **Run Completion Notifications**
- Clear message: "🎯 RUN COMPLETED - Displaying items with ALT"
- Visible even when no items dropped
- Know exactly when to pause the bot
- Added to all major runs

## 🎮 Configuration

### Character Settings - Mosaic Assassin

**Charge Skills**:
- ☑ Use Tiger Strike
- ☑ Use Cobra Strike
- ☑ Use Phoenix Strike (select charge count: 1/2/3)

**Finishers**:
- ☑ Use Dragon Flight (Teleport Finisher)
- ☑ Use Dragon Talon (Melee Finisher)

**Charge Management**:
- ☑ Use Mind Blast (Charge Refresh)
- ☑ 🔋 Use Charge Battery Strategy ← **NEW!**
- Charge Refresh Time: 10-13 seconds (default 12)
- Charge Battery Duration: 20-50 seconds (default 30) ← **NEW!**

**Special Modes**:
- ☑ Two-Pass Diablo Run (Preserves Charges) ← **FIXED!**
- ☑ Aggressive Mode (Don't wait for merc/shadow)

### Run Settings

**Diablo**:
- Two-Pass mode now properly disables pickup during first pass

**The Pit**:
- Charge Battery activates automatically after clearing Level 2

## 📊 Technical Details

### Files Modified
- `internal/character/mosaic.go` - Core Mosaic logic, charge battery
- `internal/action/clear_area.go` - Respects pickup state (Two-Pass fix)
- `internal/action/tools.go` - Run completion notifications
- `internal/run/diablo.go` - Two-Pass mode
- `internal/run/baal.go` - Persistent Baal pursuit
- `internal/run/pit.go` - Charge battery integration
- `internal/config/config.go` - New config options
- `internal/server/http_server.go` - Form handling
- `internal/server/templates/character_settings.gohtml` - UI controls

### New Methods
- `MosaicSin.MaintainChargesWithBattery(duration)` - Charge battery core
- `MosaicSin.KillBaal()` - Persistent pursuit system

### Config Options Added
```yaml
mosaic_sin:
  useChargeBattery: true          # Enable charge battery
  chargeBatteryDuration: 30       # Duration in seconds
```

## 🧪 Testing Recommendations

### Test Two-Pass Diablo
1. Enable "Two-Pass Diablo Run"
2. Run Chaos Sanctuary
3. Verify: No item pickups during seal clearing
4. Verify: Charges preserved throughout
5. Verify: Backtracking collects all items

### Test Charge Battery (The Pit)
1. Enable "Use Mind Blast" + "Use Charge Battery Strategy"
2. Set duration to 30 seconds
3. Run The Pit
4. Watch logs for: "🔋 Starting charge battery strategy"
5. Verify: Bot converts minion and maintains charges
6. Verify: Return to town with full charges

### Test Baal Fight
1. Run Throne of Destruction
2. Observe Baal spawn distance in logs
3. Verify: Bot pursues even at 35-45 distance
4. Verify: No more "skipping" messages
5. Verify: Baal always dies

## ❌ Known Limitations

**Charge Battery Cannot Work For**:
- Chaos Sanctuary (seal pop kills all minions)
- Throne of Destruction (must clear all waves)
- Any area with forced spawn mechanics

**Why**: Game mechanics automatically kill all remaining monsters when triggering spawns.

**Workaround**: Use Two-Pass mode for Diablo to preserve charges naturally.

## 🚀 Future Enhancements

**Potential Additions**:
- Charge battery for Ancient Tunnels
- Charge battery for Mausoleum
- Charge battery for Arachnid Lair
- Pre-fight charge building for Ubers
- Charge state HUD indicator

## 📜 Credits

**Original Koolo Bot**: [hectorgimenez/koolo](https://github.com/hectorgimenez/koolo)

**Mosaic Improvements**: Fork by kwader2k with AI assistance (Cascade/Claude)

## 📝 Changelog

### v1.0.0 - Mosaic Assassin Improvements (2025-01-30)

**Added**:
- Mind Blast charge battery strategy
- Persistent Baal kill system (pursues teleports)
- Extended Dragon Flight range (20 → 30)
- Run completion notifications
- Charge battery support for The Pit

**Fixed**:
- Two-Pass Diablo mode pickup bug
- Baal fight abandonment at distance >30
- Charge loss during item pickup
- ClearAreaAroundPosition pickup state handling

**Changed**:
- Dragon Flight max range: 20 → 30
- DisplayItemsWithAlt() added to all major runs
- Improved finisher selection logic

---

## 🤝 Contributing

Feel free to:
- Report issues
- Suggest improvements
- Submit pull requests
- Fork and modify further

This is an open-source improvement to the Koolo bot focused on making Mosaic Assassin gameplay smoother and more efficient!

## 📄 License

Same as original Koolo project. See LICENSE file.
