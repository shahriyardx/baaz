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
                actions
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
        VStack(alignment: .leading, spacing: 10) {
            // Icon beside the name rather than centred above it: a 56pt
            // block of empty space used to push the one thing you came here
            // to read below the fold.
            HStack(alignment: .top, spacing: 10) {
                FileIcon(url: job.fileURL, fallback: job.name)
                    .frame(width: 38, height: 38)

                Text(job.name)
                    .font(.title3.weight(.semibold))
                    .lineLimit(3)
                    .textSelection(.enabled)
                    .fixedSize(horizontal: false, vertical: true)
            }

            if job.state != "done", let f = job.fraction {
                HStack(alignment: .firstTextBaseline) {
                    Text(statusText)
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(job.isFailed ? AnyShapeStyle(Color.red)
                                                      : AnyShapeStyle(.tint))
                    Spacer()
                    Text("\(Int(f * 100))%")
                        .font(.subheadline.monospacedDigit())
                        .foregroundStyle(.secondary)
                }
                ProgressView(value: f)
            }
        }
    }

    /// One action leads and the rest follow it. Pause and Cancel used to be
    /// the same size and weight, so nothing said which one you probably want.
    @ViewBuilder
    private var actions: some View {
        HStack(spacing: 8) {
            if job.state == "done" {
                Button { open() } label: {
                    Label("Open", systemImage: "arrow.up.forward.app")
                        .frame(maxWidth: .infinity)
                }
                .buttonStyle(.borderedProminent)

                Button { model.reveal(job) } label: {
                    Label("Finder", systemImage: "folder")
                }
            } else {
                if job.isControllable {
                    Button {
                        model.act(job.isActive ? "pause" : "resume", job.id)
                    } label: {
                        Label(job.isActive ? "Pause" : "Resume",
                              systemImage: job.isActive ? "pause.fill" : "play.fill")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                }
                Button { model.act("cancel", job.id) } label: {
                    Label("Cancel", systemImage: "xmark")
                }
            }
        }
    }

    /// Grouped under headings rather than run together. Eleven rows of the
    /// same weight is a table, and a table is something you read only when
    /// you already know what you are looking for.
    private var facts: some View {
        VStack(alignment: .leading, spacing: 16) {
            section("Transfer") {
                if job.state == "done" { fact("Status", statusText) }
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
            }

            section("File") {
                fact("Kind", job.isMedia ? "Video or audio page" : "Direct download")
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
    }

    private func section<C: View>(_ title: String, @ViewBuilder _ rows: () -> C) -> some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(title)
                .font(.subheadline.weight(.semibold))
                .foregroundStyle(.secondary)
            rows()
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

    private func fact(_ label: String, _ value: String) -> some View {
        HStack(alignment: .firstTextBaseline) {
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(width: 84, alignment: .leading)
            Text(value)
                .font(.callout)
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
                .frame(width: 84, alignment: .leading)
            Button(action: action) {
                Text(value)
                    .font(.callout)
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
