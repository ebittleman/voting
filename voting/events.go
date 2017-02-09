package voting

var EventTypes = []string{
	"PollCreated",
	"PollOpened",
	"PollClosed",
	"IssueAppended",
	"BallotCast",
}

// PollCreated metadata for PollCreated event
type PollCreated struct {
	ID string `json:"id"`
}

// PollOpened metadata for PollOpened event
type PollOpened struct {
	ID string
}

// PollClosed metadata for PollClosed event
type PollClosed struct {
	ID string
}

// IssueAppended metadata for IssueAppended event
type IssueAppended struct {
	Topic      string   `json:"topic"`
	Choices    []string `json:"choices"`
	CanWriteIn bool     `json:"can_write_in"`
}

// BallotSelection metadata items for BallotCast event
type BallotSelection struct {
	IssueTopic string `json:"issue_topic"`
	WroteIn    bool   `json:"wrote_in"`
	Choice     int    `json:"choice"`
	WriteIn    string `json:"write_in"`
	Comment    string `json:"comment"`
}

// BallotCast metadata for BallotCast event
type BallotCast []BallotSelection
