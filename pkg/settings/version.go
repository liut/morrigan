package settings

var (
	version = "dev"
)

func InDevelop() bool {
	return "dev" == version
}

func Version() string {
	return version
}
