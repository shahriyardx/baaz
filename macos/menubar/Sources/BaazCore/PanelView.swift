import SwiftUI

/// Reports the measured height of the panel's list to its container.
struct ListHeightKey: PreferenceKey {
    // `let`, not `var`: a mutable static is shared global state, which Swift 6
    // rejects. The protocol only ever reads it.
    static let defaultValue: CGFloat = 0
    static func reduce(value: inout CGFloat, nextValue: () -> CGFloat) {
        value = max(value, nextValue())
    }
}

/// A borderless icon button with a hit area big enough to actually click.
///
/// `.buttonStyle(.borderless)` sizes the button to the glyph — a 16pt target
/// that SwiftUI does not hit-test reliably inside a MenuBarExtra window, so
/// clicks on the icon simply did nothing. The explicit frame and content
/// shape are what make the button respond; do not drop them.
struct IconButton: View {
    let systemName: String
    let help: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: systemName)
                .font(.system(size: 12))
                .frame(width: 22, height: 22)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .help(help)
    }
}

/// Same problem, same fix, for the small text buttons.
struct TextButton: View {
    let title: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(.caption)
                .padding(.vertical, 2)
                .padding(.horizontal, 4)
                .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .foregroundStyle(.secondary)
    }
}

/// The panel that drops out of the menu bar icon: status header, live
/// downloads, recent downloads, and a settings page behind the gear.
public struct PanelView: View {
    @EnvironmentObject var model: DownloadsModel
    @Environment(\.openWindow) private var openWindow

    private let width: CGFloat = 340
    private let maxListHeight: CGFloat = 420

    // Measured height of the list, so the ScrollView is given a definite
    // frame rather than sizing itself.
    //
    // A ScrollView has no intrinsic height, so what the menu bar panel offers
    // it is ambiguous — and the panel was seen rendering as a header and
    // footer with an empty strip between them while a download was running.
    // That could not be reproduced in an NSHostingView, so this is a
    // mitigation rather than a confirmed fix: deriving the height from the
    // content removes the ambiguity, and clamps it so the list can never
    // collapse to nothing nor push the footer off screen.
    @State private var listHeight: CGFloat = 0

    public init() {}

    public var body: some View {
        VStack(spacing: 0) {
            header
            Divider()
            ScrollView {
                VStack(alignment: .leading, spacing: 10) {
                    body_
                }
                .padding(12)
                .frame(width: width, alignment: .leading)
                .background(
                    GeometryReader { geo in
                        Color.clear.preference(key: ListHeightKey.self,
                                               value: geo.size.height)
                    }
                )
            }
            .frame(height: min(max(listHeight, 44), maxListHeight))
            .onPreferenceChange(ListHeightKey.self) { listHeight = $0 }
            Divider()
            footer
        }
        .frame(width: width)
        .onAppear {
            // The menu bar panel outlives every window, so this is where the
            // action is captured for anything that needs it later.
            WindowOpener.action = { openWindow(id: $0) }
        }
    }

    @ViewBuilder
    private var body_: some View {
        if model.jobs.isEmpty && model.recent.isEmpty {
            Text(model.cliMissing
                 ? "Install the baaz command first, then reopen this menu."
                 : "Nothing yet — downloads from Chrome land here.")
                .font(.callout)
                .foregroundStyle(.tertiary)
                .fixedSize(horizontal: false, vertical: true)
        }

        if !model.jobs.isEmpty {
            VStack(spacing: 2) {
                ForEach(model.jobs) { job in
                    LiveJobRow(job: job)
                }
            }
        }

        if !model.jobs.isEmpty && !model.recent.isEmpty {
            Divider()
        }

        if !model.recent.isEmpty {
            HStack {
                Text("Recent")
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(.secondary)
                Spacer()
                TextButton(title: "Clear all") { model.act("clear", "") }
            }
            VStack(spacing: 2) {
                ForEach(model.recent) { job in
                    RecentJobRow(job: job)
                }
            }
        }
    }

    private var header: some View {
        HStack(alignment: .center, spacing: 8) {
            VStack(alignment: .leading, spacing: 1) {
                Text("Downloads").font(.headline)
                Text(model.statusLine)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(1)
            }
            Spacer(minLength: 4)
            // Opens the Settings window rather than swapping this panel's
            // contents. Swapping changed the panel's height, and the menu bar
            // window grows to fit but never shrinks back — leaving the
            // downloads list floating in a settings-sized box.
            IconButton(systemName: "gearshape", help: "Settings") {
                NSApp.activate(ignoringOtherApps: true)
                openWindow(id: "settings")
            }

            Toggle("", isOn: Binding(
                get: { model.settings.intercept },
                set: { _ in model.toggleIntercept() }
            ))
            .toggleStyle(.switch)
            .labelsHidden()
            .controlSize(.small)
            .help("Take over Chrome downloads")
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
    }

    private var footer: some View {
        HStack {
            TextButton(title: "Open baaz") {
                NSApp.activate(ignoringOtherApps: true)
                openWindow(id: "main")
            }
            TextButton(title: "Settings…") {
                NSApp.activate(ignoringOtherApps: true)
                openWindow(id: "settings")
            }

            Spacer()
            TextButton(title: "Quit") { NSApplication.shared.terminate(nil) }
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 8)
    }
}

/// One in-flight download: name, progress, and pause/cancel.
struct LiveJobRow: View {
    @EnvironmentObject var model: DownloadsModel
    let job: Job
    @State private var hovering = false

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 6) {
                Text(job.name)
                    .font(.callout)
                    .lineLimit(1)
                    .truncationMode(.middle)
                Spacer(minLength: 4)
                if job.isControllable {
                    IconButton(systemName: job.isActive ? "pause.fill" : "play.fill",
                               help: job.isActive ? "Pause" : "Resume") {
                        model.act(job.isActive ? "pause" : "resume", job.id)
                    }
                }
                IconButton(systemName: "xmark", help: "Cancel") {
                    model.act("cancel", job.id)
                }
            }

