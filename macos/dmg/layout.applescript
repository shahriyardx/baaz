-- Lays out the DMG window: size, icon view, background, icon positions.
--
-- Run against the volume that will actually ship. Finder stores the
-- background as an alias carrying the volume and file id it was made
-- against, so a layout copied from some other volume keeps the window size
-- and icon positions but silently drops the background.
--
-- The two positions must match the icon slots in render-background.swift.
on run argv
  set volName to item 1 of argv
  tell application "Finder"
    tell disk volName
      open
      set current view of container window to icon view
      set toolbar visible of container window to false
      set statusbar visible of container window to false
      set the bounds of container window to {200, 140, 840, 540}
      set opts to the icon view options of container window
      set arrangement of opts to not arranged
      set icon size of opts to 112
      set text size of opts to 12
      set label position of opts to bottom
      set background picture of opts to file ".background:background.tiff"
      set position of item "Baaz.app" of container window to {160, 205}
      set position of item "Applications" of container window to {480, 205}
      update without registering applications
      delay 2
      close
    end tell
  end tell
end run
