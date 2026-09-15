import Foundation
import UserNotifications

/// Posts Baaz's desktop notifications.
///
/// These used to come from the daemon, which shelled out to `osascript` with
/// an AppleScript `display notification`. macOS credits such a notification to
/// the *script host* rather than to Baaz, because osascript carries no bundle
/// identity of its own — so the permission prompt read "Script Editor wants to
/// send you notifications", which from a download manager looks alarming
/// enough to refuse. Refusing then killed the notifications with no hint why.
///
/// Posting from the app instead gets Baaz's own name, its own icon, and its
/// own entry in System Settings. That only became possible once there was a
/// real bundled app to post from.
public final class Notifier: NSObject, UNUserNotificationCenterDelegate {
    public static let shared = Notifier()

    private var authorized = false
    private var asked = false

    /// True when a notification centre is actually reachable.
    ///
    /// It is not under xctest or in a loose binary, and
    /// UNUserNotificationCenter.current() traps rather than returning nil
    /// there — so this has to be decided before touching it. A bundle
    /// identifier alone is not enough: the test runner has one. Being an
    /// .app is the thing that matters.
    private let available: Bool = Bundle.main.bundleURL.pathExtension == "app"

    override private init() {
        super.init()
        guard available else { return }
        UNUserNotificationCenter.current().delegate = self
    }

    /// Asks once, at launch, so the prompt arrives while the user is looking
    /// at the app they just opened rather than mid-download.
    public func requestPermission() {
        guard available, !asked else { return }
        asked = true
        UNUserNotificationCenter.current()
            .requestAuthorization(options: [.alert, .sound]) { [weak self] granted, _ in
                self?.authorized = granted
            }
    }

    public func post(title: String, body: String) {
        guard available else { return }
        let content = UNMutableNotificationContent()
        content.title = title
        content.body = body
        let req = UNNotificationRequest(
            identifier: UUID().uuidString, content: content, trigger: nil)
        UNUserNotificationCenter.current().add(req)
    }

    /// Show the banner even when Baaz is the frontmost app. Downloads finish
    /// while you are looking at the window, and silence there reads as a
    /// missing notification rather than a deliberate one.
    public func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        completionHandler([.banner, .sound])
    }
}
