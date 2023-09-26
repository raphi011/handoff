package model

type NotFoundError struct{}

func (e NotFoundError) Error() string {
	return "not found"
}
