package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/atgjack/prob"
)

// TransactionType describes the type of transaction for an account
type TransactionType string

const (

	// Deposit represents account deposits
	Deposit TransactionType = "DEPOSIT"

	// Withdrawal represents account withdrawals
	Withdrawal TransactionType = "WITHDRAWAL"

	// MonthlyPayment represents the payments from loans
	MonthlyPayment TransactionType = "MONTHLYPAYMENT"
)

// Transaction represents a monetary transaction
type Transaction struct {
	Date        time.Time
	Type        TransactionType
	Description string
	Amount      USD
}

func (txn Transaction) String() string {
	return fmt.Sprintf("[%s] %s - %s - %s", txn.Date.Format("2006/01/02"), txn.Type, txn.Description, txn.Amount)
}

// LineItem represents a cash flow line item. Either an asset or liability
type LineItem interface {
	Description() string
	Process(date time.Time, bank *Bank) error
}

type MonthlyTransaction struct {
	Account    string
	Name       string
	Type       TransactionType
	Amount     USD
	DayOfMonth int
	StartDate  time.Time
	EndDate    time.Time
}

func (m *MonthlyTransaction) Description() string {
	return fmt.Sprintf("%20s\t%s", m.Name, m.Amount)
}

func (m *MonthlyTransaction) Process(date time.Time, bank *Bank) error {
	if date.After(m.EndDate) || date.Day() != m.DayOfMonth {
		return nil
	}

	if date.After(m.StartDate) || date.Equal(m.StartDate) {
		return bank.Append(m.Account, Transaction{Date: date, Type: m.Type, Description: m.Name, Amount: m.Amount})
	}
	return nil
}

type MonthlyTransfer struct {
	From       string
	To         string
	Amount     USD
	DayOfMonth int
	StartDate  time.Time
	EndDate    time.Time
}

func (m *MonthlyTransfer) Description() string {
	return fmt.Sprintf("TRANSFER %s to %s\t%s", m.From, m.To, m.Amount)
}

func (m *MonthlyTransfer) Process(date time.Time, bank *Bank) error {
	if date.After(m.EndDate) || date.Day() != m.DayOfMonth {
		return nil
	}

	if date.After(m.StartDate) || date.Equal(m.StartDate) {
		return bank.Transfer(date, m.From, m.To, m.Amount)
	}
	return nil
}

type OneTimeTransaction struct {
	Account string
	Name    string
	Type    TransactionType
	Amount  USD
	Date    time.Time
}

func (m *OneTimeTransaction) Description() string {
	return fmt.Sprintf("%20s\t%s", m.Name, m.Amount)
}

func (m *OneTimeTransaction) Process(date time.Time, bank *Bank) error {
	if !equalDates(date, m.Date) {
		return nil
	}
	return bank.Append(m.Account, Transaction{Date: date, Type: m.Type, Description: m.Name, Amount: m.Amount})
}

type DailyRandomTransaction struct {
	Account     string
	Name        string
	Type        TransactionType
	BaseAmount  USD
	MaxAmount   USD
	Beta        prob.Beta
	Percentages map[time.Weekday]float64
	StartDate   time.Time
	EndDate     time.Time
}

func (m *DailyRandomTransaction) Description() string {
	return fmt.Sprintf("%20s\t%s - %s", m.Name, m.BaseAmount, m.MaxAmount)
}

func (m *DailyRandomTransaction) Process(date time.Time, bank *Bank) error {
	if date.After(m.EndDate) {
		return nil
	}

	if date.After(m.StartDate) || date.Equal(m.StartDate) {
		perc, ok := m.Percentages[date.Weekday()]
		if ok {
			if rand.Float64() < perc {
				amount := m.BaseAmount + USD(m.Beta.Random()*float64(m.MaxAmount-m.BaseAmount))
				return bank.Append(m.Account, Transaction{Date: date, Type: m.Type, Description: m.Name, Amount: amount})
			}
		}
	}
	return nil
}
