import AppKit
import SwiftUI

/// Size the Settings window opens at. Kept next to the view so the two
/// cannot drift apart: a window shorter than its content clips the last row.
public enum SettingsWindowMetrics {
    public static let width: CGFloat = 520
    public static let defaultHeight: CGFloat = 450
    /// Everything in the window that is not tab content: the title bar, the
    /// tab strip, and the padding above and below. Measured against the real
    /// window, not guessed.
    public static let chromeHeight: CGFloat = 106
    /// Floor for the tab content. defaultSize alone is not enough — macOS
    /// restores whatever size the window was last left at, so an older,
    /// shorter frame would keep clipping the tallest tab forever.
    public static let contentMinHeight: CGFloat = defaultHeight - chromeHeight
}

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
        .frame(minHeight: SettingsWindowMetrics.contentMinHeight)
        .padding(20)
    }
}

struct GeneralSettings: View {
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
                Toggle("Open Baaz at login", isOn: Binding(
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

struct TransferSettings: View {
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
                numberRow("Downloads at the same time",
                          key: "max-active", value: model.settings.maxActive, in: 1...10)
                numberRow("Parts per file",
                          key: "segments", value: model.settings.segments, in: 1...32)
                Text("A file is pulled down in this many pieces at once. More is not always faster, and some sites only ever allow one.")
                    .font(.caption).foregroundStyle(.secondary)
            }

            Section {
                numberRow("Only take over files above",
                          key: "min-size", value: model.settings.minSizeMB,
                          in: 0...500, step: 5, unit: " MB")
                Text("Smaller downloads stay with Chrome.")
                    .font(.caption).foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
    }

    /// A settings line: label on the left, the value and its stepper together
    /// on the right, where the other controls in this window sit. Putting the
    /// number inside the label left it stranded mid-row.
    private func numberRow(_ label: String, key: String, value: Int,
                           in range: ClosedRange<Int>, step: Int = 1,
                           unit: String = "") -> some View {
        LabeledContent(label) {
            HStack(spacing: 8) {
                Text("\(value)\(unit)")
                    .monospacedDigit()
                    .foregroundStyle(.secondary)
                Stepper("", value: bind(key, value), in: range, step: step)
                    .labelsHidden()
            }
        }
    }

    /// Writes through to the daemon; the snapshot brings the value back.
    private func bind(_ key: String, _ value: Int) -> Binding<Int> {
        Binding(get: { value }, set: { model.setConfig(key, String($0)) })
    }
}

struct BrowserSettings: View {
    @EnvironmentObject var model: DownloadsModel
    @State private var reran = false

    var body: some View {
        Form {
            Section {
                Toggle("Take over Chrome downloads", isOn: Binding(
                    get: { model.settings.intercept },
                    set: { _ in model.toggleIntercept() }
                ))
                Text("Off means Chrome downloads by itself, as if Baaz were not installed.")
                    .font(.caption).foregroundStyle(.secondary)
            }

            Section("Extension") {
                Text("Baaz takes over downloads through its Chrome extension. If downloads still go to Chrome, the extension is probably missing or switched off.")
                    .font(.caption).foregroundStyle(.secondary)
                Text("While it waits on Chrome Web Store review it installs by hand: download the zip, unzip it, open chrome://extensions, switch on Developer mode, then \"Load unpacked\" and choose the baaz-extension folder.")
                    .font(.caption).foregroundStyle(.secondary)
                HStack {
                    Button("Download the Extension") {
                        NSWorkspace.shared.open(Setup.extensionURL)
                    }
                    Button("Re-run Chrome Setup") {
                        Setup.rerunChromeSetup()
                        reran = true
                    }
                    if reran {
                        Text("Done — restart Chrome.")
                            .font(.caption).foregroundStyle(.secondary)
                    }
                }
            }
        }
        .formStyle(.grouped)
    }
}
