package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/atgjack/prob"
)

func main() {
	log.SetOutput(os.Stdout)

	startDate := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(31, 0, 0)
	// endDate := time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create the LineItem types (asset, expense/liability, monthly, daily)
	// Create the data processors and bank service

	foodBeta, _ := prob.NewBeta(1, 4)

	ctx, cancel := context.WithCancel(context.Background())

	// Create the bank accounts
	bank := Bank{
		Accounts: map[string]Account{
			"Checking":   NewBankAccount("Checking", startDate, Dollars(500)),
			"Investment": NewPeer2PeerAccount("Investment", startDate, Dollars(8000+40000+30000+2e6), Dollars(25)),
		},
		LineItems: []LineItem{
			&MonthlyTransaction{Account: "Checking", Name: "Salary", Amount: Dollars(7000), Type: Deposit, DayOfMonth: 1, StartDate: startDate, EndDate: endDate},
			&MonthlyTransaction{Account: "Checking", Name: "Salary", Amount: Dollars(7000), Type: Deposit, DayOfMonth: 15, StartDate: startDate, EndDate: endDate},
			&MonthlyTransaction{Account: "Checking", Name: "BCBS", Amount: Dollars(630), Type: Withdrawal, DayOfMonth: 17, StartDate: startDate, EndDate: endDate},
			&MonthlyTransaction{Account: "Checking", Name: "Mortgage", Amount: Dollars(1154), Type: Withdrawal, DayOfMonth: 2, StartDate: startDate, EndDate: endDate},
			&MonthlyTransaction{Account: "Checking", Name: "Water", Amount: Dollars(60), Type: Withdrawal, DayOfMonth: 20, StartDate: startDate, EndDate: endDate},
			&MonthlyTransaction{Account: "Checking", Name: "Electricity", Amount: Dollars(115), Type: Withdrawal, DayOfMonth: 10, StartDate: startDate, EndDate: endDate},
			&MonthlyTransaction{Account: "Checking", Name: "Internet", Amount: Dollars(40), Type: Withdrawal, DayOfMonth: 12, StartDate: startDate, EndDate: endDate},
			&MonthlyTransaction{Account: "Checking", Name: "Phones", Amount: Dollars(100), Type: Withdrawal, DayOfMonth: 8, StartDate: startDate, EndDate: endDate},
			&DailyRandomTransaction{Account: "Checking", Name: "Restaurant Food", BaseAmount: Dollars(25), MaxAmount: Dollars(60), Beta: foodBeta, Type: Withdrawal, StartDate: startDate, EndDate: endDate, Percentages: map[time.Weekday]float64{
				time.Monday:    .25,
				time.Tuesday:   .25,
				time.Wednesday: .25,
				time.Thursday:  .25,
				time.Friday:    .75,
				time.Saturday:  .50,
				time.Sunday:    .75,
			}},
			&MonthlyTransfer{From: "Checking", To: "Investment", Amount: Dollars(2000), DayOfMonth: 2, StartDate: startDate, EndDate: endDate},
			&MonthlyTransfer{From: "Checking", To: "Investment", Amount: Dollars(2000), DayOfMonth: 17, StartDate: startDate, EndDate: endDate},
		},
	}

	dailyOutput, err := os.OpenFile("daily.csv", os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		log.Fatal(err)
		return
	}
	dailyOutput.Truncate(0)
	defer dailyOutput.Close()

	monthlyOutput, err := os.OpenFile("monthly.csv", os.O_CREATE|os.O_RDWR, 0755)
	if err != nil {
		log.Fatal(err)
		return
	}
	monthlyOutput.Truncate(0)
	defer monthlyOutput.Close()

	var wg sync.WaitGroup
	engine := NewEngine(ctx, cancel, ProcessList{
		NewDefaultProcess(ctx, "Date Process", &DayGenerator{startDate, endDate}, ProcessList{
			NewDefaultProcess(ctx, "Bank Process", &bank, ProcessList{
				NewDefaultProcess(ctx, "Monthly Output", &MonthlyOutput{monthlyOutput}, ProcessList{}),
				NewDefaultProcess(ctx, "Daily Output", &DailyOutput{dailyOutput}, ProcessList{}),
			}),
		}),
	})
	engine.Start(&wg)

	wg.Wait()

	fmt.Println()
	fmt.Println(bank.Accounts["Checking"])
	fmt.Println(bank.Accounts["Investment"])
	fmt.Println("\nExiting...")
}
