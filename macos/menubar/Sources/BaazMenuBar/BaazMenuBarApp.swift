import AppKit
import BaazCore
import SwiftUI

@main
struct BaazMenuBarApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var delegate

    var body: some Scene {
        // The real app: a window with every download and the controls for
        // them. The menu bar item below stays the glance.
        Window("baaz", id: "main") {
            MainWindow()
                .environmentObject(delegate.model)
        }
        .defaultSize(width: 900, height: 560)
        .commands {
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

private struct MenuBarLabel: View {
    @ObservedObject var model: DownloadsModel

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
    }
}

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate {
    let model = DownloadsModel()
    private var sigterm: DispatchSourceSignal?

    func applicationDidFinishLaunching(_ notification: Notification) {
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
        Setup.runIfNeeded { firstRun in
            if firstRun { Self.offerExtensionSetup() }
        }
    }

    /// Shown once per version, after setup has unpacked the extension.
    private static func offerExtensionSetup() {
        let dir = FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Downloads/baaz-extension")
        guard FileManager.default.fileExists(atPath: dir.path) else { return }

        let alert = NSAlert()
        alert.messageText = "One last step: add the Chrome extension"
        alert.informativeText = """
        Chrome does not allow an app to install an extension for you, so it         has to be added by hand once — it stays after that.

        In Chrome open chrome://extensions, turn on Developer mode, click         Load unpacked, and choose the baaz-extension folder in your Downloads.
        """
        alert.addButton(withTitle: "Show the Folder")
        alert.addButton(withTitle: "Later")
        alert.alertStyle = .informational
        NSApp.activate(ignoringOtherApps: true)
        if alert.runModal() == .alertFirstButtonReturn {
            NSWorkspace.shared.activateFileViewerSelecting([dir])
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
