import AppKit
import WebKit

MainActor.assumeIsolated {
let app = NSApplication.shared
app.setActivationPolicy(.accessory)
let dir = FileManager.default.currentDirectoryPath

for name in ["scene1", "scene2", "scene3"] {
    let cfg = WKWebViewConfiguration()
    // 1280x800 is the Web Store's preferred screenshot size.
    let web = WKWebView(frame: NSRect(x: 0, y: 0, width: 1280, height: 800), configuration: cfg)
    web.appearance = NSAppearance(named: .darkAqua)
    web.setValue(false, forKey: "drawsBackground")
    let url = URL(fileURLWithPath: "\(dir)/\(name).html")
    web.loadFileURL(url, allowingReadAccessTo: URL(fileURLWithPath: dir))
    RunLoop.current.run(until: Date().addingTimeInterval(2.2))

    let snap = WKSnapshotConfiguration()
    snap.rect = NSRect(x: 0, y: 0, width: 1280, height: 800)
    snap.snapshotWidth = 1280
    let sem = DispatchSemaphore(value: 0)
    web.takeSnapshot(with: snap) { img, err in
        defer { sem.signal() }
        guard let img, let tiff = img.tiffRepresentation,
              let rep = NSBitmapImageRep(data: tiff) else {
            print("  \(name): FAILED \(err?.localizedDescription ?? "")"); return
        }
        rep.size = NSSize(width: 1280, height: 800)
        if let png = rep.representation(using: .png, properties: [:]) {
            try? png.write(to: URL(fileURLWithPath: "\(dir)/baaz-\(name).png"))
            print("  \(name): \(rep.pixelsWide)x\(rep.pixelsHigh), \(png.count/1024)KB")
        }
    }
    while sem.wait(timeout: .now()) == .timedOut {
        RunLoop.current.run(until: Date().addingTimeInterval(0.1))
    }
}
}
