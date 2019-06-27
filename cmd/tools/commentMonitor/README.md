# commentMonitor
A simple cli intended to be used with GitHub Actions, it takes a [RE2 flavour regex](https://github.com/google/re2/wiki/Syntax) as the argument and compares it to the body of the `issue_comment`, if matches it writes the matching groups to `/github/home`, otherwise it exits with [error code 78.](https://developer.github.com/actions/creating-github-actions/accessing-the-runtime-environment/#exit-codes-and-statuses)

The matches can be found inside `/github/home` in the form `ARG0`, `ARG1` and so on.

> `/github` is the where GitHub mounts a shared filesystem to be accessed by multiple actions.

> **Important Note**
>
> GitHub Actions uses the HCL language for the workflow files, it has a [known issue related to
> backslash](https://github.com/hashicorp/terraform/issues/4052). So, `\` needs to be replaced by
> `\\` when specifying the argument in the HCL file.

Example:
```
normal: (?mi)^/benchmark\s*(master|[0-9]+\.[0-9]+\.[0-9]+\S*)?\s*$
when using in HCL: (?mi)^/benchmark\\s*(master|[0-9]+\\.[0-9]+\\.[0-9]+\\S*)?\\s*$
```

## Usage
```
usage: main [<flags>] <regex>

simpleargs github comment extract

Flags:
  --help  Show context-sensitive help (also try --help-long and --help-man).
  --eventfile="/github/workflow/event.json"
          path to event.json
  --writepath="/github/home/simpleargs"
          path to write args to

Args:
  <regex>  Regex pattern to match
```

**Local usage example:**
```
./simpleargs --eventfile=./event.json --writepath=./ "^myregex$"
```
