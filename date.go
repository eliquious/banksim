package main

import (
	"context"
	"fmt"
	"time"
)

// TypeDate is the message type for random numbers
const TypeDate = MessageType("Date")

type DayGenerator struct {
	StartDate time.Time
	EndDate   time.Time
}

// Handle sends date messages to all the child processes.
func (d *DayGenerator) Handle(ctx context.Context, proc Process, msg Message) {
	switch msg.Type {
	default:
		fmt.Println("Unknown message type: ", msg.Type)
	case MessageTypeStart:
		for !d.StartDate.After(d.EndDate) {
			// fmt.Println(start)
			proc.Children().Dispatch(Message{
				Forward:   false,
				Type:      TypeDate,
				Timestamp: time.Now().UTC(),
				Value:     d.StartDate,
			})
			d.StartDate = d.StartDate.AddDate(0, 0, 1)
		}
		proc.SetState(StateKilled)
	}
}
