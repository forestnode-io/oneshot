source = ["./dist/oneshot-darwin-arm64_darwin_arm64/oneshot"]
bundle_id = "uno.oneshot.oneshot"

apple_id {
    username = "raphaelreyna@protonmail.com"
    password = "@env:APPLE_DEV_PASSWORD"
}

sign {
    application_identity = "Developer ID Application: Raphael Reyna"
}