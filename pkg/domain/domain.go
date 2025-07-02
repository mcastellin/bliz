package domain

// Wrapper is a wrapper struct for fuzz payloads
type Wrapper struct {
	Host    string
	Fuzz    []string
	Request string
}

type FuzzResponse struct {
	Req        Wrapper
	Body       string
	Status     string
	StatusCode int
	Size       int64
	Lines      int
	Words      int
}

type NumericGen struct {
	Start int
	End   int
	Step  int
}
