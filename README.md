# Gogh2Posh Color Scheme Converter

A command-line-tool to convert the gogh color schemes to regedit reg file
to apply the themes to the microsoft console.

https://mayccoll.github.io/Gogh/


## Install

```bash
go get github.com/jblawatt/go-gogh2posh-converter
```

## Run

```bash
go-gogh2posh-converter -goghTheme="elementary" -out="elementary.reg"
```

## TODO
- Convert other formats then Gogh shell Scripts.
- Find a solution how handle the given foregound and background colors if the color is not listended in the colortable.