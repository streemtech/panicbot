package main

import (
	"time"

	"github.com/streemtech/panicbot/ticker"
)

func (c *Container) startReloadRolesTimer() error {
	ticker.SimpleTickerFunc(time.Minute*30, func() {
		err := c.reloadRoles()
		if err != nil {
			c.Logger.Errorf("failed to reload users and roles: %s", err.Error())
		}
	})
	return nil
}

func (c *Container) reloadRoles() error {
	return nil
}
