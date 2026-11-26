package cli

import (
	"errors"
	"fmt"
	"strings"
)

// secretDeliverables lists the writes whose response is the only place a
// credential's secret appears: a replay of one carries the key without it.
var secretDeliverables = map[string]bool{
	"createBucketKey":       true,
	"revealBucketKeySecret": true,
}

// missingSecret is the error for a secret-delivering response that holds no
// secret, so a script capturing stdout does not read an empty value as success.
// name is the key's name as the user gave it.
func missingSecret(operationID, name string, resp []byte) error {
	if !secretDeliverables[operationID] {
		return nil
	}
	v := decode(resp)
	if field(v, "secretKey") != "" {
		return nil
	}
	if operationID == "revealBucketKeySecret" {
		if field(v, "available") == "false" {
			return &hintedError{
				err:  errors.New("the backend holds no secret for this key"),
				hint: "A key created now carries a secret: ca bucket-keys create --name NAME --access-type read|full",
			}
		}
		return &hintedError{
			err:  errors.New("the response carried no secret key"),
			hint: fmt.Sprintf("Run it again: ca bucket-keys reveal-secret %[1]s", name),
		}
	}
	return &hintedError{
		err:  errors.New("the key was created, but the response carried no secret key"),
		hint: fmt.Sprintf("A replayed create omits the secret. Read it with: ca bucket-keys reveal-secret %[1]s", name),
	}
}

// urlDeliverables lists the mints whose stdout is a URL a script feeds to curl.
var urlDeliverables = map[string]bool{
	"createObjectDownloadUrl": true,
	"createObjectUploadUrl":   true,
}

// missingURL is the error for a URL mint whose response holds no url, so
// `curl "$(ca ... -q)"` does not run against an empty string.
func missingURL(operationID string, resp []byte) error {
	if !urlDeliverables[operationID] || field(decode(resp), "url") != "" {
		return nil
	}
	return &hintedError{
		err:  errors.New("the response carried no URL"),
		hint: "Run the command again: each run mints a new URL",
	}
}

// tableNote is the stderr line a table owes for what its columns leave out.
func tableNote(operationID string, resp []byte) string {
	v := decode(resp)
	switch operationID {
	case "getBucketLifecycle":
		if field(v, "unsupported") == "true" {
			return "This bucket also holds lifecycle rules ca cannot show. -o json shows the flag; replace them only with --replace-unsupported."
		}
	case "getBucketObjectMetadata":
		return "Tags and user metadata are in -o json."
	case "createObjectUploadUrl":
		return "The PUT must send the signed headers too: -o json lists them."
	case "getVpcInternalDns", "setVpcInternalDns":
		return zoneNote(v)
	}
	return ""
}

// zoneNote says when a VPC's internal DNS zone leaves names out, which its
// columns do not show: the state can read active while a name is unanswered.
func zoneNote(v any) string {
	var parts []string
	if n := listLen(lookup(v, "clashes")); n > 0 {
		parts = append(parts, fmt.Sprintf("names claimed more than once: %d", n))
	}
	if n := listLen(lookup(v, "skipped")); n > 0 {
		parts = append(parts, fmt.Sprintf("VMs and virtual IPs left out of the zone: %d", n))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "; ") + ". -o json lists them."
}

func listLen(v any) int {
	items, _ := v.([]any)
	return len(items)
}
