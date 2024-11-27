package model

type NotFoundError struct{}

func (e NotFoundError) Error() string {
	return "not found"
}

type DuplicateError struct{}

func (e DuplicateError) Error() string {
	return "duplicate entry"
}
