source = ["./build-output/goreleaser/oneshot-darwin-amd64_darwin_amd64_v1/oneshot"]
bundle_id = "io.forestnode.oneshot"

apple_id {
    username = "raphaelreyna@protonmail.com"
    password = "@env:APPLE_DEV_PASSWORD"
}

sign {
    application_identity = "Developer ID Application: Raphael Reyna"
}