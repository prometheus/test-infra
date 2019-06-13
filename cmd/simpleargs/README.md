# simpleargs

A simple cli intended to be used with GitHub Actions, it takes a regex as the argument and compares it to the body of the `issue_comment`, if matches it writes the matching groups to `/github/home/simpleargs`, otherwise it exits with [error code 78.](https://developer.github.com/actions/creating-github-actions/accessing-the-runtime-environment/#exit-codes-and-statuses)

> `/github` is the where GitHub mounts a shared filesystem to be accessed by multiple actions.

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