import XCTest
@testable import BaazCore

/// The one-time video-tool download happens in the background at first
/// launch. If nothing says so, the app looks idle while it quietly pulls
/// 80MB — and someone who then tries a video wonders why it is slow.
@MainActor
final class SetupNoticeTests: XCTestCase {
    /// Fed the same way the app is: one JSON snapshot per line, as
    /// `baaz watch` emits them.
    private func model(_ json: String) -> DownloadsModel {
        let m = DownloadsModel()
        m.ingest(Data((json.replacingOccurrences(of: "\n", with: " ") + "\n").utf8))
        return m
    }

    func testSetupProgressIsShownWhileIdle() {
        let m = model(#"{"type":"snapshot","setup":"setting up video support — 12.3MB of 35.4MB","settings":{"intercept":true}}"#)
        XCTAssertEqual(m.statusLine, "setting up video support — 12.3MB of 35.4MB")
    }

    func testARealDownloadOutranksTheBackgroundErrand() {
        let m = model(#"""
        {"type":"snapshot","active":1,"totalSpeed":500000,
         "setup":"setting up video support — 12.3MB of 35.4MB",
         "jobs":[{"id":"a","name":"x","state":"active"}],
         "settings":{"intercept":true}}
        """#)
        XCTAssertTrue(m.statusLine.contains("active"),
                      "a download the user started must not be hidden behind setup: \(m.statusLine)")
    }

    func testNothingIsShownOnceSetupIsDone() {
        let m = model(#"{"type":"snapshot","settings":{"intercept":true}}"#)
        XCTAssertEqual(m.statusLine, "idle")
    }

    func testAbsentSetupFieldDecodesAsEmpty() throws {
        let snap = try JSONDecoder().decode(Snapshot.self, from: Data(#"{"type":"snapshot"}"#.utf8))
        XCTAssertEqual(snap.setup, "")
    }

    /// Turning interception off is a state the user chose; it still wins.
    func testInterceptOffStillTakesPrecedence() {
        let m = model(#"{"type":"snapshot","setup":"setting up video support — 1MB of 35MB","settings":{"intercept":false}}"#)
        XCTAssertTrue(m.statusLine.contains("off"), m.statusLine)
    }
}
