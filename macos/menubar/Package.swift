// swift-tools-version: 5.9
import PackageDescription

// The menu bar UI for baaz.
//
// The decodable model lives in its own BaazCore target so the tests can
// import it: an executable target with @main cannot be linked into a test
// bundle. `make macos-app` wraps the executable in Baaz.app with the
// Info.plist whose LSUIElement keeps it out of the Dock.
let package = Package(
    name: "BaazMenuBar",
    platforms: [.macOS(.v13)],
    dependencies: [
        // Sparkle delivers in-app updates. It checks an appcast, verifies the
        // archive's EdDSA signature, replaces the bundle and relaunches —
        // none of which needs an Apple Developer ID.
        .package(url: "https://github.com/sparkle-project/Sparkle", from: "2.6.0"),
    ],
    targets: [
        .target(name: "BaazCore", path: "Sources/BaazCore"),
        .executableTarget(
            name: "BaazMenuBar",
            dependencies: [
                "BaazCore",
                .product(name: "Sparkle", package: "Sparkle"),
            ],
            path: "Sources/BaazMenuBar"
        ),
        .testTarget(
            name: "BaazCoreTests",
            dependencies: ["BaazCore"],
            path: "Tests/BaazCoreTests",
            resources: [.copy("snapshot.json")]
        ),
    ]
)
