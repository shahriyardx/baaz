import AppKit
import Foundation
import ServiceManagement

/// First-run setup, so dragging Baaz.app out of the DMG is the whole install.
///
/// The bundle carries the `baaz` CLI, which is also the daemon and Chrome's
/// native-messaging host. None of this needs root: the only step that does is
/// the machine-wide "ask where to save" policy, which `install-chrome` prints
/// a sudo hint for and which is optional.
public enum Setup {
    private static let versionKey = "baaz.setupCompletedForVersion"

    /// The CLI inside this bundle.
    static var bundledCLI: URL? {
        Bundle.main.url(forResource: "baaz", withExtension: nil)
    }

    static var installedCLI: URL {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent(".local/bin/baaz")
    }

    private static var appVersion: String {
        Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "0"
    }

    /// Runs setup when this version has not been set up yet. Cheap and
    /// idempotent, so re-running after an upgrade refreshes the CLI, the
    /// native-messaging manifest and the unpacked extension.
    public static func runIfNeeded(onFinished: @escaping (Bool) -> Void) {
        let done = UserDefaults.standard.string(forKey: versionKey) == appVersion
        DispatchQueue.global(qos: .utility).async {
            let installed = install()
            if installed {
                UserDefaults.standard.set(appVersion, forKey: versionKey)
            }
            DispatchQueue.main.async { onFinished(installed && !done) }
        }
    }

    /// Copies the CLI into ~/.local/bin, wires up Chrome, and registers the
    /// login item. Returns whether the CLI is in place afterwards.
    @discardableResult
    static func install() -> Bool {
        guard let src = bundledCLI else { return false }
        let dst = installedCLI
        let fm = FileManager.default
        do {
            try fm.createDirectory(at: dst.deletingLastPathComponent(),
                                   withIntermediateDirectories: true)
            // Replace rather than overwrite: macOS caches a malware verdict
            // per inode, so a binary once blocked at this path stays blocked
            // even after new bytes are written into the same file.
            if fm.fileExists(atPath: dst.path) {
                try fm.removeItem(at: dst)
            }
            try fm.copyItem(at: src, to: dst)
            try fm.setAttributes([.posixPermissions: 0o755], ofItemAtPath: dst.path)
        } catch {
            return false
        }

        // User-level Chrome wiring: native-messaging manifest, the no-save
        // prompt for this account, and the extension unpacked into Downloads.
        runCLI(dst, ["install-chrome"])
        registerLoginItem()
        return true
    }

    private static func runCLI(_ url: URL, _ args: [String]) {
        let p = Process()
        p.executableURL = url
        p.arguments = args
        p.standardOutput = FileHandle.nullDevice
        p.standardError = FileHandle.nullDevice
        try? p.run()
        p.waitUntilExit()
    }

    /// Registers the app to open at login.
    ///
    /// SMAppService is the supported route on macOS 13+: it registers this
    /// bundle wherever it happens to live, and the user can see and revoke it
    /// in System Settings → General → Login Items. Driving launchctl from
    /// inside the running app was unreliable — bootstrap reported success and
    /// left nothing registered.
    private static func registerLoginItem() {
        let service = SMAppService.mainApp
        guard service.status != .enabled else { return }
        do {
            try service.register()
        } catch {
            // Not fatal: the app still runs, it just will not start itself
            // after a reboot. Leave the old LaunchAgent alone if one exists.
            NSLog("baaz: could not register the login item: \(error.localizedDescription)")
        }
    }

    private static func run(_ path: String, _ args: [String]) {
        let p = Process()
        p.executableURL = URL(fileURLWithPath: path)
        p.arguments = args
        p.standardOutput = FileHandle.nullDevice
        p.standardError = FileHandle.nullDevice
        try? p.run()
        p.waitUntilExit()
    }
}
