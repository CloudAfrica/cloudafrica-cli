## ca bucket object

List a bucket's objects and get short-lived upload and download URLs

```
ca bucket object [flags]
```

### Options

```
  -h, --help   help for object
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca bucket](ca_bucket.md)	 - Manage object storage buckets
* [ca bucket object download-url](ca_bucket_object_download-url.md)	 - Mint a short-lived URL that downloads one object
* [ca bucket object get](ca_bucket_object_get.md)	 - Get one object's metadata, tags and retention
* [ca bucket object list](ca_bucket_object_list.md)	 - List the objects in a bucket
* [ca bucket object list-deleted](ca_bucket_object_list-deleted.md)	 - List deleted objects that can still be recovered
* [ca bucket object upload-url](ca_bucket_object_upload-url.md)	 - Mint a short-lived URL that uploads one new object
* [ca bucket object versions](ca_bucket_object_versions.md)	 - List the versions of one object

