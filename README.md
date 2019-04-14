# breakcheck

breackcheck checks exported values, types and function declarations in your working tree for potential breaking changes against a given git reference. 

## Caveats

* If a function's argument is changed to an alias of the same type, breakcheck will fail to detect this and will report it as a change. Technically this is not a breaking change.
* Detecting changes in exported package level value declarations is limited to their name and type (when known). 
