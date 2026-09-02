import { Clipboard } from '@wailsio/runtime'

// WKWebView (Wails v3, macOS) has an unreliable navigator.clipboard — both read
// and write can reject outside a user gesture. Prefer the Wails runtime bridge
// and fall back to the web API; see issue 0004 for the original diagnosis.

/** Write text to the system clipboard. Returns whether any path succeeded. */
export async function writeClipboard(text: string): Promise<boolean> {
  try {
    await Clipboard.SetText(text)
    return true
  } catch {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      return false
    }
  }
}

/** Read text from the system clipboard; '' when unavailable. */
export async function readClipboard(): Promise<string> {
  try {
    return await Clipboard.Text()
  } catch {
    try {
      return await navigator.clipboard.readText()
    } catch {
      return ''
    }
  }
}
