package clipboardutil

import "github.com/atotto/clipboard"

// Copy writes the provided text to the system clipboard. It can be stubbed in
// tests by replacing this variable.
var Copy = clipboard.WriteAll
