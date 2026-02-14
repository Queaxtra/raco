package version

var (
	Version = "dev"
	Commit  = "unknown"
)

func String() string {
	return "raco " + Version
}
