package myfsm

type ProcessFunc func([]*Event)

func ProcessLogs(rootDir string, processFunc ProcessFunc) {

	events := WalkDir(rootDir)
	processFunc(events)

}
