import AppKit
import SwiftUI

/// The main window: every download, with the controls to manage them.
///
/// The menu bar panel stays the glance; this is where the work happens.
public struct MainWindow: View {
    @EnvironmentObject var model: DownloadsModel
    @Environment(\.openWindow) private var openWindow
    @State private var filter: JobFilter = .all
    @State private var selection: String?
    @State private var addingDownload = false
    @State private var search = ""
    @State private var showInspector = true

    public init() {}

    private var visible: [Job] {
        let rows = model.jobs(matching: filter)
        guard !search.isEmpty else { return rows }
        return rows.filter { $0.name.localizedCaseInsensitiveContains(search) }
    }

    public var body: some View {
        NavigationSplitView {
            List(JobFilter.allCases, selection: Binding(
                get: { filter },
                set: { filter = $0 ?? .all }
            )) { f in
                Label {
                    HStack {
                        Text(f.title)
                        Spacer()
                        Text("\(model.count(f))")
                            .font(.caption.monospacedDigit())
                            .foregroundStyle(.secondary)
                    }
                } icon: {
                    Image(systemName: f.symbol)
                }
                .tag(f)
            }
            .navigationSplitViewColumnWidth(min: 170, ideal: 190, max: 240)
            .safeAreaInset(edge: .bottom) {
                VStack(spacing: 0) {
                    Divider()
                    Button {
                        openWindow(id: "settings")
                    } label: {
                        Label("Settings", systemImage: "gearshape")
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 7)
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .keyboardShortcut(",", modifiers: .command)
                }
                .background(.bar)
            }
        } detail: {
            detail
        }
        .navigationTitle("baaz")
        .navigationSubtitle(model.statusLine)
        .toolbar { toolbar }
        .searchable(text: $search, placement: .toolbar, prompt: "Filter by name")
        .sheet(isPresented: $addingDownload) {
            AddDownloadSheet().environmentObject(model)
        }
        .frame(minWidth: 720, minHeight: 420)
        .onReceive(NotificationCenter.default.publisher(for: .baazAddDownload)) { _ in
            addingDownload = true
        }
    }

    @ViewBuilder
    private var detail: some View {
        if visible.isEmpty {
            ContentUnavailableViewCompat(
                title: emptyTitle,
                message: emptyMessage,
                systemImage: filter.symbol,
                actionTitle: filter == .all && search.isEmpty ? "Add a Download" : nil,
                action: { addingDownload = true }
            )
        } else {
            HStack(spacing: 0) {
                ScrollView {
                    LazyVStack(spacing: 6) {
                        ForEach(visible) { job in
                            DownloadCard(job: job, selected: selection == job.id)
                                .onTapGesture { selection = job.id }
                        }
                    }
                    .padding(12)
                }

                if showInspector, let job = selectedJob {
                    Divider()
                    InspectorView(job: job)
                        .frame(width: 270)
                        .background(Color(nsColor: .underPageBackgroundColor))
                        .transition(.move(edge: .trailing))
                }
            }
        }
    }

    /// Follows the live job rather than a copy taken at click time, so the
    /// inspector keeps updating as the download runs.
    private var selectedJob: Job? {
        guard let id = selection else { return nil }
        return model.allJobs.first { $0.id == id }
    }

    private var emptyTitle: String {
        if !search.isEmpty { return "No matches" }
        switch filter {
        case .all: return "No downloads yet"
        case .active: return "Nothing downloading"
        case .paused: return "Nothing paused"
        case .done: return "Nothing finished yet"
        case .failed: return "No failures"
        }
    }

    private var emptyMessage: String {
        if !search.isEmpty { return "No download matches “\(search)”." }
        if filter == .all {
            return "Downloads from Chrome land here automatically, or add a link yourself."
        }
        return ""
    }

    @ToolbarContentBuilder
    private var toolbar: some ToolbarContent {
        ToolbarItemGroup {
            Button { addingDownload = true } label: {
                Label("Add Download", systemImage: "plus")
            }
            .help("Add a download by link (⌘N)")

            Button { model.pauseAll() } label: {
                Label("Pause All", systemImage: "pause.circle")
            }
            .help("Pause every running download")
            .disabled(model.activeCount == 0)

            Button { model.resumeAll() } label: {
                Label("Resume All", systemImage: "play.circle")
            }
            .help("Resume everything paused or failed")
            .disabled(!model.jobs.contains { $0.state == "paused" || $0.isFailed })

            Spacer()

            Button { model.openDownloadDir() } label: {
                Label("Open Downloads Folder", systemImage: "folder")
            }
            .help("Open the folder downloads are saved to")

            Button { openWindow(id: "settings") } label: {
                Label("Settings", systemImage: "gearshape")
            }
            .help("Settings (⌘,)")

            Button { showInspector.toggle() } label: {
                Label("Details", systemImage: "sidebar.right")
            }
            .help("Show or hide the details panel")
            .disabled(selection == nil)
        }
    }
}

/// Notification so the app's File menu can open the add sheet in the window.
public extension Notification.Name {
    static let baazAddDownload = Notification.Name("baaz.addDownload")
}

/// ContentUnavailableView is macOS 14+; this keeps the deployment target at 13.
struct ContentUnavailableViewCompat: View {
    let title: String
    let message: String
    let systemImage: String
    var actionTitle: String?
    var action: () -> Void

    var body: some View {
        VStack(spacing: 10) {
            Image(systemName: systemImage)
                .font(.system(size: 40, weight: .thin))
                .foregroundStyle(.tertiary)
            Text(title).font(.title3.weight(.medium))
            if !message.isEmpty {
                Text(message)
                    .font(.callout)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: 360)
            }
            if let actionTitle {
                Button(actionTitle, action: action)
                    .buttonStyle(.borderedProminent)
                    .padding(.top, 4)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
