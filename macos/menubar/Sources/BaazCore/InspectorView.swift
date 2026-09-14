import AppKit
import SwiftUI
import UniformTypeIdentifiers

/// Details for the selected download, beside the list.
struct InspectorView: View {
    @EnvironmentObject var model: DownloadsModel
    let job: Job

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 14) {
                header
                Divider()
                actions
                Divider()
                facts
                if !job.segments.isEmpty {
                    Divider()
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Parts").font(.subheadline.weight(.semibold))
                        SegmentBars(job: job)
                    }
                }
                if job.isFailed && !job.error.isEmpty {
                    Divider()
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Error").font(.subheadline.weight(.semibold))
                        Text(job.error)
                            .font(.caption)
                            .foregroundStyle(.red)
                            .textSelection(.enabled)
                            .fixedSize(horizontal: false, vertical: true)
                    }
                }
            }
            .padding(14)
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 8) {
            FileIcon(url: job.fileURL, fallback: job.name)
                .frame(width: 56, height: 56)
                .frame(maxWidth: .infinity, alignment: .center)

            Text(job.name)
                .font(.headline)
                .textSelection(.enabled)
                .fixedSize(horizontal: false, vertical: true)

            if job.state != "done", let f = job.fraction {
                ProgressView(value: f)
                Text("\(Int(f * 100))%")
                    .font(.caption.monospacedDigit())
                    .foregroundStyle(.secondary)
            }
        }
    }

    @ViewBuilder
    private var actions: some View {
        HStack(spacing: 8) {
            if job.state == "done" {
                Button { open() } label: { Label("Open", systemImage: "arrow.up.forward.app") }
                Button { model.reveal(job) } label: { Label("Finder", systemImage: "folder") }
            } else {
                if job.isControllable {
                    Button {
                        model.act(job.isActive ? "pause" : "resume", job.id)
                    } label: {
                        Label(job.isActive ? "Pause" : "Resume",
                              systemImage: job.isActive ? "pause.fill" : "play.fill")
                    }
                }
                Button { model.act("cancel", job.id) } label: {
                    Label("Cancel", systemImage: "xmark")
                }
            }
        }
        .controlSize(.small)
    }

    private var facts: some View {
        VStack(alignment: .leading, spacing: 8) {
            fact("Status", statusText)
            if job.total > 0 {
                fact("Size", human(job.total))
                if job.state != "done" { fact("Downloaded", human(job.done)) }
            } else if job.done > 0 {
                fact("Downloaded", "\(human(job.done)) (size unknown)")
            }
            if job.isActive {
                fact("Speed", "\(human(job.speed))/s")
                if job.eta >= 0 {
                    fact("Time left", job.eta > 90 ? "\(Int(ceil(Double(job.eta) / 60)))m" : "\(job.eta)s")
                }
            }
            fact("Parts", partsText)
            fact("Type", job.isMedia ? "Media (yt-dlp)" : "Direct download")
            if !job.dir.isEmpty {
                factButton("Saved to", job.dir) { model.reveal(job) }
            }
            if !job.url.isEmpty {
                factButton("Source", job.url) {
                    NSPasteboard.general.clearContents()
                    NSPasteboard.general.setString(job.url, forType: .string)
                }
            }
            if let t = format(job.createdAt) { fact("Added", t) }
            if let t = format(job.completedAt) { fact("Finished", t) }
        }
    }

    private var statusText: String {
        if job.isFailed { return "Failed" }
        switch job.state {
        case "done": return "Completed"
        case "active": return "Downloading"
        case "paused": return "Paused"
        case "queued": return "Waiting"
        default: return job.state.capitalized
        }
    }

    /// Says why a download is not split when it is not, since that is the
    /// usual reason one is no faster than the browser's.
    private var partsText: String {
        if job.segments.isEmpty { return "—" }
        if job.segments.count == 1 { return job.singlePartDetail }
        return "\(job.segments.count) at once"
    }

    private func fact(_ label: String, _ value: String) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: 78, alignment: .leading)
            Text(value)
                .font(.caption)
                .textSelection(.enabled)
                .fixedSize(horizontal: false, vertical: true)
            Spacer(minLength: 0)
        }
    }

    private func factButton(_ label: String, _ value: String,
                            action: @escaping () -> Void) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: 78, alignment: .leading)
            Button(action: action) {
                Text(value)
                    .font(.caption)
                    .lineLimit(2)
                    .truncationMode(.middle)
                    .multilineTextAlignment(.leading)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .foregroundStyle(.tint)
            Spacer(minLength: 0)
        }
    }

    private func open() {
        guard let url = job.fileURL,
              FileManager.default.fileExists(atPath: url.path) else {
            model.reveal(job)
            return
        }
        NSWorkspace.shared.open(url)
    }

    /// RFC3339 from the daemon, shown in the viewer's own locale.
    private func format(_ stamp: String) -> String? {
        guard !stamp.isEmpty else { return nil }
        let iso = ISO8601DateFormatter()
        iso.formatOptions = [.withInternetDateTime]
        guard let d = iso.date(from: stamp) else { return nil }
        let out = DateFormatter()
        out.dateStyle = .medium
        out.timeStyle = .short
        return out.string(from: d)
    }
}

/// The real Finder icon for the file, falling back to one picked from the
/// extension before the file exists.
struct FileIcon: NSViewRepresentable {
    let url: URL?
    let fallback: String

    func makeNSView(context: Context) -> NSImageView {
        let v = NSImageView()
        v.imageScaling = .scaleProportionallyUpOrDown
        return v
    }

    func updateNSView(_ v: NSImageView, context: Context) {
        if let url, FileManager.default.fileExists(atPath: url.path) {
            v.image = NSWorkspace.shared.icon(forFile: url.path)
        } else {
            let ext = (fallback as NSString).pathExtension
            let type = ext.isEmpty ? nil : UTType(filenameExtension: ext)
            v.image = NSWorkspace.shared.icon(for: type ?? .data)
        }
    }
}
