package cash_entries

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	firebase "firebase.google.com/go"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/deaglefrenzy/cash_entries/models"
	"github.com/deaglefrenzy/cash_entries/usecase"
)

func init() {
	functions.CloudEvent("createcashentries", CreateCashEntries)
}

func CreateCashEntries(ctx context.Context, event event.Event) error {
	before, after, err := usecase.ConvertEventToStruct(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to convert event to struct: %w", err)
	}

	if after == nil {
		fmt.Println("Document deleted. No new document data available.")
		return nil // No new document data, nothing to process
	}

	cashEntriesBefore := before.CashEntries
	cashEntriesAfter := after.CashEntries

	for _, entry := range cashEntriesAfter {
		found := false
		for _, oldEntry := range cashEntriesBefore {
			if reflect.DeepEqual(entry, oldEntry) {
				found = true
				break
			}
		}
		if !found {
			var PendingEntries models.PendingEntries
			PendingEntries.BranchUUID = after.BranchUUID
			PendingEntries.Resolved = false
			PendingEntries.ResolvedAt = nil
			PendingEntries.ResolvedBy = nil
			PendingEntries.Notes = nil
			PendingEntries.ShiftData.UUID = after.UUID
			PendingEntries.ShiftData.StartTime = after.CreatedAt
			PendingEntries.ShiftData.MainShiftUser = after.Username
			PendingEntries.CashEntry = entry
			PendingEntries.Indexes = IndexString(entry.Description)

			app, err := firebase.NewApp(ctx, nil)
			if err != nil {
				return fmt.Errorf("fail to connect: %w", err)
			}

			fs, err := app.Firestore(ctx)
			if err != nil {
				return fmt.Errorf("fail to connect: %w", err)
			}

			ref := fs.Collection("pending_expense_entries").NewDoc()
			if _, err := ref.Set(ctx, PendingEntries); err != nil {
				return fmt.Errorf("failed to create pending expense entries: %w", err)
			}
		}
	}

	return nil
}

func IndexString(s string) []string {
	s = strings.ToLower(s)
	words := strings.Fields(s) // Split into words, removing all whitespace
	if len(words) == 0 {
		return []string{}
	}

	var result []string

	firstWord := words[0]
	limit := min(len(firstWord), 10)
	for i := 1; i <= limit; i++ {
		result = append(result, firstWord[:i])
	}

	// Add the remaining words
	if len(words) > 1 {
		rest := strings.Join(words[1:], " ")
		result = append(result, firstWord+" "+rest)
	}

	return result
}
