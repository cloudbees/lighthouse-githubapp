package testhelpers

// SchedulerFile contains a list of leaf files to build the scheduler from
type SchedulerFile struct {
	// Filenames is the hierarchy with the leaf at the right
	Filenames []string
	Org       string
	Repo      string
}
