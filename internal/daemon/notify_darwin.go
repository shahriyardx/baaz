package daemon

// notify does nothing on macOS. The app posts the notifications instead.
//
// The only way for a plain process to raise a banner here is to shell out to
// osascript with an AppleScript `display notification`, and macOS credits
// that to the script host rather than to the app that asked for it —
// osascript carries no bundle identity of its own. So the permission prompt
// read "Script Editor wants to send you notifications", which from a
// download manager is alarming enough that refusing is the sensible
// response; after which there were no notifications and no hint why. The
// README used to carry a line telling people to go and allow Script Editor.
//
// The menu bar app watches the same snapshot stream it already draws from,
// works out what changed, and posts through UNUserNotificationCenter under
// Baaz's own name and icon. That means no banners when the app is not
// running — on macOS the app is how Baaz is used, and a quiet daemon is a
// better answer than a prompt for an app the user never installed.
func notify(title, body string) {}
