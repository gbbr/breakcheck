# breakcheck

breackcheck checks exported values, types and function declarations in your working tree for potential breaking changes against a given git reference. 

## Usage

```
$ breakcheck --help
Example usage:
  breakcheck               # compares working tree against git head
  breakcheck --base=v1.0.0 # compares against tag v1.0.0

Flags:
  -base string
    	git reference to compare against (default "head")
  -private
    	include exported methods with private receivers
  -v	enable verbose mode
```

## Caveats

* If a function's argument is changed to an alias of the same type, breakcheck will fail to detect this and will report it as a change. Technically this is not a breaking change.
* Detecting changes in exported package level value declarations is limited to their name and type (when known). 
