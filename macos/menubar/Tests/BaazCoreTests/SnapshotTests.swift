import XCTest
@testable import BaazCore

// snapshot.json is real `baaz status --json` output, not a hand-written
// fixture. internal/ipc/protocol.go owns this schema; these tests are what
// catch the Swift side drifting away from it.
final class SnapshotTests: XCTestCase {
    private func fixture() throws -> Data {
        let url = try XCTUnwrap(Bundle.module.url(forResource: "snapshot", withExtension: "json"))
        return try Data(contentsOf: url)
    }

    func testDecodesRealDaemonOutput() throws {
        let snap = try JSONDecoder().decode(Snapshot.self, from: fixture())
        XCTAssertEqual(snap.type, "snapshot")
        XCTAssertFalse(snap.recent.isEmpty, "fixture should carry a finished job")
        let done = try XCTUnwrap(snap.recent.first)
        XCTAssertFalse(done.id.isEmpty)
        XCTAssertFalse(done.name.isEmpty)
        XCTAssertFalse(done.dir.isEmpty)
        XCTAssertGreaterThan(done.total, 0)
        XCTAssertEqual(snap.settings.segments, 8)
        XCTAssertTrue(snap.settings.intercept)
    }

    // Go marshals a nil slice as null and omits empty strings, so no field
    // may be required. A snapshot that fails to decode would freeze the UI on
    // stale data with no visible error.
    func testDecodesNullSlicesAndMissingFields() throws {
        let json = Data("""
        {"type":"snapshot","active":0,"totalSpeed":0,"jobs":null,"recent":null}
        """.utf8)
        let snap = try JSONDecoder().decode(Snapshot.self, from: json)
        XCTAssertEqual(snap.jobs, [])
        XCTAssertEqual(snap.recent, [])
        XCTAssertTrue(snap.settings.intercept, "absent settings must not read as intercept off")
    }

    func testUnknownFieldsAreIgnored() throws {
        let json = Data(#"{"type":"snapshot","somethingNew":42,"jobs":[]}"#.utf8)
        XCTAssertNoThrow(try JSONDecoder().decode(Snapshot.self, from: json))
    }

    // A job with no advertised size must not render as a full bar.
    func testFractionIsNilWithoutTotal() throws {
        let json = Data(#"[{"id":"a","name":"x","state":"active","total":0,"done":900}]"#.utf8)
        let jobs = try JSONDecoder().decode([Job].self, from: json)
        XCTAssertNil(jobs[0].fraction)
        XCTAssertTrue(jobs[0].caption.contains("size unknown"))
    }

    func testFractionClampsToOne() throws {
        let json = Data(#"[{"id":"a","name":"x","state":"active","total":100,"done":150}]"#.utf8)
        let jobs = try JSONDecoder().decode([Job].self, from: json)
        XCTAssertEqual(jobs[0].fraction, 1)
    }

    func testOverallPercentIgnoresUnsizedJobs() throws {
        let json = Data("""
        {"type":"snapshot","jobs":[
          {"id":"a","name":"a","state":"active","total":100,"done":50},
          {"id":"b","name":"b","state":"active","total":0,"done":999},
          {"id":"c","name":"c","state":"paused","total":100,"done":100}
        ]}
        """.utf8)
        let snap = try JSONDecoder().decode(Snapshot.self, from: json)
        XCTAssertEqual(snap.overallPercent, 50, "only sized active jobs count")
    }

    func testHumanSizes() {
        XCTAssertEqual(human(512), "512B")
        XCTAssertEqual(human(2048), "2KB")
        XCTAssertEqual(human(5 << 20), "5.0MB")
        XCTAssertEqual(human(3 << 30), "3.0GB")
    }
}
