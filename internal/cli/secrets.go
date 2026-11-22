package cli

import (
	"errors"
	"fmt"
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
				hint: fmt.Sprintf("Delete the key and create it again: ca bucket-key delete %[1]s", name),
			}
		}
		return &hintedError{
			err:  errors.New("the response carried no secret key"),
			hint: fmt.Sprintf("Run it again: ca bucket-key reveal-secret %[1]s", name),
		}
	}
	return &hintedError{
		err:  errors.New("the key was created, but the response carried no secret key"),
		hint: fmt.Sprintf("A replayed create omits the secret. Read it with: ca bucket-key reveal-secret %[1]s", name),
	}
}
