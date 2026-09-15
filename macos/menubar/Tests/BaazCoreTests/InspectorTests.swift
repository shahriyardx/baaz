import AppKit
import SwiftUI
import XCTest
@testable import BaazCore

/// Nothing covered the inspector before this, which is why a flat background
/// and a header that pushed the filename below a 56pt gap both survived.
@MainActor
final class InspectorTests: XCTestCase {
    private func job(_ state: String, total: Int64 = 248_000_000,
                     done: Int64 = 94_900_000, parts: Int = 8) throws -> Job {
        let segs = (0..<parts).map { _ in #"{"done":1,"total":10}"# }.joined(separator: ",")
        let json = #"""
        [{"id":"a","name":"Why Your Company Copied Netflix's Architecture.mp4",
          "state":"\#(state)","total":\#(total),"done":\#(done),"speed":6900000,"eta":22,
          "dir":"/Users/x/Downloads/Videos","kind":"media",
          "url":"https://www.youtube.com/watch?v=I_of74HSHiA",
          "error":"\#(state == "failed" ? "could not read that link" : "")",
          "segments":[\#(segs)]}]
        """#
        return try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
    }

    /// Pinned to 270pt, the way MainWindow pins it. Measuring it
    /// unconstrained reports the ideal width of the longest unbroken word,
    /// which is a number the user never sees.
    private func host(_ job: Job) -> NSHostingView<AnyView> {
        let v = NSHostingView(rootView: AnyView(
            InspectorView(job: job)
                .frame(width: 270)
                .environmentObject(DownloadsModel())))
        v.frame = NSRect(x: 0, y: 0, width: 270, height: 800)
        v.layoutSubtreeIfNeeded()
        return v
    }

    func testItLaysOutInEveryState() throws {
        for state in ["queued", "active", "paused", "done", "failed"] {
            let h = host(try job(state))
            XCTAssertGreaterThan(h.fittingSize.height, 0, "\(state) laid out to nothing")
        }
    }

    /// It lives in a ScrollView, so being taller than the window is fine —
    /// but not taller than any plausible screen, which would mean something
    /// is expanding without bound.
    func testItStaysWithinAReasonableHeight() throws {
        for state in ["active", "done", "failed"] {
            let h = host(try job(state)).fittingSize.height
            XCTAssertLessThan(h, 1400, "\(state) needs \(Int(h))pt, which is runaway growth")
        }
    }

    /// A download with no size, no parts and no link still has to render —
    /// that is every job in the second before its headers come back.
    func testABareJobStillRenders() throws {
        let json = #"[{"id":"a","name":"x","state":"queued"}]"#
        let bare = try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
        XCTAssertGreaterThan(host(bare).fittingSize.height, 0)
    }

    /// The failure reason has to be in the panel, not only in the list row.
    func testAFailureShowsItsReason() throws {
        let j = try job("failed")
        XCTAssertFalse(j.error.isEmpty)
        XCTAssertGreaterThan(host(j).fittingSize.height, 0)
    }

    /// A very long filename must not force the panel wider than its column.
    func testALongNameDoesNotWidenThePanel() throws {
        let long = String(repeating: "verylongfilename", count: 12) + ".mp4"
        let json = #"[{"id":"a","name":"\#(long)","state":"active","total":100,"done":50}]"#
        let j = try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
        let h = host(j)
        XCTAssertLessThanOrEqual(h.fittingSize.width, 270,
                                 "a long name pushed the panel to \(h.fittingSize.width)pt")
        XCTAssertLessThan(h.fittingSize.height, 1400,
                          "a long name made the panel \(Int(h.fittingSize.height))pt tall")
    }
}

/// Clicking a download row has to select it wherever you click, not only on
/// the filename. SwiftUI hit-tests the drawn content, so padding, spacing and
/// the glass behind them swallow the tap unless the row declares its shape —
/// the same trap that once made the menu bar panel's rows look clickable and
/// do nothing.
@MainActor
final class CardHitAreaTests: XCTestCase {
    private func click(_ point: NSPoint, in host: NSView, window win: NSWindow) {
        let inWindow = host.convert(point, to: nil)
        for (type, num) in [(NSEvent.EventType.leftMouseDown, 1),
                            (NSEvent.EventType.leftMouseUp, 2)] {
            if let e = NSEvent.mouseEvent(
                with: type, location: inWindow, modifierFlags: [],
                timestamp: ProcessInfo.processInfo.systemUptime,
                windowNumber: win.windowNumber, context: nil, eventNumber: num,
                clickCount: 1, pressure: type == .leftMouseDown ? 1 : 0) {
                win.sendEvent(e)
            }
        }
        RunLoop.current.run(until: Date().addingTimeInterval(0.25))
    }

    private final class Probe { var hits = 0 }

    /// Proves synthetic clicks reach SwiftUI here at all, so a dead row and a
    /// runner that routes no events cannot be confused.
    private func canRouteClicks() -> Bool {
        let probe = Probe()
        let host = NSHostingView(rootView: AnyView(
            Button { probe.hits += 1 } label: {
                Color.clear.frame(width: 200, height: 200).contentShape(Rectangle())
            }.buttonStyle(.plain)))
        let win = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 200, height: 200),
                           styleMask: [.titled], backing: .buffered, defer: false)
        win.contentView = host
        win.makeKeyAndOrderFront(nil)
        defer { win.orderOut(nil) }
        host.layoutSubtreeIfNeeded()
        RunLoop.current.run(until: Date().addingTimeInterval(0.25))
        click(NSPoint(x: 100, y: 100), in: host, window: win)
        return probe.hits > 0
    }

    func testTheWholeRowIsClickable() throws {
        try XCTSkipUnless(canRouteClicks(),
                          "this environment does not deliver synthetic clicks to SwiftUI")

        let json = #"[{"id":"a","name":"x.zip","state":"active","total":8000,"done":2400,"segments":[{"done":1,"total":10}]}]"#
        let job = try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
        let probe = Probe()
        let host = NSHostingView(rootView: AnyView(
            Button { probe.hits += 1 } label: {
                DownloadCard(job: job, selected: false)
                    // Pinned to the list width, as LazyVStack stretches it
                    // in the real window.
                    .frame(width: 560)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .environmentObject(DownloadsModel())))
        let win = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 560, height: 200),
                           styleMask: [.titled], backing: .buffered, defer: false)
        win.contentView = host
        win.makeKeyAndOrderFront(nil)
        defer { win.orderOut(nil) }
        host.layoutSubtreeIfNeeded()
        RunLoop.current.run(until: Date().addingTimeInterval(0.35))

        // Sweep a column well clear of the filename and of the pause and
        // cancel buttons on the trailing edge, rather than pinning a
        // coordinate that font metrics can move.
        let h = host.fittingSize.height
        var landed = false
        for y in stride(from: CGFloat(8), through: max(h - 8, 8), by: 4) where !landed {
            click(NSPoint(x: 300, y: y), in: host, window: win)
            landed = probe.hits > 0
        }
        XCTAssertTrue(landed,
                      "no click anywhere down the row selected it (card is \(Int(h))pt tall)")
    }

    /// Making the whole row a button must not swallow the pause and cancel
    /// buttons sitting on its trailing edge — a row that selects instead of
    /// pausing is worse than one that only selects on the filename.
    func testThePauseAndCancelButtonsStillGetTheirClicks() throws {
        try XCTSkipUnless(canRouteClicks(),
                          "this environment does not deliver synthetic clicks to SwiftUI")

        let json = #"[{"id":"a","name":"x.zip","state":"active","total":8000,"done":2400,"segments":[{"done":1,"total":10}]}]"#
        let job = try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
        let outer = Probe()
        let host = NSHostingView(rootView: AnyView(
            Button { outer.hits += 1 } label: {
                DownloadCard(job: job, selected: false)
                    .frame(width: 560)
                    .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .environmentObject(DownloadsModel())))
        let win = NSWindow(contentRect: NSRect(x: 0, y: 0, width: 560, height: 200),
                           styleMask: [.titled], backing: .buffered, defer: false)
        win.contentView = host
        win.makeKeyAndOrderFront(nil)
        defer { win.orderOut(nil) }
        host.layoutSubtreeIfNeeded()
        RunLoop.current.run(until: Date().addingTimeInterval(0.35))

        // The controls sit hard against the trailing edge, on the first row.
        click(NSPoint(x: 540, y: 22), in: host, window: win)
        XCTAssertEqual(outer.hits, 0,
                       "the row swallowed a click meant for pause or cancel")
    }
}
