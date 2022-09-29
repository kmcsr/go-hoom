
package main

import (
	"fmt"
	"strings"
)

type TextCommander struct{
}

func (TextCommander)Execute(cmd string, fields ...string)(res string, err error){
	cmd = strings.ToLower(cmd)
	c, ok := commands[cmd]
	if !ok {
		return "", fmt.Errorf("No command called '%s'", cmd)
	}
	resn, err := c.Execute(fields...)
	if err != nil {
		return fmt.Sprintln("error", err), nil
	}
	return fmt.Sprintln(resn...), nil
}
