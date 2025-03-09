package about

// Info represents version information
type Info struct {
	Version   string `json:"version"`
	AppName   string `json:"appName"`
	Copyright string `json:"copyright"`
	URL       string `json:"url"`
}

const Project = "FauxBase"
const AppName = "FauxBase"
const ApiName = "FauxBase API"
const Version = "0.0.1"
const URL = "https://github.com/ytjohn/fauxbase"
const Copyright = "© 2025 ytjohn. Licensed under the MIT License."
