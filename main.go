package main

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/functions/metadata"
)

// FirestoreEvent is the payload of a Firestore event.
type FirestoreEvent struct {
	OldValue   FirestoreValue `json:"oldValue"`
	Value      FirestoreValue `json:"value"`
	UpdateMask struct {
		FieldPaths []string `json:"fieldPaths"`
	} `json:"updateMask"`
}

// FirestoreValue holds Firestore fields.
type FirestoreValue struct {
	CreateTime string                 `json:"createTime"`
	Fields     map[string]interface{} `json:"fields"`
	Name       string                 `json:"name"`
	UpdateTime string                 `json:"updateTime"`
}

// HelloFirestore is triggered by a change to a Firestore document.
func HelloFirestore(ctx context.Context, e FirestoreEvent) error {
	meta, err := metadata.FromContext(ctx)
	if err != nil {
		return fmt.Errorf("metadata.FromContext: %v", err)
	}

	log.Printf("Hello World! Function triggered by change to: %v", meta.Resource)
	log.Printf("New document: %+v", e.Value)

	// You can access specific fields from the document
	if name, ok := e.Value.Fields["name"]; ok {
		log.Printf("Name field value: %v", name)
	}

	return nil
}
