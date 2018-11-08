package main

import (
	"context"
	"fmt"
	"os"
	"time"
)

const TypeDailyAccountInfo = MessageType("DailyAccountInfo")
const TypeMonthlyAccountInfo = MessageType("MonthlyAccountInfo")

type AccountInfo struct {
	Date          time.Time
	AvailableCash USD
	AccountValue  USD
	CashFlow      USD
	Interest      USD
}

// DailyOutput writes a log entry for each day during the simulation
type DailyOutput struct {
	File *os.File
}

// Handle sends date messages to all the child processes.
func (d *DailyOutput) Handle(ctx context.Context, proc Process, msg Message) {
	switch msg.Type {
	case MessageTypeStart:
		d.File.WriteString("date,available,value,cashflow,interest\n")
	case TypeDailyAccountInfo:
		info := msg.Value.(AccountInfo)
		d.File.WriteString(fmt.Sprintf("%s,%.2f,%.2f,%.2f,%.2f\n",
			info.Date.Format("2006-01-02"),
			float64(info.AvailableCash)/100,
			float64(info.AccountValue)/100,
			float64(info.CashFlow)/100,
			float64(info.Interest)/100,
		))
	case MessageTypeStop:
	}
}

// MonthlyOutput writes a log entry on the first of each month during the simulation
type MonthlyOutput struct {
	File *os.File
}

// Handle sends date messages to all the child processes.
func (d *MonthlyOutput) Handle(ctx context.Context, proc Process, msg Message) {
	switch msg.Type {
	case MessageTypeStart:
		d.File.WriteString("date,available,value,cashflow,interest\n")
	case TypeMonthlyAccountInfo:
		info := msg.Value.(AccountInfo)
		d.File.WriteString(fmt.Sprintf("%s,%.2f,%.2f,%.2f,%.2f\n",
			info.Date.Format("2006-01-02"),
			float64(info.AvailableCash)/100,
			float64(info.AccountValue)/100,
			float64(info.CashFlow)/100,
			float64(info.Interest)/100,
		))
	case MessageTypeStop:
	}
}
