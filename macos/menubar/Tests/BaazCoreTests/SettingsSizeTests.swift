import AppKit
import SwiftUI
import XCTest
@testable import BaazCore

/// The Settings window opens at a fixed size. If a tab's content is taller
/// than that, its last row is cut in half — which is what shipped: the
/// Downloads tab ended part-way through "Only take over files above".
///
/// Measuring SettingsWindow itself does not catch this. A TabView reports the
/// size of the tab it is showing, so the window looks fine as long as the
/// first tab fits. Each tab has to be measured on its own.
@MainActor
final class SettingsSizeTests: XCTestCase {
    private func fittingHeight<V: View>(_ view: V) -> CGFloat {
        let model = DownloadsModel()
        let host = NSHostingView(rootView: AnyView(view.environmentObject(model)))
        host.layoutSubtreeIfNeeded()
        return host.fittingSize.height
    }

    private var tabs: [(String, CGFloat)] {
        [("General", fittingHeight(GeneralSettings())),
         ("Downloads", fittingHeight(TransferSettings())),
         ("Browser", fittingHeight(BrowserSettings()))]
    }

    func testEveryTabFitsTheWindow() {
        // The window's own height also carries the tab bar and the padding
        // around the content, so a tab may only use what is left.
        let available = SettingsWindowMetrics.defaultHeight
            - SettingsWindowMetrics.chromeHeight
        for (name, h) in tabs {
            XCTAssertGreaterThan(h, 0, "could not measure the \(name) tab")
            XCTAssertLessThanOrEqual(
                h, available,
                "the \(name) tab needs \(Int(h))pt but only \(Int(available))pt is available, "
                + "so its last row is clipped")
        }
    }

    func testTheWindowIsNotMuchTallerThanItNeeds() {
        let tallest = tabs.map(\.1).max() ?? 0
        let available = SettingsWindowMetrics.defaultHeight
            - SettingsWindowMetrics.chromeHeight
        XCTAssertLessThan(available - tallest, 120,
                          "the window opens well beyond its tallest tab, leaving dead space")
    }
}
