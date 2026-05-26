import AppKit
import UserNotifications

// motivation-notify: post one macOS notification via UNUserNotificationCenter
// and, if -open URL is given, open that URL when the user clicks the toast.
// -emoji renders the emoji as the notification's icon (attachment thumbnail).
// Exits after click or after -timeout seconds (default 30).

struct Args {
    var title = ""
    var message = ""
    var openURL: String?
    var emoji: String?
    var sound = true
    var timeout: TimeInterval = 30
}

func parseArgs() -> Args {
    var a = Args()
    let argv = CommandLine.arguments
    var i = 1
    while i < argv.count {
        let k = argv[i]
        let next: () -> String = {
            i += 1
            return i < argv.count ? argv[i] : ""
        }
        switch k {
        case "-title":   a.title = next()
        case "-message": a.message = next()
        case "-open":    a.openURL = next()
        case "-emoji":   a.emoji = next()
        case "-no-sound": a.sound = false
        case "-timeout":
            if let t = TimeInterval(next()) { a.timeout = t }
        default:
            FileHandle.standardError.write(Data("unknown flag: \(k)\n".utf8))
        }
        i += 1
    }
    return a
}

// renderEmoji draws the emoji centred on a transparent square canvas and
// returns the in-memory NSImage plus its PNG bytes (for file output).
func renderEmoji(_ emoji: String, side: CGFloat = 512) -> (NSImage, Data)? {
    let font = NSFont.systemFont(ofSize: side * 0.78)
    let attrs: [NSAttributedString.Key: Any] = [.font: font]
    let str = NSAttributedString(string: emoji, attributes: attrs)
    let textSize = str.size()
    let canvas = NSSize(width: side, height: side)

    let image = NSImage(size: canvas)
    image.lockFocus()
    NSColor.clear.setFill()
    NSRect(origin: .zero, size: canvas).fill()
    let origin = NSPoint(
        x: (canvas.width - textSize.width) / 2,
        y: (canvas.height - textSize.height) / 2
    )
    str.draw(at: origin)
    image.unlockFocus()

    guard let tiff = image.tiffRepresentation,
          let rep = NSBitmapImageRep(data: tiff),
          let png = rep.representation(using: .png, properties: [:])
    else { return nil }
    return (image, png)
}

// writePNGToTmp writes PNG bytes to a unique temp file and returns its URL.
func writePNGToTmp(_ png: Data) -> URL? {
    let tmp = FileManager.default.temporaryDirectory
        .appendingPathComponent("motivation-emoji-\(UUID().uuidString).png")
    do { try png.write(to: tmp); return tmp } catch { return nil }
}

// swapBundleIcon overwrites this app's Contents/Resources/AppIcon.png with
// the rendered emoji and touches the bundle so LaunchServices invalidates
// its cached icon for com.motivation.notifier. Notification Center is
// stubborn about icon caching — this is best-effort.
func swapBundleIcon(png: Data) {
    let bundlePath = Bundle.main.bundlePath
    let resources = (bundlePath as NSString).appendingPathComponent("Contents/Resources")
    try? FileManager.default.createDirectory(
        atPath: resources, withIntermediateDirectories: true)
    let dest = (resources as NSString).appendingPathComponent("AppIcon.png")
    try? png.write(to: URL(fileURLWithPath: dest))
    // Touch the bundle root so LaunchServices notices.
    try? FileManager.default.setAttributes(
        [.modificationDate: Date()], ofItemAtPath: bundlePath)
}

final class Delegate: NSObject, UNUserNotificationCenterDelegate {
    let openURL: String?
    init(openURL: String?) { self.openURL = openURL }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        completionHandler([.banner, .sound])
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        if let s = openURL, let url = URL(string: s) {
            NSWorkspace.shared.open(url)
        }
        completionHandler()
        exit(0)
    }
}

let args = parseArgs()
if args.title.isEmpty && args.message.isEmpty {
    FileHandle.standardError.write(Data("usage: motivation-notify -title T -message M [-open URL] [-emoji E] [-no-sound] [-timeout S]\n".utf8))
    exit(2)
}

let center = UNUserNotificationCenter.current()
let delegate = Delegate(openURL: args.openURL)
center.delegate = delegate

let authSema = DispatchSemaphore(value: 0)
center.requestAuthorization(options: [.alert, .sound]) { _, _ in
    authSema.signal()
}
authSema.wait()

let content = UNMutableNotificationContent()
content.title = args.title
content.body = args.message
if args.sound { content.sound = .default }
if let s = args.openURL { content.userInfo = ["openURL": s] }

if let e = args.emoji, !e.isEmpty, let (image, png) = renderEmoji(e) {
    swapBundleIcon(png: png)
    // Also set the running-process icon — Notification Center sometimes
    // reads this when posting from the same process that owns the icon.
    NSApplication.shared.applicationIconImage = image
    if let pngURL = writePNGToTmp(png),
       let att = try? UNNotificationAttachment(identifier: "emoji-icon", url: pngURL) {
        content.attachments = [att]
    }
}

let req = UNNotificationRequest(
    identifier: UUID().uuidString,
    content: content,
    trigger: nil
)

let addSema = DispatchSemaphore(value: 0)
var addErr: Error?
center.add(req) { err in
    addErr = err
    addSema.signal()
}
addSema.wait()
if let err = addErr {
    FileHandle.standardError.write(Data("add notification failed: \(err)\n".utf8))
    exit(1)
}

// Wait for the user to click, or time out and exit silently.
DispatchQueue.main.asyncAfter(deadline: .now() + args.timeout) {
    exit(0)
}
RunLoop.main.run()
