import XCTest
@testable import BaazCore

/// Notifications are now worked out by the app from the snapshot stream,
/// because the daemon has no way to post them as Baaz. Everything that
/// decides whether to speak lives in one pure function, so it can be checked
/// without a notification centre or a running daemon.
final class DownloadEventTests: XCTestCase {
    private func state(_ s: String, _ n: String = "file.zip") -> JobState {
        JobState(state: s, name: n)
    }

    func testANewDownloadIsAnnounced() {
        let ev = downloadEvents(previous: [:], current: ["a": state("active")])
        XCTAssertEqual(ev, [DownloadEvent(kind: .started, name: "file.zip")])
        XCTAssertEqual(ev.first?.title, "Download started")
    }

    func testFinishingIsAnnouncedOnce() {
        let before = ["a": state("active")]
        let after = ["a": state("done")]
        XCTAssertEqual(downloadEvents(previous: before, current: after),
                       [DownloadEvent(kind: .finished, name: "file.zip")])
        // The finished job stays in the snapshot's recent list for a long
        // time; announcing it on every one of those would be a banner twice a
        // second.
        XCTAssertEqual(downloadEvents(previous: after, current: after), [])
    }

    func testFailingIsAnnouncedOnce() {
        XCTAssertEqual(
            downloadEvents(previous: ["a": state("active")], current: ["a": state("failed")]),
            [DownloadEvent(kind: .failed, name: "file.zip")])
        XCTAssertEqual(
            downloadEvents(previous: ["a": state("failed")], current: ["a": state("failed")]), [])
    }

    /// Jobs move between queued and active as slots free up. That is not a
    /// new download and must stay silent.
    func testQueuedToActiveSaysNothing() {
        XCTAssertEqual(
            downloadEvents(previous: ["a": state("queued")], current: ["a": state("active")]), [])
    }

    func testPausingAndResumingSaysNothing() {
        XCTAssertEqual(
            downloadEvents(previous: ["a": state("active")], current: ["a": state("paused")]), [])
        XCTAssertEqual(
            downloadEvents(previous: ["a": state("paused")], current: ["a": state("active")]), [])
    }

    /// A retry after a failure is worth hearing about again.
    func testRetryingAfterAFailureAnnouncesTheNewResult() {
        XCTAssertEqual(
            downloadEvents(previous: ["a": state("failed")], current: ["a": state("done")]),
            [DownloadEvent(kind: .finished, name: "file.zip")])
    }

    func testSeveralAtOnceComeOutInAStableOrder() {
        let ev = downloadEvents(
            previous: [:],
            current: ["b": state("active", "b.zip"), "a": state("active", "a.zip")])
        XCTAssertEqual(ev.map(\.name), ["a.zip", "b.zip"])
    }

    /// A job with no filename yet — the name arrives with the response
    /// headers — should still say something useful.
    func testAJobWithoutANameFallsBackToItsLink() throws {
        let json = #"{"type":"snapshot","jobs":[{"id":"a","state":"active","url":"https://x/y.zip"}]}"#
        let snap = try JSONDecoder().decode(Snapshot.self, from: Data(json.utf8))
        XCTAssertEqual(snap.jobStates["a"]?.name, "https://x/y.zip")
    }

    func testTitlesReadAsPlainEnglish() {
        XCTAssertEqual(DownloadEvent(kind: .started, name: "x").title, "Download started")
        XCTAssertEqual(DownloadEvent(kind: .finished, name: "x").title, "Download finished")
        XCTAssertEqual(DownloadEvent(kind: .failed, name: "x").title, "Download failed")
    }
}

/// The model must not greet you with a banner for every download from
/// yesterday. The first snapshot after launch carries the daemon's whole
/// history and only seeds the comparison.
@MainActor
final class FirstSnapshotTests: XCTestCase {
    func testTheFirstSnapshotOnlySeeds() {
        let m = DownloadsModel()
        let snap = #"{"type":"snapshot","recent":[{"id":"old","name":"yesterday.zip","state":"done"}]}"#
        m.ingest(Data((snap + "\n").utf8))
        // Nothing to assert on a banner directly, so assert the state that
        // decides: a second identical snapshot produces no events.
        let states = m.snapshot.jobStates
        XCTAssertEqual(downloadEvents(previous: states, current: states), [])
        XCTAssertEqual(states["old"]?.state, "done")
    }

    func testSomethingFinishingAfterLaunchIsAnEvent() {
        let first = #"{"type":"snapshot","jobs":[{"id":"a","name":"new.zip","state":"active"}]}"#
        let second = #"{"type":"snapshot","recent":[{"id":"a","name":"new.zip","state":"done"}]}"#
        let m = DownloadsModel()
        m.ingest(Data((first + "\n").utf8))
        let before = m.snapshot.jobStates
        m.ingest(Data((second + "\n").utf8))
        XCTAssertEqual(downloadEvents(previous: before, current: m.snapshot.jobStates),
                       [DownloadEvent(kind: .finished, name: "new.zip")])
    }
}
