import AppKit
import SwiftUI

/// Add a download by link.
///
/// The URL is prefilled from the clipboard when it looks like one, since
/// copying a link and reaching for "add" is the usual order of events. The
/// daemon does the real detection — filename, size, whether the host needs
/// yt-dlp — so this only has to collect the link.
public struct AddDownloadSheet: View {
    @EnvironmentObject var model: DownloadsModel
    @Environment(\.dismiss) private var dismiss

    @State private var url = ""
    @State private var filename = ""
    @State private var format = "best"
    @FocusState private var urlFocused: Bool

    public init() {}

    /// Media hosts get quality options; a direct file link has none.
    private var isMedia: Bool { Self.looksLikeMediaHost(url) }

    public var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("Add Download").font(.headline)

            VStack(alignment: .leading, spacing: 4) {
                Text("Link").font(.caption).foregroundStyle(.secondary)
                TextField("https://…", text: $url, axis: .vertical)
                    .textFieldStyle(.roundedBorder)
                    .lineLimit(1...3)
                    .focused($urlFocused)
                    .onSubmit(submit)
            }

            VStack(alignment: .leading, spacing: 4) {
                Text("Save as").font(.caption).foregroundStyle(.secondary)
                TextField("leave empty to use the server's name", text: $filename)
                    .textFieldStyle(.roundedBorder)
                    .onSubmit(submit)
            }

            if isMedia {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Quality").font(.caption).foregroundStyle(.secondary)
                    Picker("", selection: $format) {
                        Text("Best — plays anywhere").tag("best")
                        Text("2160p — VP9 or AV1").tag("2160")
                        Text("1440p — VP9 or AV1").tag("1440")
                        Text("1080p").tag("1080")
                        Text("720p").tag("720")
                        Text("480p").tag("480")
                        Text("Audio only (mp3)").tag("audio")
                    }
                    .labelsHidden()
                    .pickerStyle(.menu)
                    Text("Up to 1080p is H.264, which every player opens. Larger sizes are VP9 or AV1 — fine in Chrome, VLC and recent Macs, refused by older QuickTime.")
                        .font(.caption)
                        .foregroundStyle(.tertiary)
                }
            }

            HStack {
                Text(model.settings.downloadDir.isEmpty
                     ? "" : "Saving to \(model.settings.downloadDir)")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .lineLimit(1)
                    .truncationMode(.middle)
                Spacer()
                Button("Cancel") { dismiss() }
                    .keyboardShortcut(.cancelAction)
                Button("Download", action: submit)
                    .keyboardShortcut(.defaultAction)
                    .disabled(!looksLikeURL(url))
            }
        }
        .padding(18)
        .frame(width: 440)
        .onAppear {
            if let clip = Self.clipboardURL(), url.isEmpty {
                url = clip
            }
            urlFocused = true
        }
    }

    private func submit() {
        guard looksLikeURL(url) else { return }
        model.add(url: url,
                  filename: filename.trimmingCharacters(in: .whitespaces),
                  format: isMedia ? format : "")
        dismiss()
    }

    private func looksLikeURL(_ s: String) -> Bool {
        Self.normalizedURL(s) != nil
    }

    /// A pasted link is usable if it parses and carries a host.
    static func normalizedURL(_ s: String) -> String? {
        let t = s.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !t.isEmpty, t.count < 4096,
              let u = URL(string: t),
              let scheme = u.scheme?.lowercased(),
              scheme == "http" || scheme == "https",
              let host = u.host, host.contains(".")
        else { return nil }
        return t
    }

    static func clipboardURL() -> String? {
        guard let s = NSPasteboard.general.string(forType: .string) else { return nil }
        return normalizedURL(s)
    }

    /// Mirrors the host list in internal/downloader/ytdlp.go. Only decides
    /// whether to offer quality options — the daemon makes the real call.
    static func looksLikeMediaHost(_ s: String) -> Bool {
        guard let t = normalizedURL(s), let host = URL(string: t)?.host?.lowercased()
        else { return false }
        let h = host.hasPrefix("www.") ? String(host.dropFirst(4)) : host
        let hosts = ["youtube.com", "youtu.be", "vimeo.com", "twitch.tv", "tiktok.com",
                     "x.com", "twitter.com", "instagram.com", "facebook.com",
                     "dailymotion.com", "soundcloud.com"]
        return hosts.contains { h == $0 || h.hasSuffix("." + $0) }
    }
}
