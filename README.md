## Deploy

```
$ gcloud functions deploy imageResizer \
  --entry-point ResizeImage \
  --runtime go111 \
  --set-env-vars 'BUCKET_NAME=<your_source_bucket_name>' \
  --trigger-bucket <your_dest_bucket_name> \
  --project ... \
  --region ...
```
