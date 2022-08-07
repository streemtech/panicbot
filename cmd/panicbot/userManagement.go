package main

import (
	"time"

	"github.com/streemtech/panicbot/ticker"
)

func (c *Container) startReloadRolesTimer() error {
	duration := time.Minute * 30
	ticker.SimpleTickerFunc(duration, func() {
		err := c.reloadRoles()
		if err != nil {
			c.Logger.Errorf("failed to reload users and roles: %s", err.Error())
		}
	})
	c.Logger.Debugf("successfully started %s role reload timer", duration)
	return nil
}

func (c *Container) reloadRoles() error {
	c.Logger.Debugf("successfully reloaded roles")
	return nil
}
