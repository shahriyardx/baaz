import AppKit
import Combine
import Foundation

/// Owns the connection to the daemon.
///
/// A long-lived `baaz watch` streams one JSON snapshot per line; launching it
/// also auto-starts the daemon, so the UI never polls — it renders whatever
/// the last snapshot said. If the process dies (daemon killed, binary gone) a
/// timer restarts it after a pause, so a missing binary cannot spin the CPU.
/// Holds SwiftUI's openWindow action so code outside the view hierarchy can
/// open a window too — the Dock-reopen handler lives on the app delegate,
/// which has no SwiftUI environment of its own.
@MainActor
public enum WindowOpener {
    public static var action: ((String) -> Void)?

    public static func open(_ id: String) { action?(id) }
}

/// Which downloads the main window is showing.
public enum JobFilter: String, CaseIterable, Identifiable {
    case all, active, paused, done, failed
    public var id: String { rawValue }

    public var title: String {
        switch self {
        case .all: return "All"
        case .active: return "Downloading"
        case .paused: return "Paused"
        case .done: return "Completed"
        case .failed: return "Failed"
        }
    }

    var symbol: String {
        switch self {
        case .all: return "tray.full"
        case .active: return "arrow.down.circle"
        case .paused: return "pause.circle"
        case .done: return "checkmark.circle"
        case .failed: return "exclamationmark.triangle"
        }
    }
}

@MainActor
public final class DownloadsModel: ObservableObject {
    public init() {}

    @Published private(set) var snapshot = Snapshot()
    /// What the previous snapshot said, for working out what to announce.
    /// nil until the first one arrives.
    private var lastStates: [String: JobState]?
    @Published private(set) var daemonUp = false
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

    /// What the background one-time setup is doing, if anything.
    var setupNote: String { snapshot.setup }

    var statusLine: String {
        // A live snapshot outranks the static binary check: data is flowing,
        // so whatever launched the stream clearly works.
        if !daemonUp {
            if cliMissing { return "not set up yet" }
            return "starting…"
        }
        if !settings.intercept { return "off — Chrome downloads on its own" }
        if activeCount > 0 { return "\(activeCount) active · \(human(snapshot.totalSpeed))/s" }
        // Only when nothing is downloading: a real download outranks a
        // background errand the user did not ask for.
        if !setupNote.isEmpty { return setupNote }
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
            // Swift 6: a Task may only capture constants; `self` from a weak
            // capture list is a var, so rebind it first.
            guard let self else { return }
            Task { @MainActor in self.ingest(chunk) }
        }
        p.terminationHandler = { [weak self] _ in
            guard let self else { return }
            Task { @MainActor in self.handleExit() }
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
    /// Posts a banner for anything that changed since the last snapshot.
    ///
    /// The very first snapshot after launch only seeds the comparison. It
    /// arrives carrying everything the daemon already knows, and announcing
    /// all of it would greet the user with a banner per download from
    /// yesterday.
    private func announce(_ snap: Snapshot) {
        let now = snap.jobStates
        defer { lastStates = now }
        guard let before = lastStates else { return }
        for event in downloadEvents(previous: before, current: now) {
            Notifier.shared.post(title: event.title, body: event.name)
        }
    }

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
            announce(snap)
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

    /// Queues a URL. The daemon does the detecting: it probes for Range
    /// support, works out the filename and size, and routes media pages to
    /// yt-dlp — so the UI only has to hand over the link.
    func add(url: String, filename: String = "", format: String = "") {
        let url = url.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !url.isEmpty else { return }
        var args = ["add"]
        if !filename.isEmpty { args += ["--out", filename] }
        if !format.isEmpty { args += ["--format", format] }
        args.append(url)
        BaazCLI.run(args)
    }

    /// Everything, newest first, with the live jobs above finished ones.
    var allJobs: [Job] { jobs + recent }

    func jobs(matching filter: JobFilter) -> [Job] {
        switch filter {
        case .all: return allJobs
        case .active: return jobs.filter { $0.isActive || $0.state == "queued" }
        case .paused: return jobs.filter { $0.state == "paused" }
        case .done: return recent
        case .failed: return jobs.filter(\.isFailed)
        }
    }

    func count(_ filter: JobFilter) -> Int { jobs(matching: filter).count }

    public func pauseAll() {
        for j in jobs where j.isActive { act("pause", j.id) }
    }

    public func resumeAll() {
        for j in jobs where j.state == "paused" || j.isFailed { act("resume", j.id) }
    }

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

    /// Brings the main window forward, reopening it if it was closed.
    ///
    /// Goes through SwiftUI's own openWindow. Poking at selectors did not
    /// work: a closed Window scene is not in NSApp.windows to be raised, and
    /// raising "the first window that can become main" could pick Settings.
    public func openMainWindow() {
        NSApp.activate(ignoringOtherApps: true)
        WindowOpener.open("main")
    }

    /// Opens the configured download root in Finder.
    public func openDownloadDir() {
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
        panel.message = "Where should Baaz save downloads?"
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
