package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (

	// ErrAccountDoesNotExist means the account does not exist
	ErrAccountDoesNotExist = errors.New("Account does not exist")

	// ErrInsufficientFunds means there are insufficient funds available.
	ErrInsufficientFunds = errors.New("Insufficient funds")

	// ErrAccountAlreadyExists means the account could not be created because it already exists
	ErrAccountAlreadyExists = errors.New("Account already exists")

	// ErrUnknownTransactionType means the account could not handle transaction due to unknown type
	ErrUnknownTransactionType = errors.New("Unknown transaction type")

	// ErrInvalidTransfer means the from account has insufficient funds or the transactions were invalid.
	ErrInvalidTransfer = errors.New("Invalid transfer")
)

// DefaultFormatter formats USD currency.
var DefaultFormatter = message.NewPrinter(language.AmericanEnglish)

// USD represents the US dollar
type USD int64

func (u USD) String() string {
	if u < 0 {
		return DefaultFormatter.Sprintf("($%d.%02d)", -1*u/100, -1*u%100)
	}
	return DefaultFormatter.Sprintf("$%d.%02d", u/100, u%100)
}

func Dollars(d USD) USD {
	return d * 100
}

// Account represents a transaction ledger
type Account interface {
	CurrentBalance() USD
	Validate(tx Transaction) bool
	Append(tx Transaction) error
	Update(ctx context.Context, proc Process, bank *Bank, date time.Time)
	String() string
}

// NewBankAccount creates a new bank account. This is the simplest account type.
func NewBankAccount(name string, date time.Time, init USD) *BankAccount {
	return &BankAccount{
		Name:    name,
		Balance: init,
		Ledger: []Transaction{
			Transaction{Date: date, Description: "Initial deposit", Type: Deposit, Amount: init},
		},
	}
}

// BankAccount represents a bank account.
type BankAccount struct {
	Name    string
	Balance USD
	Ledger  []Transaction
}

// Update allows for the account to update account information periodically.
func (a *BankAccount) Update(ctx context.Context, proc Process, bank *Bank, date time.Time) {
}

// CurrentBalance returns the current balance of the account
func (a *BankAccount) CurrentBalance() USD {
	return a.Balance
}

// Append appends a transaction to the account
func (a *BankAccount) Append(tx Transaction) error {
	log.Println(a.Name, tx)
	// log.Println(date.Format("2006/01/02"), item.Description())
	if tx.Type == Deposit {
		a.Ledger = append(a.Ledger, tx)
		a.Balance += tx.Amount
	} else if tx.Type == Withdrawal {
		if tx.Amount > a.Balance {
			return ErrInsufficientFunds
		}
		a.Ledger = append(a.Ledger, tx)
		a.Balance -= tx.Amount
	} else {
		return ErrUnknownTransactionType
	}
	return nil
}

// Validate validates a transaction
func (a *BankAccount) Validate(tx Transaction) bool {
	if tx.Type == Deposit {
		return true
	} else if tx.Type == Withdrawal && a.Balance > tx.Amount {
		return true
	}
	return false
}

func (a *BankAccount) String() string {
	return fmt.Sprintf("%s\t%s", a.Name, a.Balance)
}

// Bank represents a single user bank account
type Bank struct {
	Accounts  map[string]Account
	LineItems []LineItem
}

// Append appends a transaction to the bank account ledger
func (b *Bank) Append(acct string, tx Transaction) error {

	// Check account existence
	account, ok := b.Accounts[acct]
	if !ok {
		return ErrAccountDoesNotExist
	}
	return account.Append(tx)
}

// Transfer transfers money from one account to another.
func (b *Bank) Transfer(date time.Time, from, to string, ammt USD) error {

	// Check account existence
	fromAccount, ok := b.Accounts[from]
	if !ok {
		return ErrAccountDoesNotExist
	}

	toAccount, ok := b.Accounts[to]
	if !ok {
		return ErrAccountDoesNotExist
	}

	// Check available funds
	if fromAccount.CurrentBalance() < ammt {
		return ErrInsufficientFunds
	}

	desc := fmt.Sprintf("Transfer from '%s' to '%s'", from, to)
	withdrawalTxn := Transaction{date, Withdrawal, desc, ammt}
	despositTxn := Transaction{date, Deposit, desc, ammt}

	if !fromAccount.Validate(withdrawalTxn) || !toAccount.Validate(despositTxn) {
		return ErrInvalidTransfer
	}

	// Verify withdrawal success
	if err := fromAccount.Append(withdrawalTxn); err != nil {
		return err
	}

	// Desposits should never fail.
	return toAccount.Append(despositTxn)
}

// AddLineItem adds a line item to the bank.
func (b *Bank) AddLineItem(li LineItem) {
	b.LineItems = append(b.LineItems, li)
}

// Handle handles incoming messages for the bank process.
func (b *Bank) Handle(ctx context.Context, proc Process, msg Message) {
	switch msg.Type {
	case TypeDate:
		date := msg.Value.(time.Time)

		// Process line items
		for _, item := range b.LineItems {
			if err := item.Process(date, b); err != nil {
				log.Println("ERR: ", err)
			}
		}

		// Update account information if necessary
		for _, acct := range b.Accounts {
			acct.Update(ctx, proc, b, date)
		}
	}
}
