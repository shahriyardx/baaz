import AppKit
import SwiftUI

/// Dragging a finished download straight out of Baaz and into something else
/// — a chat window, an upload field, a Finder folder — the way Chrome's own
/// download bar works.
///
/// The receiving app is handed the real file, not a path or a URL string, so
/// anything that accepts a file drop takes it: Messages, WhatsApp, a browser
/// upload control, Mail, Finder.
///
/// Dragging out of the menu bar panel works, which was not a given: that
/// panel is a non-activating NSPanel and dismisses when it loses focus, so
/// the drag could have been cancelled the moment it began. Confirmed by
/// hand, since nothing about it is reachable from a test — worth knowing
/// before anyone reaches for an NSWindow-level workaround that is not
/// needed.
extension View {
    /// Makes this row draggable when the download has actually finished and
    /// the file is where the daemon said it is.
    ///
    /// Both checks matter. A job in flight has only a .part file under a
    /// hidden directory, and dropping that into a chat would send something
    /// truncated; a finished job whose file has since been moved or deleted
    /// would drag an item that resolves to nothing at the far end.
    @ViewBuilder
    func draggableDownload(_ job: Job) -> some View {
        if let url = job.draggableFileURL {
            onDrag {
                // contentsOf registers the file itself, with its type, which
                // is what makes a real drop possible. NSItemProvider(object:
                // url as NSURL) would hand over only a URL, and an upload
                // field has nothing to do with that.
                NSItemProvider(contentsOf: url) ?? NSItemProvider()
            } preview: {
                DragPreview(url: url, name: job.name)
            }
        } else {
            self
        }
    }
}

/// What follows the pointer during the drag: the file's own icon and name,
/// so it is obvious which download is being carried.
struct DragPreview: View {
    let url: URL
    let name: String

    var body: some View {
        HStack(spacing: 6) {
            Image(nsImage: NSWorkspace.shared.icon(forFile: url.path))
                .resizable()
                .frame(width: 22, height: 22)
            Text(name)
                .font(.callout)
                .lineLimit(1)
                .truncationMode(.middle)
        }
        .padding(.horizontal, 8)
        .padding(.vertical, 5)
        .background(.regularMaterial, in: RoundedRectangle(cornerRadius: 7))
    }
}

extension Job {
    /// The file to hand over, or nil if there is nothing safe to drag.
    var draggableFileURL: URL? {
        guard state == "done", let url = fileURL,
              FileManager.default.fileExists(atPath: url.path)
        else { return nil }
        return url
    }
}
