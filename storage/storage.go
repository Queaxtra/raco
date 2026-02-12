package storage

type Storage struct {
	basePath string
}

func NewStorage(basePath string) *Storage {
	return &Storage{basePath: basePath}
}
