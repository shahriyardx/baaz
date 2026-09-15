import AppKit

/// The baaz falcon, drawn as a menu bar glyph.
///
/// The paths are the same ones in `extension/icons/icon.svg` (a 128×128
/// viewBox), minus the rounded-square backdrop — a menu bar icon is a
/// silhouette, not a tile. Drawing it as a vector rather than shipping a PNG
/// keeps it crisp on every display and keeps the artwork in one place.
public enum FalconIcon {
    /// Template image for the status item. `isTemplate` is what makes macOS
    /// tint it: white on a dark menu bar, black on a light one, and inverted
    /// while the menu is open. Do not bake a color in.
    public static let menuBar: NSImage = {
        let image = render(height: 16)
        image.isTemplate = true
        image.accessibilityDescription = "Baaz"
        return image
    }()

    /// Fits the falcon into an image of the given height, preserving aspect.
    static func render(height: CGFloat) -> NSImage {
        let glyph = path()
        let bounds = glyph.bounds
        let scale = height / bounds.height
        let size = NSSize(width: (bounds.width * scale).rounded(), height: height)

        // Bake the fit into the path rather than juggling the CTM: the
        // translation belongs in unscaled user space, and applying it to an
        // already-scaled context silently scales it twice.
        let fit = NSAffineTransform()
        fit.scale(by: scale)
        fit.translateX(by: -bounds.minX, yBy: -bounds.minY)
        let fitted = fit.transform(glyph)

        return NSImage(size: size, flipped: true) { _ in
            NSColor.black.setFill() // alpha is all a template image keeps
            fitted.fill()
            return true
        }
    }

    /// The fitted path, exposed so tests can check it fills its box.
    static func fittedBounds(height: CGFloat) -> NSRect {
        let glyph = path()
        let b = glyph.bounds
        let scale = height / b.height
        let fit = NSAffineTransform()
        fit.scale(by: scale)
        fit.translateX(by: -b.minX, yBy: -b.minY)
        return fit.transform(glyph).bounds
    }

    /// The falcon in SVG user space: two swept wings, a body tapering to a
    /// point, and the bar it dives toward.
    private static func path() -> NSBezierPath {
        let p = NSBezierPath()

        // Left wing.
        p.move(to: CGPoint(x: 62, y: 34))
        p.curve(to: CGPoint(x: 20, y: 42),
                controlPoint1: CGPoint(x: 48, y: 42),
                controlPoint2: CGPoint(x: 34, y: 45))
        p.curve(to: CGPoint(x: 59, y: 66),
                controlPoint1: CGPoint(x: 30, y: 57),
                controlPoint2: CGPoint(x: 47, y: 64))
        p.line(to: CGPoint(x: 62, y: 58))
        p.close()

        // Right wing.
        p.move(to: CGPoint(x: 66, y: 34))
        p.curve(to: CGPoint(x: 108, y: 42),
                controlPoint1: CGPoint(x: 80, y: 42),
                controlPoint2: CGPoint(x: 94, y: 45))
        p.curve(to: CGPoint(x: 69, y: 66),
                controlPoint1: CGPoint(x: 98, y: 57),
                controlPoint2: CGPoint(x: 81, y: 64))
        p.line(to: CGPoint(x: 66, y: 58))
        p.close()

        // Body.
        p.move(to: CGPoint(x: 64, y: 30))
        p.line(to: CGPoint(x: 73, y: 62))
        p.line(to: CGPoint(x: 64, y: 102))
        p.line(to: CGPoint(x: 55, y: 62))
        p.close()

        // The bar it lands on, which doubles as a download tray.
        p.appendRoundedRect(NSRect(x: 38, y: 106, width: 52, height: 7),
                            xRadius: 3.5, yRadius: 3.5)
        return p
    }
}
