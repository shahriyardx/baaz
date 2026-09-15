import AppKit

/// Keeps Baaz out of the Dock unless it has a window open.
///
/// Baaz is a menu bar app that happens to have a window, not a windowed app
/// that happens to have a menu bar item. Left as a regular app it sat in the
/// Dock permanently, labelled "Running in Background", which is a tile you
/// cannot get rid of without quitting the thing doing the downloading.
///
/// The obvious fix — LSUIElement in the Info.plist — is too blunt: an
/// accessory app has no menu bar at all, so ⌘N, ⌘, and every other shortcut
/// the app defines would stop working whenever the window *is* open.
///
/// So the policy moves instead. Accessory while only the menu bar item is
/// showing; regular the moment a real window appears, which brings back the
/// Dock tile, the app switcher entry and the menus; accessory again once the
/// last one closes.
@MainActor
public enum DockPresence {
    private static var observers: [NSObjectProtocol] = []

    public static func start() {
        apply()
        for name in [NSWindow.didBecomeKeyNotification,
                     NSWindow.willCloseNotification,
                     NSWindow.didBecomeMainNotification] {
            observers.append(NotificationCenter.default.addObserver(
                forName: name, object: nil, queue: .main
            ) { _ in
                // willClose arrives before the window leaves the list, so the
                // count is taken on the next pass rather than this one.
                Task { @MainActor in apply() }
            })
        }
    }

    /// Windows that should put Baaz in the Dock: the main window and
    /// Settings. Not the menu bar panel, which is an NSPanel, and not the
    /// off-screen scaffolding AppKit and SwiftUI keep around.
    static func realWindowCount(_ windows: [NSWindow]) -> Int {
        windows.filter { w in
            guard w.isVisible, !(w is NSPanel) else { return false }
            return w.styleMask.contains(.titled) && w.canBecomeMain
        }.count
    }

    private static func apply() {
        let wanted: NSApplication.ActivationPolicy =
            realWindowCount(NSApp.windows) > 0 ? .regular : .accessory
        guard NSApp.activationPolicy() != wanted else { return }
        NSApp.setActivationPolicy(wanted)
        // Becoming regular mid-flight leaves the new window behind whatever
        // was in front; without this the window opens and is not on top.
        if wanted == .regular {
            NSApp.activate(ignoringOtherApps: true)
        }
    }
}
