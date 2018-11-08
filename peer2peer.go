package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/atgjack/prob"
)

func equalDates(d1, d2 time.Time) bool {
	return d1.Year() == d2.Year() && d1.Month() == d2.Month() && d1.Day() == d2.Day()
}

// NewPeer2PeerAccount creates a new peer-to-peer lending account.
func NewPeer2PeerAccount(name string, date time.Time, init, perInvestment USD) *Peer2PeerAccount {
	return &Peer2PeerAccount{
		Name:                 name,
		AccountValue:         init,
		PerInvestment:        perInvestment,
		AvailableCash:        init,
		Deposits:             init,
		Withdrawals:          0,
		Invested:             0,
		Interest:             0,
		OutstandingPrincipal: 0,
		Ledger: []Transaction{
			Transaction{Date: date, Description: "Initial deposit", Type: Deposit, Amount: init},
		},
	}
}

type MicroLoan struct {
	ID                   int
	StartDate            time.Time
	DueDate              time.Time
	PayDay               time.Time
	InterestRate         float64
	MonthlyPrincipal     USD
	MonthlyInterest      USD
	OutstandingPrincipal USD
	TotalPaid            USD
}

func (m *MicroLoan) chargeOff(date time.Time, acct *Peer2PeerAccount) error {
	err := acct.Append(Transaction{
		Date:        date,
		Type:        MonthlyPayment,
		Description: fmt.Sprintf("Charge-off for loan #%d", m.ID),
		Amount:      m.OutstandingPrincipal,
	})
	if err != nil {
		return err
	}
	acct.Interest += m.MonthlyInterest
	acct.AccountValue += m.MonthlyInterest
	acct.AvailableCash += m.MonthlyInterest + m.OutstandingPrincipal
	acct.OutstandingPrincipal -= m.OutstandingPrincipal
	acct.MonthlyCashFlow += m.MonthlyInterest + m.MonthlyPrincipal
	acct.MonthlyInterest += m.MonthlyInterest
	acct.DailyCashFlow += m.MonthlyInterest + m.MonthlyPrincipal
	acct.DailyInterest += m.MonthlyInterest

	m.TotalPaid += m.OutstandingPrincipal + m.MonthlyInterest
	m.OutstandingPrincipal = 0
	return nil
}

func (m *MicroLoan) Process(date time.Time, acct *Peer2PeerAccount) error {

	if m.OutstandingPrincipal > 0 {

		// Paid on time
		if equalDates(m.PayDay, date) {

			// 2% charge-off chance
			if rand.Float64() < 0.005 {
				return m.chargeOff(date, acct)
			}

			// // 75% late payment chance
			// if rand.Float64() < 0.75 {
			// 	beta, _ = prob.NewBeta(2, 4)
			// 	m.PayDay.AddDate(0, 0, int(beta.Random()*14))
			// 	return nil
			// } else {
			// 	beta, _ = prob.NewBeta(5, 10)
			// 	m.PayDay.AddDate(0, 1, int(beta.Random()*30))
			// 	return nil
			// }

			// // failed payment
			// if rand.Float64() > 0.98 {
			// 	m.TotalPaid += m.OutstandingPrincipal
			// 	m.OutstandingPrincipal = 0
			// 	return nil
			// }

			err := acct.Append(Transaction{
				Date:        date,
				Type:        MonthlyPayment,
				Description: fmt.Sprintf("Payment on loan #%d", m.ID),
				Amount:      m.MonthlyInterest + m.MonthlyPrincipal,
			})
			if err != nil {
				return err
			}
			m.TotalPaid += m.MonthlyInterest + m.MonthlyPrincipal
			m.OutstandingPrincipal -= m.MonthlyPrincipal
			acct.Interest += m.MonthlyInterest
			acct.AccountValue += m.MonthlyInterest
			acct.AvailableCash += m.MonthlyInterest + m.MonthlyPrincipal
			acct.OutstandingPrincipal -= m.MonthlyPrincipal
			acct.MonthlyCashFlow += m.MonthlyInterest + m.MonthlyPrincipal
			acct.MonthlyInterest += m.MonthlyInterest
			acct.DailyCashFlow += m.MonthlyInterest + m.MonthlyPrincipal
			acct.DailyInterest += m.MonthlyInterest

			// Increment due date for next payment
			m.DueDate = date.AddDate(0, 1, 0)
			m.PayDay = date.AddDate(0, 0, int(payDateBeta.Random()*60))
		}
	}
	return nil
}

// Peer2PeerAccount represents a bank account.
type Peer2PeerAccount struct {
	Name                 string
	AccountValue         USD
	PerInvestment        USD
	AvailableCash        USD
	Deposits             USD
	Withdrawals          USD
	Invested             USD
	Interest             USD
	OutstandingPrincipal USD
	MonthlyCashFlow      USD
	MonthlyInterest      USD
	DailyCashFlow        USD
	DailyInterest        USD
	Ledger               []Transaction
	MicroLoans           []*MicroLoan
}

// CurrentBalance returns the current balance of the account
func (a *Peer2PeerAccount) CurrentBalance() USD {
	return a.AvailableCash
}

// randBetaDate increments the given date by sum number of days that cooresponds to the beta distrbution
func randBetaDate(beta prob.Beta, date time.Time, max int) time.Time {
	newDate := date.AddDate(0, 0, 1).AddDate(0, 0, int(beta.Random()*float64(max)))
	if newDate.Weekday() == time.Saturday {
		return newDate.AddDate(0, 0, 2)
	} else if newDate.Weekday() == time.Sunday {
		return newDate.AddDate(0, 0, 1)
	}
	return newDate
}

