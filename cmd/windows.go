// +build windows

package cmd

func init() {
	shellDefault = `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
	noUnixNorm = true
	archiveMethodDefault = "zip"
}
