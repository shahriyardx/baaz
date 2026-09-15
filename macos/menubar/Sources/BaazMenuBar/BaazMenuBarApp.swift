import AppKit
import BaazCore
import SwiftUI

@main
struct BaazMenuBarApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var delegate
    @StateObject private var updater = UpdaterController()

    var body: some Scene {
        // The real app: a window with every download and the controls for
        // them. The menu bar item below stays the glance.
        Window("Baaz", id: "main") {
            MainWindow()
                .environmentObject(delegate.model)
        }
        .defaultSize(width: 900, height: 560)
        .commands {
            // Replaces the app menu's Settings item so ⌘, opens the window
            // above. The stock Settings scene gave no way to open it from the
            // UI at all.
            CommandGroup(after: .windowList) {
                WindowMenuButton()
            }
            CommandGroup(after: .appInfo) {
                CheckForUpdatesButton(updater: updater)
            }
            CommandGroup(replacing: .appSettings) {
                SettingsMenuButton()
            }
            CommandGroup(replacing: .newItem) {
                Button("New Download…") {
                    NSApp.activate(ignoringOtherApps: true)
                    NotificationCenter.default.post(name: .baazAddDownload, object: nil)
                }
                .keyboardShortcut("n")
            }
            CommandGroup(after: .newItem) {
                Button("Pause All") { delegate.model.pauseAll() }
                    .keyboardShortcut(".", modifiers: [.command])
                Button("Resume All") { delegate.model.resumeAll() }
                    .keyboardShortcut("r", modifiers: [.command, .shift])
                Divider()
                Button("Open Downloads Folder") { delegate.model.openDownloadDir() }
                    .keyboardShortcut("d", modifiers: [.command, .shift])
            }
        }

        Window("Settings", id: "settings") {
            SettingsWindow()
                .environmentObject(delegate.model)
        }
        .defaultSize(width: SettingsWindowMetrics.width, height: SettingsWindowMetrics.defaultHeight)
        .windowResizability(.contentSize)

        MenuBarExtra {
            PanelView()
                .environmentObject(delegate.model)
        } label: {
            // The icon alone when idle; icon plus speed and percent while
            // downloads run, so the menu bar stays quiet when nothing is
            // happening.
            MenuBarLabel(model: delegate.model)
        }
        .menuBarExtraStyle(.window)
    }
}

private struct WindowMenuButton: View {
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        Button("Baaz Downloads") {
            NSApp.activate(ignoringOtherApps: true)
            openWindow(id: "main")
        }
        .keyboardShortcut("0", modifiers: .command)
    }
}

/// Lives in the commands builder, which has no environment of its own.
private struct SettingsMenuButton: View {
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        Button("Settings…") {
            NSApp.activate(ignoringOtherApps: true)
            openWindow(id: "settings")
        }
        .keyboardShortcut(",", modifiers: .command)
    }
}

private struct MenuBarLabel: View {
    @ObservedObject var model: DownloadsModel
    @Environment(\.openWindow) private var openWindow

    var body: some View {
        HStack(spacing: 4) {
            // The baaz falcon, as a template image so the menu bar tints it
            // for light and dark itself. Activity shows in the text beside it
            // rather than a second icon, so the mark never changes shape.
            Image(nsImage: FalconIcon.menuBar)
            if !model.barText.isEmpty {
                Text(model.barText).font(.system(size: 11).monospacedDigit())
            }
        }
        // The menu bar label exists from launch, before any window or panel
        // has been shown, so it is the earliest reliable place to capture
        // SwiftUI's openWindow for code outside the view hierarchy.
        .onAppear { WindowOpener.action = { openWindow(id: $0) } }
    }
}

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    let model = DownloadsModel()
    private var sigterm: DispatchSourceSignal?

    func applicationDidFinishLaunching(_ notification: Notification) {
        // Ask under Baaz's own name. The prompt used to say "Script Editor",
        // because the daemon posted through osascript.
        Notifier.shared.requestPermission()
        guard !terminateIfDuplicate() else { return }

        // launchd stops the login item with SIGTERM. AppKit's own handling
        // does not run applicationWillTerminate for it, and skipping the
        // cleanup below would strand `baaz watch` holding a subscription the
        // daemon never drops while idle.
        signal(SIGTERM, SIG_IGN)
        let src = DispatchSource.makeSignalSource(signal: SIGTERM, queue: .main)
        src.setEventHandler { NSApp.terminate(nil) }
        src.resume()
        sigterm = src

        model.start()

        // Dragging the app out of the DMG is the install: this puts the CLI
        // on PATH, wires up Chrome and registers the login item. Loading the
        // extension is the one step Chrome will not let an app do.
        // The callback is @Sendable and therefore nonisolated; the alert has
        // to be raised on the main actor explicitly rather than relying on
        // the callback happening to arrive there.
        Setup.runIfNeeded { firstRun in
            guard firstRun else { return }
            Task { @MainActor in Self.offerExtensionSetup() }
        }
    }

    /// Shown once per version. The extension is the one part of the install
    /// Chrome will not let an app do for you.
    private static func offerExtensionSetup() {
        let alert = NSAlert()
        alert.messageText = "One last step: add the Chrome extension"
        alert.informativeText = """
            Baaz takes over downloads through its Chrome extension.             It is waiting on Chrome Web Store review, so for now it installs             by hand — once, and it keeps working after the listing goes live.

            1. Download the zip and unzip it.
            2. Open chrome://extensions and switch on Developer mode.
            3. Click "Load unpacked" and choose the baaz-extension folder.
            """
        alert.addButton(withTitle: "Download the Extension")
        alert.addButton(withTitle: "Later")
        alert.alertStyle = .informational
        NSApp.activate(ignoringOtherApps: true)
        if alert.runModal() == .alertFirstButtonReturn {
            NSWorkspace.shared.open(Setup.extensionURL)
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        model.stop()
    }

    /// Closing the window is not quitting: the menu bar item is still there
    /// and downloads keep running. Quit is explicit, from the menu or ⌘Q.
    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        false
    }

    /// Clicking the Dock icon with no window open brings it back, which is
    /// what every other Mac app does.
    func applicationShouldHandleReopen(_ sender: NSApplication,
                                       hasVisibleWindows flag: Bool) -> Bool {
        if !flag { model.openMainWindow() }
        return true
    }

    /// `install-menubar` registers a login item that launchd starts, so a
    /// second copy opened by hand would put two icons in the menu bar. The
    /// newer process is the one that yields.
    private func terminateIfDuplicate() -> Bool {
        guard let id = Bundle.main.bundleIdentifier else { return false }
        let me = NSRunningApplication.current
        let older = NSRunningApplication.runningApplications(withBundleIdentifier: id)
            .contains { other in
                guard other.processIdentifier != me.processIdentifier,
                      let theirs = other.launchDate, let mine = me.launchDate
                else { return false }
                return theirs < mine
            }
        if older { NSApp.terminate(nil) }
        return older
    }
}
