package proto_funcs

import (
	"context"
	"fmt"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/deaglefrenzy/cash_entries/models"
	"github.com/googleapis/google-cloudevents-go/cloud/firestoredata"
	"google.golang.org/protobuf/proto"
)

func ConvertEventToStruct(ctx context.Context, event event.Event) (*models.EmployeeShifts, *models.EmployeeShifts, error) {
	if event.DataContentType() != "application/protobuf" {
		return nil, nil, fmt.Errorf("unexpected content type: %s", event.DataContentType())
	}

	var data firestoredata.DocumentEventData
	err := proto.Unmarshal(event.Data(), &data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshall protobuf data from cloudevent: %w", err)
	}

	pbDocBefore := data.GetOldValue()
	pbDocAfter := data.GetValue()

	var before *models.EmployeeShifts
	if pbDocBefore != nil {
		var tmp models.EmployeeShifts
		if err := FirestoreDataTo(pbDocBefore.Fields, &tmp); err != nil {
			return nil, nil, fmt.Errorf("failed to convert event data to struct: %w", err)
		}
		before = &tmp
	}

	var after *models.EmployeeShifts
	if pbDocAfter != nil {
		var tmp models.EmployeeShifts
		if err := FirestoreDataTo(pbDocAfter.Fields, &tmp); err != nil {
			return nil, nil, fmt.Errorf("failed to convert event data to struct: %w", err)
		}
		after = &tmp
	}

	return before, after, nil
}
