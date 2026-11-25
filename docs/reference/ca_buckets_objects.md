## ca buckets objects

List a bucket's objects and get short-lived upload and download URLs

```
ca buckets objects [flags]
```

### Options

```
  -h, --help   help for objects
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca buckets](ca_buckets.md)	 - Manage object storage buckets
* [ca buckets objects download-url](ca_buckets_objects_download-url.md)	 - Mint a short-lived URL that downloads one object
* [ca buckets objects get](ca_buckets_objects_get.md)	 - Get one object's metadata, tags and retention
* [ca buckets objects list](ca_buckets_objects_list.md)	 - List the objects in a bucket
* [ca buckets objects list-deleted](ca_buckets_objects_list-deleted.md)	 - List deleted objects that can still be recovered
* [ca buckets objects upload-url](ca_buckets_objects_upload-url.md)	 - Mint a short-lived URL that uploads one new object
* [ca buckets objects versions](ca_buckets_objects_versions.md)	 - List the versions of one object

