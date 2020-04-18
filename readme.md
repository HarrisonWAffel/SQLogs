# SQLogs

A simple Golang program utilizing Gocui which parses binary MySQL logs and displays their contents in a color coded interactive menu. Developed for A Wentworth Digital Forensics Course, 2020

## Installing 
Ensure that you have Golang installed, and have access to MySQL binary logs. Clone this repository and run `go build *.go`. 

## Using the program
To run the program first install and build the source code, then execute `main` and pass a directory containing MySQL binary logs as an argument.
For example 

```
./main /usr/local/var/mysql
```

![](demo.gif)