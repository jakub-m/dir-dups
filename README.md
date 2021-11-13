
`bin/listfiles` - list files in a directory, with sizes.

`bin/analyze` - compare two lists of files and suggest which directories are duplicated.
 
Uses only filename and size as a "hash" of a file, not really a hash.

Output format, columns, unordered:

1. size of the match in KB
2. size of the match, numer of files. "1" is a single file, more than is a
directory.
3. left path
4. right path
