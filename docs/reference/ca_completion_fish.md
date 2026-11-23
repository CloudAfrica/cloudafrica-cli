## ca completion fish

Generate the autocompletion script for fish

### Synopsis

Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	ca completion fish | source

To load completions for every new session, execute once:

	ca completion fish > ~/.config/fish/completions/ca.fish

You will need to start a new shell for this setup to take effect.


```
ca completion fish [flags]
```

### Options

```
  -h, --help              help for fish
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

