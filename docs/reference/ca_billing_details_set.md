## ca billing details set

Update the account's billing details

### Synopsis

Changes the fields you give and leaves the rest. On an account with no billing record yet, the address must be complete. Needs protected:billing.

```
ca billing details set [flags]
```

### Examples

```
  ca billing details set --billing-email accounts@example.com
```

### Options

```
      --billing-email string     billing email address; leave it out to keep the current value
      --company-number string    company registration number; leave it out to keep the current value
      --country-code string      ISO-3166 alpha-2 country code, e.g. ZA. Absent leaves it unchanged; it cannot be cleared.
  -d, --data string              the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                     help for set
      --postal-address1 string   postal address line 1; leave it out to keep the current value
      --postal-address2 string   postal address line 2; leave it out to keep the current value
      --postal-address3 string   postal address line 3; leave it out to keep the current value
      --postal-address4 string   postal address line 4; leave it out to keep the current value
      --postal-address5 string   postal address line 5; leave it out to keep the current value
      --vat-number string        VAT registration number; leave it out to keep the current value
  -y, --yes                      do not ask for confirmation
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca billing details](ca_billing_details.md)	 - Read and change the billing details

