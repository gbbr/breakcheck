# breakcheck

breackcheck checks exported values, types and function declarations in your working tree for potential breaking changes against a given git reference. 

## Caveats

* If a function's argument is changed to an alias of the same type, breakcheck will fail to detect this and will report it as a change. Technically this is not a breaking change.
* Changes in exported package level value declarations are limited to name and type (if known). 
