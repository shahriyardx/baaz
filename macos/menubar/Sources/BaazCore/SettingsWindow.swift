import AppKit
import SwiftUI

/// The app's Settings window (⌘,).
///
/// Every control writes straight through to the daemon, which owns the
/// configuration — the next snapshot is what redraws the value, so the UI
/// never keeps its own copy of the truth. The gear in the menu bar panel
/// stays as the quick version of the same settings.
public struct SettingsWindow: View {
    @EnvironmentObject var model: DownloadsModel

    public init() {}

    public var body: some View {
        TabView {
            GeneralSettings().tabItem { Label("General", systemImage: "gearshape") }
            TransferSettings().tabItem { Label("Downloads", systemImage: "arrow.down.circle") }
            BrowserSettings().tabItem { Label("Browser", systemImage: "globe") }
        }
        .frame(width: 480)
        .padding(20)
    }
}

private struct GeneralSettings: View {
    @EnvironmentObject var model: DownloadsModel
    @State private var opensAtLogin = Setup.opensAtLogin

    var body: some View {
        Form {
            Section {
                LabeledContent("Save to") {
                    HStack(spacing: 8) {
                        Text(model.settings.downloadDir.isEmpty ? "—" : model.settings.downloadDir)
                            .lineLimit(1)
                            .truncationMode(.middle)
                            .foregroundStyle(.secondary)
                        Spacer()
                        Button("Change…") { model.chooseDownloadDir() }
                        Button("Open") { model.openDownloadDir() }
                            .disabled(model.settings.downloadDir.isEmpty)
                    }
                }
                Toggle("Sort into category folders", isOn: Binding(
                    get: { model.settings.categorize },
                    set: { _ in model.toggleCategorize() }
                ))
                Text("Files land in Videos, Music, Documents and so on inside that folder.")
                    .font(.caption).foregroundStyle(.secondary)
            }

            Section {
                Toggle("Open baaz at login", isOn: Binding(
                    get: { opensAtLogin },
                    set: { opensAtLogin = Setup.setOpensAtLogin($0) }
                ))
                Text("Downloads keep running while the window is closed.")
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
    }
}

private struct TransferSettings: View {
    @EnvironmentObject var model: DownloadsModel

    private let presets: [(String, Int)] = [
        ("Unlimited", 0), ("500 KB/s", 500), ("1 MB/s", 1024),
        ("2 MB/s", 2048), ("5 MB/s", 5120), ("10 MB/s", 10240),
    ]

    var body: some View {
        Form {
            Section {
                Picker("Speed limit", selection: Binding(
                    get: { model.settings.speedLimitKB },
                    set: { model.setConfig("speed-limit", String($0)) }
                )) {
                    ForEach(presets, id: \.1) { Text($0.0).tag($0.1) }
                    if !presets.map(\.1).contains(model.settings.speedLimitKB) {
                        Text("\(model.settings.speedLimitKB) KB/s")
                            .tag(model.settings.speedLimitKB)
                    }
                }
                Text("A cap on everything at once, not per download.")
                    .font(.caption).foregroundStyle(.secondary)
            }

            Section {
                Stepper("Downloads at the same time: \(model.settings.maxActive)",
                        value: bind("max-active", model.settings.maxActive), in: 1...10)
                Stepper("Parts per file: \(model.settings.segments)",
                        value: bind("segments", model.settings.segments), in: 1...32)
                Text("A file is fetched as this many pieces at once. More is not always faster, and servers without range support always get one.")
                    .font(.caption).foregroundStyle(.secondary)
            }

            Section {
                Stepper("Only take over files above: \(model.settings.minSizeMB) MB",
                        value: bind("min-size", model.settings.minSizeMB), in: 0...500, step: 5)
                Text("Smaller downloads stay with Chrome.")
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
    }

    /// Writes through to the daemon; the snapshot brings the value back.
    private func bind(_ key: String, _ value: Int) -> Binding<Int> {
        Binding(get: { value }, set: { model.setConfig(key, String($0)) })
    }
}

private struct BrowserSettings: View {
    @EnvironmentObject var model: DownloadsModel
    @State private var reran = false

    private var extensionDir: URL {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Downloads/baaz-extension")
    }

    var body: some View {
        Form {
            Section {
                Toggle("Take over Chrome downloads", isOn: Binding(
                    get: { model.settings.intercept },
                    set: { _ in model.toggleIntercept() }
                ))
                Text("Off means Chrome downloads by itself, as if baaz were not installed.")
                    .font(.caption).foregroundStyle(.secondary)
            }

            Section("Extension") {
                Text("Chrome will not let an app install an extension from disk, so it is added by hand once. It stays after that.")
                    .font(.caption).foregroundStyle(.secondary)
                LabeledContent("Folder") {
                    HStack {
                        Text("~/Downloads/baaz-extension")
                            .foregroundStyle(.secondary)
                            .lineLimit(1).truncationMode(.middle)
                        Spacer()
                        Button("Show in Finder") {
                            NSWorkspace.shared.activateFileViewerSelecting([extensionDir])
                        }
                        .disabled(!FileManager.default.fileExists(atPath: extensionDir.path))
                    }
                }
                HStack {
                    Button("Re-run Chrome Setup") {
                        Setup.rerunChromeSetup()
                        reran = true
                    }
                    if reran {
                        Text("Done — restart Chrome to pick it up.")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
        }
        .formStyle(.grouped)
    }
}
