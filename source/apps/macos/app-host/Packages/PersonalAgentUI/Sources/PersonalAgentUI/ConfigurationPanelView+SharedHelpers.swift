import SwiftUI

extension ConfigurationPanelView {
    func sectionTitle(_ text: String) -> some View {
        Text(text)
            .font(.subheadline.weight(.semibold))
    }

    func settingRow(label: String, value: String) -> some View {
        HStack(alignment: .top, spacing: 10) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(width: 122, alignment: .leading)
            Text(value)
                .font(.callout)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    func identitySettingRow(
        label: String,
        displayValue: IdentityDisplayValue,
        labelWidth: CGFloat
    ) -> some View {
        HStack(alignment: .top, spacing: 10) {
            Text(label)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .frame(width: labelWidth, alignment: .leading)
            IdentityValueInlineView(
                displayText: displayValue.displayText,
                rawID: displayValue.rawID,
                valueFont: .callout
            )
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }
}
