package nginx

import "embed"

// Bundled for Docker/runtime when web/assets is not on disk.
//
//go:embed assets/error.png assets/429.png
var errorAssetFS embed.FS
