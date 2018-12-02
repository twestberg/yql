yql is a simple go cli program to query and set single values in
a yaml text file. If symlinks are made to the yql binary named "yset"
and "yget" there is no need to use the "set" or "get" command in the argument list

Usage:
yql [-q] [-f *datafile*] get *keypath*
yql [-q] [-f *datafile*] [-stdin] set *keypath* *value*
yget [-q] [-f *datafile*] *keypath*
yset [-q] [-f *datafile*] [-stdin] *keypath* *value*

-q quiets warnings which normally go to stdout
-f *datafile* chooses the yaml file. This may also be set in the **YQL_FILE** environment variable
-stdin makes the *value* be read from stdin rather than the command line

*keypath* navigates through a map[string]interface{} where any deeper interface{} item in the data file may be another map, an array, or a final value (int or string)

If the *value* parses as an int, it will be set as an int
