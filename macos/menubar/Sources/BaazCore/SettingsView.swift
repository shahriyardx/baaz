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

            SettingRow("Sort into category folders") {
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

            SettingRow("Speed limit") {
                Picker("", selection: Binding(
                    get: { model.settings.speedLimitKB },
                    set: { model.setConfig("speed-limit", String($0)) }
                )) {
                    Text("Unlimited").tag(0)
                    Text("500 KB/s").tag(500)
                    Text("1 MB/s").tag(1024)
                    Text("2 MB/s").tag(2048)
                    Text("5 MB/s").tag(5120)
                    Text("10 MB/s").tag(10240)
                    // A value set from the CLI that is not one of the presets
                    // still needs somewhere to show.
                    if ![0, 500, 1024, 2048, 5120, 10240].contains(model.settings.speedLimitKB) {
                        Text("\(model.settings.speedLimitKB) KB/s")
                            .tag(model.settings.speedLimitKB)
                    }
                }
                .labelsHidden()
                .pickerStyle(.menu)
                .controlSize(.small)
            }

            StepperRow(label: "Min size to grab", key: "min-size",
                       value: model.settings.minSizeMB, range: 0...500, step: 5, unit: " MB")

            Divider()

            VStack(alignment: .leading, spacing: 3) {
                HStack {
                    Text("Saving to").font(.caption).foregroundStyle(.secondary)
                    Spacer()
                    // TextButton, not a one-off Button: it is the construction
                    // that was actually click-tested in this panel.
                    TextButton(title: "Change…") { model.chooseDownloadDir() }
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

/// One settings line: label on the left, control in a fixed column on the
/// right. The fixed width is what keeps the switch, the steppers and the
/// pop-up button on a single edge instead of each ending wherever its own
/// intrinsic width happens to fall.
struct SettingRow<Control: View>: View {
    let label: String
    let control: Control

    /// Wide enough for the pop-up button's longest preset; everything else
    /// is right-aligned within it.
    static var controlWidth: CGFloat { 116 }

    init(_ label: String, @ViewBuilder control: () -> Control) {
        self.label = label
        self.control = control()
    }

    var body: some View {
        HStack(spacing: 8) {
            Text(label)
                .font(.callout)
                .lineLimit(1)
                .truncationMode(.tail)
            Spacer(minLength: 4)
            control
                .frame(width: Self.controlWidth, alignment: .trailing)
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
        SettingRow(label) {
            HStack(spacing: 6) {
                Spacer(minLength: 0)
                Text("\(value)\(unit)")
                    .font(.callout.monospacedDigit())
                    .foregroundStyle(.secondary)
                Stepper("") { bump(step) } onDecrement: { bump(-step) }
                    .labelsHidden()
                    .controlSize(.small)
            }
        }
    }

    private func bump(_ delta: Int) {
        let next = min(range.upperBound, max(range.lowerBound, value + delta))
        guard next != value else { return }
        model.setConfig(key, String(next))
    }
}
