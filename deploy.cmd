gcloud functions deploy createcashentries ^
  --gen2 ^
  --runtime=go122 ^
  --source=. ^
  --region=asia-southeast1 ^
  --entry-point=createcashentries ^
  --trigger-location=asia-southeast1 ^
  --trigger-event-filters=type=google.cloud.firestore.document.v1.written ^
  --trigger-event-filters=database=(default) ^
  --trigger-event-filters-path-pattern=document=employee_shifts/{documentId}
