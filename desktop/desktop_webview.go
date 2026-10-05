//go:build !windows && webview

package desktop

import webview "github.com/webview/webview_go"

func StartWebView(url string, debug bool) {
	w := webview.New(debug)
	defer w.Destroy()
	w.SetTitle("OpenBooks")
	w.SetSize(1200, 800, webview.HintNone)
	w.Navigate(url)
	w.Run()
}
