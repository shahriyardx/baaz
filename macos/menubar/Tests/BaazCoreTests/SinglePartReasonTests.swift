import XCTest
@testable import BaazCore

/// A YouTube download is always one part because video downloads are fetched
/// whole, not because the site refused to split it. Saying the latter sent
/// the user hunting for a fault that was not there — and made the "parts"
/// setting look broken when it was working.
final class SinglePartReasonTests: XCTestCase {
    private func job(kind: String, noRange: Bool, parts: Int) throws -> Job {
        let segs = (0..<parts).map { _ in #"{"done":1,"total":10}"# }.joined(separator: ",")
        let json = #"""
        [{"id":"a","name":"x","state":"active","total":100,"done":10,
          "kind":"\#(kind)","noRange":\#(noRange),"segments":[\#(segs)]}]
        """#
        return try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
    }

    func testMediaIsNotBlamedOnTheServer() throws {
        let j = try job(kind: "media", noRange: false, parts: 1)
        XCTAssertTrue(j.isMedia)
        XCTAssertTrue(j.singlePartReason.contains("video sites"), j.singlePartReason)
        XCTAssertFalse(j.singlePartReason.contains("allow splitting"),
                       "a video download must not be blamed on the site: " + j.singlePartReason)
        XCTAssertTrue(j.singlePartDetail.contains("video sites"), j.singlePartDetail)
    }

    func testServerWithoutRangeIsNamedAsSuch() throws {
        let j = try job(kind: "", noRange: true, parts: 1)
        XCTAssertTrue(j.singlePartReason.contains("doesn't allow splitting"), j.singlePartReason)
    }

    /// Ranged, but the file was under the minimum split size.
    func testRangedButTooSmallSaysSo() throws {
        let j = try job(kind: "", noRange: false, parts: 1)
        XCTAssertTrue(j.singlePartReason.contains("too small"), j.singlePartReason)
        XCTAssertFalse(j.singlePartReason.contains("allow splitting"),
                       "a small file must not be blamed on the site: " + j.singlePartReason)
    }

    func testASplitDownloadIsStillReportedAsSplit() throws {
        XCTAssertTrue(try job(kind: "", noRange: false, parts: 9).isSplit)
        XCTAssertFalse(try job(kind: "", noRange: false, parts: 1).isSplit)
    }

    /// Every branch must say something; an empty explanation is a blank line
    /// under the progress bar.
    func testEveryReasonIsNonEmpty() throws {
        for (kind, noRange) in [("media", false), ("", true), ("", false)] {
            let j = try job(kind: kind, noRange: noRange, parts: 1)
            XCTAssertFalse(j.singlePartReason.isEmpty)
            XCTAssertFalse(j.singlePartDetail.isEmpty)
        }
    }
}

/// A one-part download read "1 parts".
final class PartsLabelTests: XCTestCase {
    private func job(parts: Int) throws -> Job {
        let segs = (0..<parts).map { _ in #"{"done":1,"total":10}"# }.joined(separator: ",")
        let json = #"[{"id":"a","name":"x","state":"active","segments":[\#(segs)]}]"#
        return try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
    }

    func testOnePartIsSingular() throws {
        XCTAssertEqual(try job(parts: 1).partsLabel, "1 part")
    }

    func testSeveralPartsArePlural() throws {
        XCTAssertEqual(try job(parts: 8).partsLabel, "8 parts")
        XCTAssertEqual(try job(parts: 2).partsLabel, "2 parts")
    }
}
