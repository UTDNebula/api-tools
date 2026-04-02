package utils

type SetupError struct {
	Message string
}

func (e *SetupError) Error() string {
	return e.Message
}

type NetworkError struct {
	Message string
}

func (e *NetworkError) Error() string {
	return e.Message
}
