import AppKit
import SwiftUI
import XCTest
@testable import BaazCore

/// Layout invariants for the panel: the list must always have room, must grow
/// with content, and must stay bounded.
///
/// These do NOT reproduce the reported "header says 1 active, list is empty"
/// bug — they pass against the older sizing too. NSHostingView lays the panel
/// out correctly in isolation; whatever goes wrong needs the real MenuBarExtra
/// panel window, which cannot be driven from a test. They are guards against
/// the list collapsing, not proof that bug is fixed.
@MainActor
final class PanelLayoutTests: XCTestCase {
    private func panel() -> (NSHostingView<AnyView>, DownloadsModel) {
        let model = DownloadsModel()
        let host = NSHostingView(rootView: AnyView(
            PanelView().environmentObject(model)
        ))
        return (host, model)
    }

    /// Lays the panel out at the size MenuBarExtra would give it and returns
    /// the height the list actually gets on screen.
    private func listHeight(_ host: NSHostingView<AnyView>) -> CGFloat {
        host.layoutSubtreeIfNeeded()
        RunLoop.current.run(until: Date().addingTimeInterval(0.3))
        host.frame = NSRect(origin: .zero, size: host.fittingSize)
        host.layoutSubtreeIfNeeded()

        var clip: NSClipView?
        func walk(_ v: NSView) {
            if clip == nil, let c = v as? NSClipView { clip = c }
            v.subviews.forEach(walk)
        }
        walk(host)
        return clip?.frame.height ?? -1
    }

    /// The daemon speaks JSON-lines: one snapshot per line. A fixture that
    /// wraps across lines decodes as nothing at all.
    private func feed(_ model: DownloadsModel, _ json: String) {
        XCTAssertFalse(json.contains("\n"), "fixture must be a single line")
        model.ingest(Data((json + "\n").utf8))
    }

    func testListAreaSurvivesWhenEmpty() {
        let (host, model) = panel()
        feed(model, #"{"type":"snapshot","active":0,"jobs":[],"recent":[]}"#)
        let h = listHeight(host)
        XCTAssertGreaterThan(h, 10, "empty panel collapsed to \(h)pt — the message is invisible")
    }

    func testActiveJobGetsVisibleRoom() {
        let (host, model) = panel()
        feed(model, #"{"type":"snapshot","active":1,"totalSpeed":0,"jobs":[{"id":"a1","name":"big-file.iso","state":"active","total":1000,"done":250,"speed":0,"eta":-1,"dir":"/tmp"}],"recent":[]}"#)
        XCTAssertEqual(model.jobs.count, 1, "precondition: the job decoded")
        let h = listHeight(host)
        XCTAssertGreaterThan(h, 60, "a running download needs a visible row, got \(h)pt")
    }

    /// The reported failure: the header said "1 active" while the list showed
    /// nothing at all.
    func testHeaderAndListAgree() {
        let (host, model) = panel()
        feed(model, #"{"type":"snapshot","active":1,"totalSpeed":0,"jobs":[{"id":"a1","name":"clip.webm","state":"active","total":-1,"done":0,"speed":0,"eta":-1,"dir":"/tmp"}],"recent":[]}"#)
        XCTAssertTrue(model.statusLine.contains("1 active"))
        XCTAssertGreaterThan(listHeight(host), 60,
                             "header claims a download but the list has no room for it")
    }

    func testListGrowsWithContent() {
        let (empty, m1) = panel()
        feed(m1, #"{"type":"snapshot","active":0,"jobs":[],"recent":[]}"#)
        let emptyH = listHeight(empty)

        let (full, m2) = panel()
        feed(m2, #"{"type":"snapshot","active":0,"jobs":[],"recent":[{"id":"r1","name":"a.webm","state":"done","total":100,"done":100,"eta":-1,"dir":"/tmp"},{"id":"r2","name":"b.webm","state":"done","total":100,"done":100,"eta":-1,"dir":"/tmp"},{"id":"r3","name":"c.webm","state":"done","total":100,"done":100,"eta":-1,"dir":"/tmp"}]}"#)
        XCTAssertGreaterThan(listHeight(full), emptyH,
                             "three finished downloads should take more room than none")
    }

    /// And it must stay bounded, or a long history would push the footer off
    /// the bottom of the screen.
    func testListStaysBounded() {
        let (host, model) = panel()
        let rows = (0..<40).map {
            #"{"id":"r\#($0)","name":"file-\#($0).bin","state":"done","total":100,"done":100,"eta":-1,"dir":"/tmp"}"#
        }.joined(separator: ",")
        feed(model, #"{"type":"snapshot","active":0,"jobs":[],"recent":[\#(rows)]}"#)
        XCTAssertLessThanOrEqual(listHeight(host), 420, "the list must cap and scroll")
    }
}
