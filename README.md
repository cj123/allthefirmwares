Download All Firmwares
======================
A utility to download all (or specific sets of) Apple's iOS firmware using the [IPSW Downloads API](https://api.ipsw.me/)

You can find releases of allthefirmwares [on the releases page](https://github.com/cj123/allthefirmwares/releases).

Usage

```
$ ./allthefirmwares --help
Usage of ./allthefirmwares:
  -c	just check the integrity of the currently downloaded files (if any)
  -d string
    	the location to save/check IPSW files.
    		Can include templates e.g. {{.Identifier}} or {{.BuildID}} (default "./")
  -filter string
    	filter by a specific struct field
  -filterValue string
    	the value to filter by (used with -filter)
  -i string
    	only download for the specified device
  -l	only download the latest firmware for the specified devices
  -r	redownload the file if it fails verification (w/ -c)
  -s	only download signed firmwares
```
