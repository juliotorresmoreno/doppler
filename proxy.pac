function FindProxyForURL(url, host) {
    return "HTTPS {{.Host}}:{{.TLSPort}}";
}
