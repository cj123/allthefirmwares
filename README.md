Download All Firmwares
======================
A utility to download all (or specific sets of) Apple's iOS firmware using the [IPSW Downloads API](https://api.ipsw.me/)

![Download all the firmwares!](https://dl.dropboxusercontent.com/u/38032597/content/blogs/BsyVbxlCIAAiBtC.jpg)

Usage

<<<<<<< HEAD
```
$ ./allthefirmwares --help
Usage of ./allthefirmwares:
  -c	just check the integrity of the currently downloaded files
  -d string
    	the location to save/check IPSW files.
	 Can include templates e.g. {{.Identifier}} or {{.BuildID}} (default "./")
  -i string
    	only download for the specified device
  -r	redownload the file if it fails verification (w/ -c)
=======
```bash
$ ./allthefirmwares --help
Usage of :./allthefirmwares:
  -c=false: just check the integrity of the currently downloaded files
  -d="./": the location to save/check IPSW files.
	 Can include templates e.g. {{.Identifier}} or {{.BuildID}}
  -i="": only download for the specified device
  -r=false: redownload the file if it fails verification (w/ -c)
  -s=false: only download signed firmwares
>>>>>>> 97c4bfbf80bb4ef83af8c84a3bc8b5afc35ea08e
```
