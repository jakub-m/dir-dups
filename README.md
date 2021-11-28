Tools helping find duplicated directories in the backups.

The motivation was a bloated backup with 0.5TB of familiy photos.

### `listfiles`

`listfiles` lists all the files recursively, prints size and the hash. The hash is used to determine if files are duplicates. The different hash options are:

* Full file - read the whole file and calculate the hash. Slow, requires reading all of the content.
* Sampled - use file name, file size and 1KB of bytes from the middle of the file to calculate the hash. This is "good enough" e.g. for family photos.
* Name and size - use only file name and size. The fastest to use, but obviously error prone. Might be a good way to have a first look at the data.

### `analyze`

tbd
