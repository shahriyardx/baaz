import AppKit
import SwiftUI
import UniformTypeIdentifiers
import XCTest
@testable import BaazCore

/// Dragging a finished download out of Baaz and into a chat or an upload
/// field. What decides whether a row can be dragged is worth pinning down:
/// handing over a half-written file would send someone a truncated video.
@MainActor
final class DragOutTests: XCTestCase {
    private var dir: URL!

    override func setUpWithError() throws {
        dir = URL(fileURLWithPath: NSTemporaryDirectory())
            .appendingPathComponent("baaz-drag-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
    }

    override func tearDownWithError() throws {
        try? FileManager.default.removeItem(at: dir)
    }

    private func job(state: String, name: String, onDisk: Bool) throws -> Job {
        if onDisk {
            try Data("hello".utf8).write(to: dir.appendingPathComponent(name))
        }
        let json = #"[{"id":"a","name":"\#(name)","state":"\#(state)","dir":"\#(dir.path)","total":5,"done":5}]"#
        return try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
    }

    func testAFinishedDownloadCanBeDragged() throws {
        let j = try job(state: "done", name: "clip.mp4", onDisk: true)
        XCTAssertNotNil(j.draggableFileURL)
        XCTAssertEqual(j.draggableFileURL?.lastPathComponent, "clip.mp4")
    }

    /// A download in flight has only a partial file. Dropping that into a
    /// chat would send something truncated, so it must not be draggable.
    func testAnUnfinishedDownloadCannotBeDragged() throws {
        for state in ["active", "queued", "paused", "failed"] {
            let j = try job(state: state, name: "\(state).bin", onDisk: true)
            XCTAssertNil(j.draggableFileURL, "\(state) should not be draggable")
        }
    }

    /// Finished, but the file has since been moved or deleted — dragging it
    /// would hand over something that resolves to nothing at the far end.
    func testAMissingFileCannotBeDragged() throws {
        let j = try job(state: "done", name: "gone.zip", onDisk: false)
        XCTAssertNil(j.draggableFileURL)
    }

    func testAJobWithNoFolderCannotBeDragged() throws {
        let json = #"[{"id":"a","name":"x.zip","state":"done"}]"#
        let j = try JSONDecoder().decode([Job].self, from: Data(json.utf8))[0]
        XCTAssertNil(j.draggableFileURL)
    }

    /// The receiving app has to be handed the file itself, with its type —
    /// not a URL string, which an upload control has no use for.
    func testTheItemProviderCarriesTheRealFile() throws {
        let j = try job(state: "done", name: "clip.mp4", onDisk: true)
        let url = try XCTUnwrap(j.draggableFileURL)
        let provider = try XCTUnwrap(NSItemProvider(contentsOf: url))

        let types = provider.registeredTypeIdentifiers
        XCTAssertFalse(types.isEmpty, "nothing registered, so nothing would drop")
        XCTAssertTrue(types.contains(UTType.mpeg4Movie.identifier)
                        || types.contains(UTType.fileURL.identifier)
                        || types.contains(UTType.data.identifier),
                      "expected a file type, got \(types)")
    }

    /// The type follows the file, so a pdf arrives as a pdf.
    func testTheTypeFollowsTheFile() throws {
        let j = try job(state: "done", name: "paper.pdf", onDisk: true)
        let url = try XCTUnwrap(j.draggableFileURL)
        let provider = try XCTUnwrap(NSItemProvider(contentsOf: url))
        XCTAssertTrue(provider.registeredTypeIdentifiers.contains(UTType.pdf.identifier),
                      "got \(provider.registeredTypeIdentifiers)")
    }

    func testTheDragPreviewRenders() throws {
        let j = try job(state: "done", name: "clip.mp4", onDisk: true)
        let url = try XCTUnwrap(j.draggableFileURL)
        let host = NSHostingView(rootView: AnyView(DragPreview(url: url, name: j.name)))
        host.layoutSubtreeIfNeeded()
        XCTAssertGreaterThan(host.fittingSize.width, 0)
        XCTAssertGreaterThan(host.fittingSize.height, 0)
    }
}
