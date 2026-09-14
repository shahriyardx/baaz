import Foundation

/// Locates and runs the `baaz` binary.
///
/// A menu bar app launched by launchd inherits a near-empty PATH, so the
/// binary is found by looking in the places the installers use before asking
/// a login shell as a last resort.
enum BaazCLI {
    static let url: URL? = locate()

    private static func locate() -> URL? {
        let fm = FileManager.default
        var candidates: [URL] = []
        if let home = fm.homeDirectoryForCurrentUser as URL? {
            candidates.append(home.appendingPathComponent(".local/bin/baaz"))
        }
        candidates += [
            URL(fileURLWithPath: "/opt/homebrew/bin/baaz"),
            URL(fileURLWithPath: "/usr/local/bin/baaz"),
            URL(fileURLWithPath: "/usr/bin/baaz"),
        ]
        // The copy inside this bundle is the last-resort fallback: it always
        // exists, so the app still works before (or instead of) the copy into
        // ~/.local/bin that first-run setup makes.
        if let bundled = Setup.bundledCLI {
            candidates.append(bundled)
        }
        for c in candidates where fm.isExecutableFile(atPath: c.path) {
            return c
        }
        // Last resort: a login shell knows the user's own PATH additions.
        if let path = loginShellLookup(), fm.isExecutableFile(atPath: path) {
            return URL(fileURLWithPath: path)
        }
        return nil
    }

    private static func loginShellLookup() -> String? {
        let shell = ProcessInfo.processInfo.environment["SHELL"] ?? "/bin/zsh"
        let p = Process()
        p.executableURL = URL(fileURLWithPath: shell)
        p.arguments = ["-lc", "command -v baaz"]
        let pipe = Pipe()
        p.standardOutput = pipe
        p.standardError = FileHandle.nullDevice
        do { try p.run() } catch { return nil }
        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        p.waitUntilExit()
        guard p.terminationStatus == 0 else { return nil }
        return String(data: data, encoding: .utf8)?
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .nilIfEmpty
    }

    /// Fire-and-forget CLI call. Control commands are idempotent from the
    /// UI's point of view — the next snapshot reports the real outcome, so
    /// there is nothing useful to do with a failure here.
    static func run(_ args: [String]) {
        guard let url else { return }
        let p = Process()
        p.executableURL = url
        p.arguments = args
        p.standardOutput = FileHandle.nullDevice
        p.standardError = FileHandle.nullDevice
        try? p.run()
    }

    /// A long-lived `baaz watch`, wired up but not started.
    static func watchProcess() -> Process? {
        guard let url else { return nil }
        let p = Process()
        p.executableURL = url
        p.arguments = ["watch"]
        return p
    }
}

extension String {
    var nilIfEmpty: String? { isEmpty ? nil : self }
}
