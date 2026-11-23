## ca billing details set

Update the account's billing details

### Synopsis

Partial: an omitted or null field is left unchanged. An account with no billing record yet stores the edit and pushes it to billing; the address must then be complete. Requires protected:billing — the backend requires a recent second factor for a session and accepts this scope in its place for a token.

```
ca billing details set [flags]
```

### Examples

```
  ca billing details set --billing-email accounts@example.com
```

### Options

```
      --billing-email string     Billing email address. Null or absent leaves it unchanged.
      --company-number string    Company registration number. Null or absent leaves it unchanged.
      --country-code string      ISO-3166 alpha-2 country code, e.g. ZA. Absent leaves it unchanged; it cannot be cleared.
  -d, --data string              the JSON request body: a literal, @file or @- (stdin); flags override its fields
  -h, --help                     help for set
      --postal-address1 string   Postal address line 1. Null or absent leaves it unchanged.
      --postal-address2 string   Postal address line 2. Null or absent leaves it unchanged.
      --postal-address3 string   Postal address line 3. Null or absent leaves it unchanged.
      --postal-address4 string   Postal address line 4. Null or absent leaves it unchanged.
      --postal-address5 string   Postal address line 5. Null or absent leaves it unchanged.
      --vat-number string        VAT registration number. Null or absent leaves it unchanged.
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

