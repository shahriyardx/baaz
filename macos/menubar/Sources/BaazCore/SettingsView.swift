import SwiftUI

/// The gear page. Every control writes straight through to the daemon via
/// `baaz config`; the next snapshot is what redraws the value, so the UI
/// never holds its own copy of the truth.
struct SettingsView: View {
    @EnvironmentObject var model: DownloadsModel

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text("Settings")
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)

            HStack {
                Text("Sort into category folders").font(.callout)
                Spacer()
                Toggle("", isOn: Binding(
                    get: { model.settings.categorize },
                    set: { _ in model.toggleCategorize() }
                ))
                .toggleStyle(.switch)
                .labelsHidden()
                .controlSize(.small)
            }

            StepperRow(label: "Parallel downloads", key: "max-active",
                       value: model.settings.maxActive, range: 1...10, step: 1)
            StepperRow(label: "Segments per file", key: "segments",
                       value: model.settings.segments, range: 1...16, step: 1)
            StepperRow(label: "Min size to grab", key: "min-size",
                       value: model.settings.minSizeMB, range: 0...500, step: 5, unit: " MB")

            Divider()

            VStack(alignment: .leading, spacing: 3) {
                HStack {
                    Text("Saving to").font(.caption).foregroundStyle(.secondary)
                    Spacer()
                    Button {
                        model.chooseDownloadDir()
                    } label: {
                        Text("Change…")
                            .font(.caption)
                            .padding(.vertical, 2)
                            .padding(.horizontal, 4)
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .foregroundStyle(.tint)
                }
                Text(model.settings.downloadDir.isEmpty ? "—" : model.settings.downloadDir)
                    .font(.caption)
                    .foregroundStyle(.tertiary)
                    .lineLimit(1)
                    .truncationMode(.middle)
            }
        }
    }
}

struct StepperRow: View {
    @EnvironmentObject var model: DownloadsModel
    let label: String
    let key: String
    let value: Int
    let range: ClosedRange<Int>
    let step: Int
    var unit: String = ""

    var body: some View {
        HStack {
            Text(label).font(.callout)
            Spacer()
            Text("\(value)\(unit)")
                .font(.callout.monospacedDigit())
                .foregroundStyle(.secondary)
            Stepper("") { bump(step) } onDecrement: { bump(-step) }
                .labelsHidden()
        }
    }

    private func bump(_ delta: Int) {
        let next = min(range.upperBound, max(range.lowerBound, value + delta))
        guard next != value else { return }
        model.setConfig(key, String(next))
    }
}
