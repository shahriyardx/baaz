// Renders the DMG window background: a title, the two icon slots the Finder
// layout drops icons into, and an arrow between them.
//
// Run it when the design changes; the generated background.tiff is committed
// so building a DMG needs no rendering step (and no window server) of its own:
//
//   swift macos/dmg/render-background.swift macos/dmg
//
// Coordinates below are top-left origin — the context is flipped, so they
// read the same way as the Finder icon positions in make-dmg.sh.

import AppKit
import Foundation

let W = 640.0, H = 400.0

// Icon slot centres. make-dmg.sh must place the icons on these exact points.
let leftSlot = NSPoint(x: 160, y: 205)
let rightSlot = NSPoint(x: 480, y: 205)

func render(scale: CGFloat) -> Data {
    let pw = Int(W * scale), ph = Int(H * scale)
    guard let rep = NSBitmapImageRep(
        bitmapDataPlanes: nil, pixelsWide: pw, pixelsHigh: ph,
        bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false,
        colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)
    else { fatalError("could not allocate the bitmap") }
    rep.size = NSSize(width: W, height: H)

    guard let base = NSGraphicsContext(bitmapImageRep: rep) else {
        fatalError("could not make a drawing context")
    }
    NSGraphicsContext.saveGraphicsState()
    let cg = base.cgContext

    // Flip to top-left origin so these coordinates match the Finder's. The
    // context has to be rewrapped as flipped, not just transformed: AppKit
    // text drawing reads isFlipped, not the CTM, and renders upside down
    // otherwise.
    cg.translateBy(x: 0, y: CGFloat(H))
    cg.scaleBy(x: 1, y: -1)
    NSGraphicsContext.current = NSGraphicsContext(cgContext: cg, flipped: true)

    // A barely-there vertical gradient: enough to keep the window from
    // looking like a blank sheet, not enough to compete with the icons.
    let grad = NSGradient(colors: [
        NSColor(srgbRed: 0.988, green: 0.988, blue: 0.992, alpha: 1),
        NSColor(srgbRed: 0.937, green: 0.941, blue: 0.953, alpha: 1),
    ])!
    grad.draw(in: NSRect(x: 0, y: 0, width: W, height: H), angle: -90)

    func text(_ s: String, _ font: NSFont, _ color: NSColor, centerX: CGFloat, y: CGFloat) {
        let style = NSMutableParagraphStyle()
        style.alignment = .center
        let attrs: [NSAttributedString.Key: Any] =
            [.font: font, .foregroundColor: color, .paragraphStyle: style]
        let size = (s as NSString).size(withAttributes: attrs)
        (s as NSString).draw(in: NSRect(x: centerX - size.width / 2, y: y,
                                        width: size.width, height: size.height),
                             withAttributes: attrs)
    }

    let ink = NSColor(srgbRed: 0.11, green: 0.12, blue: 0.15, alpha: 1)
    let muted = NSColor(srgbRed: 0.42, green: 0.45, blue: 0.50, alpha: 1)
    let faint = NSColor(srgbRed: 0.60, green: 0.63, blue: 0.68, alpha: 1)

    text("Baaz", .systemFont(ofSize: 26, weight: .semibold), ink, centerX: W / 2, y: 44)
    text("Drag the app onto Applications to install",
         .systemFont(ofSize: 13, weight: .regular), muted, centerX: W / 2, y: 80)

    // The arrow: a shaft and a solid head, centred between the two slots.
    // It sits on the slot centre line, which is where Finder draws the icon
    // artwork — measured against a mounted volume, not assumed.
    let y = leftSlot.y
    let x0 = leftSlot.x + 108, x1 = rightSlot.x - 108
    let headLen = 20.0, headHalf = 9.0
    ink.withAlphaComponent(0.55).setStroke()
    let shaft = NSBezierPath()
    shaft.move(to: NSPoint(x: x0, y: y))
    shaft.line(to: NSPoint(x: x1 - headLen + 4, y: y))
    shaft.lineWidth = 3
    shaft.lineCapStyle = .round
    shaft.stroke()
    ink.withAlphaComponent(0.55).setFill()
    let head = NSBezierPath()
    head.move(to: NSPoint(x: x1, y: y))
    head.line(to: NSPoint(x: x1 - headLen, y: y - headHalf))
    head.line(to: NSPoint(x: x1 - headLen, y: y + headHalf))
    head.close()
    head.fill()

    text("First launch: right-click Baaz and choose Open.",
         .systemFont(ofSize: 11, weight: .regular), faint, centerX: W / 2, y: H - 52)

    NSGraphicsContext.restoreGraphicsState()
    guard let data = rep.representation(using: .png, properties: [:]) else {
        fatalError("could not encode the PNG")
    }
    return data
}

let outDir = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : "."
for (scale, name) in [(1.0, "background.png"), (2.0, "background@2x.png")] {
    let url = URL(fileURLWithPath: outDir).appendingPathComponent(name)
    try! render(scale: CGFloat(scale)).write(to: url)
    print("wrote \(url.path)")
}
