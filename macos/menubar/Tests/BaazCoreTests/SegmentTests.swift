import AppKit
import SwiftUI
import XCTest
@testable import BaazCore

/// Clicking a running download opens its byte ranges. The daemon sends these
/// only while a job is in flight (internal/ipc/protocol.go).
@MainActor
final class SegmentTests: XCTestCase {
    private func job(_ segmentJSON: String) throws -> Job {
        let json = #"[{"id":"a","name":"x.zip","state":"active","total":8000,"done":2400,"segments":[\#(segmentJSON)]}]"#
        return try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
    }

    func testDecodesSegments() throws {
        let j = try job(#"{"done":250,"total":1000},{"done":1000,"total":1000}"#)
        XCTAssertEqual(j.segments.count, 2)
        XCTAssertEqual(j.segments[0].fraction, 0.25)
        XCTAssertFalse(j.segments[0].isComplete)
        XCTAssertTrue(j.segments[1].isComplete)
        XCTAssertEqual(j.segmentsComplete, 1)
        XCTAssertTrue(j.isSplit)
    }

    /// A server that refuses Range yields one segment. Drawing that as a
    /// "parts" display would claim a split that never happened.
    func testSingleSegmentIsNotASplit() throws {
        let j = try job(#"{"done":500,"total":1000}"#)
        XCTAssertFalse(j.isSplit)
    }

    func testUnknownSegmentLengthDoesNotReadAsComplete() throws {
        let j = try job(#"{"done":900,"total":-1}"#)
        XCTAssertEqual(j.segments[0].fraction, 0, "no length means no measurable progress")
        XCTAssertFalse(j.segments[0].isComplete)
    }

    func testSegmentFractionClamps() throws {
        let j = try job(#"{"done":1500,"total":1000}"#)
        XCTAssertEqual(j.segments[0].fraction, 1)
    }

    /// Finished jobs carry no segments, and older daemons send none at all.
    func testMissingSegmentsDecodeEmpty() throws {
        let json = #"[{"id":"a","name":"x.zip","state":"done","total":10,"done":10}]"#
        let j = try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
        XCTAssertEqual(j.segments, [])
        XCTAssertFalse(j.isSplit)
    }

    func testExpansionTogglesAndSurvivesSnapshots() {
        let model = DownloadsModel()
        let snap = #"{"type":"snapshot","active":1,"jobs":[{"id":"a1","name":"x.zip","state":"active","total":8000,"done":100,"segments":[{"done":1,"total":10},{"done":2,"total":10}]}],"recent":[]}"#
        model.ingest(Data((snap + "\n").utf8))

        XCTAssertFalse(model.isExpanded("a1"))
        model.toggleExpanded("a1")
        XCTAssertTrue(model.isExpanded("a1"))

        // A new snapshot arrives twice a second; the open row must stay open.
        model.ingest(Data((snap + "\n").utf8))
        XCTAssertTrue(model.isExpanded("a1"), "expansion must survive a refresh")

        model.toggleExpanded("a1")
        XCTAssertFalse(model.isExpanded("a1"))
    }

    /// Sends a real click through the window, because that is the only thing
    /// that catches this class of bug: `.onTapGesture` renders a row that
    /// looks clickable and silently does nothing inside the menu bar panel.
    /// Layout assertions pass either way.
    /// Clicks `point` in `host` through the window, the way a real click
    /// arrives. Shared by the canary below and the test itself.
    private func click(_ point: NSPoint, in host: NSView, window win: NSWindow) {
        let inWindow = host.convert(point, to: nil)
        for (type, num) in [(NSEvent.EventType.leftMouseDown, 1),
                            (NSEvent.EventType.leftMouseUp, 2)] {
            if let e = NSEvent.mouseEvent(with: type, location: inWindow, modifierFlags: [],
                timestamp: ProcessInfo.processInfo.systemUptime,
                windowNumber: win.windowNumber, context: nil, eventNumber: num,
                clickCount: 1, pressure: type == .leftMouseDown ? 1 : 0) {
                win.sendEvent(e)
            }
        }
        RunLoop.current.run(until: Date().addingTimeInterval(0.3))
    }

    /// Whether synthetic clicks reach SwiftUI at all here.
    ///
    /// A headless CI runner routes no events, and the assertion below cannot
    /// tell that apart from a button that is genuinely dead — which is the
    /// whole point of the test. So prove the mechanism on a control that is
    /// certainly wired up first: if even this does not fire, the environment
    /// is at fault and the real test is skipped rather than failed.
    private func canRouteSyntheticClicks() -> Bool {
        final class Probe { var clicked = false }
        let probe = Probe()
        let host = NSHostingView(rootView: AnyView(
            Button { probe.clicked = true } label: {
                Color.clear.frame(width: 200, height: 200).contentShape(Rectangle())
            }
            .buttonStyle(.plain)
        ))
        let win = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 200, height: 200),
                           styleMask: [.titled], backing: .buffered, defer: false)
        win.contentView = host
        win.makeKeyAndOrderFront(nil)
        defer { win.orderOut(nil) }
        host.layoutSubtreeIfNeeded()
        RunLoop.current.run(until: Date().addingTimeInterval(0.3))
        click(NSPoint(x: 100, y: 100), in: host, window: win)
        return probe.clicked
    }

    func testClickingTheRowActuallyExpandsIt() throws {
        try XCTSkipUnless(canRouteSyntheticClicks(),
                          "this environment does not deliver synthetic clicks to SwiftUI")

        let model = DownloadsModel()
        let segs = (0..<8).map { #"{"done":\#($0 * 100),"total":1000}"# }.joined(separator: ",")
        model.ingest(Data((#"{"type":"snapshot","active":1,"totalSpeed":900,"jobs":[{"id":"a1","name":"x.zip","state":"active","total":8000,"done":2400,"speed":900,"eta":12,"dir":"/tmp","segments":[\#(segs)]}],"recent":[]}"# + "\n").utf8))

        let host = NSHostingView(rootView: AnyView(PanelView().environmentObject(model)))
        let win = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 340, height: 700),
                           styleMask: [.titled], backing: .buffered, defer: false)
        win.contentView = host
        win.makeKeyAndOrderFront(nil)
        host.layoutSubtreeIfNeeded()
        RunLoop.current.run(until: Date().addingTimeInterval(0.4))

        // Sweep the whole upper panel rather than pinning magic coordinates —
        // font metrics shift the row between runners.
        var toggled = false
        for y in stride(from: CGFloat(60), through: 260, by: 4) where !toggled {
            click(NSPoint(x: 90, y: y), in: host, window: win)
            if model.isExpanded("a1") { toggled = true }
        }
        XCTAssertTrue(toggled, "no click anywhere on the row opened the segment breakdown")

        win.orderOut(nil)
    }

    /// The panel has to actually grow, or the breakdown would be clipped.
    func testExpandingMakesRoomForTheBreakdown() {
        let model = DownloadsModel()
        let segs = (0..<8).map { #"{"done":\#($0 * 100),"total":1000}"# }.joined(separator: ",")
        model.ingest(Data((#"{"type":"snapshot","active":1,"jobs":[{"id":"a1","name":"x.zip","state":"active","total":8000,"done":2400,"segments":[\#(segs)]}],"recent":[]}"# + "\n").utf8))

        let host = NSHostingView(rootView: AnyView(PanelView().environmentObject(model)))
        func settle() -> CGFloat {
            host.layoutSubtreeIfNeeded()
            RunLoop.current.run(until: Date().addingTimeInterval(0.3))
            host.frame = NSRect(origin: .zero, size: host.fittingSize)
            host.layoutSubtreeIfNeeded()
            return host.fittingSize.height
        }
        let collapsed = settle()
        model.toggleExpanded("a1")
        let expanded = settle()
        XCTAssertGreaterThan(expanded, collapsed,
                             "expanding must add room for the segment bars")
    }
}
