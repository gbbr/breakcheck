# breakcheck

breakcheck checks exported values, types and function declarations in your working tree for potential breaking changes against a given git reference. 

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

## Example

Below is an example output from running breakcheck against the datadog-agent repository:

```
$ breakcheck --base=HEAD~40
pkg/util/clusteragent:
  
• Removed struct field "ClusterAgentAPIEndpoint":
    - clusteragent.go:42@HEAD~40:
        struct DCAClient
    - clusteragent.go:57:
        struct DCAClient
  
• Struct field "ClusterAgentVersion" type changed from string to version.Version:
    - clusteragent.go:42@HEAD~40:
        DCAClient
  
• Return value (0) changed from *DCAClient to DCAClientInterface:
    - clusteragent.go:60@HEAD~40:
        func GetClusterAgentClient() (*DCAClient, error)
    - clusteragent.go:75:
        func GetClusterAgentClient() (DCAClientInterface, error)
```

## Caveats

* If a function's argument is changed to an alias of the same type, breakcheck will fail to detect this and will report it as a change. Technically this is not a breaking change.
* Detecting changes in exported package level value declarations is limited to their name and type (when known). 

## Similar work

* https://golang.org/x/tools/internal/apidiff
* https://github.com/bradleyfalzon/apicompat
