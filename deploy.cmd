gcloud functions deploy detectCashEntriesChanges_v2 ^
    --gen2 ^
    --runtime=go122 ^
    --source=. ^
    --region=asia-southeast1 ^
    --entry-point=detectCashEntriesChanges_v2 ^
    --trigger-event-filters="type=google.cloud.firestore.document.v1.written" ^
    --trigger-event-filters="database=(default)" ^
    --trigger-event-filters="document=employee_shifts/{documentId}" ^
    --trigger-location=asia-southeast1
