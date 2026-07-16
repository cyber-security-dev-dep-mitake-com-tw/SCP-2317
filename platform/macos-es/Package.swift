// swift-tools-version: 5.10
import PackageDescription

let package = Package(
	name: "SCP2317ES",
	platforms: [.macOS(.v13)],
	products: [
		.executable(name: "SCP2317ES", targets: ["SCP2317ES"]),
	],
	targets: [
		.executableTarget(name: "SCP2317ES", path: "Sources/MailGuardES"),
	]
)
