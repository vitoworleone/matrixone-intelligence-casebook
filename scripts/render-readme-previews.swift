import AppKit
import Foundation

// Render README presentation cards without changing the original screenshots.
// Run from the repository root: swift scripts/render-readme-previews.swift

let canvasWidth = 1200
let canvasHeight = 680
let imageWidth: CGFloat = 1080
let imageHeight: CGFloat = 539
let imageRect = NSRect(x: 60, y: 71, width: imageWidth, height: imageHeight)

func renderPreview(source: String, destination: String) throws {
    guard let screenshot = NSImage(contentsOfFile: source),
          let bitmap = NSBitmapImageRep(
            bitmapDataPlanes: nil,
            pixelsWide: canvasWidth,
            pixelsHigh: canvasHeight,
            bitsPerSample: 8,
            samplesPerPixel: 4,
            hasAlpha: true,
            isPlanar: false,
            colorSpaceName: .deviceRGB,
            bytesPerRow: 0,
            bitsPerPixel: 0
          ),
          let context = NSGraphicsContext(bitmapImageRep: bitmap) else {
        throw NSError(domain: "READMEPreview", code: 1, userInfo: [NSLocalizedDescriptionKey: "Cannot open \(source)"])
    }

    NSGraphicsContext.saveGraphicsState()
    NSGraphicsContext.current = context
    context.imageInterpolation = .high
    context.shouldAntialias = true

    let background = NSBezierPath(roundedRect: NSRect(x: 0, y: 0, width: canvasWidth, height: canvasHeight), xRadius: 26, yRadius: 26)
    NSColor(calibratedRed: 0.965, green: 0.973, blue: 0.984, alpha: 1).setFill()
    background.fill()

    let frame = NSBezierPath(roundedRect: imageRect, xRadius: 12, yRadius: 12)
    NSGraphicsContext.saveGraphicsState()
    let shadow = NSShadow()
    shadow.shadowColor = NSColor(calibratedWhite: 0.12, alpha: 0.28)
    shadow.shadowBlurRadius = 36
    shadow.shadowOffset = NSSize(width: 0, height: -18)
    shadow.set()
    NSColor.white.setFill()
    frame.fill()
    NSGraphicsContext.restoreGraphicsState()

    NSGraphicsContext.saveGraphicsState()
    frame.addClip()
    screenshot.draw(in: imageRect, from: .zero, operation: .copy, fraction: 1)
    NSGraphicsContext.restoreGraphicsState()

    NSColor(calibratedWhite: 0.70, alpha: 0.28).setStroke()
    frame.lineWidth = 1
    frame.stroke()

    context.flushGraphics()
    NSGraphicsContext.restoreGraphicsState()

    guard let png = bitmap.representation(using: .png, properties: [:]) else {
        throw NSError(domain: "READMEPreview", code: 2, userInfo: [NSLocalizedDescriptionKey: "Cannot encode \(destination)"])
    }
    try png.write(to: URL(fileURLWithPath: destination))
}

do {
    try renderPreview(
        source: "assets/screenshots/moi-platform/data-workbench-overview.png",
        destination: "assets/screenshots/moi-platform/data-workbench-overview-preview.png"
    )
    try renderPreview(
        source: "assets/screenshots/moi-platform/agent-workbench-home.png",
        destination: "assets/screenshots/moi-platform/agent-workbench-home-preview.png"
    )
} catch {
    fputs("\(error.localizedDescription)\n", stderr)
    exit(1)
}
