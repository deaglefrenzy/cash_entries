package cash_entries

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"

	firebase "firebase.google.com/go"
	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/deaglefrenzy/trigger-test/handler"
	"github.com/deaglefrenzy/trigger-test/models"
	"github.com/googleapis/google-cloudevents-go/cloud/firestoredata"
	"google.golang.org/protobuf/proto"
)

func init() {
	functions.CloudEvent("detectCashEntriesChanges_v2", DetectCashEntriesChanges)
}

func DetectCashEntriesChanges(ctx context.Context, event event.Event) error {
	if event.DataContentType() != "application/protobuf" {
		return fmt.Errorf("unexpected content type: %s", event.DataContentType())
	}

	var data firestoredata.DocumentEventData
	err := proto.Unmarshal(event.Data(), &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshall protobuf data from cloudevent: %w", err)
	}

	pbDocBefore := data.GetOldValue()
	if pbDocBefore == nil {
		log.Println("Document deleted. No new document data available.")
		return nil
	}

	var before models.EmployeeShifts
	if err := handler.To(pbDocBefore.Fields, &before); err != nil {
		return fmt.Errorf("failed to convert event data to struct: %w", err)
	}

	pbDoc := data.GetValue()
	if pbDoc == nil {
		log.Println("Document deleted. No new document data available.")
		return nil
	}

	var after models.EmployeeShifts
	if err := handler.To(pbDoc.Fields, &after); err != nil {
		return fmt.Errorf("failed to convert event data to struct: %w", err)
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
			PendingEntries.Indexes = BuildSearchIndex(entry.Description)

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

func BuildSearchIndex(s string) []string {
	words := strings.Fields(s) // Split into words, removing all whitespace
	if len(words) == 0 {
		return []string{}
	}

	var result []string

	// Process first word progressively
	firstWord := words[0]
	for i := 1; i <= len(firstWord); i++ {
		result = append(result, firstWord[:i])
	}

	// Add remaining words as full entries
	if len(words) > 1 {
		result = append(result, words[1:]...)
	}

	return result
}
