## Rxparse - text parsing in the spirit of Rexx

`rxparse` is a tool that brings the power and intuitive simplicity of the Rexx `parse` statement to the command line.

`rxparse` allows you to split up and reassemble tokens from a line of input using a simple format string. For complex cases with looping and conditions, `awk` or `perl` may be better choices. But for simple extraction of tokens, `rxparse` makes the job much easier.

```
Usage of rxparse:
  -d string
        The default field delimiter (default " *")
  -n    Do not emit newline at the end of each output line
  -o string
        The output expression. 
        Can either be the word 'json', in which case an array of JSON objects is output,
        or a Go template which is used to format the output
  -p string
        The parse expression (default: no values are captured "all")
  -t    Do not trim whitespace from start and end of parsed values
```

## Text delimiters

The `-p` flag is where the power lies. It should be read as a space-separated list of names, ie unquoted, non-numeric text, optionally separated by quoted or numeric delimiters. Input is captured into a name up to the following delimiter. Surrounding whitespace is dropped from the capture unless the `-t` option is provided. `.` acts like a name but any input captured is ignored. 

Eg: 

```
$ date -I | rxparse -p "year '-' month '-' day" -o '{{ .day }}/{{ .month }}/{{ .year }}'
22/07/2021
```

The `year` name is filled from the input up to the first `-`. `month` is filled up to the next `-`. `day` is filled with the remainder. Delimiters have to be in quotes to distinguish them from names. If a delimiter ends in `*`, then all contiguous occurrences of the delimiter text are skipped, otherwise just the first is skipped.  Eg:

```
$ echo "one---two" | rxparse -p "first '-*' second" -o "{{ .second }} : {{ .first }}" 
two : one
```

but:

```
$ echo "one---two" | rxparse -p "first '-' second" -o "{{ .second }} : {{ .first }}" 
--two : one
```

Note the delimiter in the first example ends in `*` while the second doesn't. If you want an asterisk as the final matching character in your delimiter text, precede it by a backslash `\`.

## Numeric delimiters

The following does the same job but using numeric positional delimiters. This is particularly useful for fixed format records. Eg:

```
$ date +%Y%m%d | rxparse -p "year 4 month 6 day" -o '{{ .day }}/{{ .month }}/{{ .year }}'
23/07/2021
```

In this case the delimiter is an unsigned integer, indicating an *absolute* character position. Here, `year` is filled with input up to the 4th character, then `month` up to the 6th character and `day` the rest.

Finally, the following shows the same job but using signed integer, representing *relative* character positions:

```
$ date +%Y%m%d | rxparse -p "year 4 month +2 day" -o '{{ .day }}/{{ .month }}/{{ .year }}'
23/07/2021
```

Again `year` is filled from input with the first 4 characters, but `month` is filled with the next 2 characters thereon, and `day` the remainder.

### Back-tracking

A useful feature of relative delimiters is that they can be used for back-tracking. Eg:

```
$ date -I | rxparse -p "fulldate +10 . -10 year '-'  month '-'  day" -o '{{ .fulldate }} = {{ .day }}/{{ .month }}/{{ .year }}'
2021-07-23 = 23/07/2021
```

First, all 10 characters are read into `fulldate`. Then the input is captured back to 10 characters before (ie. the start) and dropped, then parsing into the individual parts continues as before.

### Default delimiters

It is possible to omit delimters entirely. If a name is followed immediately be another name, they are assumed to be delimited by the *default* delimiter. By default, this is ' *', ie. any number of spaces, but you can change it using the `-d` flag. So all the above examples could be replaced by:

```
$ date -I | rxparse -d '-' -p "year month day" -o '{{ .day }}/{{ .month }}/{{ .year }}'
22/07/2021
```

## Examples

```
$ ls -la | tail -n +2 | rxparse -p ". . . . size . . . name" -o json
[
  {"name":".","size":"4096"},
  {"name":"..","size":"4096"},
  {"name":".git","size":"4096"},
  {"name":".github","size":"4096"},
  {"name":".gitignore","size":"16"},
  {"name":"go.mod","size":"50"},
  {"name":"main.go","size":"7188"},
  {"name":"README.md","size":"4104"},
  {"name":"rxparse","size":"3548218"},
  {"name":".vscode","size":"4096"}
]
```

```
ls -la | tail -n +2 | rxparse -p ". . . . size . . . name" -o "File {{ .name }} is {{ .size }} bytes"
File go.mod is 50 bytes
File main.go is 6608 bytes
File parse is 2334720 bytes
File rxparse is 3355526 bytes
File test.txt is 12 bytes
```
