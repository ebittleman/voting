package voting

type PollCreated struct {
	ID string `json:"id"`
}

type PollOpened struct {
	ID string
}

type PollClosed struct {
	ID string
}
