import SwiftUI

/// Liquid Glass, applied where Baaz draws its own chrome.
///
/// Standard controls — the window, its toolbar, buttons, toggles — get the
/// new appearance for free once the app is built against the macOS 26 SDK;
/// there is nothing to call for those. These helpers are for the surfaces
/// Baaz fills itself, which the system cannot know about: the download cards
/// and the menu bar panel's own strips.
///
/// Every one falls back to the material that was there before, so nothing
/// changes on macOS 13 through 25. The deployment target stays at 13.
public extension View {
    /// A glass surface in the given shape, or the previous translucent fill.
    @ViewBuilder
    func baazGlass(
        in shape: some Shape,
        tinted: Bool = false,
        fallback: Color = Color.primary.opacity(0.03)
    ) -> some View {
        if #available(macOS 26.0, *) {
            // .clear lets the desktop through more than .regular, which is
            // too much behind a list of text; regular keeps rows readable.
            glassEffect(tinted ? .regular.tint(.accentColor.opacity(0.28)) : .regular,
                        in: shape)
        } else {
            background(shape.fill(fallback))
        }
    }

    /// Groups adjacent glass surfaces so they bleed into one another rather
    /// than stacking as separate panes. A no-op before macOS 26.
    @ViewBuilder
    func baazGlassGroup(id: some Hashable & Sendable, in namespace: Namespace.ID)
        -> some View
    {
        if #available(macOS 26.0, *) {
            glassEffectID(id, in: namespace)
        } else {
            self
        }
    }
}

/// Wraps a set of glass surfaces so they can merge. Renders its content
/// unchanged before macOS 26.
public struct BaazGlassContainer<Content: View>: View {
    private let spacing: CGFloat
    private let content: Content

    public init(spacing: CGFloat = 12, @ViewBuilder content: () -> Content) {
        self.spacing = spacing
        self.content = content()
    }

    public var body: some View {
        if #available(macOS 26.0, *) {
            GlassEffectContainer(spacing: spacing) { content }
        } else {
            content
        }
    }
}
