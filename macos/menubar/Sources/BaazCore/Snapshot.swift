import Foundation

// Mirrors internal/ipc/protocol.go — the daemon is the single source of
// truth for this schema. Every field decodes defensively: Go marshals nil
// slices as null and omits empty strings, so nothing here may be required.

struct BaazSettings: Codable, Equatable {
    var intercept = true
    var segments = 0
    var maxActive = 0
    var minSizeMB = 0
    var downloadDir = ""
    var categorize = true

    init() {}

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        intercept = try c.decodeIfPresent(Bool.self, forKey: .intercept) ?? true
        segments = try c.decodeIfPresent(Int.self, forKey: .segments) ?? 0
        maxActive = try c.decodeIfPresent(Int.self, forKey: .maxActive) ?? 0
        minSizeMB = try c.decodeIfPresent(Int.self, forKey: .minSizeMB) ?? 0
        downloadDir = try c.decodeIfPresent(String.self, forKey: .downloadDir) ?? ""
        categorize = try c.decodeIfPresent(Bool.self, forKey: .categorize) ?? true
    }
}

/// One byte range of a download. The daemon sends these only while a job is
/// in flight, which is what lets the panel show the split actually working.
struct JobSegment: Codable, Equatable {
    var done: Int64 = 0
    var total: Int64 = -1

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        done = try c.decodeIfPresent(Int64.self, forKey: .done) ?? 0
        total = try c.decodeIfPresent(Int64.self, forKey: .total) ?? -1
    }

    var fraction: Double {
        guard total > 0 else { return 0 }
        return min(1, Double(done) / Double(total))
    }

    var isComplete: Bool { total > 0 && done >= total }
}

struct Job: Codable, Equatable, Identifiable {
    var id = ""
    var name = ""
    var state = ""
    var total: Int64 = 0
    var done: Int64 = 0
    var speed: Int64 = 0
    var eta: Int64 = -1
    var dir = ""
    var error = ""
    var segments: [JobSegment] = []

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        state = try c.decodeIfPresent(String.self, forKey: .state) ?? ""
        total = try c.decodeIfPresent(Int64.self, forKey: .total) ?? 0
        done = try c.decodeIfPresent(Int64.self, forKey: .done) ?? 0
        speed = try c.decodeIfPresent(Int64.self, forKey: .speed) ?? 0
        eta = try c.decodeIfPresent(Int64.self, forKey: .eta) ?? -1
        dir = try c.decodeIfPresent(String.self, forKey: .dir) ?? ""
        error = try c.decodeIfPresent(String.self, forKey: .error) ?? ""
        segments = try c.decodeIfPresent([JobSegment].self, forKey: .segments) ?? []
    }

    /// True when the server supported Range and the file was actually split.
    /// A single segment means one stream — worth saying so rather than
    /// drawing a "parts" display with one part in it.
    var isSplit: Bool { segments.count > 1 }

    var segmentsComplete: Int { segments.filter(\.isComplete).count }

    /// Fraction complete, or nil when the server never advertised a size —
    /// the UI shows a sweeping bar rather than a fake 100% fill.
    var fraction: Double? {
        guard total > 0 else { return nil }
        return min(1, Double(done) / Double(total))
    }

    var isActive: Bool { state == "active" }
    var isFailed: Bool { state == "failed" }
    var isControllable: Bool {
        ["active", "paused", "failed", "queued"].contains(state)
    }

    var caption: String {
        switch state {
        case "active":
            guard total > 0 else {
                return "\(human(done)) · \(human(speed))/s · size unknown"
            }
            var s = "\(human(done)) / \(human(total)) · \(human(speed))/s"
            if eta >= 0 {
                s += eta > 90 ? " · \(Int(ceil(Double(eta) / 60)))m left" : " · \(eta)s left"
            }
            return s
        case "failed":
            return error.isEmpty ? "failed" : "failed: \(error)"
        case "done":
            return human(total)
        default:
            return state
        }
    }
}

struct Snapshot: Codable, Equatable {
    var type = ""
    var active = 0
    var totalSpeed: Int64 = 0
    var jobs: [Job] = []
    var recent: [Job] = []
    var settings = BaazSettings()

    init() {}

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        type = try c.decodeIfPresent(String.self, forKey: .type) ?? ""
        active = try c.decodeIfPresent(Int.self, forKey: .active) ?? 0
        totalSpeed = try c.decodeIfPresent(Int64.self, forKey: .totalSpeed) ?? 0
        jobs = try c.decodeIfPresent([Job].self, forKey: .jobs) ?? []
        recent = try c.decodeIfPresent([Job].self, forKey: .recent) ?? []
        settings = try c.decodeIfPresent(BaazSettings.self, forKey: .settings) ?? BaazSettings()
    }

    /// Combined progress over sized active jobs, or nil when none has a size.
    var overallPercent: Int? {
        var done: Int64 = 0, total: Int64 = 0
        for j in jobs where j.isActive && j.total > 0 {
            done += j.done
            total += j.total
        }
        guard total > 0 else { return nil }
        return Int(done * 100 / total)
    }
}

func human(_ n: Int64) -> String {
    let v = Double(n)
    if n >= 1 << 30 { return String(format: "%.1fGB", v / Double(1 << 30)) }
    if n >= 1 << 20 { return String(format: "%.1fMB", v / Double(1 << 20)) }
    if n >= 1 << 10 { return String(format: "%.0fKB", v / Double(1 << 10)) }
    return "\(n)B"
}
