package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"
)

// NewLoan creates a new loan.
func NewLoan(name string, P USD, apr float64, years int, payments int) *LoanAccount {
	periods := years * 12.
	r := apr / 100. / 12.
	rP := r * P.Float64()
	rN := math.Pow(1+r, -float64(periods))
	c := (rP / (1 - rN))
	// log.Println(c)
	a := &LoanAccount{
		Name:                name,
		LoanAmount:          P,
		Periods:             periods,
		MonthlyPayment:      USD(math.Round(c * 100)),
		MonthlyInterestRate: r,
		InterestRate:        apr,
		InterestPaid:        0,
		PrincipalPaid:       0,
		MonthsPaid:          1,
		RemainingBalance:    USD(math.Round(c * float64(periods) * 100)),
		Ledger:              []Transaction{},
	}
	log.Println(a)

	// Starting month adjustment
	if payments > 0 {
		a.RemainingBalance -= USD(int64(a.MonthlyPayment) * int64(payments))

		// Cumulative interest
		interest := (rP-c)*(math.Pow(1+r, float64(payments))-1)/r + c*float64(payments)
		a.InterestPaid = USD(math.Round(interest * 100))

		// Cumulative principal
		principal := c*float64(payments) - interest
		a.PrincipalPaid = USD(math.Round(principal * 100))
		a.MonthsPaid += payments
	}
	log.Println(a)
	return a
}

// LoanAccount represents a loan account.
type LoanAccount struct {
	Name                string
	LoanAmount          USD
	InterestRate        float64
	MonthlyInterestRate float64
	Periods             int
	MonthlyPayment      USD
	RemainingBalance    USD
	PrincipalPaid       USD
	InterestPaid        USD
	MonthsPaid          int
	Ledger              []Transaction
}

// Update allows for the account to update account information periodically.
func (a *LoanAccount) Update(ctx context.Context, proc Process, bank *Bank, date time.Time) {
}

// CurrentBalance returns the current balance of the account
func (a *LoanAccount) CurrentBalance() USD {
	return a.RemainingBalance
}

// Append appends a transaction to the account
func (a *LoanAccount) Append(tx Transaction) error {
	log.Println(a.Name, tx)
	if a.RemainingBalance <= 0 {
		return errors.New("Loan has been paid off")
	}

	// log.Println(date.Format("2006/01/02"), item.Description())
	if tx.Type == Deposit {
		// log.Println(a)

		if a.MonthsPaid < a.Periods {
			a.Ledger = append(a.Ledger, tx)
			a.RemainingBalance -= a.MonthlyPayment

			i := a.MonthsPaid
			P, r, c := a.LoanAmount.Float64(), a.MonthlyInterestRate, a.MonthlyPayment.Float64()

			// Cumulative interest
			interest := (P*r-c)*(math.Pow(1+r, float64(i))-1)/r + c*float64(i)
			a.InterestPaid = USD(math.Round(interest * 100))

			// Cumulative principal
			principal := c*float64(i) - interest
			a.PrincipalPaid = USD(math.Round(principal * 100))
		} else {
			a.RemainingBalance -= tx.Amount
			a.PrincipalPaid += tx.Amount
		}
		a.MonthsPaid++
		// log.Println(a)

	} else {
		return ErrUnknownTransactionType
	}
	return nil
}

// Validate validates a transaction
func (a *LoanAccount) Validate(tx Transaction) bool {
	if tx.Type == Deposit {

		if a.MonthsPaid < a.Periods {
			return true
		} else if a.RemainingBalance > 0 {
			return true
		}
	}
	return false
}

// String returns the string representation of the account
func (a *LoanAccount) String() string {
	return fmt.Sprintf("%s\t%s\n\t- %s\t%s\n\t- %s\t%d\n\t- %s\t%.3f%%\n\t- %s\t%s\n\t- %s\t%s\n\t- %s\t%s\n",
		a.Name, a.RemainingBalance,
		"Loan Amount:\t", a.LoanAmount,
		"Periods:\t", a.Periods,
		"APR:\t\t", a.InterestRate,
		"Monthly Payment:", a.MonthlyPayment,
		"Principal Paid:", a.PrincipalPaid,
		"Interest Paid:", a.InterestPaid,
	)
}

// LoanPayment is a monthly expense to pay down a loan.
type LoanPayment struct {
	From       string
	To         string
	DayOfMonth int
}

func (l *LoanPayment) Description() string {
	return fmt.Sprintf("LOAN PAYMENT %s to %s", l.From, l.To)
}

func (l *LoanPayment) getLoan(bank *Bank) (*LoanAccount, error) {
	acct, ok := bank.Accounts[l.To]
	if !ok {
		return nil, ErrAccountDoesNotExist
	}

	loan, ok := acct.(*LoanAccount)
	if !ok {
		return nil, ErrInvalidTransfer
	}
	return loan, nil
}

func (l *LoanPayment) Process(date time.Time, bank *Bank) error {
	if date.Day() != l.DayOfMonth {
		return nil
	}

	// Get loan account info
	loan, err := l.getLoan(bank)
	if err != nil {
		return err
	}

	if loan.MonthsPaid >= loan.Periods && loan.RemainingBalance > 0 {
		return bank.Transfer(date, l.From, l.To, loan.RemainingBalance)
	} else if loan.MonthsPaid < loan.Periods {
		return bank.Transfer(date, l.From, l.To, loan.MonthlyPayment)
	}
	return nil
}
