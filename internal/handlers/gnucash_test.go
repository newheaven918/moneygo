package handlers_test

import (
	"github.com/aclindsa/moneygo/internal/handlers"
	"net/http"
	"testing"
)

func importGnucash(client *http.Client, filename string) error {
	return uploadFile(client, filename, "/v1/imports/gnucash")
}

func gnucashAccountBalanceHelper(t *testing.T, client *http.Client, account *handlers.Account, balance string) {
	t.Helper()
	transactions, err := getAccountTransactions(client, account.AccountId, 0, 0, "")
	if err != nil {
		t.Fatalf("Couldn't fetch account transactions for '%s': %s\n", account.Name, err)
	}

	if transactions.EndingBalance != balance {
		t.Errorf("Expected ending balance for '%s' to be '%s', but found %s\n", account.Name, balance, transactions.EndingBalance)
	}
}

func TestImportGnucash(t *testing.T) {
	RunWith(t, &data[0], func(t *testing.T, d *TestData) {
		// Ensure there's only one USD currency
		oldDefault, err := getSecurity(d.clients[0], d.users[0].DefaultCurrency)
		if err != nil {
			t.Fatalf("Error fetching default security: %s\n", err)
		}
		d.users[0].DefaultCurrency = d.securities[0].SecurityId
		if _, err := updateUser(d.clients[0], &d.users[0]); err != nil {
			t.Fatalf("Error updating user: %s\n", err)
		}
		if err := deleteSecurity(d.clients[0], oldDefault); err != nil {
			t.Fatalf("Error removing default security: %s\n", err)
		}

		// Import and ensure it didn't return a nasty error code
		if err = importGnucash(d.clients[0], "handlers_testdata/example.gnucash"); err != nil {
			t.Fatalf("Error importing from Gnucash: %s\n", err)
		}

		// Next, find the Expenses/Groceries account and verify it's balance
		var income, equity, liabilities, expenses, salary, creditcard, groceries, cable, openingbalances *handlers.Account
		accounts, err := getAccounts(d.clients[0])
		if err != nil {
			t.Fatalf("Error fetching accounts: %s\n", err)
		}
		for i, account := range *accounts.Accounts {
			if account.Name == "Income" && account.Type == handlers.Income && account.ParentAccountId == -1 {
				income = &(*accounts.Accounts)[i]
			} else if account.Name == "Equity" && account.Type == handlers.Equity && account.ParentAccountId == -1 {
				equity = &(*accounts.Accounts)[i]
			} else if account.Name == "Liabilities" && account.Type == handlers.Liability && account.ParentAccountId == -1 {
				liabilities = &(*accounts.Accounts)[i]
			} else if account.Name == "Expenses" && account.Type == handlers.Expense && account.ParentAccountId == -1 {
				expenses = &(*accounts.Accounts)[i]
			}
		}
		if income == nil {
			t.Fatalf("Couldn't find 'Income' account")
		}
		if equity == nil {
			t.Fatalf("Couldn't find 'Equity' account")
		}
		if liabilities == nil {
			t.Fatalf("Couldn't find 'Liabilities' account")
		}
		if expenses == nil {
			t.Fatalf("Couldn't find 'Expenses' account")
		}
		for i, account := range *accounts.Accounts {
			if account.Name == "Salary" && account.Type == handlers.Income && account.ParentAccountId == income.AccountId {
				salary = &(*accounts.Accounts)[i]
			} else if account.Name == "Opening Balances" && account.Type == handlers.Equity && account.ParentAccountId == equity.AccountId {
				openingbalances = &(*accounts.Accounts)[i]
			} else if account.Name == "Credit Card" && account.Type == handlers.Liability && account.ParentAccountId == liabilities.AccountId {
				creditcard = &(*accounts.Accounts)[i]
			} else if account.Name == "Groceries" && account.Type == handlers.Expense && account.ParentAccountId == expenses.AccountId {
				groceries = &(*accounts.Accounts)[i]
			} else if account.Name == "Cable" && account.Type == handlers.Expense && account.ParentAccountId == expenses.AccountId {
				cable = &(*accounts.Accounts)[i]
			}
		}
		if salary == nil {
			t.Fatalf("Couldn't find 'Income/Salary' account")
		}
		if openingbalances == nil {
			t.Fatalf("Couldn't find 'Equity/Opening Balances")
		}
		if creditcard == nil {
			t.Fatalf("Couldn't find 'Liabilities/Credit Card' account")
		}
		if groceries == nil {
			t.Fatalf("Couldn't find 'Expenses/Groceries' account")
		}
		if cable == nil {
			t.Fatalf("Couldn't find 'Expenses/Cable' account")
		}

		gnucashAccountBalanceHelper(t, d.clients[0], salary, "-998.34")
		gnucashAccountBalanceHelper(t, d.clients[0], creditcard, "-272.03")
		gnucashAccountBalanceHelper(t, d.clients[0], openingbalances, "-21014.33")
		gnucashAccountBalanceHelper(t, d.clients[0], groceries, "287.56") // 87.19 from preexisting transactions and 200.37 from Gnucash
		gnucashAccountBalanceHelper(t, d.clients[0], cable, "89.98")

		var ge *handlers.Security
		securities, err := getSecurities(d.clients[0])
		if err != nil {
			t.Fatalf("Error fetching securities: %s\n", err)
		}
		for i, security := range *securities.Securities {
			if security.Symbol == "GE" {
				ge = (*securities.Securities)[i]
			}
		}
		if ge == nil {
			t.Fatalf("Couldn't find GE security")
		}

		prices, err := getPrices(d.clients[0], ge.SecurityId)
		if err != nil {
			t.Fatalf("Error fetching prices: %s\n", err)
		}
		var p1787, p2894, p3170 bool
		for _, price := range *prices.Prices {
			if price.CurrencyId == d.securities[0].SecurityId && price.Value == "17.87" {
				p1787 = true
			} else if price.CurrencyId == d.securities[0].SecurityId && price.Value == "28.94" {
				p2894 = true
			} else if price.CurrencyId == d.securities[0].SecurityId && price.Value == "31.70" {
				p3170 = true
			}
		}
		if !p1787 || !p2894 || !p3170 {
			t.Errorf("Error finding expected prices\n")
		}
	})
}
