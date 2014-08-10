Download All Firmwares
======================

![Download all the firmwares!](https://dl.dropboxusercontent.com/u/38032597/content/blogs/BsyVbxlCIAAiBtC.jpg)

Usage

```
$ go run download.go --help
Usage of :./download:
  -c=false: just check the integrity of the currently downloaded files
  -d="./": the location to save/check IPSW files.
         Can include templates e.g. {{.Identifier}} or {{.BuildID}}
  -i="": only download for the specified device
  -r=false: redownload the file if it fails verification (w/ -c)
```
