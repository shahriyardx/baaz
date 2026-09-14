import Sparkle
import SwiftUI

/// In-app updates, via Sparkle.
///
/// Sparkle fetches the appcast published with each GitHub release, checks the
/// archive's EdDSA signature against the public key in Info.plist, replaces
/// the bundle and relaunches. None of that needs an Apple Developer ID — the
/// EdDSA signature is what establishes the update is ours.
///
/// The update is downloaded by the app itself rather than a browser, so it
/// carries no quarantine flag and launches without a Gatekeeper prompt.
@MainActor
final class UpdaterController: ObservableObject {
    private let controller: SPUStandardUpdaterController

    /// Sparkle keeps `canCheckForUpdates` live so the menu item disables
    /// itself while a check is already running.
    @Published var canCheck = false

    private var observation: NSKeyValueObservation?

    init() {
        // startingUpdater: true begins the scheduled background checks the
        // user consented to on first run.
        controller = SPUStandardUpdaterController(startingUpdater: true,
                                                  updaterDelegate: nil,
                                                  userDriverDelegate: nil)
        canCheck = controller.updater.canCheckForUpdates
        // The KVO callback can arrive on any thread, so the value is carried
        // over to the main actor. `self` is re-captured on the inner closure:
        // reaching for the outer closure's captured `self` from inside a Task
        // is a concurrency error.
        observation = controller.updater.observe(\.canCheckForUpdates, options: [.new]) {
            [weak self] _, change in
            guard let value = change.newValue else { return }
            Task { @MainActor [weak self] in
                self?.canCheck = value
            }
        }
    }

    func checkForUpdates() {
        NSApp.activate(ignoringOtherApps: true)
        controller.checkForUpdates(nil)
    }
}

/// The "Check for Updates…" item, in the app menu where macOS apps put it.
struct CheckForUpdatesButton: View {
    @ObservedObject var updater: UpdaterController

    var body: some View {
        Button("Check for Updates…") { updater.checkForUpdates() }
            .disabled(!updater.canCheck)
    }
}
