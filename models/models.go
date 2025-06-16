package models

import "time"

type CashEntry struct {
	CreatedAt   time.Time `firestore:"created_at" json:"created_at"`
	Description string    `firestore:"description" json:"description"`
	Expense     bool      `firestore:"expense" json:"expense"`
	Username    string    `firestore:"username" json:"username"`
	UUID        string    `firestore:"uuid" json:"uuid"`
	Value       float64   `firestore:"value" json:"value"`
}

type PendingEntries struct {
	CashEntry  CashEntry  `firestore:"cash_entry" json:"cash_entry"`
	BranchUUID string     `firestore:"branch_uuid" json:"branch_uuid"`
	Resolved   bool       `firestore:"resolved" json:"resolved"`
	ResolvedBy *string    `firestore:"resolved_by" json:"resolved_by"`
	ResolvedAt *time.Time `firestore:"resolved_at" json:"resolved_at"`
	Notes      *string    `firestore:"notes" json:"notes"`
	ShiftData  ShiftData  `firestore:"shift_data" json:"shift_data"`
}

type ShiftData struct {
	UUID          string    `firestore:"uuid" json:"uuid"`
	StartTime     time.Time `firestore:"start_time" json:"start_time"`
	MainShiftUser string    `firestore:"main_shift_user" json:"main_shift_user"`
}
