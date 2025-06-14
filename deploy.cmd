gcloud run deploy detectcashentrieschanges ^
    --base-image=go122 ^
    --source=. ^
    --function detectCashEntriesChanges ^
    --region=asia-southeast1
    --event-filters=type=google.cloud.firestore.document.v1.written ^
    --event-filters=database='(default)' ^
    --event-data-content-type=application/protobuf ^
    --event-filters-path-pattern=document='employee_shifts/{documentId}'
