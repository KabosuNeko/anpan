package engine

// UserAgent identifies anpan to remote APIs; cmd sets the versioned value at startup.
var UserAgent = "anpan"

// browserUserAgent is used by site scrapers that reject non-browser clients.
const browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
