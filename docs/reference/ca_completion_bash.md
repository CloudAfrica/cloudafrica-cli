## ca completion bash

Generate the autocompletion script for bash

### Synopsis

Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(ca completion bash)

To load completions for every new session, execute once:

#### Linux:

	ca completion bash > /etc/bash_completion.d/ca

#### macOS:

	ca completion bash > $(brew --prefix)/etc/bash_completion.d/ca

You will need to start a new shell for this setup to take effect.


```
ca completion bash
```

### Options

```
  -h, --help              help for bash
      --no-descriptions   disable completion descriptions
```

### Options inherited from parent commands

```
  -o, --output string    output format: table, json or yaml (default table on a terminal, json otherwise)
      --profile string   profile to use (default $CLOUDAFRICA_PROFILE, then "default")
  -q, --quiet            print ids only
```

### SEE ALSO

* [ca completion](ca_completion.md)	 - Generate the autocompletion script for the specified shell