            // No advertised size: an indeterminate bar, not a fake full one.
            if let f = job.fraction {
                ProgressView(value: f)
                    .progressViewStyle(.linear)
                    .tint(job.isFailed ? .red : .accentColor)
            } else if job.isActive {
                ProgressView()
                    .progressViewStyle(.linear)
            } else {
                ProgressView(value: 0)
                    .progressViewStyle(.linear)
                    .tint(job.isFailed ? .red : .accentColor)
            }

            // A Button, not .onTapGesture: tap gestures do not fire inside
            // the menu bar panel, so the row looked clickable and did nothing.
            Button {
                model.toggleExpanded(job.id)
            } label: {
                HStack(spacing: 4) {
                    Text(job.caption)
                        .font(.caption)
                        .foregroundStyle(job.isFailed ? AnyShapeStyle(Color.red) : AnyShapeStyle(.secondary))
                        .lineLimit(1)
                        .truncationMode(.tail)
                    Spacer(minLength: 0)
                    if !job.segments.isEmpty {
                        Image(systemName: expanded ? "chevron.down" : "chevron.right")
                            .font(.system(size: 8, weight: .semibold))
                            .foregroundStyle(.tertiary)
                    }
                }
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .disabled(job.segments.isEmpty)
            .help(job.segments.isEmpty ? "" : "Show the parts this file is split into")

            if expanded {
                SegmentBars(job: job)
            }
        }
        .padding(.horizontal, 6)
        .padding(.vertical, 6)
        .background(
            RoundedRectangle(cornerRadius: 6)
                .fill(hovering ? Color.primary.opacity(0.06) : .clear)
        )
        .onHover { hovering = $0 }
    }

    private var expanded: Bool { model.isExpanded(job.id) }
}

/// The file drawn as its actual byte ranges: one chunk per segment, each
/// filling independently. This is the download's real shape, not a decorative
/// meter — eight chunks advancing at their own rates is what eight parallel
/// connections look like.
struct SegmentBars: View {
    let job: Job

    private let spacing: CGFloat = 1

    var body: some View {
        VStack(alignment: .leading, spacing: 3) {
            GeometryReader { geo in
                let n = max(job.segments.count, 1)
                let w = max(0, (geo.size.width - spacing * CGFloat(n - 1)) / CGFloat(n))
                HStack(spacing: spacing) {
                    ForEach(Array(job.segments.enumerated()), id: \.offset) { _, seg in
                        ZStack(alignment: .leading) {
                            Rectangle().fill(Color.primary.opacity(0.12))
                            Rectangle()
                                .fill(seg.isComplete ? Color.green : Color.accentColor)
                                .frame(width: w * seg.fraction)
                        }
                        .frame(width: w)
                        .clipShape(RoundedRectangle(cornerRadius: 1))
                    }
                }
            }
            .frame(height: 7)

            Text(summary)
                .font(.system(size: 10).monospacedDigit())
                .foregroundStyle(.tertiary)
        }
        .padding(.top, 2)
    }

    private var summary: String {
        guard job.isSplit else {
            // One range means the server refused to split it, which explains
            // why this download is not any faster than the browser's.
            return "1 part — this server doesn't support splitting"
        }
        return "\(job.segments.count) parts in parallel · \(job.segmentsComplete) finished"
    }
}

/// One finished download: click to reveal in Finder, trash to delete.
struct RecentJobRow: View {
    @EnvironmentObject var model: DownloadsModel
    let job: Job
    @State private var hovering = false

    var body: some View {
        HStack(spacing: 6) {
            Button {
                model.reveal(job)
            } label: {
                VStack(alignment: .leading, spacing: 1) {
                    Text(job.name)
                        .font(.callout)
                        .lineLimit(1)
                        .truncationMode(.middle)
                    Text("\(human(job.total)) · click to show in Finder")
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                        .lineLimit(1)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            Spacer(minLength: 4)
            IconButton(systemName: "trash", help: "Delete file") {
                model.act("delete", job.id)
            }
            .opacity(hovering ? 1 : 0)
        }
        .padding(.horizontal, 6)
        .padding(.vertical, 6)
        .background(
            RoundedRectangle(cornerRadius: 6)
                .fill(hovering ? Color.primary.opacity(0.06) : .clear)
        )
        .onHover { hovering = $0 }
    }
}
