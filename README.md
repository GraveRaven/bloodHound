##bloodHound

bloodHound finds interesting files matching certain criteria (regexps).

###Install

Download the package
```
$ go get github.com/graveraven/bloodHound
```

Install
```
$ go install bloodHound
```

###Configuration
The config.cfg file is currently not read.

The regexps.cfg has 4 sections, a new section is started with ";section section_name"
and can appear anywhere in the file.  
Each entry in a section can be grouped into a category. A new category is
started with ";category category_name".  
Categories are used only for structuring the report. The default category is "none".  
Each line not preceded by a ";" is treated as a regular expression.  
Lines beginning with "#" are treated as comments.

Entries under section "filename" will match filenames of files. It currently does not test directories.
If a filename matches it will be reported without further testing the content.

Entries under section "content" will match the content of a file.
Only files with size <= the max file size will be tested.

Files with a name matching an entry under the section "ignore-content"
will be matched for filename but never checked for content.

Files with a name matching an entry under the section "ignore-filename"
will be ignored completely and never matched against any other category.



###Usage
```
Usage: bloodHound.exe [OPTIONS] <path>

  -config string
        Config file (default "config.cfg")
  -debug
        Debugging
  -max string
        Max size of files to scan (default "10MB")
  -wait int
        Wait delay before completion (default 5)
  -workers int
        Number of concurrent workers (default 8)
```

The max file size can be  
B (bytes)  
K, KB (kilobytes)  
M, MB (megabytes)  
G, GB (gigabytes)  
T, TB (terabyte)

The wait delay is in seconds.

###Notes


###TODO
* Read configuration from config file
* Include directories in filename check