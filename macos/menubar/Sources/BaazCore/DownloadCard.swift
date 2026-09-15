import AppKit
import SwiftUI

/// One download in the main window: name, progress, live figures, and the
/// controls for it. Richer than the menu bar row, which has to stay compact.
struct DownloadCard: View {
    @EnvironmentObject var model: DownloadsModel
    let job: Job
    let selected: Bool

    @State private var hovering = false
    @State private var confirmingDelete = false

    private var expanded: Bool { model.isExpanded(job.id) }

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack(alignment: .top, spacing: 10) {
                Image(systemName: icon)
                    .font(.system(size: 22))
                    .foregroundStyle(tint)
                    .frame(width: 28)

                VStack(alignment: .leading, spacing: 3) {
                    Text(job.name)
                        .font(.body.weight(.medium))
                        .lineLimit(1)
                        .truncationMode(.middle)
                    Text(job.caption)
                        .font(.caption)
                        .foregroundStyle(job.isFailed ? AnyShapeStyle(Color.red)
                                                      : AnyShapeStyle(.secondary))
                        .lineLimit(2)
                }

                Spacer(minLength: 8)
                controls
            }

            if job.state != "done" {
                if let f = job.fraction {
                    ProgressView(value: f).progressViewStyle(.linear).tint(tint)
                } else if job.isActive {
                    ProgressView().progressViewStyle(.linear)
                }
            }

            if !job.segments.isEmpty {
                Button {
                    model.toggleExpanded(job.id)
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: expanded ? "chevron.down" : "chevron.right")
                            .font(.system(size: 9, weight: .semibold))
                        Text(expanded ? "Hide parts" : job.partsLabel)
                            .font(.caption)
                        Spacer()
                    }
                    .foregroundStyle(.secondary)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)

                if expanded { SegmentBars(job: job) }
            }
        }
        .padding(12)
        .baazGlass(
            in: RoundedRectangle(cornerRadius: 12),
            tinted: selected,
            fallback: selected ? Color.accentColor.opacity(0.12)
                : hovering ? Color.primary.opacity(0.05)
                : Color.primary.opacity(0.03)
        )
        .overlay(
            RoundedRectangle(cornerRadius: 12)
                .stroke(selected ? Color.accentColor.opacity(0.5) : .clear, lineWidth: 1)
        )
        .onHover { hovering = $0 }
        .contextMenu { contextMenu }
        .alert("Delete \(job.name)?", isPresented: $confirmingDelete) {
            Button("Delete File", role: .destructive) { model.act("delete", job.id) }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("This removes the entry and the downloaded file. It cannot be undone.")
        }
    }

    @ViewBuilder
    private var controls: some View {
        HStack(spacing: 2) {
            if job.state == "done" {
                IconButton(systemName: "folder", help: "Show in Finder") {
                    model.reveal(job)
                }
            } else if job.isControllable {
                IconButton(systemName: job.isActive ? "pause.fill" : "play.fill",
                           help: job.isActive ? "Pause" : "Resume") {
                    model.act(job.isActive ? "pause" : "resume", job.id)
                }
            }
            IconButton(systemName: job.state == "done" ? "trash" : "xmark",
                       help: job.state == "done" ? "Delete file" : "Cancel") {
                if job.state == "done" { confirmingDelete = true }
                else { model.act("cancel", job.id) }
            }
        }
    }

    @ViewBuilder
    private var contextMenu: some View {
        if job.state == "done" {
            Button("Show in Finder") { model.reveal(job) }
            Button("Delete File…") { confirmingDelete = true }
        } else {
            if job.isControllable {
                Button(job.isActive ? "Pause" : "Resume") {
                    model.act(job.isActive ? "pause" : "resume", job.id)
                }
            }
            Button("Cancel") { model.act("cancel", job.id) }
        }
        Divider()
        Button("Copy Name") {
            NSPasteboard.general.clearContents()
            NSPasteboard.general.setString(job.name, forType: .string)
        }
    }

    private var icon: String {
        if job.isFailed { return "exclamationmark.triangle.fill" }
        switch job.state {
        case "done": return "checkmark.circle.fill"
        case "paused": return "pause.circle.fill"
        case "queued": return "clock.fill"
        default: return "arrow.down.circle.fill"
        }
    }

    private var tint: Color {
        if job.isFailed { return .red }
        switch job.state {
        case "done": return .green
        case "paused": return .orange
        case "queued": return .secondary
        default: return .accentColor
        }
    }
}
