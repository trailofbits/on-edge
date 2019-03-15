//====================================================================================================//
// Copyright 2019 Trail of Bits. All rights reserved.
// account.go
//====================================================================================================//

package main

import (
	"fmt"
	"log"
	"math/rand"
	onedge "trailofbits/on-edge"
)

var balance = 100

func main() {
	r := rand.New(rand.NewSource(0))
	for i := 0; i < 5; i++ {
		if r.Intn(2) == 0 {
			credit := r.Intn(50)
			fmt.Printf("Depositing %d...\n", credit)
			deposit(credit)
		} else {
			debit := r.Intn(100)
			fmt.Printf("Withdrawing %d...\n", debit)
			onedge.WrapFunc(func() { withdraw(debit) })
		}
		fmt.Printf("New balance: %d\n", balance)
	}
}

func deposit(credit int) {
	balance += credit
}

func withdraw(debit int) {
	defer func() {
		if r := onedge.WrapRecover(recover()); r != nil {
			log.Println(r)
		}
	}()
	/* if balance-debit < 0 {
		panic("Insufficient funds")
	} //*/
	balance -= debit
	if balance < 0 {
		panic("Insufficient funds")
	}
}

//====================================================================================================//
