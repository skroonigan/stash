package manager

type JobStatus int

const (
	Idle     JobStatus = 0
	Import   JobStatus = 1
	Export   JobStatus = 2
	Scan     JobStatus = 3
	Generate JobStatus = 4
	Clean    JobStatus = 5
	Scrape   JobStatus = 6
)

func (s JobStatus) String() string {
	statusMessage := ""

	switch s {
	case Idle:
		statusMessage = "Idle"
	case Import:
		statusMessage = "Import"
	case Export:
		statusMessage = "Export"
	case Scan:
		statusMessage = "Scan"
	case Generate:
		statusMessage = "Generate"
	}

	return statusMessage
}
