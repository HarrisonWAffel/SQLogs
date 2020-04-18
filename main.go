package main

import (
	"fmt"
	"github.com/fatih/color"
	"github.com/jroimartin/gocui"
	"github.com/prometheus/common/log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"strings"
)

var logDir = ""

func main() {
	if len(os.Args) == 1 {
		fmt.Print("useage, ./sqlogs <directory of binary logs>")
		os.Exit(1)
	}

	logDir = os.Args[1]

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatal(err)
	}
	defer g.Close()
	//get managers and keybindings

	g.SetManagerFunc(layout)

	if err := initKeybindings(g); err != nil {
		log.Fatalln(err)
	}

	if err = g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		panic(err)
	}
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatal(err)
	}
}

func renderLog(g *gocui.Gui, v *gocui.View) {

	_, cy := v.Cursor()
	if l, err := v.Line(cy); err != nil {
		l = ""
	} else {
		fmt.Println(l)
	}

	fmt.Fprintln(v, "reading")

}

func getLogFiles() []string {

	ext := "binlog"
	var files []string

	//find all bin logs in the logDir
	filepath.Walk(logDir, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			r, err := regexp.MatchString(ext, f.Name())
			if err == nil && r {
				files = append(files, f.Name())
			}
		}
		return nil
	})
	fmt.Print(len(files))

	return files
}

func layout(g *gocui.Gui) error {

	maxX, maxY := g.Size()
	if v, err := g.SetView("sidebar", 0, 0, maxX/2-25, maxY/2-1); err != nil {
		fmt.Fprintf(v, "List Of Binary Log Files in path\n")
		//list log files, add listener on click for opening with mysqlbinlog exec
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		fmt.Fprintf(v, strings.Join(getLogFiles(), "\n"))
	}

	if v, err := g.SetView("logview", maxX/2-23, 0, maxX-1, maxY/2-1); err != nil {
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
		fmt.Fprintf(v, "Log Files will be shown here")
		if _, err := g.SetCurrentView("logview"); err != nil {
			return err
		}
		//wait to load a log file
	}

	//x0, y0 < x1, y1
	if v, err := g.SetView("helpScreen", 0, maxY/2+1, maxX-1, maxY/2+7); err != nil {
		fmt.Fprintf(v, " Ctr+Space: Change Screen \n  ctr+C: quit")
	}

	return nil
}

func nextView(g *gocui.Gui, v *gocui.View) error {
	if v == nil || v.Name() == "sidebar" {
		_, err := g.SetCurrentView("logview")
		return err
	}
	_, err := g.SetCurrentView("sidebar")
	return err
}

func initKeybindings(g *gocui.Gui) error {

	if err := g.SetKeybinding("logview", 'a', gocui.ModNone, autoscroll); err != nil {
		return err
	}

	if err := g.SetKeybinding("logview", gocui.KeyArrowUp, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, -1)
			return nil
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("logview", gocui.KeyArrowDown, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, 1)

			return nil
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("sidebar", 'a', gocui.ModNone, autoscroll); err != nil {
		return err
	}

	if err := g.SetKeybinding("sidebar", gocui.KeyArrowUp, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {

			scrollView(v, -1)

			//get current rows contents -> demo.go
			var fileName string
			var err error

			_, cy := v.Cursor()
			if fileName, err = v.Line(cy); err != nil {
				fileName = ""
			}

			//read file with mysqlbinlog
			re := regexp.MustCompile(`[-]?\d[\d,]*[\.]?[\d{2}]*`)
			var asciiStringContent = ""
			if re.MatchString(fileName) {
				args := []string{"--base64-output=auto", "--verbose", logDir + "/" + fileName}
				cmd := exec.Command("mysqlbinlog", args...)
				asciiByteContent, err := cmd.Output()
				if err != nil {
					log.Error(err)
					log.Fatal("Could not run mysqlbinlog")
				}
				//get the logview and write to it
				asciiStringContent = string(asciiByteContent)
			}

			v, e := g.View("logview")
			if e != nil {
				log.Fatal(e)
			}
			v.Clear()

			//if line has # then color it
			isSqlStatment := false
			lines := strings.Split(asciiStringContent, "\n")
			for _, e := range lines {

				if isSqlStatment {
					if strings.Contains(e, ";") {
						isSqlStatment = false
					}
					fmt.Fprintf(v, color.GreenString(e)+"\n")
				} else {
					if strings.Contains(e, "###") {
						fmt.Fprintf(v, color.GreenString(e)+"\n")
					} else if strings.Contains(e, "#") {
						fmt.Fprintf(v, color.BlueString(e)+"\n")
					} else if strings.Contains(e, "CREATE") {
						isSqlStatment = true
						fmt.Fprintf(v, color.GreenString(e)+"\n")
					} else {
						fmt.Fprintf(v, e+"\n")
					}
				}
			}

			return nil
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("sidebar", gocui.KeyArrowDown, gocui.ModNone,
		func(g *gocui.Gui, v *gocui.View) error {
			scrollView(v, 1)
			//update logview with the contents of file on the selected row.

			//get current rows contents -> demo.go
			var fileName string
			var err error

			_, cy := v.Cursor()
			if fileName, err = v.Line(cy); err != nil {
				fileName = ""
			}

			//read file with mysqlbinlog
			re := regexp.MustCompile(`[-]?\d[\d,]*[\.]?[\d{2}]*`)
			var asciiStringContent = ""
			if re.MatchString(fileName) {
				args := []string{"--base64-output=auto", "--verbose", logDir + "/" + fileName}
				cmd := exec.Command("mysqlbinlog", args...)
				asciiByteContent, err := cmd.Output()
				if err != nil {
					log.Error(err)
					log.Fatal("Could not run mysqlbinlog")
				}
				//get the logview and write to it
				asciiStringContent = string(asciiByteContent)
			}

			v, e := g.View("logview")
			if e != nil {
				log.Fatal(e)
			}
			v.Clear()

			//if line has # then color it
			isSqlStatment := false
			lines := strings.Split(asciiStringContent, "\n")
			for _, e := range lines {

				if isSqlStatment {
					if strings.Contains(e, ";") {
						isSqlStatment = false
					}
					fmt.Fprintf(v, color.GreenString(e)+"\n")
				} else {
					if strings.Contains(e, "###") {
						fmt.Fprintf(v, color.GreenString(e)+"\n")
					} else if strings.Contains(e, "#") {
						fmt.Fprintf(v, color.BlueString(e)+"\n")
					} else if strings.Contains(e, "CREATE") {
						isSqlStatment = true
						fmt.Fprintf(v, color.GreenString(e)+"\n")
					} else {
						fmt.Fprintf(v, e+"\n")
					}
				}
			}

			return nil
		}); err != nil {
		return err
	}

	if err := g.SetKeybinding("logview", gocui.KeyCtrlSpace, gocui.ModNone, nextView); err != nil {
		return err
	}

	if err := g.SetKeybinding("sidebar", gocui.KeyCtrlSpace, gocui.ModNone, nextView); err != nil {
		return err
	}

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func autoscroll(g *gocui.Gui, v *gocui.View) error {
	v.Autoscroll = true
	return nil
}

func scrollView(v *gocui.View, dy int) error {
	if v != nil {
		v.Autoscroll = false
		ox, oy := v.Origin()
		if err := v.SetOrigin(ox, oy+dy); err != nil {
			return err
		}
	}
	return nil
}
