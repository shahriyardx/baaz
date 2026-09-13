import AppKit
import Combine
import Foundation

/// Owns the connection to the daemon.
///
/// A long-lived `baaz watch` streams one JSON snapshot per line; launching it
/// also auto-starts the daemon, so the UI never polls — it renders whatever
/// the last snapshot said. If the process dies (daemon killed, binary gone) a
/// timer restarts it after a pause, so a missing binary cannot spin the CPU.
@MainActor
public final class DownloadsModel: ObservableObject {
    public init() {}

    @Published private(set) var snapshot = Snapshot()
    @Published private(set) var daemonUp = false
    @Published var showingSettings = false

    /// Job IDs whose segment breakdown is open. Held here rather than in the
    /// row so it survives the view being rebuilt on every snapshot.
    @Published private(set) var expanded: Set<String> = []

    /// Set when no `baaz` binary exists at all — a different problem from a
    /// daemon that has not come up yet, and worth saying out loud.
    let cliMissing = BaazCLI.url == nil

    private var process: Process?
    private var buffer = Data()
    private var restartTask: Task<Void, Never>?
    private let retryDelay: Duration = .seconds(3)

    var jobs: [Job] { snapshot.jobs }
    var recent: [Job] { snapshot.recent }
    public var activeCount: Int { snapshot.active }
    var settings: BaazSettings { snapshot.settings }

    /// The menu bar title: count, combined speed, and overall percent.
    public var barText: String {
        guard activeCount > 0 else { return "" }
        var t = "\(activeCount)  \(human(snapshot.totalSpeed))/s"
        if let pct = snapshot.overallPercent { t += "  \(pct)%" }
        return t
    }

    var statusLine: String {
        if cliMissing { return "baaz command not found" }
        if !daemonUp { return "daemon starting…" }
        if !settings.intercept { return "intercept off — Chrome downloads normally" }
        if activeCount > 0 { return "\(activeCount) active · \(human(snapshot.totalSpeed))/s" }
        if !jobs.isEmpty { return "\(jobs.count) waiting" }
        return "idle"
    }

    public func start() {
        guard process == nil, let p = BaazCLI.watchProcess() else { return }
        let pipe = Pipe()
        p.standardOutput = pipe
        p.standardError = FileHandle.nullDevice
        buffer = Data()

        pipe.fileHandleForReading.readabilityHandler = { [weak self] handle in
            let chunk = handle.availableData
            guard !chunk.isEmpty else { return }
            Task { @MainActor in self?.ingest(chunk) }
        }
        p.terminationHandler = { [weak self] _ in
            Task { @MainActor in self?.handleExit() }
        }

        do {
            try p.run()
            process = p
        } catch {
            scheduleRestart()
        }
    }

    public func stop() {
        restartTask?.cancel()
        restartTask = nil
        if let p = process {
            (p.standardOutput as? Pipe)?.fileHandleForReading.readabilityHandler = nil
            p.terminationHandler = nil
            p.terminate()
        }
        process = nil
    }

    /// Internal rather than private so tests can drive the view from a known
    /// snapshot without a live daemon.
    func ingest(_ chunk: Data) {
        buffer.append(chunk)
        // Snapshots are newline-delimited; a read can split one mid-line or
        // carry several, so only whole lines are decoded.
        while let nl = buffer.firstIndex(of: UInt8(ascii: "\n")) {
            let line = buffer[buffer.startIndex..<nl]
            buffer.removeSubrange(buffer.startIndex...nl)
            guard !line.isEmpty,
                  let snap = try? JSONDecoder().decode(Snapshot.self, from: Data(line)),
                  snap.type == "snapshot"
            else { continue } // partial or garbage line: keep the last snapshot
            snapshot = snap
            daemonUp = true
        }
    }

    private func handleExit() {
        process = nil
        daemonUp = false
        // Keep the recent list: it is still true, and blanking it makes the
        // panel flicker every time the daemon restarts.
        var s = Snapshot()
        s.recent = snapshot.recent
        s.settings = snapshot.settings
        snapshot = s
        scheduleRestart()
    }

    private func scheduleRestart() {
        restartTask?.cancel()
        restartTask = Task { [weak self, retryDelay] in
            try? await Task.sleep(for: retryDelay)
            guard !Task.isCancelled else { return }
            self?.start()
        }
    }

    // MARK: - Actions

    func act(_ verb: String, _ id: String) {
        BaazCLI.run(id.isEmpty ? [verb] : [verb, id])
    }

    func setConfig(_ key: String, _ value: String) {
        BaazCLI.run(["config", key, value])
    }

    func toggleExpanded(_ id: String) {
        if expanded.contains(id) {
            expanded.remove(id)
        } else {
            expanded.insert(id)
        }
    }

    func isExpanded(_ id: String) -> Bool { expanded.contains(id) }

    func toggleIntercept() {
        setConfig("intercept", settings.intercept ? "false" : "true")
    }

    func toggleCategorize() {
        setConfig("categorize", settings.categorize ? "false" : "true")
    }

    /// The download root as an absolute path. The daemon stores it with a
    /// leading `~` when the user set it that way.
    var resolvedDownloadDir: String {
        let dir = settings.downloadDir
        guard dir == "~" || dir.hasPrefix("~/") else { return dir }
        return FileManager.default.homeDirectoryForCurrentUser.path + String(dir.dropFirst())
    }

    /// Opens the configured download root in Finder.
    func openDownloadDir() {
        let dir = resolvedDownloadDir
        guard !dir.isEmpty else { return }
        NSWorkspace.shared.open(URL(fileURLWithPath: dir))
    }

    /// Asks for a new download folder and hands it to the daemon.
    ///
    /// An accessory app has no windows and never becomes active on its own,
    /// so the panel would open behind everything without activating first.
    /// The path goes to the CLI as its own argv entry, so spaces in it are
    /// not a quoting problem.
    func chooseDownloadDir() {
        let panel = NSOpenPanel()
        panel.canChooseDirectories = true
        panel.canChooseFiles = false
        panel.allowsMultipleSelection = false
        panel.canCreateDirectories = true
        panel.prompt = "Choose"
        panel.message = "Where should baaz save downloads?"
        let current = resolvedDownloadDir
        if !current.isEmpty {
            panel.directoryURL = URL(fileURLWithPath: current)
        }
        NSApp.activate(ignoringOtherApps: true)
        guard panel.runModal() == .OK, let url = panel.url else { return }
        setConfig("dir", url.path)
    }

    /// Reveals the finished file in Finder, falling back to opening the
    /// folder when the file is already gone.
    func reveal(_ job: Job) {
        guard !job.dir.isEmpty else { return }
        let dir = URL(fileURLWithPath: job.dir)
        let file = dir.appendingPathComponent(job.name)
        if FileManager.default.fileExists(atPath: file.path) {
            NSWorkspace.shared.activateFileViewerSelecting([file])
        } else {
            NSWorkspace.shared.open(dir)
        }
    }
}
