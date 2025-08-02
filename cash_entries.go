package cash_entries

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	firebase "firebase.google.com/go"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	firestoredata_to_struct "github.com/Lucy-Teknologi/firestoredata-to-struct"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/deaglefrenzy/cash_entries/models"
)

func init() {
	functions.CloudEvent("createcashentries", CreateCashEntries)
}

func CreateCashEntries(ctx context.Context, event event.Event) error {
	before, after, err := firestoredata_to_struct.ConvertEventToStruct[models.EmployeeShifts](ctx, event)
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
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{}
	}

	resultMap := make(map[string]bool)
	var result []string

	for i, word := range words {
		if len(result) >= 35 {
			break
		}

		if !resultMap[word] {
			resultMap[word] = true
			result = append(result, word)
			if len(result) >= 35 {
				break
			}
		}

		if i != 0 && len(word) < 3 {
			continue
		}

		var limit int
		switch i {
		case 0:
			limit = min(len(word), 10)
		case 1:
			limit = min(len(word), 5)
		default:
			limit = min(len(word), 3)
		}

		for j := 1; j <= limit; j++ {
			prefix := word[:j]
			if !resultMap[prefix] {
				resultMap[prefix] = true
				result = append(result, prefix)
				if len(result) >= 35 {
					break
				}
			}
		}
	}

	if len(words) > 1 {
		full := strings.Join(words, " ")
		if !resultMap[full] {
			result = append(result, full)
		}
	}

	return result
}
