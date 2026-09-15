import Foundation

/// Something worth telling the user about, worked out by comparing one
/// snapshot to the last.
public struct DownloadEvent: Equatable {
    public enum Kind: Equatable { case started, finished, failed }
    public let kind: Kind
    public let name: String

    public var title: String {
        switch kind {
        case .started: return "Download started"
        case .finished: return "Download finished"
        case .failed: return "Download failed"
        }
    }
}

/// Works out what changed between two snapshots.
///
/// The daemon used to post notifications itself. It no longer does on macOS,
/// because it has no way to post them as Baaz — so the app watches the same
/// snapshot stream it already draws from and decides here. Keeping that as a
/// plain function over two dictionaries means it can be tested without a
/// notification centre, an app bundle, or a running daemon.
public func downloadEvents(previous: [String: JobState], current: [String: JobState])
    -> [DownloadEvent]
{
    var out: [DownloadEvent] = []
    // Sorted so the order is the same every run; a dictionary's is not.
    for id in current.keys.sorted() {
        guard let now = current[id] else { continue }
        let before = previous[id]
        switch now.state {
        case "done" where before?.state != "done":
            out.append(DownloadEvent(kind: .finished, name: now.name))
        case "failed" where before?.state != "failed":
            out.append(DownloadEvent(kind: .failed, name: now.name))
        case "active", "queued":
            // Only when the job is new. A job flipping between queued and
            // active as slots free up is not a fresh download.
            if before == nil {
                out.append(DownloadEvent(kind: .started, name: now.name))
            }
        default:
            break
        }
    }
    return out
}

/// The bit of a job that decides whether to say anything.
public struct JobState: Equatable {
    public let state: String
    public let name: String
    public init(state: String, name: String) {
        self.state = state
        self.name = name
    }
}

extension Snapshot {
    /// Every job the snapshot knows about, in flight or finished.
    var jobStates: [String: JobState] {
        var m: [String: JobState] = [:]
        for j in jobs + recent {
            m[j.id] = JobState(state: j.state, name: j.name.isEmpty ? j.url : j.name)
        }
        return m
    }
}
