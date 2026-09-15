import AppKit
import SwiftUI
import XCTest
@testable import BaazCore

/// Liquid Glass is applied to the surfaces Baaz draws itself. Standard
/// controls get it from the SDK with no code, so there is nothing to test
/// there; these cover the helpers, which have to keep working on macOS 13
/// through 25 where the APIs do not exist.
@MainActor
final class GlassTests: XCTestCase {
    private func job(_ state: String) throws -> Job {
        let json = #"[{"id":"a","name":"x.zip","state":"\#(state)","total":100,"done":40,"segments":[{"done":1,"total":10}]}]"#
        return try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
    }

    private func render(_ view: some View) -> NSImage? {
        let host = NSHostingView(rootView: AnyView(view.environmentObject(DownloadsModel())))
        host.frame = NSRect(x: 0, y: 0, width: 600, height: 120)
        host.layoutSubtreeIfNeeded()
        guard let rep = host.bitmapImageRepForCachingDisplay(in: host.bounds) else { return nil }
        host.cacheDisplay(in: host.bounds, to: rep)
        let img = NSImage(size: host.bounds.size)
        img.addRepresentation(rep)
        return img
    }

    /// Both states have to draw. The selected one takes a different path —
    /// tinted glass rather than plain — and a crash there would only show up
    /// when someone clicked a row.
    func testACardDrawsInBothStates() throws {
        for selected in [false, true] {
            let img = render(DownloadCard(job: try job("active"), selected: selected))
            XCTAssertNotNil(img, "selected=\(selected) did not render")
            XCTAssertGreaterThan(img?.size.width ?? 0, 0)
        }
    }

    func testEveryJobStateDraws() throws {
        for state in ["queued", "active", "paused", "done", "failed"] {
            XCTAssertNotNil(render(DownloadCard(job: try job(state), selected: false)),
                            "\(state) did not render")
        }
    }

    /// The container is a plain passthrough before macOS 26, so its content
    /// must survive either way.
    func testTheGlassContainerPassesContentThrough() {
        let img = render(BaazGlassContainer(spacing: 8) { Text("inside").padding() })
        XCTAssertNotNil(img)
    }

    func testGlassHelpersDrawOnAnyShape() {
        for shape in [AnyShape(RoundedRectangle(cornerRadius: 12)), AnyShape(Capsule())] {
            XCTAssertNotNil(render(Text("x").padding().baazGlass(in: shape)))
            XCTAssertNotNil(render(Text("x").padding().baazGlass(in: shape, tinted: true)))
        }
    }
}
