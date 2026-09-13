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
    targets: [
        .target(name: "BaazCore", path: "Sources/BaazCore"),
        .executableTarget(
            name: "BaazMenuBar",
            dependencies: ["BaazCore"],
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
