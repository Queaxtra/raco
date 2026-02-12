package cmd

import "raco/storage"

type Context struct {
	StoragePath string
}

func (c *Context) Storage() *storage.Storage {
	return storage.NewStorage(c.StoragePath)
}
