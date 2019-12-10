/* Apache v2 license
*  Copyright (C) <2019> Intel Corporation
*
*  SPDX-License-Identifier: Apache-2.0
 */

package statemodel

const (
	//MovedEvent is the constant for the moved event
	MovedEvent = "moved"
	//CycleCountEvent is the constant for the cyclecount event
	CycleCountEvent = "cycle_count"
	//ArrivalEvent is the constant for the arrival event from the RSP Controller
	ArrivalEvent = "arrival"
	//DepartedEvent is the constant for the departed event
	DepartedEvent = "departed"
	//ReturnedEvent is the constant for the returned event
	ReturnedEvent = "returned"
	//UnknownQualifiedState is the constant for the qualified state to be set initially
	UnknownQualifiedState = "unknown"
	//PresentEpcState is the constant for epc state of present
	PresentEpcState = "present"
	//DepartedEpcState is the constant for epc state of present
	DepartedEpcState = "departed"
	//MaxLocationHistory is the constant for max number of location history entries
	MaxLocationHistory = 10
)