var rateBeta prob.Beta
var startDateBeta prob.Beta
var payDateBeta prob.Beta

func init() {
	rateBeta, _ = prob.NewBeta(3, 8)
	startDateBeta, _ = prob.NewBeta(3, 5)
	payDateBeta, _ = prob.NewBeta(20, 20)
}

func (a *Peer2PeerAccount) broadcastMonthly(proc Process, date time.Time) {
	if date.Day() == 1 {
		proc.Children().Dispatch(Message{
			Timestamp: time.Now().UTC(),
			Type:      TypeMonthlyAccountInfo,
			Value: AccountInfo{
				Date:          date,
				AccountValue:  a.AccountValue,
				AvailableCash: a.AvailableCash,
				CashFlow:      a.MonthlyCashFlow,
				Interest:      a.MonthlyInterest,
			},
			Forward: false,
		})
		// log.Printf("%s Monthly %s %s %s %d\n", date.Format("2006-01-02"), a.AccountValue, a.MonthlyCashFlow, a.MonthlyInterest, len(a.MicroLoans))
		a.MonthlyCashFlow = 0
		a.MonthlyInterest = 0
	}
}

func (a *Peer2PeerAccount) broadcastDaily(proc Process, date time.Time) {
	proc.Children().Dispatch(Message{
		Timestamp: time.Now().UTC(),
		Type:      TypeDailyAccountInfo,
		Value: AccountInfo{
			Date:          date,
			AccountValue:  a.AccountValue,
			AvailableCash: a.AvailableCash,
			CashFlow:      a.DailyCashFlow,
			Interest:      a.DailyInterest,
		},
		Forward: false,
	})
	a.DailyCashFlow = 0
	a.DailyInterest = 0
}

// Update allows for the account to update account information periodically.
func (a *Peer2PeerAccount) Update(ctx context.Context, proc Process, bank *Bank, date time.Time) {
	a.broadcastMonthly(proc, date)
	a.broadcastDaily(proc, date)

	months, periods := 36, 3.
	dailyInvestments := 85
	for a.AvailableCash > a.PerInvestment && dailyInvestments > 0 {
		rate := 1.08 + (rateBeta.Random() * 12 / 100)
		totalRate := math.Pow(rate, periods)
		principalPayment := float64(a.PerInvestment/100) / float64(months)
		interestPayment := (float64(a.PerInvestment/100)*totalRate - float64(a.PerInvestment/100)) / float64(months)

		start := randBetaDate(startDateBeta, date.In(date.Location()), 7)
		a.MicroLoans = append(a.MicroLoans, &MicroLoan{
			ID:                   len(a.MicroLoans),
			StartDate:            start,
			DueDate:              start.AddDate(0, 1, 0),
			PayDay:               start.AddDate(0, 0, int(payDateBeta.Random()*60)),
			InterestRate:         totalRate,
			MonthlyPrincipal:     USD(principalPayment * 100),
			MonthlyInterest:      USD(interestPayment * 100),
			OutstandingPrincipal: a.PerInvestment,
			TotalPaid:            0,
		})
		a.OutstandingPrincipal += a.PerInvestment
		a.AvailableCash -= a.PerInvestment
		a.Invested += a.PerInvestment

		dailyInvestments--
	}

	// Process micro-loans
	startingValue := a.AccountValue
	for _, loan := range a.MicroLoans {
		//  - if due and incomplete, create txn and increment account totals with principal and interest
		if err := loan.Process(date, a); err != nil {
			log.Println("err: ", err)
		}
	}
	if a.AccountValue-startingValue > a.PerInvestment*4 {
		a.PerInvestment *= 2
	}
}

// Append appends a transaction to the account
func (a *Peer2PeerAccount) Append(tx Transaction) error {
	// log.Println(a.Name, tx)
	// log.Println(date.Format("2006/01/02"), item.Description())
	if tx.Type == Deposit {
		a.Ledger = append(a.Ledger, tx)
		a.AccountValue += tx.Amount
		a.AvailableCash += tx.Amount
		a.Deposits += tx.Amount
	} else if tx.Type == Withdrawal {
		if tx.Amount > a.AvailableCash {
			return ErrInsufficientFunds
		}
		a.Ledger = append(a.Ledger, tx)
		a.AvailableCash -= tx.Amount
		a.AccountValue -= tx.Amount
		a.Withdrawals += tx.Amount
	} else if tx.Type == MonthlyPayment {
		a.Ledger = append(a.Ledger, tx)
	} else {
		return ErrUnknownTransactionType
	}
	return nil
}

// Validate validates a transaction
func (a *Peer2PeerAccount) Validate(tx Transaction) bool {
	if tx.Type == Deposit {
		return true
	} else if tx.Type == Withdrawal && a.AvailableCash > tx.Amount {
		return true
	}
	return false
}

// String returns the string representation of the account
func (a *Peer2PeerAccount) String() string {
	return fmt.Sprintf("%s\t%s\n\t- %s\t%s\n\t- %s\t%s\n\t- %s\t%s\n\t- %s\t%d\n\t- %s\t%s\n\t- %s\t%s\n",
		a.Name, a.AvailableCash,
		"Account Value:", a.AccountValue,
		"Deposits:\t", a.Deposits,
		"Invested:\t", a.Invested,
		"Loans:\t", len(a.MicroLoans),
		"Per Loan:\t", a.PerInvestment,
		"Outstanding:\t", a.OutstandingPrincipal,
	)
}
