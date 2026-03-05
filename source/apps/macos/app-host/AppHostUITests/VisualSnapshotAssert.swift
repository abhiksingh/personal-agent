import AppKit
import CoreGraphics
import XCTest

struct VisualSnapshotAssert {
    private let baselineDirectory: URL
    private let failureDirectory: URL
    private let shouldUpdateBaselines: Bool
    private let tolerance: Double

    init(filePath: StaticString = #filePath) {
        let environment = ProcessInfo.processInfo.environment
        shouldUpdateBaselines =
            environment["PA_UI_UPDATE_VISUAL_BASELINES"] == "1" ||
            Self.visualUpdateMarkerExists(environment: environment)
        tolerance = max(0.0, environment["PA_UI_VISUAL_DIFF_TOLERANCE"].flatMap(Double.init) ?? 0.015)
        baselineDirectory = Self.resolveBaselineDirectory(environment: environment, filePath: filePath)
        failureDirectory = baselineDirectory.appendingPathComponent("__Failures__", isDirectory: true)
    }

    func assertSnapshot(
        of element: XCUIElement,
        named snapshotName: String,
        file: StaticString = #filePath,
        line: UInt = #line
    ) {
        let fileManager = FileManager.default
        let baselineURL = baselineDirectory.appendingPathComponent("\(snapshotName).png")
        let screenshot = element.screenshot()
        let actualImage = screenshot.image

        guard let actualData = actualImage.pngDataRepresentation else {
            XCTFail("Failed to encode snapshot PNG for \(snapshotName).", file: file, line: line)
            return
        }

        do {
            try fileManager.createDirectory(at: baselineDirectory, withIntermediateDirectories: true)
        } catch {
            XCTFail("Failed to create baseline directory: \(error)", file: file, line: line)
            return
        }

        if shouldUpdateBaselines || !fileManager.fileExists(atPath: baselineURL.path) {
            do {
                try actualData.write(to: baselineURL, options: .atomic)
            } catch {
                XCTFail("Failed to write baseline snapshot \(snapshotName): \(error)", file: file, line: line)
                return
            }

            if shouldUpdateBaselines {
                return
            }

            XCTFail(
                "Missing baseline snapshot \(snapshotName) in \(baselineDirectory.path). " +
                "Re-run with PA_UI_UPDATE_VISUAL_BASELINES=1 to create baselines.",
                file: file,
                line: line
            )
            return
        }

        guard
            let baselineImage = NSImage(contentsOf: baselineURL),
            let baselineCGImage = baselineImage.cgImageRepresentation,
            let actualCGImage = actualImage.cgImageRepresentation
        else {
            XCTFail("Failed to decode baseline or actual image for \(snapshotName).", file: file, line: line)
            return
        }

        guard
            baselineCGImage.width == actualCGImage.width,
            baselineCGImage.height == actualCGImage.height
        else {
            XCTFail(
                "Snapshot size mismatch for \(snapshotName): expected \(baselineCGImage.width)x\(baselineCGImage.height), got \(actualCGImage.width)x\(actualCGImage.height).",
                file: file,
                line: line
            )
            return
        }

        let diff = normalizedMeanAbsoluteDifference(lhs: baselineCGImage, rhs: actualCGImage)
        if diff <= tolerance {
            return
        }

        do {
            try fileManager.createDirectory(at: failureDirectory, withIntermediateDirectories: true)
            try actualData.write(to: failureDirectory.appendingPathComponent("\(snapshotName)-actual.png"), options: .atomic)
        } catch {
            XCTFail("Snapshot regression for \(snapshotName) (diff=\(diff)); failed to write artifacts: \(error)", file: file, line: line)
            return
        }

        XCTFail(
            "Snapshot regression for \(snapshotName): normalized diff \(String(format: "%.6f", diff)) exceeded tolerance \(String(format: "%.6f", tolerance)). " +
            "Inspect \(failureDirectory.path)/\(snapshotName)-actual.png (baseline: \(baselineDirectory.path)/\(snapshotName).png) " +
            "or re-run with PA_UI_UPDATE_VISUAL_BASELINES=1 to refresh.",
            file: file,
            line: line
        )
    }

    private static func resolveBaselineDirectory(environment: [String: String], filePath: StaticString) -> URL {
        if let overridePath = environment["PA_UI_VISUAL_BASELINE_DIR"], !overridePath.isEmpty {
            return URL(fileURLWithPath: overridePath, isDirectory: true)
        }

        if let applicationSupport = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first {
            return applicationSupport
                .appendingPathComponent("PersonalAgentUI", isDirectory: true)
                .appendingPathComponent("VisualBaselines", isDirectory: true)
        }

        let sourceDirectory = URL(fileURLWithPath: "\(filePath)").deletingLastPathComponent()
        return sourceDirectory.appendingPathComponent("Baselines", isDirectory: true)
    }

    private static func visualUpdateMarkerExists(environment: [String: String]) -> Bool {
        let markerPath = environment["PA_UI_VISUAL_UPDATE_MARKER_FILE"] ?? "/tmp/personalagent-ui-update-baselines.flag"
        return FileManager.default.fileExists(atPath: markerPath)
    }

    private func normalizedMeanAbsoluteDifference(lhs: CGImage, rhs: CGImage) -> Double {
        guard
            let lhsBytes = rgbaBytes(for: lhs),
            let rhsBytes = rgbaBytes(for: rhs),
            lhsBytes.count == rhsBytes.count,
            !lhsBytes.isEmpty
        else {
            return 1.0
        }

        var totalDifference: UInt64 = 0
        for index in lhsBytes.indices {
            totalDifference += UInt64(abs(Int(lhsBytes[index]) - Int(rhsBytes[index])))
        }

        let denominator = Double(lhsBytes.count) * 255.0
        guard denominator > 0 else {
            return 0
        }
        return Double(totalDifference) / denominator
    }

    private func rgbaBytes(for image: CGImage) -> [UInt8]? {
        let width = image.width
        let height = image.height
        let bytesPerPixel = 4
        let bytesPerRow = width * bytesPerPixel
        let bitsPerComponent = 8
        var bytes = [UInt8](repeating: 0, count: height * bytesPerRow)

        guard
            let colorSpace = CGColorSpace(name: CGColorSpace.sRGB),
            let context = CGContext(
                data: &bytes,
                width: width,
                height: height,
                bitsPerComponent: bitsPerComponent,
                bytesPerRow: bytesPerRow,
                space: colorSpace,
                bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue | CGBitmapInfo.byteOrder32Big.rawValue
            )
        else {
            return nil
        }

        context.draw(image, in: CGRect(x: 0, y: 0, width: width, height: height))
        return bytes
    }
}

private extension NSImage {
    var cgImageRepresentation: CGImage? {
        var proposedRect = CGRect(origin: .zero, size: size)
        return cgImage(forProposedRect: &proposedRect, context: nil, hints: nil)
    }

    var pngDataRepresentation: Data? {
        guard
            let tiffRepresentation,
            let bitmap = NSBitmapImageRep(data: tiffRepresentation)
        else {
            return nil
        }
        return bitmap.representation(using: .png, properties: [:])
    }
}
