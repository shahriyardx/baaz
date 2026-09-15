// Renders the README hero banner.
//
//   swift docs/render-banner.swift build/Baaz.app/Contents/Resources/Baaz.icns docs/images
//
// Dark, so it sits well on GitHub's default theme, and built around the
// falcon the app is named for (baaz — বাজ — is Bengali for falcon). The
// streaks behind it are the point of the app: one file, several pieces,
// arriving at once.

import AppKit
import Foundation

let W = 1280.0, H = 440.0

let iconPath = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : ""
let outDir = CommandLine.arguments.count > 2 ? CommandLine.arguments[2] : "."

func render(scale: CGFloat) -> Data {
    guard let rep = NSBitmapImageRep(
        bitmapDataPlanes: nil, pixelsWide: Int(W * scale), pixelsHigh: Int(H * scale),
        bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false,
        colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)
    else { fatalError("could not allocate the bitmap") }
    rep.size = NSSize(width: W, height: H)

    guard let base = NSGraphicsContext(bitmapImageRep: rep) else {
        fatalError("could not make a drawing context")
    }
    NSGraphicsContext.saveGraphicsState()
    let cg = base.cgContext
    cg.translateBy(x: 0, y: CGFloat(H))
    cg.scaleBy(x: 1, y: -1)
    NSGraphicsContext.current = NSGraphicsContext(cgContext: cg, flipped: true)

    // Ground: a near-black that matches the app's own chrome.
    let bg = NSGradient(colors: [
        NSColor(srgbRed: 0.055, green: 0.067, blue: 0.094, alpha: 1),
        NSColor(srgbRed: 0.086, green: 0.102, blue: 0.145, alpha: 1),
    ])!
    bg.draw(in: NSRect(x: 0, y: 0, width: W, height: H), angle: -90)

    let accent = NSColor(srgbRed: 0.35, green: 0.55, blue: 1.0, alpha: 1)

    // Parallel streaks either side of the falcon, fading outward: the point
    // of the app, which is one file arriving as several pieces at once.
    let midY = 168.0
    for (i, off) in [0.0, 34.0, -34.0, 68.0, -68.0].enumerated() {
        let alpha = [0.55, 0.38, 0.38, 0.20, 0.20][i]
        let len = [340.0, 280.0, 280.0, 200.0, 200.0][i]
        let y = midY + off
        let leftEnd = W / 2 - 110
        let gl = NSGradient(colors: [accent.withAlphaComponent(0),
                                     accent.withAlphaComponent(alpha)])!
        gl.draw(in: NSRect(x: leftEnd - len, y: y - 1, width: len, height: 2), angle: 0)
        let rightStart = W / 2 + 110
        let gr = NSGradient(colors: [accent.withAlphaComponent(alpha),
                                     accent.withAlphaComponent(0)])!
        gr.draw(in: NSRect(x: rightStart, y: y - 1, width: len, height: 2), angle: 0)
    }

    // The falcon, with a soft round halo so it lifts off the background.
    let side = 132.0
    let iconBox = NSRect(x: (W - side) / 2, y: midY - side / 2, width: side, height: side)
    if let icon = NSImage(contentsOfFile: iconPath) {
        let haloBox = iconBox.insetBy(dx: -150, dy: -150)
        NSGraphicsContext.current?.saveGraphicsState()
        NSBezierPath(ovalIn: haloBox).addClip()
        NSGradient(colors: [accent.withAlphaComponent(0.16),
                            accent.withAlphaComponent(0.05),
                            accent.withAlphaComponent(0)])!
            .draw(in: haloBox, relativeCenterPosition: .zero)
        NSGraphicsContext.current?.restoreGraphicsState()

        // The context is already flipped for top-left coordinates, and
        // NSImage flips again on its own, so the icon lands upside down.
        // Mirror it back about its own centre.
        NSGraphicsContext.current?.saveGraphicsState()
        let t = NSAffineTransform()
        t.translateX(by: 0, yBy: iconBox.midY * 2)
        t.scaleX(by: 1, yBy: -1)
        t.concat()
        icon.draw(in: iconBox, from: .zero, operation: .sourceOver, fraction: 1)
        NSGraphicsContext.current?.restoreGraphicsState()
    }

    func text(_ s: String, _ font: NSFont, _ color: NSColor, x: CGFloat, y: CGFloat) {
        let attrs: [NSAttributedString.Key: Any] = [.font: font, .foregroundColor: color]
        (s as NSString).draw(at: NSPoint(x: x, y: y), withAttributes: attrs)
    }
    func centered(_ s: String, _ font: NSFont, _ color: NSColor, y: CGFloat) {
        let style = NSMutableParagraphStyle()
        style.alignment = .center
        let attrs: [NSAttributedString.Key: Any] =
            [.font: font, .foregroundColor: color, .paragraphStyle: style]
        (s as NSString).draw(in: NSRect(x: 0, y: y, width: W, height: 90), withAttributes: attrs)
    }

    centered("Baaz", .systemFont(ofSize: 58, weight: .bold), .white, y: 268)
    centered("Downloads that arrive in a fraction of the time",
             .systemFont(ofSize: 20, weight: .regular),
             NSColor(srgbRed: 0.60, green: 0.66, blue: 0.76, alpha: 1), y: 348)

    _ = text
    NSGraphicsContext.restoreGraphicsState()
    guard let data = rep.representation(using: .png, properties: [:]) else {
        fatalError("could not encode the PNG")
    }
    return data
}

for (scale, name) in [(1.0, "banner.png"), (2.0, "banner@2x.png")] {
    let url = URL(fileURLWithPath: outDir).appendingPathComponent(name)
    try! render(scale: CGFloat(scale)).write(to: url)
    print("wrote \(url.path)")
}
