package repository

type rowScanner interface {
	Scan(...any) error
}
