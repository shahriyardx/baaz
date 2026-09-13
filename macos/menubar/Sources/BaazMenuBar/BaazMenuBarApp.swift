import AppKit
import BaazCore
import SwiftUI

@main
struct BaazMenuBarApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) private var delegate

    var body: some Scene {
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
    }

    func applicationWillTerminate(_ notification: Notification) {
        model.stop()
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
