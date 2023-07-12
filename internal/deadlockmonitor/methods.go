package deadlockmonitor

import (
	"github.com/windnow/tlanalyzer/internal/myfsm"
)

func (e *event) getEvents() []myfsm.Event {

	return nil
}

func (e *event) getHelper() *eventHelper {

	getProcesses := func(refs []processRef) map[string]process {
		result := make(map[string]process)

		for _, ref := range refs {
			for _, proc := range e.Processes {
				if ref.Id == proc.Id {
					result[ref.Id] = proc
				}
			}
		}

		return result

	}

	helper := &eventHelper{}
	helper.victims = getProcesses(e.Victims)

	/*getObjects := func(objLocks []objectlock) []object {
		objects := make([]object, len(objLocks))
	}*/

	return helper
}

type eventHelper struct {
	victims map[string]process
	objects []object
}

type object struct {
	objectName string
	mode       string
	owners     map[string]process
	waiters    map[string]process
}

/*

func (e *event) getEvents() []myfsm.Event {
	if len(e.Victims) == 0 {
		return nil
	}

	victimProcess := e.Victims[0]
	var victim *process
	for _, p := range e.Processes {
		if p.Id != victimProcess.Id {
			continue
		}
		victim = &p
		break
	}

	resourceList := e.Objectlocks
	resourceList = append(resourceList, e.Pagelocks...)

	events := make([]myfsm.BulkEvent, 1)

EventFilling:
	for _, resource := range resourceList {
		for _, waiter := range resource.Waiters {
			if waiter.Id == victim.Id {

				owners := resource.Owners
				for _, owner := range owners {

					event := myfsm.BulkEvent{
						Name:        "MSSQLDEADLOCK",
						ProcessName: "sqlservr",
						Time:        e.Timestamp,
						Level:       "1",
						Duration:    strconv.Itoa(victim.Waittime),
						Fields:      make(map[string]string),
					}

					event.Fields["victim"] = strconv.Itoa(victim.SPID)

					events = append(events, event)

				}

				break EventFilling

			}
		}

	}

	return events
}

*/
