import XCTest
@testable import BaazCore

/// The Add sheet prefills from the clipboard and offers quality options for
/// video pages. Both decisions come from the link alone.
final class AddDownloadTests: XCTestCase {
    func testAcceptsRealLinks() {
        for s in ["https://example.com/file.zip",
                  "http://example.com/a/b?c=d#e",
                  "  https://example.com/x  ",              // trimmed
                  "https://sub.domain.co.uk/f.iso"] {
            XCTAssertNotNil(AddDownloadSheet.normalizedURL(s), "should accept \(s)")
        }
    }

    /// Anything else on the clipboard must not be offered as a download.
    func testRejectsNonLinks() {
        for s in ["", "   ", "not a url", "example.com/file.zip",   // no scheme
                  "ftp://example.com/f", "file:///etc/passwd",
                  "javascript:alert(1)", "https://", "https://localhost",
                  String(repeating: "h", count: 5000)] {
            XCTAssertNil(AddDownloadSheet.normalizedURL(s), "should reject \(s.prefix(20))")
        }
    }

    func testDetectsMediaHosts() {
        for s in ["https://www.youtube.com/watch?v=abc",
                  "https://youtu.be/abc",
                  "https://m.youtube.com/watch?v=abc",
                  "https://vimeo.com/123",
                  "https://www.facebook.com/watch?v=1"] {
            XCTAssertTrue(AddDownloadSheet.looksLikeMediaHost(s), "\(s) is a media page")
        }
    }

    /// A direct file link must not be offered quality options — it is not
    /// going through yt-dlp.
    func testDirectLinksAreNotMedia() {
        for s in ["https://example.com/video.mp4",
                  "https://github.com/x/y/releases/download/v1/file.zip",
                  "https://notyoutube.com/watch",
                  "https://youtube.com.evil.test/watch"] {
            XCTAssertFalse(AddDownloadSheet.looksLikeMediaHost(s), "\(s) is a direct link")
        }
    }
}
