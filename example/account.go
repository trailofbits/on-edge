//====================================================================================================//
// Copyright 2019 Trail of Bits
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//====================================================================================================//

package main

import (
	"fmt"
	"log"
	"math/rand"

	onedge "github.com/trailofbits/on-edge"
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
			withdraw(debit)
		}
		fmt.Printf("New balance: %d\n", balance)
	}
}

func deposit(credit int) {
	balance += credit
}

func withdraw(debit int) {
	onedge.WrapFunc(func() {
		defer func() {
			if r := onedge.WrapRecover(recover()); r != nil {
				log.Println(r)
			}
		}()
		// sam.moelius: Uncommenting the next if statement prevents global state changes from occurring
		// before a panic.
		/* if balance-debit < 0 {
			panic("Insufficient funds")
		} // */
		balance -= debit
		if balance < 0 {
			panic("Insufficient funds")
		}
	})
}

//====================================================================================================//
