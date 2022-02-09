## `rxparse` - text parsing in the spirit of Rexx

`rxparse` is a tool that brings the power and intuitive simplicity of the Rexx `parse` statement to the command line. It allows you to extract items from a line of standard input and assemble them into useful output, usually for feeding into other processes or display. 

For complex cases with looping and conditions, `awk` or `perl` may be better choices. But for simple extraction of tokens, `rxparse` makes the job very easy.

```
Usage of rxparse:
  rxparse [OPTIONS] <parse-string>

  <parse-string>
        The parse expression, optionally surrounded by quotes (default: "all")

  Options:
  -o string
        The output expression. 
        Can either be the word 'json', in which case an array of JSON objects is output,
        or a Go template which is used to format the output (default: json)
  -d string
        The default field delimiter (default " *")
  -n    Do not emit newline at the end of each output line
  -t    Do not trim whitespace from start and end of parsed values
```

The parse string is a list of tokens, ie. (unquoted) names that identify strings of captured text (or `.` to drop captured text), separated by (quoted or numeric) delimiters which define the limit of input captured into the preceding name. However delimiters can be omitted, in which case a default delimiter is used. If the parse string includes quoted delimiters, either include the whole parse string in alternative style of quotes, or precede the quotes with backslash `\`. 

The `-o` flag is either a Go template that allows the names to be referenced directly, or simply `json`, in which case the 
output is a JSON array of objects with names and captured text as each `<key>: <value>`. 

## Examples

```
$ date -I | rxparse -o '{{ .day }}/{{ .month }}/{{ .year }}' "year '-' month '-' day" 
22/07/2021
```
```
$ ls -la | tail -n +2 | rxparse ". . . . size . . . name"
[
  {"name":".","size":"4096"},
  {"name":"..","size":"4096"},
  {"name":".git","size":"4096"},
  {"name":".gitignore","size":"16"},
  {"name":"go.mod","size":"50"},
  {"name":"main.go","size":"7188"},
  {"name":"README.md","size":"4104"}
]
```

## Text delimiters

The parse string is where the power lies. Input is captured into a named variable up to the delimiter that follows the name in the parse string. Surrounding whitespace is dropped from the captured text unless the `-t` option is provided. `.` acts like a name but any input captured is dropped. 

Eg: 

```
$ date -I | rxparse  -o '{{ .day }}/{{ .month }}/{{ .year }}' "year '-' month '-' day"
22/07/2021
```

The `year` name is filled from the input up to the first `-`. `month` is filled up to the next `-`. `day` is filled with the remainder. Text delimiters have to be in quotes to distinguish them from names. If a delimiter ends in `*`, then all contiguous occurrences of the delimiter text are skipped, otherwise just the first is skipped.  Eg:

```
$ echo "one---two" | rxparse -o "{{ .second }} : {{ .first }}" "first '-*' second"  
two : one
```

but:

```
$ echo "one---two" | rxparse -o "{{ .second }} : {{ .first }}" "first '-' second"
--two : one
```

Note the delimiter in the first example ends in `*` while the second doesn't. If you want an asterisk as the final matching character in your delimiter text, precede it by a backslash `\`.

## Numeric delimiters

The following does the same job but using numeric positional delimiters. This is particularly useful for fixed format records. Eg:

```
$ date +%Y%m%d | rxparse -o '{{ .day }}/{{ .month }}/{{ .year }}' "year 4 month 6 day"
23/07/2021
```

In this case the delimiter is an unsigned integer, indicating an *absolute* character position. Here, `year` is filled with input up to the 4th character, then `month` up to the 6th character and `day` the rest.

Finally, the following shows the same job but using signed integer, representing *relative* character positions:

```
$ date +%Y%m%d | rxparse -o '{{ .day }}/{{ .month }}/{{ .year }}' "year 4 month +2 day"
23/07/2021
```

Again `year` is filled from input with the first 4 characters, but `month` is filled with the next 2 characters thereon, and `day` the remainder.

### Back-tracking

A useful feature of relative delimiters is that they can be used for back-tracking. Eg:

```
$ date -I | rxparse -o '{{ .fulldate }} = {{ .day }}/{{ .month }}/{{ .year }}' "fulldate +10 . -10 year '-'  month '-'  day" 
2021-07-23 = 23/07/2021
```

First, all 10 characters are read into `fulldate`. Then the input is captured back to 10 characters before (ie. the start) and dropped, then parsing into the individual parts continues as before.

### Default delimiters

It is possible to omit delimters entirely. If a name is followed immediately be another name, they are assumed to be delimited by the *default* delimiter. By default, this is ' *', ie. any number of spaces, but you can change it using the `-d` flag. So all the above examples could be replaced by:

```
$ date -I | rxparse -d '-'  -o '{{ .day }}/{{ .month }}/{{ .year }}' "year month day"
22/07/2021
```

