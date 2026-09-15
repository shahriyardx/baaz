import AppKit
import XCTest
@testable import BaazCore

/// Baaz sat in the Dock permanently, labelled "Running in Background" — a
/// tile you cannot dismiss without quitting the thing doing the downloading.
/// It should only be there while it has a window open.
///
/// The counting is what decides that, and it has to tell a real window from
/// the menu bar panel and from the scaffolding AppKit keeps around, so that
/// is what these cover.
@MainActor
final class DockPresenceTests: XCTestCase {
    private func window(titled: Bool = true, visible: Bool = true) -> NSWindow {
        let w = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 200, height: 120),
                         styleMask: titled ? [.titled, .closable] : [.borderless],
                         backing: .buffered, defer: false)
        if visible { w.orderFront(nil) } else { w.orderOut(nil) }
        return w
    }

    override func tearDown() {
        NSApp.windows.forEach { $0.orderOut(nil) }
        super.tearDown()
    }

    func testNoWindowsMeansNoDockIcon() {
        XCTAssertEqual(DockPresence.realWindowCount([]), 0)
    }

    func testARealWindowCounts() {
        XCTAssertEqual(DockPresence.realWindowCount([window()]), 1)
    }

    /// The menu bar panel is an NSPanel and must never pull Baaz into the
    /// Dock — it is open every time someone glances at their downloads.
    func testTheMenuBarPanelDoesNotCount() {
        let panel = NSPanel(contentRect: NSRect(x: 0, y: 0, width: 300, height: 200),
                            styleMask: [.nonactivatingPanel, .titled],
                            backing: .buffered, defer: false)
        panel.orderFront(nil)
        XCTAssertEqual(DockPresence.realWindowCount([panel]), 0)
        panel.orderOut(nil)
    }

    /// SwiftUI and AppKit keep untitled and off-screen windows around; they
    /// are not something the user opened.
    func testScaffoldingDoesNotCount() {
        XCTAssertEqual(DockPresence.realWindowCount([window(titled: false)]), 0)
        XCTAssertEqual(DockPresence.realWindowCount([window(visible: false)]), 0)
    }

    func testSeveralWindowsCountSeparately() {
        let ws = [window(), window()]
        XCTAssertEqual(DockPresence.realWindowCount(ws), 2)
        ws.forEach { $0.orderOut(nil) }
    }

    /// A mixed bag is the real case: main window, settings, the panel, and
    /// whatever else is floating about.
    func testAMixtureCountsOnlyTheRealOnes() {
        let panel = NSPanel(contentRect: NSRect(x: 0, y: 0, width: 300, height: 200),
                            styleMask: [.nonactivatingPanel], backing: .buffered, defer: false)
        panel.orderFront(nil)
        let ws: [NSWindow] = [window(), panel, window(titled: false), window()]
        XCTAssertEqual(DockPresence.realWindowCount(ws), 2)
        ws.forEach { $0.orderOut(nil) }
    }
}
