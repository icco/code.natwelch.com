package static

import "embed"

// Assets are our static files for sharing. Web assets are listed explicitly so
// this package's own .go source is never embedded or served.
//
//go:embed app.js index.html favicon.ico
var Assets embed.FS
