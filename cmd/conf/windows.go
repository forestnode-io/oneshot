// +build windows

package conf

func init() {
	archiveMethodDefault = "zip"
	shellDefault = `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`
	noUnixNormDefault = true
}
